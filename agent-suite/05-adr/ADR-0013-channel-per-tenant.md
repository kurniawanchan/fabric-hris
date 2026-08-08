# ADR-0013: Channel-per-tenant isolation, with per-tenant client-org multiplicity

- **Status:** **Accepted**
- **Date:** 2026-08-06
- **Deciders:** fabric-architect
- **Rests on assumption(s):** G-03 — the channel-per-tenant *model* is ratified (closed), but two
  sub-assumptions synthesized in this ADR (marked below) remain open: Org3's per-tenant membership
  scope, and the per-tenant client-org multiplicity resolution itself.
- **Supersedes:** ADR-0004
- **Superseded-by:** none
- **Ratifying authority / date:** PRD `prd-fabric-hris-2026-08-02/prd.md` §5.1 (Batasan Teknologi),
  §7 Kelompok D (FR-20…FR-24), INV-5, §11.3 (ADR manifest, row ADR-0013) — human decision, Chandra
  Kurniawan, 2026-08-02/03.

> **Technical correction — 2026-08-06, empirical finding at NET-1.** This ADR's decision to give
> each tenant its own client-org MSP identity is unaffected. Only the literal separator character in
> that MSP ID's string form is corrected: the originally-written underscore form (`OrgClient` + `_`
> + `<tenantID>` + `MSP`) → **`OrgClient-<tenantID>MSP`** (hyphen). Running the real Fabric 2.5
> `configtxgen` binary against a channel profile using the underscore form fails with Fabric's own
> signature-policy-rule tokenizer rejecting the `_` character inside an MSP-ID token used in an
> `AND(...)`/`OR(...)` principal reference (observed literally: `invalid signature policy rule
> "AND('Org1MSP.peer','OrgClient` + `_` + `tenant01MSP.peer')": Unable to access unexported field
> 'peer' in token 'OrgClient` + `_` + `tenant01MSP.peer'`). Confirmed empirically: the identical
> profile with the MSP ID changed to the hyphen form generates the genesis block with no error. This
> document's body below
> is updated in place to the hyphen form rather than left inconsistent with the rest of the design
> package (which was corrected the same day, same finding). Not a redesign, not a new alternative —
> a syntax-level fix forced by the platform this ADR targets.

## Context

ADR-0004 chose a single shared channel with `companyId`-prefixed composite keys, explicitly accepting
"no hard cross-tenant isolation" as a trade-off to be revisited "before any multi-tenant production
use." That revisit is now forced by the ratified requirements themselves, not by a change in
preference:

- **INV-5:** "Isolasi antartenant ditegakkan secara kriptografis melalui *channel-per-tenant* + MSP,
  bukan secara logis" — the composite-key model is a *logical* control (chaincode/application logic
  must remember to filter by prefix on every read); this is precisely **W-3** in PRD §1.2 ("Isolasi
  antartenant bersifat logis, bukan kriptografis... berisiko *cross-tenant leakage* bila terjadi
  kesalahan query").
- **P3**'s falsification test — "Peer tenant B berhasil membaca record tenant A" — must be **structurally
  impossible**, not merely application-logic-prevented, for the predicate to mean what it claims.
- **FR-20/FR-21:** each tenant maps to a **separate Fabric channel**; cross-tenant peer reads are
  rejected by **MSP channel-membership enforcement**, at the cryptographic layer, not the application
  layer.
- **FR-24:** tenant onboarding is an explicit, first-class capability — "penyediaan channel untuk
  tenant baru... termasuk pendaftaran identitas MSP dan penyebaran chaincode ke channel tersebut."
  The phrase **"pendaftaran identitas MSP"** per onboarding is load-bearing for this ADR's decision
  below: it means a new tenant is not merely a new *channel* under an already-fixed org set — it
  brings a **new MSP identity** with it.

This ADR builds on **ADR-0012**, which fixes the three org *roles* (Org1 platform, Org2 enterprise
client, Org3 auditor) but does not by itself say how "Org2" behaves once there is more than one
enterprise-client tenant. That is the specific gap this ADR closes.

Capabilities **D7** (channels) and **D6** (MSP identity) ground the mechanics
`[docs: channels.rst]`, `[docs: msp.rst]`.

## Decision

We will operate **one Fabric channel per tenant**. For `N` onboarded tenant companies there are `N`
channels, each independently created.

**Structural clarification this ADR adds (synthesized from FR-24 + INV-5, not directly quoted PRD
text — flagged as such):** FR-24's requirement to register a new MSP identity per onboarded tenant is
only consistent with "channel-per-tenant" if each tenant is given **its own client-org MSP**, not a
single shared "Org2" identity reused across tenants. We therefore resolve Org2 (ADR-0012) as a
**role**, instantiated **once per tenant** as `OrgClient-<tenantID>` — its own MSP, its own CA
enrollment, its own peer — rather than one fixed org joining every tenant's channel.

Per-tenant channel `tenant-<tenantID>` membership:

| Org | Membership | Notes |
|---|---|---|
| **Org1** (platform) | Every tenant channel | Same 2 peers, same 3 orderer nodes reused across all `N` channels — **not** provisioned per tenant (see Consequences) |
| **`OrgClient-<tenantID>`** (Org2 role) | Exactly that tenant's own channel | New MSP + peer provisioned at onboarding time (FR-24) |
| **Org3** (auditor) | Every tenant channel, **by default** | `[ASSUMPTION] (gap G-03, residual)` — **`TBD:`** does one auditor org service every tenant by default, or is audit scope contracted per tenant (narrowing Org3's membership to only the tenants under active audit)? Neither FR-23 nor §5.1 qualifies this; this ADR assumes the broader default and flags it for correction if a real audit-engagement model exists. |

**Endorsement per channel:** `AND(Org1MSP.peer, OrgClient-<tenantID>MSP.peer)` — the ADR-0012 policy,
instantiated per tenant.

**World-state key consequence (handed off, not redesigned here):** because tenant scoping now lives
at the channel level, on-chain keys **drop the `companyId` prefix** used under ADR-0004 — the
composite key becomes `("profile", employeeID, profileSection)` per rencana-rekonsiliasi.md Gelombang
3 #15. This is `data-model.md`'s change to make, referenced here only as the direct consequence of
this decision.

## Alternatives considered

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. Channel-per-tenant, with per-tenant client-org MSP** | Isolation enforced by MSP + channel membership (cryptographic, per INV-5); matches FR-20/21/24 as written; P3's falsification test becomes a structural impossibility, not an application-logic promise | Lifecycle fan-out and onboarding privilege cost (see Consequences) | **Chosen** |
| B. Single channel + composite-key prefix (ADR-0004, prior) | Simplest; no fan-out cost; already-built pattern | Isolation is application-logic-enforced only — exactly W-3, exactly what INV-5 and P3 now rule out; one query bug anywhere in the read path is a cross-tenant leak | Superseded |
| C. Single channel + a PDC per tenant | Field-level confidentiality without full channel proliferation | A PDC hides a *value* within a still-shared channel — every tenant-scoped peer still observes the *existence*, timing, and keys of every other tenant's transactions on the shared ledger (weaker than channel isolation); PDC collection definitions and `blockToLive` are immutable once set (ADR-0010's corpus fact), so per-tenant PDC provisioning is just as heavyweight as channel provisioning without the isolation benefit | Rejected |
| D. One shared "Org2" MSP reused across all tenant channels (channel-per-tenant, but without the per-tenant-org clarification above) | Fewer MSPs to provision; closer to a literal reading of ADR-0012's fixed 3-org table | Cannot satisfy FR-24's "pendaftaran identitas MSP" per onboarding if the identity is never new; also means one client company's cryptographic identity would be a member of every other tenant's channel config by construction unless explicitly removed each time — an operational foot-gun | Rejected — internally inconsistent with FR-24 |

## Consequences

- **Positive:** FR-20/21/22/23/24 and INV-5 are jointly satisfiable; P3 is testable as a structural
  claim (attempt a cross-channel read from a non-member peer and expect an MSP-layer rejection, not
  an application-layer one); `data-model.md`'s key shape simplifies (drops `companyId`).

- **Negative / trade-off — lifecycle fan-out (stated numerically, per FR-24).** For `N` tenants, the
  chaincode's approve+commit lifecycle must be repeated **once per channel**: `N` approvals by Org1
  **and** `N` approvals by the respective `OrgClient-<tenantID>` (2 approving orgs × `N` channels),
  followed by `N` commit operations — even though the chaincode package and logic are byte-identical
  across every channel. This is **O(N)**, not O(1), lifecycle operations for a single logical
  chaincode. **Not** fanned out, by contrast: the **ordering service** — the same 3 Org1-operated
  Raft nodes serve every tenant channel; ordering capacity is provisioned once, not per tenant.

- **Negative / trade-off — onboarding is a privileged operation (FR-24), not a self-service action.**
  Onboarding tenant `N+1` requires, at minimum: (a) minting a new MSP + CA-enrolled identity for
  `OrgClient-<N+1>`; (b) authoring and submitting a new-channel transaction via the orderer's
  channel-participation API, which requires an **Org1 orderer-admin identity** (Org1 is present on
  every channel and operates the only ordering service); (c) joining Org1's 2 peers, the new client's
  1 peer, and (by the default above) Org3's peer to the new channel; (d) running the chaincode
  approve/commit sequence for that channel. None of this is exposable through an ordinary HRIS admin
  UI without a dedicated, carefully-scoped automation layer sitting in front of Fabric's admin APIs —
  that automation layer is unbuilt and is a real, non-trivial FR-24 deliverable, not a footnote. This
  concentrates onboarding privilege in whoever holds Org1's orderer-admin identity — the same
  organization already flagged in ADR-0012's residual-ordering-risk consequence, compounding it: Org1
  now also unilaterally controls **whether a new tenant can be onboarded at all**.

- **Negative / trade-off:** Org1's 2 peers join **every** one of the `N` tenant channels — their
  ledger storage and gossip/discovery load scale with `N`. No hard ceiling on `N` is available: PRD
  §3.1 confirms the performance targets (P2, Caliper) are still placeholder numbers, so this scale
  limit cannot yet be quantified. Flag as a watch-item, not a currently-known breach.

- **Follow-ups:**
  - `data-model.md` owner: update the world-state key shape to `("profile", employeeID,
    profileSection)`, dropping `companyId` (already tracked in rencana-rekonsiliasi.md Gelombang 3
    #15) — handed off, not redesigned here.
  - Risk register (owned elsewhere, reported here): add "chaincode lifecycle fan-out at scale (O(N)
    approve/commit operations)" and "onboarding privilege concentration in Org1's orderer-admin
    identity" as rows, cross-referencing ADR-0012's ordering-residual-risk row.
  - `sre`/`fabric-operations`, post-G9: the onboarding automation layer implied by FR-24 is a real
    build item; this ADR only establishes that it must exist and what privilege it must hold, not how
    it is built.
  - **`TBD`** carried forward: Org3's per-tenant membership scope (default-every-tenant vs.
    per-engagement) needs a real answer before FR-23 can be tested precisely against P3.

## Related

- Supersedes: **ADR-0004** (its Status field is flipped to `Superseded by ADR-0013` as part of this
  ADR landing — see that file).
- Relates to: **ADR-0012** (fixes the org *role* shape this ADR multiplies per tenant), `data-model.md`
  (world-state key shape consequence, handed off), forthcoming **ADR-0011** (digest scheme — unaffected
  by channel count).
- Knowledge-graph: **D6** (MSP identity), **D7** (channels for tenant isolation). Grounding gap:
  **G-03** (channel-per-tenant model closed; Org3 per-tenant scope sub-assumption open). Context:
  `context/FABRIC-CHANNELS.md`, `context/FABRIC-MSP.md`.

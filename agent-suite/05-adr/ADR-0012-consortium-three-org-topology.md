# ADR-0012: Three-organization consortium topology with platform-operated ordering

- **Status:** **Accepted**
- **Date:** 2026-08-06
- **Deciders:** fabric-architect
- **Rests on assumption(s):** G-04 — the platform/client/auditor triad itself is ratified (no longer
  an open assumption), but G-04 stays **partially open**: a fourth org (regulator-org) and the
  employee-org line from the original gap remain `[ASSUMPTION] (gap G-04)`, deferred, not decided
  against.
- **Supersedes:** ADR-0003
- **Superseded-by:** none
- **Ratifying authority / date:** PRD `prd-fabric-hris-2026-08-02/prd.md` §5.1 (Batasan Teknologi),
  §11.3 (ADR manifest, row ADR-0012) — human decision, Chandra Kurniawan, 2026-08-02/03.

## Context

ADR-0003 accepted a **two vendor-operated orgs** (HR-org + Audit-org) topology and stated its own
limitation plainly: because both orgs answered to the same operator, the trust model was
*honest-but-curious internal separation of duties*, not adversarial multi-party. That limitation is
no longer acceptable once the PRD's ratified objective (§2.1) is read literally: *"klien tidak lagi
perlu mempercayai reputasi vendor, karena dapat memverifikasi integritas datanya sendiri secara
matematis"* — a client that cannot be verified to be independent of the vendor cannot carry that
claim. The ratified success predicate **P3** ("Isolasi data antartenant bersifat kriptografis…
Peer tenant B berhasil membaca record tenant A" as the falsifying test) and **INV-4** ("Salinan
ledger dipegang minimal tiga organisasi independen, sehingga penyedia platform tidak dapat mengubah
catatan sendirian") both require a real, non-vendor-operated party to hold a ledger copy — not
merely a second vendor-operated MSP.

The PRD (§5.1, §11.3) ratifies the consortium shape directly: **three organizations** — the
platform provider (2 peers + 3 Raft orderer nodes), the enterprise client (1 peer, its own
independent ledger replica), and an auditor (1 read-only peer). FR-22 ("Klien enterprise HARUS
memiliki peer yang menyimpan salinan ledger independen") and FR-23 ("Auditor HARUS memiliki peer
read-only") both name these roles explicitly. UJ-1 step 7's closing line — *"Saya harus menguasai
ketiga organisasi sekaligus untuk mengubah catatan itu"* — is the demonstration this topology exists
to make literally true at the endorsement layer.

Knowledge-graph capabilities **D6** (MSP/CA identity) and **D7** (channels) remain the grounding
layer for org/channel mechanics `[docs: msp.rst]`, `[docs: channels.rst]`; this ADR does not revisit
D6/D7 mechanics, only the org count and role assignment that D6/D7 get instantiated against.

## Decision

We will operate a **three-organization Fabric 2.5 consortium**:

| Org | Role | Peers | Ordering | Operated by |
|---|---|---|---|---|
| **Org1** | Platform provider — owns the HRIS write path that triggers `RecordProfileSection`; hosts the ordering service | **2** endorsing + committing peers | **All 3** `etcdraft` consenter nodes | SaaS platform vendor |
| **Org2** (role; multiplicity resolved in ADR-0013) | Enterprise client — independent co-endorser and independent ledger holder | **1** endorsing + committing peer | none | Enterprise client, **client-operated** — not the platform vendor |
| **Org3** | Auditor — independent, read-only verifier | **1** committing peer, **no endorsement role** | none | Auditor (internal compliance function or third party — operator not fixed by any ratified requirement; `**TBD**`) |

**Endorsement policy:** `AND(Org1MSP.peer, Org2MSP.peer)` — a write is valid only with signatures
from at least one Org1 peer **and** at least one Org2 peer. Org3 is deliberately excluded from the
endorsement set (FR-23 scopes it to read-only verification, not co-signing).

**Ordering:** a single `etcdraft` service of **3 consenter nodes, all operated by Org1**. Quorum is a
majority of the consenter set: `quorum = 2` of `3`. Crash-fault tolerance: `f = floor((3-1)/2) = 1` —
the service keeps ordering with **1** node down and **halts ordering** (no new blocks on any channel
it serves) if **2 or more** of the 3 nodes are unavailable simultaneously.

**Stated as a Consequence, not concealed:** this ordering placement is a **residual risk we accept**,
not a solved problem — see Consequences below.

## Alternatives considered

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. Three orgs — Org1 (platform, 2 peers + ordering) / Org2 (client, 1 peer) / Org3 (auditor, 1 peer, read-only)** | Matches INV-4's "≥3 independent orgs" numerically; Org2 is genuinely client-operated, closing ADR-0003's own stated gap; Org3 gives an independent compliance-facing read path (FR-23) distinct from the endorsing pair | Ordering itself is not distributed across the three orgs (see Consequences) | **Chosen** |
| B. Two vendor-operated orgs (ADR-0003, prior) | Simplest; lowest operational footprint | Both orgs answer to one operator — the *"independent client"* premise behind P1/P3 cannot be true; ADR-0003 itself flagged this as a limitation requiring correction, not a stable end state | Superseded |
| C. Four orgs (add a regulator-org alongside Org1/Org2/Org3) | Strongest governance story; matches the original G-04 question's full list (HR/Audit/Employee/Regulator) | No regulator counterparty has been identified or onboarded for this prototype; adding an org with nothing for it to do is scope creep with no requirement behind it | Deferred — `[ASSUMPTION] (gap G-04)`, not rejected; revisit if a regulator counterparty is named |
| D. Distribute the 3 Raft consenters across Org1 **and** Org2 (e.g. 2 Org1 + 1 Org2, or 1-1-1 across Org1/Org2/Org3) | Would remove the single-operator-ordering residual risk noted below; makes "no single org can rewrite the sequence" literally true, not just "no single org can endorse" | No operational agreement exists for Org2 or Org3 to run ordering infrastructure — that is a governance/procurement commitment beyond a client peer, and the PRD's ratified topology (§5.1) explicitly places all 3 orderer nodes with the platform, not distributed | Rejected for this iteration — flagged as the most important future revisit (see Consequences) |

## Consequences

- **Positive:** FR-22 and FR-23 are satisfiable as written (Org2 is truly independent; Org3 has its
  own read-only peer). P3's falsification test now has a real non-vendor actor (Org2's own peer,
  or an attempted cross-channel read from another tenant's peer) to run against. The endorsement
  policy makes UJ-1 step 7's claim literally true **for the write path**: Org1 alone cannot produce a
  valid endorsement — Org2's signature is structurally required.

- **Negative / trade-off — residual risk, stated explicitly, not hidden.** All **3** Raft consenter
  nodes are Org1-operated. This means:
  1. **Availability is unilaterally Org1's to grant or withhold.** Org1 can halt ordering on every
     channel it services (which, under channel-per-tenant, is *every* tenant channel — ADR-0013) by
     itself, with no action from Org2 or Org3 required or possible to prevent it. A 2-of-3 outage of
     Org1's own infrastructure — or a deliberate Org1 action — stops all new blocks network-wide.
  2. **This is in tension with the premise "Org2 independen" that P1's trust story rests on.** P1's
     trust story is about *tamper-evidence of already-committed content* (Org1 cannot forge an
     endorsement it doesn't have), which the AND-policy genuinely secures. It does **not** extend to
     *availability* or to *transaction sequencing/inclusion timing* — both remain entirely within
     Org1's operational control, because ordering is a distinct function from endorsement and this
     topology concentrates it in one org. Stating this precisely: Org1 cannot rewrite or forge a
     committed, endorsed transaction; Org1 **can** unilaterally decide whether and when any
     transaction — including Org2's own submissions — gets ordered at all.
  3. This must be carried into the G6 threat model as a named residual risk (not merged into or
     hidden behind the "honest-but-curious" language ADR-0003 used) — the risk shape has changed from
     "both endorsers are the same operator" to "the endorsers are genuinely independent, but ordering
     is not," which is a materially different and narrower risk, and should be described as such.

- **Negative / trade-off:** Org3's operator is not fixed by any ratified requirement — `**TBD**`
  whether it is an internal compliance function of the platform, an external third-party auditor, or
  per-client-contracted. This affects whether Org3 is a genuinely independent trust boundary or, if
  operated by the platform itself, a second Org1-controlled entity wearing an "auditor" label — which
  would weaken (without falsifying) INV-4's "≥3 independent organizations" framing. Recorded as an
  open question, not resolved here.

- **Follow-ups:**
  - Append a residual-risk row to the risk register (owned elsewhere — reported, not authored here):
    "all 3 Raft consenters are Org1-operated; Org1 can unilaterally halt ordering / control inclusion
    timing network-wide even though it cannot forge endorsements."
  - Recommendation (not a decision — flagged for `sre`/`fabric-operations`, post-G9): even while all
    3 consenters stay Org1-operated, spread them across **≥3 independent availability zones** within
    Org1's own infrastructure. This mitigates the *infrastructure-failure* half of the risk (a single
    site outage no longer halts ordering) but does **not** address the *organizational* half (Org1
    alone still controls all 3 identities/keys) — the two halves need to be told apart when this is
    revisited.
  - Revisit Option D (distributing consenters across orgs) and Option C (regulator-org) together if
    a future engagement requires the ordering-availability risk itself to be closed, not just
    infrastructure-hardened.
  - Org2's multiplicity across tenants (one Org2 identity vs. one per-tenant client org) is
    **specified in ADR-0013**, not here — this ADR fixes the *role* shape, ADR-0013 fixes how that
    role multiplies under channel-per-tenant.

## Related

- Supersedes: **ADR-0003** (its Status field is flipped to `Superseded by ADR-0012` as part of this
  ADR landing — see that file).
- Relates to: **ADR-0013** (channel-per-tenant — resolves Org2's per-tenant multiplicity), **ADR-0005**
  (MSP/OU mapping — org names updated, mapping shape unchanged), forthcoming **ADR-0011** (digest
  scheme — unaffected by org count, consumes whichever peers endorse).
- Knowledge-graph: Layer A (actors), **D6** (MSP/CA identity), **D7** (channels). Grounding gap:
  **G-04** (platform/client/auditor triad closed; regulator-org line remains open). Context:
  `context/FABRIC-MSP.md`, `context/FABRIC-ORDERING.md`, `context/FABRIC-CHANNELS.md`.

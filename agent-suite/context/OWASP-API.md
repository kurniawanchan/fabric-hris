# OWASP API Security Top 10 — mapping for the Fabric 2.5 + HRIS system

> **Re-derived 2026-08-06** against the ratified design in `prd-fabric-hris-2026-08-02/prd.md`
> (status: final) and **ADR-0012/ADR-0013/ADR-0014/ADR-0016/ADR-0020**. The pre-reconciliation
> version of this doc (a standalone anchor-service consuming a Kafka topic, a Private Data
> Collection as the confidentiality control, a two-organization consortium) is **superseded in
> substance** — see `../11-execution/grounding-gaps.md` and `../05-adr/README.md`.

> **What this is.** A context doc (gate **G1**) mapping the OWASP API Security Top 10 (2023) — Layer E
> frame · **OWASP API Security Top 10**, mandated by `[brief: agents-guide.md]` — onto this system's
> API surfaces: the internal/public HTTP APIs, the gateway trust boundary, and the Fabric
> gateway-client/chaincode surface. The two risks the brief calls out — **API1 Broken Object Level
> Authorization (BOLA)** and **API5 Broken Function Level Authorization (BFLA)** — get the deepest
> treatment because they map directly to the existing **IDOR Guard** and RBAC.
>
> **What this is NOT.** Not a pentest and not the finding report (that is G6, `08-security/`).
> Companion lenses: `OWASP-ASVS.md` (V4 access control in verification terms),
> `SECURITY-BY-DESIGN.md` (least-privilege posture). Chaincode-control depth →
> `fabric-security-review/references/chaincode-security-review.md`.
>
> **Citations:** `[code: …]` real repo · `[docs: …]` pinned Fabric corpus · `[prd: …]` ratified PRD ·
> `[brief: …]` brief · `[ASSUMPTION]` + gap ID (`../11-execution/grounding-gaps.md`).

## The API attack surfaces (where these risks live)

1. **Internal service API** — S2S callers with an internal-scope header/API-key
   `[code: ems/internal/base/handler/base.go]`.
2. **Public API via the gateway** — external / partner integrators, feature-flag-gated per-endpoint
   API keys `[code: ems/app/api/public_router.go]`, `[code: ems/app/appconf/config.go]`. Highest-
   exposure surface.
3. **Employee self-service API** — profile reads and edits routed through the per-section
   change-approval workflow `[code: ems/app/api/public_router.go]`.
4. **Fabric Gateway client / chaincode contract** — the overlay's transaction surface. **There is no
   standalone anchor-service API here** — the write call (`RecordProfileSection`) runs in-band from
   the HRIS write path (ADR-0014), and the three read-only functions
   (`GetProfileSectionRecord`/`GetProfileHistory`/`GetEmployeeProfileSummary`, `api-contracts.md`,
   ADR-0020) are the only other surface. Authorized by X.509 identity (D6) and chaincode-level
   MSP/CID verification (D9).

**The trust boundary that governs all four:** the gateway injects identity/scope/tenant headers.
Whether the gateway reliably **strips inbound copies and re-injects** these is unverified
`[ASSUMPTION G-18/PB-1]`. If a client can forge a tenant or internal-scope header, BOLA/BFLA collapse
regardless of downstream checks. This is the top API-security question for the design, and it is
**unaffected by the 2026-08-06 reconciliation** — it was open before and remains open now.

---

## API1 · Broken Object Level Authorization (BOLA) — primary

**Risk:** a caller manipulates an object ID (employee ID, tenant ID) to read/edit a record it does
not own — classic IDOR. For an HRIS this exposes another employee's national ID, salary, or bank
data, or another tenant's profile-section digests.

**Controls in this system:**

- **App tier — the IDOR Guard.** The application enforces object-ownership so a caller only reaches
  records in its scope `[code: ems/internal/base/authz/guard.go]`. This is the existing BOLA control
  and the pattern the ledger side mirrors.
- **Tenant scoping.** A tenant-ID header scopes every request to one tenant; the cross-tenant
  consultant/support actor is the one identity allowed to switch it — and therefore the one most
  able to trigger BOLA if the header is spoofable `[ASSUMPTION G-18/PB-1]`.
- **Ledger tier — channel membership is the first BOLA control, ahead of any application check
  (D7).** Tenant isolation is now achieved entirely by **channel-per-tenant** membership
  (ADR-0013): a peer that is not on a tenant's channel cannot even simulate a read against that
  tenant's world state — there is no PDC `memberOnly` policy in this design for a BOLA to bypass,
  because there is no PDC at all.
- **Ledger tier — chaincode ABAC (D9), inside a channel.** Chaincode reads the caller's identity via
  CID (`GetID`/`GetMSPID`) to enforce FR-6 before any write
  [docs: identity/identity.md]. Read functions (`GetProfileSectionRecord` et al.) accept only an
  `employeeID`/`profileSection` pair and return only digest-bearing fields — never a value that would
  make a successful BOLA read useful for anything beyond confirming a digest exists (ADR-0020).
- **Confidentiality-by-construction (D2/D3).** Even a BOLA that reaches the ledger returns only a
  digest, never plaintext — no sensitive PII is on-chain by contract shape, not by a collection
  policy (ADR-0020; see `SECURITY-BY-DESIGN.md` Tenet 1).

**Fact vs. recommendation.** That a peer outside a tenant's channel cannot read that tenant's world
state is FACT (ADR-0013, [docs: channels.rst]). That chaincode *should* re-derive the tenant/employee
scope from the CID rather than trust a transaction argument alone is a recommendation — the axis is
whether the tenant/employee identifier is an authenticated attribute or a caller-supplied parameter
(prefer the former wherever the underlying Fabric API allows it).

---

## API5 · Broken Function Level Authorization (BFLA) — primary

**Risk:** a lower-privileged actor invokes an admin-only function (e.g. an employee triggers a
bulk-export, or writes an anchor that only the HRIS write path itself should be able to produce).

**Controls in this system:**

- **App tier — role RBAC.** The application gates functions on view/edit permissions and **excludes
  the Employee and Finance roles** from broad employee-data operations
  `[code: ems/internal/base/authz/employee_rbac.go]`. Deny-by-role is the existing BFLA control.
- **Ledger tier — endorsement policy (D5) + ACLs (D8).** The one write function,
  `RecordProfileSection`, requires co-signature from **both** the platform org and the anchoring
  employee's tenant client org (`AND(Org1MSP.peer, OrgClient-<tenantID>MSP.peer)`, ADR-0012); a
  low-privilege identity, or the platform operator acting alone, cannot satisfy it
  [docs: endorsement-policies.rst]. The auditor org is excluded from the endorsement set entirely,
  consistent with its read-only role (FR-23). ACLs bind each resource to a policy so system functions
  are not callable by the wrong role [docs: access_control.md].
- **Role → OU mapping is the correctness hinge.** BFLA on-ledger is only as good as the HRIS-role →
  MSP-OU/attribute mapping `[ASSUMPTION G-11]`. If "admin" is not a distinguishable attribute in the
  cert, chaincode cannot enforce function-level authz beyond the coarse FR-6 MSP-verification check.

**Blast radius.** A BFLA on the anchor-write function would let an unauthorized actor forge
tamper-evident "audit" evidence — poisoning the very integrity guarantee the overlay exists to
provide. The two-org co-signature requirement (not a single endorsing org, and not the platform
operator alone) is the mitigation [docs: endorsement-policies.rst], and it is stronger in this design
than the pre-reconciliation two-org model because the anchoring employee's **own** client org, not a
platform-controlled second org, must co-sign (ADR-0012) — contingent on that org's operator being
genuinely independent (`security-architecture.md` T4, `[ASSUMPTION G-04]`).

---

## The rest of the Top 10 (mapped, briefer)

| # | Risk | Control in this system / note | Source |
|---|---|---|---|
| **API2** | Broken Authentication | Gateway-injected identity + S2S API-key; ledger uses X.509/MSP across three orgs (**D6**). Root of trust is the gateway boundary — verify it. | `[code: ems/internal/base/handler/base.go]`, [docs: security_model.md], `[ASSUMPTION G-18/PB-1]` |
| **API3** | Broken Object Property Level Authz (excessive data exposure / mass assignment) | Return only masked/needed PII fields at the app tier; edits go through the change-approval workflow, not blind mass-assignment. On-ledger, no function accepts or returns a section value or a salt at all — enforced by the chaincode contract's argument shape, not by field filtering after the fact (ADR-0020, FR-34/35). | `[code: ems/pkg/pii/mask.go]`, ADR-0020 |
| **API4** | Unrestricted Resource Consumption | Rate-limiting/quotas are a gateway concern. **recommendation:** rate-limit the public API-key surface and the profile-write/anchor path; not verified here. | — |
| **API5** | Broken Function Level Authz | *(primary — see above)* | — |
| **API6** | Unrestricted Access to Sensitive Business Flows | Each of the five profile-section change-approval workflows is the sensitive flow whose commit triggers an anchor; keep its trigger authorized exactly as the underlying section write already is (ADR-0014's in-band call adds no new trigger surface). | `[prd: §4]`, ADR-0014 |
| **API7** | SSRF | Low relevance to the ledger surface itself; applies to the IPFS-fetch path when a supporting document's CID is resolved. **recommendation:** allowlist the Fabric gateway endpoint and the IPFS cluster API only. | ADR-0016 |
| **API8** | Security Misconfiguration | Default Fabric policies/ACLs/TLS-off are the misconfiguration class — override per `SECURITY-BY-DESIGN.md` Tenet 1. The IPFS Private Cluster's own defaults (swarm-key rotation, membership) are a **new** misconfiguration surface with no ADR'd hardening checklist yet. | [docs: policies/policies.md], [docs: enable_tls.rst], `[ASSUMPTION]` (IPFS leg) |
| **API9** | Improper Inventory Management | Public API exposure is feature-flag gated with per-endpoint keys — a documented inventory boundary; keep the read-only verification functions out of the public surface unless deliberately exposed to a verifier. | `[code: ems/app/api/public_router.go]`, `[code: ems/app/appconf/config.go]` |
| **API10** | Unsafe Consumption of APIs | **Retired as a Kafka-consumption row** — there is no Kafka consumer and no anchor-service in this design (ADR-0014). The equivalent unsafe-consumption surface is now **IPFS CID handling**: validate CID format before fetch/store, and never trust a CID from an unauthenticated source, since a CID is a retrieval capability, not a secret (ADR-0016). | ADR-0016 |

---

## The BOLA/BFLA enforcement chain (Zero-Trust view)

```
request ─▶ Gateway: strip+reinject trust headers          [trust boundary · G-18/PB-1]
        ─▶ App: scope/API-key auth (API2)                 [code: base/handler/base.go]
        ─▶ App: Guard object-ownership (API1/BOLA)          [code: base/authz/guard.go]
        ─▶ App: RBAC role gate    (API5/BFLA)               [code: base/authz/employee_rbac.go]
        ─▶ Fabric: X.509 identity, 3 orgs (D6)               [docs: msp.rst]
        ─▶ Fabric: channel membership (API1/D7)              rejects cross-tenant before any app check
        ─▶ chaincode: CID/ABAC per-function (API1/API5/D9)   [prd: §7 FR-6]
        ─▶ endorsement policy AND(Org1,OrgClient) (API5/D5)  [docs: endorsement-policies.rst]
        ─▶ result: digest only, no plaintext, ever (D2/D3)   ADR-0011/ADR-0020
```

Each hop re-authorizes; no hop trusts the one before it (NIST Zero Trust, Layer E). The chain is only
as strong as its weakest link, and today that link is still **G-18/PB-1** (the header trust
boundary) — resolve it before security sign-off. The channel-membership hop is a genuinely **new**
early rejection point this reconciliation adds; it did not exist under the single-shared-channel
model.

---

## Traceability

| API risk | Fabric capability | Existing code control | Gaps |
|---|---|---|---|
| API1 BOLA | D7, D9 | Guard/IDOR, tenant scoping | G-18/PB-1 |
| API5 BFLA | D5, D8 | RBAC exclusions | G-11 |
| API2 auth | D6 | gateway/S2S auth | G-18/PB-1 |
| API3 property-level | D2 | PII masking, chaincode contract shape (ADR-0020) | — |
| API8 misconfig | (config) | — (→ hardening ref); IPFS cluster leg open | `[ASSUMPTION]` (IPFS) |
| API9 inventory | — | public-API feature flag | — |
| API10 unsafe consumption | — | IPFS CID validation (re-scoped from the retired Kafka row) | — |

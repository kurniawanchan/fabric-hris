# Security-by-Design — Fabric 2.5 + HRIS PII overlay

> **Re-derived 2026-08-06** against the ratified design in `prd-fabric-hris-2026-08-02/prd.md`
> (status: final) and **ADR-0011/ADR-0012/ADR-0013/ADR-0015/ADR-0019/ADR-0021**. The
> pre-reconciliation version of this doc (a two-org topology, Private Data Collections as the
> on-chain confidentiality control, and `PurgePrivateData`/`blockToLive` as the erasure layer) is
> **superseded in substance** — see `../11-execution/grounding-gaps.md` and `../05-adr/README.md`.

> **What this is.** A context doc (DSRM phase 2 / gate **G1** objective) that applies the
> Security-by-Design frame (Layer E · **SbD**) to *this* system: the existing HRIS application
> (generic register — see `project-context.md` AUTHORITY NOTICE) on a shared operational database,
> plus the Fabric 2.5 audit/anchor overlay. It states the secure-default, least-privilege and
> defense-in-depth posture the design commits to and traces each to a Fabric capability
> (**D1..D15**) or an existing code control.
>
> **What this is NOT.** Not the review workflow and not the finding report. The *how-to-harden*
> checklist and the STRIDE pass live in the skill; the ratified control set is produced in **G6**
> (`08-security/`). Reference, do not restate:
> `fabric-security-review/references/config-hardening.md` (cited hardening items),
> `fabric-security-review/references/stride-threat-model.md` (threat catalog),
> `../08-security/security-architecture.md` (the T1–T21 STRIDE model actually ratified for this
> system).
>
> **Citations.** `[docs: …]` = pinned Fabric 2.5 corpus; `[code: …]` = real repo; `[ASSUMPTION]` +
> gap ID = requirement-derived specific, unresolved (see `../11-execution/grounding-gaps.md`).
> Full convention in `../11-execution/knowledge-graph.md`.

Security-by-Design means the secure configuration is the *default* one, privilege is *granted* not
*revoked*, and no single control is load-bearing. The three tenets below are the design contract.

---

## Tenet 1 — Secure defaults (the safe path is the default path)

Fabric ships several settings **off** by design; SbD requires flipping them before, not after,
exposure. These are documented FACTs, not recommendations.

| Control | Documented default | SbD posture for this system | Capability |
|---|---|---|---|
| **TLS in transit** | Server TLS is configured per node; **client auth is OFF by default** [docs: enable_tls.rst] | TLS on every peer/orderer link across all **three** orgs; mutual TLS on cross-org and admin/operator links | **D13** |
| **Channel / ACL / endorsement policies** | Defaults are "for getting started" and "can and should be overridden in a production environment" [docs: policies/policies.md] [docs: access_control.md] | Override every `Readers`/`Writers`/`Admins` and the ACL table before pilot; the ratified write-endorsement policy already requires **both** the platform org and the tenant's own client org to co-sign — no single org, including the platform operator, can satisfy it alone (ADR-0012) | **D5, D8** |
| **Operations service** | Does **not** use the MSP; relies entirely on mutual TLS with client-cert auth [docs: security_model.md] | Reachable only from the operator network, client-cert list kept minimal | **D13** |
| **On-chain confidentiality — no PDC to misconfigure** | A plain world-state write commits the same way regardless of content [docs: ledger/ledger.md] | No sensitive PII plaintext, and no reversible derivative, ever on-chain — only a per-section salted digest and HMAC-derived pseudonymous identifiers (ADR-0011); this is enforced at the **chaincode-contract shape** (no function accepts a section value, ADR-0020), not by a collection's `memberOnly` policy that could be left misconfigured | **D2, D3** |
| **App-level PII at rest** | Already enforced: AES-256 envelope encryption with versioned keys `[code: ems/pkg/db/encryption_plugins.go]` | Reuse — the ledger overlay never weakens the existing at-rest default | (existing, Domain B) |

> Item-by-item cited settings and the FACT/RECOMMENDATION split:
> `fabric-security-review/references/config-hardening.md`. Do not duplicate that list here.

**recommendation:** treat "default config still in place at pilot" as a release blocker. The docs
mandate override intent but not a timeline; the axis is exposure — an internet-reachable orderer with
default `Admins` is Critical, an internal-only pilot is lower.

---

## Tenet 2 — Least privilege (grant, never blanket-trust)

Every actor gets the narrowest identity and the narrowest resource grant that lets it do its job.
This mirrors the existing app-tier RBAC, which already *excludes* the Employee and Finance roles
from broad employee-data reads `[code: ems/internal/base/authz/employee_rbac.go]`.

- **Identity is per-actor and cryptographic.** Each actor is an X.509 identity from a CA, validated
  by an MSP; policies restrict actions to specific MSPs and roles
  [docs: security_model.md] [docs: identity/identity.md]. This is **D6** and is the foundation the
  other grants build on — now spanning **three** organizations (platform, per-tenant enterprise
  client, auditor), not two (ADR-0012).
- **Map HRIS roles to MSP OUs / chaincode attributes.** HRIS roles should map to X.509 OUs or
  attributes read via CID inside chaincode, reproducing the existing app-tier RBAC exclusions
  on-ledger `[ASSUMPTION G-11]` `[code: ems/internal/base/authz/employee_rbac.go]`. This is **D9**
  (chaincode ABAC) layered on **D6**; in this design it also enforces the narrower FR-6 rule ("reject
  if the caller is not MSP-verified") on the one write function, `RecordProfileSection`.
- **Endorsement policy is an authorization gate, not just availability.** The write-endorsement
  policy scopes co-signature to exactly the platform org and the anchoring employee's tenant client
  org (`AND(Org1MSP.peer, OrgClient-<tenantID>MSP.peer)`, ADR-0012) — the auditor org is excluded
  from the endorsement set entirely, consistent with its read-only role
  [docs: endorsement-policies.rst]. This is **D5**.
- **ACLs bind each resource to a policy.** Tighten resource→policy bindings from defaults so a
  low-privilege identity cannot invoke a privileged system chaincode [docs: access_control.md]. **D8**.
- **Tenant isolation is the least-privilege boundary now — no collection to also configure.** Every
  peer on a tenant's channel holds only that tenant's world state; there is no Private Data
  Collection layered underneath to additionally scope (channel-per-tenant, ADR-0013, **D7**) — one
  fewer confidentiality surface to misconfigure than the pre-reconciliation model.
- **The cross-tenant consultant actor is still the sharp edge.** Any app-tier identity that can
  switch its tenant scope mid-session (`[code: ems/internal/base/handler/base.go]`) is the one
  identity whose Fabric-side channel membership must be re-verified per switched tenant, never
  granted a union of channels.

**Fact vs. recommendation.** That identity/role/policy enforcement exists is FACT (cited). *Which*
OU each HRIS role maps to, and whether Finance is excluded on-ledger exactly as in code, is a
recommendation pending the role-mapping decision (G-11).

---

## Tenet 3 — Defense-in-depth (no single point of trust)

PII protection does not rest on one wall. Layers, outermost to innermost, with the control at each:

```
 ┌─ Gateway ─ strips/re-injects trust headers ── trust boundary  [ASSUMPTION G-18/PB-1]
 │  ┌─ App authZ ─ Guard/IDOR + RBAC  [code: ems/internal/base/authz/guard.go]
 │  │  ┌─ HRIS write path (in-band, ADR-0014) ─ computes DataHash/EmployeeID/UpdatedBy off-chain
 │  │  │  ┌─ Transport ─ TLS / mutual TLS (D13)  [docs: enable_tls.rst]
 │  │  │  │  ┌─ Fabric identity ─ MSP + X.509, three orgs (D6)  [docs: msp.rst]
 │  │  │  │  │  ┌─ Chaincode ABAC ─ CID/OU/role (D9) + endorsement, AND(Org1,OrgClient) (D5)
 │  │  │  │  │  │  ┌─ Confidentiality ─ off-chain PII + on-chain digest (D2), salt (D3) — no PDC
 │  │  │  │  │  │  │  ┌─ At rest ─ AES-256 envelope (Domain B) + KEY_EMPLOYEE for docs (Domain B′)
 │  │  │  │  │  │  │  │  └─ Key protection ─ HSM via PKCS#11 (D14)  [docs: hsm.md]
 └──┴──┴──┴──┴──┴──┴──┴─────────────────────────────────────────────────────────────────────
 Immutability + audit spans all layers ─ ledger, per-section-per-employee chain (D12)  [docs: ledger/ledger.md]
```

- **The overlay adds a layer, it does not replace one.** The operational database stays
  system-of-record and stays AES-encrypted; Fabric adds a tamper-evident audit layer over it
  `[ASSUMPTION G-09]`. If the ledger is compromised, the confidential PII is still protected at rest;
  if the database is tampered with, the on-chain digest detects it (**D2/D12** → CIA-Integrity),
  contingent on genuine independence of the tenant client org's operator (`security-architecture.md`
  T4, `[ASSUMPTION G-04]`).
- **Key models are deliberately separate — four domains now, not two.** Fabric MSP/HSM signing keys
  (**D14**, [docs: hsm.md]), the application AES PII key, the per-employee document-encryption key
  (`KEY_EMPLOYEE`), and the identifier-pseudonymization secret (`employeeKey_i` + per-record salt)
  are four distinct domains; no domain may be derived from, wrap, or be wrapped by another
  (ADR-0019, reaffirmed by ADR-0021). HSM protects MSP identity keys, **not** TLS keys — TLS uses
  file-based keys [docs: hsm.md]. A design that assumes one key store covers everything is wrong,
  and now there are six pairwise boundaries to keep separate, not one.
- **Erasure is a layer, too — entirely off-chain.** Right-to-erasure is a four-part crypto-shred:
  delete the operational-database field, destroy `KEY_EMPLOYEE`, destroy the per-record salts,
  destroy `employeeKey_i` (ADR-0015). **Neither `PurgePrivateData` nor `blockToLive` is used** —
  there is no PDC anywhere in this design for either mechanism to act on (ADR-0015 Context). On-chain
  state is never touched; surviving digests become permanent orphans (`[ASSUMPTION G-06]` on the
  regulatory retention window; see the distinct, open Pasal-42 gap in `PRIVACY-BY-DESIGN.md` §6).
  This supports Privacy-by-Design.

**Blast radius (FR-11).** Turning on mutual TLS without distributing client root CAs first drops
every client at the handshake [docs: enable_tls.rst]. Raising the endorsement policy without the
endorsing orgs live blocks all writes on that tenant's channel. Sequence hardening changes; do not
big-bang them into pilot.

---

## Traceability

| SbD tenet | Fabric capabilities | Existing code control | Frame edges (Layer E) |
|---|---|---|---|
| Secure defaults | D2, D3, D5, D8, D13 (no D1 — no PDC to default-misconfigure) | AES-at-rest `[code: ems/pkg/db/encryption_plugins.go]` | SbD, CIA-C |
| Least privilege | D5, D6, D7, D8, D9 | RBAC/Guard `[code: ems/internal/base/authz/guard.go]`, `[code: ems/internal/base/authz/employee_rbac.go]` | SbD, Zero Trust, OWASP BOLA/BFLA |
| Defense-in-depth | D2, D3, D6, D9, D12, D13, D14 (no D4 — erasure is off-chain crypto-shred, ADR-0015) | AES + masking `[code: ems/pkg/pii/mask.go]` | SbD, CIA (all), PbD |

**Related context docs:** access-control specifics → `OWASP-API.md` (BOLA/BFLA) and `OWASP-ASVS.md`
(V4). The end-to-end threat enumeration and current residual-risk disposition →
`../08-security/security-architecture.md` (G6, T1–T21).

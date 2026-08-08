<!--
Immutable once Accepted. Supersede with a new ADR; do not edit in place.
-->

# ADR-0005: Map HRIS roles to X.509 identity read via chaincode CID for attribute-based access control

- **Status:** **Accepted**
- **Date:** 2026-07-13
- **Deciders:** security-architect
- **Rests on assumption(s):** G-11 (HRIS-role → OU/attribute mapping is not yet fixed by a requirement doc). Grounded technically in `[code: ems/internal/base/authz/employee_rbac.go]` and `[docs: identity/identity.md]`.

> **Implementation-status note — 2026-08-07, empirical finding at `QA-1`.** This ADR's Decision
> below is unchanged and not being corrected — unlike ADR-0013's hyphen fix, this is not a
> syntax-level error in the ADR's own text. It is a disclosure that the chaincode actually shipped
> this session (`fabric-network/chaincode/employeeprofilerecord/chaincode/identity.go`) implements
> **Option B** (coarse org-level MSP-ID allow-listing via `cid.GetMSPID`/`cid.GetID`) — the option
> this ADR explicitly **rejected** — not Option A (enrolled `hris.role`/`hris.company` X.509
> attributes read via `cid.GetAttributeValue`/`AssertAttributeValue`), which was chosen. Confirmed
> by grep: zero `GetAttributeValue` calls exist anywhere in the chaincode's non-vendor source. The
> practical consequence is exactly the one this ADR's own Alternatives table already named for
> Option B: any authenticated member of an org's MSP can submit/evaluate for that org regardless of
> HRIS role — the employee/finance read-exclusion and consultant company-scoping this ADR exists to
> provide are **not enforced at the chaincode layer today**. Tracked as grounding gap **G-30**
> (`grounding-gaps.md`), not silently reinterpreted as "this ADR meant org-level all along." Closing
> this requires either a new ADR formally accepting Option B's trade-off for this prototype's scope,
> or implementing the attribute-based mechanism this document actually decided — neither is done
> here.

## Context
The anchor/audit overlay must decide, at ledger-write time, *which actor may anchor or read which PII change* — not merely *which org* is calling. The five HRIS roles are grounded: `tbl_user.role` `1=super admin, 2=employee, 3=admin` `[code: ems/internal/base/authz/employee_rbac.go]`, plus **finance** and **consultant (multi-company)** actors (knowledge-graph Layer A). The existing app RBAC gates on `EmployeeView`/`EmployeeEdit` and **explicitly excludes the employee and finance roles from broad employee-data reads** `[code: ems/internal/base/authz/employee_rbac.go]`; the consultant is a privileged cross-tenant actor. The ledger authorization model must mirror that boundary or it will over- or under-expose PII.

Fabric supplies the mechanism: X.509 identity interpreted by an MSP (D6, context `FABRIC-IDENTITY`, `FABRIC-MSP`), and the chaincode **CID (client identity) API** for attribute-based access control (D9) — reading the caller's MSP-ID, OU, and enrolled attributes inside the transaction. Endorsement/ACL policies (D5/D8, context `FABRIC-POLICIES`) reference the same principals. This ADR fixes the *mapping*; policy wiring and the org set are separate decisions (G-04). This is a design decision only — no prototype code.

## Decision
We will represent each Fabric actor with an **X.509 identity** whose **organization membership + Fabric NodeOU (client/peer/admin)** are set by the MSP, and whose **HRIS business role and company scope are carried as enrolled X.509 attributes** (issued at Fabric-CA enrollment, e.g. `hris.role ∈ {super_admin, admin, employee, finance, consultant}` and `hris.company`). Chaincode reads these via the **CID API** (`GetMSPID`, `GetID`, `GetAttributeValue`, `AssertAttributeValue`) and enforces attribute-based access control **at transaction time**, replicating the existing app-layer exclusions (employee and finance excluded from broad PII reads; consultant scoped by asserted company). This realizes the Zero-Trust (E6) and OWASP BOLA/BFLA (E5) frames per-request, mirroring the existing IDOR Guard `[code: ems/internal/base/authz/guard.go]`.

## Alternatives considered
| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. HRIS role/company as enrolled X.509 attributes + org/NodeOU via MSP, enforced in chaincode via CID (ABAC, D9)** | Role-level least privilege at the ledger; mirrors existing RBAC exclusions; per-request verification (Zero Trust) | Requires an enrollment convention + CA attribute issuance (G-11 to ratify) | **Chosen** |
| B. Coarse per-org identity only (org-level MSP membership, no role/company attributes) | Simplest MSP setup | **Cannot enforce role-level ABAC**; any org member can read/anchor any PII change — **over-exposes PII** and breaks the employee/finance exclusion | Rejected because it collapses five distinct HRIS roles into one principal, defeating least privilege |
| C. Idemix `ou`+`role` attributes only | Anonymity/unlinkability | Idemix 2.5 exposes only `ou`+`role`, **no custom attributes** (cannot carry `finance`/`consultant`/company) and **cannot endorse** — see ADR-0008 | Rejected because it cannot express the required role/company granularity and is verify-only |
| D. Keep all authorization in the app layer; Fabric identity stays coarse | No new identity model | A compromised anchor service could write on behalf of any role; no independent ledger-side check — violates Zero-Trust defense-in-depth | Rejected because it removes the ledger's own authorization boundary |

## Consequences
- **Positive:** Least-privilege at the ledger; the on-chain check is independent of and consistent with the app RBAC; consultant cross-tenant access is bounded by an asserted `hris.company` attribute; endorsement/ACL policies (D5/D8) can name these principals directly.
- **Negative / trade-off:** Introduces an enrollment convention that must be kept in sync with `tbl_user.role` changes; attribute issuance and rotation become an operational concern (cert lifecycle D15). X.509 attributes are visible to endorsers (no anonymity) — acceptable here because the anchor overlay runs inside the vendor's two-org consortium, not to external verifiers.
- **Follow-ups:** Ratify the exact attribute keys/values against a real requirement when G-11 closes. Define the endorsement/ACL policies that reference these principals (depends on the consortium-topology decision, G-04). Add "attribute/`tbl_user.role` drift" as an operational risk to `10-risk/risk-register.md`.

## Related
- Relates to: ADR-0008 (ZKP/Idemix scope — why Idemix is not the identity vehicle here); ADR-0009 (key-management separation — the signing keys behind these identities). Depends on the consortium-topology decision (G-04) and the no-plaintext-PII-on-chain decision (G-02) authored separately.
- Knowledge-graph: Layer D — **D6** (MSP + CA X.509 identity), **D9** (chaincode ABAC via CID); frames **E6** (Zero Trust), **E5** (OWASP API Top 10 BOLA/BFLA), **E4** (Security-by-Design). Grounding gap(s): **G-11** (role→attribute map), with **G-04** (org set) as a dependency. Context doc(s): `context/FABRIC-IDENTITY.md`, `context/FABRIC-MSP.md`, `context/FABRIC-POLICIES.md`, `context/CRYPTOGRAPHY.md` §2.1.

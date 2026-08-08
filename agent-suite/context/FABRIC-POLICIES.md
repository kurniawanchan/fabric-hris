# Fabric Policies, Endorsement & ACLs — context stub

**What it is.** Policies are how Fabric expresses "who must agree" for an action. **Signature** and
**ImplicitMeta** policies (e.g. `AND('Org1.member','Org2.member')`, `MAJORITY Admins`) govern channel
config and administration; **endorsement policies** state which orgs/principals must sign a valid
transaction (settable per chaincode, per collection, or per key); and **ACLs** map named network
resources (e.g. event/query APIs) to a policy. Together they are the declarative authorization layer
above MSP identities.

> **Depth: see fabric-identity-security/references/policies-and-acl.md and
> fabric-chaincode-dev/references/endorsement.md — do not restate here.** The first owns
> signature/ImplicitMeta syntax + ACLs; the second owns endorsement-policy design at chaincode/
> collection/key scope.

**Corpus citations (verified).**
- `[docs: access_control.md]` — ACLs mapping resources to policies
- `[docs: endorsement-policies.rst]` — endorsement policy syntax and scopes

**HRIS hooks (knowledge-graph Layer D).**
- **D5** (key-level endorsement policies — no collections; PDC is retired, ADR-0015) — gates who must
  sign a PII-anchor write: **`AND(Org1MSP.peer, OrgClient-<tenantID>MSP.peer)`** per tenant channel
  (ADR-0012/ADR-0013); the auditor org (`Org3MSP`) commits and reads only, never endorses.
  **D8** (ACLs on resources) — restricts who may query/subscribe to ledger resources. Together they
  serve Security-by-Design (E4) and OWASP BFLA/BOLA (E5).
- Gap **G-11** — the HRIS-role → principal mapping that these policies reference is unresolved.
  **G-04** — closed (ADR-0012): the org set is `Org1` / `OrgClient-<tenantID>` / `Org3`.

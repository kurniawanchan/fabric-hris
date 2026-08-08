# Fabric Chaincode (Smart Contracts) & Lifecycle — context stub

**What it is.** Chaincode is Fabric's smart-contract program (Go/Node/Java) that reads and writes the
world state through the contract/shim API; a package can hold multiple contracts. Fabric 2.x uses the
**decentralized lifecycle**: package → install (on peers) → approve-for-my-org → commit-to-channel,
so a chaincode definition (name, version, endorsement policy, collections) is agreed by org policy
before it runs.

> **Depth: see fabric-chaincode-dev/references/contract-api.md and
> fabric-chaincode-dev/references/lifecycle.md — do not restate here.** Those own the contract API
> surface, deterministic-transaction rules, `peer lifecycle` commands, and the approve/commit flow
> (private-data handling lives in FABRIC-PRIVATE-DATA; the client seam in gateway-client.md).

**Corpus citations (verified).**
- `[docs: smartcontract/smartcontract.md]` — smart-contract vs chaincode concepts, contract API
- `[docs: chaincode_lifecycle.md]` — package/install/approve/commit lifecycle

**HRIS hooks (knowledge-graph Layer D).**
- **D9** (attribute-based access control in chaincode via CID / OUs / roles) — the anchor chaincode
  reads the caller's X.509 attributes to gate writes, mirroring the existing IDOR/RBAC Guard.
- Gap **G-11** — how HRIS roles (super admin / admin / employee / finance / consultant) map to MSP
  OUs / chaincode attributes is unresolved. **G-05** (closed) — the chaincode transaction surface
  anchors **profile sections** (`PERSONAL`/`EMPLOYMENT`/`EDUCATION`/`ADDITIONAL`/`PAYROLL`), one
  `RecordProfileSection` call per section write, not discrete business events —
  `EVENT_UPDATE_PERSONAL` and its siblings are retired, superseded by section-based anchoring
  (PRD §4).

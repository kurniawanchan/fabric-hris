# Fabric Architecture & Transaction Flow — context stub

**What it is.** Hyperledger Fabric is a permissioned, modular DLT whose peers run an
**execute-order-validate** transaction lifecycle rather than order-execute: a client proposal is
first *endorsed* (executed against chaincode by endorsing peers), then *ordered* into blocks by the
ordering service, then *validated* (endorsement-policy + MVCC checks) and committed to each peer's
ledger. The ledger itself is two linked structures — an append-only, hash-chained **blockchain** of
transactions plus a mutable **world state** (see FABRIC-WORLD-STATE).

> **Depth: see fabric-core/references/architecture-overview.md and
> fabric-core/references/transaction-flow.md — do not restate here.** Those own the component
> topology, proposal→endorse→order→validate→commit sequence, and MVCC read/write-set semantics
> (cross-refs fabric-performance/references/mvcc-and-read-write.md).

**Corpus citations (verified).**
- `[docs: architecture.rst]` — component architecture, execute-order-validate rationale
- `[docs: txflow.rst]` — endorsement / ordering / validation transaction flow
- `[docs: ledger/ledger.md]` — blockchain + world-state ledger structure, immutability

**HRIS hooks (knowledge-graph Layer D).**
- **D12** (ledger immutability & audit) — the append-only, hash-chained transaction log is the
  integrity/audit substrate for the anchor-overlay design; underpins CIA-Integrity (frame E1).
- Gap **G-09** — working assumption is Fabric as an *audit/anchor overlay*, not source-of-truth;
  MySQL stays system-of-record. Revisit if a real requirement makes any field ledger-authoritative.

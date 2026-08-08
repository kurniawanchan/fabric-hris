# Fabric Private Data Collections (PDC) — context stub

**What it is.** Private Data Collections let a subset of organizations on a channel share confidential
data **without putting it on the shared ledger**: the actual private data is gossiped peer-to-peer and
stored in a side database on authorized peers, while only a **hash of the data** is written to the
channel ledger as tamper-evident proof. PDCs support per-collection membership + endorsement policies,
`blockToLive` auto-purge, and (v2.5) explicit `PurgePrivateData` for erasure — the core primitives
Fabric offers, in general, for confidentiality + right-to-erasure.

> **Not used in this design.** ADR-0015 retires PDC, `PurgePrivateData`, and `blockToLive` entirely
> for this project. Confidentiality is field-level via a per-section salted `DataHash` on plain world
> state — the section content itself never goes on-chain, collection or no collection. Erasure is
> **crypto-shred** (destroy `KEY_EMPLOYEE` + the record's salts + `employeeKey_i`), not a ledger-side
> purge. Multi-tenant isolation is **channel-per-tenant** (ADR-0013), not PDC-within-a-shared-channel.
> The Fabric-concept explanation above stays correct as general knowledge — keep it loaded if a
> sibling agent needs to reason about *why* PDC was rejected — but do not instantiate any of it
> against this design without first reopening ADR-0015.

> **Depth: see fabric-chaincode-dev/references/private-data.md — do not restate here.** It owns
> collection config, the hash-on-ledger / data-off-ledger split, salting predictable values,
> `blockToLive`/`PurgePrivateData`, and collection-level endorsement, as general Fabric capability —
> independent of whether this project uses it.

**Corpus citations (verified).**
- `[docs: private-data/private-data.md]` — PDC concept, hash-on-ledger, purge / `blockToLive`
- `[docs: private-data-arch.rst]` — architecture, dissemination, salting against brute-force

**HRIS hooks (knowledge-graph Layer D).**
- **D1** (field-level confidentiality), **D2** (on-chain hash/commitment vs off-chain data) are now
  realized **without** PDC — a per-section `DataHash` on plain world state (ADR-0011/ADR-0020) is the
  confidentiality mechanism. **D4** (purge / `blockToLive` retention & erasure) is realized by
  crypto-shred (ADR-0015), not by any PDC purge primitive.
- Gap **G-02** — closed (ADR-0011: exactly the per-section `DataHash` plus two HMAC-derived
  identifiers, nothing else, on-chain). Gap **G-06** — closed (ADR-0015: crypto-shred erasure, no
  PDC).

# Fabric World State & State Database — context stub

**What it is.** The world state is the current-value view of the ledger — the latest key/value for
every state key — rebuildable by replaying the blockchain. It is backed by a pluggable state DB:
**LevelDB** (embedded key/value, default, simple keys/range queries) or **CouchDB** (document store
enabling rich JSON queries and indexes when chaincode state is modeled as JSON). The choice affects
query capability and performance, not ledger semantics.

> **Depth: see fabric-performance/references/state-database.md and
> fabric-core/references/ledger-world-state.md — do not restate here.** Those own the LevelDB-vs-
> CouchDB trade-off, indexing, and tuning (dev-side rich queries live in
> fabric-chaincode-dev/references/couchdb-queries.md).

**Corpus citations (verified).**
- `[docs: couchdb_as_state_database.rst]` — LevelDB vs CouchDB, rich queries, indexes
- `[docs: ledger/ledger.md]` — world state as current-value projection of the ledger

**HRIS hooks (knowledge-graph Layer D).**
- **D11** (world state: CouchDB vs LevelDB) — determines whether anchor records support rich queries;
  low prototype volume favors LevelDB unless query needs force CouchDB.
- Gap **G-08** — NFR targets (tenant volume, throughput, latency) are unresolved; working
  [ASSUMPTION] is 1 pilot tenant / low volume with default LevelDB.

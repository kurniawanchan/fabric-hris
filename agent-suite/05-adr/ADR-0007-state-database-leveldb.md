# ADR-0007: LevelDB world-state for the prototype

- **Status:** **Accepted**
- **Date:** 2026-07-13
- **Deciders:** fabric-architect
- **Rests on assumption(s):** G-08 (non-functional targets / query needs not fixed by any requirement doc)

## Context
Fabric peers back the world state with either **LevelDB** (embedded key/value) or **CouchDB** (external,
supports rich JSON queries). The anchor records defined in the G5 data model are **key-addressed** —
retrieved by the composite key `("anchor", companyId, employeeRef, changeSeq)` — with no need for
ad-hoc JSON field queries. Capability **D11**. `[docs: couchdb_as_state_database.rst]`.

## Decision
Use **LevelDB** as the peer state database for the prototype.

## Alternatives considered
| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. LevelDB** | Fastest key/composite-key lookups; embedded (no extra service to deploy/secure); sufficient because anchor reads are by key | No rich JSON/range queries | **Chosen** |
| B. CouchDB | Rich JSON queries + indexes on anchor metadata | Adds an external DB to operate, back up, and secure; unnecessary for key-addressed access | Deferred |

## Consequences
- **Positive:** simplest, fastest path for the prototype's key-addressed audit lookups; smaller attack
  surface (no external DB).
- **Trade-off:** no server-side rich queries — acceptable because audit retrieval is by composite key
  (employee + change sequence). If G-08 later reveals a rich-query requirement (e.g. "all anchors for
  a company in a date range" without a key scan), **reopen this ADR and switch to CouchDB**.
- **Follow-up:** the composite-key access pattern is specified in the G5 data model (ADR-0001); the
  performance test plan (G7) validates that key-addressed reads meet the (assumed low) volume targets.

## Related
- Relates to: **ADR-0001** (composite-key anchor model), **ADR-0004** (single channel). 
- Knowledge-graph: D11. Grounding gap: **G-08**. Context: `context/FABRIC-WORLD-STATE.md`.

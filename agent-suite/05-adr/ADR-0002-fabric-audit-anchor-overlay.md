# ADR-0002: Fabric is an audit/anchor overlay; MySQL remains system-of-record

- **Status:** **Accepted**
- **Date:** 2026-07-13
- **Deciders:** fabric-engineer, solution-architect
- **Rests on assumption(s):** G-09 (Fabric is a pure audit/anchor overlay, not source-of-truth). Ledger immutability/audit grounded in `[docs: ledger/ledger.md]`.

## Context
The Talenta HRIS already has a system-of-record: the shared MySQL owned by talenta-core (PHP/Yii2), with EMS (Go) as a strangler-fig extraction reading/writing the **same** schema `[code: ems/dbconfig.yml]`. PII is encrypted at rest there with a versioned key model, and the DDL is owned by talenta-core — the prototype may not alter the shared schema (**G-16**). The question this ADR settles: does Fabric become authoritative for any field, or is it purely an overlay?

Fabric's value here is its immutable, ordered, replicated history — the audit guarantee (**D12**) `[docs: ledger/ledger.md]`. That guarantee is exactly what an *anchor/overlay* needs and does not require Fabric to own any authoritative data. Making Fabric authoritative for a field, by contrast, would put it on the request path of the system-of-record and force a dual-write across two stores that are not in one distributed transaction.

Relevant capability: **D12** (ledger immutability & audit). See context `context/BLOCKCHAIN-INTEGRATION.md` (the integration seam, call path, and failure modes) and knowledge-graph Layer C.

## Decision
We will treat **Fabric strictly as an audit/anchor overlay**: the existing AES-encrypted MySQL (`tbl_user` et al.) remains the **single system-of-record** for every PII field, and **zero authoritative fields live on Fabric.** The ledger is written *after* the MySQL write is committed and is only ever read to **verify** a change (recompute-and-compare, ADR-0001), never to **reconstruct** a value. The two stores are deliberately **not** in one distributed transaction; the seam is eventually consistent and reconcilable (persist salt/commitment → submit → confirm `CommitStatus` → commit Kafka offset; at-least-once with idempotent composite keys).

**Numeric constraints (set by this ADR):**
- Authoritative fields on Fabric: **0**.
- Shared-MySQL schema changes introduced by the prototype: **0** (G-16) — any new state lives on-ledger or in the anchor-service's own store.
- Distributed (2-phase) transactions across MySQL and Fabric: **0** — reconciliation is by a periodic MySQL-change-log ↔ on-chain-anchor diff job, not by a coordinator.

## Alternatives considered
| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. Audit/anchor overlay (chosen)** | Ledger dependency + its failure modes are isolated from the system-of-record request path; approval flow never blocks on the ledger; no migration; matches the "no schema change" constraint (G-16); Fabric contributes exactly what it is good at (immutable audit, D12) | Eventual consistency between MySQL and ledger; needs a reconciliation job to close the dual-write gap; Fabric is not directly queryable as a data source | **Chosen** |
| B. Fabric as source-of-truth for some fields | "One authoritative place" for those fields; on-chain query without a second store | Rejected — forces **dual-write consistency** across two non-transactional stores, a **data-migration** of live PII out of MySQL, and puts ledger latency/availability on the write path of the HRIS; delivers **no prototype benefit** over the overlay for a tamper-evidence demo | Rejected |
| C. Replace MySQL with Fabric entirely | Single store | Rejected — Fabric is not a general-purpose PII datastore (confidentiality, erasure, and rich query would all regress); enormous migration risk; out of scope for a pilot | Rejected |

## Consequences
- **Positive:** the system-of-record and its proven AES-at-rest crypto are untouched; the ledger's failure modes (orderer down, endorsement timeout) degrade only *anchor latency*, never the HRIS; clean separation makes ADR-0001 (commitments only) and ADR-0006 (a separate host service) natural; no shared-schema risk.
- **Negative / trade-off:** the design must own an eventual-consistency reconciliation path (a periodic job comparing `tbl_employee_data_request` to on-chain anchors, re-driving any unanchored approved change from Kafka); "verify, not reconstruct" means the ledger is useless without the off-chain companion store.
- **Follow-ups:** append prototype-phase risk rows for **Kafka→gateway seam failure/eventual-consistency drift** and **salt/companion-store loss** to `10-risk/risk-register.md` at G3+; the reconciliation job is a build task; failure-mode table is maintained in `context/BLOCKCHAIN-INTEGRATION.md` §5.

## Related
- Relates to: **ADR-0001** (only commitments cross to the ledger), **ADR-0006** (the overlay is hosted in a separate Go service, reinforcing isolation), **ADR-0010** (erasure on an overlay = crypto-shred, no source-of-truth rewrite).
- Knowledge-graph: Layer C (services & seam), Layer D capability **D12**, Layer E frame **CIA-Integrity (E1)**; traceability spine chain #1 (anchor). Grounding gap(s): **G-09** (and G-16). Context doc(s): `context/BLOCKCHAIN-INTEGRATION.md`.

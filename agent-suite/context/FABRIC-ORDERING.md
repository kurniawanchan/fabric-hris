# Fabric Ordering Service (Raft) — context stub

**What it is.** The ordering service establishes total transaction order and cuts blocks; it does not
execute chaincode or hold world state. Fabric 2.x uses **Raft** (etcd/raft) — a crash-fault-tolerant,
leader-follower consensus where each orderer node keeps a replicated log; a cluster of N nodes
tolerates the loss of (N-1)/2. Block cutting is governed by `BatchTimeout` / `BatchSize`, and channel
membership of orderers is managed via channel config.

> **Depth: see fabric-network-architect/references/ordering-design.md — do not restate here.** It owns
> Raft node sizing, fault tolerance, orderer-org topology, and block-cutting parameters
> (operational deploy/monitor bits cross-ref fabric-operations; block tuning cross-refs
> fabric-performance/references/block-cutting.md).

**Corpus citations (verified).**
- `[docs: orderer/ordering_service.md]` — ordering-service role, Raft consensus, block cutting
- `[docs: raft_configuration.md]` — Raft cluster / consenter configuration

**HRIS hooks (knowledge-graph Layer D).**
- **D12** (ledger immutability & audit) — Raft replication of the ordered log provides the
  Availability leg of the CIA triad (frame E1) for the audit ledger.
- Gap **G-04** — closed (ADR-0012): **3-node Raft, all platform-org (`Org1`) operated** — there is no
  separate ordering org. Quorum = majority = 2 of 3; crash-fault tolerance `f = 1`. This concentrates
  ordering availability in one org — an accepted, disclosed residual risk (ADR-0012 Consequences),
  not an open gap.

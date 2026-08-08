# Fabric Channels — context stub

**What it is.** A channel is a private communication subnet between a specific set of network members,
with its own ledger, chaincode, and configuration; peers only see the data of channels they have
joined. Channels are Fabric's coarse-grained **hard isolation** boundary — a separate ledger per
channel — as distinct from private data collections, which give field-level confidentiality *within*
a shared channel (see FABRIC-PRIVATE-DATA).

> **Depth: see fabric-network-architect/references/channel-strategy.md — do not restate here.** It
> owns channel-vs-PDC trade-offs, channel-per-tenant sizing, and channel config/policy design
> (cross-refs create_channel/* and configtx in fabric-operations).

**Corpus citations (verified).**
- `[docs: channels.rst]` — channel concept, per-channel ledger isolation and membership

**HRIS hooks (knowledge-graph Layer D).**
- **D7** (channels for tenant / data isolation) — the ratified mechanism for multi-tenant PII
  isolation on the ledger: **channel-per-tenant**, one channel per onboarded tenant (ADR-0013). No
  Private Data Collection is used anywhere in this design (ADR-0015) — single-channel + PDC is not a
  live alternative here.
- Gap **G-03** — closed: channel-per-tenant; world-state keys drop the old `companyId` prefix. The
  residual open question is narrower — whether multiple simultaneous tenants share one client org or
  each gets its own `OrgClient-<tenantID>` — low priority for a single-pilot-tenant prototype.

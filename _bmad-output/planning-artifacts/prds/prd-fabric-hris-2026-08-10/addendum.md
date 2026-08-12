# Addendum: Caliper Performance-Benchmark GUI Dashboard (PRD)

Technical-how depth for whoever picks this up next (architecture phase), kept out of `prd.md`
per this PRD's capabilities-not-implementation discipline. For the *product* rationale behind this
tool (reference-repo research, rejected scope options, DSRM citation), see the upstream brief's own
addendum: `_bmad-output/planning-artifacts/briefs/brief-fabric-hris-2026-08-10/addendum.md` — not
re-copied here.

## Candidate live-data transport mechanisms (Open Question 1)

Not decided — flagged for architecture. Three real options for FR-1, FR-2, FR-4, FR-5, FR-7:

- **Polling** — simplest to build against zero existing frontend infra; the dashboard's backend
  polls Caliper's output files / operations endpoints on an interval and the frontend polls the
  backend. Lowest engineering risk given the two-week window; some inherent lag.
- **Server-Sent Events (SSE)** — one-directional push, simpler than WebSocket, good fit since the
  dashboard only ever needs server → browser updates (no browser → server live channel needed for
  any FR in this PRD, since Configuration (FR-8) is read-only).
- **WebSocket** — full duplex, more complex than this product needs given no FR requires
  browser-to-server live messages.

**Leaning:** SSE or polling, given nothing here requires bidirectional live communication.

## Concrete data sources per feature

- **Dashboard (FR-1, FR-2):** `qa-tests/performance/results/raw-latencies/*.jsonl` (per-round raw
  latency records, already written by every live run) and/or tailing the same
  `npx caliper launch manager` process's own report generation.
- **Network Profile (FR-3, FR-4, FR-5):**
  - *Topology (FR-3):* static topology from `fabric-network/network/configtx/configtx.yaml`'s
    `TenantChannelGenesis` profile only (`Org1`, `OrgClient-tenant01`, `Org3`) — hand-modeled per the
    PRD's Assumption in §4.2/§9, not auto-discovered. `Tenant02ChannelGenesis`
    (`OrgClient-tenant02`/`peer0.tenant02`) is a second profile in the same file and a real running
    container in `network-docker-compose.yaml` — deliberately excluded per §4.2/§9/§12 of `prd.md`
    (found during the PRD's reviewer pass, not the brief).
  - *Live metrics (FR-4, FR-5):* each node's own Prometheus-format `/metrics` HTTP endpoint.
    Ports, per `network-docker-compose.yaml` (cited in `grounding-gaps.md` `G-37`): peers
    `9444`/`9445`/`11446`/`9446`/`9447`; orderers `7071`/`8070`/`9070`.
  - *Carried-forward disclosure from G-37:* these operations endpoints currently run with **no**
    TLS/client-cert auth — reachable in plaintext. Not a new risk this dashboard introduces (the
    endpoints are already reachable today, dashboard or not), but architecture should decide whether
    the dashboard's read-only consumption of them needs any safeguard of its own, or whether
    "localhost-only, defense-day-only" is a sufficient boundary given the tool's lifespan.
- **Table List (FR-6):** `qa-tests/performance/RESULTS.md`, `QA6-RESULTS.md` — parsed/rendered as
  static content, read once.
- **Notifications (FR-7):** the same `npx caliper launch manager ... --caliper-flow-only-test`
  process's stdout/stderr (`docs/REPRODUCING-RESULTS.md`'s documented invocation).
- **Configuration (FR-8):** `qa-tests/performance/benchconfigs/*.yaml` — the actual file, read and
  displayed verbatim.

## Follow-up not done in this PRD

- Formalizing the "no frontend stack" exception (§10 of `prd.md`) as an ADR note or a
  `grounding-gaps.md` row — a candidate action for whoever owns this once it moves to architecture.

---
title: Reconcile — ARCHITECTURE-SPINE.md vs UX spines (DESIGN.md / EXPERIENCE.md)
status: final
created: '2026-08-11'
---

# Reconcile: Architecture Spine vs UX Spines

**Checked against:**
- `_bmad-output/planning-artifacts/ux-designs/ux-fabric-hris-2026-08-10/DESIGN.md`
- `_bmad-output/planning-artifacts/ux-designs/ux-fabric-hris-2026-08-10/EXPERIENCE.md`
- `_bmad-output/planning-artifacts/architecture/architecture-fabric-hris-2026-08-11/ARCHITECTURE-SPINE.md`

## Gaps found

### 1. Three-tier node status (Healthy/Slow/Unreachable) has no home in the data contract

DESIGN.md's status-badge component and EXPERIENCE.md's State Patterns both commit to three node
states, with the amber "Slow" tier explicitly called out as a deliberate addition beyond the PRD's
binary FR-4 (`[ASSUMPTION]` in DESIGN.md, restated in EXPERIENCE.md). The architecture spine's
`node-status` SSE event (AD-4) is defined only as a `type` discriminant string — the spine never
states the status field's value set. Nothing in AD-4, the Consistency Conventions table, or the
Capability→Architecture Map for FR-4 says the payload must carry three states rather than two.
Since AD-4 is the only place a builder would look for the event shape, a literal binary
healthy/down implementation satisfies every rule in the spine as written, silently dropping the
UX's Slow tier. The spine's own Deferred section only defers the Slow *threshold value* — it
never confirms the Slow *state* itself is part of the contract.

### 2. Slow vs. Down resource-tile semantics collide with AD-6's uniform null-on-failure rule

EXPERIENCE.md is explicit that a Slow node keeps showing live resource numbers (it's "still
responding, just slowly"), while a Down node's resource tile shows "—" instead of a stale number.
That means the resource-usage data point needs (at least) two distinct "not a normal live number"
representations tied to *why* it isn't normal. AD-6 specifies exactly one degraded representation
for any failing source: "an explicit `null`/unavailable value." Applied literally to node
resource-usage, a Slow node's slow-but-real poll response is not a failure at all (AD-6 doesn't
apply), but the spine never says so — it never distinguishes "poll succeeded, values are real but
node is slow" from "poll failed/timed out, value is null" as two different code paths feeding the
same `node-status`/resource event. A builder following AD-6's single null convention could easily
null out a Slow node's resource tile too, contradicting the UX requirement that Slow nodes keep
their live numbers.

### 3. Mode indicator (Idle/Live) is not represented anywhere in the backend's state or contract

EXPERIENCE.md/DESIGN.md make the Idle/Live mode indicator persistent chrome on every tab, driven
by whether a benchmark is actually running. AD-5's singleton backend state is defined as exactly
one thing: "which benchconfig is selected, which file paths are being watched." There is no
running/not-running flag anywhere in AD-5, no corresponding SSE event type in AD-4's three-value
`type` enum (`metric-update` | `node-status` | `log-line`), and no REST route for it in the
Structural Seed's `routes/` list. A scenario being *selected* (AD-5's state) is explicitly not the
same thing as a benchmark being *live* (EXPERIENCE.md's Interaction Primitives calls this out:
selecting a scenario "does not restart or affect an in-progress live run," and first-load default
selection is "distinct from the already-defined Idle state"). The architecture gives the frontend
no field to derive Idle vs. Live from at all.

### 4. No write path for the persistent benchmark-scenario selector

EXPERIENCE.md's persistent chrome includes an interactive selector for which `benchconfigs/*.yaml`
run is active, and AD-5 says the backend "holds exactly one mutable 'current scenario' state."
But every route the spine actually lists (`/api/scenarios`, `/api/history`, `/api/config`) is
described as read-only/static ("Static/read-once data... is served over plain REST"), and the
Capability→Architecture Map's FR rows likewise show only reads. Nothing in AD-4, the Consistency
Conventions REST-naming row, or the Structural Seed names the endpoint that actually mutates
AD-5's "current scenario" state when the operator changes the selector. The state AD-5 promises is
mutable, but the contract never says how it gets mutated.

### 5. Staleness-muting has no signal to key off

EXPERIENCE.md requires a metric tile to visually mute if "no update has arrived within roughly
FR-1's 'a few seconds'" — i.e., the frontend must detect *silence*, not just an explicit failure
value. AD-6 only defines what the backend pushes when a source *actively fails* (an explicit
null); it says nothing about a heartbeat, keep-alive, or periodic tick on the one SSE channel that
would let the frontend distinguish "source is fine but nothing changed" from "connection is dead"
within the same few-second window the UX requires. Without such a signal in the contract, a
literal implementation of AD-4/AD-6 gives the frontend no reliable way to satisfy the
staleness-muting requirement — it would have to guess a timeout against a channel that offers no
guaranteed cadence.

## Not gaps (checked, spine covers these adequately)

- FR-2's "zero renders as literal 0" is fine under the spine's data convention: explicit JSON
  `null` (unavailable) is already distinguished from a numeric `0`, so the contract has room for
  both without confusion.
- The tenant02 exclusion (AD-7) matches EXPERIENCE.md's standing guardrail exactly, including the
  "requires a PRD update, not a code judgment call" framing.

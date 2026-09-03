---
stepsCompleted: ['requirements-extraction', 'user-confirmed', 'epic-design', 'story-generation', 'final-validation']
inputDocuments:
  - '_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-10/prd.md'
  - '_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-10/addendum.md'
  - '_bmad-output/planning-artifacts/architecture/architecture-fabric-hris-2026-08-11/ARCHITECTURE-SPINE.md'
  - '_bmad-output/planning-artifacts/ux-designs/ux-fabric-hris-2026-08-10/DESIGN.md'
  - '_bmad-output/planning-artifacts/ux-designs/ux-fabric-hris-2026-08-10/EXPERIENCE.md'
---

# Caliper Performance-Benchmark GUI Dashboard - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for the Caliper Performance-Benchmark
GUI Dashboard — a bounded, thesis-defense demonstration tool (single operator, ~2-week build) —
decomposing the requirements from its PRD, UX design contract, and architecture spine into
implementable stories.

## Requirements Inventory

### Functional Requirements

FR1: The system shows transaction and read throughput, and latency (min/max/average), updating continuously while a benchmark round is executing.
FR2: The system shows the running count of successful vs. failed transactions/reads for the in-progress round.
FR3: The system displays the network's peers, orderers, organizations, and channel membership as a diagram, scoped to the active benchmark channel's topology only (the three `TenantChannelGenesis` orgs — never `OrgClient-tenant02`/`peer0.tenant02`).
FR4: The system shows each peer's and orderer's live reachability/health status, sourced from that node's own operations endpoint, for the same FR3-scoped node set only.
FR5: The system shows each node's current CPU and memory usage, sourced from that node's own operations endpoint, for the same FR3-scoped node set only.
FR6: The system displays a list of the project's already-collected benchmark runs (PT-1/PT-2 write, PT-3 read, PT-5 bulk comparison, QA-6 MVCC contention), with each run's key headline numbers visible without navigating elsewhere.
FR7: While a benchmark is running, the system displays that run's Caliper CLI process output (stdout/stderr) as it's produced.
FR8: The system displays the actual benchmark config YAML driving the currently selected or in-progress run, read-only.

### NonFunctional Requirements

NFR1: Live throughput/latency/success-rate updates must not visibly lag more than a few seconds behind the underlying benchmark process.
NFR2: Node health/resource monitoring must not require any new instrumentation added to the live network's peer/orderer containers — it reads only from operations endpoints already reachable.
NFR3: The dashboard must never crash or drop its live data stream because one data source (a timed-out operations-endpoint poll, a not-yet-created log file) failed — that one data point degrades to explicit "unavailable," everything else keeps flowing.
NFR4: The backend's HTTP/SSE server must bind to `127.0.0.1` only, never a real network interface — the operations endpoints it polls carry no TLS/auth of their own (grounding-gaps.md G-37).
NFR5: On the actual defense day, both processes must run in production/built mode, never dev/watch mode — an unrelated file change or watcher hiccup must not be able to drop the live connection mid-demo.
NFR6: No content this product passes through (CLI stdout, historical run data, benchmark config YAML) may introduce a real company/product name that wasn't already present in its source file — this product performs no scrubbing of its own and must not be the first thing to leak one.

### Additional Requirements

- **Starter template (flags Epic 1 Story 1):** frontend is scaffolded from shadcn/ui's own CLI v4 Vite template (`npx shadcn@latest init -t vite`) — React 19.2.7 + Vite 8.2.x + TypeScript + Tailwind CSS, shadcn's own default project structure. Backend has no starter template — a plain Express 5.2.1 app on Node.js 24.x LTS, hand-structured per the architecture spine's Structural Seed.
- **New top-level repo directory:** `caliper-dashboard/` (with `backend/` and `frontend/` subdirectories) — a disclosed, one-off exception to `project-context.md`'s "no frontend stack" rule, scoped to this artifact only.
- **Two-process topology:** backend is a standalone, always-running Node process; frontend is a separate build. Frontend never imports backend code — only the documented HTTP/SSE contract crosses the boundary (AD-1, AD-2).
- **Caliper observation is file-watching only:** the backend never spawns the Caliper CLI. The operator's terminal invocation redirects stdout to a fixed log path (`... | tee caliper-run.log`); the backend tails that file and watches `qa-tests/performance/results/raw-latencies/*.jsonl` (AD-3).
- **One relay channel, literal event schema:** exactly one SSE endpoint carrying four exact event shapes (`metric-update`, `node-status`, `log-line`, `heartbeat`), each with `context: {scenario, isLive}` always present. REST for everything static (`/api/scenarios`, `/api/topology`, `/api/history`, `/api/config`) plus the one mutating action (`POST /api/scenario`) (AD-4).
- **Backend singleton state:** exactly two mutable singletons — selected scenario and is-live — independent axes; is-live flips on the first successfully parsed data record, never on file-open (AD-5).
- **Streaming resilience:** every source read/poll isolated so its own failure degrades only that data point; heartbeat event on a fixed interval reports per-source liveness; no reconnect-replay buffer (AD-6).
- **Canonical node identity:** `backend/topology.ts` is the single source of every node's `nodeId` — the poller imports and polls/emits by that id, never inventing its own hostname-based scheme (AD-7).
- **Passthrough-content confidentiality:** this product adds no filtering of its own to FR6/FR7/FR8's real file content — safety rests entirely on those source files already being generic-register-clean, per this repo's existing discipline (Consistency Conventions).

### UX Design Requirements

UX-DR1: Implement the design-token set from `DESIGN.md` — navy primary (`#1E3A5F` light / `#7FA8D9` dark), three semantic status colors (healthy/warning/down, each with a verified dark-mode pair), and three typography roles (`metric` 32px/600, `metric-label`, `data-mono`) layered on shadcn/ui's own inherited defaults.
UX-DR2: Build the **status badge** component — pill-shaped, exactly three states (Healthy/Slow/Unreachable), always icon *and* color *and* text label together, never color alone.
UX-DR3: Build the **mode indicator** component — a persistent, always-visible pill in the same position across all five tabs, reading "Live" or "Idle" only (no third state; the recorded fallback is a separate video played entirely outside this product — never an in-app mode).
UX-DR4: Build the **metric tile** component — value updates in place with no layout shift; a value of zero renders as a literal "0", never a blank field; the tile visually mutes if no update arrives within the NFR1 window, rather than silently showing an increasingly stale number.
UX-DR5: Build the **topology diagram** component — nodes as cards (orderers as circles), grouped into three columns by org, the shared orderer set shown once rather than duplicated per column, each node carrying its own status badge positioned top-right.
UX-DR6: Build the **code block** component — monospace, muted background, read-only with no edit affordance of any kind (no cursor, no focus ring on click), for the Configuration tab's YAML display.
UX-DR7: Build the **log line** component — monospace, live-tailing Notifications feed; new lines append and auto-scroll, pausing auto-scroll if the operator has manually scrolled up to read history.
UX-DR8: Implement the five-tab Information Architecture (Dashboard, Network Profile, Table List, Notifications, Configuration — in that priority order) with persistent chrome: the mode indicator and a benchmark-scenario selector (dropdown), the latter defaulting to the first `benchconfigs/` entry alphabetically on first load.
UX-DR9: Implement the Voice and Tone discipline — plain, precise, unembellished status text ("Committed" not "Success! 🎉"); calm, factual empty/idle states; and the generic-register constraint applied to every piece of UI copy written for this product (not the passthrough content itself, which NFR6 already governs).
UX-DR10: Implement the Accessibility Floor — viewing-distance legibility as the primary concern (verify the 32px metric type and status badges against the actual defense room before finalizing, not just on a laptop up close); status never conveyed by color alone; standard shadcn/Radix keyboard-navigability and focus-visible behavior inherited as-is.
UX-DR11: Implement Key Flow UJ-1 end-to-end — the live-benchmark defense-day flow, including the "on request" beat (tab-switching to Table List/Notifications/Configuration if asked) and the edge case (switching to the out-of-app recorded fallback on a stall/failure, with no in-dashboard replay mode).
UX-DR12: Implement the "Slow" node-status tier's distinct data treatment — amber styling, a *live* (not muted) resource-usage tile, keyed on operations-endpoint poll response latency specifically (per the architecture spine's AD-7) — distinct from "Node down"'s "—" treatment.

### FR Coverage Map

FR1: Epic 1 - Live throughput/latency display (Dashboard)
FR2: Epic 1 - Live success/failure rate (Dashboard)
FR3: Epic 1 - Topology visualization, 3-org scoped (Network Profile)
FR4: Epic 1 - Live per-node health status (Network Profile)
FR5: Epic 1 - Live per-node resource usage (Network Profile)
FR6: Epic 2 - Historical run browsing (Table List)
FR7: Epic 2 - Live CLI output tail (Notifications)
FR8: Epic 2 - Read-only active benchmark config (Configuration)
NFR1: Epic 1 - Live-update lag threshold
NFR2: Epic 1 - No new Fabric instrumentation
NFR3: Epic 1 - Per-source failure isolation
NFR4: Epic 1 - Localhost-only binding
NFR5: Epic 1 - Production-mode discipline on defense day
NFR6: Epic 2 - Passthrough-content confidentiality
UX-DR1: Epic 1 - Design tokens
UX-DR2: Epic 1 - Status badge component
UX-DR3: Epic 1 - Mode indicator component
UX-DR4: Epic 1 - Metric tile component
UX-DR5: Epic 1 - Topology diagram component
UX-DR6: Epic 2 - Code block component
UX-DR7: Epic 2 - Log line component
UX-DR8: Split - Epic 1 ships Dashboard/Network Profile tabs + persistent chrome; Epic 2 completes the remaining three tabs
UX-DR9: Epic 1 - Voice and tone discipline
UX-DR10: Epic 1 - Accessibility floor
UX-DR11: Epic 2 - Full UJ-1 flow (on-request tour + recorded-fallback edge case)
UX-DR12: Epic 1 - "Slow" node-status tier treatment

## Epic List

### Epic 1: Live Benchmark Monitoring
The operator can watch a real benchmark execute — throughput/latency numbers changing live, and
the network's own topology with per-node health/resource — the two "fully live" tabs that carry
the whole demonstration's weight. Includes standing up the project itself (frontend scaffolded
from shadcn's Vite template, backend as a plain Express app) and the entire relay substrate (the
one SSE channel, its full 4-shape event schema, backend singleton state, streaming resilience,
localhost-only binding) since both live tabs need it. Standalone: proves the core "is this really
live" premise end-to-end on its own.
**FRs covered:** FR1, FR2, FR3, FR4, FR5 (+ NFR1-5, UX-DR1-5, UX-DR9, UX-DR10, UX-DR12)

### Epic 2: Historical Context & Live Transparency
The operator can show, on request, the already-collected historical runs, the live CLI output
proving the demo isn't staged, and the exact config driving the run — the three "presence over
depth" tabs, plus the full defense-day flow (the on-request tour and the recorded-fallback edge
case) tying all five tabs together. Builds on Epic 1's relay substrate (one more event type,
`log-line`, and three REST routes) but Epic 1 does not depend on Epic 2 to function.
**FRs covered:** FR6, FR7, FR8 (+ NFR6, UX-DR6, UX-DR7, UX-DR11)

## Epic 1: Live Benchmark Monitoring

The operator can watch a real benchmark execute — throughput/latency numbers changing live, and
the network's own topology with per-node health/resource — the two "fully live" tabs that carry
the whole demonstration's weight. Includes standing up the project itself and the entire relay
substrate. Standalone: proves the core "is this really live" premise end-to-end on its own.

### Story 1.1: Project & Relay Foundation

As the operator,
I want the dashboard's two processes scaffolded and running, with the persistent chrome (mode
indicator, scenario selector) working end-to-end,
So that every later feature has a correctly-configured, already-running base to build on.

**Acceptance Criteria:**

**Given** a fresh checkout of `caliper-dashboard/`
**When** I run the project's start script
**Then** the backend starts as a standalone Node/Express process bound to `127.0.0.1` only (AD-8),
and the frontend starts as a separate Vite+React+shadcn app (scaffolded via
`npx shadcn@latest init -t vite`) — two processes, one command

**Given** the app is running with no benchmark active
**When** I open the dashboard in a browser
**Then** the mode indicator reads "Idle" and the scenario selector defaults to the first
`benchconfigs/` entry alphabetically

**Given** I select a different scenario from the dropdown
**When** the selection changes
**Then** `POST /api/scenario` updates the backend's singleton state, and `GET /api/scenario`
reflects the new selection immediately — with no effect on any in-progress run (there is none yet)

**Given** the backend is running
**When** I inspect its network bindings
**Then** it is reachable only via `localhost`/`127.0.0.1` — never `0.0.0.0` (NFR4/AD-8)

### Story 1.2: Live Benchmark Metrics Display

As the operator,
I want the Dashboard tab to show live throughput, latency, and success/failure counts while a
benchmark runs,
So that I can prove to the committee, live, that a real benchmark is executing.

**Acceptance Criteria:**

**Given** I've redirected a running Caliper benchmark's stdout to the configured log path and
it's actively writing to `results/raw-latencies/*.jsonl`
**When** the backend's watcher parses the first real data record
**Then** the singleton `isLive` flag flips `true` (never on mere file-open) and the mode
indicator flips to "Live" (FR1, FR2, AD-5)

**Given** a benchmark round is executing
**When** new throughput/latency/success/failure data is parsed
**Then** the Dashboard's metric tiles update in place within a few seconds, with no page reload
(NFR1) — a round with zero failures shows a literal "0", never a blank field

**Given** no new data has arrived for longer than the staleness window
**When** the heartbeat event's `sources.jsonl` timestamp goes stale
**Then** the affected metric tile visually mutes rather than silently showing an old number as
current

**Given** the raw-latencies watcher itself fails (e.g. the file briefly disappears)
**When** that failure occurs
**Then** only the Dashboard's data degrades to unavailable — the backend process and every other
feed keep running (NFR3)

### Story 1.3: Network Topology Visualization

As the operator,
I want to see the network's three-org topology as a diagram,
So that I can visually reinforce the tamper-detection premise (an independently-operated client
org) while narrating the defense.

**Acceptance Criteria:**

**Given** `backend/topology.ts`'s hand-modeled data
**When** the frontend requests `GET /api/topology`
**Then** it returns exactly the three `TenantChannelGenesis` orgs (`Org1`, `OrgClient-tenant01`,
`Org3`) and their peers/orderers — never `OrgClient-tenant02`/`peer0.tenant02`, even though that
container is running (FR3, AD-7)

**Given** the Network Profile tab is open
**When** the topology loads
**Then** nodes render as cards grouped into three org columns (orderers as circles, per
DESIGN.md), with the shared 3-node orderer set shown once, not duplicated per column

### Story 1.4: Live Node Health & Resource Monitoring

As the operator,
I want each node's live health and resource usage shown on the topology diagram,
So that the committee sees the network's actual condition, not a static picture, while the
benchmark runs.

**Acceptance Criteria:**

**Given** the poller is running
**When** it polls each of the three-org node set's operations `/metrics` endpoint
**Then** it keys every poll and emitted `node-status` event by the `nodeId` `topology.ts`
assigned that node — never an independently-invented hostname scheme (AD-7)

**Given** a node responds within the healthy threshold
**When** a `node-status` event is emitted
**Then** its status badge reads "Healthy" (icon + color + text together) and `resourceUsage` is
a populated `{cpuPct, memPct}` object

**Given** a node's poll response latency crosses the "slow" threshold
**When** that's detected
**Then** its badge reads "Slow" (amber) and its resource tile keeps showing live numbers —
distinct from a down node's treatment (UX-DR12)

**Given** a node is unreachable
**When** its poll fails
**Then** its badge reads "Unreachable" and `resourceUsage` is `null` in its entirety (never an
object with null fields inside) — matching the "—" display convention, and this one node's
failure never affects any other node's status or the rest of the app (NFR2, NFR3)

## Epic 2: Historical Context & Live Transparency

The operator can show, on request, the already-collected historical runs, the live CLI output
proving the demo isn't staged, and the exact config driving the run — the three "presence over
depth" tabs, plus the full defense-day flow tying all five tabs together. Builds on Epic 1's
relay substrate (one more event type, `log-line`, and three REST routes) but Epic 1 does not
depend on Epic 2 to function.

### Story 2.1: Historical Run Browsing

As the operator,
I want to show the committee already-collected benchmark results,
So that I can point to prior evidence (PT-1 through PT-5, QA-6) without leaving the dashboard.

**Acceptance Criteria:**

**Given** `RESULTS.md`/`QA6-RESULTS.md` contain the project's already-collected runs
**When** the frontend requests `GET /api/history`
**Then** it returns pre-parsed structured JSON (`{ runs: [{id, timestamp, tps, latencyP99, ...}] }`),
never raw Markdown (FR6, AD-4)

**Given** the Table List tab is open
**When** the historical runs load
**Then** each run's key headline numbers are visible without navigating elsewhere

**Given** the backend performs no scrubbing of its own
**When** any text from `RESULTS.md`/`QA6-RESULTS.md` reaches the JSON response
**Then** it matches the source file exactly — this feature never introduces a name not already
present in the source (NFR6)

### Story 2.2: Live CLI Output Tail

As the operator,
I want the Caliper CLI's own console output shown live,
So that the committee can see the raw process output as further proof the run is real, not staged.

**Acceptance Criteria:**

**Given** Caliper's stdout is redirected to the configured log path (per Story 1.2's setup)
**When** new lines are written to that file
**Then** the Notifications tab's feed appends them live, in order, with no page reload (FR7)

**Given** I've scrolled up to read earlier output
**When** new lines continue arriving
**Then** auto-scroll pauses until I scroll back to the bottom (UX-DR7)

**Given** the log file goes silent
**When** the heartbeat's `sources.log` timestamp goes stale
**Then** the Notifications feed's staleness is indicated independently of the other two feeds
(AD-6, NFR3)

### Story 2.3: Read-Only Config Display

As the operator,
I want the active benchmark's exact config shown, unmodifiable,
So that I can prove the demo isn't rigged without risking a live edit mid-defense.

**Acceptance Criteria:**

**Given** a scenario is selected
**When** the Configuration tab is opened
**Then** `GET /api/config` returns that scenario's actual benchconfig YAML as raw text, displayed
byte-for-byte — never a paraphrase or summary (FR8, AD-4)

**Given** the Configuration tab is open
**When** I click or focus the displayed YAML
**Then** there is no edit affordance of any kind — no cursor, no focus ring — reinforcing the
read-only contract visually (UX-DR6)

### Story 2.4: End-to-End Defense Flow

As the operator,
I want the full five-tab flow rehearsed and working together,
So that I have a real, working demo — and a real recorded fallback — ready for the defense.

**Acceptance Criteria:**

**Given** a live benchmark is running with Dashboard and Network Profile both showing live data
**When** asked to show something else
**Then** I can switch to Table List, Notifications, or Configuration and show real data on each,
without disrupting the live run on the other two tabs (UX-DR11)

**Given** the live system stalls or a node goes unreachable mid-demo
**When** I switch to the recorded fallback
**Then** I do so entirely outside the dashboard — a separate video player — the dashboard itself
never enters an in-app "replaying" mode

**Given** all five tabs are functioning
**When** I run one full rehearsal end-to-end
**Then** every tab shows real, unmocked data (no manual data-faking), and that same rehearsal
recording becomes the defense-day fallback

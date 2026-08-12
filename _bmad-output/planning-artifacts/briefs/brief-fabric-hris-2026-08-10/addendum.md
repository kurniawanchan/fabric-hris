# Addendum: Caliper Performance-Benchmark GUI Dashboard

Depth captured during the brief conversation that doesn't belong in the 1-2 page brief itself, but
matters for whoever picks this up next (a PRD, an architecture doc, or Mr. Chan re-reading this in
a week).

## Reference-repo research: why it's inspiration, not a base to build on

`github.com/Jasonyou1995/caliper-gui-dashboard` was checked directly (repo metadata, README,
`package.json`, and file tree — not from recollection) before any brief conversation started:

- It's a React front-end only (Creative Tim's "paper-dashboard-react" template). There is **no**
  `server/`, `api/`, or backend directory anywhere in the repo.
- The README describes an Express.js + MongoDB layer that would bridge to Caliper's CLI, submit
  configs, and persist historical runs — none of that exists in the actual file tree. Every feature
  bullet in the README's structure section is an unchecked `[ ]` checkbox.
- View components (checked `DashboardVisualization.jsx` directly) are purely presentational — they
  render whatever chart data is passed in via props. No `fetch`/`axios`/API call exists anywhere in
  `src/`.
- `package.json` describes the project as "Production-ready... with real-time visualization and
  advanced analytics" (v1.2.0) — this does not match the actual file tree. The repo also carries
  `.roomodes`/`.windsurfrules` (AI-coding-assistant config), consistent with cosmetic polish from an
  AI assistant, without the real backend integration ever landing.
- 5 stars, last pushed 2025-06-17, single author, 2019 copyright notice on the original template.

**Conclusion carried into the brief:** the only thing worth reusing is the *tab layout concept*
(Dashboard / Notifications / Network Profile / Table List / Configuration). Everything behind it —
the live-data wiring, the topology visualization, the CLI-tail — has to be built fresh against this
project's real outputs.

## Options considered and why they were rejected

**Retrospective/static dashboard instead of live.** Initially recommended by the coach as the
lower-risk default (a thesis PDF can only embed static figures anyway). Rejected by Mr. Chan in
favor of live, on the grounds that DSRM's Demonstration activity (Phase 4) specifically calls for
showing the artifact *functioning* in a realistic setting — a live run is stronger evidence for a
systems thesis than static charts, provided the risk is genuinely mitigated — and it is: a recorded
run from a prior successful pass exists as the rehearsed fallback.

**Full parity across all five areas, fully live and interactive.** This was the scope initially
stated, before the two-week timeline was disclosed (defense target: ~2026-08-24). Once the timeline
was on the table, it was re-examined against starting from **zero existing frontend
infrastructure** in this repo — no build tooling, no dev-serve setup, nothing. Settled on tiered
depth instead (see brief's Scope): Dashboard + Network Profile fully live, the other three present
at reduced engineering weight. This keeps "all five areas exist and are demonstrable" without each
one costing a full build — the stated reason for needing all five was hedging against not knowing
what the committee will ask about, not a specific per-feature narrative need. Recorded as the
stated rationale; not pressed further after two rounds of pushback, since this reads as a
reasonable defense-prep instinct rather than an unexamined assumption.

## Technical grounding for feasibility

- **Network Profile's live topology/resource data** doesn't need new instrumentation: Fabric's
  peer/orderer operations service already exposes Prometheus-format `/metrics` on its operations
  port. This was confirmed reachable (if currently unauthenticated — see `grounding-gaps.md` `G-37`)
  during an unrelated audit earlier in this project. That's real CPU/memory/connection data
  available today.
- **Dashboard's live throughput/latency** has a direct source: this project's own Caliper workspace
  (`qa-tests/performance/`) already writes per-round raw latency data
  (`results/raw-latencies/*.jsonl`) and Caliper's own report — the dashboard's job is to visualize
  a run as it happens, not invent a new measurement pipeline.
- **Notifications** (live CLI tail) is a straightforward process-output stream from the same
  `npx caliper launch manager` invocation `docs/REPRODUCING-RESULTS.md` already documents — not a
  new integration surface.

## The "no frontend stack" rule

`_bmad-output/planning-artifacts/project-context.md`'s Technology Stack section states plainly:
"No frontend stack — this repo does not own UI." This brief proposes crossing that once, narrowly,
for a disclosed thesis-communication artifact — not reopening the rule generally. If this moves
past the brief stage, the exception should be recorded formally (a grounding-gap row, or a short
ADR note) rather than left as an implicit one-off. That formal recording is not done as part of
this brief.

## DSRM framing, for precision

Per `agent-suite/context/DSRM.md` (Peffers et al., 2007, as adopted by this project): Caliper
benchmarks (PT-1 through PT-5, QA-6) are explicitly mapped to **Phase 4 — Demonstration**
("Experimental (controlled experiment, simulation)... Phase-4 demonstration" row, §4). The thesis
defense itself is **Phase 6 — Communication** ("The problem, artifact, rigor and utility conveyed to
researchers + practitioners," §1). This dashboard is Demonstration-activity tooling, built
specifically to be used *during* the Communication-phase defense — it doesn't introduce a new DSRM
activity; it gives the existing Demonstration evidence a live presentation surface.

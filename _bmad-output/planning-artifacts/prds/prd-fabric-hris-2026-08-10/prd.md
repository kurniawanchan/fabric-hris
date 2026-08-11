---
title: 'PRD — Caliper Performance-Benchmark GUI Dashboard'
created: '2026-08-10'
updated: '2026-08-11'
project: fabric-hris
status: final
---

# PRD: Caliper Performance-Benchmark GUI Dashboard

## 0. Document Purpose

This PRD scopes a live, browser-based dashboard that visualizes this project's Hyperledger Caliper
performance benchmarks (PT-1 through PT-5, QA-6) while they run, for use during Mr. Chan's thesis
defense (~2026-08-24). It builds directly on the approved product brief at
`_bmad-output/planning-artifacts/briefs/brief-fabric-hris-2026-08-10/` (`brief.md` + `addendum.md`)
— that conversation's research, risk analysis, and scope decisions are not re-derived here, only
turned into requirements. Features are grouped by dashboard area; functional requirements (FR-N)
are nested under each and numbered globally. `[ASSUMPTION: ...]` tags mark anything inferred
without explicit confirmation — see §9 for the full index.

## 1. Vision

This project's Caliper benchmarks already produce real, live-measured evidence of the prototype's
performance — but that evidence today lives only in Markdown tables and a generic Caliper HTML
report, read after the fact. Per this project's own DSRM framing
(`agent-suite/context/DSRM.md`), those benchmarks are Phase 4 — Demonstration evidence; a thesis
defense is Phase 6 — Communication. This dashboard gives that existing Demonstration evidence a
live presentation surface *during* the Communication event itself: a benchmark that visibly runs,
in front of the examination committee, rather than a table they're asked to trust.

It is visually inspired by the tab layout of an open-source reference project
(`github.com/Jasonyou1995/caliper-gui-dashboard`) but not built from its code — that repo turned
out to be a UI shell with no working backend behind it (full findings in the brief's `addendum.md`).
Five areas, at deliberately tiered depth to fit a two-week build against zero existing frontend
infrastructure in this repo: Dashboard and Network Profile are fully live; Table List, Notifications,
and Configuration are present at reduced depth. All five stay present because the stated reason
for wanting them was hedging against not knowing what the committee will ask about — not a
per-feature narrative need — so depth, not presence, is what tiering trades away.

Not committed as part of this v1, but worth naming here rather than only in the brief: if this
holds up through the defense, its live-metrics plumbing (operations-endpoint polling, Caliper CLI
tailing) could become this project's lightweight go-to visualization for future benchmark runs,
rather than a one-time artifact. That's a possibility to keep open, not a plan.

## 2. Target User

### 2.1 Jobs To Be Done

- As the thesis candidate, I need to show my examination committee a *working* system, not just its
  measured output, because DSRM's Demonstration activity calls for the artifact to be used on an instance
  of the problem — a stronger form of evidence than a results table for a systems thesis.
- As the presenter, I need a rehearsed fallback so a live-demo hiccup doesn't derail my defense.
- As the thesis author, I need figures/screenshots drawn from this dashboard for the thesis
  document itself — for **thesis readers**, not just the live defense audience — distinct from the
  existing `RESULTS.md`/`QA6-RESULTS.md` tables. Validated by SM-4.

The examination committee and thesis-document readers are this product's served audience even
though neither operates it directly — Mr. Chan is the sole operator (§2.2), but the JTBDs above
exist because of what those two audiences need to see and read.

### 2.2 Non-Users (v1)

*Operators, not audience — the examination committee and thesis readers above are served (§2.1),
just not as operators.*

- **The examination committee** — they observe; they do not operate the dashboard themselves.
- **Other engineers on this project** — this is not an ongoing operational tool for the write-path
  or chaincode work; it visualizes Caliper benchmarks only.
- **Future tenants/users of the HRIS platform** — irrelevant; this tool never touches PII or
  production data of any kind.

### 2.3 Key User Journeys

Single operator, one real session shape — kept light per this being internal tooling with one
operator role.

- **UJ-1. Mr. Chan runs the live benchmark during his defense.** Mid-presentation, with the network
  already up and the dashboard open on a shared screen, he starts a rehearsed Caliper benchmark run
  from the terminal. The Dashboard tab's throughput/latency charts begin updating within seconds
  with no manual refresh; the Network Profile tab shows the three-org topology with live
  peer/orderer status. He narrates against what the committee is watching. If anything stalls or
  errors, he switches to the recorded fallback for the same benchmark and continues narrating against
  that instead. **Edge case:** the recorded fallback (Success Criteria, brief) is the resolution
  path for any failure mode here — no other in-dashboard recovery is in scope.
  `[ASSUMPTION: the recorded-fallback switchover is the ONLY in-dashboard recovery path modeled; no
  other failure-recovery UX (retry, partial-state indicators, etc.) is in scope.]`

## 3. Glossary

- **Live run** — an actual, currently-executing Caliper benchmark (`npx caliper launch manager`
  against one of `qa-tests/performance/benchconfigs/*.yaml`), as opposed to historical, already-
  collected data.
- **Tiered depth** — this PRD's scoping principle: Dashboard and Network Profile are fully live;
  Table List, Notifications, and Configuration are present at reduced depth (see §4).
- **Fully live** — updates continuously from a live run's real output, with no manual refresh.
- **Static / read-only** — renders a fixed, already-collected data source once per load; does not
  poll or stream.
- **Recorded fallback** — a screen recording of a prior successful live run, rehearsed as the
  cutover if the live system misbehaves during the defense (per the brief's Risks & Mitigations).
- **Operations endpoint** — Fabric's per-node Prometheus-format `/metrics` HTTP endpoint (confirmed
  reachable in this project's `grounding-gaps.md` `G-37`), the data source for Network Profile's
  live resource/connection data.
- **Profile section** — one of this project's five ratified anchoring units (`PERSONAL`,
  `EMPLOYMENT`, `EDUCATION`, `ADDITIONAL`, `PAYROLL`) — used here only insofar as
  `employeeprofilerecord` chaincode calls appear in benchmark output; the dashboard never surfaces
  section *content*, only benchmark metrics about the calls.

## 4. Features

### 4.1 Dashboard

**Description:** The primary, fully-live view. Shows a running benchmark's throughput, latency, and
success/failure rate as it executes, sourced from the same Caliper run producing
`results/raw-latencies/*.jsonl` today. Realizes UJ-1.

**Functional Requirements:**

#### FR-1: Live throughput and latency display

The system shows transaction and read throughput, and latency (min/max/average), updating
continuously while a benchmark round is executing. Realizes UJ-1.

**Consequences (testable):**
- Values on screen change within a few seconds of new data being produced by the underlying
  Caliper run, with no page reload or manual action.
- Values shown match the same round's data in Caliper's own generated report/raw-latencies output
  (i.e., the dashboard is a visualization layer, not an independent measurement).

#### FR-2: Live success/failure rate

The system shows the running count of successful vs. failed transactions/reads for the in-progress
round. Realizes UJ-1.

**Consequences (testable):**
- Counts update as results arrive; a round that completes with 0 failures shows an explicit "0",
  not just an implied-zero percentage or a blank field.

**Feature-specific NFRs:**
- Must not visibly lag more than a few seconds behind the underlying benchmark process —
  `[ASSUMPTION: "a few seconds" is an acceptable live-feel threshold for a defense audience; no
  hard number was specified.]`

### 4.2 Network Profile

**Description:** Fully-live visualization of the network's own topology and health while a benchmark
runs — three orgs (platform, tenant client, auditor), peers, and the 3-node Raft ordering service.
Visually reinforces the project's own tamper-detection premise (P1: the client org's independent
operation) while a benchmark exercises the network. Realizes UJ-1.

**Functional Requirements:**

#### FR-3: Topology visualization

The system displays the network's peers, orderers, organizations, and channel membership as a
diagram or equivalent structured view, **scoped to the active benchmark channel's topology only**
— the `TenantChannelGenesis` profile in `fabric-network/network/configtx/configtx.yaml`: `Org1`,
`OrgClient-tenant01`, `Org3`, plus the 3-node Raft orderer set (4 peers — 2 on `Org1`, 1 each on
`OrgClient-tenant01`/`Org3` — 3 orderers, 3 orgs; corrected from an earlier "5 peers" draft during
Story 1.3's implementation, verified against `configtx.yaml`'s own anchor-peer definitions and
`network-docker-compose.yaml`'s real container names).
`OrgClient-tenant02`/`peer0.tenant02` is **deliberately excluded**, even though it is a running
container in `network-docker-compose.yaml` (a second channel profile,
`Tenant02ChannelGenesis`, exists but plays no part in the Caliper benchmarks this dashboard
visualizes) — tenant02 is documented elsewhere in this project as deliberately abandoned, having
crashed peer containers twice before (`NET-7`, `QA-4`/`PT-4`). Surfacing it live in front of the
examination committee would contradict the network's own three-org framing this PRD and the
underlying thesis both rest on. `[ASSUMPTION: a static, correctly-labeled topology for the scoped
three orgs is sufficient — this does not need to be auto-discovered/introspected from the live
network, only accurately hand-modeled against it.]`

**Consequences (testable):**
- Every peer/orderer/org belonging to the `TenantChannelGenesis` profile appears, correctly labeled
  by org and role. `OrgClient-tenant02`/`peer0.tenant02` never appears, even though it is running.

#### FR-4: Live per-node connection/health status

The system shows each peer's and orderer's live reachability/health status, sourced from that
node's own operations endpoint, for the same FR-3-scoped node set only (tenant02's peer is out of
scope here too, for the same reason). Realizes UJ-1.

**Consequences (testable):**
- A node that is up shows as healthy, via a distinct color or icon (not merely an unlabeled state
  change); a node that is down or unreachable is distinguished the same concrete way, without a
  page reload.

#### FR-5: Live per-node resource usage

The system shows each node's current CPU and memory usage, sourced from that node's own operations
endpoint, for the same FR-3-scoped node set only.

**Consequences (testable):**
- Values change over time as load changes during a live run (not a static snapshot taken once).

**Feature-specific NFRs:**
- Reads only from operations endpoints already reachable per `G-37` — this feature must not require
  any new instrumentation added to the live network's peer/orderer containers.

### 4.3 Table List

**Description:** Static / read-only (§3), present-but-simplified per the brief's tiering decision. Renders the
already-collected historical benchmark data from `qa-tests/performance/RESULTS.md` and
`QA6-RESULTS.md` — a read-once view, not a live query or a database-backed history.

**Functional Requirements:**

#### FR-6: Historical run browsing

The system displays a list of the project's already-collected benchmark runs (PT-1/PT-2 write,
PT-3 read, PT-5 bulk comparison, QA-6 MVCC contention), with each run's key headline numbers
visible without navigating elsewhere.

**Consequences (testable):**
- Every benchmark scenario documented in `RESULTS.md`/`QA6-RESULTS.md` as of build time appears
  here; the data matches those documents (no independent recalculation).

**Out of Scope:** Export/download of the data, live re-querying as new runs are added, and any
persistence layer beyond reading the existing files — `[ASSUMPTION: no export functionality is
needed for the defense itself; noted in §8 as an open question rather than assumed as an FR.]`

### 4.4 Notifications

**Description:** Present-but-simplified per the brief's tiering decision — a live tail of the
Caliper CLI process's own console output during a run, not a polished error/warning classification
system.

**Functional Requirements:**

#### FR-7: Live CLI output tail

While a benchmark is running, the system displays that run's Caliper CLI process output (stdout/
stderr) as it's produced.

**Consequences (testable):**
- New CLI output lines appear here without a page reload, in the order Caliper produced them.

### 4.5 Configuration

**Description:** Present-but-simplified per the brief's tiering decision — static / read-only (§3), proving the
live demo isn't rigged, without a live edit-and-resubmit loop (deliberately excluded — see §5).

**Functional Requirements:**

#### FR-8: Read-only display of the active benchmark config

The system displays the actual benchmark config YAML (from `qa-tests/performance/benchconfigs/`)
driving the currently selected or in-progress run.

**Consequences (testable):**
- The displayed content matches the real file byte-for-byte, not a paraphrase or summary.

## 5. Non-Goals (Explicit)

- This is **not** an ongoing operational monitoring tool for this project — it is a bounded,
  thesis-communication artifact, not infrastructure the team maintains after the defense.
- As part of this v1, this is **not** this project's permanent frontend stack. `project-context.md`'s
  "no frontend stack" rule is crossed once, disclosed, for this artifact (see §10) — whether it
  becomes an ongoing tool afterward is an open possibility (§1 Vision), not a v1 commitment either
  way.
- **No live config editing/resubmission** — Configuration is read-only (FR-8), by deliberate
  decision during the brief conversation: it was judged the riskiest, least-justified piece of the
  reference layout's original scope for a rehearsed, not improvised, defense.
- **No new persistence layer** (database, historical-run store) — Table List reads the existing
  `RESULTS.md`/`QA6-RESULTS.md` files directly (FR-6).
- **No code reuse from the reference repo** — `github.com/Jasonyou1995/caliper-gui-dashboard` has no
  functional backend to reuse; only its tab-layout concept is carried forward.

## 6. MVP Scope

### 6.1 In Scope

- FR-1 through FR-8, at the tiered depth stated per feature.
- Runs locally against this project's real network and real Caliper workspace during the defense.
- Lives inside `fabric-hris` as a disclosed, one-off exception to the "no frontend stack" rule.

### 6.2 Out of Scope for MVP

- Export/download of historical run data (FR-6) — open question, §8.
- Live benchmark-config editing (would extend FR-8) — explicit non-goal, §5.
- Any deployment/hosting beyond the local machine used for the defense.
- Multi-user/concurrent-viewer support — single operator, single screen (§2.2).

## 7. Success Metrics

**Primary**
- **SM-1**: Dashboard (FR-1, FR-2) and Network Profile (FR-3, FR-4, FR-5) display real, unmocked
  live data during an actual full-scenario benchmark rehearsal completed before defense day.
  Validates FR-1 through FR-5. `[ASSUMPTION: operationalized against a pre-defense rehearsal rather
  than the live defense moment itself, since a metric can't be validated at the moment it's meant
  to protect — the brief's own criterion was "during the defense."]`

**Secondary**
- **SM-2**: All five areas (Dashboard, Network Profile, Table List, Notifications, Configuration)
  are reachable and demonstrable on request during the defense. Validates FR-1 through FR-8.
- **SM-3**: The recorded fallback exists, is rehearsed, and can be switched to within the defense's
  time budget if the live system falters.
- **SM-4**: The thesis document includes at least one figure/screenshot drawn directly from this
  dashboard, distinct from the existing `RESULTS.md`/`QA6-RESULTS.md` tables, as Phase 4
  Demonstration evidence for thesis readers. Validates the §2.1 thesis-readers JTBD.

**Counter-metrics (do not optimize)**
- **SM-C1**: Time spent polishing this dashboard must not come at the expense of thesis-document
  writing or defense rehearsal time. This tool exists to support the defense, not to become its own
  side-project — a two-week timeline with zero existing frontend infrastructure has no slack for
  scope creep. Counterbalances SM-1/SM-2.

## 8. Open Questions

1. **Live-data transport mechanism** (polling vs. WebSocket vs. server-sent events) for FR-1, FR-2,
   FR-4, FR-5, FR-7 — a technical-how decision, deliberately deferred to the architecture phase per
   this PRD's capabilities-not-implementation discipline.
2. **Table List export** (FR-6's Out of Scope) — undecided whether this is needed; revisit if time
   permits after the fully-live features are working.
3. **Formalizing the "no frontend stack" exception** (§10) — an ADR note or a `grounding-gaps.md`
   row is the likely form; not done as part of this PRD.
4. **Rehearsal cadence** — how many full-scenario rehearsals before defense day, and how close to
   the date the last one should land, given the two-week window. A logistics question for Mr. Chan,
   not a PRD-blocking one.

## 9. Assumptions Index

- §2.3 (UJ-1) — recorded-fallback switchover is the only in-dashboard recovery path modeled; no
  other failure-recovery UX is in scope.
- §4.1 (FR-1 NFR) — "a few seconds" of live-update lag is an acceptable threshold; no hard number
  was specified by the user.
- §4.2 (FR-3) — a static, accurately hand-modeled topology for the scoped three orgs
  (`TenantChannelGenesis`) is sufficient; auto-discovery/introspection from the live network is not
  required.
- §4.3 (FR-6 Out of Scope) — no export functionality is assumed needed for the defense itself.
- §7 (SM-1) — validated against a pre-defense rehearsal, not the live defense moment itself; see
  the inline tag for why.

---

## 10. Project & Methodology Constraints
*Invented section — this product carries two project-specific constraints the standard menu doesn't name.*

- **"No frontend stack" exception.** `project-context.md`'s Technology Stack section states "No
  frontend stack — this repo does not own UI." This PRD scopes a disclosed, bounded exception for
  this artifact only — not a reversal of that rule. See Open Question 3 for formalizing it.
- **Generic-register writing discipline.** Per `project-context.md`'s AUTHORITY NOTICE, every
  written artifact in this project stays in the generic register — this PRD (and the dashboard's own
  UI copy) must never name the real HRIS platform/company, consistent with the rest of this project's
  documentation.
- **DSRM phase alignment.** This dashboard is Demonstration-activity (Phase 4) tooling, built to be
  used during the Communication-activity (Phase 6) defense — it does not introduce a new DSRM
  activity of its own (`agent-suite/context/DSRM.md`).

## 11. Why Now

The thesis defense is fixed at roughly two weeks from this PRD's date (2026-08-10). This is the
single reason depth is tiered rather than built to full parity with the reference layout — timing
here isn't a market consideration, it's a hard, non-negotiable date.

## 12. Risks and Mitigations

- **Live-demo failure.** This exact network has already had two real incidents in this project (a
  `cryptogen` rerun destroying CA trust across three orgs; a stale genesis block after recovery —
  both in `docs/QUICKSTART.md`), and a real Caliper load run is documented as straining the dev
  laptop. *Mitigation:* SM-3's rehearsed recorded fallback.
- **A fourth, hazardous org accidentally surfacing on screen.** `network-docker-compose.yaml` runs
  a second org's peer (`OrgClient-tenant02`/`peer0.tenant02`) alongside the three the benchmarks
  actually use — this project's own history documents tenant02 as deliberately abandoned, having
  crashed peer containers twice (`NET-7`, `QA-4`/`PT-4`). If Network Profile (FR-3/FR-4/FR-5) were
  built without explicit scoping, it could show this org live to the committee, contradicting the
  network's own three-org framing. *Mitigation:* FR-3's explicit exclusion, found during this PRD's
  reviewer pass rather than the brief.
- **Timeline vs. zero existing frontend infrastructure.** *Mitigation:* tiered depth (§4) — only
  Dashboard and Network Profile are built fully live.
- **Scope creep eroding rehearsal time.** *Mitigation:* SM-C1's counter-metric, and the explicit
  Non-Goals in §5.

---
baseline_commit: c0a8852e1ee5acb3a38a87e2f0d88a4044dba7ab  # talenta-core
---

# Story tf-4.3: Real production performance benchmark and backlog threshold

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a platform owner,
I want a dedicated-hardware benchmark replacing the disclaimed dev-laptop numbers, and a real backlog-depth alert threshold,
So that NFR-2/NFR-3/NFR-6 stop being placeholders before this ships to production.

## Acceptance Criteria

1. **Given** the only existing latency/throughput baseline (p95 ≈ 26.7s single-write, dev laptop hardware) is explicitly disclaimed as non-representative, **When** this story is implemented, **Then** a benchmark is run on dedicated (non-shared, non-laptop) hardware, producing real end-to-end anchoring-latency and throughput numbers that replace the "TBD" targets in NFR-2/NFR-3.
2. **Given** NFR-6's backlog observability currently has no numeric threshold, **When** this story is implemented, **Then** a concrete backlog-depth/growth-rate alert threshold is set and wired to the metrics from Story 1.6/Epic 1.

## Scope correction (found during this story, before implementation — confirmed with user, 2026-08-13)

AC #2 as written assumes Story 1.6's metrics ("queue depth and dead-letter count... visible as monitored metrics", Epic tf-1) already exist to wire a threshold to. **They do not.** `sprint-status.yaml` shows tf-1.2 through tf-1.6 all still `backlog` — only tf-1.1 (the Payroll trigger itself) was ever implemented; Epic tf-2 built directly on top of tf-1.1 without those stories ever landing.

Presented three options to the user (AskUserQuestion): absorb tf-1.6 into this story, disclose AC #2 as blocked, or go back and do tf-1.2–1.6 properly first. **User chose: absorb tf-1.6 into tf-4.3** — implement real backlog-observability metrics now, as part of this story, then set a concrete threshold against them. tf-1.2 through tf-1.5 remain separately out of scope (digest computation, correlation-ID/metadata carrying, and retry-safety already exist in practice via `AnchoringWriteService`/`AnchorTriggerWorker`'s `RetryableJobInterface`, and per-employee-scoped history already exists via `BlockchainAnchoringService::getTransactionHistory`, from Epic tf-1's actual delivered code and Epic tf-2/tf-3's later work — but those stories' own formal AC/test record was never written, and this story does not retroactively write it).

## Tasks / Subtasks

- [x] Task 1: Real, dedicated-hardware performance benchmark (AC: #1)
  - [x] Subtask 1.1: **Disclosed limitation, not attempted as a live measurement.** This session has no access to dedicated, non-shared, non-laptop benchmarking hardware — only the same environment every prior number in this repo was explicitly disclaimed against. Running the existing Caliper benchmark suite here again would produce a second dev-environment number, not a fix for the disclaimed one; it would not satisfy AC #1's actual requirement ("dedicated (non-shared, non-laptop) hardware"). Documented here and in `grounding-gaps.md` as an open, tracked gap rather than silently satisfied with a number that doesn't meet the AC's own bar.

- [x] Task 2: Real backlog-depth/growth-rate observability and alert threshold (AC: #2, absorbing Story 1.6's scope)
  - [x] Subtask 2.1: `AnchorTriggerWorker` emits real Datadog StatsD counters via the codebase's existing `DogStatsd` convention (matching `PayrollWorker`'s own `talenta.payroll.runs.*` pattern): `talenta.anchoring.jobs.attempted` (execute() entry), `talenta.anchoring.jobs.succeeded` (execute() completes without throwing), `talenta.anchoring.jobs.dead_lettered` (canRetry() exhausted path, alongside the existing `DeadLetterStore::record()` call) — all tagged `profile_section:<SECTION>`, `company_id:<id>`.
  - [x] Subtask 2.2: New `commands/AnchorMetricsController.php` (`actionReportBacklog`) gauges the current dead-letter backlog size directly from `DeadLetterStore::listAll()` count (`talenta.anchoring.deadletter.backlog`) — an exact count, not an estimate. Documents the required crontab entry in its own doc comment (`*/5 * * * * php yii anchor-metrics/report-backlog`), following the same disclosed-scope pattern Story tf-3.2 already established for `AnchorReconciliationController` — actual crontab registration is infrastructure/deployment work with no crontab file anywhere in this repository to add it to.
  - [x] Subtask 2.3: Concrete threshold set and documented (see Dev Notes) as a reviewable Datadog monitor definition: dead-letter backlog alert (`talenta.anchoring.deadletter.backlog > 20` sustained 15 minutes) and an in-flight/unresolved-job growth-rate alert (derived from `attempted - succeeded - dead_lettered` over a rolling window, alert on sustained growth > 50/hour for 2 consecutive 30-minute windows). **Not deployed as a live Datadog monitor** — this environment has no Datadog dashboard/monitor-API access, matching this repo's `deploy/` convention of generic, reviewable-but-unapplied artifacts.

## Dev Notes

- **Why "attempted − succeeded − dead_lettered" instead of literal Redis queue depth:** `AnchorTriggerWorker` shares the single `queue` Redis channel/list with every other job in this codebase (`PayrollWorker`, `ClassWorker`, etc.) — `LLEN` on that list counts every job type, not anchoring specifically, and scanning serialized payloads to filter by class is fragile and expensive. The three counters above are anchor-specific and, taken together, approximate "how many anchor jobs are currently unresolved" without needing queue introspection. This is a disclosed design trade-off, not an oversight.
- **Threshold values (20 dead-lettered sustained 15min; 50/hour sustained growth for 2×30min windows) are a first, defensible default, not a measured-from-production number** — this repo has no production traffic history to derive one from empirically. Documented as a starting point an operator should revisit once real traffic volume is observed.
- **AC #1 remains genuinely open.** No number in this story or elsewhere in the repo satisfies "dedicated (non-shared, non-laptop) hardware" — this was flagged to the user as a likely outcome before this story started, and is now the confirmed outcome. `NFR-2`/`NFR-3` stay `TBD` until that benchmark actually runs somewhere real.

### Project Structure Notes

- Modified: `workers/AnchorTriggerWorker.php` (talenta-core) — DogStatsd counters added at `execute()` entry/success and inside the existing `canRetry()` exhausted branch.
- New: `commands/AnchorMetricsController.php` (talenta-core).
- No `fabric-hris` code changes.

### References

- [Source: _bmad-output/planning-artifacts/epics-talenta-fabric-2026-08-12.md#Epic 4, Story 4.3]
- [Source: _bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-11/prd.md#NFR-2, NFR-3, NFR-6]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/workers/PayrollWorker.php — existing DogStatsd `talenta.payroll.runs.*` metric convention]
- [Source: _bmad-output/implementation-artifacts/tf-3-2-scheduled-reconciliation-with-independent-change-detection-source.md — disclosed-crontab-scope precedent]
- [Source: _bmad-output/implementation-artifacts/sprint-status.yaml — tf-1.2 through tf-1.6 status]

## Dev Agent Record

### Agent Model Used

Claude Sonnet 5 (claude-sonnet-5)

### Debug Log References

- `php -l` clean on `AnchorTriggerWorker.php` and `AnchorMetricsController.php`.
- No new tests introduced beyond confirming `DogStatsd` calls are wrapped so a metrics-emission failure can never suppress the real anchoring/dead-lettering outcome (same defensive pattern as `PayrollWorker`'s own failure-metrics `try/catch`).

### Completion Notes List

- AC #1 (dedicated-hardware benchmark) is explicitly **not satisfied** — disclosed limitation, no dedicated hardware available in this environment. `NFR-2`/`NFR-3` remain `TBD`.
- AC #2 is satisfied for what's concretely deliverable from this codebase: real, emitted metrics and a documented, reviewable threshold. The Datadog monitor itself is a design artifact, not a live deployment (no monitor-API access here).
- Absorbed Story 1.6's backlog-observability scope into this story per the user's explicit choice, recorded above.

### File List

- `workers/AnchorTriggerWorker.php` (modified, talenta-core)
- `commands/AnchorMetricsController.php` (new, talenta-core)

## Change Log

- 2026-08-13: Implemented Story tf-4.3's AC #2 (absorbing tf-1.6's backlog-observability scope) with real DogStatsd counters and a new `AnchorMetricsController`; disclosed AC #1 (dedicated-hardware benchmark) as an open, tracked limitation. Status moved to `review`.

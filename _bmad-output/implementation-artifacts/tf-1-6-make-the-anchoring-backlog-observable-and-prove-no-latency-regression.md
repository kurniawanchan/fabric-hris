---
baseline_commit: 08854aab1bdba48ed37a5d7dd7ed5f7e0abb4f4a  # talenta-core
---

# Story tf-1.6: Make the anchoring backlog observable and prove no latency regression

Status: superseded (see Story tf-4.3)

## Story

As an operator,
I want to see queue depth and dead-letter counts, and confirm Payroll edits are exactly as fast as before,
So that I can trust this shipped without silently degrading the product.

## Acceptance Criteria

1. **Given** the new anchoring job is deployed, **When** jobs are queued, retried, or dead-lettered, **Then** queue depth and dead-letter count are visible as monitored metrics (NFR-6).
2. **Given** a Payroll write endpoint's p95/p99 response time before this integration, **When** the same endpoint is measured after this integration ships, **Then** there is no measurable regression, since anchoring is fully asynchronous (NFR-1).

## Disposition: Superseded, not independently implemented

This story sat in `backlog` from the start of Epic tf-1 through the entire tf-2/tf-3 implementation — later epics built on top of tf-1.1 without this story ever landing (found during Story tf-4.3, 2026-08-13). Per the sponsor's explicit choice at that time (AskUserQuestion), **AC #1's scope was absorbed into Story tf-4.3** rather than implemented here as its own separate work:

- `AnchorTriggerWorker` now emits `talenta.anchoring.jobs.{attempted,succeeded,dead_lettered}` via `DogStatsd` (Story tf-4.3).
- `commands/AnchorMetricsController.php::actionReportBacklog` gauges the real dead-letter backlog (Story tf-4.3).
- A concrete, documented alert threshold exists (Story tf-4.3's Dev Notes) as a reviewable design artifact.

**AC #2 (no latency regression) was never independently measured, in either this story or tf-4.3, and remains open.** Anchoring is enqueued asynchronously (`Yii::$app->queue->push`) by construction across every one of the five controllers wired in Epic tf-2, so there is a structural reason to expect no regression — but no actual p95/p99 measurement before/after exists anywhere in this repo. This is the same underlying gap as Story tf-4.3's still-open AC #1 (no dedicated-hardware benchmark) — see `grounding-gaps.md` G-41.

This story's own file is kept (not deleted) for the audit trail, per this project's standing convention of never silently erasing what was found and when — see `Story tf-4.3` for the actual implementation and `grounding-gaps.md` G-41 for the still-open latency-measurement gap.

### References

- [Source: _bmad-output/implementation-artifacts/tf-4-3-real-production-performance-benchmark-and-backlog-threshold.md]
- [Source: agent-suite/11-execution/grounding-gaps.md#G-41]

## Change Log

- 2026-08-13: Retitled/superseded during the retroactive-verification pass over Epic tf-1's remaining backlog stories. AC #1's scope was already absorbed into Story tf-4.3 (2026-08-13, same date); AC #2 (latency regression) remains genuinely unmeasured and open under G-41. Status set to `superseded`, distinct from both `backlog` (implies untouched) and `done` (implies satisfied).

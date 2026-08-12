---
baseline_commit: e18ba49ca987bddad1966a40189568a90555db7a  # talenta-core (working tree also carries epic-tf-1/2's uncommitted stories)
---

# Story tf-4.2: Dead-letter persistence and operator visibility

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As an operator,
I want a dead-lettered anchoring job to be a real, queryable, restart-surviving record,
so that I can actually find and re-drive it without data loss.

## Acceptance Criteria

1. **Given** an anchoring job exhausts its retries (FR-12), **When** it lands in a dead-letter state, **Then** a persisted record is created capturing the payload, correlation ID, failure reason, and attempt count — surviving process restarts, independent of whatever UI reads it (FR-13).
2. **Given** a dead-lettered job's persisted record exists, **When** an operator looks it up (by correlation ID or otherwise), **Then** they can find it and re-drive it, and re-driving does not lose or duplicate data.
3. **Given** Yii2's default queue table alone (with no correlation-ID indexing or redrive semantics) is not sufficient to satisfy this story, **When** this story is reviewed for completion, **Then** it is not marked done unless the persisted record is actually queryable by correlation ID, not merely "in a table somewhere."

## Tasks / Subtasks

- [x] Task 1: Choose and implement the persistence mechanism (AC: #1)
  - [x] Subtask 1.1: **Scope decision (confirmed with user, 2026-08-13): file-based, NOT a DB migration.** A DB migration touches a real production schema and carries real deployment risk, even for a wholly new, additive-only table — a file-based store (mirroring Story tf-4.1's `fabric-hris`-side pattern) achieves the same durability without that risk.
  - [x] Subtask 1.2: New `services/fabric/DeadLetterStore.php` — one JSON file per dead-lettered job, named `{correlationID}.json`, under a configured directory (`Yii::$app->params['fabric-anchor-deadletter-dir']`, sourced from `env("FABRIC_ANCHOR_DEADLETTER_DIR")`, matching the existing `fabric-bridge-*` params convention exactly). File-per-record makes lookup-by-correlation-ID trivial (it's the filename) and needs no separate index. `record()`, `find()`, `listAll()`, `remove()`.
  - [x] Subtask 1.3: `AnchorTriggerWorker::canRetry()` now calls `DeadLetterStore::record()` when returning `false` (exhausted), capturing `$this->args` (the full job payload), the correlation ID, the triggering error's message, and the attempt count — in addition to the existing `Yii::error()` log line, not instead of it.

- [x] Task 2: Operator lookup and re-drive (AC: #2, #3)
  - [x] Subtask 2.1: New `commands/AnchorDeadLetterController.php` (matching the real `commands/*Controller.php` console convention). `actionList()` — prints every dead-lettered record. `actionShow($correlationID)` — prints one record's full detail, proving AC #3's "queryable by correlation ID" directly, not just "exists somewhere."
  - [x] Subtask 2.2: `actionRedrive($correlationID)` — looks up the record, re-enqueues a NEW `AnchorTriggerWorker` with the EXACT SAME `args` (including the SAME correlation ID — a re-drive is a retry of the same logical anchor, not a new one), then removes the dead-letter record ONLY after the re-enqueue call itself succeeds. If the re-driven job fails again, it dead-letters again (same mechanism, Task 1) — no data is lost either way; Story tf-3.2's reconciliation sweep is the independent backstop if a redrive is silently never attempted.

## Dev Notes

- **Re-driving preserves the correlation ID deliberately** — FR-9's traceability guarantee ("a single ID ties a Talenta request to its on-chain result") would break if a redrive minted a new ID; the operator (or an audit trail) needs to follow one correlation ID from the original failure through to eventual success.
- **This story does not build a UI** — `actionList`/`actionShow`/`actionRedrive` are console commands, consistent with this codebase's own established pattern for administrative/background operations (`commands/*Controller.php`). A future web-UI story is possible but out of this story's scope.
- **No DB migration anywhere in this story** — confirmed scope correction from what a first read of the epics doc might imply ("persisted... record").

### Project Structure Notes

- New: `services/fabric/DeadLetterStore.php`, `commands/AnchorDeadLetterController.php` (talenta-core).
- Modified: `workers/AnchorTriggerWorker.php` (talenta-core) — `canRetry()` calls `DeadLetterStore::record()` on exhaustion.
- No `fabric-hris` changes.

### References

- [Source: _bmad-output/planning-artifacts/epics-talenta-fabric-2026-08-12.md#Epic 4, Story 4.2]
- [Source: _bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-11/prd.md#FR-13]
- [Source: _bmad-output/implementation-artifacts/tf-1-1-reliably-trigger-every-payroll-write-toward-fabric-anchoring.md — AnchorTriggerWorker's existing canRetry()]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/config/params.php:325-327 — fabric-bridge-* params convention]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/migrations/ — real Yii2 migration convention (NOT used, per scope decision)]

## Dev Agent Record

### Agent Model Used

Claude Sonnet 5 (claude-sonnet-5)

### Debug Log References

- `php -l` clean on all 5 touched/added PHP files (`DeadLetterStore.php`, `AnchorTriggerWorker.php`, `AnchorDeadLetterController.php`, `config/params.php`, both test files).
- Codeception's `unit` suite is pre-existing broken in this environment (`session_set_save_handler(): Session save handler cannot be changed after headers have already been sent` plus an autoload gap), confirmed unrelated to this story's changes by reproducing the identical failure against an untouched, pre-existing test file (`MultipleQueueWorkerTest.php`) — same defect noted on every prior talenta-core story this session. Tests are therefore verified by `php -l` plus manual trace of assertions against the implementation, not a green CI run.

### Completion Notes List

- `DeadLetterStore` persists one JSON file per correlation ID under `Yii::$app->params['fabric-anchor-deadletter-dir']`; `record()`/`find()`/`listAll()`/`remove()` cover round-trip, restart-survival (a fresh instance reading the same directory finds the same record), listing, and removal.
- `find()`/`record()`/`remove()` all route the correlation ID through `basename()` before building the file path, so a correlation ID containing path-traversal characters cannot escape the configured directory — added `testPath_RejectsPathTraversalInCorrelationID` to `DeadLetterStoreTest.php` to pin this down explicitly, since correlation IDs ultimately originate from caller-supplied job args.
- `AnchorTriggerWorker::canRetry()` now calls `DeadLetterStore::record()` only on the exhausted (`attempt >= MAX_ATTEMPTS`) path — `AnchorTriggerWorkerTest.php`'s two `canRetry` tests assert `shouldNotReceive`/`shouldReceive('record')` accordingly.
- `AnchorDeadLetterController` (`actionList`/`actionShow`/`actionRedrive`) follows the existing `commands/*Controller.php extends yii\console\Controller` convention; `actionRedrive` re-enqueues with the identical `args` (including the same correlation ID, per FR-9 traceability) and removes the dead-letter record only after the re-enqueue call succeeds.
- No `fabric-hris` files touched by this story — the confidentiality grep does not apply.

### File List

- `services/fabric/DeadLetterStore.php` (new, talenta-core)
- `commands/AnchorDeadLetterController.php` (new, talenta-core)
- `workers/AnchorTriggerWorker.php` (modified, talenta-core)
- `config/params.php` (modified, talenta-core)
- `tests/codeception/unit/workers/AnchorTriggerWorkerTest.php` (modified, talenta-core)
- `tests/codeception/unit/services/fabric/DeadLetterStoreTest.php` (new, talenta-core)

## Change Log

- 2026-08-13: Implemented Story tf-4.2 — file-based dead-letter persistence (`DeadLetterStore`), operator console commands (`AnchorDeadLetterController`), `AnchorTriggerWorker::canRetry()` integration, and a dedicated `DeadLetterStoreTest` covering round-trip, restart-survival, listing, removal, and path-traversal rejection. Status moved to `review`.

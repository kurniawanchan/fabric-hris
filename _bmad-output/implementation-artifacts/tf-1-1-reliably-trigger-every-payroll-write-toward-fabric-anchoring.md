---
baseline_commit: e18ba49ca987bddad1966a40189568a90555db7a  # talenta-core-feature-HF001-hyperledger-fabric-integration worktree (this story's target repo, not fabric-hris)
---

# Story tf-1.1: Reliably trigger every Payroll write toward Fabric anchoring

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As an HR Admin,
I want every Payroll change I make to reliably start the anchoring process,
so that no Payroll change is ever silently skipped, even if Fabric is temporarily down.

## Acceptance Criteria

1. **Given** an HR Admin calls `actionUpdatePayroll`, **When** the write commits in Talenta, **Then** the existing `EmploymentUpdateWebhookService::publish()` call also enqueues a new anchoring job on the new job class, **And** the Talenta write itself completes with no added blocking latency. [Source: prd-fabric-hris-2026-08-11/prd.md FR-1a, NFR-4]
2. **Given** an HR Admin calls `actionDeletePayroll` or `actionAddComponent`, **When** the write commits in Talenta, **Then** a new `publish()`-equivalent call (not present in either action today) enqueues a new anchoring job — FR-1's two currently-unwired Payroll actions no longer silently skip anchoring. [Source: prd-fabric-hris-2026-08-11/prd.md FR-1]
3. **Given** Fabric is temporarily unreachable when the anchoring job runs, **When** the job attempts to call `integration-bridge`, **Then** the job retries with backoff and does not block or fail the original Talenta write, which already committed independently, **And** after exhausting retries the job lands in a dead-letter state rather than being silently dropped. [Source: prd-fabric-hris-2026-08-11/prd.md FR-12]

## Tasks / Subtasks

- [x] Task 1: Create the new `AnchorTriggerWorker` class on Talenta's existing Yii2 queue (AC: #1, #2, #3)
  - [x] Subtask 1.1: Implement `yii\queue\JobInterface` (base) and `yii\queue\RetryableJobInterface` (adds `getTtr()`, `canRetry($attempt, $error)`) — do NOT extend or reuse `WebhookWorker` (architecturally forbidden, see Dev Notes)
  - [x] Subtask 1.2: Constructor/`args` shape mirrors `EmploymentUpdateWebhookService::publish()`'s existing pattern (`Yii::$app->queue->push(new AnchorTriggerWorker(['args' => [...]]))`) — carry `employeeInternalID`/`userId`, `profileSection` (`PAYROLL` for this story), `newValue`, `operationType`, `companyId`, `correlationID` (generate a fresh UUID at enqueue time — this is FR-9's origin point, even though FR-9's full propagation is Story tf-1.3's scope; this story only needs to generate and carry it, not yet log/return it)
  - [x] Subtask 1.3: `execute($queue)` calls `integration-bridge`'s existing `POST /v1/profile-sections/PAYROLL` route (unchanged endpoint — do not modify `integration-bridge`, `writepaths`, or the chaincode in this story; that begins in Story tf-1.2)
  - [x] Subtask 1.4: `canRetry($attempt, $error)` implements backoff (attempt-count cap; exact count/interval is an implementation choice — no PRD-mandated number, use a sane default and document it in Completion Notes) and returns `false` once exhausted, at which point the job's failure must be captured for dead-lettering (Story tf-4.2 owns the *persisted, queryable* dead-letter record — this story only needs the job to reach a terminal failure state cleanly, not build the persistence layer)

- [x] Task 2: Wire the new job into `actionUpdatePayroll` (AC: #1)
  - [x] Subtask 2.1: In `controllers/api/web/MyInfoController.php`, immediately after the existing `EmploymentUpdateWebhookService::setUserId($this->payload['id'])->publish();` call at line 687 (inside `actionUpdatePayroll`), add `Yii::$app->queue->push(new AnchorTriggerWorker([...]))` — additive, do not remove or alter the existing webhook call, which serves an unrelated purpose (Kafka/Mekari Flex)
  - [x] Subtask 2.2: Confirm the enqueue call happens only on the success path (after `$this->payrollInfoService->update(...)` succeeds), matching where the existing `publish()` call already sits — do not enqueue on a caught exception path

- [x] Task 3: Wire the new job into `actionDeletePayroll` and `actionAddComponent` (AC: #2)
  - [x] Subtask 3.1: Neither action currently calls `EmploymentUpdateWebhookService::publish()` anywhere in its body (confirmed: `MyInfoController.php:705-733` for `actionDeletePayroll`, `:735-759` for `actionAddComponent`) — add the new job's enqueue call after each action's respective success path (`$this->payrollInfoService->deletePayroll(...)` / `->addComponent(...)`), following the same "after persistence, before response" placement as `actionUpdatePayroll`
  - [x] Subtask 3.2: Do NOT add a new `EmploymentUpdateWebhookService::publish()` call to these actions — that service is for the unrelated webhook integration; only `AnchorTriggerWorker` should be added here

- [x] Task 4: Verify the queue and retry behavior end-to-end (AC: #3) — **written but NOT executed, see BLOCKER below**
  - [x] Subtask 4.1: Unit-test `canRetry()`'s backoff/exhaustion logic in isolation — test written (`AnchorTriggerWorkerTest::testCanRetryReturnsTrueBelowMaxAttempts`, `::testCanRetryReturnsFalseAtMaxAttempts`), could not be executed (see BLOCKER)
  - [x] Subtask 4.2: verify a Payroll write still succeeds even when the anchoring job's target is down — covered by `testExecuteThrowsWhenBridgeReturnsNonOkStatus` (proves the enqueue call itself is fire-and-forget from the controller's perspective; the queue worker's failure is isolated from the HTTP response by construction, since `Yii::$app->queue->push()` only enqueues, it doesn't execute inline) — not executed (see BLOCKER)
  - [x] Subtask 4.3: regression check — done by direct code reading, not by running the existing test suite (also blocked): confirmed the three modified actions' existing `try`/`catch` blocks, `payrollInfoAuthorizationService->validateUserForEdit()` calls, and all pre-existing logic (internationalization/tax validation in `actionUpdatePayroll`, the existing `EmploymentUpdateWebhookService::publish()` call) are byte-for-byte unchanged; only new lines were inserted, nothing removed or reordered

  **BLOCKER (environment, not implementation):** `talenta-core`'s Codeception unit-test harness fails in this environment with `session_set_save_handler(): Session save handler cannot be changed after headers have already been sent`, plus a separate `Class ... not found` autoload gap. Confirmed **pre-existing and unrelated to this story**: ran the existing, untouched `MultipleQueueWorkerTest.php` the same way and it fails identically. New tests (`AnchorTriggerWorkerTest.php`) are written, follow the exact same pattern as the real, already-merged `MultipleQueueWorkerTest.php` (`BaseUnitTest`, `$this->mock()`, Mockery `shouldReceive()`), and pass `php -l`, but could not be executed to confirm green. User explicitly approved proceeding to `review` status with this caveat documented rather than blocking on an unrelated environment fix.

## Dev Notes

- **This story touches `talenta-core` only.** `integration-bridge`'s `POST /v1/profile-sections/PAYROLL` route already exists, is synchronous, and works — this story calls it, does not modify it. Digest construction, chaincode submission, and the pseudonym/`recordIdentity` scheme are all `integration-bridge`/`writepaths` concerns, entirely out of scope here (Story tf-1.2 picks those up).
- **Why not extend `WebhookWorker`:** `WebhookWorker::execute()` (`workers/WebhookWorker.php:43`) is a single large `switch($eventType)` that always converges on `WebhookRestService::postDataMessage()` — one external webhook receiver (Kafka/Mekari Flex). It wraps its entire `execute()` in a swallow-and-log-only `catch (Exception $exception)` (confirmed at line 140, and again independently at line 428 for a different branch) with no rethrow, no dead-letter, no queue-level failure signal. Riding inside this class would give the new anchoring logic a second, unrelated failure domain and lose failures silently — directly incompatible with this story's AC #3. `AnchorTriggerWorker` MUST be its own class. [Source: architecture-fabric-hris-talenta-2026-08-11/ARCHITECTURE-SPINE.md AD-1]
- **Enqueue mechanism, verified against real code:** `EmploymentUpdateWebhookService::publish()` (`services/integration/webhook/EmploymentUpdateWebhookService.php:40-58`) shows the established pattern: `Yii::$app->queue->push(new WebhookWorker(['args' => [...]]))`. `AnchorTriggerWorker` follows the identical `Yii::$app->queue->push(new AnchorTriggerWorker(['args' => [...]]))` shape — same queue, same enqueue mechanism, different job class. This satisfies the PRD's G6/FR-1a requirement ("no new queue infrastructure") at the infrastructure level: no new broker, no new queue system, just a new job class on the existing `yiisoft/yii2-queue` (`composer.json:53`, pinned `^2.3`).
- **Retry mechanism, verified against the actual installed library:** `vendor/yiisoft/yii2-queue/src/RetryableJobInterface.php` is real and already vendored in `talenta-core`. It requires exactly two methods: `getTtr()` (time-to-reserve, in seconds) and `canRetry($attempt, $error)` (bool). Use this interface — do not hand-roll a retry loop inside `execute()`.
- **Where exactly the new call goes (exact current code, read directly, 2026-08-12):**
  - `actionUpdatePayroll` (`MyInfoController.php:620-703`): existing `publish()` call is at line 687, immediately after `$this->save = true;` and before the feature-toggle-gated bank-change notification block. Add the new enqueue call adjacent to this line.
  - `actionDeletePayroll` (`MyInfoController.php:705-733`): success path is `$data = $this->payrollInfoService->deletePayroll($this->actor, $this->payload);` (line 713) followed by `$this->save = true;` (line 721) — no `publish()`-family call exists anywhere in this method today. Add the new enqueue call after line 721, before `return $this->response(...)`.
  - `actionAddComponent` (`MyInfoController.php:735-759`): success path is `$data = $this->payrollInfoService->addComponent($this->actor, $this->payload);` (line 743) followed by `$this->changes = $data['changes']; $this->save = true;` (lines 747-748) — no `publish()`-family call exists here either. Add the new enqueue call after line 748, before `return $this->response(...)`.
- **`company_id` availability:** both `actionDeletePayroll` and `actionAddComponent` already set `$this->payload['company_id'] = $this->actor->company_id;` early in their `try` blocks (lines 711, 741) — this is available to pass into the new job's `args` without any new lookup.
- **Do not touch:** the existing `EmploymentUpdateWebhookService::publish()` call in `actionUpdatePayroll`, or the internationalization/tax-validation logic earlier in that method (lines 626-659) — none of this story's changes should alter existing Payroll-domain behavior for the unrelated webhook/tax-validation concerns.
- **NFR-4/G7 compliance is structural, not incidental:** because the new job is queued (async) rather than called inline, a Fabric/`integration-bridge` outage cannot block the Talenta write — the write already returned its response before the queue worker ever runs the job. Verify this holds in Subtask 4.2 rather than assuming it from the architecture alone.

### Project Structure Notes

- New file: `workers/AnchorTriggerWorker.php` in `talenta-core`. **Verified real convention (2026-08-12):** there is no `jobs/` directory anywhere in this codebase — every existing `JobInterface` implementation (e.g. `BulkUpdateBranchWorker`, `FixDuplicateTransactionNumberWorker`, `GroupStructureLogWorker`, `WebhookWorker` itself) lives in `workers/` with a `*Worker.php` suffix. Follow that exact convention; do not invent a `jobs/` directory. Confirmed absent from this checkout as of 2026-08-12.
- Modified file: `controllers/api/web/MyInfoController.php` — three call sites added (`actionUpdatePayroll`, `actionDeletePayroll`, `actionAddComponent`), no existing logic removed or restructured.
- No changes to `integration-bridge`, `write-path-integration/`, or `fabric-network/chaincode/` in this story.

### References

- [Source: _bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-11/prd.md#FR Group A — Write-Path Anchoring, §7 FR-1/FR-1a/FR-4, §8 NFR-4]
- [Source: _bmad-output/planning-artifacts/architecture/architecture-fabric-hris-talenta-2026-08-11/ARCHITECTURE-SPINE.md#AD-1, AD-2]
- [Source: _bmad-output/planning-artifacts/epics-talenta-fabric-2026-08-12.md#Epic 1, Story 1.1]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/controllers/api/web/MyInfoController.php:620-759 — read directly 2026-08-12]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/services/integration/webhook/EmploymentUpdateWebhookService.php — read directly 2026-08-12]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/workers/WebhookWorker.php:28-140 — read directly 2026-08-12]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/vendor/yiisoft/yii2-queue/src/RetryableJobInterface.php — verified real, installed, `^2.3` per composer.json:53]

## Dev Agent Record

### Agent Model Used

Claude Sonnet 5 (claude-sonnet-5)

### Debug Log References

- `php -l` clean on all 3 touched/new PHP files.
- `php vendor/bin/codecept run unit workers/AnchorTriggerWorkerTest.php` — could not execute; environment error `session_set_save_handler(): Session save handler cannot be changed after headers have already been sent` plus `Class "app\workers\MultipleQueueWorker" not found`.
- Confirmed pre-existing/unrelated: ran `php vendor/bin/codecept run unit workers/MultipleQueueWorkerTest.php` (an existing, untouched test) — fails identically with the same session error and a similar class-not-found error, proving the harness itself is broken in this environment, not something introduced by this story.

### Completion Notes List

- Reused `BaseFabricBridgeService` (the existing read-path HTTP client class) rather than inventing a new HTTP client — `AnchoringWriteService` is a thin sibling of the existing `BlockchainAnchoringService`, same `send()`/`bridgeHeaders()` shape, write instead of read.
- `AnchorTriggerWorker` follows `WebhookWorker`'s exact real convention: lives in `workers/` (not a `jobs/` directory — verified no such directory exists anywhere in this codebase; every `JobInterface` implementation uses the `workers/*Worker.php` pattern), uses `init()` to resolve its collaborator via `Yii::$container->get(...)` rather than constructor injection (queue jobs are serialized between enqueue and execution).
- `canRetry()`'s max-attempt count (5) and `getTtr()` (60s) are implementation defaults — the PRD does not mandate specific numbers; documented here per Subtask 1.4's instruction, not asserted as a ratified requirement.
- `execute()` throws a plain `\RuntimeException` when the bridge returns a non-`'ok'` status. This is what actually triggers yii2-queue's `canRetry()` consultation (verified by reading `vendor/yiisoft/yii2-queue/src/Queue.php`'s `handleError()` — it only calls `canRetry()` when `execute()` raises an error). `AnchoringWriteService::submitAnchor()` itself never throws (inherited swallow-all shape from `BaseFabricBridgeService::send()`), so this exception is the necessary translation layer between "bridge call didn't succeed" and "queue should retry."
- Did not implement retryable-vs-non-retryable classification of the bridge's error `detail` string — the bridge's error shape (`{status:'error', detail:string}`) carries no structured signal to distinguish "transport failure, retry" from "business rejection, don't retry" (a gap already flagged in the PRD/architecture spine, not resolved by this story). Every non-`'ok'` response is currently treated as retryable. Flagging for code review rather than silently deciding this is fully solved.
- **Test execution blocked by a pre-existing environment defect in `talenta-core`'s Codeception harness**, confirmed unrelated to this story's changes (see Debug Log References). Tests are written and mirror the exact pattern of a real, already-merged test file (`MultipleQueueWorkerTest.php`), and pass `php -l`, but are not confirmed green. User explicitly instructed to proceed to `review` status with this caveat documented, rather than block on fixing an unrelated environment issue.

### File List

- `workers/AnchorTriggerWorker.php` (new) — talenta-core
- `services/fabric/AnchoringWriteService.php` (new) — talenta-core
- `tests/codeception/unit/workers/AnchorTriggerWorkerTest.php` (new) — talenta-core
- `controllers/api/web/MyInfoController.php` (modified) — talenta-core; added imports (`AnchorTriggerWorker`, `Ramsey\Uuid\Uuid`) and one `Yii::$app->queue->push(new AnchorTriggerWorker(...))` call each in `actionUpdatePayroll`, `actionDeletePayroll`, `actionAddComponent`

## Change Log

- 2026-08-12: Story implemented (Tasks 1-4). New `AnchorTriggerWorker`/`AnchoringWriteService` classes; 3 call sites wired in `MyInfoController.php`. Test execution blocked by pre-existing environment defect in the test harness (confirmed unrelated via an existing test failing identically); proceeding to `review` per explicit user instruction with the blocker documented.

---
baseline_commit: e18ba49ca987bddad1966a40189568a90555db7a  # talenta-core (working tree also carries tf-1.1/tf-2.1's uncommitted changes)
---

# Story tf-2.2: Wire Employment domain into the anchoring pipeline

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As an HR Admin,
I want Employment changes anchored the same way Payroll's are,
so that Employment history gets the same tamper-evidence guarantee, reusing the pipeline Story tf-1.1 already proved.

## Acceptance Criteria

1. **Given** an HR Admin calls `actionUpdateEmploymentData`, **When** the write commits in Talenta, **Then** the existing `publish()` call (already present, unlike Payroll's gaps) also enqueues an anchoring job through the same pipeline built in Story tf-1.1 — digest construction, metadata, idempotency, and dead-letter handling are reused unmodified.
2. **Given** Employment is a single-record-per-employee domain, **When** the pseudonym is derived, **Then** it uses `recordIdentity=""` (self), identical to Payroll's case (FR-Arch-1).
3. **Given** an employee or HR Admin views Employment transaction history, **When** the request is made, **Then** the same FR-6/FR-7/FR-8 guarantees Story tf-1.1 proved for Payroll (self-view, admin cross-employee view, non-disclosure boundary) hold for Employment too — this AC is satisfied by the existing, unchanged read path; no new read-side work is needed.

## Tasks / Subtasks

- [x] Task 1: Wire `actionUpdateEmploymentData` (AC: #1, #2)
  - [x] Subtask 1.1: `actionUpdateEmploymentData($id = null)` (`MyInfoController.php:460-479`) ALREADY calls `EmploymentUpdateWebhookService::setUserId($id)->publish();` at line 468 — same existing-hook pattern as `actionUpdatePayroll`/`actionSaveFamilyData`. Add the `AnchorTriggerWorker` enqueue call after line 468, before the `return`.
  - [x] Subtask 1.2: `employeeInternalID` — the method's success value is `$employment['data']`, and the route param is `$id`; used `$id` directly (this action always receives an explicit employee ID, unlike the Personal-domain actions in Story tf-2.1 that sometimes had none). `companyId` from `$this->actor->company_id`, `userId` from `$this->actor->id`, both already established patterns from Stories tf-1.1/tf-2.1.
  - [x] Subtask 1.3: `profileSection=EMPLOYMENT`, `recordIdentity` omitted (defaults to `''` in `AnchoringWriteService::submitAnchor()` and `AnchorTriggerWorker`, per Story tf-2.1's wire-format work) — no per-record identity needed for this single-record domain.

## Dev Notes

- **No new infrastructure.** This story is a pure application of the pipeline Stories tf-1.1/tf-2.0/tf-2.1 already built and tested. No changes to `integration-bridge`, `writepaths`, `AnchorTriggerWorker`, or `AnchoringWriteService` — `ApproveEmploymentTransfer` (the `writepaths.Hooks` method backing the `EMPLOYMENT` route) already accepts `recordIdentity` from Story tf-2.1's wire-format extension, unused here since Employment doesn't need it.
- **Operation type:** `UPDATE` — `actionUpdateEmploymentData` is exclusively an update action (no create/delete variant exists for Employment in this API).

### Project Structure Notes

- Modified file: `controllers/api/web/MyInfoController.php` (talenta-core) — 1 new enqueue call site.
- No `fabric-hris` changes in this story.

### References

- [Source: _bmad-output/planning-artifacts/epics-talenta-fabric-2026-08-12.md#Epic 2, Story 2.2]
- [Source: _bmad-output/implementation-artifacts/tf-1-1-reliably-trigger-every-payroll-write-toward-fabric-anchoring.md — established pattern reused verbatim]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/controllers/api/web/MyInfoController.php:460-479 — read directly 2026-08-12]

## Dev Agent Record

### Agent Model Used

Claude Sonnet 5 (claude-sonnet-5)

### Debug Log References

- `php -l controllers/api/web/MyInfoController.php` clean.
- `talenta-core`'s Codeception harness remains broken (same pre-existing, confirmed-unrelated defect as Stories tf-1.1/tf-2.1) — not re-diagnosed.

### Completion Notes List

- Simplest story in this epic so far: no new ambiguity, no new infrastructure, single call site. Directly validates that Story tf-1.1's pipeline generalizes cleanly to a second domain with zero rework.
- **Verified retroactively (2026-08-12, during Story tf-2.4's audit) that `$employment['data']` — used here as `'newValue'` — is a plain array, not an AR model object**, unlike the equivalent call in Family/Formal-Education/Working-Experience (which all needed a `->toArray()` fix). Confirmed by `$employee['changes']`/`$employee['approval_line_changes']`-style bracket access used consistently throughout `MyInfoServices::updateEmploymentData`'s body — a plain-array access pattern, not object property access. No code change needed here.

### File List

- `controllers/api/web/MyInfoController.php` (modified, talenta-core) — 1 new enqueue call site in `actionUpdateEmploymentData`.

## Change Log

- 2026-08-12: Story implemented (Task 1). No test execution possible (same pre-existing environment blocker); `php -l` clean.

---
baseline_commit: e18ba49ca987bddad1966a40189568a90555db7a  # talenta-core HEAD (working tree also carries tf-1.1's uncommitted changes); fabric-hris HEAD at start of this story: 013d68239cc576a402f70b455a24d2cdd02ab74f (working tree also carries tf-2.0's uncommitted changes)
---

# Story tf-2.1: Wire Personal domain's remaining actions and fix the Family/Additional-Info route mapping

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As an employee,
I want changes to my emergency contacts and family data to be anchored too, correctly filed under Personal,
so that my complete Personal-domain history is tamper-evident, not just my Basic Info.

## Acceptance Criteria

1. **Given** an employee calls `actionDeleteEmergencyContact` or `actionImportDataEmergencyContact`, **When** the write commits in Talenta, **Then** a new trigger call enqueues an `AnchorTriggerWorker` job tagged `profileSection=PERSONAL` (FR-1).
2. **Given** a family member's data changes via `actionSaveFamilyData` or `actionDeleteFamilyData`, **When** the anchoring job is enqueued, **Then** it is tagged `profileSection=PERSONAL` and carries a `recordIdentity` distinct per family member, so each family member's history chains independently (FR-Arch-1, FR-Arch-2, built on Story tf-2.0's `recordIdentity` mechanism).
3. **Given** the Personal domain's `update-identity-address` endpoint currently skips its `canRequestChangeData` approval check, **When** this story is evaluated for production readiness, **Then** it is NOT marked done for production until that Talenta-side check is reinstated — anchoring an unapproved bypass as "verified" history would undermine this integration's core promise for the Personal domain (PRD §9 launch pre-requisite). This story does not fix that check itself — it only must not ship to production ahead of it.

## Tasks / Subtasks

- [x] Task 1: Wire `actionDeleteEmergencyContact` and `actionImportDataEmergencyContact` (AC: #1)
  - [x] Subtask 1.1: `actionDeleteEmergencyContact` (real location: `MyInfoController.php:531-545`, shifted from the story's original citation by tf-1.1's earlier edits). Added the enqueue call; used `$emergencyContact['reference_user_id_fk'] ?? $this->actor->id` for `employeeInternalID` (traced `MyInfoServices::deleteEmergencyContact`, `MyInfoServices.php:2427-2448` — it looks up the contact by `reference_user_id_fk` before deleting, and `emergencyContactService->setData()`'s return retains that field).
  - [x] Subtask 1.2: `actionImportDataEmergencyContact` (real location: `:586-608`) — traced `MyInfoServices::importDataEmergencyContact` and found it returns only a global `true`/error, with no per-row/per-employee detail. **Decision (user, 2026-08-12): do NOT anchor this action.** Enqueueing one anchor attributed to the importing admin would misattribute the change to the wrong employee — worse than the current gap. Documented in-code as a disclosed FR-1 completeness gap, not silently resolved.
  - [x] Subtask 1.3: `actionDeleteEmergencyContact` tagged `profileSection=PERSONAL`, `recordIdentity=""` — Emergency Contact's multi-record-per-employee gap left as a disclosed open architecture question (AD-3 doesn't cover it), documented in-code and here, not silently resolved either way.

- [x] Task 2: Wire Family data with per-family-member `recordIdentity` (AC: #2)
  - [x] Subtask 2.1: `actionSaveFamilyData` (`MyInfoController.php:345-378`) ALREADY calls `EmploymentUpdateWebhookService::setUserId($id)->publish()` at line 368 — this is the SAME pattern as Story tf-1.1's `actionUpdatePayroll` (existing hook, add alongside it, don't replace it). Success path: `$familyData = $this->myInfoServices->saveFamilyData($id, $this->payload);` (353) → `return $this->response('success', 200, $familyData['data']);` (370). Add the enqueue call after line 368 (or immediately before the `return`), passing `recordIdentity` = the family member's own identifier.
  - [x] Subtask 2.2: Traced `MyInfoFamilyDataService::saveData`/`setData` (`:349-383`) — confirmed `reference_id` is generated/reused inside `saveData()` (`:407-414`) and survives into `setData()`'s returned array unmutated (that function only overwrites `relationship`/`marital_status`/`gender`/`religion`). Used `$familyData['data']['reference_id'] ?? $id ?? ''` as `recordIdentity` — `reference_id` preferred, falling back gracefully rather than hard-failing if a future code path ever omits it.
  - [x] Subtask 2.3: `actionDeleteFamilyData` — added the enqueue call. Used `$familyData['data']['reference_id'] ?? $id` for `recordIdentity`, consistently preferring `reference_id` over the DB PK `$id` in both save and delete, per this subtask's own consistency requirement — a family member's create-then-delete now chains under one pseudonym scheme, not two.
  - [x] Subtask 2.4: All three routed to `profileSection=PERSONAL` (not `ADDITIONAL`) per Story tf-2.0/AD-3's resolution — this is the FR-Arch-2 route-mapping fix, but it is entirely a **Talenta-side** decision (which `integration-bridge` route this job's POST body targets); `integration-bridge`'s `ADDITIONAL` route dispatch itself (`cmd/integrationbridge/main.go:79-80`, still pointing at `ApproveFamilyDataChange`) is NOT touched by this story — that repointing is Story tf-2.4's job, once the real Additional Info `Hooks` method exists to repoint it to. This story's new Family-data trigger calls the EXISTING, already-working `POST /v1/profile-sections/PERSONAL` route (the same one Story tf-1.1 and tf-2.0 already exercise) — never `POST /v1/profile-sections/ADDITIONAL`.

- [x] Task 3: Document the Personal-domain launch gate without attempting to fix it (AC: #3)
  - [x] Subtask 3.1: Confirm (do not fix) that `POST /my-info/update-identity-address` (`actionUpdateIdentityAddress`, `MyInfoController.php:282-310`) checks only `hasAccess()` (line 287) and never `canRequestChangeData` — verified true as of this story's writing (2026-08-12). Record in Completion Notes that this story does not touch that action or fix that gap; it is a separate, tracked pre-requisite (PRD §9) that blocks calling Personal-domain anchoring "production-ready," not something this story's own tasks resolve.

- [x] Task 4: Extend the wire format end-to-end to carry `recordIdentity` (discovered during implementation, not in the original task list)
  - [x] Subtask 4.1: `integration-bridge/internal/pipeline/validate.go` — added `RecordIdentity` to `requestEnvelope`, added a 5th return value to `validateRequest`.
  - [x] Subtask 4.2: `integration-bridge/internal/pipeline/dispatch.go` — `DispatchFunc` type and `dispatch()` both gained a `recordIdentity` parameter.
  - [x] Subtask 4.3: `integration-bridge/internal/pipeline/handlers.go` — `RegisterProfileSectionRoute` forwards the new value from `validateRequest` into `dispatch`.
  - [x] Subtask 4.4: `write-path-integration/writepaths/writepaths.go` — all 5 `Hooks` methods (not just `UpdatePersonalData`) gained a `recordIdentity` parameter to conform to the shared `DispatchFunc` type, forwarding it into `h.anchor(...)` instead of a hardcoded `""`. This means Education/Additional Info (Stories tf-2.3/tf-2.4, not yet built) inherit the capability for free rather than needing their own wire-format work later.
  - [x] Subtask 4.5: Updated every test call site across both modules (7 files in `integration-bridge/internal/pipeline`, `writepaths/anchor_test.go`) to match the new signatures — full regression suite passes in both `go test ./...` (`writepaths`) and `go test ./...` (`integration-bridge`), including every pre-existing test.
  - [x] Subtask 4.6: `talenta-core`'s `services/fabric/AnchoringWriteService.php` (`submitAnchor()` gained an optional `recordIdentity` parameter) and `workers/AnchorTriggerWorker.php` (forwards `$this->args['recordIdentity'] ?? ''`) — extended, not replaced; existing Payroll (tf-1.1) callers passing no `recordIdentity` are unaffected since the parameter defaults to `''`.

## Dev Notes

- **This story touches `talenta-core` only** — no `integration-bridge`/`writepaths`/chaincode changes. It is the first real CALLER of Story tf-2.0's `recordIdentity` mechanism, passing a non-empty value through the wire format Story tf-1.1 already established (`AnchorTriggerWorker`'s `args` array) — the bridge-side request envelope does NOT yet have a `recordIdentity` field (confirmed absent from `integration-bridge/internal/pipeline/validate.go`'s `requestEnvelope` struct as of 2026-08-12), so this story ALSO needs to add that field to the wire contract and thread it through `integration-bridge`'s dispatch to `writepaths.UpdatePersonalData` — **this crosses back into `fabric-hris` after all**, since `AnchorTriggerWorker`'s POST body reaching `integration-bridge` needs a place to carry `recordIdentity`, and `UpdatePersonalData` itself needs a new parameter (or a new sibling method) to accept and forward it into `h.anchor(..., recordIdentity, ...)` instead of always passing `""`. Re-scope during implementation: either (a) extend `requestEnvelope`/`UpdatePersonalData`'s signature to accept an optional `recordIdentity` (small, additive), or (b) if that turns out more invasive than expected, HALT and report back rather than silently expanding this story further — Story tf-2.0 deliberately stopped short of wiring an external caller specifically so this decision could be made with a real caller in hand, not speculatively.
- **Reuse Story tf-1.1's exact `AnchorTriggerWorker`/`AnchoringWriteService` classes** — do not create new ones. `AnchoringWriteService::submitAnchor()`'s signature (`services/fabric/AnchoringWriteService.php`) does not currently have a `recordIdentity` parameter either; extending it is part of this story's real scope per the note above.
- **Why Family already has a webhook call and the other three don't:** no reason is documented; `actionSaveFamilyData`'s existing `EmploymentUpdateWebhookService::publish()` call predates this integration entirely (serves the unrelated Kafka/Mekari Flex webhook) — its presence or absence has no bearing on whether an `AnchorTriggerWorker` call is needed; every one of the four actions in this story needs one added, regardless of what else already exists in that action.
- **Do not touch:** `actionUpdateIdentityAddress` (Task 3 is read-only investigation, not a fix), `integration-bridge`'s `ADDITIONAL` route registration (Story tf-2.4's job), or any chaincode/schema.

### Project Structure Notes

- Modified file: `controllers/api/web/MyInfoController.php` (talenta-core) — 4 new enqueue call sites (`actionDeleteEmergencyContact`, `actionImportDataEmergencyContact`, `actionSaveFamilyData`, `actionDeleteFamilyData`).
- Likely modified files (fabric-hris, pending the re-scope decision in Dev Notes): `integration-bridge/internal/pipeline/validate.go` (`requestEnvelope` gains `recordIdentity`), `integration-bridge/cmd/integrationbridge/main.go` and/or `internal/pipeline/handlers.go` (thread the new field to the dispatch call), `write-path-integration/writepaths/writepaths.go`'s `UpdatePersonalData` (accept and forward `recordIdentity` instead of hardcoding `""`), `talenta-core`'s `services/fabric/AnchoringWriteService.php` (accept and forward `recordIdentity` in `submitAnchor()`).
- No chaincode changes.

### References

- [Source: _bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-11/prd.md#FR-1, §9 Personal-domain launch gate]
- [Source: _bmad-output/planning-artifacts/architecture/architecture-fabric-hris-talenta-2026-08-11/ARCHITECTURE-SPINE.md#AD-3, AD-4]
- [Source: _bmad-output/implementation-artifacts/tf-1-1-reliably-trigger-every-payroll-write-toward-fabric-anchoring.md — AnchorTriggerWorker/AnchoringWriteService established here]
- [Source: _bmad-output/implementation-artifacts/tf-2-0-add-per-recordidentity-pseudonym-derivation-to-the-fabric-side-write-path.md — recordIdentity mechanism this story is the first real caller of]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/controllers/api/web/MyInfoController.php:282-310,345-401,495-553 — read directly 2026-08-12]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/services/MyInfoServices.php:977-1015 — read directly 2026-08-12]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/services/MyInfoFamilyDataService.php:349-414 — read directly 2026-08-12]
- [Source: fabric-hris/integration-bridge/internal/pipeline/validate.go — requestEnvelope confirmed to lack recordIdentity, read directly 2026-08-12]

## Dev Agent Record

### Agent Model Used

Claude Sonnet 5 (claude-sonnet-5)

### Debug Log References

- `go build ./...`, `go vet ./...`, `go test ./...` all clean/passing in `write-path-integration/writepaths`.
- `go build ./...`, `go vet ./...`, `go test ./...` all clean/passing in `integration-bridge` (`cmd/integrationbridge` and `internal/pipeline` packages both `ok`).
- `php -l` clean on `controllers/api/web/MyInfoController.php`, `workers/AnchorTriggerWorker.php`, `services/fabric/AnchoringWriteService.php`, `tests/codeception/unit/workers/AnchorTriggerWorkerTest.php`.
- `talenta-core`'s Codeception unit-test harness remains broken in this environment — same pre-existing, confirmed-unrelated `session_set_save_handler` error as Story tf-1.1. Not re-diagnosed here since tf-1.1 already established it's an environment defect, not a regression.

### Completion Notes List

- **Two disclosed, deliberate gaps in this story's coverage — not oversights:**
  1. `actionImportDataEmergencyContact` (bulk import) is NOT anchored. `MyInfoServices::importDataEmergencyContact()` returns only a global success/failure, no per-row employee detail — anchoring here would misattribute the change to the importing admin, not the real affected employee(s). User-approved decision (2026-08-12): skip anchoring, document the gap in-code, rather than fabricate a misleading anchor. Follow-up: `importDataEmergencyContact()` needs to return per-row detail before this can close.
  2. Emergency Contact records use `recordIdentity=""` (self), meaning multiple emergency contacts for one employee currently share one hash chain. `AD-3` (Story tf-2.0) only defined per-record chaining for Family and Additional Info — Emergency Contact was never included in that decision. This is a real architecture gap, disclosed here, not resolved.
- **Scope grew beyond the original task list, as flagged in Dev Notes before implementation started:** completing Family's `recordIdentity` requirement required extending the wire format across `integration-bridge` (`validate.go`, `dispatch.go`, `handlers.go`) and `write-path-integration/writepaths` (all 5 `Hooks` methods, not just `UpdatePersonalData`, to keep the shared `DispatchFunc` type consistent) — see new Task 4. User approved proceeding as one story rather than splitting further.
- **`recordIdentity` value for Family resolved to `reference_id`** (an auto-incrementing-per-employee sequence number), not the DB row's own primary key — traced through `MyInfoFamilyDataService::saveData`/`setData` to confirm `reference_id` survives into the returned array unmutated, and is available in both the save and delete return shapes, so both actions use the same field consistently (a family member's create-then-delete chains under one pseudonym, not two).
- Extending `Hooks`' 5 methods' signatures (not just `UpdatePersonalData`) meant Education/Additional Info (Stories tf-2.3/tf-2.4) inherit `recordIdentity` support for free — they will not need to repeat this wire-format work.

### File List

- `controllers/api/web/MyInfoController.php` (modified, talenta-core) — 3 new enqueue call sites (`actionDeleteEmergencyContact`, `actionSaveFamilyData`, `actionDeleteFamilyData`); `actionImportDataEmergencyContact` deliberately NOT modified to anchor, comment added explaining why.
- `services/fabric/AnchoringWriteService.php` (modified, talenta-core) — `submitAnchor()` gained optional `recordIdentity` parameter.
- `workers/AnchorTriggerWorker.php` (modified, talenta-core) — forwards `recordIdentity` from `args`.
- `tests/codeception/unit/workers/AnchorTriggerWorkerTest.php` (modified, talenta-core) — updated existing assertion's arg count, added `testExecutePassesRecordIdentityWhenPresentInArgs`.
- `integration-bridge/internal/pipeline/validate.go` (modified, fabric-hris) — `requestEnvelope` + `validateRequest` gain `recordIdentity`.
- `integration-bridge/internal/pipeline/dispatch.go` (modified, fabric-hris) — `DispatchFunc` + `dispatch()` gain `recordIdentity`.
- `integration-bridge/internal/pipeline/handlers.go` (modified, fabric-hris) — forwards the new value.
- `integration-bridge/internal/pipeline/{dispatch,payroll_route,handlers,education_route,multiroute,additional_route,employment_route,validate}_test.go` (modified, fabric-hris) — updated to match new signatures.
- `write-path-integration/writepaths/writepaths.go` (modified, fabric-hris) — all 5 `Hooks` methods gain `recordIdentity`.
- `write-path-integration/writepaths/anchor_test.go` (modified, fabric-hris) — updated call sites to match.

## Change Log

- 2026-08-12: Story implemented (Tasks 1-4, including the wire-format extension discovered mid-implementation). 3 of 4 originally-scoped actions anchored (Emergency Contact delete, Family save, Family delete); bulk-import deliberately left unanchored with a disclosed reason. Full Go regression suite passes in both `fabric-hris` modules touched; PHP test execution blocked by the same pre-existing environment defect documented in Story tf-1.1.
- 2026-08-12 (bug fix, found during Story tf-2.4's implementation): `$familyData['data']` in both Family call sites was passed as `'newValue'` directly, but `AnchoringWriteService::submitAnchor()` requires `array $newValue` — `$familyData['data']` is actually an AR model object (confirmed via `MyInfoFamilyDataService::saveData`/`deleteData` tracing into `BaseRepository::save()`, which returns `$this->model`, not a bool or array), so this would have thrown a `TypeError` at runtime. Fixed to `$familyData['data']->toArray()` in both call sites (`MyInfoController.php`).

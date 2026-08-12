---
baseline_commit: e18ba49ca987bddad1966a40189568a90555db7a  # talenta-core (working tree also carries tf-1.1/2.1/2.2's uncommitted changes)
---

# Story tf-2.3: Wire Education & Experience into the anchoring pipeline (new trigger point)

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As an employee,
I want my education, certification, and work-experience changes anchored,
so that this domain gets the same tamper-evidence guarantee, even though it lives in controllers that have never had an anchoring hook.

## Acceptance Criteria

1. **Given** `FormalEducationController::actionSave`/`actionDelete`, `InformalEducationController::actionSave`/`actionUpdate`/`actionDelete`, and `WorkingExperienceController::actionSave`/`actionDelete` are the six well-understood, single-record mutating actions across these three controllers, **When** each commits, **Then** a new trigger call enqueues an `AnchorTriggerWorker` job tagged `profileSection=EDUCATION` — this is new wiring, not an extension of any Fabric-anchoring hook (none exists in any of the three controllers today; 5 of these 6 already fire an unrelated `EmploymentUpdateWebhookService` call, which stays untouched, same non-replacement pattern as every prior story).
2. **Given** each of the three record types (formal education, informal education, working experience) can have multiple entries per employee, **When** the anchoring job is enqueued, **Then** it carries `recordIdentity` = that record's own `reference_id` column (confirmed present and distinct from the DB primary key on all three underlying models), so each entry chains independently — extending the same pattern Story tf-2.1 established for Family data.
3. **Given** `FormalEducationController::actionImport`, `InformalEducationController::actionImport`, `WorkingExperienceController::actionImport`, and `*::actionUploadCertificate` are bulk/opaque-payload actions with no per-employee or per-record identifier surfaced at the controller level, **When** this story ships, **Then** none of them are anchored — same disclosed-gap pattern as Story tf-2.1's `actionImportDataEmergencyContact`, not silently attempted with a guessed identity.

## Tasks / Subtasks

- [x] Task 1: Add `$this->actor` to all three controllers (prerequisite for AC #1/#2)
  - [x] Subtask 1.1: None of `FormalEducationController`, `InformalEducationController`, `WorkingExperienceController` declares `$this->actor` — confirmed absent (unlike `MyInfoController`, which sets it). Added `$this->actor = Yii::$app->user->identity;` to each controller's existing `init()` method, alongside their existing service-construction lines. No name collision with any existing property in any of the three files.

- [x] Task 2: Wire `FormalEducationController` (AC: #1, #2)
  - [x] Subtask 2.1: `actionSave($id = null)` (`FormalEducationController.php:99-118`) already calls `EmploymentUpdateWebhookService::setUserId($id)->publish();` at line 109 — `$id` here is the EMPLOYEE id (same convention as `actionSaveFamilyData`: null on create, defaults to self). Added the enqueue call after line 109, using `$formalEdu['data']->reference_id` for `recordIdentity` (the model returned in `$formalEdu['data']` carries `reference_id` as a column per `EducationHistory`'s schema).
  - [x] Subtask 2.2: `actionDelete($id = null)` (`:120-134`) — **`$id` here means something different: it is the EDUCATION RECORD's own id, not the employee's** (confirmed via `FormalEducationServices::deleteFormalEducation`, which looks the record up by `$id` and resolves the owning employee separately). No existing webhook call in this action. Added the enqueue call using `$formalEdu['data']->user_id` for `employeeInternalID` (the returned model carries this) and `$formalEdu['data']->reference_id` for `recordIdentity`.

- [x] Task 3: Wire `InformalEducationController` (AC: #1, #2)
  - [x] Subtask 3.1: `actionSave()` (`:69-100`) and `actionUpdate()` (`:102-130`) already call `EmploymentUpdateWebhookService::setUserId($this->payload['user_id'])->publish();` — employee id is `$this->payload['user_id']` in BOTH (not a route param here, unlike Formal Education). `$result` in both is a `InformalEducationHistory` model with `->toArray()` already called — `$result->reference_id` is a direct attribute read. Added the enqueue call after each existing `publish()` call.
  - [x] Subtask 3.2: `actionDelete($id = null)` (`:132-151`) — `$id` is the record's own id (same asymmetry as Formal Education's delete). `MyInfoInformalEducationServices::delete($id)` returns a **bare boolean**, exposing no employee id at all — unlike the other two record types' delete methods. **Resolved by adding a read-only model lookup in the controller, before calling `delete()`** (the row must be read before it's removed): `InformalEducationHistory::findOne($id)` (the model class is already imported in this controller file) to capture `user_id`/`reference_id` before deletion. This is new controller-level code, not a change to the service layer — deliberately, to avoid an unknown blast radius from changing `delete()`'s return shape for other possible callers.

- [x] Task 4: Wire `WorkingExperienceController` (AC: #1, #2)
  - [x] Subtask 4.1: `actionSave($id = null)` (`:62-82`) already calls `EmploymentUpdateWebhookService::setUserId($id)->publish();` — `$id` is the employee id (same Family/Formal-Education convention). Added the enqueue call using `$workExp['data']->reference_id` for `recordIdentity`.
  - [x] Subtask 4.2: `actionDelete($id = null)` (`:84-98`) — `$id` is the record's own id. `MyInfoWorkingExperienceServices::deleteWorkingExperience($id)` returns `$workExp['data']` carrying `user_id`/`reference_id` as model attributes (same shape as Formal Education's delete, not Informal Education's bare-boolean case). Added the enqueue call using those.

- [x] Task 5: Confirm bulk/opaque actions are NOT anchored (AC: #3)
  - [x] Subtask 5.1: `FormalEducationController::actionImport`/`actionUploadCertificate`, `InformalEducationController::actionImport`/`actionUploadCertificate`, `WorkingExperienceController::actionImport` — none modified. Each takes only an opaque `$this->payload` with no per-employee or per-record identifier visible at the controller level, same disclosed-gap reasoning as Story tf-2.1's bulk-import decision. Documented, not silently attempted.

## Dev Notes

- **The `$id`-means-employee vs. `$id`-means-record asymmetry is the single most dangerous trap in this story.** `actionSave`'s `$id` (when present) is the employee being edited; `actionDelete`'s `$id` is the record being deleted. Using the wrong meaning for either would either anchor under a nonsensical "employee id" (a record's own auto-increment id passed as if it were a user id) or fail outright. Verified per-action via the exact service-layer trace in Dev Agent Record, not assumed from the parameter name alone.
- **All three underlying models (`EducationHistory`, `InformalEducationHistory`, `WorkingExperience`) have a `reference_id` column distinct from their AR primary key** (`id`) — confirmed via direct model inspection, not inferred. This is the same shape as Family data's `reference_id` (Story tf-2.1) and is used identically here.
- **No `writepaths`/`integration-bridge`/wire-format changes needed** — Story tf-2.1 already extended the full chain (`AnchorTriggerWorker` → `AnchoringWriteService` → bridge `requestEnvelope` → `writepaths.RecordEducationHistory`) to accept `recordIdentity` generically. This story is a pure application of that existing capability to a new domain, the same relationship Story tf-2.2 (Employment) had to Story tf-1.1 (Payroll) — except EDUCATION genuinely needs `recordIdentity` non-empty, unlike Employment.
- **`InformalEducationController`'s asymmetry with the other two:** its `delete($id)` service method is the ONLY one of the three record types' delete paths that returns nothing useful (`true`/`false`). The controller-level `InformalEducationHistory::findOne($id)` lookup added here is a deliberate, minimal, read-only workaround — it does not touch `MyInfoInformalEducationServices::delete()` itself, since that method's return shape may have other callers not audited in this story.
- **Operation types:** `actionSave` on `$id === null` is CREATE, otherwise UPDATE (same convention as Family data); `actionUpdate` is always UPDATE; `actionDelete` is always DELETE.

### Project Structure Notes

- Modified files: `controllers/api/web/FormalEducationController.php`, `controllers/api/web/InformalEducationController.php`, `controllers/api/web/WorkingExperienceController.php` (all talenta-core) — `init()` gains `$this->actor` in each; 6 new enqueue call sites total (2 per controller).
- No `fabric-hris` changes in this story (fully covered by Story tf-2.1's wire-format work).

### References

- [Source: _bmad-output/planning-artifacts/epics-talenta-fabric-2026-08-12.md#Epic 2, Story 2.3]
- [Source: _bmad-output/implementation-artifacts/tf-2-1-wire-personal-domains-remaining-actions-and-fix-the-family-additional-info-route-mapping.md — recordIdentity/reference_id pattern and bulk-action non-anchoring precedent, both reused here]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/controllers/api/web/FormalEducationController.php:20-25,99-134 — read directly 2026-08-12]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/controllers/api/web/InformalEducationController.php:22-27,69-151 — read directly 2026-08-12]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/controllers/api/web/WorkingExperienceController.php:17-21,62-98 — read directly 2026-08-12]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/services/FormalEducationServices.php:127-180, services/MyInfoInformalEducationServices.php:186,239,407, services/MyInfoWorkingExperienceServices.php:83-130 — read directly 2026-08-12]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/models/EducationHistory.php:20, models/InformalEducationHistory.php (reference_id in rules), models/WorkingExperience.php:18 — read directly 2026-08-12]

## Dev Agent Record

### Agent Model Used

Claude Sonnet 5 (claude-sonnet-5)

### Debug Log References

- `php -l` clean on `FormalEducationController.php`, `InformalEducationController.php`, `WorkingExperienceController.php`.
- Same pre-existing, confirmed-unrelated Codeception harness defect as Stories tf-1.1/tf-2.1/tf-2.2 — no test execution possible.

### Completion Notes List

- The `$id`-means-employee (save) vs. `$id`-means-record (delete) asymmetry held across all three controllers, confirmed by tracing each save/delete pair's service call — used consistently, not assumed.
- `InformalEducationController::actionDelete` required a genuinely different fix (a pre-delete model lookup) because its service's `delete()` method, uniquely among the three record types, returns a bare boolean with no employee/record identity at all. Handled entirely at the controller level, not by changing the service's return shape (unknown blast radius on other callers).
- All six wired actions use `reference_id` for `recordIdentity`, consistent with the Family-data precedent (Story tf-2.1) and directly confirmed present on all three underlying models (`EducationHistory`, `InformalEducationHistory`, `WorkingExperience`).
- 5 bulk/opaque actions (2 imports + 2 upload-certificate + 1 more import) deliberately left unanchored, same disclosed-gap reasoning as Story tf-2.1's bulk import.

### File List

- `controllers/api/web/FormalEducationController.php` (modified, talenta-core) — `$actor` property + `init()` assignment; 2 new enqueue call sites (`actionSave`, `actionDelete`).
- `controllers/api/web/InformalEducationController.php` (modified, talenta-core) — `$actor` property + `init()` assignment; 3 new enqueue call sites (`actionSave`, `actionUpdate`, `actionDelete` with pre-delete lookup).
- `controllers/api/web/WorkingExperienceController.php` (modified, talenta-core) — `$actor` property + `init()` assignment; 2 new enqueue call sites (`actionSave`, `actionDelete`).

## Change Log

- 2026-08-12: Story implemented (Tasks 1-5). 6 mutating actions across 3 controllers wired to the anchoring pipeline; 5 bulk/opaque actions deliberately left unanchored with disclosed reasons. `php -l` clean on all 3 files; no test execution possible (same pre-existing environment blocker as prior stories in this epic).
- 2026-08-12 (bug fix, found during Story tf-2.4's implementation): `$formalEdu['data']` and `$workExp['data']` were passed as `'newValue'` directly in all 4 Formal-Education/Working-Experience call sites, but both are AR model objects (confirmed via tracing `FormalEducationRepository`/`WorkingExperienceRepository`'s `setModel()`/`save()`, both returning `$this->model`), not arrays — `AnchoringWriteService::submitAnchor()` requires `array $newValue`, so this would have thrown a `TypeError` at runtime. Fixed to `->toArray()` in all 4 call sites. `InformalEducationController`'s 3 call sites were already correct (used `$response`/`->toArray()` from the start).

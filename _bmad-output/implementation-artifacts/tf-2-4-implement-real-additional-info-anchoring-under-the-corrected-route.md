---
baseline_commit: 013d68239cc576a402f70b455a24d2cdd02ab74f  # fabric-hris; talenta-core baseline: e18ba49ca987bddad1966a40189568a90555db7a (both working trees carry this epic's prior uncommitted stories)
---

# Story tf-2.4: Implement real Additional Info anchoring under the corrected route

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As an auditor,
I want custom-field value changes anchored under their own real Additional Info implementation,
so that Additional Info history is genuinely custom-field data, not accidentally the mislabeled Family-data hook.

## Acceptance Criteria

1. **Given** `AdditionalInfoController::actionSave` has no anchoring trigger today, **When** this story is implemented, **Then** a new trigger call is added, enqueuing an anchoring job through Story tf-1.1's pipeline.
2. **Given** `integration-bridge`'s `POST /v1/profile-sections/ADDITIONAL` route currently dispatches to `ApproveFamilyDataChange` (a mislabeled Family-data hook, already moved to `PERSONAL` in Story tf-2.1), **When** this story is implemented, **Then** the route dispatches to a new `writepaths.Hooks` method built for real Additional Info custom-field values instead (FR-Arch-2).
3. **Given** each custom field can be added/removed independently of the others, **When** the pseudonym is derived, **Then** it is `(employeeID, customFieldID)`, so each custom field's history chains independently of the others (FR-Arch-1).
4. **Given** an employee or HR Admin views Additional Info transaction history, **When** the request is made, **Then** the same FR-6/FR-7/FR-8 guarantees hold for this domain too — satisfied by the existing, unchanged read path.

## Tasks / Subtasks

- [x] Task 1: Add a new `writepaths.Hooks` method for real Additional Info (AC: #2, #3)
  - [x] Subtask 1.1: Added `UpdateAdditionalInfo(ctx, employeeInternalID, userID, recordIdentity string, newValue, document []byte)` to `writepaths.go`, following the exact same shape as `UpdatePersonalData`/`ApproveEmploymentTransfer` — `h.Store.SaveSection(ctx, employeeInternalID, "ADDITIONAL", newValue)` then `h.anchor(ctx, employeeInternalID, userID, "ADDITIONAL", recordIdentity, newValue, version, document)`. `recordIdentity` is REQUIRED here conceptually (each custom field needs its own pseudonym), but the method signature does not enforce non-empty — that's the caller's responsibility (Task 3), consistent with how `recordIdentity` is optional at the type level everywhere else.
  - [x] Subtask 1.2: `ApproveFamilyDataChange` left in place, UNCHANGED — it is now unreferenced by any route (Family moved to `PERSONAL` in Story tf-2.1), but not deleted; removing dead code is a separate cleanup decision, not this story's scope, and deleting it risks breaking any test that still references it directly.

- [x] Task 2: Repoint the `ADDITIONAL` route (AC: #2)
  - [x] Subtask 2.1: `cmd/integrationbridge/main.go`'s `POST /v1/profile-sections/ADDITIONAL` route's `Dispatch` field changed from `hooks.ApproveFamilyDataChange` to `hooks.UpdateAdditionalInfo`.

- [x] Task 3: Wire `AdditionalInfoController::actionSave` (AC: #1, #3)
  - [x] Subtask 3.1: `actionSave($id = null)` (`AdditionalInfoController.php:67-87`) already calls `EmploymentUpdateWebhookService::setUserId($id)->publish();` — `$id` is the employee (defaults to self, same Family/Payroll convention). Traced `MyInfoAdditionalInfoServices::saveAdditionalInfo`/`doSave` (`:143-172,174+`) and confirmed the payload is an ARRAY of `{custom_field_id, value, editing_method}` entries (matches the PRD's own §6 field description exactly) — ONE `actionSave` call can update MULTIPLE custom fields for one employee in a single request. `$additionalInfo['data']` is `$result['modelList']` — an array of saved models, each carrying its own `custom_field_id` attribute (confirmed: `$result->custom_field_id = $param['custom_field_id'];`, `MyInfoAdditionalInfoServices.php:197`).
  - [x] Subtask 3.2: Enqueue ONE `AnchorTriggerWorker` job PER model in `$additionalInfo['data']`, not one job for the whole batch — each tagged `profileSection=ADDITIONAL`, `recordIdentity` = that model's own `custom_field_id`, per AC #3's independent-chaining requirement. This is different from the bulk-import non-anchoring precedent (Story tf-2.1/tf-2.3): those cases had NO per-row identity available at all; this case does (`custom_field_id` on every model), so per-item anchoring is both possible and required here, not skipped.

## Dev Notes

- **Why a new method, not reusing `ApproveFamilyDataChange`'s body:** the method itself is fine (identical shape to every other `Hooks` method), but its NAME and doc comment describe Family data specifically — reusing it for Additional Info would be exactly the same kind of mislabeling this story exists to fix, just moved one level down instead of eliminated. A fresh, correctly-named method costs nothing extra (identical body) and removes the confusion entirely.
- **This story anchors per-field, not per-request** — a single `actionSave` call touching 3 custom fields enqueues 3 separate `AnchorTriggerWorker` jobs. This is intentional and required by AC #3, not an inefficiency to "optimize" away; each field's independent chain is the whole point of `recordIdentity`.
- **No `talenta-core` service-layer changes** — `saveAdditionalInfo`/`doSave` are read-only from this story's perspective; the controller loop reads `$additionalInfo['data']`, an already-existing return value, nothing added there.

### Project Structure Notes

- Modified files: `write-path-integration/writepaths/writepaths.go` (fabric-hris, new `UpdateAdditionalInfo` method), `integration-bridge/cmd/integrationbridge/main.go` (fabric-hris, route repoint), `controllers/api/web/AdditionalInfoController.php` (talenta-core, per-field enqueue loop).

### References

- [Source: _bmad-output/planning-artifacts/epics-talenta-fabric-2026-08-12.md#Epic 2, Story 2.4]
- [Source: _bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-11/prd.md#§6 Additional Info field shape]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/controllers/api/web/AdditionalInfoController.php:1-89 — read directly 2026-08-12]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/services/MyInfoAdditionalInfoServices.php:143-197 — read directly 2026-08-12]
- [Source: write-path-integration/writepaths/writepaths.go — ApproveFamilyDataChange's shape mirrored for the new method]
- [Source: integration-bridge/cmd/integrationbridge/main.go:79-80 — ADDITIONAL route registration]

## Dev Agent Record

### Agent Model Used

Claude Sonnet 5 (claude-sonnet-5)

### Debug Log References

- `go build ./...`, `go vet ./...`, `go test ./...` all clean/passing in both `write-path-integration/writepaths` and `integration-bridge` (including the pre-existing `additional_route_test.go`, which uses its own inline `Dispatch` stub and so is insensitive to which real `Hooks` method `main.go` wires — this is the known G-35 gap, not something this story's changes are covered by).
- `php -l` clean on `AdditionalInfoController.php` and (retroactively, for the bug found and fixed here) `MyInfoController.php`, `FormalEducationController.php`, `WorkingExperienceController.php`.

### Completion Notes List

- **Real bug found while implementing this story's own `newValue` line, then audited backward across the whole epic:** `AnchoringWriteService::submitAnchor()` requires `array $newValue`, but several earlier stories (tf-2.1 Family, tf-2.3 Formal Education/Working Experience) passed AR model objects directly — a `TypeError` waiting to happen at the first real invocation. Dispatched a verification pass across every affected service method before fixing anything, rather than guess; fixed all confirmed cases (see each story's own Change Log for its specific fix), including this story's own `$customFieldModel->toArray()`. `MyInfoServices::updateEmploymentData` (Story tf-2.2) was verified NOT affected — its return is a genuine plain array, confirmed by its consistent bracket-access usage pattern, not object property access.
- `ApproveFamilyDataChange` intentionally left in the codebase, unreferenced — a disclosed piece of dead code, not deleted, since removing it is a separate decision from this story's scope.

### File List

- `write-path-integration/writepaths/writepaths.go` (modified, fabric-hris) — new `UpdateAdditionalInfo` method; `ApproveFamilyDataChange` left in place with an updated doc comment noting it's now unreferenced.
- `integration-bridge/cmd/integrationbridge/main.go` (modified, fabric-hris) — `ADDITIONAL` route's `Dispatch` repointed.
- `controllers/api/web/AdditionalInfoController.php` (modified, talenta-core) — per-custom-field enqueue loop added to `actionSave`.

## Change Log

- 2026-08-12: Story implemented (Tasks 1-3). Full Go regression suite passes in both `fabric-hris` modules. Discovered and fixed a real `array`-vs-object `TypeError` bug spanning 3 prior stories in this epic (see each story's own Change Log).

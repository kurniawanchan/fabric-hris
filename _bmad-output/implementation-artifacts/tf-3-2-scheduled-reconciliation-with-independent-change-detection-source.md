---
baseline_commit: e18ba49ca987bddad1966a40189568a90555db7a  # talenta-core (working tree also carries epic-tf-1/2's uncommitted stories)
---

# Story tf-3.2: Scheduled reconciliation with independent, suppression-resistant change-detection source

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a security operator,
I want a scheduled sweep to catch anchors that silently never happened,
so that even an attacker who suppresses the anchoring trigger itself still gets caught.

## Acceptance Criteria

1. **Given** an anchoring job was dropped due to an unrecovered failure, or an attacker with direct DB access suppressed the anchoring trigger entirely, **When** the scheduled reconciliation job runs, **Then** it determines "what changed in Talenta" from the DB's own `updated_date` column — never from the anchoring trigger's own success/failure log — so a suppressed trigger does not also blind reconciliation (FR-11, AC-8).
2. **Given** a DB row's `updated_date` is newer than its latest on-chain anchor for that pseudonym, **When** reconciliation compares them, **Then** this is surfaced as a missing-anchor finding for retry or investigation, distinct from Story tf-3.1's hash-mismatch finding — and this works even for a wholly new record that was never anchored at all, since its absence is itself the finding.
3. **Given** Personal and Payroll are the highest-sensitivity domains, **When** the reconciliation schedule runs, **Then** it runs hourly for Personal and Payroll, and daily for Employment (NFR-7) — **scope correction, see Dev Notes:** Education & Experience and Additional Info are NOT covered by this story; their underlying tables have no discoverable "last modified" timestamp column at all.

## Tasks / Subtasks

- [x] Task 1: Build the reconciliation console command (AC: #1, #2)
  - [x] Subtask 1.1: New `commands/AnchorReconciliationController.php` (Yii2 console controller, matching the real, already-established convention — every existing console command in this codebase lives in `commands/*Controller.php` extending `yii\console\Controller`; confirmed via `HelloController.php` and 20+ sibling files).
  - [x] Subtask 1.2: `actionRun($domain, $sinceMinutes)` — queries `User::find()->where(['>=', 'updated_date', $cutoff])` for `$domain` in `{PERSONAL, EMPLOYMENT, PAYROLL}` (all three share `tbl_user`'s single `updated_date` column — confirmed: no separate per-domain timestamp exists on this table), or `FamilyData::find()->where(['>=', 'updated_date', $cutoff])` for `$domain = FAMILY` (a sub-case of PERSONAL, per Story tf-2.1's `recordIdentity` convention).
  - [x] Subtask 1.3: For each matched row, resolve `employeeInternalID`/`recordIdentity` using the EXACT SAME rules already established per domain in Stories tf-1.1/tf-2.1/tf-2.2 (User rows: `employeeInternalID = id`, `recordIdentity = ""`; FamilyData rows: `employeeInternalID = reference_user_id_fk`, `recordIdentity = reference_id`) — reconciliation must resolve pseudonyms the same way the write path does, or it would be comparing against the wrong chain entirely.
  - [x] Subtask 1.4: Call `integration-bridge`'s EXISTING `GET /v1/profile-sections/history` route (built in the original read-path work, reused here — no new bridge endpoint needed) with the resolved `employeeInternalID`/`profileSection`/`recordIdentity` (query param), find the latest entry's `timestamp`. If no history exists OR the latest anchor's timestamp is older than the row's `updated_date` (with a configurable grace-period buffer for clock skew and normal anchoring lag), log a missing-anchor finding via `Yii::warning(...)` — **not a persisted, queryable record yet**; that is Story tf-4.2's explicit scope (dead-letter/operator-visibility persistence), not duplicated here.

- [x] Task 2: Wire the domain-differentiated cadence (AC: #3)
  - [x] Subtask 2.1: Document the required crontab entries in the command's own doc comment: `0 * * * * php yii anchor-reconciliation/run PERSONAL 60` and `... PAYROLL 60` (hourly, NFR-7), `0 0 * * * php yii anchor-reconciliation/run EMPLOYMENT 1440` (daily). **Actual crontab registration is infrastructure/deployment work, out of this codebase's scope** — no crontab file exists anywhere in this repository to add these lines to; documenting the exact commands an operator must schedule is what this story can concretely deliver.

- [x] Task 3: Document the Education/Experience/Additional-Info scope gap explicitly (AC: #3's correction)
  - [x] Subtask 3.1: `EducationHistory`, `InformalEducationHistory`, `WorkingExperience`, and `EmployeeCustomField` models have NO discoverable timestamp column anywhere (`@property` docblock, `rules()`, or `behaviors()`) — confirmed by direct inspection, not assumed. Reconciliation for these three domains requires either a DB migration (adding a tracked "last modified" column) or a different detection mechanism entirely — both out of this story's scope. Documented in the command's own doc comment and here, not silently narrowed without a trace.

## Dev Notes

- **`FamilyData::$updated_date`'s reliability is itself unverified, not just assumed reliable.** The column exists in the model's `@property` docblock (`models/FamilyData.php:22`) but is set nowhere in the model or `FamilyDataRepository`/`MyInfoFamilyDataService` (confirmed by direct search) — it is very likely maintained by a DB-level mechanism (e.g. MySQL's `ON UPDATE CURRENT_TIMESTAMP`) that application code never needs to touch, but this codebase's own PHP files provide no positive confirmation either way. **This must be verified against the actual live database schema (not just PHP source) before this story is trusted in production** — flagged explicitly, not silently assumed correct.
- **`User::$updated_date` is genuinely manually-set application code**, not a Yii2 `TimestampBehavior` — confirmed no `behaviors()` method exists in `User.php`. This means its reliability depends on every code path that mutates a user row actually setting it — a broader claim than this story can verify for the whole codebase, only that the column exists and is a `'safe'`-rule attribute.
- **Why the bridge's history route, not a new endpoint:** `GET /v1/profile-sections/history` already returns exactly what's needed (the latest anchor's timestamp, keyed by the correctly-resolved pseudonym) — building a new endpoint duplicating this would violate the same "reuse, don't reinvent" principle every prior story in this epic followed.
- **Grace period is a real, unresolved number, not invented here:** the exact buffer (to avoid a false "missing anchor" finding for a row that changed 30 seconds ago and simply hasn't been picked up by its `AnchorTriggerWorker` job yet) depends on the queue's actual processing latency, which this story has no measured baseline for (that's Story tf-4.3's scope). Implemented as a configurable parameter, not hardcoded, with a placeholder default flagged as needing real-world tuning.
- **This story does NOT implement persisted, queryable missing-anchor records** — `Yii::warning()` logging is the floor this story delivers; Story tf-4.2 (dead-letter persistence) is the natural place to give these findings the same "visible to operators, re-driveable" treatment FR-13 already requires for dead-lettered anchoring jobs. Not duplicated here.

### Project Structure Notes

- New file: `commands/AnchorReconciliationController.php` (talenta-core).
- No `fabric-hris` changes — the existing history route is reused as-is.

### References

- [Source: _bmad-output/planning-artifacts/epics-talenta-fabric-2026-08-12.md#Epic 3, Story 3.2]
- [Source: _bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-11/prd.md#FR-11, NFR-7, AC-8]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/models/User.php:947 — updated_date rule, read directly 2026-08-13]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/models/FamilyData.php:21-22 — updated_date docblock, read directly 2026-08-13]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/models/EducationHistory.php, InformalEducationHistory.php, WorkingExperience.php, EmployeeCustomField.php — confirmed NO timestamp column, read directly 2026-08-13]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/commands/HelloController.php — console command convention]
- [Source: integration-bridge/internal/pipeline/history.go — reused GET /v1/profile-sections/history route]

## Dev Agent Record

### Agent Model Used

Claude Sonnet 5 (claude-sonnet-5)

### Debug Log References

- `php -l` clean on `commands/AnchorReconciliationController.php` and `services/fabric/BlockchainAnchoringService.php` (talenta-core).
- `go build ./...`, `go vet ./...`, `go test ./...` all clean/passing in `integration-bridge`, including a new test (`TestRegisterProfileHistoryRoute_RecordIdentity_ResolvesDistinctPseudonym`) proving the history-route fix below actually works, plus the full pre-existing suite (regression-safe).

### Completion Notes List

- **Found and fixed a real, pre-existing gap while implementing this story, not invented for it:** `GET /v1/profile-sections/history` (built well before `recordIdentity` existed, Stories tf-2.0/2.1) never accepted a `recordIdentity` parameter — it always resolved the "self" pseudonym, silently wrong for any caller (this reconciliation command, or a real end-user viewing a specific family member's own history) needing a sub-record's pseudonym. Fixed by threading an optional `recordIdentity` query param through to `writepaths.ResolveEmployeeID` (the same shared helper Story tf-3.1 already established), with a new test proving the fix. `BlockchainAnchoringService::getTransactionHistory()` (the Talenta-side caller) also updated to accept and forward it, defaulting to `''` for full backward compatibility with existing callers.
- **Scope corrected from the epics doc's original "all 5 domains" framing to what's actually implementable**, per user decision: Education & Experience and Additional Info are excluded, with the exact reason (no timestamp column exists anywhere in their models) documented in the command's own doc comment, not silently dropped.
- **Grace period exists to prevent false positives** for a row that changed seconds ago and simply hasn't been picked up by its queue job yet — implemented as a parameter, not hardcoded, since no real queue-latency baseline exists to tune it against yet.
- Findings are logged via `Yii::warning()` only — not yet a persisted, queryable record. That's explicitly Story tf-4.2's scope, not duplicated here.

### File List

- `commands/AnchorReconciliationController.php` (new, talenta-core) — the reconciliation console command.
- `services/fabric/BlockchainAnchoringService.php` (modified, talenta-core) — `getTransactionHistory()` gains optional `recordIdentity`.
- `integration-bridge/internal/pipeline/history.go` (modified, fabric-hris) — accepts and resolves `recordIdentity`; removed now-unused `gatewayclient` import.
- `integration-bridge/internal/pipeline/history_route_test.go` (modified, fabric-hris) — 1 new test.

## Change Log

- 2026-08-13: Story implemented (Tasks 1-3). Found and fixed a real pre-existing gap in the shared history route (missing `recordIdentity` support) as part of this story's own dependency chain. Scope corrected to the 2 domains (User-backed: Personal/Employment/Payroll, and Family) where a real timestamp signal exists; the other 3 domains' absence of any timestamp column is documented, not silently narrowed. `php -l` clean; full Go regression suite passes in `fabric-hris`.

---
baseline_commit: 013d68239cc576a402f70b455a24d2cdd02ab74f  # fabric-hris (working tree also carries epic-tf-2's uncommitted stories)
---

# Story tf-3.1: On-demand integrity verification

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a compliance auditor,
I want to check any employee record's current data against its latest anchor on demand,
so that I can confirm data integrity or detect tampering immediately when asked.

## Acceptance Criteria

1. **Given** an employee's data in Talenta's DB has been modified outside the authorized workflow (out-of-band tampering), **When** verification runs against that employee+domain+recordIdentity, **Then** it recomputes the current digest, compares it to the latest on-chain anchor at that same pseudonym, and reports a hash mismatch as "compromised/inconsistent" (FR-10, AC-4).
2. **Given** the pseudonym resolution for a given employee+domain+recordIdentity, **When** verification runs, **Then** it resolves the pseudonym identically to how the write path (AD-3) does — never via a second, independent derivation (AD-8).
3. **Given** the latest anchor for an employee+domain+recordIdentity has no way to be flagged as a deletion on-chain (`operationType` is off-chain-only metadata, AD-5 — no chaincode field exists for it), **When** the caller asserts a deletion by omitting `currentValue` from the request, **Then** verification reports "deletion verified" and stops — it does NOT attempt to hash-compare against an absent value.
4. **Given** no anchor exists yet for a given employee+domain+recordIdentity, **When** verification runs, **Then** it reports this distinctly (`no_anchor_found`) rather than as a hash mismatch, since there is nothing yet to compare against.

## Tasks / Subtasks

- [x] Task 1: Extract a shared pseudonym-resolution helper (AC: #2)
  - [x] Subtask 1.1: Extracted the compound-key derivation previously inline inside `writepaths.doAnchor` into two new exported functions in `writepaths.go`: `ResolveEmployeeKey(ctx, keys, employeeInternalID, recordIdentity) ([]byte, error)` (raw key bytes) and `ResolveEmployeeID(ctx, keys, employeeInternalID, recordIdentity) (string, error)` (the derived pseudonym, via `gatewayclient.ComputeEmployeeID`). `doAnchor` itself now calls `ResolveEmployeeKey` (it needs the raw key for `ComputeUpdatedBy` too, not just the ID) — refactor verified behavior-neutral by the full pre-existing `writepaths` test suite still passing unchanged.
  - [x] Subtask 1.2: This satisfies AD-8's "resolve the pseudonym identically to how the write path does" for real, not by convention alone — the verify route (Task 2) calls this exact same function, not a hand-copied version of the compound-key logic.

- [x] Task 2: Build the verify route in `integration-bridge` (AC: #1, #3, #4)
  - [x] Subtask 2.1: New file `internal/pipeline/verify.go`. Request envelope: `{employeeInternalID, profileSection, recordIdentity (optional), currentValue (optional, nullable — omitted/null means "caller asserts this record is deleted")}`.
  - [x] Subtask 2.2: Resolve `employeeID` via `writepaths.ResolveEmployeeID` (Task 1). Call `cfg.Ledger.EvaluateGetProfileHistory(ctx, cfg.TenantID, employeeID, profileSection)` (same `LedgerHistoryReader` interface `history.go` already defines — reused, not duplicated) and pick the highest-`Version` entry as "latest."
  - [x] Subtask 2.3: No history entries at all → `status: "no_anchor_found"` (AC #4) — distinct from a hash mismatch, since there's nothing to compare against yet (this is FR-11/reconciliation's job to catch, not FR-10's).
  - [x] Subtask 2.4: `currentValue` omitted/null in the request → `status: "deletion_verified"` (AC #3), regardless of what the latest anchor's hash says — no hash-compare attempted. This is the resolution to the "no on-chain operationType" gap: the caller (who has application-level knowledge of whether the record still exists) states it explicitly, rather than the bridge trying to infer it from ledger data that doesn't carry it.
  - [x] Subtask 2.5: `currentValue` present → fetch the salt for the latest anchor's exact version via `cfg.Salts.GetSalt(ctx, employeeID, profileSection, latest.Version)`, recompute `gatewayclient.ComputeDataHash(salt, currentValueJSON)`, compare to `latest.DataHash`. Match → `status: "verified"`. Mismatch → `status: "compromised"` (AC #1/AC-4 — "reported as compromised/inconsistent").
  - [x] Subtask 2.6: Registered `POST /v1/profile-sections/verify` in `buildMux` (`main.go`), reusing `hooks.Keys`/`hooks.Salts`/the same `historyReader` (`gw`) already wired for the history route — no new dependency construction.

## Dev Notes

- **This story does NOT implement FR-11 (scheduled reconciliation)** — that is Story tf-3.2's separate scope. This story is purely the on-demand, single-record check.
- **Why the caller states deletion explicitly, not the bridge inferring it:** the chaincode has no `operationType` field (a deliberate design choice, AD-5 — it's off-chain metadata only). There is therefore no way for the bridge to know from ledger data alone whether the latest anchor was a DELETE. The caller (Talenta) already knows this — it's the one deciding what `currentValue` to send. Omitting it is the caller's own assertion, not something the bridge derives. This was a deliberate design decision (confirmed with the user, 2026-08-13), not a default assumed silently.
- **`ResolveEmployeeKey`/`ResolveEmployeeID` extraction is a refactor of existing, tested code** — verify their extraction didn't change `doAnchor`'s behavior by running the full pre-existing `writepaths` test suite (not just the two new AD-3 tests from Story tf-2.0) before considering Task 1 done.
- **No `talenta-core` changes in this story** — verification is purely a new `integration-bridge` capability. A future story (not yet scoped) would need to add a Talenta-side caller of this new route; that caller doesn't exist yet and is out of this story's scope.

### Project Structure Notes

- Modified: `write-path-integration/writepaths/writepaths.go` (2 new exported functions, `doAnchor` refactored to use one of them).
- New: `integration-bridge/internal/pipeline/verify.go`.
- Modified: `integration-bridge/cmd/integrationbridge/main.go` (new route registration in `buildMux`).

### References

- [Source: _bmad-output/planning-artifacts/epics-talenta-fabric-2026-08-12.md#Epic 3, Story 3.1]
- [Source: _bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-11/prd.md#FR-10]
- [Source: _bmad-output/planning-artifacts/architecture/architecture-fabric-hris-talenta-2026-08-11/ARCHITECTURE-SPINE.md#AD-3, AD-5, AD-8]
- [Source: integration-bridge/internal/pipeline/history.go — LedgerHistoryReader interface and pseudonym-resolution pattern reused]
- [Source: write-path-integration/writepaths/writepaths.go — doAnchor's pre-refactor compound-key logic]

## Dev Agent Record

### Agent Model Used

Claude Sonnet 5 (claude-sonnet-5)

### Debug Log References

- `go build ./...`, `go vet ./...`, `go test ./...` all clean/passing in `write-path-integration/writepaths` (including the full pre-existing suite, proving the `ResolveEmployeeKey`/`ResolveEmployeeID` extraction is behavior-neutral).
- `go build ./...`, `go vet ./...`, `go test ./...` all clean/passing in `integration-bridge` (both packages), including 5 new tests in `verify_route_test.go` run individually with `-v` to confirm genuine pass, not just compile: matching-digest→verified, mismatched-digest→compromised, omitted-currentValue→deletion_verified (with an explicit assertion that `Salts.GetSalt` is never called in that path), no-anchor→no_anchor_found, auth-failure→never reaches any dependency.

### Completion Notes List

- **Interface-narrowing correction found during test-writing, not left as a silent design flaw:** `ResolveEmployeeKey`/`ResolveEmployeeID` were initially typed to accept the full 2-method `keystore.EmployeeKeyStore`, which would have forced every caller (including this route's own test stubs) to implement `DeleteEmployeeKey` too, defeating the narrow-interface testability pattern `history.go`'s `EmployeeKeyResolver` already established. Fixed by introducing `writepaths.EmployeeKeyGetter` (1-method interface) as the actual parameter type — `keystore.EmployeeKeyStore` still satisfies it structurally, so `doAnchor`'s real call site needed no change, but the verify route's stub-based tests no longer need an unnecessary type assertion.
- Reused `history.go`'s already-defined `chaincodeHistoryEntry`/`isKnownProfileSection` and the `LedgerHistoryReader` interface directly (same package) — no duplicate decoding logic.

### File List

- `write-path-integration/writepaths/writepaths.go` (modified) — new `EmployeeKeyGetter` interface, `ResolveEmployeeKey`/`ResolveEmployeeID` exported functions; `doAnchor` refactored to call `ResolveEmployeeKey`.
- `integration-bridge/internal/pipeline/verify.go` (new) — the verify route.
- `integration-bridge/internal/pipeline/verify_route_test.go` (new) — 5 tests.
- `integration-bridge/cmd/integrationbridge/main.go` (modified) — new route registered in `buildMux`.

## Change Log

- 2026-08-13: Story implemented (Tasks 1-2). Full Go regression suite passes in both `fabric-hris` modules, including 5 new tests for the verify route covering every AC. Found and fixed a real interface-design mismatch during test-writing (see Completion Notes) before it could propagate.

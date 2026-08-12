---
baseline_commit: 013d68239cc576a402f70b455a24d2cdd02ab74f  # fabric-hris repo (this story's target)
---

# Story tf-2.0: Add per-`recordIdentity` pseudonym derivation to the Fabric-side write path

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the architecture's own tamper-evidence guarantee (AD-3),
I want each family member's (and later, each custom field's) data to chain independently under one shared employee+domain,
so that a change to one family member's record cannot be confused with, or hide behind, another's on-chain history.

## Acceptance Criteria

1. **Given** `writepaths.Hooks.anchor`/`doAnchor` currently take no `recordIdentity` parameter, **When** this story is implemented, **Then** both gain an optional `recordIdentity string` parameter (empty string = today's existing single-record behavior — exact backward compatibility, not a new default to reason about). [Source: architecture-fabric-hris-talenta-2026-08-11/ARCHITECTURE-SPINE.md AD-3]
2. **Given** `doAnchor` currently calls `h.Keys.GetOrCreateEmployeeKey(ctx, employeeInternalID)` (`writepaths.go:206`), **When** `recordIdentity` is non-empty, **Then** it instead calls `GetOrCreateEmployeeKey(ctx, compoundKey)` where `compoundKey` deterministically combines `employeeInternalID` and `recordIdentity` — **no change to the `EmployeeKeyStore` interface or `ComputeEmployeeID`**; both already work correctly on whatever string/key bytes they're given.
3. **Given** existing callers (`UpdatePersonalData`, `ApproveEmploymentTransfer`, `RecordEducationHistory`, `ApproveFamilyDataChange`, `UpdatePayrollBankAccount`) pass no `recordIdentity` today, **When** this story ships, **Then** every one of them continues to resolve the exact same pseudonym as before — verified by a regression test, not by inspection alone.
4. **Given** the read path (`integration-bridge/internal/pipeline/history.go:142-147`) resolves pseudonyms via `GetOrCreateEmployeeKey` + `ComputeEmployeeID` for the single-record case, **When** a future story (tf-2.1) needs to look up a family member's history, **Then** the same compound-key convention this story establishes is the one it must use — this story does not implement the read side, only establishes the convention clearly enough for tf-2.1 to reuse it correctly.

## Tasks / Subtasks

- [x] Task 1: Add `recordIdentity` to the anchor/doAnchor call chain (AC: #1, #2)
  - [x] Subtask 1.1: Change `func (h *Hooks) anchor(ctx, employeeInternalID, userID, profileSection string, sectionValueJSON []byte, version int, document []byte)` (`writepaths.go:174`) to `anchor(ctx, employeeInternalID, userID, profileSection, recordIdentity string, sectionValueJSON []byte, version int, document []byte)` — thread `recordIdentity` through to `doAnchor` unchanged (same wrapping/error-handling shape, no other behavior change)
  - [x] Subtask 1.2: Same signature change to `doAnchor` (`writepaths.go:205`)
  - [x] Subtask 1.3: Inside `doAnchor`, before calling `h.Keys.GetOrCreateEmployeeKey`, compute `keyLookupID := employeeInternalID` if `recordIdentity == ""`, else a compound of both — used `employeeInternalID + "\x00" + recordIdentity`
  - [x] Subtask 1.4: Call `h.Keys.GetOrCreateEmployeeKey(ctx, keyLookupID)` instead of `GetOrCreateEmployeeKey(ctx, employeeInternalID)` — confirmed by re-reading `doAnchor`'s full body that no other line needed changing
  - [x] Subtask 1.5: `h.DocumentKeys.GetOrCreateDocumentKey(ctx, employeeInternalID)` (`writepaths.go:230`) left untouched, confirmed by diff review

- [x] Task 2: Update every existing `Hooks` method's call site (AC: #3)
  - [x] Subtask 2.1: All 5 call sites (`writepaths.go:268,277,286,296,305`) updated to pass `""` for `recordIdentity`, same position (before `sectionValueJSON`). No new "identity-aware" caller added.

- [x] Task 3: Regression test (AC: #3)
  - [x] Subtask 3.1: `TestAnchor_EmptyRecordIdentity_MatchesPreExistingSingleRecordPseudonym` added to `anchor_test.go`, using the existing `fakeLedger`/`newTestHooksAndFakes()` conventions exactly. Passes.
  - [x] Subtask 3.2: `TestAnchor_DifferentRecordIdentities_ProduceDifferentPseudonyms` added, asserting two different `recordIdentity` values for the same employee produce two different pseudonyms, and that repeating the same `recordIdentity` resolves back to the same pseudonym. Passes.

## Dev Notes

- **This story touches `fabric-hris` only** (`write-path-integration/writepaths/`) — no `talenta-core` changes, no `integration-bridge` request-envelope changes yet (that's deferred to tf-2.1, which is the first real caller passing a non-empty `recordIdentity` from outside `writepaths`). Keep this story's diff scoped to `writepaths.go` and its own tests.
- **Why not change `EmployeeKeyStore`'s interface:** the original architecture spine (AD-3) implied a store-level change ("a new lookup method" or similar). Investigation during story-writing found this unnecessary and worse: `GetOrCreateEmployeeKey(ctx, employeeInternalID string) ([]byte, error)` (`keystore/keystore.go:79-83`) already accepts an arbitrary string — it has no semantic dependency on that string literally being an employee ID, it just needs a unique key-space identifier. Composing the compound key *before* calling this interface, entirely inside `doAnchor`, achieves the identical outcome (a distinct key per `recordIdentity`) with zero interface changes, zero ripple into `InMemoryEmployeeKeyStore` or any future persistent implementation, and zero risk to `DocumentKeyStore`/`SaltStore` (untouched). Prefer this over the interface-change approach implied by the epics doc's original story text — that text is now superseded by this Dev Notes section for implementation purposes.
- **Full body of the function you're modifying** (`writepaths.go:205-253`, read directly 2026-08-12):
  ```go
  func (h *Hooks) doAnchor(ctx context.Context, employeeInternalID, userID, profileSection string, sectionValueJSON []byte, version int, document []byte) ([]byte, error) {
      employeeKey, err := h.Keys.GetOrCreateEmployeeKey(ctx, employeeInternalID)
      // ... employeeID := gatewayclient.ComputeEmployeeID(employeeKey)
      // ... salt := h.Salts.PutSalt(ctx, employeeID, profileSection, version)
      // ... dataHash := gatewayclient.ComputeDataHash(salt, sectionValueJSON)
      // ... updatedBy := gatewayclient.ComputeUpdatedBy(employeeKey, userID)
      // ... (IPFS document handling, keyed on employeeInternalID — leave untouched)
      // ... buildArgs(prevHash) — tenantID, employeeID, profileSection, dataHash, prevHash, updatedBy, ipfsCIDsJSON, ...
      return h.Gateway.SubmitRecordProfileSection(ctx, tenantID, employeeID, profileSection, buildArgs)
  }
  ```
  Every downstream value (`employeeID`, `salt`, `dataHash`, `updatedBy`, the chaincode call itself) is already derived from `employeeKey`/`employeeID` — changing only the input to `GetOrCreateEmployeeKey` is genuinely sufficient. Do not touch `ComputeEmployeeID`, `ComputeDataHash`, `ComputeUpdatedBy`, `PutSalt`, or `SubmitRecordProfileSection`'s own signatures.
- **`EmployeeKeyStore`'s real interface** (`write-path-integration/keystore/keystore.go:79-88`, unmodified by this story):
  ```go
  type EmployeeKeyStore interface {
      GetOrCreateEmployeeKey(ctx context.Context, employeeInternalID string) ([]byte, error)
      DeleteEmployeeKey(ctx context.Context, employeeInternalID string) error
  }
  ```
  `DeleteEmployeeKey` is out of scope for this story — a compound-keyed employee will get compound-keyed entries in whatever store implements this interface; erasure semantics for a specific `recordIdentity` (e.g. "forget just this one family member's key, not the whole employee's") are not addressed here and should be flagged as a follow-up if erasure is ever needed for a sub-record identity, not silently assumed solved.
- **What tf-2.1 (the next story) will need from this one:** a `recordIdentity` parameter on `anchor`/`doAnchor` that it can pass a real family-member `reference_id` into, and confidence that passing `""` for every other domain doesn't change anything. That is exactly this story's AC #1 and #3 — tf-2.1 should not need to touch `writepaths.go`'s pseudonym-derivation logic at all, only call the now-parameterized `anchor` with a real value.

### Project Structure Notes

- Modified file: `write-path-integration/writepaths/writepaths.go` — `anchor`, `doAnchor` signatures gain one parameter; all 5 existing `Hooks` methods' call sites updated to pass `""`.
- Extended test file: `write-path-integration/writepaths/anchor_test.go` — confirmed to exist (2026-08-12) and is the correct place for both Task 3 tests; do not create a new test file for this.
- No changes to `integration-bridge/`, `talenta-core`, or the chaincode in this story.

### References

- [Source: _bmad-output/planning-artifacts/architecture/architecture-fabric-hris-talenta-2026-08-11/ARCHITECTURE-SPINE.md#AD-3, AD-8]
- [Source: _bmad-output/planning-artifacts/epics-talenta-fabric-2026-08-12.md#Epic 2, Story 2.0 (scope-split note)]
- [Source: write-path-integration/writepaths/writepaths.go:174-302 — read directly 2026-08-12]
- [Source: write-path-integration/keystore/keystore.go:79-88 — read directly 2026-08-12]
- [Source: write-path-integration/gateway-client/digestbuilder.go:80-87 — read directly 2026-08-12]
- [Source: integration-bridge/internal/pipeline/history.go:142-147 — read directly 2026-08-12]

## Dev Agent Record

### Agent Model Used

Claude Sonnet 5 (claude-sonnet-5)

### Debug Log References

- `go build ./...`, `go vet ./...`, `go test ./...` all clean in `write-path-integration/writepaths`.
- `go build ./...` clean in `write-path-integration/gateway-client`, `write-path-integration/keystore`, `write-path-integration/ipfsclient` (unaffected, confirmed).
- `go build ./...` clean in `integration-bridge` (confirms `Hooks`' exported method signatures are unchanged — only the unexported `anchor`/`doAnchor` gained a parameter, so the module that imports `writepaths` as a dependency needed no changes and none were made).
- Full `writepaths` test suite (`go test ./...`) passes, including every pre-existing test (`TestDoAnchor_HappyPath_*`, `TestDoAnchor_AllFiveHooksSubmitTheirOwnProfileSection`, `TestDoAnchor_EachFailurePointProducesPartialFailureError`, etc.) — genuine regression proof, not just the two new tests.

### Completion Notes List

- Superseded the epics doc's original implied design (change `EmployeeKeyStore`'s interface) with a minimal-diff alternative discovered during story-writing: compose the compound key (`employeeInternalID + "\x00" + recordIdentity`) entirely inside `doAnchor`, before calling the existing, unmodified `GetOrCreateEmployeeKey(ctx, string)`. Zero interface changes anywhere; `ComputeEmployeeID`, `PutSalt`, `ComputeDataHash`, `ComputeUpdatedBy`, and the chaincode call all already derive from this call's output, so they needed no changes either — confirmed by re-reading `doAnchor`'s full body before touching anything.
- NUL byte (`\x00`) chosen as the compound-key separator. Not independently verified against every possible real `employeeInternalID`/`recordIdentity` value in this story (no real caller exists yet — tf-2.1 is the first), but it's a safe choice for typical string identifiers (UUIDs, numeric IDs, `reference_id` values) since none of those legitimately contain a NUL byte. Flagging this assumption explicitly rather than treating it as fully proven, per Subtask 1.3's own instruction.
- Document-key derivation (`DocumentKeys.GetOrCreateDocumentKey`) deliberately left keyed on the real `employeeInternalID` alone, unaffected by `recordIdentity` — no per-family-member document use case exists yet, and changing this without one would be speculative.
- This story only builds the mechanism; no caller anywhere yet passes a non-empty `recordIdentity`. That's intentionally tf-2.1's job (the next story), not this one's.

### File List

- `write-path-integration/writepaths/writepaths.go` (modified) — `anchor`/`doAnchor` gain a `recordIdentity` parameter; compound-key derivation added inside `doAnchor`; all 5 existing `Hooks` methods' call sites updated to pass `""`.
- `write-path-integration/writepaths/anchor_test.go` (modified) — 2 new tests appended: `TestAnchor_EmptyRecordIdentity_MatchesPreExistingSingleRecordPseudonym`, `TestAnchor_DifferentRecordIdentities_ProduceDifferentPseudonyms`.

## Change Log

- 2026-08-12: Story implemented (Tasks 1-3). Compound-key pseudonym derivation added to `doAnchor`, all existing call sites updated, 2 new tests added and passing alongside the full existing regression suite. Build/vet/test all clean in `writepaths`, `gateway-client`, `keystore`, `ipfsclient`, and the dependent `integration-bridge` module.

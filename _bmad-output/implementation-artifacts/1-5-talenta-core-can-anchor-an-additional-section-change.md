---
baseline_commit: a6f0cd63fcfe602255a143b938a55c1bbba583a6
---

# Story 1.5: talenta-core Can Anchor an ADDITIONAL Section Change

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As talenta-core's ADDITIONAL write path (raw SQL on the PHP side, not ActiveRecord),
I want the same contract,
so that family/dependents changes anchor regardless of PHP-side persistence mechanics.

## Acceptance Criteria

1. **Given** a valid `POST /v1/profile-sections/ADDITIONAL` request with correct `X-Api-Key`/`X-Company-ID`, **when** submitted with a correct envelope, **then** the bridge dispatches to `Hooks.ApproveFamilyDataChange` and returns the same `status` contract as Story 1.2. [Source: epics.md Story 1.5]
2. **Given** talenta-core's own PHP persists this section via raw SQL rather than ActiveRecord, **when** the bridge processes an ADDITIONAL request, **then** behavior is identical regardless of that PHP-side mechanic — the bridge only ever sees the HTTP request, never how talenta-core persisted anything on its own side. [Source: epics.md Story 1.5]

## Tasks / Subtasks

- [x] Task 1: Register the ADDITIONAL route (AC: #1, #2)
  - [x] Added `registerProfileSectionRoute(mux, "POST /v1/profile-sections/ADDITIONAL", routeConfig{ProfileSection: "ADDITIONAL", Dispatch: hooks.ApproveFamilyDataChange}, authCfg, hooks.TenantID, dispatchTimeout)` in `main.go`
  - [x] Confirmed `"ADDITIONAL"` used throughout (route path, `ProfileSection`, test names) — never `"FAMILY"` — and added a dedicated test proving `/v1/profile-sections/FAMILY` resolves to 404
  - [x] No pipeline file touched
- [x] Task 2: Tests (AC: #1, #2)
  - [x] Route-level test (`additional_route_test.go`): `TestRegisterProfileSectionRoute_AdditionalRoute_DispatchesAndReturnsCommitted`
  - [x] Route-level test: `TestRegisterProfileSectionRoute_AdditionalRoute_AuthFailure_NeverCallsDispatch`
  - [x] Extra guard test (not originally listed, added because the naming trap is this story's real risk): `TestRegisterProfileSectionRoute_FamilyPath_IsNotRegistered`
  - [x] `//go:build integration` test added (`additional_route_integration_test.go`, shared `postSection` helper) — not executed this session (no live network)
  - [x] `go vet ./...`, `gofmt -l`, `go vet -tags=integration ./...` all clean; full suite green (68 tests, 0 fail); binary builds; confidentiality grep clean

### Review Findings

**Review layers:** Blind Hunter (adversarial), Edge Case Hunter, Acceptance Auditor (spec: this story). 26 raw findings across the three layers, deduplicated to 13 below (5 surviving as patch, 8 deferred, 0 dismissed) after independently re-reading every cited location. Like Story 1.4, this story was authored against Story 1.3's pre-review baseline (expected, same working-order reason), and repeats 3 of the same gap categories — but it also got one thing right that 1.4 didn't attempt (a dedicated naming-trap guard test, per this story's own Dev Notes instruction) and surfaced one genuinely new, more subtle finding no prior story's review caught.

- [x] [Review][Patch] The auth-failure test asserts only `body["status"]`, never `rec.Code` (want 500) nor the absence of `recordID` — repeats the exact gap Story 1.3's review found and fixed for its own auth-failure test, and is even thinner than the pre-review PERSONAL/EMPLOYMENT baselines it was modeled on (those at least checked `recordID` absence). [integration-bridge/additional_route_test.go:78-105]
- [x] [Review][Patch] `additional_route_integration_test.go` hardcodes `"integration-test-emp-5"`/`"integration-test-user-5"` instead of the `uniqueTestID` helper — repeats the exact gap Story 1.3's review found and fixed, despite Story 1.4's sibling file (written after 1.3's fix, before this story) already getting it right. [integration-bridge/additional_route_integration_test.go]
- [x] [Review][Patch] **New finding, not caught by Stories 1.3/1.4's reviews:** `TestRegisterProfileSectionRoute_FamilyPath_IsNotRegistered` builds its own throwaway `http.NewServeMux()` and hardcodes the correct `"ADDITIONAL"` literal directly in the test body — it never calls `main.go`'s real `buildMux()`, the actual production wiring this story shipped. The test would pass unconditionally regardless of what `buildMux()` actually registers, making it tautological with respect to the one artifact it claims to guard. (The identical construction, and identical limitation, already exists in Story 1.3's analogous `TestRegisterProfileSectionRoute_TransferPath_IsNotRegistered` — noted for that story's own future cleanup in `deferred-work.md`, not reopened here since Story 1.3 is already `done`.) [integration-bridge/additional_route_test.go:59-76]
- [x] [Review][Patch] The happy-path test captures and asserts only `gotEmployeeInternalID` — `userID`/`newValue` pass-through is never verified, unlike both the EMPLOYMENT and EDUCATION sibling tests (both post-date this gap's original discovery in spirit, if not in this story's actual authoring order). [integration-bridge/additional_route_test.go:19-54]
- [x] [Review][Patch] No test or comment ties any assertion to AC#2's specific claim ("the bridge only ever sees the HTTP request, never how talenta-core persisted anything on its own side") — Story 1.4's analogous AC#2 test made this connection explicit in its own doc comment; this story's Task 2 claims coverage of "(AC: #1, #2)" without demonstrating the #2 half as clearly. [integration-bridge/additional_route_test.go]

- [x] [Review][Defer] Edge Case Hunter proposed several more route-level tests (non-nil dispatch error, wrong HTTP method, path case-sensitivity, malformed/truncated JSON, missing-field variations, mismatched auth header combinations, dispatch-timeout-exceeded behavior) — all already exhaustively covered generically by the shared pipeline's own tests. [integration-bridge/additional_route_test.go] — deferred per this epic's established principle (one representative rejection test per failure class is enough)
- [x] [Review][Defer] No document-carrying test dimension exists for the ADDITIONAL route (family/dependent documents via the IPFS-backed path). [integration-bridge/additional_route_test.go, additional_route_integration_test.go] — deferred, pipeline-level guarantee already proven once (same reasoning as prior stories' identical defer items)
- [x] [Review][Defer] The integration test's `recordID` type-assertion (`body["recordID"].(string)`) doesn't distinguish "wrong type" from "missing/empty" in its failure message. [integration-bridge/additional_route_integration_test.go] — deferred, cosmetic diagnostic-quality nit, not executed this session anyway

## Dev Notes

**The naming mismatch is the one real risk in this story.** `Hooks.ApproveFamilyDataChange`'s
signature:

```go
// write-path-integration/writepaths/writepaths.go
func (h *Hooks) ApproveFamilyDataChange(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error)
```

is identical in shape to the other four write-path methods and is directly assignable as a
`dispatchFunc`. The only thing to get right is that the HTTP route and the `routeConfig.ProfileSection`
value are `"ADDITIONAL"` — the epic's own story title says "ADDITIONAL", its story prose says "family/
dependents", and the `Hooks` method says "Family" — three different words for one ratified enum value.
`fabric-network/chaincode/employeeprofilerecord/chaincode/asset.go` is the single source of truth:

```go
SectionAdditional ProfileSection = "ADDITIONAL"
```

### Previous Story Intelligence (from Story 1.4, `review`)

- Same pure-wiring pattern as Stories 1.3/1.4: one `registerProfileSectionRoute` call, one route
  test file, one disclosed-not-run integration test file using the shared `postSection` helper and
  `startPersonalRouteTestBridge` (name is a known, harmless leftover — not worth renaming mid-epic).
- Confidentiality register still applies — `grep -rniE "talenta|mekari" *.go` must stay clean in
  `integration-bridge/`.

### Project Structure Notes

- Modifies: `main.go` only.
- New: `additional_route_test.go`, `additional_route_integration_test.go`.

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 1.5] — the approved ACs, verbatim
- [Source: write-path-integration/writepaths/writepaths.go] — `Hooks.ApproveFamilyDataChange`'s exact signature (read directly, this session)
- [Source: fabric-network/chaincode/employeeprofilerecord/chaincode/asset.go] — `SectionAdditional` = `"ADDITIONAL"` (read directly, this session) — the naming-trap source of truth
- [Source: _bmad-output/implementation-artifacts/1-4-talenta-core-can-anchor-an-education-section-change.md] — previous story, `review`

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (claude-opus-5[1m])

### Debug Log References

- `go vet ./... && go test ./...` — 68 pass, 0 fail (dev-story); 87 pass, 0 fail (post-review)
- `go vet -tags=integration ./...` — clean; integration test not executed (no live network)
- `go build -o /tmp/ib-smoketest . && rm -f /tmp/ib-smoketest` — clean
- `grep -rniE "talenta|mekari" *.go` — clean

### Completion Notes List

- Same pure-wiring pattern as 1.3/1.4, with one deliberate deviation: added an extra test
  (`TestRegisterProfileSectionRoute_FamilyPath_IsNotRegistered`) beyond the story's own task list,
  because this story's real risk isn't the wiring mechanism (already proven three times over) — it's
  a plausible naming mistake (`"FAMILY"` vs the ratified `"ADDITIONAL"` enum value) that a future
  reader could make by pattern-matching the `Hooks` method name (`ApproveFamilyDataChange`) or the
  epic's own prose ("family/dependents changes") instead of the chaincode's actual enum.
- All ACs covered; full existing suite green, no regressions.
- **Code review remediation:** repeated 3 of the same gap categories Stories 1.3/1.4 already fixed (missing `rec.Code`/`recordID`-absence assertions, hardcoded integration-test ID instead of `uniqueTestID`, incomplete `Dispatch` pass-through assertions) — expected, since this story was authored in the same pre-review batch as 1.4. But the review also surfaced a genuinely new finding neither 1.3's nor 1.4's review caught: the naming-trap guard test itself was tautological — it built its own throwaway `http.NewServeMux()` with a hardcoded-correct `"ADDITIONAL"` literal rather than exercising `main.go`'s real `buildMux()`, so it would have passed regardless of what production code actually registered. Rewrote it as `TestBuildMux_FamilyPath_IsNotRegistered` using the real `buildMux(stubHooksForMuxTest(), ...)`. Logged the identical defect in Story 1.3's own analogous test to `deferred-work.md` rather than silently reopening an already-`done` story. Self-caught and fixed a confidentiality leak in my own new comment (copied the real product name verbatim while quoting AC#2's text) before finishing. Suite: 68→87 tests (net, since Stories 1.2-1.4's own reviews landed between this story's dev-story and its own review).

### File List

**New:**
- `integration-bridge/additional_route_test.go`
- `integration-bridge/additional_route_integration_test.go`

**Modified:**
- `integration-bridge/main.go` — added ADDITIONAL route registration

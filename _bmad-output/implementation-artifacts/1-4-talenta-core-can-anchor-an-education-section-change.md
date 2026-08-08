---
baseline_commit: a6f0cd63fcfe602255a143b938a55c1bbba583a6
---

# Story 1.4: talenta-core Can Anchor an EDUCATION Section Change

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As talenta-core's EDUCATION write path (no approval gate, no `EmployeeDataRequest` envelope),
I want the same contract,
so that a direct write still anchors correctly.

## Acceptance Criteria

1. **Given** a valid `POST /v1/profile-sections/EDUCATION` request with correct `X-Api-Key`/`X-Company-ID`, **when** submitted with a correct envelope, **then** the bridge dispatches to `Hooks.RecordEducationHistory` and returns the same `status` contract as Story 1.2. [Source: epics.md Story 1.4]
2. **Given** the bridge has no concept of PHP-side approval state, **when** an EDUCATION request (which, unlike PERSONAL/EMPLOYMENT, has no approval gate on the talenta-core side) is processed, **then** no bridge behavior differs for the missing approval gate — the bridge only ever sees the HTTP request, never talenta-core's internal workflow state. [Source: epics.md Story 1.4]

## Tasks / Subtasks

- [x] Task 1: Register the EDUCATION route (AC: #1, #2)
  - [x] Added `registerProfileSectionRoute(mux, "POST /v1/profile-sections/EDUCATION", routeConfig{ProfileSection: "EDUCATION", Dispatch: hooks.RecordEducationHistory}, authCfg, hooks.TenantID, dispatchTimeout)` in `main.go`, alongside PERSONAL/EMPLOYMENT
  - [x] No pipeline file touched — confirmed AC#2 holds by construction
- [x] Task 2: Tests (AC: #1, #2)
  - [x] Route-level test (`education_route_test.go`): `TestRegisterProfileSectionRoute_EducationRoute_DispatchesAndReturnsCommitted`
  - [x] Route-level test: `TestRegisterProfileSectionRoute_EducationRoute_ValidateFailure_NeverCallsDispatch` (chose validate-rejection this time rather than repeating Story 1.3's auth-rejection test verbatim, for a bit more of the matrix's edge coverage across the epic)
  - [x] `//go:build integration` test added (`education_route_integration_test.go`, using the shared `postSection` helper) — not executed this session (no live network)
  - [x] `go vet ./...`, `gofmt -l`, `go vet -tags=integration ./...` all clean; full suite green (65 tests, 0 fail); binary builds; confidentiality grep clean

### Review Findings

**Review layers:** Blind Hunter (adversarial), Edge Case Hunter, Acceptance Auditor (spec: this story). 27 raw findings across the three layers, deduplicated to 12 below (4 surviving as patch, 6 deferred, 2 dismissed as false positives) after independently re-reading every cited location. All three layers independently confirmed this story was authored against Story 1.3's pre-review baseline (its own Debug Log's "65 tests" = 1.3's pre-review 63 + this story's 2 new tests, not 1.3's post-review 87) — expected given the actual working order this session (all four route stories were dev-storied before any of them were code-reviewed), but it means Story 1.3's own review fixes hadn't landed yet when this story was written, and three of them are repeated here.

- [x] [Review][Patch] AC#1's equivalence-to-PERSONAL claim is unverified for the rejection path — the validate-failure test asserts only `body["status"]`, never `rec.Code`, even though `respond.go` maps `"rejected"` to HTTP 400. Exact repeat of the gap Story 1.3's review found and fixed for its own auth-failure test. [integration-bridge/education_route_test.go:53-82]
- [x] [Review][Patch] `education_route_integration_test.go` hardcodes `"integration-test-emp-4"`/`"integration-test-user-4"` instead of the `uniqueTestID` helper — exact repeat of the gap Story 1.3's review found and fixed for its own EMPLOYMENT integration test. [integration-bridge/education_route_integration_test.go]
- [x] [Review][Patch] The happy-path test's `Dispatch` closure captures only `employeeInternalID` — `userID`/`newValue` pass-through is never asserted, weaker than even EMPLOYMENT's pre-review version since the closure doesn't capture the other args at all. Exact repeat of the gap Story 1.3's review found and fixed. [integration-bridge/education_route_test.go:16-51]
- [x] [Review][Patch] The second test (`ValidateFailure_NeverCallsDispatch`) re-proves generic pipeline behavior already covered by Story 1.2's own PERSONAL-route test, contributing no EDUCATION-specific coverage dimension — same gap class Story 1.3's review flagged (its own second test had the identical problem). [integration-bridge/education_route_test.go:53-82]

- [x] [Review][Defer] No document-carrying test dimension exists for the EDUCATION route. [integration-bridge/education_route_test.go, education_route_integration_test.go] — deferred, the underlying guarantee is pipeline-level and already proven once (same reasoning as Story 1.3's identical defer item)
- [x] [Review][Defer] Hardcoded fixed integration-test port (`18085`) has no dynamic-allocation or cross-file collision check. [integration-bridge/education_route_integration_test.go] — deferred, systemic pre-existing pattern already logged in Story 1.2's deferred-work entry, not introduced here
- [x] [Review][Defer] `startPersonalRouteTestBridge` keeps its PERSONAL-specific name despite being reused generically by a fourth route's test file now. [integration-bridge/education_route_integration_test.go] — deferred, same cosmetic nit already logged in Story 1.3's deferred-work entry
- [x] [Review][Defer] Route path case-sensitivity (`/v1/profile-sections/education` vs `EDUCATION`) is untested. [integration-bridge/education_route_test.go] — deferred, stdlib `ServeMux` behavior, not app logic; low value to re-test per route
- [x] [Review][Defer] Edge Case Hunter proposed several more route-level tests (non-nil dispatch error, mismatched auth headers, missing userID/malformed-JSON variations, wrong HTTP method) — all already exhaustively covered generically by the shared pipeline's own tests. [integration-bridge/education_route_test.go] — deferred per this epic's established principle (one representative rejection test per failure class is enough)
- [x] [Review][Defer] No integration-level test forces a non-committed outcome (rejection/timeout) for EDUCATION end-to-end. [integration-bridge/education_route_integration_test.go] — deferred, same reasoning as the document-carrying gap above; pipeline-level, proven once

**Dismissed (2), both false positives verified against the actual source:** (1) Edge Case Hunter's claim that `hooks.RecordEducationHistory` could be `nil` if "never wired" — `RecordEducationHistory` is a method on `*Hooks` (`func (h *Hooks) RecordEducationHistory(...)`), not a struct field; Go methods aren't independently "unwired," so this scenario cannot occur. (2) Edge Case Hunter's claim that no `t.Cleanup`/`defer` shuts down the test bridge server — `startPersonalRouteTestBridge` (the shared helper this test calls, unmodified by this story) already registers `t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })`, confirmed by direct grep.

## Dev Notes

**Scope boundary — pure wiring, same as Story 1.3.** `RecordEducationHistory`'s signature is
identical to the other four `Hooks` write-path methods:

```go
// write-path-integration/writepaths/writepaths.go
func (h *Hooks) RecordEducationHistory(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error)
```

Directly assignable as a `dispatchFunc` method value. `ProfileSection` string is `"EDUCATION"`
(confirmed against `asset.go`'s `SectionEducation` constant).

**On AC#2's "no approval gate" wording:** this is a talenta-core-side (PHP) workflow distinction —
EDUCATION writes commit directly rather than through an approval queue. Nothing in
`integration-bridge` has ever modeled approval state (it only ever sees one HTTP request per write,
with no concept of "pending" vs "approved"), so this AC is satisfied automatically by the pipeline's
existing design, not by a new check. Do not add an approval-related field or branch — there is
nothing to add.

### Previous Story Intelligence (from Story 1.3, `review`)

- Story 1.3 established the exact pattern this story repeats: one `registerProfileSectionRoute` call
  in `main.go`, one route-level unit test file mirroring the PERSONAL tests, one disclosed-not-run
  `//go:build integration` test using the shared `postSection(t, httpAddr, section, payload)` helper
  (`personal_route_integration_test.go`) — reuse that helper, do not write a new one.
  `startPersonalRouteTestBridge` is also shared/reusable as-is (its name predates multi-route reuse
  but its behavior is already generic).
  - **Correction if a future story reads this comment before renaming:** `startPersonalRouteTestBridge`
    is a slightly misleading name now that it starts the bridge for every route's tests, not just
    PERSONAL's — a harmless naming leftover from Story 1.2, not worth a disruptive rename mid-epic. A
    future cleanup pass could rename it (e.g. `startBridgeSubprocess`), not this story.
- No new config, no new confidentiality-register risk beyond what Story 1.2/1.3 already established
  (still applies: `grep -rniE "talenta|mekari" *.go` must stay clean).
- **A checkbox claiming something the diff doesn't actually do is a real finding** — only check a
  task off once genuinely done (repeated precedent from Stories 1.1–1.3).

### Project Structure Notes

- Modifies: `main.go` only (one new registration line).
- New: `education_route_test.go`, `education_route_integration_test.go`.

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 1.4] — the approved ACs, verbatim
- [Source: write-path-integration/writepaths/writepaths.go] — `Hooks.RecordEducationHistory`'s exact signature (read directly, this session)
- [Source: fabric-network/chaincode/employeeprofilerecord/chaincode/asset.go] — `SectionEducation` = `"EDUCATION"` (read directly, this session)
- [Source: _bmad-output/implementation-artifacts/1-3-talenta-core-can-anchor-an-employment-section-change.md] — previous story, `review`, the pattern this story repeats

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (claude-opus-5[1m])

### Debug Log References

- `go vet ./... && go test ./...` — 65 pass, 0 fail (dev-story); 87 pass, 0 fail (post-review)
- `go vet -tags=integration ./...` — clean; integration test not executed (no live network)
- `go build -o /tmp/ib-smoketest . && rm -f /tmp/ib-smoketest` — clean
- `grep -rniE "talenta|mekari" *.go` — clean

### Completion Notes List

- Repeats Story 1.3's exact pattern: one `registerProfileSectionRoute` call, no pipeline changes. AC#2 ("no bridge behavior differs for the missing approval gate") required no code — the bridge has never modeled approval state, so there was nothing to add or guard.
- All ACs covered; full existing suite green, no regressions.
- **Code review remediation:** all 3 layers independently noticed this story was authored against Story 1.3's pre-review baseline (65 tests = 1.3's pre-review 63 + this story's 2 new — not 1.3's post-review 87), which is expected given the actual working order this session (all four route stories were dev-storied before any were code-reviewed) but meant 3 of Story 1.3's own review fixes hadn't landed yet and were repeated here: missing `rec.Code` assertion for an AC equivalence claim, a hardcoded integration-test ID instead of `uniqueTestID`, and incomplete `Dispatch` pass-through assertions. Fixed all 3, plus rewrote the second unit test to cover a genuinely EDUCATION-specific dimension (AC#2's "no approval gate" claim) rather than just re-proving generic pipeline behavior — unlike Story 1.5's ADDITIONAL/FAMILY naming trap, EDUCATION has no analogous naming-confusion risk to build a guard test around, so the fix targeted the AC's actual content instead. 2 findings dismissed as false positives after verification (a claimed nil-method risk that can't occur in Go, and a claimed missing test cleanup that already exists in the shared helper). Also caught and fixed a confidentiality leak in my own new test comment before finishing. 6 low-severity findings deferred. Suite: 65→87 tests (net, since Story 1.2's and 1.3's own reviews landed between this story's dev-story and its own review).

### File List

**New:**
- `integration-bridge/education_route_test.go`
- `integration-bridge/education_route_integration_test.go`

**Modified:**
- `integration-bridge/main.go` — added EDUCATION route registration

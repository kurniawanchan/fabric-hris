---
baseline_commit: a6f0cd63fcfe602255a143b938a55c1bbba583a6
---

# Story 1.6: talenta-core Can Anchor a PAYROLL Section Change — Completing All Five Routes

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As talenta-core's PAYROLL write path (sharing its PHP commit site with PERSONAL),
I want a dedicated route with a digest that exactly matches what was stored,
so that bank-account numbers never silently lose precision.

## Acceptance Criteria

1. **Given** a `POST /v1/profile-sections/PAYROLL` request where `newValue` has a bank-account number greater than 2^53, **when** `[validate]` captures it via `json.RawMessage`, **then** the digit string reaching `Hooks.UpdatePayrollBankAccount` is byte-identical to what talenta-core sent — verified by a test asserting no `float64` conversion occurred anywhere in the path. [Source: epics.md Story 1.6]
2. **Given** all five routes now exist, **when** any other path or a 6th section name is requested, **then** the server returns 404 — FR-8 satisfied by construction, not an app-level check. [Source: epics.md Story 1.6]
3. **Given** PERSONAL and PAYROLL are dispatched from two separate bridge routes, **when** both are exercised, **then** they remain two independent operations at the bridge's API, even though talenta-core's own PHP happens to call both from one commit function (AD-5). [Source: epics.md Story 1.6]

## Tasks / Subtasks

- [x] Task 1: Register the PAYROLL route (AC: #1, #2, #3)
  - [x] Added `registerProfileSectionRoute(mux, "POST /v1/profile-sections/PAYROLL", routeConfig{ProfileSection: "PAYROLL", Dispatch: hooks.UpdatePayrollBankAccount}, authCfg, hooks.TenantID, dispatchTimeout)` — the fifth and final route
  - [x] No `[auth]`/`[validate]`/`[dispatch]`/`[map-error]`/`[respond]` logic touched
- [x] Task 2: Byte-precision test for PAYROLL (AC: #1)
  - [x] `payroll_route_test.go`'s `TestRegisterProfileSectionRoute_PayrollRoute_LargeBankAccountNumber_PreservedByteForByte` — the 19-digit fixture reaches the stub `Dispatch`'s `newValue` byte-for-byte, end-to-end through the real HTTP body
- [x] Task 3: 404-by-construction test (AC: #2)
  - [x] Extracted `buildMux(hooks *writepaths.Hooks, authCfg authConfig, dispatchTimeout time.Duration) *http.ServeMux` out of `run()` in `main.go` (pure extraction, all 5 registrations plus the new PAYROLL one); `mux_test.go`'s `TestBuildMux_SixthSectionName_Is404` and `TestBuildMux_AllFiveRoutesRegistered_UnrelatedPathIs404` prove it
  - [x] `TestBuildMux_AllFiveRatifiedRoutesExist` added as a companion check that none of the 5 real routes accidentally 404
- [x] Task 4: Independence test for PERSONAL vs. PAYROLL (AC: #3)
  - [x] `mux_test.go`'s `TestBuildMux_PersonalAndPayrollAreIndependentRoutes` (structural: both paths routable) plus `TestRegisterProfileSectionRoute_TwoRoutesOnOneMux_NeverCrossDispatch` (behavioral: two distinct stub dispatch functions, proven neither is reachable from the other's path)
  - [x] Self-caught and fixed a confidentiality-register leak in this test file's own doc comment ("talenta-core" / real product name) before finishing — see Completion Notes
- [x] Task 5: `//go:build integration` test (AC: #1, #2, #3)
  - [x] `payroll_route_integration_test.go`: `TestIntegration_PayrollRoute_CommitsWithLargeBankAccountNumber` and `TestIntegration_UnregisteredSectionName_Is404` (the latter built without the shared `postSection` helper, since a real 404 response body is plain text, not the JSON envelope `postSection` expects to decode) — not executed this session (no live network)
- [x] Task 6: Full regression
  - [x] `go vet ./...`, `gofmt -l`, `go vet -tags=integration ./...` all clean; full suite green (75 tests, 0 fail, up from 68 before this story); real binary builds; confidentiality grep clean; `run()`'s behavior confirmed unchanged (it now just calls `buildMux`, same 5 registrations as before, same `authCfg`/`dispatchTimeout` variables)

### Review Findings

**Review layers:** Blind Hunter (adversarial), Edge Case Hunter, Acceptance Auditor (spec: this story). 26 raw findings across the three layers, deduplicated to 14 below (4 surviving as patch, 9 deferred — 2 backed by new grounding gaps G-35/G-36 — 1 dismissed as a verified false positive) after independently re-reading every cited location. This is the final story in the epic; two of the deferred findings are genuine, previously-undocumented structural verification limits (not code defects) that this story's own AC#3 and AC#1 live-network claims run into.

- [x] [Review][Patch] The PAYROLL happy-path test has **zero** pass-through or `recordID` assertions — the `Dispatch` closure doesn't even capture `employeeInternalID`/`userID`/`newValue`, and the only body assertion is `status`. Every sibling route's equivalent test (PERSONAL, EMPLOYMENT, EDUCATION, ADDITIONAL) captures and asserts all three plus `recordID`; this is a step backward from a pattern the epic enforced four times over. [integration-bridge/payroll_route_test.go:15-42]
- [x] [Review][Patch] `payroll_route_integration_test.go` hardcodes `"integration-test-emp-6"`/`"integration-test-user-6"` and `"integration-test-emp-7"`/`"integration-test-user-7"` instead of the `uniqueTestID` helper — the 4th recurrence of the exact gap Story 1.3's review found and fixed, this time with the fix long predating this story's own authorship (the file already imports and uses `startPersonalRouteTestBridge`/`postSection` from the same source file `uniqueTestID` lives in). [integration-bridge/payroll_route_integration_test.go]
- [x] [Review][Patch] The byte-precision unit test uses `strings.Contains(string(gotNewValue), largeAccountNumber)` rather than an exact match — every sibling happy-path test's pass-through assertion uses exact equality (`string(gotNewValue) != <exact literal>`) for the identical kind of check, and `Contains` cannot detect surrounding-structure corruption (extra/reordered fields, injected whitespace), only digit-string deletion. [integration-bridge/payroll_route_test.go:74-76]
- [x] [Review][Patch] No PAYROLL-specific rejection-path test exists (`payroll_route_test.go` has exactly 2 tests: happy-path and byte-precision) — every other route-adding story paired its happy path with at least one route-scoped rejection test, and this story's own second test is a genuinely new coverage dimension rather than a redundant proof, so there was room for a third without violating the "one representative is enough" principle. [integration-bridge/payroll_route_test.go]

- [x] [Review][Defer] **No test proves `buildMux` wires PAYROLL (or any route) to its correct, distinct `Hooks` method** — every existing test either checks non-404 (can't distinguish "correctly wired" from "two routes aliased by mistake") or exercises `registerProfileSectionRoute`'s own stub-closure mechanism, never the real `buildMux` assignments. This is AC#3's actual core claim and it has no test coverage. Recorded as `grounding-gaps.md` **G-35** rather than attempting a contrived fix — properly closing it needs a full stub `Hooks` (all 4 store interfaces) or a live-network read-back, both nontrivial new infrastructure beyond this review's remit. [integration-bridge/main.go, integration-bridge/mux_test.go]
- [x] [Review][Defer] **The live-network integration test for AC#1's byte-precision claim only proves "a commit succeeded," not "the digits survived intact"** — the same structural pseudonym-opacity problem `grounding-gaps.md` G-34 already documents prevents any read-back verification. The unit-level test already proves the mechanism (byte-for-byte pass-through to the dispatch stand-in); reworded the integration test's own doc comment to stop overclaiming "AC#1, live-network half" as fully proven. Recorded as **G-36**. [integration-bridge/payroll_route_integration_test.go]
- [x] [Review][Defer] `mux_test.go`'s `TestBuildMux_PersonalAndPayrollAreIndependentRoutes` is a strict subset of `TestBuildMux_AllFiveRatifiedRoutesExist` (both only check non-404), and `TestBuildMux_AllFiveRoutesRegistered_UnrelatedPathIs404` is near-tautological next to `TestBuildMux_SixthSectionName_Is404`. [integration-bridge/mux_test.go] — deferred, harmless redundancy, not worth the churn of removing passing tests
- [x] [Review][Defer] No naming-trap guard test exists for PAYROLL, unlike ADDITIONAL/EMPLOYMENT — arguably correctly scoped, since `UpdatePayrollBankAccount` has no name-divergence from `"PAYROLL"` the way `ApproveFamilyDataChange`/`ApproveEmploymentTransfer` diverge from their enum values, but this was never stated as a deliberate decision anywhere in the diff. [integration-bridge/payroll_route_test.go] — deferred; noted as a deliberate scope call in Completion Notes below rather than silently left ambiguous
- [x] [Review][Defer] `TestIntegration_UnregisteredSectionName_Is404`'s live-network value is thin relative to `TestBuildMux_SixthSectionName_Is404`, which proves the same property with no live network required. [integration-bridge/payroll_route_integration_test.go] — deferred, kept for live-network end-to-end assurance despite the overlap
- [x] [Review][Defer] Manually-incremented sequential ports (18081-18088) and ID suffixes across all 5 route integration test files have no shared registry/constant. [integration-bridge/payroll_route_integration_test.go and siblings] — deferred, systemic pattern already logged across prior stories' deferred-work entries
- [x] [Review][Defer] Wrong-HTTP-method (GET/PUT/DELETE) requests to the 5 ratified routes are untested. [integration-bridge/mux_test.go] — deferred, stdlib `ServeMux` guarantee, established epic-wide principle
- [x] [Review][Defer] `bankAccountNumber` sent as a quoted JSON string (not a bare number) is untested. [integration-bridge/payroll_route_test.go] — deferred, outside AC#1's literal scope (a JSON number)
- [x] [Review][Defer] `buildMux` would nil-dereference `hooks.TenantID` if ever called with a nil `*writepaths.Hooks`. [integration-bridge/main.go] — deferred, not reachable via the single real call site in `run()`

**Dismissed (1), verified false positive:** Edge Case Hunter's claim that an empty `{}` request body in `mux_test.go`'s tests risks a nil-pointer panic inside `Dispatch` on `stubHooksForMuxTest()`'s nil `Store`/`Keys` fields — independently confirmed false by re-reading `validate.go`: every `{}` body is rejected at the `employeeInternalID`-required check inside `[validate]`, before the request can ever reach `[dispatch]`/a real `Hooks` method.

## Dev Notes

**This is the one story in 1.3–1.6 that needs a real (small) structural change, not pure wiring.**
Every AC in Stories 1.3–1.5 could be tested against a single route built directly via
`registerProfileSectionRoute` in a test-local `http.NewServeMux()`. AC#2 and AC#3 here are explicitly
about the *complete, five-route system* — "all five routes now exist," "PERSONAL and PAYROLL... two
separate bridge routes" — which by definition cannot be tested by building one route at a time.
Extracting `main.go`'s route-registration block into a `buildMux` function is the minimal change that
makes the complete route set testable without a live network, mirroring the same dependency-inversion
technique already established for `buildHooks` (Story 1.1) and `dispatch`'s `dispatchFunc` parameter
(Story 1.2) — extract exactly the seam needed, nothing more.

### `Hooks.UpdatePayrollBankAccount`'s exact signature — confirmed

```go
// write-path-integration/writepaths/writepaths.go
func (h *Hooks) UpdatePayrollBankAccount(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error)
```

Same shape as the other four. `ProfileSection` = `"PAYROLL"` (confirmed against `asset.go`'s
`SectionPayroll` constant) — no naming trap here, unlike Story 1.5's ADDITIONAL/FAMILY mismatch.

### Why AC#1 needs no new validation logic

Story 1.2's Dev Notes already traced the exact call chain protecting large integers:
`validateRequest` captures `newValue` as `json.RawMessage` (never re-parsed into
`map[string]interface{}`), and `ComputeDataHash` canonicalizes those exact raw bytes via
`jsoncanonicalizer.Transform`. This is route-agnostic — nothing about PAYROLL specifically needs new
code. AC#1 is a regression test proving PAYROLL didn't accidentally reintroduce a parse-and-remarshal
step, not a new mechanism.

### Current `main.go` structure — what `buildMux` extracts

```go
// integration-bridge/main.go, current state inside run(), after authCfg/dispatchTimeout are set:
mux := http.NewServeMux()
registerProfileSectionRoute(mux, "POST /v1/profile-sections/PERSONAL", ...)
registerProfileSectionRoute(mux, "POST /v1/profile-sections/EMPLOYMENT", ...)
registerProfileSectionRoute(mux, "POST /v1/profile-sections/EDUCATION", ...)
registerProfileSectionRoute(mux, "POST /v1/profile-sections/ADDITIONAL", ...)
```

Extract this block (plus the new PAYROLL line) into `buildMux(hooks *writepaths.Hooks, authCfg
authConfig, dispatchTimeout time.Duration) *http.ServeMux`, and have `run()` call
`mux := buildMux(hooks, authCfg, dispatchTimeout)` instead of constructing it inline. `run()`'s
observable behavior must not change — this is a pure extraction, not a redesign.

### Previous Story Intelligence (from Story 1.5, `review`)

- Stories 1.3–1.5 each added one `registerProfileSectionRoute` call directly in `run()`; this story's
  `buildMux` extraction is the first change to that pattern, and the last one needed — no more routes
  remain after this story (all 5 are ratified, per `IsValidProfileSection`).
- The shared `postSection(t, httpAddr, section, payload)` and `startPersonalRouteTestBridge` helpers
  (in `personal_route_integration_test.go`) are reused again here for the integration test.
- Confidentiality register still applies — `grep -rniE "talenta|mekari" *.go` must stay clean.
- A checkbox claiming something the diff doesn't actually do is a real finding — verify `buildMux`
  is actually called from `run()` (not just defined and left unused) before marking Task 1 done.

### Project Structure Notes

- Modifies: `main.go` (extract `buildMux`, add PAYROLL registration, `run()` calls `buildMux`).
- New: `payroll_route_test.go`, `payroll_route_integration_test.go`.
- This is the last story in Epic 1 — after this story, all 5 AD-5 routes exist and Epic 1's stated
  goal ("talenta-core Can Reliably Anchor Any Profile-Section Change") is fully met.

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 1.6] — the approved ACs, verbatim
- [Source: write-path-integration/writepaths/writepaths.go] — `Hooks.UpdatePayrollBankAccount`'s exact signature (read directly, this session)
- [Source: fabric-network/chaincode/employeeprofilerecord/chaincode/asset.go] — `SectionPayroll` = `"PAYROLL"`, `IsValidProfileSection`'s exactly-five-values contract (read directly, this session)
- [Source: integration-bridge/main.go] — current route-registration structure (read directly, this session)
- [Source: _bmad-output/implementation-artifacts/1-2-talenta-core-can-anchor-a-personal-section-change.md] — `json.RawMessage`/`ComputeDataHash` call chain (established there, unchanged)
- [Source: _bmad-output/implementation-artifacts/1-5-talenta-core-can-anchor-an-additional-section-change.md] — previous story, `review`

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (claude-opus-5[1m])

### Debug Log References

- `go vet ./... && go test ./...` — 75 pass, 0 fail (dev-story); 88 pass, 0 fail (post-review)
- `go vet -tags=integration ./...` — clean; integration tests not executed (no live network, confirmed via `docker ps`)
- `go build -o /tmp/ib-smoketest . && rm -f /tmp/ib-smoketest` — clean
- `grep -rniE "talenta|mekari" *.go` — clean (after one self-caught fix, see below)

### Completion Notes List

- This story required one real structural change (not pure wiring, unlike 1.3-1.5): extracting
  `buildMux` out of `run()`, because AC#2 and AC#3 are claims about the *complete, five-route system*
  that no single-route unit test can demonstrate. `run()`'s own behavior is unchanged — confirmed by
  reading the resulting diff, not just by the test suite passing.
- AC#1's byte-precision guarantee needed no new production code (the mechanism is Story 1.2's
  `json.RawMessage` handling in `validate.go`, already route-agnostic) — this story only adds an
  end-to-end regression test proving PAYROLL specifically doesn't reintroduce a parse-and-remarshal
  step.
- **Self-caught confidentiality issue:** `mux_test.go`'s own doc comments used "talenta-core" by name
  twice (once describing dispatch independence, once in the cross-dispatch test's comment) — caught
  by this task's own closing `grep -rniE "talenta|mekari" *.go` check, same discipline established in
  Stories 1.2 and 1.3. Fixed by rewording to generic terms ("the caller", "callers on the other side
  of this API") consistent with this repo's established convention. First attempt at the fix edited
  the wrong of two near-identical comment blocks — caught immediately by re-running the grep rather
  than assuming the fix worked, and corrected on the second pass.
- **Epic 1 is now feature-complete at the bridge level:** all 5 AD-5 routes (PERSONAL, EMPLOYMENT,
  EDUCATION, ADDITIONAL, PAYROLL) exist, share one pipeline, and are provably independent and
  exhaustive (404-by-construction for anything else). All ACs across Stories 1.2-1.6 covered; full
  suite green, no regressions.
- **Code review remediation (final story in the epic):** repeated the by-now-familiar hardcoded-ID
  gap a 4th time and the pass-through-assertion gap a 5th time (both fixed again here), plus a
  weaker-than-sibling assertion technique (`strings.Contains` instead of exact equality) and a
  missing rejection-path test. More significantly, the review surfaced two genuine, previously
  undocumented structural verification limits rather than code defects: no test proves `buildMux`
  wires each route to its *correct* `Hooks` method (AC#3's actual core claim has zero coverage,
  since every existing test either checks non-404 or exercises the shared mechanism with its own
  stub closures, never the real production wiring) — recorded as `grounding-gaps.md` **G-35**. And
  the live-network PAYROLL byte-precision test only proves a commit succeeded, not that the digits
  survived intact, for the same pseudonym-opacity reason G-34 already documents — recorded as
  **G-36**. Also self-caught (twice, on the second attempt after the first fix missed the actual
  occurrence — same pattern as the original dev-story session) and fixed a stray compiled binary
  (`integration-bridge/integrationbridge`) that `go build ./...` had been silently writing into the
  repo all session; added it to `.gitignore`. 9 low-severity/structural findings deferred; 1 false
  positive dismissed after verification.

### File List

**New:**
- `integration-bridge/mux_test.go`
- `integration-bridge/payroll_route_test.go`
- `integration-bridge/payroll_route_integration_test.go`

**Modified:**
- `integration-bridge/main.go` — extracted `buildMux`, added PAYROLL route registration, `run()` now calls `buildMux`
- `.gitignore` — code review: added `integration-bridge/integrationbridge` (a stray compiled binary `go build ./...` had been writing into the repo)
- `agent-suite/11-execution/grounding-gaps.md` — code review: added **G-35** (no test verifies `buildMux`'s per-route `Hooks` method wiring), **G-36** (PAYROLL's live-network byte-precision test doesn't prove no corruption occurred)

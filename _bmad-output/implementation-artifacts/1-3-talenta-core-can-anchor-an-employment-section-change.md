---
baseline_commit: a6f0cd63fcfe602255a143b938a55c1bbba583a6
---

# Story 1.3: talenta-core Can Anchor an EMPLOYMENT Section Change

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As talenta-core's EMPLOYMENT write path (entirely separate from PERSONAL's own commit site),
I want the same calling contract,
so that transfers get identical anchoring guarantees with zero new integration work.

## Acceptance Criteria

1. **Given** a valid `POST /v1/profile-sections/EMPLOYMENT` request with correct `X-Api-Key`/`X-Company-ID`, **when** submitted with a correct envelope, **then** the bridge dispatches to `Hooks.ApproveEmploymentTransfer` and returns the same `status` contract as Story 1.2 (`committed` with `recordID` on success). [Source: epics.md Story 1.3]
2. **Given** the exact same auth/validate/timeout/logging rules Story 1.2 already implements, **when** an EMPLOYMENT request is rejected at `[auth]` or `[validate]`, or `[dispatch]` times out, **then** it behaves identically to the PERSONAL route (same `status` values, same HTTP codes) — this route adds a dispatch target to the existing pipeline, it does not add any new pipeline logic. [Source: epics.md Story 1.3]

## Tasks / Subtasks

- [x] Task 1: Register the EMPLOYMENT route (AC: #1, #2)
  - [x] In `main.go`, added `registerProfileSectionRoute(mux, "POST /v1/profile-sections/EMPLOYMENT", routeConfig{ProfileSection: "EMPLOYMENT", Dispatch: hooks.ApproveEmploymentTransfer}, authCfg, hooks.TenantID, dispatchTimeout)`, alongside the existing PERSONAL registration
  - [x] Confirmed no other pipeline file needed a change — `handlers.go`/`pipeline.go`/`dispatch.go`/`classify.go`/`validate.go`/`auth.go`/`respond.go` are untouched
- [x] Task 2: Tests (AC: #1, #2)
  - [x] Route-level test: `employment_route_test.go`'s `TestRegisterProfileSectionRoute_EmploymentRoute_DispatchesAndReturnsCommitted` — a valid EMPLOYMENT request dispatches to a stub matching `Hooks.ApproveEmploymentTransfer`'s signature, returns `committed` + `recordID`
  - [x] Route-level test: `TestRegisterProfileSectionRoute_EmploymentRoute_AuthFailure_NeverCallsDispatch` — one representative rejection test proving the route runs through the same `[auth]`→`[map-error]` path
  - [x] `//go:build integration` test added (`employment_route_integration_test.go`, `TestIntegration_EmploymentRoute_Commits`) — this is the only test that can actually prove `main.go` wired `ApproveEmploymentTransfer` (not e.g. a copy-paste of `UpdatePersonalData`) rather than the generic pipeline mechanism, since `main.go`'s route table has no unit-testable seam by design. **Not executed this session** (no live network, same as Story 1.2's `personal_route_integration_test.go`)
  - [x] Generalized `personal_route_integration_test.go`'s `postPersonalSection` helper into `postSection(t, httpAddr, section, payload)` so this and future route integration tests don't duplicate it — a one-line addition to a shared helper, not new pipeline logic
  - [x] `go vet ./...`, `gofmt -l`, `go vet -tags=integration ./...`, full existing suite still green (63 tests, 0 fail), real binary builds, confidentiality grep clean

### Review Findings

**Review layers:** Blind Hunter (adversarial), Edge Case Hunter, Acceptance Auditor (spec: this story). 23 raw findings across the three layers, deduplicated to 16 below (7 surviving as patch, 8 deferred, 1 dismissed) after independently re-reading every cited location.

- [x] [Review][Patch] AC#2's "same HTTP codes" equivalence-to-PERSONAL claim is never actually asserted by this story's own auth-failure test — it checks `body["status"]` but never reads `rec.Code`, unlike the sibling happy-path test in the same file. [integration-bridge/employment_route_test.go]
- [x] [Review][Patch] `employment_route_integration_test.go` hardcodes `"integration-test-emp-3"`/`"integration-test-user-3"` instead of using the `uniqueTestID` helper `personal_route_integration_test.go` just established specifically to stop repeated runs colliding against the live, append-only ledger. [integration-bridge/employment_route_integration_test.go]
- [x] [Review][Patch] This story's own Dev Notes explicitly flag a naming-confusion risk ("EMPLOYMENT is correct... not e.g. TRANSFER") but never convert it into an executable guard, unlike Story 1.5's `TestRegisterProfileSectionRoute_FamilyPath_IsNotRegistered` precedent for the identical class of risk. The auth-failure test that was added instead re-proves generic pipeline behavior already covered by Story 1.2's own PERSONAL-route test, contributing no new coverage dimension. [integration-bridge/employment_route_test.go]
- [x] [Review][Patch] The copied auth-failure test dropped the "recordID must be absent on an auth-rejected response" assertion present in the Story 1.2 test it was modeled on. [integration-bridge/employment_route_test.go]
- [x] [Review][Patch] `CLAUDE.md`'s confidentiality-register hard-clean list (`fabric-network/`, `write-path-integration/`, `ipfs-cluster/`, `qa-tests/`, `deploy/`, `docs/`) does not actually include `integration-bridge/`, despite every story in this epic treating it as hard-clean and running `grep -rniE "talenta|mekari"` against it as a closing gate. [CLAUDE.md]
- [x] [Review][Patch] The EMPLOYMENT happy-path test only asserts `employeeInternalID` reached the stub `Dispatch` — `userID`/`newValue`/`document` pass-through is unverified. [integration-bridge/employment_route_test.go]
- [x] [Review][Patch] `postSection` unconditionally JSON-decodes the response body regardless of `resp.StatusCode` — a routing/registration mismatch (e.g. a 404 with a plain-text body) surfaces as a misleading "decoding response body" failure instead of the real cause; `payroll_route_integration_test.go`'s own 404 test already had to hand-roll a separate request rather than reuse this helper because of exactly this gap. [integration-bridge/personal_route_integration_test.go]

- [x] [Review][Defer] No document-carrying integration test exists for the EMPLOYMENT route (PERSONAL got one, proving the shared IPFS nil-pointer-panic fix). [integration-bridge/employment_route_integration_test.go] — deferred, the underlying guarantee is pipeline-level and already proven once
- [x] [Review][Defer] `postSection` cannot gracefully handle a non-JSON-decodable response at all (see the Patch item above for the specific symptom) — a fuller fix (conditional decode, or a separate raw-response helper) is a test-infrastructure design question, not a one-line change. [integration-bridge/personal_route_integration_test.go] — deferred, needs its own design pass
- [x] [Review][Defer] `postSection`'s `section` parameter is concatenated into the URL with no escaping/validation. [integration-bridge/personal_route_integration_test.go:92] — deferred, no current caller can trigger it (every call site passes a hardcoded literal)
- [x] [Review][Defer] Auth header literal values (`"secret-key"`, `"tenant01"`) are hand-duplicated across every route test file instead of derived from `testAuthConfig()`. [integration-bridge/employment_route_test.go] — deferred, pre-existing convention from Story 1.2, reproduced rather than introduced
- [x] [Review][Defer] `"tenant01"` appears twice in each route test with no shared symbol tying the registration's `tenantID` argument to the `X-Company-ID` header value. [integration-bridge/employment_route_test.go] — deferred, informational
- [x] [Review][Defer] At least 5 `//go:build integration` tests now exist across this epic and none have ever actually executed against a live network; nothing beyond prose comments tracks that they eventually need to. [integration-bridge/*_integration_test.go] — deferred, epic-level process concern, not fixable by a single story's patch
- [x] [Review][Defer] `startPersonalRouteTestBridge` keeps its PERSONAL-specific name despite being reused generically by every route's integration test. [integration-bridge/personal_route_integration_test.go] — deferred, cosmetic naming nit, no functional effect
- [x] [Review][Defer] Edge Case Hunter proposed several more route-level tests (a non-nil dispatch-error case, partial/mismatched auth header combinations, a wrong-HTTP-method request, empty-field rejection) — all already exhaustively covered generically by the shared pipeline's own tests (`handlers_test.go`, `auth_test.go`, `validate_test.go`) against the PERSONAL route. [integration-bridge/employment_route_test.go] — deferred per this epic's own established principle (this story's Task 2: "one representative rejection test per failure class is enough to prove the route is wired through the same pipeline")

**Dismissed (1):** the Acceptance Auditor's and Blind Hunter's shared observation that no test executed this session proves `main.go` dispatches to `Hooks.ApproveEmploymentTransfer` specifically (as opposed to a copy-paste of `UpdatePersonalData`) — this is not a new finding, it is the exact limitation this story's own Task 9 checkbox and Completion Notes already transparently disclosed, and no further code fix exists without adding a test-only seam to `main.go` that Story 1.2 already explicitly declined to add for the identical reason.

## Dev Notes

**Scope boundary — this is a wiring-only story.** Every piece of pipeline logic (`[auth]`, `[validate]`,
`[dispatch]`, `[map-error]`, `[respond]`, the Stage Contract) was built and exhaustively tested in
Story 1.2 and is route-agnostic by construction — `registerProfileSectionRoute` takes a `routeConfig`
precisely so that adding a route never means touching pipeline code (see `handlers.go`'s own doc
comment: "adding a route... means adding a routeConfig value and a mux registration, never a new
handler body"). If implementing this story tempts you to add a branch, special-case, or new file
outside `main.go` + this story's own test file, stop and reconsider — that would mean AD-5's
"identical apart from which Hooks method they call" premise is being violated.

### `Hooks.ApproveEmploymentTransfer`'s exact signature — confirmed, matches `dispatchFunc`

```go
// write-path-integration/writepaths/writepaths.go
func (h *Hooks) ApproveEmploymentTransfer(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error)
```

Identical shape to `UpdatePersonalData` (same four Hooks methods this epic wires all share this
signature) — assignable directly as a `dispatchFunc` method value, exactly as `main.go` already does
for `hooks.UpdatePersonalData`. No adapter needed.

### `main.go`'s exact current registration block — where the new line goes

```go
// integration-bridge/main.go, inside run(), after buildHooks/authCfg/dispatchTimeout are set:
mux := http.NewServeMux()
registerProfileSectionRoute(mux, "POST /v1/profile-sections/PERSONAL",
	routeConfig{ProfileSection: "PERSONAL", Dispatch: hooks.UpdatePersonalData},
	authCfg, hooks.TenantID, dispatchTimeout)
```

Add the EMPLOYMENT registration as a second call right after this one, before `srv := &http.Server{...}`.

### Previous Story Intelligence (from Story 1.2, `review`)

- **The shared pipeline is fully built and fully tested already** — `auth.go`, `validate.go`,
  `dispatch.go`, `classify.go`, `respond.go`, `pipeline.go` (Stage Contract), `handlers.go`
  (`registerProfileSectionRoute` + `routeConfig`). None of these need to change for this story.
- **`ProfileSection` string must be the exact chaincode enum value**, case-sensitive:
  `PERSONAL`/`EMPLOYMENT`/`EDUCATION`/`ADDITIONAL`/`PAYROLL` (confirmed against
  `fabric-network/chaincode/employeeprofilerecord/chaincode/asset.go`'s `SectionEmployment` constant
  and `IsValidProfileSection`) — `"EMPLOYMENT"` is correct for this story, not e.g. `"TRANSFER"`.
- **Confidentiality register applies to `integration-bridge/`** (a hard-clean-zone directory, per this
  repo's register) — Story 1.2 self-caught and fixed two accidental real-name mentions in code
  comments before finishing; run `grep -rniE "talenta|mekari" *.go` in `integration-bridge/` before
  calling this story done, same as Story 1.2 did.
- **`go.work` gotcha still applies** — run all Go commands from inside `integration-bridge/`.
- **A checkbox claiming something the diff doesn't actually do is a real finding**, not cosmetic
  (caught twice already, Stories 1.1 and 1.2's own code reviews) — only check a task off once the
  route is genuinely registered and dispatching, not merely present in a test stub.

### Project Structure Notes

- Modifies (not creates fresh): `main.go` only (one new `registerProfileSectionRoute` call).
- New file: this story's own `_test.go` for the EMPLOYMENT route (e.g. `employment_route_test.go`,
  following `handlers_test.go`'s existing pattern of one test file per route once more than one route
  exists — or add to `handlers_test.go` directly if that reads more naturally; either is fine, this
  story does not mandate a specific file split).
- No new config, no new dependencies, no chaincode/network changes.

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 1.3] — the approved ACs, verbatim
- [Source: _bmad-output/planning-artifacts/architecture/architecture-fabric-hris-2026-08-08/ARCHITECTURE-SPINE.md] — AD-5 (five identical routes)
- [Source: write-path-integration/writepaths/writepaths.go] — `Hooks.ApproveEmploymentTransfer`'s exact signature (read directly, this session)
- [Source: fabric-network/chaincode/employeeprofilerecord/chaincode/asset.go] — the five ratified `ProfileSection` enum values (read directly, this session)
- [Source: integration-bridge/main.go, handlers.go] — current registration pattern (read directly, this session)
- [Source: _bmad-output/implementation-artifacts/1-2-talenta-core-can-anchor-a-personal-section-change.md] — previous story, `review`, the pipeline this story builds on

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (claude-opus-5[1m])

### Debug Log References

- `go vet ./... && go test ./...` — 63 pass, 0 fail (dev-story); 87 pass, 0 fail (post-review)
- `go vet -tags=integration ./...` — clean; neither `//go:build integration` test executed (no live network, confirmed via `docker ps`)
- `go build -o /tmp/ib-smoketest . && rm -f /tmp/ib-smoketest` — real binary builds clean
- `grep -rniE "talenta|mekari" *.go` — clean

### Completion Notes List

- This was a pure wiring story: one new `registerProfileSectionRoute` call in `main.go`, confirming the shared pipeline built in Story 1.2 generalizes to a second route with zero pipeline changes, as its own Dev Notes scope boundary required.
- The generic route tests (mirroring Story 1.2's) passed immediately without any implementation change, which is itself a meaningful confirmation that `registerProfileSectionRoute`/`dispatch`/`classify`/`respond` are genuinely route-agnostic — not just untested with a second route.
- Disclosed a real, structural test-coverage limit: no unit test can prove `main.go`'s one new line dispatches to the *correct* `Hooks` method (as opposed to a copy-paste reuse of `UpdatePersonalData`) without either a live-network integration test or adding a test-only seam to `main()` — chose not to add that seam for a one-line story, consistent with Story 1.2's "don't invent scope this story doesn't need" precedent.
- All ACs covered; full existing suite green, no regressions.
- **Code review remediation:** 3 independent layers found 23 raw findings, deduplicated to 16. No production-code bugs this time (the shared pipeline already absorbed those in Story 1.2's own review) — every patch was a test-rigor or documentation gap: missing HTTP-code assertion for AC#2's equivalence claim, a hardcoded test ID bypassing the `uniqueTestID` helper, a missing naming-trap guard test this story's own Dev Notes had flagged but never converted to a test (Story 1.5 already had the pattern to copy), a dropped assertion from the test this one was modeled on, incomplete pass-through assertions, `postSection`'s misleading failure mode on non-JSON responses, and — most notably — `CLAUDE.md`'s confidentiality-register hard-clean list genuinely never listed `integration-bridge/`, despite every story in this epic enforcing it as such. Fixed `CLAUDE.md` directly. 8 low-severity findings deferred; 1 already-disclosed limitation dismissed (no new fix available). Suite grew 86→87 tests (net, since Story 1.2's own review landed between this story's dev-story and its own review).

### File List

**New:**
- `integration-bridge/employment_route_test.go`
- `integration-bridge/employment_route_integration_test.go`

**Modified:**
- `integration-bridge/main.go` — added EMPLOYMENT route registration
- `integration-bridge/personal_route_integration_test.go` — generalized `postPersonalSection` → `postSection(t, httpAddr, section, payload)`; code review: `postSection` now reports status code + raw body on decode failure
- `CLAUDE.md` — code review: added `integration-bridge/` to the confidentiality register's hard-clean list

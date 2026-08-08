# Deferred Work

## Deferred from: code review of 1-1-module-scaffold-and-long-lived-fabric-connection (2026-08-08)

- Two distinct failure classes (config/bind failures vs. post-serving shutdown failures) are
  collapsed into one `log.Fatal` exit path in `integration-bridge/main.go`. Deferred: no
  operational tooling consumes exit codes yet — deployment topology itself is still an open
  Deferred item in `ARCHITECTURE-SPINE.md`. Revisit once a real deployment target exists.
- `wiring.go`'s `newGatewayClientFunc` type could in principle be satisfied by a stub returning
  `(nil, nil)` with no guard against it in `buildHooks`. Deferred: verified unreachable via the
  only real implementation (`gatewayclient.NewGatewayClient` never returns `(nil, nil)` on any
  return path, checked directly). A defensive nil-check would be reasonable future hardening if a
  second implementation of the function type is ever introduced, but is not urgent today.

## Deferred from: code review of 1-2-talenta-core-can-anchor-a-personal-section-change (2026-08-08)

- `registerProfileSectionRoute`/`dispatch` never nil-checks `route.Dispatch` before calling it — a
  misconfigured `routeConfig` would panic. Deferred: not reachable via any of the 5 current real
  call sites (all pass real, bound `Hooks` methods). `integration-bridge/handlers.go`,
  `integration-bridge/dispatch.go`.
- No rate limiting or lockout on repeated failed authentication attempts. Deferred: infra-level
  concern beyond this story's scope, no AC calls for it. `integration-bridge/auth.go`.
- Status strings (`"committed"`/`"rejected"`/`"partial_failure"`/`"error"`) are untyped
  magic-string literals repeated across files rather than a shared typed constant. Deferred:
  style/maintainability only. `integration-bridge/classify.go`, `integration-bridge/respond.go`.
- `subtle.ConstantTimeCompare`'s documented length-mismatch short-circuit theoretically leaks
  configured-credential length via timing. Deferred: low real-world exploitability over HTTP; a
  real fix is a design change (hash-then-compare), not a one-line patch.
  `integration-bridge/auth.go:28,33`.
- No IPFS reachability check at startup — a misconfigured IPFS endpoint surfaces only on the first
  document-carrying request, not at boot. Deferred: operational nice-to-have, not a correctness
  bug. `integration-bridge/wiring.go`.
- Millisecond-scale timeouts (5ms/10ms budgets) in `dispatch_test.go` are a latent CI-flakiness
  source under scheduler jitter. Deferred: no observed flake yet, revisit if CI proves flaky.
  `integration-bridge/dispatch_test.go`.
- `personal_route_integration_test.go`'s environment-setup block is duplicated (not shared) from
  `main_integration_test.go`'s equivalent block — future config additions require updating both by
  hand. Deferred: cosmetic duplication, low file count today.
  `integration-bridge/personal_route_integration_test.go:32-49`.
- Hardcoded fixed test ports (e.g. `18082`, `18083`) have no collision check against another
  process already bound to that port. Deferred: pre-existing convention from Story 1.1's own
  already-reviewed `main_integration_test.go`, continued rather than introduced here.
  `integration-bridge/personal_route_integration_test.go`.

## Deferred from: code review of 1-3-talenta-core-can-anchor-an-employment-section-change (2026-08-08)

- No document-carrying integration test exists for the EMPLOYMENT route (PERSONAL got one, proving
  the shared IPFS nil-pointer-panic fix). Deferred: the underlying guarantee is pipeline-level and
  already proven once. `integration-bridge/employment_route_integration_test.go`.
- `postSection` cannot gracefully handle a non-JSON-decodable response at all — a routing mismatch
  surfaces as a misleading decode-failure message. Deferred: a fuller fix (conditional decode, or a
  separate raw-response helper) is a test-infrastructure design question, not a one-line change.
  `integration-bridge/personal_route_integration_test.go`.
- `postSection`'s `section` parameter is concatenated into the URL with no escaping/validation.
  Deferred: no current caller can trigger it (every call site passes a hardcoded literal).
  `integration-bridge/personal_route_integration_test.go:92`.
- Auth header literal values (`"secret-key"`, `"tenant01"`) are hand-duplicated across every route
  test file instead of derived from `testAuthConfig()`. Deferred: pre-existing convention from
  Story 1.2, reproduced rather than introduced. `integration-bridge/employment_route_test.go`.
- `"tenant01"` appears twice in each route test with no shared symbol tying the registration's
  `tenantID` argument to the `X-Company-ID` header value. Deferred: informational.
  `integration-bridge/employment_route_test.go`.
- At least 5 `//go:build integration` tests now exist across this epic and none have ever actually
  executed against a live network; nothing beyond prose comments tracks that they eventually need
  to. Deferred: epic-level process concern, not fixable by a single story's patch.
  `integration-bridge/*_integration_test.go`.
- `startPersonalRouteTestBridge` keeps its PERSONAL-specific name despite being reused generically
  by every route's integration test. Deferred: cosmetic naming nit, no functional effect.
  `integration-bridge/personal_route_integration_test.go`.
- Edge Case Hunter proposed several more route-level tests (a non-nil dispatch-error case,
  partial/mismatched auth header combinations, a wrong-HTTP-method request, empty-field rejection)
  — all already exhaustively covered generically by the shared pipeline's own tests against the
  PERSONAL route. Deferred per this epic's own established principle (one representative rejection
  test per failure class is enough). `integration-bridge/employment_route_test.go`.

## Deferred from: code review of 1-4-talenta-core-can-anchor-an-education-section-change (2026-08-08)

- No document-carrying test dimension exists for the EDUCATION route. Deferred: the underlying
  guarantee is pipeline-level and already proven once (same reasoning as Story 1.3's identical
  defer item). `integration-bridge/education_route_test.go`, `education_route_integration_test.go`.
- Hardcoded fixed integration-test port (`18085`) has no dynamic-allocation or cross-file collision
  check. Deferred: systemic pre-existing pattern already logged in Story 1.2's deferred-work entry,
  not introduced here. `integration-bridge/education_route_integration_test.go`.
- `startPersonalRouteTestBridge` keeps its PERSONAL-specific name despite being reused generically
  by a fourth route's test file now. Deferred: same cosmetic nit already logged in Story 1.3's
  deferred-work entry. `integration-bridge/education_route_integration_test.go`.
- Route path case-sensitivity (`/v1/profile-sections/education` vs `EDUCATION`) is untested.
  Deferred: stdlib `ServeMux` behavior, not app logic; low value to re-test per route.
  `integration-bridge/education_route_test.go`.
- Edge Case Hunter proposed several more route-level tests (non-nil dispatch error, mismatched
  auth headers, missing userID/malformed-JSON variations, wrong HTTP method) — all already
  exhaustively covered generically by the shared pipeline's own tests. Deferred per this epic's
  established principle. `integration-bridge/education_route_test.go`.
- No integration-level test forces a non-committed outcome (rejection/timeout) for EDUCATION
  end-to-end. Deferred: same reasoning as the document-carrying gap above; pipeline-level, proven
  once. `integration-bridge/education_route_integration_test.go`.

## Deferred from: code review of 1-5-talenta-core-can-anchor-an-additional-section-change (2026-08-09)

- **Cross-story note, not fixed here:** Story 1.3's `TestRegisterProfileSectionRoute_TransferPath_IsNotRegistered`
  (`integration-bridge/employment_route_test.go`) has the identical tautology found and fixed in this
  story's own naming-trap test — it builds a throwaway mux with a hardcoded-correct `"EMPLOYMENT"`
  literal rather than exercising `main.go`'s real `buildMux()`, so it would pass regardless of what
  `buildMux()` actually registers. Deferred: Story 1.3 is already `done`; reopening it is out of this
  review's scope, but a future cleanup pass should apply the same fix (use `buildMux(stubHooksForMuxTest(),
  testAuthConfig(), time.Second)` instead of a fresh `http.NewServeMux()` + manual `routeConfig`).
- Edge Case Hunter proposed several more route-level tests (non-nil dispatch error, wrong HTTP
  method, path case-sensitivity, malformed/truncated JSON, missing-field variations, mismatched
  auth header combinations, dispatch-timeout-exceeded behavior) — all already exhaustively covered
  generically by the shared pipeline's own tests. Deferred per this epic's established principle.
  `integration-bridge/additional_route_test.go`.
- No document-carrying test dimension exists for the ADDITIONAL route (family/dependent documents
  via the IPFS-backed path). Deferred: pipeline-level guarantee already proven once.
  `integration-bridge/additional_route_test.go`, `additional_route_integration_test.go`.
- The integration test's `recordID` type-assertion doesn't distinguish "wrong type" from
  "missing/empty" in its failure message. Deferred: cosmetic diagnostic-quality nit, not executed
  this session anyway. `integration-bridge/additional_route_integration_test.go`.

## Deferred from: code review of 1-6-talenta-core-can-anchor-a-payroll-section-change-completing-all-five-routes (2026-08-09)

- No test proves `buildMux` wires each route to its correct, distinct `Hooks` method (AC#3's actual
  core claim). Deferred and tracked as `grounding-gaps.md` **G-35** — properly closing it needs a
  full stub `Hooks` (all 4 store interfaces) or a live-network read-back, both nontrivial new
  infrastructure. `integration-bridge/main.go`, `integration-bridge/mux_test.go`.
- The live-network integration test for AC#1's byte-precision claim only proves "a commit
  succeeded," not that the digits survived intact — the same pseudonym-opacity problem G-34
  documents blocks any read-back verification. Deferred and tracked as **G-36**.
  `integration-bridge/payroll_route_integration_test.go`.
- `mux_test.go`'s `TestBuildMux_PersonalAndPayrollAreIndependentRoutes` is a strict subset of
  `TestBuildMux_AllFiveRatifiedRoutesExist`, and `TestBuildMux_AllFiveRoutesRegistered_UnrelatedPathIs404`
  is near-tautological next to `TestBuildMux_SixthSectionName_Is404`. Deferred: harmless redundancy,
  not worth the churn of removing passing tests. `integration-bridge/mux_test.go`.
- No naming-trap guard test exists for PAYROLL. Deferred: arguably correctly scoped (no
  name-divergence exists for this route, unlike ADDITIONAL/EMPLOYMENT), noted as a deliberate
  scope call rather than an oversight. `integration-bridge/payroll_route_test.go`.
- `TestIntegration_UnregisteredSectionName_Is404`'s live-network value is thin relative to
  `TestBuildMux_SixthSectionName_Is404`. Deferred: kept for live-network end-to-end assurance
  despite the overlap. `integration-bridge/payroll_route_integration_test.go`.
- Manually-incremented sequential ports/ID suffixes across all 5 route integration test files have
  no shared registry. Deferred: systemic pattern already logged across prior stories.
  `integration-bridge/payroll_route_integration_test.go` and siblings.
- Wrong-HTTP-method requests to the 5 ratified routes are untested. Deferred: stdlib `ServeMux`
  guarantee, established epic-wide principle. `integration-bridge/mux_test.go`.
- `bankAccountNumber` sent as a quoted JSON string is untested. Deferred: outside AC#1's literal
  scope (a JSON number). `integration-bridge/payroll_route_test.go`.
- `buildMux` would nil-dereference `hooks.TenantID` if ever called with a nil `*writepaths.Hooks`.
  Deferred: not reachable via the single real call site in `run()`. `integration-bridge/main.go`.

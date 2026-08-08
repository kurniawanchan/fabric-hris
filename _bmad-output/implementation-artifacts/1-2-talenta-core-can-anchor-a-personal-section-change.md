---
baseline_commit: NO_VCS
---

# Story 1.2: talenta-core Can Anchor a PERSONAL Section Change

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As talenta-core's PERSONAL write path,
I want to call the bridge with a section update and get back a clear outcome,
so that I can safely act on it without understanding Fabric internals.

## Acceptance Criteria

1. **Given** a valid `POST /v1/profile-sections/PERSONAL` request with correct `X-Api-Key`/`X-Company-ID`, **when** it carries `employeeInternalID`, `userID`, a nested-JSON `newValue`, no `document`, **then** the bridge dispatches to `Hooks.UpdatePersonalData` and returns `status: "committed"` with the resulting `recordID`. [Source: epics.md Story 1.2 AC1]
2. **Given** a missing/incorrect `X-Api-Key` or `X-Company-ID`, **when** `[auth]` rejects it, **then** the response carries `status: "error"` per AD-3's table, and no `Hooks` method is ever called. [Source: epics.md Story 1.2 AC2; ARCHITECTURE-SPINE.md AD-3]
3. **Given** a malformed envelope (missing `employeeInternalID`, invalid `newValue`), **when** `[validate]` rejects it, **then** the response carries `status: "rejected"`, and no `Hooks` method is ever called. [Source: epics.md Story 1.2 AC3; ARCHITECTURE-SPINE.md AD-3]
4. **Given** `newValue` contains a large-integer field, **when** `[validate]` parses the envelope, **then** it is captured via `json.RawMessage` and passed through byte-for-byte — never `map[string]interface{}`/`interface{}` decoding. [Source: epics.md Story 1.2 AC4; ARCHITECTURE-SPINE.md Consistency Conventions]
5. **Given** `Hooks.UpdatePersonalData` returns `*writepaths.PartialFailureError`, **when** `[map-error]` classifies it, **then** the response carries `status: "partial_failure"`, with `employeeInternalID`/`profileSection` sourced from the Stage Contract struct — never from unwrapping the error. [Source: epics.md Story 1.2 AC5; ARCHITECTURE-SPINE.md AD-1/AD-3]
6. **Given** the configured dispatch timeout elapses mid-call, **when** the deadline fires, **then** the response carries `status: "partial_failure"` (not `"timeout"`), and `[dispatch]` never races a goroutine against the call — it blocks synchronously (AD-4). [Source: epics.md Story 1.2 AC6; ARCHITECTURE-SPINE.md AD-4]
7. **Given** any response, successful or not, **when** it is logged or returned, **then** the actual bytes of `newValue`/`document` never appear in any log line or in `detail`. [Source: epics.md Story 1.2 AC7]
8. **Given** a request includes a `document` (base64-encoded, FR-30), **when** the bridge decodes it and passes the raw bytes to `Hooks`, **then** `Hooks.IPFS.EncryptAndAdd` is invoked and the resulting CID appears in the record's `ipfsCIDs` — the bridge itself never persists, forwards, or logs the decoded document bytes anywhere other than into that one call (FR-31). [Source: epics.md Story 1.2 AC8]

## Tasks / Subtasks

- [x] Task 1: Wire a real `ipfsclient.Client` — Story 1.1 deliberately left this `nil`; this story needs it (AC: #8)
  - [x] Add two new required config values (extend `LoadConfig`'s existing fail-fast set): `BRIDGE_IPFS_PRIMARY_API`, `BRIDGE_IPFS_REPLICA_API`
  - [x] In `wiring.go`'s `buildHooks`, replace `IPFS: nil` with `IPFS: ipfsclient.NewClient(cfg.IPFSPrimaryAPI, cfg.IPFSReplicaAPI)`
  - [x] **Read this before touching anything else:** `doAnchor` (`write-path-integration/writepaths/writepaths.go`) calls `h.IPFS.EncryptAndAdd(ctx, documentKey, document)` **unconditionally whenever `len(document) > 0`**, with no nil-check on `h.IPFS` itself — leaving `IPFS: nil` while any route can accept a `document` is a guaranteed nil-pointer panic on the first request that includes one, not a theoretical risk
- [x] Task 2: Stage Contract — the request-scoped struct every later stage reads from (AC: #5, #7)
  - [x] Define a struct (e.g. `bridgectx`) carrying `EmployeeInternalID`, `ProfileSection`, `TenantID`, `Result []byte`, `Err error` — attach it via one `context.WithValue` call as the request enters `[auth]` (AD-1)
  - [x] Every later stage and all logging reads identifiers from this struct — never from `errors.As`-unwrapping an error (the timeout path in Task 4 has no `*PartialFailureError` to unwrap)
- [x] Task 3: `[auth]` stage (AC: #2)
  - [x] Check `X-Api-Key` and `X-Company-ID` headers against configured values
  - [x] On failure, route to `[map-error]` (not straight to `[respond]`) so AD-3's "only `[map-error]` sets `status`" holds without exception — do not special-case auth/validate rejections as an exempt path (`authenticate` returns an `*authError`; the actual routing to `[map-error]` happens in Task 8's `runPipeline`, not inside `auth.go` itself — `classify` already maps `*authError` → `"error"`)
- [x] Task 4: `[validate]` stage (AC: #1, #3, #4, #7, #8)
  - [x] Parse the JSON body into an envelope struct: `employeeInternalID`, `userID` string fields; `newValue json.RawMessage` (never `map[string]interface{}`/`interface{}` — see Dev Notes "Why json.RawMessage is non-negotiable"); `document` as a base64 string, decoded to `[]byte` here (empty/absent → `nil`, matching `Hooks`' own "no document" convention)
  - [x] Reject (route to `[map-error]`) if `employeeInternalID` or `userID` is empty, or `newValue` isn't valid JSON, or `document` (if present) isn't valid base64
  - [x] Store `EmployeeInternalID`/`ProfileSection` (literal `"PERSONAL"` for this route) into the Stage Contract struct here, before dispatch — this is the only place they're set (wired in Task 8's handler, immediately after `validateRequest` succeeds)
- [x] Task 5: `[dispatch]` stage (AC: #1, #5, #6)
  - [x] `context.WithTimeout(parentCtx, budget)` as the **literal first statement** of this stage's handler body — after `[auth]`/`[validate]` have already returned success (AD-4); `budget`'s exact duration is a config value, not hardcoded (Deferred in the spine pending real Caliper numbers — pick any placeholder duration via config, don't invent a magic number as if it were final) — implemented as `dispatch.go`'s `dispatch()` + `config.go`'s `ResolveDispatchTimeout` (env `BRIDGE_DISPATCH_TIMEOUT`, default 30s placeholder)
  - [x] Call `Hooks.UpdatePersonalData(ctx, employeeInternalID, userID, []byte(newValue), document)` synchronously — **do not** race a goroutine against the timeout (AD-4's Rule explicitly rejects this: a detached still-running submit racing an already-sent HTTP response is a double-submission risk against the one shared, retained connection) — `dispatch()` takes a `dispatchFunc`-typed parameter (dependency-inverted, same technique as Story 1.1's `newGatewayClientFunc`) so Task 8 wires the real `hooks.UpdatePersonalData` while tests use a controllable stub; `TestDispatch_NeverRacesGoroutine_BlocksUntilFnReturns` proves the synchronous-blocking property
  - [x] Store the raw `(result []byte, err error)` into the Stage Contract struct for `[map-error]` to read — wired in Task 8's handler (`bridgectx.Result`/`bridgectx.Err`)
- [x] Task 6: `[map-error]` stage — AD-3's decision table (AC: #1, #2, #3, #5, #6)
  - [x] Implement exactly this mapping, nowhere else in the codebase:

    | Failure origin | `status` |
    |---|---|
    | `[auth]` rejection | `error` |
    | `[validate]` rejection | `rejected` |
    | `Hooks` untyped error (pre-`anchor()`, e.g. `SaveSection` failure) | `error` |
    | `*writepaths.PartialFailureError` from `anchor()`, **or** a `context.DeadlineExceeded` from the `[dispatch]` timeout | `partial_failure` |
    | no error | `committed` |
  - [x] For the `partial_failure` row, use `errors.As` only to detect the error TYPE (is it a `*PartialFailureError`, or a timeout, or something else) — never to SOURCE `employeeInternalID`/`profileSection` for the response; those always come from the Stage Contract struct (Task 2), because the timeout case has no `*PartialFailureError` to unwrap them from
- [x] Task 7: `[respond]` stage (AC: #1, #7)
  - [x] Write `{"status": "...", "recordID"?: string, "detail"?: string}` — `recordID` only present on `committed` — implemented as `respond.go`; `recordID` is extracted from `result`'s raw chaincode-JSON bytes (`RecordProfileSectionResult`, itself PII-free) via a minimal `chaincodeResult` struct, `omitempty` on both optional fields
  - [x] **Never** put `newValue` or `document` bytes into `detail` under any circumstance, including error cases — grepped the diff: `detail` is populated only from `err.Error()`, and every error type reachable here (`authError`, `validationError`, `*writepaths.PartialFailureError`, plain `Hooks` errors, `context.DeadlineExceeded`) is built from fixed strings and pseudonymous identifiers, never from echoing request body content — dedicated full-pipeline proof deferred to Task 9's test (this task's own tests only cover `respond()` in isolation). **New finding, not fixed here:** AD-3's HTTP-status convention (`4xx`/`5xx` as pure transport) doesn't cleanly cover the `"error"` body status, which `classify()` deliberately collapses from two different-shaped origins ([auth] credential fault vs. a bridge/dependency fault) — chose `500` for `"error"`; recorded as `grounding-gaps.md` **G-33**, not silently resolved
- [x] Task 8: Wire the PERSONAL route (AC: #1)
  - [x] `mux.HandleFunc("POST /v1/profile-sections/PERSONAL", ...)` in `handlers.go`, feeding the shared pipeline (Tasks 3-7) with `ProfileSection = "PERSONAL"` and `Hooks.UpdatePersonalData` as the dispatch target — implemented as `handlers.go`'s `registerProfileSectionRoute(mux, path, routeConfig, authCfg, tenantID, timeout)`, a reusable helper Stories 1.3-1.6 call again with a different `routeConfig`/path rather than duplicating the handler body; also added `Config.APIKey`/`Config.CompanyID` (new required env vars `BRIDGE_API_KEY`/`BRIDGE_COMPANY_ID`) and `ResolveDispatchTimeout` (optional, `BRIDGE_DISPATCH_TIMEOUT`, 30s placeholder default) since nothing previously wired real values into `authConfig`/`dispatch`'s budget
  - [x] `main.go`'s `hooks` (already retained per Story 1.1's own remediation — see Previous Story Intelligence) is what this route dispatches through; no second `Hooks`/`GatewayClient` construction anywhere — `hooks.UpdatePersonalData` (a method value) is passed directly as the route's `dispatchFunc`
  - **Note on AC#5's wording:** AC#5 says the *response* should source `employeeInternalID`/`profileSection` from the Stage Contract, but AD-3's ratified wire envelope (`{status, recordID?, detail?}`) has no slot for them — there is no logging/observability sink in this codebase yet to put them in either. Implemented as: `bc` (the Stage Contract) is correctly populated with both fields immediately after `[validate]` succeeds, before `[dispatch]` runs, and stays populated (unlike unwrapping `*PartialFailureError`) even on the timeout path — satisfying the underlying intent (identifiers are reliably available regardless of failure origin) without inventing wire-format or logging scope this story doesn't otherwise need. Flagged, not silently resolved.
- [x] Task 9: Tests
  - [x] Unit tests for `[validate]`: reject missing `employeeInternalID`/`userID`, invalid `newValue` JSON, invalid `document` base64; confirm `newValue` survives round-trip as `json.RawMessage` byte-for-byte for a large-integer field (e.g. a 19-digit number) without becoming `float64`-corrupted — `validate_test.go` (Task 4)
  - [x] Unit test for `[map-error]`: every row of the decision table above, using stub errors (a `*writepaths.PartialFailureError`, a plain `errors.New`, a `context.DeadlineExceeded`, nil) — assert `status` and assert identifiers come from the Stage Contract struct, not from unwrapping — `classify_test.go` (Task 6)
  - [x] Unit test: confirm no log line or `detail` field ever contains the literal bytes of a test `newValue`/`document` fixture, across every status outcome — `handlers_test.go`'s `TestRegisterProfileSectionRoute_NeverLeaksNewValueOrDocumentBytesInResponse`, 4 subtests (committed/rejected/auth-error/partial_failure)
  - [x] Unit test: `[dispatch]`'s timeout fires before a deliberately slow stub `Hooks` call returns → `status: "partial_failure"`, not a distinct timeout value — `dispatch_test.go`'s `TestDispatch_TimeoutFires_ReturnsDeadlineExceeded` (isolated) and `handlers_test.go`'s `TestRegisterProfileSectionRoute_DispatchTimeout_ReturnsPartialFailureNotTimeout` (full pipeline)
  - [x] Integration test (`//go:build integration`): a real `POST /v1/profile-sections/PERSONAL` against the live network, with and without a `document`, confirming `ipfsCIDs` is populated in the latter case — **not executed this session** (`docker ps` confirmed no Fabric/IPFS containers running, only unrelated dev containers). Written as `personal_route_integration_test.go`, compiles clean under `-tags=integration`. The "with a `document`" case only proves the request commits without the nil-pointer panic Dev Notes flagged — it does **not** prove `ipfsCIDs` actually lands on the ledger, because verifying that needs the on-chain `employeeID` pseudonym, which is never returned to any external caller by design. Recorded as `grounding-gaps.md` **G-34**, not silently claimed as covered. Also fixed `main_integration_test.go`'s env fixture, which was missing the 4 config vars this story added (`BRIDGE_IPFS_PRIMARY_API`/`BRIDGE_IPFS_REPLICA_API`/`BRIDGE_API_KEY`/`BRIDGE_COMPANY_ID`) and would otherwise fail `LoadConfig` if run
  - [x] `go vet ./...`, `gofmt -l`, full existing suite (Story 1.1's 17 tests) still green — no regressions — 61 tests pass, 0 fail; `go build ./...` and a real binary build both clean; confidentiality grep (`grep -rniE "talenta|mekari" *.go`) re-run clean after fixing two comment leaks introduced and caught in this same task (see Completion Notes)

### Review Findings

**Review layers:** Blind Hunter (adversarial), Edge Case Hunter, Acceptance Auditor (spec: this story). 32 raw findings across the three layers; merged/deduplicated to 23 below (20 surviving, 3 dismissed) after independently reading the actual source at every location before rating.

- [x] [Review][Decision→Patch] `dispatch()` discards a genuinely successful result when the call runs past its budget but still completes — `status` becomes `partial_failure` with the real `recordID` thrown away, even though the write definitively committed and the caller is still on the same open, blocked connection the entire time. Tension with AC#6's literal wording ("the deadline fires... status: partial_failure"), which as written doesn't distinguish "still running" from "finished successfully after the clock ran out." **Resolved by user:** only override to `partial_failure` when `fn` ALSO returned a non-nil error (a genuine success is always reported as `committed`, however long it took). **Applied:** `dispatch.go`'s condition is now `err != nil && dispatchCtx.Err() == context.DeadlineExceeded`; 2 new tests (`TestDispatch_TimeoutFires_FnStillSucceeds_ReturnsRealSuccess`, `TestRegisterProfileSectionRoute_SlowButSuccessfulDispatch_ReturnsCommitted`), 1 existing test's stub updated to also return an error. — [integration-bridge/dispatch.go:32-33]
- [x] [Review][Decision→Patch] The Stage Contract (`bridgectx`/`withBridgeCtx`/`bridgeCtxFrom`) is documented in three places (`pipeline.go`, `classify.go`, and this story's own Task 2) as the mechanism every later stage and all logging use to read `EmployeeInternalID`/`ProfileSection` — but `bridgeCtxFrom` is never called from any production code path (only its own unit test calls it); `classify`/`respond` don't even accept a `*bridgectx` parameter. The real mechanism is direct closure-variable mutation in `handlers.go`, which produces correct output today but makes the documented architectural guarantee (AD-1) unenforced by the compiler and only true by construction of one specific closure shape. **Resolved by user:** keep the closure-based approach (no functional change) and correct the doc comments in `pipeline.go`/`classify.go` to describe what's actually true — identifiers are available via closure capture, and `withBridgeCtx`/`bridgeCtxFrom` exist for a future logging stage to adopt, not because `classify()`/`respond()` use them today. **Applied:** doc comments only, no functional change. — [integration-bridge/pipeline.go, integration-bridge/classify.go:24-25, integration-bridge/handlers.go]

- [x] [Review][Patch] A real internal file path and method name (`services/ems/BaseEmsService.php`'s `companyLevelHeaders()`) is cited verbatim in a code comment inside `integration-bridge/` (a hard-clean-zone directory) — violates this story's own explicit Dev Notes constraint and contradicts its Completion Notes' claim of "no other occurrences found" (the closing `grep -rniE "talenta|mekari"` check cannot match this string). **Applied:** reworded to cite `ADR-0022` by name only, matching `config.go`'s own established pattern; confidentiality grep re-verified clean. [integration-bridge/auth.go:18]
- [x] [Review][Patch] No request body size limit anywhere in the request path — `document` is specifically designed to carry base64-encoded file bytes, making this endpoint an unmitigated memory-exhaustion vector. **Applied:** `http.MaxBytesReader` capped at 10 MiB (placeholder, not a ratified number); `validateRequest` now takes `w http.ResponseWriter`; new test `TestValidateRequest_BodyExceedsLimit_Rejected`. [integration-bridge/validate.go:37]
- [x] [Review][Patch] `authenticate` returns two textually distinct error messages depending on which of `X-Api-Key`/`X-Company-ID` failed, and `respond` echoes `err.Error()` verbatim into `detail` — creates a credential-validation oracle that lets a caller distinguish "wrong API key" from "right key, wrong company ID," undermining the constant-time comparison's purpose even though no request-body content is leaked. **Applied:** both branches now return the identical `errMissingOrIncorrectCredentials`; new test `TestAuthenticate_APIKeyAndCompanyIDFailures_ProduceIdenticalMessage`. [integration-bridge/auth.go:29,34; integration-bridge/respond.go:62]
- [x] [Review][Patch] `validateRequest` checks `strings.TrimSpace(...) == ""` for the required-field emptiness check but returns the untrimmed raw `env.EmployeeInternalID`/`env.UserID` — a padded identifier (`"  emp-1  "`) passes validation, then produces a different downstream pseudonym/hash than its trimmed form would. **Applied:** both identifiers are now trimmed before being returned; new test `TestValidateRequest_PaddedIdentifiers_TrimmedBeforeReturn`. [integration-bridge/validate.go:41-44 vs. 59]
- [x] [Review][Patch] `newValue` accepts JSON `null` and any other non-object JSON value, since `len(env.NewValue) == 0` only catches full omission, not the 4-byte `null` literal — bypasses the "required" check and would anchor a meaningless section value. **Applied:** new `isJSONObject` helper rejects anything whose first non-whitespace byte isn't `{`, without ever parsing the value (preserving the byte-for-byte guarantee); new tests `TestValidateRequest_NullNewValue_Rejected`, `TestValidateRequest_NonObjectNewValue_Rejected`. [integration-bridge/validate.go:47]
- [x] [Review][Patch] `ResolveDispatchTimeout` accepts a syntactically valid but non-positive duration (e.g. `"-5s"`) with no floor check — produces an already-expired context, so every request would immediately misclassify as `partial_failure` regardless of whether the real call would have succeeded. **Applied:** `d <= 0` now falls back to the placeholder default, same as an unparseable value; new tests for negative and zero durations. [integration-bridge/config.go:92-103]
- [x] [Review][Patch] The configurable dispatch timeout (explicitly meant to eventually exceed today's 30s placeholder, per its own doc comment) is never reconciled against the HTTP server's independently hardcoded `WriteTimeout: 30 * time.Second` — raising `BRIDGE_DISPATCH_TIMEOUT` above 30s would let the server force-close the connection before the handler can ever write a response. **Applied:** new `serverWriteTimeout(dispatchTimeout)` derives `WriteTimeout` as `dispatchTimeout + 10s`, so it can never fall below the dispatch budget it must contain; new test `TestServerWriteTimeout_AlwaysExceedsDispatchTimeout`. [integration-bridge/main.go]
- [x] [Review][Patch] Integration tests use fixed, non-unique identifiers (`"integration-test-emp-1"`, etc.) against an append-only ledger with no teardown — every re-run appends another permanent history entry under the same identifiers. **Applied:** new `uniqueTestID(base)` helper (base + `time.Now().UnixNano()`) used in both `personal_route_integration_test.go` tests; not executed this session (no live network) — same 4 sibling files under Stories 1.3-1.6 have the identical pattern and are noted for their own future review, not silently fixed here since that's out of this review's scope. [integration-bridge/personal_route_integration_test.go:110-114,149-153]
- [x] [Review][Patch] `cmd.Env = append(cmd.Env, ...)` starts from a nil slice, so the spawned bridge subprocess loses the parent's entire environment (`PATH`, `HOME`, etc.) instead of inheriting it — should seed from `os.Environ()` first. Same pattern also exists in `main_integration_test.go` (Story 1.1's file); bundling that fix in since it's the identical one-line change and this review is already touching the sibling file. **Applied:** both files now use `append(os.Environ(), ...)`; compiles clean under `-tags=integration`, not executed this session. [integration-bridge/personal_route_integration_test.go:32; integration-bridge/main_integration_test.go]
- [x] [Review][Patch] A `"committed"` status with an empty or unparseable `result` silently produces `{"status":"committed"}` with no `recordID` and no error signal — nothing treats this as the invariant violation it would be. **Applied:** `respond` now downgrades to `status: "error"` (with a `detail` explaining why) whenever `result` doesn't parse to a non-empty `recordID` on the committed path; existing test renamed/updated (`TestRespond_Committed_MalformedResultBytes_ReportsErrorNotCommitted`), new test `TestRespond_Committed_EmptyRecordID_ReportsErrorNotCommitted`. [integration-bridge/respond.go:55-59]

- [x] [Review][Defer] `registerProfileSectionRoute`/`dispatch` never nil-checks `route.Dispatch` before calling it — a misconfigured `routeConfig` would panic. Not reachable via any of the 5 current real call sites (all pass real, bound `Hooks` methods). [integration-bridge/handlers.go, integration-bridge/dispatch.go] — deferred, pre-existing pattern, no current call site can trigger it
- [x] [Review][Defer] No rate limiting or lockout on repeated failed authentication attempts. [integration-bridge/auth.go] — deferred, infra-level concern beyond this story's scope, no AC calls for it
- [x] [Review][Defer] Status strings (`"committed"`/`"rejected"`/`"partial_failure"`/`"error"`) are untyped magic-string literals repeated across files rather than a shared typed constant. [integration-bridge/classify.go, integration-bridge/respond.go] — deferred, style/maintainability only
- [x] [Review][Defer] `subtle.ConstantTimeCompare`'s documented length-mismatch short-circuit theoretically leaks configured-credential length via timing. [integration-bridge/auth.go:28,33] — deferred, low real-world exploitability over HTTP; a real fix is a design change (hash-then-compare), not a one-line patch
- [x] [Review][Defer] No IPFS reachability check at startup — a misconfigured IPFS endpoint surfaces only on the first document-carrying request, not at boot. [integration-bridge/wiring.go] — deferred, operational nice-to-have, not a correctness bug
- [x] [Review][Defer] Millisecond-scale timeouts (5ms/10ms budgets) in `dispatch_test.go` are a latent CI-flakiness source under scheduler jitter. [integration-bridge/dispatch_test.go] — deferred, no observed flake yet, revisit if CI proves flaky
- [x] [Review][Defer] `personal_route_integration_test.go`'s environment-setup block is duplicated (not shared) from `main_integration_test.go`'s equivalent block — future config additions require updating both by hand. [integration-bridge/personal_route_integration_test.go:32-49] — deferred, cosmetic duplication, low file count today
- [x] [Review][Defer] Hardcoded fixed test ports (e.g. `18082`, `18083`) have no collision check against another process already bound to that port. [integration-bridge/personal_route_integration_test.go] — deferred, pre-existing convention from Story 1.1's own already-reviewed `main_integration_test.go`, continued rather than introduced here

**Dismissed (3):** (1) Edge Case Hunter's claim that parent-context cancellation is conflated with the budget timeout — disproven by reading `dispatch.go`: the code checks `dispatchCtx.Err() == context.DeadlineExceeded` specifically, so a canceled (not timed-out) context falls through to `fn`'s own return value unchanged; the underlying "fn ignores ctx entirely" phenomenon is the same one already captured by the first Decision item above, not a separate bug. (2) Edge Case Hunter's claim that `TenantID`/`CompanyID` need a cross-consistency check — these are legitimately independent identifiers by design (AD-2's Consistency Conventions: one is talenta-core's own header convention, the other is this bridge's Fabric-channel identifier), not a validation gap. (3) Edge Case Hunter's finding about a second SIGTERM having no shutdown-escalation effect — that code (`main.go`'s signal-handling block) is Story 1.1's original, already-reviewed diff, untouched by this story.

## Dev Notes

**Scope boundary.** This story builds the entire shared pipeline (`pipeline.go`) and wires exactly
one route (`PERSONAL`) through it, per `ARCHITECTURE-SPINE.md`'s Structural Seed. Stories 1.3–1.6
each add one more route on top of this same, unmodified pipeline — if this story's pipeline needs
route-specific special-casing to make PERSONAL work, that's a design smell to flag, not silently
work around, since AD-5 requires all 5 routes to behave identically apart from which `Hooks` method
they call.

### Why `json.RawMessage` is non-negotiable (AC#4) — the exact call chain, verified

`Hooks.UpdatePersonalData(ctx, employeeInternalID, userID string, newValue, document []byte)` passes
`newValue` straight through as `sectionValueJSON` into `doAnchor`, which calls
`gatewayclient.ComputeDataHash(salt, sectionValueJSON)` — read directly:

```go
// write-path-integration/gateway-client/digestbuilder.go
func ComputeDataHash(salt []byte, sectionJSON []byte) (string, error) {
	canonical, err := jsoncanonicalizer.Transform(sectionJSON)
	...
	h.Write(canonical)
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}
```

**No re-serialization happens anywhere in between.** The exact bytes `[validate]` captures are the
exact bytes `jsoncanonicalizer.Transform` canonicalizes and hashes. If `[validate]` unmarshals
`newValue` into `map[string]interface{}` before re-marshaling it (e.g. to run field-level checks),
Go's `encoding/json` converts every JSON number to `float64` — silently changing the digit string
for anything past `float64`'s exact-integer boundary (2^53), which PAYROLL-shaped bank-account
numbers (Story 1.6) and potentially other large IDs can exceed. `json.RawMessage` captures the raw
bytes without ever parsing the numbers inside them, so this can't happen. This is not a style
preference — it is the literal mechanism protecting `DataHash`'s correctness.

### The exact nil-pointer risk this story must close (AC#8) — verified against `doAnchor`'s real body

```go
// write-path-integration/writepaths/writepaths.go, inside doAnchor
ipfsCIDsJSON := "[]"
if len(document) > 0 {
	documentKey, err := h.DocumentKeys.GetOrCreateDocumentKey(ctx, employeeInternalID)
	...
	cid, err := h.IPFS.EncryptAndAdd(ctx, documentKey, document)  // <-- h.IPFS dereferenced here, unconditionally
	...
}
```

There is **no nil-check on `h.IPFS`** anywhere in this path. Story 1.1 left `IPFS: nil` correctly
(no route existed yet to carry a document) — this story is exactly the point where that stops being
safe. `wiring.go`'s `buildHooks` must construct a real `ipfsclient.Client` (`ipfsclient.NewClient(primaryAPI,
replicaAPI string) *Client` — confirmed signature) before this route can accept a `document` at all.
Get this wrong and the failure mode is a panic, not a clean error response.

### `PartialFailureError`'s exact fields (Task 6, AC#5) — for the decision-table implementation

```go
// write-path-integration/writepaths/writepaths.go
type PartialFailureError struct {
	EmployeeInternalID string
	ProfileSection     string
	Version            int
	Err                error
}
func (e *PartialFailureError) Unwrap() error { return e.Err }
```

Note this type's own fields already carry `EmployeeInternalID`/`ProfileSection` — it would be
tempting to source `[map-error]`'s response identifiers from `errors.As`-unwrapping this struct
when it IS present. **Don't** — AD-1's Stage Contract rule is to source them from the shared struct
unconditionally, precisely because the timeout case (AC#6) has no `PartialFailureError` at all, and
a `[map-error]` implementation that works two different ways depending on error type is exactly the
kind of two-builders-diverge risk the architecture spine's reviewer gate flagged and closed.

### The sentinel matchers are unexported — physically, not just conventionally, off-limits

`isExpectedRetryableRejection` (`gateway-client/gatewayclient.go`) and `isNotFoundRejection`
(`gateway-client/verify.go`) are both **lowercase, unexported functions** — `integration-bridge`
cannot import and call them even if a developer wanted to; Go's own compiler enforces this, not
just the Consistency Conventions table's stated rule. Do not copy-paste their string-matching logic
into this module either — if a `Hooks` call ever returns an error whose string happens to resemble
one of those sentinels, `[map-error]` still classifies it as a plain untyped error (`error`), not
as anything Fabric-retry-aware. This module has no visibility into that classification and isn't
meant to.

### Previous Story Intelligence (from Story 1.1, `done`, code-reviewed and remediated)

- **`hooks` is already retained in `main.go`, not discarded.** Story 1.1's own code review caught
  `_, gw, err := buildHooks(...)` as a real finding (three independent reviewers flagged the dead
  wiring) — it's now `hooks, gw, err := buildHooks(...)`, with `hooks.TenantID` logged at startup.
  This story's `handlers.go` dispatches through that exact same `hooks` value; do not construct a
  second one.
- **The `closer`/`errors.Join` pattern for "always run both cleanup steps, never let one short-circuit
  the other" already exists** (`shutdown.go`'s `closeBoth`, `main.go`'s `shutdownSequence.Close()`) —
  the original version of that code had a real bug (skipped the second close on the first one's
  error) that shipped past initial dev-story and was only caught in code review. `[map-error]`'s own
  logic has a similar shape risk: don't let an early return in one classification branch skip
  something a later branch depends on (e.g. always populate the Stage Contract's `Err` field before
  branching on its type, not conditionally).
- **Dependency-inversion for testability, applied twice already** (`closer` for `GatewayClient.Close`,
  `newGatewayClientFunc` for `NewGatewayClient`) — apply the same technique here: `[dispatch]`'s call
  to `Hooks.UpdatePersonalData` should be testable against a way to control timing/errors without a
  live network. Consider a small function-typed field or parameter for the `Hooks` method call so
  `[dispatch]`'s timeout-vs-completion race (AC#6) is unit-testable with a deliberately slow stub,
  the same way Story 1.1 tested `buildHooks` against a stub `newGatewayClientFunc`.
- **A checkbox that claims something the diff doesn't actually do is a real, not cosmetic, finding.**
  Story 1.1's code review caught two Task checkboxes asserting work that hadn't happened
  (`ipfsclient.Client` construction; a top-of-file `main.go` comment that was actually in
  `wiring.go`). Don't check off Task 8 above until the route is genuinely wired and dispatching
  through the retained `hooks` — not merely registered on the mux.
- **Confidentiality register:** no code comment in this story should cite a real internal file path
  (Story 1.1's code review found and fixed exactly one such leak in `config.go`, first occurrence,
  now corrected). This story's Dev Notes above cite only this repo's own code and public Go stdlib
  behavior — keep new comments in the diff the same way.
- **`go.work` gotcha still applies** — run all Go commands from inside `integration-bridge/`, never
  the repo root or `write-path-integration/`.

### Project Structure Notes

- Modifies (not creates fresh): `wiring.go` (add `ipfsclient` wiring), `config.go` (two new required
  fields), `main.go` (register the mux's one route — or delegate registration to `handlers.go`, your
  call, but `main.go` must not itself contain per-route business logic).
- New files: `handlers.go` (the PERSONAL handler + shared route-registration helper future stories
  reuse), `pipeline.go` (the 5 stages + Stage Contract struct), plus each file's own `_test.go`.
- No database/entity creation applies.

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 1.2] — the approved ACs, verbatim
- [Source: _bmad-output/planning-artifacts/architecture/architecture-fabric-hris-2026-08-08/ARCHITECTURE-SPINE.md] — AD-1 (Stage Contract), AD-3 (decision table), AD-4 (timeout scoping), AD-5 (route shape), Consistency Conventions (request/response envelope, `json.RawMessage`)
- [Source: write-path-integration/writepaths/writepaths.go] — `Hooks.UpdatePersonalData` signature, `PartialFailureError` struct, `doAnchor`'s exact document/IPFS branch (read directly, this session)
- [Source: write-path-integration/gateway-client/digestbuilder.go] — `ComputeDataHash`'s exact call to `jsoncanonicalizer.Transform` (read directly, this session)
- [Source: write-path-integration/gateway-client/gatewayclient.go, verify.go] — confirmed `isExpectedRetryableRejection`/`isNotFoundRejection` are unexported
- [Source: write-path-integration/ipfsclient/ipfsclient.go] — `NewClient(primaryAPI, replicaAPI string) *Client` signature
- [Source: _bmad-output/implementation-artifacts/1-1-module-scaffold-and-long-lived-fabric-connection.md] — previous story, `done`, its Review Findings section (the specific bugs/patterns listed above)

## Open Questions (not blocking this story — surface before it ships to any real deployment)

1. **Exact `[dispatch]` timeout duration** is still not a ratified number (`ARCHITECTURE-SPINE.md`'s
   own Deferred item — needs real Caliper data). This story should make it configurable, not pick a
   final value.
2. **Config exhaustiveness:** this story adds `BRIDGE_IPFS_PRIMARY_API`/`BRIDGE_IPFS_REPLICA_API` to
   the required set — confirm these are the correct real endpoint conventions (`ipfs-cluster/`'s own
   dev compose exposes kubo APIs on `:5001`/`:5002` per `docs/QUICKSTART.md` §7; production values
   are a deployment-topology question still open in the spine).

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (claude-opus-5[1m])

### Debug Log References

- `go vet ./... && go test ./...` — 61 tests pass, 0 fail, run repeatedly after every task
- `go build -o /tmp/integration-bridge-smoketest . && rm -f /tmp/integration-bridge-smoketest` — real binary builds clean
- `go vet -tags=integration ./...` — the two `//go:build integration` files compile clean; neither executed (no live network this session, confirmed via `docker ps`)
- `grep -rniE "talenta|mekari" *.go` in `integration-bridge/` — clean after fixing two self-introduced leaks (see Completion Notes)
- `gofmt -l .` — no output (clean) after every task

### Completion Notes List

- Tasks 1–2 (IPFS wiring, Stage Contract) were completed in a prior session slice; Tasks 3 (`[auth]`), 4 (`[validate]`), 6 (`[map-error]`/`classify`) had their code and tests finished before this slice began but their checkboxes were caught up retroactively at the start of this slice, in task-number order, to match actual code state.
- Task 5 (`[dispatch]`): added `dispatch.go` plus `ResolveDispatchTimeout` (new optional config, `BRIDGE_DISPATCH_TIMEOUT`, 30s placeholder default — the real budget is still an open architecture question). 5 tests, including one that proves the synchronous-blocking property AD-4's Rule requires (never racing a goroutine against the timeout).
- Task 7 (`[respond]`): added `respond.go`. Found and disclosed a real ambiguity in `ARCHITECTURE-SPINE.md` AD-3 — its HTTP-status convention doesn't cleanly cover the `"error"` body status, which conflates two different-shaped failure origins. Chose `500`, recorded as `grounding-gaps.md` **G-33** rather than silently resolved.
- Task 8 (route wiring): added `handlers.go`'s `registerProfileSectionRoute` + `routeConfig`, reusable as-is by Stories 1.3–1.6. Had to add `Config.APIKey`/`Config.CompanyID` (two new required env vars) since nothing previously constructed a real `authConfig`. Flagged a wording tension in AC#5 (asks for identifiers "in the response" that AD-3's ratified envelope has no field for) rather than silently picking an interpretation or inventing wire-format scope.
- Task 9 (tests/closeout): added the AC#7 leak-proof test (4 outcome variants), the `//go:build integration` PERSONAL-route test, and fixed `main_integration_test.go`'s env fixture (it predates this story's 4 new required config vars and would have failed `LoadConfig` if run as-is). Disclosed honestly that neither integration test ran this session, and that the "document → CID" half of AC#8 can only be proven up to "doesn't panic and commits" from outside the bridge's process boundary — recorded as `grounding-gaps.md` **G-34**.
- **Self-caught confidentiality issue:** while writing Task 9's doc comments, two used "talenta-core" by name inside `integration-bridge/` (a hard-clean-zone directory per this repo's confidentiality register) — caught by this task's own closing `grep -rniE "talenta|mekari" *.go` check before considering the story done, and fixed by rewording to "the caller", matching `write-path-integration/writepaths/writepaths.go`'s own established generic phrasing. No other occurrences found.
- All 8 ACs are covered by the tests above; full existing suite (Story 1.1's 17 + this story's 44 = 61) is green with no regressions.
- **Code review remediation (post-`review`):** 3 independent layers (Blind Hunter, Edge Case Hunter, Acceptance Auditor) found 32 raw findings, deduplicated to 23 after independently re-reading every cited location. 2 decision-needed items resolved by the user (dispatch must not discard a genuinely successful slow result; Stage Contract docs corrected rather than the mechanism forcibly wired in), both converted to patches. All 11 patches applied with TDD (new/updated tests for every behavioral change), including a second, independently-caught confidentiality leak (`auth.go`'s comment cited a real internal file path/method name, missed by the original `talenta|mekari` grep since that pattern couldn't match it — found by the Acceptance Auditor, not self-caught this time). 8 low-severity findings deferred to `deferred-work.md`; 3 findings dismissed after verification against the actual source disproved them. Full suite grew from 61 to 86 tests, still 0 failures; confidentiality grep re-verified clean after the fix.

### File List

**New:**
- `integration-bridge/dispatch.go`, `dispatch_test.go`
- `integration-bridge/respond.go`, `respond_test.go`
- `integration-bridge/handlers.go`, `handlers_test.go`
- `integration-bridge/personal_route_integration_test.go`
- `integration-bridge/pipeline.go`, `pipeline_test.go` (Task 2, prior slice)
- `integration-bridge/classify.go`, `classify_test.go` (Task 6, prior slice)
- `integration-bridge/validate.go`, `validate_test.go` (Task 4, prior slice)
- `integration-bridge/auth.go`, `auth_test.go` (Task 3, prior slice)

**Modified:**
- `integration-bridge/config.go`, `config_test.go` — added `APIKey`, `CompanyID` (required); added `ResolveDispatchTimeout` (optional)
- `integration-bridge/wiring.go` (prior slice — real `ipfsclient.Client`, Task 1)
- `integration-bridge/main.go` — builds the mux, registers the PERSONAL route, wires `authConfig`/dispatch timeout
- `integration-bridge/main_integration_test.go` — env fixture extended with the 4 new required config vars

**Other:**
- `agent-suite/11-execution/grounding-gaps.md` — added **G-33** (AD-3 HTTP-status ambiguity), **G-34** (no external ipfsCIDs verification path)

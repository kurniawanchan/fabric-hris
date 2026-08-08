# Adversarial Review — Integration Bridge Architecture Spine

- **Reviewed artifact:** `_bmad-output/planning-artifacts/architecture/architecture-fabric-hris-2026-08-08/ARCHITECTURE-SPINE.md`
- **Reviewed against:** `write-path-integration/writepaths/writepaths.go`, `write-path-integration/gateway-client/gatewayclient.go`, `write-path-integration/gateway-client/digestbuilder.go`, `fabric-network/chaincode/employeeprofilerecord/chaincode/{asset.go,errors.go}`, `agent-suite/05-adr/ADR-0022-*.md`, `agent-suite/05-adr/ADR-0014-*.md`
- **Method:** for each Rule, construct two independent builders (or two independent handlers/files) who each satisfy every AD's literal text, then check whether their outputs interoperate. A gap is "real" only if both builders can point to spine text justifying their own choice.

## Verdict

The spine correctly pins the big shape decisions (pipeline paradigm, module layout, one shared timeout enforced in one named stage, exactly 5 POST operations, stdlib-only) but leaves five concrete, independently-exploitable ambiguities below that altitude — an inter-stage state-handoff contract, a status-enum decision table, a clock-start/deadline-propagation mechanism, a URL path convention, and a wire-shape for `newValue` — any one of which lets two AD-compliant builders produce code that is individually correct and mutually incompatible, in one case (finding 5) silently corrupting the on-chain digest rather than just producing inconsistent HTTP responses.

---

## Finding 1 — AD-3's `rejected` vs `error` boundary is undefined, and nothing in the actual error surface populates `rejected` at all

**The gap.** AD-3 fixes the enum (`committed | partial_failure | rejected | error`) and fixes *who* sets it (only the error-mapping filter) and fixes *one* mapping precisely (`*writepaths.PartialFailureError` → `partial_failure`). It never says which of the remaining three failure sources maps to `rejected` vs `error`:

1. An auth failure (`[auth]` stage, bad/missing `X-Api-Key`).
2. A validation failure (`[validate]` stage, malformed envelope / missing required field) — never reaches `Hooks` at all.
3. A plain, untyped error from `Hooks.SaveSection` failing (per `writepaths.go:260-303`, every one of the five hook methods returns this error **unwrapped**, *before* `anchor()` — nothing committed, and by the code's own doc comment this is deliberately kept structurally apart from `PartialFailureError`).

**Two compliant builders, incompatible outcomes.**

- **Builder A** (owns `pipeline.go`'s auth/validate stages) reads AD-3's framing "4xx = bad input from talenta-core" and treats *every* pre-dispatch rejection (bad API key, malformed body) as `status: "rejected"` — a request that was refused before any work happened.
- **Builder B** (owns the map-error stage) reads the same AD-3 text as "5xx = the bridge itself is broken" and treats an auth failure as `status: "error"` (a transport/credential problem is not a *business* rejection in B's mental model — `rejected` is reserved for something the *domain* refused, e.g. what would come back from a chaincode-level ABAC denial).

Both cite the same sentence. Nothing in AD-3 adjudicates it. Worse: because the "Error classification" convention says *"every other error stays opaque"* and `anchor()` in `writepaths.go:175-190` wraps **every** `doAnchor` failure — including a genuine chaincode `ErrUnauthorized`/`ErrInvalidArgument` rejection surfaced as a `gatewayclient.LedgerError` — into the *same* `*PartialFailureError`, there is **no error shape in the actual Hooks/gateway-client surface that would ever legitimately produce `rejected`** once a request passes validation. `rejected` is a dead enum value for anything past `[validate]`, and an ambiguous one before it.

**What's missing.** A decision table on AD-3, e.g.:

| Failure origin | `status` |
| --- | --- |
| `[auth]` stage rejection | `error` (transport/credential fault, not a business outcome) |
| `[validate]` stage rejection | `rejected` (bad input, never reached the domain) |
| `Hooks` untyped error (pre-`anchor()`, e.g. `SaveSection` failure) | `error` (nothing committed; treat as bridge/dependency fault) |
| `*writepaths.PartialFailureError` (from `anchor()`, timeout included per AD-4) | `partial_failure` |

— or equivalent, but *some* explicit table, not a one-line framing sentence two people can read oppositely.

---

## Finding 2 — AD-4 pins *which stage* arms the timeout, but not *where the clock starts relative to stage entry*, and the spine's own partial_failure-equivalence claim depends on getting this right

**The gap.** AD-4 says: "one shared, centrally-configured timeout (`context.WithTimeout`), enforced once in the dispatch filter." That pins the *stage*, but two builders can both write "`context.WithTimeout` in the dispatch filter" and still start the clock at different points:

- **Builder A** puts `ctx, cancel := context.WithTimeout(r.Context(), budget)` as the first statement inside the dispatch stage's own closure — i.e., the clock starts *after* `[auth]` and `[validate]` have already run and consumed some of the wall-clock. Every route gets the *full* configured budget for the actual `Hooks` call.
- **Builder B** reads "one shared... timeout... enforced once" as "one deadline for the whole request" and realizes it as `http.TimeoutHandler(mux, budget, ...)` wrapping the *entire* `ServeMux` — which is still stdlib `net/http` (AD-6-compliant), still `context.WithTimeout` under the hood, and still "enforced once." Now `[auth]` and `[validate]` time is deducted from the same budget the `Hooks` call gets.

Both satisfy AD-4's literal text. Under load (slow auth lookup, large base64 `document` decode in `[validate]`), Builder B's requests reach `Hooks` with measurably less budget than Builder A's — different timeout rates for the identical route, depending only on which builder wrote the bridge.

**A sharper consequence.** AD-4's own justification for collapsing timeout into `partial_failure` is: *"every one of the five Hooks methods calls Store.SaveSection... and returns before anchor() runs, so a timeout mid-anchor() has the identical consequence as the typed error."* That claim is only true if the deadline cannot fire **before** `SaveSection` has actually completed. If the clock starts too early (Builder B's shape, or even Builder A's shape under a tight budget plus slow validation), `ctx` can already be past its deadline when `Hooks.UpdatePersonalData` is invoked, or can expire *while* `SaveSection` itself is still in flight — a state `PartialFailureError`'s own doc comment (`writepaths.go:132-144`) explicitly does *not* cover ("by construction, `anchor()` is only ever reached once the operational-DB write has already succeeded" — a premise that requires `SaveSection` to have already returned, which a mid-`SaveSection` cancellation violates). AD-4 mandates reporting `partial_failure` uniformly on timeout regardless of which of these actually happened, meaning the bridge can report "the DB write already landed" when it may not have — the exact false-safety AD-3 says it exists to prevent, reintroduced by AD-4's own rule.

**What's missing.** Tighten AD-4's Rule to name the exact mechanism, not just the stage: *"the timeout context MUST be created via a fresh `context.WithTimeout(parentCtx, budget)` call as the literal first statement of the dispatch stage's handler body, after `[auth]`/`[validate]` have already returned success — never a `http.TimeoutHandler`/deadline attached earlier in the chain."* Separately, flag (the spine's own Deferred section gestures at this but doesn't resolve it) that AD-4's `timeout ⇒ partial_failure` equivalence is unsound unless dispatch additionally guarantees the budget is never so tight that it can expire before a plausible `SaveSection` round-trip completes — otherwise a distinct `status` (or at minimum a `detail` flag distinguishing "timeout after Hooks call started" from "timeout before/during the operational write") is needed.

---

## Finding 3 — AD-1's pipeline never specifies how stage state (which section, which employeeInternalID, the Hooks result/error) crosses stage boundaries, so map-error and any logging can silently lose or duplicate it

**The gap.** The Design Paradigm section describes 5 conceptual stages and says pipeline.go realizes them as composable `func(http.Handler) http.Handler` middleware, while handlers.go owns "the per-section dispatch target each route feeds into the chain." That sentence is read two different, both-defensible ways:

- **Builder A** treats `[dispatch]` as itself one of pipeline.go's 5 generic middleware stages: it looks up which `Hooks` method to call (via a small table populated from handlers.go), invokes it directly inside the dispatch stage, and stashes `(result []byte, err error, profileSection, employeeInternalID string)` into the request's `context.Context` via an unexported key so `[map-error]` (the next stage in the chain) can read it back out.
- **Builder B** treats handlers.go's 5 functions as the actual terminal `http.Handler`s each route's chain wraps — i.e., "the per-section dispatch target... feeds into the chain" means the Hooks call happens *inside each of the 5 route handlers*, and `[dispatch]` as a pipeline.go stage does nothing but arm the AD-4 timeout and call `next.ServeHTTP`. In this shape, the per-route handler itself must independently write its own result/error somewhere for `[map-error]` to pick up.

Both comply with AD-1's literal Rule ("no route may call a Hooks method directly from its own handler body outside the shared pipeline" — A's handler bodies never call Hooks directly; B's *do*, but B would argue the handler body *is* the dispatch stage of the shared pipeline, not something "outside" it).

**Concrete harm.** Neither AD-1 nor any convention row specifies the hand-off mechanism (context key type, a shared request-scoped struct, a custom `ResponseWriter`, or something else) — nor a logging contract at all. This bites hardest exactly on the path Finding 2 identifies as fragile: a **timeout**-sourced `partial_failure` has no `*PartialFailureError` to extract `employeeInternalID`/`profileSection` from (per `writepaths.go:152-166`, those fields live *only* on that struct, which a bare `context.DeadlineExceeded` bubbling out of a cancelled `ctx` is not). A builder who only ever calls `errors.As(err, &writepaths.PartialFailureError{})` in `[map-error]` to obtain those fields (correct for the typed-error case) silently produces a `partial_failure` response/log with an *empty* employeeInternalID/profileSection on the timeout case — while a builder who separately threads the original request's identifiers through the stage hand-off (whatever mechanism Finding's resolution picks) does not. Both are AD-1/AD-3/AD-4-compliant; one is silently less debuggable, and which one you get depends on which engineer wrote `[map-error]`.

**What's missing.** A Rule (or a new "Stage Contract" convention row) naming: (a) the exact hand-off mechanism between stages (e.g. "a single exported `bridgectx` struct, attached via one `context.WithValue` call in `[validate]` once the envelope is parsed, carrying `EmployeeInternalID`, `ProfileSection`, and later `Result`/`Err`"); (b) a rule that `[map-error]`/`[respond]`/any logging must always source `employeeInternalID`/`profileSection` from that shared struct, never from `errors.As`-unwrapping the error — precisely because the timeout path cannot populate the latter.

---

## Finding 4 — AD-5 fixes cardinality and a name-association, but not an actual URL path convention; two builders would not produce the same route strings

**The gap.** AD-5's Rule: "exactly five operations, one per `ProfileSection` enum value... never fewer, never a combined operation." The Naming convention: "Route/operation names match the five `ProfileSection` enum values verbatim." AD-6 confirms HTTP method (`POST`, stated explicitly in its own Prevents clause: "five simple POST routes") and that Go 1.22+ `ServeMux` method+path patterns are used. Nothing pins: a path prefix/version segment, a flat-vs-parameterized structure, or casing.

**Two compliant builders, different wire contracts.**

- **Builder A** (implements PERSONAL + EMPLOYMENT): `mux.HandleFunc("POST /personal", ...)`, `mux.HandleFunc("POST /employment", ...)` — no prefix, lower-cased.
- **Builder B** (implements EDUCATION + ADDITIONAL + PAYROLL): `mux.HandleFunc("POST /v1/profile-sections/EDUCATION", ...)`, etc. — versioned, prefixed, upper-cased to match the enum string exactly.

Both can claim their route "name" — read as the internal `ProfileSection` string each route dispatches to — "matches the five `ProfileSection` enum values verbatim"; the spine never states whether "verbatim" governs the URL's literal text or merely an internal Go identifier/lookup key. Builder A's lowercase URL arguably even *violates* "verbatim" more than Builder B's, yet A's shape (flat, unprefixed) is just as plausible a reading of "5 operations" as B's. talenta-core (the one real caller, per ADR-0022) needs one, single, agreed set of 5 URL strings to point its Guzzle client at — the spine, as written, does not deliver that even though AD-5 explicitly claims to bind "the bridge's API contract."

**What's missing.** A Rule pinning the literal path template, e.g.: *"exactly `POST /v1/profile-sections/{PERSONAL|EMPLOYMENT|EDUCATION|ADDITIONAL|PAYROLL}` — one static route per section, uppercase, under a `/v1/` prefix; no parameterized `{section}` path variable, so `ServeMux`'s own routing (not app-level string matching) is what enforces AD-5's 'never a combined operation.'"* (or whatever the architect actually intends) — stated once, explicitly, as wire text, not inferred from the Naming row.

---

## Finding 5 — the request envelope never pins `newValue`'s JSON wire-shape, and the two plausible shapes are not equivalent: one can silently corrupt the on-chain digest for numeric PAYROLL fields

**The gap.** The Request envelope convention says `newValue` "mirrors the matching Hooks method's own signature exactly" — i.e., ends up as a Go `[]byte`. It never says whether the *inbound JSON* carries `newValue` as (a) a nested JSON object (`"newValue": {"bankAccountNumber": 1234567890123, ...}`) or (b) a JSON string containing pre-serialized JSON text (`"newValue": "{\"bankAccountNumber\":1234567890123,...}"`, produced by talenta-core doing its own `json_encode` before the HTTP call). This matters because `gatewayclient.ComputeDataHash` (`digestbuilder.go:62-74`) canonicalizes whatever `[]byte` it is handed via RFC 8785 JCS and hashes the *result* — JCS fixes key ordering and whitespace, but it cannot recover precision already lost **upstream of it**.

**Two compliant builders, one silently wrong.**

- **Builder A** treats `newValue` as shape (b): decodes the JSON string field directly into a Go `string`, then `[]byte(that string)` — passing talenta-core's own original bytes straight through to `Hooks`, untouched. Byte-faithful, whatever PHP's `json_encode` chose for number formatting.
- **Builder B** treats `newValue` as shape (a): unmarshals the whole envelope, including `newValue`, into `map[string]interface{}` (or a generic envelope struct) so it can validate individual fields per AD-1's `[validate]` stage, then re-marshals `NewValue` with `encoding/json` to produce the `[]byte` `Hooks` expects.

Go's `encoding/json` decodes untyped JSON numbers into `float64`. For PAYROLL's bank-account-number-shaped fields (large integers, easily > 2^53, the exact-integer boundary of `float64`), Builder B's unmarshal→remarshal round-trip can change the digit string before it ever reaches `ComputeDataHash`/JCS — JCS then faithfully canonicalizes the *already-corrupted* number. The resulting on-chain `DataHash` no longer matches what talenta-core's own record actually contains, and — per this repository's own zero-PII-on-chain design — there is no way to detect this after the fact except a client-side recompute-and-compare, which will simply always fail for that employee/section going forward. Builder A's route for the same section never has this problem. Both builders satisfy the Request envelope convention as written; only one produces a correct digest, and which one you get depends on which of two structurally-identical-looking HTTP handlers you happened to write.

**What's missing.** A Rule on the Request envelope convention pinning the wire shape explicitly, e.g.: *"`newValue` is transmitted as a JSON string containing the section's pre-serialized JSON (produced once, by talenta-core, via a single `json_encode` call at the commit site) — never as a nested JSON object. The bridge's `[validate]` stage MUST NOT unmarshal `newValue`'s contents into any Go value; it is decoded only as a string and passed to `Hooks` as `[]byte(that string)`, byte-for-byte."* If nested-object transmission is instead the intended shape, the Rule must instead mandate `json.RawMessage` capture (never `map[string]interface{}`/`interface{}` decoding) for the `newValue` field specifically, everywhere it is touched.

---

## Summary of required spine changes

| Finding | Existing AD/convention | Fix |
| --- | --- | --- |
| 1 | AD-3 | Add an explicit failure-origin → `status` decision table |
| 2 | AD-4 | Name the exact `context.WithTimeout` call site (first statement of dispatch, post-auth/validate) and flag the soundness gap in the timeout=partial_failure equivalence |
| 3 | AD-1 | Add a Stage Contract: the exact inter-stage hand-off mechanism, and a rule that identifiers are always sourced from it, never from `errors.As` |
| 4 | AD-5 / Naming convention | Pin the literal path template (prefix, casing, static-vs-parameterized) |
| 5 | Request envelope convention | Pin `newValue`'s wire shape (pre-serialized string vs nested object) and, whichever is chosen, mandate byte-faithful handling in `[validate]` |

# Rubric Review — Integration Bridge Architecture Spine

- **Target:** `_bmad-output/planning-artifacts/architecture/architecture-fabric-hris-2026-08-08/ARCHITECTURE-SPINE.md`
- **Lens:** the seven-point "good spine" checklist supplied for this review — divergence coverage,
  AD enforceability, Deferred-list integrity, named-tech accuracy, brownfield ratification,
  structural-dimension completeness, and AD-vs-AD consistency.
- **Method:** every claim below was checked against the live repo (not training-data recall on Go
  modules or the Fabric Gateway SDK). Two claims were verified by actually running code, not just
  reading it — see Findings 1 and 2.
- **Related prior reviews in this same folder:** `review-tech-verify.md` (tech-claim verification)
  and `review-reconcile.md` (spine-vs-sources reconciliation) already cover other angles — the
  AD-6 ServeMux oversell, the AD-2 "no existing precedent" gap, the circuit-breaker naming
  mismatch, the missing single-warm-connection Rule, the multi-tenancy silence, AD-3's overclaim
  about talenta-core's internals, and the missing NFR-9/NFR-1 traceability. This review does not
  repeat those; it adds two new, empirically-verified defects (Findings 1 and 2) that go a level
  deeper than "unprecedented" into "will not build" / "will not enforce as claimed," plus three
  smaller items the checklist's specific framing surfaced.

## Overall verdict

**Not yet fit to hand to independent builders.** The spine's prose reads as authoritative and its
six ADs are well-motivated, but two of them — AD-2 (module layout) and AD-4 (timeout enforcement)
— describe mechanisms that do not actually work against the real code they sit on top of. AD-2's
`replace` directive, exactly as specified, fails to compile (verified by reproducing it). AD-4's
"enforced once in the dispatch filter" timeout does not bound the dominant blocking call in the
path it governs, because that call never receives the context it would need to honor a deadline
(verified by reading `gatewayclient.go` line by line). Both are exactly the kind of "genuinely
load-bearing decision hiding in Deferred" the checklist warns about: AD-4's own Deferred section
gestures at the second problem as something to "verify," when the answer is already sitting in a
file one hop from the spine's own cited sources.

## Checklist walkthrough

1. **Divergence coverage** — Mostly adequate for the request/response contract (AD-3, AD-5,
   Consistency Conventions), but two divergence points a builder will hit are left unfixed: the
   literal URL path scheme (Finding 4) and process shutdown / connection lifecycle (Finding 5).
2. **AD enforceability** — AD-1, AD-3, AD-5, AD-6 are concrete and checkable. AD-2 and AD-4 are
   **not enforceable as written** — see Findings 1 and 2; their Rules assert mechanisms that don't
   hold up against the actual code.
3. **Deferred-list integrity** — The cancellation-semantics bullet is the correct instinct but
   undersells a problem that is already decidable, not merely "needs verification" (Finding 2).
   Everything else in Deferred (timeout duration, transport security, PAYROLL coverage gap,
   deployment topology) is a clean, appropriately-scoped deferral.
4. **Named tech** — Verified independently: `go.work` and all six `fabric-network/tools/*/go.mod`
   files pin `go 1.25.9`; the chaincode module alone is `go 1.21`. The spine's claim is accurate.
   (Same conclusion as `review-tech-verify.md`, re-confirmed here independently.)
5. **Brownfield ratification** — The request envelope, the five `Hooks` method signatures, the
   `PartialFailureError` type and its `SaveSection`-before-`anchor()` invariant, the private
   (unexported) status of `isExpectedRetryableRejection`/`isNotFoundRejection`, and the five
   `ProfileSection` enum values in `chaincode/asset.go` all check out exactly as the spine states.
   The one place ratification breaks down is AD-2's claimed module-layout precedent (Finding 1).
6. **Structural-dimension completeness** — Deployment/environment topology is explicitly named as
   undecided (both in prose right after the Structural Seed and again in Deferred) — this is not a
   silent skip, it satisfies the checklist. Process lifecycle (start/stop, connection cleanup) is a
   narrower structural dimension that *is* silently skipped — see Finding 5.
7. **AD-vs-AD consistency** — AD-1's pipeline order (`auth -> validate -> dispatch -> map-error ->
   respond`) structurally prevents `map-error` from ever running on a request rejected at the
   `auth` or `validate` stage, which is in tension with AD-3's claim that `map-error` is "the only
   place that sets status" combined with the stated design intent that talenta-core "never reads
   the HTTP code" — see Finding 3.

## Findings, tiered by severity

### Finding 1 (Critical) — AD-2's Rule, exactly as written, produces a module that does not compile

**AD-2's Rule** (spine lines 58–61): `integration-bridge/go.mod` gets *one* `replace writepaths =>
../write-path-integration/writepaths` directive, is not added to `write-path-integration/go.work`,
and this is presented as sufficient — matching, per the Rule's own text, "the `fabric-network/
tools/*` precedent." The Structural Seed (line 141) repeats the same single directive.

This does not build, and the reason is visible directly in the sources the spine already read:

- `write-path-integration/writepaths/go.mod` is `module writepaths` / `go 1.25.9` with **zero
  `require` entries** — not even for the three sibling packages `writepaths.go` itself imports
  (`gatewayclient`, `ipfsclient`, `keystore`, lines 31–33 of that file). Today, those imports
  resolve *only* because `write-path-integration/go.work`'s `use` block lists all four modules
  together — Go workspace mode is doing the resolution, not each module's own `go.mod`.
- `integration-bridge/` is deliberately **not** added to that `go.work` (AD-2's own Rule). Outside
  the workspace, a bare `replace writepaths => ...` gives Go no way to find `gatewayclient`,
  `ipfsclient`, or `keystore` — they aren't published modules, and `writepaths`'s own `go.mod`
  doesn't declare them as dependencies.
- I reproduced this directly rather than asserting it: a minimal two-module repro (a `writepaths`-
  shaped module with an unresolved local import, `go.work`-only resolution, and an `integration-
  bridge`-shaped module with exactly AD-2's single `replace`) fails with `go build` producing
  `package gatewayclient is not in std`. Adding `require`+`replace` for **all four** sibling
  modules (matching `go.work`'s own `use` list exactly) fixes it. Also note `main.go` will need to
  construct concrete `keystore.EmployeeKeyStore`/`SaltStore`/`DocumentKeyStore` implementations
  directly (the `Hooks` struct fields, `writepaths.go` lines 98–101) — so `keystore` needs a direct
  import from `integration-bridge` too, not just a transitive one through `writepaths`.

This also means AD-2's precedent claim is not merely unprecedented (as `review-tech-verify.md`
already found — no `fabric-network/tools/*` module has any local cross-module dependency, via
`replace` or otherwise) but that the specific mechanism prescribed doesn't work on its own terms.
**Fix:** state plainly that `integration-bridge/go.mod` needs `require`+`replace` pairs for all
four `write-path-integration/*` modules (`writepaths`, `gatewayclient`, `ipfsclient`, `keystore`),
not one, and drop or correct the "matching the precedent" framing per Finding 1's evidence above.

### Finding 2 (Critical) — AD-4's timeout is not actually enforced against the one call it exists to bound

**AD-4's Rule** (spine lines 79–84): "one shared, centrally-configured timeout
(`context.WithTimeout`), enforced once in the dispatch filter, applied identically to every
route... a timeout mid-`anchor()` has the identical consequence as the typed error."

Read `write-path-integration/gateway-client/gatewayclient.go`:

- `SubmitRecordProfileSection` (line 114) takes `ctx context.Context` and threads it into the
  *seed* read (`EvaluateGetProfileSectionRecord`, line 119) and into the re-reads inside its
  retry loop (line 136) — both go through `EvaluateWithContext(ctx, ...)` (line 164), which does
  honor the caller's context.
- But the actual submit call — the one that does endorsement, ordering, and commit, i.e. the
  slow part AD-4 exists to bound — is `g.contract.SubmitTransaction("RecordProfileSection",
  args...)` (line 126). **This does not take `ctx` at all.** It is governed instead by the fixed,
  connect-time timeouts set once in `NewGatewayClient` (lines 77–80): `WithEndorseTimeout(15s)`,
  `WithSubmitTimeout(5s)`, `WithCommitStatusTimeout(1 min)` — none of which derive from, or can be
  shortened by, the bridge's own per-request `context.WithTimeout`.

Consequence: cancelling the dispatch filter's context does **nothing** to interrupt an in-flight
`SubmitTransaction` call. If the dispatch filter is implemented the way AD-4's Rule text plainly
reads — a synchronous wrap around the `Hooks` method call — the HTTP handler goroutine keeps
blocking until the SDK's own internal timeouts elapse (up to ~20s per attempt, times up to
`MaxSubmitRetries+1` = 4 attempts on the retryable path, i.e. potentially over a minute), not the
"shared budget" the AD implies. To actually cut the response off at the configured budget, the
dispatch filter would have to run the `Hooks` call in a separate goroutine and race it against
`ctx.Done()` — a materially different, and materially riskier, implementation (the orphaned
goroutine keeps running against the one shared, retained `GatewayClient` connection after the
HTTP response has already gone out, which is exactly the double-submission race the spine's own
Deferred bullet worries about one paragraph later).

The spine's Deferred section (lines 159–165) already flags a version of this — "does that actually
abort the in-flight Fabric submission... verify... before trusting AD-4 in production" — but frames
it as an open verification task. It isn't one: the answer is already decidable by reading the file
the spine's own source list is one hop from (`writepaths.go` → `gatewayclient.SubmitRecordProfileSection`),
and the answer is "no, it does not abort it." That makes this a **load-bearing decision** the
checklist explicitly warns about hiding in Deferred, not a clean deferral: whether the dispatch
filter races a goroutine against the timeout (early response, risk of a detached in-flight submit)
or blocks synchronously (late response, AD-4's stated budget is fiction) is a fork two independent
builders will resolve differently, and the two outcomes have different failure modes for
talenta-core's synchronous caller.

**Fix:** either (a) change AD-4's Rule to state explicitly which of the two shapes is required —
most likely goroutine+`select` on `ctx.Done()`, with an explicit note that a timed-out submit keeps
running detached and must be reconciled later (the spine's own suggestion, `GetProfileHistory`) —
or (b) scope AD-4 down to only what it can currently guarantee (bounding the pre-submission local
work) and openly state that submission-phase latency is currently governed by the SDK's own fixed,
connect-time timeouts, not by the bridge's request-level budget, until `gatewayclient` itself is
changed to thread `ctx` into `SubmitTransaction`.

### Finding 3 (Medium) — AD-1's pipeline order and AD-3's "only the error-mapping filter sets status" are in tension for auth/validate rejections

AD-1's pipeline (lines 30–36) is `auth -> validate -> dispatch -> map-error -> respond`. AD-3's
Rule (lines 68–72) states the error-mapping filter is "the **only** place that sets `status`,"
and the Consistency Conventions table states talenta-core's client "reads `status`, never the
HTTP code" — a blanket claim about every response. But a request rejected at `auth` or `validate`
never reaches `dispatch` or `map-error` in this linear chain — so either:

- those short-circuited responses carry no `status` field at all, which means talenta-core must
  still special-case those two failure modes by HTTP code after all, quietly breaking the very
  guarantee AD-3 exists to provide; or
- `auth`/`validate` must themselves set `status` directly, which contradicts AD-3's "only" claim.

Neither resolution is chosen. This is exactly the kind of ambiguity the checklist's "no AD
contradicts another" and "enforceable, not vague" criteria are meant to catch — two builders could
reasonably pick either horn, and only one is consistent with the stated client contract.

**Fix:** either state that `auth`/`validate` failures are the sole exception (no `status` field;
HTTP code carries the outcome for exactly these two failure classes, and talenta-core is expected
to check the code first, only falling through to `status` on 2xx/5xx) or add auth/validate outcomes
into the same status-setting responsibility explicitly, and update AD-3's "only" wording to match.

### Finding 4 (Low) — Route/URL path scheme is not actually fixed, only the operation *names* are

The Naming convention (line 128) fixes that "route/operation names match the five `ProfileSection`
enum values verbatim," and AD-5 fixes the *count* (five, 1:1 with `Hooks` methods). Neither fixes
the concrete URL pattern two builders would otherwise diverge on independently — e.g. `POST
/PERSONAL` vs. `POST /v1/profile-sections/PERSONAL` vs. `POST /record?section=PERSONAL`. Given
AD-6 already commits to Go 1.22+ `ServeMux`'s method+literal-path registration specifically (per
`review-tech-verify.md`'s finding that no path wildcards are actually used here), one concrete
example route in the Structural Seed (e.g. `mux.HandleFunc("POST /PERSONAL", ...)`) would close
this cheaply.

### Finding 5 (Low) — Process shutdown and the retained `GatewayClient` connection's lifecycle are unaddressed

`gatewayclient.go`'s own `Close()` doc comment (lines 93–99) is explicit: "Call once, at
host-process shutdown — never per transaction." The Structural Seed's description of `main.go`
("config, `Hooks` construction, route registration, wiring") never mentions calling `Close()` on
shutdown, and no AD addresses signal handling (`SIGTERM`/`SIGINT`) or graceful drain. This sits
squarely inside what this spine claims to own ("this service's build and dependency setup" /
"module boundaries") rather than the explicitly-deferred deployment/environment topology — it's a
narrower, in-process structural dimension that was silently skipped rather than named as open.
Two builders could reasonably diverge here (one wires signal handling and a clean `Close()`, one
doesn't), with a real resource-leak consequence given there's exactly one retained connection per
process by design.

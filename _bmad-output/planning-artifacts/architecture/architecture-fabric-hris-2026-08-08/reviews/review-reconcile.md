# Reconciliation Review — Integration Bridge Architecture Spine vs. Its Load-Bearing Sources

- **Target:** `_bmad-output/planning-artifacts/architecture/architecture-fabric-hris-2026-08-08/ARCHITECTURE-SPINE.md`
- **Lens:** independent re-read of the four sources the spine's own frontmatter lists (`sources:`), checking whether every real constraint/caveat/nuance each source states actually shows up in an AD's Binds/Prevents/Rule, a Convention, or Deferred — and whether the spine asserts anything those sources don't actually support.
- **Sources re-read directly:** `agent-suite/05-adr/ADR-0022-integration-bridge-gateway-client-host.md`, `agent-suite/05-adr/ADR-0014-in-band-recording.md`, `agent-suite/context/REAL-INTEGRATION-TRIGGER-FLOW.md`, `write-path-integration/writepaths/writepaths.go` (plus a quick grep of `write-path-integration/gateway-client/*.go` to check two claims that reach past the four sources into code the spine names by symbol).
- **Date:** 2026-08-08

## Overall verdict

Mostly faithful, but not fully. The spine gets the headline facts right — the 4-commit-sites-vs-5-sections
distinction, the actor-identity gap at 3-of-4 sites, the `BaseEmsService` calling convention, the
`PartialFailureError`/`SaveSection`-before-`anchor()` ordering, the disclosed PAYROLL coverage gap, and the
still-open transport-security question all land correctly and are traceable to the right source. But two
of ADR-0022's/ADR-0014's most consequential *open questions* — the ones both ADRs go out of their way to
flag as explicitly unresolved, with a named follow-up action — are missing from the spine's Deferred list
entirely, and one AD's own header claims to cover a mechanism ("circuit-breaker") its Rule text never
actually defines. There's also a real structural fact sitting in the one Go file the spine was told to
read closely (`Hooks.TenantID` is fixed per instance) that the spine's HTTP contract silently ignores.

## Findings

### Finding 1 (High) — AD-4's own title promises a circuit-breaker; its Rule delivers only a timeout, and the actual open circuit-breaker/fallback question is missing from Deferred

AD-4 is titled **"Timeout and circuit-breaker: one shared budget, timeout maps to partial_failure."** Read
the Rule text in full: it specifies one `context.WithTimeout` value, enforced once, mapping a timeout to
`status=partial_failure`. Nowhere does it define failure-threshold tripping, an open/half-open state, or
any other actual circuit-breaker semantic. The header oversells what the AD contains.

That would be a naming nit on its own, except the *substance* it's standing in for is a real, twice-named
open question in the sources this spine is supposed to reconcile against:

- ADR-0014, Consequences: "...this ADR does **not** yet specify a circuit-breaker/local-queue fallback for
  that case; it is flagged as an open design question for whoever fixes the Gateway-client host."
- ADR-0022, Consequences: "...this ADR does **not** specify a circuit-breaker or local-queue fallback
  either," plus an explicit Follow-up: "Append a risk-register row for 'write-path availability now coupled
  to bridge/ledger availability' to `10-risk/risk-register.md`."

Both ADRs treat "should this bridge have a circuit-breaker/fallback for a Fabric or bridge outage, so a
ledger outage doesn't become a full HRIS write-path outage" as a live, unresolved, explicitly-flagged
question — not a settled non-requirement. The spine's Deferred section has five bullets (exact timeout
duration, transport security, cancellation semantics, the 3 uncovered PAYROLL surfaces, deployment
topology) and **none of them is this question**. A reader who only reads AD-4's header could reasonably
believe the outage-fallback question is handled; a reader who reads the Rule text correctly concludes it
isn't, but then finds no Deferred item pointing them back to the two ADRs that already flagged it. This is
exactly the kind of quiet requirement the AD-1..AD-6 structure flattened: real, load-bearing, and dropped.

**Fix:** rename AD-4 to drop "circuit-breaker" (it's a timeout policy, not a breaker), and add a Deferred
bullet carrying forward ADR-0014's/ADR-0022's own open item verbatim — including the named follow-up
(risk-register row) — so it isn't silently lost between two ADRs and a spine that doesn't cite either on
this point.

### Finding 2 (High) — The "one warm, reused Gateway connection" property that ADR-0022 uses to justify the whole design is never stated as a Rule anywhere in AD-1..AD-6

This is the actual reason a standalone bridge process exists at all, not incidental color. ADR-0022's own
Alternatives table lists as a *Pro* of the chosen option: "can hold one warm Gateway connection reused
across every write, restoring the connection-reuse property `ADR-0006` originally required." And Option D
("no bridge process; each write path opens its own connection at call time") is **rejected even
hypothetically** specifically because "PHP-FPM's per-request process model cannot hold a long-lived
connection reused across events... requires a real persistent process to hold it, regardless of which
language ends up hosting the client."

The spine's Structural Seed implies single construction — `main.go` owns "config, `Hooks` construction" —
but that's a filename comment, not a Rule. Nowhere in AD-1 through AD-6 does the spine state, as an
explicit Prevents/Rule, that the `*gatewayclient.GatewayClient` (and the `Hooks` that wraps it) must be
constructed exactly once at process startup and reused across every request — never per-request, never
per-handler. Given the spine's own stated purpose is "fix only the invariants that would let two
independent builders of this one service diverge incompatibly," this is precisely the kind of thing two
builders could diverge on: one holds the connection in `main.go` and threads it through; another,
following AD-1's per-route dispatch language literally, constructs a fresh client inside `handlers.go`'s
per-section dispatch function. Both would satisfy every AD as currently written. Only one preserves the
property ADR-0022 exists to deliver.

**Fix:** add this as its own Rule (or fold into AD-2), e.g.: "the `*gatewayclient.GatewayClient` is
constructed exactly once, in `main.go`, and passed into every dispatch call by reference — no route,
handler, or pipeline stage may construct its own."

### Finding 3 (Medium) — Multi-tenancy at the HTTP boundary is unaddressed, despite a direct structural fact in the required source file and a header the spine's own cited precedent uses

`write-path-integration/writepaths/writepaths.go`'s `Hooks` struct has a `TenantID string` field with this
doc comment: *"tenantID is fixed per Hooks instance — one HRIS process serves one tenant's write paths at a
time in this design (channel-per-tenant, ADR-0013)."* `doAnchor` reads `tenantID := h.TenantID` and passes
it straight into `SubmitRecordProfileSection` — it is never a per-call parameter on any of the five `Hooks`
methods.

The spine's Request-envelope Convention lists exactly four fields — `employeeInternalID`, `userID`,
`newValue`, `document` — and states this "mirrors the matching `Hooks` method's own signature exactly."
That's a true statement about the `Hooks` method signatures, but it silently inherits their tenant-blind
shape into the HTTP contract without ever asking the question the source file's own comment raises: since
`Hooks.TenantID` is fixed per instance, does the bridge run **one process per tenant** (so tenant
identification is implicit in which deployment/URL you call), or does one bridge process need to multiplex
several tenants (which the current `Hooks` shape cannot do without restructuring)?

This connects to a second dropped detail: `REAL-INTEGRATION-TRIGGER-FLOW.md` §3 and `ADR-0022`'s own Context
section both cite `BaseEmsService::companyLevelHeaders()` (lines 98-105) as the calling-convention
precedent, and the trigger-flow doc is explicit that this method sets **`X-Api-Key` *plus* `X-Scope`/
`X-Company-ID`**. The spine's Convention table cites the same precedent and the same line numbers but
keeps only `X-Api-Key`, dropping the company/tenant-scoping headers from the very source it names. Given
`Hooks.TenantID`'s fixed-per-instance shape, `X-Company-ID` (or an equivalent) is exactly the kind of header
that would resolve the open question above — and it's already sitting in the cited precedent.

**Fix:** either (a) state explicitly that one bridge deployment serves exactly one tenant (matching
channel-per-tenant, ADR-0013) and fold that into AD-2's module/deployment layout and the "deployment target
and environment topology" Deferred item, or (b) if a single process must multiplex tenants, add a tenant
header/route-prefix to the Convention table and flag that `Hooks` construction would need to become
per-tenant, not process-wide. Currently neither is decided nor named as open.

### Finding 4 (Medium) — AD-3's "talenta-core's retry/idempotency logic reads status" is stated as settled fact; the source it's drawn from marks the underlying policy as an unratified assumption

AD-3's Rule states, flatly: *"talenta-core's retry/idempotency logic reads `status`, never the HTTP code."*
This is written as a description of an existing (or definitely-forthcoming) capability. But the only source
that discusses what happens after a `PartialFailureError`/`partial_failure` is `writepaths.go`'s own doc
comment on `PartialFailureError`, and it is explicit that this is **not** a ratified design:

> "[ASSUMPTION — this is a provisional, clearly-labeled policy call filling ADR-0014's own Consequences
> section, which explicitly left the Fabric-outage fallback/retry policy for the write path OPEN as future
> work (grounding gap G-10 / SEC threat T6b); it is not a ratified policy and a real follow-on ADR from
> `architect` may supersede it.]"

and, on the type itself: "to flag the record for a later re-anchor pass — deciding what that pass looks
like is out of this item's scope." Nothing in any of the four sources establishes that talenta-core
*currently has*, or has committed to building, retry/idempotency logic keyed on a `status` field. The
spine's own Deferred section is actually more careful elsewhere on this exact point (the cancellation-
semantics bullet: a raced timeout "would need reconciling... not just reporting" — correctly hedged as
unresolved), which makes AD-3's flat present-tense claim inconsistent with the spine's own better-hedged
treatment two sections later.

**Fix:** reword AD-3 as a constraint on the contract, not an assertion about talenta-core's internals —
e.g., "the body's `status` field is the only channel through which any future talenta-core retry/idempotency
logic should observe outcome; the bridge does not assume such logic already exists" — and note the
retry-after-partial-failure policy itself is still open per `writepaths.go`'s own ASSUMPTION tag.

### Finding 5 (Low) — Named performance budgets (NFR-9, NFR-1) that both ADRs tie directly to this exact decision never appear in the spine, even in Deferred

ADR-0014's Consequences: *"the write-path latency budget added by anchoring (**NFR-9, target < 500 ms**
added to the write path) becomes directly attributable to one endorse-and-commit round trip per section
write, which is measurable once the Gateway-client host is fixed"* — and its Follow-ups: *"**NFR-1** (< 3 s
@ 500 TPS write) and **NFR-9** (< 500 ms added latency) become measurable only once that host is fixed."*
ADR-0022's Consequences repeats the NFR-9 figure verbatim as a benefit of holding a warm connection (see
Finding 2).

The spine's only nod to this is a Deferred bullet: "Exact timeout duration for AD-4's shared budget — needs
tuning against `QA-4`'s real measured Fabric latencies... not picked arbitrarily." That points at the right
data source but never names the actual target (NFR-9's <500ms, or NFR-1's <3s@500TPS) that the timeout
should be judged against, nor cites the requirement IDs. A future implementer tuning AD-4's timeout has no
traceable link back to the number two ADRs already committed to.

**Fix:** name NFR-9 and NFR-1 explicitly in the Deferred bullet so the eventual timeout choice is
checkable against a ratified target, not just "real measured latencies" in the abstract.

### Finding 6 (Low, note only) — The spine faithfully reproduces a miscount already present in ADR-0022, without flagging it

ADR-0022's own "Numeric constraints" section states: *"PAYROLL write surfaces NOT reached by this bridge as
scoped: **3** (`MyInfoController::actionEditPayrollInfo()`, bulk import, a cron job, and a third-party
integration importer)"* — four items named, labeled "3." (`REAL-INTEGRATION-TRIGGER-FLOW.md` §4 has the
same slipperiness: "at least two more independent write surfaces," then names what reads as four.) The
spine's Deferred bullet copies this list verbatim, including the four named items, without a count — which
avoids repeating the wrong number, but also doesn't flag that the source ADR's own "3" doesn't match its
own list. Not the spine's error to fix, but worth a note back to whoever owns ADR-0022, since the spine is
the one artifact positioned to notice it during reconciliation.

### Finding 7 (Low) — AD-4's "timeout mid-`anchor()` has the identical consequence... ledger state uncertain" slightly overstates what's true for the pre-submission portion of `anchor()`'s own work

`doAnchor` (the body `anchor()` wraps) does five things in sequence before it ever calls
`Gateway.SubmitRecordProfileSection`: fetch/derive `employeeKey_i`, compute `EmployeeID`, generate a salt,
compute `DataHash`/`UpdatedBy`, and — if a document is present — encrypt and pin it to IPFS. All of that is
local/off-chain work; the ledger is never contacted until the final call. AD-4's Rule says a timeout
"mid-`anchor()`" has "the identical consequence as the typed error: the DB write already landed, ledger
state uncertain." For a timeout that fires during any of those first steps, the ledger wasn't contacted at
all — "uncertain" isn't quite accurate; "never submitted" is. Operationally this doesn't change the
recommended handling (the bridge/caller can't distinguish which sub-step timed out, so treating it
uniformly as `partial_failure` is still the safe default) — but the stated rationale is broader than the
code actually supports, and a careful reader checking the claim against `writepaths.go` (as this task asked)
will notice the mismatch between "ledger state uncertain" and "ledger possibly never contacted."

## Claims independently verified as accurate (not just plausible)

- `SubmitRecordProfileSection`'s signature (`write-path-integration/gateway-client/gatewayclient.go:114`)
  matches the spine's implicit `([]byte, error)` → `recordID` mapping.
- `isExpectedRetryableRejection` (`gatewayclient.go:183`) and `isNotFoundRejection` (`verify.go:99`) both
  exist exactly as named in the spine's Error-classification convention.
- Every one of the five `Hooks` methods (`UpdatePersonalData`, `ApproveEmploymentTransfer`,
  `RecordEducationHistory`, `ApproveFamilyDataChange`, `UpdatePayrollBankAccount`) calls
  `Store.SaveSection` first, returns early on its error, and only then calls `anchor()` — AD-4's core
  factual claim about this file is correct.
- The actor-identity convention ("read directly by talenta-core at the call site, per the actor-identity
  gap at 3 of 4 real commit sites; never resolved by the bridge") accurately reflects both ADR-0022's
  Decision text and `REAL-INTEGRATION-TRIGGER-FLOW.md` §2.
- The 4-commit-sites-vs-5-sections fact and AD-5's "never fewer, never a combined operation" rule are
  correctly grounded in ADR-0022's Decision and `REAL-INTEGRATION-TRIGGER-FLOW.md` §1.
- The Deferred item on transport security (mTLS vs. `BaseEmsService`'s TLS+API-key-only precedent needing
  `security-architect` sign-off) accurately mirrors ADR-0022's own Follow-ups almost verbatim.
- The Deferred item on cancellation semantics is a reasonable, appropriately-hedged addition consistent
  with the sources' general posture of naming gaps rather than silently resolving them — not asserted by
  any source directly, but not contradicted either.

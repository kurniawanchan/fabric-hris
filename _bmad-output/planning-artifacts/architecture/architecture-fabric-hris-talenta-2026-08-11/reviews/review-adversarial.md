# Adversarial Review — ARCHITECTURE-SPINE.md (Talenta HRIS × Fabric Write-Path Integration)

**Method:** construct two independent implementers (Engineer T = builds `talenta-core`'s
`AnchorTriggerJob` + `MyInfoController` call sites; Engineer B = builds `integration-bridge` +
`writepaths` changes), each reading only the spine and each other's code through the documented
interface (the HTTP contract + Consistency Conventions table). Each obeys every AD to the letter.
Find pairs of choices that are individually compliant but jointly incompatible.

Reviewed 2026-08-11 against `ARCHITECTURE-SPINE.md` as committed.

---

## H1 — AD-6 dedup: both engineers can legally assume the OTHER side owns dedup, yielding NO dedup

**Severity: Critical.**

AD-6's rule only obligates Engineer T to *generate and forward* a stable `correlationID` across
retries. It explicitly punts *enforcement* to Deferred, and even suggests the bridge "does not
need to persist dedup state, since it has no store of its own beyond the operational store
`writepaths` already writes to" — which reads as a hint that dedup already happens somewhere near
`writepaths`' `OperationalStore`.

Walk it through:

- Engineer T reads AD-1 ("no layer above the bridge ever retries... retry authority lives in
  exactly one place — the new Talenta-side job") and AD-6's phrase "the bridge itself does not
  need to persist dedup state." Reasonable reading: *my job is to retry and pass the ID; the
  bridge/writepaths layer is responsible for recognizing a repeat ID and no-op'ing it.* T ships
  `AnchorTriggerJob` with retry/backoff and stops there — no local "have I already succeeded"
  check beyond what the HTTP response tells it.
- Engineer B reads AD-2 ("`integration-bridge` is the only Fabric-facing write component") and
  AD-1 ("bridge... stateless per request, with no queue, retry loop, or dead-letter of its own")
  and concludes the bridge is deliberately *not* the dedup authority — statelessness is called out
  as an invariant, and "dedup mechanism... left open" in Deferred is explicit. B implements
  `correlationID` as pass-through metadata only (stored alongside the anchor for audit, per FR-9),
  with no existence check before submit, reasoning that Deferred items are explicitly *not*
  designed here and it would be presumptuous to build enforcement the spine didn't ask for.
- Result: T retries on a timeout where the first `SubmitTransaction` actually landed on-chain
  (a genuine possibility — see H4). B's route re-runs `writepaths.Hooks` → `gatewayclient` →
  chaincode with the same `correlationID` but does nothing with it except log/store it. The
  chaincode's own key is `(employee, profileSection, recordIdentity)` (AD-3) — a **different** key
  than `correlationID` — so even a chaincode-side existence check (one of the three Deferred
  options) would need `correlationID` to be part of that composite key or the record body, which
  the spine never specifies. Two independently-compliant, fully-shipped components, zero dedup
  enforced anywhere. Both engineers can point to spine text justifying non-ownership.

**Close with:** promote AD-6 from "mechanism deferred" to "ownership ratified, mechanism deferred."
At minimum name *one* of {bridge, chaincode, Talenta table} as the enforcing party in an AD (not
Deferred), even if the storage detail is designed later. Leaving *ownership* itself open — not just
mechanism — is what produces the double-negative outcome above.

---

## H2 — AD-3 `recordIdentity`: no canonical wire representation, only a description in prose

**Severity: High.**

AD-3 says `recordIdentity` is `"self"` for singleton domains, a family member's `reference_id` for
Family, and `custom_field_id` for Additional Info — but never states the *type* or *string
encoding* to put on the wire. `reference_id` and `custom_field_id` in Talenta's own DB are typically
integers or UUIDs; nothing in the spine says whether the JSON field carries a bare value or a
disambiguated form.

- Engineer T, writing the job that builds the POST body, might reasonably serialize
  `recordIdentity` as the raw value from the webhook payload: `"482"` for a family member,
  `"cf_1029"` for a custom field (if that's the DB column's own format) — i.e., "just forward what
  the diff gave me," consistent with AD-5's mandate to forward caller-supplied fields unmodified.
- Engineer B, writing `writepaths`' new Additional-Info method and the Family call-site change,
  independently reads AD-3's prose ("Family... `recordIdentity` is that family member's
  `reference_id`") and, to keep the composite key legible/collision-proof across the two domains
  that both currently key under `profileSection=PERSONAL` (Family) or a domain with numeric IDs
  that could theoretically collide with another domain's numeric IDs, prefixes it defensively:
  `"family:482"`, `"customfield:1029"`. This is a completely reasonable, arguably *more correct*
  implementation given AD-4's own worry about domains "colliding... under the same
  `profileSection`" — B is applying that same worry one level down, at the `recordIdentity` level.
- Now the value T puts on the wire (`"482"`) is not the value B's hash-chain composite key
  actually stores/looks up (`"family:482"`) if B does any transformation — or, if B does NOT
  transform and stores exactly what's received, then the *next* write for the same family member,
  triggered by a differently-coded caller (e.g. a backfill script, or a future second Talenta job)
  that happens to send `"family:482"` instead of `"482"`, silently opens a second, parallel
  hash-chain for the same family member. AD-3 constrains the *conceptual* key; it does not
  constrain the *canonical string form*, so two compliant writers can desync the actual on-chain
  chain without either one violating a stated rule.

**Close with:** an AD (or a tightened AD-3) pinning the exact wire type and string form of
`recordIdentity` per domain — e.g. "the JSON field `recordIdentity` is always a bare string, equal
to `strconv.Itoa(reference_id)` / `strconv.Itoa(custom_field_id)` with no prefix, no domain tag,
and the bridge does not alter it," plus a stated invariant that `writepaths` never transforms it
before it becomes part of the composite key.

---

## H3 — AD-5 operation-type field: name/casing/vocabulary not specified anywhere

**Severity: Medium-High.**

AD-5 says operation type is "passed as an explicit field in the POST body" and "forwarded
unmodified." The Consistency Conventions table lists the body shape as
`{employeeInternalID, userID, newValue, document, correlationID, operationType}` — so the *field
name* `operationType` is actually fixed there, which is good. But nothing fixes:

- The **value vocabulary** — is it `"CREATE"/"UPDATE"/"DELETE"`, `"create"/"update"/"delete"`,
  or Talenta's own internal action-log vocabulary (which may use different words, e.g.
  `"insert"`/`"edit"`/`"remove"`, or numeric action codes)?
- Whether `validate.go` (called out in AD-5 as needing "no change" to its `newValue` object check)
  validates `operationType`'s value set at all, or accepts anything and forwards it verbatim into
  chaincode metadata — the spine explicitly says the bridge forwards it "unmodified," which reads
  as *bridge does not validate/normalize it*.

Concretely: Engineer T pulls the operation type from Talenta's Yii2 model-event constants
(`ActiveRecord::EVENT_AFTER_INSERT` etc. → T's own job maps these to strings T invents, e.g.
`"created"`), because AD-5 says "the new Talenta-side job determines operation type from its own
webhook payload/diff" — T owns the *determination*, so T also owns the string T decides to emit,
absent a fixed enum. Engineer B, writing `writepaths`'/chaincode consumers of that metadata field
(e.g. anything downstream that branches on it for the DELETE-carries-`newValue` special case in
AD-5 itself), assumes the chaincode-side `ProfileSection`-adjacent constants convention and expects
upper-case `"CREATE"/"UPDATE"/"DELETE"` to match existing enum-casing conventions elsewhere in the
codebase (`PERSONAL, EMPLOYMENT`, all upper-case per the Consistency Conventions table). B's code
that special-cases DELETE (per AD-5's "DELETE anchors carry `newValue` populated...") does
`if operationType == "DELETE"` and never matches T's `"created"/"updated"/"deleted"`. Both are
individually spine-compliant (spine never gave the vocabulary); DELETE's own special handling is
the one place this silently produces a behavior bug (wrong `newValue` shape assumed, or the
special case never triggers).

**Close with:** a fixed enum for `operationType`'s three values in the Consistency Conventions
table or a new AD, analogous to the existing fixed `profileSection` enum row.

---

## H4 — AD-1 "single retry authority" doesn't cover partial failure *inside* the bridge, and the spine's own idempotency story (AD-6) doesn't hold across it

**Severity: Critical.**

Walk the concrete sequence the prompt asks for:

1. Talenta job POSTs to `integration-bridge`.
2. Bridge's pipeline: validate → auth → **writepaths.Hooks.anchor()** → this internally does (a)
   whatever the `OperationalStore` write is (the "operational store `writepaths` already writes
   to," per AD-6's own text) and (b) `gatewayclient.SubmitRecordProfileSection` → Fabric.
3. Suppose (a) succeeds — the operational-store row is written — but (b) times out client-side
   *after* the orderer/peers actually committed the transaction (a classic Fabric
   ambiguous-timeout: the gRPC call to the peer/orderer exceeds the bridge's timeout, but the
   transaction still gets committed moments later). The bridge, per AD-1, can only return "a
   retryable failure" to its caller (it has no basis to know the ledger write actually landed) —
   this is explicitly within the bridge's *stated* contract ("return success, a retryable failure,
   or a non-retryable business rejection").
4. Talenta's `AnchorTriggerJob`, being the sole retry authority (AD-1), retries the entire POST.
5. The retried POST hits `writepaths.Hooks.anchor()` again. Per the walk-through in H1, nothing in
   the spine requires a pre-submit existence check keyed on `correlationID` (dedup mechanism is
   Deferred) — so step (b) resubmits to Fabric. On-chain, this is not idempotent: `RecordProfileSection`
   presumably appends to a hash chain keyed on `(employee, profileSection, recordIdentity)`
   (AD-3), not on `correlationID` — a second submit for the same logical write becomes a **second,
   spurious hash-chain entry**, indistinguishable on-chain from a legitimate second edit, unless
   the chaincode itself has content/version guards the spine never mentions.
6. Separately, step (a) — the operational-store write — also re-runs on retry. Whether that's
   idempotent depends entirely on whether `OperationalStore.Write(...)` is an upsert keyed on
   something stable or an append; the spine never says, and Engineer B could implement it either
   way while remaining compliant, since "operational store" is mentioned only in passing (AD-6)
   and never given its own AD.

So: AD-1 correctly centralizes retry *triggering*, but says nothing about retry *idempotency* at
the point where the bridge's own two side effects (operational-store write, ledger submit) can
individually succeed while the overall bridge call is reported as failed/retryable. The spine's
claim structure implies AD-1 + AD-6 together give you exactly-once; concretely traced, they only
give you at-least-once triggering with no enforced dedup (H1) landing on a non-idempotent target
(chaincode append + possibly-append-only operational store) — i.e. duplicate anchors are not just
possible, they're the *expected* outcome of the documented ambiguous-timeout case, and no AD
assigns anyone the job of preventing it.

**Close with:** either (a) an AD establishing the bridge's write to `gatewayclient` +
`OperationalStore` as a single idempotent unit keyed on `correlationID` (turning H1's Deferred item
into a ratified requirement, at least for ownership), or (b) an AD requiring `RecordProfileSection`
itself to reject/no-op a resubmission carrying a `correlationID` (or content hash) it has already
committed — i.e. push the idempotency guarantee to the one component every retry path passes
through unconditionally, the chaincode, rather than relying on two independently-built layers
above it to agree on who checks first.

---

## H5 — AD-4's "either a renamed/repurposed method or a new one; `writepaths` owns which" leaves the route/method contract unfixed across the boundary

**Severity: Medium.**

AD-4 explicitly defers *which* `writepaths.Hooks` method name backs Family-under-`PERSONAL` to
whoever implements `writepaths` — reasonable, since both call sites end at `h.anchor(...)` with
"the same signature regardless." But the Structural Seed shows `cmd/integrationbridge/main.go` and
`write-path-integration/writepaths/writepaths.go` as two files in the *same* repo family, implying
one engineer touches both. The prompt's premise — Engineer T builds the Talenta job, Engineer B
builds bridge+writepaths — means this particular ambiguity is contained inside Engineer B alone,
so it is not actually a cross-engineer hole. Noted for completeness but not counted as a
closable-by-new-AD finding; the existing "writepaths owns which" is sufficient because both ends
of that decision sit with one implementer.

---

## H6 — `employeeInternalID` computed by *which* side, for domains routed through the repurposed `ADDITIONAL`/`PERSONAL` paths?

**Severity: Low-Medium.**

The Consistency Conventions table says `employeeInternalID` "is already computed via
`keystore.EmployeeKeyStore` + `gatewayclient.ComputeEmployeeID`... the new write trigger reuses
this, never invents a second identity scheme." This assigns computation responsibility only in
prose, not in the request-shape table — it's unclear from the spine whether Engineer T's PHP job
calls out to some existing service to *get* `employeeInternalID` before POSTing (i.e. Talenta
computes it) or whether it just forwards a plaintext `userID`/employee key and the bridge computes
`employeeInternalID` internally via the same `keystore`/`gatewayclient` calls. The request-body
convention lists both `employeeInternalID` *and* `userID` as separate fields, which is consistent
with either engineer owning the derivation — Engineer T could send a raw `userID` and expect B to
derive `employeeInternalID`, while Engineer B could build the bridge assuming T already resolved
and is sending the pseudonymized ID directly (since read-path already does this client-side,
per the same sentence). If both assume the other computes it, the field is either double-derived
(bridge re-derives from a value that's already the derived ID, producing a nonsense/[email
protected]"wrong ID"-shaped bug) or never derived (bridge receives a raw ID and anchors under the
wrong on-chain identity — a PII/design-invariant violation for a system whose stated point is zero
PII on-chain).

**Close with:** an explicit statement of which field is authoritative and who computes it —
e.g. "the bridge always computes `employeeInternalID` itself from `userID` via
`gatewayclient.ComputeEmployeeID`; Talenta never sends a pre-computed `employeeInternalID`" (or the
reverse), removing the two-owner ambiguity the Consistency Conventions row currently permits.

---

## Summary

| # | Hole | Severity |
|---|------|----------|
| H1 | AD-6 dedup ownership left fully open (not just mechanism) → both engineers can legally assume the other owns dedup → net zero dedup | Critical |
| H2 | `recordIdentity` wire encoding/type unspecified → prefixed vs. bare-value writers desync the same hash chain | High |
| H3 | `operationType` value vocabulary/casing unspecified → DELETE special-case (itself an AD-5 rule) can silently fail to match | Medium-High |
| H4 | AD-1 covers retry triggering, not retry idempotency across the bridge's own two side effects (operational-store write + ledger submit) → ambiguous-timeout retries produce duplicate anchors, and no AD assigns responsibility to prevent it | Critical |
| H5 | Family/Additional-Info method-naming ambiguity in AD-4 | contained within one implementer — not a cross-engineer hole |
| H6 | Two plausible, mutually exclusive owners for computing `employeeInternalID` (Talenta vs. bridge) → double-derivation or wrong-identity anchoring | Low-Medium |

Full review written to:
`/Users/chan/www/hyperledger/fabric-hris/_bmad-output/planning-artifacts/architecture/architecture-fabric-hris-talenta-2026-08-11/reviews/review-adversarial.md`

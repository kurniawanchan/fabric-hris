# Reconciliation: ARCHITECTURE-SPINE.md vs. real code

Date: 2026-08-11. Scope: AD-1 through AD-6, Structural Seed, Deferred section.

## 1. AD-1/AD-6 — bridge statelessness and "operational store" claims

**AD-1 ("no queue, retry loop, or dead-letter of its own") — CONFIRMED.**
`integration-bridge/cmd/integrationbridge/main.go` builds `hooks`/`gw` once at startup, wires a
synchronous `http.Server`, and `internal/pipeline/dispatch.go`'s `dispatch()` runs `fn`
synchronously inside a single `context.WithTimeout` — no goroutine detachment, no retry loop, no
persisted queue anywhere in `integration-bridge/` or `internal/pipeline/`. Confirmed accurate.

**AD-6's "operational store `writepaths` already writes to" — MISLEADING, not false.**
`write-path-integration/writepaths/writepaths.go:38-44` defines the actual interface:

```go
type OperationalStore interface {
    SaveSection(ctx context.Context, employeeInternalID, profileSection string, valueJSON []byte) (version int, err error)
    DeleteSection(ctx context.Context, employeeInternalID, profileSection string) error
}
```

This is the entire interface — two methods, no `Get`/`Exists`/lookup-by-ID method of any kind, and
no correlation-ID parameter on either method. It is keyed only by
`employeeInternalID+"/"+profileSection` (see `InMemoryOperationalStore`, lines 53-88) and stores
only the latest section value + a monotonic version counter — nothing resembling an idempotency
ledger. So: the store exists and `writepaths` does write to it, but it supports **nothing**
resembling a dedup-by-correlation-ID lookup today. The spine's Deferred section already says this
correctly ("left open pending a look at what `writepaths`' `OperationalStore` interface already
supports") — but AD-6's own rule text ("the bridge itself does not need to persist dedup state,
since it has no store of its own beyond the operational store `writepaths` already writes to")
reads as if that existing store is a plausible dedup mechanism. It is not, as currently shaped —
it would need a new method and a correlation-ID column/key. This is a nuance the AD-6 rule text is
missing even though the Deferred section next to it gets the caveat right. Recommend tightening
AD-6's own wording to state explicitly that the existing store cannot be reused for dedup without
an interface change, rather than leaving that only implicit in Deferred.

## 2. AD-2 — "the only Fabric-facing write component"

**CONFIRMED, no counter-evidence found.** Searched all of `write-path-integration/` and
`integration-bridge/` for `SubmitRecordProfileSection` callers. Every non-test call site is
`write-path-integration/writepaths/writepaths.go:252` (`h.Gateway.SubmitRecordProfileSection(...)`
inside `doAnchor`), reached only via `Hooks`' five exported methods, all of which are wired as
`integration-bridge`'s route dispatch targets in `cmd/integrationbridge/main.go`'s `buildMux`. All
other matches are test files (`anchor_test.go`, `gatewayclient_integration_test.go`,
`verify_integration_test.go`, `verify_notfound_integration_test.go`,
`tamper_detection_all_sections_integration_test.go`) or a doc-comment reference in
`internal/pipeline/respond.go:26`. No second production caller exists anywhere in the repo today.

## 3. AD-4/AD-5 — validate.go:44 citation and the DELETE-shape claim

**The DELETE-shape substantive claim is correct; the line-number citation is wrong.**
`internal/pipeline/validate.go` line 44 is:

```go
func validateRequest(w http.ResponseWriter, r *http.Request) (employeeInternalID, userID string, newValue, document []byte, err error) {
```

— the function *signature*, not the check itself. The actual "`newValue` must be a JSON object"
check is at **lines 60-62**:

```go
if !isJSONObject(env.NewValue) {
    return "", "", nil, nil, &validationError{errors.New("integrationbridge: newValue is required and must be a JSON object")}
}
```

(backed by `isJSONObject` at lines 81-84, which only checks the first non-whitespace byte is
`{` — never parses the value, exactly as its own doc comment describes). AD-4/AD-5's substantive
claim — that this check is shape-only and therefore already tolerates a DELETE payload carrying
last-known-state JSON with no code change needed — is accurate. Fix: the citation should read
`validate.go:60` (or `:60-62`), not `:44`.

## 4. Structural Seed file paths

- `talenta-core/jobs/AnchorTriggerJob.php` — confirmed absent (`find` for `AnchorTriggerJob*`
  returns nothing) and the entire `talenta-core/` directory does not exist in this checkout. The
  Seed table does mark the whole `talenta-core/` block with `# NEW` only on the job-class line,
  not on the tree header or on `controllers/api/web/MyInfoController.php` — but per CLAUDE.md,
  `talenta-core` is genuinely a different, real repo this workspace doesn't fork/rehost, so its
  absence here is expected and not a defect in the spine; the spine does correctly mark
  `AnchorTriggerJob.php` itself as `# NEW`.
- `integration-bridge/cmd/integrationbridge/main.go` — exists, confirmed by direct read.
- `write-path-integration/writepaths/writepaths.go` — exists, confirmed by direct read.
- `integration-bridge/internal/pipeline/` — exists; `validate.go`, `dispatch.go`, `classify.go` all
  present as cited (plus `respond.go`, `pipeline.go`, route registration files not mentioned by
  name but consistent with the Seed's directory-level description).

No path errors found beyond the line-number citation issue in item 3.

## 5. dispatch.go / classify.go vs. AD-5 — missing plumbing, and a bigger contradiction with AD-3/AD-6

**(a) No `operationType` (or `correlationID`) parameter exists anywhere in the current call chain.**
`internal/pipeline/dispatch.go`'s `DispatchFunc` type is:

```go
type DispatchFunc func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error)
```

— four data parameters, no operation-type or correlation-ID slot. `internal/pipeline/classify.go`
classifies purely by Go error *type* (`*authError`, `*validationError`, `context.DeadlineExceeded`,
`*writepaths.PartialFailureError`) and never inspects operation type at all, so nothing in
`classify.go` conflicts with AD-5. The spine's own Consistency Conventions table already flags
`correlationID`/`operationType` as "additive... not yet present in the route today," so this by
itself isn't a contradiction — just confirming the gap is real and exactly where the spine says
it is (the request envelope in `validate.go`'s `requestEnvelope` struct, `dispatch.go`'s
`DispatchFunc`, and every `Hooks` method signature would all need a new parameter).

**(b) The bigger, unflagged issue: there is no field anywhere downstream — including on-chain —
for `recordIdentity`, `operationType`, or `correlationID` to land in.** Tracing the full path past
`writepaths.go` into the actual chaincode:

- `writepaths.go`'s `doAnchor` builds the on-chain argument list at lines 246-251:
  ```go
  buildArgs := func(prevHash string) []string {
      return []string{
          tenantID, employeeID, profileSection, dataHash, prevHash, updatedBy,
          ipfsCIDsJSON, gatewayclient.CanonicalizationVersion, gatewayclient.HashAlgo, "",
      }
  }
  ```
  Exactly 10 positional args; the trailing `""` is `clientTimestamp` (confirmed against the
  chaincode signature below), not a metadata/free-form slot.
- The chaincode function actually invoked,
  `fabric-network/chaincode/employeeprofilerecord/chaincode/record_profile_section.go`'s
  `RecordProfileSection`, has the signature:
  ```go
  func (s *SmartContract) RecordProfileSection(
      ctx contractapi.TransactionContextInterface,
      tenantId string, employeeID string, profileSection string,
      dataHash string, prevHash string, updatedBy string,
      ipfsCIDs []string, canonicalizationVersion string, hashAlgo string,
      clientTimestamp string,
  ) (*RecordProfileSectionResult, error)
  ```
  — 10 parameters, matching `buildArgs` 1:1. There is no parameter for `recordIdentity`,
  `operationType`, or `correlationID`.
- The hash-chain's actual composite key, used identically in `contract.go:26` and
  `queries.go:84`, is:
  ```go
  ctx.GetStub().CreateCompositeKey(profileKeyObjectType, []string{employeeID, profileSection})
  ```
  — i.e. `(employeeID, profileSection)` only. **Not** `(employee, profileSection,
  recordIdentity)` as AD-3 asserts is the anchor key today.

This means AD-3's key-shape claim ("Every anchor's hash-chain key is the triple `(employee,
profileSection, recordIdentity)`") describes a *target* state, not the current one — the real
on-chain composite key has exactly two segments today, and adding a third (`recordIdentity`) is
not a plumbing-only change confined to `writepaths.go` as the Structural Seed implies ("recordIdentity
plumbed through per AD-3"). It requires changing `RecordProfileSection`'s own signature and/or the
composite-key construction inside the chaincode package — which CLAUDE.md explicitly flags as "a
ratified-design change, not a casual edit" (the on-chain schema is described there as "exactly 12
fields," and extending it needs its own ratification). The spine's Structural Seed table lists only
`write-path-integration/writepaths/writepaths.go` as gaining recordIdentity plumbing and does not
mention `fabric-network/chaincode/employeeprofilerecord/chaincode/` at all — so a change this spine
requires (AD-3) is invisible in its own Structural Seed and outside its own stated scope ("Does not
govern... chaincode internals beyond the ProfileSection/key mapping" — but the composite key IS the
key mapping, so this is arguably already in-scope and simply omitted).

The same gap applies transitively to AD-5 (operation type "forwarded... into the anchor
transaction's metadata" — there is no metadata field to forward it into on-chain; today it could at
most ride inside `sectionValueJSON`/`newValue`, which is exactly the data-shape validate.go checks,
or be dropped entirely before reaching the chaincode) and to AD-6 (an idempotency key "travels with
the request" but nothing carries it past `writepaths.go`'s call into `SubmitRecordProfileSection`
today).

## Summary of gaps

1. AD-6 rule text overstates what the existing `OperationalStore` interface can do for dedup (no
   lookup/Get method, no correlation-ID key) — Deferred section already has the correct caveat,
   but AD-6 itself doesn't.
2. AD-4/AD-5's citation `validate.go:44` points at the function signature, not the actual
   `isJSONObject` check, which is at lines 60-62 (backed by 81-84). Substantive claim is correct;
   only the line number is wrong.
3. **AD-3's key-shape claim is stated as current fact but is not true of the code today** — the
   real composite key is `(employeeID, profileSection)` only, built in `contract.go:26` /
   `queries.go:84`; `RecordProfileSection`'s 10-parameter signature
   (`record_profile_section.go`) has no `recordIdentity` slot, and `writepaths.go`'s `buildArgs`
   (lines 246-251) has no third key segment. Adding one is a chaincode-schema change the spine's
   own Structural Seed doesn't list and that CLAUDE.md flags as needing separate ratification.
4. By the same gap, AD-5's "forwarded unmodified into the anchor transaction's metadata" and AD-6's
   "idempotency key travels with the request" both describe a metadata channel that does not exist
   on-chain or in `buildArgs` today — the trailing `""` arg in `buildArgs` is `clientTimestamp`,
   not a free-form metadata field.
5. Structural Seed / Capability map otherwise check out: `talenta-core/jobs/AnchorTriggerJob.php`
   is correctly marked NEW and confirmed absent from the repo; `integration-bridge/cmd/
   integrationbridge/main.go` and `write-path-integration/writepaths/writepaths.go` exist as
   described; AD-2's "only Fabric-facing write component" claim holds with no second caller found
   anywhere in the codebase (only test files call `SubmitRecordProfileSection` directly).

Net assessment: the spine ratifies the bridge/writepaths boundary correctly (AD-1, AD-2, AD-4/AD-5
data-shape claim) but AD-3 in particular, and by extension the metadata-forwarding language in
AD-5/AD-6, asserts a chaincode-level capability (a third key segment / a metadata slot) that does
not exist in the ratified on-chain schema — this is the one place the spine risks *contradicting*
rather than *ratifying* the brownfield code, because it reads as describing today's key structure
when it is actually proposing tomorrow's.

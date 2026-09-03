**Classification: INTERNAL**

# Code Review: fabric-hris — full-codebase pass (fabric-network/, write-path-integration/, ipfs-cluster/, qa-tests/)

**Reviewer:** Claude (code-review skill)
**Date:** 2026-08-29
**Scope:** `fabric-network/chaincode/employeeprofilerecord/`, `fabric-network/tools/*`, `fabric-network/network/` (config only), `write-path-integration/{gateway-client,keystore,writepaths,ipfsclient}`, `ipfs-cluster/`, `qa-tests/`. Not a single diff — a general-quality pass over these directories as they stand on `main`.
**Intent of change:** N/A (whole-repo review, not one PR) — evaluated against this repo's own stated design intent (CLAUDE.md, in-code ADR references) rather than a ticket.
**Stakes:** DSRM thesis prototype, not production-bound — but the project's own confidentiality register and PII-by-construction claims are load-bearing for the thesis's validity, so findings that undermine those claims are scored at full severity regardless of "prototype" status.

---

## Summary

The chaincode module (`employeeprofilerecord`) is unusually well-disciplined: every non-obvious design decision is documented in-line with its ADR/FR citation, self-flagged gaps are tracked with `G-#` IDs rather than silently shipped, and `go build`/`go vet` are clean across all five Go modules sampled. The one functional gap worth fixing is that `tenantId` is accepted and stored by `RecordProfileSection`/the three query functions but never validated against the caller's MSP identity — it's trusted, unauthenticated metadata. Separately, an uncommitted `.playwright-mcp/` capture directory sitting inside the "hard-clean" `fabric-network/` path contains the real product name and a real internal hostname, which is exactly the leak class CLAUDE.md's confidentiality register exists to catch — not yet committed, but sitting in a path that has leaked twice before per this repo's own history.

**Recommendation:** 🟡 Approve with comments — no correctness-breaking bug found in the sampled code, but the tenantId-trust gap and the confidentiality-register leak should be resolved before this state is committed/shipped.

| Severity | Count |
|----------|-------|
| 🔴 Critical | 1 |
| 🟠 Major | 2 |
| 🟡 Minor | 3 |
| 🔵 Nits | 1 |
| ✅ Praise | 4 |

---

## 🔴 Critical

### C1 — Real product name and internal hostname present, uncommitted, inside the "hard-clean" `fabric-network/` path
**File:** `fabric-network/network/compose/.playwright-mcp/page-2026-08-11T12-01-13-525Z.yml:21`, `fabric-network/network/compose/.playwright-mcp/console-2026-08-11T12-00-52-418Z.log:1`
**Problem:** These files (browser-automation capture artifacts) contain `© 2026 Talenta.co - Advanced Payroll Automation & HR Solution` and `https://hr.talenta.local/site/sign-in`. `git status` shows `.playwright-mcp/` as untracked (`??`) — it is **not** covered by any `.gitignore` (unlike the sibling `.remember/` directory, which has its own `.gitignore: *`). CLAUDE.md's confidentiality register names `fabric-network/` as hard-clean, "verified zero hits" as of 2026-08-07, and explicitly notes: *"`integration-bridge/` code comments have leaked twice already ... the pattern itself doesn't catch every possible real-name shape, so treat the grep as a floor, not a guarantee."*
**Why it matters:** A `git add -A` / `git add .` from this directory (both of which the org's own commit protocol warns against, but a less careful contributor could still run) would commit real product/company identifying material into a path this repo's own rules certify as scrubbed. This is precisely the failure mode CLAUDE.md's grep floor is meant to catch, and this artifact type (structured YAML/log dumps of a live UI session) isn't shaped like the prior two leaks (a file path, a product name in a comment), so the standard `grep -rniE "talenta|mekari"` sweep would catch it only if run recursively including dotfiles — worth confirming it does.
**Suggested fix:** Delete `fabric-network/network/compose/.playwright-mcp/` (uncommitted, so no history to worry about), and add `.playwright-mcp/` to `.gitignore` at the repo root or under `fabric-network/network/compose/` the same way `.remember/` is already excluded, so this class of artifact can't land in the hard-clean zone again.

```
fabric-network/network/compose/.playwright-mcp/page-2026-08-11T12-01-13-525Z.yml:21:
    - generic [ref=f3e19]: © 2026 Talenta.co - Advanced Payroll Automation & HR Solution
```

---

## 🟠 Major

### M1 — `tenantId` is accepted, persisted, and echoed back, but never validated against the caller's MSP identity
**File:** `fabric-network/chaincode/employeeprofilerecord/chaincode/record_profile_section.go:71-123`, `fabric-network/chaincode/employeeprofilerecord/chaincode/queries.go:21-172`
**Problem:** All four contract functions take `tenantId` as a parameter. `requiredStringArgs` checks only that it's non-empty (`record_profile_section.go:114-123`); it is written into `EmployeeProfileRecord.TenantID` (`record_profile_section.go:179`) and never compared against the caller's MSP ID (`OrgClient-<tenantID>MSP`, established in `identity.go:24-34`). The world-state composite key is also, by design, `("profile", employeeID, profileSection)` with no tenant component (`contract.go:56-60`, deliberately — "tenant isolation is a channel-membership property, not a key-shape property"). The net effect: within one tenant's channel, a caller authorized only for that channel can submit (or query) with an arbitrary `tenantId` string, and nothing in the chaincode notices the mismatch.
**Why it matters:** Channel membership does bound *which* channel a caller can reach, so this is not a cross-tenant data-read bypass. But `TenantID` is on-chain, persisted metadata that downstream audit/reconciliation tooling (and the thesis's own tamper-evidence claims) will treat as trustworthy provenance — a caller (or a bug in the write path) that submits the wrong `tenantId` string produces a record whose stored tenant attribution is simply wrong, silently, with no chaincode-level defense-in-depth. This is the same class of gap the code already self-flags for `employeeID`/`updatedBy`/hash-shape validation as `G-31` — but `G-31` doesn't mention `tenantId` specifically, and unlike the hash-shape fields, `tenantId` has an unambiguous, cheap validation available: compare it against `callerMSPID(ctx)`'s `OrgClient-<tenantId>MSP` shape.
**Suggested fix:** In `authorizeSubmit`/`authorizeEvaluate` (or immediately after), when the caller's MSP is `OrgClient-<X>MSP`, require `tenantId == X`; when the caller is Org1/Org3, either continue trusting it (documented as such) or route through whatever tenant-scoping the write path already asserts. At minimum, add a `G-#` entry to `grounding-gaps.md` alongside `G-31` so this is tracked as a known, disclosed gap rather than an implicit assumption — per this repo's own stated practice of recording every new limitation there.

### M2 — `GetProfileHistory` re-sorts on every call instead of relying on a documented invariant, silently masking a test-double limitation
**File:** `fabric-network/chaincode/employeeprofilerecord/chaincode/queries.go:59-70, 127`
**Problem:** The function explicitly sorts by `Version` because "this project's corpus does not pin [`GetHistoryForKey`'s] iteration order," and — separately — because the unit-test double doesn't implement `GetHistoryForKey` at all. Both are disclosed in the comment, which is good, but the practical result is that this code path has **zero unit-test coverage** for its main behavior (ordering under a real history iterator), only for the sort function's logic in isolation (if that). This isn't wrong to ship, but it's an untested code path being relied on for a correctness guarantee (`api-contracts.md`'s "ascending chronological order").
**Suggested fix:** Not blocking, since Evaluate-only/no-determinism-consequence is a reasonable argument — but worth a note in `grounding-gaps.md` if one doesn't already exist specifically for "GetProfileHistory ordering has no test against a real GetHistoryForKey iterator," distinct from the general MockStub limitation, so the gap is traceable independent of this comment.

---

## 🟡 Minor

### m1 — `deterministicRecordID`'s literal-spec deviation is well-flagged but has no test asserting its determinism property across "endorsers"
**File:** `fabric-network/chaincode/employeeprofilerecord/chaincode/identity.go` (n/a — actually `record_profile_section.go:34-36`)
**Suggestion:** The function is short and the property being relied on (same `txID` in ⇒ same UUID out) is trivially true of `uuid.NewSHA1`, so this is genuinely low risk — but since the whole justification for this deviation is "determinism across independently-simulating endorsing peers," a one-line test asserting `deterministicRecordID(x) == deterministicRecordID(x)` for a couple of inputs would make that property explicit rather than implicit, and would catch a future accidental swap to `uuid.New()` immediately instead of at integration time.

### m2 — `isOrgClientMSP` accepts any non-empty tenant segment, including e.g. `OrgClient-MSP` is correctly rejected but `OrgClient-<anything>MSP` (no format check on the tenant segment) is accepted
**File:** `fabric-network/chaincode/employeeprofilerecord/chaincode/identity.go:30-34`
**Suggestion:** Purely a robustness note, not a security bypass (the tenant segment still has to correspond to a real MSP the ordering service and channel config recognize, so this can't be spoofed by an unauthenticated party) — but if `tenantId`-vs-MSP cross-checking is added per M1, it's worth deciding here whether the tenant segment extracted from the MSP ID should be format-validated (e.g., matches the same charset tenant IDs are provisioned with) rather than accepted as any non-empty string between the fixed prefix/suffix.

### m3 — `qa-tests/performance/results/raw-latencies/*.jsonl` and `report.html` are tracked, binary-ish generated artifacts that are being hand-edited/re-run in place
**File:** `qa-tests/performance/results/raw-latencies/pt1-diag-20tps-5w.worker{0..4}.jsonl`, `qa-tests/performance/report.html`
**Suggestion:** These show as modified in `git status` for this session. CLAUDE.md's own performance-benchmark section warns "every benchmark result in this repo was invalidated once by concurrent load and had to be re-measured... a careless re-run is not free." Since these are already committed/tracked files being overwritten in place with no accompanying commit message explaining the re-run's cause, it's worth confirming (before committing) that this run was solo/quiet-network per the documented procedure — not a code-quality issue per se, but the kind of "what isn't there" gap (no note on *why* these changed) this review step is meant to surface.

---

## ✅ Praise

- `fabric-network/chaincode/employeeprofilerecord/chaincode/asset.go:63-89` — the on-chain struct's field-order comment (alphabetic by Go identifier, to keep `encoding/json`'s deterministic non-sorting behavior byte-identical across endorsers) is exactly the kind of non-obvious correctness reasoning that's easy to silently break in a future edit; documenting it in place is the right call.
- `fabric-network/chaincode/employeeprofilerecord/chaincode/identity.go:36-59` — the comment distinguishing the chaincode-level ABAC gate from the channel-level endorsement policy (NET-5) is genuinely useful: it's a real, non-obvious distinction (identity-of-caller vs. identity-of-endorser) that a future reviewer could easily conflate, and getting it wrong would either weaken the deny-by-default posture or cause confused debugging.
- `write-path-integration/keystore/filestore.go:110-119` — the write-temp-file-then-rename pattern for the encrypted key stores is correct crash-safety practice, and the comment explaining why (never leave a half-written unreadable store) shows it was a deliberate choice, not an accident.
- `fabric-network/chaincode/employeeprofilerecord/chaincode/record_profile_section.go:100-113` — self-flagging `G-31` (validation gap) *in the code, at the exact point the gap exists*, with a citation to the scan that found it and an explicit statement that "no real PII was found... but the underlying enforcement gap is real," is a good practice — it keeps the disclosure adjacent to the code instead of only in a separate doc that can drift out of sync.

---

## 🔵 Nits

- `fabric-network/chaincode/employeeprofilerecord/chaincode/queries.go:143-171` — `GetEmployeeProfileSummary`'s two near-identical `ProfileSectionSummaryEntry` construction blocks (found vs. not-found) could be a single construction with conditional field values, but the current form is also perfectly readable — not worth insisting on.

---

## What I didn't review

- `fabric-network/network/configtx/`, `fabric-network/network/crypto-config/` — config/generated-material directories, skimmed for structure only, not evaluated line-by-line.
- `ipfs-cluster/` — read only `README.md`-level context (CLAUDE.md's summary of the CRDT-handshake defect); did not review its Go/compose internals in depth.
- `qa-tests/` suites beyond the performance-results directory noted in m3 — coverage reports and evaluation docs were not read in full.
- `write-path-integration/writepaths` and `write-path-integration/ipfsclient` internals — confirmed they build/vet clean but were not read function-by-function the way `keystore` and the chaincode package were.
- Full diff history / commit-by-commit provenance — this was a snapshot review of `main`, not a review of a specific PR's commit sequence.

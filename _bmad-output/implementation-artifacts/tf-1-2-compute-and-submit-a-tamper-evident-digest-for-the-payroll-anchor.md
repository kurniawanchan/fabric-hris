---
baseline_commit: 08854aab1bdba48ed37a5d7dd7ed5f7e0abb4f4a  # talenta-core
---

# Story tf-1.2: Compute and submit a tamper-evident digest for the Payroll anchor

Status: review

<!-- Retroactive-verification pass, not new implementation (see Dev Notes) -->

## Story

As the anchoring job,
I want to canonicalize and hash the changed Payroll data and submit it to the existing bridge,
So that the change is provably fingerprinted on the ledger without any raw payroll data ever leaving Talenta.

## Acceptance Criteria

1. **Given** a Payroll anchoring job has the post-write field set, **When** the job computes the digest, **Then** it uses `fabric-hris`'s existing JCS-canonicalization + salted HMAC-SHA256 construction, and the raw field values are never transmitted to `integration-bridge` or written on-chain (NFR-5).
2. **Given** this is the first-ever anchor for a given employee's Payroll record, **When** the anchor is submitted, **Then** the previous-state hash is an explicit all-zero genesis sentinel, never null or omitted (FR-3).
3. **Given** Payroll is a single-record-per-employee domain, **When** the pseudonym is derived for this anchor, **Then** it pseudonymizes on `employeeID` alone (`recordIdentity = self`), per AD-3's trivial case (FR-Arch-1).
4. **Given** a full-ledger PII scan (`pilscan`) is run after this anchor exists, **When** the scan checks the new Payroll field set, **Then** it finds zero raw PII on-chain (NFR-5).

## Verification Method

This story's behavior was already delivered by Epic tf-1's real code (`writepaths.doAnchor`, `gatewayclient.ComputeDataHash`) before this story's own record ever existed. Rather than re-implement, this pass traces each AC against the actual shipped code and reports PASS/FAIL/PARTIAL per AC, per the sponsor's explicit direction (2026-08-13).

## Verification Findings

**AC #1 — PASS, with one terminology correction.** `write-path-integration/writepaths/writepaths.go:266` (`doAnchor`) calls `gatewayclient.ComputeDataHash(salt, sectionValueJSON)`. `write-path-integration/gateway-client/digestbuilder.go:62-74` shows the actual construction: `jsoncanonicalizer.Transform` (real RFC-8785 JCS) then `sha256(salt || canonical)` — a **salted SHA-256 digest**, not the formal HMAC construction (`HMAC(key, message)` with its own inner/outer padding). `ComputeEmployeeID`/`ComputeUpdatedBy` (same file, lines 80-102) DO use real `hmac.New(sha256.New, ...)`. The AC's phrase "salted HMAC-SHA256" is imprecise for `DataHash` specifically — it is salted, and SHA-256, but not HMAC. Functionally equivalent for the tamper-evidence property this AC actually cares about (raw values never leave Talenta; `AnchoringWriteService::submitAnchor`'s body only ever carries `newValue` to `integration-bridge`, which computes the digest server-side — confirmed no raw-value logging anywhere in `write-path-integration/writepaths/writepaths.go` or `integration-bridge/internal/pipeline/`). Not a defect; a naming correction worth making in the epics doc if this is ever revised.

**AC #2 — FAIL as literally worded, functionally correct.** `write-path-integration/gateway-client/gatewayclient.go:114-121` (`SubmitRecordProfileSection`): `prevHash := ""`; if `EvaluateGetProfileSectionRecord` errors (no prior record — the first-ever-anchor case), `prevHash` stays the **empty string**, not an explicit all-zero sentinel (e.g. `sha256:0000...0000`). The chaincode (`record_profile_section_test.go`'s `"non-empty prevHash supplied when no head exists yet"` case) correctly REJECTS a non-empty prevHash when no head exists, and correctly ACCEPTS the empty string as the implicit genesis value — so there is no functional bug (a first-ever anchor commits correctly, tested). But the AC's literal text ("never null or omitted") is not met — an empty string is exactly the "omitted" shape the AC says must not happen. Logged as `grounding-gaps.md` G-42 rather than silently marked passing.

**AC #3 — PASS.** All five talenta-core controllers' single-record-per-employee domains (Payroll included, via `MyInfoController::actionUpdatePayroll`/`actionDeletePayroll`/`actionAddComponent`) enqueue `AnchorTriggerWorker` with no `recordIdentity` key set, so `$this->args['recordIdentity'] ?? ''` resolves to `''`; `writepaths.ResolveEmployeeKey`/`ResolveEmployeeID` (`writepaths.go`) treat `''` as the trivial self-pseudonym case exactly as AD-3 specifies — confirmed by `TestAnchor_EmptyRecordIdentity_MatchesPreExistingSingleRecordPseudonym` (Story tf-2.0's own test).

**AC #4 — NOT VERIFIED in this pass.** Re-running `fabric-network/tools/pilscan` (a multi-minute full-ledger scan per `CLAUDE.md`) against a live network with a real Payroll anchor was out of scope for a documentation-tracing pass; this AC's prior general satisfaction is evidenced by `qa-tests/security/`'s existing full-ledger confidentiality scan (per `README.md`), not re-run specifically for this story.

## Dev Notes

- **This is a verification pass, not new code.** No files were modified to satisfy this story; `Project Structure Notes`/`File List` are empty by design.
- **G-42 (new, this pass):** first-ever-anchor `prevHash` is passed as `""`, not an explicit all-zero genesis sentinel, contradicting FR-3's literal wording despite the chaincode handling the empty-string case correctly today. See `grounding-gaps.md`.

### References

- [Source: _bmad-output/planning-artifacts/epics-talenta-fabric-2026-08-12.md#Epic 1, Story 1.2]
- [Source: write-path-integration/gateway-client/digestbuilder.go]
- [Source: write-path-integration/gateway-client/gatewayclient.go:114-143]
- [Source: fabric-network/chaincode/employeeprofilerecord/chaincode/record_profile_section_test.go]

## Dev Agent Record

### Agent Model Used

Claude Sonnet 5 (claude-sonnet-5)

### Completion Notes List

- AC #1: PASS (terminology note on HMAC vs salted-SHA256 for `DataHash` specifically).
- AC #2: FAIL as literally worded; functionally correct. New gap G-42.
- AC #3: PASS.
- AC #4: Not independently re-verified in this pass.

### File List

(none — verification-only pass, no code changed)

## Change Log

- 2026-08-13: Retroactive verification pass against already-shipped code. 2/4 ACs PASS, 1 FAIL-as-worded-but-functionally-correct (new gap G-42 logged), 1 not independently re-verified. Status set to `review` pending the sponsor's decision on whether G-42 needs fixing before this closes to `done`.

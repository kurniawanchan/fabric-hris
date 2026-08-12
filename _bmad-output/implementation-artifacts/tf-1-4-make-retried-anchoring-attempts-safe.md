---
baseline_commit: 08854aab1bdba48ed37a5d7dd7ed5f7e0abb4f4a  # talenta-core
---

# Story tf-1.4: Make retried anchoring attempts safe

Status: done

<!-- Retroactive-verification pass, not new implementation (see Dev Notes) -->

## Story

As the anchoring job,
I want a retried submission of the same logical write to never create a duplicate ledger entry,
So that transient failures don't corrupt an employee's history.

## Acceptance Criteria

1. **Given** the same anchoring job is submitted twice (e.g. due to a retry after an ambiguous timeout), **When** both submissions reach the ledger, **Then** no duplicate chain entry is created (FR-5).
2. **Given** the chaincode's own `RecordProfileSection` is a verified no-op on an identical `(dataHash, prevHash)` pair, **When** this story is implemented, **Then** the anchoring job's own retry logic relies on that no-op rather than re-implementing idempotency itself.

## Verification Findings — PASS

**AC #1 — PASS.** `fabric-network/chaincode/employeeprofilerecord/chaincode/record_profile_section_test.go`'s `TestRecordProfileSection_NoOpOnIdenticalDataHash` proves a second `RecordProfileSection` call with the same `dataHash` for the same `(employeeID, profileSection)` — even with a caller-supplied `prevHash` equal to the unchanged head — echoes the existing head's `RecordID`/`Version`/`Timestamp` and mutates zero world-state keys (`require.Equal(t, stateLenAfterFirst, len(stub.State))`). Separately, `write-path-integration/gateway-client/gatewayclient.go:114-143` (`SubmitRecordProfileSection`) retries only on the two EXPECTED rejection classes (`MVCC_READ_CONFLICT`, stale-prevHash), re-reading the real current head and rebuilding `prevHash` before each retry — so an ambiguous-timeout retry submits with a freshly correct `prevHash`, landing on the chaincode's no-op path if the first attempt actually committed, rather than corrupting the chain with a stale reference.

**AC #2 — PASS.** Nothing in `write-path-integration/writepaths/writepaths.go` or `gatewayclient.go` re-implements deduplication, idempotency keys, or a separate "already submitted" check — `SubmitRecordProfileSection`'s retry loop (`gatewayclient.go:124-142`) relies entirely on the chaincode's own no-op behavior plus its own re-read-then-resubmit pattern, exactly as AC #2 specifies.

Separately, `AnchorTriggerWorker::canRetry` (talenta-core) provides an independent, higher-level layer of retry-safety at the job-queue level (yii2-queue's `RetryableJobInterface`, capped at 5 attempts before dead-lettering) — not what this AC is about (this AC is about the chaincode/ledger layer specifically), but confirms the safety property holds end-to-end across both layers.

## Dev Notes

- **This is a verification pass, not new code.** No files were modified.
- No gaps found for this story.

### References

- [Source: _bmad-output/planning-artifacts/epics-talenta-fabric-2026-08-12.md#Epic 1, Story 1.4]
- [Source: fabric-network/chaincode/employeeprofilerecord/chaincode/record_profile_section_test.go]
- [Source: write-path-integration/gateway-client/gatewayclient.go:114-143]

## Dev Agent Record

### Agent Model Used

Claude Sonnet 5 (claude-sonnet-5)

### Completion Notes List

- AC #1: PASS — chaincode-tested no-op, retry loop re-reads head before resubmitting.
- AC #2: PASS — no separate idempotency mechanism was built; the job relies on the chaincode's own no-op, as specified.

### File List

(none — verification-only pass, no code changed)

## Change Log

- 2026-08-13: Retroactive verification pass against already-shipped code. Both ACs PASS. Status set to `done`.

---
baseline_commit: 08854aab1bdba48ed37a5d7dd7ed5f7e0abb4f4a  # talenta-core
---

# Story tf-1.3: Carry full anchor metadata and a traceable correlation ID

Status: review

<!-- Retroactive-verification pass, not new implementation (see Dev Notes) -->

## Story

As an auditor,
I want every anchoring transaction to carry who/what/when/which-endpoint plus a correlation ID,
So that I can trace any anchored change back to the exact Talenta request that produced it.

## Acceptance Criteria

1. **Given** a Payroll anchoring job submits its transaction, **When** the anchor lands, **Then** it carries employee `id`, `company_id`, domain, operation type, actor identity, source endpoint, timestamp, previous-state hash, new-state hash, and a correlation ID (FR-4).

## Verification Findings — FAIL

**employeeID, domain (`profileSection`), actor identity, dataHash/prevHash — PASS.** `write-path-integration/writepaths/writepaths.go:292-299` (`doAnchor`'s `buildArgs`) shows the actual on-chain `RecordProfileSection` args: `tenantID, employeeID, profileSection, dataHash, prevHash, updatedBy, ipfsCIDsJSON, CanonicalizationVersion, HashAlgo, ""`. `employeeID` (pseudonym), `profileSection` (domain), `updatedBy` (HMAC-derived actor identity, `digestbuilder.go:95-102`), `dataHash`/`prevHash` all genuinely land on-chain.

**`company_id` — present, but off-chain only.** `AnchoringWriteService::submitAnchor` puts `companyId` in the JSON body (`services/fabric/AnchoringWriteService.php:55`), but `BaseFabricBridgeService::bridgeHeaders()` (line 58-64) sends company scoping via the `X-Company-ID` HTTP header instead, and `integration-bridge`'s `requestEnvelope` struct (`integration-bridge/internal/pipeline/validate.go:16-21`) has no `CompanyID`/`companyId` field at all — Go's `encoding/json` silently drops unknown JSON fields, so the body's `companyId` is dead data, never read. The header-carried tenant scoping (`tenantID`) IS what reaches the chaincode call (`buildArgs`'s first positional arg) — so company/tenant identity does land on-chain, just not via the field this AC's wording implies.

**`operationType`, correlation ID, source endpoint, timestamp — FAIL, confirmed dropped.** A repo-wide case-insensitive search (`grep -rni "correlationID\|correlation_id" integration-bridge/ write-path-integration/`) returns **zero results** — `correlationID` never appears anywhere in the Go codebase. `requestEnvelope` (`validate.go:16-21`) has fields for `EmployeeInternalID`, `UserID`, `NewValue`, `Document`, `RecordIdentity` only — no `OperationType`, no `CorrelationID`. `dispatch.go`'s `DispatchFunc`/`dispatch()` (lines 8-38) carry no operationType, correlationID, or source-endpoint parameter either. `AnchoringWriteService::submitAnchor` (talenta-core) sends `operationType` and `correlationID` in its request body; `integration-bridge` silently discards both on arrival — `AD-5`'s own claim ("operationType... logged in integration-bridge's own request handling") does not match the actual code; there is no such logging anywhere in `integration-bridge`. **`clientTimestamp`, the 10th and final `RecordProfileSection` argument, is hardcoded to the literal empty string `""`** (`writepaths.go:296`) — every anchor ever submitted carries no client-supplied timestamp on-chain at all (the chaincode's own commit timestamp is a separate, ledger-assigned value, not what this AC asks for). Source endpoint (which of the five `/v1/profile-sections/<SECTION>` routes handled the request) is likewise never logged or carried anywhere beyond the HTTP routing itself.

**Net result:** correlationID exists and is genuinely useful *within talenta-core alone* — `AnchorTriggerWorker`'s retry loop and `DeadLetterStore` both use it, so an operator CAN trace a dead-lettered/retried job back to one correlation ID on the Talenta side. But the AC's actual claim — "trace any anchored change back to the exact Talenta request that produced it," implying end-to-end traceability through `integration-bridge` to the ledger — does not hold. Nothing observable on the bridge or ledger side carries a correlationID, operationType, source endpoint, or client timestamp.

## Dev Notes

- **This is a verification pass, not new code.** No files were modified.
- **New gap G-43** logged in `grounding-gaps.md`: correlationID/operationType/source-endpoint/client-timestamp are all silently dropped by `integration-bridge`, contradicting both this story's AC and `AD-5`'s own claim about operationType logging.
- **Why this matters beyond this one story:** every story built on top of Epic tf-1 (tf-2 through tf-4) assumed the write path this story describes was complete and correct — none of them independently re-checked correlationID propagation past talenta-core's own boundary, because AD-5 said it was handled. It was not.

### References

- [Source: _bmad-output/planning-artifacts/epics-talenta-fabric-2026-08-12.md#Epic 1, Story 1.3]
- [Source: integration-bridge/internal/pipeline/validate.go:16-21]
- [Source: integration-bridge/internal/pipeline/dispatch.go]
- [Source: write-path-integration/writepaths/writepaths.go:252-300]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/services/fabric/AnchoringWriteService.php]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/services/fabric/BaseFabricBridgeService.php]

## Dev Agent Record

### Agent Model Used

Claude Sonnet 5 (claude-sonnet-5)

### Completion Notes List

- AC #1: FAIL. employeeID/domain/actor-identity/dataHash/prevHash land on-chain; company_id lands via header (tenantID), not the body field this AC implies; operationType, correlationID, source endpoint, and client timestamp are all silently dropped between talenta-core and integration-bridge. New gap G-43 logged.
- This story is left **not done** — closing it requires an actual code fix (adding fields to `requestEnvelope`/`dispatch`/`doAnchor`'s call chain, or a documented, deliberate scope-narrowing decision), not something this verification pass can resolve on its own.

### File List

(none — verification-only pass, no code changed)

## Change Log

- 2026-08-13: Retroactive verification pass against already-shipped code. AC #1 FAILS — correlationID/operationType/source-endpoint/timestamp are silently dropped at the integration-bridge boundary (new gap G-43). Status set to `review`; NOT marked `done` — recommend the sponsor decide whether to fix this or accept it as a disclosed, permanent scope gap.

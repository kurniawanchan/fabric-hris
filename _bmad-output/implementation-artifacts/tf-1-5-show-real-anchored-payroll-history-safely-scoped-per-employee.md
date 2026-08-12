---
baseline_commit: 08854aab1bdba48ed37a5d7dd7ed5f7e0abb4f4a  # talenta-core
---

# Story tf-1.5: Show real anchored Payroll history, safely scoped per employee

Status: done

<!-- Retroactive-verification pass, not new implementation (see Dev Notes) -->

## Story

As an employee,
I want to see my own verified Payroll change history,
So that I can confirm my data hasn't been tampered with — and as an HR Admin, I want to see it for employees in my company.

## Acceptance Criteria

1. **Given** any employee, regardless of role, **When** they open their own Payroll transaction history, **Then** they see their own real anchored history via the existing `TransactionHistory.vue`/`MyInfoBlockchainHistoryService` read path, with no elevated role required (FR-6).
2. **Given** an HR Admin, Super Admin, or Consultant, **When** they view another employee's Payroll history within their own company, **Then** they see it, per today's existing `hasAccess()` permission model — no new permission is introduced (FR-7).
3. **Given** an employee requests their own Payroll transaction history, **When** the response is returned, including any error or empty-state case, **Then** it contains no field values, digests, or metadata belonging to any other employee (FR-8, AC-7).

## Verification Findings — PASS

**AC #1 — PASS.** `controllers/api/web/MyInfoController.php:172` (`actionBlockchainHistory`) is registered in the `'blockchain-history'` action list (`beforeAction`, line 102), reachable by any authenticated employee (`hasAccess($request)` gate, line 111 — the same gate every other My Info action uses, no elevated role check specific to this action). It calls `services/MyInfoBlockchainHistoryService.php:33` (`list()`), which calls `BlockchainAnchoringService::getTransactionHistory((string) $userId, $profileSection)` — `$userId` is `checkUserId($id)`'s resolved target, defaulting to the caller's own ID. `vue/pages/my-info/history/TransactionHistory.vue` exists at the exact path the AC names.

**AC #2 — PASS by design.** The AC's own wording ("per today's existing `hasAccess()` permission model — no new permission is introduced") is exactly what the code does: `actionBlockchainHistory` accepts an optional `$id` (`checkUserId($id)`, same pattern as `actionPersonalData`/every other My Info read action), gated by the identical `hasAccess($request)` check `beforeAction` already applies to `personal-data`/`family-data`/`employment-data`. No new ACL rule, role check, or permission was added for this action specifically — it inherits the platform's own pre-existing HR-Admin/Super-Admin/Consultant company-scoping logic wholesale. (This pass did not re-verify `hasAccess()`'s own internal ACL rules — they pre-date this integration and are that service's own, separately-owned correctness concern, not this story's.)

**AC #3 — PASS.** `MyInfoBlockchainHistoryService::list()` passes exactly one `$userId` (int, the target employee's own talenta-core ID) into `getTransactionHistory`, which resolves that employee's on-chain pseudonym server-side (`fabric-hris`'s `writepaths.ResolveEmployeeID`) — the whole ledger query is scoped to one employee's own pseudonym by construction; there is no code path in this service that could return another employee's entries. `integration-bridge`'s own `history_route_test.go` (fabric-hris side) independently tests this same per-employee isolation property at the route level.

## Dev Notes

- **This is a verification pass, not new code.** No files were modified.
- No gaps found for this story.

### References

- [Source: _bmad-output/planning-artifacts/epics-talenta-fabric-2026-08-12.md#Epic 1, Story 1.5]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/controllers/api/web/MyInfoController.php:90-189]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/services/MyInfoBlockchainHistoryService.php]
- [Source: talenta-core-feature-HF001-hyperledger-fabric-integration/vue/pages/my-info/history/TransactionHistory.vue]
- [Source: integration-bridge/internal/pipeline/history_route_test.go]

## Dev Agent Record

### Agent Model Used

Claude Sonnet 5 (claude-sonnet-5)

### Completion Notes List

- AC #1: PASS.
- AC #2: PASS by design — reuses the pre-existing `hasAccess()` model exactly as the AC specifies; that model's own internal rules were not independently re-audited (out of this story's scope).
- AC #3: PASS.

### File List

(none — verification-only pass, no code changed)

## Change Log

- 2026-08-13: Retroactive verification pass against already-shipped code. All 3 ACs PASS. Status set to `done`.

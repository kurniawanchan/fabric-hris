---
title: 'Reconcile — PRD vs talenta-core HF001 codebase'
status: draft
created: '2026-08-11'
project: fabric-hris
register: 'real-names (evidence-linked reconciliation against talenta-core — see repo confidentiality register in CLAUDE.md)'
source_prd: _bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-11/prd.md
source_codebase: /Users/chan/www/talenta-core-worktrees/talenta-core-feature-HF001-hyperledger-fabric-integration
---

# Reconcile: PRD-fabric-hris-2026-08-11 vs. talenta-core HF001

Method: read the PRD in full, then read (in full or by relevant method) the seven cited
talenta-core files plus the Vue history pages/repositories, and cross-checked every class name,
method name, hook point, and architectural claim in the PRD against the actual code.

## Summary verdict

The PRD's core architectural claim — **read path exists, write-path anchor does not** — is
accurate and directly corroborated by the code, including a comment in
`MyInfoBlockchainHistoryService` that states it almost verbatim. The class/method names the PRD
cites for the read path (`BaseFabricBridgeService`, `BlockchainAnchoringService::getTransactionHistory`,
`MyInfoBlockchainHistoryService::list`, `TransactionHistory.vue`, `BlockchainHistoryRepository.list`)
all match exactly. However, the PRD's write-path framing overstates how uniform and reusable the
existing webhook hook is. Below are the gaps found.

## Gaps

### Gap 1 — "fires on every mutating MyInfoController action" is false; several Payroll mutations never call `publish()`

- **File:line**: `controllers/api/web/MyInfoController.php:705-752` (`actionDeletePayroll`,
  `actionAddComponent`)
- **PRD section/claim**: §2 Current-State/Proposed-State table, row "Write path": *"Profile writes
  hit Talenta's DB only; `EmploymentUpdateWebhookService::publish()` fires a `WebhookWorker` job for
  unrelated downstream integrations"* — phrased as a general description of the write path, and
  relied on directly by FR-1 ("For every completed write to a Personal/.../Payroll endpoint...the
  system SHALL enqueue an anchoring job via the existing `EmploymentUpdateWebhookService` →
  `WebhookWorker` hook").
- **What the code shows**: Scanning every `action*` method in `MyInfoController.php` (33 actions
  total) against every `EmploymentUpdateWebhookService::setUserId(...)->publish()` call site shows
  the hook fires from only 7 of them: `actionUpdatePersonalData`, `actionUpdateIdentityAddress`,
  `actionSaveFamilyData`, `actionDeleteFamilyData`, `actionUpdateEmploymentData`,
  `actionStoreEmergencyContact`/`actionUpdateEmergencyContact`, and `actionUpdatePayroll` (line 687).
  Two Payroll-domain mutations that are squarely in the PRD's §6 scope —
  `actionDeletePayroll` (deletes a payroll record) and `actionAddComponent` (adds a payroll
  component) — persist data (`payrollInfoService->deletePayroll`/`addComponent`, followed by
  `$this->save = true`) but never call `EmploymentUpdateWebhookService::publish()` anywhere in their
  bodies. `actionDeleteEmergencyContact` (line 493) and `actionImportDataEmergencyContact` (line 529)
  also have no `publish()` call.
- **Impact on PRD**: FR-1 as written ("for every completed write... the system SHALL enqueue an
  anchoring job via the existing hook") will silently under-anchor Payroll deletes/component adds
  and some emergency-contact mutations if implemented as a pure extension of the *existing*
  `publish()` call sites — those endpoints have no call site to extend. The PRD needs either (a) a
  corrected endpoint inventory that adds new `publish()` calls to these actions, or (b) an explicit
  note that FR-1's "every completed write" is aspirational relative to today's call-site coverage,
  not a description of what merely extending the existing hook will achieve. This also weakens
  NFR-6/FR-11 framing, since reconciliation (FR-11) would need to treat these as a known, permanent
  gap-source rather than a transient failure.

### Gap 2 — `WebhookWorker` is not a generic extensible job; it is a single-purpose dispatcher hard-wired to one downstream webhook POST, undermining G6's "no new queue infrastructure" framing

- **File:line**: `workers/WebhookWorker.php:28-145` (class `WebhookWorker`, `init()`, `execute()`)
- **PRD section/claim**: §3 G6 ("Reuse the existing `EmploymentUpdateWebhookService` →
  `WebhookWorker` hook — no new queue infrastructure") and §2 row "Write path" ("Same webhook hook is
  extended to also enqueue a Fabric-anchoring job per write").
- **What the code shows**: `WebhookWorker::init()` injects exactly one collaborator,
  `WebhookService`, and `execute()` is a single large `switch ($eventType)` that builds one
  `$body['payload']` array and always ends by calling `(new WebhookRestService())->postDataMessage($body)`
  — i.e. every branch of the switch converges on POSTing to the one external webhook receiver
  (the Kafka/Mekari-Flex consumer referenced in the PRD's own table). There is no branch, hook, or
  injected dependency anywhere in this class that calls out to a second, unrelated downstream
  system (e.g. a Fabric Gateway client) — adding Fabric-anchoring here is not "reusing the hook",
  it is modifying `WebhookWorker::execute()`'s core control flow (a new case that does NOT
  `postDataMessage`, or a parallel dispatch inside the same job) and, more importantly, giving this
  job a second responsibility and a second failure domain it was not built for. `WebhookWorker` also
  has a swallow-and-log-only `catch (Exception $exception)` at the end of `execute()`
  (`workers/WebhookWorker.php:140-144`, `echo $errorMessage` with no rethrow, no dead-letter, no
  queue-level failure signal) — this directly conflicts with the PRD's FR-12/FR-13 resilience
  requirements (retry-with-backoff, dead-letter visibility) which assume the job framework already
  supports that; today a Fabric-anchoring branch added inside this same `try` would have its
  failures silently swallowed exactly like every existing webhook failure is, unless the PRD's
  design explicitly gives the new Fabric branch its own catch/retry/dead-letter path outside this
  existing swallow-all block.
- **Impact on PRD**: G6 and FR-1/FR-12/FR-13 should be revised to say the design reuses the
  **enqueue mechanism** (`Yii::$app->queue->push(new WebhookWorker(...))`) but requires either a new
  job class (still using the Yii2 queue infra, so "no new queue infrastructure" remains true) or a
  new case + new error-handling path inside `WebhookWorker::execute()` that does not reuse
  `WebhookRestService::postDataMessage` and does not fall into the existing catch-and-swallow
  block. As written, the PRD reads as if the anchoring job rides passively inside the existing
  single-target dispatch, which is not what the code supports.

### Gap 3 — Read-path claim is accurate and fully corroborated (not a gap, recorded as verification)

- **File:line**: `services/MyInfoBlockchainHistoryService.php:9-14` (class doc comment),
  `controllers/api/web/MyInfoController.php:170-188` (`actionBlockchainHistory`),
  `vue/pages/my-info/history/TransactionHistory.vue:63,164`,
  `vue/repositories/my-info/BlockchainHistoryRepository.js:1-43`
- **PRD section/claim**: §1 Overview ("already ships a read path... behind the
  `feature_my_info_blockchain_history` flag... A branch commit says so directly: 'HF001 is bridge +
  read-path only so far'") and §10 Data Flows ("History read: UI → `MyInfoBlockchainHistoryService`
  → integration-bridge `GET /v1/profile-sections/history`").
- **What the code shows**: `MyInfoBlockchainHistoryService`'s own doc comment states verbatim "No
  real write-path hook exists on this branch yet (HF001 is bridge + read-path only so far)" —
  this is the PRD's cited commit language, present in the code itself, not just a commit message.
  The call chain matches exactly: `TransactionHistory.vue` → `BlockchainHistoryRepository.list()` →
  `GET /api/web/my-info/blockchain-history` → `MyInfoController::actionBlockchainHistory` →
  `MyInfoBlockchainHistoryService::list()` → `BlockchainAnchoringService::getTransactionHistory()`
  → `BaseFabricBridgeService::send('/v1/profile-sections/history', 'GET', ...)`. One caveat: this
  code search did **not** find a reference to `feature_my_info_blockchain_history` inside
  `MyInfoController.php` itself or the Vue files under `vue/pages/my-info/history/` — the flag may
  gate the route/menu entry point elsewhere (e.g. routing config or a parent component) rather than
  inside these specific files. The PRD's flag citation could not be directly confirmed from the
  seven files in scope for this reconciliation and should be spot-checked against wherever
  `feature_my_info_blockchain_history` is actually evaluated before this PRD is treated as fully
  verified on that specific point.

### Gap 4 — `BaseFabricBridgeService`/`BlockchainAnchoringService` citations are accurate, but the PRD never surfaces the error-swallowing shape that FR-10/FR-12 will inherit

- **File:line**: `services/fabric/BaseFabricBridgeService.php:35-56` (`send()`)
- **PRD section/claim**: §7 FR-2 ("using `fabric-hris`'s existing gateway-client digest
  construction") and §11 Failure & Consistency table row "Fabric returns a business-rule rejection".
- **What the code shows**: `BaseFabricBridgeService::send()` catches `RequestException | Exception`
  broadly and returns `['status' => 'error', 'detail' => $e->getMessage()]` rather than throwing —
  i.e. every bridge call (read or, presumably, a future write) degrades to a string-keyed array with
  no typed error and no distinction between "bridge unreachable" (should retry per FR-12) and "bridge
  rejected the request as a business-rule violation" (should not endlessly retry, per the existing
  `fabric-hris` sentinel-error pattern the PRD's §11 table already invokes). `MyInfoBlockchainHistoryService::list()`
  correctly handles this today by checking `$response['status'] !== 'ok'`, but this is read-path-only
  code; no equivalent write-path error-classification logic exists yet anywhere in the seven files
  reviewed.
- **Impact on PRD**: FR-12 ("retry with backoff... never silently dropped") should explicitly note
  that a *new* write-path client (extending or replacing `BaseFabricBridgeService::send()`'s
  swallow-and-return-array pattern) needs its own error classification to distinguish retryable
  transport failures from non-retryable business rejections — the existing bridge client, as written,
  does not give the write path that distinction for free just by being "reused."

## Notes on scope

`InformalEducationController` and `/additional-info/index` auth-gap claims in PRD §9 were not
re-verified in this pass (out of the seven cited files); this reconciliation is scoped strictly to
the read/write-path infrastructure claims listed above.

---
title: 'PRD — Talenta HRIS × Hyperledger Fabric Integration'
status: final
created: '2026-08-11'
updated: '2026-08-11 (corrected)'
project: fabric-hris
register: 'real-names (research substrate — see repo confidentiality register in CLAUDE.md)'
---

# PRD: Talenta HRIS × Hyperledger Fabric Integration

## 1. Overview & Vision

Talenta's employee-profile data — across Personal, Employment, Education & Experience,
Additional Info, and Payroll — lives in a conventional relational DB with no cryptographic
integrity guarantee. A privileged insider, a bug, or a change made outside the application
entirely leaves no tamper-evident trail; DB audit logs are themselves mutable.

The `talenta-core` `HF001` branch already ships a **read path**: `BaseFabricBridgeService` →
`BlockchainAnchoringService` → `MyInfoBlockchainHistoryService` → a Vue `TransactionHistory.vue`
view, gated by the `feature_my_info_blockchain_history` config flag (`config/params.php`/
`.env.example`; reconciliation did not pin down exactly where this flag is evaluated in the
controller/routing layer — spot-check before build). It presumes a tamper-evident history
exists. A branch commit says so directly: *"HF001 is bridge + read-path only so far."*

**Correction (found during downstream architecture work, 2026-08-11):** that commit's framing is
only true from `talenta-core`'s vantage point. The **receiving side of the write path already
exists** on the `fabric-hris` side: `integration-bridge/` exposes real `POST
/v1/profile-sections/{PERSONAL,EMPLOYMENT,EDUCATION,ADDITIONAL,PAYROLL}` routes
(`cmd/integrationbridge/main.go`), each calling a fixed hook function
(`UpdatePersonalData`, `ApproveEmploymentTransfer`, `RecordEducationHistory`,
`ApproveFamilyDataChange`, `UpdatePayrollBankAccount`) through `write-path-integration/writepaths`
→ `gatewayclient.SubmitRecordProfileSection` — a real `SubmitTransaction`, not a query. What is
genuinely missing is narrower than originally stated: (a) nothing on `talenta-core`'s side calls
this bridge yet (still true, and still this PRD's core scope); (b) the bridge itself is **purely
synchronous** — no queue, retry, idempotency, or dead-letter handling exists anywhere in it; (c) no
CREATE/UPDATE/DELETE operation-type dispatch exists — each route is hard-wired to exactly one
chaincode hook; (d) its salt/key stores are in-memory only (a bridge restart silently loses every
salt and `employeeKey_i` ever issued — `wiring.go`'s own comment says so directly). This PRD's
scope is therefore: build the missing `talenta-core`-side trigger (FR-1/FR-1a), and treat (b)–(d)
as hardening work on the existing bridge rather than new-component design — not build a
Fabric-facing write path from scratch.

**Vision:** every authorized change to an employee's profile, across all five domains, is anchored
asynchronously as a tamper-evident commitment on the existing permissioned Hyperledger Fabric
ledger (`fabric-hris`). Talenta's database remains the system of record; Fabric holds only salted
digests, never raw PII. Every employee can see their own verified change history. Any change made
outside the application is provably detectable — not blocked in real time, but undeniable once
checked.

## 2. Current State vs. Proposed State

| | Current State | Proposed State |
|---|---|---|
| Write path (Talenta side) | Profile writes hit Talenta's DB only; `EmploymentUpdateWebhookService::publish()` fires a `WebhookWorker` job for unrelated downstream integrations (Kafka, Mekari Flex); nothing calls `integration-bridge` | Same webhook trigger point extended with a new job class (FR-1a) that calls the existing `integration-bridge` write routes |
| Write path (Fabric side) | `integration-bridge` already exposes working, synchronous `POST /v1/profile-sections/{DOMAIN}` routes → `writepaths` → `gatewayclient.SubmitRecordProfileSection` for all 5 domains — built, not missing | Unchanged in shape; hardened with the resilience this PRD requires (queue/retry/dead-letter sit in the new Talenta-side job, since the bridge has none — see FR-1a, FR-12/13) |
| Read path | `TransactionHistory.vue` + `MyInfoBlockchainHistoryService` already query `GET /v1/profile-sections/history` on the integration-bridge | Unchanged in shape — now returns real anchored history instead of an empty/placeholder set |
| Authorization | Yii2 session-based; `hasAccess()` / `checkUserId()`; role + feature-flag gates (Admin, Super Admin, Consultant, Employee) | **Unchanged.** No new synchronous check added to the write path (see §9, decision D1) |
| On-chain data model | `EmployeeProfileRecord` chaincode (12 fields) exists in `fabric-hris`; its `ProfileSection` enum already matches this PRD's 5-domain split 1:1, but no real HRIS field set has been mapped into it yet | This PRD's real field sets (§6) mapped into the existing schema's generic `dataHash`+`profileSection` fields — no schema extension needed |
| Tenant isolation | Channel-per-tenant Fabric topology already ratified (`ADR-0013`) | Reused as-is — this PRD does not redesign Fabric topology |
| Correlation | No generic correlation ID in the Talenta API today | Introduced: Talenta request → webhook job → Fabric tx ID |

## 3. Goals

- **G1** Anchor every authorized write across all 5 domains into Fabric, per employee, per domain.
- **G2** Self-view immutable transaction history for every employee, any role, on their own record.
- **G3** Reuse ratified channel-per-tenant isolation (`ADR-0013`) — no new multi-tenant Fabric
  design in this PRD.
- **G4** Detect out-of-band tampering via hash-chain verification (on-demand + scheduled
  reconciliation).
- **G5** Zero raw PII on-chain — extend `fabric-hris`'s existing JCS+HMAC-SHA256 digest scheme to
  these domains' real fields.
- **G6** Reuse the existing `EmploymentUpdateWebhookService` → `WebhookWorker` hook — no new queue
  infrastructure.
- **G7** Negligible added latency on normal API requests — anchoring is fully asynchronous.
- **G8** Traceability from a Talenta write to its Fabric transaction via a new correlation ID.

**Counter-metrics** (so G7 in particular isn't gamed): p95/p99 API latency for the 5 domains'
write endpoints must not regress vs. pre-integration baseline (see NFR-1); anchoring backlog depth
must not silently grow (see NFR-6).

## 4. Non-Goals

- Replacing Talenta's DB as system of record.
- Any synchronous/blocking Fabric check before a Talenta write is allowed to commit (see §9,
  decision D1 — descoped in favor of detective-only enforcement).
- Cross-employee history visibility beyond today's existing admin permission model.
- `PayrollComponentController`'s ~28 other actions (bulk cross-employee payroll adjustments,
  Compensation Planning) and custom-field *definition* CRUD — explicitly out of scope per the API
  doc itself.
- Second-tenant (`tenant02`) onboarding/scaling automation — `fabric-hris` already flags
  `tenantprovision` as the most dangerous command in that repo and `tenant02` as deliberately
  abandoned; not reopened here.
- Claiming legal GDPR/PDP compliance — this PRD defines architectural controls only (§14).

## 5. Actors

- **Employee (self-service)** — views their own profile and, per this PRD, their own anchored
  history. No new role.
- **Company HR Admin / Super Admin / Consultant** — edits employees within their company per
  existing `hasAccess()` rules; unchanged by this PRD.
- **Platform/Compliance Auditor** (read-only, per `fabric-hris`'s auditor org) — cross-tenant,
  read-only visibility into anchored history for audit purposes; existing Fabric org, not a new
  Talenta role.
- **Blockchain Integration Service** — new/extended service identity that owns the write-path
  anchoring job and the reconciliation job; a Fabric client identity, not a human actor.

## 6. Data Classification

Fields below are extracted from `/Users/chan/Downloads/employee-profile-api-documentation.md`
(the actual documented request/response fields — nothing invented). "On-chain" means a salted
digest per `fabric-hris`'s existing HMAC scheme, never the raw value.

| Domain | Representative fields (real, from API doc) | On-chain treatment |
|---|---|---|
| Personal | Basic info: `full_name, phone, email, birth_date, nik/citizen_id, passport, address`. Family sub-tab: `full_name, relationship, no_ktp, address` (per family member). Emergency Contact sub-tab: `full_name, phone_number, relationship`. | Digest only. `nik`/`citizen_id`, `passport`, and Family's `no_ktp` (a family member's own national-ID number — same sensitivity class) are the highest-sensitivity fields in this domain — never on-chain in any form, hashed with a per-record salt (`fabric-hris` Domain C: `SaltStore` + `EmployeeKeyStore`). Emergency Contact's `phone_number` gets the same digest-only treatment as the employee's own `phone`. |
| Employment | `organization_id, job_id, status_employee, join_date, branch_id, manager, approval_line` | Digest only; `status_employee` already has a masking precedent in Talenta's own API for non-admin roles. |
| Education & Experience | `institution_name, education_degree, majors, certification, filename` | Digest of metadata; any certificate **file** stays in the existing IPFS private-swarm path (`ipfs-cluster`), encrypted under `KEY_EMPLOYEE` — not duplicated on-chain. |
| Additional Info | `custom_field_id, field_name, field_type, value` | Digest only. Field *definitions* (schema) are out of scope (§4); only *values* are anchored. |
| Payroll | `new_salary, npwp, bpjstk, bank_account, bank_account_holder, payment_account[]` | Highest-sensitivity domain. Digest only, salted per-record. `feature_hash_bank_account` (an existing Talenta toggle) suggests the platform already treats bank fields as sensitive — this PRD does not depend on that toggle being on, since nothing bank-related ever leaves Talenta's DB in raw form regardless. |

**Never on-chain, in any domain:** salary amounts, NIK/passport numbers, bank account numbers,
BPJS numbers, phone/email, addresses — consistent with `fabric-hris`'s zero-PII-on-chain design
already verified by its `pilscan` tool.

**Chaincode fit (reconciliation finding):** `EmployeeProfileRecord`'s on-chain schema is exactly 12
fields by ratified design (`fabric-hris` CLAUDE.md) — the domain split above does not require
extending it. `asset.go`'s existing `ProfileSection` enum already ratifies this identical 5-way
split, and the schema's generic `dataHash`+`profileSection` fields accommodate an arbitrary field
set per section without adding columns. The remaining chaincode-design work for
`fabric-engineer`/`fabric-architect` is mapping this PRD's real field sets into digest inputs per
existing section, not schema extension.

## 7. Functional Requirements

### FR Group A — Write-Path Anchoring (Feature F1, F2, F3)

- **FR-1** For every completed write to a Personal/Employment/Education&Experience/Additional
  Info/Payroll endpoint, the system SHALL enqueue an anchoring job. **Important gap found in
  reconciliation** (`reconcile-codebase.md`): today's `EmploymentUpdateWebhookService::publish()`
  call is present in only 7 of `MyInfoController`'s 33 actions. Four in-scope mutations —
  `actionDeletePayroll`, `actionAddComponent` (both Payroll), `actionDeleteEmergencyContact`,
  `actionImportDataEmergencyContact` (both Personal) — persist data but never call `publish()`
  today. FR-1 is therefore **not** satisfied by merely extending existing call sites; it requires
  *adding* new `publish()`-equivalent calls to these four actions as part of this PRD's scope, or
  they will permanently under-anchor regardless of how well reconciliation (FR-11) is implemented.
- **FR-1a** The anchoring enqueue mechanism SHALL reuse the existing Yii2 queue infrastructure
  (`Yii::$app->queue->push(...)`), but **not** ride inside `WebhookWorker::execute()`'s existing
  single-purpose dispatch. Reconciliation found `WebhookWorker` is hard-wired to always end in
  `WebhookRestService::postDataMessage()` (one external webhook receiver) and wraps its whole
  `execute()` in a swallow-and-log-only `catch` with no rethrow, no dead-letter, and no queue-level
  failure signal (`workers/WebhookWorker.php:140-144`) — directly incompatible with FR-12/FR-13's
  retry/dead-letter requirements. This PRD therefore requires a **new job class** on the same Yii2
  queue (satisfying G6's "no new queue infrastructure" at the infra level, since no new broker/queue
  system is introduced) with its own error-handling path, separate from `WebhookWorker`'s
  catch-and-swallow block. This new job's target is the existing `integration-bridge`'s
  `POST /v1/profile-sections/{DOMAIN}` route for that domain (§1 correction) — **all** of
  FR-12/13's retry/dead-letter behavior must live in this new job, because the bridge itself is a
  synchronous request-response service with no queue, retry, or dead-letter handling of its own.
- **FR-2** Each anchoring job SHALL compute a canonical (JCS) serialization and HMAC-SHA256 digest
  of the changed domain's post-write field set, using `fabric-hris`'s existing gateway-client
  digest construction, extended to the real field sets in §6.
- **FR-3** Each anchor SHALL reference the previous digest for that `(employee id, domain)` pair,
  forming a per-employee-per-domain hash chain. For a CREATE operation (no prior anchor exists for
  that pair), the previous-state hash SHALL be a defined genesis sentinel (e.g. an all-zero hash),
  not null or omitted — so FR-10's verification logic has an unambiguous first-entry case rather
  than a special-cased absence.
- **FR-4** Each anchoring transaction SHALL carry: employee `id`, `company_id`, domain, operation
  type (CREATE/UPDATE/DELETE), actor identity, source endpoint, timestamp, previous-state hash,
  new-state hash, and the correlation ID (FR-9). **Design note (reconciliation finding):** the HTTP
  verb of the source endpoint is *not* a reliable signal for operation type — the API doc's own
  "Verb enforcement caveat" states most of these controllers define no `VerbFilter`, so the
  documented method is frontend convention, not server-enforced. Operation type MUST be derived
  from the webhook payload's own action/diff semantics, not the transport-level HTTP verb.
- **FR-5** The anchoring job SHALL be idempotent under retry (duplicate submission of the same
  webhook event must not create a duplicate chain entry).

### FR Group B — History & Query (Feature F4)

- **FR-6** Any employee SHALL be able to view their own anchored transaction history across all 5
  domains via the existing `TransactionHistory.vue` / `MyInfoBlockchainHistoryService` read path,
  regardless of their role (Employee, Admin, Super Admin, Consultant all see their *own* history
  the same way).
- **FR-7** Company HR Admins/Super Admins/Consultants SHALL be able to view any employee's history
  within their own company, per today's existing `hasAccess()` cross-employee permission model —
  no new permission is introduced.
- **FR-8** History views SHALL be filterable by employee, domain, date range, operation type, and
  transaction status. Concretely: a history request scoped to one employee SHALL NOT include
  another employee's field values, digests, or metadata anywhere in its response — including in
  aggregate counts, error messages, or empty-state payloads — regardless of the requester's role.
  This is a boundary condition on top of, not a restatement of, the existing employee-record
  authorization model (§9).

### FR Group C — Traceability (Feature F6)

- **FR-9** The system SHALL introduce a correlation ID at webhook-enqueue time, propagated through
  the queue job, the Fabric submission, and returned in the Fabric tx metadata, so a single ID
  ties a Talenta request to its on-chain result.

### FR Group D — Integrity Verification (Feature F5)

- **FR-10** An on-demand verification action SHALL recompute the current Talenta DB state's digest
  for a given employee+domain and compare it to the latest on-chain anchor; a mismatch SHALL be
  reported as "compromised/inconsistent."
- **FR-11** A scheduled reconciliation job SHALL identify employee changes that exist in Talenta
  but have no corresponding Fabric anchor (missed anchors), and surface them for retry or
  investigation. **Design requirement (closes a critical gap found in adversarial review,
  `review-adversarial-security.md` F1):** reconciliation's determination of "what changed in
  Talenta" MUST be derived independently of the anchoring trigger itself — e.g. from each table's
  own `updated_date`/row-version columns — and NOT from the webhook/queue's own success log. If
  reconciliation instead trusted "the anchoring job says it ran," an attacker with enough DB
  privilege to write directly to the database would, by the same access level, also be able to
  suppress the `publish()` call that triggers anchoring — defeating detection entirely with no
  extra effort. Sourcing "what changed" independently means a suppressed-anchor attack still
  produces a detectable signal: a DB row with a newer `updated_date` than its latest on-chain
  anchor, surfaced as a **missing-anchor finding** — a distinct signal from FR-10's **hash-mismatch**
  finding, and one that also covers a wholly new record that was never anchored at all (no prior
  anchor to hash-compare against, but its very absence is now itself the finding).

### FR Group E — Resilience (Feature F7, F8)

- **FR-12** If Fabric is unreachable when an anchoring job runs, the job SHALL retry with backoff
  and land in a dead-letter state after exhausting retries — never silently dropped.
- **FR-13** A dead-lettered anchoring job SHALL be visible to operators and re-driveable without
  data loss.

## 8. Non-Functional Requirements

- **NFR-1 (Latency)** Adding anchoring MUST NOT measurably regress the p95/p99 response time of
  any of the 5 domains' write endpoints, since anchoring is fully asynchronous (G7). Baseline to
  be captured from current `HF001` branch before/after comparison. `[ASSUMPTION: exact ms target
  not yet set by stakeholder — recommend p95 delta budget of <10ms attributable to the
  enqueue-only call, confirm before build.]`
- **NFR-2 (Anchoring latency)** End-to-end time from webhook enqueue to on-chain commit.
  **Measured dev-environment baseline** (`qa-tests/performance/RESULTS.md`, scenario
  `pt1-diag-20tps-5w`: 5 workers, 20 tps requested, 0 failures): p50 ≈ 23.0s, **p95 ≈ 26.7s, p99 ≈
  27.7s**, max ≈ 28.2s for a single write to commit. This was measured on a single 8 vCPU/16GB
  laptop running the entire Fabric network + IPFS cluster simultaneously (peers already at
  91–128% CPU before load), and the report itself explicitly disclaims these figures as **not
  representative of dedicated production hardware**. Under any real concurrent load (500–2000
  tps in the same report), most requests hit the 60s gRPC deadline — e.g. 7056/7562 failed at the
  500-tps round. **Production target: TBD** — requires a dedicated-hardware benchmark before this
  NFR can be called implementation-ready; the dev-environment number above is a known-bad floor,
  not a target. Because anchoring is fully async (D1/G7), this latency does not block or degrade
  the user-facing Talenta API — it primarily bounds detection-lag (how stale a "latest anchor" can
  be) and backlog depth (NFR-6).
- **NFR-3 (Throughput)** Must sustain the write volume of all 5 domains across all active tenants
  on the currently-provisioned Fabric network (single tenant, `tenant02` abandoned per Non-Goals)
  without unbounded queue growth. **Production target: TBD**, for the same reason as NFR-2 — the
  same report shows the network's real achieved throughput collapsing under its own dev-hardware
  constraints (e.g. 128.9 TPS achieved against a 2000-tps request, with the vast majority of
  requests timing out), so no dev-measured number here should be treated as a capacity ceiling
  either.
- **NFR-4 (Availability)** A Fabric outage MUST NOT block or degrade any Talenta write operation
  (direct consequence of detective-only enforcement, D1).
- **NFR-5 (Data minimization)** Zero raw PII fields listed in §6 SHALL ever appear on-chain;
  verified by extending `fabric-hris`'s existing `pilscan` full-ledger PII scan to the new field
  set as part of acceptance testing.
- **NFR-6 (Backlog observability)** Anchoring queue depth and dead-letter count SHALL be visible
  as monitored metrics, alertable if backlog grows unbounded.
- **NFR-7 (Reconciliation cadence)** The scheduled reconciliation job (FR-11) SHALL run daily for
  Employment, Education & Experience, and Additional Info. For **Personal and Payroll** — the two
  domains §6 classifies as highest-sensitivity, and Payroll is the Phase-1 pilot domain (§12) —
  reconciliation SHALL run **hourly**, per adversarial review's finding that a uniform 24h
  detection-lag was too coarse for the highest-stakes domains.
- **NFR-6a (Backlog threshold)** `[ASSUMPTION: no numeric backlog-depth/growth-rate threshold is
  set yet — deferred to Phase 4 (§12) production hardening, consistent with NFR-6 being a
  monitoring/alerting capability rather than a hard SLA in this PRD's scope.]`

## 9. Authorization & Security Model

**D1 (ratified in this PRD, 2026-08-11):** authorization enforcement is **detective, not
preventive**. Fabric is never consulted synchronously before a Talenta write commits. Talenta's
existing role/feature-flag authorization (`hasAccess()`, `checkUserId()`, company-scoped queries
via `company_id`) continues to gate every write exactly as it does today. Fabric's role is to make
any change — authorized or not — provably part of an immutable history, and to make any *out-of-
band* change (one that bypassed the app's own authorization entirely) detectable via hash-chain
mismatch on verification (FR-10) or reconciliation (FR-11).

This is a deliberate trade-off: it avoids Fabric becoming a blocking single point of failure or a
source of added write-path latency (G7, NFR-4), at the cost of not being able to reject an
unauthorized write in real time — only detect it afterward. `fabric-hris`'s chaincode-side
endorsement policy and MSP identity checks (existing, ratified design) still govern who may submit
an anchoring transaction at all — a compromised or wrong Fabric identity cannot forge an anchor for
a company/employee it doesn't own, even though the Talenta-side authorization decision itself was
already made before Fabric ever saw the request.

**Detection boundary (named explicitly, per adversarial review):** this design detects two distinct
failure classes, and it is important not to overclaim beyond them. (1) **Out-of-band tampering** —
a change made by bypassing Talenta's application entirely (e.g. a direct SQL write) — is caught as
either a hash mismatch (FR-10) or, thanks to FR-11's independent change-detection source, a
missing-anchor finding even for a record with no prior anchor. (2) **In-band but improperly
authorized writes** — a request that goes through Talenta's normal app path but should have been
blocked by Talenta's own authorization logic and wasn't (see the three pre-existing auth gaps
below) — are **not** caught by this design at all. They pass through the normal pipeline and get
faithfully anchored as legitimate history, because from the anchoring pipeline's point of view they
are indistinguishable from any other authorized write. This PRD does not claim otherwise; closing
class (2) requires fixing Talenta's own authorization logic, not anything Fabric can do after the
fact.

**Known pre-existing auth gaps in Talenta** (documented by the API doc extraction, not introduced
by this PRD, but relevant risk context): `InformalEducationController`'s mutating actions skip the
base framework login guard, relying on service-layer checks only; `/additional-info/index` doesn't
verify the target employee belongs to the caller's company; and `POST
/my-info/update-identity-address` does not check `canRequestChangeData` at all — it always writes
directly, bypassing the approval flow an employee would otherwise need for that same Personal-domain
data. This third gap is especially relevant to this PRD's premise, since it means an "authorized
write" in the Personal domain is not always actually approval-gated the way the rest of the domain
is — and per the Detection Boundary above, this specific class of gap is **not** something
anchoring can catch; it gets anchored as legitimate history. Unlike the other two gaps (which are
tracked as risk without blocking this PRD), the `update-identity-address` gap is elevated to a
**launch pre-requisite for the Personal domain specifically**: it should be fixed in Talenta
(reinstating the `canRequestChangeData` check on that endpoint) before this integration's
Personal-domain anchoring can be represented as a meaningful integrity claim, since otherwise the
feature would faithfully immortalize an unauthorized bypass as "verified" history. The other two
gaps (`InformalEducationController`'s missing login guard, `/additional-info/index`'s missing
company-membership check) remain tracked as risk (§13), not blocking, since they are lower-severity
and don't specifically launder an unapproved write as verified.

## 10. Data Flows

- **Write (any of the 5 domains):** Talenta API validates & persists (unchanged) → existing
  `EmploymentUpdateWebhookService::publish()` fires → extended to enqueue a new anchoring job
  (FR-1/FR-1a) → new Talenta-side job (not `WebhookWorker`) calls `integration-bridge`'s existing
  `POST /v1/profile-sections/{DOMAIN}` route → `writepaths` → canonicalize + digest (FR-2,
  existing `gatewayclient` construction) → `gatewayclient.SubmitRecordProfileSection` → chaincode
  commit → tx ID returned synchronously to the Talenta-side job, which logs it against the
  correlation ID (FR-9) and only then considers the job complete (retry/dead-letter, FR-12/13,
  wrap this entire synchronous call from the Talenta-side job, since the bridge itself has none).
- **History read:** UI → `MyInfoBlockchainHistoryService` → integration-bridge
  `GET /v1/profile-sections/history` (existing, unchanged shape) → now backed by real anchors.
- **Integrity verification:** operator/admin action → recompute current-state digest → compare to
  latest anchor (FR-10) → report match/mismatch.
- **Reconciliation:** scheduled job → diff Talenta write log against Fabric anchor log → surface
  gaps (FR-11).

## 11. Failure & Consistency Handling

| Scenario | Behavior |
|---|---|
| Fabric unreachable at write time | Talenta write succeeds normally; anchoring job retries with backoff (FR-12) |
| Anchoring job exhausts retries | Job dead-letters, visible to operators, re-driveable (FR-13) |
| Duplicate webhook delivery | Anchoring job is idempotent per FR-5; no duplicate chain entry |
| Talenta write succeeds, anchor never happens | Caught by reconciliation (FR-11), not silently lost |
| Out-of-band DB tampering | Caught by verification (FR-10) via hash-chain mismatch, not prevented |
| Fabric returns a business-rule rejection (e.g. bad previous-state hash) | `fabric-hris`'s existing string-matched sentinel-error pattern applies (no typed error crosses the Gateway RPC boundary — an established, deliberate limitation, not new to this PRD). **Reconciliation finding:** the existing `BaseFabricBridgeService::send()` on the talenta-core side compounds this — it catches all exceptions and returns a plain `{status:'error', detail:...}` array, with no distinction between "bridge unreachable, retry" and "bridge rejected as a business-rule violation, don't retry." The new write-path client MUST implement its own error classification on top of this; it is not free from reusing the existing bridge call shape. |

## 12. Migration & Rollout

- **Phase 1** — Instrument one domain (recommend Payroll, as the highest-sensitivity domain, even
  though all 5 are in scope for G1 — sequencing which ships first is a build-order choice, not a
  scope cut) end-to-end: webhook hook extension, digest construction, chaincode field mapping,
  history read-path verification against real anchors.
- **Phase 2** — Remaining 4 domains, reusing Phase 1's pipeline.
- **Phase 3** — Integrity verification (FR-10) + scheduled reconciliation (FR-11) across all 5
  domains.
- **Phase 4** — Production hardening: dead-letter operator tooling, observability dashboards,
  `pilscan` extension (NFR-5), load testing against NFR-2/NFR-3 targets.

Feature-flagged behind (an extension of) the existing `feature_my_info_blockchain_history` toggle;
rollback is a flag flip — the write path can be disabled without touching Talenta's DB, since
Fabric is never in the synchronous write path (D1).

## 13. Risks & Open Questions

- **[OPEN]** NFR-2/NFR-3 numeric targets need a stakeholder-confirmed number or a measured baseline
  from `qa-tests/performance` before this PRD can be called implementation-ready on performance.
- **[RISK]** Pre-existing Talenta auth gaps (§9) are outside this PRD's fix scope but should be
  tracked as a linked defect, since they weaken the "authorized write" assumption this whole
  design rests on.
- **[RISK]** `integration-bridge`'s salt/key stores (`keystore.InMemory*Store`) are in-memory only
  — `wiring.go` states directly that a process restart silently loses every salt and
  `employeeKey_i` ever issued. This is a pre-existing gap in the component this PRD's write path
  depends on, not introduced by this PRD, but it must be resolved (a persistent store
  implementation) before this integration can be trusted in production — an unrecoverable key loss
  breaks digest verification for every record anchored before the restart.
- **[OPEN]** No CREATE/UPDATE/DELETE operation-type dispatch exists in `integration-bridge` today
  — each of its 5 routes calls exactly one fixed chaincode hook regardless of what kind of change
  occurred. FR-4's operation-type field (already specified as coming from webhook payload
  semantics, not the HTTP verb) needs a concrete carrier: either the new Talenta-side job passes
  operation type as a field in its POST body for the bridge to forward into the anchor, or the
  bridge infers it from `newValue`'s shape. Not decided here — a real design question for the
  architecture session.
- **[OPEN]** `integration-bridge`'s `ADDITIONAL` domain route calls a hook named
  `ApproveFamilyDataChange` — that name describes Personal-domain Family data (§6), not Additional
  Info custom fields. Either the hook is misnamed, or `ADDITIONAL` and Personal/Family are mapped
  differently in the bridge than in this PRD's §6 domain split. Needs verification against the
  bridge's actual chaincode call before the architecture session finalizes the domain-to-route
  mapping.
- **[RISK]** The new "Blockchain Integration Service" identity (§5) is a Fabric client identity
  whose key material this PRD does not analyze for isolation from the same privileged-attacker
  class assumed elsewhere in §9 (e.g. an insider with DB access). If that same insider can also
  read this service's runtime credentials, digest unforgeability is not actually guaranteed by
  this design alone — it would rely entirely on `fabric-hris`'s existing MSP/key-management
  posture (Domain A, outside this workspace). Flagged for the `security-architect` agent's
  downstream threat model, not resolved here.
- **[RESOLVED]** FR-10's behavior for a legitimately deleted record: when the latest anchor for an
  employee+domain pair has operation type DELETE, verification SHALL report "deletion verified"
  and stop — it MUST NOT attempt to hash-compare against current DB state, since the record no
  longer exists by design. Any tampering after a legitimate delete (e.g. the row reappearing) is
  caught as a new missing-anchor finding via FR-11, not as a hash mismatch.

## 14. Data Privacy

- On-chain data is limited to salted digests (§6) — no raw PII, by construction and by the existing
  `pilscan` verification tool, extended in scope (NFR-5).
- Off-chain document material (certificates, §6 Education) uses the existing IPFS
  private-swarm + `KEY_EMPLOYEE` encryption path — not duplicated here.
- Right-to-delete: a hash-chain is intentionally difficult to erase. This PRD does not claim
  deletion of anchored digests is possible; any "delete" in Talenta's DB removes the raw record but
  the historical digest chain remains as evidence a record existed and changed — this is a known,
  disclosed limitation, not a compliance claim (per Non-Goals, §4).

## 15. Acceptance Criteria

**AC-1 — Authorized update anchored**
Given an authorized HR Admin in Company A updates Employee A's payroll bank account,
When the write commits in Talenta,
Then an anchoring job is enqueued and eventually commits to Fabric,
And the change is visible in Employee A's transaction history.

**AC-2 — Cross-company isolation preserved**
Given a user belongs to Company A,
When they attempt to modify Employee B in Company B (already rejected by Talenta's existing auth),
Then no Talenta write occurs,
And no anchoring job is ever enqueued for that attempt.

**AC-3 — Self-view history, any role**
Given any employee, regardless of role,
When they open their own transaction history,
Then they see their own anchored history without needing an elevated role.

**AC-4 — Out-of-band tampering detected**
Given Employee A's payroll data is modified directly in the DB, bypassing Talenta's app entirely,
When integrity verification (FR-10) runs against that employee+domain,
Then a hash-chain mismatch is reported as compromised/inconsistent.

**AC-5 — Fabric unavailable does not block writes**
Given Talenta successfully processes an employee update,
When Fabric is temporarily unavailable,
Then the Talenta write still succeeds,
And the anchoring job retries and eventually succeeds once Fabric recovers,
And no anchoring event is silently lost (reconciliation would catch it if it were).

**AC-6 — Missed anchor caught by reconciliation**
Given an anchoring job was dropped due to an unrecovered failure,
When the scheduled reconciliation job runs,
Then the missing anchor is surfaced for retry/investigation.

**AC-7 — History view never leaks another employee's data**
Given Employee A requests their own transaction history,
When the response is returned, including any error or empty-state case,
Then it contains no field values, digests, or metadata belonging to any employee other than
Employee A.

**AC-8 — Suppressed anchor still detected**
Given an attacker with direct DB write access modifies Employee A's payroll record AND suppresses
the anchoring enqueue call for that write,
When the scheduled reconciliation job (FR-11) next runs,
Then the record's DB `updated_date` is newer than its latest on-chain anchor,
And this is surfaced as a missing-anchor finding, even though no hash-chain entry exists to compare
against.

## 16. Glossary

- **Domain** — one of the five profile-data groupings this PRD anchors: Personal, Employment,
  Education & Experience, Additional Info, Payroll (§6). Matches `asset.go`'s existing
  `ProfileSection` enum one-to-one (§13).
- **Anchor** (noun) — one on-chain transaction recording a digest of a domain's post-write state
  for one employee; (verb) — the act of submitting that transaction to Fabric.
- **Digest** — a JCS-canonicalized, salted HMAC-SHA256 hash of a domain's field set (§6, FR-2) —
  never the raw field values themselves.
- **Correlation ID** — the new identifier (FR-9) tying one Talenta write request to its queue job
  and its resulting Fabric transaction ID.
- **Reconciliation** — the scheduled job (FR-11, NFR-7) that independently re-derives "what changed
  in Talenta" from DB row timestamps (not from the anchoring trigger's own success log) and
  compares it against what was actually anchored, surfacing missing-anchor findings.
- **Dead-letter** — the terminal state of an anchoring job that exhausted its retries (FR-12/13);
  visible to operators and re-driveable without data loss.


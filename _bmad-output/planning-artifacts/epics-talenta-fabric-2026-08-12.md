---
stepsCompleted: [1, 'requirements-confirmed', 2, 'epics-approved', 3, 'stories-complete']
inputDocuments:
  - '_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-11/prd.md'
  - '_bmad-output/planning-artifacts/architecture/architecture-fabric-hris-talenta-2026-08-11/ARCHITECTURE-SPINE.md'
  - '_bmad-output/planning-artifacts/architecture/architecture-fabric-hris-talenta-2026-08-11/SOLUTION-DESIGN.md'
---

# Talenta HRIS x Hyperledger Fabric Integration - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for the Talenta HRIS x Hyperledger
Fabric write-path integration, decomposing the requirements from `prd-fabric-hris-2026-08-11` and
the architecture decisions in `architecture-fabric-hris-talenta-2026-08-11/ARCHITECTURE-SPINE.md`
into implementable stories. No UX design contract exists for this integration (the only UX run in
this repo belongs to the unrelated Caliper dashboard feature) — no UX-DRs are extracted.

## Requirements Inventory

### Functional Requirements

FR-1: For every completed write to a Personal/Employment/Education&Experience/Additional Info/Payroll endpoint, the system SHALL enqueue an anchoring job. Four currently-unwired MyInfoController actions (actionDeletePayroll, actionAddComponent, actionDeleteEmergencyContact, actionImportDataEmergencyContact) require NEW publish()-equivalent calls, not extension of existing ones.
FR-1a: The anchoring enqueue mechanism SHALL reuse existing Yii2 queue infrastructure via a NEW job class (not WebhookWorker, which is hard-wired to one external webhook receiver and swallows errors with no retry/dead-letter). This new job calls integration-bridge's existing POST /v1/profile-sections/{DOMAIN} route; all retry/dead-letter logic (FR-12/13) lives here, since the bridge itself has none.
FR-2: Each anchoring job SHALL compute a canonical (JCS) serialization and HMAC-SHA256 digest of the changed domain's post-write field set, using fabric-hris's existing gateway-client digest construction.
FR-3: Each anchor SHALL reference the previous digest for that (employee, domain, recordIdentity) pseudonym, forming an independent hash chain per pseudonym. CREATE anchors SHALL use a defined genesis sentinel (all-zero hash) as previous-state hash, never null/omitted.
FR-4: Each anchoring transaction SHALL carry: employee id, company_id, domain, operation type (CREATE/UPDATE/DELETE), actor identity, source endpoint, timestamp, previous-state hash, new-state hash, and correlation ID. Operation type MUST be derived from webhook payload/diff semantics, not the HTTP verb (verbs are frontend convention, mostly unenforced).
FR-5: The anchoring job SHALL be idempotent under retry (duplicate submission of the same webhook event must not create a duplicate chain entry). Dedup enforcement ownership belongs to the Talenta-side job (integration-bridge is stateless and has no lookup-by-ID store); backstopped by the chaincode's own no-op-on-identical-dataHash behavior.
FR-6: Any employee SHALL be able to view their own anchored transaction history across all 5 domains via the existing read path, regardless of role.
FR-7: Company HR Admins/Super Admins/Consultants SHALL be able to view any employee's history within their own company, per today's existing hasAccess() permission model.
FR-8: History views SHALL be filterable by employee, domain, date range, operation type, and transaction status, and SHALL NOT expose another employee's field values/digests/metadata anywhere in the response, including error/empty-state cases.
FR-9: The system SHALL introduce a correlation ID at webhook-enqueue time, propagated through the queue job, the Fabric submission, and returned in the Fabric tx metadata.
FR-10: An on-demand verification action SHALL recompute the current Talenta DB state's digest for a given employee+domain+recordIdentity and compare it to the latest on-chain anchor; a mismatch SHALL be reported as compromised/inconsistent. A latest anchor with operation type DELETE SHALL report "deletion verified" and stop, never hash-comparing against an absent DB row.
FR-11: A scheduled reconciliation job SHALL identify employee changes that exist in Talenta but have no corresponding Fabric anchor (missed anchors), and surface them for retry/investigation. Its determination of "what changed" MUST be derived independently of the anchoring trigger itself (DB updated_date/row-version columns, not the webhook/queue's own success log) so a suppressed-anchor attack still produces a detectable missing-anchor finding.
FR-12: If Fabric is unreachable when an anchoring job runs, the job SHALL retry with backoff and land in a dead-letter state after exhausting retries — never silently dropped.
FR-13: A dead-lettered anchoring job SHALL be visible to operators and re-driveable without data loss, backed by a persisted, queryable record (payload, correlation ID, failure reason, attempt count) surviving process restarts.
FR-Arch-1 (AD-3): recordIdentity for multi-record domains (Family sub-records under PERSONAL, Additional Info custom fields under ADDITIONAL) SHALL be encoded into a distinct employeeID pseudonym per (realEmployeeID, recordIdentity) pair, derived by integration-bridge — never as a new chaincode field, since RecordProfileSection's signature and composite key are fixed and unchanged.
FR-Arch-2 (AD-4): integration-bridge's POST /v1/profile-sections/ADDITIONAL route SHALL dispatch to a NEW writepaths.Hooks method for real Additional Info custom-field values, not the existing ApproveFamilyDataChange (which is genuinely Family/dependents data mislabeled under ADDITIONAL). Family-data writes SHALL anchor under profileSection=PERSONAL instead.

### NonFunctional Requirements

NFR-1: Adding anchoring MUST NOT measurably regress p95/p99 response time of any of the 5 domains' write endpoints, since anchoring is fully asynchronous. [ASSUMPTION: p95 delta budget <10ms for the enqueue-only call — confirm before build.]
NFR-2: End-to-end anchoring latency (webhook enqueue to on-chain commit) production target is TBD pending a dedicated-hardware Caliper benchmark — the only measured baseline (p95 ~26.7s, single-write, dev laptop hardware) is explicitly disclaimed as non-representative and must not be treated as a target.
NFR-3: Throughput must sustain the write volume of all 5 domains on the currently-provisioned Fabric network without unbounded queue growth. Production target is TBD for the same reason as NFR-2.
NFR-4: A Fabric outage MUST NOT block or degrade any Talenta write operation (direct consequence of detective-only enforcement).
NFR-5: Zero raw PII fields (nik/citizen_id, passport, no_ktp, salary, bank details, BPJS numbers, phone/email, addresses) SHALL ever appear on-chain; verified by extending fabric-hris's existing pilscan full-ledger PII scan to the new field set.
NFR-6: Anchoring queue depth and dead-letter count SHALL be visible as monitored metrics, alertable if backlog grows unbounded. [ASSUMPTION: no numeric backlog-depth/growth-rate threshold set yet — deferred to production-hardening phase.]
NFR-7: Scheduled reconciliation SHALL run hourly for Personal and Payroll (highest-sensitivity domains), daily for Employment, Education & Experience, and Additional Info.

### Additional Requirements

- Personal-domain launch gate: `POST /my-info/update-identity-address` must have its missing `canRequestChangeData` check reinstated in Talenta BEFORE Personal-domain anchoring can be represented as a meaningful integrity claim — otherwise an unapproved bypass gets faithfully anchored as "verified" history. This is a cross-cutting rollout dependency, not a story in its own right, but blocks any Personal-domain story from being marked done for production.
- `integration-bridge`'s salt/key stores (`keystore.InMemory*Store`) are in-memory only — a process restart silently loses every salt and `employeeKey_i` ever issued. A persistent store implementation is required before production trust, independent of new anchoring logic.
- Wire-format conventions (from architecture spine Consistency Conventions): cross-boundary request body extends `{employeeInternalID, userID, newValue, document}` with `correlationID`, `operationType` (uppercase CREATE/UPDATE/DELETE), `recordIdentity` (bare string, unprefixed), `companyId`, and `sourceEndpoint`. `operationType`/`sourceEndpoint` are off-chain metadata only (no chaincode field exists for them) — logged in `integration-bridge`'s own request log alongside the correlation ID.
- No chaincode/schema changes are in scope anywhere in this work — `RecordProfileSection`'s signature and the `(employeeID, profileSection)` composite key stay exactly as they are today (ratified design, changing them requires separate approval per fabric-hris CLAUDE.md).
- Deferred (explicitly out of this epic set per the architecture spine, needing their own future design pass): exact idempotency dedup mechanism on the Talenta-side job; whether `writepaths.OperationalStore.SaveSection` is idempotent on identical input; reconciliation job's concrete placement (only its cadence and independent-source constraint are fixed); dead-letter persistence data model (only the requirement is fixed, not the schema); operator-facing dead-letter UI/tooling.

### UX Design Requirements

Not applicable — no UX design contract exists for this integration.

### FR Coverage Map

FR-1: Epic 1 (Payroll's 2 unwired actions) + Epic 2 (remaining 4 unwired actions) - new publish()-equivalent trigger calls
FR-1a: Epic 1 - new async job class calling integration-bridge, with retry/dead-letter
FR-2: Epic 1 - digest construction, reused as-is by Epic 2
FR-3: Epic 1 - hash-chain/genesis sentinel, reused as-is by Epic 2
FR-4: Epic 1 - anchoring transaction fields incl. operation-type derivation
FR-5: Epic 1 - idempotency under retry
FR-6: Epic 1 - self-view history (existing read path, verified against real anchors)
FR-7: Epic 1 - admin cross-employee history view (existing permission model, verified)
FR-8: Epic 1 - history non-disclosure boundary
FR-9: Epic 1 - correlation ID
FR-10: Epic 3 - on-demand verification incl. DELETE-record behavior
FR-11: Epic 3 - scheduled reconciliation, independent change-detection source
FR-12: Epic 1 - retry with backoff, dead-letter on exhaustion
FR-13: Epic 1 (mechanism) + Epic 4 (persisted data model/operator visibility)
FR-Arch-1: Epic 1 (trivial `self` case) + Epic 2 (Family/Additional-Info pseudonym cases)
FR-Arch-2: Epic 2 - ADDITIONAL/Family route + hook fix
NFR-1: Epic 1 - latency non-regression
NFR-2: Epic 4 - real throughput/latency benchmark, replacing disclaimed dev baseline
NFR-3: Epic 4 - same as NFR-2
NFR-4: Epic 1 - availability (Fabric outage never blocks Talenta writes)
NFR-5: Epic 1 - PII minimization verification (pilscan extension)
NFR-6: Epic 1 (metrics exist) + Epic 4 (numeric threshold)
NFR-7: Epic 3 - reconciliation cadence (hourly Personal/Payroll, daily others)
Additional — Personal-domain launch gate: Epic 2 (blocking dependency within the Personal-domain story)
Additional — persistent key store: Epic 4
Additional — wire-format conventions: Epic 1 (established), reused by Epic 2/3/4
Additional — no chaincode/schema changes: cross-cutting constraint, not a story; enforced by review in every epic
Additional — Deferred items (dedup mechanism, OperationalStore idempotency, reconciliation placement, dead-letter data model, operator UI): Epic 3/4 as applicable, each its own design decision at story time

## Epic List

### Epic 1: Payroll Write-Path Anchoring (pilot)
Payroll changes get anchored to Fabric end-to-end and show up correctly in the employee's transaction history — the full pipeline proven on the highest-sensitivity domain first, so any pipeline-level mistake surfaces before being replicated 4x.
**FRs covered:** FR-1 (Payroll's 2 unwired actions), FR-1a, FR-2, FR-3, FR-4, FR-5, FR-6, FR-7, FR-8, FR-9, FR-12, FR-13 (mechanism), FR-Arch-1 (trivial case), NFR-1, NFR-4, NFR-5, NFR-6 (metrics)

### Story 1.1: Reliably trigger every Payroll write toward Fabric anchoring

As an HR Admin,
I want every Payroll change I make to reliably start the anchoring process,
So that no Payroll change is ever silently skipped, even if Fabric is temporarily down.

**Acceptance Criteria:**

**Given** an HR Admin calls `actionUpdatePayroll`
**When** the write commits in Talenta
**Then** the existing `EmploymentUpdateWebhookService::publish()` call also enqueues a new anchoring job on the new job class
**And** the Talenta write itself completes with no added blocking latency (FR-1a, NFR-4)

**Given** an HR Admin calls `actionDeletePayroll` or `actionAddComponent`
**When** the write commits in Talenta
**Then** a new `publish()`-equivalent call (not present in either action today) enqueues a new anchoring job — FR-1's two currently-unwired Payroll actions no longer silently skip anchoring

**Given** Fabric is temporarily unreachable when the anchoring job runs
**When** the job attempts to call `integration-bridge`
**Then** the job retries with backoff and does not block or fail the original Talenta write, which already committed independently
**And** after exhausting retries the job lands in a dead-letter state rather than being silently dropped (FR-12)

### Story 1.2: Compute and submit a tamper-evident digest for the Payroll anchor

As the anchoring job,
I want to canonicalize and hash the changed Payroll data and submit it to the existing bridge,
So that the change is provably fingerprinted on the ledger without any raw payroll data ever leaving Talenta.

**Acceptance Criteria:**

**Given** a Payroll anchoring job has the post-write field set (`new_salary`, `npwp`, `bpjstk`, `bank_account`, `bank_account_holder`, `payment_account[]`)
**When** the job computes the digest
**Then** it uses `fabric-hris`'s existing JCS-canonicalization + salted HMAC-SHA256 construction, and the raw field values are never transmitted to `integration-bridge` or written on-chain (NFR-5)

**Given** this is the first-ever anchor for a given employee's Payroll record
**When** the anchor is submitted
**Then** the previous-state hash is an explicit all-zero genesis sentinel, never null or omitted (FR-3)

**Given** Payroll is a single-record-per-employee domain
**When** the pseudonym is derived for this anchor
**Then** it pseudonymizes on `employeeID` alone (`recordIdentity = self`), per AD-3's trivial case (FR-Arch-1)

**Given** a full-ledger PII scan (`pilscan`) is run after this anchor exists
**When** the scan checks the new Payroll field set
**Then** it finds zero raw PII on-chain (NFR-5)

### Story 1.3: Carry full anchor metadata and a traceable correlation ID

As an auditor,
I want every anchoring transaction to carry who/what/when/which-endpoint plus a correlation ID,
So that I can trace any anchored change back to the exact Talenta request that produced it.

**Acceptance Criteria:**

**Given** a Payroll anchoring job submits its transaction
**When** the anchor lands
**Then** it carries employee `id`, `company_id`, domain, operation type, actor identity, source endpoint, timestamp, previous-state hash, new-state hash, and a correlation ID (FR-4)

**Given** operation type must be determined
**When** the anchoring job derives it
**Then** it comes from the webhook payload's own action/diff semantics, never from the HTTP verb of the source endpoint (FR-4's design note)

**Given** a correlation ID is generated at enqueue time
**When** the job is retried or eventually succeeds
**Then** the same correlation ID is propagated through the queue job, the Fabric submission, and the resulting transaction metadata, so one ID ties the original Talenta request to its on-chain result (FR-9)

### Story 1.4: Make retried anchoring attempts safe

As the anchoring job,
I want a retried submission of the same logical write to never create a duplicate ledger entry,
So that transient failures don't corrupt an employee's history.

**Acceptance Criteria:**

**Given** the same anchoring job is submitted twice (e.g. due to a retry after an ambiguous timeout)
**When** both submissions reach the ledger
**Then** no duplicate chain entry is created (FR-5)

**Given** the chaincode's own `RecordProfileSection` is a verified no-op on an identical `(dataHash, prevHash)` pair
**When** a retried submission carries the same digest as an already-committed anchor
**Then** the retry is a safe no-op on-chain, backstopping this story even before any Talenta-side dedup mechanism exists

### Story 1.5: Show real anchored Payroll history, safely scoped per employee

As an employee,
I want to see my own verified Payroll change history,
So that I can confirm my data hasn't been tampered with — and as an HR Admin, I want to see it for employees in my company.

**Acceptance Criteria:**

**Given** any employee, regardless of role
**When** they open their own Payroll transaction history
**Then** they see their own real anchored history via the existing `TransactionHistory.vue`/`MyInfoBlockchainHistoryService` read path, with no elevated role required (FR-6)

**Given** an HR Admin, Super Admin, or Consultant
**When** they view another employee's Payroll history within their own company
**Then** they see it, per today's existing `hasAccess()` permission model — no new permission is introduced (FR-7)

**Given** an employee requests their own Payroll transaction history
**When** the response is returned, including any error or empty-state case
**Then** it contains no field values, digests, or metadata belonging to any other employee (FR-8, AC-7)

### Story 1.6: Make the anchoring backlog observable and prove no latency regression

As an operator,
I want to see queue depth and dead-letter counts, and confirm Payroll edits are exactly as fast as before,
So that I can trust this shipped without silently degrading the product.

**Acceptance Criteria:**

**Given** the new anchoring job is deployed
**When** jobs are queued, retried, or dead-lettered
**Then** queue depth and dead-letter count are visible as monitored metrics (NFR-6)

**Given** a Payroll write endpoint's p95/p99 response time before this integration
**When** the same endpoint is measured after this integration ships
**Then** there is no measurable regression, since anchoring is fully asynchronous (NFR-1)

### Epic 2: Remaining Domains — Personal, Employment, Education & Experience, Additional Info
The other 4 domains reuse Epic 1's proven pipeline, including the domain-taxonomy fix (Family moves under Personal; Additional Info gets its own real implementation). Personal-domain anchoring is blocked on Talenta's `canRequestChangeData` fix landing first. Education & Experience and Additional Info live in controllers never audited for an existing anchoring hook — their wiring is genuinely new, not an extension.

**Scope split (added 2026-08-12, during Story 2.1's story-creation research):** the per-record
pseudonym mechanism AD-3 requires (Family sub-records, Additional Info custom fields each chaining
independently) does not exist anywhere in the stack today — not in `keystore.EmployeeKeyStore`, not
in `gatewayclient.ComputeEmployeeID`, not in `integration-bridge`'s request envelope, not in
`writepaths`. All four are keyed strictly on `employeeInternalID` alone. Building this is
foundational, cross-repo (`fabric-hris`) infrastructure work, not "wiring two Talenta actions" —
too large and too different in kind to bundle into Story 2.1 as originally scoped. Split into a new
**Story 2.0**, which Story 2.1 (and later Story 2.4) now depend on as a completed prerequisite,
not a peer.

**FRs covered:** FR-1 (remaining unwired actions), FR-Arch-1 (Family/Additional-Info cases), FR-Arch-2, Personal-domain launch gate (blocking dependency)

### Story 2.0: Add per-`recordIdentity` pseudonym derivation to the Fabric-side write path

As the architecture's own tamper-evidence guarantee (AD-3),
I want each family member's (and later, each custom field's) data to chain independently under one shared employee+domain,
so that a change to one family member's record cannot be confused with, or hide behind, another's on-chain history.

**Acceptance Criteria:**

**Given** `keystore.EmployeeKeyStore.GetOrCreateEmployeeKey(ctx, employeeInternalID)` (`write-path-integration/keystore/keystore.go:79-83`) currently takes only an employee ID
**When** this story is implemented
**Then** it (or a new sibling method) accepts an optional `recordIdentity` string, and derives a **distinct** key per `(employeeInternalID, recordIdentity)` pair — `recordIdentity=""` or absent MUST resolve identically to today's existing single-record behavior, so Payroll/Employment/Education (Epic 1, Story 2.2, Story 2.3) are unaffected

**Given** `gatewayclient.ComputeEmployeeID(employeeKeyI []byte)` (`write-path-integration/gateway-client/digestbuilder.go:80-87`) currently HMACs a fixed message `"id"`
**When** this story is implemented
**Then** the distinct key produced above (already unique per `recordIdentity`) is sufficient — `ComputeEmployeeID` itself does not need to change, since it already derives purely from whatever key bytes it's given; the uniqueness lives in key derivation (previous AC), not here — confirm this by tracing the call chain before writing new code, not by assumption

**Given** `integration-bridge/internal/pipeline/validate.go`'s `requestEnvelope` (`:16-21`) has no `recordIdentity` field today
**When** this story is implemented
**Then** it gains an optional `RecordIdentity string` json field (`recordIdentity`), validated as an ordinary string (no new shape requirement beyond what `isJSONObject`-style checks already do for other fields), and threaded through to the pseudonym-resolution call this story updates

**Given** `write-path-integration/writepaths/writepaths.go`'s `Hooks` methods (e.g. `UpdatePersonalData`, `:259-265`) currently call `h.Store.SaveSection(ctx, employeeInternalID, "PERSONAL", newValue)` and `h.anchor(...)` with no `recordIdentity` parameter
**When** this story is implemented
**Then** the `anchor`/`doAnchor` call chain (`:174`, `:205`) accepts and threads an optional `recordIdentity` through to the pseudonym derivation from the first AC — existing callers passing none continue to resolve exactly as before (regression safety for Epic 1/Story 2.2/2.3)

**Given** the read path (`integration-bridge/internal/pipeline/history.go:142-147`) already resolves `employeeInternalID` → pseudonym via `GetOrCreateEmployeeKey` + `ComputeEmployeeID`
**When** this story is implemented
**Then** the write path's new `recordIdentity`-aware resolution uses the exact same two-call pattern (AD-8's "resolve the pseudonym identically to how the write path does" cuts both ways — history's read-side resolution and this story's write-side resolution must not diverge in shape)

### Story 2.1: Wire Personal domain's remaining actions and fix the Family/Additional-Info route mapping

**Depends on Story 2.0 being complete** — this story is the first real caller of the `recordIdentity`
mechanism Story 2.0 builds; it does not build that mechanism itself.

As an employee,
I want changes to my emergency contacts and family data to be anchored too, correctly filed under Personal,
So that my complete Personal-domain history is tamper-evident, not just my Basic Info.

**Acceptance Criteria:**

**Given** an employee calls `actionDeleteEmergencyContact` or `actionImportDataEmergencyContact`
**When** the write commits in Talenta
**Then** a new trigger call (neither action has one today) enqueues an anchoring job on the same job class built in Epic 1 (FR-1)

**Given** a family member's data changes (e.g. `actionSaveFamilyData`, `actionDeleteFamilyData`)
**When** the anchoring job derives the pseudonym
**Then** it is tagged `profileSection=PERSONAL` (not `ADDITIONAL`) and pseudonymized on `(employeeID, familyMemberReferenceID)`, so each family member's history chains independently (FR-Arch-1, FR-Arch-2)

**Given** the Personal domain's `update-identity-address` endpoint currently skips its `canRequestChangeData` approval check
**When** this story is evaluated for production readiness
**Then** it is NOT marked done for production until that Talenta-side check is reinstated — anchoring an unapproved bypass as "verified" history would undermine this integration's core promise for the Personal domain (PRD §9 launch pre-requisite)

### Story 2.2: Wire Employment domain into the anchoring pipeline

As an HR Admin,
I want Employment changes anchored the same way Payroll's are,
So that Employment history gets the same tamper-evidence guarantee, reusing the pipeline Epic 1 already proved.

**Acceptance Criteria:**

**Given** an HR Admin calls `actionUpdateEmploymentData`
**When** the write commits in Talenta
**Then** the existing `publish()` call (already present, unlike Payroll's gaps) also enqueues an anchoring job through the same pipeline built in Epic 1 — digest construction, metadata, idempotency, and dead-letter handling are reused unmodified

**Given** Employment is a single-record-per-employee domain
**When** the pseudonym is derived
**Then** it uses `recordIdentity = self`, identical to Payroll's case (FR-Arch-1)

**Given** an employee or HR Admin views Employment transaction history
**When** the request is made
**Then** the same FR-6/FR-7/FR-8 guarantees Epic 1 proved for Payroll (self-view, admin cross-employee view, non-disclosure boundary) hold for Employment too

### Story 2.3: Wire Education & Experience into the anchoring pipeline (new trigger point)

As an employee,
I want my education, certification, and work-experience changes anchored,
So that this domain gets the same tamper-evidence guarantee, even though it lives in controllers that have never had an anchoring hook.

**Acceptance Criteria:**

**Given** `FormalEducationController`, `InformalEducationController`, and `WorkingExperienceController` have no existing `publish()`-equivalent call today
**When** this story is implemented
**Then** a new trigger call is added to each of their mutating actions (`save`, `delete`, `update`, `upload-certificate`, `import`), enqueuing an anchoring job through Epic 1's pipeline — this is new wiring, not an extension of an existing hook

**Given** an Education & Experience write commits
**When** the anchoring job runs
**Then** it is tagged `profileSection=EDUCATION` using `recordIdentity = self` (aggregate per-employee record, per §6), and any certificate file stays in the existing IPFS path — never duplicated on-chain

**Given** an employee or HR Admin views Education & Experience transaction history
**When** the request is made
**Then** the same FR-6/FR-7/FR-8 guarantees hold for this domain too

### Story 2.4: Implement real Additional Info anchoring under the corrected route

As an auditor,
I want custom-field value changes anchored under their own real Additional Info implementation,
So that Additional Info history is genuinely custom-field data, not accidentally the mislabeled Family-data hook.

**Acceptance Criteria:**

**Given** `AdditionalInfoController` has no existing `publish()`-equivalent call today
**When** this story is implemented
**Then** a new trigger call is added to its save action, enqueuing an anchoring job through Epic 1's pipeline

**Given** `integration-bridge`'s `POST /v1/profile-sections/ADDITIONAL` route currently dispatches to `ApproveFamilyDataChange` (a mislabeled Family-data hook)
**When** this story is implemented
**Then** the route dispatches to a new `writepaths.Hooks` method built for real Additional Info custom-field values instead (FR-Arch-2)

**Given** each custom field can be added/removed independently
**When** the pseudonym is derived
**Then** it is `(employeeID, customFieldID)`, so each custom field's history chains independently of the others (FR-Arch-1)

**Given** an employee or HR Admin views Additional Info transaction history
**When** the request is made
**Then** the same FR-6/FR-7/FR-8 guarantees hold for this domain too

### Epic 3: Integrity Verification & Reconciliation
Administrators can check any employee record's integrity on demand, and a scheduled sweep proactively catches anchors that silently never happened — closing the "detective" half of the design across whatever domains are live.
**FRs covered:** FR-10, FR-11, NFR-7

### Story 3.1: On-demand integrity verification

As a compliance auditor,
I want to check any employee record's current data against its latest anchor on demand,
So that I can confirm data integrity or detect tampering immediately when asked.

**Acceptance Criteria:**

**Given** an employee's data in Talenta's DB has been modified outside the authorized workflow (out-of-band tampering)
**When** verification runs against that employee+domain+recordIdentity
**Then** it recomputes the current digest, compares it to the latest on-chain anchor at that same pseudonym, and reports a hash mismatch as "compromised/inconsistent" (FR-10, AC-4)

**Given** the pseudonym resolution for a given employee+domain+recordIdentity
**When** verification runs
**Then** it resolves the pseudonym identically to how the write path (AD-3) does — never via a second, independent derivation (AD-8)

**Given** the latest anchor for an employee+domain+recordIdentity has operation type DELETE
**When** verification runs against that pseudonym
**Then** it reports "deletion verified" and stops — it does NOT attempt to hash-compare against a DB row that no longer exists (FR-10's `[RESOLVED]` deletion behavior)

**Given** a record that reappears in the DB after a legitimate delete (tampering after deletion)
**When** the next reconciliation sweep runs (Story 3.2)
**Then** this is caught as a new missing-anchor finding, not by on-demand verification's hash-mismatch path

### Story 3.2: Scheduled reconciliation with an independent, suppression-resistant change-detection source

As a security operator,
I want a scheduled sweep to catch anchors that silently never happened,
So that even an attacker who suppresses the anchoring trigger itself still gets caught.

**Acceptance Criteria:**

**Given** an anchoring job was dropped due to an unrecovered failure, or an attacker with direct DB access suppressed the anchoring trigger entirely
**When** the scheduled reconciliation job runs
**Then** it determines "what changed in Talenta" from the DB's own `updated_date`/row-version columns — never from the anchoring trigger's own success/failure log — so a suppressed trigger does not also blind reconciliation (FR-11, AC-8)

**Given** a DB row's `updated_date` is newer than its latest on-chain anchor for that pseudonym
**When** reconciliation compares them
**Then** this is surfaced as a missing-anchor finding for retry or investigation, distinct from Story 3.1's hash-mismatch finding — and this works even for a wholly new record that was never anchored at all, since its absence is itself the finding

**Given** Personal and Payroll are the highest-sensitivity domains
**When** the reconciliation schedule runs
**Then** it runs hourly for Personal and Payroll, and daily for Employment, Education & Experience, and Additional Info (NFR-7) — not a uniform cadence

### Epic 4: Production Hardening & Operability
Operators can trust this in production: dead-lettered jobs are recoverable with a real audit trail, keys survive a restart, and real throughput/latency numbers replace the disclaimed dev-laptop baseline.
**FRs covered:** NFR-2, NFR-3, NFR-6 (threshold), FR-13 (persistence model), persistent key store (additional requirement)

### Story 4.1: Persistent salt/key store for `integration-bridge`

As a platform operator,
I want the bridge's salt and per-record key store to survive a restart,
So that a routine deploy doesn't silently break digest verification for every record anchored before it.

**Acceptance Criteria:**

**Given** `integration-bridge` currently uses `keystore.InMemory*Store` (salt store, employee-key store) with no persistence
**When** this story is implemented
**Then** these stores are backed by durable storage, and every salt/`employeeKey_i` issued before a restart is still available after it

**Given** a bridge restart occurs after this story ships
**When** an anchor or verification request needs a previously-issued salt/key
**Then** it resolves correctly — no unrecoverable key loss, and no previously-anchored record becomes unverifiable

### Story 4.2: Dead-letter persistence and operator visibility

As an operator,
I want a dead-lettered anchoring job to be a real, queryable, restart-surviving record,
So that I can actually find and re-drive it without data loss.

**Acceptance Criteria:**

**Given** an anchoring job exhausts its retries (FR-12)
**When** it lands in a dead-letter state
**Then** a persisted record is created capturing the payload, correlation ID, failure reason, and attempt count — surviving process restarts, independent of whatever UI reads it (FR-13)

**Given** a dead-lettered job's persisted record exists
**When** an operator looks it up (by correlation ID or otherwise)
**Then** they can find it and re-drive it, and re-driving does not lose or duplicate data

**Given** Yii2's default queue table alone (with no correlation-ID indexing or redrive semantics) is not sufficient to satisfy this story
**When** this story is reviewed for completion
**Then** it is not marked done unless the persisted record is actually queryable by correlation ID, not merely "in a table somewhere"

### Story 4.3: Real production performance benchmark and backlog threshold

As a platform owner,
I want a dedicated-hardware benchmark replacing the disclaimed dev-laptop numbers, and a real backlog-depth alert threshold,
So that NFR-2/NFR-3/NFR-6 stop being placeholders before this ships to production.

**Acceptance Criteria:**

**Given** the only existing latency/throughput baseline (p95 ≈ 26.7s single-write, dev laptop hardware) is explicitly disclaimed as non-representative
**When** this story is implemented
**Then** a benchmark is run on dedicated (non-shared, non-laptop) hardware, producing real end-to-end anchoring-latency and throughput numbers that replace the "TBD" targets in NFR-2/NFR-3

**Given** NFR-6's backlog observability currently has no numeric threshold
**When** this story is implemented
**Then** a concrete backlog-depth/growth-rate alert threshold is set and wired to the metrics from Story 1.6/Epic 1

---
stepsCompleted: ['document-discovery', 'prd-analysis', 'epic-coverage-validation', 'ux-alignment', 'epic-quality-review', 'final-assessment']
inputDocuments:
  - '_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-11/prd.md'
  - '_bmad-output/planning-artifacts/architecture/architecture-fabric-hris-talenta-2026-08-11/ARCHITECTURE-SPINE.md'
  - '_bmad-output/planning-artifacts/architecture/architecture-fabric-hris-talenta-2026-08-11/SOLUTION-DESIGN.md'
  - '_bmad-output/planning-artifacts/epics-talenta-fabric-2026-08-12.md'
---

# Implementation Readiness Assessment Report

**Date:** 2026-08-12
**Project:** fabric-hris — Talenta HRIS x Hyperledger Fabric Write-Path Integration

## Document Inventory

Three parallel, unrelated document families exist in this repo under similar filenames
(`*fabric-hris*`). Resolved by content identity, not filename glob, to the following set for this
assessment:

- **PRD:** `prds/prd-fabric-hris-2026-08-11/prd.md`
- **Architecture:** `architecture/architecture-fabric-hris-talenta-2026-08-11/ARCHITECTURE-SPINE.md`
  (+ companion `SOLUTION-DESIGN.md`)
- **Epics & Stories:** `epics-talenta-fabric-2026-08-12.md`
- **UX:** none exists for this integration (correctly absent)

Excluded as unrelated: `prd-fabric-hris-2026-08-02` + `architecture-fabric-hris-2026-08-08` +
`epics.md` (original thesis prototype); `prd-fabric-hris-2026-08-10` +
`architecture-fabric-hris-2026-08-11` + `epics-caliper-dashboard-2026-08-11.md` (Caliper dashboard
feature).

## PRD Analysis

### Functional Requirements

FR-1: For every completed write to a Personal/Employment/Education&Experience/Additional Info/Payroll endpoint, the system SHALL enqueue an anchoring job. Four in-scope mutations (actionDeletePayroll, actionAddComponent, actionDeleteEmergencyContact, actionImportDataEmergencyContact) require NEW publish()-equivalent calls, not extension of existing call sites.
FR-1a: The anchoring enqueue mechanism SHALL reuse existing Yii2 queue infrastructure via a NEW job class (not WebhookWorker, which is hard-wired to one external webhook receiver and swallows errors with no retry/dead-letter). This new job's target is integration-bridge's existing POST /v1/profile-sections/{DOMAIN} route; all of FR-12/13's retry/dead-letter behavior lives in this new job.
FR-2: Each anchoring job SHALL compute a canonical (JCS) serialization and HMAC-SHA256 digest of the changed domain's post-write field set, using fabric-hris's existing gateway-client digest construction.
FR-3: Each anchor SHALL reference the previous digest for that (employee, domain) pair, forming a hash chain. CREATE operations SHALL use a defined genesis sentinel (all-zero hash) as previous-state hash, never null/omitted.
FR-4: Each anchoring transaction SHALL carry: employee id, company_id, domain, operation type (CREATE/UPDATE/DELETE), actor identity, source endpoint, timestamp, previous-state hash, new-state hash, and correlation ID. Operation type MUST be derived from webhook payload/diff semantics, not the HTTP verb.
FR-5: The anchoring job SHALL be idempotent under retry (duplicate submission of the same webhook event must not create a duplicate chain entry).
FR-6: Any employee SHALL be able to view their own anchored transaction history across all 5 domains via the existing read path, regardless of role.
FR-7: Company HR Admins/Super Admins/Consultants SHALL be able to view any employee's history within their own company, per today's existing hasAccess() permission model.
FR-8: History views SHALL be filterable by employee, domain, date range, operation type, and transaction status, and SHALL NOT expose another employee's field values/digests/metadata anywhere in the response, including error/empty-state cases.
FR-9: The system SHALL introduce a correlation ID at webhook-enqueue time, propagated through the queue job, the Fabric submission, and returned in the Fabric tx metadata.
FR-10: An on-demand verification action SHALL recompute the current Talenta DB state's digest for a given employee+domain and compare it to the latest on-chain anchor; a mismatch SHALL be reported as compromised/inconsistent. A latest anchor with operation type DELETE SHALL report "deletion verified" and stop, never hash-comparing against an absent DB row.
FR-11: A scheduled reconciliation job SHALL identify employee changes that exist in Talenta but have no corresponding Fabric anchor (missed anchors), and surface them for retry/investigation. Its determination of "what changed" MUST be derived independently of the anchoring trigger itself (DB updated_date/row-version columns, not the webhook/queue's own success log).
FR-12: If Fabric is unreachable when an anchoring job runs, the job SHALL retry with backoff and land in a dead-letter state after exhausting retries — never silently dropped.
FR-13: A dead-lettered anchoring job SHALL be visible to operators and re-driveable without data loss.

Total FRs: 13 (FR-1 through FR-13, with FR-1a as a direct amendment to FR-1's mechanism)

### Non-Functional Requirements

NFR-1: Adding anchoring MUST NOT measurably regress p95/p99 response time of any of the 5 domains' write endpoints, since anchoring is fully asynchronous. [ASSUMPTION: exact ms target not yet set by stakeholder.]
NFR-2: End-to-end anchoring latency (webhook enqueue to on-chain commit). Measured dev-environment baseline (p95 ~26.7s, single-write) explicitly disclaimed as non-representative of production hardware. Production target: TBD pending dedicated-hardware benchmark.
NFR-3: Must sustain the write volume of all 5 domains across all active tenants without unbounded queue growth. Production target: TBD, same reason as NFR-2.
NFR-4: A Fabric outage MUST NOT block or degrade any Talenta write operation (direct consequence of detective-only enforcement, D1).
NFR-5: Zero raw PII fields (§6 classification) SHALL ever appear on-chain; verified by extending fabric-hris's existing pilscan full-ledger PII scan to the new field set.
NFR-6: Anchoring queue depth and dead-letter count SHALL be visible as monitored metrics, alertable if backlog grows unbounded.
NFR-6a: [ASSUMPTION: no numeric backlog-depth/growth-rate threshold set yet — deferred to Phase 4 production hardening.]
NFR-7: The scheduled reconciliation job (FR-11) SHALL run hourly for Personal and Payroll (highest-sensitivity domains), daily for Employment, Education & Experience, and Additional Info.

Total NFRs: 7 (NFR-1 through NFR-7, plus NFR-6a as a direct amendment to NFR-6)

### Additional Requirements

- **D1 (Authorization model, §9):** Detective, not preventive enforcement — Fabric never consulted synchronously before a Talenta write commits. Talenta's existing role/feature-flag authorization continues to gate every write unchanged.
- **Detection boundary (§9):** Explicitly disclosed — this design catches out-of-band tampering (bypassing the app) but NOT in-band improperly-authorized writes that pass through Talenta's normal pipeline with a broken authorization check.
- **Personal-domain launch gate (§9):** `POST /my-info/update-identity-address`'s missing `canRequestChangeData` check is a launch pre-requisite specifically for the Personal domain — must be fixed in Talenta before Personal-domain anchoring can be considered a meaningful integrity claim.
- **Chaincode fit (§6):** `EmployeeProfileRecord`'s 12-field schema and `ProfileSection` enum already match the PRD's 5-domain split 1:1 — no schema extension needed or permitted; this is a ratified design per fabric-hris CLAUDE.md.
- **Write-path correction (§1, §2):** `integration-bridge` already implements the write-receiving side (all 5 domains, synchronous, tested) — this PRD's core scope is narrower than a from-scratch build: the missing Talenta-side trigger, plus hardening (b)-(d): no retry/queue/dead-letter in the bridge, no operation-type dispatch, in-memory-only key stores.
- Non-Goals (§4): no synchronous/blocking Fabric check; no new Talenta role; `PayrollComponentController`'s ~28 other actions and custom-field definition CRUD out of scope; `tenant02` scaling out of scope; no legal GDPR/PDP compliance claim.
- Glossary (§16): Domain, Anchor, Digest, Correlation ID, Reconciliation, Dead-letter — all formally defined.

### PRD Completeness Assessment

The PRD is unusually well-grounded for a brownfield integration: every core claim is traced to a
real file:line citation from either the Talenta API documentation or the actual `talenta-core`/
`fabric-hris` codebases (reconciliation reports exist for both). It underwent an adversarial
security review that surfaced and closed one critical gap (FR-11's independent-source requirement,
closing a detection-suppression attack) before being finalized. Two NFR numeric targets (NFR-2,
NFR-3) are honestly marked TBD rather than fabricated, with the disclaimed baseline clearly
separated from any actual target. The PRD was also corrected in-place, post-finalization, once
the downstream architecture session discovered the write-path premise was inaccurate (the bridge
already existed) — this is documented transparently in §1 rather than silently patched. Assessment:
complete and implementation-grounded, no vague or unverified claims found.

## Epic Coverage Validation

### Coverage Matrix

| FR Number | PRD Requirement (summary) | Epic Coverage | Status |
| --- | --- | --- | --- |
| FR-1 | Every completed write enqueues an anchoring job; 4 mutations need new trigger calls | Epic 1 Story 1.1 (Payroll's 2) + Epic 2 Stories 2.1/2.3/2.4 (remaining 2, plus Education & Additional Info's brand-new wiring) | ✓ Covered |
| FR-1a | New async job class, not WebhookWorker; targets integration-bridge | Epic 1 Story 1.1 | ✓ Covered |
| FR-2 | Canonical digest construction | Epic 1 Story 1.2 | ✓ Covered |
| FR-3 | Hash chain + genesis sentinel | Epic 1 Story 1.2 | ✓ Covered |
| FR-4 | Anchor metadata fields + operation-type derivation | Epic 1 Story 1.3 | ✓ Covered |
| FR-5 | Idempotency under retry | Epic 1 Story 1.4 | ✓ Covered |
| FR-6 | Self-view history, any role | Epic 1 Story 1.5 | ✓ Covered |
| FR-7 | Admin cross-employee history view | Epic 1 Story 1.5 | ✓ Covered |
| FR-8 | History non-disclosure boundary | Epic 1 Story 1.5 | ✓ Covered |
| FR-9 | Correlation ID traceability | Epic 1 Story 1.3 | ✓ Covered |
| FR-10 | On-demand verification incl. DELETE behavior | Epic 3 Story 3.1 | ✓ Covered |
| FR-11 | Scheduled reconciliation, independent source | Epic 3 Story 3.2 | ✓ Covered |
| FR-12 | Retry with backoff, dead-letter on exhaustion | Epic 1 Story 1.1 (trigger) + Epic 4 Story 4.2 (persistence) | ✓ Covered |
| FR-13 | Dead-letter visible/re-driveable | Epic 4 Story 4.2 | ✓ Covered |
| NFR-1 | No latency regression | Epic 1 Story 1.6 | ✓ Covered |
| NFR-2 | Real anchoring-latency benchmark | Epic 4 Story 4.3 | ✓ Covered |
| NFR-3 | Real throughput benchmark | Epic 4 Story 4.3 | ✓ Covered |
| NFR-4 | Fabric outage never blocks writes | Epic 1 Story 1.1 | ✓ Covered |
| NFR-5 | Zero raw PII on-chain (pilscan) | Epic 1 Story 1.2 | ✓ Covered |
| NFR-6 | Backlog metrics visible | Epic 1 Story 1.6 (metrics exist) + Epic 4 Story 4.3 (numeric threshold) | ✓ Covered |
| NFR-7 | Reconciliation cadence (hourly/daily split) | Epic 3 Story 3.2 | ✓ Covered |
| Additional — Personal launch gate | `canRequestChangeData` fix as blocking dependency | Epic 2 Story 2.1 (explicit blocking AC) | ✓ Covered |
| Additional — chaincode fit / no schema change | No schema extension anywhere | Cross-cutting constraint, enforced by review in every epic (not a story) | ✓ Covered (by design, correctly not a story) |
| Additional — persistent key store | In-memory store production blocker | Epic 4 Story 4.1 | ✓ Covered |
| Additional — Detection boundary (§9) | In-band improperly-authorized writes not caught by design | Documented in PRD/architecture; not a story since it's a disclosed non-goal, not a build task | ✓ Covered (correctly not a story) |

### Missing Requirements

None. All 13 FRs, all 7 NFRs, and all "Additional Requirements" from the PRD/architecture trace to
at least one story, or are correctly identified as cross-cutting constraints / disclosed
limitations rather than buildable stories.

### Coverage Statistics

- Total PRD FRs: 13 (FR-1 through FR-13, FR-1a as an amendment)
- Total PRD NFRs: 7 (NFR-1 through NFR-7, NFR-6a as an amendment)
- Architecture-sourced FRs (FR-Arch-1, FR-Arch-2): 2, both covered (Epic 1 trivial case; Epic 2 Family/Additional-Info cases; Epic 2 route fix)
- FRs covered in epics: 13/13 (100%)
- NFRs covered in epics: 7/7 (100%)
- Coverage percentage: 100%

## UX Alignment Assessment

### UX Document Status

Not Found (correctly — no UX design contract exists for this integration; the only UX run in this
repo belongs to the unrelated Caliper dashboard feature).

### Alignment Issues

None. UI is implied at the margins (FR-6/FR-7/FR-8 reference the "Transaction History" screen), but
the PRD explicitly scopes this as **already-existing UI, unchanged in shape** (`TransactionHistory.vue`
+ `MyInfoBlockchainHistoryService`) — this integration makes that screen show real data instead of
an empty placeholder, but does not add, remove, or restyle any UI component. No new UI surface is
in scope anywhere in the epics/stories.

### Warnings

None. UX is implied but not missing in any load-bearing sense — the existing UI is being fed real
data, not redesigned. No warning needed.

## Epic Quality Review

Applying create-epics-and-stories standards rigorously against `epics-talenta-fabric-2026-08-12.md`.

### Epic Structure Validation

| Epic | User Value Check | Independence Check |
| --- | --- | --- |
| 1 — Payroll Write-Path Anchoring | Pass — user-centric title, delivers complete standalone Payroll-anchoring capability | Pass — stands alone completely, no forward reference to Epic 2/3/4 |
| 2 — Remaining Domains | Pass — extends user value to 4 more domains | Pass — uses only Epic 1's pipeline as output, doesn't require Epic 3/4 to function |
| 3 — Integrity Verification & Reconciliation | Pass — auditor/operator user value, explicitly "works against whatever domains are live" | Pass — doesn't block on Epic 2 completing, correctly noted in the epics doc itself |
| 4 — Production Hardening & Operability | Borderline — framed via "operator" persona per FR-13's own visibility requirement, which is a real requirement, not manufactured value; still the closest to a "technical milestone" of the four | Pass — cross-cutting hardening, doesn't gate the other three |

No critical epic-level violations found. Epic 4's operator framing is legitimate (FR-13 explicitly
requires operator-facing dead-letter visibility as a functional requirement, not incidental
tech debt), but it's the one epic worth watching if the team's stakeholders expect every epic to
read as end-user-facing.

### Story Quality Assessment

**No forward dependencies found.** Every story in every epic builds only on prior stories within
its epic (Story 1.2 uses 1.1's job class; Story 1.4 uses 1.2's submission mechanism; Story 3.1
uses Epic 1's anchors; Story 4.2 uses Epic 1's dead-letter trigger point). The one external
blocking dependency found — Story 2.1's Personal-domain launch gate — is a dependency on a
**Talenta-side code fix outside this story set**, not a forward reference to a future BMad story;
structurally compliant, but flagged again here as a real-world scheduling risk (see Final
Assessment).

**Database/entity creation timing:** Compliant. The one new persistent store (Story 4.1's
key-store persistence) is created exactly where first needed, not front-loaded in Epic 1.

**Findings by severity:**

#### 🟠 Major Issues

- **Story 1.6 combines two distinct concerns.** "Make the anchoring backlog observable" (an
  operator-facing capability) and "prove no latency regression" (a verification/testing activity)
  are bundled into one story. They have different audiences and different definitions of done.
  *Recommendation:* consider splitting into 1.6 (observability metrics, operator value) and a
  separate verification task (regression proof) — though not severe enough to block, since both
  halves are already independently testable ACs within the one story.

#### 🟡 Minor Concerns

- Stories 2.3 and 2.4 both note their trigger wiring is "new, not an extension" but neither story's
  ACs test that the new hook doesn't disturb the existing (unrelated) behavior of
  `FormalEducationController`/`InformalEducationController`/`WorkingExperienceController`/
  `AdditionalInfoController`'s other actions. Worth a regression-safety AC at story-writing-for-dev
  time, not a blocker for this readiness check.
- FR-1a's constraint that the new job must NOT ride inside `WebhookWorker::execute()` is satisfied
  by Story 1.1's design intent but isn't independently testable as a standalone AC (it's an
  implementation-structure constraint, not a behavior). Low risk since the architecture spine
  (AD-1) already carries this as an enforceable rule a code reviewer can check directly.

No critical violations (no technical-milestone epics, no forward dependencies, no epic-sized
unsizeable stories) were found.

## Summary and Recommendations

### Overall Readiness Status

**READY**

### Critical Issues Requiring Immediate Action

None. No critical violations were found in FR/NFR coverage (100%), epic independence, or story
dependency structure.

### Recommended Next Steps

1. **Track the Personal-domain launch gate as an explicit external dependency, not just prose.**
   Story 2.1 correctly encodes `POST /my-info/update-identity-address`'s missing
   `canRequestChangeData` check as a blocking AC, but this is a fix in a *different* codebase
   (`talenta-core`) outside this epic set's own delivery control. Before Sprint Planning, confirm
   who owns filing/tracking that fix as its own ticket, so Epic 2's Personal-domain work doesn't
   silently stall waiting on an untracked external dependency.
2. **Consider splitting Story 1.6** (observability metrics vs. latency-regression proof) if the
   team wants stories to map cleanly to a single audience/definition-of-done — not blocking, but
   worth a quick look before Sprint Planning assigns it.
3. **Add regression-safety ACs to Stories 2.3/2.4 at dev-story time** (Create Story step) — confirm
   the new trigger wiring in `FormalEducationController`/`InformalEducationController`/
   `WorkingExperienceController`/`AdditionalInfoController` doesn't disturb those controllers'
   existing unrelated behavior. Minor, but cheap to add now versus discovering it in code review.
4. **Proceed to Sprint Planning.** All FRs/NFRs trace to stories, no forward dependencies exist,
   and epic independence holds — the plan is ready for `bmad-sprint-planning` to sequence into an
   actual implementation schedule.

### Final Note

This assessment identified 3 issues across 2 categories (1 major process-shape concern in Epic
Quality Review; 2 minor concerns spanning Epic Quality Review and dependency tracking) — no
critical issues. All are advisory, not blocking. The PRD → Architecture → Epics/Stories chain for
this integration is unusually well cross-referenced for a brownfield integration: every major claim
traces to a real file:line citation, two rounds of adversarial/reconciliation review already ran on
the PRD and the architecture spine independently, and this readiness check found no coverage gaps
introduced in translating either into the epics document. You may proceed to implementation as-is,
or address the three advisory items first — neither path is blocked.

---
**Assessed by:** bmad-check-implementation-readiness
**Date:** 2026-08-12

# Reconciliation: ARCHITECTURE-SPINE.md vs prd.md (prd-fabric-hris-2026-08-11)

Scope: does every FR the spine claims to bind (`binds: [FR-1, FR-1a, FR-2, FR-3, FR-4, FR-5, FR-9,
FR-10, FR-11, FR-12, FR-13]`) get an actual AD or an explicit Deferred item, and did anything the
PRD requires get silently dropped along the way.

## 1. FR → AD coverage audit

| FR | Spine treatment | Verdict |
|---|---|---|
| FR-1 | Mentioned only as a code comment in Structural Seed ("gains publish()-equivalent calls at the 4 currently-unwired actions"); no AD states it as a rule | **Partial gap** — see §5.1 |
| FR-1a | AD-1, AD-6 | Covered |
| FR-2 | AD-2, AD-4 | Covered |
| FR-3 | AD-3 (key shape only) | **Partial gap** — genesis sentinel not addressed, see §5.2 |
| FR-4 | AD-4, AD-5 | **Partial gap** — wire-shape convention omits fields FR-4 mandates, see §5.3 |
| FR-5 | AD-6 | Covered |
| FR-9 | AD-6 | Covered |
| FR-10 | Capability map lumps it into "Verification/reconciliation (FR-10, FR-11)" → Deferred, but the Deferred bullet text only discusses FR-11 | **Silent drop** — see §5.4 |
| FR-11 | Deferred (explicit) | Covered as an acknowledged gap |
| FR-12 | AD-1 | Covered |
| FR-13 | AD-1 + Deferred ("Operator-facing dead-letter UI/tooling") | **Partial gap** — Deferred only names the UI, not the underlying persistence requirement, see §5.6 |

Net: of 11 bound FRs, 4 have real gaps (FR-1, FR-3, FR-4, FR-13) and 1 is claimed-but-unaddressed
(FR-10).

## 2. FR-8 / AC-7 (history non-disclosure boundary)

Not a gap. FR-6/FR-7/FR-8 are never in the spine's `binds` list at all — the spine's `scope` line
frames itself as the write-path boundary and explicitly says the read path is "unchanged in shape"
per the PRD's own §2 table. Because FR-8 was never claimed, its absence is an honest scope
exclusion, not a silent drop. It should stay untouched by this spine; a companion read-path spine
(if `TransactionHistory.vue`/`MyInfoBlockchainHistoryService` are ever touched) is where AC-7's
per-employee isolation boundary belongs.

## 3. NFR-7 cadence + FR-11 "what changed" independence

Acceptable as a scope boundary for a write-path-focused spine — but the Deferred stub is
incomplete. The existing bullet:

> "Reconciliation job's placement and 'what changed' source (FR-11, AD from the PRD requiring
> DB-timestamp-derived change detection independent of the trigger) — not designed here..."

preserves *only* the independent-detection-source constraint (AC-8's core requirement). It says
nothing about NFR-7's domain-differentiated cadence (hourly for Personal/Payroll, daily for the
other three). A future reconciliation-focused spine that reads only this Deferred stub — not the
PRD — has no signal that a uniform cadence would violate NFR-7. Recommend the Deferred bullet be
expanded to also carry the cadence split, e.g.: "...must run hourly for Personal/Payroll and daily
for Employment/Education/Additional Info per NFR-7 — not a uniform cadence."

## 4. Does AD-1 fully satisfy FR-12/FR-13?

AD-1 fully covers FR-12 (retry authority, single owner, backoff/dead-letter logic lives in the
Talenta job). It only partially covers FR-13. FR-13 has two parts: "visible to operators" and
"re-driveable without data loss." AD-1's mermaid diagram draws a box labeled "Operator-visible DLQ"
but no AD or Deferred item specifies what backs that box — a dead-lettered job needs a **persisted,
queryable record** (payload, correlation ID, failure reason, attempt count) surviving process
restarts, independent of whatever UI eventually reads it. The Deferred section's "Operator-facing
dead-letter UI/tooling" bullet names only the *interface* gap, not the *data-model/persistence*
gap underneath it. A builder could satisfy AD-1 literally (retry logic lives in the job class) while
leaving dead-lettered jobs sitting only in Yii2's default queue table with no correlation-ID
indexing or redrive semantics — technically "in a table somewhere" but not actually meeting "visible
to operators... re-driveable without data loss" as an architectural property.

## 5. Other PRD constraints a spine-only reader could violate

### 5.1 FR-1's four unwired actions (silent, low-visibility placement)

FR-1 is explicit that it is "not satisfied by merely extending existing call sites" — it requires
*new* `publish()`-equivalent calls in `actionDeletePayroll`, `actionAddComponent`,
`actionDeleteEmergencyContact`, and `actionImportDataEmergencyContact`, none of which fire today.
The spine records this fact only as a parenthetical code-comment inside the Structural Seed's file
listing, with no AD, no "Prevents" framing, and no Capability Map row calling it out on its own. A
builder skimming the AD list (which is where "what must be true" normally lives) could wire
`AnchorTriggerJob` cleanly at the 7 existing `publish()` sites and ship, permanently
under-anchoring exactly the four actions FR-1 called out — the omission that would previously
require FR-11 to ever detect, and FR-11 is deferred.

### 5.2 FR-3's genesis sentinel

FR-3 requires an explicit, defined genesis sentinel (e.g. all-zero hash) as the previous-state hash
for a CREATE — "not null or omitted." AD-3 only defines the *key shape*
(`employee, profileSection, recordIdentity`) for chain identity; it says nothing about what value a
first-ever anchor's previous-hash field should carry. A builder following AD-3 alone could leave
this null/omitted, which FR-3 explicitly forbids (and which FR-10's verification logic is designed
around not having to special-case).

### 5.3 FR-4's field list vs. the wire-shape convention

FR-4 mandates the anchoring transaction carry: employee id, **company_id**, domain, operation type,
actor identity, **source endpoint**, timestamp, previous-hash, new-hash, and correlation ID. The
spine's Consistency Conventions table fixes the cross-boundary JSON shape as
`{employeeInternalID, userID, newValue, document, correlationID, operationType}`. Missing entirely:
`company_id` and source endpoint; timestamp and "actor identity" are not explicitly named either
(`userID` may or may not be intended as actor identity — the spine doesn't say). Since this table is
presented as the definitive wire contract, a builder implementing `AnchorTriggerJob`'s POST body
strictly from this table would omit `company_id` — a field the PRD calls out by name, and one whose
absence matters given the design's own cross-company-isolation acceptance criterion (AC-2).

### 5.4 FR-10 has no addressal at all

FR-10 (on-demand verification, recompute-and-compare) is claimed in `binds` but never appears in an
AD, and the Capability Map's "Verification/reconciliation (FR-10, FR-11)" row points only at
Deferred — yet the actual Deferred bullet text discusses only FR-11's reconciliation-job placement
and its independent-source requirement. FR-10 is a distinct capability (on-demand hash-mismatch
check, not scheduled sweep) and gets zero design ink anywhere in the document body. This is the
clearest case of an FR the frontmatter claims as bound but the body silently drops.

### 5.5 FR-10's DELETE-verification behavior

Related to 5.4: PRD §13 has a `[RESOLVED]` item specifying that FR-10 verification against a
DELETE-latest-anchor must report "deletion verified" and stop, never hash-comparing against a
now-absent DB row. Since FR-10 has no AD at all, this resolved design decision has nowhere to live
in the spine either — it would need to be picked up whenever FR-10 finally gets designed.

### 5.6 Personal-domain launch gate (§9) — entirely unmentioned

The PRD elevates one specific pre-existing Talenta auth gap — `POST
/my-info/update-identity-address` never checking `canRequestChangeData`, so it always writes
directly, bypassing the approval flow — to a **launch pre-requisite specifically for the
Personal domain**: "it should be fixed in Talenta... before this integration's Personal-domain
anchoring can be represented as a meaningful integrity claim." This is a rollout/scope-gating
constraint, not a pure architecture concern, but it directly affects when Personal-domain wiring
(which the spine's AD-3/AD-4 both govern in detail — Family's `recordIdentity`, `PERSONAL` routing)
is safe to ship. It appears nowhere in the spine — not in `binds`, not in Deferred, not in the
Migration/Rollout-adjacent parts of the Structural Seed. A team building strictly off this spine
would have no signal that shipping Personal-domain anchoring is gated on a Talenta-side fix
landing first, and could ship a "verified" integrity claim over an unapproved bypass path the PRD
already flagged as laundering risk.

## Summary

**7 gaps found**, plus one confirmed-correct exclusion (FR-8/AC-7, out of scope by honest
omission, not silently dropped):

1. FR-1's 4 unwired actions — noted only in a Structural Seed comment, not an enforceable AD rule.
2. FR-3's genesis-sentinel value convention — not addressed by AD-3 (key-shape only).
3. FR-4's wire-shape convention omits `company_id` and source endpoint from the mandated field list.
4. FR-10 — claimed as bound, has no AD and no real Deferred addressal (Capability Map row is a
   pointer to a Deferred bullet that doesn't actually discuss it).
5. FR-10's DELETE-verification resolved behavior (§13) has nowhere to live, downstream of gap 4.
6. NFR-7's hourly/daily cadence split is missing from the reconciliation Deferred stub (only the
   independent-source constraint survived).
7. FR-13's "visible to operators, re-driveable without data loss" needs a persisted dead-letter
   data model, not just a UI — Deferred only names the UI gap.
8. The PRD's Personal-domain launch gate (§9, the `update-identity-address` auth-gap fix) is
   entirely absent from the spine, despite directly bearing on when the spine's own Personal-domain
   ADs (AD-3, AD-4) are safe to ship against.

File: `/Users/chan/www/hyperledger/fabric-hris/_bmad-output/planning-artifacts/architecture/architecture-fabric-hris-talenta-2026-08-11/reconcile-prd.md`

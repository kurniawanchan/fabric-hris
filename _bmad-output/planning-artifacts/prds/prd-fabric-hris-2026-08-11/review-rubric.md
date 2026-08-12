# PRD Quality Review — prd-fabric-hris-2026-08-11

## Overall verdict

This PRD is unusually well-grounded for a first pass: it's reconciled against real code (specific
file:line citations, actual controller action counts, real performance numbers with their caveats
intact) rather than written from template. Its weakest points are downstream mechanics (no
Glossary, no FR/UJ/SM ID scheme beyond FRs, no Assumptions Index despite one inline
`[ASSUMPTION]`) and one substantive risk left underdeveloped — §9's Talenta-side auth gaps are named
as "risk context" but never load-bear on D1's detective-only decision the way they should. It is
implementation-ready for engineering scoping today; it is not yet ready to hand to a downstream
UX/architecture/story pipeline without a Glossary and ID pass.

## Decision-readiness — strong

D1 ("authorization enforcement is detective, not preventive," §9) is stated as an actual ratified
decision with a date, not softened into a consideration, and its cost is named explicitly: "at the
cost of not being able to reject an unauthorized write in real time — only detect it afterward."
The Migration & Rollout section's Payroll-first sequencing (§12) is also framed correctly as "a
build-order choice, not a scope cut," distinguishing sequencing from scope honestly.

The two `[OPEN]` items in §13 (NFR-2/NFR-3 targets, NFR-7 cadence) are genuinely open — NFR-7 is
flatly stated as "SHALL run daily" in §8 but then re-flagged open in §13, which is an inconsistency
(see Mechanical notes) rather than a real open question dressed up.

### Findings
- **medium** NFR-7 contradicts its own "open" flag (§8 vs §13) — §8 states a firm daily cadence
  with SHALL language, but §13 lists "NFR-7 reconciliation cadence" as `[OPEN]`. A reader can't tell
  if daily is decided or still pending. *Fix:* either drop NFR-7 from the Open Questions list or
  soften §8's SHALL to reflect that the cadence is provisional.
- **low** §9's Talenta auth-gap disclosure ("Known pre-existing auth gaps...") is flagged as risk but
  the PRD doesn't say what happens to D1 if those gaps get exploited before they're fixed — e.g. the
  `update-identity-address` gap means an "authorized write" by the PRD's own definition may not
  really be approval-gated. This is close to a real objection to D1 that the PRD names but doesn't
  fully engage. *Fix:* add one sentence on whether reconciliation/verification (FR-10/FR-11) is
  expected to catch this class of gap or whether it's an acknowledged blind spot.

## Substance over theater — strong

No persona theater — actors (§5) are terse, functional, tied directly to FR-6/FR-7's role-based
history visibility, not decorative. No Vision theater: the Vision paragraph (§1) is specific to this
system (`talenta-core`, `HF001`, named services) — it could not be swapped into another PRD
unchanged. NFR-2/NFR-3 are the opposite of NFR boilerplate: they cite a real dev-environment
benchmark with explicit numbers (p95 ≈ 26.7s, 7056/7562 failures at 500 tps) and explicitly disclaim
those numbers as "a known-bad floor, not a target" rather than dressing a placeholder up as a
threshold.

### Findings
- **low** G7 ("Negligible added latency") and NFR-1 pair a vague goal-level adjective with a
  quantified NFR right below it (`<10ms` `[ASSUMPTION]`) — the vagueness at G-level is fine since
  the NFR immediately grounds it, but it's worth noting the Goals section alone (§3) still reads as
  adjective-only if read in isolation from §8.

## Strategic coherence — strong

The thesis is explicit and singular: make Fabric detective-only so it never becomes a write-path
liability, trading real-time rejection for provable tamper-evidence (D1, §9). Every FR group serves
that thesis — Group A anchors, Group D verifies/reconciles, Group E makes the async path resilient
so the "never blocks writes" promise (G7/NFR-4) actually holds under Fabric outages. The
counter-metrics in §3 (p95/p99 regression check, backlog-depth check) are the right shape — they
guard against G7 being gamed by an anchoring path that quietly degrades the write endpoints it
claims not to touch.

MVP scope kind is a hybrid of platform-capability and compliance/trust-infrastructure — appropriate
given the domain — and the phased rollout (§12) follows the thesis (highest-sensitivity domain
first) rather than "easiest first."

## Done-ness clarity — adequate

Most FRs are testable: FR-5's idempotency, FR-9's correlation-ID propagation, FR-12/13's
retry/dead-letter behavior all have a clear pass/fail shape, reinforced by AC-1 through AC-6 in §15
which give Given/When/Then acceptance criteria for the core flows. FR-1's finding about the 4 unwired
controller actions (`actionDeletePayroll`, `actionAddComponent`, `actionDeleteEmergencyContact`,
`actionImportDataEmergencyContact`) is a rare example of a PRD naming its own gap precisely enough
to be immediately actionable.

Two FRs fall short of "an engineer would know what done looks like":

### Findings
- **high** FR-8's non-disclosure clause is untestable as written: "SHALL NOT expose the underlying
  employee data to a viewer who is not otherwise authorized to see that data" (§7) restates the
  existing authorization model by reference rather than stating what the history view itself must
  never leak (e.g., does a filtered/paginated history response ever include another employee's
  values in an aggregate count, in an error message, in a "next domain" preview?). There's no AC
  covering this specific surface, only AC-2's cross-company case. *Fix:* add a concrete boundary
  case to FR-8 (e.g., "a history list request scoped to one employee must never include another
  employee's field values even in error/empty-state responses") and a matching AC.
- **medium** FR-4's field list omits a testable boundary on the previous-state/new-state hash pair
  for CREATE operations — is "previous-state hash" a defined sentinel (null, zero-hash, genesis
  value) or does FR-4 as written imply CREATE has no valid record until this is specified? The hash
  chain (FR-3) depends on this being unambiguous. *Fix:* state the CREATE-time previous-hash
  convention explicitly.
- **low** NFR-6 ("visible as monitored metrics, alertable if backlog grows unbounded") lacks a bound
  — "unbounded" is doing the adjective's job here without a number or growth-rate threshold, unlike
  NFR-1's `<10ms` `[ASSUMPTION]` treatment of a similarly underspecified target. *Fix:* either add an
  `[ASSUMPTION: threshold TBD]` tag matching NFR-1's pattern, or state it's intentionally deferred to
  Phase 4 (§12 already implies this — make it explicit in NFR-6 itself).

## Scope honesty — strong

Non-Goals (§4) does real work — it explicitly descopes `tenant02`, the 28 other
`PayrollComponentController` actions, and legal GDPR/PDP compliance claims, each with a one-line
reason tied back to existing repo state (`tenantprovision` danger, API-doc scope, §14). The single
`[ASSUMPTION]` (NFR-1's `<10ms` budget) is tagged inline and explicitly flagged as needing
confirmation, which is the right move — but see Mechanical notes on the missing index. `[RISK]` and
`[OPEN]` tags in §13 are used correctly rather than as rhetorical hedges. Open-items density (1
assumption, 2 opens, 1 risk, no NOTE-FOR-PM tags) is proportionate to a PRD of this technical depth
and is not being used to bury a real fight.

## Downstream usability — thin

No Glossary section exists despite heavy, consistent reuse of domain nouns (`anchor`,
`digest`, `domain`, `correlation ID`) across FRs, NFRs, and Data Flows — the terms are used
consistently in practice, so this is a downstream-friction gap rather than a drift problem, but a
PRD of this density should still define them once. There is no FR/UJ/SM ID scheme beyond
FR-1…FR-13 — no UJs exist at all (see Shape fit, where this is likely correct for this product type)
and no SM-numbered success metrics, only prose Goals with G-numbers that double as both goals and
implicit metrics. Cross-references (e.g. "§9, decision D1", "see §6") are section-based rather than
Glossary-anchored, which is workable for a document this size but would not scale to a larger PRD.

### Findings
- **medium** No Glossary. Terms like "domain" (meaning one of the 5 profile sections), "anchor"
  (verb and noun), and "digest" are used dozens of times consistently but never formally defined in
  one place — a downstream architecture or story-writing pass would have to infer definitions from
  context. *Fix:* add a short Glossary section defining Domain, Anchor, Digest, Correlation ID,
  Reconciliation, Dead-letter.
- **low** Goals (G1–G8) function as both goals and success metrics with no distinct SM-numbered
  section — acceptable for this capability-spec shape (see Shape fit) but worth flagging since nothing
  distinguishes "goal" from "metric" structurally.

## Shape fit — strong

This is correctly shaped as a capability/integration spec, not a consumer-product PRD — there are no
UJs, and that's the right call: the actors (§5) are existing roles whose behavior is explicitly
"unchanged," and the interesting content is data flow, failure handling, and authorization
boundary, not journeys. The brownfield-accuracy bar is met well: every existing-code claim
(`BaseFabricBridgeService`, `WebhookWorker::execute()` catch-and-swallow at
`workers/WebhookWorker.php:140-144`, the 7-of-33 action count) reads as checked against real code,
not asserted. This PRD also correctly treats itself as chain-top-adjacent (feeds
`fabric-engineer`/`fabric-architect` per §13's positive finding) — appropriately lighter on
downstream traceability apparatus than a UX-facing PRD would need, which justifies the "thin"
downstream-usability verdict above being non-fatal.

## Mechanical notes

- **Assumptions Index roundtrip — fails.** NFR-1 contains one inline `[ASSUMPTION: exact ms target
  not yet set...]` (§8), but there is no Assumptions Index anywhere in the document collecting it.
  With only one assumption this is low-cost to fix but the roundtrip as specified by the rubric does
  not exist.
- **ID continuity — mostly clean.** FR-1 through FR-13 are contiguous and unique, with FR-1a as the
  only non-numeric insertion (justified — it's a direct amendment to FR-1's mechanism, not a new
  requirement). NFR-1 through NFR-7 and AC-1 through AC-6 are likewise contiguous. No UJs or SM IDs
  exist to check (see Shape fit — not expected for this PRD type).
- **Glossary drift — not applicable in the negative sense, but no Glossary exists to check drift
  against** (see Downstream usability finding above). Terminology usage itself is internally
  consistent (e.g. "domain" is used the same way in §6, §7, §10 throughout).
- **Cross-references** — section-number references ("§9, decision D1", "see §6", "per FR-5") all
  resolve correctly on inspection; no dangling references found.

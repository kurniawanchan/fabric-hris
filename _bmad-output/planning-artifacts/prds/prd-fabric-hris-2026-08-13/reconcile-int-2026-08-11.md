---
title: 'Reconciliation — [INT] PRD (2026-08-11) vs COMBINED PRD (2026-08-13)'
status: draft
created: '2026-08-13'
---

# Reconciliation: SOURCE ([INT] `prd-fabric-hris-2026-08-11/prd.md`) vs COMBINED (`prd-fabric-hris-2026-08-13/prd.md`)

Scope: judge COMBINED's §3.B, §5, §6/§8, and §7 for omissions, misrepresentations, and factual
errors against SOURCE. COMBINED's §3.B/[INT] rows are an intentional high-level index — under-
detail alone is not a finding.

## 1. Business features SOURCE has that COMBINED's §3.B omits entirely

1. **NONE FOUND at the "completely omitted" bar.** Every SOURCE FR group (write-path anchoring,
   history/query, traceability, integrity verification, resilience) has at least one corresponding
   §3.B bullet. The closest candidates below are judged under-detailed (expected), not omitted:
   - SOURCE FR-7/FR-8 (admin cross-employee history view; per-response leak boundary, AC-7) has no
     explicit §3.B bullet — §3.B only covers self-view via reconciliation/verification framing.
     This is a genuine business capability (admin auditing another employee's own-company history)
     that a reader of §3.B alone would not learn exists. Borderline: arguably index-worthy but not
     "wrong," so listed here as the one candidate rather than a hard omission finding.
   - SOURCE §9's three **named pre-existing Talenta authorization gaps** (`InformalEducationController`
     missing login guard; `/additional-info/index` missing company check; `update-identity-address`
     bypassing `canRequestChangeData` — the last elevated to a **launch pre-requisite for the
     Personal domain**) are not mentioned anywhere in COMBINED's §3.B, §6, §7, or §8. This is not a
     "business feature" in the strict Group-A/B/C/D/E sense, but it is real product-significant
     content (a launch blocker) with no index trace at all — see Category 3, Finding 3 below for
     the fuller writeup; noted here because its absence also weakens §3.B's completeness as a
     feature/capability index.

## 2. SOURCE NFRs COMBINED's §5 table misrepresents, drops, or gets wrong

1. **NFR-7 (reconciliation cadence) is materially altered, not just summarized.** SOURCE NFR-7:
   "daily for Employment, Education & Experience, and Additional Info... hourly" for Personal and
   Payroll. COMBINED §5 row NFR-G: "Hourly (Personal, Payroll); daily (Employment) —
   Education/Additional Info have **no reconciliation signal available at all** (no timestamp
   column in their tables)." COMBINED does not just compress SOURCE's NFR-7 — it adds a new factual
   claim (no timestamp column for two of the five domains) that appears nowhere in SOURCE's NFR-7
   text, which instead requires all three non-sensitive domains to reconcile **daily**, implying a
   timestamp signal exists for all of them. This isn't a case of legitimate under-detail; it is a
   contradiction — SOURCE says Education & Additional Info are cadence "daily" (implying they *do*
   reconcile), while COMBINED's NFR-G row says they have *no* reconciliation signal *at all*. One of
   the two must be wrong, or COMBINED is asserting new information not sourced to [INT]'s PRD as
   the row's "Source" column claims. Matters because it is the same substance flagged in Category 4
   below (the "3-domains-no-reconciliation-signal" gap) — but here it is silently folded into the
   NFR table as if it were part of NFR-7 itself, rather than disclosed as a divergence from what
   NFR-7 actually specifies.
2. **NFR-2/NFR-3 "Production target: TBD" carried correctly** — COMBINED's NFR-B/NFR-C accurately
   reflect SOURCE's TBD framing and the dev-laptop-benchmark caveat. No finding.
3. **NFR-1/NFR-4/NFR-5/NFR-6 carried accurately** as NFR-A, NFR-E, NFR-D, NFR-F respectively — no
   material distortion found for these.
4. **NFR-6a is dropped from COMBINED's §5 table entirely** (no backlog-threshold row at all), but
   this is arguably intentional since NFR-6a is itself just an `[ASSUMPTION]` placeholder deferring
   the number to Phase 4 — its substance (default threshold not yet empirically derived) does
   surface later in COMBINED §8's bullet on backlog thresholds. Not counted as a hard finding, but
   worth flagging since §5's own header claims "consolidated, deduplicated," and NFR-6a-as-SOURCE-ID
   never appears in the ID column, unlike every other SOURCE NFR ID.

## 3. SOURCE constraints/decisions/out-of-scope items COMBINED should have carried forward as an index entry but didn't

1. **D1 (detective-only authorization) — carried forward correctly.** COMBINED's §3.B bullet and §6
   bullet both name D1 explicitly and preserve its "never consulted synchronously" framing and the
   in-band/out-of-band detection-boundary distinction. No finding.
2. **The FR-3 genesis-sentinel requirement is dropped from §6 (constraints) even though its actual
   non-conformance is discussed in §7.** SOURCE FR-3 states the previous-hash for a CREATE SHALL be
   "a defined genesis sentinel (e.g. an all-zero hash), not null or omitted." COMBINED's §7 bullet
   3 discusses the empty-string-vs-sentinel gap (see Category 4 below), but nowhere does COMBINED's
   §6 (constraints/design decisions) index FR-3's *original* requirement itself — a reader who only
   reads §6 would not know an explicit genesis-sentinel design rule was ever ratified, only that a
   gap exists relative to some undescribed baseline. Minor, since §7 does carry the substance, but
   §6 is the more natural section for a ratified design rule like FR-3 to appear.
3. **The three pre-existing Talenta authorization gaps (§9), and specifically the launch
   pre-requisite status of the `update-identity-address` gap for the Personal domain, are not
   carried forward anywhere in COMBINED** — not in §6 (constraints), §7 (known limitations), or §8
   (out of scope). This is the most significant omission in this category: SOURCE explicitly
   elevates one of these three to a **launch blocker** ("it should be fixed in Talenta... before
   this integration's Personal-domain anchoring can be represented as a meaningful integrity
   claim"), and frames the risk that anchoring would otherwise "faithfully immortalize an
   unauthorized bypass as verified history." COMBINED's own §3.A/§3.B narrative repeatedly leans on
   the integrity/detection value proposition ("shift the trust basis... verify mathematically") —
   omitting the one concrete, named case where that exact value proposition is undermined for a
   specific domain is a real index gap, not mere under-detail, especially since COMBINED is meant to
   be the reference a reader consults instead of re-reading all three source PRDs.
4. **SOURCE §4 Non-Goals' explicit `PayrollComponentController`'s ~28-other-actions / custom-field-
   definition-CRUD out-of-scope carve-out is dropped from COMBINED §8.** COMBINED §8 has a generic
   "Building a full HRIS, or logging user activity" bullet that gestures at a similar idea but never
   names this specific, already-litigated scope boundary. Minor — the general non-goal is
   preserved in spirit — but the specific carve-out (useful to a reader deciding whether a future
   payroll feature request is in-scope) is gone.

## 4. Accuracy of COMBINED's §7 "Known Limitations" against the 4 real gaps

1. **tf-1.3 dropped metadata (G-43) — UNDERSTATED / mischaracterized in one respect, otherwise
   accurate.** COMBINED §7 bullet 2 says: "a later verification pass found `correlationID`,
   `operationType`, source endpoint, and client-supplied timestamp are all silently dropped at the
   bridge boundary — useful only within the HRIS platform's own retry/dead-letter bookkeeping,
   never reaching the bridge's logs or the ledger." This matches G-43/tf-1.3's finding closely and
   correctly scopes what still works (talenta-core-side correlation ID tracing) vs. what doesn't
   (end-to-end traceability through the bridge to the ledger). However, COMBINED's bullet omits
   G-43/tf-1.3's finding that `AD-5`'s own claim ("operationType... logged in integration-bridge's
   own request handling") **does not match the actual code** — there is no such logging anywhere.
   That's a documentation-drift symptom (a design doc claiming something the code doesn't do) that
   is arguably worth a phrase in a "known limitations" section aimed at an honest reference, since
   it signals the gap wasn't just unimplemented but was previously *believed* implemented. Not a
   major miss, but a slight understatement of how the gap was discovered/how confidently wrong the
   prior belief was.
2. **tf-1.2 genesis-sentinel wording (G-42) — ACCURATE.** COMBINED §7 bullet 3: "A first-ever
   anchor's 'previous hash' is an empty string, not an explicit all-zero sentinel — functionally
   correct and chaincode-tested, but not literally what [INT]'s own acceptance criteria describe."
   This matches G-42 exactly, including the "not a functional bug" framing and the AC-wording-vs-
   actual-behavior distinction. Confirmed accurate, appropriately low-alarm in tone matching the
   underlying gap's own low severity.
3. **Missing dedicated-hardware benchmark (G-41) — ACCURATE, though the causal framing is
   incomplete.** COMBINED §7 bullet 1: "No dedicated-hardware performance benchmark exists. Every
   latency/throughput number in this project comes from a single shared dev laptop, explicitly
   disclaimed as non-representative." This is factually correct and matches G-41's core finding.
   What COMBINED omits is G-41's *other* half — that several tf-1.x stories (tf-1.2 through tf-1.5)
   remain formally unimplemented as their own AC/test record even though later stories deliver
   equivalent behavior, and that tf-4.3's benchmark AC was never attempted at all because no
   dedicated hardware was available in this environment (not merely "doesn't exist yet" but "was
   not attempted here"). COMBINED's phrasing reads as passive/circumstantial ("no benchmark
   exists") rather than G-41's more pointed "not attempted at all... would only produce a second
   disclaimed dev-environment number." This is a mild understatement of how deliberately the gap
   was left open versus how much it was actively worked around by scope-absorption into tf-4.3.
4. **"3 domains with no reconciliation signal" gap — OVERSTATED relative to SOURCE, and internally
   inconsistent with COMBINED's own NFR-G row.** COMBINED §7 bullet 5: "Reconciliation has no
   signal at all for 3 of 5 domains (Education & Experience's two sub-types, Additional Info) —
   their underlying tables carry no last-modified timestamp to reconcile against." SOURCE's NFR-7
   explicitly assigns these same domains (Employment, Education & Experience, Additional Info — note
   COMBINED's bullet swaps in "Education & Experience's two sub-types" instead of Employment,
   which is itself a discrepancy worth flagging) a **daily** reconciliation cadence, which
   presupposes a working reconciliation signal for them, just measured less frequently than
   Personal/Payroll's hourly cadence. SOURCE's FR-11 also states reconciliation "SHALL identify
   employee changes that exist in Talenta but have no corresponding Fabric anchor" as a
   requirement across the board, with no domain-specific carve-out for "no signal available."
   COMBINED's claim that these domains have **no signal at all** is a new, more severe claim not
   grounded in SOURCE's PRD text and not attributed to any of the four evidence artifacts this task
   was asked to check (tf-1.3/G-41/G-42/G-43) — it does not appear in any of them either. Unless
   this is grounded in some other, unreviewed implementation-time finding, this bullet
   overstates/introduces an unsourced claim, and — as flagged in Category 2 Finding 1 — it
   contradicts COMBINED's own §5 NFR-G row, which asserts the identical "no reconciliation signal"
   claim about the identical two domains (Education/Additional Info) while also correctly quoting
   SOURCE's real cadence numbers in the same row. The self-consistency between §5 and §7 masks that
   both may be introducing information beyond what SOURCE or the four grounding gaps actually
   establish. **This is the single most significant accuracy finding in this reconciliation** — it
   reads as an authoritative, evidence-backed limitation but its evidentiary basis is not traceable
   to any of the documents this task was scoped to check.

## 5. Factual errors in COMBINED's characterization of SOURCE

1. **§5 NFR-G's domain list swap (Employment vs. Education & Experience) — see Category 4, Finding
   4 above.** COMBINED's §7 bullet 5 names "Education & Experience's two sub-types, Additional
   Info" as the two (stated as three, oddly — "2 of 5" is math against a bullet that names what
   reads as 2 domains but is labeled "3 of 5" citing "two sub-types" of one domain to reach 3) as
   having no signal, while §5's NFR-G row separately lists "Employment" as the *daily*-cadence domain
   alongside Education/Additional Info per SOURCE. The two sections do not name the same domain set
   consistently: §5 says daily = {Employment, Education, Additional Info}, matching SOURCE's NFR-7
   verbatim; §7 says no-signal = {Education (x2 subtypes), Additional Info}, silently excluding
   Employment from the "no signal" claim without explanation for why Employment (also daily-cadence
   per SOURCE) would have a signal when the other two don't. This inconsistency is either a
   drafting error or an undisclosed distinction the reader has no way to evaluate.
2. **References section citation for §7's limitations.** COMBINED's References line says
   "`agent-suite/11-execution/grounding-gaps.md` — G-38 through G-43 for the specific evidence
   behind §7's limitations." G-41/G-42/G-43 do support §7 bullets 1–3, but no reviewed evidence
   (G-41/42/43, nor SOURCE's own PRD text) supports §7 bullet 5's specific "no signal at all"
   claim about Education/Additional Info — see Category 4 Finding 4. If G-38/G-39/G-40 (not
   reviewed as part of this task) are the actual source for that bullet, the reconciliation cannot
   confirm this without reading them; as it stands relative to the four artifacts this task was
   scoped against, bullet 5 is unattributed.
3. No other FR numbers, story IDs, or NFR values found misquoted in COMBINED relative to SOURCE.

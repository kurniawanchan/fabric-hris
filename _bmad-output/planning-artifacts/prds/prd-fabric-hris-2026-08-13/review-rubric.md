# PRD Quality Review — prd-fabric-hris-2026-08-13 (Combined Synthesis)

## Overall verdict

This is a synthesis/reference document over three already-final PRDs, and it is judged as that, not as a from-scratch PRD — it correctly defers detailed FRs/BDD to the originals and is unusually honest about what's unverified. The core weakness is not missing content but a **confidence mismatch between §4 (Success Metrics) and §7 (Known Limitations)**: two of the four headline predicates (P1, P3) are stated as clean, achieved facts in the table a reader might stop at, while §7 (and the doc's own sourcing) reveals both are actually untested/unverified claims. A second, smaller gap: §4 has no metric sourced from [INT] despite [INT] carrying a full business-feature section (§3.B) — an asymmetry the doc doesn't acknowledge. Fix both and this is publication-ready as a reference document.

## Decision-readiness — strong

Trade-offs are named with real cost, not smoothed. D1 (detective not preventive authorization) is stated as a decision with an explicit consequence ("explicitly not a defense against an in-band write that went through the app but should have been blocked"), not a soft "consideration." §6's bullet on the `canRequestChangeData` auth gap is a genuine `[NOTE FOR PM]`-shaped callout at a real tension — anchoring an unauthorized write would "immortalize" it as verified history — and it's framed as a launch prerequisite, not a nice-to-have. The PB-1/PB-3 open-gate callout in §6 correctly tells downstream readers to re-check the source PRD rather than assuming resolution. No findings needed here.

## Substance over theater — strong

No persona theater (none exist, appropriately, for a synthesis/reference doc). NFRs carry real numbers and honest status flags (NFR-A1: "<500ms, provisional… pending recalibration"; NFR-B: "TBD — open. Only dev-laptop numbers exist (p95≈26.7s), explicitly disclaimed as non-representative") rather than boilerplate ("must be scalable/secure"). The Vision statement is specific to this system (five named profile-section domains, the trust-shift framing) rather than swappable template prose.

### Findings
- **low** Repeated "documented, defensible default, not yet production-tuned" language (NFR-F in §5, echoed verbatim in §8 Out-of-Scope) — not wrong, just redundant across two sections with no cross-reference between them. *Fix:* have one be the source and the other a one-line pointer ("see NFR-F").

## Strategic coherence — adequate

The thesis ("shift the trust basis from reputation to independent verification") is stated once in §1 and the three business-feature blocks (§3.A/B/C) do trace back to it, though [DASH] (§3.C) is a demonstration surface for the thesis rather than an extension of it — the doc is honest about this ("does not change how anchoring works, only how its performance is watched"), which is the right move rather than forcing false unity.

### Findings
- **medium** §4 Success Metrics has no entry sourced from [INT], despite [INT] owning a full business-feature block (§3.B) with substantial claims (reliable delivery, reconciliation, per-record pseudonyms). The table jumps from [CORE]'s P1–P4 straight to [DASH]'s SM-1–4. Either [INT] genuinely has no top-line success metric worth surfacing at this level (state that explicitly) or one was dropped in synthesis. *Fix:* add a one-line note in §4 explaining the omission, or pull [INT]'s own headline metric in.

## Done-ness clarity — not primary for this doc, but checked

Per the given context, this document defers FR-level acceptance criteria to the source PRDs by design, so this dimension is lighter-weight here. Within what it does assert directly (NFRs, §4 predicates), bounds are given rather than adjectives — no "reasonable performance" or "handles gracefully" language found. No findings.

## Scope honesty — strong

This is the document's best dimension. §7 Known Limitations is unusually candid, including a documentation-drift admission that is rare to see stated this plainly: "a later verification pass found neither claim holds... This is a documentation-drift issue as much as a missing feature: the design record and the shipped code disagreed, and nothing caught it until this was specifically traced." §8 Out-of-Scope is concrete (down to naming `PayrollComponentController`'s ~28 excluded actions) rather than vague. Legal open questions (Pasal 16(1)(e) transfer status, crypto-shredding vs. statutory erasure) are flagged as genuinely undetermined, not answered rhetorically.

### Findings
- **critical** §4's Success Metrics table states P1 ("Every direct-DB manipulation is detected via hash mismatch," threshold "100% across all five sections") and P3 ("Cross-tenant isolation is cryptographic, not logical," threshold "Cross-tenant read denied by MSP") as flat predicate/threshold pairs with no caveat — but §7 discloses that P1 "was not actually tested across all five [sections] at the time [CORE]'s PRD was ratified" (Additional Info untested) and P3 "has never been tested against a real second tenant." A reader who only reads §4 — which is exactly what a synthesis doc's own stated purpose invites, since §7 is easy to skip — comes away believing both are demonstrated facts. *Fix:* add an inline caveat marker in the §4 table itself (e.g. a `†` next to P1 and P3 pointing to §7), not just a separate limitations section the reader has to cross-reference unprompted.

## Downstream usability — matters less (standalone/reference doc), light check performed

The doc explicitly positions itself as pointing back to the three source PRDs for FR/UJ/SM IDs rather than owning them, so ID-contiguity concerns mostly belong to the originals. Within this document, its own IDs (P1–P4, SM-1–4, NFR-A–J) are unique and non-colliding, and none are orphaned.

### Findings
- **low** Glossary (§9) defines 5 terms (Anchor, Detective control, Pseudonym, Reconciliation sweep, Dead-letter) but the body uses several other domain-load-bearing terms without definition here: JCS/RFC-8785 canonicalization, crypto-shredding, `TenantChannelGenesis`, D1. Given the doc's own framing ("read this first to understand the whole"), a first-time reader hits these undefined. *Fix:* either add 3–4 more glossary entries or note explicitly that undefined terms are covered in the cited source PRD.

## Shape fit — strong

The doc doesn't force UJs onto what is fundamentally a capability/integrity-guarantee spec plus a demonstration dashboard — correct call, since none of the three source efforts are consumer-facing multi-stakeholder products. The tiered depth disclosure in §3.C ("deliberately tiered by depth rather than uniform... a ~2-week build-to-defense timeline meant not every area could be fully live") is exactly the kind of honest de-scoping the rubric wants and is shape-appropriate for a thesis-defense artifact.

## Mechanical notes

- **Glossary drift**: none observed — terms that are defined (Anchor, Pseudonym, Dead-letter, etc.) are used consistently wherever they recur.
- **ID continuity**: P1–P4, SM-1–4, NFR-A–J, D1 all unique, no gaps or duplicates within this document. Cross-references to source-PRD IDs (FR-9, AD-5, NFR-9, NFR-6a, etc.) are cited but obviously not independently verifiable from this doc alone — that's expected and disclosed via the References section.
- **Assumptions Index roundtrip**: N/A — this doc uses no `[ASSUMPTION]` tags; given it's synthesizing three already-final PRDs rather than eliciting new requirements, that's appropriate rather than a gap.
- **Structural placement**: §6 (Key Constraints & Design Decisions) and §7 (Known Limitations) sit adjacent and have some conceptual overlap — e.g. the PB-1/PB-3 open gates and the `canRequestChangeData` auth gap in §6 read like limitations too, just framed as "constraints." The boundary (design-time decision vs. discovered-after-the-fact gap) is defensible but not stated explicitly anywhere; a one-line rubric at the top of §6 or §7 distinguishing the two would remove the reader's need to infer it.
- **Required sections for stakes/type**: present and correctly shaped for a "top-level synthesis reference" — Vision, composition diagram, business features, success metrics, NFRs, constraints, limitations, out-of-scope, glossary, references. Nothing is missing that this document's own stated purpose would require.

# Input Reconciliation: PRD vs. Product Brief

**Source input:** `_bmad-output/planning-artifacts/briefs/brief-fabric-hris-2026-08-10/brief.md` +
`addendum.md`
**PRD checked:** `_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-10/prd.md` +
`addendum.md`
**Method:** full read of all four documents; brief walked section-by-section against the PRD's
corresponding structure (Vision → §1, Who This Serves → §2, Solution/five areas → §4, Scope →
§5/§6, Risks → §12, Vision (aspirational) → nowhere).

Most of the brief's content survived the FR-structuring well — the five areas, their tiered depth,
the rationale for excluding live config-editing, the "no frontend stack" exception, and the DSRM
Phase 4/6 framing are all carried through faithfully, often near-verbatim. The gaps below are the
material exceptions: places where the FR/Non-Goal/Success-Metric structure either dropped a brief
item outright or flattened a qualitative nuance (rationale, tone, deliberate ambiguity) into
something more definite or more absent than the brief intended.

---

## Gap 1 — The brief's 4th Success Criterion (thesis-document figures) has no home in the PRD

**Brief, Success Criteria (4th bullet):**
> "The thesis document includes figures/screenshots drawn directly from the dashboard as Phase 4
> Demonstration evidence, distinct from the existing `RESULTS.md` tables."

This is one of exactly four bullets under "Success Criteria" — co-equal with the live-run,
recorded-fallback, and five-areas-demonstrable criteria that all *did* make it into the PRD as
SM-1/SM-2/SM-3 (§7). This fourth one did not. It surfaces once, obliquely, as a JTBD *need*
statement in §2.1 ("As the thesis author, I need figures drawn from this dashboard for the thesis
document itself...") — but a JTBD "need" is not a requirement or a success metric, and nothing
downstream operationalizes it:

- No FR addresses producing, exporting, or capturing a figure/screenshot from the dashboard.
- §7 Success Metrics (SM-1/SM-2/SM-3) never mentions the thesis document.
- §8 Open Questions doesn't list it as deferred, either — it's simply absent, not flagged.
- The only export-adjacent language in the PRD (§4.3's "Out of Scope: Export/download of the
  data") is about Table List's historical numeric data, a different concern from capturing
  dashboard screenshots for the thesis document.

Net effect: a reader of the PRD alone would not know that "the thesis document should end up with
screenshots/figures from this dashboard" was ever a stated success condition, let alone one of only
four. Whether that becomes an FR-9, a Non-Goal, or an Open Question is a judgment call for whoever
owns this next — but right now it's none of the three; it just isn't there.

---

## Gap 2 — The brief's explicit "not committed, but worth naming" Vision is silently foreclosed

**Brief, Vision section (verbatim, the brief's final section):**
> "Not committed, but worth naming: if this holds up through the defense, its live-metrics plumbing
> (Fabric operations endpoints, Caliper CLI tailing) could become this project's lightweight go-to
> visualization for future `QA-4`/`QA-6`-style benchmark runs, rather than a one-time artifact."

The brief is careful here: it explicitly labels this as *not committed*, but still worth writing
down as a live option for a future decision-maker to pick up.

The PRD does not carry this forward in any form — not as a Non-Goal caveat, not as an Open
Question, not as a footnote. Instead, §5 Non-Goals states flatly:
> "This does **not** become this project's permanent frontend stack."

and §6.2 / §12 reinforce the same absolute framing. This isn't factually wrong for v1 scope — but
it silently converts the brief's deliberately-left-open "worth naming, not committed" possibility
into what reads as a closed door, with no trace that the option was ever discussed. A reader
downstream (e.g., whoever eventually decides whether to formalize the "no frontend stack"
exception per Open Question 3) loses the one piece of context that would tell them this was
already flagged as a live possibility worth revisiting, not a rejected idea.

---

## Gap 3 — "Who This Serves" is restructured in a way that drops one named audience and demotes another

**Brief, Who This Serves (verbatim):**
> "**Primary:** the examination committee, watching live during the defense. **Secondary:** thesis
> document readers, via screenshots/an embedded recording of the run; and Mr. Chan himself, for
> rehearsal and confidence going into the defense."

Three named audiences, ranked by primacy: (1) the examination committee — primary, i.e. the
product's main served user; (2) thesis document readers; (3) Mr. Chan, for rehearsal/confidence.

PRD §2 restructures this around Jobs-To-Be-Done + a "Non-Users (v1)" list, and in doing so:

- **The examination committee moves from "Primary" served audience to a listed Non-User**
  ("they observe; they do not operate the dashboard themselves"). This is defensible as an
  *operator* framing (they don't click buttons), but the brief's framing was about who the artifact
  is *for* / who it serves, not who operates it — and on that axis the committee was explicitly
  primary. The PRD's reclassification changes what "success" is oriented around without flagging
  the change.
- **"Thesis document readers" disappears as a named audience entirely.** It's not listed as a
  user, a non-user, or even folded into a JTBD bullet by name — the closest proxy is the "thesis
  author" JTBD (§2.1, 3rd bullet), which is about Mr. Chan's need to produce figures, not about the
  readers who consume them. This compounds Gap 1: the audience the dropped success criterion was
  for is also the audience missing from §2.

This is worth flagging distinctly from Gap 1 because it's a structural framing loss (who the
product serves), not just a missing deliverable requirement.

---

## Gap 4 (minor) — the "keep all five areas" rationale is narrowed from "hedge against the unknown" to "engineering trade-off"

**Brief addendum, Options Considered:**
> "...the stated reason for needing all five was hedging against not knowing what the committee
> will ask about, not a specific per-feature narrative need."

This is a specific, slightly unusual piece of reasoning: all five areas are kept not because each
one individually earns its place in the demo narrative, but because nobody could rule out being
asked about any of them. That's a meaningfully different justification from "we want maximum
feature coverage" or "the reference layout had five tabs so we should too."

The PRD's Vision (§1) and Features intro (§4) preserve *that* five areas are kept at tiered depth,
but the stated reason comes through only as the general two-week-timeline/zero-infrastructure
trade-off (§11 Why Now, §12 Risks) — the specific "hedge against committee questions" rationale for
*why all five, rather than just two,* doesn't appear anywhere in the PRD. A reader would come away
thinking tiering was purely a scheduling compromise, missing the defense-prep-specific reasoning
that all five needed to at least exist.

---

## Gap 5 (minor) — SM-1 quietly moves the success bar from "during the defense" to "during a pre-defense rehearsal"

**Brief, Success Criteria (1st bullet):**
> "Dashboard and Network Profile run live against the real network **during the defense**, showing
> real data as a benchmark executes, with no manual data-faking."

**PRD, SM-1:**
> "...display real, unmocked live data during an actual full-scenario benchmark rehearsal
> **completed before defense day**."

This is a reasonable operationalization (you can't pre-verify a metric against an event that
hasn't happened yet), and it's arguably necessary to make SM-1 checkable at all before defense day.
But it is a real substitution of what's being validated — "worked during the actual defense"
becomes "worked during a rehearsal beforehand" — and the PRD doesn't flag this as a deliberate
narrowing or cross-reference it against Open Question 4 (rehearsal cadence), which is the one place
that could have made the substitution explicit and intentional rather than incidental.

---

## Not gaps — checked and confirmed carried through faithfully

For completeness, these brief items were checked closely and found well-preserved, so they are
*not* included above:

- The five areas' individual descriptions and tiered depth (Dashboard/Network Profile fully live;
  Table List/Notifications/Configuration simplified) — matches almost verbatim, including the
  specific rationale for excluding live config-editing (§4.5/§5 vs. brief's Solution section).
- P1 (tamper-detection premise / independent client-org operation) as Network Profile's narrative
  anchor — PRD §4.2 preserves this precisely.
- The reference-repo research conclusion (tab-layout concept only, no backend to reuse) — PRD §1
  and §5 both state this correctly.
- The "no frontend stack" exception's scope (disclosed, one-off, not a rule reversal; formal
  recording deferred) — PRD §5, §6.1, §8 (Q3), §10 all consistent with the brief and its addendum.
- DSRM Phase 4 (Demonstration) vs. Phase 6 (Communication) framing — PRD §1 and §10 match the
  addendum's citation precisely.
- Risks (live-demo failure history, dev-laptop strain under load, two-week/zero-infra timeline) —
  carried into PRD §12 almost verbatim.
- Live-data transport mechanism being deferred to architecture — correctly kept as Open Question 1
  in the PRD and expanded properly in the PRD's own addendum (SSE/polling/WebSocket options).

---

## Summary for handoff

If only one or two of the above are worth acting on before this PRD is finalized, prioritize
**Gap 1** (missing thesis-document-figures requirement) and **Gap 3** (examination committee
demoted from primary served audience, thesis-document readers dropped as a named audience) — both
affect what "done" looks like and who the product is validated against, not just narrative color.

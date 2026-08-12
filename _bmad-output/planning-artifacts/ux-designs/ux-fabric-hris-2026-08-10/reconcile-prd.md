---
title: 'UX-spine / PRD reconciliation — Caliper Benchmark Dashboard'
created: '2026-08-10'
status: draft
sources:
  - _bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-10/prd.md
  - _bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-10/addendum.md
  - _bmad-output/planning-artifacts/briefs/brief-fabric-hris-2026-08-10/brief.md (referenced by the PRD; consulted to disambiguate Gap 1)
checked-against:
  - _bmad-output/planning-artifacts/ux-designs/ux-fabric-hris-2026-08-10/DESIGN.md
  - _bmad-output/planning-artifacts/ux-designs/ux-fabric-hris-2026-08-10/EXPERIENCE.md
---

# UX-spine / PRD reconciliation

Purpose: find requirements or nuance the PRD stated that DESIGN.md/EXPERIENCE.md silently dropped
or reshaped, especially qualitative/behavioral ideas a visual-token or component-pattern structure
tends to flatten. This is an input-reconciliation pass, not a general UX critique.

## Gap 1 (high severity) — "Recorded" is designed as a live in-app replay mode, but the PRD/brief define the fallback as a separate screen-recording video

**What the PRD/brief actually say:**
- Glossary (`prd.md` L99–100): *"Recorded fallback — a **screen recording** of a prior successful
  live run, rehearsed as the cutover if the live system misbehaves."*
- `brief.md` L21–22: *"...watch throughput, latency, and network topology update in real time, with
  a rehearsed **recorded fallback** if anything misbehaves."*
- `brief.md` L68–69 (Success Criteria): *"A recorded run from a prior successful pass is rehearsed
  and ready as the cutover if the live system falters mid-defense."*
- `brief.md` L61: thesis readers are served "via screenshots/**an embedded recording** of the run" —
  again, a video artifact, not a data-driven UI state.
- `prd.md` UJ-1 edge case (L83–87): *"...he switches to the recorded fallback... **no other
  in-dashboard recovery is in scope**."* — "in-dashboard recovery" is explicitly named as out of
  scope beyond the switchover itself; nothing here specifies the dashboard *rendering* recorded data.

Nowhere in FR-1 through FR-8 is there a requirement for the dashboard to ingest, store, or replay a
prior run's raw data through its own live components. That capability — if built — would need its
own FR (data format for a "replayable" run, which run gets replayed, etc.), and none exists.

**What EXPERIENCE.md builds instead** (State Patterns, L86–90):
> *"Recorded — mode indicator reads RECORDED (violet). Same tabs render the recorded run's data
> identically in layout to how they'd render a live run — the only visible difference anywhere in
> the product between live and recorded is the mode indicator's color and label."*

This is a materially different, larger technical shape than "switch to a video": it requires the
dashboard to hold a second, parallel data path (recorded metrics, recorded topology/health
snapshots, recorded CLI log lines) that feeds the exact same Dashboard/Network Profile/Notifications
components live data feeds. DESIGN.md compounds this by minting a dedicated `status-recorded` violet
token and a whole `mode-indicator.recorded` component (DESIGN.md L18–19, L61–67, L108–113) for a
state that no FR describes.

**A concrete internal inconsistency this produces:** if the fallback really is a screen-recording of
a *prior live run*, then at the moment that recording was captured, any mode indicator on screen
would have read LIVE (green) — because it genuinely was live at capture time. For the audience to
see a RECORDED/violet indicator as EXPERIENCE.md specifies, the dashboard needs a real replay engine
that overlays a different state than what was actually on screen when the footage was taken. That is
either (a) a second, unscoped capability, or (b) a UX spec describing behavior the underlying
artifact (a video file) cannot actually produce.

**Why it matters:** whoever builds this next will read EXPERIENCE.md/DESIGN.md, not the brief. As
written, they'd scope and build a full recorded-data replay mode into every live tab — a
non-trivial addition with zero FR backing, eating into the exact rehearsal/thesis-writing time SM-C1
explicitly protects — or discover late that "RECORDED" mode has no real data source to render from.

**Suggested fix:** either (a) get an explicit product decision on whether the fallback is a literal
screen-recording (simplest: full-screen video takeover, dashboard underneath is irrelevant, no
mode-indicator concept needed at all), or (b) if an in-app replay state really is wanted, add it to
the PRD as its own FR with testable consequences, then let the UX spine's current design stand.
Right now the UX spine has quietly made decision (b) on the PRD's behalf.

## Gap 2 (high severity) — the tenant02 exclusion, the PRD's single most-emphasized scoping decision, gets almost no standing treatment in the UX spine

**What the PRD does:** repeats the `OrgClient-tenant02` exclusion four separate times — FR-3's body
(`prd.md` L159–167), FR-3's testable Consequences (L170–171), FR-4's scope note (L176–177), and a
dedicated Risk item titled *"A fourth, hazardous org accidentally surfacing on screen"* (L358–364)
that exists specifically because an unscoped build "could show this org live to the committee,
contradicting the network's own three-org framing." The addendum reinforces it again (L29–35). This
is not incidental detail — the PRD treats it as a standing hazard the design must actively guard
against, not a one-time fact to note.

**What the UX spine does:** the only place tenant02 is mentioned in either file is one parenthetical
in EXPERIENCE.md's Key Flows: *"the topology diagram (three orgs, scoped per PRD FR-3)..."*
(EXPERIENCE.md L140). Neither DESIGN.md's Do's and Don'ts (L168–175) nor EXPERIENCE.md's Component
Patterns / State Patterns carries any standing guardrail comparable to, e.g., DESIGN.md's own "Don't
introduce a fifth brand color" bullet. A requirement the PRD escalated to a dedicated Risk item is
represented, in the UX spine, at lower weight than a color-palette rule.

**A specific regression this invites:** EXPERIENCE.md's Information Architecture labels tab 2 *"live
topology + per-node health/resource (FR-3, FR-4, FR-5)"* (L34) — but the PRD's own Assumption for
FR-3 (L165–167, restated in §9 L323–325) is that topology is **static and hand-modeled**, explicitly
*not* auto-discovered/introspected from the live network — only the health/resource *readings*
(FR-4/FR-5) are live. Calling the topology itself "live" blurs exactly the line that matters here:
`peer0.tenant02` is a real, running container (`network-docker-compose.yaml`), so a future
implementer who takes "live topology" literally and wires up any form of live discovery is the
*exact* mechanism the PRD's Risk item warns against. The UX spine's own wording nudges toward the
hazard the PRD spent a dedicated Risk section trying to close off.

**Suggested fix:** add an explicit, standing bullet to DESIGN.md's Do's/Don'ts or EXPERIENCE.md's
Component Patterns — e.g., "Don't render any node/org outside the FR-3-scoped three-org set, even if
a live data source would return more — this is a defense-safety requirement, not a styling one" —
and correct the IA line to distinguish "static topology structure" from "live health/resource
overlay."

## Gap 3 (medium severity) — FR-2's explicit-zero requirement has no counterpart in either spine

**What the PRD says** (FR-2 Consequences, `prd.md` L135–137): *"Counts update as results arrive; a
round that completes with 0 failures shows an explicit '0', not just an implied-zero percentage or a
blank field."* This is called out as a specific testable behavior — the PRD anticipated that a
naive implementation would render nothing (or a misleading blank) in the best-case outcome, which is
also the outcome most likely to occur live in front of the committee.

**What's in the UX spine:** DESIGN.md's `metric-tile` component (L68–72, L161–162) and
EXPERIENCE.md's Metric tile pattern (L64–65) describe only generic "value updates in place" behavior
— nothing about zero-value or blank-value rendering. FR-2 itself is also inconsistently named ("rate"
in its title vs. "count" in its body); the UX spine doesn't resolve this either, referring to it only
as "success/failure rate" in the IA (EXPERIENCE.md L33).

**Why it matters:** this is precisely the kind of behavioral nuance a token/pattern-level spec tends
to flatten — "show a number" reads as self-evidently covering "show 0," but the PRD singled out zero
because generic dashboard code (e.g., `{count || null}` guards, or a percentage computed as
`failures/total` with a divide-by-zero guard that short-circuits to blank) commonly fails exactly
this case.

**Suggested fix:** add a one-line State Pattern: "0 failures/successes renders as an explicit '0',
never blank or omitted," and pick "count" or "rate" (or both, labeled) for FR-2's Dashboard display.

## Gap 4 (medium severity) — the PRD's UI-copy generic-register constraint is absent from Voice and Tone

**What the PRD says** (§10, `prd.md` L338–341): *"Generic-register writing discipline. Per
`project-context.md`'s AUTHORITY NOTICE, every written artifact in this project stays in the generic
register — this PRD (**and the dashboard's own UI copy**) must never name the real HRIS
platform/company, consistent with the rest of this project's documentation."* This explicitly scopes
the constraint to the dashboard's shipped UI copy, not just the PRD document itself.

**What's in the UX spine:** EXPERIENCE.md's Voice and Tone section (L49–57) is exactly the section
that owns UI copy — it gives worked examples ("Committed" not "Success! 🎉"; "Unreachable" not
"Oops, something went wrong") — but says nothing about the generic-register constraint, even though
this is the one section a future contributor would read before writing any status string, empty-state
label, or tooltip.

**Why it matters:** this dashboard will ship as real code (a disclosed one-off exception to the
"no frontend stack" rule per PRD §10/§5) — i.e., hard-clean territory under this project's
confidentiality register, not the research-substrate register `_bmad-output/` docs are normally
allowed. A future contributor following only the Voice and Tone section as written has no signal that
copy like an error string echoing an internal system/table name, or a benchmark-scenario label
derived from a real config filename, would be a leak.

**Suggested fix:** add a line to Voice and Tone restating the §10 constraint directly (not just by
reference), since this is the section someone will actually be following while writing strings.

## Gap 5 (lower severity, worth noting) — FR-1's live-lag NFR has no on-screen staleness signal, unlike Network Profile's analogous case

**What the PRD says** (FR-1 Feature-specific NFR, `prd.md` L139–142): *"Must not visibly lag more
than a few seconds behind the underlying benchmark process."* This is a liveness-credibility
requirement — the whole product's premise is proving the system isn't a slide.

**What the UX spine does elsewhere for the same problem:** EXPERIENCE.md's State Patterns explicitly
solves this for Network Profile's resource tiles: *"a down node's... resource-usage tile (FR-5) shows
'—' rather than a stale last-known number, so the operator never mistakes an old reading for a
current one"* (L91–93). No equivalent pattern exists for Dashboard's throughput/latency metric tiles
— if the underlying Caliper process pauses between rounds, stalls, or the polling connection drops
before a hard failure is detectable, nothing distinguishes "frozen because idle" from "frozen because
broken," which is the exact ambiguity a live demo in front of an examination committee is most
exposed to.

**Suggested fix:** extend the same staleness-guard pattern already designed for Network Profile
to Dashboard's metric tiles (e.g., a subtle "as of Xs ago" or last-updated affordance), rather than
leaving Dashboard as the one fully-live tab without one.

## Non-findings (checked, no gap)

- SM-4 (thesis-document screenshots, black-and-white printouts) — explicitly addressed in
  EXPERIENCE.md's Accessibility Floor (L121–123).
- FR-8 read-only contract / no live config editing (§5 Non-Goal) — explicitly addressed via the
  Code block pattern's "no edit affordance of any kind" (EXPERIENCE.md L76–77).
- FR-4's binary healthy/down naming vs. DESIGN.md's added "Slow/warning" third state — a real
  addition beyond the PRD's letter, but disclosed inline as an `[ASSUMPTION:...]` (DESIGN.md
  L104–106), which is the correct way to surface a deliberate deviation — not a silent drop.
- UJ-1's "no other in-dashboard recovery path" (retry, partial-state indicators) — correctly
  reflected as out of scope (no retry/partial-state UI proposed anywhere in either file).
- Single-operator / single-screen / no multi-device scoping — correctly carried into EXPERIENCE.md's
  Foundation section (L15–17) with its own inline assumption tag.

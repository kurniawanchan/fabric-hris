---
title: 'Caliper Performance-Benchmark GUI Dashboard'
status: draft
created: '2026-08-10'
updated: '2026-08-10'
---

# Product Brief: Caliper Performance-Benchmark GUI Dashboard

## Executive Summary

This project's Hyperledger Caliper benchmarks (PT-1 through PT-5, QA-6) already produce real,
live-measured evidence of the prototype's performance — but that evidence currently lives only in
Markdown tables and a generic Caliper HTML report. For a thesis defense, that isn't the same as
*showing* the artifact at work. DSRM's Demonstration activity (Phase 4, per this project's own
`agent-suite/context/DSRM.md`) calls for using the artifact on an instance of the problem, in front
of an audience — and this project's Caliper benchmarks are already scoped as exactly that evidence.

This brief proposes a live, in-browser dashboard — visually modeled on the tab layout of an
open-source reference project, but built fresh against this project's real outputs — that lets Mr.
Chan run a Caliper benchmark live during his thesis defense (in ~2 weeks) and watch throughput,
latency, and network topology update in real time, with a rehearsed recorded fallback if anything
misbehaves. Depth is tiered deliberately: the two most visually load-bearing views are fully live;
the other three are present, at a simpler depth, so all five areas the reference layout sketched
are demonstrable on request.

## The Problem

`qa-tests/performance/RESULTS.md`, `QA6-RESULTS.md`, and Caliper's own `report.html` are this
project's only current way to show benchmark evidence — static documents, read after the fact.
They prove the numbers are real, but they don't *demonstrate* a working system the way a live run
does, and watching a benchmark execute in front of a thesis defense committee is materially more
convincing evidence for a systems thesis than a table of numbers they're asked to trust. There is
currently no visual, real-time way to show a benchmark running, or to reinforce the network's own
multi-org topology story (the independently-operated client org that the tamper-detection premise,
P1, rests on) while the benchmark runs.

## The Solution

A five-area dashboard, visually inspired by the open-source `caliper-gui-dashboard` repo's tab
layout (Dashboard, Notifications, Network Profile, Table List, Configuration) — but **not** a reuse
of that repo's code, which turned out to have no working backend behind its UI shell (see
`addendum.md` for the research). Built fresh, wired to this project's actual outputs, at
deliberately tiered depth to fit a two-week runway starting from zero existing frontend
infrastructure in this repo:

- **Dashboard** and **Network Profile** — fully live. Dashboard streams real throughput/latency/
  success-rate from an actual Caliper run; Network Profile visualizes the live peer/orderer/org
  topology and connection status, sourced from Fabric's own operations `/metrics` endpoints (already
  confirmed reachable during this project's own `G-37` audit).
- **Table List** — a static view over the already-collected `RESULTS.md`/`QA6-RESULTS.md` data. No
  live query, no database.
- **Notifications** — a live tail of the Caliper CLI process's own console output during the run.
- **Configuration** — read-only display of the actual benchmark config YAML driving the live run,
  proving the demo isn't rigged, without a live edit-and-resubmit loop (the single riskiest,
  hardest-to-justify piece of the original reference layout for a rehearsed defense).

## Who This Serves

**Primary:** the examination committee, watching live during the defense. **Secondary:** thesis
document readers, via screenshots/an embedded recording of the run; and Mr. Chan himself, for
rehearsal and confidence going into the defense.

## Success Criteria

- Dashboard and Network Profile run live against the real network during the defense, showing real
  data as a benchmark executes, with no manual data-faking.
- A recorded run from a prior successful pass is rehearsed and ready as the cutover if the live
  system falters mid-defense.
- All five areas are present and demonstrable at their stated depth, so the committee can be shown
  any of them on request.
- The thesis document includes figures/screenshots drawn directly from the dashboard as Phase 4
  Demonstration evidence, distinct from the existing `RESULTS.md` tables.

## Scope

**In, for the defense in ~2 weeks:**
- The five areas above, at the stated tiered depth.
- Lives inside `fabric-hris` — a **disclosed, one-off exception** to `project-context.md`'s "no
  frontend stack" rule, scoped to this thesis-communication artifact, not a reversal of that rule.

**Out, for this version:**
- Live editing/resubmission of a benchmark config from the GUI.
- Any persistence layer (database, historical-run store beyond the existing `RESULTS.md` files).
- Becoming this project's ongoing/permanent frontend stack, or a general-purpose monitoring tool.
- Reusing any code from the reference repo — none of its backend exists to reuse.
- Formalizing the frontend-stack exception (an ADR note or a grounding-gap entry) — a follow-up,
  not part of this brief.

## Risks & Mitigations

- **Live-demo failure.** This exact network has already had two real incidents in this project (a
  `cryptogen` rerun destroying CA trust across three orgs; a stale genesis block after recovery —
  both documented in `docs/QUICKSTART.md`), and a real Caliper load run is documented as straining
  the dev laptop. *Mitigation:* the rehearsed recorded fallback above.
- **Two-week timeline, zero existing frontend infrastructure.** *Mitigation:* tiered depth — only
  Dashboard and Network Profile are built fully live; the other three are simplified, not full
  engineering builds.
- **"No frontend stack" project rule.** This brief proposes crossing it once, disclosed, for a
  bounded purpose. See Scope.

## Vision

Not committed, but worth naming: if this holds up through the defense, its live-metrics plumbing
(Fabric operations endpoints, Caliper CLI tailing) could become this project's lightweight go-to
visualization for future `QA-4`/`QA-6`-style benchmark runs, rather than a one-time artifact.

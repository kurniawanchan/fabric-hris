---
title: 'Caliper Performance-Benchmark GUI Dashboard'
status: final
created: '2026-08-10'
updated: '2026-08-11'
sources:
  - _bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-10/prd.md
  - _bmad-output/planning-artifacts/briefs/brief-fabric-hris-2026-08-10/brief.md
---

# Experience: Caliper Benchmark Dashboard

## Foundation

**Form factor:** web, single browser window/tab, run locally on the same laptop used for the
defense — no mobile, no multi-device handoff. `[ASSUMPTION: single-window is sufficient; the PRD
names one operator, one screen (Non-Goals §5, MVP §6.2 "single operator, single screen").]`

**UI system:** shadcn/ui on React + Tailwind (see `DESIGN.md`'s frontmatter `description` for the
`[ASSUMPTION]` tag explaining why). This EXPERIENCE.md specifies only the behavioral delta on top
of shadcn's own component behavior — shadcn's default interaction patterns (focus rings, dialog
behavior, toast timing) are inherited, not restated.

**Environment:** projected to a screen for the examination committee. This one fact shapes several
decisions below (larger metric type in `DESIGN.md`, the Accessibility Floor's viewing-distance
requirement, and the always-visible mode indicator) more than any other single constraint.

## Information Architecture

Five top-level tabs, matching the PRD's five features exactly — no sub-navigation beneath any of
them; each tab's content fits on one screen without its own internal routing.

1. **Dashboard** (default/landing tab) — live throughput, latency, success/failure rate (FR-1, FR-2).
2. **Network Profile** — static, hand-modeled topology diagram (FR-3, scoped to three orgs only —
   see the standing guardrail below) with live per-node health/resource overlaid (FR-4, FR-5).
3. **Table List** — static historical runs (FR-6).
4. **Notifications** — live CLI tail (FR-7).
5. **Configuration** — read-only active benchmark config (FR-8).

Dashboard is the entry point (UJ-1 opens here) because it's the tab that most directly answers "is
this really running" — the thing the whole product exists to demonstrate. Tab order in the nav
follows this same priority: the two fully-live tabs (Dashboard, Network Profile) come first, the
three simplified tabs follow.

**Persistent chrome across all five tabs:** the mode indicator (Idle / Live, see State Patterns)
and a benchmark-scenario selector (which `benchconfigs/*.yaml` run is active — feeds
Configuration's display and scopes Dashboard/Network Profile/Notifications to that run). Table
List is the one tab this selector doesn't affect — it always shows the full historical set.

## Voice and Tone

Plain, precise, unembellished — status text states the fact, not a reaction to it. "Committed" not
"Success! 🎉"; "Unreachable" not "Oops, something went wrong." This is a credibility posture as
much as a style choice: an academic committee reads breathless UI copy as compensating for
substance, and this product's whole thesis is that the substance is real.

Empty/idle states are calm and factual: before a run starts, Dashboard reads "No benchmark
running" — not an apology, not a call-to-action, just the true current state.

**Generic-register constraint (PRD §10).** Every string of UI copy in this product — labels,
empty states, tooltips, the Configuration tab's own chrome around the YAML it displays — stays in
this project's generic register: never the real HRIS platform/company name, never a real internal
table/column name. This dashboard shows benchmark mechanics (throughput, latency, topology), never
profile-section content, so this constraint is easy to hold in practice — but it applies to every
piece of copy written for this product, not just the data it happens to display.

## Component Patterns

- **Status badge** (peer/orderer health) — always icon *and* color *and* text label together (see
  Accessibility Floor). Three states: Healthy, Slow, Unreachable. **Scoped to the three
  `TenantChannelGenesis` orgs only, always** — see the standing guardrail below.
- **Metric tile** — value updates in place (no layout shift, no flash-of-new-value animation
  longer than ~150ms) so a room watching a projected screen can track the number changing without
  the tile itself becoming distracting. A value of zero renders as a literal "0" (per FR-2), never
  a blank field or an omitted tile. If no update has arrived within roughly FR-1's "a few seconds"
  NFR window, the tile visually mutes (e.g., dimmed text) rather than silently continuing to show
  an increasingly stale number as if it were current — the same staleness discipline Network
  Profile's node-down state already applies (below).
- **Mode indicator** — always visible, never collapsible, same position across all five tabs.
  Reads Idle or Live *only* — the dashboard itself has no "recorded" state. Clicking it does
  nothing (it's a status readout, not a control). **Recorded-fallback guardrail:** per the PRD/
  brief, the recorded fallback is a separate screen-recording video that's switched to entirely
  outside this dashboard (a different window/player) if the live system falters — the dashboard is never
  put into a "replaying" mode, and must not be built to imply otherwise. `[ASSUMPTION: no in-app
  replay/recording-playback feature exists (see Anti-patterns below for why an earlier draft's
  version of this was rejected); UJ-1's edge case describes switching away from the dashboard, not
  a mode within it.]`
- **Log line** (Notifications) — new lines append at the bottom and auto-scroll, unless the
  operator has manually scrolled up to read history, in which case auto-scroll pauses until they
  scroll back to bottom (standard "log tail" convention — prevents yanking the view away mid-read).
- **Code block** (Configuration) — read-only, no edit affordance of any kind (no cursor, no
  focus ring on click) — reinforces FR-8's read-only contract visually, not just functionally.

**Standing guardrail — `OrgClient-tenant02`/`peer0.tenant02` exclusion (PRD §4.2/§9/§12).** Every
Network Profile component (topology diagram, status badge, resource tile) renders the three
`TenantChannelGenesis` orgs only, unconditionally — never tenant02, even if a future data source
technically returns it. This is the PRD's single most heavily-flagged decision (its own dedicated
Risk entry) precisely because tenant02 is a real, running, but deliberately abandoned and
previously-hazardous container; surfacing it live in front of the committee would contradict the
network's own three-org framing. Treat any future change here as requiring an explicit PRD update,
not a component-level judgment call.

## State Patterns

- **Idle** — no benchmark selected/running. Dashboard/Network Profile/Notifications show a calm
  "No benchmark running" empty state; Table List and Configuration are unaffected (they don't
  depend on a live run).
- **Live** — mode indicator reads Live (green). Dashboard/Network Profile/Notifications update
  continuously per FR-1/FR-2/FR-4/FR-5/FR-7's "no manual refresh" requirement.
- **Recorded fallback (out-of-app, not a dashboard state)** — if the live system falters, Mr. Chan
  switches away from this dashboard entirely to play a screen-recording video of a prior successful
  run, per the PRD/brief. The dashboard has no "recorded" mode to design for — this product's
  responsibility ends at being reliable while live, and at not implying it's live when it isn't.
- **Node slow** (Network Profile) — a node whose operations-endpoint response latency crosses a
  threshold (exact value TBD at implementation — `[ASSUMPTION: a threshold will need tuning against
  the real network under real benchmark load; not specified here since no such number was given.]`)
  shows Slow (amber), distinct from both Healthy and Unreachable. Its resource-usage tile keeps
  showing live numbers (unlike Node down's "—") — the node is still responding, just slowly — so the
  operator can see whether "slow" means "degraded but working" or is trending toward "about to go
  down."
- **Node down** (Network Profile) — a down node's status badge reads Unreachable (red); its
  resource-usage tile (FR-5) shows "—" rather than a stale last-known number, so the operator never
  mistakes an old reading for a current one.
- **Never-anchored-yet** is not a state this product needs to model — Dashboard/Network
  Profile/Notifications only ever show data for a run that is either idle, live, or recorded;
  Table List's data source (`RESULTS.md`/`QA6-RESULTS.md`) is populated as of build time (FR-6), so
  an empty Table List is not an expected state in production use of this tool.

## Interaction Primitives

- **Tab switching** — standard shadcn tabs, single click, no confirmation, no data loss risk (every
  tab is read-only or live-streamed, nothing to lose by navigating away).
- **Benchmark-scenario selector** — a dropdown in the persistent chrome; changing it does not
  restart or affect an in-progress live run (per FR-1's own scope, the selector reflects which
  scenario is active; it doesn't control the Caliper process itself — that's the terminal, per
  UJ-1). On first load, before any scenario has ever been chosen, it defaults to the first entry in
  `benchconfigs/` alphabetically — distinct from the already-defined Idle state (a scenario is
  selected, just not running).
- **Section filter** (Table List, optional) — `[ASSUMPTION: Table List may benefit from a simple
  filter by benchmark type (PT-1/2, PT-3, PT-5, QA-6) given 5+ historical scenarios exist, but this
  is not named in any FR — treat as a nice-to-have, not a requirement, if time is short.]`
- No drag, no multi-select, no keyboard shortcuts beyond what shadcn's components provide by
  default — nothing in this product's flows benefits from them given a single operator who is
  narrating, not power-using, during the one session that matters.

## Accessibility Floor

- **Viewing-distance legibility is the primary accessibility concern here**, ahead of the usual web
  a11y checklist — the audience reads this from committee-room distance on a projected screen, not
  up close. Metric type (`DESIGN.md typography.metric`, 32px) and status badges must remain legible
  at that distance; verify against the actual defense room before finalizing, not just on a laptop
  screen close up.
- **Status is never color-only** — every status badge and the mode indicator pair color with a text
  label (Healthy/Slow/Unreachable, Idle/Live) so the distinction survives for anyone
  color-blind in the room, and for black-and-white printouts of thesis-document screenshots (SM-4).
- Standard shadcn/Radix keyboard-navigability and focus-visible behavior inherited as-is — not a
  deep investment area for a single-operator, one-session tool, but nothing here should regress
  what shadcn provides for free.

## Inspiration & Anti-patterns

**Inspiration:** the open-source `caliper-gui-dashboard` repo's *tab-layout concept only* —
Dashboard, Notifications, Network Profile, Table List, Configuration as five top-level areas. That
repo's actual UI (a "paper-dashboard" template) and its own visual identity are not the
inspiration; this product's brand and behavior are specified fresh in `DESIGN.md`/this file.

**Anti-patterns, explicitly rejected (PRD §5 Non-Goals, brief §"Options considered"):**
- **Reusing that repo's code.** It turned out to have no working backend behind its UI shell — a
  five-tab React template with zero data wiring. Nothing here is built from it.
- **Live benchmark-config editing/resubmission.** The Configuration tab could have let an operator
  edit and rerun a config from the browser, mirroring the reference repo's original vision. Rejected
  as the single riskiest, least-justified piece of that vision for a *rehearsed*, not improvised,
  defense — read-only display (FR-8) proves the demo isn't rigged without that risk.
- **An in-app "recorded playback" mode.** An earlier draft of this file specced the dashboard itself
  entering a replay state for the recorded fallback. Corrected during reconciliation against the
  PRD (see `reconcile-prd.md`) — the fallback is a separate video, outside this product entirely.

## Key Flows

**UJ-1. Mr. Chan runs the live benchmark during his defense.** (Elaborates the PRD's UJ-1 at
screen level.)

Entry state: dashboard open in a browser window on the shared screen, Dashboard tab active (the IA
default), mode indicator reading Idle / Live depending on whether a run is already selected.

1. Mr. Chan starts a rehearsed Caliper benchmark from the terminal (outside the dashboard, per the
   Component Patterns note above).
2. Within a few seconds, Dashboard's metric tiles begin updating — throughput and latency numbers
   changing in place, no page reload, no layout shift.
3. He switches to Network Profile; the topology diagram (three orgs, scoped per PRD FR-3) shows
   live status badges per node, updating as the benchmark load moves through the network.
4. He narrates against what's on screen, pointing at specific tiles/badges as they change.
5. **Climax:** a metric tile updates *while he's mid-sentence*, in front of the committee — the
   moment that proves this is a live system, not a slide.
6. **On request** (per SM-2's "all five areas reachable and demonstrable"): if a committee member
   asks to see something else, he clicks over to Table List to show the already-collected
   historical runs, Notifications to show the live Caliper CLI tail as further proof the run is
   real, or Configuration to show the exact benchmark config in use — each a single tab-click away,
   none requiring him to leave Dashboard/Network Profile's live run.
7. **Edge case:** if a node's badge flips to Unreachable, or a metric stalls/mutes, he switches
   away from this dashboard entirely — to a separate video player showing the recorded fallback —
   and continues narrating against that instead. This is a window/application switch outside the
   product, not a state the dashboard itself enters.

Resolution: the benchmark round completes; Dashboard's final numbers stay on screen (not cleared),
so the committee can see the completed result while he transitions to the next part of the
defense. If the recorded fallback was used instead, the same resolution applies to that video's
final frame.

# Reviewer Gate — Good-Spine Checklist Pass

Spine: `_bmad-output/planning-artifacts/architecture/architecture-fabric-hris-2026-08-11/ARCHITECTURE-SPINE.md`
Sources checked: PRD (`prd.md` + `addendum.md`), UX (`DESIGN.md` + `EXPERIENCE.md`), and both prior
reconciliation passes (`reconcile-prd.md`, `reconcile-ux.md`).

## Verdict

**Approve with minor revisions.** No showstopper — nothing here would silently produce two
incompatible builds. Both prior reconciliation passes' fixes verified as correctly landed (detail
below). Three residual/new findings, all fixable as small spine edits, not re-designs.

## Checklist walk

### 1–2. Divergence points fixed / ADs enforceable
All 9 ADs have a concrete, checkable Rule (grep-able or code-review-able: loopback bind, no
`spawn`/`exec` of Caliper, exactly one SSE endpoint, three-value status enum, hand-modeled topology
constant, etc.). No AD's Rule is vague enough to fail to prevent its stated divergence. See Finding
1–2 below for two places the Rule is *present* but *incomplete* relative to its source doc.

### 3. Deferred section safe
Four of five Deferred items are genuinely below-altitude (backend library choice, Table List
filter, doc-formalization follow-up, post-defense fate). The fifth — "the Slow threshold's exact
value" — is not fully safe as scoped; see Finding 3: it defers a *number* but the spine never
states what *dimension* that number measures, which is a contract question, not a tuning one.

### 4. Named tech plausibility
Node 24.x, Express 5.2.1, React 19.2.7, Vite 8.x/8.2, shadcn/ui CLI v4 all read as plausible
version lines for August 2026 given each project's release cadence — nothing invented or stale.
**However:** the Stack table is incomplete — see Finding 4 (Tailwind CSS, which DESIGN.md names
explicitly as a direct dependency, has no row and no version).

### 5. Capability → Architecture Map coverage
All 8 FRs (FR-1…FR-8) appear in the map, each with a "Lives in" + "Governed by" pair. Confirmed
against the PRD's own FR-1…FR-8 list — no FR silently dropped, and FR-3 (the one prior reconcile
flagged as excluded from AD-4's `Binds:` line) is now present in AD-4's binds list. Verified fixed.

### 6. Whole dimensions left silent
Deployment/environment/operations dimension is thin but not silent: the PRD itself scopes this to
"local machine only, no hosting" (§6.2), and AD-9 (run-mode discipline) plus the Structural Seed's
README note cover the two-process start sequence and dev-vs-defense-day build mode. Given the
bounded, single-operator, laptop-only scope, this is an adequately-decided dimension, not a gap.

## Findings

### Finding 1 (Medium) — AD-5 doesn't capture that "selected scenario" and "Live/Idle" are independent axes

EXPERIENCE.md's Interaction Primitives section is explicit that a scenario auto-selected on first
load ("defaults to the first entry in `benchconfigs/` alphabetically") is **"distinct from the
already-defined Idle state (a scenario is selected, just not running)"** — i.e., Idle/Live is not a
derived side-effect of scenario-selection, it's an orthogonal axis with its own first-load default
behavior. AD-5 states the two singletons ("which scenario," "whether live") but never states they
are independent, nor names the first-load default-selection rule. A builder reading only AD-5 could
plausibly wire "Live" as *following from* scenario selection (e.g., treat "just selected" as
momentarily live) rather than keeping the two orthogonal, which is exactly the kind of misreading
the prior UX reconcile's Gap 3 was raised to prevent — the general shape (a Live flag exists) was
fixed, but this narrower independence/first-load nuance was not carried into the fix.

**Fix:** add one sentence to AD-5: "Scenario selection and live/idle detection are independent —
selecting a scenario does not itself imply Live, and on first load a scenario is auto-selected
(first entry in `benchconfigs/` alphabetically) while the backend still reports Idle until watcher/
heartbeat activity is detected."

### Finding 2 (Low) — "Slow" node-status criterion's dimension is unstated, only its threshold is deferred

EXPERIENCE.md defines "Slow" as "a node whose **operations-endpoint response latency** crosses a
threshold (exact value TBD)." The spine's Deferred section defers only "the exact value," implying
the *dimension* (response latency of the poll itself) is already settled — but no AD or Consistency
Convention actually states that dimension. A builder could reasonably instead key "Slow" off CPU/
memory usage crossing a threshold (data it's already collecting for FR-5), which would satisfy
every literal AD but contradict the UX's basis for the tier.

**Fix:** state the dimension explicitly in AD-6 or the poller's structural-seed comment — "Slow"
is triggered by the poller's own operations-endpoint response latency, not by the resource-usage
values it reports.

### Finding 3 (Low) — Tailwind CSS is a named dependency in the UX source but absent from the Stack table

DESIGN.md's own frontmatter states "shadcn/ui on React + Tailwind" and shadcn/ui's CLI v4 Vite
template installs Tailwind as a hard dependency. The spine's Stack table lists Node.js, Express,
React, Vite, shadcn/ui, and TypeScript, but has no Tailwind row/version. Since checklist item 4
asks whether named tech is verified-current, the omission means Tailwind's version is neither
pinned nor checked for plausibility — a builder has no stated version to verify against, and the
Stack table reads as complete when it isn't.

**Fix:** add a Tailwind CSS row to the Stack table (version pulled from whatever `shadcn@latest
init -t vite` resolves at build time, noted as "whatever the CLI pins" if a fixed version isn't
worth hand-specifying).

## Verified: prior reconciliation fixes landed correctly

- `reconcile-prd.md` Gap 1 (FR-3 missing from AD-4's binds) — fixed, FR-3 now listed.
- `reconcile-prd.md` Gap 2 (no stated loopback-bind rule) — fixed as AD-8.
- `reconcile-prd.md` Gap 3 (no confidentiality invariant for pass-through content) — fixed as the
  "Passthrough-content confidentiality" Consistency Convention row.
- `reconcile-ux.md` Gap 1 (three-tier status has no home in the contract) — fixed: AD-4 now states
  the `status` field's exact three-value enum.
- `reconcile-ux.md` Gap 2 (Slow vs Down resource-tile semantics collide with AD-6's null rule) —
  fixed: AD-6 now explicitly splits "unreachable → null" from "slow → real numbers" as two paths.
  (Finding 2 above is a narrower residual gap in the same area, not a re-opening of this one.)
- `reconcile-ux.md` Gap 3 (no Idle/Live flag in backend state) — fixed: AD-5 now includes the live-
  detected flag. (Finding 1 above is a narrower residual gap in the same area.)
- `reconcile-ux.md` Gap 4 (no write path for scenario selection) — fixed: AD-4 states `POST
  /api/scenario`, and the Structural Seed's routes list includes it.
- `reconcile-ux.md` Gap 5 (staleness-muting has no signal) — fixed: AD-6 adds the fixed-interval
  `heartbeat` SSE event.

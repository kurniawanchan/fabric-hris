# Spine Pair Review — Caliper Performance-Benchmark GUI Dashboard

## Overall verdict

The spine is fundamentally sound and shows real reconciliation discipline: `reconcile-prd.md`
caught a genuine over-build in an earlier draft (an in-app "RECORDED" replay mode with its own
violet token) and the fix — mode indicator reduced to Idle/Live only, with the removal disclosed
inline via an `[ASSUMPTION:...]` tag — is cleanly reflected in both current files. It falls short
of "cleanly source-extractable" in a handful of concrete spots: a token reference that cannot
resolve (`{rounded.full}`), three status colors missing their dark-mode pairs while the fourth
(primary) has one, a flagship component (the topology diagram) with zero visual specification, an
incomplete state model for one of three declared status-badge states, and one dangling
cross-reference to a DESIGN.md section that doesn't exist. None of these block a competent
implementer, but each would force them to guess or invent rather than extract.

## 1. Flow coverage — adequate

The PRD names exactly one UJ (§2.3, UJ-1); EXPERIENCE.md's Key Flows section covers it 1-for-1,
verbatim-titled ("Mr. Chan runs the live benchmark during his defense"), with a named protagonist,
6 numbered steps, an explicit **Climax** beat (step 5), and an explicit **Edge case** functioning as
the failure path (step 6, correctly reflecting the PRD's "no other in-dashboard recovery is in
scope" constraint). Mechanically, this is full coverage: 1 of 1 UJs, fully shaped.

### Findings
- **medium** The sole Key Flow only ever visits Dashboard and Network Profile; it never tours Table
  List, Notifications, or Configuration, even though the PRD's SM-2 requires all five areas to be
  "reachable and demonstrable on request during the defense" (`prd.md` §7) and EXPERIENCE.md's own
  IA lists all five as tabs (EXPERIENCE.md lines 33–38). A downstream consumer gets IA/State/
  Component-level description of the three simplified tabs but no scripted usage moment for any of
  them. *Fix:* add a short "on request" beat to UJ-1 (e.g., "if asked, he clicks over to Table
  List/Notifications/Configuration to show historical data or the driving config") or a minimal
  second flow for that path.

## 2. Token completeness — thin

Frontmatter defines 10 color tokens, 3 typography roles, an (empty, intentionally-inherited)
`rounded` block, one spacing override, and 5 components. Most `{path.to.token}` references resolve
correctly (e.g., every `{colors.status-*}` and `{typography.*}` reference in `components:`, and
EXPERIENCE.md's own `DESIGN.md typography.metric` citation at line 141 correctly resolves to 32px).
Two concrete breaks:

### Findings
- **critical** `colors.status-healthy`, `status-healthy-foreground`, `status-warning`,
  `status-warning-foreground`, `status-down`, and `status-down-foreground` (DESIGN.md lines 12–17)
  have no `-dark` counterparts anywhere in frontmatter or prose, while `colors.primary` and
  `primary-foreground` do (lines 8–11, restated in prose at line 97: "`#1E3A5F` light / `#7FA8D9`
  dark"). These three status colors are brand-new tokens, not shadcn-inherited ones — unlike
  `background`/`foreground`/`muted`, there is no implicit shadcn dark-mode fallback for them, so if
  this product ever renders in a dark context, the single most safety-critical signal in the UI
  (node health) has no defined dark value. *Fix:* add `status-healthy-dark`, `status-warning-dark`,
  `status-down-dark` (+ foreground variants) to match the treatment already given to primary, or
  state explicitly that this product targets light mode only.
- **high** `{rounded.full}` is referenced three times in `components.status-badge.*.radius`
  (DESIGN.md lines 50, 54, 58) and once more in prose ("status badges and the mode indicator use
  `rounded/full`," line 145), but the `rounded:` frontmatter block (lines 37–39) contains no keys —
  only a comment stating "shadcn defaults inherited as-is." Per the design-md-spec, a
  `{path.to.token}` reference resolves by following the YAML structure; `rounded.full` does not
  exist anywhere in this document. Elsewhere in the same file, purely-inherited shadcn tokens are
  referenced as bare literals (`'muted'`, `'md'`, `'card'`) rather than bracket-paths — the
  status-badge radius entries should follow that same convention (`'full'`) instead of the
  unresolvable `{rounded.full}`. *Fix:* change the three `radius: '{rounded.full}'` lines to
  `radius: 'full'` (bare literal, consistent with `metric-tile.radius: 'md'` and
  `code-block.radius: 'md'` elsewhere in the same file).

## 3. Component coverage — thin

The five components with dedicated visual specs (status-badge, mode-indicator, metric-tile,
code-block, log-line) match exactly between DESIGN.md.Components (frontmatter + prose, lines
45–77 / 151–163) and EXPERIENCE.md.Component Patterns (lines 69–92) — identical names, real
behavioral rules on the EXPERIENCE.md side (not one-word descriptions), real visual rules on the
DESIGN.md side.

### Findings
- **high** The topology diagram — Network Profile's primary visual artifact, and one of the two
  "fully live" flagship views per the PRD's tiering decision — is named repeatedly (EXPERIENCE.md
  IA line 34, the standing guardrail's own component list at line 95, Key Flow step 3 at line 163)
  but has no corresponding row anywhere in DESIGN.md.Components. Status-badge, metric-tile,
  mode-indicator, code-block, and log-line all get anatomy/color/sizing specs; the topology diagram
  — node/edge rendering, how the three orgs are visually grouped, layout logic — gets none. *Fix:*
  add a `topology-diagram` (or equivalent) entry to DESIGN.md.Components with at minimum: node
  shape/grouping-by-org convention, how status badges attach to nodes, and layout orientation.
- **low** The persistent benchmark-scenario selector (a dropdown; EXPERIENCE.md IA line 46,
  Interaction Primitives line 126) and the 5-tab top navigation ("standard shadcn tabs," line 124)
  appear only under EXPERIENCE.md's Interaction Primitives, never under Component Patterns, and have
  no acknowledgment anywhere in DESIGN.md — not even a "used as-is, unchanged" note. The reference
  example (Drift's DESIGN.md) explicitly lists shadcn components used unchanged (Button, Card,
  Dialog, Sheet, Popover, DropdownMenu, Toast, Tabs, Avatar, Separator); this DESIGN.md has no
  equivalent list. Nothing is actually broken (both are plausibly pure shadcn defaults), but it's a
  discoverability gap for anyone building from this spine alone. *Fix:* add one sentence to
  DESIGN.md.Components naming the shadcn primitives used as-is (at minimum Tabs and Select).

## 4. State coverage — thin

Idle, Live, Recorded-fallback (correctly modeled as out-of-app, not a dashboard state), and Node
down are all covered with concrete, testable treatment (EXPERIENCE.md lines 105–120), including the
explicit non-applicability note for "Never-anchored-yet."

### Findings
- **high** DESIGN.md and EXPERIENCE.md both establish a three-state status-badge vocabulary
  (Healthy/Slow/Unreachable — EXPERIENCE.md line 70, DESIGN.md's `components.status-badge.warning`),
  but State Patterns only walks through two of the three states for Network Profile: "Live"
  (healthy, implicitly) and "Node down" (Unreachable, lines 114–116, which explicitly specifies the
  resource tile shows "—"). There is no State Patterns entry for "Slow": no trigger
  threshold/condition, and no statement of what the resource-usage tile shows in that state. This
  state is itself an `[ASSUMPTION:...]`-tagged deviation from the PRD's binary FR-4 (correctly
  disclosed in DESIGN.md line 102), which makes its incomplete behavioral spec more notable, not
  less. *Fix:* add a "Slow" row to State Patterns paralleling "Node down" — trigger condition
  (e.g., "operations-endpoint latency above a threshold") and resource-tile treatment.
- **low** The persistent benchmark-scenario selector's value on first load — before an operator has
  ever chosen a scenario — is undefined. Neither Information Architecture nor State Patterns states
  what Dashboard/Network Profile/Configuration render before any scenario has ever been selected, as
  distinct from the already-defined "Idle" state (a scenario selected, no run in progress). *Fix:*
  add one line — e.g., "on first load, the selector defaults to [X]" or "Configuration/Dashboard show
  a 'select a scenario' prompt until one is chosen."

## 5. Visual reference coverage — strong

Zero orphans, as expected: this run has no `mockups/`, `wireframes/`, or imports content by design
(Fast path, spine-only). Confirmed `imports/` exists but is empty. Not a gap — the correct outcome
for this run's stated scope.

## 6. Bloat & overspecification — strong

Content is proportionate to a 2-week, single-operator, thesis-defense tool. Every `[ASSUMPTION:...]`
tag found is load-bearing (the shadcn choice, the Slow/warning third state, the dropped in-app
recorded mode, the single-window scoping) rather than decorative. `reconcile-prd.md`'s prior pass
caught and removed a genuinely over-built capability — an in-app "RECORDED" replay mode requiring a
parallel recorded-data path through every live component — that would have been real bloat relative
to what the PRD actually asked for; the fix is cleanly reflected in the current files. No findings.

## 7. Inheritance discipline — adequate

`sources:` in EXPERIENCE.md's frontmatter resolves — both `prd.md` and `brief.md` exist on disk at
the cited paths. UJ-1's name is verbatim from the PRD (§2.3). FR-1 through FR-8 are cited
consistently by ID across both files with no title/number drift (verified against `prd.md`'s own
FR-1…FR-8 headings). All five shared component names match exactly between
DESIGN.md.Components and EXPERIENCE.md.Component Patterns. The one explicit cross-file token-name
reference (`DESIGN.md typography.metric`, EXPERIENCE.md line 141) resolves correctly to 32px.

### Findings
- **medium** EXPERIENCE.md's Foundation section (line 19) reads "see `DESIGN.md`'s Foundation note
  for why" — DESIGN.md has no section titled "Foundation" (its sections are Brand & Style, Colors,
  Typography, Layout & Spacing, Elevation & Depth, Shapes, Components, Do's and Don'ts). The
  shadcn-choice rationale this points at actually lives in DESIGN.md's frontmatter `description:`
  field as an inline `[ASSUMPTION:...]` tag, not in any body section. A consumer following this
  pointer literally will not find what it promises. *Fix:* either add a short "Foundation" note to
  DESIGN.md's Brand & Style opening (restating the shadcn-choice rationale in prose, not just
  frontmatter), or reword the EXPERIENCE.md pointer to cite the frontmatter description directly.

## 8. Shape fit — adequate

DESIGN.md follows the canonical section order exactly: Brand & Style → Colors → Typography →
Layout & Spacing → Elevation & Depth → Shapes → Components → Do's and Don'ts, with no invented
sections. EXPERIENCE.md keeps canonical ordering for the sections it has (Foundation → Information
Architecture → Voice and Tone → Component Patterns → State Patterns → Interaction Primitives →
Accessibility Floor → Key Flows) and invents nothing.

### Findings
- **medium** EXPERIENCE.md drops two defaults relative to the reference shape. "Responsive &
  Platform" is fully defensible — Foundation explicitly states single-window, single-device, "no
  mobile, no multi-device handoff," so there is genuinely nothing to say. "Inspiration &
  Anti-patterns" is only partially defensible: real material exists for it — the reference repo's
  tab-layout inspiration, the rejected code-reuse from that repo, the rejected live-config-editing
  loop — and it currently lives only in the PRD/brief (§1, §5, Non-Goals), never synthesized into
  the spine itself. A consumer reading only EXPERIENCE.md loses this "why this shape, not that one"
  trail. *Fix:* either add a short Inspiration & Anti-patterns section carrying the PRD's own
  rejected-alternatives material, or note explicitly in Foundation why it's omitted.

## Mechanical notes

- **Broken cross-reference:** EXPERIENCE.md line 19 ("see `DESIGN.md`'s Foundation note for why")
  points at a DESIGN.md section that does not exist — see Finding under §7 above.
- **Frontmatter completeness:** DESIGN.md's frontmatter matches the canonical shadcn-example shape
  (`name`/`description`/`colors`/`typography`/`rounded`/`spacing`/`components`), with no
  `status`/`created`/`updated` fields — consistent with the reference example
  (`design-example-shadcn.md`), not a defect. The `rounded:` block is present as a key with no
  values (comment-only) — valid YAML, but see the `{rounded.full}` finding under §2 for the
  resolution consequence.
- **Frontmatter key naming:** EXPERIENCE.md uses `title:` where the reference example (Drift) uses
  `name:`. This matches this project's own PRD/brief frontmatter convention elsewhere in this run
  (both `prd.md` and `brief.md` use `title:` too), so it reads as an intentional project-wide
  convention, not a one-off defect.
- **`sources:` resolution:** both entries in EXPERIENCE.md's frontmatter
  (`prds/prd-fabric-hris-2026-08-10/prd.md`, `briefs/brief-fabric-hris-2026-08-10/brief.md`) resolve
  to real files on disk. `imports/` for this spine is empty, consistent with the stated Fast-path/
  no-mockups scope for this run.
- **No naming drift found** in casing/spelling for any of the five shared components (Status badge,
  Metric tile, Mode indicator, Log line, Code block) between DESIGN.md and EXPERIENCE.md.
- **Prior reconciliation held up under a fresh pass:** all 5 gaps logged in `reconcile-prd.md`
  (in-app RECORDED mode; tenant02 guardrail weight; FR-2 explicit-zero; PRD §10 generic-register
  constraint in Voice and Tone; Dashboard staleness signal) were independently verified present and
  correctly fixed in the current DESIGN.md/EXPERIENCE.md — none of them reopened as findings above.

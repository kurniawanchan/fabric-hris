---
name: Caliper Benchmark Dashboard
description: Live monitoring dashboard for a Hyperledger Caliper performance benchmark, shown during a thesis defense. shadcn/ui on React + Tailwind; this DESIGN.md specifies the brand-layer delta only. `[ASSUMPTION: shadcn/ui is the UI system — no existing system was specified, and it fits the "zero existing frontend infrastructure, two-week timeline" constraint (PRD §11) better than a custom system would.]`
colors:
  # Brand overrides on top of shadcn defaults. Unlisted tokens (background,
  # foreground, muted, muted-foreground, popover, card, border, input, ring,
  # destructive) inherit from shadcn.
  primary: '#1E3A5F'
  primary-foreground: '#FFFFFF'
  primary-dark: '#7FA8D9'
  primary-foreground-dark: '#0B1826'
  status-healthy: '#16A34A'
  status-healthy-foreground: '#FFFFFF'
  status-healthy-dark: '#4ADE80'
  status-healthy-foreground-dark: '#052E12'
  status-warning: '#D97706'
  status-warning-foreground: '#1A1208'
  status-warning-dark: '#FBBF24'
  status-warning-foreground-dark: '#1A1208'
  status-down: '#DC2626'
  status-down-foreground: '#FFFFFF'
  status-down-dark: '#F87171'
  status-down-foreground-dark: '#2C0A0A'
typography:
  # Body/label/caption inherit shadcn's Geist Sans. Two overrides: a metric
  # display role for live numbers, and a data/mono role for hashes, IDs,
  # YAML, and CLI output — this dashboard shows a lot of both.
  metric:
    fontFamily: 'Geist Sans'
    fontSize: 32px
    fontWeight: '600'
    lineHeight: '1.1'
  metric-label:
    fontFamily: 'Geist Sans'
    fontSize: 13px
    fontWeight: '500'
    letterSpacing: 0.02em
  data-mono:
    fontFamily: 'Geist Mono'
    fontSize: 13px
    fontWeight: '400'
    lineHeight: '1.5'
rounded:
  # shadcn defaults inherited as-is — no brand delta. A monitoring tool
  # doesn't need a distinct corner language.
spacing:
  # shadcn / Tailwind scale inherited as-is. One override: this is a
  # data-dense, multi-panel dashboard, not a reading surface — no narrow
  # max-width constraint.
  content-max-width: 'none'
components:
  status-badge:
    healthy:
      background: '{colors.status-healthy}'
      foreground: '{colors.status-healthy-foreground}'
      radius: 'full'
    warning:
      background: '{colors.status-warning}'
      foreground: '{colors.status-warning-foreground}'
      radius: 'full'
    down:
      background: '{colors.status-down}'
      foreground: '{colors.status-down-foreground}'
      radius: 'full'
  mode-indicator:
    live:
      background: '{colors.status-healthy}'
      foreground: '{colors.status-healthy-foreground}'
    idle:
      background: 'muted'
      foreground: 'muted-foreground'
  metric-tile:
    background: 'card'
    value-typography: '{typography.metric}'
    label-typography: '{typography.metric-label}'
    radius: 'md'
  code-block:
    background: 'muted'
    typography: '{typography.data-mono}'
    radius: 'md'
  log-line:
    typography: '{typography.data-mono}'
    background: 'transparent'
  topology-diagram:
    node-grouping: 'by-org'
    node-shape: 'card'
    node-background: 'card'
    status-badge-position: 'top-right of node'
    orderer-node-shape: 'circle'
    layout: 'three org columns (Org1, OrgClient-tenant01, Org3); orderer set shown once, shared between columns rather than duplicated per org'
status: final
created: '2026-08-10'
updated: '2026-08-11'
---

## Brand & Style

This is an engineering monitoring surface shown to an examination committee, not a consumer
product — the aesthetic posture is *control room*, not *marketing page*. Every visual choice
optimizes for two things at once: instant legibility of a live system's state (is this node
healthy? did that transaction commit?), and credibility in front of an academic audience judging
whether the artifact is real. No decorative flourish earns a place here that doesn't also serve
one of those two goals.

Inherits shadcn/ui defaults wholesale — this file specifies only the brand-layer deltas: a single
navy primary, a semantic status-color set (healthy/warning/down — justified below, since a
monitoring tool's core job *is* signaling status, unlike most products, where a "two colors and
stop" rule normally applies), and two typography roles for the two things this dashboard is mostly
made of: live numbers and technical strings (hashes, peer IDs, YAML, CLI output).

## Colors

- **Navy Primary (`#1E3A5F` light / `#7FA8D9` dark)** — brand color. Used on primary actions, the
  active tab indicator, and section headers. Navy is the boldest non-status color here, but nothing
  about it competes with the status colors below for attention — status is the thing that should
  pull the eye first.
- **Status-Healthy Green (`#16A34A`)** — a node is reachable and healthy; a transaction committed.
- **Status-Warning Amber (`#D97706`)** — degraded but not down (e.g., a node responding slowly).
  `[ASSUMPTION: a distinct warning state is worth having, even though the PRD's FR-4 only names
  binary healthy/down — a monitoring surface with only two states tends to flatten "slow" into
  either false-healthy or false-down; amber gives the operator a true middle reading.]`
- **Status-Down Red (`#DC2626`)** — a node is unreachable; a transaction/round failed.
- **All other tokens** (background, foreground, muted, border, input, ring, card, popover,
  destructive) inherit shadcn defaults, including the mode indicator's Idle state (`muted`/
  `muted-foreground`) — Idle is a neutral, unremarkable reading, not a status that needs its own
  brand color.

## Typography

Body, label, and caption inherit shadcn's Geist Sans ramp as-is. Two additions:

- **`metric`** (32px, 600 weight) — the large live numbers on Dashboard and Network Profile
  (throughput, latency, CPU%). Sized to be readable from the back of a room on a projected screen,
  not just up close on a laptop. `[ASSUMPTION: 32px is a reasonable "readable from a projector at
  defense-room distance" size — no exact room/screen size was given; this is a starting point to
  sanity-check against the real room, not a load-bearing number.]`
- **`data-mono`** (Geist Mono, 13px) — every hash, record ID, peer address, YAML line, and CLI
  output line. Monospace signals "this is a literal technical value" and lines up columns of
  numbers (latency ms, byte counts) so they're comparable at a glance.

## Layout & Spacing

shadcn / Tailwind's default spacing scale inherited as-is. One deviation: **no max-width
constraint on content** — this is a data-dense, multi-panel dashboard meant to fill a wide,
projected screen, not a narrow reading column. Five-tab top navigation (see EXPERIENCE.md's
Information Architecture); each tab's content area uses the full viewport width.

## Elevation & Depth

Inherited from shadcn as-is — subtle shadow on hover/active, no elevation as a hierarchy device.

## Shapes

shadcn defaults inherited as-is, with one exception: status badges and the mode indicator use
`rounded/full` (pill shape) — the one place a distinct shape is worth reserving, since a pill
reads as "state indicator" at a glance, distinct from every rectangular card/tile around it.

## Components

- **Status badge** — pill-shaped, one of healthy/warning/down. Always paired with a text label
  ("Healthy" / "Slow" / "Unreachable"), never color alone — see EXPERIENCE.md's Accessibility
  Floor for why.
- **Mode indicator** — a persistent, always-visible pill (top of every tab) reading "Live" (green)
  or "Idle" (muted). It has no third state for the recorded fallback — that fallback is a separate
  video played outside this product entirely (EXPERIENCE.md's State Patterns), not a mode the
  dashboard represents.
- **Metric tile** — a card showing one live number (`metric` typography) with its label
  (`metric-label` typography) beneath. Used on Dashboard and Network Profile.
- **Code block** — monospace, muted background, used for the Configuration tab's YAML display and
  any hash/ID rendering that needs a clear visual boundary from surrounding prose.
- **Log line** — a single monospace line in the Notifications tab's live tail, no background,
  minimal vertical padding so many lines fit on screen at once.
- **Topology diagram** — Network Profile's primary artifact. Nodes rendered as cards (orderers as
  circles, to read as structurally distinct from peers at a glance), grouped into three columns by
  org (`Org1`, `OrgClient-tenant01`, `Org3` — never a fourth). The shared 3-node orderer set is
  shown once, rather than duplicated per org column. Each peer/orderer node carries its own status
  badge (above) positioned top-right, so health is legible without leaving the topology view.
- **shadcn primitives used as-is, no brand delta:** Tabs (top navigation), Select (the
  benchmark-scenario selector), Button, Card, Dialog. Everything not named above or in this list
  inherits shadcn's own visual spec unchanged.

## Do's and Don'ts

- **Do** keep the three status colors reserved for their one meaning each, everywhere in the product.
- **Do** pair every status signal with a text label or icon, never color alone.
- **Do** render a zero value as a literal "0", never a blank field (FR-2) — see EXPERIENCE.md's
  Component Patterns.
- **Don't** introduce a fourth "brand" color beyond navy — this is a monitoring tool, not a
  marketing surface, and every added color is one more thing competing with status for attention.
- **Don't** use elevation/shadow as a hierarchy device — hierarchy comes from typography and
  status color here, not depth.
- **Don't** ever render `OrgClient-tenant02`/`peer0.tenant02` in any Network Profile component,
  even if a future data source returns it — see EXPERIENCE.md's standing guardrail (PRD §4.2/§12).

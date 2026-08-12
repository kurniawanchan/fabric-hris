# PRD Quality Review — Caliper Performance-Benchmark GUI Dashboard (prd-fabric-hris-2026-08-10)

## Overall verdict

This is a disciplined Fast-path PRD that earns most of its structure: trade-offs are named with what
was given up, FRs mostly carry testable consequences, and the document is unusually self-aware about
its own shape (single UJ, capability-spec framing, an invented §10 rather than forcing a template
section that doesn't fit). It is undermined by one verified, concrete defect: FR-3's own topology
assumption ("5 peers, 3 orderers, **3 orgs**, matching this network's actual configtx.yaml") does not
match `configtx.yaml`, which defines four non-orderer orgs (`Org1`, `OrgClient-tenant01`,
`OrgClient-tenant02`, `Org3`) — and the fourth org's peer (`peer0.tenant02`) is a live container in the
same compose file this PRD cites elsewhere. Everywhere else in the document repeats "three orgs";
nothing flags this tension. A second, smaller defect (a wrong file path for the same `configtx.yaml`,
repeated in both `prd.md` and `addendum.md`) compounds it. Fix those two and this PRD is
build-ready.

## Decision-readiness — adequate

Most tensions in this PRD are surfaced honestly rather than smoothed over. §5's "No live config
editing/resubmission" names *why* it was cut ("judged the riskiest, least-justified piece of the
reference layout's original scope"), not just that it was cut. SM-1's inline tag is a genuinely good
catch: "a metric can't be validated at the moment it's meant to protect — the brief's own criterion
was 'during the defense'" — the PRD notices its own primary success metric can't literally validate
what it claims to, and says so instead of quietly redefining the metric. Open Questions (§8) are
actually open: #2 (export) is "undecided... revisit if time permits," #4 (rehearsal cadence) is
explicitly punted to Mr. Chan as "not a PRD-blocking" question — neither is a rhetorical question
with the answer already given.

What pulls this down from *strong*: one real, unacknowledged tension exists and none of the PRD's own
disclosure mechanisms ([ASSUMPTION], Open Question, or a [NOTE FOR PM]-style callout — none of which
appear anywhere in this PRD, despite the rubric's expectation that real tensions get one) catch it —
see the Done-ness clarity finding below on FR-3's org-count mismatch. A PRD that is otherwise this
careful about tagging judgment calls (five tagged assumptions, four open questions) leaving this one
specific, checkable claim untagged is itself informative: it reads as asserted without having been
checked against the file it cites, not as a considered trade-off. Separately, the addendum's
G-37 disclosure ("architecture should decide whether the dashboard's read-only consumption of them
needs any safeguard of its own") is exactly the shape of a live, unresolved tension the rubric asks
`[NOTE FOR PM]` to mark — it's disclosed in prose, just not flagged with the weight the PRD gives its
other open items.

## Substance over theater — strong

No findings. The persona/audience layer (§2.1–§2.2) is unusually disciplined for how easy it would be
to pad: there is exactly one operator (Mr. Chan) and two served-but-non-operator audiences (the
committee, thesis readers), and both audiences are used, not decorative — the thesis-readers JTBD
resolves directly to SM-4, the committee JTBD resolves directly to UJ-1. The Vision (§1) is specific
enough that it could not swap into another PRD unchanged: it cites this project's own DSRM phase
mapping (`agent-suite/context/DSRM.md`, Phase 4 → Phase 6), names the exact reference repo it took
layout inspiration from and *why* it isn't reusing that repo's code ("a UI shell with no working
backend behind it"), and hedges its own "future go-to visualization" aspiration explicitly as "a
possibility to keep open, not a plan" rather than smuggling it in as a v1 commitment. No claimed
novelty, no boilerplate NFR language ("must be scalable/secure/reliable") anywhere in §4's
feature-specific NFRs — both instances present (FR-1's lag threshold, FR-5's no-new-instrumentation
constraint) are concrete and scoped.

## Strategic coherence — strong

No findings. The PRD has a real thesis and defends it with an unusually sharp distinction: the reason
Table List/Notifications/Configuration are *present* ("hedging against not knowing what the committee
will ask about") is explicitly named as different from the reason Dashboard/Network Profile are built
*fully live* (directly realizing UJ-1, the thesis's actual demonstration need) — §1: "depth, not
presence, is what tiering trades away." That is feature prioritization following from a thesis, not
from "what's easy first," stated as such. Success Metrics are outcome-shaped, not activity-shaped
(SM-1 requires *real, unmocked* live data during an actual rehearsal, not "dashboard loads"), and a
counter-metric is named (SM-C1, guarding rehearsal/writing time against scope creep) exactly where the
rubric asks for one.

## Done-ness clarity — adequate

FRs are disciplined about testable consequences almost throughout — FR-8's "matches the real file
byte-for-byte, not a paraphrase" and FR-6's "data matches those documents (no independent
recalculation)" are exactly the bar the rubric asks for. But one FR's testable consequence conflicts
with a verified fact about the codebase it cites, and two phrasings are the kind of soft adjective the
rubric asks to be unforgiving about.

### Findings

- **high** FR-3's topology consequence conflicts with `configtx.yaml`'s actual org count (§4.2) —
  The FR-3 assumption states: *"a static, correctly-labeled topology (5 peers, 3 orderers, **3 orgs**,
  matching this network's actual configtx.yaml)"* and the testable consequence is *"Every
  peer/orderer/org in `fabric-network/network/configtx.yaml` appears."* Verified against the real file
  (`fabric-network/network/configtx/configtx.yaml`): it defines **four** non-orderer org MSPs —
  `Org1MSP`, `OrgClient-tenant01MSP`, `OrgClient-tenant02MSP`, `Org3MSP` — via two separate channel
  profiles (`TenantChannelGenesis` uses Org1/tenant01/Org3; `Tenant02ChannelGenesis` uses
  Org1/tenant02/Org3). `peer0.tenant02` is not a dormant definition — it is a live service in
  `network-docker-compose.yaml` (the same file the addendum cites for operations ports) started by the
  ordinary `docker compose up -d` used to bring the network up. "Three orgs" is repeated as fact four
  times elsewhere in this PRD (§1's tamper-detection premise — "the client org's" independent
  operation, singular — §2, §4.2's description, and the FR-3 assumption's own parenthetical), so if
  FR-3 is built literally per its own testable consequence, the live topology view will surface a
  fourth, currently-unprovisioned, known-hazardous org (CLAUDE.md: tenant02 is "deliberately
  abandoned," and provisioning it previously crashed peer containers twice — NET-7, QA-4/PT-4) that
  contradicts every other framing of this network in the document — in front of the exact audience
  (the examination committee) this PRD exists to not confuse. Nothing in §8 (Open Questions), §9
  (Assumptions Index), or §12 (Risks) names this as a decision still to make. *Fix:* add an explicit
  scope line — either "topology view is scoped to the active `TenantChannelGenesis` profile's three
  orgs; `OrgClient-tenant02`/`peer0.tenant02` is deliberately excluded despite being a running
  container" or the opposite decision, plus a one-line Risk update if tenant02's peer is in-frame for
  FR-4/FR-5's health/resource polling too.

- **low** Two FR consequences lean on the soft-adjective pattern the rubric flags — FR-2: "a round
  that completes with e.g. 0 failures shows that plainly" — "plainly" isn't a bound (bold text? a
  distinct color? a literal "0" vs a blank field?). FR-4: "visibly distinguishable" for a down/
  unreachable node has the same gap — distinguishable *how* is left to the builder. Both are adjacent
  to enough surrounding testable detail that a builder could still ship something defensible, but
  neither would survive the rubric's "flag every one" instruction literally. *Fix:* one clause each
  (e.g., "distinguishable by a specific color/icon change, not just a numeric metric going to zero")
  would close this at negligible cost.

## Scope honesty — strong

No findings beyond the mechanical note below. Non-Goals (§5) and Out-of-Scope (§4.3, §6.2) do real
work rather than gesturing at completeness — each has a stated reason (risk, effort, or "revisit if
time permits"), not a bare list. The assumption-tagging discipline is genuinely followed for four of
five inline judgment calls (see Mechanical notes for the fifth). Open-items density (4 Open Questions
+ 5 indexed assumptions + 0 NOTE-FOR-PM-style callouts) is proportionate to the stated stakes
(internal tool, single operator, bounded 2-week artifact) — this is not a green-light-to-build PRD
carrying a green-light-density of unresolved items.

## Downstream usability — adequate

This is not a standalone PRD — §0 states it "builds directly on" the approved brief, and `addendum.md`
is explicitly written "for whoever picks this up next (architecture phase)" — so this dimension
carries real weight here. ID hygiene is clean (FR-1…FR-8 contiguous; SM-1…SM-4 + SM-C1; a single,
appropriately-scoped UJ-1 with a named protagonist), and the Glossary (§3) covers the terms FRs
actually lean on ("Live run," "Fully live," "Recorded fallback," "Operations endpoint" all get reused
correctly). One verified defect keeps this from *strong*:

### Findings

- **medium** Wrong path for `configtx.yaml`, repeated in two files — Both `prd.md` §4.2 ("Every
  peer/orderer/org in `fabric-network/network/configtx.yaml` appears") and `addendum.md`'s "Concrete
  data sources per feature" ("static topology from `fabric-network/network/configtx.yaml`") cite a
  path that does not exist. The real file is at `fabric-network/network/configtx/configtx.yaml` (this
  project's own CLAUDE.md correctly refers to the directory as `network/configtx/`). Whoever picks
  this up at the architecture phase — the addendum's stated audience — will follow this exact citation
  and not find the file. *Fix:* a one-character-class fix (`configtx.yaml` → `configtx/configtx.yaml`)
  in both places.

## Shape fit — strong

No findings. This PRD is unusually self-aware about calibrating its own format rather than filling out
a template: §2.3 states outright "kept light per this being internal tooling with one operator role"
and delivers exactly one UJ, not a UJ per feature area. §10 is explicitly labeled "Invented section —
this product carries two project-specific constraints the standard menu doesn't name" rather than
silently omitting the "no frontend stack" exception or silently forcing it into a Non-Goal. For a
bounded, single-operator, two-week demo artifact, capability-spec framing over UJ density is the
right call, and the PRD names that call rather than just making it.

## Mechanical notes

- **Assumptions Index roundtrip gap.** §9's first bullet ("§2.3 (UJ-1) — recorded-fallback switchover
  is the only in-dashboard recovery path modeled") indexes a claim that has no matching inline
  `[ASSUMPTION: ...]` tag — §2.3's actual text states this as a plain "Edge case," not a tagged
  assumption. The other four index entries (§4.1, §4.2, §4.3, §7) each round-trip cleanly to a literal
  inline tag.
- **Punctuation defect in the SM-1 assumption tag** (§7) — the tag opens a nested quote
  ("the brief's own criterion was "during the defense.") that's never closed before the bracket.
  Cosmetic, but worth a pass before this leaves draft status.
- **Glossary drift, minor.** §3 defines "Static / read-only" as one compound term; usage splits it —
  Table List's description (§4.3) uses "Static," Configuration's (§4.5) uses "read-only" — without
  either section pointing back to the compound Glossary term. "Profile section" (§3) is defined but
  never invoked by that name anywhere else in the document body — an orphaned Glossary entry (likely
  present for future-proofing rather than current use, which is a defensible reason, just worth
  knowing it's currently unused).
- **Unconfirmed working title.** Line 10: "*Working title — confirm.*" — still open as of this draft,
  worth closing out before this leaves draft status given the ~2-week runway.
- **Verified accurate (worth noting, since brownfield citations were checked):** the operations-port
  numbers in `addendum.md` (peers 9444/9445/11446/9446/9447; orderers 7071/8070/9070) match
  `network-docker-compose.yaml` exactly, and `G-37`'s plaintext-operations-endpoint claim it carries
  forward is itself accurate against that file. `RESULTS.md`, `QA6-RESULTS.md`, the
  `results/raw-latencies/*.jsonl` directory, and all five `benchconfigs/*.yaml` files FR-6 names exist
  exactly as described.

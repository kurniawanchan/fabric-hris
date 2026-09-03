# Reconciliation: [DASH] PRD (2026-08-10) vs COMBINED synthesis (2026-08-13)

Scope: SOURCE = `prds/prd-fabric-hris-2026-08-10/prd.md`. COMBINED = `prds/prd-fabric-hris-2026-08-13/prd.md`
(§3.C, [DASH]-tagged rows in §4/§6/§8).

## 1. Business features/capabilities — completeness of §3.C

**Finding 1.1 — No area is fully omitted; coverage is intact at the index level.**
All five feature bullets in §3.C map onto SOURCE's five areas: Dashboard (FR-1/FR-2), Network
Profile (FR-3/FR-4/FR-5), Table List (FR-6), Notifications (FR-7), Configuration (FR-8). No
capability is completely missing — the gaps below are lost *nuance/rationale*, not missing
features, consistent with COMBINED's own stated index-level intent.

**Finding 1.2 — The tenant02 exclusion and its safety rationale are dropped, and it is the one
piece of §3.C that reads as a factual simplification rather than a compression.**
- SOURCE (FR-3, §12 Risks): Network Profile is explicitly scoped to the `TenantChannelGenesis`
  profile only (Org1, OrgClient-tenant01, Org3, 3 orderers) and `OrgClient-tenant02`/`peer0.tenant02`
  is "deliberately excluded, even though it is a running container" — flagged as a real risk
  ("a fourth, hazardous org accidentally surfacing on screen") because tenant02 crashed peer
  containers twice (NET-7, QA-4/PT-4) and its presence on screen would contradict the network's own
  three-org framing.
- COMBINED (§3.C, bullet 2): "Live network topology and per-node health/resource view, scoped to
  the network's actual active channel membership, reading only existing peer/orderer operations
  endpoints (no new instrumentation)."
- Gap: "the network's actual active channel membership" reads as if the network *has* only one
  channel/topology to show. It elides that a second, live, running org/peer is deliberately being
  hidden from the committee, and why. This is defensible as index-level compression, but it is the
  one place COMBINED's phrasing could mislead a reader into thinking there's nothing to scope
  around — worth a footnote if COMBINED is ever tightened.

## 2. Success metrics (§4 table)

**Finding 2.1 — SM-1 through SM-4 and SM-C1 are represented correctly; no misrepresentation found.**
COMBINED's SM-1–SM-4 rows match SOURCE's §7 wording and thresholds:
- SM-1: "Pre-defense rehearsal completed" — matches SOURCE's `[ASSUMPTION]`-qualified operationalization (validated pre-defense, not at the live moment).
- SM-2: "All 5 dashboard areas demonstrable on request" — matches.
- SM-3: "Rehearsed, switchable within time budget" — matches.
- SM-4: "≥1 figure distinct from results tables" — matches SOURCE's "distinct from the existing RESULTS.md/QA6-RESULTS.md tables."
- SM-C1 (counter-metric): COMBINED folds it into the shared counter-metrics line — "[DASH]'s build time must not eat into thesis-writing or rehearsal time" — an accurate paraphrase of SOURCE's SM-C1 ("Time spent polishing this dashboard must not come at the expense of thesis-document writing or defense rehearsal time").

No factual error or dropped metric found in §4 for [DASH].

## 3. Constraints not carried forward

**Finding 3.1 — The 2-week timeline is dropped entirely.**
SOURCE §11 "Why Now": "The thesis defense is fixed at roughly two weeks from this PRD's date
(2026-08-10). This is the single reason depth is tiered rather than built to full parity with the
reference layout." COMBINED's §6 constraints list for [DASH] has only one bullet (the frontend-stack
exception) and never states the two-week window or ties tiering to it.

**Finding 3.2 — Single-operator / single-screen framing is dropped as an explicit constraint.**
SOURCE §2.3: "Single operator, one real session shape — kept light per this being internal tooling
with one operator role," and §6.2 lists "Multi-user/concurrent-viewer support" as explicitly out of
scope for that reason. COMBINED's §8 Out-of-Scope line does carry "multi-viewer dashboard support"
as an exclusion, but never states *why* (single-operator, single-screen) the way SOURCE does — the
constraint's rationale, not just its consequence, is lost.

**Finding 3.3 — The tiered-depth rationale is dropped.**
SOURCE §1 Vision: "Five areas, at deliberately tiered depth to fit a two-week build against zero
existing frontend infrastructure... Dashboard and Network Profile are fully live; Table List,
Notifications, and Configuration are present at reduced depth... depth, not presence, is what
tiering trades away." COMBINED's §3.C bullets present all five areas as flat, undifferentiated
capabilities with no signal that three of them are intentionally shallower than the other two.

**Finding 3.4 — The "no frontend stack" exception is carried forward, correctly.**
COMBINED §6: "[DASH]'s frontend stack is a one-time, disclosed exception, not a reversal of this
project's general 'no frontend' posture — bounded to the dashboard, local-only, single-operator,
defense-day artifact." This matches SOURCE §5/§10 in substance (though see Finding 5.1 on "one-time"
framing).

## 4. Factual errors in COMBINED's characterization of SOURCE

**Finding 4.1 — No port numbers, file paths, or hard factual errors found.**
Checked: topology scope (Org1/OrgClient-tenant01/Org3, 3 orderers) — COMBINED doesn't restate the
specific org/peer/orderer counts at all (index-level, not wrong, just silent — see 1.2). Config file
path (`qa-tests/performance/benchconfigs/`), results docs (`RESULTS.md`/`QA6-RESULTS.md`) — COMBINED
doesn't cite specific paths either, so nothing to be wrong about. No misstatement identified.

## 5. Framing of [DASH] as a one-time bounded exception (§6)

**Finding 5.1 — COMBINED slightly overstates boundedness by dropping SOURCE's "kept open" caveat.**
SOURCE §1 Vision (paragraph 4): "Not committed as part of this v1, but worth naming here rather than
only in the brief: if this holds up through the defense, its live-metrics plumbing... could become
this project's lightweight go-to visualization for future benchmark runs, rather than a one-time
artifact. That's a possibility to keep open, not a plan." SOURCE §5 Non-Goals similarly says
"whether it becomes an ongoing tool afterward is an open possibility (§1 Vision), not a v1 commitment
either way."
COMBINED §6 states flatly: "[DASH]'s frontend stack is a one-time, disclosed exception... bounded to
the dashboard, local-only, single-operator, defense-day artifact" — with no equivalent hedge. This is
a real, if minor, framing drift: SOURCE treats "one-time" as true for v1 scope but explicitly leaves
the door open post-defense; COMBINED's phrasing reads as more categorically bounded/closed than
SOURCE intends. Not a contradiction, but an overstatement of finality.

## Summary

- No SOURCE feature/area is completely missing from §3.C (Finding 1.1); the tenant02
  exclusion/rationale is the one place compression edges toward misleading (1.2).
- SOURCE's success metrics (SM-1–4, SM-C1) are accurately represented in §4 — no misrepresentation
  found.
- Three constraints are dropped rather than compressed: the 2-week timeline (3.1), the
  single-operator/single-screen rationale (3.2), and the tiered-depth rationale (3.3). The
  "no frontend stack" exception is correctly carried forward (3.4).
- No hard factual errors (wrong ports/paths/scope numbers) found in COMBINED's [DASH] content.
- COMBINED's "one-time exception" framing (§6) is slightly more categorical/closed than SOURCE's own
  framing, which explicitly keeps a post-defense future open as a possibility, not a plan (5.1).

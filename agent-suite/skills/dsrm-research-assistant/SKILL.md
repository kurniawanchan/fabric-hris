---
name: dsrm-research-assistant
description: >-
  Drives Design Science Research Methodology (DSRM / design-science) work for this HRIS-on-Fabric
  prototype: frame a research question as a testable predicate, classify the artifact type, plan
  the rigor-relevance design cycle, map work onto the six DSRM activities and gates G0-G9, and - as
  bundled modes - plan an experiment/demonstration (the brief's "Experiment Planner") and generate
  an evaluation report ("Evaluation Report Generator"). Trigger on DSRM, design-science, a research
  question, an experiment/demonstration plan, an evaluation report, or literature synthesis for a
  design artifact - even when "DSRM" is unsaid. Orchestrates deep-research and huggingface
  paper_search for literature; publishes via the Atlassian MCP. NEGATIVE: generic sprint/backlog
  planning -> project-planning; the design doc/RFC/ADR -> rfc-writer or architecture-design; deep
  literature web-research -> deep-research (only orchestrated here); Fabric security-testing ->
  fabric-security-review; throughput/latency -> fabric-performance.
license: Apache-2.0
---

# dsrm-research-assistant

Run the **Design Science Research Methodology** thread of this package: turn a design task into a
framed research question, place it in the DSRM activity/gate map, plan how the resulting artifact
is demonstrated and evaluated, and write the evaluation report that decides whether the artifact
advances. This is the *workflow and trigger* skill; the *knowledge* — the six activities, the exact
gate mapping, the three-cycle model, and the evaluation-method families — lives in the context doc
and is cross-referenced, not restated: **read `../../context/DSRM.md` first**.

> Grounding: methodology facts trace to `../../context/DSRM.md` and the seminal literature by author/year
> (Peffers et al., 2007; Hevner et al., 2004; Hevner, 2007; March & Smith, 1995) — a bibliographic
> attribution, **not** a corpus `[docs:]` citation, because DSRM is not in the Fabric 2.5 corpus.
> Fabric facts carry `[docs: …]` via the owning `fabric-*` skill. Every requirement-shaped specific
> carries `[ASSUMPTION]` and its gap ID (`G-01..G-29`, see `../../11-execution/grounding-gaps.md`).

## When to use this skill

- "Frame / sharpen the research question for <artifact>." → **Frame & Map** mode.
- "Which DSRM activity / gate does <deliverable> belong to?" → **Frame & Map** mode (emits a `dsrmPhaseMap`).
- "Plan the experiment / demonstration for the anchor-then-verify flow." → **Experiment Planner** mode.
- "Write the G8 evaluation report for the package / this artifact." → **Evaluation Report Generator** mode.
- "Synthesize the literature to justify <design choice> against <security frame>." → orchestrate
  `deep-research` + `paper_search`, then feed the cited result into the rigor cycle.
- "Are we strong on rigor or relevance here?" → three-cycle balance check (Frame & Map mode).

## When NOT to use (route away)

- **Sprint/backlog/estimate/WBS/generic project planning** → `project-planning`. DSRM phases are a
  research lifecycle, not a delivery schedule.
- **Writing the design document, RFC, ADR, or architecture doc itself** → `rfc-writer` /
  `architecture-design`. This skill *frames and evaluates* the artifact; those skills *produce* it.
- **Deep multi-source, fact-checked literature web-research** → the `deep-research` skill. This skill
  **orchestrates** it (frames the question, consumes the cited report); it does not fan out searches.
- **The Fabric security-testing / hardening checklist and finding model** → `fabric-security-review`
  (the Analytical/Testing evaluation depth for a control review lives there; do not duplicate it).
- **Throughput, latency, block-size, CouchDB-vs-LevelDB evaluation** → `fabric-performance`
  (out of scope for the *security* evaluation; deferred behind gap **G-08**).
- **The methodology reference itself** (six activities, cycle definitions, evaluation families) →
  read `../../context/DSRM.md`; do not re-derive it here.

## The map: 6 DSRM activities → project gates G0–G9

This skill **emits** a `dsrmPhaseMap`, but the canonical 6-activity → G0–G9 mapping is **not
redefined here** — it has one home: **`../../context/DSRM.md §2`** (= knowledge-graph Layer F). Read
it there and emit against it. (The rows are deliberately not restated here, per the one-home rule.)

**Operational rules this skill applies on top of that map:**
- **Entry point** (`DSRM.md §1`): *problem-centered initiation* — the tamper-evident,
  confidentiality-preserving anchoring of employee-PII changes is the starting motivation.
- **Ordering rule** (`DSRM.md §2`): DSRM lists Demonstration before Evaluation, but the design-only
  boundary inverts that here (Evaluation G8 + Communication/approval G9 precede any running prototype).
  **Never plan a running-prototype experiment before G9 — label it Phase-4 (post-G9).**
- **G8 is a hard iteration boundary** (`DSRM.md §2`): do not advance a deliverable to G9 with an open
  CONFIRMED review finding; loop it back to its owning gate.

## Artifact types (classify before mapping)

Classify the artifact under March & Smith (1995) so the plan targets the right evaluation.
**Operational delta:** construct/model/method are produced *now*; **instantiation** (the running
prototype) is **Phase-4 only (post-G9)** — never plan its evaluation as if it exists pre-G9.

> Artifact types (definitions + this project's instances): see `../../context/DSRM.md §1`.

## Three modes

Pick the mode from the ask; each consumes the input schema below and emits one output object. Every
mode runs the same loop: **clarify → place on the map → produce the object → mark Fact vs.
Recommendation → flag `[ASSUMPTION]` (with gap ID) for anything the user has not pinned down.**

### Mode 1 — Frame & Map  → `dsrmPhaseMap`

Frames the research question and locates the work.

1. **Confirm the entry point** (problem-centered here) and the artifact type (table above).
2. **Frame/refine the research question as a *testable predicate*.** Not "does Fabric help?" but
   e.g. *"a verifier can confirm a PII change occurred and matches the off-chain record, without
   learning the plaintext."* Tie it to the solution objective — the objective itself is
   `[ASSUMPTION] (gap G-01)` until HLF-6 / Confluence land.
3. **Place the work item on exactly one primary DSRM activity + gate** (map above); note secondary
   gates if it spans several (activity 3 spans G2–G7).
4. **Balance the three cycles** (Hevner, 2007; depth `DSRM.md §3`): *relevance* (environment↔design),
   *rigor* (knowledge base↔design), *design* (build↔evaluate). Flag where relevance is **provisional**
   because requirements are unreachable, and — Recommendation — **weight rigor** (corpus + frames +
   literature) accordingly, revisiting relevance-sensitive items when a gap closes.
5. Emit the `dsrmPhaseMap`.

### Mode 2 — Experiment Planner  → `experimentPlan`

Plans a demonstration/experiment that will produce evidence an artifact meets its objective.

1. **State the objective as a testable predicate** (from Mode 1 / gap G-01).
2. **Choose the evaluation-method family** (Hevner et al., 2004; five families in `DSRM.md §4`):
   Descriptive · Analytical · Testing · Experimental · Observational. **Fact:** while design-only
   (through G9) the *currently applicable* families are **Descriptive + Analytical**; **Testing** and
   **Experimental** become primary only in **Phase-4 Demonstration (post-G9)**. Pick by the item's
   phase — do not plan a Testing experiment that presupposes a running network before G9.
3. **Define the instance of the problem** (the demonstration case), e.g. anchor-then-verify a
   `PERSONAL` profile-section write (the anchoring unit is a whole section, not a per-field change
   event — ratified 2026-08-02, gap G-05 **closed**, `[prd: §4]`).
4. **Define conditions, the success metric, and the negative/tamper case** (e.g. a mutated off-chain
   record must fail verification against the on-chain commitment).
5. **Name owner + venue** (per `DSRM.md §4`). Route security-testing depth to `fabric-security-review`;
   route any performance question to `fabric-performance` (out of scope, gap **G-08**); route ZKP /
   selective-disclosure PoCs behind gap **G-07** to Phase-4.
6. Emit the `experimentPlan`.

### Mode 3 — Evaluation Report Generator  → `evaluationReport`

Reports measured evidence that the artifact meets its objectives — the G8 activity (or a Phase-4
post-demonstration loop).

1. **Restate each objective + its success predicate.**
2. **For each:** method family used, the evidence, the result (`met` / `partial` / `not-met`), and
   the security frame(s) it satisfies (CIA / ZKP / PbD / SbD / OWASP ASVS / OWASP API / NIST ZT —
   knowledge-graph Layer E). Separate **Fact** (cited/observed) from **Recommendation** (judgment).
3. **Iteration decision:** apply the G8 hard-boundary rule — any CONFIRMED finding loops the
   deliverable back to its owning gate before G9 (Recommendation, `DSRM.md §2`).
4. **Communication hooks:** note what is conveyed to researchers vs. practitioners (Communication =
   activity 6 / G9); this is what the Confluence publish will carry.
5. Emit the `evaluationReport`; optionally publish via the Atlassian MCP (see MCP integration).

## Input / output schemas

**Input** (all modes; JSON):

```json
{
  "mode": "frame-map | experiment-plan | evaluation-report",
  "phase": "G0..G9 | post-G9",
  "artifact": { "name": "string", "type": "construct|model|method|instantiation", "owningGate": "G#" },
  "researchQuestion": "string (framed as a testable predicate)",
  "objectives": [ { "id": "OBJ-#", "predicate": "string", "frame": "CIA|ZKP|PbD|SbD|ASVS|API|ZeroTrust", "assumption": "G-##" } ],
  "evidence": [ { "objectiveId": "OBJ-#", "method": "descriptive|analytical|testing|experimental|observational", "observation": "string" } ]
}
```

`objectives` are optional for Frame & Map, required for the two report/plan modes; `evidence` is used
only by Evaluation Report. Anything requirement-shaped and unsourced is echoed back as
`[ASSUMPTION]` with its gap ID rather than silently defaulted.

**Output** (exactly one object):

```json
{
  "dsrmPhaseMap": {
    "entryPoint": "problem-centered",
    "artifactType": "construct|model|method|instantiation",
    "primaryActivity": "1..6", "primaryGate": "G#", "spansGates": ["G#"],
    "researchQuestion": "testable predicate",
    "cycleBalance": { "rigor": "strong|provisional", "relevance": "strong|provisional", "notes": "…", "assumptions": ["G-##"] }
  },
  "experimentPlan": {
    "objectiveId": "OBJ-#", "successPredicate": "string",
    "methodFamily": "descriptive|analytical|testing|experimental|observational",
    "phase": "design-only(now) | phase-4(post-G9)",
    "instance": "the demonstration case", "conditions": ["…"], "negativeCase": "tamper case",
    "metric": "string", "owner": "role", "venue": "gate/skill", "routedTo": ["fabric-security-review|fabric-performance"], "assumptions": ["G-##"]
  },
  "evaluationReport": {
    "scope": "artifact | full-package", "gate": "G8 | post-G9",
    "results": [ { "objectiveId": "OBJ-#", "method": "…", "evidence": "…", "result": "met|partial|not-met", "frames": ["…"], "type": "fact|recommendation" } ],
    "iterationDecision": "advance-to-G9 | loop-back:<gate>", "openFindings": ["…"],
    "communication": { "forResearchers": "…", "forPractitioners": "…" }, "assumptions": ["G-##"]
  }
}
```

## Examples

**Beginner (Frame & Map).** Ask: "Where does the 'no PII plaintext on-chain' ADR sit, and what's its
research question?" → artifact = **method** (the per-section salted-digest anchoring pattern,
ADR-0011); primary activity **3 / G4**; RQ predicate = *"a whole profile section can be anchored and
later verified, client-side, without any section value or salt ever crossing to a peer."*
cycleBalance: rigor **strong** (corpus + PbD frame + ratified ADR-0011/ADR-0020), relevance
**strong** on the anchoring unit itself (`[prd: §4]`, gap G-05 closed) but still **provisional** on
build-time specifics (canonicalization pin, G-24/PB-3). Emits `dsrmPhaseMap`.

**Intermediate (Experiment Planner).** Ask: "Plan the anchor-then-verify demonstration." → objective
predicate as above; methodFamily **Descriptive** *now* (walk the traceability-spine scenario), with a
**Testing** plan marked **Phase-4 (post-G9)**; instance = anchor a `PERSONAL` profile-section write;
negative case = mutate the off-chain section value and show client-side recomputation no longer
matches the anchored `DataHash`; metric = verification match/mismatch with zero plaintext disclosure;
routes white-box chaincode depth to `fabric-security-review`. Emits
`experimentPlan`.

**Enterprise (Evaluation Report Generator).** Ask: "Produce the G8 evaluation report for the package."
→ iterate every objective, method used (Descriptive + Analytical only, per the design-only boundary),
result + frame per row, Fact vs. Recommendation; one CONFIRMED finding on the threat model ⇒
`iterationDecision: loop-back:G6`; communication split researcher/practitioner; then publish to
Confluence via the Atlassian MCP. Emits `evaluationReport`.

## Guardrails

- **Design-only boundary is absolute.** No experiment/evaluation may presuppose a running prototype
  before G9; Testing/Experimental items are labelled **Phase-4 (post-G9)** `[brief: agents-guide.md]`.
- **Objectives are assumptions.** Every solution objective / success metric carries `[ASSUMPTION]
  (gap G-01)` until HLF-6 / Confluence resolve it; never present one as ratified.
- **Attribute DSRM literature by author/year.** Never fabricate a `[docs:]` corpus citation for a
  methodology claim — DSRM is not in the Fabric corpus.
- **Do not duplicate sibling depth.** Security-testing method → `fabric-security-review`; performance
  method → `fabric-performance` (gap G-08); deep literature search → `deep-research`.
- **G8 is a hard iteration gate.** A CONFIRMED finding must loop back before G9, not be waved through.
- **Orchestrate, don't inline, web research.** Frame the question and hand it to `deep-research`;
  consume its cited report — do not run ad-hoc searches inside this skill.

## Evaluation criteria (judge this skill's own output)

- [ ] Every objective is stated as a **testable predicate**, not a yes/no question.
- [ ] The work item is placed on **exactly one primary** DSRM activity + gate, with an artifact type.
- [ ] The chosen method family **matches the phase** (Descriptive/Analytical now; Testing/Experimental
      only Phase-4).
- [ ] **Fact vs. Recommendation** separated; every requirement specific carries a gap ID.
- [ ] Each evaluation result ties to **≥1 security frame** (Layer E).
- [ ] Cross-references used instead of restating `DSRM.md`, `fabric-security-review`, or
      `fabric-performance` content.
- [ ] The iteration decision honors the **G8 hard boundary**.

## MCP integration

- **huggingface `paper_search`** (`mcp__plugin_huggingface-skills__paper_search`) — locate the
  seminal DSRM / security-frame papers and related work to feed the **rigor cycle** and Communication
  bibliography. For a broad, multi-source, fact-checked review, **hand off to the `deep-research`
  skill** rather than relying on single-tool search.
- **`deep-research` skill** — orchestrated for deep literature web-research: this skill supplies the
  framed question (Mode 1 output) and consumes the returned cited report; it does not fan out searches.
- **Atlassian MCP (Rovo)** — publish the `evaluationReport` / `experimentPlan` to Confluence
  (`createConfluencePage` / `updateConfluencePage`) and link to Jira Epic **HLF-6**.
  `[ASSUMPTION] (gaps G-01/G-14)` the publish target is unreachable from this environment today (only
  `jurnal.atlassian.net` is granted; HLF-6 lives on `chandrakurniawan.atlassian.net`) — stage the page
  content and record the intended space/parent until access lands.

## Reusable prompts

- **Frame & Map:** *"Using dsrm-research-assistant Frame & Map mode, classify <artifact>, frame its
  research question as a testable predicate, place it on the DSRM activity/gate map, and report the
  rigor/relevance balance with gap IDs."*
- **Experiment Planner:** *"Using dsrm-research-assistant Experiment Planner mode, plan the
  demonstration for <objective>; pick the evaluation-method family for the current phase, define the
  instance, success metric, and tamper/negative case, and route any security-testing or performance
  depth to the owning skill."*
- **Evaluation Report Generator:** *"Using dsrm-research-assistant Evaluation Report mode, generate
  the G8 evaluation report for <scope>: per objective give method, evidence, result, and frame;
  separate fact from recommendation; decide advance-to-G9 vs loop-back; draft the Confluence page."*
- **Literature synthesis (orchestration):** *"Frame the research question for <design choice> vs
  <security frame>, then hand it to the deep-research skill and cite the result into the rigor cycle;
  use paper_search for the seminal references."*

## Behavioral rules
- **Cite the corpus/context doc.** Ground methodology claims in `../../context/DSRM.md` and the DSRM
  literature by author/year; ground Fabric factual claims in the official Fabric 2.5 corpus
  (`[docs: …]`) via the owning `fabric-*` skill. When sources conflict, say so and follow the
  primary source. [FR-8]
- **Show the reasoning and the trade-off.** Never present a tunable (block size, endorsement policy,
  state DB) as a universal truth — give the "it depends" and the axis it depends on. [FR-9]
- **Fact vs. recommendation.** Mark documented facts (cited) distinctly from engineering judgment
  ("recommendation:"). [FR-10]
- **Flag the blast radius.** Whenever a suggested action carries a production, security, or
  performance implication, state it before the how-to. [FR-11]
- **State assumptions.** When the user's context is incomplete, mark `[ASSUMPTION]` and invite
  correction rather than guessing silently. [FR-12]

## Cross-references (read on demand)

- `../../context/DSRM.md` — **the knowledge**: six activities, the exact gate mapping (§2), the three
  coupled cycles (§3), the five evaluation-method families for a security prototype (§4), and
  Communication (§5). Read before any mode.
- `../../11-execution/knowledge-graph.md` — Layer E (security frames) and Layer F (the same gate map);
  the traceability spine (§3) supplies the Descriptive-evaluation scenarios.
- `../../11-execution/grounding-gaps.md` — the `G-01..G-29` register every `[ASSUMPTION]` points at.
- `../../context/README.md` — the agent→context access matrix (the `dsrm-researcher` column).
- Sibling skills: `fabric-security-review` (Analytical/Testing depth for control reviews),
  `fabric-performance` (performance evaluation, gap G-08), `deep-research`, `rfc-writer` /
  `architecture-design` (produce the artifact this skill frames and evaluates), `project-planning`
  (delivery scheduling, not research phases).

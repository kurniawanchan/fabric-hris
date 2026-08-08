# hlf-orchestrator — the Engineering Orchestrator Workflow

The brief's "Engineering Orchestrator" realized as a **top-thread Workflow**, not a runtime subagent
(Claude Code subagents are one level deep — see
[`../00-architecture/multi-agent-architecture.md`](../00-architecture/multi-agent-architecture.md)). It
is the only actor that dispatches agents (via the Task tool) and it sequences the DSRM gates G0→G9,
embedding a blocking adversarial verify at each. This file is the reusable definition; it is **not**
executed before G9 approval (design-only).

## Contract

- **Input:** the package root (`agent-suite/`) + the knowledge-graph spine + the grounding-gaps register.
- **Sequence:** a `pipeline()` of phase-gates G0..G9, strictly ordered (each consumes prior outputs).
- **Per gate:** fan out author agents (`parallel()`), pair each with a fresh-context **critic**, then a
  **blocking verify** (file-census + reference-lint + gate-specific adversarial check). A `high`
  CONFIRMED finding loops the deliverable back to its owning gate (G8 is the hard iteration boundary).
- **Dispatch scoping:** hand each agent only its `●` primary context docs (per
  [`../context/README.md`](../context/README.md)); agents return structured reports (stigmergic — no
  agent-to-agent channel).
- **Human gate:** G9 requires an explicit human approval; agent verdicts are **not** approval.
- **No-op recovery:** verification is deterministic and blocking — never trust an agent's self-report.
  On a 0-tool-call no-op, sharpen-and-re-dispatch (retry #1); author keystone single files directly.
  (See [`../09-review/review-workflow.md`](../09-review/review-workflow.md) §5.)

## Gate → phase map (DSRM)

| Gate | Phase | Lead agents | Gate check |
|------|-------|-------------|------------|
| G0 | Problem / Objectives | pm, dsrm-researcher | citation-falsification; assumptions tagged |
| G1 | Objectives | fabric-architect, backend-engineer, security-architect | duplication + citation-rot |
| G2 | Design&Dev | (skill authoring) | skill redundancy vs suite |
| G3 | Design&Dev | architect, reviewer | cross-reference linter |
| G4 | Design&Dev | fabric-architect, fabric-engineer, security-architect | ADR rigor |
| G5 | Design&Dev | fabric-architect, fabric-engineer, architect | confidentiality invariant + ADR consistency |
| G6 | Design&Dev | security-architect, reviewer | control→threat→mechanism traceability |
| G7 | Design&Dev → Demo prep | qa, pm | control→test coverage; task→design traceability |
| G8 | Evaluation | reviewer ×N (consistency/traceability/security/completeness) | whole-package sweep |
| G9 | Communication | architect, sre, pm | **human approval** |

## Representative Workflow skeleton (illustrative — not run pre-G9)

```js
export const meta = { name: 'hlf-orchestrator', description: 'DSRM-gated SDLC for the HRIS-on-Fabric prototype', phases: [/* G0..G9 */] }

for (const gate of GATES) {          // G0..G9, strictly sequential
  phase(gate.title)
  // 1. fan out authors (each scoped to its primary context docs)
  const drafts = await parallel(gate.authors.map(a => () => agent(a.prompt, { label: a.label, phase: gate.title })))
  // 2. adversarial verify (fresh context; falsify, don't bless)
  const verdict = await agent(gate.verifyPrompt, { label: `verify:${gate.id}`, phase: gate.title, schema: GATE_VERDICT })
  // 3. census + heal: any missing file or 0-tool-call no-op → sharpen-and-re-dispatch (retry #1), else author directly
  // 4. block: a `high` CONFIRMED finding loops back to the owning gate before advancing
  if (gate.id === 'G9') break        // human approval gate — stop, do not begin prototype code
}
```

## Post-approval (Phase 4, Demonstration)

Only **after** the G9 human approval does the orchestrator dispatch the build backlog
([`../06-roadmap/implementation-backlog.md`](../06-roadmap/implementation-backlog.md)) — and only once
the blocking pre-build tasks (G-18, G-23, G-24, G-01, G-05) are closed. Prototype code is written then,
never before.

<!-- REUSED agent spec — by-reference. PLUGIN agent (no ~/.claude/agents file); specced from its known role. -->

# Agent Spec — `code-reviewer`

> **Kind:** REUSED — **plugin agent** (backed by the `code-review` / `code-review:code-review` skill). **[ASSUMPTION]** not present as a standalone file in `~/.claude/agents/` today — realization deferred to **gap G-23** (dedicated agent file vs a skill-backed hat on `reviewer`); see [`agent-catalog.md`](../agent-catalog.md) grounding note.
> **Division:** Quality
> **Realizes brief role(s):** Reviewer / code-quality review (REUSE) `[brief: agents-guide.md]`
> **Operational file:** none today — `code-review` skill loaded by `reviewer`, or a future dedicated file (gap G-23).

Specced from the known plugin role (severity-scored code-quality review against a checklist). Only the project delta is below; do not restate the generic review protocol.

## 1. Purpose
The dedicated code-**quality** review capability (correctness, simplification, efficiency, test-coverage) — distinct from `reviewer`'s security/OWASP focus and from `code-simplifier`'s apply-the-fix mandate. In this package it is a **G8 quality lens**, overlapping `reviewer`'s code-review mode.

## 2. Responsibilities (delta)
- **G8** — quality review of the authored design specs/docs for internal consistency, altitude, and reuse-discipline (no duplicated Fabric-suite content); **post-G9** — quality review of prototype diffs.
- Reports findings (most-severe first) via the `ReportFindings` tool when the review instructions ask for it; otherwise a severity-scored report.
- Does NOT assess security (→ `reviewer` security mode / `security-architect`), apply fixes (→ `code-simplifier` / `backend-engineer`), or design (→ `architect`).

## 3. Inputs
The diff/artifact under review, the relevant convention docs, `[brief: agents-guide.md]` reuse mandate + `AUTHORING-CONTRACT §7` (one home per concept).

## 4. Outputs
A severity-ranked findings list (correctness / simplification / efficiency / test-coverage) → `09-review/` or inline via `ReportFindings`.

## 5. Dependencies
Overlaps `reviewer` (quality vs security split — do not double-report the same finding). Hands fixes to `backend-engineer`/`fabric-engineer` or `code-simplifier`.

## 6. Skills used
`code-review` (`code-review:code-review` plugin) — quality checklist, severity scoring.

## 7. Context docs (per [access matrix](../../context/README.md))
Not a row in the matrix (realization pending, gap G-23). Reads, as-needed, the convention docs relevant to the artifact under review: `GO-CONVENTIONS`, `PHP-CONVENTIONS`, `GO-TESTING`.

## 8. Tools
Read, Grep, Glob, Bash, Skill, `ReportFindings`.

## 9. MCP servers
None. Optional `serena`/`lumen` for navigation. Per [`../../04-mcp/mcp-architecture.md`](../../04-mcp/mcp-architecture.md).

## 10. Quality checklist (project-specific)
- [ ] Quality-only — does not hunt for or report security findings (that is `reviewer`'s security mode).
- [ ] Verified findings only, most-severe first; each with a concrete failure scenario.
- [ ] Enforces the reuse/one-home rule: flags any doc that copies Fabric-suite content instead of cross-referencing.
- [ ] Does not apply fixes — reports them.

## 11. Success criteria
G8 quality findings are actionable and non-overlapping with `reviewer`'s security findings; the reuse/one-home discipline holds across the package.

---
*Traceability: knowledge-graph Layer F (DSRM Evaluation). **Depends on gap G-23** (agent-file realization).*

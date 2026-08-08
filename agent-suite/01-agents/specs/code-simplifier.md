<!-- REUSED agent spec — by-reference. PLUGIN agent (no ~/.claude/agents file); specced from its known role. -->

# Agent Spec — `code-simplifier`

> **Kind:** REUSED — **plugin agent** (backed by the `refactor` / `refactoring-workflow` / `simplify` skills). **[ASSUMPTION]** not present as a standalone file in `~/.claude/agents/` today — realization deferred to **gap G-23** (dedicated agent file vs a skill-backed hat on `backend-engineer`); see [`agent-catalog.md`](../agent-catalog.md) grounding note.
> **Division:** Quality
> **Realizes brief role(s):** Refactoring (REUSE) `[brief: agents-guide.md]`
> **Operational file:** none today — `refactor`/`refactoring-workflow`/`simplify` skills loaded by `backend-engineer`, or a future dedicated file (gap G-23).

Specced from the known plugin role (reviews changed code for reuse/simplification/efficiency and **applies** the fixes — quality only, not bug-hunting). Only the project delta is below.

## 1. Purpose
The refactoring/simplification capability — reduces duplication and complexity in the prototype and in the authored artifacts, and **applies** the cleanups (unlike `code-reviewer`, which only reports). Quality only; it does not find bugs (that is `reviewer` / `code-reviewer`).

## 2. Responsibilities (delta)
- **G8** — simplify authored specs/docs where the package repeats itself (collapse duplication, tighten altitude), consistent with the reuse mandate.
- **post-G9** — refactor prototype Go/PHP/chaincode for reuse, simplification, and efficiency after a change lands.
- Does NOT hunt for or fix bugs (→ `reviewer`/`code-reviewer` find, `backend-engineer`/`fabric-engineer` fix), add features, or change behaviour.

## 3. Inputs
The changed code/artifact set, the relevant convention docs, the reuse mandate (`AUTHORING-CONTRACT §7`).

## 4. Outputs
Applied edits (behaviour-preserving) + a short summary of what was simplified and why.

## 5. Dependencies
Runs after a change lands and before/alongside `reviewer`/`code-reviewer`. Hands off to `qa` if a refactor needs test updates, `reviewer` for the final diff review.

## 6. Skills used
`refactor` · `refactoring-workflow` · `simplify` (reuse / simplification / efficiency passes).

## 7. Context docs (per [access matrix](../../context/README.md))
Not a row in the matrix (realization pending, gap G-23). Reads, as-needed, `GO-CONVENTIONS`, `PHP-CONVENTIONS`.

## 8. Tools
Read, Edit, Grep, Glob, Bash, Skill.

## 9. MCP servers
None. Optional `serena` for symbol-level refactors. Per [`../../04-mcp/mcp-architecture.md`](../../04-mcp/mcp-architecture.md).

## 10. Quality checklist (project-specific)
- [ ] **Behaviour-preserving** — refactors only; no feature or behaviour change.
- [ ] Quality-only — does not hunt for bugs (route correctness concerns to `reviewer`).
- [ ] Every applied edit keeps tests passing (post-G9) or leaves the artifact citation-intact (design phase).
- [ ] Respects the one-home rule — prefers cross-reference over copied content.

## 11. Success criteria
Reduced duplication/complexity with tests still green (post-G9) or citations preserved (design phase); no behaviour change introduced.

---
*Traceability: knowledge-graph Layer F (DSRM Evaluation). **Depends on gap G-23** (agent-file realization).*

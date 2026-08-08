<!--
TEMPLATE — copy to agent-suite/01-agents/specs/<agent-name>.md and fill every field.
This is the DESIGN SPEC for an agent (Deliverable 3, the brief's 10 required fields).
An OPERATIONAL agent that runs in Claude Code is the .claude/agents/<name>.md file derived
from this spec (YAML frontmatter: name/description/tools + a prose body). Keep the two in sync:
the spec is the "why/contract", the .claude/agents file is the "runtime instruction".
Reused org agents (architect, backend-engineer, reviewer, qa, sre, pm) are specced here by
REFERENCE — point at ~/.claude/agents/<name>.md and record only the delta for this project.
-->

# Agent Spec — `<agent-name>`

> **Kind:** REUSED (`~/.claude/agents/<name>.md`) · THIN specialization (base: `<base>`) · NEW
> **Division:** Executive / Architecture / Engineering / Security / Quality / Testing / Research / Documentation / DevOps
> **Realizes brief role(s):** `<one or more of the ~40 brief-named roles this covers>`
> **Operational file:** `.claude/agents/<name>.md` (NEW/THIN only) — or "n/a, reused as-is"

## 1. Purpose
One sentence: what this agent is for and the boundary that separates it from its siblings.

## 2. Responsibilities
- Bulleted, verb-first, each a discrete accountable outcome.
- Name what it does NOT do (route-away to the sibling that owns it).

## 3. Inputs
- Artifacts/context it consumes (e.g. `context/FABRIC-PRIVATE-DATA.md`, an ADR, a PRD section, a gap ID).

## 4. Outputs
- Concrete deliverables it produces, with target paths (e.g. `05-adr/ADR-0003-*.md`, `08-security/*`).

## 5. Dependencies
- Upstream agents it waits on; downstream agents it hands off to. Reference the orchestration flow.

## 6. Skills used
- Existing skills it invokes (e.g. `architecture-design`, `fabric-chaincode-dev`, `security-review`).
- Every skill named here MUST resolve to a real skill (the G3 cross-reference linter enforces this).

## 7. Context docs
- The `context/*` docs this agent reads — and ONLY those relevant to its role (brief §5 rule).

## 8. Tools
- Claude Code tools it needs (Read, Edit, Write, Bash, Grep, Glob, Skill, Task, WebFetch, …).

## 9. MCP servers
- MCP servers it uses (Atlassian, serena, lumen, context7, semgrep/Guardian, Playwright, Terraform, huggingface) + why. Must resolve to an available server or be marked `[substitute: …]` (see `04-mcp/`).

## 10. Quality checklist
- [ ] Objective, per-invocation checks this agent must satisfy before declaring done.
- [ ] Grounding/citation discipline where it authors docs.

## 11. Success criteria
- The measurable/observable condition that means this agent succeeded at its task.

---
*Traceability: link the knowledge-graph layer(s) and any gap IDs this agent's work depends on.*

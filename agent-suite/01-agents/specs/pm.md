<!-- REUSED agent spec — by-reference. Full definition lives in the base file; this records ONLY the project delta. -->

# Agent Spec — `pm`

> **Kind:** REUSED — base: [`~/.claude/agents/pm.md`](file:///Users/chan/.claude/agents/pm.md)
> **Division:** — (no 1:1 brief role; backs SDLC entry + exit) `[brief: agents-guide.md]`
> **Realizes brief role(s):** none directly. Fills the requirements-in / tickets-out bookends the brief's nine divisions leave implicit (see [`agent-catalog.md`](../agent-catalog.md) grounding note).
> **Operational file:** n/a — reused as-is.

Base agent is the source of truth for behaviour (Job 1 = source PRD from Bitbucket; Job 2 = decompose design into Jira tickets; hard rules). Only the project delta is below.

## 1. Purpose
The SDLC bookend for this package: pull the feature requirements into a PRD (entry) and decompose the landed design into per-role tickets (exit). In this project **both sources are currently unreachable**, which shapes the delta below.

## 2. Responsibilities (delta)
- **Job 1 (entry, G0/G1)** — source the PRD from the Talenta product workspace on Bitbucket. **Environment constraint:** the SCOPE/IMPLIED-REQUIREMENTS discovery returned no content and Jira `HLF-6` / the Confluence space are unreachable ([`knowledge-graph.md`](../../11-execution/knowledge-graph.md) header). The project therefore runs in **proceed-with-assumptions** mode: `pm` records what a requirement doc *would* fix and tags each open item `[ASSUMPTION] (gap G-##)` in [`grounding-gaps.md`](../../11-execution/grounding-gaps.md) — it does **not** invent requirements.
- **Job 2 (exit, G7)** — once `architect`'s G5 design lands, decompose it into backend/chaincode/QA tickets and one QA ticket per user story, present the plan for confirmation, then create via Atlassian MCP (blocked until Jira reachable; audit report authored regardless into `docs/tickets/`).
- Does NOT do architecture (→ `architect`), estimation, or sprint management.

## 3. Inputs
The Talenta product workspace (Bitbucket, via WebFetch — currently no matching doc), `architect`'s G5 design + `docs/prd/` user stories (Job 2), `docs/api/openapi.yaml`.

## 4. Outputs
Job 1: `docs/prd/<slug>.md` **or**, when the source is absent, a documented "no requirement doc found" + the gap entries it opens. Job 2: ticket plan + audit report `docs/tickets/<slug>-<YYYY-MM-DD>.md`.

## 5. Dependencies
Job 1 feeds `architect` (design from PRD). Job 2 waits on `architect` (design), `qa` (user-story coverage), and consumes tickets to `backend-engineer`/`fabric-engineer`/`qa` via the `feature:<slug>` label.

## 6. Skills used
`requirements-elicitation` (Job 1, Bitbucket fetch-and-confirm) · `jira-ticketing` (Job 2, decomposition + Atlassian create).

## 7. Context docs (per [access matrix](../../context/README.md))
**Primary `●`:** `DSRM` (frames the problem-identification and objectives phases).
**As-needed `○`:** `PRIVACY-BY-DESIGN`, `ADR`.

## 8. Tools
Base set: Read, Edit, Write, Glob, Grep, Skill, TodoWrite, AskUserQuestion, ToolSearch, WebFetch.

## 9. MCP servers
**Atlassian** — Jira ticket creation (Job 2) and any Confluence requirement lookup. **Currently unreachable in this environment** — treat as `[blocked]`; author the audit trail offline. Per [`../../04-mcp/mcp-architecture.md`](../../04-mcp/mcp-architecture.md).

## 10. Quality checklist (project-specific)
- [ ] **Never invents requirements** — no source doc means "stop and record the gap", per proceed-with-assumptions mode `[brief: agents-guide.md]`.
- [ ] Every ticket traces to a concrete design/PRD/OpenAPI element; no ticket created before explicit user "yes".
- [ ] Records the Bitbucket source path + branch when a doc *is* found; open items become `[ASSUMPTION] (gap G-##)`, not guesses.
- [ ] No architecture, estimation, or assignee decisions.

## 11. Success criteria
Job 1: either a source-traceable PRD, or an explicit no-source finding with the gaps opened. Job 2: a confirmed ticket plan + complete audit report (creation deferred cleanly while Jira is unreachable).

---
*Traceability: knowledge-graph Layer F (DSRM Problem identification + Objectives), grounding header (proceed-with-assumptions). Depends on gaps G-01…G-22 (requirements), G-11 (role mapping).*

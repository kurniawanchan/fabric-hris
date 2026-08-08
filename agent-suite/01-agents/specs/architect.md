<!-- REUSED agent spec — by-reference. Full definition lives in the base file; this records ONLY the project delta. -->

# Agent Spec — `architect`

> **Kind:** REUSED — base: [`~/.claude/agents/architect.md`](file:///Users/chan/.claude/agents/architect.md)
> **Division:** Architecture / Documentation
> **Realizes brief role(s):** Solution Architect (REUSE) · Principal SW Engineer (COLLAPSE, with `backend-engineer`) · ADR Writer (REUSE) · Technical Writer (REUSE) · API Documentation Writer (REUSE) · Confluence Publisher (COMPOSE — an `hlf-orchestrator` Workflow step, `rfc-writer` + Atlassian MCP, not a standalone agent) `[brief: agents-guide.md]`
> **Operational file:** n/a — reused as-is.

Base agent is the source of truth for behaviour (artifact-selection, hard rules, hand-offs, tone). Do **not** restate it. Only the project delta is below.

## 1. Purpose
The design authority for this package: turns the grounded knowledge graph and requirements into the system-level architecture, ADRs, diagrams, and API contracts — never Fabric-internal topology (→ `fabric-architect`), security frames (→ `security-architect`), or prototype code (→ `backend-engineer`/`fabric-engineer`, and only after G9).

## 2. Responsibilities (delta)
- **G3** — author the multi-agent architecture doc + C4/sequence diagrams in `00-architecture/`; keep the [`agent-catalog.md`](../agent-catalog.md) mapping consistent.
- **G4** — write the ADRs that **ratify or revise** the load-bearing chains and working assumptions ([`knowledge-graph.md`](../../11-execution/knowledge-graph.md) §3; gaps **G-02 / G-03 / G-09 / G-10**) into `05-adr/`. *(Note, 2026-08-06: the Fabric-topology/digest/erasure-specific instances of these decisions — ADR-0011/0012/0013/0014/0015/0016/0017/0019/0020 — were already authored by `fabric-architect`/`fabric-engineer`/`security-architect` per [`05-adr/README.md`](../../05-adr/README.md), consistent with this spec's own "does NOT design Fabric orgs/channels/policies" boundary below; `architect`'s own G4 remit is the residual, non-Fabric-specific decision set, not a duplicate of those nine.)*
- **G5** — solution HLD/LLD and, where an HTTP surface exists, the OpenAPI 3.1 spec (`docs/api/openapi.yaml`).
- **G1 (○), G9** — contribute AUTHORED context docs and the package README / roadmap narrative; drive the **Confluence Publisher** compose step (publish RFCs/ADRs).
- Does NOT design Fabric orgs/channels/policies, own the security architecture, or emit application code. *(No Private Data Collection exists anywhere in the ratified design — ADR-0013/ADR-0020 retired PDC entirely, not merely deprioritized it — so there is no PDC surface left for any agent, including `architect`, to design.)*

## 3. Inputs
[`knowledge-graph.md`](../../11-execution/knowledge-graph.md) (spine), [`grounding-gaps.md`](../../11-execution/grounding-gaps.md), `[brief: agents-guide.md]`, its `●` context docs (§7), and topology/security design outputs from `fabric-architect` + `security-architect`.

## 4. Outputs
`00-architecture/*` (architecture + diagrams), `05-adr/ADR-NNNN-*.md`, `00-architecture/diagrams/*` (C4/sequence Mermaid), `docs/api/openapi.yaml` (G5), README/roadmap prose (G9), published Confluence pages (compose step).

## 5. Dependencies
Waits on `fabric-architect` (topology) and `security-architect` (frames/controls) before finalizing G5/G6-adjacent design. Hands off to `backend-engineer`/`fabric-engineer` (implementation, post-G9), `qa` (test plan from design), `pm` (ticketing from landed design).

## 6. Skills used
`architecture-design` (design docs, ADR mode, C4, sequence diagrams) · `api-documentation` (OpenAPI) · `rfc-writer` + `doc-keeper` (technical writing, Confluence publish).

## 7. Context docs (per [access matrix](../../context/README.md))
**Primary `●`:** `FABRIC-ARCHITECTURE`, `PHP-INTEGRATION`, `BLOCKCHAIN-DATA-MODEL`, `BLOCKCHAIN-INTEGRATION`, `ADR`, `SYSTEM-DIAGRAM`.
**As-needed `○`:** `FABRIC-CHANNELS`, `FABRIC-WORLD-STATE`, `FABRIC-PRIVATE-DATA`, `FABRIC-IDENTITY`, `FABRIC-POLICIES`, `GO-ARCHITECTURE`, `PHP-ARCHITECTURE`, `CRYPTOGRAPHY`, `ZERO-KNOWLEDGE-PROOF`, `PRIVACY-BY-DESIGN`, `SECURITY-BY-DESIGN`, `DSRM`.

## 8. Tools
Base set: Read, Edit, Write, Glob, Grep, Skill, TodoWrite, AskUserQuestion, WebFetch.

## 9. MCP servers
**Atlassian** — Confluence publish for the Confluence Publisher compose step (RFCs/ADRs). Authoritative mapping in [`../../04-mcp/mcp-architecture.md`](../../04-mcp/mcp-architecture.md) (G9).

## 10. Quality checklist (project-specific)
- [ ] **No prototype code before G9** — stops at the design boundary `[brief: agents-guide.md]`.
- [ ] Every "we chose X" carries at least one rejected alternative; constraints are numeric.
- [ ] ADR status is one of three terminal values — **Accepted / Rejected / Superseded by ADR-NNNN** (see [`context/ADR.md`](../../context/ADR.md)) — and each ADR names the chain/gap it resolves.
- [ ] Cites `[docs:]`/`[code:]`/`[brief:]`; every requirement-shaped specific is `[ASSUMPTION] (gap G-##)`, never a bare guess.
- [ ] Cross-references context docs and the graph — does not copy Fabric-suite content (`AUTHORING-CONTRACT §7`).

## 11. Success criteria
G3/G4/G5 artifacts pass their gates; the ADRs ratify or revise every load-bearing chain in the spine with full traceability; zero unresolved `[VERIFY]`.

---
*Traceability: knowledge-graph Layers C (services/seam), D (Fabric capabilities), E (frames), F (DSRM Design & Development). Depends on gaps G-02 (re-scoped, not closed — ADR-0011 Accepted), G-03 (ratified as channel-per-tenant — ADR-0013 Accepted), G-09 (unaffected), G-10 (reopened — ADR-0014 Accepted but the gateway-client host itself is still open). See `grounding-gaps.md` for current status of each.*

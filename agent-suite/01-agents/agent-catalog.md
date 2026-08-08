# Agent Catalog — 40 requested roles → 12 operational agents + 1 orchestrator Workflow

> **Deliverable 3 (index).** The brief `[brief: agents-guide.md]` names ~40 role titles across
> nine divisions. This catalog maps **every** one of them onto the **12 operational agents** that
> actually run in Claude Code — plus the **one orchestrator Workflow** that drives them. It is the
> single lookup that keeps forty brief roles and twelve real agents mutually consistent; the
> per-agent contracts live in [`specs/`](specs/) (each field defined by
> [`_TEMPLATE.agent.md`](_TEMPLATE.agent.md)).
>
> **Do not restate the graph.** Layer/role justifications, PII inventory, and capability citations
> live in [`../11-execution/knowledge-graph.md`](../11-execution/knowledge-graph.md); skill
> dispositions in [`../skills/skill-catalog.md`](../skills/skill-catalog.md); the agent→context
> access rules in [`../context/README.md`](../context/README.md); gate exit criteria in
> [`../README.md`](../README.md). This catalog **cross-references** those — it does not copy them.

## The hard constraint that shapes the whole roster

Claude Code **subagents are one level deep**: a subagent cannot spawn subagents `[brief: agents-guide.md]`.
So the brief's **"Engineering Orchestrator"** is **not** a runtime subagent — it is realized as a
**top-thread Workflow** (`hlf-orchestrator`, the same shape as this package's own gate workflows in
[`../prompts/`](../prompts/)). The Workflow runs in the main thread and dispatches the 12 operational
agents; it hands each dispatched agent only its primary (`●`) context docs, adding secondary (`○`) docs
on demand (per the access matrix). **No prototype application code is written before gate G9**
`[brief: agents-guide.md]`; every "Engineering" agent below produces **design specs**, not code, until
then.

## How ~40 collapse to 12

**Disposition legend**
- **REUSE** — an existing operational agent invoked as-is (org agent in `~/.claude/agents/`, or a NEW agent authored here).
- **COLLAPSE** — the brief role folds onto an existing agent as a *mode/hat* it wears; no separate agent is created.
- **NEW** — a genuinely new thin specialization authored in this package and installed into `.claude/agents/`.
- **COMPOSE** — realized as an **orchestrator Workflow step** combining an agent + an MCP server; not an agent on its own.

Result: **8 reused org agents + 4 new specializations = 12**, all steered by the 1 orchestrator Workflow.

---

## Division 1 · Executive

| Brief role | Disposition | Operational agent | Skills + context it binds |
|---|---|---|---|
| Engineering Orchestrator | **NEW** | **`hlf-orchestrator`** — a **Workflow, NOT a subagent** (top-thread; see hard constraint above) | Drives the G0–G9 gate flow; dispatches all 12 agents; binds `[brief: agents-guide.md]`, the [`README.md`](../README.md) gate table, and the [`knowledge-graph.md`](../11-execution/knowledge-graph.md) as the shared spine. No context doc is "owned" — it routes `●`/`○` docs to each agent per the [access matrix](../context/README.md). |

---

## Division 2 · Architecture

| Brief role | Disposition | Operational agent | Skills + context it binds |
|---|---|---|---|
| Solution Architect | REUSE | `architect` | `architecture-design` · `FABRIC-ARCHITECTURE`, `BLOCKCHAIN-DATA-MODEL`, `BLOCKCHAIN-INTEGRATION`, `PHP-INTEGRATION`, `ADR`, `SYSTEM-DIAGRAM` |
| Principal SW Engineer | COLLAPSE (`architect` + `backend-engineer`) | `architect` / `backend-engineer` | `architecture-design` + `coding-standards-backend` · `GO-ARCHITECTURE`, `PHP-ARCHITECTURE`, `BLOCKCHAIN-INTEGRATION` |
| Hyperledger Fabric Architect | **NEW** | `fabric-architect` | `fabric-network-architect`, `fabric-core` · all `FABRIC-*` docs (`●`), `BLOCKCHAIN-DATA-MODEL`, `SYSTEM-DIAGRAM` |
| Security Architect | **NEW** | `security-architect` | `fabric-security-review`, `security-review`, `privacy-by-design`, `zkp-designer` · `CRYPTOGRAPHY`, `FABRIC-MSP`, `FABRIC-CA`, `FABRIC-IDENTITY`, `FABRIC-PRIVATE-DATA`, `FABRIC-POLICIES`, `SECURITY-BY-DESIGN` |

---

## Division 3 · Engineering

| Brief role | Disposition | Operational agent | Skills + context it binds |
|---|---|---|---|
| Senior Go Engineer | REUSE | `backend-engineer` | `coding-standards-backend` · `GO-ARCHITECTURE`, `GO-CONVENTIONS`, `GO-TESTING`, `GO-APIS…GO-SECURITY` |
| Senior PHP Engineer | REUSE | `backend-engineer` | `coding-standards-backend` · `PHP-ARCHITECTURE`, `PHP-CONVENTIONS`, `PHP-INTEGRATION` |
| Smart Contract Engineer | **NEW** | `fabric-engineer` | `fabric-chaincode-dev` · `FABRIC-CHAINCODE`, `FABRIC-WORLD-STATE`, `FABRIC-POLICIES`, `BLOCKCHAIN-DATA-MODEL` |
| REST API Engineer | COLLAPSE (`backend-engineer`) | `backend-engineer` | `coding-standards-backend` (api-design) + `api-documentation` · `GO-APIS` |
| Database Engineer | COLLAPSE (`backend-engineer`) | `backend-engineer` | `coding-standards-backend` · `GO-DATABASE`, `FABRIC-WORLD-STATE` |
| Integration Engineer | COLLAPSE (`backend-engineer`) | `backend-engineer` | `coding-standards-backend` · `PHP-INTEGRATION`, `BLOCKCHAIN-INTEGRATION` |
| Blockchain Integration Engineer | COLLAPSE (`fabric-engineer`) | `fabric-engineer` | `fabric-chaincode-dev` (gateway-client) · `BLOCKCHAIN-INTEGRATION`, `FABRIC-CHAINCODE` |

---

## Division 4 · Security

| Brief role | Disposition | Operational agent | Skills + context it binds |
|---|---|---|---|
| Security Reviewer | REUSE | `reviewer` | `security-review`, `fabric-security-review` · `OWASP-ASVS`, `OWASP-API`, `THREAT-MODELING` |
| Cryptography Specialist | COLLAPSE (`security-architect`) | `security-architect` | `security-review` · `CRYPTOGRAPHY` |
| Privacy Engineer | COLLAPSE (`security-architect`) | `security-architect` | `privacy-by-design` · `PRIVACY-BY-DESIGN` |
| Threat Modeling | COLLAPSE (`reviewer` + `security-architect`) | `reviewer` / `security-architect` | `security-review` + `fabric-security-review` · `THREAT-MODELING`, `SECURITY-BY-DESIGN` |
| OWASP Reviewer | REUSE | `reviewer` (**+ semgrep/Guardian MCP**) | `security-review` · `OWASP-ASVS`, `OWASP-API` |

---

## Division 5 · Quality

| Brief role | Disposition | Operational agent | Skills + context it binds |
|---|---|---|---|
| Reviewer | REUSE | `reviewer` | `code-review` · `GO-CONVENTIONS`, `PHP-CONVENTIONS` |
| Refactoring | REUSE | `code-simplifier` ‡ | `refactor` / `refactoring-workflow` / `simplify` · `GO-CONVENTIONS` |
| Performance Optimizer | REUSE (`backend-engineer` + `fabric-performance`) | `backend-engineer` / `fabric-engineer` | `fabric-performance` (chain/ledger) + `backend-engineer` (app) · `GO-PERFORMANCE`, `FABRIC-WORLD-STATE` |
| Debugger | REUSE (`backend-engineer`/`general-purpose` + `fabric-troubleshooting`) | `backend-engineer` / general-purpose | `debug`, `fabric-troubleshooting` |
| RCA (Root Cause Analysis) | REUSE (`sre` + `fabric-troubleshooting`) | `sre` | `incident-postmortem`, `fabric-troubleshooting` |

‡ See the on-disk grounding note below — `code-simplifier` is **not** a standalone agent file today.

---

## Division 6 · Testing — **all 6 collapse onto `qa`**

| Brief role | Disposition | Operational agent | Skills + context it binds |
|---|---|---|---|
| Test Strategy | REUSE | `qa` | `test-strategy` · `GO-TESTING` |
| Test Spec | REUSE | `qa` | `test-strategy` |
| Unit Testing | REUSE | `qa` | `test-strategy` · `GO-TESTING` |
| Integration Testing | REUSE | `qa` | `api-test-automation` |
| Performance Testing | REUSE | `qa` (**+ `fabric-performance` Caliper**) | `fabric-performance` |
| Security Testing | REUSE | `qa` (**+ semgrep/Guardian MCP**) | `security-review`, `api-test-automation` |

---

## Division 7 · Research

| Brief role | Disposition | Operational agent | Skills + context it binds |
|---|---|---|---|
| DSRM Research Assistant | **NEW** | `dsrm-researcher` | `dsrm-research-assistant` · `DSRM` |
| Literature Reviewer | COLLAPSE (`dsrm-researcher` + `deep-research`) | `dsrm-researcher` | `deep-research` + `dsrm-research-assistant` (+ huggingface `paper_search` MCP) · `DSRM`, `ZERO-KNOWLEDGE-PROOF` |
| ADR Writer | REUSE | `architect` | `architecture-design` (ADR mode) · `ADR` |

---

## Division 8 · Documentation

| Brief role | Disposition | Operational agent | Skills + context it binds |
|---|---|---|---|
| Technical Writer | REUSE | `architect` / `doc-keeper` | `doc-keeper` / `architecture-design` · `SYSTEM-DIAGRAM` |
| API Documentation Writer | REUSE | `architect` | `api-documentation` · `GO-APIS` |
| Confluence Publisher | **COMPOSE** (Workflow step) | `hlf-orchestrator` step + `architect` | `rfc-writer` + **Atlassian MCP** (publish RFCs/ADRs to Confluence) |

---

## Division 9 · DevOps

| Brief role | Disposition | Operational agent | Skills + context it binds |
|---|---|---|---|
| Docker Engineer | REUSE (`sre` + `fabric-operations`) | `sre` | `fabric-operations`, `release-pipeline` |
| Kubernetes Engineer | REUSE (`sre` + `fabric-operations`) | `sre` (**+ Terraform MCP**) | `fabric-operations` + **Terraform MCP** |
| CI/CD Engineer | REUSE | `sre` | `release-pipeline` |

---

## The 12 operational agents

The orchestrator is a **Workflow, not a subagent** — it is listed separately below the twelve.

### Reused org agents (8) — source of truth `~/.claude/agents/<name>.md`

| Agent | One-line purpose | Spec |
|---|---|---|
| `architect` | Turns requirements into structured design docs, ADRs, C4/sequence diagrams, and API references — never freestyle prose. | [`specs/architect.md`](specs/architect.md) |
| `backend-engineer` | Translates specs into production-grade Go/PHP server code (endpoints, repos, migrations, jobs) — design-only until G9. | [`specs/backend-engineer.md`](specs/backend-engineer.md) |
| `reviewer` | Finds what's wrong before it ships: code-quality, security, OWASP, and threat-model reviews as severity-scored reports. | [`specs/reviewer.md`](specs/reviewer.md) |
| `qa` | Owns the whole test pyramid — strategy, unit/integration/perf/security test design, and runnable scaffolding. | [`specs/qa.md`](specs/qa.md) |
| `sre` | Gates releases (CI/CD, Docker, K8s) and runs after-the-fact RCA/postmortems for the Fabric + app stack. | [`specs/sre.md`](specs/sre.md) |
| `pm` | Pulls product requirements into a PRD and decomposes landed designs into per-role Jira tickets (SDLC entry/exit). | [`specs/pm.md`](specs/pm.md) |
| `code-reviewer` ‡ | Dedicated code-quality-review capability (backed by the `code-review` skill). | [`specs/code-reviewer.md`](specs/code-reviewer.md) |
| `code-simplifier` ‡ | Refactoring/simplification capability (backed by the `refactor` / `refactoring-workflow` / `simplify` skills). | [`specs/code-simplifier.md`](specs/code-simplifier.md) |

### New thin specializations (4) — authored here, installed into `.claude/agents/`

| Agent | One-line purpose | Spec |
|---|---|---|
| `fabric-architect` | Designs the Fabric 2.5 topology — orgs/MSPs, channels, PDCs, endorsement policies, ordering — over the reused `fabric-network-architect`/`fabric-core` skills. | [`specs/fabric-architect.md`](specs/fabric-architect.md) |
| `fabric-engineer` | Designs chaincode and the Fabric gateway-client integration (anchor/commitment logic) over `fabric-chaincode-dev`; no code before G9. | [`specs/fabric-engineer.md`](specs/fabric-engineer.md) |
| `security-architect` | Owns the security frames (CIA/ZKP/PbD/SbD/OWASP/Zero Trust) and crypto/privacy/key design over `fabric-security-review`, `privacy-by-design`, `zkp-designer`. | [`specs/security-architect.md`](specs/security-architect.md) |
| `dsrm-researcher` | Frames the work as Design Science Research — phase mapping, experiment planning, literature review, evaluation reports — over the new `dsrm-research-assistant` skill. | [`specs/dsrm-researcher.md`](specs/dsrm-researcher.md) |

### The orchestrator — a Workflow, not a subagent

| Name | What it is | Definition |
|---|---|---|
| `hlf-orchestrator` | The **top-thread Workflow** realizing the brief's *Engineering Orchestrator*. It drives gates G0–G9 and dispatches the 12 agents above. It is **not** a runtime subagent (subagents are one level deep and cannot spawn subagents). | [`../prompts/`](../prompts/) |

---

## Grounding notes

- **‡ On-disk reality of `code-reviewer` / `code-simplifier` — RESOLVED (packaging decision, not a gap).**
  The mandated roster names eight reused agents, but only **six** exist as files in `~/.claude/agents/`
  (`architect`, `backend-engineer`, `reviewer`, `qa`, `sre`, `pm`). `code-reviewer` and `code-simplifier`
  are **Claude Code built-in review subagent types** (their capabilities are also the installed
  `code-review` / `refactoring-workflow` / `simplify` skills that `reviewer` and `backend-engineer`
  already load). **Decision:** they remain skill-backed built-in review roles — **no new agent files are
  created.** This is a tooling fact, not a requirement gap, so it is intentionally NOT tracked in
  grounding-gaps.md (which holds requirement-derived assumptions only; G-01…G-24).
- **Specs.** All 12 specs exist under [`specs/`](specs/). Reused org agents are specced **by reference**
  (pointing at `~/.claude/agents/<name>.md`, recording only the project delta), per
  [`_TEMPLATE.agent.md`](_TEMPLATE.agent.md); the 4 NEW agents (`fabric-architect`, `fabric-engineer`,
  `security-architect`, `dsrm-researcher`) get full specs.
- **`pm` has no 1:1 brief role in the nine divisions** — it backs SDLC entry (PRD via
  `requirements-elicitation`) and exit (ticketing via `jira-ticketing`), invoked by the orchestrator
  Workflow rather than mapped from a single Executive/Documentation title. Retained in the roster as a
  reused org agent.
- **Every skill named above resolves to a real skill** in the org set or the 8-skill
  `fabric-skill-suite` (`fabric-chaincode-dev`, `fabric-core`, `fabric-identity-security`,
  `fabric-network-architect`, `fabric-operations`, `fabric-performance`, `fabric-security-review`,
  `fabric-troubleshooting`) or the 3 new skills authored here (`dsrm-research-assistant`,
  `zkp-designer`, `privacy-by-design`). Dispositions are justified in
  [`../skills/skill-catalog.md`](../skills/skill-catalog.md), not restated here.
- **MCP bindings** (semgrep/Guardian, Atlassian, Terraform, huggingface `paper_search`) are named as
  role deltas only; the authoritative server→agent→permission mapping is `04-mcp/mcp-architecture.md` (G9).

---

*Traceability: this catalog indexes the roster the [`knowledge-graph.md`](../11-execution/knowledge-graph.md)
reasons over (Layers A–F) and enforces the [`context/README.md`](../context/README.md) access matrix. Role
titles and the seven security frames are directives of `[brief: agents-guide.md]`. `code-reviewer` /
`code-simplifier` are skill-backed built-in review roles (no new agent file) — see the grounding note above.*

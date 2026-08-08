# Skill Catalog — brief §6 skills → disposition

**Headline: ~28 of the ~35 skills the brief asks for already exist and are reused or composed;
only 3 are genuinely new.** The brief (`../../agents-guide.md` §6) lists ~35 skills across six
categories. Recreating the ones the org and the `fabric-skill-suite` already own would violate the
reuse mandate and the AUTHORING-CONTRACT's "one home per concept" rule (§7), and is the primary
mitigation for risks R-03 (skill duplication) and R-02 (scope blow-up) in
[`../10-risk/risk-register.md`](../10-risk/risk-register.md).

**Disposition legend:** **REUSE** = invoke an existing skill as-is · **COMPOSE** = combine/mode of
existing skills · **NEW** = authored in this package (`agent-suite/skills/`, installed into `.claude/skills/`).

## Core Engineering
| Brief skill | Disposition | Target |
|---|---|---|
| Problem Decomposition | REUSE | `project-planning` / `breakdown-plan` / `superpowers:brainstorming` |
| CRUD Generator | REUSE | `backend-engineer` + `coding-standards-backend` (no standalone skill) |
| REST API Design | REUSE | `coding-standards-backend` (api-design) + `api-documentation` |
| Error Handling | REUSE | `coding-standards-backend` (error-handling) |
| Validation Logic | REUSE | `coding-standards-backend` |
| Performance Optimization | REUSE | `fabric-performance` (chain/ledger) + `backend-engineer` (app) |
| Refactoring | REUSE | `refactoring-workflow` / `code-simplifier` |

*Why reuse:* these are language/general engineering concerns already codified in `coding-standards-backend` and the org agents; a Fabric-HRIS project adds no new decomposition or CRUD semantics.

## Hyperledger — **all 8 REUSE the existing `fabric-*` suite**
| Brief skill | Disposition | Target |
|---|---|---|
| Chaincode Generator | REUSE | `fabric-chaincode-dev` |
| Fabric Network Designer | REUSE | `fabric-network-architect` |
| Fabric Identity & MSP | REUSE | `fabric-identity-security` |
| Ledger Data Modeling | REUSE | `fabric-chaincode-dev` + `fabric-core` |
| Private Data Collection Designer | REUSE | `fabric-chaincode-dev` (private-data) |
| Endorsement Policy Designer | REUSE | `fabric-chaincode-dev` + `fabric-identity-security` |
| Fabric Gateway Integration | REUSE | `fabric-chaincode-dev` (gateway-client) |
| Blockchain Security Review | REUSE | `fabric-security-review` |

*Why reuse:* the entire Hyperledger category **is** the installed 8-skill suite the brief tells us to orchestrate. The `context/FABRIC-*.md` stubs point each topic at its owning skill.

## Architecture
| Brief skill | Disposition | Target |
|---|---|---|
| System Design | REUSE | `architecture-design` |
| ADR Writer | REUSE | `architecture-design` (ADR mode) / `architect` agent |
| Architecture Review | REUSE | `architecture-design` / `code-review` |
| Threat Modeling | REUSE | `security-review` + `fabric-security-review` |
| Sequence Diagram Generator | REUSE | `architecture-design` (Mermaid) |
| C4 Model Generator | REUSE | `architecture-design` |

## Quality
| Brief skill | Disposition | Target |
|---|---|---|
| Code Reviewer | REUSE | `code-review` |
| Security Reviewer | REUSE | `security-review` |
| Performance Reviewer | REUSE | `fabric-performance` + `backend-engineer` |
| Test Generator | REUSE | `test-strategy` + `api-test-automation` |
| API Contract Validator | REUSE | `api-test-automation` + `api-documentation` |

## Research
| Brief skill | Disposition | Target |
|---|---|---|
| DSRM Research Assistant | **NEW** | `dsrm-research-assistant` |
| Literature Review | COMPOSE | `deep-research` + huggingface `paper_search` MCP (orchestrated by `dsrm-research-assistant`) |
| Experiment Planner | COMPOSE | **mode** of `dsrm-research-assistant` |
| Evaluation Report Generator | COMPOSE | **mode** of `dsrm-research-assistant` |

## Documentation
| Brief skill | Disposition | Target |
|---|---|---|
| Confluence Writer | COMPOSE | `rfc-writer` (Confluence publish) + Atlassian MCP |
| RFC Writer | REUSE | `rfc-writer` |
| API Documentation Writer | REUSE | `api-documentation` |
| Technical Specification Writer | REUSE | `rfc-writer` / `architecture-design` |

## The 3 genuinely-NEW skills (authored here)
| Skill | Why new (not covered by any existing skill) | Grounds in |
|---|---|---|
| [`dsrm-research-assistant`](dsrm-research-assistant/SKILL.md) | Design Science Research methodology — phase framing, experiment planning, evaluation reports. No org skill covers DSRM. Bundles Experiment Planner + Evaluation Report as modes. | `context/DSRM.md` |
| [`zkp-designer`](zkp-designer/SKILL.md) | Decide **whether/how** to apply ZKP to HRIS PII via Fabric Idemix. `fabric-identity-security` covers MSP/CA plumbing but not the ZKP design decision. | `context/ZERO-KNOWLEDGE-PROOF.md`, `[docs: idemix.rst]` |
| [`privacy-by-design`](privacy-by-design/SKILL.md) | PbD 7 principles + DPIA + PII data-map + right-to-erasure mapping to Fabric. No org skill covers privacy engineering. Ships a DPIA template asset. | `context/PRIVACY-BY-DESIGN.md` |

## MCP backing, by skill category
- **Hyperledger / Quality (Fabric)** — no MCP required (skills are self-contained); optional serena for chaincode nav.
- **Architecture / Documentation** — Atlassian MCP (publish RFCs/ADRs to Confluence).
- **Research** — huggingface `paper_search` (literature), Atlassian (publish findings), `deep-research` skill.
- **Quality (app code)** — semgrep/Guardian (SAST), Playwright/chrome-devtools (E2E), serena/lumen (code nav).
- Full mapping in [`../04-mcp/mcp-architecture.md`](../04-mcp/mcp-architecture.md) (G9).

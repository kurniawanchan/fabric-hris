# MCP Architecture (Deliverable 7)

Which MCP servers the suite uses, by agent, with permissions and example workflows. **MCP is optional
leverage** — every skill and agent works without it; MCP adds live data (Atlassian requirements, code
nav, SAST, docs). The mapping below reflects what is *actually available* in this environment, not the
brief's wishlist (deltas at the end).

## Wired servers

| MCP server | Purpose | Agents | Permissions (least-privilege) | Example workflow |
|---|---|---|---|---|
| **Atlassian (Rovo)** | Read Jira epic HLF-6 + Confluence PRD/architecture; publish ADRs/RFCs/roadmap | pm, dsrm-researcher, architect, orchestrator | Jira read; Confluence read + page write on the project space | G0 pull requirements (blocked here — site not granted); G9 publish the package to Confluence/TALDOC |
| **serena** | Semantic code nav + edit across Go (EMS) + PHP (talenta-core); `write_memory`/`read_memory` for cross-session facts | backend-engineer, fabric-engineer, reviewer | repo read/edit on the two sibling repos; project memory | G1 derive GO-*/PHP-* context; Phase-4 chaincode/service authoring |
| **lumen** | Fast semantic search over this repo (indexed) → knowledge-graph retrieval | all, Explore | read-only index of `fabric-talenta` | any "where is X" discovery over the package/suite |
| **context7** | Live library docs (Fabric SDK, Go modules, Yii2) | backend-engineer, fabric-engineer, fabric-architect | read-only doc fetch | Phase-4: resolve Fabric Gateway SDK / contractapi API details |
| **semgrep / Guardian** | SAST, secrets, supply-chain scanning → OWASP/security-testing | reviewer, security-architect, qa, sre (CI) | read code; findings API | ST-9 SAST gate on chaincode + anchor-service (DEP-3 CI) |
| **Playwright + chrome-devtools** | Browser E2E / Lighthouse (if a UI surface is added to the demo) | qa | headless browser | E2E of an audit-viewer UI (out of core scope) |
| **Terraform** | IaC for the Alicloud K8s deploy | sre | plan/read; apply gated on human confirm | DEP-1 provision the Fabric network + anchor-service infra |
| **huggingface (`paper_search`)** | Academic literature for the DSRM research spine | dsrm-researcher | read-only search/fetch | literature synthesis for the evaluation report |
| **MongoDB** *(conditional)* | Only if a service uses Mongo — the repos use MySQL, so **not wired** by default | backend-engineer | — | n/a for this design |

## Permissions posture

- Default **read-only**; write scoped to the specific target (Confluence project space; the two repos;
  the deploy namespace). Destructive/apply actions (Terraform apply) require an explicit human confirm.
- The anchor-service's own runtime **S2S auth is not an MCP** — it mirrors the EMS `X-Api-Key`/`X-Scope`
  header pattern `[code: ems/internal/base/handler/base.go]`.

## Brief wishlist → availability (deltas + substitutes)

| Brief-requested MCP | Status | Substitute |
|---|---|---|
| Atlassian, Git, Filesystem, Browser, Playwright | ✅ available (Atlassian, Playwright/chrome-devtools; Git via Bash; filesystem via native tools) | — |
| PostgreSQL MCP | ✗ not available (repos use MySQL anyway) | Bash MySQL client + serena |
| Docker MCP | ✗ | Bash `docker`/compose + `fabric-operations` skill |
| Kubernetes MCP | ✗ | Bash `kubectl` + **Terraform MCP** + `fabric-operations` |
| Mermaid MCP | ✗ (not needed) | Mermaid is fenced markdown; figma/Miro for visual |
| Sequential-Thinking MCP | ✗ | extended thinking / `superpowers:brainstorming` |
| Memory MCP | ✗ (no dedicated server) | **serena memories** + `.remember/` + lumen index |
| GitHub MCP | ✗ (org is **Bitbucket**) | Bash `git` + Bitbucket REST + `pull-request` skill |

**Bottom line:** ~60% of the brief's list is satisfied by better-fitting servers already present; the
genuine gaps (Postgres/Docker/K8s/sequential-thinking/memory) are all covered by CLI-via-Bash + the
Fabric operations skill + serena memory. **No new MCP server is required** to build the prototype.

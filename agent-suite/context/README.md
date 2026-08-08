# Context Knowledge Base — index & agent access matrix

> ⚠️ **Reconciliation in progress (2026-08-06).** Several docs below (`BLOCKCHAIN-DATA-MODEL`,
> `BLOCKCHAIN-INTEGRATION`, `DSRM` §4, and the Fabric-topology assumptions baked into the STUB docs)
> are being superseded per the ratifying PRD. See the AUTHORITY NOTICE at the top of
> `_bmad-output/planning-artifacts/project-context.md` before trusting any fact in this KB about
> tenancy, commitment scheme, ingest path, or erasure mechanism — read the PRD first.

The reusable knowledge base every agent consults (brief Deliverable 5). It is deliberately
**thin on Fabric internals**: those live in the 8-skill `fabric-skill-suite`, and the docs here
**cross-reference** them (per `AUTHORING-CONTRACT.md §7`) rather than copy. The single spine linking
everything is [`../11-execution/knowledge-graph.md`](../11-execution/knowledge-graph.md).

**Citation classes:** `[docs: …]` = pinned Fabric 2.5 corpus · `[code: …]` = real repo
(`talenta-core` / `employee-management-service`) · `[brief: agents-guide.md]` = the brief ·
`[ASSUMPTION] (gap G-##)` = a requirement-shaped specific not yet ratified (see
[`../11-execution/grounding-gaps.md`](../11-execution/grounding-gaps.md)).

## Inventory (35 docs)

### Fabric (10) — all **STUBS** → point to the owning `fabric-*` skill + corpus
`FABRIC-ARCHITECTURE` · `FABRIC-CHAINCODE` · `FABRIC-CHANNELS` · `FABRIC-MSP` · `FABRIC-CA` ·
`FABRIC-WORLD-STATE` · `FABRIC-PRIVATE-DATA` · `FABRIC-ORDERING` · `FABRIC-IDENTITY` · `FABRIC-POLICIES`

### Go (10) — repo-derived from `employee-management-service`
**AUTHORED:** `GO-ARCHITECTURE` · `GO-CONVENTIONS` · `GO-TESTING`
**STUB** (→ `coding-standards-backend` skill + notable repo specifics): `GO-APIS` · `GO-CONCURRENCY` ·
`GO-DATABASE` · `GO-DEPLOYMENT` · `GO-ERROR-HANDLING` · `GO-PERFORMANCE` · `GO-SECURITY`

### PHP (3) — repo-derived from `talenta-core` (brief gap we filled) — all **AUTHORED**
`PHP-ARCHITECTURE` · `PHP-CONVENTIONS` · `PHP-INTEGRATION`

### Security theory (7)
**AUTHORED:** `CRYPTOGRAPHY` · `ZERO-KNOWLEDGE-PROOF` · `PRIVACY-BY-DESIGN` · `SECURITY-BY-DESIGN` ·
`OWASP-ASVS` · `OWASP-API`
**STUB** (→ `fabric-security-review` + `security-review`; full model in G6): `THREAT-MODELING`

### Blockchain-HRIS (2) — the new bridge — **AUTHORED**
`BLOCKCHAIN-DATA-MODEL` · `BLOCKCHAIN-INTEGRATION`

### Methodology / meta (3)
**AUTHORED:** `DSRM`
**STUB:** `ADR` (→ `05-adr/` + `architecture-design` ADR mode) · `SYSTEM-DIAGRAM` (→ produced in G5 into `00-architecture/diagrams/`)

## Agent → context access matrix

The brief requires each agent reference **only** the docs relevant to its role. `●` = primary,
`○` = secondary/as-needed.

| Context doc | architect | fabric-architect | fabric-engineer | backend-engineer | security-architect | reviewer | qa | sre | pm | dsrm-researcher |
|---|---|---|---|---|---|---|---|---|---|---|
| FABRIC-ARCHITECTURE | ● | ● | ○ | ○ | ○ | ○ |  |  |  | ○ |
| FABRIC-CHAINCODE |  | ○ | ● | ○ |  | ○ | ○ |  |  |  |
| FABRIC-CHANNELS | ○ | ● |  |  | ○ |  |  | ○ |  |  |
| FABRIC-MSP |  | ● | ○ |  | ● | ○ |  | ○ |  |  |
| FABRIC-CA |  | ○ |  |  | ● |  |  | ● |  |  |
| FABRIC-WORLD-STATE | ○ | ● | ● |  |  |  | ○ | ○ |  |  |
| FABRIC-PRIVATE-DATA | ○ | ● | ● |  | ● | ○ | ○ |  |  |  |
| FABRIC-ORDERING |  | ● |  |  |  |  |  | ○ |  |  |
| FABRIC-IDENTITY | ○ | ● | ○ |  | ● | ○ |  |  |  |  |
| FABRIC-POLICIES | ○ | ● | ● |  | ● | ● |  |  |  |  |
| GO-ARCHITECTURE | ○ |  | ● | ● |  | ○ | ○ | ○ |  |  |
| GO-CONVENTIONS |  |  | ● | ● |  | ● | ○ |  |  |  |
| GO-TESTING |  |  | ○ | ● |  | ○ | ● |  |  |  |
| GO-APIS…GO-SECURITY (stubs) |  |  | ○ | ● | ○ | ○ | ○ | ○ |  |  |
| PHP-ARCHITECTURE | ○ |  | ○ | ● |  | ○ |  |  |  |  |
| PHP-CONVENTIONS |  |  |  | ● |  | ● |  |  |  |  |
| PHP-INTEGRATION | ● | ○ | ● | ● | ○ | ○ | ○ |  |  |  |
| CRYPTOGRAPHY | ○ | ○ | ○ |  | ● | ○ |  | ○ |  |  |
| ZERO-KNOWLEDGE-PROOF | ○ | ○ |  |  | ● |  |  |  |  | ● |
| PRIVACY-BY-DESIGN | ○ |  |  | ○ | ● | ○ | ○ |  | ○ | ○ |
| SECURITY-BY-DESIGN | ○ | ○ | ○ | ○ | ● | ● | ○ | ● |  |  |
| THREAT-MODELING |  | ○ | ○ |  | ● | ● | ● |  |  |  |
| OWASP-ASVS |  |  | ○ | ○ | ● | ● | ● |  |  |  |
| OWASP-API |  | ○ | ○ | ● | ● | ● | ● |  |  |  |
| BLOCKCHAIN-DATA-MODEL | ● | ● | ● |  | ● | ○ | ○ |  |  | ○ |
| BLOCKCHAIN-INTEGRATION | ● | ● | ● | ● | ○ | ○ | ○ | ● |  |  |
| DSRM | ○ |  |  |  |  |  | ○ |  | ● | ● |
| ADR | ● | ○ |  |  | ○ | ○ |  |  | ○ | ○ |
| SYSTEM-DIAGRAM | ● | ● | ○ | ○ | ○ | ○ | ○ | ○ |  |  |

> Roles are defined in [`../01-agents/`](../01-agents/). The orchestrator (`hlf-orchestrator`
> Workflow) hands each dispatched agent only its primary (`●`) docs by default, adding `○` docs on demand.

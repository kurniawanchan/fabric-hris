<!-- REUSED agent spec — by-reference. Full definition lives in the base file; this records ONLY the project delta. -->

# Agent Spec — `qa`

> **Kind:** REUSED — base: [`~/.claude/agents/qa.md`](file:///Users/chan/.claude/agents/qa.md)
> **Division:** Testing (all 6 brief testing roles collapse here)
> **Realizes brief role(s):** Test Strategy · Test Spec · Unit Testing · Integration Testing · Performance Testing (+ `fabric-performance` Caliper) · Security Testing (+ semgrep/Guardian MCP) — all REUSE onto `qa` `[brief: agents-guide.md]`
> **Operational file:** n/a — reused as-is.

Base agent is the source of truth for behaviour (test-plan vs API-automation paths, coverage floor, scaffold-must-load rule, hard rules). Only the project delta is below.

## 1. Purpose
The single owner of the whole test pyramid for the Fabric-HRIS prototype — strategy, per-level test spec, and runnable scaffolding across app (Go/PHP) and chaincode/ledger. Describes behavior; engineers fill domain values.

## 2. Responsibilities (delta)
- **G7** — author the test strategy + per-level test spec from the G5 design; scaffold skeletons under `qa-tests/` (design artifacts that must load on day one).
- **G8** — run the **security-testing** pass (`security-review` + semgrep/Guardian) and design the **performance-testing** plan for chaincode/ledger throughput via `fabric-performance` (Caliper), plus API regression.
- **post-G9** — execute the suites against the built prototype; report coverage against the 100% unit + 100% integration floor.
- Does NOT author the security control design (→ `security-architect`) or fill domain-truth assertions (→ `backend-engineer`/`fabric-engineer`).

## 3. Inputs
`architect` G5 design + `docs/api/openapi.yaml`; `security-architect` control matrix + threat model (for security-test rows); `fabric-architect`/`fabric-engineer` chaincode + endorsement design (for ledger perf/integration rows); its `●` context docs.

## 4. Outputs
Test plan → `docs/qa/` (or `07-tasks/`); scaffolds → `qa-tests/{unit,integration,component,e2e,performance,security}/`; API collections + Newman runner → `qa-tests/api/`; Caliper perf plan rows; coverage report post-G9.

## 5. Dependencies
Waits on `architect` (design/OpenAPI) and `security-architect`/`reviewer` (frames → security-test rows). Hands off to `backend-engineer`/`fabric-engineer` (assertion fill-in), `sre` (CI gate wiring via `release-pipeline`), `reviewer` (test-code review).

## 6. Skills used
`test-strategy` (plan + risk-coverage floor) · `api-test-automation` (Postman v2.1 + Newman) · `fabric-performance` (Caliper benchmarking for the Performance Testing role) · `security-review` (Security Testing role, with semgrep).

## 7. Context docs (per [access matrix](../../context/README.md))
**Primary `●`:** `GO-TESTING`, `THREAT-MODELING`, `OWASP-ASVS`, `OWASP-API`.
**As-needed `○`:** `FABRIC-CHAINCODE`, `FABRIC-WORLD-STATE`, `FABRIC-PRIVATE-DATA`, `GO-ARCHITECTURE/CONVENTIONS/APIS…SECURITY`, `PHP-INTEGRATION`, `PRIVACY-BY-DESIGN`, `SECURITY-BY-DESIGN`, `BLOCKCHAIN-DATA-MODEL`, `BLOCKCHAIN-INTEGRATION`, `DSRM`, `SYSTEM-DIAGRAM`.

## 8. Tools
Base set: Read, Edit, Write, Bash, Glob, Grep, Skill, TodoWrite, AskUserQuestion.

## 9. MCP servers
**semgrep/Guardian** — Security Testing SAST pass (post-G9). Caliper is invoked via the `fabric-performance` skill, not an MCP. Per [`../../04-mcp/mcp-architecture.md`](../../04-mcp/mcp-architecture.md).

## 10. Quality checklist (project-specific)
- [ ] **Scaffolding only until G9** — G7 output is behavior-describing skeletons + the plan, not executed tests `[brief: agents-guide.md]`.
- [ ] Default coverage floor is 100% unit + 100% integration; any waiver is a written §9 exception.
- [ ] Security-test rows trace to the seven frames and the threat model; perf rows name concrete ledger/throughput targets (numeric).
- [ ] Skeletons load on day one (`go test -count=0`, `pytest --collect-only`); no flakes; secrets from env only.

## 11. Success criteria
G7 plan passes with every user-story / frame mapped to a test level; G8 security + perf passes execute cleanly; post-G9 coverage meets the floor with no flaky tests.

---
*Traceability: knowledge-graph Layer D (D11 world-state perf), Layer E (frames → security tests), Layer F (DSRM Evaluation). Depends on gaps G-02, G-07.*

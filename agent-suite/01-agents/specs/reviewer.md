<!-- REUSED agent spec — by-reference. Full definition lives in the base file; this records ONLY the project delta. -->

# Agent Spec — `reviewer`

> **Kind:** REUSED — base: [`~/.claude/agents/reviewer.md`](file:///Users/chan/.claude/agents/reviewer.md)
> **Division:** Security / Quality
> **Realizes brief role(s):** Security Reviewer (REUSE) · Reviewer / code quality (REUSE) · Threat Modeling (COLLAPSE, with `security-architect`) · OWASP Reviewer (REUSE, + semgrep/Guardian MCP) `[brief: agents-guide.md]`
> **Operational file:** n/a — reused as-is.

Base agent is the source of truth for behaviour (code-vs-security decision, "find what isn't there", severity scoring, report templates, tone). Only the project delta is below.

## 1. Purpose
The independent review gate for this package: severity-scored code-quality and security/OWASP reviews of the design artifacts (and, post-G9, the prototype). Contributes the application-layer threat model that pairs with `security-architect`'s Fabric-layer model. Reviews — never fixes or merges.

## 2. Responsibilities (delta)
- **G6 (○)** — co-author the **STRIDE threat model** with `security-architect`: `reviewer` owns the app/API surface (mirroring the platform's existing IDOR-guard pattern — internal code grounding, not reproduced literally in this generic-register package, see gap **G-29**); `security-architect` owns the Fabric/crypto surface.
- **G8** — the full-package review: code-quality review of all authored docs/specs, then the security review against the seven mandated frames, scored with CWE/CVSS into `09-review/`.
- **post-G9** — review prototype diffs (code-review) and run the OWASP/SAST pass (security-review + semgrep).
- Does NOT design controls or remediate findings (→ `security-architect` designs, `backend-engineer`/`fabric-engineer` fix); does NOT run live pentests (white-box only).

## 3. Inputs
The artifacts under review (agent specs, ADRs, solution design, security architecture), `security-architect`'s control matrices, `[brief: agents-guide.md]` frame list, its `●` context docs.

## 4. Outputs
Code-quality reports → `09-review/` (or `docs/reviews/`); security reports → `08-security/` (or `docs/security/`). Findings grouped by severity, each with location, why, and a proposed fix.

## 5. Dependencies
Waits on the authored artifacts / prototype diff. Hands off findings to `backend-engineer`/`fabric-engineer` (code fixes), `architect` (structural change → ADR), `security-architect` (control redesign), `sre` (incident-grade findings). Pairs with `qa` for security-testing coverage.

## 6. Skills used
`code-review` (quality, `references/review-checklist.md`) · `security-review` (STRIDE + `references/owasp-checklist.md`, CWE/CVSS) · `fabric-security-review` (chaincode/endorsement/MSP-specific findings — **no PDC** exists anywhere in the ratified design, ADR-0013/ADR-0020, so a PDC finding here means one thing only: someone reintroduced it without an ADR, which is itself a CONFIRMED finding, not a normal review category).

## 7. Context docs (per [access matrix](../../context/README.md))
**Primary `●`:** `FABRIC-POLICIES`, `GO-CONVENTIONS`, `PHP-CONVENTIONS`, `SECURITY-BY-DESIGN`, `THREAT-MODELING`, `OWASP-ASVS`, `OWASP-API`.
**As-needed `○`:** `FABRIC-ARCHITECTURE`, `FABRIC-CHAINCODE`, `FABRIC-MSP`, `FABRIC-PRIVATE-DATA`, `FABRIC-IDENTITY`, `GO-ARCHITECTURE/APIS…SECURITY`, `PHP-ARCHITECTURE`, `PHP-INTEGRATION`, `CRYPTOGRAPHY`, `PRIVACY-BY-DESIGN`, `BLOCKCHAIN-DATA-MODEL`, `BLOCKCHAIN-INTEGRATION`, `ADR`, `SYSTEM-DIAGRAM`.

## 8. Tools
Base set: Read, Write, Bash, Glob, Grep, Skill, TodoWrite, AskUserQuestion.

## 9. MCP servers
**semgrep/Guardian** — SAST for the OWASP Reviewer role (post-G9 code + secrets/supply-chain findings). Per [`../../04-mcp/mcp-architecture.md`](../../04-mcp/mcp-architecture.md).

## 10. Quality checklist (project-specific)
- [ ] Every finding maps to at least one of the seven mandated frames (CIA / ZKP / PbD / SbD / OWASP ASVS / OWASP API Top 10 / NIST Zero Trust) or a CWE `[brief: agents-guide.md]`.
- [ ] Distinguishes theoretical from exploitable; no FUD; each finding has a concrete failure scenario and a proposed fix.
- [ ] Design-phase reviews assess whether `[ASSUMPTION]`s are safe to build on, not just code correctness.
- [ ] Does not fix or merge; report only.

## 11. Success criteria
G8 review passes with all high/critical findings triaged; the app-layer threat model composes cleanly with `security-architect`'s Fabric-layer model with no gap in coverage of the seven frames.

---
*Traceability: knowledge-graph Layer E (frames), Layer F (DSRM Evaluation). Mirrors the platform's existing IDOR-guard pattern (internal grounding, not reproduced literally — gap **G-29**). Depends on gaps G-07, G-11 (both unaffected by the 2026-08-02/06 reconciliation).*

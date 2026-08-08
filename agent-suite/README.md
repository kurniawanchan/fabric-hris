# Hyperledger Fabric Agent Suite — Architecture Package

A **multi-agent engineering organization** for designing, building, reviewing, testing, securing, and
documenting a blockchain-based **Personal & Employee Data Security** prototype for an HRIS SaaS on
Hyperledger Fabric 2.5 — integrating `talenta-core` (PHP/Yii2) and `employee-management-service` (Go).

This package realizes the brief in [`../agents-guide.md`](../agents-guide.md). It **reuses and
orchestrates** the existing 8-skill Fabric suite in [`../fabric-skill-suite/`](../fabric-skill-suite/)
rather than rebuilding it. It is a **design + operational-tooling** package: it produces the agent
organization and its knowledge base, but writes **no prototype application code** until the final
human approval gate (G9).

> **Methodology:** Design Science Research Methodology (DSRM).
> **Security frames:** ZKP · CIA Triad · Privacy-by-Design · Security-by-Design · OWASP ASVS · OWASP API Top 10 · NIST Zero Trust.

## Reading order

| # | Path | Deliverable |
|---|------|-------------|
| 1 | [`00-architecture/`](00-architecture/) | Multi-agent architecture, orchestration & collaboration, quality gates + Mermaid diagrams |
| 2 | [`01-agents/`](01-agents/) | Agent catalog (all ~40 brief roles → mapping) + per-agent specs |
| 3 | [`context/`](context/) | Reusable knowledge base (Fabric/Go/PHP/blockchain/security/DSRM) |
| 4 | [`skills/`](skills/) | Skill catalog + the 3 genuinely-new skills |
| 5 | [`04-mcp/`](04-mcp/) | MCP architecture (servers → agents → permissions → workflows) |
| 6 | [`05-adr/`](05-adr/) | Architecture Decision Records + index |
| 7 | [`06-roadmap/`](06-roadmap/) | DSRM-phased prototype development roadmap |
| 8 | [`07-repo-structure/`](07-repo-structure/) | Scalable repository layout |
| 9 | [`08-security/`](08-security/) | Security architecture + control matrices |
| 10 | [`09-review/`](09-review/) | Review workflow + verification logs |
| 11 | [`10-risk/`](10-risk/) | Risk register |
| 12 | [`11-execution/`](11-execution/) | Orchestrator execution strategy, knowledge graph, grounding gaps, **`sources/` (drop requirements here)** |
| — | [`prompts/`](prompts/) | The reusable `hlf-orchestrator` Workflow definition |

## Status board

> ⚠️ **This board describes two generations of the same package.** G0–G9 below (dated 2026-07-13)
> certified a design that a 2026-08-02 ratifying decision ("the thesis wins" —
> `_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-02/prd.md`) superseded on most
> load-bearing points. **G8's PASS and G9's APPROVAL are VOID** (PRD §11.1) — they certified an
> object that no longer exists. A reconciliation pass on 2026-08-06 (`rencana-rekonsiliasi.md`,
> 4 parallel specialist agents + direct edits) brought the design current; a **second table** below
> tracks that reconciliation's own status. Read the reconciliation table first — it is the one that
> is current.

Legend: ⬜ not started · 🟡 in progress · ✅ done (gate passed) · ⏸ blocked

| Gate | Phase | Status (2026-07-13) | Notes |
|------|-------|--------|-------|
| G0 | Grounding & discovery | ✅ | proceed-with-assumptions mode; `knowledge-graph.md` + 24 grounding-gaps (G-23/G-24 added at G5) + seeded risk register; repos+corpus grounding verified |
| G1 | Context knowledge base | ✅ | 35 docs + README/access-matrix; FABRIC stubs point to skills; 0 unresolved `[VERIFY]`; 47 `[ASSUMPTION]` tags tracked to gaps |
| G2 | New skills | ✅ | skill-catalog (28 reuse/compose, 3 new); 3 new skills authored, de-duplicated, installed into `.claude/skills/` |
| G3 | Architecture + agent specs + agents | ✅ | 3 arch docs + 5 diagrams + agent-catalog (40→12) + 12 specs + 4 new agents installed; xref linter: context/MCP all resolve |
| G4 | ADRs | ✅ | 10 ADRs (0001–0010) + index + gap→ADR reopen map; all Accepted, alternatives recorded, gap-linked, no contradictions |
| G5 | Solution design | ✅ | network + data-model + API contracts + integration + C4; confidentiality invariant verified (0 PII on-chain); +2 gaps (G-23/G-24) |
| G6 | Security architecture | ✅ | STRIDE (15 threats) + CIA/PbD/SbD + Zero-Trust + OWASP ASVS/API matrices; every control traces to threat+mechanism (verified PASS); T1/G-18 + G-23 flagged open |
| G7 | Tasks + test strategy | ✅ | test-strategy (control→test matrix: 15/15 STRIDE + 6/6 ASVS + 6/6 API; ST-1 P0) + backlog (35 build items, 5 blocking pre-build); prototype risks R-06–R-13 appended |
| G8 | Full package review | ❌ **VOID** | *(was ✅)* 4-dimension adversarial sweep certified a design (event-based anchoring, salted per-event commitments, two-org, single-channel+PDC) that no longer exists. Report at `09-review/verification/g8-review-report.md`, marked superseded, not deleted. |
| G9 | Roadmap, risk, MCP, repo, README + **approval** | ❌ **VOID** | *(was ✅, approved 2026-07-13)* G9 approves an **assumption set**; 6+ members of that set are now known false. The approval's object no longer exists. Re-approval requires the G8a/G8b split (S-1) and a fresh sponsor review against the current design. |

### Reconciliation status (current — 2026-08-06)

| Wave | Scope | Status |
|------|-------|--------|
| 0 — Mekanika | ADR status vocabulary, AUTHORING-CONTRACT path fix, skill symlinks, Bash permissions | ✅ Done (2026-08-03/06) |
| 1 — Pengaman | Authority notice, dead-backlog markers, sources/README repoint | ✅ Done (2026-08-06) |
| 2 — Tulang belakang | `grounding-gaps.md` rewritten; `knowledge-graph.md` and the xref linter script still **not done** | 🟡 Partial |
| 3 — 10+1 ADR | ADR-0011…0021 (11 new/amending ADRs; central index synced) | ✅ Done (2026-08-06) |
| 4 — Dokumen desain | data-model, network-design, integration-design, api-contracts, security-architecture, control-matrices, test-strategy, risk-register — all rewritten; `implementation-backlog.md` full rewrite and `repository-structure.md` still **not done** | 🟡 Partial |
| 5 — Agen & konteks | Agent/skill roster redirect, `context/` 37-doc sweep, `project-context.md` full rewrite, this status board | 🟡 In progress |
| 6 — Tesis & gate | `g8-review-report-r2.md` ✅; **S-1…S-5 all decided 2026-08-06** (`quality-gates-and-approval.md` §5.5) — **G9 re-approved, G8b build authorized**. Still open: thesis edits (needs the human author — `.docx`, not editable by an agent), Caliper benchmark run (no longer blocked on S-1 — now a G8b build item, `QA-5`) | 🟢 **Build authorized** — thesis edits + actual G8b work remain |

**Full detail, every item, every owner:**
[`rencana-rekonsiliasi.md`](../_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-02/rencana-rekonsiliasi.md).

## Future expansion & long-term maintainability

> ⚠️ Two of the four items below are **no longer future expansion — they are the current ratified
> design** as of 2026-08-06 (ADR-0012, ADR-0013). Struck through, not deleted, so the reasoning that
> once deferred them is preserved.

**Expansion (each gated by a tracked gap / ADR — revisit only what's affected):**
- ~~**Adversarial trust** — add an external org … to upgrade the honest-but-curious two-vendor-org
  model to genuine multi-party (G-04 → ADR-0003).~~ **Done.** ADR-0012: three orgs, with an
  independently-operated enterprise-client org — this is now the baseline, not an expansion path.
- ~~**Hard multi-tenancy** — channel-per-tenant when isolation/residency demands it (G-03 → ADR-0004).~~
  **Done.** ADR-0013: channel-per-tenant is now the baseline.
- **Minimal-disclosure** — build the deferred ZKP/Idemix path if a user story needs it (G-07 → ADR-0008;
  the `zkp-designer` skill already runs the necessity gate). *(Still future work, unaffected by the
  2026-08-06 reconciliation.)*
- **Rich audit queries** — switch LevelDB→CouchDB if key-addressed access stops sufficing (G-08 → ADR-0007).
  *(Still future work.)*
- **Production hardening** — HSM-backed keys, per-org peer HA (ADR-0009/ADR-0019, G-08). *(3-node Raft
  is now the baseline per ADR-0012, not future work — remove that clause from any future edit of this line.)*
- **`employeeKey_i` accidental-loss recovery** *(new item, 2026-08-06)* — an audited, dual-control
  escrow path was considered (ADR-0021 Option C) and deferred as future work, not built now; covers
  the same accidental-loss class already accepted for `KEY_EMPLOYEE` (PRD §12 OQ-5).
- **More anchored sections/events** — the section set (`PERSONAL`/`EMPLOYMENT`/`EDUCATION`/
  `ADDITIONAL`/`PAYROLL`) is fixed by the ratified scope (PRD §4); adding a sixth section is a
  data-model + chaincode-lifecycle change, not a config change — scope it against PRD §4 first.

**Maintainability (built into the package):**
- The **gap→ADR reopen map** ([`05-adr/README.md`](05-adr/)) means real requirements re-open only the
  affected decisions, not the whole design.
- The **knowledge-graph** is the single consistency spine; the **cross-reference linter** (G3, re-run at
  G8) should run as a CI check so the reference graph never rots.
- New context docs/skills follow the **AUTHORING-CONTRACT**; the Fabric corpus is re-snapshotted per the
  suite's maintenance playbook.
- Every assumption is tracked (24 gaps) and every deferred item carries its reopening trigger — nothing
  is silently dropped.

## Conventions

- New context docs & skills obey [`../fabric-skill-suite/docs/AUTHORING-CONTRACT.md`](../fabric-skill-suite/docs/AUTHORING-CONTRACT.md)
  — `[docs: <path>#<section>]` citations to the pinned corpus, `[ASSUMPTION]`/`[VERIFY]` tagging, behavioral-rules block.
- Fabric facts are grounded in the pinned corpus at
  [`../fabric-skill-suite/corpus/fabric-docs-2.5/`](../fabric-skill-suite/corpus/) (`hyperledger/fabric@release-2.5`).
- **No duplication** of Fabric-suite content — context docs cite and point, they do not copy.
- Operational assets (4 new agents, 3 new skills) install into the project `.claude/`; their source of truth lives here.

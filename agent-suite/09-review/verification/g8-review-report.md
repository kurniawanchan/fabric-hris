> ⚠️ **SUPERSEDED 2026-08-06.** This report's **PASS** verdict certified a design that a
> 2026-08-02/03 ratifying decision ("the thesis wins") and a 2026-08-06 reconciliation pass have
> since replaced on nearly every load-bearing point: event-based anchoring → section-based;
> per-event salted commitment over `changedFieldValues` → per-section `DataHash` +
> `employeeKey_i`-keyed identifiers; two vendor orgs + single-channel + PDC → three orgs (one
> independently client-operated) + channel-per-tenant, no PDC; Kafka + standalone anchor-service →
> in-band recording. **This PASS no longer authorizes anything.** See
> [`g8-review-report-r2.md`](g8-review-report-r2.md) for the current status. This file is kept
> in place, unedited below this notice, as the historical record of what G8 originally certified.

# G8 — Full-Package Review Report (SUPERSEDED — see notice above)

Adversarial review of the whole package by four fresh-context dimension reviewers
(consistency / traceability / security / completeness), per
[`../review-workflow.md`](../review-workflow.md). Findings consolidated with dispositions below.

> **Reviewer note.** The *consistency* reviewer no-op'd (0-tool-call context echo — the failure mode
> documented in review-workflow.md §5). Its scope was covered by the *traceability* reviewer
> (gap-integrity), the *security* reviewer (contradiction detection), and the orchestrator's direct
> terminology reconciliation. Residual consistency risk: LOW.

## Outcome

**PASS.** All HIGH and MED findings are **fixed**; the two remaining LOW items are **deferred with
rationale** (cosmetic, non-load-bearing). No `high` CONFIRMED finding remains open. The design-only
boundary held throughout (no prototype code).

## Findings & dispositions

| # | Sev | Finding | Disposition |
|---|-----|---------|-------------|
| S-1 | **HIGH** | **Commitment-input incompatibility** — writer hashed the `{from,to}` delta (integration-design, backlog AS-3, test U-3) but data-model + verify hash *new values only*, so verify could never reproduce the digest → tamper-evidence non-functional. | **FIXED.** Reconciled all docs to the authoritative data-model §4: `SHA-256(salt ‖ JCS(changedFieldValues))` — new values only; `from`-state proven by the `prevCommitment` hash-chain; verify recomputes from *current* values + salt. Edited integration-design §4(4-5), test-strategy U-3, backlog AS-3, grounding-gaps G-24. |
| T-1 | **HIGH** | agent-catalog mislabels the code-reviewer/code-simplifier decision as "gap G-23" (colliding with the real G-23 read-back) and falsely claims "G-01…G-22 registered". | **FIXED.** Reframed as a resolved packaging decision (built-in review types / skill-backed hats — no new agent files; not a grounding gap). Removed the G-23 misuse; corrected register range to G-01…G-24. |
| T-2 / C-1 | **HIGH** | `prompts/` empty — the `hlf-orchestrator` Workflow definition was referenced as a done G3 deliverable but did not exist. | **FIXED.** Authored [`../../prompts/hlf-orchestrator.md`](../../prompts/hlf-orchestrator.md) — the Workflow contract, gate→DSRM map, and a representative skeleton. |
| S-3 | **MED** | `pseudonymKey` de-anonymization threat unmodelled — key compromise + enumerable `tbl_user.id` retroactively re-links the immutable ledger. | **FIXED.** Added STRIDE **T16** (security-architecture §2 + §7), test **ST-12**, risk **R-14**, and control-matrices coverage (16/16). |
| S-2 | **MED** | G-24 (canonicalization) declared "ratified/closed" in security docs but still OPEN in grounding-gaps — inconsistent status. | **FIXED.** Unified language: *scheme* selected (RFC 8785 JCS over new values); gap **stays OPEN** as build precondition PB-3, pinned by test U-4. Edited security-architecture T5, control-matrices §4, grounding-gaps G-24. |
| S-5 | **MED** | T9 (field-name metadata) named in the accepted set but had no risk-register row (asymmetric). | **FIXED.** Added risk **R-15** (accepted, prototype; ST-7 guard). |
| S-4 | LOW | ST-1 confidentiality-invariant test scanned world state + PDC but not chaincode event payloads (also on-chain/immutable). | **FIXED.** ST-1 extended to scan world state + tx args + **event payloads** + PDC. |
| S-6 / T-3(part) | LOW | risk-register internal 22-vs-24 open-gap count. | **FIXED.** R-01 → 24 (matches grounding-gaps + R-13). README G0 row → 24. |
| DEP-3 | MED | backlog CI/CD task linked no ADR/G5 doc. | **FIXED.** Added ADR-0006 + NET-DESIGN §6 + `release-pipeline` skill links. |
| T-4 | LOW | review-workflow link to this report dangled. | **FIXED.** This report now exists. |
| C-2 | LOW | `09-review/verification/` was empty (referenced). | **FIXED.** This report populates it. |

## Deferred (LOW, with rationale)

- **D-1 — terminology `pseudonymKey` vs `ref-pepper`.** Reconciled in the authoritative docs (data-model,
  integration-design prose) to `pseudonymKey`. Two *diagram labels* (`integration-flow.mmd`,
  `SYSTEM-DIAGRAM.md`) may still read `ref-pepper`. **Deferred:** cosmetic diagram-label drift; the
  authoritative data model and prose are consistent; sweep at re-diagram time.
- **D-2 — stale "G-01…G-22 / 22 gaps" phrasing** in a few agent specs + `orchestration-and-collaboration.md`.
  **Deferred:** `grounding-gaps.md` is authoritative at 24; the README, risk register, and catalog (the
  reader-facing surfaces) are corrected; the internal spec mentions are low-visibility cosmetic drift.

## What the review confirmed (clean)

- **Confidentiality invariant holds** end-to-end (0 PII plaintext on-chain) across data-model, API,
  integration, network — the load-bearing property.
- **Reference graph resolves:** every agent-spec skill/context/MCP, every ADR gap+context link, every
  solution→ADR trace, all 8 `.mmd` diagrams, and the control→test matrix (now 16/16 STRIDE + 6/6 ASVS +
  6/6 API) resolve to real artifacts.
- **No overclaim:** the honest-but-curious trust limitation is stated in 4 places; no trustlessness claim.
- **No no-op artifacts on disk:** completeness grep for stray system-reminder echoes → none (the
  census-and-heal discipline worked).
- **Accepted risks explicit:** T9, T12, T16, honest-but-curious collusion — all carried in the risk
  register / security §7, none silently dropped.

## Gate result

G8 **passes**. The package proceeds to G9 (roadmap, risk, MCP, repo-layout, README + **human approval**).
The OPEN preconditions surfaced here and earlier — **G-18** (Kong), **G-23** (read-back scope),
**G-24** (canonicalization test-pin), **G-01/G-05** (objective/events) — are carried into G9 as the
build-blocking gate, not silently closed.

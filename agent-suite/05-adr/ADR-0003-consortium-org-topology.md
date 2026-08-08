# ADR-0003: Two-org consortium topology for the prototype

- **Status:** **Superseded by ADR-0012**
- **Date:** 2026-07-13
- **Deciders:** fabric-architect
- **Rests on assumption(s):** G-04 (consortium composition is not fixed by any requirement doc)
- **Superseded-by:** ADR-0012
- **Ratifying authority / date (of supersession):** PRD `prd-fabric-hris-2026-08-02/prd.md` §5.1,
  §11.3 — human decision, Chandra Kurniawan, 2026-08-02/03. The two-org, both-vendor-operated topology
  is replaced by a three-org topology in which the client org is genuinely client-operated — see
  ADR-0012 for the decision and for why this ADR's own stated limitation ("honest-but-curious… not
  adversarial multi-party") made it insufficient to carry the ratified objective (PRD §2.1).

## Context
Tamper-evidence is only meaningful if an **independent** party endorses writes — a ledger a single
operator can rewrite unilaterally proves nothing. But the real consortium composition (who the member
organizations are) is unknown until HLF-6/Confluence arrive. The prototype needs the *minimum* org
diversity that makes the integrity claim real. Knowledge-graph Layer A (actors), capabilities **D6**
(MSP identity) and **D7** (channels). `[docs: msp.rst]`, `[brief: agents-guide.md]` (CIA-Integrity).

## Decision
The prototype consortium is **two vendor-operated Fabric organizations**:
- **HR org** — owns the HRIS write path (the anchor-service submits on its behalf).
- **Audit org** — an independent endorser/verifier that co-signs every anchor write.

Anchor-write endorsement policy is **AND('HROrg.peer', 'AuditOrg.peer')**, so no single org can write
an anchor alone. Both orgs are operated by the SaaS vendor for the prototype.

## Alternatives considered
| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. Two orgs (HR + Audit)** | Minimal viable endorsement diversity; `AND` policy gives a real integrity signal; low ops | Both vendor-operated → separation is organizational, not adversarial | **Chosen** |
| B. Four orgs (HR/Audit/Employee/Regulator) | True multi-party governance; strongest trust | Regulator/employee org onboarding is a real governance effort, not needed to *demonstrate* tamper-evidence | Deferred (future expansion) |
| C. Single org | Simplest | No endorsement diversity — one org can rewrite unilaterally, defeating the entire goal | Rejected |

## Consequences
- **Positive:** minimal operational footprint; the independent Audit endorsement produces a genuine
  integrity signal for the DSRM demonstration.
- **Trade-off:** because both orgs are vendor-operated, the trust model is *honest-but-curious internal
  separation of duties*, not adversarial multi-party. **This must be stated as a limitation in the G6
  threat model** — do not overclaim decentralization.
- **Follow-up:** append a prototype risk row (vendor operates both orgs) at G6; a production consortium
  would add external orgs (revisit via G-04).

## Related
- Supersedes / superseded by: none.
- Relates to: **ADR-0004** (channel strategy), **ADR-0005** (identity/MSP), **ADR-0001** (what is anchored).
- Knowledge-graph: Layer A/D6/D7. Grounding gap: **G-04**. Context: `context/FABRIC-MSP.md`, `context/FABRIC-CHANNELS.md`.

<!--
Immutable once Accepted. Supersede with a new ADR; do not edit in place.
-->

# ADR-0008: Defer ZKP/Idemix to an evaluation-phase exploration, not the core prototype

- **Status:** **Accepted** (the decision is to *defer*)
- **Date:** 2026-07-13
- **Deciders:** security-architect
- **Rests on assumption(s):** G-07 (no known user story is confirmed to require anonymous credentials / minimal disclosure; necessity is an [ASSUMPTION]). Grounded technically in `[docs: idemix.rst#current-limitations]`.

## Context
The mandated ZKP frame (E2) `[brief: agents-guide.md]` raises the question of whether the prototype should build zero-knowledge, minimal-disclosure credentials. Fabric 2.5 ships exactly one ZKP mechanism — **Idemix** (D10). The `zkp-designer` skill exists precisely to answer *whether* before *how*, via a **necessity gate that defaults to NO** and only passes when a concrete story needs (1) identity anonymity, (2) value-hiding predicate disclosure, and (3) unlinkability across presentations (see `skills/zkp-designer/SKILL.md` Step 0; theory in context `ZERO-KNOWLEDGE-PROOF.md`).

No such story is confirmed (G-07 / A-KG-7). The plausible HRIS candidates — "prove active-employee to an external portal", "prove salary is in band X" — are documented but **not expressible in Idemix 2.5**: Idemix carries a fixed four-attribute set of which only **`ou`** and **`role`** are ever revealed to chaincode, **no custom attributes** (so "salary band"/"active-status" cannot be encoded), **no revocation** (a terminated employee's credential cannot be cleanly revoked), Idemix orgs **cannot endorse**, one Idemix MSP per channel means the **anonymity set is a single org's members**, and the user SDK is **Java** while the assumed anchor service is Go (G-10) `[docs: idemix.rst#current-limitations]` (context `ZERO-KNOWLEDGE-PROOF.md` §4). Meanwhile the confirmed prototype goal (G-01) — tamper-evident audit of PII changes — is already served by **salted commitments (D3)**, which prove existence/match but are *not* a zero-knowledge predicate proof. This is a design/scope decision only — no prototype code.

## Decision
We will **defer ZKP/Idemix to the DSRM evaluation-phase exploration (phase 5 / gate G8)** and **not** build it into the core prototype. The core uses salted commitments (D3) for audit and plain X.509 + ABAC (ADR-0005) for authorization. Any future ZKP work must first **pass the `zkp-designer` necessity gate** against a ratified user story (closing G-07); the future `zkp-designer` skill (roadmap G2) is the vehicle for that evaluation.

## Alternatives considered
| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. Defer Idemix to evaluation phase; ship salted commitments (D3) + X.509 ABAC now** | Matches the confirmed audit goal (G-01); no heavyweight new MSP type; keeps the option open behind the necessity gate | ZKP demo not in the core prototype | **Chosen** |
| B. Build Idemix selective-disclosure into the core prototype | Would demonstrate the ZKP frame directly | **Heavyweight**, and Idemix 2.5 limits block the candidate predicates: `ou`/`role` only (no value-band/custom attrs), **no revocation**, **cannot endorse**, one Idemix MSP per channel (anonymity set = one org), Java-only user SDK vs Go anchor (G-10); **no user story confirms the need** (G-07) | Rejected because the mechanism cannot express the HRIS predicates and no requirement justifies the cost |
| C. Present salted commitments as if they were ZKP | Cheap; already in-repo (`bankhasher`) | A commitment proves *existence/match*, **not** a predicate in zero knowledge — would misrepresent the privacy property | Rejected because it conflates two distinct guarantees |
| D. Drop ZKP from the roadmap entirely | Simplest | The minimal-disclosure use cases are plausible future needs; deleting the option is premature | Rejected because deferring (not removing) preserves the option at low cost |

## Consequences
- **Positive:** Prototype scope stays proportional to confirmed requirements; no new MSP type, no Java SDK dependency, no false claims about revocation/endorsement; the ZKP option is preserved behind an explicit gate.
- **Negative / trade-off:** The ZKP frame (E2) is demonstrated only via commitments (D3) in the core, not via true selective disclosure; if a real minimal-disclosure story lands later it will need a purpose-built external ZK range-proof system (out of the Fabric corpus), because Idemix 2.5 will still not express value-band predicates.
- **Follow-ups:** Revisit when G-07 closes with a real user story; run the `zkp-designer` necessity gate at that point. Track "value-band predicate needs external ZK, not Idemix" as a design risk in `10-risk/risk-register.md`.

## Related
- Relates to: ADR-0005 (identity/role mapping — why X.509 attributes, not Idemix `ou`/`role`, carry HRIS roles); the no-plaintext-PII-on-chain decision (G-02) and the audit-overlay decision (G-09) authored separately.
- Knowledge-graph: Layer D — **D10** (Idemix ZKP), with **D3** (salted commitments) as the lighter-weight path actually used; frame **E2** (ZKP), supporting **E3** (Privacy-by-Design minimal disclosure). Grounding gap(s): **G-07** (is ZKP required), with **G-04** (bounds the anonymity set) and **G-10** (Go vs Java SDK). Context doc(s): `context/ZERO-KNOWLEDGE-PROOF.md` §4, `context/FABRIC-IDENTITY.md`, `context/CRYPTOGRAPHY.md` §5. Skill: `skills/zkp-designer/SKILL.md` (necessity gate, Step 0). Roadmap: future `zkp-designer` skill build (G2).

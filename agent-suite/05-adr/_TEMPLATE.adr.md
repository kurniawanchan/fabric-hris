<!--
TEMPLATE — copy to agent-suite/05-adr/ADR-<NNNN>-<slug>.md (zero-padded; next number = 1 + max existing).
Conventions (aligned with the org `architect` agent + architecture-design skill):
- Status is one of THREE terminal values: **Accepted**, **Rejected**, or **Superseded by ADR-NNNN**.
  Nothing else — no `Proposed`, `Deprecated`, or other free-form value. Derived from the decision (or
  from the supersession event), never asserted freely.
- Every decision names at least one considered-and-rejected alternative, with the reason.
- Constraints are numeric where they exist (latency, throughput, sizes) — numbers, not adjectives.
- ADRs are IMMUTABLE once Accepted — the Context/Decision/Alternatives/Consequences sections are
  never rewritten. If the situation changes, write a NEW ADR that supersedes this one (record it under
  Related in the new ADR), THEN flip this ADR's own **Status** field to `Superseded by ADR-<NNNN>` and
  fill its `Superseded-by` field below — this is the one field a supersession is allowed to touch; the
  body stays as a historical record of what was decided and why.
- Because the project runs in proceed-with-assumptions mode, EVERY ADR must state which grounding
  assumption (gap ID G-01..G-24) it rests on, so it can be revisited if the real requirement differs.
-->

# ADR-<NNNN>: <short imperative title, describe the system not the toolchain>

- **Status:** Proposed → **Accepted** | **Rejected** | **Superseded by ADR-<NNNN>**  <!-- final value only -->
- **Date:** <YYYY-MM-DD>
- **Deciders:** <agent role(s) — e.g. fabric-architect, security-architect>
- **Rests on assumption(s):** <gap IDs, e.g. G-02, G-09 — or "none; grounded in [code:]/[docs:]">
- **Supersedes:** <ADR-<NNNN> — or "none">
- **Superseded-by:** <ADR-<NNNN>, set only when this ADR's Status becomes Superseded — or "none">
- **Ratifying authority / date:** <who/what ratified the supersession, e.g. "PRD prd-fabric-hris-2026-08-02 §5.2, human decision 2026-08-02" — omit if not superseded>

## Context
The forces at play: the requirement (or the [ASSUMPTION] standing in for it), the constraints, the
relevant knowledge-graph capabilities (D#) and prior ADRs. Cite `[docs:]` / `[code:]` / `[brief:]`.

## Decision
"We will …" — one clear, active statement of the choice made.

## Alternatives considered
| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. <chosen>** | … | … | **Chosen** |
| B. <alt> | … | … | Rejected because … |
| C. <alt> | … | … | Rejected because … |

## Consequences
- **Positive:** what gets easier / safer.
- **Negative / trade-off:** what gets harder or is now constrained.
- **Follow-ups:** new risks (append to `10-risk/risk-register.md`), tasks, or dependent ADRs.

## Related
- Supersedes / superseded by / relates to: ADR-<NNNN>. (Mirror in the Supersedes/Superseded-by
  header fields above — this section gives the narrative reason, the header gives the machine fact.)
- Knowledge-graph: <layer / capability IDs>. Grounding gap(s): <G-##>. Context doc(s): `context/<...>.md`.

<!--
Immutable once Accepted. Supersede with a new ADR; do not edit in place.
-->

# ADR-0021: `employeeKey_i` is an independently-random per-employee secret — drop the `pseudonymKey` master-key hierarchy

- **Status:** **Accepted**
- **Date:** 2026-08-06
- **Deciders:** Chandra Kurniawan (human decision, explicit choice among three presented options)
- **Rests on assumption(s):** None — closes a disclosed residual risk (T16b) by explicit human
  choice, not an `[ASSUMPTION]`. Grounded in `[prd: §5.2]` (post-2026-08-06 correction),
  `08-security/security-architecture.md` T16a/T16b, `ADR-0019` §Consequences.
- **Supersedes:** ADR-0011 (identifier-derivation clause only — the `DataHash`/salt construction is
  **unchanged and reaffirmed**), ADR-0019 (Domain C description only — Domains A/B/B′ are
  **unchanged and reaffirmed**)
- **Superseded-by:** none
- **Ratifying authority / date:** Human decision, Chandra Kurniawan, 2026-08-06, choosing explicitly
  among three options presented (keep the hierarchy / independent-random `employeeKey_i` / hybrid
  audited-escrow) after `security-architect`'s ADR-0019 disclosed T16b.

## Why this is a supersession of two ADRs, and why most of both bodies survives unchanged

Per `_TEMPLATE.adr.md`, an ADR body is immutable; a changed decision is recorded as a new ADR, and
only the `Status` field of the old ADR may then change. This ADR changes exactly **one derivation
step** — how `employeeKey_i` comes into existence — which happens to be defined inside two other
ADRs' bodies (ADR-0011's Decision §2, ADR-0019's Domain C row). Both ADRs are therefore marked
`Superseded by ADR-0021`, even though:
- ADR-0011's `DataHash = SHA-256(salt ‖ JCS(section))` construction, its chain-linkage rule, its
  canonicalization requirement, and its rejection of Options B/C/D/E are **all unaffected** and
  **fully reaffirmed** by this ADR.
- ADR-0019's Domains A, B, and B′ (MSP/TLS, application AES, `KEY_EMPLOYEE`) — their custody model,
  destroy-ability rules, and cross-domain prohibition — are **all unaffected** and **fully
  reaffirmed**. Only Domain C's internal structure changes.

## Context

ADR-0019 (`security-architect`, same day) disclosed, in its own Consequences section, a residual
property of the two-tier `pseudonymKey → employeeKey_i` hierarchy (ADR-0011, corrected 2026-08-03
from an even weaker single-shared-key design): because `employeeKey_i = HMAC-SHA256(pseudonymKey,
tenantId ‖ employeeInternalId)` is a **pure, stateless function** of `pseudonymKey` plus an
**enumerable** `employeeInternalId`, compromising `pseudonymKey` does not just de-anonymize current
employees — it lets an attacker **re-derive every `employeeKey_i` that tenant has ever had**,
including ones deliberately destroyed to satisfy a completed right-to-erasure request. `08-security/
security-architecture.md` recorded this as **T16b**, rated "accepted with mitigation, elevated
severity."

Three options were presented to the ratifying human (not authored by any specialist agent, since
this is a value trade-off between two failure modes, not a technical correctness question a design
agent can resolve unilaterally):

1. **Keep the hierarchy** — accept T16b as documented (HSM-only custody, never backed up outside
   the HSM boundary) in exchange for a real operational benefit: an accidentally-lost
   `employeeKey_i` is recoverable by recomputing it from `pseudonymKey`.
2. **Independently-random `employeeKey_i`, no `pseudonymKey`** — treat it exactly like the `DataHash`
   salt already is: a CSPRNG value generated once, stored off-chain, with **no master secret that
   can regenerate it**. This closes T16b completely — there is no key whose compromise reverses
   history — but an accidentally-lost `employeeKey_i` becomes **indistinguishable from, and as
   permanent as, an intentional erasure**. This is not a new class of risk for this package: it is
   the **same** failure mode already accepted for `KEY_EMPLOYEE` (`prd: §12 OQ-5`, the thesis's own
   admitted limitation, BAB IV 4.5).
3. **Hybrid — independently-random `employeeKey_i` plus an audited, dual-control backup escrow** —
   closes T16b **and** the accidental-loss risk, at the highest design/build cost of the three (a
   second, access-controlled key-management surface with mandatory audit logging on every recovery
   use, rather than a silent recompute).

**Decision: Option 2.**

## Decision

`employeeKey_i` is generated **once per employee**, at first write, by a CSPRNG — **not derived from
any master key**. `pseudonymKey` is **removed from the design entirely**; there is no tenant-scoped
root secret for identifier pseudonymization.

```
employeeKey_i  ← CSPRNG(≥ 128 bits), generated once, at the employee's first EmployeeProfileRecord write
EmployeeID     = HMAC-SHA256(employeeKey_i, "id")
UpdatedBy      = HMAC-SHA256(employeeKey_i, "actor" ‖ user_id)
```

Storage and destruction are otherwise **unchanged** from ADR-0011/ADR-0019: `employeeKey_i` lives
off-chain, one record per employee, in the same class of secure store as the per-record `DataHash`
salts (Domain C, ADR-0019) — it is now **structurally identical** to a salt: independently random,
generated once, destroyed on that employee's erasure. Erasure (ADR-0015) is **unchanged in
procedure** — destroy the employee's `employeeKey_i` — but the **consequence of that destruction**
is now unconditional: there is no key anywhere that can regenerate it.

**Numeric constraint (set by this ADR):** `employeeKey_i` entropy: **≥ 128 bits**, CSPRNG, generated
**once** per employee (not per record — unlike the `DataHash` salt, which is per-record; `employeeKey_i`
must stay stable across a given employee's records so `EmployeeID` remains searchable, per ADR-0011's
Option E rejection, which still applies and is unaffected by this ADR).

## Alternatives considered

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. Keep the `pseudonymKey`→`employeeKey_i` hierarchy** | Accidental loss of one `employeeKey_i` is recoverable | T16b: `pseudonymKey` compromise silently reverses every completed erasure for that tenant, forever — the single highest-severity residual risk in the whole key-management model (ADR-0019 §Consequences) | Rejected — the ratifying human judged retroactive mass-reversal of a right-to-erasure guarantee to be a worse failure mode than a recoverability convenience, for a system whose entire thesis claim is data-integrity non-reversibility |
| **B. Independently-random `employeeKey_i`, no master key (chosen)** | T16b eliminated entirely — no key's compromise can reverse history; structurally identical to the already-accepted, already-battle-tested `DataHash` salt pattern (ADR-0011) | Accidental loss of a single `employeeKey_i` is now permanent and indistinguishable from an intentional erasure | **Chosen** — this is the same accepted-limitation shape as `KEY_EMPLOYEE` (OQ-5), not a new one; the thesis already carries this limitation for documents and states it openly (BAB IV 4.5) |
| C. Hybrid — independently-random key + audited dual-control escrow | Closes both T16b and the accidental-loss risk | Highest cost: a second key-management surface, mandatory dual control, and audit logging on every recovery use — build-time complexity disproportionate to a prototype's stakes | Deferred to future work — record as a **Saran untuk Penelitian Lanjutan**-class recommendation (production-hardening path), not built now |

## Consequences

- **Positive:** T16b is **closed**, not merely mitigated — there is no longer any single secret whose
  compromise can silently undo a completed erasure across an entire tenant's history. The fix costs
  nothing against any ratified predicate: **P1–P4 are unaffected** (identifiers are still
  deterministic per employee, still searchable, still chainable — only *how* the key originates
  changed, not what it guarantees operationally). The off-chain key/salt store's operational model
  is now **uniform**: every secret in Domain C (content salts and identifier keys alike) is
  independently random, generated once, destroyed on erasure, never derived from anything else.
- **Negative / trade-off:** accidental loss of one employee's `employeeKey_i` (backup corruption,
  operator error) is now **unrecoverable and indistinguishable from an intentional erasure** — this
  is the **same** limitation the package already accepts for `KEY_EMPLOYEE` (OQ-5); it is not a new
  class of exposure, but it is a second instance of it, and both should be covered by the **same**
  backup-integrity discipline when that is eventually designed (still open, still future work, per
  OQ-5).
- **Follow-ups:**
  - `08-security/security-architecture.md`: **T16b is now CLOSED** (not "accepted with mitigation")
    — the threat's precondition (a key whose compromise reverses history) no longer exists. **T16a**
    is unaffected — a single `employeeKey_i` compromise still only affects that one employee, which
    was always the intended, contained blast radius. Re-verify and update the STRIDE table's T16b row
    accordingly (this ADR does not itself edit that document — flagged for `security-architect` or
    the next doc-sync pass).
  - `10-risk/risk-register.md`: the `pseudonymKey`-custody risk row (if one was added per ADR-0019
    §Follow-ups item "b" — `pseudonymKey` custody/compromise as the single highest-impact key event)
    should be **retired**, and a note added that `employeeKey_i` accidental-loss risk is now
    symmetric with the existing `KEY_EMPLOYEE`-loss risk (OQ-5) rather than a distinct concern.
  - The Option C hybrid (audited escrow) is recorded here as a **named future-work path**, not
    silently dropped — worth a line in the thesis's "Saran untuk Penelitian Lanjutan" alongside the
    already-listed key-recovery mechanism for `KEY_EMPLOYEE` (BAB V 5.2.1), since both are instances
    of the same underlying gap.

## Related

- Supersedes (identifier-derivation clause only): **ADR-0011**, **ADR-0019** — both otherwise
  reaffirmed in full; see the explanation above.
- Relates to: **ADR-0015** (erasure procedure — unchanged in mechanics, strengthened in guarantee);
  **PRD §12 OQ-5** (the parallel `KEY_EMPLOYEE` accidental-loss limitation this ADR's trade-off now
  mirrors); `08-security/security-architecture.md` T16a/T16b (T16b to be marked closed).
- Knowledge-graph: Layer D capability **D3** (salt/predictable-value defense — now uniformly applied
  across Domain C). No gap ID currently tracks the accidental-loss backup-integrity procedure for
  Domain C secrets generally (both `KEY_EMPLOYEE` and `employeeKey_i`) — recommend `dsrm-researcher`
  open one gap covering both, rather than treating them as two separate open questions.

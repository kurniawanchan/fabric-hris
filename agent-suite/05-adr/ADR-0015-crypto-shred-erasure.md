<!--
Immutable once Accepted. Supersede with a new ADR; do not edit in place.
-->

# ADR-0015: Right-to-erasure via off-chain crypto-shred (delete field + destroy `KEY_EMPLOYEE` + destroy salt + destroy `employeeKey_i`) — no PDC, no `PurgePrivateData`, no `blockToLive`

- **Status:** **Accepted**
- **Date:** 2026-08-06
- **Deciders:** security-architect
- **Rests on assumption(s):** **G-06** (regulatory basis and exact retention window remain open — see below, this ADR does **not** close G-06). No dedicated gap ID yet exists for the two PDP-Law findings this ADR inherits (DPIA obligation, Pasal 34; automatic time/purpose-based retention duty, Pasal 42) — cited directly to `[prd: §9.3]` pending a gap row from `dsrm-researcher`.
- **Supersedes:** ADR-0010
- **Superseded-by:** none
- **Ratifying authority / date:** PRD `prd-fabric-hris-2026-08-02/prd.md` §5.2, §7 Kelompok E (FR-25..29), §9.2 row 4 — human decision (Chandra Kurniawan), ratified 2026-08-02/03.

## Context

ADR-0010 satisfied right-to-erasure with Fabric-native `PurgePrivateData`/`blockToLive` on a Private
Data Collection (PDC), on the premise that a two-org, single-channel+PDC topology might place an
actual field *value* in Fabric-managed private state at some point (ADR-0010 §Context, ADR-0004).
That premise no longer holds. The ratified topology is **three organizations, one Fabric channel
per tenant, and no PDC anywhere in the design** `[prd: §5.1]` — the confidentiality invariant
(INV-1) keeps **zero** PII plaintext or reversible derivative in Fabric-managed state, on-chain or
in a collection, full stop. There is therefore nothing inside Fabric's control for
`PurgeprivateData`/`blockToLive` to purge; the erasure problem is now, and always was under §5.2,
**entirely an off-chain key-management problem**.

The construction that must be un-done on erasure is §5.2's own (PRD §5.2, ratified 2026-08-02,
corrected 2026-08-03 for the two-tier identity hierarchy):

- `DataHash = SHA-256(salt ‖ JSON kanonik section)` — one **per-record** salt (≥128-bit CSPRNG),
  stored **off-chain only** `[prd: §5.2 FR-3]`.
- `EmployeeID = HMAC-SHA256(employeeKey_i, "id")`, `UpdatedBy = HMAC-SHA256(employeeKey_i, "actor" ‖ user_id)`
  — where `employeeKey_i = HMAC-SHA256(pseudonymKey, tenantId ‖ employeeInternalId)` is a **per-employee**
  key derived from, but never identical to, the tenant-scoped master `pseudonymKey` `[prd: §5.2 FR-5]`.
- `FR-30..32`: every supporting document is encrypted with a **per-employee** `KEY_EMPLOYEE` before
  upload to the IPFS Private Cluster; only its CID is anchored `[prd: §7 Kelompok F]`.

FR-25..29 (§7 Kelompok E) state the required erasure behaviour directly; this ADR formalizes it
and adjudicates the one point FR-28 leaves implicit — **which stored artifact must die, and which
must not**, given the two-tier identity hierarchy. The hierarchy exists precisely so that erasure of
one employee cannot damage another (PRD §5.2 note on the 2026-08-03 correction): if `EmployeeID` and
`UpdatedBy` were derived directly from `pseudonymKey`, destroying `pseudonymKey` to erase one
employee would break every other employee sharing that key. **`employeeKey_i` is therefore the
erasure-critical secret for on-chain identifiers; `pseudonymKey` is not, and must never be treated as
one.**

INV-3 (ledger is append-only) and FR-27 forbid the other lever entirely: erasure must **never**
attempt to alter, overwrite, or delete on-chain state. What survives an erasure is by design an
**orphan** — a `DataHash`/`EmployeeID`/`UpdatedBy` triple nobody can any longer open or attribute
`[prd: §7 FR-27]`.

## Decision

We will satisfy the right-to-erasure requirement (FR-25..29) with a **four-part, wholly off-chain
crypto-shred**, and with nothing else:

1. **Delete the operational-database field values** for the employee's five profile sections in the
   authoritative HRIS store (FR-25). This is an ordinary row/column delete, not a cryptographic
   operation.
2. **Destroy the employee's `KEY_EMPLOYEE`** (FR-26). Every supporting document this employee ever
   uploaded to the IPFS Private Cluster becomes permanently undecipherable. This is the *entire*
   erasure mechanism for IPFS content — **not** removal of the IPFS objects themselves (§11.4 of the
   PRD: `unpin` is not `delete`; any node that ever fetched a block may still hold it). The compliance
   claim must be stated as "encrypted and the only key is destroyed," never as "the file is gone."
3. **Destroy the per-record salt(s)** for every `DataHash` this employee's five sections ever
   produced (FR-28). Without the salt, `DataHash` cannot be recomputed or dictionary-attacked.
4. **Destroy the employee's `employeeKey_i`** (FR-28). Without it, `EmployeeID` and `UpdatedBy` for
   every record this employee ever anchored can no longer be recomputed or matched by anyone who does
   **not** also hold `pseudonymKey` (see Residual risk, T16b in `08-security/security-architecture.md`).

**What is explicitly never touched:**
- **On-chain state** (FR-27, INV-3) — no rewrite, no delete, no tombstone transaction. The surviving
  `DataHash`/`EmployeeID`/`UpdatedBy`/CID values become permanent, unattributable orphans on that
  tenant's channel.
- **`pseudonymKey`** (the tenant-scoped master) — never destroyed to satisfy one employee's erasure.
  It may only be **rotated at tenant scope** (ADR-0019 §Domain C), and only as an operational
  response to a *suspected compromise* of the master itself, never as a routine erasure step.

**Evidence of execution (FR-29).** Because on-chain state cannot record the erasure (that would
itself contradict FR-27 and could act as a new re-identification signal for *which* pseudonymous
chain went orphan), the system records a **signed, off-chain deletion certificate** — request ID,
timestamp, scope of destroyed artifacts (DB fields / `KEY_EMPLOYEE` / salts / `employeeKey_i`),
authorizing actor — in the operational audit store, satisfying the notification/proof duty
(Pasal 44(1)/45, PRD §9.2 row 4) without any ledger operation.

## Alternatives considered

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. Four-part off-chain crypto-shred: delete DB fields + destroy `KEY_EMPLOYEE` + destroy salt + destroy `employeeKey_i` (chosen)** | No ledger operation of any kind; consistent with append-only INV-3; blast radius is exactly one employee thanks to the `employeeKey_i` tier; needs no Fabric capability beyond what already exists | Per-employee key/salt stores become erasure-critical, high-cardinality assets whose own backup/replication must be provably destructive, not soft-delete; on-chain orphans accumulate forever with no reclaim path (feeds the Pasal 42 retention gap, tracked in the risk register, not resolved by this ADR) | **Chosen** |
| B. Fabric-native `PurgePrivateData`/`blockToLive` on a PDC (ADR-0010's mechanism) | Corpus-documented v2.5 primitive; ledger-native audit trail of the purge itself | **Requires a PDC to exist.** The ratified topology has none — introducing one *solely* to have something to purge would (a) create a new PII-bearing Fabric surface purely to delete it, violating data minimization, and (b) contradict the ratified three-org/channel-per-tenant/no-PDC topology `[prd: §5.1]` | Rejected — inapplicable by construction, not inferior |
| C. Rewrite or delete the on-chain `EmployeeProfileRecord` state directly | Would make "erasure" visibly complete on the ledger | No such primitive exists for committed Fabric blocks; violates INV-3 (append-only); if simulated via a new "tombstone" transaction it would still leave the tombstone itself as permanent evidence, and worse, could retroactively break every other employee's `PrevHash` chain on that channel if implemented carelessly | Rejected |
| D. Revoke application-level access to the row/documents but keep DB field, `KEY_EMPLOYEE`, salt, and `employeeKey_i` intact | Reversible if the erasure request is disputed; cheapest to implement | Not erasure — Pasal 43/44/45 require the data to actually become unusable/unrecoverable for identification, not merely access-gated; fails FR-25/26/28 on their own terms | Rejected |

## Consequences

- **Positive:** the erasure story requires zero Fabric ledger operations and is fully consistent with
  the append-only invariant (INV-3); the two-tier identity hierarchy (ADR-0019 §Domain C) means one
  employee's erasure is provably contained to that employee — it does not touch any other employee's
  verifiability, unlike a design that hashed identifiers directly from `pseudonymKey`.
- **Negative / trade-off:**
  - **Unbounded on-chain growth with no reclaim.** Every erased employee still leaves a permanent,
    orphaned `DataHash`/`EmployeeID`/`UpdatedBy`/CID set on that tenant's channel forever — this is
    the concrete mechanism behind the Pasal 42 "cannot be claimed" retention finding (PRD §9.3 row
    11): crypto-shred is triggered by a *request* (Pasal 43(1)(c)), never by the passage of time or
    the end of a purpose, which is what Pasal 42 actually requires. **This ADR does not resolve that
    gap — it is recorded as an open compliance risk, not swept under this decision.**
  - **The per-employee key/salt stores are now erasure-critical, high-cardinality assets.** Unlike
    ADR-0009's original two domains (rotated, never destroyed-per-subject), `KEY_EMPLOYEE` and
    `employeeKey_i` must support **provable, irreversible, per-record destruction** — a backup or
    replica that quietly retains a "deleted" key defeats the entire mechanism silently. This is a
    materially different operational discipline than key *rotation* and must be designed as such at
    build time (fabric-engineer / ops).
  - **The IPFS erasure claim is a key-destruction claim, not an object-removal claim.** `unpin` does
    not delete a block from every node that ever fetched it `[prd: §11.4]`; the legal defensibility of
    calling this "penghapusan"/"pemusnahan" rests on an argument that has **not** been tested against
    the statute (PRD §9.4: the law defines "memusnahkan" (Pasal 44) by result — *"tidak lagi dapat
    digunakan untuk mengidentifikasi"* — which fits a destroyed-key argument better than "menghapus"
    (Pasal 43), which the law never defines).
  - **`pseudonymKey` compromise silently reverses every erasure ever completed on that tenant.**
    Because `employeeKey_i = HMAC-SHA256(pseudonymKey, tenantId ‖ employeeInternalId)` is a pure,
    stateless function, destroying the *stored copy* of `employeeKey_i` does not make it unknowable
    to anyone who separately holds `pseudonymKey` and can guess/enumerate `employeeInternalId` (shown
    empirically enumerable in seconds, PRD §12 OQ-1/errata E-1). This is not a defect this ADR can fix
    — it is the price of the erasure-containment property in Consequences/Positive above — but it must
    be disclosed, not hidden. See `08-security/security-architecture.md` T16b and ADR-0019 §Domain C
    custody requirement (HSM, never leaves protected custody).
- **Follow-ups:**
  - Append to `10-risk/risk-register.md`: (a) per-employee key/salt-store backup retention defeating
    crypto-shred, (b) `pseudonymKey` compromise reversing completed erasures tenant-wide, (c) DPIA
    obligation (Pasal 34) unmet, (d) unbounded ledger retention (Pasal 42) unmet — the latter two as a
    **legal-compliance** risk class, not a technical one.
  - The exact regulatory basis and any numeric retention window remain **open** (`G-06`); this ADR's
    mechanism is deliberately law-agnostic (it holds under any plausible basis) but does not itself
    close G-06.
  - `OQ-5` (PRD §12) — accidental `KEY_EMPLOYEE` loss is operationally indistinguishable from an
    unrequested erasure — remains an open operational-continuity risk this ADR inherits but does not
    solve.
  - The concrete per-employee key/salt store design (schema, backup policy, destruction procedure) is
    a build-time task for `fabric-engineer`, informed by the custody table in ADR-0019.

## Related

- **Supersedes:** ADR-0010 (its `PurgePrivateData`/`blockToLive` mechanism is inapplicable by
  construction once no PDC exists in the ratified topology — see Alternatives, Option B).
- **Relates to:** ADR-0019 (adjudicates custody and destroy-ability of the four key domains this ADR
  destroys/preserves); ADR-0001/its pending successor (owned by `fabric-engineer`, not yet authored)
  for the exact `EmployeeProfileRecord` field layout this ADR's destruction targets; the pending IPFS
  Private Cluster topology ADR (owned by `fabric-architect`/`fabric-engineer`, not yet authored — this
  ADR only states the security requirement the IPFS tier must satisfy, not its topology).
- Knowledge-graph: Layer D — no PDC capability (**D1**) is invoked by this design; **D12** (ledger
  immutability, the reason erasure cannot touch chain state). Frame **Privacy-by-Design (E3)**,
  **CIA-Confidentiality (E1)**. Traceability spine chain #4 (erasure) — re-derived here, not the
  PDC-based version in `knowledge-graph.md` (that document is owned by `dsrm-researcher`; this ADR's
  mechanism supersedes its erasure-chain description without editing it). Grounding gap(s): **G-06**
  (open); PRD §9.3 rows 10–11 (DPIA, retention) have no gap ID yet.

# ADR-0011: Per-section salted digest with two-tier keyed pseudonym identifiers

- **Status:** **Superseded by ADR-0021** (identifier-derivation clause only — the `DataHash`/salt
  construction and everything else in this ADR is reaffirmed, not reversed)
- **Date:** 2026-08-06
- **Deciders:** fabric-engineer (formalization), security-architect (crypto-scheme authority)
- **Rests on assumption(s):** None for the digest/HMAC construction itself — ratified by
  `[prd: §5.2]`, human decision (Chandra Kurniawan), 2026-08-02/03. The concrete canonicalization
  **implementation** to pin remains open — **G-24 / PB-3** (unchanged by this ADR).
- **Supersedes:** ADR-0001
- **Superseded-by:** ADR-0021 (partial — identifier-derivation clause only)
- **Ratifying authority / date:** `prd-fabric-hris-2026-08-02/prd.md` §5.2 ("Konstruksi
  Kriptografis — diratifikasi"), human decision 2026-08-02, corrected 2026-08-03 (two-tier key
  hierarchy) — see `errata-tesis.md` **E-1** and `rencana-rekonsiliasi.md` Gelombang 3 #13.

## Context

ADR-0001 anchored a **per-event, bare-salted commitment over only the changed fields of one
event** (`SHA-256(salt ‖ canonical(changedFieldValues))`), with `employeeRef`/`actorRef` derived
from a single, tenant-scoped `pseudonymKey` used directly. Two rulings since then require a new
construction, not an amendment of the old one:

1. **The anchoring unit changed** (`[prd: §4]`, superseding the event surface `EVENT_*`): the unit
   is now a **profile section** — one of five values (`PERSONAL`, `EMPLOYMENT`, `EDUCATION`,
   `ADDITIONAL`, `PAYROLL`) — not a per-event delta of changed fields. A digest scheme keyed to
   "the fields that changed in one event" has no object to attach to once the object being
   fingerprinted is "the whole section, as of this write."
2. **The original identifier construction is empirically broken.** A prototype-package draft
   before ratification (and the thesis's own BAB IV 4.2.3, verbatim) computed
   `EmployeeID = SHA-256(internal_emp_id)` and `UpdatedBy = SHA-256(user_id)` with **no key at
   all**, over a small/enumerable input domain. Adversarial verification recovered a target
   `emp_id` from its digest in **0.0002 s**, and swept the entire 10⁷ internal-ID space in
   **4.4 s**, on a single CPU core running an interpreted SHA-256 implementation — three to four
   orders of magnitude before even considering optimized/GPU attacks (`errata-tesis.md` **E-1**).
   An unkeyed digest over low-entropy content is not a pseudonym; it is a reversible encoding, and
   every peer holding a ledger replica — including the enterprise-client org and the auditor org —
   could invert it.
3. **A single shared key for individual erasure is itself broken.** A corrected draft of §5.2
   (2026-08-02, before the 2026-08-03 fix) proposed keying `EmployeeID`/`UpdatedBy` directly off
   one tenant-wide `pseudonymKey` via HMAC. This closes the brute-force hole but opens a different
   one: crypto-shredding **one** employee's identifier requires destroying the key that produced
   it — and if that key is shared by every employee in the tenant, destroying it to satisfy one
   employee's erasure request silently breaks verifiability for **every other employee in that
   tenant**. This is the defect the two-tier hierarchy below exists to close (`[prd: §5.2]`,
   correction dated 2026-08-03).

Relevant grounding: content salting against low-entropy predictable data
`[docs: private-data-arch.rst#protecting-private-data-content]`; keyed-MAC pseudonymization as
the standard remedy for unkeyed-hash reversibility (`[prd: §5.2]`, `errata-tesis.md` **E-1**);
canonical serialization as a precondition for deterministic recompute (`[prd: §5.2]`, INV-6,
NFR-7 — RFC 8785 JCS, open build item **PB-3/G-24**).

## Decision

We will anchor, per profile-section write, a **`DataHash`** computed over the whole section and
**two HMAC-derived, per-employee pseudonymous identifiers**, built from a **two-tier key
hierarchy** rather than a single shared secret:

**1. Content digest — random salt, not a key (because content is non-deterministic by nature):**

```
DataHash = SHA-256( salt ‖ JCS(section) )
```

- `salt` — **≥ 128-bit** CSPRNG output, **new for every record** (not reused across versions or
  sections), stored **off-chain only**.
- `JCS(section)` — the RFC 8785 JSON Canonicalization Scheme serialization of the section's full
  field set (not a delta), one pinned implementation shared by every writer and every verifier
  (`[prd: §5.2]`, INV-6; canonicalization is declared a **versioned interface** — see ADR-0020).

**2. Identifiers — a two-tier keyed hierarchy, not a salt (because identifiers must be
deterministic to remain searchable and chainable):**

| Layer | Derivation | Scope | Stored | Destroyed when |
|---|---|---|---|---|
| `pseudonymKey` (parent) | — (root secret) | whole tenant | off-chain, tenant-scoped secret store | **never**, for an individual erasure |
| `employeeKey_i` (child) | `HMAC-SHA256(pseudonymKey, tenantId ‖ employeeInternalId)` | one employee | off-chain, per-employee | when **that** employee's erasure request is executed |

```
EmployeeID = HMAC-SHA256(employeeKey_i, "id")
UpdatedBy  = HMAC-SHA256(employeeKey_i, "actor" ‖ user_id)
```

`pseudonymKey` is **never** used directly to compute an on-chain field — only `employeeKey_i`
is. This is the load-bearing property: destroying one employee's `employeeKey_i` un-links that
employee's on-chain identifiers without touching any other employee's, because no other
employee's identifiers were ever derived from the same key.

**3. Chain linkage — per section, per employee, not per event and not whole-profile:**

```
PrevHash(record_n) = DataHash(record_n-1)   — same (EmployeeID, ProfileSection) only
```

Five sections per employee ⇒ **five independent chains** per employee, each advancing only when
its own section is written (`[prd: §4]`).

**Numeric constraints (set by this ADR):**

- Digest algorithm: **SHA-256** (256-bit output), identifier MAC: **HMAC-SHA256**.
- Per-record content salt: **≥ 128 bits**, CSPRNG, unique per record, off-chain only.
- Key-hierarchy depth: **2** (`pseudonymKey` → `employeeKey_i`); a bare, ungated third tier is out
  of scope — see Alternatives.
- Canonicalization: **1** pinned scheme (RFC 8785 JCS) per `canonicalizationVersion` tag (ADR-0020);
  the concrete library/version string is **not** pinned by this ADR — open dependency **PB-3/G-24**.
- Chains per employee: **5** (one per `ProfileSection` value), independently versioned.

## Alternatives considered

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. Two-tier HMAC hierarchy + per-record salted digest (chosen)** | Content digest is non-reversible against low-entropy sections (random salt); identifiers are deterministic (searchable, chainable) yet non-reversible without the key; per-employee erasure is scoped to exactly one employee's `employeeKey_i`, never the tenant-wide root | Off-chain custody of a two-level key hierarchy **and** a per-record salt store is now security-critical infrastructure; losing `employeeKey_i` for an employee is functionally an (intended) erasure, losing `pseudonymKey` is a tenant-wide incident | **Chosen** |
| B. Single shared `pseudonymKey` used directly for `EmployeeID`/`UpdatedBy` (the 2026-08-02 pre-correction draft) | Simpler — one key, one derivation step | **Rejected** — destroying the key to erase one employee breaks every other employee sharing it; the key cannot be destroyed for an individual without a blast radius of the whole tenant | Rejected |
| C. Unkeyed `SHA-256(internal_id)` / `SHA-256(user_id)` (thesis BAB IV 4.2.3, verbatim) | Trivial to implement; matches the literal thesis text | **Rejected** — empirically reversible in **0.0002 s** for a targeted lookup and **4.4 s** for a full 10⁷-space sweep on a single interpreted CPU core (`errata-tesis.md` E-1); not a pseudonym, a reversible encoding; every peer holding a ledger replica can invert it | Rejected |
| D. Bare per-event salted commitment over `changedFieldValues` only (ADR-0001, superseded) | Smaller hash input; matches an event-sourced mental model | **Rejected** — the anchoring unit is now the profile section, not a per-event delta (`[prd: §4]`); a "changed fields only" digest cannot represent "the section as of this write," which is what a verifier must recompute against; also inherited the single-key identifier flaw of Option B | Superseded by this ADR |
| E. Per-record salt for identifiers too (salt, not key, for `EmployeeID`/`UpdatedBy`) | Uniform treatment — one primitive, one mental model | **Rejected** — a random salt changes on every write, so the same employee would produce a different `EmployeeID` on every record; this breaks both the `(EmployeeID, ProfileSection)` lookup key (data-model.md §5) and the `PrevHash` chain, which both require a **stable** identifier. Identifiers need a **key** (deterministic), content needs a **salt** (non-deterministic) — one primitive cannot serve both | Rejected |

## Consequences

- **Positive:** the ledger's identifiers are deterministic enough to search and chain, yet
  non-reversible without off-chain key material held in a domain distinct from Fabric's own
  signing keys and from the document-encryption key (`[prd: §5.2]` three-domain rule — custody and
  exact domain count formalized separately, see Related); the content digest defeats dictionary
  attacks against low-entropy sections (bank account, NIK, salary band, education level) because
  the salt is random and per-record, never derived from the content itself; individual erasure
  (destroy one `employeeKey_i`) is now surgical — it does not require touching `pseudonymKey` or
  any other employee's derived key or salts.
- **Negative / trade-off:** the off-chain store now holds **two** classes of secret with different
  lifecycles — a long-lived, rarely-rotated `pseudonymKey` per tenant, and a large number of
  per-employee `employeeKey_i` values plus per-record salts — and losing either the wrong
  `employeeKey_i` (by mistake, not by a legitimate erasure) or the salt store is functionally
  identical to an unrequested erasure; canonicalization must stay byte-identical between every
  writer and every verifier or every recompute silently mismatches (ties to the still-open
  **PB-3/G-24**, not closed by this ADR).
- **Follow-ups:** the exact custody model, rotation policy, and domain count/boundaries for
  `pseudonymKey`/`employeeKey_i`/the salt store are **security-architect's** to formalize
  (tracked as ADR-0019 in `rencana-rekonsiliasi.md` Gelombang 3 #18 — not written by this ADR);
  the concrete canonicalization library/version string to satisfy `canonicalizationVersion` is a
  build precondition (**PB-3/G-24**); a risk-register row for "off-chain key/salt store as
  security-critical infrastructure" belongs to security-architect's `10-risk/risk-register.md`.

## Related

- Supersedes: **ADR-0001** (this ADR's Status flip is the only edit made to ADR-0001's body per
  the immutability rule in `_TEMPLATE.adr.md`).
- Relates to: **ADR-0020** (consumes `DataHash`/`EmployeeID`/`UpdatedBy`/`PrevHash` as chaincode
  arguments computed off-chain by this scheme; declares `canonicalizationVersion` a versioned
  interface); **ADR-0014** (this scheme's computation happens in-band, in the HRIS write path,
  not in a separate service); the pending **ADR-0019** (security-architect — key-domain custody).
- Knowledge-graph: Layer D capabilities **D2/D3** (digest, salting), Layer C (identity
  derivation); traceability spine chains #1 (anchor) and #2 (confidentiality). Grounding gap(s):
  **G-24/PB-3** (canonicalization pin, open). Context doc(s): `context/BLOCKCHAIN-DATA-MODEL.md`,
  `context/CRYPTOGRAPHY.md`. Ratifying source: `prd-fabric-hris-2026-08-02/prd.md` §5.2,
  `errata-tesis.md` E-1, `rencana-rekonsiliasi.md` Gelombang 3 #13.

<!--
Immutable once Accepted. Supersede with a new ADR; do not edit in place.
-->

# ADR-0019: Amend key-domain separation from two domains to four — MSP/TLS, application AES-at-rest, `KEY_EMPLOYEE` (IPFS documents), and the `pseudonymKey`→`employeeKey_i` pseudonymization hierarchy

- **Status:** **Superseded by ADR-0021** (Domain C's internal structure only — Domains A/B/B′ are
  reaffirmed, not reversed)
- **Date:** 2026-08-06
- **Deciders:** security-architect
- **Rests on assumption(s):** **G-15** (original — whether Fabric key management integrates with the
  existing application key-versioning; still open in the sense that the two are confirmed **not**
  integrated). No gap ID yet exists for the per-employee key/salt rotation-and-backup procedure this
  ADR's custody table requires; recommend `dsrm-researcher` open one. Grounded in `[prd: §5.2]`,
  `[prd: §7 FR-26, FR-28, FR-30]`, and `[docs: hsm.md]` (HSM/file-based-TLS split, reused unchanged
  from ADR-0009).
- **Supersedes:** ADR-0009
- **Superseded-by:** ADR-0021 (partial — Domain C only)
- **Ratifying authority / date:** PRD `prd-fabric-hris-2026-08-02/prd.md` §5.2 ("Tiga domain kunci
  yang tidak boleh pernah dicampur" + the 2026-08-03 two-tier identity correction) — human decision
  (Chandra Kurniawan), ratified 2026-08-02/03.

## Why this is a supersession, not a rewrite, and why that is the correct template status

ADR-0009's **decision is not reversed by anything below.** Its Domain A (Fabric MSP/HSM signing +
TLS keys) and Domain B (application AES-at-rest PII keys) are carried forward **unchanged**, with
the same custody model, the same rationale, and the same cross-domain prohibition. What changed is
that ADR-0009's body literally says "the **two** key domains" and enumerates exactly two — and that
sentence is no longer a complete statement of the model. §5.2 introduces **two more** key domains
with a lifecycle property ADR-0009 never had to address at all: **per-subject destroy-ability**
(crypto-shredding, ADR-0015), as opposed to domain-wide *rotation*. An ADR's body is immutable
(`_TEMPLATE.adr.md`); the only field a supersession may touch is `Status`. Because "the two key
domains" can no longer be edited in place to say "four," the only status the three-value vocabulary
offers that reflects reality is **`Superseded by ADR-0019`** — even though, read narratively, this
is an amendment that extends ADR-0009's principle rather than a decision that contradicts it. This
ADR restates ADR-0009's original two domains verbatim in §Decision below precisely so that nothing
it protected is lost in the supersession.

## Context

The knowledge-graph and ADR-0009 originally modelled exactly two cryptographic subsystems: Domain A
(Fabric node crypto — X.509 signing, TLS, HSM, cert lifecycle) and Domain B (application-level PII
crypto — the existing AES-256-CBC envelope encryption and PBKDF2 bank-field hashing already in the
repos, reused, not reinvented `[code: ems/pkg/db/encryption_plugins.go]`, `[code: ems/pkg/bankhasher/bankhasher.go]`).
ADR-0009's governing rule was: **no domain may be derived from, wrap, or be wrapped by another; a key
that anchors on-chain must never also decrypt PII.**

§5.2 of the ratified PRD introduces two more key purposes that do not fit either existing domain:

1. **`KEY_EMPLOYEE`** — a **per-employee** symmetric key that encrypts supporting documents before
   they are uploaded to the IPFS Private Cluster (FR-30). This is not the same purpose as the
   existing application AES-at-rest key (Domain B), which protects **operational-database columns**
   under a **tenant/version-wide**, rotated key — `KEY_EMPLOYEE` protects **document content** under
   a key that must be **destroyable per individual employee** (FR-26, ADR-0015 step 2).
2. **`pseudonymKey` → `employeeKey_i` (two-tier hierarchy)** — the tenant-scoped master
   `pseudonymKey` derives a per-employee `employeeKey_i = HMAC-SHA256(pseudonymKey, tenantId ‖ employeeInternalId)`,
   and only `employeeKey_i` is ever used to compute the on-chain `EmployeeID`/`UpdatedBy` pseudonyms
   (FR-5). This hierarchy exists **specifically** so that destroying one employee's identifier key
   (erasure) cannot damage any other employee's verifiability — a property neither of ADR-0009's
   original domains needed, because neither is destroyed per-subject.

Both new purposes carry a lifecycle requirement — **provable, irreversible, per-subject
destruction** — that is categorically different from *rotation* (which is all ADR-0009's two domains
ever needed). Conflating either new purpose with an existing domain, or with each other, would either
(a) force a destroy-per-subject requirement onto a key that dozens of other subjects share (breaking
them on erasure, exactly the defect the 2026-08-03 hierarchy correction fixed once already — PRD
§5.2 note), or (b) let a single compromised key open both on-chain identifiers **and** IPFS document
content, maximizing blast radius in violation of ADR-0009's own founding rule.

## Decision

We will govern **four** key domains, adjudicating custody and destroy-ability for each:

| Domain | Key material | Purpose | Custody | Destroy-ability |
|---|---|---|---|---|
| **A** *(from ADR-0009, unchanged)* | Fabric MSP signing keys (X.509) + TLS keys | Node/actor identity signing; transport | HSM via PKCS#11 for signing keys; **file-based** for TLS keys — a documented Fabric constraint, not a choice `[docs: hsm.md]` | **Never destroyed for erasure.** Rotated/revoked via CRL only (D15). One employee's erasure has zero effect on Domain A. |
| **B** *(from ADR-0009, unchanged)* | Application `PII_ENCRYPTION_KEY[_Vn]` (AES-256-CBC) | Keep operational-database PII columns unreadable at rest | App/ops-managed, versioned, tenant/global-wide `[code: ems/pkg/db/encryption_plugins.go]` | **Not employee-specific.** Rotated tenant/version-wide. Never used to satisfy an individual erasure request — that is a plain row/field delete (ADR-0015 step 1), not a key operation. |
| **B′** *(new)* | `KEY_EMPLOYEE` — per-employee symmetric key | Encrypt IPFS-hosted supporting documents before upload (FR-30) | Off-chain key-store, **one record per employee** | **MUST be destroyable per employee** (FR-26). Destruction *is* the erasure mechanism for IPFS content — encryption + key-destruction, never object-removal (`unpin ≠ delete`, PRD §11.4). |
| **C** *(new)* | `pseudonymKey` (tenant master) → `employeeKey_i` (per-employee, `HMAC-SHA256(pseudonymKey, tenantId‖employeeInternalId)`) + per-record salt store | Pseudonymize on-chain `EmployeeID`/`UpdatedBy`; defeat dictionary attacks on `DataHash` (FR-3, FR-5) | Off-chain, **hierarchical**: `pseudonymKey` is tenant-scoped (one per tenant, never shared across tenants); `employeeKey_i` and salts are per-employee/per-record | `employeeKey_i` and salts **MUST be destroyable per employee** (FR-28, ADR-0015 steps 3–4). **`pseudonymKey` MUST NEVER be destroyed to satisfy an individual erasure** — only **rotatable at tenant scope**, and only as a compromise response, never routinely (see Consequences: rotation does not retroactively protect already-derived `employeeKey_i` values). |

**Cross-domain rule (restated from ADR-0009, now spanning four domains, not two):** no domain's key
material may be derived from, wrap, or be wrapped by another domain's. In particular:
- `KEY_EMPLOYEE` (B′) must never be derivable from, or used to derive, `pseudonymKey`/`employeeKey_i`
  (C) — a compromised document key must not open the identifier pseudonymization, and vice versa.
- Neither B′ nor C may be derived from Domain A or B, and Domain A/B signing/at-rest keys must never
  be used to compute an on-chain pseudonym or decrypt an IPFS document.
- This is the same principle ADR-0009 stated as "never anchor on-chain with a key that also decrypts
  PII" — now enforced across all four pairwise boundaries, not one.

## Alternatives considered

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. Four separate domains, each with its own custody + destroy-ability adjudication (chosen)** | Erasure blast radius is provably scoped (one employee's IPFS documents and on-chain pseudonyms, nothing else); ADR-0009's original rationale is preserved intact for A/B and now applies uniformly to B′/C | Four key-management surfaces to provision, back up, and audit instead of two; B′ and C are **high-cardinality** (one record per employee, not per tenant/version) | **Chosen** |
| B. Fold `KEY_EMPLOYEE` into the existing application AES `PII_ENCRYPTION_KEY` domain (reuse Domain B) | One fewer key-management surface | `PII_ENCRYPTION_KEY` is versioned tenant/global-wide and rotated, never destroyed per-subject; forcing per-employee destroy-ability onto it means either destroying one employee's document key forces a version bump affecting every other employee sharing that version, or the versioning scheme must be split down to per-employee — collapsing two incompatible retention regimes onto one key | Rejected — lifecycle collision |
| C. Derive `employeeKey_i` directly from `KEY_EMPLOYEE` (merge domains B′ and C) | One fewer per-employee key to manage | `KEY_EMPLOYEE` protects IPFS **document content**; `employeeKey_i` protects **on-chain identifier pseudonymization** — a compromise of one must not open the other. Mirrors exactly the case ADR-0009 already rejected ("using the PII-decrypting key to sign anchors") | Rejected — maximizes blast radius, violates the founding rule this ADR restates |
| D. Derive `EmployeeID`/`UpdatedBy` directly from `pseudonymKey`, no `employeeKey_i` tier (the original 2026-08-02 draft of §5.2, corrected 2026-08-03) | One fewer key tier | Sharing `pseudonymKey` directly across every employee at a tenant means destroying it to erase one employee breaks the verifiability of every other employee sharing it — the exact defect PRD §5.2's 2026-08-03 correction fixed | Rejected — already superseded within the PRD itself; this ADR formalizes the corrected hierarchy into key-domain governance, it does not re-open the question |

## Consequences

- **Positive:** erasure blast radius is now provably bounded — destroying `employeeKey_i` +
  `KEY_EMPLOYEE` + the employee's salts affects exactly that employee's on-chain pseudonyms and IPFS
  documents, nothing else; ADR-0009's original compromise-containment rationale is preserved for
  Domains A/B and now extends symmetrically to B′/C.
- **Negative / trade-off:**
  - **Four key-management surfaces instead of two.** Each needs its own provisioning, monitoring,
    rotation, and (for B′/C) destruction runbook; operators must not "simplify" by merging any pair.
  - **B′ and C are high-cardinality, erasure-critical stores.** Unlike A/B (rotated on a schedule),
    a B′/C record's disappearance must be *provable and irreversible* — a backup or replica silently
    retaining a "deleted" `employeeKey_i`/`KEY_EMPLOYEE`/salt defeats ADR-0015 without any visible
    symptom. This is a materially different operational discipline than key rotation.
  - **`pseudonymKey` rotation is a tenant-wide event with an important limitation.** Rotating the
    tenant master stops an attacker from deriving `employeeKey_i` for employees **not yet** derived
    under the old key, but it does **not** retroactively protect employees whose `employeeKey_i` was
    already derived and used under the compromised key — those remain permanently re-derivable by
    anyone who captured `pseudonymKey` before rotation, **including employees who have since been
    erased** (ADR-0015 §Consequences; `08-security/security-architecture.md` T16b). Rotation is a
    forward-looking containment measure, not a remediation for the past.
  - **Custody of `pseudonymKey` is now the single highest-value secret in the whole system.** Its
    compromise doesn't just de-anonymize current employees at that tenant (bounded by tenant scope,
    an improvement over the pre-hierarchy design) — it silently reverses **every erasure that tenant
    has ever completed**, because `employeeKey_i` is a pure, stateless function of it plus an
    enumerable `employeeInternalId` (PRD §12 OQ-1/errata E-1: shown brute-forceable in seconds). It
    must never leave HSM-protected custody and must never be included in any backup that leaves that
    boundary.
- **Follow-ups:** append to `10-risk/risk-register.md`: (a) B′/C backup/replication retention
  defeating ADR-0015 crypto-shred, (b) `pseudonymKey` custody/compromise as the single highest-impact
  key event in the system, (c) `KEY_EMPLOYEE` accidental loss as involuntary erasure (PRD §12 OQ-5,
  cross-referenced, not solved here). The concrete B′/C store schema, backup policy, and destruction
  procedure are build-time tasks for `fabric-engineer`.

## Related

- **Supersedes:** ADR-0009 (amendment-in-substance — see the status-choice explanation above; ADR-0009's
  Decision is fully re-affirmed for Domains A/B, not reversed).
- **Relates to:** ADR-0015 (executes the destroy-ability this ADR adjudicates for Domains B′/C);
  the pending `EmployeeProfileRecord` construction ADR (owned by `fabric-engineer`, not yet authored)
  for where `DataHash`/`EmployeeID`/`UpdatedBy` are computed from these keys; the pending IPFS Private
  Cluster topology ADR (owned by `fabric-architect`/`fabric-engineer`, not yet authored) for where
  Domain B′ physically lives.
- Knowledge-graph: Layer D — **D3** (salt/predictable-value defense), **D6** (X.509 identity),
  **D14** (HSM key protection); no PDC capability (**D1**) is invoked. Frames **E1** (CIA —
  Confidentiality), **E4** (Security-by-Design — key separation, least privilege, defense-in-depth).
  Grounding gap(s): **G-15** (carried from ADR-0009); no gap ID yet for B′/C rotation-and-backup
  procedure — flagged above for `dsrm-researcher`. Context doc(s): `context/CRYPTOGRAPHY.md` §1–§4,
  `context/FABRIC-CA.md`.

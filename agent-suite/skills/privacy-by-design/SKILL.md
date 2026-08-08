---
name: privacy-by-design
description: >-
  Privacy-by-Design (PbD) engineering for the HRIS-on-Fabric system: apply the seven PbD
  principles and produce a PII data-map / inventory, a DPIA, and data-minimization,
  purpose-limitation, and right-to-erasure / be-forgotten analysis for
  HRIS features — mapping each control to a Fabric mechanism (off-chain PII + on-chain per-section
  salted digests, channel-per-tenant isolation, off-chain four-part crypto-shred erasure) and to
  the existing app crypto (AES-256, PBKDF2 hashes, the per-employee document-encryption and
  identifier-pseudonymization key domains). Trigger on privacy-by-design, PbD, DPIA, data
  protection impact, PII data-map / inventory, data minimization, purpose limitation,
  right-to-erasure / be-forgotten, consent model / data-subject rights, or PDP Law / GDPR privacy
  controls. NEGATIVE: for threat modeling / STRIDE / audit workflows use security-review or
  fabric-security-review; for ASVS / API verification checklists use the OWASP-ASVS / OWASP-API
  docs; for crypto mechanics, keys, and salting see the CRYPTOGRAPHY context doc and
  fabric-identity-security.
license: Apache-2.0
---

# privacy-by-design

Turns the Privacy-by-Design frame into concrete design constraints on the HRIS data model. This
skill runs the **analysis workflow** — data-map → DPIA → minimization/purpose/erasure → Fabric
control mapping — over employee PII and emits a reviewable artifact. It owns the *privacy
reasoning and the deliverables*; it does **not** own the crypto primitives, the threat model, or
the verification checklists (see boundaries below).

> **Re-derived 2026-08-06** against the ratified design (`prd-fabric-hris-2026-08-02/prd.md`,
> ADR-0011/ADR-0012/ADR-0013/ADR-0015/ADR-0016/ADR-0019/ADR-0021). **There is no Private Data
> Collection, no `PurgePrivateData`, and no `blockToLive` anywhere in this design.** Erasure is a
> wholly off-chain, four-part **crypto-shred** (ADR-0015): delete the operational-database field +
> destroy the per-employee document-encryption key (`KEY_EMPLOYEE`) + destroy the per-record
> `DataHash` salt + destroy the per-employee identifier-pseudonymization secret (`employeeKey_i`).
> If you are reading a description of this skill's erasure step that mentions a private data
> collection or a purge API, it is describing the **superseded** pre-reconciliation design — follow
> this file, not that memory.

> **Grounding.** The domain knowledge lives in `../../context/PRIVACY-BY-DESIGN.md` (the seven
> principles, the profile-section data-map, the erasure tension, the two open DPIA-adjacent
> compliance gaps, the consent seam). This skill cross-references that doc for depth and does
> **not** restate it — read it before running the workflow. Fabric confidentiality/erasure
> mechanics are cited to the ADRs and the corpus below.

## When to use this skill

Any task that asks to: apply the 7 PbD principles to a feature; build or extend a **PII
data-map / inventory**; run or template a **DPIA** (including closing the standing **DPIA
obligation itself**, gap **G-25** — this skill's DPIA template is the instrument for that); argue
**data minimization** or **purpose limitation** for what goes on-chain; design a
**right-to-erasure / right-to-be-forgotten** path against an append-only ledger; wire a
**consent / data-subject-rights** model; or map a privacy requirement (UU PDP / GDPR) onto Fabric +
the existing app crypto.

## When NOT to use (route away)

- **Threat modeling, STRIDE, attack surface, hardening checklists, security audit workflows** →
  `security-review` / `fabric-security-review`. (PbD asks "is this data minimal, purposeful,
  erasable?"; security-review asks "who attacks it and how?") The current STRIDE model and its
  residual-risk disposition live in `../../08-security/security-architecture.md`.
- **ASVS or API verification checklists** → the `OWASP-ASVS` / `OWASP-API` context docs.
- **Encryption mechanics, key management, salting scheme, hashing, cert lifecycle** → the
  `CRYPTOGRAPHY` context doc and `fabric-identity-security`. (This skill *decides which fields
  need a control*; those own *how the control is built*.)
- **Channel-per-tenant topology, endorsement policy, chaincode data model** →
  `fabric-chaincode-dev` / `fabric-network-architect`. This skill consumes channel-per-tenant
  (ADR-0013) as the tenant-isolation *fact*; it does not design the topology.
- **Whether ZKP/Idemix applies to a minimal-disclosure use case** → `zkp-designer`
  (`../../context/ZERO-KNOWLEDGE-PROOF.md`); this skill's erasure/minimization analysis is a
  different question from "can a predicate be proven without revealing the value."

## Inputs and outputs

### Input schema
```yaml
feature: <HRIS feature under assessment, e.g. "PAYROLL section change request">
entities:                      # PII-bearing profile sections / models in scope
  - name: <profile_section_or_model>          # e.g. one of PERSONAL/EMPLOYMENT/EDUCATION/ADDITIONAL/PAYROLL
    fields: [<national_id>, <tax_id>, <phone>, <salary>, <bank_account>]   # generic placeholders — never a literal real column name
persistence: <authoritative store — operational DB tables / app models, generic register>
regulatory_basis: <UU PDP No. 27/2022 | GDPR | other>   # UU PDP is now the confirmed basis, [prd: §9.5] — do not silently assume GDPR instead
anchoring_scope: <which profile section(s) are in scope for on-chain digesting>   # ratified boundary: whole section, per [prd: §4] — not a per-field delta
existing_crypto: <AES-256 columns (Domain B), PBKDF2 bank hashes, KEY_EMPLOYEE (Domain B'), employeeKey_i + salt (Domain C)>
```

### Output schema
```yaml
pii_data_map:
  - {entity, field(s), sensitivity, at_rest_state, on_chain_treatment, lawful_basis}
dpia:
  risks: [{id, title, severity, likelihood, affected_entities, recommendation, fabric_or_crypto_control}]
  residual_risk: <statement after controls>
  dpia_instantiated: <yes/no — gap G-25; a "no" here is itself a DPIA finding, not a silent gap>
minimization: [<what NOT to collect / NOT to anchor, and why>]
purpose_limitation: [<visibility confinement: channel-per-tenant D7 decision — no PDC exists>]
erasure_analysis:
  {db_action, key_employee_action, salt_action, employee_key_i_action, onchain_action,
   what_survives, retention_gap_note}
data_subject_rights: [{right, existing_seam, fabric_action}]
fabric_control_mapping: [{control, pbd_principle, fabric_mechanism, capability_id, citation}]
assumptions: [{tag: ASSUMPTION, gap: G-##, statement}]
```

## Core workflow

Run these six steps in order; each produces one block of the output schema.

### 1 — Apply the seven-principle lens
Score the feature against the 7 PbD principles (proactive; privacy-as-default; embedded into
design; positive-sum; end-to-end lifecycle; visibility/transparency; respect for the user).
Depth and this system's reading of each principle: `../../context/PRIVACY-BY-DESIGN.md` §1. The
three that carry the design weight are **data minimization**, **purpose limitation**, and
**right-to-erasure** — steps 4–5.

### 2 — Build the PII data-map / inventory
For every in-scope profile section (or, for a non-anchoring feature, every in-scope entity) list
the sensitive fields, a **sensitivity** rating, the **at-rest state today** (AES-256 / PBKDF2 hash /
**plaintext**), and the proposed **on-chain treatment** — for anything that anchors, this is
**always** "salted `DataHash` of the whole section," never a per-field commitment and never
"not in scope" for one of the five ratified sections (`PERSONAL`/`EMPLOYMENT`/`EDUCATION`/
`ADDITIONAL`/`PAYROLL`, `[prd: §4]`). Anchor field categories and current crypto to the grounding
in `../../context/PRIVACY-BY-DESIGN.md` §2. Guardrail G-17: the inventory is non-exhaustive — treat
any unlisted field as sensitive until classified.

### 3 — DPIA risk scoring
Turn the data-map into scored risks: give each a **severity**, affected entities, and a named
Fabric-or-crypto remediation, then add feature-specific ones. Seed the list with the four standing
risk classes **R1** (unencrypted high-sensitivity content), **R2** (third-party PII in the
`ADDITIONAL` section → never anchor a value, only the section digest), **R3** (small-domain
reversibility), **R4** (partial-encryption rollout, G-21) — full descriptions in
`../../context/PRIVACY-BY-DESIGN.md` §2. **R3 is decisive:** national-ID / phone / salary-band
domains are enumerable, so an unsalted digest is reversible → **salt is mandatory and is the
ratified design's default** (mechanics: `../../context/CRYPTOGRAPHY.md` §3).

**Do not stop at feature-level risks — check whether the DPIA obligation itself is discharged.**
UU PDP Article 34 requires a DPIA through the "use of new technology" trigger, and this artifact's
own novelty claim satisfies it (`[prd: §9.2 row 10, §9.3]`, gap **G-25**). If no DPIA has been
instantiated for the artifact under assessment, that is itself a **DPIA finding to report**, not a
silent precondition — set `dpia_instantiated: no` in the output and route the user to
`assets/dpia-template.md`.

### 4 — Data minimization + purpose limitation → Fabric
Depth: `../../context/PRIVACY-BY-DESIGN.md` §3. Operational decisions this step records:
- **Minimization (D2/D3).** Anchor only a **salted digest of the whole profile section as of the
  write**, never a field value and never a per-field "which fields changed" list — the digest lets a
  verifier confirm a section's value existed and matches, without the ledger ever storing it
  (ADR-0011). **There is no per-event/per-field commitment scheme in this design** — the unit is the
  section, full stop.
- **Purpose limitation — channel-per-tenant, not a collection.** Confine visibility to the tenant
  that owns the data: **channels (D7)** isolate tenants, and this is now the **sole** isolation
  primitive — there is no Private Data Collection anywhere in this design to additionally scope
  field-level visibility within a channel (ADR-0013, ADR-0015 Context). A cross-tenant read is
  rejected at the channel-membership layer before any application-level check runs.
  > Depth: channel-per-tenant topology and its residual `[ASSUMPTION]`s live in
  > `fabric-network-architect` and `00-architecture/solution/fabric-network-design.md`.

### 5 — Right-to-erasure vs. ledger immutability
The core tension (append-only ledger vs. erasure) is in `../../context/PRIVACY-BY-DESIGN.md` §4.
**Fabric resolves nothing here — the entire erasure mechanism is off-chain (ADR-0015):**

1. **Delete the operational-database field values** for the employee's profile sections (ordinary
   row/column delete, not cryptographic).
2. **Destroy the employee's `KEY_EMPLOYEE`** (Domain B′) — every supporting document this employee
   ever uploaded to the IPFS Private Cluster becomes permanently undecipherable. This is the
   *entire* mechanism for document content — **not** removal of the IPFS object; `unpin` is not
   `delete` (ADR-0016) — the compliance claim is "encrypted, and the only key is destroyed," never
   "the file is gone."
3. **Destroy the employee's per-record `DataHash` salts.**
4. **Destroy the employee's `employeeKey_i`.** Since ADR-0021 (2026-08-06), this key is
   independently random with no master key behind it — its destruction is unconditional and cannot
   be reversed by recomputing it from anything else.

**On-chain state is never touched** (FR-27) — no rewrite, no delete, no tombstone. The surviving
`DataHash`/`EmployeeID`/`UpdatedBy`/CID values become permanent, unattributable orphans on that
tenant's channel `[docs: ledger/ledger.md]` (append-only, by design — this is not a defect this step
can fix).

**Operational deltas this skill adds:**
- **Never present crypto-shred as satisfying the automatic-retention duty (recommendation, ties to
  gap G-26).** UU PDP Pasal 42 requires time/purpose-triggered cessation; crypto-shred is
  **request**-triggered (Pasal 43(1)(c)). The two are not the same obligation — a completed erasure
  answers "did the subject's request get honored," not "does the ledger stop growing." Say so
  explicitly in the erasure analysis; do not let a completed crypto-shred read as if it closed G-26.
- **Blast radius (recommendation).** `employeeKey_i` and `KEY_EMPLOYEE` backups/replicas must be
  *provably* destructive, not soft-delete — a silently-retained backup defeats the whole mechanism
  with no visible symptom (ADR-0015/ADR-0019 follow-up).
- **Accidental-loss risk (fact, disclosed, not hidden).** Losing an employee's `employeeKey_i` or
  `KEY_EMPLOYEE` by operator error is now indistinguishable from an intentional erasure — an accepted
  trade-off (ADR-0021), not an oversight.

### 6 — Consent & data-subject-rights seam
**Operational decision:** reuse the **existing employee-initiated change-approval workflow** — one
per profile section — as the consent/rectification seam (recommendation / [ASSUMPTION → G-12])
rather than building a parallel one. Each already captures the proposed edit, the approval trail,
and triggers that section's anchor. Map each right (Access / Rectify / Erase / Consent-record) to
that seam and a Fabric action per the mapping table in `../../context/PRIVACY-BY-DESIGN.md` §5.

## Fabric control mapping (fact table)

| Control (PbD) | PbD principle | Fabric mechanism | Cap. | Citation |
|---|---|---|---|---|
| PII stays off-chain; only a per-section digest is anchored | privacy-as-default / minimization | `DataHash = SHA-256(salt‖JCS(section))`, computed off-chain | D2/D3 | ADR-0011; `[docs: private-data-arch.rst#protecting-private-data-content]` (salting guidance) |
| Tenant isolation, the sole confidentiality boundary (no PDC) | purpose limitation | channel-per-tenant | D7 | ADR-0013 |
| No section value or salt ever a chaincode argument, on Submit or Evaluate | minimization / privacy-embedded | chaincode contract shape | — | ADR-0020 |
| Right-to-erasure — four-part off-chain crypto-shred | respect for user / lifecycle | delete DB field + destroy `KEY_EMPLOYEE` + destroy salt + destroy `employeeKey_i` | — | ADR-0015 |
| Document erasure basis | respect for user / lifecycle | `KEY_EMPLOYEE` destruction (never object removal — `unpin ≠ delete`) | — | ADR-0016 |
| Identifier pseudonymization with no reversal path | minimization / lifecycle | `employeeKey_i` independently random, no master key | — | ADR-0021 |
| Tamper-evident transparency of changes | visibility/transparency | immutable per-section-per-employee audit chain | D12 | `[docs: ledger/ledger.md]` |

## Worked example: a `PAYROLL` profile-section change request

**Input.** feature: PAYROLL section change request; entity `PAYROLL`; regulatory_basis: UU PDP No.
27/2022.

**Data-map (excerpt):**

| Section | Sensitivity | At-rest today | On-chain treatment |
|---|---|---|---|
| `PAYROLL` (salary, bank-account fields) | High (financial) | Bank fields as PBKDF2-SHA256 hashes for lookup `[code: ems/pkg/bankhasher/bankhasher.go]`; other payroll fields plaintext today | Salted `DataHash` of the **whole section** — never a field value |

**DPIA finding (R1/R3).** Several `PAYROLL` fields beyond the bank hash are plaintext at rest →
severity **High** at the operational-database layer, independent of anchoring. Because salary/bank
values are small-domain and enumerable, the digest **must** be salted (R3) — this is the ratified
design's default (ADR-0011), not an opt-in. **DPIA-obligation check (G-25):** if no DPIA has yet
been instantiated for this artifact, that is reported as a finding, not assumed away.

**Minimization.** Anchor a salted `DataHash` of the entire `PAYROLL` section as of the write, not a
delta of the changed fields (ADR-0011). **Purpose limitation.** Tenant isolation is by channel
membership alone (D7, ADR-0013) — no PDC exists to additionally scope this within the tenant's
channel.

**Erasure.** On an erasure request through the payroll change-approval workflow: (1) delete the
operational-database `PAYROLL` field values; (2) destroy the employee's `KEY_EMPLOYEE` (any attached
payslip becomes undecipherable, not "deleted" — `unpin ≠ delete`); (3) destroy the employee's
`DataHash` salts for `PAYROLL`; (4) destroy the employee's `employeeKey_i`. On-chain digests remain
as permanent, unattributable orphans. **Retention-gap note (G-26):** this satisfies the erasure
*request*; it does not, by itself, satisfy UU PDP Pasal 42's automatic-retention duty, since nothing
about crypto-shred is triggered by the passage of time.

**Assumptions emitted.** `[ASSUMPTION G-12]` consent seam = existing change-approval workflow;
`[ASSUMPTION G-21]` partial encryption rollout, do not assume plaintext columns are empty;
`[ASSUMPTION G-27]` Controller/Processor role undetermined, affects who the DPIA/retention
obligations actually bind.

## Guardrails

- **No PII plaintext, and no reversible derivative, on-chain — ever** (ratified invariant INV-1,
  `[prd: §5.2]`). Anchor a salted section digest, never a value, and never a per-field breakdown of
  what changed.
- **Salt every `DataHash`** — national-ID / phone / salary-band domains are enumerable (R3); this is
  the design's default, not a recommendation to add.
- **Third-party PII (the `ADDITIONAL` section's family/dependent data) is still anchored as a whole
  section digest, never as a separately-exposed value** — minimization and lawful basis are weakest
  for third-party data subjects (R2).
- **Don't assume encrypted columns are populated** — partial rollout (G-21).
- **Unlisted fields are sensitive until classified** — inventory is non-exhaustive (G-17).
- **Never invent a retention period, and never claim crypto-shred satisfies the Pasal 42
  automatic-retention duty** — that is a distinct, open gap (G-26); crypto-shred only ever answers a
  *request*.
- **There is no `PurgePrivateData`, no `blockToLive`, and no PDC in this design.** Erasure is the
  four-part off-chain crypto-shred (ADR-0015) — do not describe erasure any other way.
- **Do not present the DPIA obligation (G-25) as satisfied by this skill's analysis alone** — running
  this workflow over a feature is not the same as having instantiated and signed off
  `assets/dpia-template.md` for the artifact as a whole.

## Behavioral rules
- **Cite the corpus/context doc.** Ground factual claims in the official Fabric 2.5
  documentation, the ratified PRD/ADRs, and `../../context/PRIVACY-BY-DESIGN.md`; when official docs
  and community lore conflict, say so and follow the docs. [FR-8]
- **Show the reasoning and the trade-off.** Never present a tunable (block size, endorsement
  policy, state DB) as a universal truth — give the "it depends" and the axis it depends on. [FR-9]
- **Fact vs. recommendation.** Mark documented facts (cited) distinctly from engineering judgment
  ("recommendation:"). [FR-10]
- **Flag the blast radius.** Whenever a suggested action carries a production, security, or
  performance implication, state it before the how-to. [FR-11]
- **State assumptions.** When the user's context is incomplete, mark `[ASSUMPTION]` and invite
  correction rather than guessing silently. [FR-12]

## Evaluation criteria

The deliverable is done when:
- Every in-scope profile section (or entity, for a non-anchoring feature) appears in the data-map
  with sensitivity + at-rest state + on-chain treatment (unlisted fields flagged, not dropped — G-17).
- Each DPIA risk carries a severity and a named Fabric-or-crypto remediation, and the DPIA-obligation
  check itself (G-25, `dpia_instantiated`) is reported, not skipped.
- The erasure analysis names all **four** crypto-shred steps, states that on-chain state is never
  touched, and does not conflate a completed erasure with satisfying the Pasal 42 retention duty
  (G-26).
- Every control maps to a named Fabric capability (D#) or an explicitly off-chain mechanism, and
  cites the corpus/ADR.
- No PII plaintext, or reversible derivative, is proposed for on-chain storage — no exceptions.
- Regulatory specifics beyond the now-settled basis (UU PDP) are tagged `[ASSUMPTION]` with the
  correct gap ID (G-26 retention, G-27 Controller/Processor), never asserted as fact.

## MCP integration

**None required.** This skill is analysis-only over local context docs, ADRs, and the corpus; it
emits a document artifact and reads no external services.

## Reusable prompts

- **DPIA (primary asset):** fill in `assets/dpia-template.md` — a structured DPIA the workflow
  populates step-by-step (processing description → PII inventory → necessity/proportionality →
  lawful basis → risks → Fabric control mapping → data-subject rights → erasure/retention →
  residual risk & sign-off). Use this to close gap **G-25** — instantiate it for the artifact as a
  whole, not just one feature.
- **Data-map from a model:** "Given these profile sections/entities and their at-rest crypto state,
  emit a PII data-map per the output schema; rate sensitivity, flag plaintext high-sensitivity
  content as DPIA risks, and confirm the on-chain treatment is a whole-section salted digest for
  each."
- **Erasure-path check:** "For employee X, produce the right-to-erasure path: the four crypto-shred
  steps in order, what survives on-chain, and the accidental-loss/retention-gap notes."

## References, scripts, and assets

Assets:
- `assets/dpia-template.md` — fill-in Data Protection Impact Assessment for one HRIS feature (or
  the artifact as a whole, to close gap G-25); the step-1..6 workflow above populates it. Read/copy
  it when the task is "run/write a DPIA."

Grounding (read for depth, do not restate):
- `../../context/PRIVACY-BY-DESIGN.md` — the 7 principles, profile-section data-map, erasure
  mechanism, the two open DPIA-adjacent compliance gaps (G-25/G-26), consent seam, and the full
  gap/traceability list.
- `../../context/CRYPTOGRAPHY.md` — at-rest AES-256, PBKDF2 bank hashes, the four key domains
  (A/B/B′/C), the `DataHash`/`employeeKey_i` construction.
- `../../context/ZERO-KNOWLEDGE-PROOF.md` — minimal-disclosure / salary-band proofs (route the
  applicability decision itself to `zkp-designer`).
- `../../05-adr/ADR-0011-per-section-digest-scheme.md`, `ADR-0015-crypto-shred-erasure.md`,
  `ADR-0016-ipfs-private-cluster.md`, `ADR-0019-key-domain-amendment.md`,
  `ADR-0021-employeekey-independent-random-no-master-key.md` — the ratified decisions this skill's
  Fabric control mapping instantiates.

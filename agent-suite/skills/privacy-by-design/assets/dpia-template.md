# Data Protection Impact Assessment (DPIA) — <feature name>

> Fill-in template driven by the `privacy-by-design` skill (workflow steps 1–6). Replace every
> `<...>`. Keep `[ASSUMPTION G-##]` tags where a fact is not yet fixed — do **not** invent it.
> Grounding: `../../context/PRIVACY-BY-DESIGN.md`. Fabric/crypto facts cite the ratified ADRs
> (ADR-0011/ADR-0015/ADR-0016/ADR-0019/ADR-0021) and the 2.5 corpus.
>
> **This template exists specifically to close gap G-25** — UU PDP Article 34 requires a DPIA
> through at least the "use of new technology" trigger, and as of 2026-08-06 no DPIA had ever been
> instantiated for this artifact (`[prd: §9.2 row 10, §9.3]`). Filling this in for the artifact as a
> whole — not only for one feature — is how G-25 gets closed. A partially-filled template with
> honest `[ASSUMPTION]` tags is still real progress against G-25; a template that asserts facts it
> cannot ground is not.

| Field | Value |
|---|---|
| Feature / processing (or "whole artifact" to close G-25) | `<name>` |
| Assessed by / date | `<name>` / `<YYYY-MM-DD>` |
| Regulatory basis | **UU PDP No. 27/2022** — confirmed `[prd: §9.5]`. Do not silently substitute GDPR or leave this as `[ASSUMPTION]`; the basis question (gap G-06) is settled. What remains open is **which obligations bind which party** — see row below and §8. |
| Data-subject population | `<employees; + third parties via the ADDITIONAL section — family/dependent data>` |
| Authoritative store | `<operational database — generic register, no literal table names>` |
| On-chain anchoring scope | The **whole profile section**, per write — `PERSONAL`/`EMPLOYMENT`/`EDUCATION`/`ADDITIONAL`/`PAYROLL` (`[prd: §4]`). **Never** a per-field delta, and never anything but a salted digest. |
| Controller / Processor role | **`[ASSUMPTION G-27]`** — undetermined. Whether the platform org is Controller or Processor (and whether the enterprise-client org instead holds some obligations) is an open legal question `[prd: §9.4]`; do not assume the platform org bears every obligation below by default. |

## 1. Processing description
`<What the feature does with PII: collected, read, updated, anchored (per-section digest only), and
erased. Data flow in 2–4 sentences — describe generically, no literal service/class names.>`

## 2. PII inventory / data-map
> G-17: inventory is non-exhaustive — list every field you touch; treat unlisted fields as
> sensitive until classified.

| Profile section | Sensitive content | Sensitivity (H/M/L) | At-rest state today | On-chain treatment | Lawful basis |
|---|---|---|---|---|---|
| `PERSONAL` | national-ID/tax-ID, contact details | High | partially AES-256 (rollout incomplete, G-21) | salted `DataHash` of the whole section | `<UU PDP basis — name the specific lawful ground>` |
| `EMPLOYMENT` | position/grade/transfer history | Medium–High | operational-DB controls, unchanged | salted `DataHash` of the whole section | `<...>` |
| `EDUCATION` | degree/institution/score | Low–Medium | plaintext | salted `DataHash` of the whole section | `<...>` |
| `ADDITIONAL` | marital status, dependents (**third-party data subjects**) | High | plaintext | salted `DataHash` of the whole section — never a field value | `<...>` — weakest lawful basis; third-party consent is rarely direct |
| `PAYROLL` | salary / bank-account fields | High | bank fields PBKDF2-SHA256 hashed; other fields plaintext | salted `DataHash` of the whole section | `<...>` |
| Supporting documents | identity docs, certificates, payslips | High | encrypted with per-employee `KEY_EMPLOYEE` before IPFS upload; only the CID is anchored | CID only, never content | `<...>` |

## 3. Necessity & proportionality (minimization + purpose limitation)
- **Minimization:** anchor only a salted `DataHash` of the **whole section as of the write**;
  never a field value, never a per-field "which fields changed" breakdown (ADR-0011).
- **Purpose limitation:** tenant isolation is by **channel membership alone** (D7, ADR-0013) — there
  is no Private Data Collection anywhere in this design to additionally scope visibility within a
  channel.

## 4. Risks to data subjects
> Seed from the standing classes R1–R4 (`../../context/PRIVACY-BY-DESIGN.md` §2), then add
> feature-specific risks.

| ID | Risk | Severity | Likelihood | Affected sections | Mitigation (Fabric / crypto control) |
|---|---|---|---|---|---|
| R1 | Unencrypted high-sensitivity content at rest | `<High>` | `<...>` | `<PAYROLL, ADDITIONAL, EDUCATION>` | `<AES-256 rollout; anchoring only ever produces a salted digest, never a value>` |
| R2 | Third-party PII (`ADDITIONAL` section) | `<...>` | `<...>` | `<ADDITIONAL>` | `<the section is anchored as a whole digest; no third-party field value is ever anchored or exposed separately>` |
| R3 | Small-domain reversibility of a digest | `<...>` | `<...>` | `<national-ID, phone, salary-band fields>` | `<mandatory salt — CRYPTOGRAPHY §3, ADR-0011 default>` |
| R4 | Partial encryption rollout (empty columns) | `<...>` | `<...>` | `<non-pilot tenants>` | `<do not assume plaintext absent — G-21>` |
| R5 (compliance) | DPIA obligation itself unmet | `<High until this document is completed and signed off>` | `<certain, until closed>` | `<whole artifact>` | `<this document — G-25>` |
| R6 (compliance) | Automatic retention duty (Pasal 42) unmet by crypto-shred | `<High — disclosed limitation, not yet resolved by design>` | `<certain — structural, not incidental>` | `<every anchored record, indefinitely>` | `<no design-level control yet exists; tracked as G-26, not closed by this document>` |

## 5. Fabric control mapping
| Control | PbD principle | Fabric mechanism | Citation |
|---|---|---|---|
| PII off-chain; anchor a whole-section salted digest only | privacy-as-default / minimization | `DataHash = SHA-256(salt‖JCS(section))`, computed off-chain | ADR-0011 |
| No section value or salt ever a chaincode argument | minimization / privacy-embedded | chaincode contract shape (no such parameter exists) | ADR-0020 |
| Tenant isolation, the sole confidentiality boundary | purpose limitation | channel-per-tenant (D7) — no PDC | ADR-0013 |
| Right-to-erasure | respect for user / lifecycle | four-part off-chain crypto-shred | ADR-0015 |
| Document erasure basis | respect for user / lifecycle | `KEY_EMPLOYEE` destruction; `unpin ≠ delete` | ADR-0016 |
| Identifier pseudonymization, no reversal path | minimization / lifecycle | `employeeKey_i` independently random, no master key | ADR-0021 |
| Transparency of changes | visibility/transparency | immutable per-section-per-employee audit chain (D12) | `[docs: ledger/ledger.md]` |

## 6. Data-subject-rights handling
> Seam **[ASSUMPTION G-12]**: reuse the existing per-section employee change-approval workflow.

| Right | Existing seam | Fabric action |
|---|---|---|
| Access | employee self-service read of their own profile | read the per-section digest history (`GetProfileHistory`, D12) |
| Rectify | change-approval workflow for the relevant section | anchor a new `DataHash` version for that `(EmployeeID, ProfileSection)` chain |
| Erase | erasure request through the same operational workflow | four-part crypto-shred (§7); on-chain digests become permanent, unattributable orphans |
| Consent record | the approval trail itself | the trail *is* the evidence; nothing further is anchored for consent purposes |

## 7. Erasure & retention plan
- **Step 1 — operational-database action:** `<delete the authoritative field values for this employee's profile sections>`
- **Step 2 — document-key action:** `<destroy this employee's KEY_EMPLOYEE — every attached supporting document becomes permanently undecipherable; this is NOT object removal, "unpin ≠ delete" (ADR-0016)>`
- **Step 3 — salt action:** `<destroy this employee's per-record DataHash salts>`
- **Step 4 — identifier-key action:** `<destroy this employee's employeeKey_i — independently random since ADR-0021, so destruction is unconditional and cannot be reversed by recomputing it from any master key>`
- **On-chain action:** **none, ever** — no rewrite, no delete, no tombstone (FR-27). The surviving `DataHash`/`EmployeeID`/`UpdatedBy`/CID values become permanent, unattributable orphans.
- **Evidence of execution:** a signed, off-chain deletion certificate in the operational audit store (FR-29) — never a ledger operation.
- **Retention period:** `<UU PDP basis is settled (regulatory basis), but no automatic-cessation mechanism exists in this design — see R6/G-26. Do not assert a numeric retention window here; the ledger's own digests persist indefinitely regardless of any DB-side retention policy.>`
- **Accidental-loss disclosure:** `<losing this employee's KEY_EMPLOYEE or employeeKey_i by operator error is indistinguishable from an intentional erasure — an accepted trade-off (ADR-0021, mirrors the already-accepted KEY_EMPLOYEE-loss risk, OQ-5), not a defect to silently omit from this assessment.>`

## 8. Residual risk & sign-off
- **Residual risk after controls:** `<statement — must explicitly address R5 (DPIA obligation) and R6 (retention gap) rather than only the feature-level risks R1–R4>`
- **Open gaps:** `<G-06 (basis settled; obligations-allocation open), G-12, G-17, G-21, G-25 (this document), G-26 (retention, not closed by this document), G-27 (Controller/Processor role)>`
- **Decision:** `<proceed / proceed-with-conditions / do-not-proceed>` — `<approver>` / `<date>`
- **What signing this off does and does not do:** completing this document closes gap **G-25**
  (the DPIA obligation existing). It does **not** close **G-26** (the retention duty) or **G-27**
  (the Controller/Processor determination) — those require a design decision and a legal
  determination respectively, neither of which this DPIA can substitute for.

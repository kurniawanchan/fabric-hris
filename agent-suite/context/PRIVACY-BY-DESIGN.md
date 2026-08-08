# Privacy by Design in the HRIS-on-Fabric System

> **Re-derived 2026-08-06** against the ratified design in `prd-fabric-hris-2026-08-02/prd.md`
> (status: final) and **ADR-0011/ADR-0012/ADR-0013/ADR-0015/ADR-0016/ADR-0019/ADR-0021**. The
> pre-reconciliation version of this doc (Private Data Collections + `PurgePrivateData`/
> `blockToLive` as the erasure mechanism, a single-channel + PDC tenancy model) is **superseded in
> substance** — see `../11-execution/grounding-gaps.md` and `../05-adr/README.md`.

> **What this doc is.** A context/theory map of Privacy by Design (PbD) applied to employee PII,
> plus a first-pass **PII data-map / DPIA sketch** over the five ratified profile sections. It is a
> REFERENCE doc, not a skill: the analysis *workflow* (data-map → DPIA → minimization/erasure →
> Fabric mapping) lives in the `privacy-by-design` skill and is **cross-referenced, not restated**
> here. Its job is to turn the PbD frame into concrete design constraints on *this* data model.

> **Regulatory basis is now settled; retention and the DPIA obligation are not.** Indonesia's UU PDP
> No. 27/2022 is the confirmed regulatory basis (gap **G-06** closed on this point,
> `[prd: §9.5]`) — but two compliance gaps surfaced by the ratifying source remain **open, unmet by
> the current design**, and must not be presented as resolved: the **DPIA obligation itself**
> (Pasal 34, gap **G-25**) has never been discharged, and the ratified crypto-shred erasure does
> **not** satisfy Pasal 42's automatic, time/purpose-triggered retention duty (gap **G-26**). See §6.

---

## 1. The seven PbD principles (the lens)

PbD is a mandated security frame `[brief: agents-guide.md]` (`knowledge-graph.md` §Layer E). Its
seven foundational principles, applied to this system:

1. **Proactive not reactive** — anchor tamper-evidence *before* a breach, not forensics after.
2. **Privacy as the default** — PII is encrypted at rest by default (Domain B, `CRYPTOGRAPHY.md`
   §4); plaintext never leaves the operational database onto the ledger, by construction — no
   chaincode function ever accepts a profile-section value as an argument, on Submit or Evaluate
   (ADR-0020).
3. **Privacy embedded into design** — confidentiality is a data-model property (off-chain PII +
   on-chain per-section digests), not a bolt-on; there is no PDC that could be misconfigured into
   over-exposure, because none exists (ADR-0015 Context).
4. **Full functionality (positive-sum)** — tamper-evident, independently co-signed audit *and*
   confidentiality, not one traded for the other (ADR-0012's three-org endorsement).
5. **End-to-end security / full lifecycle protection** — from write (application AES, Domain B)
   through anchor (salted digest, Domain C) to erasure (four-part crypto-shred, ADR-0015).
6. **Visibility and transparency** — the immutable, per-section-per-employee audit chain is the
   transparency mechanism (D12); an enterprise-client or auditor verifier recomputes and compares
   client-side, against their own peer, never trusting the platform's word for it (FR-37).
7. **Respect for user privacy** — data-subject rights routed through the existing change-approval
   consent seam (§5); erasure is a genuine, irreversible key-destruction event (ADR-0015), not a
   cosmetic access restriction.

Three operational patterns carry most of the weight: **data minimization**, **purpose limitation**,
and **right-to-erasure** (§3–§4).

---

## 2. PII data-map / DPIA sketch — the five ratified profile sections (Layer B)

The anchoring unit is a **profile section**, not a per-field table (`[prd: §4]`, superseding the
pre-reconciliation event-based surface entirely — see gap G-05, closed). Sensitivity and lawful-basis
columns below reflect the settled regulatory basis (UU PDP) but not yet a completed DPIA (G-25); the
on-chain treatment follows the ratified confidentiality invariant (INV-1: zero PII plaintext or
reversible derivative on-chain, always).

| Profile section | Representative PII content | Sensitivity | At-rest state today | On-chain treatment |
|---|---|---|---|---|
| **PERSONAL** | national-ID/tax-ID, contact details, demographic fields | High (national-ID, tax) | Partially AES-256 at rest; rollout incomplete for non-pilot tenants (G-21) `[code: ems/pkg/db/encryption_plugins.go]` | `DataHash` of the whole section only — never a field value |
| **EMPLOYMENT** | position, grade, transfer/mutation history | Medium–High | Existing operational-DB controls, unchanged | `DataHash` only |
| **EDUCATION** | degree, institution, score | Low–Medium | Plaintext today | `DataHash` only |
| **ADDITIONAL** | marital status, dependents (family PII — third-party data subjects) | High (third-party PII) | Plaintext today | `DataHash` only — **never anchor a value for a section this identifiable and this sensitive** |
| **PAYROLL** | salary/bank-account fields | High (financial) | Bank fields as PBKDF2-SHA256 hashes for lookup `[code: ems/pkg/bankhasher/bankhasher.go]`; other payroll fields plaintext today | `DataHash` only |
| **Supporting documents** (identity docs, certificates, payslips) | Attached evidence for any section | High | Encrypted with per-employee `KEY_EMPLOYEE` before upload to the IPFS Private Cluster (ADR-0016); only the CID is anchored | CID only, never document content (FR-30/31) |

**DPIA risk flags (recommendations, seed for the `privacy-by-design` skill's DPIA):**

- **R1 — Unencrypted high-sensitivity content at rest.** Several sections (`PAYROLL` fields beyond
  the bank hash, `ADDITIONAL`, `EDUCATION`) are plaintext today. Anchoring only ever produces a
  salted `DataHash` — never a value — but the *operational-database* exposure itself is a standing
  DPIA risk independent of anchoring (ties to G-17: the PII inventory is non-exhaustive; treat any
  unlisted field as sensitive until classified).
- **R2 — Third-party PII.** `ADDITIONAL` (family/dependent data) concerns people who are *not* the
  employee data subject; minimization and lawful basis are weakest here. The section is still
  anchored as a whole (per `[prd: §4]`), but no third-party field value is ever anchored — only the
  section digest.
- **R3 — Small-domain reversibility.** National-ID, phone, and salary-band values have enumerable
  domains — an unsalted digest is reversible (see `CRYPTOGRAPHY.md` §3). Salt is mandatory and is
  the ratified design's default, not an opt-in (ADR-0011).
- **R4 — Partial encryption rollout (G-21).** Encrypted columns may be empty for non-pilot tenants;
  the HRIS write path computing `DataHash` must read through the application layer, never assume the
  encrypted column is populated.

---

## 3. Data minimization & purpose limitation → Fabric mapping

- **Minimization.** Anchor the **minimum** needed for tamper-evidence: a salted digest of the
  **whole profile section as of the write**, never a field value, and never even a per-field
  "which fields changed" breakdown (ADR-0011; the pre-reconciliation `changedFieldNames` array does
  not exist in this design — see `security-architecture.md` T9). The digest is enough to later prove
  a value existed and matches, without the ledger ever storing it.
- **Purpose limitation — channel-per-tenant is now the *sole* isolation primitive.** The
  pre-reconciliation design confined field-level visibility with a Private Data Collection (D1)
  layered under a single shared channel. **Neither exists now.** Tenant isolation is achieved
  entirely by **channel membership** (D7, ADR-0013): every peer on a tenant's channel holds only
  that tenant's keys, and a cross-tenant read is rejected at the MSP/channel-membership layer before
  any application-level ownership check runs. There is no PDC configuration surface left to
  misconfigure (`security-architecture.md` §5, "Microsegmentation").
  > Depth: channel-per-tenant topology and its residual `[ASSUMPTION]`s (Org3 per-tenant membership
  > scope) live in `fabric-network-architect` and `00-architecture/solution/fabric-network-design.md`.

---

## 4. Right-to-erasure vs. ledger immutability (the core tension — resolved off-chain, not on-chain)

A blockchain is append-only; PbD and UU PDP both demand erasure. **This design resolves the tension
entirely off-chain, with a four-part crypto-shred (ADR-0015) — there is no PDC, no
`PurgePrivateData`, and no `blockToLive` anywhere in this design; none of the three has anything to
act on, because no PII value or reversible derivative is ever inside Fabric-managed state to purge.**

**The four-part crypto-shred (ADR-0015), and nothing else:**

1. **Delete the operational-database field values** for the employee's five profile sections — an
   ordinary row/column delete, not a cryptographic operation (FR-25).
2. **Destroy the employee's `KEY_EMPLOYEE`** (Domain B′) — every supporting document this employee
   ever uploaded becomes permanently undecipherable (FR-26). This is the *entire* mechanism for IPFS
   content; it is **not** removal of the IPFS object itself — `unpin` is not `delete` (ADR-0016).
3. **Destroy the employee's per-record `DataHash` salts** (FR-28) — without them, no `DataHash` can
   be recomputed or dictionary-attacked.
4. **Destroy the employee's `employeeKey_i`** (FR-28) — without it, `EmployeeID`/`UpdatedBy` for
   every record this employee ever anchored can no longer be recomputed or matched by anyone.

**What is never touched:** on-chain state (FR-27, INV-3) — no rewrite, no delete, no tombstone
transaction. The surviving `DataHash`/`EmployeeID`/`UpdatedBy`/CID values become permanent,
unattributable orphans on that tenant's channel. **Evidence of execution** is a signed, off-chain
deletion certificate in the operational audit store (FR-29) — never a ledger operation, since a
ledger-side erasure record would itself be a new re-identification signal.

**Design honesty this doc must carry, not soften:**

- **`employeeKey_i` compromise/loss is now the accidental-loss risk, not the T16b reversal risk.**
  ADR-0021 (2026-08-06) removed the `pseudonymKey` master key that could have retroactively reversed
  a completed erasure; the remaining, accepted trade-off is that an accidentally *lost*
  `employeeKey_i` is permanent and indistinguishable from an intentional erasure — symmetric with the
  already-accepted `KEY_EMPLOYEE`-loss risk (OQ-5).
- **Erasure does not stop the ledger from growing, and it never will on its own.** Every erased
  employee still leaves a permanent orphaned digest set on that tenant's channel forever. This is not
  a bug to fix later — it is the concrete mechanism behind the open **Pasal 42 gap (G-26)**: UU PDP's
  retention duty is **automatic**, triggered by time or the end of a purpose; crypto-shred is
  triggered by a **request** (Pasal 43(1)(c)). The two are not the same obligation, and satisfying
  one does not satisfy the other. **Do not present crypto-shred as closing the retention question.**

> Depth: erasure/retention as a control, and the STRIDE residual for the identifier-key domain
> (T16a/T16b), are reviewed in `../08-security/security-architecture.md`. `fabric-security-review`
> owns the build-time hardening checklist.

---

## 5. Consent & data-subject-rights seam (G-12)

**Recommendation / `[ASSUMPTION → G-12]`:** reuse the **existing employee-initiated change-approval
workflow** (one per profile section — a personal-data change-approval workflow, a transfer/mutation
approval action, an education-history write action, a family-data change-approval workflow, a
payroll/bank-account update action) as the consent/rectification seam rather than building a
parallel one. Each already captures a proposed edit and an approval trail before the section commits
and its digest anchors (`integration-design.md` §1).

Mapping data-subject rights onto existing seams:

| Right | Existing seam | Fabric action |
|---|---|---|
| **Access** | employee self-service read of their own profile | read the per-section digest history (`GetProfileHistory`, D12 audit) |
| **Rectify** | change-approval workflow for the relevant section | anchor a new `DataHash` version for that `(EmployeeID, ProfileSection)` chain |
| **Erase** | erasure request through the same operational workflow | four-part crypto-shred (§4, ADR-0015); on-chain digests become permanent, unattributable orphans |
| **Consent record** | the approval trail itself | the trail *is* the evidence; nothing further is anchored for consent purposes |

---

## 6. Two open compliance gaps this document must not launder into "done"

**G-25 — the DPIA obligation itself has never been discharged.** UU PDP Article 34 requires a Data
Protection Impact Assessment through at least the "use of new technology" trigger — the ratified
design's own novelty claim `[prd: §9.2 row 10, §9.3]`. A DPIA template exists
(`../skills/privacy-by-design/assets/dpia-template.md`) but has never been instantiated for this
artifact. It cannot be filled in meaningfully until the key-hierarchy and erasure ADRs exist — they
now do (ADR-0011/ADR-0015/ADR-0016/ADR-0019/ADR-0021) — so instantiating the DPIA is now unblocked
and should be treated as a live, closeable action, not a standing assumption.

**G-26 — crypto-shred does not satisfy Pasal 42's retention duty.** See §4's honesty note. This is
distinct from G-06 (regulatory basis, now settled): G-26 is specifically about the missing
**automatic-cessation mechanism**, which a request-triggered crypto-shred structurally cannot
provide by itself. No design change in this reconciliation closes it.

**G-27 (adjacent, not this doc's to resolve) — Controller/Processor role.** Which UU PDP obligations
bind the platform org versus the enterprise-client org depends on an undetermined legal role
question (`[prd: §9.4]`) — this affects *who* the DPIA (G-25) and retention design (G-26) obligations
actually bind, but the legal determination itself is out of this document's scope.

---

## 7. Fact vs. recommendation (summary)

| # | Statement | Type |
|---|---|---|
| No PDC, `PurgePrivateData`, or `blockToLive` exists anywhere in this design | ADR-0015 Context, ADR-0020 Alternative E | Fact |
| Erasure = delete DB field + destroy `KEY_EMPLOYEE` + destroy salt + destroy `employeeKey_i`; on-chain state is never touched | ADR-0015 | Fact |
| `unpin` is not `delete` for IPFS-hosted supporting documents | ADR-0016 | Fact |
| `employeeKey_i` is independently random, no master key; T16b (retroactive erasure-reversal) is closed | ADR-0021 (2026-08-06) | Fact |
| Regulatory basis is UU PDP No. 27/2022 | `[prd: §9.5]` | Fact |
| Crypto-shred does not satisfy Pasal 42's automatic retention duty | gap G-26 | Fact (disclosed limitation) |
| A DPIA has never been instantiated for this artifact, despite Pasal 34 triggering it | gap G-25 | Fact (disclosed limitation) |
| Keep third-party (`ADDITIONAL`-section) PII minimized; never anchor a value, only the section digest | R2 (DPIA) | Recommendation |
| Reuse the existing change-approval workflow as the consent seam | gap G-12 | Recommendation |
| Controller/Processor role, and therefore who the DPIA/retention obligations bind | gap G-27 | Assumption |

---

## 8. Traceability

- **Capabilities:** D2/D3 (digest, salting — no D1 PDC, no D4 purge, both retired) · D7 (channel
  isolation, sole tenancy primitive) · D12 (immutable audit) — `knowledge-graph.md` §Layer D.
- **Frame:** Privacy-by-Design (E3), supporting CIA-Confidentiality (E1).
- **Spine:** confidentiality chain #2 and erasure chain #4 in `knowledge-graph.md` §3 — re-derived
  here per ADR-0015's mechanism; the PDC-based version in `knowledge-graph.md` itself is not edited
  by this doc (owned by `dsrm-researcher`).
- **Gaps:** G-02 (re-scoped: section digest, not event), G-03 (closed: channel-per-tenant), G-06
  (regulatory basis closed; retention split into G-26), G-12 (consent seam, open), G-17 (PII
  inventory completeness, open), G-21 (rollout phase, open), **G-25 (DPIA obligation unmet, new)**,
  **G-26 (Pasal 42 retention unmet by design, new)**, **G-27 (Controller/Processor role
  undetermined, new, adjacent)**.
- **Sibling docs:** at-rest & digest crypto → `CRYPTOGRAPHY.md`; minimal-disclosure →
  `ZERO-KNOWLEDGE-PROOF.md`; full STRIDE residual → `../08-security/security-architecture.md`.

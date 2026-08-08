# ADR-0017: Keep the existing operational relational database as the authoritative store for the five profile sections

- **Status:** **Accepted**
- **Date:** 2026-08-06
- **Deciders:** fabric-architect
- **Rests on assumption(s):** **OQ-2** (PRD §12) — the sponsor decision **S-2**
  (`rencana-rekonsiliasi.md` §2) recommending this exact answer is, as of this writing, **still
  pending formal human ratification** (it does not appear in PRD Lampiran A's ratification trail).
  This ADR records the working recommendation the package proceeds on; it must be reconfirmed at the
  next G9 re-approval, not treated as independently self-ratifying.
- **Supersedes:** none (new — no predecessor ADR fixes which relational store is authoritative)
- **Superseded-by:** none
- **Ratifying authority / date:** `rencana-rekonsiliasi.md` §2 (S-2 recommendation), PRD §11.3 (ADR
  manifest, row ADR-0017), PRD §5.1/§4.1 (five-section table) — recommendation authored 2026-08-03;
  **formal sponsor sign-off pending**.

## Context

PRD §12 OQ-2 records an unresolved inconsistency: the thesis's BAB IV 4.1.1 describes the platform's
*as-is* operational store as one relational product ("MySQL di Alicloud RDS"), while its own Tabel
4.2 specifies a *different* one ("PostgreSQL di AWS RDS") for profile data, with no migration
rationale given anywhere. `rencana-rekonsiliasi.md` §2 (S-2) records the working recommendation —
**keep the existing store, do not migrate** — as "konsisten dengan posisi 'berdampingan, bukan
menggantikan'" and to avoid "migrasi yang tidak pernah dijustifikasi."

This is not a peripheral naming question. **ADR-0002** already fixes Fabric as a pure audit/anchor
overlay — the relational tier, not the ledger, is where PII plaintext lives and where the canonical
JSON view of each profile section is read from to compute `DataHash` (PRD §5.2, FR-2/FR-3). Whichever
store is authoritative here is the **input to every anchored digest** and the **recovery source** for
OQ-6's Option A (point-in-time-recovery as the operational answer to "rekonstruksi nilai asli"). "Which
store is authoritative" is therefore load-bearing for P1 (the write side) and for OQ-6 (the recovery
side), not a footnote.

The five profile sections (PRD §4.1) are the unit this ADR must place:

| Section | Table (PRD §4.1 register) | Update trigger |
|---|---|---|
| PERSONAL | `employees` | Jarang |
| EMPLOYMENT | `emp_employment` | Promosi / mutasi |
| EDUCATION | `emp_education` | Pendidikan baru selesai |
| ADDITIONAL | `emp_additional` | Status perkawinan / tanggungan berubah |
| PAYROLL | `emp_payroll` | Penyesuaian gaji / ganti rekening |

**Real-system grounding note (generic, credibility-only — no vendor or literal schema names).** A
structural review of the class of system this prototype targets (a commercial multi-tenant SaaS HRIS
platform's employee module) shows that PERSONAL and EMPLOYMENT attributes are commonly **co-located in
a single wide employee master record**, distinguished by field-level validation/update-scenario
grouping rather than by separate physical tables; EDUCATION history and dependents/ADDITIONAL data are
typically separate one-to-many child tables; PAYROLL/disbursement data is typically a separate,
often-versioned history table. This matters for this ADR's decision: the PRD's five tables should be
read as **five logical projections a write path must produce**, not a guarantee of five physical
tables in any real implementation. That projection-per-section contract (already flagged as **PB-6**,
`rencana-rekonsiliasi.md` Gelombang 4 #21) is a `data-model.md`/integration consequence, noted here,
not designed here.

## Decision

We will keep the platform's **existing, already-in-production operational relational database** as
the sole authoritative (system-of-record) store for all five profile sections. **No migration to a
different relational product is undertaken.** The "PostgreSQL/AWS RDS" text in some prior artifacts
is treated as the drafting inconsistency OQ-2 identifies — closed by this ADR as *not* a target
state — rather than as a decision to migrate.

Fabric's ledger continues to be a pure overlay on top of this tier (ADR-0002, unaffected by this
ADR): the relational tier is read from to (a) compute `DataHash` at write time for `RecordProfileSection`,
and (b) serve, later, as the source an auditor reads a **candidate** value from before using
`VerifyProfileIntegrity` to confirm it matches the anchored `DataHash` (OQ-6 Option C's narrowed
claim + Option A's PITR-based recovery).

**No migration project exists in this design's scope.** The only new work this decision implies is
additive: instrumenting the section-write path to also compute and anchor a digest (Kelompok A). No
schema migration, no engine cutover, no dual-write period is required by this ADR.

## Alternatives considered

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. Keep the existing operational DB, no migration (chosen)** | Zero migration risk/cost; consistent with the PRD's own "lapisan berdampingan, bukan menggantikan" framing; OQ-2 closes without inventing a justification the source material never gave | Does not itself resolve the physical-table-vs-logical-section mapping question (see grounding note above) — a real integration cost, but one that exists regardless of which store is chosen | **Chosen** |
| B. Migrate to a different relational product (a literal reading of Tabel 4.2's PostgreSQL text as a deliberate decision) | Would match one of the two conflicting thesis passages | No requirement or success predicate (P1–P4) depends on which relational product stores the operational data — only on whether the anchored digest matches its source; OQ-2's own framing shows the "different product" text is most likely a drafting inconsistency, not a considered migration decision; introduces live-cutover schedule/consistency risk for zero benefit to the ratified objective | Rejected |
| C. Split out a new, dedicated relational store just for the five profile sections, leaving the rest of the platform's data where it is | Could isolate the anchored domain's schema from unrelated platform data | Creates a **second** system-of-record for employee data — every consuming feature (payroll runs, reporting, other HRIS modules) would need to decide which store is authoritative for which field, re-introducing at the DB-vs-DB layer exactly the ambiguity ADR-0002 already closed at the ledger-vs-DB layer; both stores would sit under the same operator (Org1) and the same trust boundary regardless, so there is no isolation benefit to justify the migration | Rejected |
| D. Make the ledger itself authoritative for section content (store full values on-chain or in a PDC) | Would remove any "which off-chain store is authoritative" question entirely | Directly forecloses INV-1/FR-31 (no PII plaintext on-chain, ever); listed only for completeness | Rejected outright |

## Consequences

- **Positive:** zero migration risk or cost. The build work this reconciliation wave adds
  (write-path digest computation, read-path verify) stays strictly additive to the existing
  operational tier. OQ-2 closes on the "berdampingan, bukan menggantikan" position stated in the
  PRD's own executive summary. OQ-6's recovery story (Option A, PITR) is now grounded in a single,
  unambiguous operational store instead of an unresolved MySQL-vs-PostgreSQL split.

- **Negative / trade-off:** this ADR does **not** settle whether the five PRD sections are five
  physical tables or fewer (per the grounding note, PERSONAL+EMPLOYMENT are commonly co-located in
  systems of this class). The write-path digest computation (`fabric-engineer`/data-model owner) must
  therefore build its canonical-JSON projection **per logical section**, independent of physical table
  count — a real integration cost that a naive "one table = one section" assumption would silently
  miss.

- **Negative / trade-off:** because the relational tier remains the sole plaintext source of truth,
  its **existing** operational security posture (at-rest encryption, access control, and — critically
  — backup/PITR retention window) becomes load-bearing for two claims it was not originally scoped to
  carry: OQ-6's forensic-recovery story (P1) and FR-25's "hapus field data profil" step of the
  crypto-shred erasure design. This ADR does not harden that tier — it is inherited as-is. Any
  pre-existing weakness there (most notably: **no retention-window policy is specified anywhere in the
  PRD**, the same gap underlying §9.3 row 11's "Tidak dapat diklaim" finding on Pasal 42) is a
  pre-existing risk, not one newly introduced by this ADR, but it is now explicitly load-bearing for
  two more claims than before.

- **Follow-ups:**
  - **`TBD` escalation:** the concrete backup/PITR retention window needed to make OQ-6 Option A
    concrete is unspecified. Recommend it be fixed in the same sponsor session that formally ratifies
    S-2/OQ-2, since both are "which properties does the authoritative operational tier guarantee"
    questions.
  - `data-model.md` owner: formalize the per-section canonical-JSON projection contract (PB-6,
    `rencana-rekonsiliasi.md` Gelombang 4 #21) — this ADR names the requirement, does not design the
    contract.
  - This ADR's own **Status** should be revisited if S-2 is formally ratified with a different answer
    at the next G9 session — per the "Rests on assumption(s)" field above.

## Related

- Supersedes: none.
- Relates to: **ADR-0002** (Fabric as overlay — this ADR is the relational-tier complement to that
  decision), forthcoming **ADR-0011** (digest scheme reads section JSON from this tier), OQ-6 (§12,
  recovery story), PB-6 (`rencana-rekonsiliasi.md` Gelombang 4 #21, projection contract).
- Knowledge-graph: Layer B (HRIS data entities), **D12** (immutable audit — what the ledger anchors
  is derived from this tier). Grounding gap: none of G-02/03/04/09/10/11 directly; rests on **OQ-2**
  (PRD §12) as recorded above. Context: `context/BLOCKCHAIN-INTEGRATION.md`, `context/BLOCKCHAIN-DATA-MODEL.md`.

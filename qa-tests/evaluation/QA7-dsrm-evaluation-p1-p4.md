# QA-7 — DSRM Demonstration & Evaluation Against P1–P4

Backlog item `QA-7` (`implementation-backlog.md`). Reports the four ratified, falsifiable success
predicates (`prd.md` §3) against REAL evidence gathered across this build phase (`NET-*` through
`QA-6`) — not asserted, not re-derived here. Every claim below cites the specific test/tool that
produced it; nothing in this report is a new measurement. Per `prd.md` §10.1/§10.5, `P1`/`P2`/`P4`
all required a running prototype, which the sponsor's `S-1` decision (2026-08-06) explicitly
authorized building for this purpose (`Testing` + `Experimental` Hevner families, now closed).

**This report does not touch the actual thesis `.docx` file.** Per the user's explicit instruction
(2026-08-07): a separate findings report only — folding this into the thesis document itself is
the user's own decision and action.

## 0. Verdict summary

| Predicate | Ratified pass bar (`prd.md` §3) | Verdict | Evidence |
|---|---|---|---|
| **P1** ⭐ | 100% hash-mismatch detection across all five `ProfileSection` values | ✅ **PASS** | §1 |
| **P2** | Write latency < 3s @ 500 TPS | ❌ **FAIL** (by a wide margin, environment-attributed) | §2 |
| **P3** | Cross-tenant access rejected by MSP, demonstrated with a REAL second tenant | ✅ **PASS** | §3 |
| **P4** | Every §9 compliance-matrix row honestly reported at its real status tier | ✅ **PASS** (as a *reporting* predicate — see §4 for what that does and does not mean) | §4 |

**Not one of the four predicates tests confidentiality** — `prd.md` §3.2 says this explicitly and
calls it "the most important open issue in this document" (`OQ-1`). That question now has a real
answer, reported honestly in §5: mostly good news, with one real, newly-disclosed gap.

---

## 1. P1 — Tamper detection, all five sections

**Bar:** "100% pada kelima section" (`prd.md` §3, Tabel 3 row P1) — falsified by **one** manipulation
scenario passing undetected. `prd.md` §3 ⚠️ explicitly flags that the thesis's own Tabel 4.4 only
ever tested four sections (PAYROLL/PERSONAL/EMPLOYMENT/EDUCATION as SC-A..D) and never `ADDITIONAL`
— closing that exact gap (errata **E-2**) was this build phase's own explicit mandate.

**Evidence, live, all five sections independently anchor-then-verify-then-tamper-then-reject:**

| Section | Live test | Result |
|---|---|---|
| PERSONAL | `write-path-integration/gateway-client/verify_integration_test.go` (`REC-7`) — 3 verifier classes (Org1, OrgClient-tenant01, Org3), each against their own peer | `Matched: false` on tampered value, `Matched: true` on the genuine value from all 3 verifiers |
| EMPLOYMENT | `write-path-integration/gateway-client/tamper_detection_all_sections_integration_test.go` (`QA-5` `ST-5`) | `Matched: false`, hashes shown differing |
| EDUCATION | same file | `Matched: false`, hashes shown differing |
| **ADDITIONAL** (labeled **SC-F**, per `test-strategy.md`'s naming instruction — tampers with marital-status/dependent-count data) | same file | `Matched: false`, hashes shown differing — **the gap `prd.md` §3 flagged is now closed** |
| PAYROLL | same file | `Matched: false`, hashes shown differing |

Every case includes a positive control (`Verify()` against the original, un-tampered value still
reports `Matched: true`) proving the mismatch is caused by the tamper, not a broken verify path.
The underlying mechanism (`SHA-256(salt‖JCS(section))` comparison, `gateway-client/verify.go`) does
not branch on `ProfileSection` value — per-section coverage was a **test-authoring gap**, not a
code gap, confirming that closing it required no new production code, only new test scenarios.

**Verdict: PASS, 100%, five for five, live.**

---

## 2. P2 — Write latency < 3s @ 500 TPS

**Bar:** p50-class write latency under 3 seconds at a sustained 500 TPS write load (`prd.md` §3.1
explicitly: the ratified success criterion is the **threshold**, not any particular measured
number — the thesis's own printed 850.4/1,950.5 TPS, 2.85s figures were placeholders, never a real
measurement, and `QA-4`'s entire purpose was to replace them with one).

**Real measurement (`QA-4`, `qa-tests/performance/RESULTS.md`):**

| Load point (requested) | Achieved successful throughput | p50 latency | p99 latency |
|---|---|---|---|
| 200 TPS | ~36 TPS (500/2910 succeeded) | 62.9s | 70.2s |
| **500 TPS** | ~208 TPS (506/7562 succeeded) | **21.1s** | 34.1s |
| 1000 TPS | ~126 TPS (531/6976 succeeded) | 52.0s | 54.7s |
| 2000 TPS | ~129 TPS (507/10323 succeeded) | 53.1s | 63.2s |

**Verdict: FAIL, decisively, at every load point** — the 500 TPS load point specifically (the
predicate's own named bar) measures **21.1s p50**, roughly **7×** over the 3-second budget; p99
figures across all four load points range 9×–23× over budget.

**Root cause, independently confirmed, not assumed:** a raw, non-Caliper `peer chaincode invoke`
CLI call took 20.3 seconds for a single **unloaded** write, and `docker stats` showed `peer0.org1`
at 128% CPU **before** any Caliper load was applied. This one laptop was simultaneously hosting the
entire 3-org/4-peer/3-orderer Fabric network, the CCaaS chaincode server, a 2-node IPFS+cluster
pair, and unrelated background processes throughout this whole build phase. **This is a real,
honestly-reported measurement of this specific environment on this date — it is not evidence
against the chaincode/endorsement-policy design itself**, and re-running the identical artifact on
dedicated, non-oversubscribed hardware would very likely produce materially different numbers.
Neither claim should be made without the other: the FAIL is real and must not be hidden; the
environment attribution is also real and must not be omitted to make the FAIL look like a design
defect it has not been shown to be.

**Supplementary, real finding (`QA-6`, not itself part of P2's bar but directly relevant to `DATA`
§5.2's previously-`[ASSUMPTION]`-tagged concurrent-write question):** when multiple writers target
the exact same head key, genuine Fabric `MVCC_READ_CONFLICT` rates of 50%/80%/90% were measured at
2/5/10 concurrent racers respectively — independently reproduced and independently cross-verified
against both endorsing peers' own raw ledger validation logs. This is a mechanical `(N-1)/N`
property of "exactly one winner per burst," not a general background contention rate to
extrapolate from, but it does confirm the endorsement/ordering pipeline's MVCC enforcement is
real and observable, separately from — and not the cause of — P2's timeout-dominated failure mode.

**Verdict: FAIL. Reported honestly, with its real, verified cause.**

---

## 3. P3 — Cross-tenant isolation, demonstrated not asserted

**Bar:** "Peer tenant B berhasil membaca record tenant A" falsifies this predicate (`prd.md` §3).
`prd.md` §9.6 and errata **E-11** are explicit: this must be **demonstrated** with a genuinely
provisioned second tenant and its own peer — a single-tenant network or a mock **cannot** produce
evidence for P3, only assert it by construction. `NET-7` provisioning `tenant02` as a real,
independent second tenant (its own channel, its own `OrgClient-tenant02` peer) is the hard
prerequisite this predicate needed and did not have before this build phase.

**Evidence, live, both directions (`write-path-integration/gateway-client/
tenant_isolation_integration_test.go`, `INT-4`, re-confirmed fresh by `QA-2` `IT-8` and `QA-5`
`ST-14`):**

- `OrgClient-tenant02`'s own identity, targeting `tenant-tenant01`'s real, populated channel:
  **rejected** — `"could not get last config for channel tenant-tenant01"`. `peer0.tenant02` was
  never joined to that channel; the rejection happens at the channel-membership layer itself, not
  merely a chaincode-level ACL check.
- The symmetric direction (`OrgClient-tenant01` targeting `tenant-tenant02`): **rejected** the same
  way.

**Verdict: PASS**, and — per `prd.md` §9.6's own instruction not to overclaim the mechanism — stated
precisely: this is **identity-bound / channel-membership-enforced** isolation (a real Fabric peer
holds no blocks at all for a channel it never joined), the correct, defensible characterization
this session's own evidence supports, not a looser "cryptographic" label.

---

## 4. P4 — UU PDP compliance, reported at its real, tiered status

**Bar (`prd.md` §3.1a, §9.2):** every row in the ratified 11-row compliance matrix is reported at
its actual status — **Comply**, **Comply sebagian**, **Comply dengan catatan**, or **Tidak dapat
diklaim** — and every **Tidak dapat diklaim** row is stated openly as a research limitation, never
silently rounded up to "5 pasal terpenuhi" (the thesis's own original, verified-incorrect claim).

**This predicate is a REPORTING obligation, not a technical pass/fail this build phase can move.**
The matrix itself is a legal-compliance analysis already ratified in `prd.md` §9.2 (verified
against the primary UU PDP text via ABNR's bilingual copy, cross-checked against three other
sources, `prd.md` §9.5) — QA-7's job is to carry it forward faithfully and note where this
session's own technical evidence bears on it, not to re-adjudicate Indonesian law.

**The 11 rows, exactly as ratified** (note: `prd.md` §9.2's own prose says "12 baris" but the
table it introduces contains 11 numbered rows — reported as found, not silently corrected to 12
or trimmed to match the prose; this is a pre-existing, small inconsistency in the ratified source
document, not something this report introduces or resolves):

| # | Pasal | Status | This build phase's evidence, where applicable |
|---|---|---|---|
| 1 | Pasal 6/30/14 (rektifikasi data) | Comply sebagian | `PrevHash` version chaining (`REC-1`/`CC-2`) provides the chronological evidence half; the operational-DB mutability half is outside this repo's scope (ADR-0017) |
| 2 | Pasal 29/16(2)(d) (akurasi & verifikasi) | Comply sebagian | `Verify()` (`REC-7`) is the verification operation cited — live-proven this session |
| 3 | Pasal 39/35/52 (cegah akses tidak sah) | Comply sebagian | MSP validation (`identity.go`) + channel-per-tenant + `KEY_EMPLOYEE`-encrypted IPFS docs (`REC-5`) all live-proven; **caveat below** |
| 4 | Pasal 8/43/44/45 (hapus/musnah) | Comply dengan catatan | Crypto-shred (`REC-6`) now has LIVE proof this build phase added: `QA-2`'s `IT-5` (on-chain state unchanged post-erasure) and `IT-9` (old ciphertext un-decryptable post-erasure, `ST-13`) — strengthens the technical mechanism's evidence base; the **legal** caveat (crypto-shred vs. "menghapus" vs. "memusnahkan", `prd.md` §9.4) is unaffected by this — that remains an unresolved legal-argument question, not a technical one |
| 5 | Pasal 38/39/37 (awasi pemrosesan) | Comply dengan catatan | Org3's independent read-only auditor peer, live-proven (`REC-7`'s 3-verifier-class test) |
| 6 | Pasal 31/52 (rekam kegiatan) | Comply sebagian | Append-only ledger + `PrevHash` chaining + `Version`/`Timestamp`/`UpdatedBy`, live on every anchored write this whole session |
| 7 | Pasal 47/16(2)(h) (akuntabilitas terbukti) | **Comply** ✅ (strongest row) | Client-side verification independent of Org1's good faith (`REC-7`) — live-proven from 3 independent org identities |
| 8 | Pasal 36/52 (kerahasiaan) | **Comply** ✅ | See §5 below — this build phase's `ST-1` full-ledger scan is the FIRST real test evidence for this row (previously argument-only per `prd.md` §3.2); **one real caveat now attached, not a status downgrade — see §5** |
| 9 | Pasal 4(2)(f)/35(b) (data finansial) | Comply dengan catatan | PAYROLL section + IPFS documents encrypted, CID-only on-chain (`REC-5`) |
| 10 | Pasal 34 (DPIA) | **Tidak dapat diklaim** ⛔ | No DPIA exists. Unaffected by this build phase — tracked separately as grounding gap `G-25`. **Must be stated as a limitation, not omitted.** |
| 11 | Pasal 42/21 (retensi otomatis) | **Tidak dapat diklaim** ⛔ | Crypto-shred is request-triggered, not automatic time/purpose-triggered retention. Unaffected by this build phase — tracked separately as grounding gap `G-26`. **Must be stated as a limitation, not omitted.** |

**Distribution: 2 Comply, 4 Comply sebagian, 3 Comply dengan catatan, 2 Tidak dapat diklaim — 11
rows, none silently rounded up.**

**Verdict: PASS** (as a reporting predicate — the matrix is carried forward at its real, tiered
status, both "Tidak dapat diklaim" rows are named as open limitations, not hidden).

---

## 5. The question no predicate tests — confidentiality (`prd.md` §3.2, `OQ-1`)

`prd.md` §3.2 states plainly that none of P1–P4 test confidentiality, and calls this "the most
important open issue in this document." `ST-1` (`QA-5`, P0 priority in `test-strategy.md`) is the
real test evidence this section says did not yet exist. It now does, and the honest answer has two
parts:

**The good part, triple-independently-confirmed:** `fabric-network/tools/pilscan` parsed every raw
block across both tenant channels (232 blocks, 36,489 transaction arguments, 3,628 world-state
writes, 3,622 events) and found **zero actual PII** — a synthetic-fixture-string dictionary check,
a raw-block email/SSN/name-pattern backstop, and the verifier's own separate independent pattern
search all returned zero hits.

**The real gap:** the same scan found 31,895 identifier-SHAPE violations — `employeeID`/`updatedBy`
values recorded as raw, human-readable test-fixture strings instead of proper 64-hex HMAC-SHA256
pseudonyms, because several test suites across this build phase called `RecordProfileSection`
directly with hand-rolled literals, bypassing the real `writepaths.Hooks.anchor()` pseudonymization
pipeline — and **the chaincode itself performs no shape validation and would have accepted a real
name string exactly the same way.** Tracked as new grounding gap `G-31`, disclosed at
`record_profile_section.go` itself.

**How this bears on row 8 of the compliance matrix:** row 8's `Comply` status is about REAL PII
confidentiality, which held. But the row's supporting claim — "tidak ada plaintext/turunan
reversibel di ledger" — currently rests on **disciplined caller behavior**, not a guarantee the
chaincode enforces itself. This is worth stating as an explicit caveat alongside row 8 in the
thesis, not a reason to downgrade its status: what was tested passed; what was *not* tested (a
caller that fails to pseudonymize) is now a *named*, *disclosed* gap rather than an unstated
assumption.

---

## 6. Explicit scope boundary — what this report does not close

- **`ST-2`/`PB-1`** (API-gateway identity-header trust) remains genuinely open and unverifiable
  from within this workspace — it concerns the real external HRIS application's own public HTTP
  edge, out of scope by standing decision.
- **`ST-3`** (cert revocation) surfaced its own real findings (`G-32`) but could not be executed —
  not resolved here.
- **`prd.md` §10.6's errata `E-4`/`E-9`** (the thesis's own BAB III/BAB IV methodology-reporting
  gaps — RQ1 instrument never reported; Phase 4 has no results section yet) are **writing tasks for
  the thesis document itself**, not something this technical evaluation report closes. This report
  and `QA8-thesis-numbers-report.md` are the evidence a Phase-4 results section would cite; writing
  that section is the user's own next step.
- **FGD (qualitative evaluation, `prd.md` §10.3 jalur 5)** is out of scope entirely for this
  technical build phase — a different research instrument this repo's tooling cannot produce.

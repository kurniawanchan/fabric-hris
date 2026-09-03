# Reconciliation notes — COMBINED (2026-08-13) vs SOURCE [CORE] (2026-08-02)

Scope: COMBINED's §3.A, §4, §5, §6, §7, §8 (the [CORE]-tagged content) checked against the
founding PRD in full. Findings only — no rewrite proposed here.

## 1. Missing business feature — bulk/mass-operation batching

SOURCE (NFR-10): "Operasi massal (mis. penyesuaian gaji tahunan ribuan karyawan) ditangani dengan
*batching*." This is a real operational capability (annual mass salary adjustment across
thousands of employees) tied directly to a tesis reference (4.3.1). COMBINED's §3.A has no
mention of batch/bulk-operation handling anywhere, nor does §5's NFR table carry an NFR-equivalent
row. Not under-detailed — completely absent.

## 2. Missing constraint — reproducibility of the benchmark environment

SOURCE (NFR-11): the benchmark environment must be rebuildable and re-runnable so the results are
independently checkable by the examiner ("dapat dibangun ulang dan dijalankan ulang..."). COMBINED
has no corresponding NFR row (§5) and no mention in §6/§8, even though §7 discusses benchmark
representativeness (dev-laptop caveat) — the *reproducibility* requirement itself is a different,
dropped point.

## 3. Missing/altered NFR — dampak-minimal-latency threshold and its provisional status

SOURCE (NFR-9): a specific **<500ms added latency** threshold on the HRIS write path, explicitly
flagged as "ambang awal — kalibrasi ulang setelah Caliper §10.4 butir 3 berjalan" (a provisional
number, pending recalibration once the Caliper harness runs). COMBINED's NFR-A only says
"Structurally true (fully async) — not independently benchmarked," dropping both the concrete
number and the "provisional, to be recalibrated" framing entirely. The synthesis reads as if no
number was ever proposed, when SOURCE proposed one and explicitly flagged it as unverified.

## 4. Missing NFR — determinism/canonicalization (NFR-7) and its live blocking-gate status

SOURCE (NFR-7, tied to INV-6 and gate PB-3/gap G-24): JSON canonicalization must produce
byte-identical output between writer and verifier — and Lampiran A lists **PB-3 as still an open,
blocking pre-build gate** ("⛔ Masih terbuka — kanonikalisasi RFC-8785 JCS"). COMBINED's §3.A does
mention JCS/RFC-8785 as a design fact ("RFC-8785 JSON Canonicalization... pinned") but nowhere
states this is still an *open, unresolved* gate item — §6 (constraints) and §8 (out of scope) are
silent on it. A reader of COMBINED would reasonably assume JCS pinning is settled; SOURCE says it
is not.

## 5. Missing constraint — PB-1 gateway API trust-boundary gap, also still open

SOURCE's Lampiran A lists **PB-1/G-18 as still open** ("⛔ Masih terbuka — batas kepercayaan header
*gateway* API") alongside PB-3, and explicitly states these are the only two build-blocking items
remaining ("Gate blocking pre-build: dari 5 butir menjadi 2"). COMBINED never mentions PB-1 in any
section (§3.A, §6, §7, §8) — a live, named, still-unresolved blocking item from SOURCE's own
authoritative tracking table is simply absent from the synthesis.

## 6. Qualitative nuance lost — "proves integrity, not accuracy"

SOURCE §9.6 explicitly states a boundary the whole compliance/security narrative depends on:
"Sistem membuktikan integritas, bukan akurasi" — hash verification proves data hasn't changed
since it was recorded, not that the recorded data was ever correct; wrong data recorded from the
start still passes verification cleanly. COMBINED's §3.A/§4 describe the integrity guarantee
(P1, "every manipulation detected") without ever stating this limit. This is exactly the kind of
caveat a synthesis silently drops — it reads as a stronger guarantee than SOURCE claims.

## 7. Qualitative nuance lost — running the system increases regulatory exposure

SOURCE §9.4 discloses a self-aware paradox: Penjelasan Pasal 46(1) defines a "failure of personal
data protection" to include integrity failures and unauthorized changes — so every successful
manipulation-detection by this system is itself a trigger for the 3×24-hour breach-notification
obligation. SOURCE states plainly: running this in production *raises* compliance exposure, not
lowers it, and the author should disclose this rather than have an examiner find it. COMBINED's
§7 (Known Limitations) and §3.A's compliance bullet mention only the two "cannot be claimed" rows
(DPIA, retention) — this distinct, sharper self-critical point about detection-triggering-
disclosure is completely absent.

## 8. Qualitative nuance lost — crypto-shredding's legal basis is an untested argument

SOURCE §9.4 draws a specific distinction: PDP Law text has zero occurrences of "anonymization" /
"pseudonymization" / "encryption" as defined safe-harbor concepts (unlike GDPR Art. 4(5)/Recital
26), so the claim that crypto-shredding satisfies "memusnahkan" (destroy, Art. 44 — outcome-based
definition, stronger fit) versus "menghapus" (delete, Art. 43 — undefined, weaker fit) is an
**untested legal argument**, not textual compliance. COMBINED's §3.A bullet on regulatory
demonstrability and §7's limitations list state the matrix is "honesty-graded" and name the two
"cannot be claimed" rows, but never surface that even several of the *compliant* rows rest on
unproven legal argument rather than settled law. This flattens "argued, not yet legally settled"
into what reads like a completed compliance story.

## 9. Qualitative nuance lost — ledger replication to the auditor org may itself be a regulated "transfer"

SOURCE §9.4: Penjelasan Pasal 16(1)(e) defines "transfer" to include copying data to another
party — meaning replicating the ledger to Org3 (auditor) is literally a transfer under that
definition, and the processing legal basis (Pasal 20(2)) for that flow "belum dinyatakan" (not yet
stated). This is a concrete, specific open compliance question distinct from the two "cannot be
claimed" DPIA/retention rows. COMBINED does not mention it anywhere in §3.A, §6, §7, or §8.

## 10. Success metric context dropped — P1's "100% across all five sections" had an untested section as of SOURCE

SOURCE §3 flags with a warning that the thesis's Table 4.4 only tested hash-mismatch detection on
**four** of five sections (PAYROLL/PERSONAL/EMPLOYMENT/EDUCATION); the ADDITIONAL section "belum
pernah dimanipulasi dalam pengujian apa pun" despite the abstract's "100% on five sections" claim —
and states an additional test scenario is required before P1 as ratified is actually proven.
COMBINED's §4 states P1's threshold flatly as "100% across all five sections" with no caveat that,
per SOURCE, this figure was contingent on a test that had not yet been run. If that gap is still
open, COMBINED overstates P1's evidentiary status; if it has since closed, COMBINED should say so
rather than imply the number was always fully substantiated.

## 11. Constraint/decision dropped — the founding thesis's placeholder Caliper numbers and their governing ADR

SOURCE §3.1 states forcefully that the thesis's published benchmark numbers (850.4/1950.5 TPS,
2.85s latency) are **placeholders, not measured results**, and that ADR-0018 governs that "no
artifact may cite these numbers." COMBINED's §7 (Known Limitations) discusses a *different*
disclaimer — dev-laptop, non-representative numbers from [INT]/[DASH] work — but never mentions
that [CORE]'s own founding thesis numbers were placeholders subject to a specific ADR prohibition
on citation. Given [DASH] exists specifically to surface real Caliper numbers, losing this
lineage (why the placeholder numbers must never be cited) is a relevant dropped constraint for
readers who might otherwise pull a number from the original thesis.

## 12. Minor — Non-goals list omits "not logging user activity" framing nuance

Carried correctly in COMBINED §8 ("logging user *activity*") — checked, no issue. (Included here
to record that this specific non-goal was verified faithfully reproduced, not silently dropped.)

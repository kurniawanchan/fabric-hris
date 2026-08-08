<!--
Immutable once Accepted. Supersede with a new ADR; do not edit in place.
-->

# ADR-0018: Performance evaluation is in scope; Hyperledger Caliper v0.5 is the harness

- **Status:** **Accepted**
- **Date:** 2026-08-06
- **Deciders:** architect (gap in the 2026-08-06 agent dispatch — no specialist agent was assigned
  this ADR; authored directly to close the omission before it became a silent hole in Gelombang 3)
- **Rests on assumption(s):** none — grounded directly in a human decision, not an `[ASSUMPTION]`.
  `[prd: §3.1]`, `[prd: §10]`.
- **Supersedes:** none (amends the performance-scope clause of `context/DSRM.md §4`, which is prose
  guidance, not an ADR, and has already been corrected by `dsrm-researcher` in the same pass that
  authored `dsrm-phase-artifact-map.md`)
- **Superseded-by:** none
- **Ratifying authority / date:** PRD `prd-fabric-hris-2026-08-02/prd.md` §3.1 ("Yang diratifikasi
  sebagai kriteria sukses adalah ambang P2… Konsekuensi metodologis: … Gap **G-08 dibuka kembali**")
  — human decision (Chandra Kurniawan), ratified 2026-08-02.

## Context

The original G6 security architecture deferred performance-oriented evaluation behind gap **G-08**,
on the reasoning that a *security* artifact is argued and tested, not benchmarked, and performance
was a separate, later concern. That reasoning held **only while the package was design-only**. Once
the thesis (the ratifying source, per the 2026-08-02 "thesis wins" decision) was adopted, its own
success criteria explicitly include a measured performance threshold — PRD **P2**: write latency
**< 3 seconds at 500 TPS offered load** — and the thesis's Tabel 4.3/Abstrak/BAB V figures (850.4 TPS
write, 1,950.5 TPS read, 2.85s latency) are **placeholders**, not measured results (PRD §3.1). A
placeholder cannot be replaced by a real number without actually running a benchmark. Performance
evaluation is therefore back in scope, not as a nice-to-have but as the **only way P2 can be
evidenced** — and per PRD §10.4/§10.5, this evaluation cannot execute until the **G8a/G8b split**
(sponsor decision S-1) lifts the "0 lines of prototype code" constraint for the measured-evaluation
lane.

A ready-made harness already exists in-repo — `fabric-skill-suite/skills/fabric-performance/` —
authored for the unrelated `fabric-skill-suite` deliverable, but directly reusable here. Two concrete
defects in it are already known and must be fixed before it is trustworthy for this prototype:

1. `scripts/run-caliper.sh` line 26 hardcodes `SUT_BIND="${SUT_BIND:-fabric:2.2}"` with an inline
   `[VERIFY]` tag claiming "fabric connector covers 2.x incl. 2.5" — **unverified**, and this
   prototype targets Fabric **2.5** (INV/§5.1). Binding to the wrong connector version silently
   invalidates every number the harness produces.
2. `assets/caliper-workload-write.js` already implements a `hotKeyFraction` round argument
   (0 = disjoint keys, conflict-free; >0 = a shared-key fraction to induce MVCC contention) — this
   means the **MVCC-contention run this prototype needs is a configuration change, not new code**,
   which matters for the "no code before the gate lifts" accounting.

## Decision

We will treat performance evaluation as **in scope**, gated on **S-1 (G8a/G8b split)**, and reuse the
existing Hyperledger Caliper v0.5 harness with two required fixes before any number is trusted:

1. **Fix `SUT_BIND`** to the Fabric 2.5 connector (remove the `2.2` default and the unverified
   `[VERIFY]` tag once confirmed against the connector's own compatibility statement — this is a
   `fabric-performance`/`sre` build task, not resolved by this ADR).
2. **Reuse `hotKeyFraction`** for the MVCC-contention load point rather than writing a new workload
   module.
3. **Load points** (per PRD §3, Tabel 4.3 lineage): write sweep at 200 / 500 / 1,000 / 2,000 TPS
   offered load (500 is the **pass/fail gate for P2**; the others are the saturation curve, informational
   only); read (`GetStoredDigest`/history/summary reads under ADR-0020) sweep at 500 / 2,000 TPS.
4. **Test bed:** three nodes, 4 vCPU / 16 GB RAM each (per the thesis's own bench description,
   generalized — no cloud-vendor name), LevelDB state database (ADR-0007, unaffected by this ADR).
5. **Binding pass rule:** P2 is satisfied **only** by the write-sweep result at the 500 TPS load
   point; the 1,000/2,000 points and the read sweep are reported for completeness but do not gate
   P2.
6. **No figure from Tabel 4.3, the Abstrak, or BAB V 5.1 of the thesis may be cited by any artifact
   in this repo as a measured result** until this harness produces it — this includes ADRs, the
   architecture docs, and `dsrm-phase-artifact-map.md`. Every reference to those figures elsewhere in
   this repo must carry the "placeholder, not measured" caveat PRD §3.1 already states.

## Alternatives considered

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. Reuse `fabric-performance`'s existing Caliper harness with the two fixes above (chosen)** | Harness already exists, already implements the exact `hotKeyFraction` contention pattern needed; avoids writing a second benchmarking tool for the same platform | Harness was authored for a different deliverable (`fabric-skill-suite`) and needs the `SUT_BIND` fix verified against Fabric 2.5 before trust | **Chosen** |
| B. Write a bespoke benchmark harness for this prototype specifically | Full control, no cross-deliverable coupling | Duplicates `hotKeyFraction`'s contention logic and the whole Caliper wiring for no benefit; violates the project's own "cross-references, not duplication" convention | Rejected |
| C. Leave performance out of scope, keep the thesis's placeholder figures as-is, drop P2 | Zero build cost | Retracts a target the thesis's own Tabel 4.3 already meets on its face (485.2 TPS / 1.12s at the 500 TPS point, per PRD §3.1) and forces edits to the Abstrak/RQ3/Objectives/Conclusion to remove a claim that costs nothing to keep once measured for real | Rejected — PRD §3.1 already rejected this path explicitly |

## Consequences

- **Positive:** P2 becomes evidenced by a real, reproducible number instead of a placeholder; the
  MVCC-contention scenario (a real operational risk under channel-per-tenant fan-out per ADR-0013)
  is measurable with zero new code, only configuration.
- **Negative / trade-off:** this ADR cannot be executed until S-1 (G8a/G8b split) is decided by the
  sponsor — it formalizes the *plan*, not permission to run it yet. The harness fix (`SUT_BIND`) is
  itself an unverified `[VERIFY]` claim being carried forward, not resolved here.
- **Follow-ups:** `fabric-performance`/`sre` to verify and fix `SUT_BIND`; `qa` to own the actual
  benchmark run (QA-5 in `implementation-backlog.md`, Gelombang 4 #29) and the write-back of measured
  figures into the thesis (QA-8) once measured. Append a risk-register row for "harness authored for
  an unrelated deliverable, reused here without an independent audit of its Fabric-2.5 compatibility."

## Related

- Supersedes / amends: the performance-out-of-scope prose in `context/DSRM.md §4` (already corrected
  by `dsrm-researcher`, 2026-08-06 — this ADR is the formal decision record behind that prose fix,
  authored after the fact to close the gap left by the original agent dispatch not assigning it).
- Relates to: **ADR-0007** (LevelDB — the state-DB choice this benchmark measures against, unaffected);
  the pending write-path ADR by `fabric-engineer` (ADR-0020) whose Evaluate-only read design this
  harness's read sweep exercises.
- Knowledge-graph: gap **G-08** (reopened, PRD §3.1) — this ADR is its closing decision record, pending
  the S-1 gate for execution. Context doc(s): `fabric-skill-suite/skills/fabric-performance/`.

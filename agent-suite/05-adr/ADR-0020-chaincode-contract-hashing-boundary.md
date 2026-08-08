# ADR-0020: Chaincode contract surface and the off-chain hashing boundary

- **Status:** **Accepted**
- **Date:** 2026-08-06
- **Deciders:** fabric-engineer
- **Rests on assumption(s):** None for the boundary rule itself — ratified by `[prd: §5.2]` and
  `[prd: §7 Kelompok B′]` (FR-34..FR-37), human decision 2026-08-02. The concrete
  `canonicalizationVersion` string/library binding remains open — **G-24 / PB-3** (unchanged by
  this ADR). Exact endorsing-org MSP names/policy string are `fabric-architect`'s to ratify in
  ADR-0012 — this ADR states the **mechanism and shape**, not the topology decision.
- **Supersedes:** none (new)
- **Superseded-by:** none
- **Ratifying authority / date:** `prd-fabric-hris-2026-08-02/prd.md` §5.2 and §7 Kelompok B′
  (FR-34..FR-37), human decision 2026-08-02; flagged as the single most consequential correction
  in `rencana-rekonsiliasi.md` Gelombang 3 #16 ("ITEM PALING BERBAHAYA").

## Context

The source thesis (BAB IV 4.2.3) specifies a chaincode design in which the verification function
receives the section's plaintext data as a transaction argument and computes/compares the digest
**inside chaincode**: `VerifyProfileIntegrity(sectionData)`. Read literally, and combined with the
three-organization topology this package's PRD ratifies (`[prd: §5.1]` — platform org, an
enterprise-client-operated org, an auditor org), this design has two compounding defects that
`rencana-rekonsiliasi.md` names as the package's single most dangerous unresolved divergence
(Gelombang 3 #16):

1. **Plaintext becomes a transaction argument**, submitted or evaluated. A Fabric transaction
   proposal is simulated on every peer the endorsement policy names, and an Evaluate query executes
   on whichever peer answers it — in both cases the section's PII crosses a network boundary to a
   peer process outside the caller's own control. Under the ratified topology this means an
   enterprise-client peer or an auditor peer receives the platform's employee PII as a matter of
   routine verification traffic. This is a direct violation of **INV-1** ("no PII plaintext, nor
   any reversible derivative, ever on-chain") and **FR-34** (`[prd: §7]`), on **both** the write
   path and the verify path.
2. **It inverts the objective the whole prototype exists to demonstrate.** The ratified objective
   (`[prd: §2.1]`) is that a client verifies integrity **without having to trust the platform
   vendor**. A verify call that requires sending the plaintext value **to** a peer (so that peer can
   hash it) requires the verifier to trust that peer not to retain, log, or leak what it was just
   handed — the precise trust the design is supposed to remove. `[prd: §7]` names this "Cacat 1" of
   the pre-revision Kelompok B design.

`[prd: §7 Kelompok B′]` (FR-34..FR-37) is the ratified correction, and this ADR's job is to
transcribe it into a concrete four-function chaincode contract and an explicit boundary rule —
not to re-derive it.

## Decision

We will contract **four** chaincode transaction functions, under one boundary rule that binds all
four without exception:

> **Boundary rule (FR-34).** No chaincode function — on **Submit** or **Evaluate** — ever accepts
> a profile-section field value, or any other plaintext PII, as an argument or returns a *salt* in
> any response. A digest may be **compared to another digest** inside chaincode (byte comparison,
> e.g. the duplicate-content check below) without violating this rule, because no plaintext is
> read to do so — only the boundary against *content* is absolute.

### 1. `RecordProfileSection` — Submit (write)

| | |
|---|---|
| **Args** | `tenantId`, `employeeID` (pre-computed `EmployeeID`, ADR-0011 — chaincode never computes this from a raw internal ID), `profileSection` (enum: `PERSONAL`\|`EMPLOYMENT`\|`EDUCATION`\|`ADDITIONAL`\|`PAYROLL`), `dataHash` (pre-computed off-chain, ADR-0011), `prevHash` (hex, or empty iff no prior record exists for this key), `updatedBy` (pre-computed, ADR-0011), `ipfsCIDs` (`[]string`, optional), `canonicalizationVersion`, `hashAlgo` |
| **Returns** | `{RecordID, Version, Timestamp}` — echo, non-PII |
| **Endorsement** | Chaincode-level policy requiring the platform org **and** the enterprise-client org to co-sign — no single org, including the platform operator, can write an anchor alone. Exact MSP identifiers are ADR-0012's (`fabric-architect`); this ADR only fixes the **shape**: two independent signers, the auditor org excluded from the write set (consistent with its read-only role, FR-23). |
| **Rejects when** | `profileSection` is not one of the five enum values (FR-8); MSP/CID identity verification fails (FR-6); `prevHash` does not equal the current head's `dataHash` for `(employeeID, profileSection)` (chain-sequencing check — a **hash-to-hash** comparison, not a content read); `dataHash` is byte-identical to the current head's `dataHash` — a no-op rather than a new record (FR-9, dedupe). |
| **Never receives** | The section's field values, in whole or in part; the salt used to build `dataHash`; a raw (non-pseudonymized) employee or actor identifier. |

`RecordID` is assigned server-side as a UUID v4 (FR-7); `Version` increments monotonically per
`(employeeID, profileSection)`; `Timestamp` is the **ledger transaction timestamp** — never a
client-supplied value — matching the pattern already fixed by the (superseded) prototype design.
A successful commit **should** emit a chaincode event carrying the same non-PII fields, so
downstream audit projections can react on commit `[docs: gateway.md#listening-for-events]` — this
is a recommendation, not a numeric constraint, and does not change the argument/return contract
above.

### 2. Read-only integrity read — Evaluate (replaces `VerifyProfileIntegrity`)

The thesis's `VerifyProfileIntegrity(sectionData)` is retired in shape and in name. Its
replacement — named `GetProfileSectionRecord` here as an illustrative, non-binding name; the
implementer may choose another as long as the shape below holds — is a **pure read**:

| | |
|---|---|
| **Args** | `tenantId`, `employeeID`, `profileSection`. **Nothing else** — no section data, no candidate value, no salt (FR-34, FR-35). |
| **Returns** | Exactly `{DataHash, Version, Timestamp, UpdatedBy}` as stored. **Not found** is a distinct, typed outcome from any other result (FR-15) — chaincode has no candidate value to mismatch against, so it can only ever report "a record exists with these stored fields" or "no record exists for this key"; it never reports "mismatch," because it never received anything to mismatch. |
| **Recomputation / comparison** | **None, inside chaincode.** The caller recomputes `SHA-256(salt ‖ JCS(currentSectionValue))` **locally**, from the off-chain section value it already holds plus the salt obtained via the separate channel below, and compares the two digests itself (FR-35 — this is the entire "verification" step; chaincode's role ends at returning the stored record). |
| **Endorsement** | **Evaluate** — single-peer query, no ordering, no cross-org endorsement required for the call itself `[docs: gateway.md#fabric-gateway]`. |
| **Who calls it, against which peer** | The enterprise-client org and the auditor org each run this against **their own** peer (FR-37) — never the platform org's peer for a verification they intend to rely on — so no org needs to trust another org's peer to answer honestly. |
| **Salt handoff (FR-36)** | Explicitly **not** part of this function's request or response. A separate, audited channel — owned by `security-architect`/`backend-engineer`, out of scope for this ADR — hands the salt to whoever already holds the plaintext lawfully (the employee over their own record; an auditor within the scope of their audit). This ADR fixes only the negative constraint: the salt never appears in any chaincode argument or return value, on Submit or Evaluate, under any function. |

### 3. `GetProfileHistory` — Evaluate (chronological audit trail)

| | |
|---|---|
| **Args** | `tenantId`, `employeeID`, `profileSection` |
| **Returns** | The chronological list of every version recorded for `(employeeID, profileSection)`: `[{RecordID, Version, DataHash, PrevHash, UpdatedBy, Timestamp}, …]` — sourced from the ledger's own per-key history (`GetHistoryForKey`, `[docs: Fabric-FAQ.rst]`), not a chaincode-maintained secondary index (data-model.md §5 explains why the world-state key design makes this possible without a separate range index). |
| **"Last valid version"** | This function reports **what was recorded and when** (FR-16). It does **not** — and per the boundary rule, **cannot** — adjudicate which past version was "valid," because chaincode never held a value to validate against any of them. A caller that has independently run function 2 above against successive versions and recorded the outcomes may use *that* history to identify the last version it itself verified as matching (FR-17); this ADR does not claim chaincode performs that adjudication. |
| **Never receives/returns** | Section values or salts, at any point in the history. |

### 4. `GetEmployeeProfileSummary` — Evaluate (five-section fan-out)

| | |
|---|---|
| **Args** | `tenantId`, `employeeID` |
| **Returns** | The current head `{ProfileSection, DataHash, Version, Timestamp, UpdatedBy}` for **all five** sections in one call. |
| **Implementation choice** | Whether this is one `Evaluate` call performing five internal `GetState` reads, or a client-side fan-out of five separate `Evaluate` calls, has no confidentiality or correctness consequence either way and is left to the implementer post-G9. **Recommendation** (not a numeric constraint of this ADR): prefer the single-call form — **1** round trip instead of **5** — for interactive audit use (NFR-2). |

### Canonicalization as a versioned interface

`canonicalizationVersion` (e.g. an illustrative `"JCS-RFC8785-v1"`) is a first-class stored field,
not an implicit convention, so that a future change to the canonicalization library — or a future
migration away from JCS entirely — does not retroactively break the verifiability of records
already on the ledger: each record keeps the tag it was written under, and a verifier reads that
tag to select the matching recompute implementation. `hashAlgo` is versioned the same way. This
satisfies NFR-7/INV-6's "one deterministic scheme, identical between writer and verifier" **per
version**, without requiring the scheme to be frozen forever. The concrete v1 library/version
string to bind `"JCS-RFC8785-v1"` to remains the open build precondition **PB-3/G-24** — this ADR
fixes the *mechanism* for versioning, not the v1 pin itself.

## Alternatives considered

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. Off-chain digest; four-function contract; Evaluate-only, argument-free verify (chosen)** | No function, on Submit or Evaluate, ever receives a plaintext value; verification needs no trust in another org's peer (FR-37 holds literally); canonicalization-as-versioned-interface future-proofs already-anchored records | Chaincode can no longer independently attest that a digest matches "the real" section content — the entire integrity guarantee now rests on the caller computing `DataHash` correctly and honestly before Submit (see Consequences) | **Chosen** |
| B. Thesis-literal: `VerifyProfileIntegrity(sectionData)` computes and compares inside chaincode (BAB IV 4.2.3) | Matches the literal thesis text; chaincode can attest to the match itself | **Rejected outright** — makes PII plaintext a transaction argument simulated/executed on every peer the call reaches, including enterprise-client and auditor peers under the ratified topology; requires a verifier to send its plaintext to a peer it is trying not to trust, inverting the ratified objective (`[prd: §7]` "Cacat 1"); this is the one thesis chaincode design this package's PRD explicitly overrides despite the "thesis wins" default | Rejected |
| C. Chaincode computes `DataHash` itself at `RecordProfileSection` time, from a section-plaintext write argument (write-side-only variant of B) | Keeps verify off-chain-only while letting chaincode attest the write | **Rejected** — identical defect on the write side: the plaintext must still cross into a transaction proposal simulated on every endorsing peer (platform **and** enterprise-client org under the AND policy) and is logged/ordered into the block — exactly the INV-1 violation `rencana-rekonsiliasi.md` Gelombang 3 #16 calls out as the "koreksi wajib" | Rejected |
| D. Keep the digest off-chain (as chosen) but allow verify to accept plaintext on **Evaluate only** (never Submit), reasoning that Evaluate results are not ordered onto the ledger | Avoids ever writing plaintext to the immutable log | **Rejected** — "not written to the ledger" is not the same guarantee as "never disclosed to a peer outside your control": an Evaluate proposal is still a real message executed on a live peer process, visible to whatever operates or instruments that peer `[docs: gateway.md#fabric-gateway]`; still requires trusting that peer to discard the plaintext honestly — exactly the trust FR-37 exists to remove | Rejected |
| E. Narrow Private Data Collection (PDC) for the section value; only its hash on the public ledger (the pre-reconciliation package's ADR-0001 fallback pattern) | Native Fabric mechanism for confidential field exchange between named orgs | **Rejected for this contract** — PDC/`transient`-for-bulk-PII is already retired by the STOP-LIST (`project-context.md`); more fundamentally it does not solve Cacat 1/2 either: a PDC write still requires the plaintext to reach the proposal's `transient` field and be disseminated peer-to-peer to every collection member (still platform + enterprise-client by construction) `[docs: private-data-arch.rst#how-to-pass-private-data-in-a-chaincode-proposal]`, and a PDC read still requires the reading org to be a collection member — it relabels where the plaintext travels without removing the need to trust a peer outside your own org. It is also measurably slower: an asset held in a PDC endorses at roughly half the throughput of an equivalent world-state asset `[docs: performance.md#private-data-collections-pdcs-vs-world-state]`, a cost paid for no confidentiality benefit this design still needs, since the value never becomes a proposal argument at all under Option A | Rejected |

## Consequences

- **Positive:** closes the divergence `rencana-rekonsiliasi.md` itself flags as the package's most
  dangerous open item; makes "verify without trusting the platform vendor" literally true rather
  than aspirational — the enterprise-client and auditor orgs each recompute against their own
  peer's stored record; the versioned canonicalization field means a future crypto-agility change
  does not retroactively strand already-anchored records.
- **Negative / trade-off:** chaincode enforces the **chain's structure** — sequencing via
  `prevHash`, per-section dedupe, MSP-verified identity, monotonic versioning — but not the
  **content correctness** of any single write, because it is never shown the content. A caller
  that computes and submits a false `DataHash` for real, honestly-written content produces a
  self-consistent but false chain entry, and only an independently-sourced, independently-trusted
  copy of the section value (outside this ADR's scope) could ever catch that at the chain-integrity
  layer — this is an accepted, explicitly-stated limitation, not an oversight, because every
  alternative that lets chaincode attest content correctness does so by making the content itself
  a transaction argument, which Options B/C above show is strictly worse. This should be recorded
  as a stated limitation in the thesis write-up and as a risk-register row ("honest-caller
  assumption at write time") for `security-architect`.
- **Follow-ups:** **PB-3/G-24** remains open — the concrete `canonicalizationVersion="JCS-RFC8785-v1"`
  binding must be pinned before build (joint with `backend-engineer`/`qa`); the salt hand-off
  channel (FR-36) needs its own design, owned by `security-architect`/`backend-engineer` — this
  ADR only constrains where the salt must *never* appear; the exact endorsing-org MSP identifiers
  and policy string are ADR-0012's to ratify (`fabric-architect`) — this ADR's "platform org /
  enterprise-client org / auditor org" language describes the ratified topology
  (`[prd: §5.1]`), it does not decide it.

## Related

- Relates to: **ADR-0011** (supplies `DataHash`/`EmployeeID`/`UpdatedBy`/`PrevHash`, computed
  off-chain, that this contract's functions consume as pre-built arguments); **ADR-0014** (the
  in-band write path that calls `RecordProfileSection`); the pending **ADR-0012**
  (`fabric-architect` — endorsing-org MSP identifiers/policy string).
- Knowledge-graph: Layer D capabilities (contract API, evaluate/submit semantics), traceability
  spine chains #1 (anchor) and #3 (access — Org2/Org3 reading their own peer). Grounding gap(s):
  **G-24/PB-3** (canonicalization pin, open). Context doc(s): `context/BLOCKCHAIN-DATA-MODEL.md`,
  `context/FABRIC-CHAINCODE.md`, `context/FABRIC-PRIVATE-DATA.md`. Ratifying source:
  `prd-fabric-hris-2026-08-02/prd.md` §5.2, §7 Kelompok B′; `rencana-rekonsiliasi.md` Gelombang 3 #16.

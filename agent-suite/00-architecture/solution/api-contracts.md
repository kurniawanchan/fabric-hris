# Chaincode Contract Interface Reference — `EmployeeProfileRecord`

- **Version:** v1 (design-only — no chaincode/application code before a re-approved G9) ·
  **Interface kind:** Fabric chaincode transaction functions, invoked via the Fabric Gateway
  (Submit/Evaluate) — **not** a REST API. This design has **no** standalone service surface: the
  in-band write path (ADR-0014) and any verifier-side application call the Gateway directly.
- **Author:** fabric-engineer · **Date:** 2026-08-06
- **Instantiates:** **ADR-0020** (the four-function contract and the off-chain hashing boundary),
  **ADR-0011** (the digest/identifier construction consumed by every function below), **ADR-0014**
  (who calls `RecordProfileSection`, and when).
- **Companion:** [`data-model.md`](data-model.md) (the `EmployeeProfileRecord` this contract
  reads/writes); [`integration-design.md`](integration-design.md) (the write and verify sequences);
  diagrams `00-architecture/diagrams/anchor-write-sequence.mmd`.
- **Conventions:** follows `api-documentation` reference style (error-first, realistic examples),
  adapted for a chaincode contract rather than REST. The blocks below are an **interface
  specification** — request/response **shapes** — not implementation code in any language; no
  block here is intended to compile or run.
- **Register note.** No product name, codebase, file, class, or table name appears in this
  document — see `project-context.md` AUTHORITY NOTICE.

---

## Overview

Four transaction functions, one contract, on the employee's tenant channel:

| Function | Invocation | Purpose | Ledger op |
|---|---|---|---|
| `RecordProfileSection` | **Submit** | Anchor one profile-section write, in-band from the HRIS write path. | Endorse + Order + Commit |
| `GetProfileSectionRecord` | **Evaluate** | Return the stored head record's hash-bearing fields — the read-only replacement for the thesis's `VerifyProfileIntegrity`. Recomputation and comparison happen **client-side** (§ Client-side verification protocol). | Query only, no ordering |
| `GetProfileHistory` | **Evaluate** | Return the full chronological version list for one employee's one section. | Query only, no ordering |
| `GetEmployeeProfileSummary` | **Evaluate** | Return the current head of all five sections in one call. | Query only, no ordering |

**Function names are illustrative.** ADR-0020 fixes the argument/return **shape** and the
Submit-vs-Evaluate **behavior** of each function; the exact identifier a future implementer binds
to each is not mandated by this design.

---

## The confidentiality boundary (ADR-0020) — binds every function below without exception

> **No function in this contract — Submit or Evaluate — ever accepts a profile-section field
> value, or any other plaintext PII, as an argument. No function ever returns a *salt*, in any
> response.** A stored digest may be compared to another stored digest **inside** chaincode (a
> byte comparison — see `RecordProfileSection`'s duplicate-content check) without violating this
> rule, because no plaintext is read to do so; only the boundary against *content* is absolute.

Concretely, across all four functions:

- Arguments are drawn **only** from: pseudonymous identifiers (`employeeID`, `updatedBy`), digests
  (`dataHash`, `prevHash`), the `profileSection` enum, `ipfsCIDs`, version/algorithm tags
  (`canonicalizationVersion`, `hashAlgo`), and `tenantId`.
- Return values are drawn **only** from the same set, plus ledger-assigned metadata
  (`recordID`, `version`, `timestamp`).
- A caller that attempts to pass a section field value, or a salt, in any argument gets a
  interface-level validation rejection before the value is ever read as a "value" by chaincode
  logic — because no parameter in any function's signature is typed to accept one (there is no
  `sectionData`/`values` parameter anywhere in this contract, unlike the thesis's literal
  `VerifyProfileIntegrity(sectionData)`).

---

## Conventions

- **Identifiers.** `employeeID` and `updatedBy` are the pseudonymous HMAC outputs defined by
  ADR-0011 — never a raw internal employee/user id, never returned as anything else.
- **Digest format.** `dataHash`/`prevHash` are hex-encoded SHA-256 output, illustratively prefixed
  `sha256:` for readability (a design convention, not a PRD-mandated format) — see `data-model.md`
  §2 for the on-chain shape.
- **Timestamps.** RFC 3339 UTC. The **authoritative** anchor time is the ledger transaction
  timestamp returned as `timestamp` in every function's response — a client-supplied timestamp,
  where one exists in a request, is informational only.
- **Versioning.** `canonicalizationVersion`/`hashAlgo` are per-record tags (ADR-0020) — a future
  scheme change adds a new tag value, it does not replace this contract's version.
- **Idempotency.** `RecordProfileSection` is a no-op (not an error, not a new record) when the
  submitted `dataHash` is byte-identical to the current head's `dataHash` for the same
  `(employeeID, profileSection)` (FR-9) — this is the contract's only replay-safety mechanism, and
  it needs no companion off-chain store to work, unlike the pre-reconciliation design's
  `approvalRef`-keyed idempotency probe.

---

## `RecordProfileSection` — Submit (write)

Anchors one profile section's current state. Called **in-band**, by the HRIS write path that just
committed that section to the operational database (ADR-0014) — never as a batch/replay endpoint
in this design, since there is no separate anchor-service to replay from.

### Request (transaction proposal arguments)

```
RecordProfileSection(
  tenantId:                 string,                 // tenant identifier — metadata only (data-model.md §5)
  employeeID:                string,                 // pre-computed off-chain: HMAC-SHA256(employeeKey_i, "id")
  profileSection:            enum { PERSONAL, EMPLOYMENT, EDUCATION, ADDITIONAL, PAYROLL },
  dataHash:                  string,                 // pre-computed off-chain: SHA-256(salt || JCS(section))
  prevHash:                  string,                 // DataHash of the current head for this key, or "" if none exists
  updatedBy:                 string,                 // pre-computed off-chain: HMAC-SHA256(employeeKey_i, "actor" || user_id)
  ipfsCIDs:                  string[],               // optional; CIDs of encrypted documents attached to THIS version
  canonicalizationVersion:   string,                 // e.g. "JCS-RFC8785-v1" — tag only, ADR-0020
  hashAlgo:                  string,                 // e.g. "SHA-256" — tag only, ADR-0020
  clientTimestamp:           timestamp?              // OPTIONAL, informational only — never authoritative
) -> RecordProfileSectionResult
```

| Field | Required | Constraint | Notes |
|---|---|---|---|
| `tenantId` | yes | non-empty | Metadata; tenant isolation is by channel membership, not this field. |
| `employeeID` | yes | opaque, HMAC output | Never a raw internal id — rejected if it looks like one (implementer's validation choice, not mandated by shape). |
| `profileSection` | yes | one of exactly 5 values | Any other value is rejected (FR-8). |
| `dataHash` | yes | digest string | Computed off-chain — chaincode never recomputes it from a value it does not have. |
| `prevHash` | yes | digest string or `""` | `""` is valid **only** if no prior record exists for this `(employeeID, profileSection)`; otherwise must equal the current head's `dataHash`. |
| `updatedBy` | yes | opaque, HMAC output | Pseudonymous actor, not the platform's own submitting service identity. |
| `ipfsCIDs` | no | array, possibly empty | Present only when a supporting document accompanies this version. |
| `canonicalizationVersion` | yes | version tag | Stored verbatim; not validated against a fixed enum by this contract (future versions are additive). |
| `hashAlgo` | yes | algorithm tag | Same versioning posture as above. |
| `clientTimestamp` | no | RFC 3339 | Never used as the stored `timestamp` — see Response. |

### Behavior (specification, not implementation)

1. Reject if `profileSection` is not one of the five enum values (FR-8).
2. Reject if the calling identity fails MSP/CID verification (FR-6) — `[docs: identity/identity.md]`.
3. Read the current head for `(employeeID, profileSection)`, if any.
4. If `dataHash` equals the current head's `dataHash` exactly → **no-op**: return the existing
   head's `RecordProfileSectionResult` unchanged, do **not** write a new version (FR-9).
5. Else, if `prevHash` does not equal the current head's `dataHash` (or the head does not exist and
   `prevHash` is non-empty) → reject as a **stale chain reference** — the caller's view of the
   chain is out of date; per `integration-design.md` §5.2, the caller re-reads the head and
   resubmits.
6. Else → assign a new `recordID` (UUID v4), increment `version`, write the record with the
   **ledger transaction timestamp** as the stored `timestamp` (never `clientTimestamp`).

### Response — `RecordProfileSectionResult`

```
{
  recordID:  string,    // UUID v4 assigned by this write (or the existing record's, on no-op replay)
  version:   integer,   // the version number just written (or the existing head's, on no-op replay)
  timestamp: timestamp  // the AUTHORITATIVE ledger commit time
}
```

No field of the submitted section value, and no salt, is echoed — there was none to echo (§ The
confidentiality boundary).

### Errors (specification-level; exact codes are an implementer's choice)

| Condition | Outcome |
|---|---|
| `profileSection` outside the five-value enum | Rejected — invalid argument. |
| Caller identity fails MSP verification | Rejected — not authorized (FR-6). |
| `prevHash` does not match the current head | Rejected — stale chain reference; caller must re-read and resubmit (not the same as a Fabric-level `MVCC_READ_CONFLICT`, which can additionally occur at commit time if another write raced between this call's simulation and its ordering — both cases resolve the same way: re-read, resubmit). |
| `dataHash` identical to the current head | **Not an error** — no-op, existing record's result returned (FR-9). |

### Endorsement

Chaincode-level policy requiring co-signature from the platform org **and** the enterprise-client
org; the auditor org is excluded from the write-endorsement set (consistent with its read-only
role, FR-23). Exact MSP identifiers are `fabric-architect`'s to fix (pending ADR-0012) — this
contract only requires that **no single org, including the platform operator, can satisfy this
policy alone.**

---

## `GetProfileSectionRecord` — Evaluate (read-only; replaces `VerifyProfileIntegrity`)

Returns exactly what is stored for one employee's one section — **nothing else**. All
recomputation and comparison happen on the caller's own side (§ Client-side verification protocol,
below) — this function does not receive a candidate value and therefore cannot, and does not,
report a "match"/"mismatch" itself.

### Request

```
GetProfileSectionRecord(
  tenantId:        string,
  employeeID:       string,
  profileSection:   enum { PERSONAL, EMPLOYMENT, EDUCATION, ADDITIONAL, PAYROLL }
) -> ProfileSectionHead | NotFound
```

No other parameter exists on this function — there is no `values`/`sectionData`/`salt` argument
anywhere in its signature (FR-34/FR-35).

### Response — `ProfileSectionHead`

```
{
  dataHash:  string,    // the stored digest — nothing to compare it to, on this side
  version:   integer,
  timestamp: timestamp, // the ledger commit time of this version
  updatedBy: string     // pseudonymous actor identifier
}
```

Exactly these four fields (FR-35) — no `recordID`, `prevHash`, `ipfsCIDs`, or version-tag fields
here; those are available via `GetProfileHistory` for a caller that needs them.

### `NotFound`

Returned, distinctly from any other outcome, when no record exists yet for
`(employeeID, profileSection)` — i.e. the section has never been anchored. This is a **different**
outcome from "found, but my local recompute doesn't match" — the latter can only ever be
determined by the caller, since chaincode never received a value to test (FR-15).

### Endorsement / access

**Evaluate** — single-peer query, no ordering, no cross-org endorsement required for the call
itself `[docs: gateway.md#fabric-gateway]`. An enterprise-client or auditor verifier calls this
against **their own** peer (FR-37) — never a peer they do not operate, for a verification they
intend to rely on.

---

## Client-side verification protocol (uses `GetProfileSectionRecord`, does not extend it)

This is the full replacement for the thesis's in-chaincode verify step (FR-34..FR-37) — it is a
**client-side procedure**, not an additional chaincode function:

1. Obtain the salt for the version being checked, via a separate, audited hand-off channel (FR-36)
   whose authorization follows data ownership — **not** via any chaincode call.
2. Obtain the current (or historical, via an operational-database snapshot) section value from a
   source the verifier already lawfully holds.
3. Canonicalize that value with the **same** `canonicalizationVersion` implementation the stored
   record was written under (read the tag from `GetProfileHistory`, or from context, if not
   already known).
4. Compute `recomputedHash = SHA-256(salt ‖ JCS(sectionValue))` — **locally**.
5. Call `GetProfileSectionRecord(tenantId, employeeID, profileSection)` against the verifier's own
   peer.
6. Compare `recomputedHash` to the returned `dataHash` — **locally**. `match = (recomputedHash ==
   dataHash)`.

No step above sends the section value, or the salt, to any peer. Step 5 is the **only** network
call to the ledger in the entire protocol, and it carries none of the sensitive inputs.

| | Thesis's literal design | This protocol |
|---|---|---|
| Section value crosses to a peer | Yes (as a Submit/Evaluate argument) | **Never** |
| Salt crosses to a peer | N/A (not modeled) | **Never** (separate channel, and never to a peer either) |
| Comparison performed by | Chaincode | **The verifier, locally** |
| Requires trusting a peer you don't operate | Yes | **No** (FR-37) |

---

## `GetProfileHistory` — Evaluate (chronological version list)

### Request

```
GetProfileHistory(
  tenantId:        string,
  employeeID:       string,
  profileSection:   enum { PERSONAL, EMPLOYMENT, EDUCATION, ADDITIONAL, PAYROLL }
) -> ProfileVersionEntry[]
```

### Response — `ProfileVersionEntry[]`, ascending chronological order

```
[
  {
    recordID:                 string,
    version:                  integer,
    dataHash:                 string,
    prevHash:                 string,
    updatedBy:                string,
    timestamp:                timestamp,
    ipfsCIDs:                 string[],
    canonicalizationVersion:  string,
    hashAlgo:                 string
  },
  …
]
```

Sourced from the ledger's own per-key history for the `(employeeID, profileSection)` composite key
(`data-model.md` §5), not a chaincode-maintained secondary index. An empty array means the section
has never been anchored — the same "not found" concept as `GetProfileSectionRecord`'s `NotFound`,
expressed as an empty list rather than a distinct error, since a history query naturally has a
"zero results" shape.

**This function does not, and cannot, mark any entry as "the last valid version."** A caller that
has independently run the verification protocol above against successive entries may track which
one it confirmed matching — that judgment lives entirely on the caller's side (`[prd: §12 OQ-6]`).

### Endorsement / access

**Evaluate** — same posture as `GetProfileSectionRecord`.

---

## `GetEmployeeProfileSummary` — Evaluate (five-section fan-out)

### Request

```
GetEmployeeProfileSummary(
  tenantId:   string,
  employeeID:  string
) -> ProfileSectionSummaryEntry[5]
```

### Response — `ProfileSectionSummaryEntry[]`, one entry per section

```
[
  { profileSection: "PERSONAL",   dataHash: "sha256:…", version: 3, timestamp: "…", updatedBy: "…" },
  { profileSection: "EMPLOYMENT", dataHash: "sha256:…", version: 1, timestamp: "…", updatedBy: "…" },
  { profileSection: "EDUCATION",  dataHash: "sha256:…", version: 2, timestamp: "…", updatedBy: "…" },
  { profileSection: "ADDITIONAL", dataHash: "sha256:…", version: 1, timestamp: "…", updatedBy: "…" },
  { profileSection: "PAYROLL",    dataHash: "sha256:…", version: 7, timestamp: "…", updatedBy: "…" }
]
```

A section that has never been anchored is represented by an entry with `version: 0` and an empty
`dataHash` (an implementer's choice of "not found" representation for a fixed five-element array;
alternatively, omit the entry — either shape satisfies FR-18, the exact representation is not
mandated).

### Endorsement / access

**Evaluate.** **Recommendation** (not a numeric constraint): implement as a single Evaluate call
performing five internal reads, rather than five separate Evaluate round trips — cheaper for
interactive audit use (NFR-2). Either implementation satisfies this contract's request/response
shape identically from the caller's perspective.

---

## Assumptions & gaps carried

| Gap | Bearing on this contract |
|---|---|
| **G-24 / PB-3** | `canonicalizationVersion` values are accepted as opaque tags by this contract; the concrete v1 binding is not pinned here. |
| **G-10 (reopened)** | Which HRIS process calls `RecordProfileSection` is not fixed here (ADR-0014); this contract is agnostic to the caller's identity beyond MSP verification. |
| Pending ADR-0012 | Exact endorsing-org MSP identifiers for the write policy (fabric-architect). |
| FR-36 (salt hand-off) | This contract fixes only that no function carries a salt; the hand-off channel itself is out of scope here. |

---

## What this document deliberately does not contain

Unlike the pre-reconciliation `api-contracts.md`, there is **no REST endpoint list, no
`X-Api-Key`/`X-Company-ID` header contract, and no OpenAPI document** here — this design has no
standalone anchor-service for such a surface to belong to (ADR-0014). If a future operator-facing
tool (an audit dashboard, a support console) is built on top of these four functions, its REST or
GraphQL shape is that tool's own design, layered on top of this contract — not part of it.

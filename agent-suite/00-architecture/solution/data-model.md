# Solution Data Model — the on-ledger `EmployeeProfileRecord` and its off-chain companions

- **Gate:** re-opened design (G8/G9 void per `[prd: §11.1]`) · **Author:** fabric-engineer · **Date:** 2026-08-06
- **Status:** Draft (design-only — no chaincode/application code before a re-approved G9)
- **Instantiates:** **ADR-0011** (per-section salted digest, two-tier keyed identifiers),
  **ADR-0014** (in-band recording), **ADR-0020** (chaincode contract + hashing boundary). Topology
  facts (org/MSP set, channel-per-tenant, state DB) are consumed, not decided, here — see
  `fabric-network-design.md` (fabric-architect, pending its own rewrite per
  `rencana-rekonsiliasi.md` Gelombang 4 #23).
- **Supersedes (content, not file):** the `AnchorRecord` shape and every section of this document
  as it stood under ADR-0001/0006 — no field name from that version survives unchanged.
- **Grounds in / cross-references (does not restate):** `[prd: §4, §5.2, §5a, §7]`
  (anchoring unit, cryptographic construction, glossary, functional requirements),
  `context/BLOCKCHAIN-DATA-MODEL.md`, `context/CRYPTOGRAPHY.md`, `context/FABRIC-PRIVATE-DATA.md`,
  `11-execution/knowledge-graph.md` (Layers B/C/D, spine chains 1/2/4),
  `11-execution/grounding-gaps.md`. Fabric mechanics live in the skill suite —
  `> See fabric-chaincode-dev/references/contract-api.md`,
  `> See fabric-performance/references/mvcc-and-read-write.md`.
- **Register note.** This document describes the HRIS write paths that produce each profile
  section only in generic terms (a commercial multi-tenant HRIS SaaS platform's employee-data
  write paths). It does not name the underlying product, codebase, or any literal
  file/class/table — see `project-context.md` AUTHORITY NOTICE, "Confidentiality framing."

---

## 1. Confidentiality invariant (unchanged in substance, restated for the new scheme)

> **INVARIANT (ADR-0011/ADR-0020).** Zero bytes of profile-section plaintext — and zero
> reversible derivative of it — are ever written on-chain, or ever appear as a chaincode
> transaction argument, **on Submit or on Evaluate**. The ledger holds only a per-record salted
> digest (`DataHash`), two HMAC-derived pseudonymous identifiers (`EmployeeID`, `UpdatedBy`), and
> non-PII structural metadata.

This is stricter than the pre-reconciliation invariant in one specific way that matters: it binds
**read** operations, not only writes. The thesis's literal chaincode design would have violated it
on the read side (`VerifyProfileIntegrity(sectionData)`) even though the write side alone might
have looked compliant — see ADR-0020's Alternatives B–D for why "not written to the ledger" is not
a sufficient guarantee.

- Digest algorithm: **SHA-256** (256-bit), identifier MAC: **HMAC-SHA256** (ADR-0011).
- Per-record content salt: **≥ 128 bits**, CSPRNG, unique per record, **off-chain only**.
- Profile-section plaintext as a chaincode argument, on Submit or Evaluate: **0 bytes**, always
  (ADR-0020).
- Salt as a chaincode argument or return value, on Submit or Evaluate: **0 bytes**, always
  (FR-14/FR-34/FR-36).

The authoritative section data stays in the platform's existing operational database, under its
existing at-rest encryption; Fabric never becomes source-of-truth for a field (unchanged posture
from the pre-reconciliation design's overlay principle — `[prd: §2.3]` non-goal "menyimpan data
profil itu sendiri di blockchain").

---

## 2. The `EmployeeProfileRecord` asset (public world state, per tenant channel)

One record per profile-section write. Written and read as plain world state — **no Private Data
Collection is used anywhere in this design** (§9 explains why explicitly, since the
pre-reconciliation design reserved one).

```jsonc
{
  "recordID":                "9f2c1a7b-4e0d-4f61-8a3c-1b2c3d4e5f60", // UUID v4 (FR-7)
  "employeeID":              "3a7e9c…",           // HMAC-SHA256(employeeKey_i, "id") — hex (ADR-0011)
  "tenantID":                "t_00842",            // tenant identifier; metadata only — isolation is by
                                                     // channel membership (ADR-0013, fabric-architect), not this field (§5)
  "profileSection":           "PAYROLL",            // one of 5 enum values (§8)
  "dataHash":                "sha256:9c1e2f…",     // SHA-256(salt ‖ JCS(section)) — computed OFF-CHAIN (ADR-0011)
  "prevHash":                "sha256:04d2ab…",     // DataHash of the PRIOR record for this (employeeID, profileSection); "" for the first
  "version":                 7,                     // monotonic per (employeeID, profileSection)
  "updatedBy":                "b81f3e…",            // HMAC-SHA256(employeeKey_i, "actor" ‖ user_id) — hex (ADR-0011)
  "timestamp":               "2026-08-06T04:11:22Z", // client-reported; AUTHORITATIVE time is the ledger tx timestamp
  "ipfsCIDs":                 ["bafy…d1", "bafy…d2"], // CIDs of encrypted supporting documents tied to THIS version (§10); [] if none
  "canonicalizationVersion": "JCS-RFC8785-v1",      // versioned canonicalization interface (ADR-0020) — v1 binding open, PB-3/G-24
  "hashAlgo":                "SHA-256"               // versioned digest-algorithm interface (ADR-0020)
}
```

Written with `PutState(compositeKey, marshaled)`, read with `GetState` /
`GetHistoryForKey` `[docs: Fabric-FAQ.rst]` (`> See fabric-chaincode-dev/references/contract-api.md`).
The key is **not** append-only-by-suffix (contrast the pre-reconciliation `changeSeq` design, §5) —
each write **overwrites** the current head for its `(employeeID, profileSection)` key, and the
version history is recovered from the ledger's own per-key history, not from a range of sibling
keys.

### 2.1 Field reference

| Field | Type | On-chain? | Description / grounding |
|---|---|---|---|
| `recordID` | string (UUID v4) | yes | Unique per write (FR-7). Not derivable from anything else — assigned at write time. |
| `employeeID` | string (hex) | yes | Pseudonymous, deterministic identifier — `HMAC-SHA256(employeeKey_i, "id")` (ADR-0011). **Never** a raw internal employee id. |
| `tenantID` | string | yes | Tenant identifier, metadata only (§5 explains why it is **not** part of the key). |
| `profileSection` | enum string | yes | `PERSONAL` \| `EMPLOYMENT` \| `EDUCATION` \| `ADDITIONAL` \| `PAYROLL` — validated against exactly these five values, nothing else (FR-8). |
| `dataHash` | string | yes | `SHA-256(salt ‖ JCS(section))`, computed **off-chain** by the HRIS write path before Submit (ADR-0011/ADR-0020). Non-reversible without the salt. |
| `prevHash` | string \| `""` | yes | Chain link to the prior `dataHash` for the **same** `(employeeID, profileSection)` — one chain per section per employee, five chains per employee total (`[prd: §4.2]`). |
| `version` | integer | yes | Monotonic per `(employeeID, profileSection)`; starts at `1`. |
| `updatedBy` | string (hex) | yes | Pseudonymous actor identifier — `HMAC-SHA256(employeeKey_i, "actor" ‖ user_id)` (ADR-0011). Not the platform's own submitting service identity — see §2.2. |
| `timestamp` | RFC 3339 UTC | yes | Client-reported event time; the **authoritative** time is the block/tx timestamp `[docs: ledger/ledger.md]`. |
| `ipfsCIDs` | string[] | yes | CIDs of encrypted supporting documents linked to this specific version (§10); empty array when none. Only the CID is on-chain — never document content (FR-31). |
| `canonicalizationVersion` | string | yes | Tag identifying which canonicalization implementation produced `dataHash`'s input — a versioned interface (ADR-0020), not a hard-coded assumption. |
| `hashAlgo` | string | yes | Tag identifying the digest algorithm — versioned the same way as `canonicalizationVersion`. |

### 2.2 `submittedBy` vs `updatedBy` — two different identities, only one on-chain

The **submitting Fabric client identity** (the platform's own X.509 identity used to call
`RecordProfileSection`) is read by chaincode via the CID API at endorsement time
(`GetID()`/`GetMSPID()`, `[docs: identity/identity.md]`) purely to enforce FR-6 ("reject if the
caller is not MSP-verified"). It is **not** stored as a field on `EmployeeProfileRecord` — the
record's `updatedBy` field is the pseudonymized **human actor** (the HR admin, employee, or system
user who made the change), derived exactly as `employeeID` is, from the same per-employee
`employeeKey_i` (ADR-0011). This mirrors the pre-reconciliation design's separation of "who
technically submitted the transaction" from "who is recorded as having made the change" without
carrying over any of its field names.

---

## 3. Identifier and digest construction — cross-reference, not restatement

The exact construction of `employeeID`, `updatedBy`, and `dataHash` — the independently-random
`employeeKey_i` (ADR-0021; no master key), the ≥128-bit per-record salt, and the RFC 8785 JCS
canonicalization requirement — is fixed by **ADR-0011/ADR-0021** and is not repeated here. This document
only fixes where each output lands in the asset shape (§2) and the world-state key (§5).

**What this data model adds beyond ADR-0011:** the `canonicalizationVersion`/`hashAlgo` fields
(§2.1) exist specifically so that this asset shape survives a future change to either without
invalidating already-anchored records' verifiability — a data-model consequence of ADR-0020's
"canonicalization as a versioned interface" decision.

---

## 4. Anchoring trigger — one write path per section, in-band

Per **ADR-0014**, each of the five sections is written by its own path inside the HRIS
application's existing write flow (a personal-data change-approval workflow for `PERSONAL`; a
transfer/mutation approval action for `EMPLOYMENT`; a formal/informal education-history write
action for `EDUCATION`; a family-data — marital status / dependents — change-approval workflow for
`ADDITIONAL`; a payroll/bank-account update action for `PAYROLL`). Each path, at the moment it
commits its section to the operational database, computes `dataHash`/`employeeID`/`updatedBy`
off-chain and calls `RecordProfileSection` (ADR-0020) as part of that same request. No message
queue or standalone consumer sits between the write and the anchor (ADR-0014) — the full
end-to-end sequence is `integration-design.md` §4.

---

## 5. World-state key strategy

```
key = CreateCompositeKey("profile", [employeeID, profileSection])
    → "profile" \x00 3a7e9c… \x00 PAYROLL \x00
```

One key holds the **current head** for a given employee's given section; each `RecordProfileSection`
call **overwrites** it (rather than writing a new, version-suffixed key). The chronological version
list is recovered via the ledger's **own per-key history** —
`GetHistoryForKey("profile", [employeeID, profileSection])` `[docs: Fabric-FAQ.rst]` — which
returns every past value written to that key, not just the current one. This is what makes
`GetProfileHistory` (ADR-0020 function 3) possible without a chaincode-maintained secondary index.

> **Operational precondition.** `GetHistoryForKey` depends on the peer's history database being
> enabled. This is a peer-level (`core.yaml`) configuration concern, not a chaincode-level one —
> it is flagged here as a coordination point with whoever stands up the network
> (`fabric-architect`/`sre`, via `fabric-operations`), not decided by this document.

- `tenantID` is **deliberately not part of the key**. Tenant isolation in the ratified topology is
  a **channel-per-tenant** property (`[prd: §5.1]`, FR-20/21 — formalized by fabric-architect's
  pending ADR-0013): every peer on a tenant's channel already only ever holds that tenant's keys,
  so prefixing the key with `tenantID` would duplicate an isolation guarantee the channel boundary
  already provides, at the cost of a longer key with no additional query the application needs.
  `tenantID` is kept **in the record value** (§2) purely as self-describing metadata for tooling
  that reads a record outside its channel context (e.g. a cross-tenant audit export process),
  not as an access-control mechanism.
- `employeeID` leads the key (rather than a raw internal id) so the key itself carries no
  brute-forceable identifier (ADR-0011) while still being a **stable, deterministic** lookup path.

### 5.1 Alternatives considered for the key shape

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. One head key per `(employeeID, profileSection)`, history via `GetHistoryForKey` (chosen)** | Simplest write path — one `PutState` per record, no risk of forgetting a companion index key; leverages a native ledger mechanism instead of a chaincode-built one | Depends on the peer's history database being enabled (operational precondition, above); `GetHistoryForKey` returns the **entire** history in one call, which is fine at expected per-employee-per-section cardinality (business-relevant profile changes, not high-frequency events) but would not scale gracefully to a section with unbounded version churn | **Chosen** |
| B. Version-suffixed composite key per record (`("profile", employeeID, profileSection, version)`), range-scanned for history | Explicit, bounded range queries (`GetStateByPartialCompositeKey` with a `fromVersion`/`toVersion` bound) instead of "return everything" | **1 extra write per record** (a head-pointer key plus a version-suffixed key, or a scan-then-pick-latest read pattern on every read of "the current value"); duplicated storage; no ledger-history dependency traded away, since it substitutes a chaincode-built index for one anyway | Rejected — no offsetting benefit at this design's expected write volume (five sections, business-paced changes, not an event stream) |
| C. One key per employee holding all five sections' latest hash in a single blob | Fewer keys; one read for `GetEmployeeProfileSummary` | **Rejected outright** — breaks FR-4/`[prd: §4.2]`'s explicit requirement that each section keeps its **own** independent chain; a write to `PAYROLL` would force a read-modify-write of a blob that also carries `EDUCATION`'s untouched data, creating an artificial contention point between five logically-independent chains that have no reason to serialize against each other | Rejected |

### 5.2 MVCC contention note (state-DB/performance cross-reference)

Because the head key for a given `(employeeID, profileSection)` is overwritten on every write, two
concurrent `RecordProfileSection` submissions targeting the **exact same** employee's **exact
same** section race under Fabric's optimistic concurrency control, and one is invalidated at
commit (`MVCC_READ_CONFLICT`) `[docs: readwrite.rst#example-simulation-and-validation]` — this is
the **correct**, intended outcome (it prevents a lost update on one section's chain), not a defect
to be engineered around. The submitting write path (ADR-0014) must treat this as a **retryable**
outcome, not a hard failure. **[ASSUMPTION]** the expected concurrent-write rate against the same
employee's same section is low (an approval workflow already tends to serialize edits to one
record at the application layer) and has not been measured — the existing Caliper workload
tooling's `hotKeyFraction` parameter (`rencana-rekonsiliasi.md` §4.1) is the intended instrument to
measure it once a benchmark environment exists; not re-designed here (`> See
fabric-performance/references/mvcc-and-read-write.md`).

State database remains **LevelDB** (unchanged prototype decision, not reopened by this document):
every query this design needs is key or per-key-history addressed — no rich/JSON query is
required, so there is no basis to reconsider CouchDB here
`[docs: performance.md#avoid-couchdb-for-high-throughput-applications]`.

---

## 6. On-chain vs off-chain split

| Data | Home | Rationale |
|---|---|---|
| Profile-section field values (all five sections) | **Operational database only** (existing at-rest encryption, unchanged) | System-of-record; never leaves the encrypted store; Fabric is never source-of-truth for a field. |
| Per-record content **salt** | **Off-chain**, secret store (ADR-0011) | Secret digest input; its destruction is one erasure lever (§9). |
| `employeeKey_i` (independently random, ADR-0021) | **Off-chain**, secret store, separate custody from the salt store (ADR-0011; domain boundaries formalized by the pending ADR-0019, security-architect) | Identifier-derivation secrets; destroying one employee's `employeeKey_i` is the other erasure lever. |
| **`EmployeeProfileRecord`** (digest + identifiers + metadata) | **On-chain** public world state, per tenant channel | Non-reversible integrity anchor; every org on the tenant's channel holds it for independent verification. |
| Encrypted supporting documents | **IPFS private cluster** (§10) — only the CID on-chain | Field-level document evidence without ever putting content on the ledger (FR-30/31). |

No PII ever crosses into the on-chain row, in either direction: not on write (ADR-0020's
`RecordProfileSection` never receives a section value) and not on read (the read-only functions
never receive one either).

---

## 7. Off-chain companion stores & key domains (cross-reference)

The salt store and the `employeeKey_i` store are **off-chain, security-critical
infrastructure** whose exact custody model, rotation policy, and domain boundaries are
**security-architect's** to formalize (tracked as the pending ADR-0019,
`rencana-rekonsiliasi.md` Gelombang 3 #18) — not designed in this document. This data model only
fixes the invariant both domains must respect: **neither the salt store nor the identifier-key
hierarchy is ever derived from, wraps, or is wrapped by** the Fabric MSP/TLS signing-key domain or
the supporting-document encryption key (`[prd: §5.2]` "Tiga domain kunci yang tidak boleh pernah
dicampur").

---

## 8. `ProfileSection` enum

Exactly **five** values, ratified as the anchoring unit (`[prd: §4]`, superseding the
pre-reconciliation event surface `EVENT_*` entirely — that surface is retired, not one of its
values carries over):

| `profileSection` | Typical trigger |
|---|---|
| `PERSONAL` | A personal-data field (identity, contact) is created or changed via the platform's change-approval workflow. |
| `EMPLOYMENT` | A promotion, transfer, or other employment-status change is approved. |
| `EDUCATION` | A formal or informal education record is created or updated. |
| `ADDITIONAL` | Marital status or dependent/family data changes. |
| `PAYROLL` | A salary adjustment or bank-account change is approved. |

`RecordProfileSection` validates `profileSection` against exactly these five values and rejects
anything else (FR-8, ADR-0020).

---

## 9. No Private Data Collection in this design

The pre-reconciliation data model reserved a narrow HR↔Audit PDC for the case where an actual
field *value* needed to be exchanged between two orgs. **This design uses none, anywhere.**
ADR-0020 (Alternative E) explains why in the chaincode-contract context: a PDC still requires the
plaintext to reach a proposal's `transient` field and be disseminated to every collection member —
it relabels where the plaintext travels without removing the need to trust a peer outside your own
org, and it is measurably slower than an equivalent world-state asset
`[docs: performance.md#private-data-collections-pdcs-vs-world-state]`. Because no function in this
contract ever receives a section value as an argument in the first place (§1), there is no value
left over for a PDC to protect. This is a direct consequence of ADR-0020, not a new decision of
this document — recorded here so a reader of the data model does not go looking for a
`--collections-config` that no longer exists.

---

## 10. Supporting documents (IPFS-linked, FR-30..33)

Each `EmployeeProfileRecord` version may carry zero or more `ipfsCIDs` (§2) pointing at documents
relevant to that section version (an identity document for `PERSONAL`, a certificate for
`EDUCATION`, a payslip for `PAYROLL`, and so on — FR-32). Documents are encrypted with the
employee's own document key **before** upload, and only the resulting CID — never document
content — becomes part of the record (FR-30/31). The IPFS private-cluster architecture itself
(replication scope, pinning policy, the document-encryption key's custody) is **not** designed in
this document — it is tracked as the pending ADR-0016 (`rencana-rekonsiliasi.md` Gelombang 3 #18).
Two properties of a content-addressed store are worth carrying into this data model explicitly,
because they shape the field's semantics:

- A CID is a **retrieval capability**, not merely a label — anyone holding it can request the
  object from the cluster. Because `ipfsCIDs` is on-chain, every org on the tenant's channel can
  read the CID; what actually prevents retrieval is the cluster's own access control, not the CID
  being secret (`[prd: §11.4]`).
- Re-encrypting a document (e.g. after a document-key rotation) produces a **new** CID, which is
  why a new CID is attached to a **new** record version rather than mutating an existing record's
  `ipfsCIDs` in place — consistent with this design's append-by-new-version, never-mutate-history
  posture (§5).

---

## 11. Erasure (cross-reference, FR-25..29)

The mechanics of crypto-shredding — deleting the operational-database field, destroying the
document key, and destroying the salt and `employeeKey_i` for one employee — are ratified at the
requirement level (`[prd: §7 Kelompok E]`) but their storage/custody design is **not** this
document's to make (tracked as the pending ADR-0015, `rencana-rekonsiliasi.md` Gelombang 3 #18,
owned jointly by security-architect and fabric-architect). What this data model fixes, because it
follows directly from §2–§5: after an erasure, the on-chain `EmployeeProfileRecord`s for that
employee are **not** modified, moved, or deleted — the ledger stays append-only-by-overwrite in
its normal write path, and erasure acts **only** off-chain. The surviving records become orphaned
digests: `GetProfileHistory`/`GetProfileSectionRecord` still return them, but no salt or
`employeeKey_i` remains to recompute or re-derive anything from them (FR-27/28).

---

## 12. Assumptions & gaps carried

| Gap | Impact on this data model |
|---|---|
| **G-24 / PB-3** | `canonicalizationVersion="JCS-RFC8785-v1"` binding is a versioned field (§2/§3), but the concrete library pin remains open. |
| **G-10 (reopened)** | Which HRIS process hosts the Fabric Gateway client is not decided here (ADR-0014); this document assumes only that "some in-band caller" exists per section write path. |
| **[ASSUMPTION]**, MVCC contention rate | Unmeasured; flagged in §5.2 as the intended target of the existing Caliper `hotKeyFraction` tooling once a benchmark environment exists. |
| Key-domain custody (pending ADR-0019) | This document states the non-mixing invariant (§7) but does not design custody/rotation. |
| Erasure storage design (pending ADR-0015) | This document states the ledger-side consequence (§11) but not the off-chain mechanics. |
| IPFS cluster architecture (pending ADR-0016) | This document states the field shape and its two content-addressing consequences (§10) but not the cluster design. |

---

## 13. Traceability

- **Anchor chain (1):** HRIS write path commits a profile section → off-chain digest computed
  in-band (ADR-0011/ADR-0014) → `RecordProfileSection` → co-signed by the platform org and the
  enterprise-client org → immutable per-section-per-employee chain (ADR-0020) → **CIA-Integrity**.
- **Confidentiality chain (2):** section plaintext **never** leaves the operational database and
  **never** becomes a chaincode argument, on Submit or Evaluate (§1, ADR-0020) → **Privacy-by-Design**.
- **Erasure chain (4):** erasure request → destroy the employee's salt + `employeeKey_i` off-chain
  (§11) → surviving on-chain digests become un-openable orphans → **Privacy-by-Design**.

Sequence view: `00-architecture/diagrams/anchor-write-sequence.mmd`. Write/verify path:
`integration-design.md`. Chaincode contract surface: `api-contracts.md` (ADR-0020).

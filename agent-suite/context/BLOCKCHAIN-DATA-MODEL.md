# Blockchain Data Model — what the ledger holds

> **Scope.** Defines the on-ledger record shape for anchoring employee profile-section writes on
> Hyperledger Fabric 2.5 — the `EmployeeProfileRecord` asset, the composite-key strategy, the on-chain
> vs off-chain boundary, and the erasure design. There is **no Private Data Collection anywhere in this
> design** — see §7. Companion: [`BLOCKCHAIN-INTEGRATION.md`](BLOCKCHAIN-INTEGRATION.md) covers how the
> HRIS write paths produce these records in-band.
>
> **Rewritten 2026-08-06** (`rencana-rekonsiliasi.md` Gelombang 5 #33) to match the ratified PRD design.
> The pre-2026-08-06 version of this document described an `AnchorRecord` shape (per-changed-event
> commitment, a `companyId`-prefixed composite key, an optional HR↔Audit PDC) that is **entirely
> superseded** — no field name from that version survives. The authoritative version of everything below
> is [`../00-architecture/solution/data-model.md`](../00-architecture/solution/data-model.md)
> (`fabric-engineer`, instantiating **ADR-0011**/**ADR-0014**/**ADR-0020**); this context doc
> cross-references it rather than restating it, per `AUTHORING-CONTRACT.md §7`.
>
> **Cross-reference, do not duplicate.** Fabric mechanics for composite keys, world-state CRUD, and
> per-key history live in the skill suite — this doc points to them:
> `> See fabric-chaincode-dev/references/contract-api.md`.
>
> **Register note.** This document describes the HRIS write paths that produce each profile section only
> in generic terms — no product name, codebase, or literal file/class/table name appears here. Real-code
> grounding was consulted internally and is preserved without being reproduced verbatim on this page —
> see grounding gap **G-29**.

## 1. Design invariant: a per-section digest, never plaintext PII, never on either side of the call

The authoritative PII store stays **off-chain**, in the existing operational database under its existing
at-rest encryption; Fabric holds only a **salted digest**, two HMAC-derived pseudonymous identifiers, and
non-PII metadata, per profile section, per version.

> **INVARIANT (ADR-0011/ADR-0020).** Zero bytes of profile-section plaintext — and zero reversible
> derivative of it — are ever written on-chain, or ever appear as a chaincode transaction argument, **on
> Submit or on Evaluate**. This binds the read side as tightly as the write side: no function in this
> contract has a parameter that could carry a section value or a salt in the first place.

- Digest algorithm: **SHA-256** (256-bit) over `salt ‖ JCS(section)` — the section's complete current
  value, canonicalized per RFC 8785, not a `{from,to}` delta.
- Identifier MAC: **HMAC-SHA256**, keyed by `employeeKey_i` — an **independently-random**, per-employee
  secret (ADR-0021; there is **no** shared master/tenant key deriving it).
- Per-record content salt: **≥128 bits**, CSPRNG, unique per record, off-chain only.
- Predictable PII is brute-force-guessable against a bare hash, so every digest is salted
  `[docs: private-data-arch.rst#protecting-private-data-content]` — the salt itself never crosses to a
  peer, on either Submit or Evaluate.

> Existing crypto assets are reused, not reinvented: the operational database's own at-rest
> envelope-encryption model stays exactly as it is today (`[prd: §5.2]` "Fabric anchors on top of
> existing app-level encryption"). Fabric MSP/HSM signing keys, the app AES PII-at-rest key, the
> document-encryption key, and the `employeeKey_i`/salt domain are **four separate key domains that must
> never be derived from, wrap, or be wrapped by one another** (ADR-0019/ADR-0021) — this document
> consumes that boundary, it does not design it.

## 2. The `EmployeeProfileRecord` asset (public world state, per tenant channel)

One record per profile-section write — the **current head** for a given employee's given section. There
is **no PDC anywhere in this design** (§7).

```jsonc
{
  "recordID":                "9f2c1a7b-4e0d-4f61-8a3c-1b2c3d4e5f60", // UUID v4
  "employeeID":              "3a7e9c…",           // HMAC-SHA256(employeeKey_i, "id") — hex
  "tenantID":                "t_00842",            // metadata only — isolation is by CHANNEL membership, not this field
  "profileSection":           "PAYROLL",            // one of exactly 5 enum values
  "dataHash":                "sha256:9c1e2f…",     // SHA-256(salt ‖ JCS(section)) — computed OFF-CHAIN
  "prevHash":                "sha256:04d2ab…",     // DataHash of the PRIOR record for this (employeeID, profileSection); "" for the first
  "version":                 7,                     // monotonic per (employeeID, profileSection)
  "updatedBy":                "b81f3e…",            // HMAC-SHA256(employeeKey_i, "actor" ‖ user_id) — hex
  "timestamp":               "2026-08-06T04:11:22Z", // client-reported; AUTHORITATIVE time is the ledger tx timestamp
  "ipfsCIDs":                 ["bafy…d1", "bafy…d2"], // CIDs of encrypted supporting documents tied to THIS version; [] if none
  "canonicalizationVersion": "JCS-RFC8785-v1",      // versioned canonicalization interface
  "hashAlgo":                "SHA-256"               // versioned digest-algorithm interface
}
```

Written with `PutState(compositeKey, marshaledRecord)`, read with `GetState`/`GetHistoryForKey`
`[docs: Fabric-FAQ.rst]` (`> See fabric-chaincode-dev/references/contract-api.md`). Full field reference,
the `submittedBy`-vs-`updatedBy` identity distinction, and the versioned-interface rationale for
`canonicalizationVersion`/`hashAlgo`:
[`../00-architecture/solution/data-model.md`](../00-architecture/solution/data-model.md) §2.

## 3. Composite-key strategy

```
key = CreateCompositeKey("profile", [employeeID, profileSection])
    → "profile" \x00 3a7e9c… \x00 PAYROLL \x00
```

- Each write **overwrites** the current head for its `(employeeID, profileSection)` key — this is
  deliberately **not** an append-only-by-suffix scheme. The chronological version list is recovered from
  the ledger's own per-key history, `GetHistoryForKey("profile", [employeeID, profileSection])`
  `[docs: Fabric-FAQ.rst]`, not from a chaincode-maintained secondary index.
- **`tenantID`/`companyId` is deliberately not part of the key.** Tenant isolation is a
  **channel-per-tenant** property (ADR-0013): every peer on a tenant's channel already only ever holds
  that tenant's keys, so a leading tenant-prefix component — load-bearing in the pre-2026-08-06 version
  of this key — would duplicate an isolation guarantee the channel boundary already provides.
- `employeeID` leads the key (a pseudonymous HMAC output, never a raw internal id) so the key itself
  carries no brute-forceable identifier while remaining a stable, deterministic lookup path.
- Two concurrent writes to the **same** employee's **same** section race under Fabric's optimistic
  concurrency control, and one is invalidated at commit (`MVCC_READ_CONFLICT`) — this is the correct,
  intended outcome (it prevents a lost update on one section's chain), handled by re-reading the head and
  resubmitting, never treated as a hard failure. Full key-shape alternatives table and the MVCC-contention
  note: `data-model.md` §5/§5.2, `> See fabric-performance/references/mvcc-and-read-write.md`.

State database is **LevelDB**, not CouchDB (ADR-0007, unchanged) — every query this design needs is key
or per-key-history addressed, so there is no basis for a rich/JSON query engine
`[docs: performance.md#avoid-couchdb-for-high-throughput-applications]`.

## 4. On-chain vs off-chain split

| Data | Home | Rationale |
|---|---|---|
| Profile-section field values (all five sections) | **Operational database only**, existing at-rest encryption, unchanged | System-of-record; Fabric is never source-of-truth for a field. |
| Per-record content **salt** | **Off-chain**, secret store | Secret digest input; its destruction is one of the two erasure levers (§6). |
| `employeeKey_i` (per employee, independently random) | **Off-chain**, secret store, separate custody from the salt store | Identifier-derivation secret; destroying it is the other erasure lever. |
| **`EmployeeProfileRecord`** (digest + identifiers + metadata) | **On-chain** public world state, per tenant channel | Non-reversible integrity anchor; every org on the tenant's channel holds it for independent verification. |
| Encrypted supporting documents | **IPFS private cluster** — only the CID on-chain | Field-level document evidence without ever putting content on the ledger. |

No PII ever crosses into the on-chain row, in either direction: not on write (`RecordProfileSection`
never receives a section value) and not on read (the three read-only functions never receive one
either). Full split table and rationale: `data-model.md` §6.

## 5. Off-chain companion stores

The per-record salt store and the `employeeKey_i` store are **off-chain, security-critical
infrastructure** whose custody model, rotation policy, and domain boundaries are `security-architect`'s
to formalize (ADR-0019/ADR-0021) — not designed in this document. **Verification flow (the core demo):**
a verifier recomputes `SHA-256(salt ‖ JCS(currentSectionValue))` from the section's current value plus its
off-chain salt and compares it to the on-chain `dataHash` — entirely **client-side**, against the
verifier's own peer where one exists. There is no server-side read-back service in this design to perform
that recompute on a verifier's behalf (contrast the pre-2026-08-06 anchor-service pattern, now retired) —
see [`BLOCKCHAIN-INTEGRATION.md`](BLOCKCHAIN-INTEGRATION.md) §4 and
`integration-design.md` §6 for the full client-side verification protocol.

## 6. Erasure & right-to-erasure

- Public `EmployeeProfileRecord`s are non-reversible digests, so they carry no PII and may be retained
  indefinitely as immutable audit evidence — this is the tamper-evidence the design exists for.
- **Erasure is crypto-shredding, not a ledger operation (ADR-0015).** To honour a right-to-erasure
  request: delete the operational-database field, destroy the document-encryption key
  (`KEY_EMPLOYEE`), destroy the employee's `DataHash` salts, and destroy the employee's `employeeKey_i`.
  The on-chain digests survive, untouched, as un-openable orphans — no `PurgePrivateData` call, no
  `blockToLive`, and **no ledger transaction of any kind** is issued by erasure.
- Erasure custody/storage mechanics are `security-architect`'s and `fabric-architect`'s to formalize
  (ADR-0015/ADR-0019) — this document states only the on-chain consequence (nothing changes on-chain)
  per `data-model.md` §11.

## 7. No Private Data Collection in this design

The pre-2026-08-06 version of this document reserved a narrow PDC for the case where an actual field
*value* needed to be exchanged between two orgs. **This design uses none, anywhere.** Because no function
in the `EmployeeProfileRecord` contract ever receives a section value as an argument in the first place
(§1), there is no value left over for a PDC to protect — a PDC would still require the plaintext to reach
a proposal's `transient` field and be disseminated to every collection member, relabeling where the
plaintext travels without removing the need to trust a peer outside your own org, and it is measurably
slower than an equivalent world-state asset
`[docs: performance.md#private-data-collections-pdcs-vs-world-state]`. This is a direct consequence of
ADR-0020 (its Alternative E), recorded here so a reader of this data model does not go looking for a
`--collections-config` that does not exist. See `data-model.md` §9 for the full reasoning.

## 8. Traceability

Anchor chain: `HRIS profile-section write path → operational-database commit → in-band off-chain digest
→ co-signed RecordProfileSection → immutable per-section-per-employee chain → CIA-Integrity`. Erasure
chain: `erasure request → destroy salt + employeeKey_i off-chain → un-openable digests remain →
Privacy-by-Design`. Confidentiality chain: `section plaintext never leaves the operational database and
never becomes a chaincode argument, on Submit or Evaluate`. Full traceability, including the document
chain (IPFS CIDs): `data-model.md` §13.

# Key-domain separation checklist — DEP-2

Backlog item: **DEP-2**. This is an **operational checklist**, not a re-derivation of the design —
the four-domain model and its rationale are ratified and immutable in ADR-0019 (amends ADR-0009
from two domains to four) and ADR-0021 (Domain C's internal structure: independently-random
`employeeKey_i`, no `pseudonymKey` master key). Read those two ADRs for *why*; this file exists to
give an operator/reviewer a concrete, checkable answer to *"has anything crossed a domain
boundary"* against this repo's actual code and config, not against an abstract description of it.

**Why this checklist exists as a separate artifact, not just "read the ADRs."**
`write-path-integration/keystore/keystore.go`'s own header states the enforcement mechanism today
is "by construction, not by a runtime check... the discipline is 'never import or reference Domain
A/B material here.'" A discipline enforced only by construction has no alarm if it is ever
violated in a future change — this checklist is the periodic, human-run check that substitutes for
the runtime check that does not exist.

---

## The four domains, at a glance

| Domain | Key material | Ratifying ADR |
|---|---|---|
| **A** | Fabric MSP signing keys (X.509) + TLS keys | ADR-0009 (unchanged, carried by ADR-0019) |
| **B** | Application `PII_ENCRYPTION_KEY[_Vn]` (AES-256-CBC, operational-database columns) | ADR-0009 (unchanged, carried by ADR-0019) |
| **B′** | `KEY_EMPLOYEE` — per-employee symmetric key, IPFS document encryption | ADR-0019 (new) |
| **C** | `employeeKey_i` (per-employee) + per-record `DataHash` salts | ADR-0019 (new) as amended by ADR-0021 (independently-random, no master key) |

**Founding cross-domain rule (ADR-0009, restated across all six pairwise boundaries by
ADR-0019):** no domain's key material may be derived from, wrap, or be wrapped by another domain's
key material.

---

## Domain A — Fabric MSP signing keys + TLS keys

| | |
|---|---|
| **Generated** | `cryptogen generate` (dev/test, throwaway local CA) **or** `fabric-ca-server`/`fabric-ca-client enroll` (NET-2 path, `bccsp: SW` in this workspace today) **or** inside a PKCS11 HSM token (production option, not implemented in this repo — `identity-tls-hsm-provisioning.md` §5) |
| **Stored** | TLS keys: always file-based, `tls/server.key` under each node's local MSP dir (`hsm.md`'s documented constraint — never optional, never HSM-eligible). MSP signing keys: file-based `msp/keystore/*_sk` today (both cryptogen and NET-2 paths); HSM-resident only (never exported, `keystore/` stays empty) in the production HSM option |
| **Who/what may read** | The specific Fabric node process (peer/orderer/`fabric-ca-server`) reading its **own** local MSP/TLS directory at process start; a `fabric-ca-client` invocation reading the identity home dir it was pointed at (`FABRIC_CA_CLIENT_HOME`) for register/enroll/revoke operations |
| **Must never** | 1. Be used to derive or wrap Domain B/B′/C material, or vice versa (ADR-0009's founding rule). 2. Leave its node's local filesystem/HSM boundary — never copied into a `ConfigMap`, logged, attached to a support bundle, or embedded in an error message. 3. Have its HSM PIN (if HSM-backed) appear as a literal in any config file, Helm `values.yaml`, or this repo's git history — secret reference only (`identity-tls-hsm-provisioning.md` §5.4). 4. Be generated, read, or referenced by any process in `write-path-integration/` (`gatewayclient`, `ipfsclient`, `keystore`, `writepaths`) — those modules **consume** already-issued gateway/client Domain-A identities (TLS+MSP cert/key pairs) to open a Fabric Gateway connection; they never generate, mutate, or derive from Domain A material. `keystore.go` in particular implements only Domain B′/C stores — grep confirms no `crypto/x509`, `crypto/tls`, or MSP-path import anywhere in that file. |

## Domain B — application AES PII-at-rest key

| | |
|---|---|
| **Generated / stored** | Outside this repo entirely — the existing HRIS operational database's own `PII_ENCRYPTION_KEY[_Vn]` versioning scheme (`[code: ems/pkg/db/encryption_plugins.go]`, cited by ADR-0009/ADR-0019, not part of `fabric-hris`). `write-path-integration/` does not implement, generate, or store any Domain B material — this repo's write-path components only ever produce the **outputs** of Domain-C computations (`EmployeeID`, `UpdatedBy`, `DataHash`) that get anchored; they never touch the operational database's own PII columns or their encryption key. |
| **Who/what may read** | The existing HRIS application's own database access layer — not this repo's concern to gate, only to never duplicate |
| **Must never** | 1. Be reused to encrypt IPFS documents (that is Domain B′'s job specifically, ADR-0019's Alternative B rejection: "forcing per-employee destroy-ability onto it means... collapsing two incompatible retention regimes onto one key"). 2. Be used to derive `employeeKey_i` or any `DataHash` salt (Domain C). 3. Be used as, or derived from, Fabric MSP/TLS signing material (Domain A). 4. Be rotated per-employee — it is tenant/version-wide only; an erasure request must never trigger a Domain B key operation (ADR-0019 Domain B row: "Never used to satisfy an individual erasure request — that is a plain row/field delete... not a key operation"). |

## Domain B′ — `KEY_EMPLOYEE` (IPFS document encryption)

| | |
|---|---|
| **Generated** | `crypto/rand` exclusively, inside `write-path-integration/keystore/keystore.go`'s `DocumentKeyStore.GetOrCreateDocumentKey` — `MinKeyBytes = 16` (128-bit floor, ADR-0011's floor applied here). One key per employee, generated once at first call, never re-derived on subsequent calls (`InMemoryDocumentKeyStore` checks `s.keys[employeeInternalID]` before generating). |
| **Stored** | Off-chain, one record per employee. **Current implementation status: reference-shape only** — `InMemoryDocumentKeyStore` is explicitly documented in `keystore.go`'s own header as "NOT the production store (no persistence, no encryption-at-rest, no backup discipline)." The production store's schema/backup/destruction design is still an open build-time task (ADR-0019 §Follow-ups: "The concrete B′/C store schema, backup policy, and destruction procedure are build-time tasks for `fabric-engineer`") — re-run this checklist against that production store once it exists; it does not yet. |
| **Who/what may read** | The document-encryption call site immediately before an IPFS upload (REC-5's scope, `write-path-integration/ipfsclient`/`writepaths` hook path) — fetches the key, AES-128-GCM-encrypts the document, discards the plaintext key from that call frame. Never read by chaincode (0 bytes of document plaintext or ciphertext ever reach the ledger, per REC-5's own DoD). Never read by any Fabric MSP/CA process (Domain A). |
| **Must never** | 1. Be derived from, or used to derive, `employeeKey_i`/salts (Domain C) — ADR-0019's explicit bullet: "a compromised document key must not open the identifier pseudonymization, and vice versa." 2. Be derived from Domain A or B, or used to derive them. 3. Survive its own `DeleteDocumentKey` call in any backup/replica — a backup that "quietly retains a deleted key defeats the entire mechanism silently" (ADR-0019 §Consequences, restated for B′/C jointly). 4. Be logged, or appear in an error message — it is a raw 16+-byte secret; `keystore.go`'s own error paths never include key bytes (confirmed by reading every `fmt.Errorf` call site in the file — none interpolate `key`/`salt` values, only IDs and lengths). |

## Domain C — `employeeKey_i` + `DataHash` salts

| | |
|---|---|
| **Generated** | `crypto/rand` exclusively, inside the same `keystore.go` file: `EmployeeKeyStore.GetOrCreateEmployeeKey` (`employeeKey_i`, ≥128-bit, generated **once per employee**, stable across that employee's records — ADR-0021's numeric constraint, distinct from the salt's per-record cadence) and `SaltStore.PutSalt` (`DataHash` salt, ≥128-bit `MinSaltBytes = 16`, generated **once per record version**, rejects a second `PutSalt` call for the same key with an explicit error rather than silently overwriting — "a record's salt is generated once, not rotated in place"). **Per ADR-0021: no `pseudonymKey` or any other master/tenant-scoped secret exists anywhere in this design any more** — `employeeKey_i` is independently random, full stop. Any future code change that introduces an HKDF/HMAC derivation of `employeeKey_i` from a shared tenant secret would be reintroducing the exact hierarchy ADR-0021 explicitly rejected (its Option A) — treat that as a hard regression, not a refactor. |
| **Stored** | Off-chain, per-employee (`employeeKey_i`) / per-record-version (salt). **Current implementation status: reference-shape only**, same caveat as Domain B′ — `InMemorySaltStore`/`InMemoryEmployeeKeyStore` are not the production store; that design is the same open ADR-0019/ADR-0021 follow-up item as B′'s. |
| **Who/what may read** | The write-path component computing `EmployeeID = HMAC-SHA256(employeeKey_i, "id")` / `UpdatedBy = HMAC-SHA256(employeeKey_i, "actor" ‖ user_id)` and `DataHash = SHA-256(salt ‖ JCS(section))` at write time (`write-path-integration/writepaths`); any verifier that must recompute `DataHash` against the same input via `SaltStore.GetSalt` (the store's own doc comment: "needed by any party... that must recompute DataHash against the same input"). Never read by chaincode directly — only the HMAC/hash **outputs** are anchored, never the key/salt itself. |
| **Must never** | 1. Be derived from a tenant-scoped or any other master secret — ADR-0021 closed this (T16b) by removing `pseudonymKey` entirely; do not reintroduce it under a different name. 2. Be used to encrypt IPFS documents (Domain B′) or vice versa. 3. Be used as, or derived from, Fabric MSP/TLS material (Domain A) or the app AES key (Domain B). 4. Survive its own `DeleteEmployeeKey`/`DeleteAllSaltsForEmployee` call in any backup/replica — identical erasure-defeat concern as B′ (ADR-0015's four-part crypto-shred depends on this holding for **both** B′ and C; a backup that retains either one silently breaks the erasure guarantee for that employee with no visible symptom). 5. Have the salt "rotated in place" for an existing record version — `InMemorySaltStore.PutSalt` already enforces this at the interface level (errors if a salt exists); any production store replacing it must preserve that same one-time-generation guarantee, not relax it for convenience. |

---

## Cross-cutting verification procedure

Run this whenever `write-path-integration/keystore/keystore.go` changes, whenever a production
store replaces the `InMemory*` reference implementations, and periodically (recommend: every
release that touches any write-path module) as a standing review item — not a one-time check.

1. **No cross-domain imports.** `keystore.go` must never import `crypto/x509`, `crypto/tls`, or
   any Fabric MSP/gateway-client package — its only crypto primitive should be `crypto/rand`
   (confirmed today: `grep -n '"crypto' write-path-integration/keystore/keystore.go` returns only
   `crypto/rand`). If a future change adds `crypto/hmac` or `golang.org/x/crypto/hkdf` **inside
   this file**, treat it as a design-review trigger, not a routine dependency bump — Domain C's
   HMAC computation (`EmployeeID`/`UpdatedBy`) belongs in the **write-path** module that consumes
   the key, not inside the key-generation/storage package itself; an HMAC import appearing inside
   `keystore.go` would suggest key material is being derived where it should only be generated and
   stored.
2. **No shared byte slices across store types.** `DocumentKeyStore`, `EmployeeKeyStore`, and
   `SaltStore` must never share an underlying map, mutex, or generation call — each of the three
   `InMemory*` types (and any production replacement) must independently call `crypto/rand.Read`
   and hold its own storage map; a refactor that "unifies" them into one generic
   `map[string][]byte`-backed store keyed by a `(domain, id)` tuple would still be **acceptable
   only if** access to each domain's slice of that map is never exposed to the wrong call site —
   prefer keeping them as visibly separate types precisely so this is checkable by reading type
   signatures, not by auditing runtime access patterns.
3. **No HSM PIN, no Domain-A path, appears in `write-path-integration/`.** `grep -rniE
   "pkcs11|bccsp|libsofthsm|CORE_PEER_MSPCONFIGPATH" write-path-integration/keystore/
   write-path-integration/writepaths/` should return nothing — Domain A configuration belongs in
   `fabric-network/` and `deploy/`, never in the Go modules that consume Domain B′/C.
4. **Secret handling in logs/errors.** `grep -n "key\b\|salt\b" write-path-integration/keystore/keystore.go`
   and manually confirm every match is either a variable **name** (`key`, `salt` as local
   identifiers) or a length constant (`MinKeyBytes`, `MinSaltBytes`) — never an interpolated value
   in an `fmt.Errorf`/log call. Re-run this check against any production store implementation
   before it replaces the `InMemory*` types.
5. **Production-store follow-up gate.** Before any production `DocumentKeyStore`/`EmployeeKeyStore`/
   `SaltStore` implementation ships (closing the ADR-0019/ADR-0021 open follow-up), re-run this
   entire checklist against that implementation specifically — the "who/what may read" and
   "must never survive backup" rows above are currently verified against in-memory, non-persistent
   reference code; a real persistence layer introduces backup/replication surfaces this checklist's
   authors could not check because they did not exist yet.

## Review checklist (design-artifact review, not empirical verification)

| Item | Status |
|---|---|
| All four domains covered with generated/stored/read/must-never rows | Reviewed — A, B, B′, C above |
| Domain C reflects ADR-0021 (independently-random, no `pseudonymKey`), not the superseded ADR-0019 hierarchy | Reviewed — explicitly called out as a hard-regression trigger if reintroduced |
| Checklist is operational (greppable/checkable), not a restatement of ADR prose | Reviewed — §Cross-cutting verification procedure gives concrete `grep`/review commands against this repo's actual files |
| Open implementation gaps (in-memory-only stores) disclosed, not glossed over | Reviewed — stated for both B′ and C, with the exact ADR-0019 Follow-ups citation |

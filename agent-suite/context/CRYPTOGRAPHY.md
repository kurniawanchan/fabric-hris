# Cryptography in the HRIS-on-Fabric System

> **Re-derived 2026-08-06** against the ratified design in `prd-fabric-hris-2026-08-02/prd.md`
> (status: final) and **ADR-0011/ADR-0012/ADR-0013/ADR-0015/ADR-0016/ADR-0019/ADR-0020/ADR-0021** —
> not a new design. The pre-reconciliation version of this doc (single shared `pseudonymKey`, a
> two-org topology, and Private Data Collections as the on-chain confidentiality mechanism) is
> **superseded in substance**; see `../11-execution/grounding-gaps.md` and
> `../05-adr/README.md` for the closure trail.

> **What this doc is.** A context/theory map of *every* cryptographic mechanism this system relies
> on, and where each one lives. It is a REFERENCE doc for the agent suite, not a skill — deep Fabric
> mechanics are owned by the `fabric-*` skills and are **cross-referenced, not restated** here. Its
> job is to (a) name the crypto primitives, (b) separate documented fact from recommendation, and
> (c) draw the boundary between **Fabric node crypto**, the **application-level PII crypto that
> already exists in the repos**, and the **two new off-chain key domains** this design adds
> (document-encryption and identifier-pseudonymization keys).

> **Scope note.** This describes the ratified design. Several build-time specifics remain open —
> see `../11-execution/grounding-gaps.md` (G-11, G-15 closed via ADR-0019/ADR-0021, G-18, G-24/PB-3).

---

## 1. Four key domains, not two — and no domain may be derived from another

The pre-reconciliation model had two crypto subsystems. §5.2 of the ratified PRD adds two more —
document encryption and identifier pseudonymization — each with a lifecycle property neither
original domain needed: **provable, irreversible, per-employee destruction** (crypto-shred,
ADR-0015), not merely rotation. **ADR-0019** formalizes all four; **ADR-0021** (2026-08-06, human
decision) simplifies Domain C's internal structure without reopening the other three.

```
 DOMAIN A                    DOMAIN B                  DOMAIN B'                 DOMAIN C
 Fabric node crypto          App PII-at-rest crypto     Document-encryption key    Identifier pseudonymization
 ──────────────────         ──────────────────         ──────────────────         ──────────────────
 X.509/PKI signing (D6)      AES-256-CBC envelope        KEY_EMPLOYEE                employeeKey_i (per employee,
 TLS transport keys (D13)    encryption                  (per employee)              CSPRNG, independently random)
 HSM-held private keys       PBKDF2-SHA256 bank hash                                  + per-record DataHash salt
 (D14)                       display masking                                         (CSPRNG, per record)
 cert lifecycle/CRL (D15)
 purpose: WHO signed/        purpose: keep operational-  purpose: keep IPFS-hosted   purpose: pseudonymize on-chain
 WHO may connect             DB PII unreadable at rest    documents unreadable        EmployeeID/UpdatedBy; defeat
                                                           (FR-30)                     dictionary attacks on DataHash
 lives in: Fabric MSP/HSM    lives in: operational DB +   lives in: off-chain key     lives in: off-chain key/salt
                             app layer (unchanged)        store, one record/employee   store, one record/employee
                                                                                        (ADR-0021: no master key)
```

**Cross-domain rule (ADR-0019, reaffirmed by ADR-0021).** No domain's key material may be derived
from, wrap, or be wrapped by another domain's. A key used to anchor on-chain must never also decrypt
PII, and vice versa — enforced across all **six pairwise boundaries** now that there are four
domains, not one. `[code: ems/pkg/encryption/config.go]`, `[docs: hsm.md]`.

**Domain C's final shape (ADR-0021, 2026-08-06 — supersedes ADR-0011/ADR-0019's two-tier
`pseudonymKey → employeeKey_i` hierarchy).** `employeeKey_i` is a **CSPRNG-generated, independently
random secret, ≥128 bits, generated once per employee at that employee's first profile-section
write** — structurally identical to a `DataHash` salt, not derived from any tenant-scoped master
key. **There is no `pseudonymKey` in this design.** The two-tier hierarchy was closed because a
shared master key's compromise could silently reverse every completed erasure for a tenant (T16b,
disclosed in ADR-0019 §Consequences, closed in ADR-0021) — see §3.2. The accepted trade-off: an
accidentally-lost `employeeKey_i` is now permanent and indistinguishable from an intentional
erasure — the same class of risk already accepted for `KEY_EMPLOYEE` (Domain B′, OQ-5).

`D#` labels are the Fabric-capability IDs from `../11-execution/knowledge-graph.md` §Layer D.

---

## 2. Domain A — Fabric identity, transport, and key protection (unchanged in shape)

### 2.1 X.509 / PKI identity (D6)

Fabric's default MSP implementation uses **X.509 certificates as identities**, adopting a
traditional hierarchical **Public Key Infrastructure (PKI)** model; a valid certificate is not
enough — the MSP must also accept its issuer `[docs: identity/identity.md]`. Every peer, orderer,
and client acts under an X.509 credential; the private key produces the digital signature that
endorses a transaction.

> Depth (do not restate): PKI hierarchy, root/intermediate CAs, and MSP folder structure live in
> `fabric-identity-security/references/pki-hierarchy.md` and `.../msp-structure.md`.

**Topology fact (ADR-0012).** The ratified consortium is **three** organizations, not two: the
platform provider (Org1, hosts endorsing/committing peers and the entire 3-node Raft ordering
service), the enterprise client (`OrgClient-<tenantID>`, one independently-operated peer per
tenant), and an auditor (Org3, one read-only, non-endorsing peer). NodeOUs continue to assert
`client`/`peer`/`admin`/`orderer` roles by OU `[docs: msp.rst]` — the role-mapping **shape** is
unchanged from ADR-0005, only the MSP identifiers it is instantiated against changed.

**HRIS mapping.** HRIS roles map to X.509 OUs/attributes read via CID in chaincode — the
`[ASSUMPTION]` in gap G-11, mirroring the existing app-tier RBAC exclusions
`[code: ems/internal/base/authz/employee_rbac.go]`.

### 2.2 TLS in transit (D13)

Fabric secures node-to-node traffic with **TLS**, supporting both one-way (server-only) and two-way
(mutual) authentication `[docs: enable_tls.rst]`, `[docs: security_model.md]`. Mutual TLS is enabled
per node via `peer.tls.clientAuthRequired` / `ORDERER_GENERAL_TLS_CLIENTAUTHREQUIRED`, and each TLS
server certificate must carry a **Subject Alternative Name** matching its address
`[docs: enable_tls.rst]`.

> Depth: TLS enablement, SAN pitfalls, and mTLS handshake debugging live in
> `fabric-identity-security/references/tls.md`.

**Blast radius.** If a proxy sits in front of Fabric it must be **TLS pass-through
(non-terminating)** or the mutual-auth trust chain breaks `[docs: enable_tls.rst]`. This directly
concerns the gateway trust boundary flagged in gap G-18/PB-1 (still open, unaffected by this
reconciliation).

### 2.3 HSM key protection (D14)

An HSM protects private keys and performs the crypto operations so peers/orderers **sign and endorse
without exposing their private keys**; a **FIPS 140-2** certified HSM is available where required.
Fabric talks to an HSM via the **PKCS11** standard, configured in the node's `bccsp` section
`[docs: hsm.md]`. When an HSM is used, the node's `keystore` folder stays empty and the private key
is retrieved via the signing cert's Subject Key Identifier `[docs: hsm.md]`.

**Documented constraint (fact).** Fabric can use an HSM for peer/orderer **MSP identities**, but
**for TLS you must use file-based keys** `[docs: hsm.md]`. Domain A itself splits: signing keys can
be HSM-backed while TLS keys remain on disk — a design constraint, not a choice.

> Depth: full `bccsp` PKCS11 config, `Immutable`, AWS-HSM `AltID`, `GO_TAGS=pkcs11` build, all live
> in `fabric-identity-security/references/advanced-identity.md`.

### 2.4 Certificate lifecycle (D15)

Certificate expiry, rotation, and CRL revocation are lifecycle concerns owned by
`fabric-identity-security/references/cert-lifecycle.md`. **Recommendation:** an expired orderer/peer
TLS or signing cert halts anchoring on every channel it serves; track expiry as an operational risk
(see `../10-risk/`).

---

## 3. On-chain anchoring crypto — direct digest, no Private Data Collection (D2/D3)

**This is the section most changed by the reconciliation.** The pre-reconciliation design routed
the on-chain commitment through a Private Data Collection (D1): a PDC writes the actual data
peer-to-peer to a collection's member orgs and writes only a hash to every peer's ledger. **No PDC
exists anywhere in this design (ADR-0015 Context, ADR-0020 Alternative E).** Because no chaincode
function ever receives a profile-section value as an argument in the first place (§3.1), there is
no plaintext left over for a PDC to protect — the digest is computed **off-chain**, by the HRIS
write path that already legitimately holds the section value, and only the digest is ever submitted.

### 3.1 The construction (ADR-0011, reaffirmed by ADR-0021)

```
DataHash   = SHA-256( salt ‖ JCS(section) )
EmployeeID = HMAC-SHA256( employeeKey_i, "id" )
UpdatedBy  = HMAC-SHA256( employeeKey_i, "actor" ‖ user_id )
```

- **Digest:** SHA-256, over the RFC 8785 JSON Canonicalization Scheme (JCS) serialization of the
  **whole** profile section (`PERSONAL`/`EMPLOYMENT`/`EDUCATION`/`ADDITIONAL`/`PAYROLL`), not a
  per-event delta `[prd: §5.2]`. Canonicalization is a **versioned interface**
  (`canonicalizationVersion` tag, ADR-0020) — the concrete library/version pin is still open
  (**G-24/PB-3**).
- **Content salt:** ≥128-bit CSPRNG, **new for every record** (never reused across versions or
  sections), stored **off-chain only**. Fabric's own documented guidance — that predictable data
  should carry a random salt so a matching hash cannot realistically be found by brute force — is
  the grounding fact this construction implements directly, even though it is applied to a plain
  world-state write rather than a collection `[docs: private-data-arch.rst#protecting-private-data-content]`.
- **Identifiers:** `employeeKey_i` is the sole key (Domain C, §1) — deterministic per employee so
  `EmployeeID`/`UpdatedBy` stay searchable and chainable, yet non-reversible without the key.
- **What "on-chain, as evidence" means here.** A plain world-state write, once committed, is part of
  every peer's immutable ledger and can be independently recomputed and compared later
  `[docs: ledger/ledger.md]` — the same evidentiary property the pre-reconciliation design attributed
  to a PDC's public hash, now achieved without a collection at all.

### 3.2 Why salt and key must never be the same primitive

- **Content needs a salt, not a key** — a random salt changes on every write, which is exactly right
  for content (non-deterministic by nature) and exactly wrong for an identifier (which must stay
  stable to remain searchable/chainable). This is why `DataHash` uses a per-record **salt** while
  `EmployeeID`/`UpdatedBy` use a per-employee **key** — one primitive cannot serve both roles
  (ADR-0011 Alternative E, rejected).
- **A shared master key for identifiers is the failure this design specifically avoids.** An earlier
  draft derived `employeeKey_i` from a single tenant-wide `pseudonymKey` via
  `HMAC-SHA256(pseudonymKey, tenantId ‖ employeeInternalId)` — a **pure, stateless function**.
  Compromising `pseudonymKey` would not just de-anonymize current employees; it would let an
  attacker **re-derive every `employeeKey_i` that tenant ever had**, including ones destroyed to
  satisfy a completed right-to-erasure request (T16b). **ADR-0021 (2026-08-06, human decision)
  closes this by removing `pseudonymKey` from the design entirely** — `employeeKey_i` is now
  independently random per employee, with no master secret that can regenerate it. There is no key
  anywhere whose compromise can retroactively reverse a completed erasure.

**Recommendation:** never anchor a value on-chain using a key that also decrypts PII (Domain B) or
documents (Domain B′) — a compromise of one domain must not cascade into another (§1's cross-domain
rule).

> Depth: chaincode contract shape and the off-chain hashing boundary live in `api-contracts.md`
> (ADR-0020); world-state key strategy lives in `data-model.md` (ADR-0011/ADR-0013).

---

## 4. Domain B — Application PII crypto (reuse, do not reinvent)

These mechanisms **already exist in the repos** and are the authoritative confidentiality layer for
the operational database. Fabric anchors digests on top of them; it does not replace them (G-09
keeps the operational database as system-of-record).

### 4.1 AES envelope encryption at rest

Sensitive operational-database fields (national-ID, tax-ID, and phone-class columns) are encrypted
at rest via native `AES_ENCRYPT`/`AES_DECRYPT` into shadow encrypted columns, with a `key_version`
column `[code: ems/pkg/db/encryption_plugins.go]`. Grounded specifics:

- **Cipher:** AES-256-CBC — a 64-hex-char key and 32-hex-char (16-byte) IV per version
  `[code: ems/pkg/encryption/config.go]`.
- **Versioned keys:** encryption is keyed by version; updates re-encrypt under the current key
  version while preserving already-encrypted historical versions `[code: ems/pkg/db/encryption_plugins.go]`.
- **Phased rollout:** `encrypt` → `read-from-encrypted` (fallback to plaintext) → `write-only`
  (plaintext column cleared), gated per tenant by a feature toggle
  `[code: ems/pkg/db/encryption_plugins.go]`.

**Blast radius (gap G-21).** Rollout is per-tenant and incomplete: for non-pilot tenants the
plaintext column may still be authoritative and the encrypted column may be empty. The HRIS write
path that computes `DataHash` must **read through the application layer**, not raw columns, or it
will digest stale/empty data.

### 4.2 Deterministic bank-field hashing (PBKDF2)

Bank fields are stored as **PBKDF2-SHA256** hashes for lookup, not encryption
`[code: ems/pkg/bankhasher/bankhasher.go]`. This is a **keyed, per-record-salted** construction —
structurally the same technique §3 requires for on-chain identifiers/digests, already implemented
in-repo for a different purpose.

### 4.3 Display masking

Presentation-layer masks over national-ID and phone fields already exist
`[code: ems/pkg/pii/mask.go]` — not a confidentiality control. **Recommendation:** never treat a
masked value as safe to anchor; it is still derived from the full PII value.

---

## 5. Domain B′ — Document-encryption key (`KEY_EMPLOYEE`, ADR-0016/ADR-0019)

A **per-employee** symmetric key encrypts every supporting document (identity documents, education
certificates, payslips) **before** it is uploaded to the IPFS Private Cluster; only the resulting
CID — never document content — is anchored on-chain (FR-30/31, ADR-0016). `KEY_EMPLOYEE` is a
distinct domain from both the application AES key (Domain B, tenant/version-wide, rotated) and the
identifier-pseudonymization secrets (Domain C, per-employee, destroyable) — it must never be derived
from, or used to derive, either.

- **Destroy-ability.** `KEY_EMPLOYEE` **must** be destroyable per employee (FR-26) — destruction is
  the entire erasure mechanism for IPFS-hosted content, because `unpin` is **not** `delete`: any
  cluster node that ever fetched a block may retain it indefinitely (ADR-0016). The compliance claim
  is "encrypted, and the only key is destroyed," never "the file is gone."
- **Re-encryption produces a new CID** (key rotation, compromise response) — the old CID is never
  updated in place; a new `EmployeeProfileRecord` version carries the new CID (ADR-0016, `data-model.md` §10).
- **Accidental-loss risk (OQ-5, PRD §12).** Losing `KEY_EMPLOYEE` by operator error is
  indistinguishable from an intentional erasure — the same accepted-limitation shape now mirrored by
  `employeeKey_i` (Domain C, §1) under ADR-0021.

---

## 6. Domain C — Identifier pseudonymization (`employeeKey_i` + per-record salt, ADR-0011/ADR-0019/ADR-0021)

See §1 for the domain summary and §3 for the digest/identifier construction that consumes it. Two
points worth restating here because they are the crux of this domain's security story:

- **`employeeKey_i` and the `DataHash` salt are structurally the same kind of secret** —
  independently random, CSPRNG, generated once (per employee for the key; per record for the salt),
  destroyed on that employee's erasure, never derived from anything else (ADR-0021). Domain C is
  now internally **uniform**: every secret in it follows the same lifecycle rule.
- **No domain-root secret exists.** There is no `pseudonymKey`, and no other tenant-scoped master
  that can regenerate a destroyed `employeeKey_i`. This is the property that closes T16b — see §3.2.

---

## 7. Fact vs. recommendation (summary)

| # | Statement | Type |
|---|---|---|
| Fabric uses PKCS11 to reach an HSM; TLS still needs file-based keys | `[docs: hsm.md]` | Fact |
| A committed world-state write is part of the immutable ledger and can be recomputed/compared later | `[docs: ledger/ledger.md]` | Fact |
| Predictable data must be salted vs brute force | `[docs: private-data-arch.rst]` | Fact |
| App PII is AES-256-CBC, versioned keys, phased rollout (Domain B, unchanged) | `[code: ems/pkg/db/encryption_plugins.go]`, `[code: ems/pkg/encryption/config.go]` | Fact |
| No PDC exists in this design; the digest is computed off-chain and never carries a value into chaincode | ADR-0015/ADR-0020 | Fact |
| `employeeKey_i` is independently random, no master key (`pseudonymKey` removed) | ADR-0021 | Fact (ratified 2026-08-06) |
| Keep all four key domains separate; never derive one from another | ADR-0019, reaffirmed ADR-0021 | Recommendation (enforced) |
| Salt every `DataHash` (small domains are reversible) | derived from D3 | Recommendation |
| Anchor via the app layer, never raw columns (rollout is partial) | gap G-21 | Recommendation |

---

## 8. Traceability

- **Capabilities:** D2, D3 (digest/salting, no D1/D4 — PDC and purge are retired) · D6 (identity,
  three orgs) · D13 (TLS) · D14 (HSM) · D15 (cert lifecycle) — see
  `../11-execution/knowledge-graph.md` §Layer D.
- **Frames:** CIA-Confidentiality & Integrity, Security-by-Design (secure defaults, four-domain key
  separation).
- **Gaps:** G-02 (re-scoped: profile-section digest, not event delta), G-09 (audit-overlay,
  unaffected), G-11 (role→OU map, open), G-15 (extended to four domains, closed via
  ADR-0019/ADR-0021), G-18/PB-1 (gateway trust, open), G-21 (rollout phase, open), G-24/PB-3
  (canonicalization pin, open).
- **Sibling docs:** minimal-disclosure crypto → `ZERO-KNOWLEDGE-PROOF.md`; erasure & data-map →
  `PRIVACY-BY-DESIGN.md`. Full threat model / residual risk on all four domains (T7, T8, T14, T16,
  T17, T18) → `../08-security/security-architecture.md`.

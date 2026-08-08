# OWASP ASVS — control mapping for the Fabric 2.5 + HRIS system

> **Re-derived 2026-08-06** against the ratified design in `prd-fabric-hris-2026-08-02/prd.md`
> (status: final) and **ADR-0011/ADR-0012/ADR-0013/ADR-0015/ADR-0019/ADR-0020/ADR-0021**. Every
> mention of a Private Data Collection, `PurgePrivateData`/`blockToLive`, or a two-organization
> consortium below described the pre-reconciliation design and is **superseded in substance** — see
> `../11-execution/grounding-gaps.md` and `../05-adr/README.md`.

> **What this is.** A context doc (gate **G1**) mapping the OWASP Application Security Verification
> Standard (Layer E frame · **OWASP ASVS**, mandated by `[brief: agents-guide.md]`) onto the concrete
> controls this system already has or will add. It covers the six ASVS chapters most relevant to a
> PII-handling write path + ledger overlay — **V2** Authentication, **V3** Session Management, **V4**
> Access Control, **V6** Stored Cryptography, **V7** Error Handling & Logging, **V9** Communication —
> and ties each to code or a Fabric capability (**D1..D15**).
>
> **What this is NOT.** Not the verification report and not a certification. ASVS is the checklist
> lens; findings are produced by the review skills and ratified in **G6** (`08-security/`,
> `security-architecture.md`/`control-matrices.md`). For the API-risk lens see `OWASP-API.md`; for the
> secure-default posture see `SECURITY-BY-DESIGN.md`.
>
> **Citations:** `[docs: …]` pinned Fabric corpus · `[code: …]` real repo · `[prd: …]` ratified PRD ·
> `[brief: …]` project brief · `[ASSUMPTION]` + gap ID (`../11-execution/grounding-gaps.md`).

## Target verification level — undecided

ASVS defines three levels (L1 opportunistic, L2 standard for apps handling sensitive data, L3 for
the highest-value apps). This system handles national-ID / tax / salary / bank PII, so **L2 is the
working target** `[ASSUMPTION G-14]`; L3 for the crypto and access-control chapters is worth
considering given the data class. The actual required level is a requirement decision (no ratified
NFR reaches this environment) and the NFR scope that would justify it is also open
`[ASSUMPTION G-08]`. Do not report an ASVS pass/fail against an unconfirmed level — state the level
as an assumption in any G6 finding. Note also that build-time work of any kind (a real verification
scan against running code) is gated behind sponsor decision **S-1** (the G8a/G8b split) — this doc
only maps the *design-time* control set.

> Chapter numbering follows ASVS 4.0. The requirement text is the OWASP standard's, not the Fabric
> corpus'; the *controls* mapped to each are cited to code/docs/PRD.

---

## V2 — Authentication

**What ASVS asks:** identities are authenticated with sufficient strength; credentials are stored
and rotated safely; no anonymous access to protected functions.

| Requirement area | Control in this system | Source |
|---|---|---|
| Every caller of the HRIS write path authenticated | Gateway-injected identity headers + S2S API-key auth | `[code: ems/internal/base/handler/base.go]` |
| Ledger actor authentication | **D6** — every Fabric actor, across **three** organizations (platform, per-tenant client, auditor — ADR-0012), is an X.509 identity from a CA, validated by an MSP; no un-enrolled actor can transact | [docs: security_model.md], [docs: identity/identity.md], [docs: msp.rst] |
| Chaincode-level caller verification | `RecordProfileSection` rejects any caller failing MSP/CID verification (FR-6) — a narrower, function-specific check on top of D6 | `[prd: §7 FR-6]`, ADR-0020 |
| Credential storage | Application-tier credentials and sensitive PII columns AES-encrypted at rest (Domain B, unchanged by this reconciliation) | `[code: ems/pkg/db/encryption_plugins.go]` |
| Certificate lifecycle | **D15** — expiry/rotation/CRL; MSP identities never expire, revoked only via CRL | [docs: certs_management.md], [docs: msp.rst] |

**Gap to verify:** the gateway header trust boundary is the authentication root for the HTTP tier —
if those headers can be spoofed, V2 is undermined for the whole app `[ASSUMPTION G-18/PB-1]`,
unaffected by this reconciliation and still a blocking pre-build gate. Enrollment mechanics and CA
config are out of scope here → `fabric-identity-security`.

**recommendation:** put client signing keys behind offline signing or an HSM so the app tier never
holds the private key. Axis: whether end users must keep keys secret vs the app-managed tier holding
them.

---

## V3 — Session Management

**What ASVS asks:** sessions are unpredictable, bound to the authenticated principal, and invalidated
on logout/timeout.

- **HRIS tier** carries session/identity per request, re-derived on every request rather than
  trusted from prior state `[code: ems/internal/base/app/context.go]` — this is the Zero-Trust
  "verify per request" property ASVS V3 rewards.
- **Fabric tier is sessionless by design, and now simpler than before.** There is no server-side
  session to hijack: each transaction proposal is individually signed by the submitter's X.509 key
  and evaluated against a policy [docs: security_model.md]. The anchoring write also has **no
  separate anchor-service session** to manage at all — it runs in-band, inside the HRIS write
  request's own lifecycle (ADR-0014), which is one fewer session-shaped surface than the
  pre-reconciliation Kafka-consumer design ever had. Replay/repudiation are addressed at the
  transaction layer (the idempotent-on-identical-`DataHash` rule, ADR-0020), not a session layer.

**Fact vs. recommendation.** "Each proposal is signed and policy-evaluated" is FACT (cited). "The
HTTP session token is unpredictable and bound to the principal" must be verified against the actual
gateway/SSO implementation — not asserted here.

---

## V4 — Access Control

**What ASVS asks:** access decisions are enforced server-side, deny-by-default, and object-level
ownership is checked (no IDOR).

This is the chapter with the most existing coverage and the most cross-over with `OWASP-API.md`
(BOLA/BFLA). Do not restate that mapping — the summary:

- **Object-level (IDOR):** the app-tier **Guard** enforces that a caller may only reach records it
  owns `[code: ems/internal/base/authz/guard.go]`. On-ledger, the equivalent is **D9** — chaincode
  reads the caller identity via CID (`GetID`/`GetMSPID`/attributes) to enforce FR-6, and **channel
  membership itself** (D7, ADR-0013) — not a collection's `memberOnly` policy — is what rejects a
  cross-tenant read before any application-level check runs.
- **Function-level (RBAC):** the app-tier RBAC gates on view/edit permissions and *excludes*
  Employee/Finance roles from broad reads `[code: ems/internal/base/authz/employee_rbac.go]`;
  on-ledger the analog is **D5** endorsement policy (`AND(Org1MSP.peer, OrgClient-<tenantID>MSP.peer)`,
  ADR-0012) + **D8** ACLs [docs: endorsement-policies.rst], [docs: access_control.md].
- **Deny-by-default:** Fabric ACL/policy defaults are meant to be overridden toward least privilege
  [docs: access_control.md], [docs: policies/policies.md] — see `SECURITY-BY-DESIGN.md` Tenet 1.

**Gap:** the HRIS-role → MSP-OU/attribute mapping that makes on-ledger access control match the app's
is unresolved `[ASSUMPTION G-11]`. A mismatch is an over- or under-exposure finding.

---

## V6 — Stored Cryptography

**What ASVS asks:** sensitive data encrypted at rest with strong, current algorithms; keys managed,
versioned, and protected; no predictable data left guessable.

| ASVS concern | Control | Source |
|---|---|---|
| PII encrypted at rest | AES-256 envelope encryption, versioned keys, phased rollout (Domain B) | `[code: ems/pkg/db/encryption_plugins.go]`, `[code: ems/pkg/encryption/config.go]` |
| Deterministic lookup without plaintext | PBKDF2-SHA256 hashing for bank fields | `[code: ems/pkg/bankhasher/bankhasher.go]` |
| Predictable PII not guessable on-ledger | **D3** — every `DataHash` carries a ≥128-bit CSPRNG salt, unique per record, off-chain only (ADR-0011) | [docs: private-data-arch.rst] |
| No confidential plaintext on the ledger, ever | **D2** — only a per-section salted digest and HMAC-derived pseudonymous identifiers on-chain; **no PDC exists in this design** — the boundary is enforced by the chaincode contract's argument shape (ADR-0020), not by a collection policy | [docs: ledger/ledger.md], ADR-0011/ADR-0020 |
| Signing-key protection | **D14** — HSM via PKCS#11 for MSP identity keys (not TLS keys) | [docs: hsm.md] |
| Identifier pseudonymization key | `employeeKey_i` — independently random, CSPRNG, generated once per employee, no master key (ADR-0021, closes the prior two-tier hierarchy's retroactive-reversal risk, T16b) | ADR-0011/ADR-0019/ADR-0021 |
| Document-encryption key | `KEY_EMPLOYEE` — per-employee, encrypts supporting documents before IPFS upload; destroy-ability is the erasure mechanism (ADR-0016) | ADR-0016/ADR-0019 |
| Key-domain separation | **Four** domains now (Fabric MSP/TLS, app AES, `KEY_EMPLOYEE`, `employeeKey_i`+salt); no domain may be derived from another | ADR-0019, reaffirmed ADR-0021, `[ASSUMPTION G-15]` (closed on the domain-count question, custody procedures still maturing) |

**Blast radius.** A predictable field (e.g. a national ID) digested as a bare hash is
brute-forceable off a small keyspace — **D3** salting is mandatory before any such value's section
is committed [docs: private-data-arch.rst]. This is the single most important V6 rule for the
overlay, and it is enforced by construction in this design (ADR-0011), not left to an implementer's
discretion.

**Pilot-rollout caveat.** Encryption is dual-write and only enabled for pilot tenants; plaintext is
still authoritative for others `[ASSUMPTION G-21]` `[code: ems/pkg/db/encryption_plugins.go]`. Do not
assume the encrypted columns are populated everywhere — the digest-computing write path must read
through the app layer, not raw columns.

---

## V7 — Error Handling & Logging

**What ASVS asks:** security-relevant events are logged, logs are tamper-resistant and do not leak
sensitive data, and errors fail closed.

- **Audit sink today:** the existing activity-log service records every non-read success — the
  existing audit surface `[code: ems/internal/base/service/ulms]`.
- **Tamper-evident audit is the overlay's core value:** **D12** — the ledger is immutable, and its
  per-section-per-employee chain provides an audit trail [docs: ledger/ledger.md]; a profile-section
  anchor is a non-repudiable record (**D2** digest) that the activity log alone cannot give. This is
  the CIA-Integrity chain in the knowledge graph.
- **Log hygiene (do-not-leak):** display masking already exists
  `[code: ems/pkg/pii/mask.go]`; the anchor path must never log the plaintext it is digesting.
  **There is also no per-field `changedFieldNames`-style metadata to leak** — the only revealed
  structural metadata is *which of the five known section types* changed, a required, non-secret
  field (`ProfileSection`, FR-8) — a coarser, not finer, disclosure surface than the
  pre-reconciliation design (`security-architecture.md` T9).

**[ASSUMPTION G-06/G-26]** retention of on-chain audit evidence vs. erasure of the underlying data is
a Privacy-by-Design tension resolved by crypto-shred (ADR-0015), not `PurgePrivateData` — see
`PRIVACY-BY-DESIGN.md` §4/§6 for why crypto-shred still leaves the Pasal 42 automatic-retention
obligation (**G-26**) open.

---

## V9 — Communication

**What ASVS asks:** all transport is authenticated and encrypted with current TLS; internal links are
not exempt.

| Link | Control | Source |
|---|---|---|
| Node ↔ node, client ↔ node | **D13** — TLS on every peer/orderer across all three orgs; SANs required on each server cert | [docs: enable_tls.rst], [docs: security_model.md] |
| Sensitive / admin links | Mutual TLS (`clientAuthRequired`) — **off by default**, turn on for cross-org + operator | [docs: enable_tls.rst] |
| Operations endpoint | Mutual TLS with client-cert auth, **no MSP** | [docs: security_model.md] |
| Proxies | Must be TLS-passthrough; a terminating proxy breaks node verification | [docs: enable_tls.rst] |
| HRIS ↔ gateway | Gateway TLS termination + header injection (trust boundary to verify) | `[ASSUMPTION G-18/PB-1]` |
| HRIS/Fabric ↔ IPFS Private Cluster | A **new** V9 surface (the cluster is a fourth trust boundary, ADR-0016) with no corpus-cited control yet — swarm-key gating restricts membership, but transport hardening for the cluster API itself is not yet ADR'd | `[ASSUMPTION]`, owned by `fabric-architect`/`fabric-engineer` |

Full cited TLS item list (SANs, client-auth, passthrough) lives in
`fabric-security-review/references/config-hardening.md` — do not duplicate.

---

## Tooling — SAST for the verification pass

The **Semgrep / Guardian MCP** (`mcp__plugin_semgrep_guardian__*`: SAST findings, secrets, supply
chain) is the available static-analysis tool for the app-tier ASVS chapters (V5 input handling, V6
key/secret handling, V7 logging leaks) and for CI gating against the existing application code. It
does **not** analyze chaincode controls or Fabric config — those go through `fabric-security-review`.
Recording the verification level the scan was run at is required (see the L2 assumption above,
G-14), as is recording that any such scan is evidence-gathering, not a build gate, until sponsor
decision **S-1** lifts the design-only boundary.

---

## Chapter → capability / control matrix

| ASVS chapter | Fabric capability | Existing code control | Gaps |
|---|---|---|---|
| V2 Authentication | D6, D15 | gateway/SSO auth, AES creds | G-18/PB-1 |
| V3 Session | (sessionless ledger; no anchor-service session, ADR-0014) | per-request app context | — |
| V4 Access Control | D5, D7, D8, D9 | Guard/IDOR + RBAC | G-11 |
| V6 Stored Crypto | D2, D3, D14 (no D1 — no PDC) | AES envelope + bankhasher; `employeeKey_i`/`KEY_EMPLOYEE` (ADR-0019/0021) | G-15 (closed on domain count), G-21 |
| V7 Logging | D2, D12 | activity log + PII masking | G-06, G-26 |
| V9 Communication | D13 | gateway TLS; IPFS cluster leg open | G-18/PB-1, IPFS `[ASSUMPTION]` |
| target level | — | — | **G-14, G-08** |

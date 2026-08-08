# Test Strategy — HRIS PII-anchoring prototype (G7)

> **Re-derived 2026-08-06** against the ratified PRD (`prd-fabric-hris-2026-08-02/prd.md`) and the
> re-derived STRIDE model in `../08-security/security-architecture.md` (T1–T21). Supersedes the
> pre-reconciliation test plan in substance: no PDC test, no Kafka test, no `changedFieldValues`
> commitment test — replaced by section-based digest tests, the two-tier `employeeKey_i` hierarchy,
> the four-domain key model (ADR-0019), and IPFS. Per `rencana-rekonsiliasi.md` Gelombang 4 item #28.

> **ADR-number note.** `ADR-0011`–`ADR-0020` are the numbers `rencana-rekonsiliasi.md` Gelombang 3
> pre-reserved for the decisions this rework depends on; only **ADR-0015** and **ADR-0019** (this
> agent's scope) exist on disk as of 2026-08-06. Every other number cited below (0011–0014, 0016–0018,
> 0020) is marked **(pending)** at first use and refers to a `fabric-architect`/`fabric-engineer`
> decision **not yet authored** — cited for the ratified PRD fact it will formalize, not as an
> existing document.

A full test-pyramid **plan** (design-only — no test code before G9) for the prototype designed in
[`../00-architecture/solution/`](../00-architecture/solution/) as amended by the PRD. The
load-bearing artifact is the **control→test coverage matrix** (§3): every G6 security control maps
to ≥1 test. Method depth: `test-strategy`, `api-test-automation`, and `fabric-performance` (Caliper)
skills — cross-referenced, not restated. Pairs with
[`implementation-backlog.md`](implementation-backlog.md) (its QA-* items execute these layers).

## 0. Priority

**P0 — the confidentiality-invariant test (`ST-1`) is the single highest-priority test, and it
remains ONE test, not split.** This is a direct consequence of §5.2's salt+HMAC construction being
adopted (PRD §5.2, errata E-1 ✅ adopted): because `DataHash`/`EmployeeID`/`UpdatedBy` are
non-reversible by construction, INV-1 is satisfiable **whole**, so ST-1 does not need to be split
into a weaker "ST-1a/ST-1b" pair the way a pre-adoption world would have required
(`rencana-rekonsiliasi.md` §6 — that recommendation is explicitly **no longer valid**). What changes
here is **ST-1's surface**, which is now **wider** than the pre-reconciliation version:

- **Removed from the scan:** PDC state and transient-field payloads — neither exists in the ratified
  topology (no PDC anywhere, FR-20/21).
- **Added to the scan:** the `IPFSCIDs` field(s) on `EmployeeProfileRecord` and every pseudonymous
  identifier field (`EmployeeID`, `UpdatedBy`) — proving they carry no reversible derivative, not
  merely no plaintext.
- **Widened scope:** the scan must sweep **every tenant channel**, not one shared channel — a direct
  consequence of channel-per-tenant (FR-20). A confidentiality pass on Tenant A's channel says
  nothing about Tenant B's.

On an immutable ledger a PII leak is unrecoverable, so `ST-1` gates every build phase: no
profile-write path merges until `ST-1` passes on **every** provisioned tenant channel. All other
tests are P1 (security/correctness) or P2 (performance).

## 1. Test pyramid

### Unit (`U-#`) — fast, isolated

> **Superseded-in-part — 2026-08-07, found at `QA-1`.** U-2/U-3 below describe the pre-ADR-0021
> two-tier `pseudonymKey`→`employeeKey_i` HMAC hierarchy. ADR-0021 (Accepted, 2026-08-06) removed
> `pseudonymKey` from the design entirely: `employeeKey_i` is now an independently-random CSPRNG
> value, generated once per employee (`write-path-integration/keystore/keystore.go`'s
> `EmployeeKeyStore`), no master key anywhere. Confirmed by grep: zero `pseudonymKey` occurrences
> in shipped code outside a comment stating its absence. U-2/U-3's underlying FUNCTIONAL assertions
> (a stable, non-recoverable per-employee identifier; domain-separated `"id"`/`"actor"` derivation)
> still hold and are still covered by real tests (`gateway-client/digestbuilder_test.go`,
> `keystore/keystore_test.go`) — only the literal HMAC-over-`pseudonymKey` formula and the
> "regression guard against `pseudonymKey` leakage" framing are stale text this table was never
> updated to match. Not silently reinterpreted; left visible here for anyone who reads this row
> literally.

| ID | Target | Asserts | Ties to |
|----|--------|---------|---------|
| U-1 | `DataHash` builder | `SHA-256(salt‖JCS(section))` per `ProfileSection`; salt ≥128-bit CSPRNG, **new per record**, never returned/logged | PRD §5.2 FR-3 |
| U-2 | `employeeKey_i` derivation *(stale formula, see note above — ADR-0021 superseded `pseudonymKey`-based HMAC with an independently-random CSPRNG value)* | Deterministic per employee; **not** recoverable from its outputs without the key itself | PRD §5.2 FR-5, ADR-0019 Domain C, ADR-0021 |
| U-3 | `EmployeeID`/`UpdatedBy` builder | `HMAC-SHA256(employeeKey_i, "id"/"actor"‖user_id)` — asserts the on-chain identifier is computed from `employeeKey_i` *(the "never directly from `pseudonymKey`" framing is moot post-ADR-0021 — `pseudonymKey` no longer exists to regress into)* | PRD §5.2 FR-5, errata E-14, ADR-0021 |
| U-4 | **RFC-8785 JCS canonicalization** over the section JSON | writer & verifier produce byte-identical input for equivalent JSON (key order, unicode, numbers) — **closes PB-3/G-24** | PRD §5.2, FR-2 |
| U-5 | chaincode ABAC *(divergence found, see `G-30`/`ADR-0005`'s implementation-status note, 2026-08-07 — the shipped chaincode implements org-level MSP allow-listing, `ADR-0005`'s explicitly-REJECTED Option B, not the `GetAttributeValue` role/tenant ABAC it decided)* | **As actually built:** MSP-ID org allow-list gates read/write; **explicit reject if caller identity fails MSP verification** (FR-6) — real, tested, passing. **As `ADR-0005` specifies but is NOT built:** role/tenant enrolled-attribute ABAC | ADR-0005, D9 |
| U-6 | no-duplicate-on-identical-content | Re-submitting a section whose canonical JSON is byte-identical to the last version produces **no new record** — prevents ledger bloat (FR-9) | PRD §7 FR-9 |
| U-7 | `PrevHash` chaining | Chained **per section per employee** (not one chain per profile); `Version` increments; a different section for the same employee starts its own independent chain | PRD §4.2 FR-4 |
| U-8 | record identity | `RecordID` is a fresh UUID v4; `Timestamp` present; `ProfileSection` validated against the five allowed values, others rejected | PRD §7 FR-7, FR-8 |

### Integration (`IT-#`) — real Fabric + services
| ID | Flow | Asserts | Ties to |
|----|------|---------|---------|
| IT-1 | end-to-end anchor, **in-band** (ADR-0014) | HRIS profile-write path → digest computed off-chain in the same process → `RecordProfileSection` committed on **both** Org1 and Org2 peers — **no Kafka hop, no anchor-service** | ADR-0014, PRD §5.2 Kelompok B′ |
| IT-2 | endorsement | Single-org submit **rejected**; `AND(Org1MSP.peer, Org2MSP.peer)` required (Org3 auditor is read-only, does not endorse) | ADR-0012 (pending), D5 |
| IT-3 | verify path (Kelompok B′) | Read-only call returns **only** `{DataHash, Version, Timestamp, UpdatedBy}`; client recomputes and compares locally; result **distinguishes** hash-mismatch from record-not-found (FR-15) | PRD §5.2 FR-35, T7 |
| IT-4 | salt-delivery channel (FR-36) | Authorized retrieval of a specific version's *salt* is a **separate, audited** call from the verify endpoint (FR-14 remains: verify never returns salt); scoped to data ownership — the employee for their own record, the auditor within audit scope, and (FR-37) each org's own peer for its own read | PRD §5.2 FR-36/37 |
| IT-5 | erasure (ADR-0015) | Executing the four-part crypto-shred (DB field delete + `KEY_EMPLOYEE` destroy + salt destroy + `employeeKey_i` destroy) leaves on-chain state **byte-for-byte unchanged**; the surviving `DataHash`/`EmployeeID`/`UpdatedBy`/CID become permanently unopenable; a signed off-chain deletion certificate is recorded (FR-29) | ADR-0015 |
| IT-6 | audit trail + **NEW** partial-failure detection | `UpdatedBy`+`Timestamp`+`Version` present on every committed record (unchanged assertion); **NEW:** when the operational-DB write succeeds but the paired `RecordProfileSection` call fails/times out, the failure is **at minimum detectable** (e.g. surfaced as an error/metric) — this test intentionally does **not** yet assert auto-remediation, because the compensating design (T6b) is an open `fabric-engineer` item, not yet specified | T6, `security-architecture.md` T6 |
| IT-7 | tenant onboarding (FR-24) | New tenant's channel is provisioned, chaincode is installed/approved/committed on it, and its MSP identities are registered; onboarding credentials used for this **cannot** perform any operation on a *different* tenant's channel (T20 scoping check) | PRD §7 FR-24, T20 |
| IT-8 | **cross-tenant isolation with a REAL second tenant** | A **genuinely provisioned** second tenant, with its **own** channel and its **own** Org2-equivalent peer, attempts to read Tenant A's channel data → **rejected** by MSP/channel-membership enforcement (FR-21). **This test is not satisfied by a mock or a single-tenant simulation** — errata **E-11** is explicit that the isolation claim must be demonstrated, not asserted, or it must be downgraded from "cryptographic" to "identity-bound/chaincode-enforced" in every claim that cites it | PRD §7 FR-20/21, errata E-11 |
| IT-9 | IPFS round-trip + erasure basis | Upload of a `KEY_EMPLOYEE`-encrypted document succeeds; its CID is correctly anchored to the relevant section (FR-32); fetch+decrypt succeeds pre-erasure; **post-`KEY_EMPLOYEE`-destruction, the same ciphertext (if still physically retrievable — `unpin ≠ delete`) fails to decrypt** — this is the concrete proof that the erasure claim is a key-destruction claim, not an object-removal claim (T18) | PRD §7 FR-30..33, T17/T18 |
| IT-10 | overlay availability, **3-node Raft** | One of three orderer nodes failing does **not** halt ordering (crash-fault-tolerant quorum); HRIS operational-DB writes are unaffected regardless | PRD §5.1, T12 |

### Contract (`CT-#`) — drive the profile-write/verification API (Newman / api-test-automation)

> **Superseded premise — 2026-08-07, found at `QA-3`.** This section's header and every "Endpoint"
> column below assume an HTTP endpoint surface for Newman to drive. `ADR-0014` (in-band recording)
> decides exactly zero anchor-service deployable units — write (`gatewayclient.
> SubmitRecordProfileSection`), verify (`gatewayclient.Verify`), salt-delivery (`keystore.
> SaltHandoff.RequestSalt`), and onboarding (`fabric-network/tools/tenantprovision`) are all direct
> in-process Go calls over Fabric Gateway gRPC, not HTTP endpoints — the same class of stale
> assumption already known for `PB-1`/`ST-2`'s "API-gateway identity-header trust boundary" (which
> remains genuinely open regardless, concerning the HRIS application's own separate public edge,
> untouched by this finding). `qa-tests/contract-coverage-report.md` maps each `CT-#`'s actual
> *property* onto the Go-level test that embodies it instead of a Newman collection that has
> nothing to address.


| ID | Endpoint | Asserts | Ties to |
|----|----------|---------|---------|
| CT-1 | verification (read-only, Kelompok B′) | Returns only the stored digest/metadata fields; **never** accepts a section-plaintext argument (FR-34); **never** returns *salt* under any response path (FR-14) | T7, T8, API3 |
| CT-2 | salt-delivery (FR-36) **BOLA** | An employee can fetch *salt* only for their **own** records; an auditor's fetch is bounded to their audit scope; cross-employee/cross-tenant fetch is rejected | T13, API1 |
| CT-3 | tenant onboarding **BFLA** | A non-onboarding-scoped caller cannot trigger channel/chaincode provisioning for any tenant | T20, API5 |
| CT-4 | auth | Missing/invalid caller identity ⇒ rejected at the gateway and/or chaincode layer | T1, T2, API2 |

### Performance (`PT-#`) — Caliper (fabric-performance)
| ID | Scenario | Metric | Ties to |
|----|----------|--------|---------|
| PT-1 | profile-write throughput | TPS at load points **200/500/1000/2000** writes/sec, per PRD NFR-1 (<3s @ 500 TPS) — **real measurement, not the placeholder Table 4.3 figures** (PRD §3.1) | NFR-1, ADR-0018 (pending) |
| PT-2 | profile-write latency | p50/p95/p99 endorse→commit, **`AND(Org1,Org2)`** endorsement path | NFR-1 |
| PT-3 | verify-read latency | LevelDB read latency for the read-only Kelompok B′ call — must stay sub-second, no ordering required (FR-12) | NFR-2 |
| PT-4 | per-tenant channel scaling | throughput/latency sensitivity as the **number of provisioned tenant channels** grows — a scaling axis that did not exist under single-channel+PDC | T21, NFR-10 |
| PT-5 | batched bulk operation | e.g. an annual salary-band adjustment across thousands of employees is handled via batching, not one-by-one blocking calls | PRD NFR-10 |

### Security (`ST-#`)
| ID | Test | Asserts | Ties to |
|----|------|---------|---------|
| **ST-1** | **confidentiality invariant (P0), expanded surface** | Full-ledger scan — **world state + tx args + chaincode event payloads + `IPFSCIDs` + all pseudonymous-identifier fields** — swept across **every provisioned tenant channel** — proves **0 bytes PII plaintext / no reversible derivative** anywhere on-chain. PDC is **not** part of the scan (none exists); the scan is wider, not narrower, than the pre-reconciliation version. **RUN 2026-08-07 (`QA-5`): FAIL** — `fabric-network/tools/pilscan` found 31,895 identifier-shape violations (test-fixture strings recorded as `employeeID`/`updatedBy` instead of real pseudonyms, from test suites that bypassed the real pseudonymization pipeline) — but **triple-independently-confirmed zero actual PII content**. Root cause tracked as new grounding gap **`G-31`**: the chaincode has no shape-validation defense-in-depth. See `implementation-backlog.md` `QA-5` row for full detail. | **T7**, PRD §5.2, INV-1 |
| ST-2 | header-trust / tenant isolation (gateway) | Forged tenant/user identity header rejected; requires the PB-1/G-18 pen-test — **unaffected by this rework, still open** | T1, PB-1/G-18 |
| ST-3 | cert revocation | Revoked X.509 identity rejected; CRL enforced, across **three** orgs. **RUN 2026-08-07 (`QA-5`): BLOCKED**, and surfaced two independent structural findings beyond the CA containers being down: `NET-2`'s Fabric-CA PKI root certs are not actually this network's live trust anchor, and no CRL distribution mechanism exists anywhere in this repo — `T2` is unmitigated in practice, not merely untested. Tracked as new grounding gap **`G-32`**. | T2, D15 |
| ST-4 | transport | Non-mTLS/plaintext connection refused, on Fabric endpoints **and** the IPFS Private Cluster boundary | T3, T19, D13 |
| ST-5 | tamper detection, **five sections including `ADDITIONAL`** | Altered ledger value fails verify for **all five** `ProfileSection` values — **adds a manipulation scenario for `ADDITIONAL`** (e.g. tamper with marital-status/dependent data), labeled **SC-F** to avoid reusing the already-overloaded `SC-A..E` labels (errata **E-9**'s warning) — closing errata **E-2**, the gap where the thesis claimed "100% on five sections" but tested only four | **P1**, T4, T5, errata E-2 |
| ST-6 | brute-force resistance | `DataHash` does not reveal section content; salt entropy ≥128-bit; salt absent from every on-chain field and every API response | T8 |
| ST-7 | metadata-boundary regression guard | `EmployeeProfileRecord`'s on-chain schema contains **no field beyond** `{RecordID, EmployeeID, ProfileSection, DataHash, PrevHash, Version, Timestamp, UpdatedBy, IPFSCIDs}` — a regression guard against ever reintroducing a `changedFieldNames`-style field. **Disposition resolved 2026-08-06 (S-4):** formally moot, not a live control requiring full coverage — still run as a regression guard per §3's own note | T9 (moot) |
| ST-8 | Kelompok B′ protocol compliance | The verification path never accepts a plaintext argument (FR-34) and never returns *salt* (FR-14); there is **no separate anchor-service** to audit for over-broad read-back. **Disposition resolved 2026-08-06 (S-4):** formally moot — no anchor-service exists to audit; still run as a regression guard | T10 (moot) |
| ST-9 | SAST gate | semgrep/Guardian clean on the chaincode + the HRIS profile-write path | T13, API8 |
| ST-10 | key-domain separation, **four domains** | No domain's key derives, wraps, or is derivable from another: Fabric signing (A) cannot decrypt PII or IPFS documents; app AES (B) cannot sign or pseudonymize; `KEY_EMPLOYEE` (B′) cannot sign or pseudonymize identifiers; `pseudonymKey`/`employeeKey_i` (C) cannot decrypt IPFS documents or operational PII | T14, ADR-0019 |
| ST-11 | secure-config | mTLS on, no default creds, channel capability set correctly, **and — new assertion — no PDC is defined on any channel at all** (a config-hardening check against accidentally reintroducing one) | API8, T11 (retired-object regression guard) |
| ST-12 | `employeeKey_i`/`pseudonymKey` isolation — **hierarchy-aware, two-part** | **(a)** destroying/leaking one employee's `employeeKey_i` does **not** affect any other employee's `EmployeeID`/`UpdatedBy` verifiability — direct proof of the hierarchy's containment claim (T16a). **(b)** `pseudonymKey` never leaves HSM-protected custody and is never included in any backup outside that boundary — and, as a **documented residual-risk demonstration, not a pass/fail security gate**, a `pseudonymKey` leak plus an enumerable `employeeInternalId` **is** sufficient to recompute `employeeKey_i` for **any** employee at that tenant, including already-erased ones (T16b) — this test exists to keep the disclosed residual honest, not to "fix" it | T16, ADR-0019 |
| ST-13 | IPFS erasure-basis proof | After `KEY_EMPLOYEE` destruction, a ciphertext object retrieved by CID (if still physically present in the cluster — `unpin ≠ delete`) **fails to decrypt**; CID possession alone, without `KEY_EMPLOYEE`, never yields plaintext | T17, T18 |
| ST-14 | tenant-isolation, **real second tenant** (P3) | The actual security-test embodiment of P3: a genuinely provisioned second tenant's peer attempting a cross-tenant read is rejected by MSP/channel-membership — **same provisioning prerequisite as IT-8**; without it, P3 must be reported as unverified, not passed (errata E-11) | **P3**, T20, errata E-11 |

## 2. Environments & tooling

- **Local Fabric test network: three organizations, each with the ratified role** — Org1 (2 peers +
  3-orderer Raft), Org2 (1 peer), Org3 (1 read-only peer) — for IT/ST. **No PDC is defined anywhere
  in this network.**
- **A second, fully independent tenant (its own channel, its own Org2-equivalent peer) must be
  provisioned for IT-8/ST-14 to be reproducible** — this is a new environment requirement introduced
  by errata **E-11**; the pre-reconciliation single-tenant network cannot exercise P3 at all, it can
  only assert it by construction.
- An **IPFS Private Cluster** (not public IPFS) for IT-9/ST-13/ST-4's cluster leg.
- Caliper harness for PT (bound to Fabric **2.5**, not 2.2 — a build-time defect flagged in
  `rencana-rekonsiliasi.md` §4.1, not this document's to fix); Newman for CT.
- SAST: **semgrep/Guardian** MCP in CI (ST-9). No PII fixtures use real employee data — synthetic
  only, in the generic register (no real tenant/company names in any fixture or test description).

## 3. Control → test coverage matrix (the G7 gate)

Every live G6 control (STRIDE T1–T8, T12–T14, T16–T21 + ASVS + API Top-10) maps to ≥1 test. T9/T10
are listed for traceability but their disposition is *likely moot pending S-4*, not a live control
requiring full coverage; T11/T15 are retired and carry no test rows.

| Control | Test(s) |
|---------|---------|
| T1 gateway spoofing | ST-2, CT-4 |
| T2 stolen X.509 | ST-3 |
| T3 MITM | ST-4 |
| T4 ledger tamper / Org2-independence | IT-2, ST-5 |
| T5 canonicalization (PB-3/G-24) | U-4 |
| T6 repudiation / in-band partial failure | IT-6 |
| **T7 PII on-chain** | **ST-1 (P0)**, IT-3, CT-1 |
| T8 brute-force | ST-6, U-1 |
| T9 field-name metadata (likely moot) | ST-7 |
| T10 read-back protocol (likely moot) | ST-8, IT-3 |
| T12 orderer availability | PT-1, IT-10 |
| T13 chaincode ABAC | U-5, CT-2/CT-3, ST-9 |
| T14 key domains (four) | ST-10 |
| T16 `employeeKey_i`/`pseudonymKey` | U-2, U-3, ST-12 |
| T17 IPFS CID retrieval | ST-13, IT-9 |
| T18 `unpin ≠ delete` erasure basis | ST-13, IT-9, IT-5 |
| T19 IPFS cluster operator boundary | ST-4 (transport leg only — membership/ops model is `[ASSUMPTION]`, no test can close it until an ADR exists) |
| T20 tenant-onboarding privilege | IT-7, CT-3 |
| T21 chaincode lifecycle fan-out | PT-4 (throughput/scaling signal only — a version-skew detector is a build-time tooling item, not yet testable here) |
| ASVS V2 auth | CT-4, ST-2, ST-3 |
| ASVS V3 session | CT-4 |
| ASVS V4 access control | CT-2, U-5, IT-7 |
| ASVS V6 crypto | ST-1, ST-6, ST-10, U-1, U-2, U-3, U-4 |
| ASVS V7 logging/audit | IT-6, ST-7 |
| ASVS V9 comms | ST-4 |
| API1 BOLA | CT-2 |
| API2 broken auth | CT-4 |
| API3 BOPLA | CT-1, ST-8 |
| API5 BFLA | CT-3 |
| API8 misconfig | ST-9, ST-11 |
| API10 unsafe consumption (re-scoped to IPFS) | IT-9 |

**Coverage:** 16/16 live STRIDE controls (T1–T8, T12–T14, T16–T21), 6/6 ASVS categories, 5/5 live
API-Top-10 rows (API10 re-scoped) → ≥1 test each. T9/T10 carry a test each for regression purposes
but are explicitly not counted as "closed" controls. T11/T15 carry no test rows (retired).

## 4. Assumptions

- `[ASSUMPTION]` **G-08** — precise performance targets beyond NFR-1's <3s@500TPS threshold (peak
  volume, tenant count at scale) remain open; PT-* load points (200/500/1000/2000) are the PRD's own
  stated benchmark plan, not yet a fixed ceiling.
- `[ASSUMPTION]` **G-14** — the pass/fail acceptance bar per layer is provisional pending real
  acceptance criteria.
- `[ASSUMPTION]` — no gap ID yet exists for the IPFS-cluster-membership question (T19) or the
  channel-per-tenant onboarding-privilege design (T20) or chaincode-lifecycle-fan-out tooling (T21);
  these are flagged here for `dsrm-researcher`/`fabric-architect`, not invented as fact.
- **Sponsor decision S-4** (dissolve PB-2/G-23, and by extension T9/T10's formal status) was
  **ratified 2026-08-06** — ST-7/ST-8 are formally moot as of that date (still run as regression
  guards per §3, not counted as live-control coverage).
- **Errata E-11's provisioning prerequisite is a hard blocker for IT-8/ST-14**, not a nice-to-have: a
  single-tenant test network cannot produce evidence for P3, only an assertion.
- Tests are a **plan**; test code is Phase-4 (post-G9) work (backlog QA-*).

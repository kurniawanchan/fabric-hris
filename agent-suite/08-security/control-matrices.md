# Control Matrices — OWASP ASVS · API Top-10 · Zero-Trust (G6)

> **Re-derived 2026-08-06** alongside [`security-architecture.md`](security-architecture.md) against
> the ratified PRD (`prd-fabric-hris-2026-08-02/prd.md`). Threat IDs below are the `T#` rows in that
> document's §2 (T1–T21, with T9/T10/T11/T15 disposition as stated there — T11/T15 retired, T9/T10
> likely dissolved pending sponsor S-4). **ADR-number note:** only ADR-0015/ADR-0019 exist on disk;
> other 0011–0020 numbers cited below are **(pending)** `fabric-architect`/`fabric-engineer` decisions
> per `rencana-rekonsiliasi.md` Gelombang 3, cited for the ratified PRD fact they will formalize.

Every control traces to a **threat** (a `T#` from [`security-architecture.md`](security-architecture.md) §2)
and a **mechanism** (a Fabric capability `D#` or an app control `[code:]`). This is the artifact the G6
gate checks. Depth: `context/OWASP-ASVS.md`, `context/OWASP-API.md`. Enforcement tooling: **semgrep/
Guardian** MCP (SAST) at build.

## 1. OWASP ASVS (relevant categories)

Target ASVS level remains `[ASSUMPTION] L2` (gap G-14) pending a real NFR — unaffected by this
rework.

| ASVS category | Requirement (this system) | Control / mechanism | Counters | Status |
|---------------|---------------------------|---------------------|----------|--------|
| **V2 Authentication** | Every caller of the profile-write path and the Fabric network is authenticated | Gateway-injected identity (`[ASSUMPTION]` PB-1/G-18 still open) + Fabric enrollment via CA (D6) for **three** orgs, not two | T1, T2 | Reuse + `[ASSUMPTION]` PB-1 |
| **V3 Session mgmt** | No long-lived ambient sessions on the profile-write→anchor path | In-band call, same request lifecycle as the HRIS write (ADR-0014) — no separate service session to manage at all now | T1, T6 | Design — simplified vs. the prior async model |
| **V4 Access control** | Enforce object- and function-level authz | Chaincode **ABAC via CID** (D9); existing IDOR **Guard** pattern `[code: ems/internal/base/authz/guard.go]`; **no PDC `memberOnly` control exists or is needed** — channel-per-tenant is the sole isolation layer | T13, T20 | Reuse + design |
| **V6 Cryptography** | Approved algorithms; keys protected & separated across **four** domains | Existing application AES-256-CBC at rest `[code: ems/pkg/db/encryption_plugins.go]`; **`SHA-256(salt‖JCS(section))` + ≥128-bit per-record salt** (PRD §5.2); **`HMAC-SHA256(employeeKey_i, …)`** for `EmployeeID`/`UpdatedBy`; TLS (D13); **four-domain key separation** (ADR-0019, D14) | T7, T8, T14, T16, T17 | Design — construction re-derived from PRD §5.2, not ADR-0001's per-event scheme |
| **V7 Errors & logging** | Auditable, no PII in logs | Existing activity log `[code: ems/internal/base/service/ulms]`; immutable per-section ledger audit trail (D12); PII never logged/echoed; **no `changedFieldNames`-style metadata exists to leak** (T9, likely moot) | T6, T9 | Reuse + design |
| **V9 Communications** | Encrypted transport everywhere, **including the IPFS Private Cluster boundary** | **mTLS** on all Fabric endpoints (D13) `[docs: enable_tls.rst]`; TLS/authenticated access to the IPFS cluster is a **new V9 surface (TB5)** with no existing corpus-cited control — `[ASSUMPTION]`, owned by the pending IPFS topology ADR | T3, T19 | Design + `[ASSUMPTION]` (IPFS leg) |

## 2. OWASP API Security Top-10 (profile-write path + Fabric gateway client)

> The prior "anchor-service" surface (BOLA/BFLA rows keyed to `GET /v1/anchors/...`) is retired along
> with T10/T15 objects — there is no longer a separate anchor-service API. The rows below key to the
> **read-only verification endpoints** of Kelompok B′ (FR-34..37) and the **onboarding** surface (T20),
> which are the API-shaped attack surfaces that actually exist in the ratified design.

| API risk | Exposure | Control / mechanism | Counters | Status |
|----------|----------|---------------------|----------|--------|
| **API1 BOLA** | A read-only verification call for one employee's section digest could leak another employee's/tenant's `DataHash`/`Version`/`Timestamp`/`UpdatedBy` | Ownership check bound to caller identity (Guard pattern `[code: guard.go]`) + chaincode ABAC (D9); channel-per-tenant means a cross-tenant read is rejected at the **MSP/channel-membership layer**, before it ever reaches an ownership check (FR-21) | T13, T20 | Design |
| **API2 Broken auth** | Verification/profile-write endpoints | Gateway auth (V2) + Fabric CA enrollment (D6) | T1, T2 | Design |
| **API3 BOPLA (property)** | A verification call must never accept or echo section plaintext or the *salt* | **Kelompok B′ (FR-34/35)**: the verification operation is **read-only**, returns only `{DataHash, Version, Timestamp, UpdatedBy}`; recomputation happens **client-side**; *salt* is delivered **only** through the separate, audited channel of FR-36, never via the verify endpoint (FR-14) | T7, T8 | Design |
| **API5 BFLA** | Employee/finance-scoped callers must not reach chaincode functions beyond `VerifyProfileIntegrity`/history reads | Scope/role exclusion mirroring existing RBAC exclusions; chaincode-side CID/ABAC gates `RecordProfileSection` to the HRIS write path's own identity, never an end-user identity directly | T13 | Reuse |
| **API8 Misconfiguration** | Fabric/channel/TLS defaults, **and now the IPFS cluster defaults** | Secure defaults: mTLS on, deny-by-default channel MSP policy, **no PDC config surface to get wrong at all**; IPFS cluster hardening (swarm-key rotation, private-only membership) is `[ASSUMPTION]`, no ADR yet (T19) | T1, T19 | Design + `[ASSUMPTION]` (IPFS leg open) |
| **API10 Unsafe consumption** | ~~anchor-service consumes Kafka~~ | **RETIRED as an API10 row** — there is no Kafka consumer and no anchor-service; the equivalent unsafe-consumption surface is now **IPFS CID handling** (validate CID format before fetch/store; never trust a CID from an unauthenticated source) | T17 | Design (re-scoped) |

## 3. NIST Zero-Trust control checklist

| Tenet | Control | Mechanism | Counters | Status |
|-------|---------|-----------|----------|--------|
| Verify explicitly | Per-request X.509 identity | CID at chaincode (D9), **three orgs** | T1, T13 | ✓ |
| Least privilege | Per-tenant-scoped onboarding creds; role→attribute | ABAC (D9); IDOR Guard `[code: ems/internal/base/authz/guard.go]` (ADR-0005) | T13, T20 | ✓ / `[ASSUMPTION]` T20 |
| Assume breach | Immutable per-section audit; **four-domain** key isolation | D12; key-domain separation (D14, ADR-0019) | T6, T14 | ✓ |
| Microsegmentation | **Channel-per-tenant, no PDC underneath** | D7 only (D1 not invoked); mTLS (D13) | T11 (retired) | ✓ — simplified, one less mechanism to misconfigure |
| No implicit trust | mTLS everywhere; gateway trust flagged; **IPFS cluster trust flagged** | D13; `[code: ems/internal/base/handler/base.go]` | T1, T3, T19 | ✓ / `[ASSUMPTION]` PB-1, T19 |
| Continuous verification | Cert lifecycle/CRL; **per-channel chaincode-version parity** (new, at scale) | D15; T21 tooling `[ASSUMPTION]` | T2, T21 | ✓ / `[ASSUMPTION]` T21 |

## 4. Control-coverage summary

- **Threats in the model:** 21 (T1–T21). Of the pre-reconciliation 16: **T11 and T15 retired**
  (objects no longer exist); **T9 and T10 kept, marked likely-dissolved pending sponsor S-4**; **T16
  re-verified with a changed residual** (T16a downgraded, T16b newly elevated); T1–T8, T12–T14
  carried forward with mechanism citations updated to the ratified §5.2 construction and the
  three-org/channel-per-tenant topology. **Five new threats** (T17–T21) cover IPFS confidentiality/
  erasure-basis/operator-boundary and channel-per-tenant-at-scale (onboarding privilege, chaincode
  lifecycle fan-out).
- **Threats with ≥1 mapped control:** 19/21 addressed by a named control (T9, T10 excluded from this
  count deliberately — they are *likely-moot*, not live threats needing a control; counting them
  would misrepresent an open sponsor question as closed).
- **Every live control traces:** every non-retired, non-moot threat (T1–T8, T12–T14, T16–T21) is
  countered by ≥1 control naming a concrete `D#` or `[code:]` mechanism, or is explicitly tagged
  `[ASSUMPTION]` with the owning agent named (no orphan threats silently left uncountered). Retired
  threats (T11, T15) carry no control rows by design. The G6 gate condition — no control without a
  threat, no threat without a mechanism-or-named-assumption — holds.
- **Explicitly ACCEPTED (not fully mitigated) — by design, prototype:**
  - T16a `employeeKey_i` compromise, contained blast radius (the hierarchy's intended trade-off)
  - T16b `pseudonymKey` compromise, elevated severity (can reverse completed erasures) — accepted
    because the alternative (no two-tier hierarchy) has a strictly worse cross-employee blast radius
  - T17/T18 IPFS confidentiality/erasure basis rests on key secrecy, not object removal — disclosed,
    not hidden (PRD §9.4 legal-argument risk)
- **OPEN — must close before security sign-off / build:**
  - **T1 gateway header trust (PB-1/G-18)** — HIGH, unaffected by this rework
  - **T4 Org2-operator independence unverified** — Medium-High, blocks the strongest collusion-
    resistance claim (errata E-11)
  - **T5/PB-3/G-24 canonicalization** — scheme selected (JCS over the section JSON), stays OPEN as a
    build precondition, pinned by test
  - **T6b in-band write-path partial-failure semantics** — undesigned, newly surfaced by the move away
    from async Kafka-based recording
  - **T19 IPFS cluster operator/membership** — no governing ADR yet
  - **T20/T21 channel-per-tenant-at-scale tooling** — requirement stated, mechanism not yet designed
  - **T9/T10 formal disposition** — pending sponsor decision **S-4**
- **Legal-compliance gaps that are NOT security-control gaps and must not be conflated with them:**
  DPIA obligation unmet (PRD §9.2 row 10, Pasal 34); automatic time/purpose-based retention unmet
  (PRD §9.2 row 11, Pasal 42) — tracked in `../10-risk/risk-register.md` under a legal-compliance
  category, not as accepted technical residual risk.

*These OPEN items are the security preconditions for Phase-4 (post-G9) prototype work; the ACCEPTED
items are documented limitations, not oversights.*

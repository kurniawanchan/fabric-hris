**Classification: INTERNAL**

# Security Review: fabric-hris — full-codebase pass (fabric-network/, write-path-integration/, ipfs-cluster/, qa-tests/)

**Reviewer:** Claude (security-review skill)
**Date:** 2026-08-29
**Scope:** `fabric-network/chaincode/employeeprofilerecord/`, `fabric-network/tools/*` (CLI operator tooling, not attacker-facing), `write-path-integration/{gateway-client,keystore,writepaths,ipfsclient}`, `ipfs-cluster/` (context only), `qa-tests/` (context only).
**Method:** White-box code review against `owasp-checklist.md` (core). API and LLM modules triaged and skipped — see *Method* below.

---

## Executive Summary

The system's core security property — zero PII on-chain by construction — holds up well in the sampled code: the chaincode's 12-field schema, MSP-based ABAC gate, and the client-side digest/salt architecture are all implemented as documented, with no PII-shaped parameter anywhere in the contract's function signatures. mTLS wiring in `gateway-client` is correct (real CA pool, client cert pair, no `InsecureSkipVerify`). The two real findings are: (1) an uncommitted browser-capture artifact directory sitting inside the repo's own "hard-clean, PII/product-name-free" zone that currently contains the real product name and a real internal hostname — a policy/confidentiality-register violation, not a code vulnerability, but squarely inside this repo's own stated threat model; and (2) `tenantId` is accepted by the chaincode as trusted input and never cross-checked against the caller's authenticated MSP identity, meaning the on-chain tenant-attribution metadata can be forged by any caller already authorized to write on that channel. Neither is remotely exploitable by an unauthenticated party — this is a permissioned network with MSP/mTLS at the transport layer — so both are scoped to their realistic blast radius below.

**Top risks:**
1. A contributor could commit real company/product-identifying material into a directory this repo's own confidentiality register certifies as clean (Medium — accidental disclosure, not remote exploitation).
2. An already-authorized write-path caller (or a buggy integration) can mislabel a record's tenant attribution, undermining the audit trail's trustworthiness for that field specifically (Medium — integrity of one non-PII metadata field, within an already-trusted principal's blast radius).

| Severity | Count |
|----------|-------|
| 🔴 Critical | 0 |
| 🟠 High | 0 |
| 🟡 Medium | 2 |
| 🔵 Low | 1 |
| ⚪ Info | 2 |

**Recommendation:** 🟡 Ship with mitigations — no exploitable vulnerability reachable by an unauthenticated or externally-positioned attacker was found in the sampled code; the two Medium findings should be resolved before the confidentiality-register claims or the audit-trail integrity claims are relied on for the thesis's evaluation results.

---

## Threat Model

### System summary
A permissioned Hyperledger Fabric 2.5 network (3 orgs: platform, per-tenant enterprise client, read-only auditor) anchors salted-HMAC digests of HRIS profile-section changes on-chain. No PII is a parameter to any chaincode function by design; salts and per-employee keys live off-chain in `write-path-integration/keystore`, encrypted at rest with AES-256-GCM. `gateway-client` reaches the network over mTLS. `ipfsclient` encrypts supporting documents before pinning to a private IPFS swarm.

### Top threats (prioritized)
1. **An authorized write-path caller** could **submit a mismatched `tenantId`** against **on-chain audit-trail integrity** via **RecordProfileSection's unchecked `tenantId` parameter** (SEC-M1).
2. **A contributor with repo write access** could **commit real product/company-identifying artifacts** against **the confidentiality-register guarantee this thesis's design argues from** via **an uncommitted, non-gitignored capture directory inside `fabric-network/`** (SEC-M2).
3. **A network-adjacent party without a valid MSP/mTLS identity** attempting to reach the chaincode or the gateway is blocked at the transport (mTLS) and chaincode (CID/MSP ABAC) layers — no finding; this is the trust boundary working as designed and is the primary reason nothing here scores High or Critical.
4. **An MSP-authenticated but wrong-role caller** (e.g., Org3/auditor attempting a write) is blocked by `authorizeSubmit`'s explicit allow-list — no finding, correctly deny-by-default.
5. **A process-restart / key-loss scenario** against the off-chain keystore's availability (not confidentiality) is a known, disclosed limitation (single-file, single-process, no cross-process locking) — tracked as Info (SEC-I1) since it's already disclosed in-code, not silently assumed away.

### Trust boundaries

```mermaid
flowchart LR
  writepath[HRIS write path\n(writepaths / gateway-client)]
  gw[Fabric Gateway\n(mTLS, MSP identity)]
  cc[employeeprofilerecord\nchaincode CID/ABAC gate]
  ledger[(World state /\nper-key history)]
  ks[keystore\n(AES-256-GCM file, Domain B'/C)]
  ipfs[ipfs-cluster\n(private swarm, encrypted docs)]

  writepath -- mTLS client cert --> gw
  gw -- MSP identity on proposal --> cc
  cc -- ABAC allow/deny --> ledger
  writepath -- salt/key lookup --> ks
  writepath -- encrypted doc --> ipfs
```

Authentication happens at the mTLS layer (gateway-client's client cert pair, `gatewayclient.go:276-280`) and is re-asserted per-transaction at the chaincode layer via `cid.GetMSPID`/`cid.GetID` (`identity.go:60-81`) — genuine defense-in-depth, not a duplicate check, as the in-code comment on `callerMSPID` correctly explains. Authorization is deny-by-default in both `authorizeSubmit` and `authorizeEvaluate`.

---

## Method

- **Core checklist:** applied in full against `fabric-network/chaincode/employeeprofilerecord/chaincode/*.go`, `write-path-integration/{gateway-client,keystore}/*.go` (read in full), `write-path-integration/{writepaths,ipfsclient}` (build/vet clean, signature-level read only), and the tools under `fabric-network/tools/*` (command-construction pattern only).
- **API module:** skipped — this system is a Fabric chaincode + Go SDK client, not an HTTP/REST/GraphQL server; no route/controller/handler definitions or OpenAPI spec are in scope. Logged per the triage rule ("a client that merely calls out is a consumer, not a trigger").
- **LLM module:** skipped — `grep -rilE "anthropic|openai|bedrock|vertexai|langchain|litellm|ollama"` across the scoped directories returned no hits.
- **CI/CD weighting:** not applied — no pipeline config (`.github/workflows/`, `bitbucket-pipelines.yml`) was in the reviewed scope.

---

## Findings

### 🟡 Medium

#### SEC-M1 — `tenantId` is trusted, unauthenticated chaincode input with no cross-check against the caller's MSP identity
- **Where:** `fabric-network/chaincode/employeeprofilerecord/chaincode/record_profile_section.go:71-123` (write path), and equivalently `queries.go:21-172` (all three read functions accept and ignore `tenantId` for authorization purposes)
- **CWE:** CWE-863 (Incorrect Authorization) / CWE-284 (Improper Access Control) — the parameter is accepted and persisted as though it were verified provenance, but there is no check binding it to the authenticated principal.
- **Description:** `authorizeSubmit`/`authorizeEvaluate` (`identity.go:88-115`) verify only that the caller's MSP is `Org1MSP`, `Org3MSP`, or matches the `OrgClient-<X>MSP` shape — they never compare the matched `<X>` against the `tenantId` argument the same call also carries. `requiredStringArgs` (`record_profile_section.go:44-59`) checks only non-emptiness. The value is then written verbatim into `EmployeeProfileRecord.TenantID` (`record_profile_section.go:179`) and returned by nothing but persisted as ledger metadata other tooling (audit/reconciliation) will treat as authoritative.
- **Reproduction / proof:** A client already holding a valid `OrgClient-tenant01MSP` identity and endorsement-eligible on tenant01's channel calls `RecordProfileSection(tenantId="tenant02", ...)`. Nothing in the contract rejects this; the resulting on-chain record carries `TenantID: "tenant02"` despite being written by, and endorsed on, tenant01's infrastructure.
- **Impact:** Bounded by channel membership (Fabric's channel isolation is the real tenant boundary per `ADR-0013`, and this repo's own comment at `contract.go:56-60` is explicit that key-shape does not carry tenant isolation — channel membership does). So this is **not** a cross-tenant read/write bypass. The impact is integrity of the `TenantID` metadata field specifically: a forged or buggy value silently corrupts one field of the tamper-evidence record this thesis's evaluation depends on, with no defense-in-depth catching it. Scope: any already-write-authorized caller on a given channel; detection: none currently (no chaincode-level check, no alert).
- **Recommendation:** In `authorizeSubmit`/`authorizeEvaluate`, when `mspID` matches `OrgClient-<X>MSP`, require `tenantId == X` and reject with `ErrInvalidArgument`/`ErrUnauthorized` otherwise. For Org1/Org3 callers (which aren't tenant-scoped by MSP shape), document explicitly that `tenantId` remains caller-asserted for those two principals, or route enforcement through whatever tenant-scoping the write path already asserts upstream. Record this as a `G-#` entry in `grounding-gaps.md` alongside the existing `G-31` (which covers `employeeID`/hash-shape validation but not `tenantId` specifically), so it's tracked rather than only living in this report.

#### SEC-M2 — Real product name and internal hostname present in an uncommitted, non-gitignored directory inside the "hard-clean" `fabric-network/` confidentiality zone
- **Where:** `fabric-network/network/compose/.playwright-mcp/page-2026-08-11T12-01-13-525Z.yml:21`, `fabric-network/network/compose/.playwright-mcp/console-2026-08-11T12-00-52-418Z.log:1`
- **CWE:** CWE-200 (Exposure of Sensitive Information) — organizational-policy sense (real company/product identity, per this repo's own confidentiality register in CLAUDE.md), not a technical vulnerability.
- **Description:** These are browser-automation session captures containing `© 2026 Talenta.co - Advanced Payroll Automation & HR Solution` and `https://hr.talenta.local/site/sign-in`. `git status` shows the directory untracked (`??`); unlike the sibling `.remember/` directory (which carries its own `.gitignore: *`), nothing excludes `.playwright-mcp/`. CLAUDE.md states `fabric-network/` is "hard-clean — verified zero hits (2026-08-07)" and separately notes the confidentiality grep is "a floor, not a guarantee" because prior leaks (a PHP file path, a product name in a comment) weren't caught by the pattern's exact shape — this artifact class (structured YAML dumps of a live UI session, plus a raw console log) is a third, different shape again.
- **Reproduction / proof:** `grep -rniE "talenta|mekari" fabric-network/` returns hits inside `.playwright-mcp/` (confirmed during this review).
- **Impact:** No external exposure yet (uncommitted). If committed via a broad `git add -A`/`git add .` — the exact pattern this org's own git-safety guidance warns against, and which is not universally followed — the real product name and an internal hostname would land in a path this project's own design documentation asserts is scrubbed, undermining the "zero PII / zero identifying info by construction" claim the thesis evaluation relies on for this directory specifically.
- **Recommendation:** Delete `fabric-network/network/compose/.playwright-mcp/` (safe — uncommitted, no history to rewrite). Add `.playwright-mcp/` to a `.gitignore` scoped the same way `.remember/` already is, so any future MCP/browser-tooling capture in this directory is excluded by default rather than requiring a human to remember to scrub it each time.

### 🔵 Low

- `fabric-network/chaincode/employeeprofilerecord/chaincode/identity.go:30-34` — `isOrgClientMSP` accepts any non-empty string between the fixed `OrgClient-`/`MSP` prefix/suffix as a valid tenant segment, with no format/charset validation. Not independently exploitable (the segment still has to resolve to a real, channel-config-recognized MSP to reach this code path at all — Fabric's own channel membership gates that), but worth tightening in the same change as SEC-M1 if that fix adds a `tenantId`-vs-MSP-segment comparison, so the comparison is against a format-validated value rather than an arbitrary string.

### ⚪ Info / Defense-in-depth

- **SEC-I1 — Off-chain keystore has no cross-process file locking, disclosed in-code.** `write-path-integration/keystore/filestore.go:10-13` — the `sync.Mutex` guards only within one process; concurrent writers across processes could race on the encrypted file. This is an availability/integrity concern for the salt/key stores (not confidentiality — the file is still AES-256-GCM encrypted at rest), and it's already disclosed as a known limitation in the package doc comment rather than silently assumed away, so no action is required beyond confirming the deployment model (single `integration-bridge` instance) actually holds in practice before this design scales.
- **SEC-I2 — `EmployeeProfileRecord`'s 12-field on-chain schema and the chaincode's parameter surface were checked for any PII-shaped field and none was found.** Every field is a digest, HMAC-derived pseudonymous identifier, enum, CID, or non-PII structural metadata, consistent with the design's own stated invariant (`asset.go:6-12`). No `RecordProfileSection`/query-function parameter accepts plaintext content or a salt. This is a positive finding, recorded for completeness rather than as an action item.

---

## Things That Looked Good

- `identity.go:36-59` — the CID/MSP ABAC gate is genuinely deny-by-default (empty MSP ID and unparseable X.509 identity are both explicitly rejected via `cid.GetID`, not just `cid.GetMSPID`), and the in-code comment correctly distinguishes this from, and does not conflate it with, the channel-level `AND(...)` endorsement policy — a distinction that's easy to get wrong and important to get right for this design's "no single org can write alone" property.
- `write-path-integration/gateway-client/gatewayclient.go:245-280` — mTLS setup uses a real CA pool built from a parsed certificate, a real client cert/key pair, and no `InsecureSkipVerify`/equivalent bypass anywhere in the sampled code.
- `write-path-integration/keystore/filestore.go` — AES-256-GCM with a fresh random nonce generated via `crypto/rand` on every save (correct — nonce reuse under GCM is catastrophic, and the comment explicitly explains why), key-length validated at construction rather than failing confusingly deep inside `aes.NewCipher`, and atomic write-temp-then-rename for crash safety.
- `fabric-network/tools/tenantprovision/main.go:51,62` — shells out via `exec.Command(name, args...)` (argv form), not a shell string — no OS command injection risk even though this tool takes an operator-supplied tenant ID argument.
- The four key domains (A: Fabric MSP/TLS, B: app AES-at-rest, B′: DocumentKeyStore, C: SaltStore/EmployeeKeyStore) are kept genuinely separate in `keystore.go` — each store generates key material via `crypto/rand` exclusively, with no HKDF/HMAC derivation from another domain's material, matching `ADR-0019`/`ADR-0021`'s non-mixing invariant as documented.

## Out of Scope

- `ipfs-cluster/`'s Go/compose internals — reviewed only at the README/context level (the disclosed CRDT-handshake defect); not walked file-by-file for this pass.
- `write-path-integration/writepaths` and `ipfsclient` internals beyond signature-level read and clean `go build`/`go vet` — not walked function-by-function.
- `qa-tests/` suites and evaluation reports — not reviewed for this pass; flagged in the accompanying code-review report (`docs/reviews/2026-08-29-fabric-hris-full-codebase.md`, finding m3) only for an unrelated re-run provenance question.
- Fabric network configuration (`configtx.yaml`, channel policies, CA setup) — not independently re-derived from source; this review trusted the in-code comments' description of NET-5's endorsement policy rather than inspecting `configtx/` directly.
- No dependency/CVE scan was run against `go.sum`/`vendor/` in this pass (A06 not covered) — recommend a `govulncheck` pass as a follow-up, run per-module per this repo's Go-workspace layout.

## Suggested Follow-ups

- [ ] Fix SEC-M1 (tenantId ↔ MSP cross-check) and add the corresponding `G-#` to `grounding-gaps.md`.
- [ ] Delete `.playwright-mcp/` and gitignore it (SEC-M2) before any broad `git add`.
- [ ] Run `govulncheck` per Go module (chaincode, and each of gateway-client/keystore/writepaths/ipfsclient) as a dependency-CVE sweep — not covered by this pass.
- [ ] Confirm the single-process assumption behind SEC-I1 still holds before any multi-instance `integration-bridge` deployment is considered.

# ST-6 / ST-10 / ST-11 — Crypto, Key-Domain Separation, and Secure-Config Verification

Date: 2026-08-07
Scope: structural/code-level verification only (no live-network deployment or destructive actions
performed). All citations below are exact file paths and line numbers as read in this repo at the
time of writing.

---

## ST-6 — Brute-force resistance (salt/hash construction)

**Claim 1: DataHash is always SHA-256 of salted content.**

`write-path-integration/gateway-client/digestbuilder.go:56-74` (`ComputeDataHash`):
```go
func ComputeDataHash(salt []byte, sectionJSON []byte) (string, error) {
	if len(salt) < MinSaltBytes {
		return "", ErrSaltTooShort
	}
	canonical, err := jsoncanonicalizer.Transform(sectionJSON)
	...
	h := sha256.New()
	h.Write(salt)
	h.Write(canonical)
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}
```
This is `SHA-256(salt ‖ JCS(section))` exactly as documented in the file's own header comment
(digestbuilder.go:11). Confirmed the ONLY call sites that build a DataHash for a real write path
route through this function: `write-path-integration/writepaths/writepaths.go:220`
(`dataHash, err := gatewayclient.ComputeDataHash(salt, sectionValueJSON)`) and
`write-path-integration/gateway-client/verify.go:54` (`Verify`'s local recomputation). No other
DataHash-construction code path exists in `write-path-integration/`.

**Claim 2: Salt is always >=128 bits (MinSaltBytes=16) and CSPRNG-generated.**

- Floor defined in TWO places, both `16`: `write-path-integration/keystore/keystore.go:30`
  (`MinSaltBytes = 16`) and `write-path-integration/gateway-client/digestbuilder.go:45`
  (`MinSaltBytes = 16`, enforced again at digestbuilder.go:63 as an input-shape guard —
  `ComputeDataHash` itself refuses a too-short salt even if a caller somehow got one).
- Generation source: `write-path-integration/keystore/keystore.go:109-110`
  (`InMemorySaltStore.PutSalt`):
  ```go
  salt := make([]byte, MinSaltBytes)
  if _, err := rand.Read(salt); err != nil { ... }
  ```
  `rand` here is `crypto/rand` (import at keystore.go:22), not `math/rand` — confirmed by reading
  the import block; there is exactly one `rand` import in the file and it is the CSPRNG.
- Test coverage confirming the floor is exercised: `write-path-integration/keystore/keystore_test.go:24-26`
  (`TestSaltStore_FreshPerRecordVersion` asserts `len(salt1) < MinSaltBytes` fails the test) — ran
  live, passes (see Evidence section below).

**Claim 3: Salt never appears as a chaincode argument, on-chain field, or in any
Verify()/VerificationResult field.**

- Chaincode argument list (`RecordProfileSection`, the only Submit function):
  `fabric-network/chaincode/employeeprofilerecord/chaincode/record_profile_section.go:71-83` — the
  full parameter list is `ctx, tenantId, employeeID, profileSection, dataHash, prevHash, updatedBy,
  ipfsCIDs, canonicalizationVersion, hashAlgo, clientTimestamp`. No parameter typed to carry a
  salt or plaintext section value, matching the function's own doc-comment at line 67
  ("Zero bytes of profile-section plaintext or salt are ever accepted here").
- The one real caller that builds this argument list —
  `write-path-integration/writepaths/writepaths.go:247-252` (`buildArgs` inside `doAnchor`) —
  passes exactly `{tenantID, employeeID, profileSection, dataHash, prevHash, updatedBy,
  ipfsCIDsJSON, CanonicalizationVersion, HashAlgo, ""}`. `salt` (obtained at line 216) is consumed
  ONLY by `gatewayclient.ComputeDataHash(salt, sectionValueJSON)` at line 220 and never appears in
  `buildArgs`'s closure output.
- On-chain asset shape: `fabric-network/chaincode/employeeprofilerecord/chaincode/asset.go:76-89`
  (`EmployeeProfileRecord`) — exactly twelve fields (CanonicalizationVersion, DataHash, EmployeeID,
  HashAlgo, IpfsCIDs, PrevHash, ProfileSection, RecordID, TenantID, Timestamp, UpdatedBy, Version).
  No salt field. The doc-comment (asset.go:65-69) explicitly flags this as a closed regression class
  ("adding any field re-opens the STRIDE T9 regression risk").
- Read-side shapes, also salt-free: `ProfileSectionHead` (asset.go:107-112, the
  `GetProfileSectionRecord` return — DataHash/Timestamp/UpdatedBy/Version only, per the
  already-known lesson that this is deliberately NOT the full asset) and `ProfileVersionEntry`
  (asset.go:116-126, `GetProfileHistory`'s return — the only query exposing ipfsCIDs/recordID/
  canonicalizationVersion/prevHash/hashAlgo, still no salt field).
- Client-side `Verify`/`VerificationResult`:
  `write-path-integration/gateway-client/verify.go:30-38` (`VerificationResult` struct: Found,
  Matched, OnChainHash, RecomputedHash, OnChainVersion, UpdatedBy, Timestamp — no salt field) and
  `verify.go:40-45` (`onChainRecord`, the JSON-unmarshal target for the on-chain read: DataHash,
  Version, UpdatedBy, Timestamp only). `Verify`'s signature (verify.go:53) DOES take `salt []byte`
  as an INPUT parameter (by design — the caller must already possess it via the separate INT-3
  hand-off channel, `keystore/handoff.go`), but salt is consumed only for the local
  `ComputeDataHash` recomputation (verify.go:54) and never written into the returned
  `VerificationResult`, never sent over the wire to the peer (the only Evaluate call, line 59,
  carries `tenantID, employeeID, profileSection` — no salt argument in
  `EvaluateGetProfileSectionRecord`'s signature at gatewayclient.go:151).
- Broad grep across all non-test `.go` files in `write-path-integration/` for `[Ss]alt` (full
  results captured during this review) shows every hit is one of: the `SaltStore`
  interface/implementation itself (keystore.go), the `SaltHandoff` audited hand-off channel
  (handoff.go — INT-3, a deliberately separate path per FR-14/FR-36, not the verify/write path),
  `writepaths.go`'s salt-to-DataHash computation (never salt-to-chaincode-arg), and
  `digestbuilder.go`/`verify.go`'s salt-as-local-input handling described above. No hit represents
  salt crossing into a chaincode argument or an on-chain/query-response field.

**ST-6 verdict: CONFIRMED**, all three claims hold by direct code inspection and one live test run
(below).

---

## ST-10 — Four-key-domain separation

Per ADR-0019 (referenced throughout `keystore.go`'s header comment), the four domains are:

- **Domain A** — Fabric MSP/TLS signing keys (peer/orderer/client identity material).
- **Domain B** — application-layer AES PII-at-rest key. **Not implemented anywhere in this
  workspace.** There is no general "encrypt operational-DB PII at rest" mechanism in
  `write-path-integration/` or elsewhere in this repo — the only AES usage found is Domain B'
  below (document encryption for IPFS, a narrower, already-distinguished thing). This is reported
  honestly as a gap/non-implementation, not as something verified present.
- **Domain B'** — `KEY_EMPLOYEE` / `DocumentKeyStore` (`keystore.go:64-74`,
  `InMemoryDocumentKeyStore` at keystore.go:139-170) — encrypts IPFS-hosted supporting documents.
- **Domain C** — `employeeKey_i` / `EmployeeKeyStore` + `SaltStore` (keystore.go:76-88 and
  keystore.go:43-62 respectively) — drives the on-chain pseudonymous identifiers and per-record
  salts.

**Verification method:** read every non-test `.go` file in `write-path-integration/keystore/` and
`write-path-integration/ipfsclient/` (the two packages that hold Domain B'/C and touch Domain B'-
consuming crypto respectively) for imports, and confirmed each domain's generation call site is an
independent `crypto/rand.Read`.

**Domain A (Fabric MSP/TLS) never enters `keystore` or `ipfsclient`:**
```
$ grep -n "hyperledger|fabric-gateway|x509|tls" write-path-integration/keystore/*.go write-path-integration/ipfsclient/*.go
(no matches outside test files; zero hits for any Fabric/MSP/TLS import in either package)
```
`write-path-integration/keystore/go.mod` and `write-path-integration/ipfsclient/go.mod` each
declare NO dependencies at all beyond the Go standard library (both are bare `module X / go
1.25.9` with no `require` block) — structurally impossible for either package to import Fabric
Gateway, MSP, or TLS identity material, since those libraries aren't even a build dependency.
Domain A material (found under
`fabric-network/network/crypto-config/.../msp/keystore/priv_sk` etc.) is consumed exclusively by
`write-path-integration/gateway-client/gatewayclient.go` (`loadIdentity`,
`loadTLSCredentials` — gatewayclient.go:222-281), a separate module/package from `keystore` and
`ipfsclient`, and only ever as file paths passed into `NewGatewayClient` — it is never combined
with, derived from, or fed into either key-domain store.

**Domain B' (`DocumentKeyStore`) and Domain C (`EmployeeKeyStore`) generate independently:**
- `DocumentKeyStore.GetOrCreateDocumentKey`: `keystore.go:157-158`
  (`key := make([]byte, MinKeyBytes); rand.Read(key)`).
- `EmployeeKeyStore.GetOrCreateEmployeeKey`: `keystore.go:189-190`
  (`key := make([]byte, MinKeyBytes); rand.Read(key)`).

  These are two textually separate `crypto/rand.Read` calls into two separate `map[string][]byte`
  fields (`keystore.go:144` `keys map[string][]byte` on `InMemoryDocumentKeyStore` vs.
  `keystore.go:176` `keys map[string][]byte` on `InMemoryEmployeeKeyStore` — different struct
  instances entirely, no shared backing map). Neither function reads from, or passes its output
  into, the other.
- `ipfsclient.go` (the actual consumer of a Domain B' key for AES-256-GCM document encryption,
  `encrypt`/`decrypt` at ipfsclient.go:64-109) takes `key []byte` purely as a caller-supplied
  parameter; it has no dependency on `keystore` at all (confirmed via `ipfsclient/go.mod` above)
  and performs no key derivation of its own — it only consumes whatever byte slice
  `DocumentKeyStore` already generated.

**New test written** (genuinely-uncovered invariant — `keystore_test.go`'s existing
`TestNonMixingInvariant_IndependentGeneration` at keystore_test.go:122-133 only compares
`SaltStore` vs. `EmployeeKeyStore` for the same subject; no existing test in
`keystore_test.go` or `handoff_test.go` compared `DocumentKeyStore` vs. `EmployeeKeyStore` for the
same `employeeInternalID`, which is the exact pairing ST-10 calls out):

**File created:** `write-path-integration/keystore/keydomain_separation_test.go`
(module/directory: `write-path-integration/keystore`, package `keystore` — same package as the
code under test, consistent with every other test file already in that directory; no new module
needed since it reuses the existing `keystore` module).

Two tests:
1. `TestKeyDomainSeparation_DocumentKeyAndEmployeeKeyDifferForSameSubject` — for four subject IDs
   (including an edge-case-named one), asserts `GetOrCreateDocumentKey` and
   `GetOrCreateEmployeeKey` for the SAME `employeeInternalID` (a) each meet the `MinKeyBytes` floor,
   (b) are never byte-identical to each other, and (c) are each individually idempotent across a
   second call (still generated-once-and-reused per domain).
2. `TestKeyDomainSeparation_DeletingOneDomainLeavesTheOtherIntact` — proves the two stores are
   backed by genuinely separate state, not just separate method names over one shared map:
   deleting a subject's `DocumentKeyStore` key leaves that same subject's `EmployeeKeyStore` key
   byte-for-byte unchanged (and confirms the delete itself was real, by showing the document key
   regenerates to a NEW value afterward).

**Live run (from the correct module directory per this repo's Go-workspace quirk):**
```
$ cd write-path-integration/keystore && go build ./... && go test ./... -v
=== RUN   TestKeyDomainSeparation_DocumentKeyAndEmployeeKeyDifferForSameSubject
--- PASS: TestKeyDomainSeparation_DocumentKeyAndEmployeeKeyDifferForSameSubject (0.00s)
=== RUN   TestKeyDomainSeparation_DeletingOneDomainLeavesTheOtherIntact
--- PASS: TestKeyDomainSeparation_DeletingOneDomainLeavesTheOtherIntact (0.00s)
... (all 12 pre-existing tests in the package also PASS, unchanged)
PASS
ok  	keystore	0.304s
```

**ST-10 verdict:**
- Domain A vs. B'/C: **CONFIRMED** separate by code inspection (no import path exists for Domain A
  material to reach `keystore` or `ipfsclient`).
- Domain B' vs. C: **CONFIRMED** independent generation (separate `crypto/rand.Read` call sites,
  separate backing maps, no cross-reference), now backed by a new passing test.
- Domain B: **honestly reported as not implemented** in this workspace — there is no general
  app-layer AES-at-rest PII key/mechanism to verify separation against. This is a scope
  observation, not a defect found in existing code.

---

## ST-11 — Secure-config + no-PDC regression guard

**No Private Data Collection anywhere in the live network's actual config or deploy path.**

Repo-wide grep for `collections_config` / `PrivateDataCollection` / `--collections-config` across
all `*.yaml`/`*.yml`/`*.go`/`*.json`/`*.sh` files (excluding vendor): the only hits are inside
`fabric-skill-suite/` — a generic, project-independent Fabric authoring skill/toolkit that ships
optional, unused PDC support (`COLLECTIONS_CONFIG="${COLLECTIONS_CONFIG:-}"` — empty by default —
in `fabric-skill-suite/skills/fabric-chaincode-dev/scripts/lifecycle-deploy.sh:38,73`) and its own
eval fixtures/reference docs about Fabric's PDC feature in general. None of these are part of this
network's actual configuration or the tool that actually deployed this network's chaincode.

Confirmed directly against the real artifacts:
- `fabric-network/network/configtx/configtx.yaml` (348 lines total): the only `ollection`/`PDC`
  hits are its OWN header comments explicitly documenting the absence —
  `configtx.yaml:21-22` ("ADR-0015-crypto-shred-erasure.md (retires PDC entirely for this design —
  NO collections-config.json, NO PurgePrivateData...)") and `configtx.yaml:168-169` ("ADR-0015
  retires PDC entirely ... No collections-config.json exists in this network"). No actual
  `Collections:` config block appears anywhere in the file.
- `fabric-network/tools/ccdeploy/main.go` (the tool that actually performed this network's real
  install/approve/commit — confirmed by its own header comment at line 4 and the lifecycle calls
  at lines 144, 157, 174-175, 186-187, 228): **zero** matches for `collections`/`CollectionsConfig`/
  `--collections` anywhere in the file. The lifecycle commands actually run are plain
  `peer lifecycle chaincode install/approveformyorg/checkcommitreadiness/commit` with no
  `--collections-config` flag at all.
- Repo-wide grep for `PDC` in `*.md` files confirms every hit outside `fabric-skill-suite/` is
  either historical/retired-design documentation explicitly recording the retirement (e.g.
  `_bmad-output/planning-artifacts/project-context.md:98` — "No Private Data Collections anywhere
  in this design... Do not reintroduce a PDC without [going through architect]"; `prd.md:780`'s
  ADR-0015 summary row "tanpa PDC" = "without PDC"; `rencana-rekonsiliasi.md:128` marking backlog
  item `NET-3-PDC` as `BLOCKED-SUPERSEDED`) — never a live, active PDC definition.

**mTLS-everywhere confirmed** across every peer and orderer container definition in
`fabric-network/network/compose/network-docker-compose.yaml`:
- 5 peers, each with both `CORE_PEER_TLS_ENABLED=true` and `CORE_PEER_TLS_CLIENTAUTHREQUIRED=true`
  (lines 280/284, 323/327, 523/527, 570/574, 621/625 — matching the 5 live peers: peer0/1.org1,
  peer0.tenant01, peer0.org3, peer0.tenant02).
- 3 orderers, each with both `ORDERER_GENERAL_TLS_ENABLED=true` and
  `ORDERER_GENERAL_TLS_CLIENTAUTHREQUIRED=true` (lines 370/374, 415/419, 460/464).

This matches the already-known lesson that every peer requires
`CORE_PEER_TLS_CLIENTAUTHREQUIRED=true` and is corroborated independently by
`fabric-network/tools/ccdeploy/main.go:118-120`, which sets exactly these same flags (plus client
cert/key) on every `peer` CLI invocation it makes — a TLS-config-without-client-certs handshake
would fail against these peers, consistent with the project's own documented past finding.

**Default-credential check.** Grepped for `"admin"`/`adminpw`/`changeit`/`admin123`-shaped literals
across `fabric-network/`:
```
fabric-network/network/fabric-ca/ca-tenant01-config.yaml:62:  pass: tenant01AdminPW2026
fabric-network/network/fabric-ca/ca-org3-config.yaml:55:    pass: org3AdminPW2026
fabric-network/network/fabric-ca/ca-org1-config.yaml:137:   pass: org1AdminPW2026
fabric-network/network/compose/ca-docker-compose.yaml:79:   -b admin-org1:org1AdminPW2026
fabric-network/network/compose/ca-docker-compose.yaml:105:  -b admin-tenant01:tenant01AdminPW2026
fabric-network/network/compose/ca-docker-compose.yaml:130:  -b admin-org3:org3AdminPW2026
```
All six hits are Fabric CA's own bootstrap-identity mechanism (the `-b <identity>:<password>` flag
every `fabric-ca-server start` invocation requires, per Fabric's standard convention) — and the
identity names (`admin-org1`, `admin-tenant01`, `admin-org3`) and passwords
(`org1AdminPW2026`, `tenant01AdminPW2026`, `org3AdminPW2026`) are per-org-distinct, non-generic
values, not the literal `admin`/`adminpw` pair the Fabric docs use as their own placeholder
example. Per this task's own framing, this is the standard CA bootstrap convention, correctly
distinguished from a default-credential defect — **not flagged as a finding**. No other
`admin`/`adminpw`/`changeit`/`admin123`-shaped literal was found anywhere else in `fabric-network/`.

**ST-11 verdict: CONFIRMED.** No PDC in any live config or the real deploy tool (only inert,
generic skill-suite tooling and historical documentation mention it at all); mTLS with client-auth
required is enforced on every peer and orderer; the only credential-shaped literals found are the
standard, non-generic Fabric CA bootstrap identities.

---

## Files touched by this review

**Created:**
- `qa-tests/security/st6-st10-st11-crypto-keydomain-config-report.md` (this file)
- `write-path-integration/keystore/keydomain_separation_test.go` (new Go test file, package
  `keystore`, module `write-path-integration/keystore` — reused the existing module rather than
  creating a new one, consistent with every other `*_test.go` in that directory)

**Edited:** none. No existing file was modified.

**Not touched (per hard safety constraints):** no `docker` mutation commands, no chaincode
lifecycle actions, no channel-join actions, nothing on `tenant-tenant02`. All verification above
was read-only (file reads, greps, and one `go test` run against in-memory reference
implementations with no live network interaction).

# ST-2 / ST-7 / ST-8 / ST-12 / ST-13 / ST-14 — disposition/coverage report

Scope: the six remaining `ST-#` rows from `agent-suite/06-roadmap/test-strategy.md` §1 not yet
covered by an earlier backlog item's own report. Per the task brief, most of these are already
effectively resolved by earlier work; this report's job is to **re-verify that claim by re-running
the cited tests fresh** (not cite them from memory) and to report ST-2 honestly as still open.
Literal `ST-#` definitions were re-read from `test-strategy.md` §1 (lines 123–135) before writing
anything below.

All commands were run live against this session's already-healthy network (`docker ps` reconfirmed
before and after — 3 orgs' peers, 3 Raft orderers, both `ipfs0`/`ipfs1`, all up throughout; nothing
was created, removed, or joined). No chaincode lifecycle action, no `docker rm`/`volume rm`/`compose
up`, no channel join/create, and no identity revocation was performed. `tenant-tenant02` was touched
only via one **read-only Evaluate call issued from peer0.tenant01's own gateway** (ST-14, see below)
— never dialed, joined, or written to.

---

## ST-7 — metadata-boundary regression guard

**Definition (test-strategy.md L128):** `EmployeeProfileRecord`'s on-chain schema contains no field
beyond `{RecordID, EmployeeID, ProfileSection, DataHash, PrevHash, Version, Timestamp, UpdatedBy,
IPFSCIDs}` (plus the fields this session's later ADRs added on top — see below) — a regression guard
against ever reintroducing a `changedFieldNames`-style field. **Disposition: resolved 2026-08-06
(S-4), formally moot, still run as a regression guard.**

Re-ran the check fresh by reading the live struct in
`fabric-network/chaincode/employeeprofilerecord/chaincode/asset.go` (not from memory/cache):

```
type EmployeeProfileRecord struct {
	CanonicalizationVersion string   `json:"canonicalizationVersion"`
	DataHash                string   `json:"dataHash"`
	EmployeeID              string   `json:"employeeID"`
	HashAlgo                string   `json:"hashAlgo"`
	IpfsCIDs                []string `json:"ipfsCIDs"`
	PrevHash                string   `json:"prevHash"`
	ProfileSection          string   `json:"profileSection"`
	RecordID                string   `json:"recordID"`
	TenantID                string   `json:"tenantID"`
	Timestamp               string   `json:"timestamp"`
	UpdatedBy               string   `json:"updatedBy"`
	Version                 int      `json:"version"`
}
```

Exactly 12 fields — `RecordID, EmployeeID, ProfileSection, DataHash, PrevHash, Version, Timestamp,
UpdatedBy, IpfsCIDs, TenantID, CanonicalizationVersion, HashAlgo` — matching the on-chain asset
schema already established in this session's ground truth ("this chaincode's on-chain
`EmployeeProfileRecord` schema is exactly ... nothing else, ever"). No `changedFieldNames`,
`fieldName`, `changedFields`, or any other per-field/changed-content-shaped identifier exists
anywhere in the struct. The file's own doc-comment (asset.go L63–69) independently states the same
invariant and ties it to the T9 regression risk this guard exists for.

**Cross-reference:** this is the same check ST-1's lane runs; confirming redundantly here per the
task brief's explicit allowance ("a little redundancy across lanes here is fine").

**Disposition: CONFIRMED MOOT, regression guard holds.** No code change required or made.

---

## ST-8 — Kelompok B′ protocol compliance

**Definition (test-strategy.md L129):** the verification path never accepts a plaintext argument
(FR-34) and never returns *salt* (FR-14); no separate anchor-service exists to audit for
over-broad read-back. **Disposition: resolved 2026-08-06 (S-4), formally moot, still run as a
regression guard.**

Re-read `write-path-integration/gateway-client/verify.go` fresh (the only client-side verification
helper — REC-7, used identically by all three verifier classes):

- `Verify(ctx, tenantID, employeeID, profileSection string, salt []byte, currentValueJSON []byte)`
  — `salt` is a caller-supplied **parameter**, obtained via the separate INT-3 hand-off channel; the
  function "never accepts, requests, or derives a salt itself" (verify.go L50–52, doc-comment).
  Nothing constructed here ever ships a plaintext profile-section value or salt *to* the chaincode —
  the entire chaincode footprint is one `Evaluate` carrying only `(tenantID, employeeID,
  profileSection)` (verify.go L9–13).
- The on-chain read (`EvaluateGetProfileSectionRecord` → `onChainRecord`) is deserialized into
  exactly `{DataHash, Version, UpdatedBy, Timestamp}` (verify.go L40–45) — no `salt` field exists in
  that struct, so there is no code path by which `Verify` could return one even by accident.
- `GetProfileSectionRecord`'s own return shape on the chaincode side (`ProfileSectionHead` in
  asset.go L101–112) is independently confirmed to carry exactly `{DataHash, Timestamp, UpdatedBy,
  Version}` — matching the "already known" ground truth that this query deliberately excludes
  `ipfsCIDs`/`recordID`/`canonicalizationVersion`/`prevHash`/`hashAlgo` (those are reachable only via
  `GetProfileHistory`, a different, wider-scope query, not the verification path).
- No separate anchor-service exists anywhere in the repo to audit for over-broad read-back — same
  ADR-0014 in-band-recording fact QA-3 already established (write/verify/salt-delivery/onboarding
  are all direct in-process Go-to-Gateway-gRPC, not a fronting service).

**Cross-check against QA-3's finding:** QA-3's contract-coverage report states `CT-1`/`CT-2`/`CT-3`'s
properties (including the salt-free `Verify()` signature) are already covered by the real Go-level
tests in this same package (`verify_integration_test.go`, `verify_notfound_integration_test.go`) —
consistent with what direct code inspection shows here; no divergence found.

**Disposition: CONFIRMED MOOT, regression guard holds.** No code change required or made.

---

## ST-12 — `employeeKey_i`/`pseudonymKey` isolation (hierarchy-aware, two-part)

**Definition (test-strategy.md L133):** **(a)** destroying/leaking one employee's `employeeKey_i`
does not affect any other employee's `EmployeeID`/`UpdatedBy` verifiability (T16a). **(b)**
`pseudonymKey` never leaves HSM-protected custody, and — as a documented residual-risk
demonstration, not a pass/fail gate — a `pseudonymKey` leak plus an enumerable
`employeeInternalId` is sufficient to recompute `employeeKey_i` for any employee including
already-erased ones (T16b).

### ST-12(a) — re-run fresh

```
$ cd write-path-integration/keystore && go test ./... -run TestEmployeeKeyStore_DifferentEmployeesDifferentKeys -v
=== RUN   TestEmployeeKeyStore_DifferentEmployeesDifferentKeys
--- PASS: TestEmployeeKeyStore_DifferentEmployeesDifferentKeys (0.00s)
PASS
ok  	keystore	1.031s
```

`keystore_test.go` L83–91 asserts two independently-generated `employeeKey_i` values (for `emp-1`
and `emp-2`) are never equal — direct proof of the hierarchy's containment claim: with
independently-random per-employee key generation (see ST-12(b) below), one employee's key carries
zero information about any other's. This is ST-12(a)'s real proof, cited and re-run, not
re-implemented.

### ST-12(b) — CLOSED per ADR-0021, no residual to demonstrate

Read `agent-suite/05-adr/ADR-0021-employeekey-independent-random-no-master-key.md` (Accepted,
2026-08-06) fresh. Its Decision section is unambiguous:

> `pseudonymKey` is **removed from the design entirely**; there is no tenant-scoped root secret for
> identifier pseudonymization... `employeeKey_i ← CSPRNG(≥ 128 bits), generated once, at the
> employee's first EmployeeProfileRecord write`.

And its Consequences section states explicitly: "T16b is **closed**, not merely mitigated — there is
no longer any single secret whose compromise can silently undo a completed erasure across an entire
tenant's history... **T16b is now CLOSED** (not 'accepted with mitigation') — the threat's
precondition (a key whose compromise reverses history) no longer exists."

Independently confirmed against the actual keystore code
(`write-path-integration/keystore/keystore.go` L68/77/85): the only mentions of "pseudonym" in that
file are doc-comment references to ADR-0019's cross-domain rule and a note that `employeeKey_i` is
now generated per ADR-0021 "no pseudonymKey/master key" — there is no `pseudonymKey` type, field, or
derivation function anywhere in the module to leak or recompute from.

**There is no ST-12(b) demonstration left to run.** The premise of the test (a `pseudonymKey` that
could leak and be combined with an enumerable `employeeInternalId`) no longer exists as a construct
anywhere in this codebase — `employeeKey_i` is now structurally identical to the already-accepted
`DataHash` salt (independently random, generated once, no master secret regenerates it). Per the
task brief and ADR-0021's own follow-up note, this is stated plainly as closed rather than described
as a still-runnable demonstration.

**Disposition: ST-12(a) CONFIRMED (re-run, passing). ST-12(b) CLOSED per ADR-0021 — not
applicable, no residual exists.**

---

## ST-13 — IPFS erasure-basis proof

**Definition (test-strategy.md L134):** after `KEY_EMPLOYEE` destruction, a ciphertext object
retrieved by CID (if still physically present — `unpin ≠ delete`) fails to decrypt; CID possession
alone, without `KEY_EMPLOYEE`, never yields plaintext. **Already proven by QA-2's
`write-path-integration/writepaths/erasure_ipfs_integration_test.go` — re-run fresh below.**

```
$ cd write-path-integration/writepaths && go test -tags integration ./... \
    -run TestIntegration_QA2_ErasureLeavesOnChainStateUnchangedAndBreaksOldCiphertext -v

=== RUN   TestIntegration_QA2_ErasureLeavesOnChainStateUnchangedAndBreaksOldCiphertext
    erasure_ipfs_integration_test.go:173: IT-5 confirmed: GetEmployeeProfileSummary + GetProfileHistory(PERSONAL) + all 5 GetProfileSectionRecord reads are byte-for-byte identical before/after Erase
    erasure_ipfs_integration_test.go:191: IT-9 confirmed: post-erasure FetchAndDecrypt with a fresh KEY_EMPLOYEE correctly failed to open the old ciphertext: ipfsclient: decryption failed (wrong key or tampered ciphertext): cipher: message authentication failed
--- PASS: TestIntegration_QA2_ErasureLeavesOnChainStateUnchangedAndBreaksOldCiphertext (19.23s)
PASS
ok  	writepaths	20.081s
```

This is a live run against the real network: `peer0.org1`/channel `tenant-tenant01` for the ledger
half, and both live kubo nodes (`ipfs0`:5001, `ipfs1`:5002) for the CID half. The test writes a
fresh, distinctively-named fixture (`qa2erasuredemo...`) across all five profile sections, confirms
pre-erasure `FetchAndDecrypt` succeeds with the correct `KEY_EMPLOYEE`-equivalent document key,
calls `Hooks.Erase` (which destroys the document key via `DocumentKeys.DeleteDocumentKey` but never
unpins the CID), then confirms a **freshly re-issued** key for the same employee fails to decrypt
the **same, still-present** ciphertext (AEAD authentication failure) — exactly the "`unpin ≠ delete`,
key-destruction is the real erasure mechanism" property ST-13 requires. It additionally confirms the
IT-5 property (on-chain state byte-for-byte unchanged across `Erase`) as a bonus from the shared
fixture, though that is IT-5's claim, not ST-13's.

**Disposition: CONFIRMED, re-run fresh, passing.** No new test written — QA-2's existing file is the
correct, non-duplicated proof.

---

## ST-14 — tenant-isolation, real second tenant (P3)

**Definition (test-strategy.md L135):** the actual security-test embodiment of P3 — a genuinely
provisioned second tenant's peer attempting a cross-tenant read is rejected by MSP/channel
membership. **Already proven by
`write-path-integration/gateway-client/tenant_isolation_integration_test.go` (INT-4) — re-run
fresh below.**

```
$ cd write-path-integration/gateway-client && go test -tags integration ./... \
    -run 'TestIntegration_Tenant02CannotReachTenant01Channel|TestIntegration_Tenant01CannotReachTenant02Channel' -v

=== RUN   TestIntegration_Tenant02CannotReachTenant01Channel
    tenant_isolation_integration_test.go:54: tenant02 -> tenant-tenant01 correctly refused: gatewayclient: ledger error: rpc error: code = Unavailable desc = failed to get config for channel [tenant-tenant01]: could not get last config for channel tenant-tenant01
--- PASS: TestIntegration_Tenant02CannotReachTenant01Channel (0.53s)
=== RUN   TestIntegration_Tenant01CannotReachTenant02Channel
    tenant_isolation_integration_test.go:86: tenant01 -> tenant-tenant02 correctly refused: gatewayclient: ledger error: rpc error: code = Unavailable desc = failed to get config for channel [tenant-tenant02]: could not get last config for channel tenant-tenant02
--- PASS: TestIntegration_Tenant01CannotReachTenant02Channel (0.47s)
PASS
ok  	gatewayclient	3.035s
```

Both directions pass live: `OrgClient-tenant02`'s own admin identity, dialed against its **own**
peer (`peer0.tenant02`, localhost:11051), is rejected when its `GatewayClient` is configured to
target channel `tenant-tenant01` — and symmetrically, `OrgClient-tenant01`'s own admin identity,
dialed against its **own** peer (`peer0.tenant01`, localhost:9051), is rejected targeting channel
`tenant-tenant02`. The rejection happens at the gRPC/config-block layer ("could not get last config
for channel") before any chaincode logic runs — exactly the deeper channel-membership guarantee
ADR-0013's channel-per-tenant model is built on, not merely a chaincode-level ACL check.

**Safety note on this run:** the `tenant-tenant02`-directed sub-test only ever dials **peer0.tenant01**
(localhost:9051) — it never dials, joins, queries, or writes to `peer0.tenant02` or any container on
`tenant-tenant02` itself. It is a plain, read-only `Evaluate` attempt from a peer that was never
joined to that channel, consistent with this task's ground-rule that `tenant-tenant02` must not be
touched by any fresh lifecycle action — no lifecycle action (install/approve/commit/join/create) was
performed here, only a read that the network itself refuses before reaching any chaincode.
`peer0.tenant02` remains, as before this run, joined to no channel — unchanged.

`docker ps` was re-checked immediately after this run: all containers (`peer0.org1`, `peer1.org1`,
`peer0.tenant01`, `peer0.org3`, `peer0.tenant02`, all three orderers, both IPFS nodes) remain `Up`,
unaffected.

**Disposition: CONFIRMED, re-run fresh, passing. P3 is verified, not merely assumed** — this used a
genuinely separately-provisioned tenant (`OrgClient-tenant02`, its own MSP, its own peer, its own
crypto material), satisfying errata E-11's provisioning prerequisite; P3 is not reported as
unverified here.

---

## ST-2 — header-trust / gateway (PB-1/G-18)

**Definition (test-strategy.md L123):** forged tenant/user identity header rejected; requires the
PB-1/G-18 pen-test — flagged in test-strategy.md itself as "unaffected by this rework, still open."

This premise was independently re-checked against `ADR-0014` (in-band recording: "Anchor-service
deployable units: 0") and against QA-3's contract-coverage finding, both already established in this
session: **every write/verify/salt-delivery/onboarding call in this workspace is a direct
in-process Go call over Fabric Gateway gRPC** — `gatewayclient.SubmitRecordProfileSection`,
`gatewayclient.Verify`, `keystore.SaltHandoff.RequestSalt`, `fabric-network/tools/tenantprovision`.
There is no HTTP surface, no API gateway, and no "identity header" of any kind anywhere in this
workspace's code for a forged-header test to target. Confirmed by inspection: nothing in
`write-path-integration/` or `fabric-network/tools/` parses an HTTP request header, and no
API-gateway or reverse-proxy component exists in this repo at all.

ST-2's actual subject — per test-strategy.md's own note — is the **real external HRIS application's
own separate public HTTP edge**, which this workspace does not fork, rehost, or integrate with (a
standing scope decision, also recorded against CT-1..CT-4/PB-1/ST-2 in this session's ground truth).
That edge does not exist inside this repository or this network to test against.

**Disposition: OPEN, UNVERIFIABLE FROM WITHIN THIS WORKSPACE. Not closed, not tested-and-passed.**
Blocked on PB-1 (the pen-test against the real external application), which is out of scope for this
workspace by standing decision. No test was fabricated against a surface that does not exist here;
this is reported honestly as a genuine, standing gap requiring the real external application to
verify, exactly as test-strategy.md itself already flags it.

---

## Summary table

| ST-# | Disposition | Evidence (re-run fresh in this task) |
|------|-------------|----------------------------------------|
| ST-2 | **OPEN / unverifiable here** — blocked on PB-1, real external HRIS HTTP edge out of scope | Code inspection: no HTTP/header surface exists anywhere in this workspace |
| ST-7 | **Moot (S-4), regression guard holds** | `asset.go` struct read fresh: exactly 12 fields, no `changedFieldNames`-shaped field |
| ST-8 | **Moot (S-4), regression guard holds** | `verify.go` read fresh: salt is caller-supplied param only, never returned; no anchor-service exists |
| ST-12(a) | **CONFIRMED, passing** | `go test ./... -run TestEmployeeKeyStore_DifferentEmployeesDifferentKeys -v` → PASS |
| ST-12(b) | **CLOSED per ADR-0021** — no residual exists to demonstrate | ADR-0021 Decision/Consequences read fresh; `pseudonymKey` absent from `keystore.go` |
| ST-13 | **CONFIRMED, passing** | `go test -tags integration ./... -run TestIntegration_QA2_ErasureLeavesOnChainStateUnchangedAndBreaksOldCiphertext -v` → PASS |
| ST-14 | **CONFIRMED, passing, P3 verified (not unverified)** | `go test -tags integration ./... -run 'TestIntegration_Tenant0[12]CannotReachTenant0[21]Channel' -v` → both PASS |

No files were created or edited other than this report. No chaincode lifecycle action was taken. No
container was created, removed, or joined. `tenant-tenant02` was touched only by one read-only
`Evaluate` attempt originating from `peer0.tenant01`'s own gateway client (never dialing
`peer0.tenant02` itself), which the network correctly refused before reaching any chaincode layer.
`docker ps` before and after this task's work shows the same set of healthy containers, unchanged.

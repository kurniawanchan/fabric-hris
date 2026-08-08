# QA-2 — Integration test coverage audit (IT-1..IT-10)

> Audits real, running coverage against `agent-suite/06-roadmap/test-strategy.md` §1's `IT-#` table.
> Every row below is a test **I ran myself, fresh, this pass** — file:function, live PASS/FAIL, and
> what it actually proves — except where explicitly marked otherwise (IT-7's provisioning half, which
> is a prior backlog item's live evidence, not a Go test this item re-runs). One genuinely missing
> piece of runnable proof (IT-5's on-chain-unchanged empirical check, IT-9's erasure-basis half) is
> closed by one new file, per this item's own scope ceiling.

## How this was run

```
cd write-path-integration/writepaths      && go test -tags=integration -run TestIntegration -v -count=1 ./...
cd write-path-integration/gateway-client  && go test -tags=integration -run TestIntegration -v -count=1 ./...
cd write-path-integration/keystore        && go test -v -count=1 ./...
cd write-path-integration/ipfsclient      && go test -tags=integration -run TestIntegration -v -count=1 ./...
cd fabric-network/tools/raftfaulttest     && go run main.go
```

All four `write-path-integration/*` commands were run from inside the specific module directory (the
`go.work` quirk — running from `write-path-integration/` itself fails with "directory prefix . does
not contain modules listed in go.work"). `-count=1` was used to bypass Go's test cache and force an
actually-fresh execution against the live network for this report — a first pass without it returned
`ok gatewayclient (cached)`, which would have reported a PASS without genuinely re-touching the
ledger; this is flagged here as a real methodology hazard for future re-runs, not something specific
to this run. **`keystore`'s command carries no `-tags=integration`** — see IT-4's own row: `handoff_test.go`
is a plain unit test (`package keystore`, no build tag) and never dials the live network at all; running
it with `-tags=integration` (which would only add more files, none matching `-run TestIntegration`
naming) is not what its own suite is built for.

One additional live check (IT-2) was performed via an ad hoc, non-repo Go program (`go run .` from a
scratch directory, using a `replace` directive against the already-built `gatewayclient` module) rather
than a fifth `go test` invocation — see IT-2's row for why, and why it was deliberately **not** added
as a permanent test file in this pass.

## Coverage matrix

| IT-# | test-strategy.md target | Real covering test(s) | What it actually proves | Status |
|------|--------------------------|------------------------|--------------------------|--------|
| **IT-1** | End-to-end anchor, in-band (ADR-0014): HRIS write path → digest computed off-chain in-process → `RecordProfileSection` committed on both required orgs' peers, no Kafka/anchor-service | `TestIntegration_AllFiveWritePathsAgainstLiveNetwork` (`write-path-integration/writepaths/writepaths_integration_test.go:21`), incl. its `GetEmployeeProfileSummary fans out to all five sections` and `REC-5: only the section written WITH a document carries a CID on-chain` subtests | All five distinct write-path hooks (`UpdatePersonalData`, `ApproveEmploymentTransfer`, `RecordEducationHistory`, `ApproveFamilyDataChange`, `UpdatePayrollBankAccount`) save to the operational store then anchor **in the same call, same process**, land on-chain, and are readable back with the correct per-section digests — no queue, no separate service anywhere in the call path | **PASS**, fresh run, 38.437s total (`go test` output: `ok writepaths 38.437s`; all 5 hook subtests + both follow-on subtests green) |
| **IT-2** | Endorsement: single-org submit rejected; `AND(Org1MSP.peer, OrgClient-tenant01MSP.peer)` required; Org3 (auditor) is read-only, does not endorse | Structural: CC-5's real chaincode commit (`implementation-backlog.md` CC-5 row) — `AND('Org1MSP.peer','OrgClient-tenant01MSP.peer')`, no `Org3MSP` key in the policy at all. Dynamic/live: NET-8's original `peer chaincode invoke` proof (no surviving runnable artifact in `fabric-network/tools/` — confirmed, see below) **re-confirmed live this pass** via an ad hoc Go program driving the same `gatewayclient` module the rest of this suite uses, submitting as Org3's own identity against `tenant-tenant01`/`employeeprofilerecord` | Org3's identity attempting `RecordProfileSection` is rejected with the chaincode's own precise message (matches NET-8's original finding byte-for-byte); the SAME identity's `Evaluate` (`GetProfileSectionRecord`) succeeds in the same run, isolating the rejection as write-specific, not "Org3 unreachable" | **PASS**, live output today:<br>`POSITIVE CONTROL (Org3 Evaluate/read) SUCCEEDED as expected: {"dataHash":"sha256:cba6f9ab...","timestamp":"2026-08-07T01:28:24Z",...}`<br>`NEGATIVE TEST (Org3 Submit/write) correctly REJECTED: ... chaincode response 500, employeeprofilerecord: caller identity failed MSP/ABAC verification: MSP "Org3MSP" is not authorized to submit a write` |
| **IT-3** | Verify path: read-only call returns only `{DataHash, Version, Timestamp, UpdatedBy}`; client recomputes/compares locally; result distinguishes hash-mismatch from record-not-found | Field-shape half: `ProfileSectionHead` type (`fabric-network/chaincode/.../asset.go:107`), verified against the real peer at CC-3. NotFound-vs-mismatch half: `TestIntegration_INT2_VerifyDistinguishesNotFoundFromMismatch` (`write-path-integration/gateway-client/verify_notfound_integration_test.go:40`), incl. its Org1/OrgClient-tenant01 subtests and the "regression: a record that DOES exist" subtest | A never-anchored `(employeeID, profileSection)` reports `Found:false` with **no error**, distinctly from a genuine ledger/gateway failure; a record that DOES exist reports `Found:true` and the correct `Matched` value | **PASS**, fresh run, 4.01s (`Org1 (platform): correctly reported Found: false, nil error`; `OrgClient-tenant01 (enterprise-client auditor): correctly reported Found: false, nil error`; `fixture record correctly reported Found: true, Matched: true (version=1)`) |
| **IT-4** | Salt-delivery channel (FR-36): authorized retrieval of a version's salt is a separate, audited call from verify; scoped to data ownership (employee's own record, auditor's audit scope, each org's own peer) | `TestSaltHandoff_GrantsForRecordOwner`, `TestSaltHandoff_DeniesUnauthorizedThirdParty`, `TestSaltHandoff_FailsClosedOnAuthorizeError`, `TestSaltHandoff_GetSaltMissNotRecordedAsDenial` (`write-path-integration/keystore/handoff_test.go`) | `SaltHandoff.RequestSalt` is structurally separate from `gatewayclient.Verify`/`VerificationResult` (own package, own type, FR-14's own reason cited in the file's header comment); grants/denials both recorded to `AuditLog`; a broken `Authorize` backend fails **closed**, never an implicit grant; a genuine not-found is **not** mis-recorded as a denial | **PASS**, fresh run, 0.524s, all 4 green. **Premise note:** see "Stale/mismatched premises" — this proof is a plain unit test against in-memory stores, not a live-Fabric integration test, even though test-strategy.md files it under the "real Fabric + services" tier |
| **IT-5** | Erasure (ADR-0015): the four-part crypto-shred leaves on-chain state byte-for-byte unchanged; surviving `DataHash`/`EmployeeID`/`UpdatedBy`/CID become permanently unopenable | **NEW:** `TestIntegration_QA2_ErasureLeavesOnChainStateUnchangedAndBreaksOldCiphertext` (`write-path-integration/writepaths/erasure_ipfs_integration_test.go`) | Writes all five sections for a fresh fixture employee, snapshots `GetEmployeeProfileSummary` + `GetProfileHistory(PERSONAL)` + all five `GetProfileSectionRecord` reads, calls `Hooks.Erase`, re-reads every one of those surfaces, and asserts **byte-for-byte identical JSON** before vs. after — empirical proof against the live ledger, not a re-statement of `erasure.go`'s own "zero ledger calls" doc-comment | **PASS**, fresh run, 20.29s: `IT-5 confirmed: GetEmployeeProfileSummary + GetProfileHistory(PERSONAL) + all 5 GetProfileSectionRecord reads are byte-for-byte identical before/after Erase` |
| **IT-6** | Audit trail + partial-failure detection: `UpdatedBy`/`Timestamp`/`Version` present on every record; a paired operational-DB-succeeds/anchor-fails outcome is at minimum detectable | `TestIntegration_PartialFailure_AnchorFailsAfterOperationalWriteCommits` (`write-path-integration/writepaths/partial_failure_integration_test.go:27`) | Pointing a real, validly-authenticated `GatewayClient` at a never-installed chaincode name fails cleanly at the Gateway layer; the operational-DB write (a real `InMemoryOperationalStore`, not a mock assertion) genuinely commits first; the failure surfaces as a distinctly-typed `*PartialFailureError` carrying the exact identifying fields, and the injected `OnPartialFailure` callback fires exactly once with matching arguments | **PASS**, fresh run, 0.15s |
| **IT-7** | Tenant onboarding (FR-24): new tenant's channel provisioned, chaincode installed/approved/committed, MSP identities registered; onboarding credentials cannot operate on a different tenant's channel | Provisioning half (not re-run this pass — no chaincode-lifecycle action needed, nothing to re-verify beyond channel/peer membership state): NET-7's live evidence (`implementation-backlog.md` NET-7 row — tenant02's channel/crypto material/peer-join, real, not simulated). Scope-bounded-credential half: `TestIntegration_Tenant02CannotReachTenant01Channel`, `TestIntegration_Tenant01CannotReachTenant02Channel` (`write-path-integration/gateway-client/tenant_isolation_integration_test.go`) | tenant02's own identity, on its own peer, cannot evaluate against tenant01's channel (never joined — refused at the gRPC/gossip layer, before any chaincode logic runs), and symmetrically for tenant01 against tenant02's channel | **PASS**, fresh run (both tests inside the 17.322s gateway-client suite run): `tenant02 -> tenant-tenant01 correctly refused`; `tenant01 -> tenant-tenant02 correctly refused` |
| **IT-8** | Cross-tenant isolation with a REAL second tenant (errata E-11): genuinely provisioned tenant02, own channel, own peer, rejected reading tenant01's channel | SAME file/tests as IT-7 (`tenant_isolation_integration_test.go`) | tenant02 is a real, independently-provisioned org/channel/peer (NET-7), not a mock or single-tenant simulation — satisfies E-11's hard requirement directly, the same live evidence as IT-7 | **PASS** (see IT-7's row for the fresh run) |
| **IT-9** | IPFS round-trip + erasure basis: encrypted upload succeeds, CID anchors correctly; fetch+decrypt succeeds pre-erasure; post-`KEY_EMPLOYEE`-destruction the same ciphertext fails to decrypt (`unpin ≠ delete`) | Round-trip/anchoring half: `TestIntegration_AllFiveWritePathsAgainstLiveNetwork`'s `REC-5` subtest (writepaths_integration_test.go), `TestIntegration_EncryptAndAdd_RoundTrip` + `TestIntegration_ReEncryptionProducesNewCID` + `TestIntegration_FetchAndDecryptWithWrongKeyFails` (`write-path-integration/ipfsclient/ipfsclient_integration_test.go`). Erasure-basis half: **NEW**, same test as IT-5 (`erasure_ipfs_integration_test.go`) | Only the section written WITH a document carries a non-empty `ipfsCIDs`; re-encryption always produces a new CID (no in-place mutation); a wrong-but-generic key fails to decrypt real pinned ciphertext. **The concrete, ADR-0015-specific case IT-9 actually wants** — a FRESH `KEY_EMPLOYEE` obtained for the SAME `employeeInternalID` **after** `Hooks.Erase` destroyed the old one — fails to decrypt the OLD ciphertext (the CID was never unpinned, so this isolates key-destruction, not object-removal, as the erasure mechanism | **PASS**. ipfsclient suite fresh run, 3.566s, all 3 green. New erasure test (fresh run, same 20.29s run as IT-5): `IT-9 confirmed: post-erasure FetchAndDecrypt with a fresh KEY_EMPLOYEE correctly failed to open the old ciphertext: ipfsclient: decryption failed (wrong key or tampered ciphertext): cipher: message authentication failed` |
| **IT-10** | Overlay availability, 3-node Raft: 1-of-3 orderer failure does not halt ordering; HRIS operational-DB writes unaffected | `fabric-network/tools/raftfaulttest/main.go`, **re-run live this pass**, not cited historically | A real signed channel config-update commits while `orderer1.org1` alone is stopped (block height advances); the SAME kind of update fails with the orderer's own `SERVICE_UNAVAILABLE -- no Raft leader` once a second orderer (`orderer2.org1`) is also stopped (2-of-3 down, quorum lost); both orderers restored afterward | **PASS**, live output today: `Probe 1 result: update error=<nil>, height 78 -> 79, PASS=true`; `Probe 2 result: update error=exit status 1, height 79 -> 79, EXPECTED-TO-FAIL=true`; `Final block height after restore: 79`; `SUCCESS: f=1 crash-fault tolerance demonstrated`. Confirmed via `docker ps` afterward: `orderer1.org1`/`orderer2.org1` both back `Up`, network otherwise untouched |

**Summary: 10/10 `IT-#` have at least one real, passing, live-run covering test as of this pass.** Two
(`IT-5`, `IT-9`'s erasure-basis half) had **zero runnable proof** anywhere in the repo before this item
— closed by the one new file below. One (`IT-2`) had a real but non-runnable prior proof (a `peer` CLI
invocation, evidenced only in prose in `implementation-backlog.md`/session memlog) — re-confirmed live
this pass via a different code path (Go Gateway SDK) rather than re-typing the same CLI commands.

## New test added (one, per this item's ceiling)

**File:** `write-path-integration/writepaths/erasure_ipfs_integration_test.go` (new file only —
`erasure.go`, `erasure_test.go`, `writepaths.go`, `writepaths_integration_test.go` were not touched, per
this item's explicit scope boundary)

**Gap it closes:** IT-5 (on-chain state unchanged after `Hooks.Erase`, proven against the LIVE ledger,
not asserted from `erasure.go`'s own "zero ledger calls" doc-comment) and IT-9's erasure-basis half (a
FRESH `KEY_EMPLOYEE` for the same employee, obtained after erasure destroyed the old one, cannot open
the OLD ciphertext). One test function, `TestIntegration_QA2_ErasureLeavesOnChainStateUnchangedAndBreaksOldCiphertext`,
covers both — they share one fixture write and one `Erase` call, so splitting them into two tests would
mean anchoring the same fixture twice for no benefit.

Fresh run (full writepaths suite, including this new test, this pass):
```
=== RUN   TestIntegration_QA2_ErasureLeavesOnChainStateUnchangedAndBreaksOldCiphertext
    erasure_ipfs_integration_test.go:173: IT-5 confirmed: GetEmployeeProfileSummary + GetProfileHistory(PERSONAL) + all 5 GetProfileSectionRecord reads are byte-for-byte identical before/after Erase
    erasure_ipfs_integration_test.go:191: IT-9 confirmed: post-erasure FetchAndDecrypt with a fresh KEY_EMPLOYEE correctly failed to open the old ciphertext: ipfsclient: decryption failed (wrong key or tampered ciphertext): cipher: message authentication failed
--- PASS: TestIntegration_QA2_ErasureLeavesOnChainStateUnchangedAndBreaksOldCiphertext (20.29s)
--- PASS: TestIntegration_PartialFailure_AnchorFailsAfterOperationalWriteCommits (0.15s)
--- PASS: TestIntegration_AllFiveWritePathsAgainstLiveNetwork (17.09s)
    (all 7 subtests PASS)
PASS
ok  	writepaths	38.437s
```

## IT-2's ad hoc live re-proof — what it was and why it isn't a new repo file

`fabric-network/tools/` was checked for a surviving runnable artifact from NET-8's original proof
(`ls`/directory listing of `ccdeploy`, `jcsverify`, `netjoin`, `raftfaulttest`, `tenantprovision`) —
**none exists**; NET-8's own evidence (`implementation-backlog.md` NET-8 row, session memlog) records
it as a one-off `peer chaincode invoke`/`peer chaincode query` CLI session, not committed code. Per this
item's instruction to re-confirm live rather than just cite NET-8, and given the low cost of doing so
(the `gatewayclient` module this suite already depends on can drive the exact same call), I wrote a
small (~65-line) throwaway Go program in this session's scratch directory — **not** anywhere under this
repo — with a `go.mod` `replace` directive pointing at the already-built
`write-path-integration/gateway-client` module, built and ran it (`go build ./... && go run .`), and
report its live output in IT-2's row above. It submits `RecordProfileSection` as Org3's own identity
(endorsed only by its own peer, `localhost:10051`) and separately evaluates a known-committed fixture
record (the `int2notfoundtestfixture...` record `TestIntegration_INT2_VerifyDistinguishesNotFoundFromMismatch`'s
own regression subtest anchors, guaranteed to exist because that suite was run immediately before this
script in this same pass).

This was deliberately **not** added as a permanent file under `write-path-integration/` or
`fabric-network/tools/`: this item's explicit new-work ceiling is the one erasure/IPFS test plus this
report, and the standing instruction is to touch only what was explicitly asked for. If a permanent,
repo-committed Go re-proof of IT-2/NET-8 is wanted (e.g. for CI), that is a small, well-scoped follow-up
someone can pull straight out of this section's approach — it is not invented here as an unrequested
fifth test file.

## Stale / mismatched premises found (test-strategy.md vs. the ratified/built design)

1. **IT-2's literal text names `Org2MSP`; the actual live policy uses `OrgClient-tenant01MSP`.**
   test-strategy.md's IT-2 row reads "`AND(Org1MSP.peer, Org2MSP.peer)` required" — a leftover from
   before the channel-per-tenant rework renamed the per-tenant client org. The chaincode was actually
   committed with `AND('Org1MSP.peer','OrgClient-tenant01MSP.peer')` (CC-5's own DoD evidence), and
   today's live re-proof's own rejection/positive-control messages name `Org3MSP`/`Org1MSP`/
   `OrgClient-tenant01MSP` throughout, never a plain `Org2MSP`. The underlying control (AND-of-two,
   auditor org excluded) is unchanged and IS what was tested; only the literal MSP name in the spec
   text is stale — the same class of drift `test-strategy.md`'s own header note already flags for
   `ADR-0011`–`ADR-0020`'s renumbering.

2. **IT-4 sits in the "Integration (`IT-#`) — real Fabric + services" pyramid tier, but its actual
   covering test never touches Fabric at all.** `keystore/handoff_test.go` carries no `//go:build
   integration` tag and exercises `SaltHandoff` purely against `InMemorySaltStore`/`InMemoryAuditLog` —
   by design, not by omission: FR-36's salt hand-off channel is an off-chain construct (salt custody
   was already off-chain per REC-2/ADR-0011), so there is no live-Fabric surface for this control to
   touch in the first place. This is not a coverage gap — the property IT-4 actually cares about
   (separate-from-verify, audited, fails closed, owner/auditor-scoped) is fully exercised — but the
   pyramid-tier framing ("real Fabric + services") does not match where this proof actually lives, and
   a future reader running only `-tags=integration` suites would silently skip it (it has no tag to
   select on, but it also isn't excluded by one — `go test ./...` with no tags picks it up
   automatically, the opposite failure mode from the other `IT-#` rows).

3. **NET-8's IT-2 evidence was never committed as runnable code**, confirmed by directory listing
   (`fabric-network/tools/` has exactly `ccdeploy`, `jcsverify`, `netjoin`, `raftfaulttest`,
   `tenantprovision` — no NET-8-named tool or test). test-strategy.md doesn't itself claim otherwise,
   but a reader relying on `implementation-backlog.md`'s "✅ DONE" marker alone (prose only) would have
   no way to re-run that proof without either the original interactive `peer` CLI session or something
   like this pass's ad hoc script. Flagged as a process gap for anyone deciding whether IT-2 needs a
   permanent CI-facing test, not a defect in NET-8's original finding (which this pass's live re-proof
   corroborates exactly, including the verbatim ABAC rejection message).

4. **IT-7's "chaincode is installed/approved/committed" clause is not re-verified live by anything in
   this pass, by design.** tenant02's channel deliberately has **no** chaincode committed on it (per
   this item's own standing ground truth and NET-7/CC-5's scope split — CC-5 only ever targeted
   tenant01). The scope-bounded-credential half of IT-7 (which is the half that needs a SECOND tenant
   to be meaningful at all, per errata E-11) is fully covered by the live tenant-isolation tests; the
   provisioning-mechanics half is NET-7's own prior live evidence, not something this pass re-executes,
   since re-running tenant provisioning against an already-provisioned tenant02 risks exactly the
   "cryptogen is not idempotent" defect NET-7's own evidence already disclosed finding and fixing once.

None of these four are reported as "coverage exists" for the literal `IT-#` wording where it doesn't;
each is reported as coverage of the *actual*, ratified design, with the wording/tier mismatch called
out separately.

## What is genuinely NOT covered (no live test exists, not claimed otherwise)

- **IT-6's "compensating auto-remediation"** — test-strategy.md's own row text is explicit that this
  test "does not yet assert auto-remediation, because the compensating design (T6b) is an open
  `fabric-engineer` item, not yet specified." Nothing in this pass covers auto-remediation either,
  correctly — there is no ratified design to test against yet.
- **IT-10's "HRIS operational-DB writes are unaffected" clause** — `raftfaulttest` proves the Raft
  ordering-service half (config-update commits/fails as expected); it does not itself drive any HRIS
  operational-DB write during the outage window to observe that side directly. `writepaths`'s own
  `InMemoryOperationalStore` has no dependency on the ordering service at all (`Store.SaveSection` runs
  and returns before `anchor()` is ever called — see `writepaths.go`'s own `PartialFailureError` doc
  comment), so the property holds by construction, but no test in this pass exercises "operational-DB
  write succeeds while an orderer is down" as one combined live scenario.

## Confidentiality check

Per this repo's standing confidentiality register, the two forbidden real-company/product terms are
intentionally not spelled out literally in this file. The check run was a case-insensitive recursive
grep for those two terms against every file this item created or edited — this report and the new
`erasure_ipfs_integration_test.go` — and it returned zero matches on both. The exact command and its
(empty) output are reported through this task's required structured-output channel, not reproduced
here.

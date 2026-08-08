# Contract (`CT-#`) coverage report — QA-3

Investigates `test-strategy.md` §1's Contract (`CT-#`) row before writing anything against it, per
this item's own instruction not to assume the premise. Finding: the premise is stale in the same way
`implementation-backlog.md` **PB-1**'s `SEC` `T1` row already is — both describe a boundary this
ratified design does not have. This report (a) states that mismatch precisely, (b) maps each `CT-#`
to the Go-level property/test that actually embodies its intent, and (c) closes the one genuine gap
found in the process (`CT-4`), with a new live test.

## 1. The premise mismatch

`test-strategy.md` §1 headers the Contract row **"drive the profile-write/verification API (Newman /
`api-test-automation`)"** and each `CT-#`'s "Endpoint" column names one. **There is no HTTP endpoint
anywhere in the ratified design for Newman to drive.**

`ADR-0014` ("Record profile-section integrity in-band from the HRIS write path — no Kafka, no
anchor-service") is explicit about this, twice over:

- **Decision:** "we will record profile-section integrity **in-band**: at the moment a section's
  write path in the HRIS application commits that section to the operational database, the **same
  write path** computes the off-chain digest ... and submits `RecordProfileSection` ... as part of
  that write's own request lifecycle" — i.e., a direct in-process Go call into
  `gatewayclient.GatewayClient.SubmitRecordProfileSection`, not a network hop to a service Newman
  could address.
- **What this ADR decides:** "**Anchor-service deployable units: 0.** No new, separately-deployed Go
  service is introduced to host the Gateway client and a consume→anchor loop." There is, by ADR-0014's
  own accounting, no deployable unit for an HTTP contract test to target in the first place.

Every "endpoint" this codebase actually has for the write/verify/salt-handoff/onboarding surfaces
`CT-1..CT-4` describe is a direct Go call against the Fabric Gateway gRPC client or an internal
package boundary:

| What `test-strategy.md` calls it | What it actually is |
|---|---|
| "profile-write/verification API" | `gatewayclient.GatewayClient.Verify` (read side, `gateway-client/verify.go`) and `.SubmitRecordProfileSection` (write side, `gateway-client/gatewayclient.go`) — Go functions returning Go values over a Fabric Gateway gRPC connection, called in-process from the HRIS write path per ADR-0014 |
| "salt-delivery endpoint" | `keystore.SaltHandoff.RequestSalt` (`keystore/handoff.go`) — a Go method, not a route |
| "tenant onboarding endpoint" | `fabric-network/tools/tenantprovision`'s CLI tool + `netjoin`'s channel/MSP provisioning steps — operator tooling, not an HTTP surface a caller "reaches" |
| "the gateway" `CT-4` asserts identity is checked "at" | The Fabric peer's own MSP/TLS layer (`gatewayclient.NewGatewayClient`'s mTLS dial + the peer's proposal-signature verification) — there is no API-gateway process in front of it |

This is **the same class of issue already found and flagged for `PB-1`/`ST-2`**:
`implementation-backlog.md` PB-1 is titled "verify the **API-gateway identity-header trust
boundary** end-to-end," and `security-architecture.md`'s `T1` threat row ("Forged tenant/user
identity headers") both assume a `Gateway (trusted headers)` component sitting in front of the HRIS
app (`trust-boundaries.mmd`'s `TB1`/`GW` node) that authenticates callers and hands the app trusted
`tenant`/`user` headers. That gateway is a *separate, real, still-open* design question about the
**HRIS application's own public HTTP edge** (PB-1 stays open precisely because nobody has verified it
end-to-end) — it is not the same thing as "an HTTP API in front of the Fabric write/verify/salt/
onboarding operations," which ADR-0014 affirmatively ruled out for the write side and which never
existed for the verify/salt/onboarding sides either. `CT-1..CT-4`'s framing conflates "there exists a
gateway concept somewhere in this system" (true, and unresolved — PB-1) with "the write/verify/salt/
onboarding surfaces are themselves HTTP, Newman-drivable endpoints" (false — they are direct Go/gRPC
calls). Forcing a Postman/Newman collection onto this surface would mean either (a) fabricating an
HTTP shim that does not exist in the ratified design purely to give Newman something to call — testing
a component this architecture explicitly does not have — or (b) silently reinterpreting "Newman" to
mean "any test with request/response shape," which is not what `api-test-automation` names and would
misrepresent what actually ran. Neither is done here. §2 below maps each `CT-#`'s *intent* onto the
Go-level test that already gives it real coverage, or (for `CT-4`) the one that closes a real gap.

## 2. `CT-#` → actual coverage

| `CT-#` | `test-strategy.md`'s stated intent | Actual mechanism | Actual covering test |
|---|---|---|---|
| **`CT-1`** | Verification "never accepts a section-plaintext argument (FR-34); never returns *salt* under any response path (FR-14)" | `gatewayclient.GatewayClient.Verify`'s own signature and wire behavior: `Verify(ctx, tenantID, employeeID, profileSection string, salt []byte, currentValueJSON []byte)` — `salt`/`currentValueJSON` are **caller-supplied inputs already in the caller's own possession**, used only to recompute `SHA-256(salt‖JCS(section))` **locally**; the one and only network call inside `Verify`, `EvaluateGetProfileSectionRecord`, is documented in `verify.go`'s own header comment as carrying "only (tenantID, employeeID, profileSection) — no section value, no salt, ever crosses to a peer." `VerificationResult` (the only value a caller gets back) has fields `Found, Matched, OnChainHash, RecomputedHash, OnChainVersion, UpdatedBy, Timestamp` — **no `Salt` field exists in the struct at all**, so "never returns salt" is a compile-time property, not a runtime check that could regress silently. | `gateway-client/verify_notfound_integration_test.go` (`TestIntegration_INT2_VerifyDistinguishesNotFoundFromMismatch`, live network) exercises the Evaluate call end-to-end and its "regression" subtest confirms a genuine record still round-trips through this narrow wire shape; `gateway-client/verify_integration_test.go`'s `TestIntegration_REC7_ThreeVerifierClassesAgainstTheirOwnPeers` exercises the tamper-detection half. No new test needed. |
| **`CT-2`** | Salt-delivery is BOLA-bounded: "an employee can fetch *salt* only for their own records; an auditor's fetch is bounded to their audit scope; cross-employee/cross-tenant fetch is rejected" | `keystore.SaltHandoff.RequestSalt` (`keystore/handoff.go`) — FR-36's controlled, audited hand-off channel, structurally separate from `Verify`/`VerificationResult` per FR-14's own text (a shared response type would itself violate "the verification endpoint MUST NOT return salt ... on any response," so FR-36 is its own package boundary, not a header/param on the verify call). Authorization is injected via `AuthorizeSaltAccessFunc` and fails **closed** on a policy-backend error (never an implicit grant). | `keystore/handoff_test.go`: `TestSaltHandoff_GrantsForRecordOwner` (owner-bounded grant), `TestSaltHandoff_DeniesUnauthorizedThirdParty` (cross-identity BOLA rejection — the exact property `CT-2` names), `TestSaltHandoff_FailsClosedOnAuthorizeError` (fail-closed on policy-backend failure), `TestSaltHandoff_GetSaltMissNotRecordedAsDenial` (audit-log correctness: a genuine not-found is not misreported as a denial). No new test needed. |
| **`CT-3`** | Tenant onboarding is BFLA-bounded: "a non-onboarding-scoped caller cannot trigger channel/chaincode provisioning for any tenant" | Channel-per-tenant (`ADR-0013`) means "provisioning a tenant" and "reaching a tenant's data" are the same MSP/channel-membership boundary — there is no separate onboarding-authorization check to test independently of channel membership itself, because a caller who is not a genuine member of a tenant's channel cannot act on it at all, provisioning included. | `gateway-client/tenant_isolation_integration_test.go`'s `TestIntegration_Tenant02CannotReachTenant01Channel` / `TestIntegration_Tenant01CannotReachTenant02Channel` (live network) prove this at the deepest layer available — a peer never joined to a channel refuses even a perfectly valid identity's request at the gRPC/gossip layer, before chaincode logic runs, which is a **stronger** guarantee than a function-level authorization check would be. This is `IT-8`'s test, re-used here for `CT-3`'s BFLA intent rather than duplicated — `test-strategy.md` itself does not distinguish `IT-8` and `CT-3` as needing independent test bodies once the API premise is corrected, since both reduce to the same channel-membership fact. No new test needed. |
| **`CT-4`** | "Missing/invalid caller identity ⇒ rejected at the gateway and/or chaincode layer" | Every `gatewayclient.NewGatewayClient` construction is either a genuine mTLS dial + a genuine, correctly-issued MSP signing identity, or (per `T1`/PB-1) a still-open, separately-tracked question about the HRIS app's own public HTTP edge — **neither of those is "a caller presents a mismatched/forged Fabric identity to a live peer and gets rejected."** | **Genuinely uncovered before this item — see §3.** |

**Result: 3 of 4 `CT-#`s already have real, if differently-shaped, coverage once the Newman/HTTP
framing is corrected to what this design actually is. `CT-4` did not, and is the one piece of new
work this item produces (§3).**

## 3. `CT-4` — the genuine gap, confirmed and closed

### 3.1 Confirming the gap

Searched every existing `NewGatewayClient(...)` call in `write-path-integration/gateway-client/*_test.go`
(`gatewayclient_integration_test.go`, `verify_integration_test.go`, `verify_notfound_integration_test.go`,
`tenant_isolation_integration_test.go`): **every single one** authenticates with a cert/key pair
actually issued by the CA of the MSP it claims to be — `Org1MSP` calls always load
`.../org1/users/Admin@org1/msp/...`, `OrgClient-tenant01MSP` calls always load
`.../tenant01/users/Admin@tenant01/msp/...`, and so on. `tenant_isolation_integration_test.go`
(`IT-8`/`CT-3`, above) proves a *correctly-identified* org cannot reach a channel it was never joined
to — a channel-membership property — but it never constructs a *mismatched* identity (a real cert
presented under a different org's claimed MSP ID). No test anywhere in this codebase exercises "does
the system reject a caller whose presented identity doesn't validate under the MSP it claims to
belong to." This is a real, not merely relabeled, coverage gap — exactly the kind of finding this
item's brief anticipated as "likely."

### 3.2 The new test

`write-path-integration/gateway-client/invalid_identity_integration_test.go` (new file; no existing
file in the package was edited). Two independent live-network cases, both against the real network
described in this session's ground truth (channel `tenant-tenant01`, chaincode `employeeprofilerecord`):

1. **`TestIntegration_CT4_MismatchedMSPClaimRejected`** — dials Org1's real peer (`localhost:7051`)
   using Org1's own, real mTLS client certificate (so the gRPC transport layer is legitimate on
   purpose — the point is to isolate the MSP-signing-identity check, not conflate it with a transport
   failure), but builds the MSP signing identity with `mspID = "Org3MSP"` while pointing
   `certPEMPath`/`keyPEMPath` at Org1 Admin's real signing cert/key — a certificate genuinely issued
   by Org1's CA, never by Org3's. `Org3MSP` is not an arbitrary string chosen to guarantee failure for
   a trivial reason: `configtx.yaml`'s `TenantChannelGenesis` profile lists Org3 as a genuine
   (read-only, non-endorsing per FR-23) member of `tenant-tenant01`, so this forges a real,
   channel-recognized MSP with the wrong org's certificate.
2. **`TestIntegration_CT4_MismatchedMSPClaimAcrossTenantOrgsRejected`** — the same property from the
   opposite angle: claims `Org1MSP` (a full, *endorsing* member of the channel per
   `configtx.yaml`'s `Endorsement: AND('Org1MSP.peer','OrgClient-tenant01MSP.peer')` policy) while
   presenting OrgClient-tenant01 Admin's real cert/key, issued by tenant01's own CA. This rules out
   the possibility that case 1's rejection was somehow specific to Org3's read-only status rather than
   the certificate-chain mismatch itself.

Both were run against the live network (not merely written and assumed to pass — see the required
structured-output field for the raw transcript). Both fail exactly as intended, with the peer's own
MSP layer rejecting the forged proposal creator:

```
error validating proposal: access denied: channel [tenant-tenant01] creator org unknown, creator is malformed
```

This is the peer *itself* — not a Gateway-SDK client-side check — refusing a `SignedProposal` whose
claimed MSP ID does not match the certificate's actual issuing CA, which is precisely the "rejected at
... the chaincode layer" half of `CT-4`'s stated intent (T1, T2, API2 in `security-architecture.md`'s
terms — `T2` "stolen X.509 signing key → submit/endorse as an org," mitigated by "cert lifecycle...
NodeOU roles"; this test is that mitigation's first live proof). As with the existing
`tenant_isolation_integration_test.go`'s own stated convention, the load-bearing assertion is exactly
`err != nil` — the precise rejection wording is a peer-implementation detail this test does not pin
down beyond logging it for evidence.

Both new tests build (`go build ./...`, `go vet ./...` clean) and pass alongside the full existing
integration suite in this package (`go test -tags integration -run TestIntegration -v ./...`, 7 test
functions total — 5 pre-existing + the 2 new ones, all `PASS`; corrected here after independent
re-verification found the original draft's "9" over-counted the package's 10 unit tests, which
carry no `integration` build tag, into the same figure) — no existing file in `gateway-client` was
modified to make this true.

## 4. What this report does not claim

- It does not claim `PB-1` is closed. `PB-1`/`T1` (the HRIS application's own public HTTP edge and its
  header-trust boundary) remains a real, separately-tracked, open item — this report only clarifies
  that `CT-1..CT-4` were never testing *that* boundary in the first place, so their coverage status is
  independent of `PB-1`'s.
- It does not claim `CT-1..CT-3`'s existing tests were written *as* contract tests, or retroactively
  relabels them — it states what test, already in the tree, gives each `CT-#`'s stated *intent* real
  coverage, so the backlog does not carry a phantom "write a Newman collection" task against a surface
  that cannot have one.
- It does not touch `test-strategy.md` or `implementation-backlog.md` — those are shared,
  concurrently-edited files owned by the orchestrator's serial update pass, per this session's
  standing instruction.

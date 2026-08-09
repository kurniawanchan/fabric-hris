# Codebase Map — Where Everything Lives, and How to Extend It

A guide for anyone continuing this work. Every claim below is traceable to a specific,
already-closed backlog item in `agent-suite/06-roadmap/implementation-backlog.md` — that file
remains the authoritative, evidence-backed record of what was built and why; this document is a
navigational map on top of it, not a replacement.

## 1. High-level shape

```
fabric-network/          Fabric network topology, chaincode, and Go bring-up tools (NET-*, CC-*)
write-path-integration/  Off-chain digest/identity/erasure logic + Fabric Gateway client (REC-*, INT-*)
integration-bridge/      Standalone HTTP service fronting the write-path hooks (Epic 1, closes G-10)
ipfs-cluster/             Dev-grade 2-node IPFS private swarm (REC-5)
qa-tests/                Every test suite + evaluation report (QA-*)
deploy/                  Generic, NEVER-APPLIED deployment artifacts (DEP-*) — see §6
agent-suite/             Design documentation: ADRs, PRD, security architecture, the backlog itself
docs/                    This file and its siblings — practical how-to guides
```

**Fabric's two client-facing APIs map onto exactly two files, on opposite sides of the peer
boundary — nothing else in this repo uses either.** ([overview](https://hyperledger-fabric.readthedocs.io/en/latest/sdk_chaincode.html))

| Official API | What it's for | Used exactly here | Entry point |
|---|---|---|---|
| **Contract API** (`fabric-contract-api-go/contractapi`, `v1.2.2`) | Writing the smart contract itself | `chaincode/employeeprofilerecord/chaincode/*.go` (§2) | `chaincode/employeeprofilerecord/main.go:28` — `contractapi.NewChaincode(&chaincode.SmartContract{})` |
| **Application API** — historically the full `fabric-sdk-go`/java/node SDKs; superseded from Fabric 2.4+ by the **Gateway client API**, which is what's actually vendored (`hyperledger/fabric-gateway`, `v1.12.0`) | Writing an off-chain client that connects to a peer and calls that contract | `write-path-integration/gateway-client/gatewayclient.go` (§3) | `gatewayclient.go:74` — `client.Connect(id, ...)`, then `SubmitTransaction`/`Evaluate*` |

`fabric-network/tools/*` (`netjoin`, `ccdeploy`, `tenantprovision`, etc.) use **neither** — they
`docker exec` into the `fabric-tools-net` helper container and drive the real `peer`/`osnadmin`/
`configtxgen` CLI binaries directly (see `docs/QUICKSTART.md` §4), never an SDK.
`integration-bridge/`'s HTTP handlers don't touch either API directly either — they call
`write-path-integration/writepaths`' hooks, which are the only thing holding a Gateway connection.

## 2. `fabric-network/` — the ledger side

| Path | What it is | Backlog item |
|---|---|---|
| `network/configtx/configtx.yaml`, `network/crypto-config/*.yaml` | Channel/org topology, MSP definitions | `NET-1` |
| `network/fabric-ca/` | Fabric CA server configs + enrollment runbook (optional path — see `docs/QUICKSTART.md` §3) | `NET-2` |
| `network/compose/network-docker-compose.yaml`, `ca-docker-compose.yaml` | Peer/orderer/CA container bring-up | `NET-4` |
| `tools/netjoin/` | Channel create+join automation (idempotent) | `NET-3` |
| `tools/raftfaulttest/` | Proves 1-of-3 Raft orderer failure doesn't halt ordering | `NET-6` |
| `tools/tenantprovision/` | Per-tenant onboarding automation (crypto + channel + join, one command) | `NET-7` |
| `tools/ccdeploy/` | Chaincode install/approve/commit/smoke-test (CCaaS) | `CC-5` |
| `tools/jcsverify/` | Verifies the pinned RFC 8785 JCS library against its own official test vectors | `PB-3` |
| `tools/pilscan/` | Full-ledger raw-block scanner (world state + tx args + events) — the P0 confidentiality-scan tool | `QA-5` `ST-1` |
| `chaincode/employeeprofilerecord/chaincode/` | The actual smart contract | `CC-1..4` |
| `chaincode/employeeprofilerecord/chaincode/asset.go` | On-chain schema — **the only 12 fields that may ever exist on a record**; extending this is a ratified-design change, not a casual edit | `CC-1` |
| `chaincode/employeeprofilerecord/chaincode/record_profile_section.go` | The write path; carries `[FLAGGED DEVIATION]`/`[FLAGGED GAP]` comments for two known, deliberate divergences from the original spec (deterministic RecordID; no identifier-shape validation, `G-31`) | `CC-2` |
| `chaincode/employeeprofilerecord/chaincode/identity.go` | ABAC/authorization — currently org-level MSP allow-list, **not** the attribute-based ABAC `ADR-0005` actually ratified (`G-30`, disclosed at that ADR) | `CC-4` |
| `chaincode/employeeprofilerecord/chaincode/queries.go` | The three read-only queries — note `GetProfileSectionRecord` deliberately returns a reduced view; `GetProfileHistory` is the only one exposing `ipfsCIDs` et al. | `CC-3` |

## 3. `write-path-integration/` — the off-chain integration side

A Go workspace (`go.work`) of 4 independent modules — **always `cd` into the specific module
directory before `go build`/`go test`**, see `docs/QUICKSTART.md` §8.

| Module | What it is | Backlog item(s) |
|---|---|---|
| `gateway-client/digestbuilder.go` | `ComputeDataHash`/`ComputeEmployeeID`/`ComputeUpdatedBy` — the JCS+HMAC-SHA256 construction | `REC-1` |
| `gateway-client/gatewayclient.go` | The retained Fabric Gateway connection + submit/evaluate wrapper, with automatic retry on the two EXPECTED rejection classes | `REC-3` |
| `gateway-client/verify.go` | Client-side verification (`Verify()`) — `Found`/`Matched` distinguish "never anchored" from "tampered" | `REC-7`, `INT-2` |
| `keystore/keystore.go` | `SaltStore`/`EmployeeKeyStore`/`DocumentKeyStore` interfaces + in-memory reference implementations (demo-grade, not production custody) | `REC-2` |
| `keystore/handoff.go` | The audited, access-controlled salt hand-off channel (FR-36), deliberately separate from `Verify()` | `INT-3` |
| `writepaths/writepaths.go` | The FIVE write-path hooks (one per real HRIS action) — **this is the file a real integration hooks into**, see `agent-suite/context/` for where that hook point lives in the real application | `REC-4`, `INT-1`, `INT-5` |
| `writepaths/writepaths.go` `PartialFailureError`/`OnPartialFailure` | The provisional (labeled `[ASSUMPTION]`, not ratified) outage-handling policy and its detectability hook | `INT-1`, `INT-5` |
| `writepaths/erasure.go` | The 4-step crypto-shred deletion (`Hooks.Erase`) | `REC-6` |
| `ipfsclient/ipfsclient.go` | Encrypt-before-add + cross-node pin for IPFS-hosted supporting documents | `REC-5` |

**If you're adding a 6th `ProfileSection`**: touch, in order, `chaincode/asset.go`'s
`IsValidProfileSection` enum, `writepaths.go` (a new hook function following the existing five's
exact shape), and `AllProfileSections` (used by `REC-6`'s erasure iteration) — then re-run `QA-1`'s
unit coverage and `QA-5`'s `ST-5` tamper-detection pattern for the new section.

**If you're adding a 3rd tenant**: `tools/tenantprovision` is the whole story — see
`docs/QUICKSTART.md` §10, and read its idempotency caveats first. Do not hand-roll the crypto
material or channel creation.

## 4. `integration-bridge/` — the write-path HTTP gateway

A standalone Go service (its own `go.mod`, `require`+`replace` on all 4 `write-path-integration/`
modules) that hosts the retained `gatewayclient.GatewayClient` connection and exposes the five
write-path hooks over HTTP — one route per `ProfileSection`. Closes grounding gap `G-10`
(`agent-suite/05-adr/ADR-0022-integration-bridge-gateway-client-host.md`); built via a separate
BMAD epic/story workflow (`_bmad-output/planning-artifacts/epics.md`, `sprint-status.yaml`), not
the `NET-*`/`CC-*`/`REC-*` backlog numbering above — see `README.md`'s own note on why this
directory is tracked differently.

**Package layout** (a later clean-architecture-flavored refactor split the original flat
`package main` into two packages, expressing dependency direction without changing the AD-1
pipeline paradigm — `ARCHITECTURE-SPINE.md` is unchanged by this): `cmd/integrationbridge/` is the
composition root (`package main` — config loading, concrete client wiring, route table, HTTP
server lifecycle); `internal/pipeline/` (`package pipeline`) is the 5-stage pipeline itself, and by
design never imports `gatewayclient`/`ipfsclient`/`keystore` — only `writepaths`, for
`*writepaths.PartialFailureError`. Building/running now targets `./cmd/integrationbridge`
explicitly (see §9 gotcha below).

| Path | What it is |
|---|---|
| `cmd/integrationbridge/main.go` | Process entrypoint: loads config, constructs the `GatewayClient`/`Hooks` exactly once, builds the mux (`buildMux`), runs the HTTP server, handles `SIGTERM`/`SIGINT` graceful shutdown |
| `cmd/integrationbridge/config.go` | `LoadConfig` — fails fast, naming every missing `BRIDGE_*` env var at once; `ResolveHTTPAddr`/`ResolveDispatchTimeout` — optional, silently default (never fail) on an unset/invalid value |
| `cmd/integrationbridge/wiring.go` | `buildHooks` — wires `writepaths.Hooks` to the in-memory store implementations (no persistent store exists anywhere in this codebase yet — every process restart loses every salt/`employeeKey_i`) and a real `ipfsclient.Client` |
| `internal/pipeline/auth.go` | `[auth]` stage — constant-time `X-Api-Key`/`X-Company-ID` check against one configured tenant (AD-2: one deployment serves exactly one tenant) |
| `internal/pipeline/validate.go` | `[validate]` stage — `newValue` is captured as `json.RawMessage`, never re-parsed, so large integers (e.g. a PAYROLL bank-account number past `float64`'s 2^53 boundary) can never be silently corrupted; body size capped via `http.MaxBytesReader` |
| `internal/pipeline/dispatch.go` | `[dispatch]` stage — the one shared timeout budget (`AD-4`); blocks synchronously on the real Hooks call, never races a goroutine against the deadline, and never discards a call that completed successfully just because it ran past budget |
| `internal/pipeline/classify.go` | `[map-error]` stage — the single place implementing `AD-3`'s decision table (`committed`/`partial_failure`/`rejected`/`error`); the one deliberate exception to the "no sibling-module imports" rule above, since `writepaths` is this system's domain core, not infrastructure |
| `internal/pipeline/respond.go` | `[respond]` stage — writes the `{status, recordID?, detail?}` envelope; HTTP status is pure transport (`AD-3`) |
| `internal/pipeline/pipeline.go` | The Stage Contract (`bridgectx`) — carries identifiers across the pipeline; not currently read via `context.Value` by any production stage (they use closure capture instead), only by its own test — see the Story 1.2 code review for why this is a documented, not silently overclaimed, gap; re-evaluated (and left as-is) when the package split landed, since the split introduced no new boundary for it to cross |
| `internal/pipeline/handlers.go` | `RegisterProfileSectionRoute` + `RouteConfig` — the one function every one of the 5 routes shares; adding a route means adding a `RouteConfig` value and a `buildMux` call, never a new handler. Exported (unlike the other pipeline internals) because `cmd/integrationbridge/main.go` calls it across the package boundary |
| `internal/pipeline/*_route_test.go`, `cmd/integrationbridge/*_route_integration_test.go` | One pair of test files per route (`personal_route_integration_test.go` also carries the shared `postSection`/`uniqueTestID`/`startPersonalRouteTestBridge` test helpers every other route file reuses) |

**If you're adding a 6th route**: this can't actually happen without a 6th ratified
`ProfileSection` enum value on the chaincode side first (§2's `asset.go` `IsValidProfileSection` and
a matching `writepaths.go` hook, per §3's own "adding a 6th `ProfileSection`" note) — the bridge has
no independent route beyond what the chaincode enum ratifies. Once that exists: add one
`pipeline.RouteConfig{ProfileSection: "...", Dispatch: hooks.YourNewHook}` line to `buildMux` in
`cmd/integrationbridge/main.go`, and a new `<name>_route_test.go` in `internal/pipeline/` following
any existing sibling's shape. **Double-check the literal `ProfileSection` string against the
chaincode enum, not against the `Hooks` method's own name or any prose description** —
`ADDITIONAL`'s dispatched method is named `ApproveFamilyDataChange` and is called "family/dependents
changes" in the epic's own prose, but the only string that is ever a real, routable value is
`"ADDITIONAL"`; this exact naming gap has already caused a real, caught mistake once in this
codebase's own history (Story 1.5's code review).

**No production credential/persistence store exists.** `BRIDGE_API_KEY`/`BRIDGE_COMPANY_ID` are
compared against one static configured value (no rotation, no per-caller keys); `wiring.go`'s
in-memory stores lose all state on restart. Neither is production-ready — see the story files'
own Open Questions sections before this ships to any real deployment.

## 5. `qa-tests/` — every test suite and its report

| Path | Covers |
|---|---|
| `unit-coverage-report.md` | `QA-1` — maps `U-1..U-8` to real tests |
| `integration-coverage-report.md` | `QA-2` — maps `IT-1..IT-10` to real tests |
| `contract-coverage-report.md` | `QA-3` — maps `CT-1..CT-4` to the actual Go-level surfaces (there is no HTTP API in this design, see the report's own premise correction) |
| `performance/` | `QA-4` (Caliper throughput/latency, `RESULTS.md`) and `QA-6` (MVCC contention, `QA6-RESULTS.md`) |
| `security/` | `QA-5`'s six lanes (`ST-1` full-ledger scan, `ST-5` tamper detection, `ST-3`/`ST-4` revocation+transport, `ST-6`/`10`/`11` crypto/key-domain/config, `ST-9` SAST, and the coverage/disposition report for the rest) |
| `evaluation/QA7-dsrm-evaluation-p1-p4.md` | The DSRM predicate verdicts (P1–P4) against real evidence |
| `evaluation/QA8-thesis-numbers-report.md` | The real numbers to replace the thesis's placeholder figures |

See `docs/REPRODUCING-RESULTS.md` for how to re-run each of these yourself.

## 6. `deploy/` — generic artifacts, never applied

Everything under `deploy/` (`helm/`, `provisioning/`, `ipfs-cluster/`, `ci/`, `observability/`) was
built as a **reviewable design artifact only** — none of it has ever been `helm install`-ed,
`kubectl apply`-ed, or connected to a real CI/HSM/observability backend, by explicit, standing scope
decision. Read each subdirectory's own `README.md` first; several disclose real gaps found while
writing them (e.g. `helm/fabric-network/README.md`'s TLS-SAN-vs-Kubernetes-DNS-naming gap) that
would need resolving before a real `helm install` would work.

## 7. Real defects found and fixed this session, by component (troubleshooting reference)

If you hit one of these, it's already been solved once — check here before re-debugging from
scratch. Full detail is always in the relevant backlog row.

| Symptom | Cause | Fixed at |
|---|---|---|
| `configtxgen`/genesis block generation fails on an MSP-ID token in a policy string | Underscore in MSP ID (`OrgClient_<id>MSP`) — Fabric's policy tokenizer rejects `_` | `NET-1` |
| Fabric CA state resets on container restart | Config mounted outside `FABRIC_CA_HOME`, not inside the named volume | `NET-2` |
| `peer channel fetch`/`update` silently sends no client certificate | Missing `--clientauth`/`--certfile`/`--keyfile` — these are separate from `CORE_PEER_TLS_CLIENTCERT_FILE` env vars | `NET-6` |
| `go build ./...`/`go test ./...` fails with "directory prefix . does not contain modules listed in go.work" | Run from the workspace root instead of the specific module directory | every `write-path-integration/*` module |
| A cross-process chaincode business error (not found / stale chain) isn't caught by `errors.Is` | No typed error survives the Gateway RPC boundary — must string-match `err.Error()` (see `isExpectedRetryableRejection`/`isNotFoundRejection` for the established pattern) | `REC-3`, `INT-2` |
| Re-running `cryptogen` for an already-generated org orphans its issued certs | `cryptogen` is not idempotent — regenerates a fresh CA keypair every run, leaving already-issued leaf certs signed by a CA that no longer exists on disk | `NET-7`; recurred network-wide (org1/org3/tenant01 at once, via the raw `crypto-config.yaml` command in `docs/QUICKSTART.md` §1, not `tenantprovision`) at `NET-3`'s 2026-08-09 addendum |
| `osnadmin channel join`/`peer channel join` fails with `x509: certificate signed by unknown authority` right after a crypto-material reset, even though every cert verifies fine on disk | The channel's `.block` genesis file (`docs/QUICKSTART.md` §2) bakes in each orderer's TLS cert **at generation time** — a block generated before a `cryptogen` regen still carries the old cert and must be regenerated with `configtxgen` against the current material before rejoining | `NET-3`'s 2026-08-09 addendum |
| `docker compose up -d` for `network-docker-compose.yaml` fails with `ports are not available ... address already in use` on port `7071` | AnyDesk's default relay port (was `7070`) collides with `orderer0.org1`'s host-side operations-listener mapping; remapped to `7071` in the compose file (container-side `8443` unaffected) | `NET-3`'s 2026-08-09 addendum |
| `docker run --name fabric-tools-net ...` fails with `container name "/fabric-tools-net" is already in use` | The helper container already exists (stopped) from a previous session — `docker start fabric-tools-net` instead of recreating it, once its image/network/mounts are confirmed unchanged | `NET-3`'s 2026-08-09 addendum |
| A peer crashes with "unexpected Previous block hash" on `tenant-tenant02` | Recurring, structural: a dormant/recreated channel's genesis conflicts with a peer's stale local copy — happened twice (`NET-7`, `QA-4`'s `PT-4`) | see `NET-7`'s backlog addendum; recovery = ledger-volume reset + `netjoin` rejoin, never a channel-join/create against `tenant-tenant02` casually |
| Caliper's `fabric:2.5`-bound connector can't connect at all | Zero mTLS support in any published `@hyperledger/fabric-gateway`-based connector version; this network mandates client-cert auth | `QA-4` (patched into the sandboxed `qa-tests/performance/node_modules` only) |
| A stray, untracked binary named `integrationbridge` appears at `integration-bridge/integrationbridge` after building | Only `go build ./cmd/integrationbridge` (or explicitly building that one package) writes this — since the clean-architecture package split, `integration-bridge/` has two packages (`cmd/integrationbridge` + `internal/pipeline`), so a bare `go build ./...` from the module root no longer has a single main package to default the output name against and produces no stray binary at all. Still `rm` it if you build the `cmd/integrationbridge` package directly and don't need the artifact | Story 1.6's code review; now in `.gitignore`; behavior improved by the later package-split refactor |
| "directory prefix . does not contain modules listed in go.work" when building `integration-bridge/` | This gotcha does **not** apply here — `integration-bridge/` is its own standalone module (own `go.mod`), never part of `write-path-integration/go.work`; if you see this exact error, you're almost certainly still `cd`-ed into `write-path-integration/` from a previous command | n/a — different module entirely, no fix needed once you `cd integration-bridge` |

## 8. Conformance against official Fabric 2.5 concept docs (audited 2026-08-09)

Checked this network's actual config/code against all 15 pages under
[Fabric's "Key Concepts" doc tree](https://hyperledger-fabric.readthedocs.io/en/latest/key_concepts.html)
— not from recollection, but by fetching each page (`readthedocs.io` rate-limited direct fetches;
fell back to the identical canonical source, `hyperledger/fabric`'s own `docs/source/*.md` on
GitHub, plus Wayback snapshots) and grepping/reading the repo against every concept it describes.

| Page | Verdict | Key evidence |
|---|---|---|
| `key_concepts.html` | Follows (index page — see rows below) | — |
| `fabric_model.html` | Follows, one concept instantiated differently | Privacy is via **channel-per-tenant** isolation (ADR-0013), not the page's assumed single-shared-channel-plus-PDC model — a ratified choice, not a gap |
| `network/network.html` | Follows | `network/configtx/configtx.yaml:250-321` (per-tenant channel profiles, no `Consortiums`/system channel); anchor peers `configtx.yaml:93-157` |
| `identity/identity.html` | Follows | X.509 cert + signing key per node (`crypto-config/peerOrganizations/*/peers/*/msp/{signcerts,keystore}`); root CA per org (`crypto-config.yaml:68-110`) |
| `membership/membership.html` | Follows | Node OUs enabled (`crypto-config.yaml` `EnableNodeOUs: true`), `admincerts/` correctly empty (superseded by NodeOU `admin` role per Fabric 1.4.3+ guidance), one MSP per org (`Org1MSP`/`OrgClient-tenant01MSP`/`Org3MSP`/`OrdererMSP`) |
| `policies/policies.html` | Follows | Explicit `AND('Org1MSP.peer','OrgClient-tenant01MSP.peer')` signature policy overriding the `MAJORITY Endorsement` default, `configtx.yaml:297-298`; matched again at commit time, `tools/ccdeploy/main.go:57` |
| `peers/peers.html` | Follows | `Org3` (auditor) installs the chaincode for its own read-only `Evaluate` calls but carries no `Endorsement` policy key (`configtx.yaml:148-154`) — never endorses, per ADR-0012 |
| `ledger/ledger.html` | Follows | World state via `PutState`/composite keys (`chaincode/asset.go`); history DB enabled on every peer (`CORE_LEDGER_HISTORY_ENABLEHISTORYDATABASE=true`); append-only — no `DelState`/`PurgePrivateData` anywhere |
| `orderer/ordering_service.html` | Follows | `etcdraft`, 3-node consenter set (`configtx.yaml:193,209-221`); channel participation API not a system channel (`ORDERER_CHANNELPARTICIPATION_ENABLED=true`, `osnadmin channel join` in `tools/netjoin/`); 1-of-3 CFT actually demonstrated, not just claimed, by `tools/raftfaulttest/` |
| `smartcontract/smartcontract.html` | Follows | Contract-vs-chaincode terminology matches exactly (`chaincode/contract.go`'s `SmartContract` inside the `employeeprofilerecord` chaincode); world state never asserted-written at execution time, only via `PutState` |
| `chaincode_lifecycle.html` | Follows | Full package→install→approveformyorg→checkcommitreadiness→commit sequence, `tools/ccdeploy/main.go:142-234`; `metadata.json`'s `type` is literal `"ccaas"` |
| `private-data/private-data.html` | Deliberately not used (ADR-0015) | Zero non-vendor hits for `transient`/`collections-config`/`_implicit_org_`/`GetPrivateData`/`PurgePrivateData` — the retirement ADR's claim is actually true in the code, not just asserted |
| `capabilities_concept.html` | Follows, verified against Fabric's own source | `configtx.yaml:172-175` sets `Application: V2_5` (no separate `V2_0` key) — confirmed correct against `common/capabilities/application.go`'s `V2_0Validation()`/`LifecycleV20()`, which return `ap.v20 \|\| ap.v25`, so `V2_5` alone still satisfies the lifecycle's `V2_0` gate |
| `security_model.html` | **One real gap found — see below** | — |
| `usecases.html` | Fits the pattern (descriptive page, not a technical checklist) | Multi-org consortium + read-only auditor peer + hash-chained history matches the page's "multi-party, need-a-shared-audit-trail" framing |

**The one genuine gap found — logged as `G-37`** (`agent-suite/11-execution/grounding-gaps.md`):
this project's own security architecture (`agent-suite/context/SECURITY-BY-DESIGN.md:42`,
`OWASP-ASVS.md:175`, control **D13**) claims the peer/orderer *operations* service "relies entirely
on mutual TLS with client-cert auth." The live compose file contradicts this — every node sets
`CORE_OPERATIONS_LISTENADDRESS`/`ORDERER_OPERATIONS_LISTENADDRESS`
(`network-docker-compose.yaml:288,331,385,430,475,531,578,629`) with **no** companion
`*_OPERATIONS_TLS_ENABLED`/`CLIENTAUTHREQUIRED` anywhere in the file, and Fabric's documented
default for that listener is TLS **disabled** absent that flag. Every node's host-published
operations port (`9444`–`9447` on peers, `7071`/`8070`/`9070` on orderers) is reachable in
plaintext with zero client-cert check today. Not a PII exposure (those endpoints carry no
chaincode data) — but a real, live contradiction between a ratified control and the deployed
artifact.

**Already-known, deliberate divergences** (re-confirmed still holding, not new): LevelDB not
CouchDB (`ADR-0007`); PDC retired for channel-per-tenant (`ADR-0015`/`ADR-0013`); org-level MSP
allow-list instead of the ratified attribute-based ABAC (`ADR-0005`, `G-30`); `cryptogen` material,
not Fabric CA, is the live trust anchor (`QA-5` `ST-3`, `G-32`).

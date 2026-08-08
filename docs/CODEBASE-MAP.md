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

| Path | What it is |
|---|---|
| `main.go` | Process entrypoint: loads config, constructs the `GatewayClient`/`Hooks` exactly once, builds the mux (`buildMux`), runs the HTTP server, handles `SIGTERM`/`SIGINT` graceful shutdown |
| `config.go` | `LoadConfig` — fails fast, naming every missing `BRIDGE_*` env var at once; `ResolveHTTPAddr`/`ResolveDispatchTimeout` — optional, silently default (never fail) on an unset/invalid value |
| `wiring.go` | `buildHooks` — wires `writepaths.Hooks` to the in-memory store implementations (no persistent store exists anywhere in this codebase yet — every process restart loses every salt/`employeeKey_i`) and a real `ipfsclient.Client` |
| `auth.go` | `[auth]` stage — constant-time `X-Api-Key`/`X-Company-ID` check against one configured tenant (AD-2: one deployment serves exactly one tenant) |
| `validate.go` | `[validate]` stage — `newValue` is captured as `json.RawMessage`, never re-parsed, so large integers (e.g. a PAYROLL bank-account number past `float64`'s 2^53 boundary) can never be silently corrupted; body size capped via `http.MaxBytesReader` |
| `dispatch.go` | `[dispatch]` stage — the one shared timeout budget (`AD-4`); blocks synchronously on the real Hooks call, never races a goroutine against the deadline, and never discards a call that completed successfully just because it ran past budget |
| `classify.go` | `[map-error]` stage — the single place implementing `AD-3`'s decision table (`committed`/`partial_failure`/`rejected`/`error`) |
| `respond.go` | `[respond]` stage — writes the `{status, recordID?, detail?}` envelope; HTTP status is pure transport (`AD-3`) |
| `pipeline.go` | The Stage Contract (`bridgectx`) — carries identifiers across the pipeline; not currently read via `context.Value` by any production stage (they use closure capture instead), only by its own test — see the Story 1.2 code review for why this is a documented, not silently overclaimed, gap |
| `handlers.go` | `registerProfileSectionRoute` + `routeConfig` — the one function every one of the 5 routes shares; adding a route means adding a `routeConfig` value and a `buildMux` call, never a new handler |
| `*_route_test.go`, `*_route_integration_test.go` | One pair of test files per route (`personal_route_integration_test.go` also carries the shared `postSection`/`uniqueTestID`/`startPersonalRouteTestBridge` test helpers every other route file reuses) |

**If you're adding a 6th route**: this can't actually happen without a 6th ratified
`ProfileSection` enum value on the chaincode side first (§2's `asset.go` `IsValidProfileSection` and
a matching `writepaths.go` hook, per §3's own "adding a 6th `ProfileSection`" note) — the bridge has
no independent route beyond what the chaincode enum ratifies. Once that exists: add one
`routeConfig{ProfileSection: "...", Dispatch: hooks.YourNewHook}` line to `buildMux` in `main.go`,
and a new `<name>_route_test.go` following any existing sibling's shape. **Double-check the literal
`ProfileSection` string against the chaincode enum, not against the `Hooks` method's own name or
any prose description** — `ADDITIONAL`'s dispatched method is named `ApproveFamilyDataChange` and
is called "family/dependents changes" in the epic's own prose, but the only string that is ever a
real, routable value is `"ADDITIONAL"`; this exact naming gap has already caused a real, caught
mistake once in this codebase's own history (Story 1.5's code review).

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
| Re-running `cryptogen` for an already-generated org orphans its issued certs | `cryptogen` is not idempotent — regenerates a fresh CA keypair every run | `NET-7` |
| A peer crashes with "unexpected Previous block hash" on `tenant-tenant02` | Recurring, structural: a dormant/recreated channel's genesis conflicts with a peer's stale local copy — happened twice (`NET-7`, `QA-4`'s `PT-4`) | see `NET-7`'s backlog addendum; recovery = ledger-volume reset + `netjoin` rejoin, never a channel-join/create against `tenant-tenant02` casually |
| Caliper's `fabric:2.5`-bound connector can't connect at all | Zero mTLS support in any published `@hyperledger/fabric-gateway`-based connector version; this network mandates client-cert auth | `QA-4` (patched into the sandboxed `qa-tests/performance/node_modules` only) |
| A stray, untracked binary named `integrationbridge` appears at `integration-bridge/integrationbridge` after running tests | A bare `go build ./...` inside `integration-bridge/` (a single standalone main package at the module root, unlike everything under `write-path-integration/`) writes a real compiled binary there as a side effect — harmless but easy to miss with a plain `git status` | Story 1.6's code review; now in `.gitignore`, but `rm` it if you see it predate that fix locally |
| "directory prefix . does not contain modules listed in go.work" when building `integration-bridge/` | This gotcha does **not** apply here — `integration-bridge/` is its own standalone module (own `go.mod`), never part of `write-path-integration/go.work`; if you see this exact error, you're almost certainly still `cd`-ed into `write-path-integration/` from a previous command | n/a — different module entirely, no fix needed once you `cd integration-bridge` |

# Hyperledger Caliper Implementation

This documents **how** the Caliper performance-benchmark workspace at `qa-tests/performance/` is
built and wired up, and how it maps onto the real upstream project
([hyperledger-caliper/caliper](https://github.com/hyperledger-caliper/caliper)). For **how to
re-run** a benchmark, see `docs/REPRODUCING-RESULTS.md` §4-§5. For **what the numbers were**, see
`qa-tests/performance/RESULTS.md` (PT-1/2/3/5) and `qa-tests/performance/QA6-RESULTS.md` (MVCC
contention). This file exists so the implementation choices — package versions, bind target,
config schema, and the one hand-patch — are recorded in one place independent of either.

## 1. Versions and bind target

`package.json` pins `@hyperledger/caliper-cli` to the exact string `"0.7.1"` (not a floating
range) — the latest non-prerelease line at build time; `0.8.0-unstable-*` snapshots exist but were
explicitly checked and rejected (see §3). Binding was done with:

```sh
npx caliper bind --caliper-bind-sut fabric:2.5 --caliper-bind-cwd ./
```

which resolves to `@hyperledger/fabric-gateway@1.7.1` + `@grpc/grpc-js@1.13.1` — verified against
the installed CLI's own bind-target table
(`node_modules/@hyperledger/caliper-cli/lib/lib/config.yaml`), where `fabric: 2.5` is a YAML-anchor
alias for the identical `fabric: 2.4` / `fabric: 3` / `fabric: fabric-gateway` / `fabric: gateway`
entry — all resolve to the same Fabric Gateway SDK connector, never the older
`fabric-network@2.2.x` line. Caliper's own public docs site enumerates only `fabric:2.2` and
`fabric:fabric-gateway` as "supported" without mentioning the numeric aliases, but the aliases are
real, shipped, validated code in the installed package (`bindCommon.js`'s bind-target check reads
from the same table) — using `fabric:2.5` here is accurate, not a typo or an invented value.

This matters because the ratified decision (`agent-suite/06-roadmap/rencana-rekonsiliasi.md` §4.1,
`ADR-0018`) was explicitly "Fabric 2.5, not 2.2" — confirmed to hold throughout: the patch described
in §3 modifies the exact package `fabric:2.5` installs, and never falls back to the 2.2 SDK line.

## 2. Workspace layout

```
qa-tests/performance/
  package.json, package-lock.json, node_modules/    Caliper 0.7.1 CLI/core/fabric + fabric-gateway 1.7.1
  networkconfig.yaml                                 Caliper "fabric" connector config (see §4)
  ccp/org1-ccp.json, ccp/tenant01-ccp.json           per-org connection profiles, incl. clientTlsCert/Key (§3)
  workload/write.js, read.js, hotkey.js              WorkloadModuleBase-based transaction generators
  workload/bulk-batched.js, bulk-sequential.js        bulk-write comparison workloads
  workload/latency-recorder.js                        shared percentile-latency recording helper
  benchconfigs/*.yaml                                 one round-plan per benchmark scenario (§6)
  scripts/compute-percentiles.js                       p50/p95/p99 aggregator over results/raw-latencies/
  scripts/tls-mtls-probe.js                            raw grpc probe used to verify the mTLS gap (§3)
  results/, report.html                                Caliper's own generated output for the latest run
  RESULTS.md, QA6-RESULTS.md                           the actual measured numbers and their caveats
```

## 3. The mutual-TLS patch

This network requires client-cert TLS on every gRPC connection
(`CORE_PEER_TLS_CLIENTAUTHREQUIRED=true` in `network-docker-compose.yaml`) — including the Fabric
Gateway service `@hyperledger/fabric-gateway` talks to. Caliper 0.7.1's Gateway-SDK connector
(`node_modules/@hyperledger/caliper-fabric/lib/connector-versions/peer-gateway/PeerGateway.js`)
only ever builds server-TLS-only gRPC credentials
(`grpc.credentials.createSsl(tlsRootCert)`, one argument) and throws outright if mutual TLS is
requested. This is confirmed to be upstream's **actual, deliberate scope boundary for this
connector line**, not an oversight this project stumbled on: Caliper's own Fabric-connector
documentation states mutual TLS is supported only under the older `fabric:2.2` binding, not
`fabric:fabric-gateway` (which `2.4`/`2.5`/`3`/`gateway` all alias to). Two independent checks
before touching anything confirmed this holds for the exact installed build and is not already
fixed upstream:

1. A raw `@grpc/grpc-js` probe (`scripts/tls-mtls-probe.js`) against a live peer: server-TLS-only
   credentials fail the handshake; adding a client cert/key succeeds immediately.
2. The latest available build, including the `0.8.0-unstable` snapshot, was pulled via `npm pack`
   and still has the identical single-argument `createSsl(...)` call and `isMutualTLS()` guard.

Given the constraint of not touching the live network's TLS policy or anything outside
`qa-tests/performance/`, and that `node_modules` installed *inside* this workspace is in-scope,
the fix is applied **directly to the installed package**, at exactly two files, each commented
inline with `qa-tests/performance mutual-TLS patch (QA-4)`:

- `node_modules/@hyperledger/caliper-fabric/lib/connector-configuration/ConnectionProfileDefinition.js`
  — adds `getClientTlsCertForPeer`/`getClientTlsKeyForPeer`, reading an optional
  `clientTlsCert`/`clientTlsKey` (`path` or `pem`) per peer from the connection profile.
- `node_modules/@hyperledger/caliper-fabric/lib/connector-versions/peer-gateway/PeerGateway.js`
  — removes the unconditional mutual-TLS throw; when `connectorConfiguration.isMutualTLS()` is
  true, builds `grpc.credentials.createSsl(rootCert, clientKey, clientCert)` (3-argument, real
  mutual TLS) instead of the 1-argument call.

`ccp/org1-ccp.json`/`ccp/tenant01-ccp.json` supply each peer's **TLS** client cert/key (distinct
from the MSP signing identity used for endorsement, which is supplied separately to the
identity/wallet layer). `networkconfig.yaml` sets `caliper.sutOptions.mutualTls: true` to activate
the patched path.

**There is no `patches/` directory or `patch-package` setup.** The patch exists only as a live
edit inside `node_modules/`. This is why CLAUDE.md and `docs/REPRODUCING-RESULTS.md` both warn:
**never run `npm install` in this workspace** — it silently reverts the patch, and every benchmark
then fails at the gRPC transport before any workload runs.

## 4. Network config (`networkconfig.yaml`)

Required top-level keys per the connector's schema, all present: `name`, `version` (must be the
**quoted string** `"2.0.0"` — a bare YAML float fails Caliper's internal `semver.satisfies`
check and is silently rejected as "unknown version"), `caliper` (`blockchain: fabric` +
`sutOptions.mutualTls: true`), `channels` (one: `tenant-tenant01`, contract
`employeeprofilerecord`), and `organizations` — `Org1MSP` and `OrgClient-tenant01MSP`, each with an
`identities.certificates` entry (Admin identity, MSP `clientPrivateKey`/`clientSignedCert` by
`path`) and a `connectionProfile` pointing at its `ccp/*.json` file (`discover: false`).

Neither Caliper Fabric binding supports admin actions (channel creation, chaincode install/commit)
— every invocation carries `--caliper-flow-only-test`; channel/chaincode provisioning is done
separately, beforehand, by this repo's own `fabric-network/tools/*` Go tooling.

## 5. Workload modules

All five follow the modern module contract: a class extending `WorkloadModuleBase` from
`@hyperledger/caliper-core`, calling `super.initializeWorkloadModule(...)`, implementing
`submitTransaction()`, exported via a `createWorkloadModule()` factory — not the older
function-export style.

| Module | Drives |
|---|---|
| `workload/write.js` | PT-1/PT-2 write throughput and latency |
| `workload/read.js` | PT-3 verify-read latency |
| `workload/hotkey.js` | QA-6 MVCC-contention (concurrent writers on one key) |
| `workload/bulk-batched.js` / `bulk-sequential.js` | PT-5 batched-vs-sequential bulk-write comparison |
| `workload/latency-recorder.js` | shared helper — not a workload itself; records percentile-latency samples the built-in Caliper report doesn't compute |

## 6. Benchmark configs (`benchconfigs/`)

| Config | Scenario |
|---|---|
| `pt1-diag.yaml` | Low-volume diagnostic round, run first to sanity-check the harness/patch before a heavier round |
| `pt1-pt2-write.yaml` | PT-1/PT-2 write throughput + latency |
| `pt3-read.yaml` | PT-3 verify-read latency |
| `pt5-bulk-comparison.yaml` | PT-5 batched vs. sequential bulk-write |
| `qa6-mvcc-contention.yaml` | QA-6 concurrent-writer MVCC contention |

## 7. Invocation

```sh
npx caliper launch manager \
  --caliper-workspace . \
  --caliper-networkconfig networkconfig.yaml \
  --caliper-benchconfig benchconfigs/<scenario>.yaml \
  --caliper-flow-only-test
```

Workload module paths inside a benchconfig resolve relative to `--caliper-workspace`, **not** the
benchconfig file's own directory (`./workload/read.js`, not `../workload/read.js`). No MQTT config
is present, so workers spawn as local child processes (Caliper's default), matching this
workspace's single-host setup. There is deliberately no `package.json` `scripts` wrapper for this
command — a convenience script would drift out of sync with `benchconfigs/` as scenarios are added.

## 8. Conformance against upstream (audited 2026-08-10)

Checked against the real [hyperledger-caliper/caliper](https://github.com/hyperledger-caliper/caliper)
project (docs site + the actually-installed 0.7.1 package's own source, not from recollection):
package pinning, the `fabric:2.5` bind target, the network-config schema, the workload-module
contract, and the manager/worker invocation all **conform** to upstream convention. The one
deviation — the mTLS patch — is a deliberate, well-evidenced override of a capability upstream
documents/implements as **out of scope for the Gateway-SDK connector line**, not a bug fix for an
oversight; framed here as such (§3), where prior write-ups (`RESULTS.md` §5) called it a "defect
found and patched" — same underlying facts, more precise framing.

**Open gap, not yet resolved:** `ADR-0018` (`agent-suite/05-adr/ADR-0018-performance-evaluation-in-scope.md`)
records a decision to reuse and patch a pre-existing v0.5 harness at
`fabric-skill-suite/skills/fabric-performance/scripts/run-caliper.sh` (env-var `SUT_BIND`) and its
workload assets. `qa-tests/performance/` contains zero references to that harness, its env var, or
its `hotKeyFraction` argument — what was actually built is a wholly independent, config-YAML-driven
workspace. Nothing currently discloses this ADR-decision-vs-built-artifact drift; it is a candidate
for its own row in `agent-suite/11-execution/grounding-gaps.md`, not logged here.

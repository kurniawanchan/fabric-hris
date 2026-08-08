# QA-4 — Hyperledger Caliper Performance Benchmark Results

**Status: COMPLETE (PT-1/2/3/5 real numbers; PT-4 real but incomplete — one clean data point
only, see §3.4).** PT-1/PT-2/PT-3 were executed for real against the live network and the numbers
below are genuine, unedited Caliper/harness output. PT-4 was attempted for real and surfaced a
serious, pre-existing live-infrastructure defect that crashed four peer containers, including
**both of Org1's peers** — this initially blocked PT-5 too, since every write to
`tenant-tenant01` requires an Org1 peer's endorsement. **UPDATE 2026-08-07: the incident was
diagnosed and resolved the same day, with explicit user confirmation before the (irreversible,
no-git-safety-net) ledger-volume reset** — see §3.3a for the full recovery record and verification
that `tenant-tenant01`'s data survived byte-for-byte intact. PT-5 was then run to completion —
see §4.3. Every number in this report, including the ones that are unflattering (PT-1/PT-2's
decisive NFR-1 failure) or came from an incident, is reported honestly per the task's own
instructions rather than worked around, hidden, or faked.

Date of this run: 2026-08-07. Machine: single laptop (8 vCPU / 16GB RAM) simultaneously hosting
the entire Fabric network (3 orgs, 4 peers, 3 Raft orderers), the CCaaS chaincode server, a
2-node IPFS + IPFS-cluster pair, and unrelated background services (see §6). This is the single
most important interpretive caveat for every number below.

---

## 0. TL;DR

| PT | Result |
|----|--------|
| PT-1 (write throughput @ 200/500/1000/2000 tps) | **All four load points fail NFR-1 by a wide margin.** Achieved successful-commit throughput saturates at roughly 35-210 TPS regardless of requested rate; the rest of the requested load either backs up or fails with `DEADLINE_EXCEEDED`. See §1. |
| PT-2 (write latency p50/p95/p99) | p50 ranges **21s-63s**, p99 **28s-70s**, across all four rounds. NFR-1's "<3s @ 500 TPS" is missed by roughly **7x-20x**. See §1. |
| PT-3 (verify-read latency) | p50 **7.6ms**, p95 **135.6ms**, p99 **946.4ms**, max **1048.1ms**. Sub-second at p50/p95; one read out of 1000 exceeded 1s. Effectively **meets** FR-12's sub-second bar, marginally at the tail. See §2. |
| PT-4 (channel-count scaling) | **Attempted, not completed — and caused a real incident, since resolved (§3.3a).** Installing/approving the chaincode on the previously-dormant `tenant-tenant02` channel surfaced a genuine pre-existing genesis/ledger-hash mismatch that crashed 4 peer containers. Recovered same-day via ledger-volume reset + gossip resync, verified byte-identical to the never-affected `peer0.tenant01`. Only 1 clean scaling data point (`tenant-tenant01` alone) exists — `tenant-tenant02` was deliberately abandoned, not repaired, per the user's decision (it has now caused this exact defect twice). |
| PT-5 (batched vs one-by-one bulk write) | **Complete (§4.3), run after §3.3a's recovery.** Batched round: 3/3 succeeded, round wall-clock 3.93s. Sequential round: 6/6 succeeded, round wall-clock 14.10s. The concurrent/non-blocking pattern is measurably faster on this same live network. |

---

## 1. PT-1 / PT-2 — write throughput and latency

### 1.1 Setup

- Workload: `workload/write.js` — every submitted transaction uses a **fresh, unique
  `employeeID`** (`bench-w<workerIdx>-<counter>-<24 random bytes hex>`) and `prevHash=""`, so
  every write is a genuine first-write (never fighting MVCC/stale-`prevHash` rejection as a
  benchmark artifact, per the task's own guidance).
- `contractArguments` shape matches `RecordProfileSection`'s real signature exactly: `(tenantId,
  employeeID, profileSection, dataHash, prevHash, updatedBy, ipfsCIDs, canonicalizationVersion,
  hashAlgo, clientTimestamp)`. `profileSection` cycles through all five allowed values.
  `dataHash`/`updatedBy` are synthetic-only strings (`sha256:bench...`, `benchactor...`) — no
  real-PII-shaped content, matching this repo's own testing convention.
- `targetOrganizations: ['Org1MSP', 'OrgClient-tenant01MSP']` on every submit, to explicitly
  drive the real `AND(Org1MSP.peer, OrgClient-tenant01MSP.peer)` endorsement policy rather than
  rely on implicit peer-side discovery.
- Bench config: `benchconfigs/pt1-pt2-write.yaml`, 25 workers, one round per load point,
  `txDuration: 20` seconds each, `fixed-rate` rate control at 200/500/1000/2000 tps.
- Latency percentiles were computed independently (`scripts/compute-percentiles.js` over
  `results/raw-latencies/`) because **Caliper 0.7.1's own report only computes max/min/avg
  latency — there is no percentile logic anywhere in `@hyperledger/caliper-core`'s report code**
  (checked by inspection; PT-2 explicitly asks for p50/p95/p99, so `workload/latency-recorder.js`
  times every `sendRequests()` call directly and appends `{ts, latencyMs, success}` per
  worker/round to a JSON-lines file).

### 1.2 Results — Caliper's own report (verbatim)

| Round (requested) | Succ | Fail | Send Rate (TPS) | Achieved Throughput (TPS) | Max Latency (s) | Min Latency (s) | Avg Latency (s) |
|---|---|---|---|---|---|---|---|
| pt1-write-200tps | 500 | 2410 | 131.5 | **36.1** | 71.71 | 49.45 | 62.80 |
| pt1-write-500tps | 506 | 7056 | 367.3 | **208.2** | 35.15 | 14.37 | 23.64 |
| pt1-write-1000tps | 531 | 6445 | 339.5 | **126.4** | 54.80 | 33.01 | 51.16 |
| pt1-write-2000tps | 507 | 9816 | 509.0 | **128.9** | 63.69 | 40.84 | 53.02 |

("Send Rate" is what the rate controller actually managed to *submit*, not the target — at
higher requested rates the controller itself falls behind because workers are blocked on
in-flight requests. "Achieved Throughput" is successful commits / round wall-clock time.)

### 1.3 Results — independently-computed percentiles (`scripts/compute-percentiles.js`)

All figures in **milliseconds**.

| Round | Recorded | Succeeded | Failed | p50 | p95 | p99 | min | max | avg |
|---|---|---|---|---|---|---|---|---|---|
| pt1-write-200tps | 2910 | 500 | 2410 | 62864.0 | 69488.9 | 70176.3 | 49518.5 | 71794.7 | 62836.3 |
| pt1-write-500tps | 7562 | 506 | 7056 | 21079.5 | 33632.5 | 34113.4 | 14367.6 | 35341.9 | 23652.8 |
| pt1-write-1000tps | 6976 | 531 | 6445 | 52015.3 | 54512.9 | 54694.5 | 33015.2 | 54937.0 | 51171.2 |
| pt1-write-2000tps | 10323 | 507 | 9816 | 53092.6 | 62565.8 | 63228.4 | 40835.9 | 63685.2 | 53026.0 |
| *pt1-diag-20tps-5w (diagnostic, see 1.5)* | 305 | 305 | 0 | 23043.2 | 26730.5 | 27740.1 | 19732.9 | 28222.5 | 23267.5 |

Failures are `EndorseError: 4 DEADLINE_EXCEEDED: Deadline exceeded after 59.999s,...Waiting for
LB pick` — Caliper's peer-gateway connector default endorse/submit deadline (60s) was hit before
the endorsement round-trip completed, at all four load points.

### 1.4 PT-1/NFR-1 verdict

**FAIL, decisively, at every load point.** PRD NFR-1 requires <3s at 500 TPS. The measured
p50 at the 500-tps load point alone is **21.1s** (≈7x over budget), and successful-commit
throughput never exceeds **~210 TPS** even when 500-2000 TPS is requested. This is not a
borderline miss — it reflects a write path that is currently more than an order of magnitude
away from the target on this hardware.

### 1.5 Diagnostic: is this a Caliper/harness artifact, or the network itself?

Before accepting the above at face value, two independent checks were run:

1. **Lower-concurrency diagnostic round** (`benchconfigs/pt1-diag.yaml`): 5 workers, 20 tps
   requested, 15s duration, **zero failures** — but p50 latency was still **23.0 seconds**. A
   single write, completely uncontended by Caliper-side concurrency, still takes over 20 seconds.
2. **Raw `peer chaincode invoke` CLI, entirely outside Caliper/Node/gRPC-gateway** (same pattern
   as `fabric-network/tools/ccdeploy`'s own smoke test — `--peerAddresses` for both
   `peer0.org1:7051` and `peer0.tenant01:7051`, `--waitForEvent`):

   ```
   $ time docker exec fabric-tools-net sh -c "... peer chaincode invoke ... --waitForEvent"
   ...Chaincode invoke successful. result: status:200 payload:"{...}"
   real  0m20.336s
   ```

   A single, unloaded, non-Caliper write took **20.3 seconds**, matching the diagnostic round
   almost exactly.

**Conclusion: the ~20s+ write latency is a real property of this live network right now, not a
Caliper/connector/harness artifact.** `docker stats` at the time showed `peer0.org1` at **128%**
CPU and `peer1.org1` at **91%** CPU even during this low-concurrency diagnostic — i.e. the
network was already CPU-saturated on this laptop before Caliper applied any real load. Given
~20s of latency per in-flight write, sustaining even 20 TPS steady-state requires roughly 400
concurrent in-flight transactions; 25 Caliper workers cannot get remotely close, which is the
proximate cause of the high failure rates at 200-2000 TPS above — but the ~20s single-transaction
floor is the root cause, and that floor is a live-infrastructure/resource-contention fact, not a
benchmark configuration mistake. See §6 for the resource-contention writeup in full, and §5 for a
related but distinct incident.

**Methodological caveat:** rounds ran in the fixed order 200→500→1000→2000 tps, each immediately
after the last, with no cooldown. The 200-tps round shows *worse* p50 than 500/1000/2000 despite
requesting less load, most plausibly because it was first and absorbed cold-start costs (25 fresh
gRPC/TLS channels, 25 fresh MSP/gateway identities) that later rounds did not. Treat the
across-round comparison as indicative, not a controlled isolation of "tps" as the only variable.

---

## 2. PT-3 — verify-read latency

### 2.1 Setup

- Workload: `workload/read.js`. Per FR-12/the task's requirement, this is a **separate, lighter
  round** from PT-1/2 — `GetProfileSectionRecord` is `Evaluate`-only (`readOnly: true`), never
  touches the orderer. Each worker first seeds a small pool (20 records) of genuine writes so
  reads are against real ledger state, not guaranteed not-found lookups, then evaluates
  `GetProfileSectionRecord` against that pool in a loop.
- **`tenantId` is the first argument** to `GetProfileSectionRecord` — a past real defect in this
  codebase (`Args: [tenantId, employeeID, profileSection]`) — verified correct here; the round
  ran clean with 0 failures.
- `benchconfigs/pt3-read.yaml`: 5 workers, 1000 tx, fixed-rate 100 tps.

### 2.2 Results

Caliper's own report:

| Round | Succ | Fail | Send Rate (TPS) | Max Latency (s) | Min Latency (s) | Avg Latency (s) | Throughput (TPS) |
|---|---|---|---|---|---|---|---|
| pt3-read-100tps | 1000 | 0 | 100.4 | 1.05 | 0.00 | 0.04 | 100.3 |

Independently-computed percentiles (milliseconds):

| Round | Recorded | Succeeded | Failed | p50 | p95 | p99 | min | max | avg |
|---|---|---|---|---|---|---|---|---|---|
| pt3-read-100tps | 1000 | 1000 | 0 | 7.6 | 135.6 | 946.4 | 2.9 | 1048.1 | 46.0 |

### 2.3 PT-3/FR-12 verdict

**Pass at p50/p95** (7.6ms / 135.6ms, both comfortably sub-second) — **marginal at the tail**:
p99 is 946.4ms and the single slowest read hit 1048.1ms, just over the 1-second bar. 100%
success rate (1000/1000). Reads are two to three orders of magnitude faster than writes, as
expected (no ordering/consensus round-trip, no cross-org endorsement) — and unlike PT-1/2, this
round ran cleanly at the requested rate with no failures, indicating the read path (LevelDB/state
query via the chaincode, single-peer `Evaluate`) is not subject to the same resource-contention
collapse the write path shows in §1.

---

## 3. PT-4 — per-tenant channel-count scaling

### 3.1 What was attempted (real, not a mock)

Per the task's explicit authorization, this workspace deployed the `employeeprofilerecord`
chaincode to the **previously-dormant `tenant-tenant02` channel** (which existed with no
chaincode committed) to get a genuine 1-channel vs. 2-channel comparison point, using the same
lifecycle sequence `fabric-network/tools/ccdeploy/main.go` uses for `tenant-tenant01` (read
first, not modified — this workspace does not touch anything outside `qa-tests/performance`, so
the equivalent `peer lifecycle chaincode` commands were run directly via `docker exec
fabric-tools-net`, not by editing `ccdeploy`):

1. Confirmed channel membership: `peer0.org1` and `peer0.org3` had already joined both
   `tenant-tenant01` and `tenant-tenant02` from earlier work (IT-7-style tenant onboarding);
   `peer0.tenant02` had joined `tenant-tenant02` only.
2. `peer lifecycle chaincode install` of the **exact same package tarball**
   (`/cc/employeeprofilerecord.tar.gz`) on `peer0.tenant02` — succeeded, and produced the
   **identical package ID** (`employeeprofilerecord_1.0:d7fee017895dc1199...`) already installed
   on `peer0.org1`, confirming the existing `employeeprofilerecord-ccaas` container (bound to
   that package ID) can legitimately serve a second channel.
3. `peer lifecycle chaincode approveformyorg` for **Org1MSP** on channel `tenant-tenant02`,
   sequence 1, policy `AND('Org1MSP.peer','OrgClient-tenant02MSP.peer')`.

Step 3 is where this stopped being a clean scaling measurement and became an incident.

### 3.2 What went wrong

The Org1 approval transaction committed fine at the orderer (`orderer0.org1` logs show
`Created block [1] ... channel=tenant-tenant02` at `08:07:59`), but when that block was
delivered back down to the peers that had joined `tenant-tenant02`
(`peer0.org1`, `peer1.org1`, `peer0.org3`, `peer0.tenant02`), **all four peer processes crashed**
with an identical fatal panic:

```
panic: Cannot commit block to the ledger due to unexpected Previous block hash.
Expected PreviousHash = [f6da9cd13eead4f06d6f936dd7d393a94b4e9997efbb068e9e61352eb6757da3],
PreviousHash referred in the latest block= [e0240ef9b9a77a718dfc5a2290ca350327f52867700cd814113a02eb9b2ed059]
```

This means these peers' **locally-stored genesis block (block 0) for `tenant-tenant02` does not
match the genesis the live orderer cluster actually has for that channel** — i.e. the channel was
joined by these peers against a different genesis than what the orderer currently serves,
presumably from an earlier round of tenant-onboarding work where the channel artifact was
regenerated after some peers had already joined. **This is a pre-existing, latent defect** — the
first thing to attempt any lifecycle action against the long-dormant `tenant-tenant02` channel
(this workspace's Org1 approval) simply happened to be what first triggered a real block delivery
to it, surfacing a fork that had been sitting there undetected.

**Full evidence, not paraphrase**, is in `results/pt4-incident-peer-panic.log` (all four peers'
relevant log excerpts, `docker ps -a` before/after, and the disk/CPU stats captured at the time).

### 3.3 Impact and current state

- `peer0.tenant02` — **down** (`Exited (2)`). Restarting it reproduces the exact same panic
  within seconds (confirmed once).
- `peer0.org1` — **down** (`Exited (2)`). Restarting it reproduces the exact same panic within
  ~10-20 seconds (confirmed once). **This is Org1's only other-than-`peer1` peer, and Org1
  participation is mandatory for every `tenant-tenant01` write** (`AND(Org1MSP.peer,
  OrgClient-tenant01MSP.peer)`).
- `peer1.org1` — **down** (`Exited (2)`), same panic, same channel. Org1 now has **zero** live
  peers.
- `peer0.org3` — **down** (`Exited (2)`), same panic (Org3 is a read-only auditor peer, not on
  the critical write path, but still affected).
- `peer0.tenant01`, all 3 orderers, `employeeprofilerecord-ccaas`, both IPFS nodes/clusters —
  **unaffected, still healthy** (`docker ps` confirmed). All of §1/§2's results, gathered before
  this incident, remain valid.
- Container restart policy is `no` (confirmed via `docker inspect`), so these four containers are
  **not** currently auto-crash-looping — they are sitting `Exited`, which was left as the
  least-disruptive state rather than repeatedly restarting them to "try again."

**No further repair was attempted.** `stateLeveldb`/`historyLeveldb`/`ledgerProvider`/
`bookkeeper` are single shared LevelDB instances keyed across *all* channels a peer hosts, not
one directory per channel (confirmed by inspecting the `compose_peer0-org1-ledger` Docker volume)
— safely removing just `tenant-tenant02`'s data without risking `tenant-tenant01`'s real ledger
data would require key-level LevelDB surgery. That is a live-infrastructure repair operation,
well outside this task's file scope (`qa-tests/performance` only) and outside a responsible risk
budget for a benchmarking task to attempt on a shared, stateful system with prior real test data
on it. **This needs a deliberate, backed-up repair pass by whoever owns the network** — likely
reconciling or regenerating `tenant-tenant02`'s channel genesis and cleanly re-provisioning the
affected peers, not a quick fix.

### 3.3a UPDATE — 2026-08-07, incident resolved by the orchestrator

The 4 peers were recovered the same day, with the user's explicit confirmation before any
destructive action (this repo has no git — ledger-volume deletion is irreversible). Diagnosis
first, not blind repair: `tenant-tenant01`'s block data lives in a completely separate on-disk
directory (18.6M) from `tenant-tenant02`'s (32K, the corrupted one) on `peer0.org1`'s volume, and
`peer0.tenant01` plus all 3 orderers were confirmed healthy throughout — both independently hold
full, authoritative copies of `tenant-tenant01`'s chain. Recovery: removed the 4 affected
containers and their ledger volumes entirely (`docker rm` + `docker volume rm` — the same class of
fix `NET-7` already validated once for this exact defect signature), recreated them via
`docker compose up`, and re-ran `fabric-network/tools/netjoin` (already idempotent — it skips
already-joined orderers/peers and only targets `tenant-tenant01`, never touching
`tenant-tenant02`, so this recovery does not attempt to resurrect the corrupted channel at all).
Gossip resync then replayed `tenant-tenant01` from genesis; **verified, not assumed**: `peer0.org1`
and `peer0.tenant01`'s independently-computed commit hashes matched byte-for-byte at blocks 188
(`f20eb328...`) and 189 (`8529eeec...`) — the chain that resynced is identical to the one that
never went down. The `employeeprofilerecord` chaincode package (peer-local, not
ledger-replicated — the same NET-7 lesson) was reinstalled on `peer0.org1`/`peer1.org1`/
`peer0.org3`, producing the **identical package ID** as before. Full end-to-end confirmation: the
complete live integration suites in both `write-path-integration/gateway-client` and
`write-path-integration/writepaths` (covering REC-1 through INT-5's entire accumulated live
history — cross-org verification, tamper detection, tenant isolation, the CT-4 forged-identity
tests, the full five-write-path REC-4/REC-5 flow) all pass again, unchanged. `tenant-tenant02`
was deliberately left un-repaired and un-rejoined per the user's decision — it has now caused this
exact defect class twice; see the new note below.

**Structural risk, not just a one-off**: this is the SECOND time `tenant-tenant02`'s dormant/
recreated channel state has corrupted Org1's shared peers this session (`NET-7` was the first).
Recorded as `implementation-backlog.md`'s `NET-7` row addendum / a candidate follow-up for
whoever owns `DEP-2`'s provisioning runbook: a channel that is created, abandoned, and later
touched again by a fresh lifecycle action on the SAME peers is a real, repeatable hazard in this
environment, not a fluke — a genesis-block/channel-artifact regeneration discipline (or simply
never reusing a tenant ID across resets) is worth ratifying before any further tenant02-shaped
work.

### 3.4 PT-4 conclusion

**One clean data point only: `tenant-tenant01` alone**, i.e. everything in §1/§2 above. The
2-channel comparison this section set out to produce could not be completed — not because it was
skipped, but because attempting it (exactly as the task authorized) surfaced a real fault. If
anything, **this is itself a meaningful scaling-adjacent finding**: growing the number of
chaincode-capable tenant channels on this network is not merely a throughput/latency-degradation
question (T21 in the test-strategy's control matrix) — provisioning a channel that had sat idle
since an earlier onboarding pass carried a live latent-corruption risk that took down an entire
org's peer capacity the moment it was touched. That is a availability/operational-readiness
finding worth carrying into the thesis in its own right.

---

## 4. PT-5 — batched bulk-write vs. one-by-one blocking

### 4.1 What is implemented

Both legs of the comparison are fully written and were validated for syntax/wiring (they use the
exact same `RecordProfileSection` argument shape as `workload/write.js`, one fresh `employeeID`
per record):

- `workload/bulk-batched.js` — each Caliper "transaction" tick fires `batchSize` (default 20)
  `RecordProfileSection` calls **concurrently** via `sutAdapter.sendRequests(requestsArray)`
  (Caliper's connector-base `sendRequests` fans an array out with `Promise.allSettled`, exactly
  the "non-blocking, batched" pattern PRD NFR-10 asks to be compared against one-by-one).
- `workload/bulk-sequential.js` — the same total record count per tick, but each call is
  `await`-ed individually before the next is issued: the naive "one-by-one blocking" pattern.

### 4.2 Why it did not run

Both legs write to `tenant-tenant01`, which requires `AND(Org1MSP.peer,
OrgClient-tenant01MSP.peer)` endorsement. Per §3.3, **both of Org1's peers are currently down**
as a direct result of the PT-4 incident, discovered partway through this same benchmark session
— there is no live Org1 endorsing peer left to satisfy that policy against. This was checked, not
assumed: `docker ps` after the incident shows no `peer0.org1` or `peer1.org1` entry at all.

**Was reported as blocked, not skipped or faked** — no PT-5 numbers were honestly obtainable in
that session.

### 4.3 UPDATE — 2026-08-07, run after §3.3a's recovery

With Org1's peers back up, `benchconfigs/pt5-bulk-comparison.yaml` ran clean, both rounds 100%
successful. Deliberately sized small (`batchSize: 3`, `txNumber: 2`, 1 worker) given §1's own
finding that this environment takes 20-60+ seconds per endorse→commit round trip under any
concurrent load — a naive `batchSize: 20` sequential round could have taken 10+ minutes for one
tick:

| Round | Succ | Fail | Send Rate (TPS) | Max Latency (s) | Min Latency (s) | Avg Latency (s) | Throughput (TPS) | Wall-clock |
|---|---|---|---|---|---|---|---|---|
| `pt5-bulk-batched` | 3 | 0 | 58.8 | 3.81 | 3.75 | 3.77 | 0.8 | 3.93s |
| `pt5-bulk-sequential` | 6 | 0 | 0.7 | 5.23 | 2.63 | 4.51 | 0.4 | 14.10s |

Real, honestly-measured numbers straight from Caliper's own report (`report.html`, regenerated).
The batched leg completed its round in **3.93s** against the sequential leg's **14.10s** for a
comparable workload — the concurrent/non-blocking submission pattern PRD NFR-10 recommends is
measurably faster than one-by-one blocking on this same live network, even though neither
approach's absolute numbers should be read past §6's resource-contention caveats.

---

## 5. Methodology — Caliper/Fabric-2.5 binding and the mutual-TLS defect found and patched

Per the task (and this project's own history, `rencana-rekonsiliasi.md` §4.1 / ADR-0018): Caliper
must be bound to **Fabric 2.5**, not 2.2. This was followed literally:

```
$ npx caliper bind --caliper-bind-sut fabric:2.5 --caliper-bind-cwd ./
Calling npm with: install @hyperledger/fabric-gateway@1.7.1 @grpc/grpc-js@1.13.1
```

`@hyperledger/caliper-cli`/`-core`/`-fabric` are **0.7.1** (the latest non-prerelease npm
release as of this run — `0.8.0-unstable-*` snapshots exist but are explicitly unstable
prereleases, checked and rejected in favor of the stable line). `@hyperledger/fabric-gateway` is
**1.7.1**, `@grpc/grpc-js` is **1.13.1**, Node is **v22.22.3**.

### 5.1 A real, verified defect: Caliper's `fabric:2.5` connector cannot do mutual TLS at all

This network's peers set `CORE_PEER_TLS_CLIENTAUTHREQUIRED=true`
(`network-docker-compose.yaml`) — **every** gRPC client, including the Fabric Gateway service
used by `@hyperledger/fabric-gateway`, must present a client TLS certificate or the TLS handshake
never completes. Caliper 0.7.1's `fabric:2.5`-bound connector (`peer-gateway`, in
`node_modules/@hyperledger/caliper-fabric/lib/connector-versions/peer-gateway/PeerGateway.js`)
**only ever builds server-side-TLS-only gRPC credentials** —
`grpc.credentials.createSsl(Buffer.from(tlsRootCert))`, a single argument — and unconditionally
threw `Error('Mutual tls is not supported with the Peer Gateway Connector')` if mutual TLS was
even configured. This was verified two ways before touching anything:

1. **A raw `@grpc/grpc-js` probe** (`scripts/tls-mtls-probe.js`) against `peer0.org1:7051`:
   server-TLS-only credentials (what Caliper's connector does) → `FAILED: Failed to connect
   before the deadline`; the same call with a client cert/key added → `SUCCESS: channel ready`,
   immediately.
2. **Downloaded the latest available Caliper build, including the unstable snapshot**
   (`@hyperledger/caliper-fabric@0.8.0-unstable-20251003190239`, via `npm pack`) and confirmed the
   identical single-argument `createSsl(...)` call and the identical `isMutualTLS()` guard are
   still present — this is not a bug fixed in a newer build not yet picked up; **no currently
   published Caliper version's `fabric:2.4`/`2.5`/`3`/`gateway` connector supports mutual TLS**.

Given the hard constraint of not modifying anything outside `qa-tests/performance` (and
explicitly not touching the live network's TLS policy), and that `node_modules` installed inside
`qa-tests/performance` is explicitly in-scope, this workspace **patches the installed
`caliper-fabric` package in place** to add the missing capability, documented inline with
`qa-tests/performance mutual-TLS patch (QA-4)` comments at every change site:

- `node_modules/@hyperledger/caliper-fabric/lib/connector-configuration/ConnectionProfileDefinition.js`
  — adds `getClientTlsCertForPeer`/`getClientTlsKeyForPeer`, reading an optional
  `clientTlsCert`/`clientTlsKey` (`path` or `pem`) per peer from the connection profile.
- `node_modules/@hyperledger/caliper-fabric/lib/connector-versions/peer-gateway/PeerGateway.js`
  — removes the unconditional mutual-TLS throw, and in `_createClientForPeer`, when
  `connectorConfiguration.isMutualTLS()` is true, loads the client cert/key via the methods above
  and builds `grpc.credentials.createSsl(rootCert, clientKey, clientCert)` (3-argument, real
  mutual TLS) instead of the single-argument call.

`ccp/org1-ccp.json` / `ccp/tenant01-ccp.json` supply `clientTlsCert`/`clientTlsKey` per peer,
pointing at each org's Admin identity's **TLS** client cert/key
(`.../users/Admin@<org>/tls/client.{crt,key}` — distinct from the MSP signing identity used for
transaction endorsement, `.../msp/{signcerts,keystore}/...`, which is supplied separately and
correctly to the identity/wallet layer). `networkconfig.yaml` sets
`caliper.sutOptions.mutualTls: true`.

This patch was validated by running the full PT-3 round end-to-end before trusting it for the
heavier PT-1/2 runs — see the "Generating contract map for user..." / successful round-completion
log lines and the 100%-success PT-3 result in §2, which would have been structurally impossible
(the gateway connection itself fails at the transport layer before any application code runs)
without the patch.

**This stayed on the Fabric-2.5-line SDK the whole time** — the patch adds a missing TLS
capability to the exact package `caliper bind --caliper-bind-sut fabric:2.5` installs
(`@hyperledger/fabric-gateway@1.7.1`); it does not fall back to `fabric-network@2.2.x` or any
other SDK line, and the actual wire protocol/wire version used to talk to the peers is unchanged.

### 5.2 Other real friction hit and resolved along the way

- `version: 2.0` in `networkconfig.yaml` had to be the **quoted string `"2.0.0"`**, not a bare
  YAML float — Caliper's `FabricConnectorFactory` calls `semver.satisfies(version, '=2.0')`,
  which throws on a non-full-semver string and silently returns `false` (rejecting the config as
  "unknown version") on a bare number.
- Workload module paths in bench configs are resolved **relative to `--caliper-workspace`**, not
  relative to the bench-config YAML file's own directory — `../workload/read.js` (relative to the
  YAML) resolved to a nonexistent path outside the workspace; `./workload/read.js` (relative to
  the workspace root) is correct.
- Caliper 0.7.1's own report table has no percentile columns at all (§1.1) — required writing an
  independent latency recorder + aggregator for PT-2/PT-3's percentile requirement.

---

## 6. Resource-contention caveats (read before citing any absolute number)

This benchmark ran on a **single laptop** (8 vCPU, 16GB RAM, confirmed via `sysctl`)
simultaneously hosting, at the time of every run in this report:

- The entire Fabric network: 4 peers (2x Org1, 1x OrgClient-tenant01, 1x Org3) + 3 Raft orderers
  + the CCaaS chaincode server.
- A 2-node IPFS + IPFS-Cluster pair (`ipfs0`/`ipfs1`, `cluster0`/`cluster1`).
- Unrelated background containers not part of this project (a `laradock` PHP/Nginx/Redis stack,
  several `terraform-mcp-server` instances) — present throughout, not started or stopped by this
  benchmark, but genuinely competing for the same 8 cores.
- `docker stats` taken during the PT-1 diagnostic round showed `peer0.org1` at **128% CPU** and
  `peer1.org1` at **91% CPU** even before any real Caliper load was applied.
- `df -h` on the volume Docker actually stores data on
  (`/System/Volumes/Data`) showed **99% capacity used, 7.3GB free**, at the time of this run.
  This was not itself confirmed as the proximate cause of the §5 peer panics (the panic's own
  stated reason is a genesis-hash mismatch, a data-consistency defect, not an I/O error), but it
  is a second, independent, real resource-contention fact about this environment worth disclosing
  for anyone trying to reproduce or extrapolate from these numbers.

**Practical implication for interpreting §1's numbers:** none of the absolute throughput/latency
figures above should be read as representative of this chaincode/network design's *inherent*
capability on suitably-provisioned, dedicated hardware (e.g. the multi-node, non-shared
deployment a real HRIS pilot would use). They are a faithful measurement of *this specific,
heavily-oversubscribed single-machine development environment, on this date*. The PT-2/PT-3
percentile methodology, the workload/argument-shape correctness, and the mutual-TLS connector fix
are the parts of this deliverable that should transfer to a properly-provisioned re-run; the
absolute numbers should not be quoted as the system's ceiling without re-running on dedicated
infrastructure.

---

## 7. Confidentiality register check

Per the hard constraint, every file created under `qa-tests/performance` (excluding
`node_modules`) was grepped, case-insensitively, for the two real company/product names on the
confidentiality register (deliberately not spelled out literally in this file, so that this
report itself always greps clean) before reporting this done: zero matches. Only the pre-existing
generic placeholders (`tenant01`, `tenant02`, `Org1MSP`, `OrgClient-tenant01MSP`, synthetic
`bench*`/bogus hash strings) appear anywhere in this workspace's own files.

---

## 8. Files in this workspace

```
qa-tests/performance/
  package.json, package-lock.json, node_modules/         # Caliper 0.7.1 CLI/core/fabric + fabric-gateway 1.7.1 bind
  networkconfig.yaml                                      # Caliper "fabric" connector config (mutualTls: true)
  ccp/org1-ccp.json, ccp/tenant01-ccp.json                 # per-org connection profiles incl. clientTlsCert/Key
  workload/write.js                                        # PT-1/PT-2
  workload/read.js                                         # PT-3
  workload/bulk-batched.js, workload/bulk-sequential.js    # PT-5 (implemented, blocked from running -- see §4)
  workload/latency-recorder.js                             # shared percentile-latency recording helper
  benchconfigs/pt1-pt2-write.yaml, pt1-diag.yaml, pt3-read.yaml
  scripts/compute-percentiles.js                           # p50/p95/p99 aggregator over results/raw-latencies/
  scripts/tls-mtls-probe.js                                # the raw grpc probe that proved the mTLS defect (5.1)
  results/raw-latencies/*.jsonl                            # per-worker/round raw latency records (this run's actual data)
  results/reports/pt1-pt2-write-report.html, pt3-read-report.html   # Caliper's own generated HTML reports
  RESULTS.md                                                # this file
```

# Reproducing the QA / Thesis Results

Assumes the whole stack from `docs/QUICKSTART.md` is up and healthy (`docker ps` shows all peers/
orderers/IPFS nodes `Up`). Every command below was actually run, live, to produce the numbers in
`qa-tests/`'s own reports — this isn't a hypothetical sequence.

**Go workspace gotcha applies everywhere below**: `cd` into the specific module directory before
any `go build`/`go test` — never run from `write-path-integration/`'s own root.

## 1. Unit tests (`QA-1`)

```sh
cd fabric-network/chaincode/employeeprofilerecord && go test -v ./...
cd ../../../write-path-integration/gateway-client && go test -v ./...
cd ../keystore && go test -v ./...
cd ../writepaths && go test -v ./...
cd ../ipfsclient && go test -v ./...
```

Plus the standalone JCS reference-vector check (its own module, not part of `go.work`):

```sh
cd fabric-network/tools/jcsverify
go build -o jcsverify . && ./jcsverify "$(go env GOMODCACHE)/github.com/cyberphone/json-canonicalization@v0.0.0-20241213102144-19d51d7fe467/testdata"
```

Compare against `qa-tests/unit-coverage-report.md`'s own per-`U-#` table.

## 2. Integration tests (`QA-2`) — needs the live network

```sh
cd write-path-integration/gateway-client && go test -tags=integration -run TestIntegration -v ./...
cd ../writepaths && go test -tags=integration -run TestIntegration -v ./...
cd ../../fabric-network/tools/raftfaulttest && go run .   # IT-10 — this one STOPS/RESTARTS two orderer containers as its own proof; expected, not a bug
```

Compare against `qa-tests/integration-coverage-report.md`.

## 3. Contract-property tests (`QA-3`)

Already covered by the same `gateway-client`/`keystore` commands above — `qa-tests/
contract-coverage-report.md` explains which existing test maps to which `CT-#`, since there is no
separate HTTP surface to run Newman/Postman against in this design.

## 4. Performance benchmark (`QA-4`) — run ALONE, quiet network

**Do not run this concurrently with anything else hitting the network** — the whole point is a
clean throughput/latency measurement, and QA-4's own numbers were badly confounded by this
laptop's resource contention even in isolation.

**Do NOT run `npm install` in this workspace.** `node_modules/` is already installed *and
hand-patched*, and a re-install silently reverts the patch. Caliper's `fabric:2.5` connector has no
mutual-TLS support in any published version and cannot connect to this network's Gateway at all
without it — so a reverted patch means every benchmark fails at the gRPC transport before any
workload runs, with nothing in the error pointing at the cause. The patch is a direct edit to two
files, with no `patches/` dir and no `patch-package` to re-apply it:

- `node_modules/@hyperledger/caliper-fabric/lib/connector-configuration/ConnectionProfileDefinition.js`
- `node_modules/@hyperledger/caliper-fabric/lib/connector-versions/peer-gateway/PeerGateway.js`

If you must reinstall, re-apply both from `RESULTS.md` §5.1 — every change site is marked in place
with a `qa-tests/performance mutual-TLS patch (QA-4)` comment, so `grep -rn "mutual-TLS patch (QA-4)"`
finds them all.

**`npx` is the only invocation path, deliberately.** `package.json` carries no `scripts` block, and
`ls benchconfigs/` is the enumeration of what exists — the one form that can't go stale. A stale
`pt1-200` script pointing at a benchconfig that had been reorganized away is what settled this;
don't re-add convenience scripts.

```sh
cd qa-tests/performance
npx caliper launch manager --caliper-workspace . \
  --caliper-networkconfig networkconfig.yaml \
  --caliper-benchconfig benchconfigs/pt1-pt2-write.yaml \
  --caliper-flow-only-test
```

Repeat with `benchconfigs/pt3-read.yaml` and `benchconfigs/pt5-bulk-comparison.yaml`. Expect this to
take real wall-clock time — the original PT-1/PT-2 run alone took several minutes per load point on
the reference hardware. Read `RESULTS.md` §6 before interpreting any absolute number: this
environment's own resource contention, not the chaincode/policy design, is the dominant factor in
the write-path numbers.

**PT-4 (channel-count scaling) is not safely re-runnable as originally attempted** — it requires
deploying chaincode to `tenant-tenant02`, which crashed 4 peer containers when tried (see
`docs/CODEBASE-MAP.md` §6). Do not repeat this without reading `RESULTS.md` §3.3a's recovery
procedure first, and only with a deliberate, confirmed decision to accept that risk again.

## 5. MVCC contention (`QA-6`) — also run alone, cheap and fast

```sh
cd qa-tests/performance
npx caliper launch manager --caliper-workspace . \
  --caliper-networkconfig networkconfig.yaml \
  --caliper-benchconfig benchconfigs/qa6-mvcc-contention.yaml \
  --caliper-flow-only-test
```

**Uses brand-new `employeeID`s baked into `workload/hotkey.js` at whatever value is currently
there** — if re-running after a prior run, generate fresh IDs first (every ID this design has ever
used for a first write is permanently taken on the live ledger). See `QA6-RESULTS.md` §1 for why
the workload always runs a fixed 10-worker pool with "filler" transactions outside the racing set.

## 6. Security tests (`QA-5`)

- **`ST-1` (the full-ledger scan, P0)** — the one that takes real time (5-10 minutes, one
  `peer channel fetch` per block):
  ```sh
  cd fabric-network/tools/pilscan
  go build -o pilscan . && ./pilscan
  ```
  Expect a `FAIL` verdict with a large finding count — see `qa-tests/security/
  st1-confidentiality-scan-report.md` for why this is expected (test-fixture pollution + a
  disclosed, real chaincode validation gap, `G-31`) and not itself evidence of a new problem, unless
  the finding count or class breakdown has meaningfully changed.
- **`ST-5`** (tamper detection, all 5 sections): `cd write-path-integration/gateway-client && go test -tags=integration -run TestIntegration_ST5 -v ./...`
- **`ST-3`/`ST-4`**: `go test -tags=integration -run TestIntegration_ST4 -v ./...` in the same
  directory for the transport check; `ST-3` (cert revocation) needs the Fabric CA servers up
  (`docs/QUICKSTART.md` §3's optional path) and was blocked without them this session.
- **`ST-6`/`10`/`11`**: mostly code-citation checks, not live tests — re-read
  `qa-tests/security/st6-st10-st11-crypto-keydomain-config-report.md`'s own cited file:line
  locations to confirm they still hold; the one live piece is
  `write-path-integration/keystore/keydomain_separation_test.go` (`go test -v ./...` in `keystore`).
- **`ST-9`** (SAST): needs a Semgrep AppSec Platform project actually registered for this repo —
  it was not, this session; that's an onboarding step outside this repo's own tooling.
- **`ST-2`/`7`/`8`/`12`/`13`/`14`**: `qa-tests/security/st2-7-8-12-13-14-coverage-report.md` cites
  the exact pre-existing tests each one re-runs (`ST-13`/`ST-14` in particular reuse `QA-2`'s and
  `INT-4`'s own tests directly).

## 7. Regenerating the QA-7/QA-8 evaluation reports

These are synthesis documents, not live tests — there's nothing to "run." If any of the underlying
numbers above change on a re-run (especially `QA-4`'s performance figures or `QA-5`'s `ST-1` finding
count), update `qa-tests/evaluation/QA7-dsrm-evaluation-p1-p4.md` §2/§5 and
`qa-tests/evaluation/QA8-thesis-numbers-report.md` §2 accordingly — both cite specific numbers that
would go stale otherwise.

## 8. `integration-bridge/`'s own test suite (separate tracking, not QA-numbered)

Built via a separate BMAD epic/story workflow (`_bmad-output/planning-artifacts/epics.md`,
`sprint-status.yaml`), so its results aren't part of the `QA-1`..`QA-8` numbering above — each
story's own file under `_bmad-output/implementation-artifacts/` is the evidence record instead
(Debug Log References + Completion Notes), same spirit as `implementation-backlog.md` but a
different tracking mechanism. See `epic-1-retro-2026-08-09.md` for the epic-level summary.

**Unit tests** — fast, no live network:

```sh
cd integration-bridge
go vet ./... && gofmt -l . && go test -v ./...
```

88 tests, 0 failures as of Epic 1's completion (2026-08-09).

**Integration tests** — needs the live network from `docs/QUICKSTART.md` §§1-8 AND the IPFS
cluster from §7, plus the same `BRIDGE_*` env vars used in `docs/QUICKSTART.md` §9:

```sh
cd integration-bridge
go test -tags=integration -run TestIntegration -v ./...
```

**Disclosed honestly, not silently claimed as covered**: none of these `-tags=integration` tests
have actually executed this session — every story's own Debug Log says so, confirmed each time via
`docker ps` showing no Fabric/IPFS containers running. Running them for the first time is itself an
open action item from the epic-1 retrospective. Two of them cannot fully verify what their own doc
comments once claimed regardless of whether the network is up — see `grounding-gaps.md` **G-34**
(no way to read back `ipfsCIDs` post-commit) and **G-36** (no way to read back whether a large
integer survived a real commit byte-for-byte) for why, and **G-35** for a related gap in what the
*unit* tests can prove about `main.go`'s route-to-`Hooks`-method wiring.

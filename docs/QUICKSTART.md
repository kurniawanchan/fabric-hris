# Quickstart — Bring Up the Whole Prototype From Scratch

This is the practical "how do I actually run this" guide this workspace never had — every piece
below was built and verified live across `NET-1` through `QA-8` (`agent-suite/06-roadmap/
implementation-backlog.md`), but no single script ties them together, because each was built and
run manually across many separate work sessions. This document is that missing sequence.

**Scope**: gets you a running 3-org Fabric 2.5 network + committed chaincode + a 2-node IPFS
private swarm + a working end-to-end write/verify, on your own machine. It does **not** cover the
`deploy/` directory's Helm/CI/observability artifacts — those are deliberately never-applied
design artifacts (see `docs/CODEBASE-MAP.md` §6), not part of this local bring-up.

## 0. Prerequisites

- Docker + Docker Compose v2 (`docker compose`, not the standalone `docker-compose` binary — all
  compose files in this repo assume the v2 CLI plugin).
- Go 1.25.9 (`go version` to confirm) for everything under `fabric-network/tools/*` and
  `write-path-integration/*` — each of those is its own module with its own `go.mod`; see the
  "Go workspace gotcha" in §8 before you run anything.
- Node.js v18+ (only needed if you also want to run the Caliper performance suite, §7 of
  `docs/REPRODUCING-RESULTS.md`).
- ~8GB free RAM and a few GB of free disk for Docker — this stack runs 8+ containers
  simultaneously (4 peers, 3 orderers, the CCaaS chaincode server, 2 IPFS nodes). `QA-4`'s own
  benchmark found this genuinely strains an 8-vCPU/16GB laptop under real load; idle bring-up is
  fine.

## 1. Generate crypto material

```sh
cd fabric-network/network
docker run --rm -v "$PWD":/work -w /work hyperledger/fabric-tools:2.5 \
  cryptogen generate --config=./crypto-config/crypto-config.yaml --output=./crypto-config
```

**Known gotcha (hit and fixed once this session, at `NET-7`): `cryptogen` is NOT idempotent.**
Re-running this command against an org that already has material silently regenerates a fresh CA
keypair, orphaning every cert that CA already issued. Run this exactly once per org; if you need to
add a tenant later, use `fabric-network/tools/tenantprovision` (§9), which has its own
skip-if-exists guard — don't re-run raw `cryptogen` by hand.

## 2. Generate the channel genesis blocks

This network uses Fabric's **channel participation API** (`osnadmin`), not a system-channel
genesis block — there is no `SystemChannel` profile in `configtx.yaml`, only per-tenant-channel
application profiles (`TenantChannelGenesis` for `tenant01`, `Tenant02ChannelGenesis` for
`tenant02`, added later by `tenantprovision`).

```sh
docker run --rm -v "$PWD":/work -w /work hyperledger/fabric-tools:2.5 \
  configtxgen -profile TenantChannelGenesis -outputBlock ./channel-artifacts/tenant-tenant01.block \
    -channelID tenant-tenant01 -configPath ./configtx
```

## 3. Bring up the peers and orderers

```sh
cd compose
docker compose -f network-docker-compose.yaml up -d
```

This starts `peer0.org1`, `peer1.org1`, `peer0.org3`, `peer0.tenant01`, and the 3-node Raft orderer
set (`orderer0/1/2.org1`), all on a Docker network named `fabric-network-net`.

**Fabric CA (`compose/ca-docker-compose.yaml`) is OPTIONAL, not required for a working network.**
`QA-5`'s `ST-3` finding: the Fabric-CA-issued PKI's root certs are not actually this network's live
trust anchor (disjoint from `configtx.yaml`) — the network above already works entirely off the
`cryptogen`-generated material from §1. Bring up the CA servers only if you specifically want to
exercise `fabric-network/network/fabric-ca/enrollment-runbook.md`'s dynamic-enrollment path; skip
it for a plain working demo.

## 4. Create the `fabric-tools-net` helper container

Every Go tool under `fabric-network/tools/` (this repo's own pattern for driving real `peer`/
`osnadmin`/`configtxgen` binaries, rather than reimplementing Fabric's wire protocols) works by
`docker exec`-ing into one long-lived helper container. Nothing creates it automatically — start
it once:

```sh
docker run -d --name fabric-tools-net \
  --network fabric-network-net \
  -v "$PWD/../network":/net:ro \
  -v "$PWD/../chaincode/employeeprofilerecord":/cc:rw \
  hyperledger/fabric-tools:2.5 sh -c "sleep infinity"
```

(`/cc` is where the chaincode package tarball for §6 needs to live — see that step.)

## 5. Join the channel

```sh
cd ../tools/netjoin
go run .
```

This is idempotent — safe to re-run. It joins all 3 orderers (`osnadmin channel join`) and all 4
peers (`peer channel join -b .../tenant-tenant01.block`) to `tenant-tenant01`, then verifies every
peer reports the channel joined.

## 6. Build and deploy the chaincode (CCaaS, not `--lang golang`)

```sh
cd ../../chaincode/employeeprofilerecord
docker build -t employeeprofilerecord:1.0 .
```

Package it as a Chaincode-as-a-Service tarball (see the Dockerfile and `ccaas-package/` for the
exact `metadata.json`/`connection.json` shape — **`connection.json` must be at the package source
ROOT, not a `chaincode/server/` subdirectory**, a real defect hit once at `CC-5`), copy the
resulting `employeeprofilerecord.tar.gz` to where `fabric-tools-net` mounted it (`/cc`, from §4),
then:

```sh
cd ../../tools/ccdeploy
go run .
```

This installs the package on all 4 peers, approves for Org1 and `OrgClient-tenant01`, commits with
`AND(Org1MSP.peer, OrgClient-tenant01MSP.peer)`, and runs an invoke+query smoke test. Read the
printed `Package ID` — you need it for the next step.

Start the actual chaincode server container with that exact package ID:

```sh
docker run -d --name employeeprofilerecord-ccaas \
  --network fabric-network-net \
  -e CHAINCODE_ID=<package ID from ccdeploy's output> \
  -e CHAINCODE_SERVER_ADDRESS=0.0.0.0:9999 \
  employeeprofilerecord:1.0
```

**Known gotcha**: the package ID changes any time the package's byte content changes — if you
rebuild/repackage, you must restart this container with the new ID, or every invoke will fail with
a chaincode-registration mismatch.

## 7. Bring up the IPFS private cluster (dev-grade, 2 nodes)

```sh
cd ../../../ipfs-cluster
docker compose up -d
```

Read `ipfs-cluster/README.md` first — it discloses a real, unresolved limitation (the
`ipfs-cluster` CRDT orchestration layer's two peers don't negotiate a security handshake with each
other) and explains why `write-path-integration/ipfsclient` pins directly on both kubo nodes
instead of relying on that layer. The two kubo nodes (ports 5001/5002) are what actually matters
and are fully working.

## 8. Run your first real write + verify

**Go workspace gotcha, hit repeatedly this session**: `write-path-integration/go.work` ties 4
separate Go modules together. `go build ./...`/`go test ./...` must be run from **inside the
specific module directory** — running them from the `write-path-integration` root fails with
`"directory prefix . does not contain modules listed in go.work"`. Every command below already
accounts for this.

```sh
cd write-path-integration/writepaths
go test -tags=integration -run TestIntegration_AllFiveWritePathsAgainstLiveNetwork -v ./...
```

This exercises all five write-path hooks (`UpdatePersonalData`, `ApproveEmploymentTransfer`,
`RecordEducationHistory`, `ApproveFamilyDataChange`, `UpdatePayrollBankAccount`) against the live
network end-to-end, then reads back a fanned-out summary across all five sections. If this passes,
your whole stack is genuinely working, not just "containers are up."

## 9. Bring up `integration-bridge/` and hit it over HTTP

Step 8 called the five write-path hooks directly as Go functions. `integration-bridge/` is a
standalone HTTP service that exposes those same five hooks over `/v1/profile-sections/<SECTION>`
routes — this is what a real caller of the HRIS platform's write paths would actually talk to.

```sh
cd integration-bridge
BRIDGE_PEER_ENDPOINT=localhost:7051 \
BRIDGE_TLS_SERVER_NAME=peer0.org1 \
BRIDGE_TLS_CA_CERT_PATH=../fabric-network/network/crypto-config/peerOrganizations/org1/tlsca/tlsca.org1-cert.pem \
BRIDGE_CLIENT_TLS_CERT_PATH=../fabric-network/network/crypto-config/peerOrganizations/org1/users/Admin@org1/tls/client.crt \
BRIDGE_CLIENT_TLS_KEY_PATH=../fabric-network/network/crypto-config/peerOrganizations/org1/users/Admin@org1/tls/client.key \
BRIDGE_MSP_ID=Org1MSP \
BRIDGE_SIGN_CERT_PATH=../fabric-network/network/crypto-config/peerOrganizations/org1/users/Admin@org1/msp/signcerts/Admin@org1-cert.pem \
BRIDGE_SIGN_KEY_PATH=../fabric-network/network/crypto-config/peerOrganizations/org1/users/Admin@org1/msp/keystore/priv_sk \
BRIDGE_CHANNEL_NAME=tenant-tenant01 \
BRIDGE_CHAINCODE_NAME=employeeprofilerecord \
BRIDGE_TENANT_ID=tenant01 \
BRIDGE_IPFS_PRIMARY_API=127.0.0.1:5001 \
BRIDGE_IPFS_REPLICA_API=127.0.0.1:5002 \
BRIDGE_API_KEY=dev-only-key \
BRIDGE_COMPANY_ID=tenant01 \
go run ./cmd/integrationbridge
```

`LoadConfig` fails fast and names every missing `BRIDGE_*` var at once if you drop one — there's no
silent partial startup. In a separate terminal:

```sh
curl -s -X POST http://localhost:8080/v1/profile-sections/PERSONAL \
  -H "X-Api-Key: dev-only-key" -H "X-Company-ID: tenant01" \
  -d '{"employeeInternalID":"quickstart-emp-1","userID":"quickstart-user-1","newValue":{"fullName":"Quickstart Test"}}'
```

A working stack returns `{"status":"committed","recordID":"..."}`. `status` is the **only**
business-outcome channel (HTTP status is pure transport, per `ARCHITECTURE-SPINE.md` AD-3) — a
`"partial_failure"` here means the operational write succeeded but the chain anchor didn't, not
that nothing happened. `Ctrl-C` sends `SIGTERM`, which the process handles gracefully (stops
accepting new requests, closes the retained Gateway connection exactly once).

**No persistent store exists yet** (`wiring.go`'s own doc comment) — every process restart loses
every salt and `employeeKey_i` this service has ever generated. Fine for this quickstart, not for
anything real.

## 10. Optional: provision a second tenant

```sh
cd ../../fabric-network/tools/tenantprovision
go run . tenant02   # or any new tenant ID
```

Read this tool's own doc comment before running it more than once for the same tenant ID — it has
its own idempotency guards, but **do not deploy chaincode onto a second tenant's channel casually**:
this exact action crashed 4 peer containers twice this session (`NET-7`, then again at `QA-4`'s
`PT-4`) due to a recurring genesis-hash mismatch on a channel that had been created, abandoned, and
later touched again. See `implementation-backlog.md`'s `NET-7` row addendum before attempting this
against `tenant02` specifically — it is currently deliberately abandoned.

## 11. If something goes wrong

See `docs/CODEBASE-MAP.md` §7 for a table of every real defect found and fixed this session, by
component — most bring-up failures you'll hit have already happened once and are documented there
with the exact fix, not just the symptom.

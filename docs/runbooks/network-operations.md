# Runbook — Fabric Network Operations

**Classification: INTERNAL**

Operational procedures for the live prototype network described in `docs/QUICKSTART.md` and
`docs/CODEBASE-MAP.md`. This runbook does not re-derive design rationale (see `agent-suite/` ADRs
for that) — it is the "what to run, in what order, and what can go wrong" reference for someone
operating an already-bootstrapped, or about-to-be-bootstrapped, network. Read `CLAUDE.md`'s two
override rules (confidentiality register, git-is-not-a-safety-net) before running anything here.

Skills backing this runbook: `fabric-operations` (deploy/config/monitor/backup lifecycle) and
`fabric-troubleshooting` (RCA method, decision trees, log signatures) — both under
`.claude/skills/`. Route a live incident's root-cause work to `fabric-troubleshooting`'s decision
trees rather than guessing from this document alone.

## 1. Topology at a glance

- 3 orgs: `Org1` (platform), `OrgClient-tenant01` (per-tenant enterprise client), `Org3` (read-only
  auditor — installs chaincode, never endorses).
- 1 channel per tenant (`tenant-tenant01`), no system channel — channel participation API only.
- 3-node Raft ordering service (`orderer0/1/2.org1`); 1-of-3 can fail without halting ordering
  (`tools/raftfaulttest`, `NET-6`).
- State DB is LevelDB on every peer (`ADR-0007`) — no CouchDB container in this stack.
- Chaincode runs as CCaaS (`employeeprofilerecord-ccaas`), a separate container from the peers —
  not `--lang golang`.
- Off-chain: `ipfs-cluster/` (2-node dev-grade kubo swarm) + `integration-bridge/` (HTTP front for
  the five write-path hooks) + `write-path-integration/` (Gateway client, keystore, digest logic).

## 2. Standard startup sequence (network already provisioned once)

Use this when crypto material, genesis blocks, and the chaincode package already exist from a
prior bring-up (`docs/QUICKSTART.md` §1–§2 and §6 already run once) and you're only restarting
containers.

```sh
docker ps --format '{{.Names}}\t{{.Status}}' | sort     # confirm what's actually up before acting

cd fabric-network/network/compose
docker compose -f network-docker-compose.yaml up -d      # peers + orderers

docker start fabric-tools-net || \
  docker run -d --name fabric-tools-net --network fabric-network-net \
    -v "$PWD/..":/net:ro -v "$PWD/../chaincode/employeeprofilerecord":/cc:rw \
    hyperledger/fabric-tools:2.5 sh -c "sleep infinity"

docker start employeeprofilerecord-ccaas    # only if the package ID hasn't changed — see §5

cd ../../../ipfs-cluster
docker compose up -d
```

**Blast radius:** `docker compose up -d` alone is safe and idempotent — it does not touch ledger
volumes. Only `-v` on `down` is destructive (see §4).

Full first-time bring-up (crypto generation, genesis blocks, channel join, chaincode deploy) is
`docs/QUICKSTART.md` §1–§9 — that sequence has one-time, non-idempotent steps (§1's `cryptogen`)
and is intentionally not repeated here; run it from that document, not from memory.

## 3. Health checks (read-only, safe to run any time)

Per node, over the operations-service HTTP endpoint (`fabric-operations` skill, `docs:
operations_service.rst`):

```sh
curl -s http://localhost:<ops-port>/healthz   # 200 {"status":"OK"} or 503 + failed_checks
curl -s http://localhost:<ops-port>/version
curl -s http://localhost:<ops-port>/logspec
```

Peer operations ports in this stack: `9444`–`9447`. Orderer operations ports: `7071` (`orderer0`,
remapped from `7070` — see §7 gotcha table), `8070`, `9070`.

**Known gap (`G-37`, `docs/CODEBASE-MAP.md` §8):** none of these operations listeners have TLS or
client-cert auth enabled in the live compose file, despite the security architecture claiming
mutual TLS everywhere. They carry no chaincode/PII data, but treat this as reachable-in-plaintext
today — do not expose these host ports beyond localhost until `G-37` is closed.

Channel/ledger height and consenter membership: use `tools/netjoin`'s verification output (re-run
is idempotent) or `fabric-troubleshooting/scripts/collect-diagnostics.sh` for a fuller evidence
bundle before escalating a suspected fault.

## 4. Shutdown

```sh
cd fabric-network/network/compose
docker compose -f network-docker-compose.yaml down       # safe — ledger volumes are named, survive

cd ../../../ipfs-cluster
docker compose down                                       # safe — 4 named data volumes survive
```

**`docker compose down -v` on either compose file destroys the named ledger/IPFS-data volumes —
unrecoverable in this repo (CLAUDE.md rule 2).** Never run `-v` without first confirming with
`git status`/`docker volume ls` that nothing uncommitted or unbacked-up depends on that volume, and
without explicit sign-off if this is anyone else's environment.

`fabric-tools-net` and `employeeprofilerecord-ccaas` are plain `docker run` containers, not part of
either compose file — `docker stop fabric-tools-net employeeprofilerecord-ccaas` to stop them;
`docker start` (not recreate) to bring them back, per §2.

## 5. Chaincode redeploy (CCaaS package-ID rotation)

Any time the packaged chaincode's bytes change (source edit, rebuild):

```sh
cd fabric-network/chaincode/employeeprofilerecord
docker build -t employeeprofilerecord:1.0 .
# repackage per the Dockerfile / ccaas-package/ shape — connection.json MUST sit at the package
# source root, not a subdirectory (real defect, CC-5)
# copy the new employeeprofilerecord.tar.gz to fabric-tools-net's /cc mount

cd ../../tools/ccdeploy
go run .    # install on all 4 peers, approve Org1 + OrgClient-tenant01, commit, smoke test
```

Read the printed **Package ID** from `ccdeploy`'s output, then:

```sh
docker rm -f employeeprofilerecord-ccaas   # old container is bound to the old package ID
docker run -d --name employeeprofilerecord-ccaas \
  --network fabric-network-net \
  -e CHAINCODE_ID=<new package ID> \
  -e CHAINCODE_SERVER_ADDRESS=0.0.0.0:9999 \
  employeeprofilerecord:1.0
```

**Blast radius:** every in-flight invoke fails with a chaincode-registration mismatch until this
container is restarted with the matching ID — there is no backward compatibility window. This is a
mechanical, expected step, not a fault to troubleshoot.

## 6. Tenant provisioning — danger zone

```sh
cd fabric-network/tools/tenantprovision
go run . <tenantID>
```

**This is the single most dangerous command in this stack.** It crashed 4 peer containers twice in
this project's history (`NET-7`, then again at `QA-4`'s `PT-4`), via a recurring genesis-hash
mismatch when a channel had been created, abandoned, and later touched again. `tenant02` is
currently, deliberately abandoned — do not re-provision it casually. Before running this for any
tenant ID:

1. Read `docs/QUICKSTART.md` §10 and the `NET-7` backlog addendum in
   `agent-suite/06-roadmap/implementation-backlog.md`.
2. Confirm the target tenant ID has never been provisioned-then-abandoned before.
3. Have a plan for peer-ledger-volume reset + `netjoin` rejoin as the recovery path if it fails —
   do not attempt a raw channel-join/create against a recreated channel by hand.

## 7. Known startup failure modes (check here before RCA from scratch)

| Symptom | Cause | Fix |
|---|---|---|
| `docker compose up -d` fails: port `7071` already in use | Host-side collision (historically AnyDesk's relay port on macOS) | `lsof -nP -iTCP:7071 -sTCP:LISTEN`, stop the offending process or remap the host port in `network-docker-compose.yaml` — container-side `8443` is unaffected either way |
| `docker run --name fabric-tools-net` fails: name already in use | Helper container exists (stopped) from a prior session | `docker inspect fabric-tools-net` to confirm image/network/mounts match, then `docker start fabric-tools-net` |
| `osnadmin`/`peer channel join` fails with `x509: certificate signed by unknown authority` after any crypto regen | Channel genesis `.block` bakes in orderer TLS certs at generation time — stale if `crypto-config/` changed since | Regenerate the `.block` with `configtxgen` against current crypto material, then rejoin |
| Peer crashes with "unexpected Previous block hash" on a tenant channel | Dormant/recreated channel genesis conflicts with a peer's stale local copy | Ledger-volume reset + `tools/netjoin` rejoin — never a casual channel-join/create against that channel |
| Re-running raw `cryptogen` against `crypto-config.yaml` on a live network | `cryptogen` is not idempotent — mints a fresh CA keypair per run, orphaning every already-issued cert | Never re-run it directly; `tenantprovision` has its own skip-if-exists guard for adding a tenant. Recovery: `docker compose down -v` (confirm first, destroys ledgers), regenerate crypto, then regenerate **every** channel genesis block too |
| Invoke fails with a chaincode-registration mismatch right after a rebuild | `employeeprofilerecord-ccaas` still running with the old `CHAINCODE_ID` | Restart it with the new package ID — see §5 |

For anything not in this table, hand off to `fabric-troubleshooting`'s RCA loop: reproduce & scope →
collect evidence (`/healthz`, `/version`, logs, ledger height) → classify domain via its symptom
routing index → match the log signature → escalate targeted `FABRIC_LOGGING_SPEC` only for the
suspected domain, then revert it.

## 8. Backup

Ledger + MSP copy, per node, while it's running or stopped:

```sh
docker cp <node-container>:/var/hyperledger/production ./backup-<node>-$(date +%Y%m%d)
```

No backup automation exists in this repo today — this is a manual, per-node procedure
(`fabric-operations` skill, `references/backup-restore.md`, for the fuller snapshot/`ledgerutil`
options this prototype hasn't needed yet). Take a backup before any tenant-provisioning attempt
(§6) or before a chaincode redeploy you're not confident about.

## 9. Scope boundaries

- `deploy/`'s Helm/CI/observability artifacts are **never applied** to this or any real
  infrastructure — do not treat their existence as an alternate runbook path.
- Fabric CA (`ca-docker-compose.yaml`) is optional; the live trust anchor is the `cryptogen`
  material, not Fabric CA (`QA-5` `ST-3`). Skip bringing the CA servers up unless you're
  specifically exercising `fabric-network/network/fabric-ca/enrollment-runbook.md`.
- Never run `npm install` in `qa-tests/performance/` — its `node_modules/` carries a hand-patched
  mTLS fix with no reapply mechanism; a reinstall silently reverts it.

# ST-3 / ST-4 — Certificate Revocation & Transport Security Report

Date: 2026-08-07. Scope: ST-3 (certificate revocation) and ST-4 (transport security — Fabric
gRPC endpoints and the IPFS Private Cluster HTTP boundary). All findings below were produced by
read-only inspection and live, non-mutating test calls against the running network unless stated
otherwise. No container was created, removed, recreated, or joined to a channel; no existing
identity used by any other live test in this repo was touched.

---

## ST-3 — Certificate revocation: BLOCKED, not fabricated — genuine, documented limitation

**Outcome: could not be executed live this session. No new identity was enrolled or revoked.**
Three independent, stacked reasons, found in this order:

### 1. The Fabric CA servers themselves are down

`fabric-network/network/fabric-ca/enrollment-runbook.md` (read first, per instructions) requires
`ca.org1`, `ca.tenant01`, `ca.org3` (`hyperledger/fabric-ca:1.5`) to be reachable on
`localhost:7054/8054/9054`. Live check:

```
$ docker ps -a --format 'table {{.Names}}\t{{.Status}}\t{{.Image}}' | grep -i ca
ca.org1        Exited (137) 22 hours ago   hyperledger/fabric-ca:1.5
ca.tenant01    Exited (137) 22 hours ago   hyperledger/fabric-ca:1.5
ca.org3        Exited (137) 22 hours ago   hyperledger/fabric-ca:1.5
```

`docker inspect ca.org1`: `startedAt=2026-08-06T13:44:06Z finishedAt=2026-08-06T13:51:28Z`,
`ExitCode=137` (SIGKILL). `docker logs --tail 15 ca.org1` shows its last recorded activity was a
successful `peer0.org1` enrollment (`INFO ... POST /enroll 201`) around the exact time the
enrollment-runbook documents doing that work (2026-08-06) — i.e. these CAs were used once for
that runbook's own verification pass and have not been running since; this looks like the
expected "parked after use" state of an exploratory apparatus, not an active crash.

This session's standing safety rules forbid `docker compose up` (creating/recreating any
container) and instruct: *"If you observe something unhealthy, STOP and report it in your
output — do not attempt to fix live infrastructure yourself."* Bringing these three CA
containers back up (`docker start` or `docker compose up`) is exactly the kind of live-
infrastructure remediation that rule reserves for the orchestrator with explicit human
confirmation, so it was not attempted. **This alone blocks any live enrollment this session.**

### 2. Independent of #1 — the Fabric-CA PKI is not actually the live network's trust anchor

Checked what the task asked to check ("check which orgs in this network are actually
Fabric-CA-issued vs. cryptogen-issued") by comparing the **live peers' own configured MSP root**
against the **Fabric-CA servers' own generated root cert**, for all three orgs the runbook
covers:

| Org | Live peer/channel MSP root (`crypto-config/.../msp/cacerts/*.pem`) | Fabric-CA server's own root (`fabric-ca/enrollments/.../msp/cacerts/*.pem`) |
|---|---|---|
| org1 | `subject=C=US,ST=California,L=San Francisco,O=org1,CN=ca.org1` | `subject=C=ID,ST=Jakarta,O=Org1,OU=platform,CN=ca-org1` |
| tenant01 | `subject=C=US,ST=California,L=San Francisco,O=tenant01,CN=ca.tenant01` | `subject=C=ID,ST=Jakarta,O=OrgClient-tenant01,OU=client,CN=ca-tenant01` |
| org3 | `subject=C=US,ST=California,L=San Francisco,O=org3,CN=ca.org3` | (same pattern; not re-printed) |

These are **two completely disjoint root-of-trust chains** (different country/locality, different
CN naming convention — `ca.<org>` vs `ca-<org>` — and self-signed, so no cross-signing either).
`grep -rl "ca-org1-7054\|ca-tenant01\|ca-org3-7054\|Jakarta" fabric-network/network/configtx/`
returned nothing: no channel/configtx artifact references the Fabric-CA-issued root at all.

**Consequence:** even with the CA servers running, an identity freshly enrolled via
`fabric-ca-client` against `ca.tenant01` (per the runbook's own `hris-user1` example) would
present a certificate chaining to `CN=ca-tenant01` (Jakarta) — which is **not** in
`peer0.tenant01`'s local MSP trust store, nor in `tenant-tenant01`'s committed channel MSP
config (both trust only `CN=ca.tenant01`, the cryptogen-issued root). The task's own step
("Use it to make one successful call, proving it worked before revocation") would therefore fail
for an unrelated reason — untrusted issuer, not revocation — making any pass/fail signal from
that flow meaningless as a revocation test. NET-2's Fabric-CA apparatus is a self-consistent,
independently-verified PKI (per the runbook's own DoD table) that has evidently not yet been
wired into the live channels' MSP configuration as an additional/alternate root or intermediate.

### 3. Independent of #1 and #2 — no CRL distribution path exists anywhere in this repo today

Checked whether, if the above two blockers were somehow absent, CRL enforcement would even be
wired up:

- `grep -rn "RevocationList\|revocation_list\|crls\|CRL" fabric-network/network/configtx/configtx.yaml` — no hits.
- `find fabric-network/network/crypto-config -type d -iname crls` — no hits (cryptogen never
  generated a `crls/` folder in any org's MSP).
- Every Fabric-CA-issued identity's `msp/` folder (`enrollments/*/*/msp/`) contains
  `cacerts, keystore, signcerts, config.yaml, IssuerPublicKey, IssuerRevocationPublicKey` — the
  last one is an **Idemix** artifact, unrelated to X.509 CRL distribution — and **no `crls/`
  subfolder**.
- The CA config files (`ca-org1-config.yaml`, `ca-tenant01-config.yaml`, `ca-org3-config.yaml`)
  do enable revocation at the **CA-server** level (`hf.Revoker: true`, `hf.GenCRL: true`,
  `crl: {expiry: 24h}`, `crlsizelimit: 512000`) — the CA itself is capable of marking an identity
  revoked and generating a CRL on request (`fabric-ca-client revoke` /
  `fabric-ca-client certificate` / a manual `gencrl`).

Per Fabric's own MSP semantics, however, a CA marking an identity revoked in its **own registry**
does not by itself make any peer reject that identity's future calls — the peer/channel MSP only
consults a `RevocationList` that has been **explicitly fetched and placed** into that MSP's own
`crls/` folder (local MSP) or into the channel's committed MSP config (`peer channel
signconfigtx` / a config update), and re-read. No such file or config update exists anywhere in
this repository today. So even setting aside blockers #1 and #2, revoking an identity at the CA
would not currently cause any live peer to reject it — the enforcement point Fabric ships is
present in principle at the CA (capable of issuing a CRL) but **not wired to any peer's trust
decision** in this network's current configuration.

### Conclusion for ST-3

**Genuinely open / unverifiable in this workspace as currently configured**, for three
independent, stacked, verified reasons: (1) the CA servers are down and this session may not
restart live infrastructure; (2) even running, that CA's PKI is not the live channels' trust
anchor; (3) even if it were, no CRL-distribution wiring exists anywhere in this repo for a peer
to ever consult. **No new identity was enrolled. No identity — new or existing — was revoked.**
This is reported as an honest limitation per the task's own explicit instruction ("If this
network's CA setup does not actually support/enforce CRL checking at the peer level today ...
report that precisely as a genuine finding/limitation, not a fabricated pass"), not treated as a
failure to work around.

**What would need to happen, in order, before ST-3 could run live:** (a) the orchestrator brings
`ca.org1`/`ca.tenant01`/`ca.org3` back up with explicit human confirmation; (b) a decision is made
about whether NET-2's Fabric-CA PKI is meant to become (or be added as an intermediate/cross-
signed root alongside) the live channels' trust anchor, or whether ST-3 should instead be
redefined to test revocation against the cryptogen-issued PKI (cryptogen has no CA-server /
CRL-serving component at all, so that would need a different mechanism entirely — e.g. hand-built
CRL + manual channel MSP config update — which is its own, larger undertaking); (c) an explicit
CRL-distribution step (fetch CRL, place in the relevant MSP's `crls/`, update channel config) is
added to whichever PKI is chosen, since none exists today.

---

## ST-4(a) — Fabric transport security: PLAINTEXT CONNECTION REFUSED (live-verified PASS)

**Config check first** (`docker exec <peer> env`, all five live peers):

```
peer0.org1:      CORE_PEER_TLS_ENABLED=true  CORE_PEER_TLS_CLIENTAUTHREQUIRED=true
peer1.org1:      CORE_PEER_TLS_ENABLED=true  CORE_PEER_TLS_CLIENTAUTHREQUIRED=true
peer0.tenant01:  CORE_PEER_TLS_ENABLED=true  CORE_PEER_TLS_CLIENTAUTHREQUIRED=true
peer0.org3:      CORE_PEER_TLS_ENABLED=true  CORE_PEER_TLS_CLIENTAUTHREQUIRED=true
peer0.tenant02:  CORE_PEER_TLS_ENABLED=true  CORE_PEER_TLS_CLIENTAUTHREQUIRED=true
```

(`peer0.tenant02` was only read via `docker exec ... env` — a read-only config check, no channel
interaction of any kind, consistent with the standing instruction never to touch that peer's
channel membership.)

**Live behavioral proof, not just config**: new file
`write-path-integration/gateway-client/transport_security_integration_test.go` (build-tagged
`integration`, added to the `gatewayclient` package alongside its existing
`invalid_identity_integration_test.go` precedent) dials `localhost:7051` (peer0.org1) using
`google.golang.org/grpc/credentials/insecure` — no TLS negotiation, no certificate of any kind —
then issues one real RPC (`peer.EndorserClient.ProcessProposal` with an empty proposal) to force
the lazy `grpc.NewClient` connection to actually touch the wire.

Actual run against the live network:

```
$ cd write-path-integration/gateway-client && go build ./... && go vet -tags integration ./...
(no output — clean)

$ go test -tags integration -run TestIntegration_ST4_PlaintextConnectionToPeerRefused -v ./...
=== RUN   TestIntegration_ST4_PlaintextConnectionToPeerRefused
    transport_security_integration_test.go:67: plaintext (non-TLS) connection to peer0.org1:7051 correctly refused: rpc error: code = Unavailable desc = connection error: desc = "error reading server preface: EOF"
--- PASS: TestIntegration_ST4_PlaintextConnectionToPeerRefused (0.11s)
PASS
ok  	gatewayclient	0.913s
```

**Result: CONFIRMED PASS, live.** The peer's listening socket closes the connection before even
an empty gRPC frame is processed (`error reading server preface: EOF`), i.e. the TLS enforcement
happens at the transport/socket layer, before any Fabric-level (MSP/identity/proposal) logic runs
— exactly what `CORE_PEER_TLS_CLIENTAUTHREQUIRED=true` is supposed to produce. This holds in
practice, not just in config, for peer0.org1; the identical env config on the other four peers
gives no reason to expect different behavior there (not independently re-run against all five, to
avoid redundant live traffic against a network already carrying substantial synthetic history).

---

## ST-4(b) — IPFS Private Cluster HTTP API: PLAIN, UNENCRYPTED HTTP — genuine finding, not fixed

`write-path-integration/ipfsclient/ipfsclient.go` (read in full): `Client.PrimaryAPI` /
`Client.ReplicaAPI` are documented and used as `"http://localhost:5001"` /
`"http://localhost:5002"` (see the field doc-comments), and every call (`add`, `pin`, `cat`) uses
a plain `*http.Client{}` with no `tls.Config`/`https://` scheme anywhere in the file.

**Live-verified, both nodes:**

```
$ curl -s --max-time 3 -X POST http://localhost:5001/api/v0/version
{"Version":"0.43.0","Commit":"e9914bb","Repo":"18","System":"arm64/linux","Golang":"go1.26.5"}

$ curl -s --max-time 3 -X POST http://localhost:5002/api/v0/version
{"Version":"0.43.0","Commit":"e9914bb","Repo":"18","System":"arm64/linux","Golang":"go1.26.5"}

$ curl -sk --max-time 3 -o /dev/null -w "status:%{http_code}\n" https://localhost:5001/api/v0/version -X POST
status:000   # connection failure — nothing is speaking TLS on this port at all, confirming it's plain HTTP, not "HTTP that happens to also accept HTTPS"
```

`docker exec ipfs0/ipfs1 env | grep -i PNET` confirms `LIBP2P_FORCE_PNET=1` on both nodes — the
private-swarm **libp2p** layer (peer-to-peer bitswap/DHT traffic between `ipfs0`/`ipfs1`) is a
genuinely separate mechanism from the **kubo HTTP API** (`:5001`/`:5002`, used by
`ipfsclient.go` for `add`/`pin`/`cat`) that this client actually speaks to. `LIBP2P_FORCE_PNET`
does nothing to protect the HTTP API's own transport.

**Finding (reported honestly, not silently patched — out of this item's scope per the task):**
the plaintext-vs-ciphertext content protection (`EncryptAndAdd`'s AES-256-GCM, keyed on
`KEY_EMPLOYEE`) means the **document bytes themselves** are not exposed by this gap — the client
already encrypts before ever calling `add`. What IS exposed to anyone who can observe traffic on
`localhost:5001`/`:5002` (or the equivalent path in a non-local deployment) is: (1) the resulting
**CID** in the `add` response and the `pin`/`cat` request URLs (CIDs are content-addressed hashes
of the ciphertext, not secret by IPFS's own design, but still a request/response metadata leak
over an unauthenticated, unencrypted channel); (2) the **fact and timing** of add/pin/cat calls
(traffic analysis); (3) if this pattern is ever deployed anywhere the two kubo nodes are not both
`localhost` (e.g. across hosts/containers on a network with any untrusted hop), the ciphertext
payload itself would cross that hop in plaintext-transport `multipart/form-data` — still
AES-GCM-protected content, but with zero transport-layer confidentiality, integrity, or endpoint
authentication (no cert to pin, nothing preventing a MITM from swapping in different ciphertext
undetected until AES-GCM's tag check fails at decrypt time, or from tampering enroute). Today,
with both nodes on `localhost`, the practical exposure is low; the gap is real and would become
significant the moment either kubo node moves off of `localhost` relative to the caller.
**No fix applied to `ipfsclient.go`** — out of this item's scope per the task's own instruction.

---

## Files touched by this task

**Created only** (no existing file edited, per the task's constraint):
- `write-path-integration/gateway-client/transport_security_integration_test.go`
- `qa-tests/security/st3-st4-revocation-transport-report.md` (this file)

## Confidentiality check

Both files created by this task were grepped for the confidentiality register's disallowed
real-world names before reporting done: zero hits in either file.

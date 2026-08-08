# IPFS Private Cluster — production deployment artifact (DEP-3)

Backlog item `DEP-3` (`agent-suite/06-roadmap/implementation-backlog.md`):
*"Swarm-key-gated private cluster stood up; encrypt-before-add enforced at
the `REC-5` call site (never bypassable by uploading unencrypted content);
pinning-peer operator recorded as Org1-operated per ADR-0016's
recommendation (not yet ratified — flagged `[ASSUMPTION]`, not asserted as
decided)."* Authority: `agent-suite/05-adr/ADR-0016-ipfs-private-cluster.md`.

**Standing scope decision (2026-08-07, binding):** this directory is a
**generic artifact set** — Helm chart + K8s manifests, reviewable, not
applied. Nothing here has been `helm install`'d or `kubectl apply`'d against
a real cluster. It translates the already-proven `ipfs-cluster/` dev
docker-compose bring-up to Kubernetes, generalizing 2 nodes to N and adding
the production hardening (internal-only API exposure, `existingSecret`-only
secret handling, NetworkPolicy, PDB) the dev cluster's own README explicitly
deferred.

**Read `ipfs-cluster/README.md` and `ipfs-cluster/docker-compose.yaml`
first.** Every design choice below is either a direct translation of a
choice already proven working there, or an explicit, labeled addition —
none of it re-derives the kubo/ipfs-cluster configuration from scratch.

## Layout

```
deploy/ipfs-cluster/
  Chart.yaml, values.yaml, .helmignore   — chart scaffold
  values-production-example.yaml         — example override (N=3), not applied anywhere
  templates/
    _helpers.tpl                         — name/fullname/secret-name/pod-FQDN helpers
    configmap-disable-autoconf.yaml      — verbatim copy of ipfs-cluster/init-hooks/disable-autoconf.sh's logic
    statefulset-kubo.yaml                — N single-replica StatefulSets (the private swarm itself)
    service-kubo-headless.yaml           — per-ordinal stable DNS for kubo pods
    service-kubo-api.yaml                — ClusterIP-only kubo API access (ops/debug convenience)
    statefulset-ipfs-cluster.yaml        — N single-replica StatefulSets (CRDT orchestration layer — see "Disclosed limitation" below before trusting this layer for anything)
    service-cluster-headless.yaml        — per-ordinal stable DNS for cluster-peer bootstrap
    service-cluster-restapi.yaml         — ClusterIP-only cluster REST API
    networkpolicy.yaml                   — defense-in-depth, NOT the confidentiality boundary (the swarm key is)
    poddisruptionbudget.yaml
    NOTES.txt
```

No `Secret` manifest exists anywhere in this chart. See "Secrets" below —
this is deliberate, not an oversight.

## Why standalone (not a DEP-1 sub-chart)

`DEP-1` (Helm chart for the Fabric network — Org1 peers/orderers + per-tenant
`OrgClient` template) had not been built as of this writing (no `Chart.yaml`
existed anywhere in this repo when this item started — checked directly, not
assumed). Even once it exists, this chart is kept **standalone** rather than
folded in as a sub-chart, for three reasons:

1. **Different trust boundary.** ADR-0016 states plainly: *"The cluster's
   node operator is a fourth trust boundary, distinct from the three Fabric
   MSP-credentialed boundaries... governed by the cluster's own
   access-control plane (swarm key + cluster REST API auth), entirely
   outside Fabric's X.509/MSP layer."* A sub-chart under DEP-1's umbrella
   would visually and operationally imply this boundary is part of the
   Fabric MSP trust model. It is not, and this chart's independence keeps
   that fact structurally visible rather than folded away.
2. **Different lifecycle.** A Raft leader flap requiring an orderer restart
   (DEP-1's concern) has nothing to do with IPFS swarm/cluster-peer health,
   and vice versa. One Helm release per concern means one can be upgraded,
   rolled back, or paged on independently of the other — coupling them into
   one release would force every Fabric-node change and every IPFS-node
   change through the same install/rollback unit for no operational benefit.
3. **Composable later without modification.** If a future umbrella chart
   wants both under one `helm install`, the idiomatic Helm mechanism is an
   umbrella chart's own `Chart.yaml` `dependencies:` list pointing at this
   chart (and at DEP-1's), not this chart being rewritten as a sub-chart.
   Standalone-by-default is the more composable starting point, not a
   dead end.

## Kubo private swarm — the actual ADR-0016 requirement

Translated 1:1 from `ipfs-cluster/docker-compose.yaml`'s `ipfs0`/`ipfs1`
services, generalized from 2 to `kubo.nodeCount` (default 2, matching the
dev cluster's own proven count — ADR-0016's recommended floor, not a ceiling):

- `LIBP2P_FORCE_PNET=1` + `IPFS_SWARM_KEY_FILE=/swarm-key-source/swarm.key`
  — identical env vars, identical reasoning (dev compose file's own comment,
  carried forward verbatim in `templates/statefulset-kubo.yaml`): this is
  the actual private-network enforcement, and the swarm key is installed via
  the entrypoint's own documented mechanism rather than bind-mounted
  directly at `/data/ipfs/swarm.key` (which the dev cluster's *first*
  attempt tried and broke — a host/Secret-mounted RO file can't be chowned
  to the container's `ipfs` user once the entrypoint's privilege-drop runs).
- **AutoConf disabled via the identical `/container-init.d` hook mechanism**
  — `templates/configmap-disable-autoconf.yaml` mounts a ConfigMap at
  `/container-init.d/disable-autoconf.sh` containing the **exact same shell
  logic** as `ipfs-cluster/init-hooks/disable-autoconf.sh` (copied verbatim,
  not reimplemented — same five commands: `AutoConf.Enabled=false`,
  `bootstrap rm --all`, and clearing `Bootstrap`/`DNS.Resolvers`/
  `Routing.DelegatedRouters`/`Ipns.DelegatedPublishers`). Kubernetes changes
  nothing about *why* this is needed (kubo 0.43's AutoConf feature refuses
  to start on a private network unless explicitly disabled — a property of
  the kubo image's own entrypoint script, not of Docker Compose) — the same
  hook mechanism works identically because it's the same container image
  running the same entrypoint script, regardless of orchestrator.
- **No `command`/`args` override on the kubo container.** The dev cluster's
  own history includes a documented *failed* second attempt at this exact
  fix via overriding entrypoint/command — it skipped the entrypoint's
  privilege-drop and init sequencing entirely. This chart does not repeat
  that mistake: the kubo container spec sets no `command:` and no `args:`
  at all, letting the base image's own entrypoint run unmodified.
- **N single-replica `StatefulSet`s, not one N-replica `StatefulSet`.**
  Explained in `templates/statefulset-kubo.yaml`'s own header comment: kubo
  alone doesn't strictly need this (its config is homogeneous across
  ordinals), but the paired `ipfs-cluster` StatefulSet genuinely does (next
  section) — kept consistent across both so an operator debugging one
  understands the other the same way, rather than mixing two different
  "how do I get N of these" patterns in one chart.

## ipfs-cluster peers — deployed, but read this before trusting them

`templates/statefulset-ipfs-cluster.yaml` deploys `ipfsCluster.peerCount`
peers, each paired 1:1 by ordinal with a kubo node
(`CLUSTER_IPFSHTTP_NODEMULTIADDRESS` points at that same-ordinal kubo pod's
own stable DNS name) — the same pairing convention as `cluster0`→`ipfs0`,
`cluster1`→`ipfs1` in the dev compose file. `CLUSTER_SECRET_FILE` is sourced
from a K8s `Secret` (never inline — see "Secrets" below). The `--bootstrap`
flag (not `CLUSTER_PEERADDRESSES`, which the dev cluster already proved is
**not** an active CRDT bootstrap trigger — it only seeds `service.json`'s
`peer_addresses` field at init time) is passed via `args:` only, never
`command:`, for the identical reason as the kubo container above: the
`ipfs-cluster` image's own entrypoint execs `ipfs-cluster-service $@`,
performing its own init-on-first-run logic before that exec — overriding
`command:` to inject ordinal-computation shell logic would skip that
init step, the same class of mistake the kubo entrypoint override already
proved is broken.

### Cluster peer bootstrap runbook

Peer 0's own libp2p peer ID (needed to construct peers 1..N-1's
`--bootstrap` multiaddr) **cannot be known before peer 0 has booted once** —
the identical constraint the dev cluster hit (`cluster0`'s peer ID was
"read from its own first-boot log output," per that file's own comment).
This chart does not pretend to solve that with automation it hasn't
actually exercised:

1. `helm install <release> deploy/ipfs-cluster -f <env-values>.yaml --set ipfsCluster.bootstrapPeerID=""`
   (or simply leave it unset). Only peer ordinal 0 is created — peers
   1..N-1 are **templated out entirely** (see the `{{- else }}` branch in
   `statefulset-ipfs-cluster.yaml`: a real `# ordinal N SKIPPED` comment is
   rendered in their place), not started half-configured and left to fail.
2. Read peer 0's ID: `kubectl logs <release>-cluster-0-0 -n <namespace>` or
   `kubectl exec <release>-cluster-0-0 -n <namespace> -- wget -qO- --post-data='' http://localhost:9094/id`.
3. `helm upgrade <release> deploy/ipfs-cluster -f <env-values>.yaml --set ipfsCluster.bootstrapPeerID=<peer-0-id>`.
   Peers 1..N-1 now render with `--bootstrap /dns4/<release>-cluster-0.<release>-cluster-headless/tcp/9096/p2p/<peer-0-id>`.
4. If a peer is ever re-bootstrapped after a previously failed handshake
   attempt, clear its PVC-persisted `/data/ipfs-cluster/peerstore` first —
   the dev cluster found stale peerstore entries from earlier failed
   attempts caused `"dial to self attempted"` errors that persisted even
   after the bootstrap-flag fix; PVC-backed state in K8s can go stale the
   same way local Docker volumes did.

**Getting bootstrap addressing right does not mean pin replication works.**
See the next section — that is a separate, deeper, still-open problem.

## Disclosed limitation: the CRDT peer-to-peer handshake defect

`ipfs-cluster/README.md` discloses that `cluster0`/`cluster1` in the dev
cluster **never complete a libp2p security handshake with each other**
(`failed to negotiate security protocol: incoming message was too large`),
after three *other* real defects in that bring-up were found and fixed
(AutoConf, the `CLUSTER_PEERADDRESSES` non-trigger, a stale peerstore) and
with both peers' `CLUSTER_SECRET_FILE` contents confirmed byte-identical
(MD5 match — not a mismatched-secret problem). The root cause was not found
after a genuine debugging effort there and remains open.

**This is a defect in the `ipfs-cluster` peer binaries' own libp2p
application-layer negotiation — not a Docker-vs-Kubernetes difference.**
Moving these peers into `StatefulSet`s, giving them stable DNS names, and
fixing their bootstrap-addressing story (previous section) changes none of
the bytes exchanged during that handshake. Nothing in this chart claims or
implies the defect is fixed by virtue of running on Kubernetes; asserting
that without having actually reproduced a successful handshake on this
chart's manifests against a real cluster would be exactly the kind of
unverified claim this project's own discipline (NET-4, `write-path-
integration`'s own testing) has consistently avoided elsewhere.

**Production posture adopted by this artifact (option (b), not (a)):**
carry the dev cluster's own workaround forward as the accepted production
model for this prototype's scale — **explicit pin-on-every-node from the
application layer, not CRDT auto-replication.** Concretely: `write-path-
integration/ipfsclient`'s `Client.PrimaryAPI`/`Client.ReplicaAPI` 2-endpoint
pattern (`ipfsclient.go`'s `EncryptAndAdd`: add to primary, explicitly
`pin/add` on replica) is the shape a production caller should generalize to
an **N-element endpoint list**, POSTing an explicit `pin/add` to every kubo
node's own API (`service-kubo-api.yaml`'s per-ordinal headless DNS names,
not the load-balanced `service-kubo-api` Service — see that file's own
comment on why) after the initial `add`. This is an honest trade-off, stated
plainly rather than glossed over:

- **What is lost:** no automatic rebalancing or consensus-driven pin
  redistribution if a node is added, removed, or temporarily unavailable
  during a write — an operator-managed, explicitly-enumerated pin-set
  instead of a self-healing one. Adding a node to `kubo.nodeCount` does
  **not** automatically get it a copy of previously-pinned content; that
  requires either a one-time backfill job (enumerating existing CIDs from
  the ledger's `ipfsCIDs` fields and re-pinning them on the new node) or
  accepting that new nodes only hold content pinned *after* they joined.
  Neither is built here — flagged as follow-up work, not silently assumed
  solved.
- **What is kept:** the actual ADR-0016 property (N independently-durable
  copies of each ciphertext blob, swarm-key-gated) — proven true today for
  N=2 (`write-path-integration/ipfsclient`'s live cross-node round-trip
  tests), and structurally identical for larger N, since each explicit
  `pin/add` call is independent of the others and does not depend on the
  peers ever succeeding at a CRDT handshake.

**Concrete next debugging step, if a future rollout wants to keep trying
toward a working CRDT layer (option (a), offered as future work, not
required by this posture):** `"incoming message was too large"` is the
canonical error libp2p's multistream-select protocol negotiation produces
when one side's handshake message exceeds the other side's configured
buffer/length limit — a plausible next step, not yet tried in this
project's own history, is checking whether the two peer images pulled as
`ipfs/ipfs-cluster:latest` at different times resolved to **different minor
versions** with different default multistream-select limits (a version
skew the dev cluster's own floating `:latest` tag, by construction, cannot
rule out — see `values.yaml`'s own `[VERIFY]` note on pinning that tag). A
second, cheaper check before that: re-verify the secret file's byte-for-byte
content **including trailing whitespace/newline and encoding** on both
peers via `md5sum` (the dev cluster's own MD5 check compared the two files
to each other, which rules out mismatch between them, but not a shared
malformed pattern — e.g. a stray trailing newline both files share that a
stricter reader on one binary version rejects, a subtler failure mode than
a plain mismatch). Neither of these has been tried; both are stated as
next steps, not fixes.

## Encrypt-before-add enforcement

Confirmed by reading `write-path-integration/ipfsclient/ipfsclient.go`
directly (not assumed): `Client.EncryptAndAdd` calls the package-private
`encrypt()` function (AES-128-GCM, fresh random nonce every call) and only
THEN calls `c.add(ctx, c.PrimaryAPI, ciphertext)` — the HTTP POST to the
kubo `/api/v0/add` endpoint receives `ciphertext`, never `plaintext`. The
plaintext parameter never crosses a network boundary; there is no code path
in that file that calls `c.add` with unencrypted `data`.

**This is an application-level guarantee, not a cluster-config
guarantee, and this chart does not claim otherwise.** No Helm value, no
`NetworkPolicy`, no `Secret`, and no kubo/ipfs-cluster server-side setting
anywhere in this directory inspects, validates, or rejects the CONTENT of
what is POSTed to `/api/v0/add` — the kubo API has no concept of "this blob
must have been encrypted by caller X first." Anyone holding network access
to a kubo node's API (scoped by `service-kubo-api.yaml`'s `ClusterIP`-only
exposure + `networkpolicy.yaml`'s pod-selector restriction, both defense-in-
depth as stated above) and valid credentials to whatever fronts that API in
a real deployment (this chart provisions no API-level authentication of its
own — kubo's HTTP API has none built in) could `add` an unencrypted blob if
the *application* calling it didn't encrypt first. The guarantee this
project actually has is that the **one application that is wired to this
cluster** (`write-path-integration/ipfsclient`) does not have such a code
path today — that is a property of `ipfsclient.go`, verified by reading it,
not a property this cluster's configuration could ever enforce on a
different or future caller. Any future second caller of this cluster's API
would need its own equivalent encrypt-before-add discipline; nothing here
provides it for them automatically.

## Pinning-peer operator — `[ASSUMPTION]`, not ratified

ADR-0016 states: *"recommend a minimum of 2 cluster peers... both operated
by Org1 in this iteration... `TBD`: should `OrgClient-<tenantID>`... also
operate a pinning peer... The PRD does not settle this; flagged here rather
than decided silently."*

This artifact carries that flag forward **unchanged**: every StatefulSet,
Service, and Secret reference in this chart is written as a single
operator's deployment (one Helm release, one set of node identities, no
per-tenant pinning-peer template of the kind DEP-1's `OrgClient-<tenantID>`
peer template provides for Fabric peers). That is a direct structural
consequence of the ADR's own **recommendation**, not this chart independently
deciding the question ADR-0016 explicitly left open. If a future ADR
ratifies a second, tenant-operated pinning peer, that is a **new
StatefulSet/Service/Secret set** analogous to DEP-1's per-tenant peer
template — a genuine chart change, not a `values.yaml` toggle this chart
already anticipates, because the ADR gives no shape for what a
tenant-operated pinning peer's trust/network boundary should look like yet.
**`[ASSUMPTION]`** — recorded here exactly as ADR-0016 recorded it, not
upgraded to a decided fact by this item.

## Secrets

Two secrets, same purpose and same "never commit" rule as
`ipfs-cluster/swarm-key/{swarm.key,cluster-secret.txt}` in the dev cluster,
translated to K8s `Secret` objects that this chart **references by name but
never creates**:

```
kubectl create secret generic <release>-ipfs-swarm-key \
  -n <namespace> --from-file=swarm.key=./swarm.key
kubectl create secret generic <release>-ipfs-cluster-secret \
  -n <namespace> --from-file=cluster-secret.txt=./cluster-secret.txt
```

(Override the default names via `swarmKey.existingSecretName` /
`clusterSecret.existingSecretName` in `values.yaml` if a different naming
convention, or an external-secrets-operator-managed `Secret`, is preferred —
this chart only reads a name, it has no opinion on how that `Secret` object
was populated.) Generation of the key material itself — a libp2p swarm key
and an `ipfs-cluster` peer secret — is **out of this chart's scope**; use
whatever mechanism produced the dev cluster's own working
`ipfs-cluster/swarm-key/` files (not re-derived here, since this chart
never needs to read or regenerate that material, only reference a `Secret`
name). **`[VERIFY]`** the exact production key-generation/rotation procedure
against `security-architect`'s swarm-key custody policy (ADR-0016's own
follow-up: *"`security-architect`: ... own swarm-key custody/rotation
policy"*) before relying on this runbook for a real environment — that
policy is out of this item's lane.

## What this chart deliberately does NOT do

- Does not `helm install`/`kubectl apply` anything (standing scope decision).
- Does not fix the CRDT handshake defect (previous section) — offers a
  debugging lead and an accepted workaround, not a claimed fix.
- Does not decide the `OrgClient-<tenantID>` pinning-peer `TBD` (previous
  section) — carries ADR-0016's `[ASSUMPTION]` forward, does not resolve it.
- Does not create, generate, or rotate the swarm key / cluster secret.
- Does not add authentication in front of the kubo/ipfs-cluster HTTP APIs —
  neither image ships any; `NetworkPolicy` + `ClusterIP`-only Services are
  this artifact's only mitigation, both explicitly labeled defense-in-depth.
- Does not implement a re-pin/backfill job for nodes added after initial
  install (flagged in "Disclosed limitation" above as follow-up work).
- Does not modify `write-path-integration/ipfsclient/ipfsclient.go` to
  actually support an N-element endpoint list — that remains 2-element
  (`PrimaryAPI`/`ReplicaAPI`) today; generalizing it is flagged as follow-up
  work for whoever deploys `kubo.nodeCount > 2`, not done here (this item's
  scope is the deployment artifact, not the Go client).

## Confidentiality register

No real company/product name appears anywhere in this directory — checked
with the same case-insensitive extended-regex scan this repo's
confidentiality register requires for every touched file (the two banned
terms are deliberately not spelled out literally in this file, so that this
very compliance statement can never itself trip that scan). Zero hits;
the exact command and its empty output are recorded in this item's
structured report to the orchestrator, not duplicated here.

# `deploy/helm/fabric-network` — Fabric network Helm chart (DEP-1)

**Backlog item:** DEP-1 (`agent-suite/06-roadmap/implementation-backlog.md`) — "Helm chart — Fabric
network (Org1: 2 peers + 3 orderers; a per-tenant `OrgClient-<tenantID>` peer template) on the
platform's cloud K8s." Traces to ADR-0012 (three-org topology, Raft sizing), ADR-0013
(channel-per-tenant), ADR-0014 (in-band recording), `NET-DESIGN` §6 (Transport & deployment).

## Standing scope decision — read this first

This chart is a **generic, reviewable artifact only**, per this phase's binding scope decision
(human-confirmed 2026-08-07). It has **never been `helm install`-ed, `helm template`-d, or otherwise
run against any real or local (kind/minikube) Kubernetes cluster** — no cluster was created or
touched to produce it. It is not a proven deployment; it is a design document expressed as Helm
templates so it can be reviewed the way code is reviewed.

**Verification status, stated plainly:** `helm` is not installed in the environment this chart was
authored in, so `helm lint`/`helm template` could not be run even in dry-render form. Every template
in `templates/` was hand-traced for Go-template/YAML correctness (dict-passing shape, `range`
scoping, `quote`d ConfigMap values, `nindent` levels) but **not** mechanically validated by the real
tool. Run `helm lint deploy/helm/fabric-network` and `helm template deploy/helm/fabric-network` (both
fully offline, no cluster contact) before trusting this chart's YAML shape any further than "reviewed
by a person."

## What this chart translates, and from where

Every hostname, port, MSP ID, and env-var name below is copied from the actual source-of-truth files
in this repo, not invented:

- `fabric-network/network/compose/network-docker-compose.yaml` — Org1's 2 peers, the 3-node Raft
  orderer set, Org3's auditor peer, and (at the time this chart was written) `tenant01`'s and
  `tenant02`'s peers.
- `fabric-network/network/compose/ca-docker-compose.yaml` — `ca.org1`.
- `fabric-network/network/fabric-ca/ca-org1-config.yaml` — `ca.org1`'s server config content.

Both compose files carry their own extensive design-authority headers (ADR citations, empirically
resolved defects, `[VERIFY]`/`[FLAG]` markers). This README does not repeat that material — it only
records what changed in translating it to Kubernetes primitives, and why.

## Topology rendered

| Node(s) | Template file | K8s shape | Count |
|---|---|---|---|
| `peer0.org1`, `peer1.org1` | `templates/peers-org1.yaml` → `_helpers.tpl`'s `fabric-network.peer` | StatefulSet (1 replica) + headless Service + ConfigMap, each | 2 (fixed) |
| `orderer0.org1`, `orderer1.org1`, `orderer2.org1` | `templates/orderers-org1.yaml` → `fabric-network.orderer` | StatefulSet (1 replica) + headless Service + ConfigMap, each | 3 (fixed — Raft consenter set) |
| `ca.org1` | `templates/ca-org1.yaml` | StatefulSet (1 replica) + headless Service + ConfigMap | 1 |
| `peer0.org3` | `templates/peer-org3.yaml` → `fabric-network.peer` | StatefulSet + headless Service + ConfigMap | 1 |
| `peer0.<tenantID>` per tenant | `templates/peer-tenants.yaml` → `fabric-network.peer` | StatefulSet + headless Service + ConfigMap, **one set per `.Values.tenants` entry** | N (2 by default: `tenant01`, `tenant02`) |

**The per-tenant mechanism (the DoD's actual ask):** `templates/peer-tenants.yaml` is a `range` over
`.Values.tenants` that calls the *same* named template (`fabric-network.peer` in `_helpers.tpl`)
`templates/peers-org1.yaml` and `templates/peer-org3.yaml` also call. Onboarding a third tenant is a
`values.yaml` list append — no new template file, no hand-written third copy. `_helpers.tpl` computes
the tenant's k8s object name, legacy hostname, gossip self-bootstrap, and `OrgClient-<id>MSP` MSP ID
from the tenant's `id` field alone, so that naming convention lives in exactly one place
(`fabric-network.tenantMspID` and the `merge` block at the top of `peer-tenants.yaml`), not retyped
per tenant.

Org1's peer/orderer lists and Org3's peer list are implemented through the *identical* named-template
mechanism, for DRYness — **not** because their counts are meant to scale the same way. Org1's peer
count (2) and orderer count (3, the Raft consenter set already named in
`fabric-network/network/configtx/configtx.yaml`'s `EtcdRaft.Consenters`) are fixed by the ratified
topology (ADR-0012); editing `values.yaml`'s `org1.orderers` list does not, by itself, add or remove a
live Raft consenter — that requires a channel config update transaction (NET-6's scope).

## Deliberate deviations from a literal line-for-line compose translation

Each is called out again, in place, in the relevant template/values comment — collected here for a
reviewer who wants the full list in one place:

1. **Image tags pinned to an exact patch, not the compose files' floating tags.** Both source compose
   files flag this themselves (`[VERIFY] pin to an exact patch tag ... before any non-throwaway/CI
   use`) — this chart acts on that flag: `2.5.16` for peer/orderer (per
   `fabric-skill-suite/skills/fabric-core/references/version-matrix.md`'s recorded latest-pinned-corpus
   patch), `1.5.15` for CA (per `ca-org1-config.yaml`'s own `version:` field). Re-verify against
   upstream before real use.
2. **Service ports use the compose file's *internal* container ports, not its host-published ports.**
   `network-docker-compose.yaml` publishes distinct host ports per node (7051/8051/9051/10051 for
   peers, 7050/8050/9050 for orderers, etc.) purely to avoid **one Docker host** binding the same port
   twice across sibling containers. That constraint does not exist in Kubernetes — every Pod has its
   own IP, so every peer can (and does, in this chart) listen on the *same* internal port (7051 grpc,
   9443 operations) with zero collision, addressed instead by its own ClusterIP Service name. Copying
   the host-port numbers literally into K8s `Service.ports` would misrepresent a Docker-single-host
   workaround as if it were a Fabric requirement. `7052` (chaincode-listen) is declared as a
   `containerPort` but deliberately **not** exposed as a Service port, mirroring
   `network-docker-compose.yaml`'s own "NOT IN SCOPE HERE" reasoning (no chaincode-as-a-service
   workload exists yet at this build phase; CC-5's scope).
3. **Cross-org TLS CA mount set generalized to the full tenant list, not copied verbatim.**
   `network-docker-compose.yaml`'s own header states the *intent* — "each PEER's clientRootCAs is set
   to the UNION of all three peer-org TLS CAs" — but its actual per-node mount lists have drifted from
   that intent (Org1's two peers mount `{org1, tenant01, org3}`, missing `tenant02` entirely; each
   tenant peer mounts only `{org1, <self>, org3}`, never another tenant). A per-tenant Helm template
   that copied the drifted, tenant01-only snapshot would not scale to N tenants — exactly the defect
   this item exists to avoid. `_helpers.tpl` computes the union structurally instead: every peer this
   chart renders (Org1's, Org3's, and every tenant's alike) mounts the identical full union
   `{org1, org3, every entry in .Values.tenants}`; orderers mount that same union plus OrdererMSP's
   own TLS CA. Per the compose file's own "RESOLVED 2026-08-06" empirical finding, none of this is
   currently *read* by any working env var on the real image (the multi-value
   `CORE_PEER_TLS_CLIENTROOTCAS_FILES` array env var does not parse; cross-org mTLS trust is sourced
   from channel-embedded MSP config once a channel exists, not a static core.yaml field) — these
   mounts are kept for source-file parity and against a future config-file-based trust reference, not
   because they are known to do anything today.
4. **CA bootstrap admin password removed from the mounted config file.** `ca-org1-config.yaml`
   embeds the bootstrap admin's password in plaintext under `registry.identities` (disclosed there,
   by that file's own header, as a redundant "belt-and-suspenders" duplicate of the `-b` CLI flag).
   This chart's `ca.org1` ConfigMap (`templates/ca-org1.yaml`) omits that block entirely
   (`identities: []`) and sources the bootstrap identity **only** from a Secret reference
   (`org1.ca.bootstrapAdminSecretName`, keys `username`/`password`), injected via `$(...)` substitution
   into the container's `-b` flag at start — never written to any file this chart authors. Dropping the
   already-redundant, secret-bearing half of a documented duplication is a safe simplification, not a
   functional regression.
5. **The CA's `operations.listenAddress` is left at the source file's loopback-only value.**
   `ca-org1-config.yaml` binds CA operations to `127.0.0.1:9443`, which a Kubernetes `httpGet` probe
   (executed against the Pod IP, not `localhost` inside the container) cannot reach. Rather than
   silently widen that binding — a config-content decision that belongs to whichever item owns
   `ca-org1-config.yaml` (NET-2), not this chart's translation of it — this chart simply does not wire
   a liveness/readiness probe for the CA StatefulSet. Peers and orderers *do* get `httpGet` probes
   against their operations endpoints, since the compose file's own `CORE_OPERATIONS_LISTENADDRESS=
   0.0.0.0:9443` / `ORDERER_OPERATIONS_LISTENADDRESS=0.0.0.0:8443` are already reachable from outside
   the container — and a kubelet `httpGet` probe runs from the kubelet process, not from a client
   binary inside the container, so it does not hit the "base image ships no HTTP client" limitation
   `network-docker-compose.yaml`'s own "NOT IN SCOPE HERE" section cites for why it added no Docker
   Compose `healthcheck:` blocks.
6. **PVC + ConfigMap co-mount for the CA, translating a real empirical finding.**
   `ca-docker-compose.yaml`'s header records that `fabric-ca-server` writes all runtime state relative
   to the *config file's own directory*, not `FABRIC_CA_HOME` — fixed there by bind-mounting the
   config file inside the named volume's own mount path. The direct Kubernetes equivalent of "a
   read-only single-file mount coexisting with sibling files the container writes into the same named
   volume" is a ConfigMap volume mounted via `subPath` onto one file inside the PVC's mount path
   (see `templates/ca-org1.yaml`) — not a naive "mount the whole ConfigMap over the whole PVC
   directory," which would hide everything the CA itself needs to write there.

## Secret material — the contract an operator must satisfy

This chart **creates no Secret and contains no private key, password, or certificate byte anywhere**.
Every `values.yaml` field ending in `SecretName` is a reference an operator populates out of band
(DEP-2's scope: "Identity/TLS/HSM provisioning + cert-rotation runbook"). `secrets.mode` in
`values.yaml` chooses how those references resolve:

- `k8sSecret` (default) — a plain Kubernetes `Secret`, already present in the release namespace.
- `csiExternalSecret` — the same name is instead read as a `SecretProviderClass` for the [Secrets
  Store CSI Driver](https://secrets-store-csi-driver.sigs.k8s.io/) (HSM/PKCS11- or Vault-backed) —
  this is the option ADR-0019 Domain A's "Fabric MSP signing keys target HSM/PKCS11 in production"
  requirement maps onto; this chart never talks to an HSM itself.

**Required keys per Secret, and the exact path each is projected to inside the pod** (see
`fabric-network.credentialVolumeSource` / the `msp`/`tls` volumes in `_helpers.tpl`):

| Secret role | Key (flat, no `/` — a K8s Secret key name constraint) | Mounted at |
|---|---|---|
| `*MspSecretName` | `signcert` | `msp/signcerts/cert.pem` |
| | `keystorekey` | `msp/keystore/key.pem` |
| | `cacert` | `msp/cacerts/ca-cert.pem` |
| | `tlscacert` | `msp/tlscacerts/tlsca-cert.pem` |
| | `admincert` | `msp/admincerts/admin-cert.pem` |
| | `configyaml` | `msp/config.yaml` (NodeOU config — required; NodeOUs are enabled per `fabric-network-design.md` §1) |
| `*TlsSecretName` | `tlscert` | `tls/server.crt` |
| | `tlskey` | `tls/server.key` |
| | `tlscacert` | `tls/ca.crt` |
| `org1.ca.bootstrapAdminSecretName` | `username`, `password` | consumed as env vars, never mounted as files |

**Why the MSP Secret needs a repackaging step, and why the TLS Secret doesn't.** The existing
`fabric-network/network/crypto-config/` tree (generated by `cryptogen` at NET-1) already has fixed,
predictable filenames for TLS material (`tls/server.crt`, `tls/server.key`, `tls/ca.crt` — see e.g.
`crypto-config/peerOrganizations/org1/peers/peer0.org1/tls/`), so a `*TlsSecretName` Secret's
`tlscert`/`tlskey`/`tlscacert` keys map onto those files with no renaming. The MSP directory's
`keystore/` file, by contrast, is named by `cryptogen` after the key's own SKI hash (unpredictable,
different per node) — a Helm template cannot hardcode a target filename it cannot know in advance.
This chart's contract therefore fixes the *target* name (`keystorekey` → `msp/keystore/key.pem`) and
leaves the *rename-to-that-fixed-name* step to whatever process populates the Secret (DEP-2's
provisioning pipeline) — that is a provisioning-shape decision, not something a consuming chart should
invent.

**The cross-org TLS CA bundle (`crossTLSCAConfigMapName`, default `<fullname>-cross-tlsca`) is
modeled as a ConfigMap, not a Secret** — deliberately: its content is each org's/tenant's own TLS CA
*root certificate*, public material by design (that is what makes it useful as a trust anchor), not a
private key. Treating it as a Secret would misstate its confidentiality class. Like every
`*SecretName` above, this chart does not create it — an operator populates keys `org1`, `org3`, one
key per tenant `id`, and `ordererorg` (orderers only), each holding that org's `tlsca-*.pem` bytes.

## Known, unresolved gap: TLS SAN vs. Kubernetes DNS naming

`CORE_PEER_ADDRESS`, `CORE_PEER_GOSSIP_EXTERNALENDPOINT`, and `CORE_PEER_GOSSIP_BOOTSTRAP` in every
peer's ConfigMap use the **legacy docker-compose hostname verbatim** (e.g. `peer0.org1:7051`), not a
Kubernetes-native `<service>.<namespace>.svc.cluster.local` name. This is deliberate, not an oversight
— but it is also **not a solved problem**, and this chart does not pretend otherwise:

- The pre-issued TLS certificates in `fabric-network/network/crypto-config/` were generated by
  `cryptogen` with SANs matching the *docker-compose* hostnames (`peer0.org1`, `peer0.tenant01`, …).
  If this chart's Secrets simply re-package that same, already-issued material (the path of least
  resistance for a first migration), the peer's own `CORE_PEER_ADDRESS`/gossip identity strings must
  still say `peer0.org1`, or the peer would advertise an address its own certificate's SAN does not
  cover.
- Kubernetes Service DNS names are never bare hostnames like `peer0.org1` — they are always
  `<service-name>.<namespace>.svc.cluster.local` (or `<service-name>` unqualified, within the same
  namespace, still not `peer0.org1` unless the Service itself were literally named `peer0.org1`, which
  is not a legal Kubernetes object name — dots are not permitted in `metadata.name`).
- **This chart does not resolve that mismatch.** Doing so requires one of: (a) DEP-2 re-issuing certs
  with SANs matching a real in-cluster FQDN before this chart could actually be applied, (b) a
  cluster-DNS-level alias (e.g. CoreDNS custom stub domains/rewrite rules mapping `peer0.org1` to the
  Service's real FQDN), or (c) accepting `peer0.org1` as a literal `Pod.spec.hostname`/`subdomain`
  pairing under a headless Service also literally named to match — itself constrained by the "no dots
  in object names" rule above and so, at best, a partial fix. None of these is chosen here; this is
  named explicitly so a reviewer does not read "this chart mounts the real crypto-config Secrets" as
  "this chart would therefore actually gossip successfully if applied." It would not, without one of
  the above being resolved first — squarely DEP-2's identity/cert-rotation scope, not re-decided here.

## Why only one CA template (Org1's)

`ca-docker-compose.yaml` stands up three CA servers (`ca.org1`, `ca.tenant01`, `ca.org3`), but this
chart renders only `ca.org1`. This follows directly from `NET-DESIGN` §6 ("Transport & deployment"):
*"Each `OrgClient-<tenantID>` peer is, by definition (ADR-0012), operated by the client itself — it
does not sit inside the platform's own deployment boundary"* — if the tenant's *peer* is outside the
platform's own Kubernetes deployment boundary, so, a fortiori, is the tenant's own CA; a per-tenant CA
template belongs in whatever the client organization's own deployment tooling is, not in this
platform-operated chart. Org3's CA is excluded for a related but distinct reason: its operator is an
explicit, unratified **TBD** (`ADR-0012`, gap **G-04** residual) — this chart cannot decide, on its
own authority, that the platform operates Org3's CA when that question is still open. If G-04 closes
in favor of platform operation, adding an `org3.ca` block that reuses the exact same
`fabric-network.peer`-sibling pattern `ca-org1.yaml` already establishes is a small, mechanical
follow-up — not a redesign.

## Confirmation: no new deployable unit for the in-band recording component (`REC-*`)

**This chart contains no anchor-service, no Kafka-consumer workload, and no Deployment/StatefulSet/
CronJob of any kind for `REC-*`.** ADR-0014 retired the earlier per-event anchor-service design
(ADR-0006) precisely because it required a *separately-deployed* Go service consuming a Kafka topic —
ADR-0014's own Decision section states this in as many words: *"Anchor-service deployable units: 0.
No new, separately-deployed Go service is introduced to host the Gateway client and a consume→anchor
loop."* The in-band recording component instead ships **inside the existing HRIS write-path's own
deployables** — whichever platform/process hosts that write path (a question ADR-0014 itself leaves
deliberately open, its own reopened assumption **G-10**, not this chart's to resolve). Since this
chart's job is Fabric network infrastructure (peers/orderers/CA), and the write path's own deployable
is, by ADR-0014's decision, not a new or separate thing this phase introduces, there is correspondingly
nothing under `templates/` here for it. This directly contrasts with the retired anchor-service design,
which this repo has none of anywhere.

## Explicitly not in this chart

Mirrors `network-docker-compose.yaml`'s own "NOT IN SCOPE HERE" boundary, plus this item's own DoD
note:

- No CLI/admin/tools Pod or Job (`peer channel join`, `osnadmin channel join`, chaincode lifecycle —
  NET-3/NET-5/CC-5's scope).
- No chaincode / external-builder / `docker.sock`-equivalent mount.
- No CouchDB (ADR-0007 ratifies LevelDB — `CORE_LEDGER_STATE_STATEDATABASE: "goleveldb"` in every
  peer's ConfigMap is this chart's compliance evidence, not a re-decision).
- No per-tenant or Org3 Fabric CA (see "Why only one CA template" above).
- No IPFS / `ipfs-cluster` workload (DEP-3's own chart, ADR-0016).
- No anchor-service / Kafka-consumer workload for `REC-*` (see confirmation above).
- No NetworkPolicy, HPA, PodDisruptionBudget, or ServiceMonitor — none were asked for by this item's
  DoD; adding them without a citation would be scope invention, not a translation of anything.

## Resource defaults

`values.yaml`'s `resources.{peer,orderer,ca}` are generic, sane-default requests/limits, **not
measured** against real load. `QA-4` (Hyperledger Caliper, 200/500/1000/2000 TPS against **P2**) is
the item that eventually produces real throughput/latency numbers; this chart does not anticipate that
result and should be revisited once QA-4 lands.

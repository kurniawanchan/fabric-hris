# Observability — dashboards + alerts (DEP-5)

Backlog item: **DEP-5** (`agent-suite/06-roadmap/implementation-backlog.md`). Design authority:
`agent-suite/00-architecture/solution/integration-design.md` §7 ("Open availability question"),
`agent-suite/08-security/security-architecture.md` T12 (orderer-availability threat row). Depends
on **DEP-1** (the Helm chart whose pods this item's scrape targets/exporters would run alongside —
DEP-1 owns turning `CORE_METRICS_PROVIDER`/`ORDERER_METRICS_PROVIDER` on in a real deployment's
Helm values; this item does not re-litigate that, see §2 below).

**Standing scope decision (binding, human-confirmed 2026-08-07).** Everything in this directory is
a **GENERIC ARTIFACT** — a dashboard definition and an alerting-rules file to review, not something
applied against a real Grafana/Prometheus/cloud target from this workspace. No Grafana or
Prometheus instance runs anywhere in this repo today. Grafana JSON and Prometheus alerting-rules
YAML are used because they are the concrete, common reference format for this kind of artifact —
using them here is not a claim that this repo runs Grafana or Prometheus, only that "a dashboard
JSON a real Grafana could import" and "a rules file `promtool check rules` could lint" are more
reviewable than prose describing the same panels/alerts abstractly.

**Confidentiality register (standing constraint):** no real company/product name appears anywhere
in this directory — `tenant01`/`tenant02`/"Generic Company"-style placeholders only, matching every
other artifact in this repo.

---

## 1. What actually exists today vs. what this item assumes — read this before the panel/alert detail

Every panel and alert below either (a) queries a **real, already-documented Fabric Prometheus
metric** that the live network could emit once metrics are turned on, or (b) queries a **metric
name this item itself defines** as the target contract for a producer that does not exist in this
codebase yet. Both classes are marked explicitly in §3 — conflating them would misrepresent this
item's own DoD ("say so" instruction, task brief). The table below is the reality check that makes
that marking meaningful:

| Fact | Current state | Source |
|---|---|---|
| Fabric metrics provider | **`disabled`** on every peer and orderer in the live network (`CORE_METRICS_PROVIDER=disabled`, `ORDERER_METRICS_PROVIDER=disabled`) | `fabric-network/network/compose/network-docker-compose.yaml` (every peer/orderer service block) |
| Fabric operations service (health/logspec) | **Running today**, unauthenticated — no `CORE_OPERATIONS_TLS_ENABLED`/`ORDERER_OPERATIONS_TLS_ENABLED` env var is set, so per `operations_service.rst`'s own text ("When TLS is disabled, authorization is bypassed and any client that can connect... will be able to use the API") `/healthz` is reachable with zero config today, but so is `/logspec` (log-level **mutation**) — a real hardening gap, not this item's to close (flag for DEP-1/DEP-2) | `fabric-skill-suite/corpus/fabric-docs-2.5/operations_service.rst`; ports below |
| Operations ports reachable today | peers: `9444` (peer0.org1), `9445` (peer1.org1), `9446` (peer0.tenant01), `11446` (peer0.tenant02), `9447` (peer0.org3); orderers: `7070`/`8070`/`9070` (orderer0/1/2.org1) | `fabric-network/network/compose/network-docker-compose.yaml` |
| Identity path the live write path actually authenticates with | **cryptogen**-issued `Admin@<orgdir>` identities (`crypto-config/peerOrganizations/<orgdir>/users/Admin@<orgdir>/...`) — **not** the parallel Fabric-CA-enrolled `hris-user1` test identity, which the running network never wires into the write path | `write-path-integration/gateway-client/*_integration_test.go`; `fabric-network/network/compose/network-docker-compose.yaml`'s own "IDENTITY-MATERIAL SOURCE" header; `deploy/provisioning/identity-tls-hsm-provisioning.md` §1 |
| `INT-5`'s partial-failure metric | **Does not exist as a metric yet.** `Hooks.OnPartialFailure` is "a plain, caller-injectable Go function rather than a hardcoded metrics dependency... this workspace has no metrics/observability stack wired up yet (that is DEP-5's separate, later scope, and this callback is its future integration point)" | `write-path-integration/writepaths/writepaths.go`, `Hooks.OnPartialFailure`'s own doc comment |
| Write-path anchor-latency measurement | **Does not exist.** No timer wraps `Hooks.anchor`/`doAnchor` anywhere in this codebase; NFR-9's `<500ms` budget has never been measured against real traffic | `write-path-integration/writepaths/writepaths.go`; PRD §3 NFR-9 ("ambang awal — kalibrasi ulang setelah Caliper §10.4 butir 3 berjalan") |
| Cert-expiry signal | **Does not exist.** No exporter or cron scans any of the cert paths below today; producing that signal is `DEP-2`'s scope ("cert-expiry monitoring + rotation runbook"), not this item's — this item defines the panel/alert shape DEP-2's producer must feed | `agent-suite/06-roadmap/implementation-backlog.md` DEP-2 row; `deploy/provisioning/identity-tls-hsm-provisioning.md` |

Turning the "does not exist yet" rows into real signals means: (1) DEP-1/a real deployment's Helm
values flip `metrics.provider: prometheus` (equivalently `CORE_METRICS_PROVIDER=prometheus` /
`ORDERER_METRICS_PROVIDER=prometheus`) on every peer/orderer — a one-line config change, not new
code; (2) whoever embeds `write-path-integration/writepaths` in the real HRIS process constructs
`Hooks.OnPartialFailure` as a closure that increments the counter in §3 (the callback's own
signature carries `profileSection`, not `tenantID` — see the note under that metric); (3) DEP-2
ships a small cert-expiry exporter/cron reading the leaf-cert paths listed in §3.2. None of that
code is written here — this item only fixes the contract those three pieces of future work must
honor, and the dashboard/alerts that consume it.

---

## 2. Metric contract (§3 marks each LIVE vs. NOT-YET-IMPLEMENTED)

### 2.1 Real Fabric Prometheus metrics (LIVE once `metrics.provider: prometheus` is set)

Grounded against the actual corpus mirror, not recalled from memory:
`fabric-skill-suite/corpus/fabric-docs-2.5/metrics_reference.rst`.

| Metric | Type | Labels (Fabric-emitted) | Node | Meaning here |
|---|---|---|---|---|
| `ledger_blockchain_height` | gauge | `channel` | peer | Per-channel ledger height — a peer stuck at a lower height than its channel peers on the same tenant channel is the concrete "peer unhealthy per channel" signal |
| `gossip_state_height` | gauge | `channel` | peer | Corroborates `ledger_blockchain_height` from the gossip state-transfer view |
| `participation_status` | gauge | `channel` | orderer | `0` inactive / `1` active / `2` onboarding / `3` failed, **per channel, per orderer node** — the direct "is this orderer serving this tenant channel" signal |
| `participation_consensus_relation` | gauge | `channel` | orderer | `0` other / `1` consenter / `2` follower / `3` config-tracker |
| `consensus_etcdraft_active_nodes` | gauge | `channel` | orderer | Live Raft-consenter count for the channel — **the residual-risk panel's core metric, §4.3** |
| `consensus_etcdraft_cluster_size` | gauge | `channel` | orderer | Configured Raft-consenter count for the channel (fixed at `3`, ADR-0012) |
| `consensus_etcdraft_is_leader` | gauge | `channel` | orderer | `1` if this node is the Raft leader for the channel |
| `consensus_etcdraft_leader_changes` | counter | `channel` | orderer | Cumulative leader elections — a churn signal independent of quorum loss |
| `up` | gauge | `job`, `instance` (Prometheus-added, not Fabric-emitted) | both | Standard scrape-target liveness |

None of these carry an `org`/`tenant_id`/node-name label from Fabric itself — Fabric's metrics are
per-process, distinguished only by whatever `job`/`instance`/static `labels:` a scrape config
attaches (see the illustrative snippet in §4.3). Do not write a query assuming `ledger_blockchain_height{org="Org1"}` exists as a Fabric-native label — it does not.

### 2.2 New metrics this item defines (NOT YET IMPLEMENTED — target contract only)

| Metric | Type | Labels | Producer (owned by) | Grounding |
|---|---|---|---|---|
| `writepath_partial_failure_total` | counter | `tenant_id`, `profile_section` | Whoever constructs `Hooks{OnPartialFailure: ...}` in the real embedding process (not this repo's demo) | Increments exactly once per `Hooks.OnPartialFailure` invocation — i.e., exactly when `anchor()` is "about to return a `*PartialFailureError`" (`writepaths.go` comment on `OnPartialFailure`). **`employeeInternalID` and `version` are deliberately excluded as labels** — both are per-record, unbounded-cardinality, and `employeeInternalID` is exactly the kind of identifier this system otherwise keeps off any observability surface as a matter of posture (INV-1); they belong in the error/log payload the callback also has access to, not in a metric label. |
| `writepath_anchor_duration_seconds` | histogram | `tenant_id`, `profile_section`, `outcome` (`success`\|`partial_failure`) | Same producer as above, wrapping the call to `Hooks.anchor` | Measures **`anchor()`'s own wall-clock duration** — the compute-salt/digest + `SubmitRecordProfileSection` sequence. This is a proxy for, not identical to, NFR-9's "<500ms added vs. no anchoring" framing: NFR-9 is a *delta* against a no-anchoring baseline, and per ADR-0014 anchoring is purely additive time appended after the operational-DB save, so this histogram's value is that delta directly **as long as nothing else changes concurrently in the write path** — a caveat worth stating, not hiding. |
| `fabric_cert_not_after_seconds` | gauge | `org`, `tenant_id` (empty for non-per-tenant orgs), `cert_type` (`admin-signcert`\|`peer-signcert`\|`peer-tls`\|`orderer-signcert`\|`orderer-tls`\|`tls-ca`), `identity` | DEP-2's cert-expiry exporter/cron (this item does not build it — see §1) | Unix timestamp of the leaf cert's `NotAfter`. Path targets for the currently-running network (cryptogen-issued, see §1's identity-path row): `netDir + "/crypto-config/peerOrganizations/<orgdir>/users/Admin@<orgdir>/msp/signcerts/Admin@<orgdir>-cert.pem"` (the **operationally most load-bearing** one — this is what the live Gateway client authenticates as; if it expires, the first visible symptom is a spike in `writepath_partial_failure_total`, not a metric on this exporter, unless this exporter is actually wired in first) and the parallel peer/orderer/TLS-CA paths under the same `crypto-config` tree. |

`writepath_anchor_attempts_total` (a companion counter needed to express the third alert category
as a **fraction** of attempts, not an absolute count) is deliberately **not** defined here — nothing
in `writepaths.go` counts total anchor attempts today, and inventing a metric with no code path
producing it would not be "grounding the panel in the real signal shape this codebase actually
produces" (task brief). The alert in `alerts/alert-rules.yaml` uses an absolute-count threshold on
`writepath_partial_failure_total` alone for exactly this reason — see that file's own comments.

---

## 3. Dashboard — `dashboards/fabric-hris-observability.json`

A Grafana dashboard-JSON export (schema-compatible with Grafana's "export for sharing externally"
shape, `__inputs`/`${DS_PROMETHEUS}` datasource placeholder included) with four panel groups
mapping 1:1 to the task's four lettered requirements:

**(a) INT-5 partial-failure signal.** Rate-by-`profile_section` and rate-by-`tenant_id` timeseries
plus a 1-hour rolling count, all against `writepath_partial_failure_total` (§2.2) — grounded in
`OnPartialFailure`'s actual call site and label shape, not a generic "error rate" panel.

**(b) Cert expiry across all three orgs + per-tenant enrollments.** A table of
`(fabric_cert_not_after_seconds - time()) / 86400` ("days until expiry"), one row per
`(org, tenant_id, cert_type, identity)`, thresholds colored at the same 30/7-day boundaries as the
alert rules (§4), covering Org1, Org3, and every provisioned `OrgClient-<tenantID>` — not just
`tenant01`.

**(c) Per-channel peer/orderer health, every provisioned tenant channel.** `ledger_blockchain_height`
per peer per channel, `participation_status`/`participation_consensus_relation` per orderer per
channel, `up` for node-level liveness, and — the panel this item's own DoD calls out by
name — `consensus_etcdraft_active_nodes` vs. `consensus_etcdraft_cluster_size`, the **Raft residual-
risk panel** (§4.3 below). The dashboard's `$tenant` template variable defaults to
`tenant01, tenant02` (the two channels that exist today, per `NET-7`) but is a free-text/custom
variable, not hardcoded to two values — a real deployment onboarding tenant `N+1` (ADR-0013's own
documented `O(N)` fan-out) extends it by adding a value, not by editing panel queries.

**(d) Write-path anchor latency (NFR-9).** `histogram_quantile` p50/p95/p99 of
`writepath_anchor_duration_seconds` with a fixed 500ms reference line. **This panel has no data
source today and is not expected to until `QA-4`'s Caliper harness runs or real production traffic
exists** — the dashboard's own top-of-section text panel says this explicitly (not hidden in this
README alone) so a reviewer opening the dashboard directly sees the same caveat.

---

## 4. Why one Raft-health panel gets its own paragraph — the residual risk this item must surface, not resolve

**ADR-0012's own words:** *"All 3 Raft consenter nodes are Org1-operated... Org1 can halt ordering
on every channel it services (which, under channel-per-tenant, is every tenant channel — ADR-0013)
by itself, with no action from Org2 or Org3 required or possible to prevent it."* Quorum is
`2` of `3`; crash-fault tolerance is `f = floor((3-1)/2) = 1`. `security-architecture.md`'s T12 row
records this as **"Downgraded from accepted-SPOF to mitigated, tolerates 1 failure; >1 concurrent
failure remains residual"** — i.e. genuinely improved by the 3-node Raft topology, but not
eliminated: a second concurrent Org1-orderer failure still halts ordering on **every** tenant
channel simultaneously, and only Org1 can cause or fix that, structurally, regardless of how many
tenants or orgs exist.

**This item's job is to make that fact observable, not to make it stop being true.** The dashboard's
`consensus_etcdraft_active_nodes` vs. `consensus_etcdraft_cluster_size` panel (§3c) and the
`FabricRaftQuorumDegraded`/`FabricRaftQuorumCritical` alerts (§5) exist specifically because this
residual risk is accepted, not mitigated, by DEP-5 — a dashboard that quietly rolled this into a
generic "orderer up/down" check would let the specific, accepted, Org1-concentrated failure mode
(2-of-3 nodes down = **every** channel halts, not just one) hide behind an aggregate that looks
fine as long as *some* orderer process is running. Per-channel `active_nodes < cluster_size`
detects the degraded-but-still-quorate case (1 node down, tolerated) distinctly from
`active_nodes < 2` (quorum lost, ordering halted network-wide) — the alert rules encode exactly
that distinction (§5.2).

---

## 5. Alerts — `alerts/alert-rules.yaml`

**Format choice: Prometheus alerting-rules YAML, not `alert-rules.md`.** Justification: (1) it is
lintable (`promtool check rules alerts/alert-rules.yaml`) — a reviewer can mechanically verify the
PromQL parses and the YAML is well-formed, which prose cannot offer; (2) it is the format Grafana's
own alerting can import/provision directly, keeping the dashboard (§3) and the alerts in the same
toolchain family rather than describing rules a real Prometheus/Grafana pair would have to be
hand-translated from prose; (3) it matches this repo's own pattern of using the ecosystem's native
config-file format as the reviewable artifact (Helm `values.yaml`, `configtx.yaml`, `core.yaml`
env-vars) rather than a bespoke markdown table. `promtool` was not available in this workspace to
run (`which promtool` → not found) — the file was hand-validated for well-formed YAML via Ruby's
stdlib `YAML.load_file` instead (see the report accompanying this item); it has not been checked
against a real Prometheus/Alertmanager, consistent with this item's generic-artifact scope.

### 5.1 Cert expiry (§2.2's `fabric_cert_not_after_seconds`)

Two severities, thresholds at **30 days** (warning) and **7 days** (critical) — a conventional,
explicit choice (task brief's own suggested values), not derived from a ratified requirement; no
PRD/ADR fixes a cert-rotation SLA number, so this is stated as a policy default DEP-2's own
`cert-rotation-runbook.md` (referenced but not yet written as of this item, per
`deploy/provisioning/identity-tls-hsm-provisioning.md` §1) is free to supersede.

### 5.2 Channel peer/orderer health

`FabricPeerDown`/`FabricOrdererDown` (plain `up == 0`), `FabricChannelParticipationDegraded`
(`participation_status` not `1`), and the two Raft-quorum rules from §4:
`FabricRaftQuorumDegraded` (warning — `active_nodes == cluster_size - 1`, i.e. exactly the tolerated
1-node loss) and `FabricRaftQuorumCritical` (critical — `active_nodes < 2` for a `cluster_size` of
`3`, i.e. quorum lost, ordering halted on every channel this orderer set serves — annotated with
the ADR-0012 citation directly in the alert, not left for a responder to go find it).

### 5.3 INT-5 partial-failure rate

Two rules, both against `writepath_partial_failure_total` alone (§2.2 explains why no
attempts-denominator rule exists yet): `WritePathPartialFailureDetected` (info/warning — **any**
occurrence in 15 minutes fires, because `PartialFailureError`'s own doc comment establishes there is
**no reconciliation job** to catch a missed one — "at minimum surfaced as an error/metric" per
`INTEG` §7/SEC T6b means every occurrence is actionable, not just a rate) and
`WritePathPartialFailureRateElevated` (critical — a placeholder absolute-count threshold over 5
minutes, explicitly labeled `[ASSUMPTION — placeholder, pending QA-4 calibration]` in the rule's own
annotation, mirroring the PRD's own NFR-9 convention: *"ambang awal — kalibrasi ulang setelah
Caliper §10.4 butir 3 berjalan"* — the same "provisional numeric floor, recalibrate once real
throughput numbers exist" pattern, applied to this alert's threshold for the same reason).

**No latency-breach alert exists** (deliberately, not an oversight): `writepath_anchor_duration_seconds`
has no producer yet (§1/§2.2) and no measured baseline (NFR-9's own recalibration note) — an alert
threshold on a metric with no current data source would either never fire (silently, providing false
assurance) or fire spuriously on "no data" depending on the alerting engine's null-handling, neither
of which is worth shipping over a panel already labeled explicitly as a placeholder (§3d).

---

## 6. Explicit non-scope

- **IPFS cluster health** (kubo nodes, `ipfs-cluster`'s own disclosed CRDT-peering defect) is
  **not** covered by this item — it is outside DEP-5's named DoD (cert expiry, peer/orderer health
  per channel, per-tenant channel health, `INT-5`'s signal) and belongs to `DEP-3`'s scope if it is
  ever picked up as a dashboard target. Not omitted by oversight — omitted because the task brief's
  DoD does not name it and this item does not expand its own scope unilaterally.
- **CI/build health** (`DEP-4`) and **Caliper performance results** (`QA-4`/`QA-6`) are separate
  dashboards/reports by design — this item's panel (d) only reserves the *shape* the latter will
  eventually populate, it does not attempt to run or simulate a load test.
- This item does **not** flip `metrics.provider` in any Helm values file, write the cert-expiry
  exporter, or add an `OnPartialFailure` closure to any real process — those are DEP-1/DEP-2/a real
  embedding process's work respectively, tracked in §1's table, not silently absorbed into this
  item's scope.

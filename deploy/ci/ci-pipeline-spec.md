# CI/CD Pipeline Specification — Chaincode + Write-Path Integration

- **Backlog item:** `DEP-4` (`agent-suite/06-roadmap/implementation-backlog.md` row `DEP-4`) — "CI/CD —
  chaincode (`CC-*`) + write-path-integration (`REC-*`/`INT-*`) build/test/image, lint, SAST, deploy
  gate." Depends on `CC-1` (chaincode skeleton — done) and `REC-1` (digest builder — done), both
  already merged, so every command below is written against **real, existing** files, not a
  hypothetical future layout.
- **Authorities cited:** `ADR-0020` (chaincode-contract-hashing-boundary.md — the four-function
  contract this pipeline builds/tests/images); `SEC T13`
  (`agent-suite/08-security/security-architecture.md` — "ABAC bypass / missing authz in the
  profile-record chaincode... mitigated via... **semgrep/Guardian** SAST"); `TEST ST-9`
  (`agent-suite/06-roadmap/test-strategy.md` — "SAST gate: semgrep/Guardian clean on the chaincode +
  the HRIS profile-write path").
- **Standing scope decision (2026-08-07, binding):** this whole `DEP-*` phase is **generic artifacts
  only**. Nothing in this directory has been executed against a real CI provider, a real registry, or
  the live network described below. The two files here are reviewable specifications, not evidence of
  a pipeline run.
- **No CI provider is chosen in this repository.** This document is deliberately tool-agnostic — it
  specifies *stages*, their *order*, their *gates*, and the exact commands each stage runs, so it can
  be transcribed into GitHub Actions, GitLab CI, Jenkins, or anything else without re-deriving the
  content. `github-actions-reference.yml` alongside this file is **one concrete transcription**, not a
  claim that this repo uses GitHub Actions — see that file's own header.
- **Confidentiality register:** no real company/product name appears below; `tenant01`/`tenant02` are
  the same generic placeholder tenant IDs already used throughout `fabric-network/` and
  `write-path-integration/`.

## 0. What this pipeline covers, and what it explicitly does not

| In scope (this item) | Out of scope (other items) |
|---|---|
| Lint, SAST, unit tests, integration tests, chaincode CCaaS image build, and the **gate** that must go green before any deploy step runs | The Helm chart the deploy step would eventually invoke — `deploy/helm/fabric-network` (`DEP-1`) and `deploy/ipfs-cluster` (`DEP-3`) — those charts' own content is not authored here |
| The chaincode module (`fabric-network/chaincode/employeeprofilerecord`) | Identity/TLS/HSM provisioning and cert-rotation (`DEP-2`, `deploy/provisioning/`) |
| All four `write-path-integration` Go modules | Observability — dashboards/alerts on pipeline or runtime health (`DEP-5`) |

The **deploy gate** stage (§6) is a pass/fail checkpoint, not a deploy implementation: per the standing
scope decision above, this spec does not perform, and its reference implementation does not contain,
any `helm install`/`helm upgrade`/`kubectl apply`/`terraform apply` invocation against a real cluster.

## 1. Module topology this pipeline must respect

Two independently-versioned Go trees, plus one CCaaS image target:

```
fabric-network/chaincode/employeeprofilerecord/   # single Go module, go.mod "employeeprofilerecord",
                                                   # go 1.21, vendor/ present (vendor/modules.txt) —
                                                   # go build/vet/test auto-select vendor mode here,
                                                   # no extra flag needed (verified: `go build ./...`
                                                   # and `go vet ./...` both succeed unmodified).
write-path-integration/
├── go.work                                       # ties 4 modules together for LOCAL dev only
├── gateway-client/    (module "gatewayclient")    #   4 files tagged `//go:build integration`
├── ipfsclient/        (module "ipfsclient")       #   1 file tagged `//go:build integration`
├── keystore/          (module "keystore")         #   0 integration-tagged files — unit-only module
└── writepaths/        (module "writepaths")       #   2 files tagged `//go:build integration`
```

**The go.work quirk, stated precisely because it will silently break a naive pipeline:** `go build
./...` or `go test ./...` run from `write-path-integration/` itself (the directory holding `go.work`)
fails with:

```
pattern ./...: directory prefix . does not contain modules listed in go.work or their selected dependencies
```

This is a real Go limitation, reproduced this session, not a hypothetical risk — `go.work`'s `use`
block lists `./gateway-client`, `./ipfsclient`, `./keystore`, `./writepaths` as the modules, and the
workspace root itself is not one of them, so a `./...` pattern anchored at `.` resolves to nothing.
**Every lint/unit-test job below `cd`s into the specific module directory first.** A CI script that
tries to shortcut this with one `go build ./...` at the `write-path-integration` root will reproduce
the exact failure above — this is the documented reason not to do that, not an arbitrary style
preference.

Running a Go command from *inside* a module subdirectory (e.g. `write-path-integration/writepaths`)
works normally and picks up the parent `go.work` automatically in workspace mode — verified this
session (`go vet ./...` succeeds cleanly from each of the four module directories).

## 2. Trigger model

- **Pull request**, any branch → target the default branch: run stages 1–4 (lint, SAST, unit,
  integration). No image build, no deploy gate — a PR does not need a publishable artifact.
- **Push to the default branch** (post-merge): run all six stages, 1→6 in strict sequence. Stage 5
  (image build) and stage 6 (deploy gate) only execute if 1–4 are green — enforced by CI-native
  job-dependency (`needs:`/`stage` ordering), not by a script-level `if` that could be bypassed.
- **Tag push** (e.g. `chaincode-v*`): same as push-to-default-branch, additionally used as the image
  tag (§5).

## 3. Stage — Lint

**Purpose:** catch formatting/vet-level defects before spending CI minutes on SAST or the live
network. Two independent lint jobs (chaincode, write-path-integration) — never merged into one job,
since they are different module trees with different Go toolchain versions pinned (`go 1.21` for the
chaincode module's `go.mod`; `go 1.25.9` for all four write-path-integration modules' `go.mod`/`go.work`
— a single job pinned to one Go version cannot correctly build both).

**Chaincode module** (single module, no `go.work` quirk applies here):
```sh
cd fabric-network/chaincode/employeeprofilerecord
gofmt -l $(find . -name '*.go' -not -path './vendor/*')   # must print nothing
go vet ./...
```
Both commands verified clean against the current tree this session (`gofmt -l` prints nothing; `go
vet ./...` exits 0).

**write-path-integration — once per module** (the go.work quirk from §1 applies: never run from the
`write-path-integration` root):
```sh
for m in gateway-client ipfsclient keystore writepaths; do
  (cd write-path-integration/$m && gofmt -l . && go vet ./...)
done
```
`gofmt -l .` inside each of the four module directories currently prints nothing; `go vet ./...`
exits 0 in each — verified this session, not assumed.

**Recommended, not yet configured:** `golangci-lint` for a stricter rule set. No `.golangci.yml`
exists in this repo today, so this stage's *verified minimum* is `gofmt` + `go vet` above; adding
`golangci-lint` is future hardening, and if adopted it must still run per-module-directory (§1) — a
`golangci-lint` binary run from the `write-path-integration` root would hit the identical `go.work`
resolution failure, since it shells out to the same `go list`/`go build` machinery.

**Gate:** any non-empty `gofmt -l` output, or any non-zero `go vet` exit, fails the stage.

## 4. Stage — SAST

**Concrete mechanism (not a generic "some SAST tool"):** Semgrep, invoked in CI via the `semgrep ci`
subcommand — Semgrep's own CI-integration entry point, which runs the rule sets enabled for this
repository on the Semgrep AppSec Platform and uploads results back to that platform. This is the
**same backend** the Semgrep Guardian MCP tool surface available in this Claude Code session reads
from — concretely:

| MCP tool (this session) | Role relative to the CI stage |
|---|---|
| `mcp__plugin_semgrep_guardian__get_semgrep_sast_findings` | Reads the code-security findings the `semgrep ci` run in this stage uploads — the query side of the same SAST data ST-9/T13 require |
| `mcp__plugin_semgrep_guardian__get_semgrep_secrets_findings` | Reads secrets-class findings (committed credentials, private keys) from the same upload |
| `mcp__plugin_semgrep_guardian__get_semgrep_supply_chain_findings` | Reads dependency/SCA findings — relevant here because the chaincode module vendors its dependencies (`vendor/modules.txt`) and every write-path-integration module has its own `go.sum` |
| `mcp__plugin_semgrep_guardian__list_semgrep_projects` | Resolves the exact project/repository name on the platform, needed before any of the three `get_*` calls above if the name isn't already known |
| `mcp__plugin_semgrep_guardian__whoami` / `login` | Session-side auth to the platform — the CI job's own auth is a separate token (`SEMGREP_APP_TOKEN`, §7), not this session's login |

In other words: **this CI stage is the producer** (runs the scan, gates the build on its result); **the
Guardian MCP tools are the consumer** (an agent session triaging/reading the same findings
afterward) — one backend, two call sites. Do not describe this stage as invoking the MCP tools
directly — CI runners do not have this session's MCP connection; they call the `semgrep` CLI, which
talks to the platform over its own API using a project-scoped token.

**Scope (per ST-9's own wording — "the chaincode + the HRIS profile-write path"):**
```sh
semgrep ci --config auto \
  fabric-network/chaincode/employeeprofilerecord/chaincode \
  write-path-integration/gateway-client \
  write-path-integration/ipfsclient \
  write-path-integration/keystore \
  write-path-integration/writepaths
```
`fabric-network/chaincode/employeeprofilerecord/vendor/` and any `go.sum`-pinned third-party source
are excluded from the SAST-proper scan target list above (third-party code is not this item's to fix)
but **is** exactly what the supply-chain findings API (`get_semgrep_supply_chain_findings`) is for —
run `semgrep ci`'s SCA scan (governed by what's enabled for the repo on the platform side, not a
separate CLI flag this spec can pin) against the same tree so `go.sum`/`vendor/modules.txt` entries
are checked for known-vulnerable versions.

**Gate:** `semgrep ci`'s own exit code (non-zero on a policy violation, per the platform's configured
severity gate) fails the stage. **Recommended default severity floor: fail on any new `medium` or
higher finding in `open` status** — chosen to match ST-9's bar of "clean," not merely
"no criticals"; **flagged as a recommendation, not an already-ratified numeric constant** — no ADR or
test-strategy row pins an exact severity threshold, so whoever adopts a concrete CI provider should
confirm this against `security-architect`'s intent before treating it as fixed.

## 5. Stage — Unit tests

Two independent test jobs, same module-boundary reasoning as §3. **Both exclude
integration-tagged files automatically** — Go's build-tag mechanism means a file starting
`//go:build integration` is simply not compiled into a plain `go test ./...` build; no `-tags` flag,
no separate `_test` file naming convention, and no manual exclusion list is needed for this stage to
be integration-test-free.

**Chaincode module:**
```sh
cd fabric-network/chaincode/employeeprofilerecord
go test ./...
```
(Existing tests: `asset_test.go`, `identity_test.go`, `queries_test.go`,
`record_profile_section_test.go`, `testutil_test.go` — none carry a build tag, all run here.)

**write-path-integration — once per module, per §1's go.work boundary:**
```sh
for m in gateway-client ipfsclient keystore writepaths; do
  (cd write-path-integration/$m && go test ./...)
done
```
Verified this session which modules actually contribute unit-vs-integration coverage:

| Module (dir) | Unit test files (this stage) | Integration-tagged files (§ stage 6 only) |
|---|---|---|
| `gateway-client` (`gatewayclient`) | `digestbuilder_test.go` | `gatewayclient_integration_test.go`, `tenant_isolation_integration_test.go`, `verify_integration_test.go`, `verify_notfound_integration_test.go` |
| `ipfsclient` | `ipfsclient_test.go` | `ipfsclient_integration_test.go` |
| `keystore` | `keystore_test.go`, `handoff_test.go` | **none** — this module has zero integration-tagged files; its full test suite runs in this stage |
| `writepaths` | `erasure_test.go` | `partial_failure_integration_test.go`, `writepaths_integration_test.go` |

**Gate:** any non-zero `go test` exit in either job fails the stage. Both jobs may run in parallel
(they touch disjoint module trees and neither needs the live network) — this stage has no environment
dependency, unlike §6.

## 6. Stage — Integration tests

**Trigger condition:** `-tags=integration` compiles in every file listed in the right-hand column of
§5's table. These tests dial real services — they are not runnable without the environment below, and
this stage's CI job must provision it, not assume a developer's already-running laptop instance (the
posture described for this workspace's *current* live infrastructure, which this pipeline treats as a
CI-environment requirement to reconstruct, not something CI can reach into from outside).

**Required live environment, concretely (not hand-waved):**

1. **Fabric network** — bring up via the two real compose files this repo already has, in the order
   their own header comments require (`ca-docker-compose.yaml` has no runtime dependency on
   `network-docker-compose.yaml`, but identity material `network-docker-compose.yaml`'s peers/orderers
   mount is `cryptogen`-generated, independent of the CA containers — CA-first is the established
   sequence in this repo, see `network-docker-compose.yaml`'s own header):
   ```sh
   docker compose -f fabric-network/network/compose/ca-docker-compose.yaml up -d
   docker compose -f fabric-network/network/compose/network-docker-compose.yaml up -d
   ```
2. **Channel create/join + chaincode lifecycle** — this repo's existing one-shot Go tools, not shell
   scripts (`fabric-network/tools/netjoin`, `fabric-network/tools/ccdeploy`), each driving the real
   `osnadmin`/`peer` binaries inside the `fabric-tools-net` helper container. **Disclosed gap:** both
   tools are hardcoded to the `tenant01` fixture (channel ID, container names, MSP paths) and were
   written for one-shot manual execution during this build, not for idempotent rerun-safety a
   production CI job normally wants (e.g. safe re-run against an already-provisioned channel, clean
   teardown). For CI purposes this is *usable as-is* — a CI integration-test job would target the same
   `tenant01` fixture the tools already assume, not a fresh tenant per run — but hardening these into
   general-purpose, idempotent CI fixtures is follow-up work this item does not itself complete.
3. **The CCaaS chaincode server container**, built from `fabric-network/chaincode/employeeprofilerecord/
   Dockerfile`, **must be started with `CHAINCODE_ID` set to the exact package ID** `peer lifecycle
   chaincode queryinstalled` reports for the currently-installed package (per `ccdeploy/main.go`'s own
   header comment) — a mismatch here is a real, previously-encountered failure mode (CC-5's backlog
   row), not a theoretical one.
4. **IPFS private swarm** — `ipfs-cluster/docker-compose.yaml`, exposing the two kubo REST APIs
   `ipfsclient` dials directly (`http://localhost:5001`, `http://localhost:5002` — the exact constants
   `ipfsclient_integration_test.go` hardcodes):
   ```sh
   docker compose -f ipfs-cluster/docker-compose.yaml up -d
   ```
   The `ipfs-cluster` CRDT peer-orchestration layer's own known limitation (`ipfs-cluster/README.md` —
   `cluster0`/`cluster1` do not complete a libp2p handshake) does **not** block this stage: the
   integration tests exercise the two kubo nodes' REST APIs directly, not the cluster layer.

**Run:**
```sh
(cd write-path-integration/gateway-client && go test -tags=integration ./...)
(cd write-path-integration/ipfsclient    && go test -tags=integration ./...)
(cd write-path-integration/writepaths    && go test -tags=integration ./...)
# keystore: no -tags=integration invocation needed — it has no integration-tagged files (§5).
```
Same §1 module-boundary rule applies here too — each `go test -tags=integration` call is scoped to
one module directory, never the `write-path-integration` root.

**Gate:** any non-zero exit fails the stage. This stage requires a CI runner (or CI-managed service
containers) capable of running the multi-container Fabric+IPFS stack described above — a detail a
generic "check out code, run go test" runner cannot satisfy; the reference implementation (§ next
file) models this as a dedicated environment-setup job, not an inline `services:` block, precisely
because step 2 above needs custom Go tools run in sequence, not merely containers reachable on
`localhost`.

## 7. Stage — Image build

**Target:** the chaincode's CCaaS image, from the Dockerfile that already exists at
`fabric-network/chaincode/employeeprofilerecord/Dockerfile`:
```dockerfile
FROM golang:1.21 AS build
WORKDIR /src
COPY . .
RUN go build -o /out/employeeprofilerecord .
FROM debian:bookworm-slim
COPY --from=build /out/employeeprofilerecord /usr/local/bin/employeeprofilerecord
ENV CHAINCODE_SERVER_ADDRESS=0.0.0.0:9999
EXPOSE 9999
ENTRYPOINT ["/usr/local/bin/employeeprofilerecord"]
```
**Build-context correctness — a real gotcha, not a style note:** the build context **must be**
`fabric-network/chaincode/employeeprofilerecord/` itself, never the repo root. `COPY . .` copies
everything in the build context; pointed at the repo root, it would drag the entire monorepo into the
image and the subsequent `go build -o /out/employeeprofilerecord .` would run against the wrong
`go.mod` (or none, at the repo root, since there is no root-level `go.mod`).
```sh
docker build \
  -t "${REGISTRY}/employeeprofilerecord:${GIT_SHA}" \
  fabric-network/chaincode/employeeprofilerecord
```
`${REGISTRY}` is left as a placeholder — no registry is chosen in this repo (tool-agnostic, per this
spec's own charter). `${GIT_SHA}` (not a floating tag like `latest`) so every image is traceable back
to the exact commit that produced it, matching the pinning discipline this repo already applies
elsewhere (`network-docker-compose.yaml`'s own `[VERIFY]` notes on floating `hyperledger/fabric-peer:
2.5`/`fabric-ca:1.5` tags — the same caution, applied here from the start rather than retrofitted).

**Gate:** build succeeds; image pushed **only** if stages 3–6's prerequisite jobs (lint, SAST, unit
×2, integration) are all green **and** the trigger is a push to the default branch or a tag (§2) — a
PR never publishes an image.

**Not this stage's job, stated explicitly so it isn't assumed:** pushing a new image does **not**, by
itself, redeploy the running chaincode. Per `ccdeploy/main.go`'s own header comment, the CCaaS server
container must be restarted with a `CHAINCODE_ID` matching the *newly installed* package — install/
approve/commit is a separate `peer lifecycle` sequence this stage does not perform. That sequence is
this repo's existing chaincode-lifecycle tooling (§6 item 2), invoked as an operational step after a
new image is available, not as part of the build.

## 8. Stage — Deploy gate

**What "gate" means here:** a single pass/fail checkpoint, evaluated only when stages 1–5 have all
succeeded, that authorizes (but per §0/standing scope decision, does **not itself perform**) a real
deploy. Concretely, the gate:

- Requires `needs: [lint, sast, unit-chaincode, unit-writepath, integration, image-build]` (CI-native
  dependency, not a script-level check that could silently be skipped).
- Requires an **environment-protection / manual-approval boundary** before anything downstream of this
  gate could run — even though this spec does not define what runs downstream (that is DEP-1/DEP-2/
  DEP-3's Helm charts, not this item's).
- Emits a single explicit signal (e.g. a `deploy-ready` status/tag) that a *separate*, not-yet-written
  deploy workflow could key off — `helm upgrade --install ... deploy/helm/fabric-network` (`DEP-1`) and
  `deploy/ipfs-cluster` (`DEP-3`) are named here only to point at where that future invocation would
  live, not because this item authors it.

**Gate condition:** all of §3–§7 green, on the default branch or a release tag. Any failure anywhere
upstream blocks this stage entirely — it does not run in a degraded/partial mode.

## 9. "No secrets in logs" — how this spec addresses its own DoD structurally

This item's own DoD line (`implementation-backlog.md` DEP-4 row) ends "no secrets in logs." This is
distinct from — and additional to — `repository-structure.md`'s existing "no secrets in git" rule:
git-leakage and log-leakage are two different exposure surfaces for the same credential material
(MSP private keys, TLS keys, the IPFS swarm key, any CI provider secret), and a pipeline can satisfy
one while violating the other (e.g. a secret correctly kept out of git can still be leaked by a CI
step that `cat`s or `echo`s it into a build log everyone with read access to the CI provider can see).

**Structural guarantee this spec requires of every stage above and of the reference implementation:**
no step, in any stage, ever runs a command whose purpose or side effect is to print the *contents* of:
- a TLS private key or cert-plus-key bundle (`.../tls/server.key`, `.../tls/client.key`, any
  `tlsca.*-cert.pem`'s paired key),
- an MSP signing key (`.../msp/keystore/priv_sk`),
- the IPFS swarm key (`ipfs-cluster/swarm-key/swarm.key`, `cluster-secret.txt`),
- or any CI-provider secret value (`SEMGREP_APP_TOKEN`, a future registry credential, etc.).

Concretely: no stage above includes `cat`/`echo`/`printenv`/`env` against any path or variable holding
the material listed above, and no step sets `set -x`/`ACTIONS_STEP_DEBUG`-style command-echoing in a
step that touches one of those paths or variables. This is checked structurally in
`github-actions-reference.yml` (§ that file's own closing comment) — **note the specific nuance that a
CI provider's native secret-masking (e.g. GitHub Actions redacting registered `secrets.*` strings) only
masks values it was told about via the secrets mechanism; it does not retroactively mask a private-key
file's contents that a step chose to print.** Relying on masking instead of simply never printing the
value is not equivalent, and this spec does not treat masking as a substitute for the "never print"
rule above.

## 10. Known gaps (disclosed, not hidden)

- **No idempotent, general-purpose network bring-up script exists yet** in this repo — §6 step 2's
  tools (`netjoin`, `ccdeploy`, and `tenantprovision`) were written for one-shot manual execution
  during this build and are hardcoded to the `tenant01` fixture. A CI-grade rewrite (parametrized,
  safe to rerun, with teardown) is real follow-up work, not something this spec can claim is already
  done.
- **No CI provider, registry, or Semgrep deployment is actually configured** — every placeholder
  (`${REGISTRY}`, `${{ secrets.SEMGREP_APP_TOKEN }}`, etc.) is exactly that; wiring a real one in is
  explicitly out of scope under the standing scope decision (§ header).
- **The severity floor for the SAST gate (§4) is a recommendation, not a ratified constant** — no ADR
  or `test-strategy.md` row pins an exact number; flagged for `security-architect` to confirm.

## 11. Traceability

| Stage | Backlog/ADR/TEST citation |
|---|---|
| Lint | `DEP-4` (backlog) |
| SAST | `ADR-0020` (contract boundary the scan protects), `SEC T13`, `TEST ST-9` |
| Unit tests | `CC-1`, `REC-1` (the modules under test), `DEP-4` |
| Integration tests | `REC-4`/`REC-5`/`INT-5` (the live-network behavior under test), `DEP-4` |
| Image build | `CC-5` (the CCaaS packaging shape this image must match) |
| Deploy gate | `DEP-4`; hands off to `DEP-1`/`DEP-3` (not authored here) |

---
baseline_commit: NO_VCS
---

# Story 1.1: Module Scaffold and Long-Lived Fabric Connection

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the platform team operating the integration bridge,
I want the service to start with one persistent Fabric Gateway connection and shut down cleanly,
so that every anchoring request reuses a warm connection instead of racing a dangling one.

## Acceptance Criteria

1. **Given** a fresh checkout of `integration-bridge/`, **when** it is built with `go build`, **then** it compiles cleanly via `require`+`replace` pairs for all four `write-path-integration/*` modules (AD-2), with `keystore` imported directly — not only transitively — to construct concrete `EmployeeKeyStore`/`SaltStore`/`DocumentKeyStore` implementations. [Source: epics.md Story 1.1 AC1; ARCHITECTURE-SPINE.md AD-2]
2. **Given** valid configuration, **when** the process starts, **then** exactly one `*gatewayclient.GatewayClient` and one `Hooks` are constructed, once, before the HTTP listener starts, **and** no other code path constructs a second `GatewayClient`. [Source: epics.md Story 1.1 AC2; ARCHITECTURE-SPINE.md AD-2]
3. **Given** the process is running, **when** it receives `SIGTERM`/`SIGINT`, **then** it stops accepting new requests, calls `GatewayClient.Close()` exactly once, and exits, **and** `Close()` is never called more than once. [Source: epics.md Story 1.1 AC3; ARCHITECTURE-SPINE.md AD-2]

## Tasks / Subtasks

- [x] Task 1: Scaffold `integration-bridge/go.mod` (AC: #1)
  - [x] Create `integration-bridge/go.mod`, module name `integrationbridge` (matches this repo's no-hyphen Go module-name convention — see Dev Notes "Module naming"), `go 1.25.9`
  - [x] Add `require`+`replace` pairs for **all four** sibling modules: `writepaths`, `gatewayclient`, `ipfsclient`, `keystore` — pointing at `../write-path-integration/{writepaths,gateway-client,ipfsclient,keystore}` respectively (directory names don't match module names 1:1 — see Dev Notes "Exact replace targets")
  - [x] Verify `go build ./...` succeeds from inside `integration-bridge/` (per this repo's own go.work quirk, never from a parent directory)
- [x] Task 2: Implement config loading (AC: #2)
  - [x] Load `NewGatewayClient`'s 10 positional string parameters from environment variables (no config-loading convention exists elsewhere in this repo's Go code to reuse — see Dev Notes "Config: no existing pattern to extend")
  - [x] Fail fast with a clear error naming every unset required variable if config is incomplete — don't silently default
- [x] Task 3: Wire one-time construction in `main.go` (AC: #2)
  - [x] Construct the four `keystore` stores, the `OperationalStore`, and the `*gatewayclient.GatewayClient` — **exactly once**, before starting any listener. `ipfsclient.Client` is deliberately **not** constructed in this story — `IPFS` stays `nil` on `Hooks`, correct per `writepaths.go`'s own "nil is fine unless a hook call carries a document" contract, since no route exists yet to carry one (Story 1.2's job; corrected during code review — the original wording here claimed `ipfsclient.Client` construction that never happened).
  - [x] Construct one `writepaths.Hooks{}` populated with all of the above plus `TenantID` from config, and **retain the reference** — `run()` keeps `hooks` (not `_`) even though nothing dispatches through it yet
  - [x] **Flag for explicit sign-off before this ships to any real deployment** (does not block this story's own ACs — see Dev Notes "No production store implementation exists yet"): the only `OperationalStore`/`EmployeeKeyStore`/`SaltStore`/`DocumentKeyStore` implementations in this codebase today are `InMemory*` test mocks with **zero persistence** — every restart silently loses every salt and `employeeKey_i` ever created. Wire them for this story (nothing here requires real persistence to pass); the warning is a top-of-file package doc comment in `main.go` itself (corrected during code review — it originally lived only in `wiring.go`, not literally top-of-file in `main.go` as written here).
- [x] Task 4: Implement graceful shutdown (AC: #3)
  - [x] Use `signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)` (stdlib, Go 1.16+, stable through 1.25.9 — no external dependency needed)
  - [x] On context cancellation: stop accepting new connections (`http.Server.Shutdown(ctx)`), then call `GatewayClient.Close()` exactly once, then exit
  - [x] Extract the "call Close() exactly once on shutdown" logic behind a small local interface (e.g. `type closer interface { Close() error }` — `*gatewayclient.GatewayClient` already satisfies this via its existing `Close() error` method) so this logic is unit-testable without a live Fabric network — see Dev Notes "Testing AC#3 without live infra"
- [x] Task 5: Tests
  - [x] Unit test: shutdown-wiring calls `Close()` exactly once given a stub `closer`, never zero, never twice, even if the signal fires twice
  - [x] Integration test (`//go:build integration`, per this repo's established convention): construct against the live network, confirm exactly one connection, confirm `SIGTERM` produces a clean process exit with the HTTP port released — a strong (not direct) signal `Close()` ran, since the test doesn't capture stdout/stderr to observe the call itself (wording corrected during code review)
  - [x] `go vet ./...` and `gofmt -l` clean (this repo has no linter configured — CLAUDE.md's own stated convention, don't add one)

## Dev Notes

**Scope boundary — read this first.** This story creates `integration-bridge/go.mod` and `main.go`
only. It does **not** create `handlers.go`, `pipeline.go`, or any HTTP route — those are Story 1.2.
Nothing in this story's ACs requires a working HTTP endpoint; don't build one preemptively (that
would be scope creep into Story 1.2's work, and Story 1.2's own pipeline design — auth, validate,
dispatch, map-error, respond — isn't yet built for this story's `main.go` to correctly wire into).

### Exact replace targets (verified against the real go.mod files, not assumed)

`write-path-integration/writepaths/go.mod` currently declares **zero** `require` entries, even
though `writepaths.go` imports `gatewayclient`, `ipfsclient`, and `keystore` by their short local
module names (`gatewayclient "gatewayclient"`, etc. — these resolve today **only** via
`write-path-integration/go.work`'s `use` block; outside that workspace, nothing resolves them).
`integration-bridge/` is deliberately **not** added to that `go.work` (AD-2). This means a single
`replace writepaths => ../write-path-integration/writepaths` is **not sufficient** — verified by
direct inspection of `go.mod`, not assumed:

```
module integrationbridge

go 1.25.9

require (
    writepaths     v0.0.0
    gatewayclient  v0.0.0
    ipfsclient     v0.0.0
    keystore       v0.0.0
)

replace (
    writepaths    => ../write-path-integration/writepaths
    gatewayclient => ../write-path-integration/gateway-client
    ipfsclient    => ../write-path-integration/ipfsclient
    keystore      => ../write-path-integration/keystore
)
```

Note the directory-vs-module-name mismatch: the `gatewayclient` module's directory is
`gateway-client/` (hyphenated), but its `go.mod` declares `module gatewayclient` (no hyphen) — the
`replace` right-hand side is a filesystem path (`../write-path-integration/gateway-client`), the
left-hand side is the module name (`gatewayclient`). Getting this backwards is the single most
likely first build error.

`keystore` needs a **direct** `require`, not merely a transitive one through `writepaths` — because
`main.go` must call `keystore.NewInMemorySaltStore()` / `NewInMemoryDocumentKeyStore()` /
`NewInMemoryEmployeeKeyStore()` directly to populate the `Hooks` struct's fields (AD-2's own Rule
text says so explicitly — this is not incidental).

### `write-path-integration/gateway-client`'s own `go.mod` has real external deps

Confirmed by reading it directly: `github.com/hyperledger/fabric-gateway v1.12.0`,
`github.com/hyperledger/fabric-protos-go-apiv2 v0.3.7`, `google.golang.org/grpc v1.82.1`, plus
`github.com/cyberphone/json-canonicalization`. These are transitively pulled in once
`integration-bridge` requires `gatewayclient` — no action needed beyond the `require`+`replace`
above; `go build` resolves them from the existing `go.sum` machinery. Do not pin a different
version of `fabric-gateway` — this one is already the version this codebase's live integration
tests are proven against.

### `NewGatewayClient`'s exact signature — 10 positional strings, order matters

Verified directly from `write-path-integration/gateway-client/gatewayclient.go`:

```go
func NewGatewayClient(
    peerEndpoint, tlsServerNameOverride, tlsCACertPEMPath,
    clientTLSCertPEMPath, clientTLSKeyPEMPath, mspID,
    certPEMPath, keyPEMPath, channelName, chaincodeName string,
) (*GatewayClient, error)
```

All 10 are positional strings with **no field names at the call site** — the single easiest mistake
is passing them in the wrong order and having it silently compile. A real call, taken verbatim from
`gateway-client/gatewayclient_integration_test.go` (adapt paths/values from config, don't invent a
different shape):

```go
gw, err := gatewayclient.NewGatewayClient(
    "localhost:7051", "peer0.org1",
    netDir+"/crypto-config/peerOrganizations/org1/tlsca/tlsca.org1-cert.pem",
    netDir+"/crypto-config/peerOrganizations/org1/users/Admin@org1/tls/client.crt",
    netDir+"/crypto-config/peerOrganizations/org1/users/Admin@org1/tls/client.key",
    "Org1MSP",
    netDir+"/crypto-config/peerOrganizations/org1/users/Admin@org1/msp/signcerts/Admin@org1-cert.pem",
    netDir+"/crypto-config/peerOrganizations/org1/users/Admin@org1/msp/keystore/priv_sk",
    "tenant-tenant01", "employeeprofilerecord",
)
```

`channelName` (here `tenant-tenant01`) is the per-tenant channel this bridge deployment serves — it
must be sourced from config, matching AD-2's "one bridge deployment per tenant" rule (do not
hardcode a tenant name).

`Close()` is documented on the type itself: *"Call once, at host-process shutdown — never per
transaction (that would defeat the 'one retained connection' requirement this wrapper exists to
satisfy)."* This is the exact property AC#3 tests.

### Config: no existing pattern to extend

Checked directly: no Go code anywhere in this repo (`fabric-network/tools/*`, `write-path-
integration/*`) uses environment variables or a config file for runtime config — the standalone
tools take at most one positional CLI arg (e.g. `tenantprovision <tenantID>`) or hardcode constants.
There is nothing Go-side to imitate. The one real, cited precedent is PHP-side:
`services/ems/BaseEmsService.php` reads `env("EMS_SERVICE_URL")`/`env("EMS_API_KEY")`
(`ADR-0022`'s own Context section). Environment variables are the right choice here — matches that
precedent's spirit and is the idiomatic default for a Go service with no config-file convention to
inherit. Suggested variable names (not mandated by any ADR, but keep them self-describing):
`BRIDGE_PEER_ENDPOINT`, `BRIDGE_TLS_SERVER_NAME`, `BRIDGE_TLS_CA_CERT_PATH`,
`BRIDGE_CLIENT_TLS_CERT_PATH`, `BRIDGE_CLIENT_TLS_KEY_PATH`, `BRIDGE_MSP_ID`,
`BRIDGE_SIGN_CERT_PATH`, `BRIDGE_SIGN_KEY_PATH`, `BRIDGE_CHANNEL_NAME`, `BRIDGE_CHAINCODE_NAME`,
`BRIDGE_TENANT_ID`.

### No production store implementation exists yet — real, verified gap, not an assumption

Checked every implementation of the four interfaces `Hooks` needs
(`writepaths.OperationalStore`, `keystore.{EmployeeKeyStore,SaltStore,DocumentKeyStore}`) across the
whole repo. The **only** types satisfying them anywhere are:

- `writepaths.InMemoryOperationalStore` — its own doc comment: *"the mock used by this package's
  own tests"*
- `keystore.InMemorySaltStore` / `InMemoryDocumentKeyStore` / `InMemoryEmployeeKeyStore` — the
  package's own doc comment: *"NOT the production store (no persistence, no encryption-at-rest, no
  backup discipline — all explicitly out of this item's scope)"*

None persist across a process restart. For a service whose entire purpose is anchoring durable
records, wiring these into a real deployment would silently and permanently lose every salt and
`employeeKey_i` on every restart — a data-loss class of bug, not a cosmetic one. **This story's own
ACs do not require durability** (they test compile, one-time construction, and shutdown — nothing
about surviving a restart), so implementing this story with the `InMemory*` stores is honest and
sufficient for what's asked. But do not let this pass unnoticed: mark it loudly in `main.go` (Task
3), and raise it as an open question (see end of this document) — a real persistent-store
implementation is unbuilt anywhere in this codebase and is not currently anyone's assigned story.

### Testing AC#3 without live infra

`*gatewayclient.GatewayClient` is a concrete struct (private fields), not an interface — it can't be
mocked by substitution. But it already has a `Close() error` method, so a **local** interface
(`type closer interface { Close() error }`, defined inside `integration-bridge`, not in
`gatewayclient`) lets the shutdown-wiring logic be unit-tested against a stub without touching
`gateway-client` at all. This keeps "exactly once, never twice" genuinely unit-testable, while the
full "actually connects to a live network" path stays an integration test — matching this repo's
own established `//go:build integration` gating convention (CLAUDE.md), not inventing a new one.

### Module naming

Every other module in this repo uses a bare, no-hyphen Go module name matching (or close to) its
directory: `writepaths`, `gatewayclient`, `keystore`, `ipfsclient`, `ccdeploy`, `netjoin`. Follow
that: `module integrationbridge` in `go.mod`, even though the directory is `integration-bridge/`
(hyphenated) — same mismatch pattern `gatewayclient`/`gateway-client` already has, not a new
inconsistency.

### Project Structure Notes

- New top-level directory `integration-bridge/`, sibling to `fabric-network/` and
  `write-path-integration/` — not nested inside either (AD-2).
- This story's own file set: `go.mod`, `main.go`, plus whichever small internal file holds the
  `closer` interface and shutdown-wiring unit test (e.g. `shutdown.go` + `shutdown_test.go`, or
  inline in `main.go` + `main_test.go` — either is fine, this repo has no strict one-file-per-
  concern convention to violate).
- Not created in this story (Story 1.2's scope): `handlers.go`, `pipeline.go`, `pipeline_test.go`.
- No database/entity creation applies — this component owns no database of its own.

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 1.1] — the approved ACs, verbatim
- [Source: _bmad-output/planning-artifacts/architecture/architecture-fabric-hris-2026-08-08/ARCHITECTURE-SPINE.md#AD-2] — module/deployment/lifecycle Rule
- [Source: write-path-integration/gateway-client/gatewayclient.go] — `NewGatewayClient`, `Close()`, `GatewayClient` struct (read directly, this session)
- [Source: write-path-integration/writepaths/writepaths.go] — `Hooks` struct, `OperationalStore` interface, `InMemoryOperationalStore` doc comment (read directly, this session)
- [Source: write-path-integration/keystore/keystore.go] — `SaltStore`/`DocumentKeyStore`/`EmployeeKeyStore` interfaces and their `InMemory*` doc comments (read directly, this session)
- [Source: write-path-integration/gateway-client/gatewayclient_integration_test.go] — the real, working `NewGatewayClient` call this story's config-loading code should produce equivalent arguments for
- [Source: agent-suite/05-adr/ADR-0022-integration-bridge-gateway-client-host.md] — `BaseEmsService` env-var precedent cited for the config-loading approach
- [Source: _bmad-output/planning-artifacts/implementation-readiness-report-2026-08-08.md] — no blocking findings against this story specifically; Finding 1 (erasure scope) and Finding 2 (NFR-1 baseline) don't bear on Story 1.1

## Open Questions (not blocking this story — surface to architect/user before Story 1.1 ships to any real deployment)

1. **No persistent implementation of `OperationalStore`/`EmployeeKeyStore`/`SaltStore`/
   `DocumentKeyStore` exists anywhere in this codebase.** Only test-mock `InMemory*` types exist.
   This story can and should proceed with them (its ACs don't test durability), but building a real
   store is unbuilt work with no assigned story yet — confirm whether that's a gap in the epic or a
   deliberately later concern.
2. Config source confirmation: environment variables are proposed (no existing Go-side convention
   to follow; the one real precedent, `BaseEmsService`, is PHP-side `env()` calls). Confirm this is
   acceptable rather than, e.g., a flag-based or file-based config — no ADR mandates either.

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — claude-opus-5[1m]

### Debug Log References

- First `go build`/`go vet` attempt after adding real imports to `main.go` failed with 6
  `missing go.sum entry` errors — `integration-bridge` needs its **own** `go.sum`, separate from
  `write-path-integration`'s (it's deliberately not in that `go.work`). Fixed with `go mod tidy`,
  which also populated the transitive `require` block for `fabric-gateway`/`grpc`/etc. Not an error
  in the `go.mod` written for Task 1 — a normal, expected step for any new Go module with
  dependencies, worth recording since the story's Dev Notes didn't call out this exact step.
- End-to-end smoke test #1 (all config set, no live network) failed immediately with
  `listen tcp :8080: bind: address already in use` — an unrelated process already held port 8080
  on this machine. Not a code defect; confirmed by re-running on `127.0.0.1:18081` via the
  (undocumented-until-now) `BRIDGE_HTTP_ADDR` override, which succeeded. Recorded so a future
  reader doesn't mistake this log line for a real Story 1.1 bug if they see it in history.
- End-to-end smoke test #2 (free port) confirmed `grpc.NewClient` is genuinely lazy/non-blocking —
  the process started and kept listening without erroring, even with an unreachable
  `BRIDGE_PEER_ENDPOINT` (`localhost:19999`, nothing listening) and no live Fabric network,
  because the gRPC connection isn't actually dialed until the first RPC. `SIGTERM` was sent 5s
  later and the process exited cleanly (status 0) well within the 10s shutdown timeout, proving
  `runUntilShutdown`/`onceCloser`/`shutdownSequence`'s real wiring — not just the unit tests of
  the isolated pieces — actually works end-to-end.

### Completion Notes List

- All 3 ACs implemented and verified: **AC1** (`go build`/`go vet`/`gofmt -l` all clean, verified
  by direct execution — see Debug Log for the `go.sum` step this required), **AC2** (`buildHooks`
  constructs the `GatewayClient` exactly once — proven both by a unit test against a stub
  constructor and by `main.go`'s own code structure having exactly one call site), **AC3**
  (`onceCloser` proven idempotent under both sequential and concurrent calls, `-race` clean; the
  real binary confirmed to exit cleanly on `SIGTERM` in an end-to-end smoke test).
- Dependency-inversion applied consistently to both testable seams the story's Dev Notes
  anticipated: the `closer` interface for `GatewayClient.Close()`, and a matching
  `newGatewayClientFunc` for `NewGatewayClient` itself — the latter wasn't explicitly named in Dev
  Notes but follows the same pattern already endorsed there, and was necessary to make AC2's
  "constructed exactly once, with the right arguments in the right order" genuinely unit-testable
  without live infra.
- **Not independently decided, followed the story as written:** `IPFS` is wired `nil` in `Hooks`
  (Story 1.2's job, once a route can actually carry a `document`) — matches `writepaths.go`'s own
  documented "nil is fine unless a hook call carries a document," and no route exists yet in this
  story to call anything.
- **One config value beyond the story's own suggested list, added pragmatically:** `BRIDGE_HTTP_ADDR`
  (optional, defaults to `:8080`) — needed because `main.go` must start *some* `http.Server` for
  `AC3`'s "stops accepting new requests" to mean anything, but Story 1.1 predates any real routes.
  Deliberately **not** added to `LoadConfig`'s fail-fast-if-missing set (that set is exactly the 11
  vars the story's Dev Notes named for `NewGatewayClient`+`TenantID` — extending its required list
  would be scope creep the story didn't ask for); read directly via `os.Getenv` in `main.go` with a
  sensible default instead.
- **Honest disclosure on the integration test:** `main_integration_test.go` was written to spec
  (build the real binary, run it against `docs/QUICKSTART.md`'s live network, confirm the HTTP
  port opens, send `SIGTERM`, confirm clean exit and port release) and **compiles cleanly** under
  `-tags=integration` (verified with `go test -tags=integration -run xxx_nomatch ./...`), matching
  this repo's established gating convention. It was **not executed** in this session — no live
  Fabric network is available in this sandbox. This mirrors the same honesty standard the rest of
  this repo holds itself to (per `CLAUDE.md`/`docs/QUICKSTART.md`): don't claim a live-network test
  passed without actually running it against one.
- Both Open Questions from story creation remain genuinely open (not resolved during
  implementation, since resolving them wasn't this story's job): no persistent store
  implementation exists anywhere in this codebase (flagged loudly in `wiring.go`'s doc comment,
  per Task 3's own instruction), and the env-var config approach was used without an explicit
  confirmation round (no ADR contradicts it, and no blocking issue surfaced).
- **Code review remediation (2026-08-08):** 11 patch findings applied, 2 deferred (logged to
  `deferred-work.md`), 2 reviewer claims verified false and dismissed rather than assumed noise.
  The most consequential fix: `shutdownSequence.Close()` originally returned early if
  `srv.Shutdown` errored, skipping `GatewayClient.Close()` entirely on that path — the exact
  failure mode AC#3 exists to prevent, found independently by two reviewers. Fixed via a new
  `closeBoth` helper (unconditionally calls both, joins errors via `errors.Join`), with its own
  unit tests (`main_test.go`) — the original bug had zero test coverage before this. Also fixed: a
  real internal file path leaked into a code comment (confidentiality register, first occurrence,
  not a repeat of an existing leak); two task checkboxes that claimed work not actually done
  (`ipfsclient` construction, a top-of-file `main.go` comment) — corrected in place rather than
  silently left; `buildHooks`'s `*Hooks` was discarded, now retained; `http.Server` gained real
  timeouts; `BRIDGE_HTTP_ADDR` now goes through the same tested `getenv` seam as everything else;
  whitespace-only env values now correctly treated as missing. Full regression suite (17 tests,
  `-race`) and both build tags (`go build`, `go build -tags=integration`) re-verified clean after
  every change.

### File List

- `integration-bridge/go.mod` (new)
- `integration-bridge/go.sum` (new, generated by `go mod tidy`)
- `integration-bridge/main.go` (new)
- `integration-bridge/config.go` (new)
- `integration-bridge/config_test.go` (new)
- `integration-bridge/wiring.go` (new)
- `integration-bridge/wiring_test.go` (new)
- `integration-bridge/shutdown.go` (new)
- `integration-bridge/shutdown_test.go` (new)
- `integration-bridge/main_integration_test.go` (new, `//go:build integration`, not executed this session — see Debug Log/Completion Notes)
- `integration-bridge/main_test.go` (new, added during code-review remediation — unit-tests `closeBoth`)

### Review Findings

Reviewed by 3 parallel, mutually-blind layers (Blind Hunter, Edge Case Hunter, Acceptance Auditor
against this story's own ACs) — findings normalized, deduplicated, and every claim independently
re-verified against the actual source before rating (2 reviewer claims turned out to be false
positives on inspection; noted below in Dismissed, not silently dropped).

- [x] [Review][Patch] `shutdownSequence.Close()` skips `gw.Close()` entirely when `srv.Shutdown` errors [main.go:33-40] — on any `srv.Shutdown` failure (e.g. a slow in-flight request exceeding the 10s deadline), the function returns before reaching `s.gw.Close()`. AC#3 requires `Close()` called exactly once; on this path it's called zero times and the retained Fabric gRPC connection is never released. Found independently by both Blind Hunter and Edge Case Hunter.
- [x] [Review][Patch] Real internal file path cited in a code comment [config.go:12] — `services/ems/BaseEmsService.php` is a real path from the actual system this project anonymizes (confirmed: `ADR-0022`/`REAL-INTEGRATION-TRIGGER-FLOW.md` cite the same real path as part of the one disclosed real-specifics exception). Verified this is the **first occurrence** of this path anywhere outside that exception file — not a pre-existing leak being repeated. `agent-suite/context/PHP-INTEGRATION.md` (the generic-register companion meant to describe this precedent generically) does not actually contain a citable generic description, so the fix is to describe the calling convention generically inline and point to `ADR-0022`'s own Context section by name (not by restating its real-specifics content).
- [x] [Review][Patch] Task 3 checkbox claims `ipfsclient.Client` was constructed; it never is [Task 3, wiring.go] — the checked subtask literally lists "the `ipfsclient.Client`" among what's constructed, but `wiring.go` never imports `ipfsclient` and wires `IPFS: nil`. The nil-wiring itself is correct and justified (Dev Notes/doc comments explain why), but the checkbox text asserts something that didn't happen — a real record-integrity issue, not just a nitpick.
- [x] [Review][Patch] Task 3's "loud, top-of-file comment in `main.go`" doesn't exist in `main.go` [main.go, wiring.go] — the required warning about placeholder in-memory stores was placed in `wiring.go`'s doc comment instead, ~20 lines in, not top-of-file in `main.go` as the subtask literally specifies. The substance exists; the location doesn't match what was checked off.
- [x] [Review][Patch] Task 5's "`Close()` observed to have run" overstates what the integration test can observe [main_integration_test.go] — the test observes process exit + HTTP port release, which together strongly imply `Close()` ran, but neither is a direct observation of the `Close()` call itself (`cmd.Stdout`/`cmd.Stderr` aren't even captured). Wording should describe what's actually observed.
- [x] [Review][Patch] `buildHooks`'s returned `*writepaths.Hooks` is discarded [main.go:50] — `_, gw, err := buildHooks(...)`. All three independent reviewers flagged confusion about this construction being immediately unreachable; costs nothing to retain it now even though nothing uses it until Story 1.2.
- [x] [Review][Patch] `http.Server` has no `ReadHeaderTimeout`/`ReadTimeout`/`WriteTimeout`/`IdleTimeout` [main.go:62] — low exploitability today (empty mux, 404-only), but a known Go anti-pattern worth closing before Story 1.2 adds real routes rather than after.
- [x] [Review][Patch] `shutdown.Close()`'s error is silently discarded in the `ListenAndServe`-fails branch [main.go:74] — an operator debugging a failed startup can't tell whether cleanup also failed.
- [x] [Review][Patch] `BRIDGE_HTTP_ADDR` bypasses the tested `getenv`-injection pattern [main.go:58] — read directly via `os.Getenv` instead of through `LoadConfig`, inconsistent with every other config value in this module and untested.
- [x] [Review][Patch] Whitespace-only env var value bypasses fail-fast [config.go:35] — `if v == ""` doesn't catch `" "`; `strings.TrimSpace(v) == ""` would.
- [x] [Review][Patch] Integration test doesn't reap the child process on one failure path [main_integration_test.go:66-69] — `cmd.Process.Kill()` with no `cmd.Wait()` in the "listener never opened" branch. Low impact (test-only code, in a file not executed this session), cheap fix.
- [x] [Review][Defer] Two distinct failure classes (config/bind failures vs. post-serving shutdown failures) collapsed into one `log.Fatal` exit path [main.go] — deferred, no operational tooling consumes exit codes yet (deployment topology itself is still Deferred in `ARCHITECTURE-SPINE.md`); revisit once one exists.
- [x] [Review][Defer] `newGatewayClientFunc` stub could satisfy the type by returning `(nil, nil)` with no guard [wiring.go] — deferred, verified unreachable via the only real implementation (`gatewayclient.NewGatewayClient` never returns `(nil, nil)` — checked every return path); a defensive nil-check is reasonable future hardening, not urgent.

**Dismissed (2), verified false rather than assumed noise:**
- Edge Case Hunter's claim that `BRIDGE_TLS_SERVER_NAME` is "optional" (gatewayclient treats `""` as valid) — verified against `loadTLSCredentials`: the value is passed directly into `tls.Config.ServerName` for mutual-TLS verification; empty is not safe for a peer whose cert has a specific hostname (the one real precedent, `gatewayclient_integration_test.go`, always supplies a real value). The existing fail-fast requirement is correct.
- Blind Hunter's claim that the story's Completion Notes overclaim AC3 verification — re-read the actual text: it only claims the SIGTERM happy-path was confirmed end-to-end, which is literally true and was literally tested; it does not claim the `srv.Shutdown` timeout/error path (where the first finding above lives) was exercised.

**Also verified accurate, no issues:** `go build`/`go vet`/`gofmt -l`/`go test -race` all clean; `go.mod`'s `require`+`replace` pairs match Dev Notes exactly; `NewGatewayClient`'s 10-arg call order is correct; `keystore` imported directly per AD-2; `LoadConfig`'s fail-fast-naming-every-missing-var behavior is real and tested; `onceCloser`/`runUntilShutdown` idempotency is real and race-tested; zero confidentiality-register hits on the narrow `talenta`/`mekari` pattern; File List matches the diff exactly.

**Recommendation outside this diff's scope (not a patch item — a different file):** `CLAUDE.md`'s confidentiality-register rule enumerates `fabric-network/`, `write-path-integration/`, `ipfs-cluster/`, `qa-tests/`, `deploy/`, `docs/` as the hard-clean zone. `integration-bridge/` is a new sibling code directory that should obviously be governed by the same rule but isn't yet listed — worth adding so the next reviewer doesn't have to re-derive this.

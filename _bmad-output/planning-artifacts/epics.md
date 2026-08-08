---
stepsCompleted: ['requirements-extraction', 'epic-design', 'story-generation', 'final-validation']
inputDocuments:
  - '_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-02/prd.md'
  - '_bmad-output/planning-artifacts/architecture/architecture-fabric-hris-2026-08-08/ARCHITECTURE-SPINE.md'
  - 'agent-suite/05-adr/ADR-0022-integration-bridge-gateway-client-host.md'
  - 'agent-suite/05-adr/ADR-0014-in-band-recording.md'
  - 'agent-suite/context/REAL-INTEGRATION-TRIGGER-FLOW.md'
---

# fabric-hris - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for the **integration-bridge
service** (per `ARCHITECTURE-SPINE.md`, `status: final`, altitude: feature) — decomposing the
requirements from the PRD and Architecture spine into implementable stories.

**Scope note:** the source PRD (`prd-fabric-hris-2026-08-02/prd.md`) covers the whole Fabric-HRIS
prototype — 44 requirements across the 3-org network, chaincode, IPFS cluster, security, and
performance. This breakdown is deliberately scoped to only the requirements the integration-bridge
service touches; everything else in that PRD corresponds to work already built and tracked in
`agent-suite/06-roadmap/implementation-backlog.md`, not to new stories here. No UX design contract
exists for this component (it has no user-facing surface — its only client is talenta-core's own
PHP code).

## Requirements Inventory

### Functional Requirements

FR-1: The bridge MUST trigger recording one `EmployeeProfileRecord` each time talenta-core commits
a create/update to one of the five profile sections through a legitimate HRIS write path. This is
the exact trigger the bridge exists to serve — talenta-core calls the bridge synchronously at its
real commit sites (4 real PHP commit points covering the 5 sections, per `ADR-0022`), and the
bridge dispatches to the matching `writepaths.Hooks` method. `[T]`

FR-2: The section fingerprint the bridge's call ultimately produces MUST be computed from a
canonical JSON representation (RFC-8785 / JCS), byte-identical between writer and verifier. This
directly grounds AD-3's `newValue` wire-shape rule: `newValue` travels as a nested JSON object,
captured via `json.RawMessage` — never `map[string]interface{}` decoding, which would corrupt the
bytes JCS canonicalizes before hashing. `[T]` + PB-3

FR-8: The bridge MUST validate that a request targets one of the five valid `ProfileSection`
values and reject anything else. Satisfied structurally, not by application-level string checking:
AD-5 fixes exactly five static routes (`/v1/profile-sections/{PERSONAL|EMPLOYMENT|EDUCATION|
ADDITIONAL|PAYROLL}`) — a sixth value has no route to reach at all. `[D]`

FR-30: Each supporting document the bridge accepts MUST be encrypted with the employee's
`KEY_EMPLOYEE` before it reaches IPFS. The bridge does not implement this itself — it is
`Hooks`/`ipfsclient`'s job, reached via the optional `document` (base64) field in the bridge's
request envelope — but the bridge's contract is the wire boundary this requirement crosses. `[T]`

FR-31: Only the resulting CID may ever be recorded on-chain — document content must never touch
the ledger. This bounds a bridge-specific behavior not covered by any existing component: the
bridge's own error/logging conventions must never echo `document` (or `newValue`) content back in
a `detail` field or a log line, since nothing downstream would catch that leak once it happens at
this boundary. `[T]`

### NonFunctional Requirements

NFR-1: Write latency for profile-section recording MUST stay under 3 seconds at 500 TPS (P2). Cited
as the target `ARCHITECTURE-SPINE.md`'s Deferred timeout-duration item must be tuned against, using
`qa-tests/performance/RESULTS.md`'s real measured Fabric latencies.

NFR-3: Zero bytes of PII plaintext, or any reversible derivative, may ever reach the ledger (P0,
`ST-1`). For this component specifically: the bridge must never persist, log, or echo back
`newValue`/`document` content or the raw actor identity in a way that could leak it outside the
already-audited `Hooks`/chaincode boundary.

NFR-7: JSON canonicalization must produce byte-identical output between writer and verifier. The
actual reason AD-3 mandates `json.RawMessage` over generic map decoding for `newValue` — an
unmarshal-then-remarshal round-trip through Go's `encoding/json` is not guaranteed byte-stable for
large integers (PAYROLL bank-account fields), which would silently break this NFR.

NFR-8: Every profile-section change must be traceable to a verified, non-repudiable actor. Grounds
AD-1's Stage Contract: `EmployeeInternalID`/`ProfileSection`/actor identifiers are threaded through
a single request-scoped struct across every pipeline stage, specifically so they survive onto the
timeout path (which has no typed error to extract them from) and never silently drop from logs.

NFR-9: Anchoring must add less than 500ms to the HRIS write path it's attached to (calibration
pending real measurement). This is the primary NFR AD-4's shared timeout budget exists to serve —
the exact duration is not yet set; see Deferred below.

### Additional Requirements

- No starter/greenfield template. AD-6 mandates `net/http` stdlib only, Go 1.25.9, no third-party
  HTTP router or framework.
- New standalone Go module `integration-bridge/`, its own `go.mod`, with `require`+`replace` pairs
  for all four `write-path-integration/*` modules (`writepaths`, `gatewayclient`, `ipfsclient`,
  `keystore`) — one `replace` alone does not build (AD-2). This is Epic 1 Story 1 territory: the
  module scaffold itself is the first buildable unit.
- The `*gatewayclient.GatewayClient` (and the `Hooks` wrapping it) is constructed exactly once, at
  process startup in `main.go`, and reused across every request — never per-request (AD-2). This is
  the entire reason the service is a standalone process rather than a per-call connection.
- `main.go` registers `SIGTERM`/`SIGINT` handling and calls `GatewayClient.Close()` exactly once on
  shutdown (AD-2).
- One bridge deployment serves exactly one tenant — `Hooks.TenantID` is fixed per instance,
  matching channel-per-tenant (AD-2, grounds `ADR-0013`). A second tenant is a second deployment.
- Auth: `X-Api-Key` **and** `X-Company-ID` headers on every inbound request, matching
  `services/ems/BaseEmsService.php::companyLevelHeaders()`'s existing production precedent — not
  `X-Api-Key` alone (AD-2/Consistency Conventions).
- API contract: exactly five static `POST` routes under `/v1/profile-sections/` (AD-5) — the
  literal path list is fixed in the spine, not left to be invented per-implementer.
- Error contract: HTTP status is pure transport; all business outcome lives in a body `status`
  field (`committed`/`partial_failure`/`rejected`/`error`), set only by the pipeline's error-mapping
  stage, per an explicit failure-origin decision table (AD-3). Every story touching a route handler
  must respect this table, not invent its own mapping.
- Timeout: one shared `context.WithTimeout` budget, armed as the first statement of the dispatch
  stage (post-auth/validate), bounding every `ctx`-aware call inside `anchor()` — but explicitly
  **not** the final `SubmitTransaction` call, which takes no `ctx` at all (AD-4). A story
  implementing this must not "fix" it by racing a goroutine against the timeout — that was
  considered and rejected (double-submission risk against the one shared connection).
- Stage Contract: a single request-scoped struct carrying `EmployeeInternalID`/`ProfileSection`/
  `TenantID`/`Result`/`Err` across pipeline stages — every story's logging/response code must read
  from this struct, never from `errors.As`-unwrapping an error (AD-1).

**Genuinely blocking Deferred items** (from `ARCHITECTURE-SPINE.md`'s own Deferred section — these
may gate specific stories, not just background context):

- **Exact timeout duration** — needs real Caliper numbers against NFR-9/NFR-1; cannot be hardcoded
  as a magic number in a story's AC before that data exists.
- **Transport security beyond `X-Api-Key`/`X-Company-ID`** — mTLS needs explicit `security-architect`
  sign-off; a story should not silently assume the current precedent is sufficient.
- **Circuit-breaker/fallback for a bridge or Fabric outage** — explicitly unresolved by both
  `ADR-0014` and `ADR-0022`; a named risk-register follow-up (`agent-suite/10-risk/risk-register.md`)
  is still open.
- **Deployment target and environment topology** — no container/orchestration decision has been
  made; a "deploy the bridge" story cannot be written concretely yet.
- **`gatewayclient` doesn't thread `ctx` into `SubmitTransaction`** — a cross-component dependency
  this run doesn't own; closing it is a prerequisite for the timeout budget ever covering the whole
  request, not just the pre-submission phase.

### UX Design Requirements

None. This component has no user-facing surface — its only caller is talenta-core's own PHP
integration code, per `ADR-0022`.

### FR Coverage Map

FR-1:  Epic 1 - the core anchor-on-write trigger, dispatched through the pipeline
FR-2:  Epic 1 - canonical-JSON fidelity, enforced via json.RawMessage handling
FR-8:  Epic 1 - section validation, enforced structurally by the 5 static routes
FR-30: Epic 1 - the document field's encrypt-before-IPFS trigger point
FR-31: Epic 1 - the boundary rule that document content never gets logged or echoed

## Epic List

### Epic 1: talenta-core Can Reliably Anchor Any Profile-Section Change

After this epic, talenta-core's write paths can call the bridge for any of the 5 profile sections
and get back a machine-readable outcome (committed / partial_failure / rejected / error) they can
safely act on — without needing to understand Fabric, Go, or the underlying gateway client at all.
The bridge holds one warm connection, enforces one shared timeout budget, and never risks
corrupting a PAYROLL digest through unsafe JSON handling.

**FRs covered:** FR-1, FR-2, FR-8, FR-30, FR-31

## Epic 1: talenta-core Can Reliably Anchor Any Profile-Section Change

After this epic, talenta-core's write paths can call the bridge for any of the 5 profile sections
and get back a machine-readable outcome (committed / partial_failure / rejected / error) they can
safely act on — without needing to understand Fabric, Go, or the underlying gateway client at all.
The bridge holds one warm connection, enforces one shared timeout budget, and never risks
corrupting a PAYROLL digest through unsafe JSON handling.

### Story 1.1: Module Scaffold and Long-Lived Fabric Connection

As the platform team operating the integration bridge,
I want the service to start with one persistent Fabric Gateway connection and shut down cleanly,
So that every anchoring request reuses a warm connection instead of racing a dangling one.

**Acceptance Criteria:**

**Given** a fresh checkout of `integration-bridge/`
**When** it is built with `go build`
**Then** it compiles cleanly via `require`+`replace` pairs for all four `write-path-integration/*`
modules (AD-2), with `keystore` imported directly — not only transitively — to construct concrete
`EmployeeKeyStore`/`SaltStore`/`DocumentKeyStore` implementations

**Given** valid configuration
**When** the process starts
**Then** exactly one `*gatewayclient.GatewayClient` and one `Hooks` are constructed, once, before
the HTTP listener starts
**And** no other code path constructs a second `GatewayClient`

**Given** the process is running
**When** it receives `SIGTERM`/`SIGINT`
**Then** it stops accepting new requests, calls `GatewayClient.Close()` exactly once, and exits
**And** `Close()` is never called more than once

### Story 1.2: talenta-core Can Anchor a PERSONAL Section Change

As talenta-core's PERSONAL write path,
I want to call the bridge with a section update and get back a clear outcome,
So that I can safely act on it without understanding Fabric internals.

**Acceptance Criteria:**

**Given** a valid `POST /v1/profile-sections/PERSONAL` request with correct `X-Api-Key`/`X-Company-ID`
**When** it carries `employeeInternalID`, `userID`, a nested-JSON `newValue`, no `document`
**Then** the bridge dispatches to `Hooks.UpdatePersonalData` and returns `status: "committed"` with
the resulting `recordID`

**Given** a missing/incorrect `X-Api-Key` or `X-Company-ID`
**When** `[auth]` rejects it
**Then** the response carries `status: "error"` per AD-3's table, and no `Hooks` method is ever called

**Given** a malformed envelope (missing `employeeInternalID`, invalid `newValue`)
**When** `[validate]` rejects it
**Then** the response carries `status: "rejected"`, and no `Hooks` method is ever called

**Given** `newValue` contains a large-integer field
**When** `[validate]` parses the envelope
**Then** it is captured via `json.RawMessage` and passed through byte-for-byte — never
`map[string]interface{}`/`interface{}` decoding

**Given** `Hooks.UpdatePersonalData` returns `*writepaths.PartialFailureError`
**When** `[map-error]` classifies it
**Then** the response carries `status: "partial_failure"`, with `employeeInternalID`/`profileSection`
sourced from the Stage Contract struct — never from unwrapping the error

**Given** the configured dispatch timeout elapses mid-call
**When** the deadline fires
**Then** the response carries `status: "partial_failure"` (not `"timeout"`), and `[dispatch]` never
races a goroutine against the call — it blocks synchronously (AD-4)

**Given** any response, successful or not
**When** it is logged or returned
**Then** the actual bytes of `newValue`/`document` never appear in any log line or in `detail`

**Given** a request includes a `document` (base64-encoded, FR-30)
**When** the bridge decodes it and passes the raw bytes to `Hooks`
**Then** `Hooks.IPFS.EncryptAndAdd` is invoked and the resulting CID appears in the record's
`ipfsCIDs` — the bridge itself never persists, forwards, or logs the decoded document bytes
anywhere other than into that one call (FR-31)

### Story 1.3: talenta-core Can Anchor an EMPLOYMENT Section Change

As talenta-core's EMPLOYMENT write path (entirely separate from PERSONAL's own commit site),
I want the same calling contract,
So that transfers get identical anchoring guarantees with zero new integration work.

**Acceptance Criteria:**

**Given** a valid `POST /v1/profile-sections/EMPLOYMENT` request
**When** submitted with a correct envelope
**Then** the bridge dispatches to `Hooks.ApproveEmploymentTransfer` and returns the same `status`
contract as Story 1.2
**And** auth/validate/timeout/logging rules apply unchanged — this route only adds a dispatch
target to the existing pipeline

### Story 1.4: talenta-core Can Anchor an EDUCATION Section Change

As talenta-core's EDUCATION write path (no approval gate, no `EmployeeDataRequest` envelope),
I want the same contract,
So that a direct write still anchors correctly.

**Acceptance Criteria:**

**Given** a valid `POST /v1/profile-sections/EDUCATION` request
**When** submitted with a correct envelope
**Then** the bridge dispatches to `Hooks.RecordEducationHistory` and returns the same status
contract as Story 1.2
**And** no bridge behavior differs for the missing approval gate — the bridge has no concept of
PHP-side approval state

### Story 1.5: talenta-core Can Anchor an ADDITIONAL Section Change

As talenta-core's ADDITIONAL write path (raw SQL on the PHP side, not ActiveRecord),
I want the same contract,
So that family/dependents changes anchor regardless of PHP-side persistence mechanics.

**Acceptance Criteria:**

**Given** a valid `POST /v1/profile-sections/ADDITIONAL` request
**When** submitted with a correct envelope
**Then** the bridge dispatches to `Hooks.ApproveFamilyDataChange` and returns the same status
contract as Story 1.2
**And** behavior is identical regardless of PHP-side raw-SQL vs. ActiveRecord — the bridge only
ever sees the HTTP request

### Story 1.6: talenta-core Can Anchor a PAYROLL Section Change — Completing All Five Routes

As talenta-core's PAYROLL write path (sharing its PHP commit site with PERSONAL),
I want a dedicated route with a digest that exactly matches what was stored,
So that bank-account numbers never silently lose precision.

**Acceptance Criteria:**

**Given** a `POST /v1/profile-sections/PAYROLL` request where `newValue` has a bank-account number
greater than 2^53
**When** `[validate]` captures it via `json.RawMessage`
**Then** the digit string reaching `Hooks.UpdatePayrollBankAccount` is byte-identical to what
talenta-core sent — verified by a test asserting no `float64` conversion occurred anywhere in the
path

**Given** all five routes now exist
**When** any other path or a 6th section name is requested
**Then** the server returns 404 — FR-8 satisfied by construction, not an app-level check

**Given** PERSONAL and PAYROLL are dispatched from two separate bridge routes
**When** both are exercised
**Then** they remain two independent operations at the bridge's API, even though talenta-core's own
PHP happens to call both from one commit function (AD-5)

# HRIS PII-Anchoring Prototype on Hyperledger Fabric

A DSRM (Design Science Research Methodology) thesis prototype: a permissioned Hyperledger Fabric
2.5 network anchors tamper-evident integrity proofs of an HRIS platform's profile-section changes
— **zero bytes of PII on-chain**, by construction and by test. Three orgs (the platform, a
per-tenant enterprise-client org, and a read-only auditor org), channel-per-tenant isolation, and
an off-chain salted-HMAC digest scheme so nothing that reaches the ledger is reversible to the
underlying data.

Every claim in this repository is backed by a live test against a real, running instance of this
stack, or is explicitly disclosed as unverified/open — nothing here is asserted from design intent
alone. `agent-suite/06-roadmap/implementation-backlog.md` is the single authoritative,
evidence-linked record of what was built, what passed, and what didn't.

## Start here

| Doc | What it's for |
|---|---|
| [`docs/QUICKSTART.md`](docs/QUICKSTART.md) | Bring up the whole stack from scratch — network, chaincode, IPFS cluster, first real write |
| [`docs/CODEBASE-MAP.md`](docs/CODEBASE-MAP.md) | Where every piece lives, how to extend it, and a troubleshooting table of real defects already found and fixed |
| [`docs/REPRODUCING-RESULTS.md`](docs/REPRODUCING-RESULTS.md) | Re-run any test suite and regenerate the evaluation numbers yourself |
| [`docs/CALIPER-IMPLEMENTATION.md`](docs/CALIPER-IMPLEMENTATION.md) | How the Caliper benchmark workspace is built and wired — versions, bind target, the mutual-TLS patch, and its conformance against upstream Caliper |

## Code

| Path | What it is |
|---|---|
| [`fabric-network/`](fabric-network/) | Network topology, the chaincode (smart contract), and the Go tools that drive real Fabric bring-up/deployment |
| [`write-path-integration/`](write-path-integration/) | Off-chain digest/identity/erasure logic, the Fabric Gateway client, and the write-path hooks (Go workspace, 4 modules) |
| [`ipfs-cluster/`](ipfs-cluster/) | Dev-grade 2-node IPFS private swarm for encrypted supporting documents |
| [`integration-bridge/`](integration-bridge/) | Standalone Go service hosting the Fabric Gateway client for the HRIS platform's real write paths — closes grounding gap G-10 (ADR-0022). One HTTP route per profile section (`PERSONAL`/`EMPLOYMENT`/`EDUCATION`/`ADDITIONAL`/`PAYROLL`), all five sharing one `[auth]→[validate]→[dispatch]→[map-error]→[respond]` pipeline |

## Design documentation (`agent-suite/`)

| Path | What it is |
|---|---|
| [`06-roadmap/implementation-backlog.md`](agent-suite/06-roadmap/implementation-backlog.md) | Every build item, its status, and its evidence — the source of truth |
| [`06-roadmap/test-strategy.md`](agent-suite/06-roadmap/test-strategy.md) | The full test pyramid plan (unit/integration/contract/performance/security) and its control-coverage matrix |
| [`05-adr/`](agent-suite/05-adr/) | Architecture Decision Records — the ratified design choices, including a few technical-correction/disclosure notes added after real implementation surfaced a gap |
| [`08-security/security-architecture.md`](agent-suite/08-security/security-architecture.md) | STRIDE threat model and control mapping |
| [`11-execution/grounding-gaps.md`](agent-suite/11-execution/grounding-gaps.md) | Every open question, assumption, and disclosed limitation, tracked by ID |
| [`_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-02/prd.md`](_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-02/prd.md) | The ratified PRD — success predicates, functional requirements, the compliance matrix |
| [`_bmad-output/planning-artifacts/architecture/architecture-fabric-hris-2026-08-08/ARCHITECTURE-SPINE.md`](_bmad-output/planning-artifacts/architecture/architecture-fabric-hris-2026-08-08/ARCHITECTURE-SPINE.md) | `integration-bridge/`'s architecture spine — the pipeline paradigm, the five architecture decisions (AD-1 through AD-6), and what's still deferred |
| [`_bmad-output/planning-artifacts/epics.md`](_bmad-output/planning-artifacts/epics.md) | `integration-bridge/`'s epic and its 6 stories, each with acceptance criteria |
| [`_bmad-output/implementation-artifacts/sprint-status.yaml`](_bmad-output/implementation-artifacts/sprint-status.yaml) | `integration-bridge/`'s own story-by-story build record — separate tracking from `implementation-backlog.md` above, since this work was built via a different (BMAD epic/story) workflow |

`agent-suite/context/` holds additional internally-grounded technical reference. Almost all of it
follows the same generic register as everything else in this list (no real company/product names,
even where the underlying research was done against a real codebase — see that directory's own
files for the disclosed reasoning). One file is a deliberate, explicitly-disclosed exception,
naming real integration specifics for practical engineering use rather than the generic register —
that file says so prominently at its own top; nothing else in this repository does the same.

## Test results and evaluation (`qa-tests/`)

| Path | Covers |
|---|---|
| `unit-coverage-report.md`, `integration-coverage-report.md`, `contract-coverage-report.md` | Unit, integration, and contract-property test coverage |
| `performance/RESULTS.md`, `performance/QA6-RESULTS.md` | Real Hyperledger Caliper throughput/latency numbers and MVCC contention measurements — replacing every placeholder figure this design started with |
| `security/` | The full security test suite: a full-ledger confidentiality scan, tamper detection across every profile section, transport/revocation checks, key-domain verification, and SAST |
| `evaluation/QA7-dsrm-evaluation-p1-p4.md` | The four ratified success predicates, each verdict backed by cited live evidence |
| `evaluation/QA8-thesis-numbers-report.md` | The real numbers meant to replace this prototype's thesis's placeholder performance figures |

## Deployment artifacts (`deploy/`)

Helm charts, provisioning runbooks, a CI/CD pipeline spec, and observability dashboards — all
**generic, reviewable design artifacts only**. None of it has been applied to any real cluster,
CI system, or HSM; each subdirectory's own `README.md` says so explicitly and discloses what would
need resolving before a real deployment could use it.

## Status

Every backlog phase (`NET`, `CC`, `REC`, `INT`, `DEP`, `QA`) is closed as of 2026-08-07 — done,
honestly partial, or explicitly and openly blocked. Nothing is marked passing without live
evidence. See `implementation-backlog.md` for the row-by-row record, and `grounding-gaps.md` for
what remains open.

`integration-bridge/`'s own Epic 1 (reliably anchoring any profile-section change from the HRIS
platform's real write paths) is separately done as of 2026-08-09 — all 6 stories, all 5
profile-section routes, 88 tests passing. Every story went through the same adversarial code-review
process before being marked done; two real, disclosed limitations came out of it
(`grounding-gaps.md` G-35, G-36) rather than being silently claimed as covered. None of its
`//go:build integration` tests have executed against a live network yet — see `sprint-status.yaml`'s
recorded action items and `epic-1-retro-2026-08-09.md` for the full retrospective.

A second, separately-tracked epic set (4 epics, 15 stories — see `_bmad-output/planning-artifacts/`)
covers the other side of this integration: the HRIS platform's own application code now has a real
trigger that calls `integration-bridge/`'s existing routes automatically on every profile-section
write, instead of requiring a manual caller. As of 2026-08-13, all 15 stories (tf-1.1 through
tf-4.3) are implemented and in review — see `docs/QUICKSTART.md` §12 for the design shape and its
disclosed open gap (retryable-vs-permanent error classification). What's landed:

- All five profile-section write actions (not just one) now enqueue the trigger.
- A per-record pseudonym (`recordIdentity`) so repeating sub-records (family members, education
  entries, etc.) anchor and verify independently instead of colliding under one employee pseudonym.
- An on-demand integrity-verification route and an independent, timestamp-derived reconciliation
  sweep (deliberately never trusting the trigger's own success/failure log, so a suppressed trigger
  can't blind it) — scoped to the two domains that actually carry a last-modified signal, with the
  rest disclosed as a tracked gap rather than silently narrowed.
- Encrypted-file-backed persistence (both for `integration-bridge`'s salt/key material and for the
  HRIS platform's own dead-lettered jobs) replacing what was previously in-memory or log-only state,
  plus operator console commands to list, inspect, and re-drive a dead-lettered job by its
  correlation ID without losing or duplicating data.
- Real backlog-depth observability (Datadog StatsD counters on every anchoring attempt/success/
  dead-letter, plus a gauged dead-letter backlog) and a documented, reviewable alert threshold —
  absorbing the scope of a never-implemented earlier story (tf-1.6) once that gap was found.

Story tf-4.3's other half — a benchmark on dedicated, non-shared, non-laptop hardware, replacing
the disclaimed dev-laptop p95≈26.7s figure — genuinely did not close; no such hardware is available
in this environment, and this is disclosed rather than papered over with another dev-environment
number (`grounding-gaps.md` G-41). Story-level test execution for the HRIS-platform-side stories was
blocked throughout by a pre-existing, confirmed-unrelated defect in that platform's own test
harness — documented, not silently skipped, in each story's own record.

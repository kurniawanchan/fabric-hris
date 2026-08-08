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

## Code

| Path | What it is |
|---|---|
| [`fabric-network/`](fabric-network/) | Network topology, the chaincode (smart contract), and the Go tools that drive real Fabric bring-up/deployment |
| [`write-path-integration/`](write-path-integration/) | Off-chain digest/identity/erasure logic, the Fabric Gateway client, and the write-path hooks (Go workspace, 4 modules) |
| [`ipfs-cluster/`](ipfs-cluster/) | Dev-grade 2-node IPFS private swarm for encrypted supporting documents |

## Design documentation (`agent-suite/`)

| Path | What it is |
|---|---|
| [`06-roadmap/implementation-backlog.md`](agent-suite/06-roadmap/implementation-backlog.md) | Every build item, its status, and its evidence — the source of truth |
| [`06-roadmap/test-strategy.md`](agent-suite/06-roadmap/test-strategy.md) | The full test pyramid plan (unit/integration/contract/performance/security) and its control-coverage matrix |
| [`05-adr/`](agent-suite/05-adr/) | Architecture Decision Records — the ratified design choices, including a few technical-correction/disclosure notes added after real implementation surfaced a gap |
| [`08-security/security-architecture.md`](agent-suite/08-security/security-architecture.md) | STRIDE threat model and control mapping |
| [`11-execution/grounding-gaps.md`](agent-suite/11-execution/grounding-gaps.md) | Every open question, assumption, and disclosed limitation, tracked by ID |
| [`_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-02/prd.md`](_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-02/prd.md) | The ratified PRD — success predicates, functional requirements, the compliance matrix |

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

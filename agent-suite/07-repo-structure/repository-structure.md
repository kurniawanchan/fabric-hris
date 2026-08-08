# Repository Structure (Deliverable 9)

A scalable layout spanning the three concerns the brief names: **Claude tooling**, the **Fabric +
service prototype**, and **documentation/design**. It separates *design-time* (this package — exists
now), *build-time* (the prototype — created post-G9), and *external* (the existing HRIS application's
own repo — this workspace never forks or rehosts it; ADR-0002 keeps Fabric an anchor/audit overlay,
never source-of-truth, and ADR-0014's in-band call site lives inside that repo's own write path, not
duplicated here).

> **Register note.** Per the AUTHORITY NOTICE confidentiality framing (`project-context.md`), the
> external HRIS application is referred to only generically below — never by its real product/company
> name, and never by literal real table/column names.

```
hyperledger/fabric-hris/                    # this workspace (design-first)
├── AGENTS.md                               # workspace instructions (HLF 2.5 LTS; design-before-code)
├── agents-guide.md                         # the originating brief
├── fabric-skill-suite/                     # DESIGN-TIME · the 8-skill HLF suite + corpus (reused, not rebuilt)
│   ├── skills/fabric-*/                     #   (core, chaincode-dev, network-architect, identity-security,
│   │                                        #    operations, performance, security-review, troubleshooting)
│   ├── corpus/fabric-docs-2.5/              #   pinned grounding corpus
│   └── docs/AUTHORING-CONTRACT.md           #   authoring governance
├── agent-suite/                            # DESIGN-TIME · this package (the AI engineering org + design)
│   ├── 00-architecture/ (+ solution/, diagrams/)   # multi-agent arch + prototype solution design
│   ├── 01-agents/ (agent-catalog + specs/)  # roster + specs
│   ├── agents/                              # the operational agent files (→ .claude/agents/)
│   ├── context/                             # knowledge docs (Fabric/Go/PHP/blockchain/security/DSRM)
│   ├── skills/                              # new org-specific skills (→ .claude/skills/)
│   ├── 04-mcp/ 05-adr/ 06-roadmap/ 07-repo-structure/ 08-security/ 09-review/ 10-risk/ 11-execution/
│   └── prompts/hlf-orchestrator.md          # the orchestrator Workflow definition
├── .claude/                                # CLAUDE TOOLING · live installs
│   ├── agents/  → symlinks to agent-suite/agents/*        (fabric-architect, fabric-engineer, …)
│   └── skills/  → symlinks to fabric-skill-suite + agent-suite/skills/*
│
├── fabric-network/                         # BUILD-TIME (post-G9) · Fabric artifacts
│   ├── chaincode/employeeprofilerecord/     #   Go chaincode — the `EmployeeProfileRecord` asset +
│   │                                        #   `RecordProfileSection`/verify contract (ADR-0011,
│   │                                        #   ADR-0020; interface: api-contracts.md)
│   ├── network/ (configtx, crypto-config, compose/helm)   #   three-org, channel-per-tenant topology
│   │                                        #   (ADR-0012/ADR-0013) — `Org1` (platform) +
│   │                                        #   `OrgClient-<tenantID>` (one per tenant) + `Org3`
│   │                                        #   (auditor); no PDC config anywhere (ADR-0015)
│   └── channel-artifacts/                   #   per-tenant genesis/channel tx (git-ignored if secret)
│
├── ipfs-cluster/                           # BUILD-TIME (post-G9) · IPFS Private Cluster infra (ADR-0016)
│   └── cluster-config/                      #   swarm-key-gated peer config + pinning policy — cluster
│                                            #   operator identity and node count are recommendation-only,
│                                            #   not yet ratified (`fabric-network-design.md` §6)
│
├── write-path-integration/                 # BUILD-TIME (post-G9) · the in-band anchor call site (ADR-0014).
│   │                                        #   NOT a standalone "anchor-service" deployable — recording is
│   │                                        #   synchronous, in the same request as the section write, no
│   │                                        #   queue. WHICH PROCESS HOSTS THIS IS STILL OPEN (gap G-10,
│   │                                        #   reopened) — it may end up vendored into the existing HRIS
│   │                                        #   application's own repo instead of living here as a separate
│   │                                        #   component; these subfolders name the *code*, not a fixed
│   │                                        #   deployment boundary.
│   ├── gateway-client/                      #   Fabric Gateway client wrapper — Submit/Evaluate,
│   │                                        #   off-chain DataHash/HMAC computation (ADR-0011/ADR-0020);
│   │                                        #   plaintext section content is never a chaincode argument
│   ├── relational-access/                   #   read layer over the EXISTING operational relational
│   │                                        #   database — fetches the current section values needed to
│   │                                        #   compute DataHash, pre-hash (ADR-0017: no migration, no
│   │                                        #   new store, this package only reads/wraps it)
│   └── ipfs-client/                         #   encrypt-before-add / CID wrapper, invoked only when a
│                                        #   section write carries a supporting document (ADR-0016);
│                                        #   talks to ipfs-cluster/ above
│
├── experiments/caliper/                     # DSRM evaluation runs. Network-config + workload configs that
│   │                                        #   invoke the REUSABLE harness already in
│   │                                        #   `fabric-skill-suite/skills/fabric-performance/` — fix its
│   │                                        #   `SUT_BIND` (2.2→2.5) first (ADR-0018); `hotKeyFraction` for
│   │                                        #   MVCC-contention runs already exists in that harness, so this
│   │                                        #   directory is configuration, not new harness code
│   └── results/                             #   raw Caliper reports + evaluation write-ups feeding QA-8
│
├── qa-tests/                                # executable tests once built (unit/integration/contract/perf/security)
│
└── (external, this workspace neither forks nor rehosts it)
    ../<existing-hris-application>/          #   the existing HRIS platform's own repo — system-of-record;
                                              #   ADR-0014's in-band call site lives inside its own write
                                              #   path, not duplicated in this workspace. Referred to only
                                              #   generically in every written artifact (AUTHORITY NOTICE).
```

## Rules

- **No secrets in git** (AGENTS.md): crypto material, MSP dirs, Fabric/TLS keys, per-record salts,
  `employeeKey_i` values, `KEY_EMPLOYEE` material, IPFS swarm keys, connection profiles with embedded
  certs → out of version control (K8s secrets / HSM refs).
- **Design-time vs build-time:** everything under `agent-suite/` + `fabric-skill-suite/` is design/tooling
  and exists now; `fabric-network/`, `ipfs-cluster/`, `write-path-integration/`, `experiments/`, `qa-tests/`
  are created in DSRM activity 4 (post-approval) from the backlog — none of them exist yet.
- **The external HRIS application repo is not forked or rehosted here** (ADR-0002; ADR-0006 is
  superseded by ADR-0014). The pre-reconciliation "read-back endpoint" (old INT-1) and its Kafka
  producer are retired: under in-band recording the section values needed for hashing are already
  in-process, in the same request, at write time (ADR-0014) — there is no separate service to read them
  back from. **0 shared-relational-DB schema changes** (G-16) — ADR-0017 keeps that database external
  and unmigrated; `relational-access/` above only reads it.
- **Skills self-contained** (AGENTS.md): each `fabric-skill-suite/` skill works standalone; new
  agent-suite skills cross-reference context docs, not each other's internals.

## Scaling notes

- Adding a tenant = a new `OrgClient-<tenantID>` MSP + a new channel config under
  `fabric-network/network/` (ADR-0012/ADR-0013) — no PDC config, since none exists anywhere in this
  design. The chaincode package itself does not change; its lifecycle (approve-by-Org1 +
  approve-by-`OrgClient-<tenantID>` + commit) repeats per tenant, `O(N)` operations for `N` tenants, not
  `O(1)` (`fabric-network-design.md` §5) — no code change to `write-path-integration/`.
- Adding a new profile section to anchor = extending the `profileSection` enum + a chaincode case + a
  test; the `EmployeeProfileRecord` is **section-agnostic** by design (`data-model.md`). There are no
  discrete anchored "events" in this design — contrast the retired `changeType`/`AnchorRecord` model,
  which was event-agnostic for a different reason (event-based anchoring itself is superseded, PRD §4).
- The agent-suite is additive: a new capability = one agent-role binding + its context doc + catalog row.

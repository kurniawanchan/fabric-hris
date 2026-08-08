# Architecture Decision Records — index

The load-bearing decisions for the HRIS-on-Fabric prototype. Each ADR follows
[`_TEMPLATE.adr.md`](_TEMPLATE.adr.md): **Status is Accepted, Rejected, or Superseded by ADR-NNNN**
(three terminal values, nothing else), every decision names a rejected alternative with the reason,
and — because the project runs in **proceed-with-assumptions** mode — every ADR records the
**grounding gap** it rests on, so it can be revisited if the real requirement (Jira HLF-6 /
Confluence, or a ratifying artifact such as the thesis + PRD below) later contradicts the assumption.

> **Reopen rule:** when a gap in [`../11-execution/grounding-gaps.md`](../11-execution/grounding-gaps.md)
> is closed by a real artifact, re-open every ADR whose "Rests on" names it (see the gap→ADR column).
> An ADR's **body** (Context/Decision/Alternatives/Consequences) is otherwise **immutable** — a
> changed decision is recorded as a **new** ADR that supersedes the old one; the old ADR's **Status**
> field is then flipped to `Superseded by ADR-NNNN` (see `_TEMPLATE.adr.md`) rather than left silently
> `Accepted` for a decision no longer in force.

> ✅ **Reconciliation executed (2026-08-06).** "The thesis wins" (2026-08-02/03) plus the real-repo
> grounding pass (2026-08-06) has now superseded **six** of the original ten ADRs. Ten new ADRs
> (0011–0020) formalize the reconciled design — nine authored by three specialist agents dispatched
> in parallel (`fabric-architect`, `fabric-engineer`, `security-architect`), one (0018, performance)
> authored directly to close a dispatch gap. The **index below is now current** — every Status cell
> reflects the ADR body on disk, not a pending intention. Two residual notes:
> - **0017** (relational tier) and its dependency **OQ-2/S-2** are still a sponsor-pending
>   recommendation, not a closed decision — see PRD §12 OQ-2 and `rencana-rekonsiliasi.md` §2.
> - Pre-2026-08-06 ADR **bodies** (0002, 0005, 0007, 0008 — untouched, still Accepted) predate the
>   generic-register confidentiality rule in `_bmad-output/planning-artifacts/project-context.md`
>   and may still name the real system literally; this is a known, accepted gap (ADR immutability
>   forbids editing an Accepted body in place) — not a live leak, since these are internal
>   engineering ADRs, not thesis-adjacent material.

## Index

| ADR | Decision (short) | Status | Rests on | Context |
|-----|------------------|--------|----------|---------|
| [0001](ADR-0001-onchain-salted-commitments.md) | Anchor **only salted hash commitments** of PII-change events on-chain; PII plaintext never on-chain | **Superseded by ADR-0011** | G-02 | BLOCKCHAIN-DATA-MODEL, FABRIC-PRIVATE-DATA, CRYPTOGRAPHY |
| [0002](ADR-0002-fabric-audit-anchor-overlay.md) | Fabric is an **audit/anchor overlay**; operational DB stays system-of-record | Accepted | G-09 | BLOCKCHAIN-INTEGRATION |
| [0003](ADR-0003-consortium-org-topology.md) | **Two orgs** (HR + Audit) for the prototype; Employee/Regulator deferred | **Superseded by ADR-0012** | G-04 | FABRIC-MSP, FABRIC-CHANNELS |
| [0004](ADR-0004-multi-tenancy-channel-strategy.md) | **Single channel + PDC** for field confidentiality; channel-per-tenant deferred | **Superseded by ADR-0013** | G-03 | FABRIC-CHANNELS, FABRIC-PRIVATE-DATA |
| [0005](ADR-0005-identity-role-mapping.md) | Map HRIS roles → **X.509 OUs/attributes** read via chaincode CID (ABAC) | Accepted | G-11 | FABRIC-IDENTITY, FABRIC-POLICIES |
| [0006](ADR-0006-anchor-service-host.md) | A **new thin Go anchor-service** consumes `employee_info` Kafka + hosts the Gateway client | **Superseded by ADR-0014** | G-10 | BLOCKCHAIN-INTEGRATION, GO-ARCHITECTURE |
| [0007](ADR-0007-state-database-leveldb.md) | **LevelDB** for the prototype; CouchDB deferred to rich-query need | Accepted | G-08 | FABRIC-WORLD-STATE |
| [0008](ADR-0008-zkp-idemix-scope.md) | **Defer ZKP/Idemix** to an evaluation-phase exploration; not core | Accepted (defer) | G-07 | ZERO-KNOWLEDGE-PROOF, FABRIC-IDENTITY |
| [0009](ADR-0009-key-management-separation.md) | Fabric MSP/HSM keys **separate** from the app AES PII keys (originally two domains) | **Superseded by ADR-0019** | G-15 | CRYPTOGRAPHY, FABRIC-CA |
| [0010](ADR-0010-erasure-purge-privatedata.md) | Right-to-erasure via **PurgePrivateData/blockToLive** — only the hash survives | **Superseded by ADR-0015** | G-06 | PRIVACY-BY-DESIGN, FABRIC-PRIVATE-DATA |
| [0011](ADR-0011-per-section-digest-scheme.md) | Per-section `DataHash` (salted digest) + keyed-HMAC pseudonymous identifiers | **Superseded by ADR-0021** (identifier-derivation clause only) | G-02 (closed) | data-model.md, PRD §5.2 |
| [0012](ADR-0012-consortium-three-org-topology.md) | **Three-org** consortium — platform (2 peer + 3 Raft), enterprise-client org(s), read-only auditor | Accepted | G-04 (closed; regulator-org line open) | FABRIC-MSP, FABRIC-ORDERING, FABRIC-CHANNELS |
| [0013](ADR-0013-channel-per-tenant.md) | **Channel-per-tenant**; per-tenant client org resolved as `OrgClient-<tenantID>` | Accepted | G-03 (closed; Org3 per-tenant scope open) | FABRIC-CHANNELS, FABRIC-MSP |
| [0014](ADR-0014-in-band-recording.md) | **In-band recording** from the HRIS write path; 0 Kafka topics, 0 anchor-service deployables | Accepted | G-10 (reopened; closed by ADR-0022) | BLOCKCHAIN-INTEGRATION |
| [0015](ADR-0015-crypto-shred-erasure.md) | Right-to-erasure via **crypto-shred**: delete DB field + destroy `KEY_EMPLOYEE` + salt + `employeeKey_i` — no PDC | Accepted | G-06 (closed) | PRIVACY-BY-DESIGN |
| [0016](ADR-0016-ipfs-private-cluster.md) | **IPFS Private Cluster**, swarm-key gated, encrypt-before-add; erasure = key destruction, never object deletion | Accepted | G-02 (partial — cluster-operator topology not yet gap-tracked) | — (outside pinned Fabric corpus) |
| [0017](ADR-0017-relational-tier.md) | Keep the **existing operational relational DB** authoritative for the five sections; no migration | Accepted (working recommendation) | OQ-2 / **S-2 — sponsor confirmation pending** | BLOCKCHAIN-INTEGRATION, BLOCKCHAIN-DATA-MODEL |
| [0018](ADR-0018-performance-evaluation-in-scope.md) | Performance evaluation **in scope**; reuse the Caliper v0.5 harness with the `SUT_BIND`/2.5 fix | Accepted | G-08 (reopened) | fabric-performance skill |
| [0019](ADR-0019-key-domain-amendment.md) | **Four** key domains (MSP/TLS, app AES, `KEY_EMPLOYEE`, identifier pseudonymization) — custody & destroy-ability adjudicated | **Superseded by ADR-0021** (Domain C only) | G-15 | CRYPTOGRAPHY, FABRIC-CA |
| [0020](ADR-0020-chaincode-contract-hashing-boundary.md) | Chaincode contract + hashing boundary — digest **off-chain**, plaintext never a chaincode argument, Evaluate-only verify | Accepted | G-02 (closed) | api-contracts.md |
| [0021](ADR-0021-employeekey-independent-random-no-master-key.md) | `employeeKey_i` **independently random**, no `pseudonymKey` master key — closes T16b (retroactive erasure-reversal risk) | Accepted | — (human decision, closes a disclosed residual risk) | security-architecture.md T16a/T16b |
| [0022](ADR-0022-integration-bridge-gateway-client-host.md) | **New standalone Go bridge**, sibling to EMS — hosts the Gateway client, called synchronously from talenta-core's 4 real commit sites via the `BaseEmsService` convention | Accepted | G-10 (closed) | REAL-INTEGRATION-TRIGGER-FLOW.md, BLOCKCHAIN-INTEGRATION |

## Gap → ADR (reopen map)

`G-02 → 0001 (superseded) / 0011 / 0016 (partial) / 0020` · `G-03 → 0004 (superseded) / 0013` ·
`G-04 → 0003 (superseded) / 0012` · `G-06 → 0010 (superseded) / 0015` · `G-07 → 0008` ·
`G-08 → 0007 / 0018 (reopened)` · `G-09 → 0002` · `G-10 → 0006 (superseded) / 0014 (reopened)` ·
`G-11 → 0005` · `G-15 → 0009 (superseded) / 0019`

*(Twenty-one ADRs now form the design spine. Next ADR number = 0022. Ten new ADRs (0011–0020) were
authored 2026-08-06 executing `rencana-rekonsiliasi.md` Gelombang 3 against real repo grounding; a
21st (ADR-0021) closes T16b — a residual risk ADR-0019 itself disclosed the same day — by explicit
human decision among three presented options. See `prd.md` §11.3 and §5.2 for the full rationale.)*

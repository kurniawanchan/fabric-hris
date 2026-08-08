---
project_name: 'fabric-hris'
user_name: 'Mr. Chan'
date: '2026-08-06'
sections_completed: ['authority_notice', 'technology_stack', 'domain_architecture_rules', 'service_integration_rules', 'go_conventions', 'testing_rules', 'known_open_items', 'critical_dont_miss']
status: 'complete — reconciled 2026-08-06 against prd-fabric-hris-2026-08-02/prd.md'
rule_count: 34
optimized_for_llm: true
---

# Project Context for AI Agents

_This file contains critical rules and patterns that AI agents must follow when implementing code in this project. Focus on unobvious details that agents might otherwise miss._

---

## ⚠️ AUTHORITY NOTICE (2026-08-06) — READ FIRST

**The thesis wins.** Where this file or any `agent-suite/` document conflicts with the ratified
PRD below, **the PRD is authoritative** — this file and `agent-suite/` are being reconciled toward
it, not the other way round. Ratified 2026-08-02/03 by Chandra Kurniawan (human decision).

- **Ratifying source:** `_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-02/prd.md`
  (status: final) + `errata-tesis.md` + `rencana-rekonsiliasi.md` (40-step execution plan, 6 waves).
- **Citation class for this ratification:** `[thesis: BAB <roman> §<n>]` when citing the source
  thesis directly; `[prd: §<n>]` when citing the ratified PRD.
- **STOP-LIST — do not build against these, they are superseded/dead:**
  - Event-based anchoring surface (`EVENT_UPDATE_PERSONAL`, `EVENT_ADD`, `EVENT_UPDATE_EMPLOYMENT`,
    `EVENT_RESIGN`) — superseded by **section-based** anchoring (PRD §4).
  - Kafka `employee_info` consumer + standalone anchor-service — superseded by **in-band recording**
    from the HRIS profile-write path (PRD §11.3 ADR-0014; rencana-rekonsiliasi.md Gelombang 3 #17).
  - Single-channel + composite-key (`companyId` prefix) tenancy — superseded by **channel-per-tenant**
    (PRD §5.1, §11.3 ADR-0013).
  - Two-org consortium (HROrgMSP/AuditOrgMSP) — superseded by **three orgs** (platform, enterprise
    client, auditor) with client-operated Org2 peer (PRD §5.1, §11.3 ADR-0012).
  - PDC / `PurgePrivateData` / `blockToLive` erasure — superseded by **crypto-shred**: delete
    operational-DB field + destroy `KEY_EMPLOYEE` + destroy *salt* + `employeeKey_i` (PRD §5.2, §7
    FR-25..29, §11.3 ADR-0015).
  - Bare `SHA-256(salt‖changedFields)` commitment scheme with single shared `pseudonymKey` — superseded
    by **§5.2's two-tier construction**: `DataHash = SHA-256(salt‖JSON kanonik section)`,
    `EmployeeID/UpdatedBy = HMAC-SHA256(employeeKey_i, …)` where `employeeKey_i` derives per-employee
    from `pseudonymKey` (PRD §5.2 — this is the **one exception** where the PRD itself deliberately
    follows `ADR-0001`/`data-model.md` construction logic rather than the thesis's original unsalted
    hash, per errata **E-1**).
  - "Performance evaluation out of scope" (`context/DSRM.md §4` as originally written) — reopened;
    performance is **in scope** (PRD §3.1, §10).
  - Gate **G8 and G9 are VOID** as of this ratification (PRD §11.1) — do not treat prior G8/G9 sign-off
    as current authorization. A sponsor decision (S-1, PRD/rencana-rekonsiliasi.md §2) on the
    **G8a/G8b split** is required before "no code before G9" is enforceable again in its old form.
- **Confidentiality framing (2026-08-06):** this reconciliation is now grounded against the real
  `talenta-core` Employee module for technical accuracy, but every **written artifact** (docs,
  diagrams, code comments) must stay in the **generic register** the InfoSec-revised thesis uses —
  "platform SaaS HRIS multi-tenant komersial" / "penyedia platform" — never "Talenta", "Mekari", or
  literal real table/column names. Real names are for internal grounding only, never for the page.

---

## Technology Stack & Versions

- **Distributed ledger:** Hyperledger Fabric 2.5.x (per `agent-suite/context/FABRIC-*.md`)
- **State database:** LevelDB (ADR-0007) — NOT CouchDB. Do not design rich queries requiring CouchDB indexes.
- **Chaincode/service language:** Go (per `agent-suite/context/GO-*.md` — architecture, concurrency, conventions, database, deployment, error-handling, performance, security, testing)
- **Calling application:** the existing HRIS application's employee-update write path (generic
  register — do not name the real platform in any written artifact, PRD §11 confidentiality framing).
  **Which process/language hosts the Fabric Gateway client is an OPEN QUESTION** (gap G-10, reopened
  by ADR-0014) — do not assume it is embedded in the calling application itself; a thin Go
  component is the more likely shape given Fabric's SDK support, but this is not yet decided.
- **Document storage:** IPFS Private Cluster, swarm-key gated, encrypt-before-add (ADR-0016) —
  supporting documents only, never profile-section content.
- **Operational database:** the existing relational database stays authoritative for the five
  profile sections; **no migration** (ADR-0017 — working recommendation, sponsor confirmation
  pending, OQ-2/S-2).
- **Identity/crypto:** Fabric MSP + optional Idemix/ZKP, scope decided in ADR-0008 (ZKP/Idemix is scoped, not full — check ADR-0008 before assuming ZKP everywhere)
- **No frontend stack** — this repo does not own UI; `design-artifacts/` is WDS product-design output for a separate consuming team

## Critical Implementation Rules

### Domain & Architecture Rules (non-negotiable, from ADRs — current as of 2026-08-06)

- **0-PII on-chain, always.** Only `DataHash = SHA-256(salt ‖ JSON-kanonik(section))` (per profile
  section, per version) and the HMAC-derived `EmployeeID`/`UpdatedBy` identifiers may go on-chain.
  Never write plaintext, masked values, or reversible ciphertext on-chain — not even "just for now"
  or "just in dev." (ADR-0011)
- **Salt discipline:** every `DataHash` needs a per-record salt ≥128 bits from a CSPRNG, unique per
  record, stored **off-chain only**. Never derive salt deterministically from the content itself, or
  the digest becomes brute-forceable. (ADR-0011)
- **`EmployeeID`/`UpdatedBy` derive from `employeeKey_i`, an independently-random per-employee
  secret — never from any master/tenant key.** `employeeKey_i` is CSPRNG-generated once per
  employee, stored off-chain, and used directly for `HMAC-SHA256(employeeKey_i, …)`. **There is no
  `pseudonymKey` in this design** — an earlier two-tier hierarchy (`pseudonymKey → employeeKey_i`)
  was superseded 2026-08-06 (ADR-0021) because a shared master key's compromise could retroactively
  reverse every completed erasure for a tenant (T16b). Never reintroduce a master key for
  identifiers — the accepted trade-off is that an accidentally-lost `employeeKey_i` is permanent
  (same class of risk as `KEY_EMPLOYEE` loss, below).
- **Erasure = crypto-shred, not ledger rewrite.** Right-to-erasure deletes the operational-DB field
  + destroys `KEY_EMPLOYEE` + destroys the employee's `DataHash` salts + destroys `employeeKey_i`.
  Never attempt to "delete" or mutate on-chain state to satisfy erasure. (ADR-0015)
- **No Private Data Collections anywhere in this design.** PDC, `PurgePrivateData`, and
  `blockToLive` are all **retired** — confidentiality and erasure are handled entirely by the
  key-domain model above, not by Fabric's private-data feature. Do not reintroduce a PDC without
  reopening ADR-0015.
- **State DB is LevelDB, not CouchDB** (ADR-0007) — never design chaincode around rich/JSON queries or CouchDB indexes.
- **Tenancy is channel-per-tenant, not single-channel.** Each tenant is mapped to its own Fabric
  channel; the world-state key drops the old `companyId` prefix (`("profile", employeeID,
  profileSection)`) because tenant scoping now happens at the channel level. (ADR-0013) Don't design
  a shared-channel, composite-key tenant boundary — that model is superseded.
- **Three organizations, not two.** Platform org (2 peers + 3-node Raft, all platform-operated),
  enterprise-client org (1 peer, **independently operated** — this independence is the premise
  behind the tamper-detection claim, P1), auditor org (1 read-only peer). Endorsement policy:
  `AND(Org1MSP.peer, Org2MSP.peer)`. (ADR-0012) **Open question:** whether multiple simultaneous
  tenants each get their own client org (`OrgClient-<tenantID>`) or share one — unresolved, low
  priority for a single-pilot-tenant prototype, must be resolved before real multi-tenant scale-out.
- **Fabric anchors on top of existing app-level encryption** — the existing AES-256-at-rest PII
  encryption model stays; Fabric does not replace it. This is a **separate key domain** from
  everything above — never derive one from the other.
- **Documents are encrypted before IPFS upload, and `unpin` is never treated as `delete`.**
  Supporting documents (not profile-section content) are encrypted with `KEY_EMPLOYEE` before
  upload to a swarm-key-gated IPFS Private Cluster; only the CID goes on-chain. Erasure of a
  document is **key destruction**, never object removal — any node that ever fetched a block may
  retain it. (ADR-0016)

### Service & Integration Rules

- **No standalone anchor-service. No Kafka.** Recording happens **in-band**, synchronously, from
  the HRIS application's own profile-section write path — not from a consumed event topic. The
  metadata-only `employee_info` topic (if it still exists elsewhere in the platform) physically
  cannot carry the EDUCATION/ADDITIONAL/PAYROLL payloads this design needs, which is *why* the
  event-based/Kafka path was retired, not merely deprioritized. (ADR-0014)
- **Moving to a synchronous write path is a live, unresolved availability question.** Under the old
  async/Kafka design, a Fabric outage only delayed anchoring. Under in-band recording, a Fabric
  outage sits directly in the HRIS write request unless a block/retry/degrade policy is designed.
  **This has no decided policy yet** — do not assume either "block until anchored" or "fire and
  forget" without checking whether that decision has since been made (`architect`/`backend-engineer`
  own this).
- **Plaintext profile-section content must never be a chaincode argument — submit or evaluate.**
  `DataHash` is computed **off-chain**, by whichever component holds the section values in memory to
  write them anyway. Verification is a **read-only** operation returning the stored
  `{DataHash, Version, Timestamp, UpdatedBy}` — the verifier recomputes and compares locally, never
  inside chaincode. (ADR-0020) This inverts the literal thesis text (BAB IV 4.2.3), which sent
  section data *into* `VerifyProfileIntegrity` — that form is superseded, not a model to follow.
- **Salt and `employeeKey_i` must never be returned by the verify endpoint.** Authorized
  verifiers (the employee, or an org with its own peer) obtain salt/key material through a
  **separate, audited, access-controlled channel** — never bundled into a verification response.
  (PRD §7 FR-14, FR-36)
- **Four key domains are never mixed:** (A) Fabric MSP/TLS signing keys, (B) application AES
  PII-at-rest key, (B′) `KEY_EMPLOYEE` (per-employee, IPFS document encryption), (C)
  `employeeKey_i` + `DataHash` salts (per-employee/per-record, identifier pseudonymization and
  content confidentiality). No domain's key may be derived from, wrap, or be wrapped by another's.
  A key used for anchoring must never also decrypt PII, and vice versa. (ADR-0019, ADR-0021)

### Go Conventions (matches the existing Go codebase, `agent-suite/context/GO-CONVENTIONS.md`)

- Package = feature directory, **snake_case**; interface files prefixed `i` (`irepository.go`); constructors are `NewX`; interface names bare (`Repository`), impl struct unexported (`repo`).
- **Never return bare `error`.** Use the `errs.Error` interface (`errs.New`, `errs.Wrap`, `errs.WrapWithErrorType`, etc.). Map Fabric SDK/gateway failures to `errs.WrapWithErrorType(..., errs.ErrorTypeInternalError)`; reserve 4xx for caller-input problems.
- Wrap errors **at the boundary you cross** (e.g., repository wraps raw GORM/SDK errors).
- Thread `context.Context` everywhere; use `*app.Context` when identity/company scoping is needed.
- goimports local-prefix = module path; stdlib → third-party → module imports, in that order.

### Testing Rules (design-only test plan today — build gated on S-1, see below)

- **ST-1 (confidentiality invariant) is P0 and gates everything** — no anchor write path merges
  until a full-ledger scan (world state + tx args + chaincode event payloads, **no PDC to scan
  anymore**) proves 0 bytes of PII plaintext or reversible derivative on-chain. This test **passes
  as a single, unsplit test** now that identifiers use independently-random `employeeKey_i` — do
  not reintroduce a scheme that would force splitting it into a satisfiable/unsatisfiable pair.
- **Commitment tests must use RFC-8785 JCS canonicalization** — writer and verifier must produce
  byte-identical input for equivalent JSON (key order, unicode, numbers). This is still an open
  build precondition (**PB-3/G-24**) — a specific library/version is not yet pinned.
- **`DataHash` is built from the whole section as of the write**, not a `{from,to}` delta —
  verification recomputes from the current section values + salt.
- **A manipulation scenario for the `ADDITIONAL` section must exist** — the thesis's own test suite
  (Tabel 4.4) never manipulated this section; without it, the "100% detection across five sections"
  claim (P1) is unproven for one of the five.
- **Tenant-isolation tests need a real second tenant and a real second peer to be reproducible** —
  a single-tenant test bed cannot exercise cross-tenant denial (P3), regardless of how correct the
  channel-per-tenant design is on paper.
- **Salt and `employeeKey_i` must never be returned by any API**, including verify endpoints — a
  leak of either defeats the confidentiality/erasure model.

### Known Open Items (ratified vs. still open — don't conflate the two)

**Ratified, closed — do not reopen without a new human decision:**
- **G-01 (objective)** and **G-05 (anchored surface)** — closed verbatim by PRD §2/§3 and §4 (human
  decision, 2026-08-02).
- **G-02 (field scope)** — closed by ADR-0011 (per-section `DataHash`).
- **OQ-1 / T16b (identifier key scheme)** — closed by ADR-0021 (2026-08-06, `employeeKey_i`
  independently random).
- **S-1 (G8a/G8b gate split)** — ✅ **decided 2026-08-06.** G8a (document consistency) PASSED;
  G8b (measured evaluation) build is **authorized**, item by item against
  `agent-suite/06-roadmap/implementation-backlog.md`'s Definition of Done. The "0 lines of
  prototype code" rule stays in force for G8a-type work; it does **not** apply to a tracked G8b
  item. See `agent-suite/00-architecture/quality-gates-and-approval.md` §5.5 for the full record.
- **S-2 (operational database, OQ-2)** — ✅ **decided 2026-08-06.** Keep the existing DB, no
  migration (ADR-0017 confirmed, no longer a "working recommendation").
- **S-3 (reconstruction claim, OQ-6)** — ✅ **decided 2026-08-06.** Narrowed claim confirmed
  (PRD UJ-1 already reflects this since 2026-08-03).
- **S-4 (PB-2/G-23)** — ✅ **decided 2026-08-06 — dissolved.** No anchor-service, no Kafka, no
  cross-service read-back surface exists to secure. `G-19`/`G-21`/`G-23` all closed; STRIDE `T9`/`T10`
  retired (not "likely dissolved" — formally retired).
- **S-5 (G9 re-approval)** — ✅ **decided 2026-08-06.** G9 is re-approved against the *current*
  design (not a resurrection of the void 2026-07-13 approval).

**Still genuinely open:**
- **PB-1 / G-18 (gateway header trust)** — tenant/user identity at the gateway is spoofable until
  independently verified; treat as an open risk. **Item-scoped** — blocks `QA-5`/`ST-2` only, not
  the whole build.
- **PB-3 / G-24 (canonicalization pin)** — RFC-8785 JCS is the *scheme*, but no specific
  library/version is pinned yet. **Item-scoped** — blocks `REC-1`/`QA-1`/`U-4` only.
- **G-25 (DPIA never conducted)**, **G-26 (Pasal 42 retention unmet by design)** — legal-compliance
  gaps found 2026-08-06, not technical debt; see PRD §9.3. Not build-blocking, but must not be
  silently claimed as satisfied either.
- **G-27 (Controller/Processor role undetermined)** — affects which UU PDP obligations bind which
  organization; see PRD §9.4.
- **`employeeKey_i` accidental-loss risk** — same class as the already-accepted `KEY_EMPLOYEE`-loss
  risk (OQ-5); both need the same backup-integrity discipline when designed (not yet).
- **This whole package is proceed-with-assumptions mode** for everything not listed above as
  closed — every requirement-derived decision not yet ratified is tagged `[ASSUMPTION]`; see
  `agent-suite/11-execution/grounding-gaps.md` for the full, current gap list. When extending design
  docs, preserve this tagging discipline rather than presenting assumptions as settled fact.

### Critical Don't-Miss Rule — *revised 2026-08-06: build is authorized, but scoped*

- **G8b build is authorized (S-1 + S-5 decided, 2026-08-06) — but only inside a tracked backlog
  item.** Chaincode, the in-band write-path integration, network bring-up, and the Caliper harness
  are now the *expected* deliverables, not a forbidden boundary crossing. **The condition that
  authorizes this is: the code belongs to a specific `implementation-backlog.md` item, and that
  item's dependencies (design doc, ADR, and — where named — PB-1/PB-3) are actually satisfied.**
  Code written with no tracked item behind it is **still** a boundary breach (risk R-05, re-scoped
  not retired) — "the gate is open" does not mean "anything goes." If asked to write code with no
  corresponding backlog item, name the gap rather than inventing scope. Full record:
  `agent-suite/00-architecture/quality-gates-and-approval.md` §5.5.

---

## Usage Guidelines

**For AI Agents:**

- Read this file before implementing any code or extending any design artifact in this repo.
- Follow ALL rules exactly as documented — especially the 0-PII-on-chain invariant and the no-code-until-S-1 gate.
- When in doubt, prefer the more restrictive option (design-only, more isolation, less PII surface).
- **Every written artifact stays in the generic register** (see AUTHORITY NOTICE above) — never
  name the real platform, company, or literal real table/column names, even when grounding
  internally against the real repo for technical accuracy.
- Update this file if new ADRs are accepted or new blockers are resolved — and log the change in
  the PRD workspace's `.memlog.md`, not just here, so the audit trail survives.

**For Humans:**

- Keep this file lean and focused on agent needs.
- Update when an ADR is superseded, a grounding gap (G-##) closes, or the tech stack changes.
- Review at each gate transition (especially the S-1 decision and any subsequent G9 re-approval).

Last Updated: 2026-08-06

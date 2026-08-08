# Blockchain Integration — how the HRIS write paths attach to Fabric

> **Scope.** Defines the integration seam between the existing HRIS stack and Hyperledger Fabric 2.5:
> the five profile-section write-path anchor triggers, the **in-band recording component** (no
> standalone service, no Kafka), the Gateway call path, and the seam's remaining open question
> (outage handling). Companion: [`BLOCKCHAIN-DATA-MODEL.md`](BLOCKCHAIN-DATA-MODEL.md) defines the
> record this component writes.
>
> **Rewritten 2026-08-06** (`rencana-rekonsiliasi.md` Gelombang 5 #33) to match **ADR-0014** (in-band
> recording) and **ADR-0020** (the chaincode contract + hashing boundary). The pre-2026-08-06 version of
> this document described a Kafka `employee_info` consumer, an `EVENT_UPDATE_PERSONAL` approval hook, and
> a new thin Go anchor-service — **all three are retired**, not merely relocated. The authoritative
> version of everything below is
> [`../00-architecture/solution/integration-design.md`](../00-architecture/solution/integration-design.md)
> (`fabric-engineer`); this context doc cross-references it rather than restating it, per
> `AUTHORING-CONTRACT.md §7`.
>
> **Cross-reference, do not duplicate.** The Gateway connection/endorse/submit mechanics live in the
> skill suite: `> See fabric-chaincode-dev/references/gateway-client.md`. Identity/MSP/TLS setup:
> `> See fabric-identity-security` (D6/D13).
>
> **Register note.** No product name, codebase, file, class, or table name appears in this document —
> only generic descriptions of the five write paths, per `project-context.md` AUTHORITY NOTICE. Real-code
> grounding for these five paths was consulted internally and is preserved without being reproduced
> verbatim here — see grounding gap **G-29**.

## 1. Where Fabric attaches — five write paths, in-band, no event bus

Per **ADR-0014**, each of the five profile sections is anchored from its **own** existing write path,
inside the same request that commits the section to the operational database — not from a Kafka topic,
and not from a single "personal-data change" hook the way the pre-2026-08-06 design assumed.

| Profile section | Generic description of the triggering write path |
|---|---|
| `PERSONAL` | A personal-data change-approval workflow. |
| `EMPLOYMENT` | A transfer/mutation approval action. |
| `EDUCATION` | An education-history create/update write action. |
| `ADDITIONAL` | A family-data (marital status / dependents) change-approval workflow. |
| `PAYROLL` | A payroll/bank-account update action. |

Each path, at the point it commits its section to the operational database, computes the section's
digest and pseudonymous identifiers off-chain and calls `RecordProfileSection` **in the same request** —
no message queue or standalone consumer sits between the write and the anchor. Full sequence:
`integration-design.md` §4.

**Why the event-based surface was retired, not merely reordered.** The pre-2026-08-06 design's metadata-
only event topic physically cannot carry the `EDUCATION`/`ADDITIONAL`/`PAYROLL` payloads this design
needs to hash — this is *why* the event-based/Kafka path was retired outright, not a deprioritization
(`prd.md §11.3 ADR-0014`).

## 2. The in-band recording component (no standalone service, G-10 reopened)

There is **no new deployable service** in this design. The digest builder, the `employeeKey_i`/salt
lookup, and the Fabric Gateway client are a **shared component** embedded directly into each of the five
write paths above — the same distinction the ratified design draws throughout.

- **Which existing process hosts this component is an open question (grounding gap G-10, reopened by
  ADR-0014)** — do not assume it is a new service, a shared internal library, or a co-located helper
  without checking whether that has since been decided.
- The component holds a Fabric X.509 identity in the platform org's MSP for the Submit call; a verifier
  querying `Evaluate` does so under **their own** org's identity, against **their own** peer, not this
  component's (FR-37).
- Identity provisioning, TLS, and cert lifecycle: `> See fabric-identity-security` (D6/D13/D15) — not
  restated here.

### 2.1 Internal shape

- **One retained gRPC/Gateway connection** per host process, opened at startup and reused for every
  call — connection setup is expensive and must not be per-call
  `[docs: write_first_app.rst#first-establish-a-grpc-connection-to-the-gateway]`.
- **No consume loop, no offset, no reconciliation job.** Because the digest is computed **in** the write
  path rather than relayed to a separate consumer, this design has no idempotency probe against a
  companion store and no periodic job comparing two stores for drift — the pre-2026-08-06 design needed
  all three only to close a gap this design does not open. What replaces "idempotency probe" is the
  chaincode's own no-op-on-identical-`dataHash` rule (`RecordProfileSection` function 1): a retried Submit
  with identical inputs after a partial prior failure lands on that same no-op path.
- **MVCC-conflict retry, not out-of-order prevention via partitioning.** A `MVCC_READ_CONFLICT` (or a
  stale-`prevHash` rejection) means another write to the *same* `(employeeID, profileSection)` committed
  concurrently — the correct response is to re-read the current head and resubmit with the corrected
  `prevHash`, not to treat it as terminal. There is no Kafka partition key doing this job in this design;
  the chain-reference check inside chaincode does it instead.
- **Read results as bytes.** Every Gateway response is a byte payload the component decodes
  `[docs: write_first_app.rst#run-the-sample-application]`; errors external to the gateway carry the
  endpoint + MSP ID in the `Details` field, an empty `Details` means the fault is the gateway peer itself
  `[docs: gateway.md#error-handling]`.

## 3. Call path

```mermaid
sequenceDiagram
  autonumber
  actor HR as HR admin / employee
  participant WP as HRIS write path<br/>(one of five sections)
  participant DB as Operational database (SoR)
  participant OFF as Off-chain salt +<br/>employeeKey_i store
  participant GW as Fabric Gateway client<br/>(host: open, ADR-0014/G-10)
  participant CL as Enterprise-client org peer
  participant ORD as Orderer / ledger (tenant channel)

  HR->>WP: submit/approve a profile-section change
  WP->>DB: commit the section (system-of-record write)
  WP->>WP: JCS-canonicalize the CURRENT full section value
  WP->>OFF: fetch employeeKey_i; draw a NEW salt (CSPRNG, >=128 bits)
  WP->>WP: dataHash = SHA-256(salt || JCS(section)); employeeID/updatedBy = HMAC-SHA256(employeeKey_i, ...)
  WP->>OFF: persist {salt, dataHash} off-chain (never on-chain)
  WP->>GW: SubmitTransaction("RecordProfileSection", ...) — 0 bytes plaintext, 0 bytes salt
  GW->>CL: gather co-signature — AND(platform org, enterprise-client org)
  CL-->>GW: endorsement
  GW->>ORD: submit envelope -> order -> commit (tenant channel)
  ORD-->>WP: CommitStatus (VALID | MVCC_READ_CONFLICT | ...)
```

`SubmitTransaction` combines **Endorse + Submit + CommitStatus** into one blocking call
`[docs: gateway.md#fabric-gateway]`, `[docs: gateway.md#client-application-apis]`. Reuse **one** gRPC
connection — establishing them is expensive
`[docs: write_first_app.rst#first-establish-a-grpc-connection-to-the-gateway]`. Full connect snippet (Go)
and endorser-selection behaviour: `> See fabric-chaincode-dev/references/gateway-client.md`. Full stage
detail (canonicalize → salt → digest → `prevHash` read → Submit → resolve outcome): `integration-design.md`
§4.

- **Never send plaintext or PII in the proposal — and never a salt, on Submit or Evaluate.** Only a
  digest, two pseudonymous identifiers, an enum, CIDs, and version tags are ever transaction arguments
  (ADR-0020). There is **no `transient` field in this design** — nothing here ever needs one, because no
  function receives a value in the first place.
- The digest and salt are computed once and persisted off-chain **before** the Submit call, so a failed
  Submit is retryable from identical inputs.

## 4. Query / verify path

Verification and audit reads use **Evaluate** (single-peer, no ordering)
`[docs: gateway.md#fabric-gateway]` against `GetProfileSectionRecord`/`GetProfileHistory`/
`GetEmployeeProfileSummary`. The comparison itself happens **entirely client-side**:

1. The verifier obtains the salt for the version being checked over a **separate, audited** channel —
   never via any chaincode call (FR-36).
2. The verifier obtains the current section value from a source they already lawfully hold.
3. The verifier canonicalizes and recomputes `SHA-256(salt ‖ JCS(sectionValue))` **locally**.
4. The verifier calls `GetProfileSectionRecord` against **their own** peer (an enterprise-client or
   auditor org verifier never needs to trust a peer they do not operate for a verification they intend to
   rely on, FR-37).
5. The verifier compares the recomputed digest to the returned `dataHash` **locally**.

There is no server-side "verify" endpoint in this design that performs steps 3/5 on a caller's behalf —
that pattern belonged to the retired anchor-service. Full protocol, including the
`404`-vs-mismatch distinction and why original values are not "reconstructed" from the ledger:
`integration-design.md` §6, `api-contracts.md` "Client-side verification protocol."

## 5. Seam failure modes

Because the operational database is the system-of-record and Fabric is an overlay, and recording is now
**in-band** rather than async, a Fabric outage sits **inside** the same request that is committing the
profile-section write — this is a direct consequence of retiring the async Kafka path, not a
pre-existing property carried over from it.

| Failure | Effect | Handling |
|---|---|---|
| Endorsement/gateway/orderer unreachable, mid-request | The write path's own request is affected directly (no queue to absorb the delay) | **Open, undecided (`SEC` T6b, `GAPS` G-10 reopened).** Whether the write path blocks until anchored, retries bounded then degrades, or fires-and-forgets with a flagged "not yet anchored" state is **not fixed by this document** — `architect`/`backend-engineer` own this decision. |
| Duplicate delivery / retried Submit | Same section write submitted twice (e.g. after a client-side retry) | Idempotent: identical `dataHash` to the current head is a no-op inside chaincode (FR-9) — no companion store needed. |
| `MVCC_READ_CONFLICT` / stale `prevHash` | Concurrent write to the same `(employeeID, profileSection)` | Re-read the current head, resubmit with the corrected `prevHash` — expected, not exceptional (`data-model.md` §5.2). |
| Salt or `employeeKey_i` lost / store corrupted | Digest un-openable | The salt/`employeeKey_i` store is critical infrastructure (ADR-0019); back it up **except** where deletion is the intended erasure lever (`BLOCKCHAIN-DATA-MODEL.md` §6). |
| Spoofed gateway/tenant headers | Wrong actor/tenant attribution | **Open (PB-1/G-18).** Verify end-to-end that the gateway strips + re-injects trust headers before security sign-off. |
| Non-pilot tenant, encryption rollout incomplete | Encrypted columns possibly empty, plaintext authoritative | Read through the application's own encryption layer, not raw columns, unaffected by this reconciliation wave (G-21, pending S-4). |

**No reconciliation job in this design.** The pre-2026-08-06 design ran a periodic job comparing the
operational change log to on-chain anchors, because the two could drift across the Kafka hop. In-band
recording removes that hop — the write path is the **only** place the digest is ever computed, so there
is no second store for it to drift from (`integration-design.md` §5.1). The residual risk that job used
to catch — an approved change silently missing its anchor — is instead a **within-request
detectability** requirement (surface the failure as an error/metric, per the open item above), not a
batch reconciliation.

## 6. What is deliberately out of scope here

- Chaincode implementation of `RecordProfileSection` / the three query functions — record shape and
  contract mechanics: `data-model.md`, `api-contracts.md`, `> See fabric-chaincode-dev/references/contract-api.md`.
- MSP enrolment, TLS, HSM, cert rotation for the recording component's identity:
  `> See fabric-identity-security`.
- Endorsement-policy design (who must sign an anchor): `> See fabric-chaincode-dev/references/endorsement.md`;
  ratified as co-signature of the platform org and the tenant's client org (ADR-0012).
- Throughput/latency sizing and state-DB choice: `> See fabric-performance`; state DB is **LevelDB**
  (ADR-0007, unchanged).
- The salt/`employeeKey_i` store's custody and rotation model: `security-architect`'s (ADR-0019/ADR-0021).

## 7. Traceability

Seam realizes the anchor chain: `HRIS profile-section write path → operational-database commit → in-band
off-chain digest → co-signed RecordProfileSection → immutable per-section chain`. Open questions carried:
**G-10** (reopened — recording-component host + outage-handling policy), with **G-18** (header trust) and
**G-21** (encryption-rollout phase) touching the trigger and trust surface. `G-19`/`G-20`/`G-23` — all
scoped to the retired event-bus read-back pattern — are recorded pending sponsor confirmation (**S-4**)
in [`../11-execution/grounding-gaps.md`](../11-execution/grounding-gaps.md), not asserted closed here.

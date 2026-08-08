# ADR-0006: A new thin Go anchor-service hosts the Fabric Gateway client

- **Status:** **Superseded by ADR-0014**
- **Date:** 2026-07-13
- **Deciders:** fabric-engineer, solution-architect
- **Rests on assumption(s):** G-10 (which service hosts the Fabric gateway client). SDK-support fact grounded in `[docs: gateway.md#writing-client-applications]`.
- **Supersedes:** none
- **Superseded-by:** ADR-0014
- **Ratifying authority / date:** `rencana-rekonsiliasi.md` Gelombang 3 #17, consistent with `prd-fabric-hris-2026-08-02/prd.md` §4 (anchoring unit changed to profile section; the metadata-only Kafka topic this ADR relied on cannot physically carry the EDUCATION/ADDITIONAL/PAYROLL section payloads the new digest scheme requires).

## Context
Something must consume the PII-change signal, build the salted commitment (ADR-0001), and submit it to Fabric. Two natural seams already exist in code and need no schema change: the Kafka `employee_info` topic (talenta-core produces `[code: talenta-core/services/kafka/EmployeeInfoProducerService.php]`; EMS consumes/emits `[code: ems/internal/employee/domain/sync_kafka.go]`) and the `EVENT_UPDATE_PERSONAL` approval hook `[code: talenta-core/services/ChangeDataService.php]`. The `employee_info` payload is identity/org metadata with no raw sensitive PII (**G-19**, pending EBupot review).

Where should the Fabric Gateway client live? Three candidate hosts: embed in talenta-core (PHP), embed in EMS (Go), or a new dedicated service. Two hard facts decide it: (1) the supported Fabric Gateway client SDKs from v2.4 are **Go, Node, and Java** — there is **no** first-class PHP SDK `[docs: gateway.md#writing-client-applications]`; (2) EMS already speaks Kafka, so a sibling Go consumer is idiomatic and needs no new transport `[code: ems/internal/employee/domain/sync_kafka.go]`. Isolating the ledger dependency from the system-of-record's request path is also exactly what the overlay posture (ADR-0002) wants.

Relevant capability: **D6** (MSP/X.509 identity for the client). See context `context/BLOCKCHAIN-INTEGRATION.md` §2 (the thin service and its internal shape), `context/PHP-INTEGRATION.md` §5 (why not PHP), `context/GO-ARCHITECTURE.md` §8 (EMS layering the service reuses).

## Decision
We will build a **new, thin Go anchor-service** that consumes the `employee_info` Kafka topic and hosts the Fabric Gateway client. It holds an X.509 identity in the HR-org MSP (**D6**), opens **one** long-lived gRPC/Gateway connection per replica at startup and reuses it for every event, and runs a single consume→anchor loop (not a request/response API). Neither talenta-core nor EMS hosts the Fabric client; talenta-core keeps emitting the existing events unchanged, and EMS is untouched. `submitTransaction` combines Endorse + Submit + CommitStatus in one call `[docs: gateway.md#fabric-gateway]`.

**Numeric constraints (set by this ADR):**
- First-class Fabric Gateway SDKs available: **3** (Go, Node, Java); for PHP: **0** — this eliminates option C `[docs: gateway.md#writing-client-applications]`.
- Retained gRPC/Gateway connections: **1 per replica**, opened at startup, reused for every event (connection setup is expensive and must not be per-message).
- Kafka delivery semantics: **at-least-once** — commit the offset only **after** the anchor reaches a terminal state; in-flight events processed **1 at a time per Kafka partition** (partitioned by `employeeRef`) to preserve the per-employee `prevCommitment` hash chain.
- Shared-MySQL schema changes: **0** (G-16) — the companion `anchor_map` lives in the service's own store.

## Alternatives considered
| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. New thin Go service (chosen)** | First-class Go Gateway SDK; reuses EMS Kafka idioms and layering; ledger dependency + failure modes isolated from the system-of-record (reinforces ADR-0002); independent deploy/scale; no schema change | One more deployable unit to operate; its own identity/TLS/cert lifecycle to provision (owned by fabric-identity-security) | **Chosen** |
| B. Embed the client in EMS (Go) | Go SDK works; no new service | Rejected — **couples the ledger lifecycle to the EMS deploy cadence**; ledger failure modes and an extra gRPC dependency bleed into the HRIS extraction's request path; violates the overlay-isolation intent of ADR-0002 | Rejected |
| C. Embed the client in talenta-core (PHP) | Closest to the `EVENT_UPDATE_PERSONAL` source | Rejected — **no first-class Fabric SDK for PHP** `[docs: gateway.md#writing-client-applications]`; would require a bespoke/FFI bridge with no corpus support; puts a blockchain dependency inside the monolith system-of-record | Rejected |

## Consequences
- **Positive:** the language matches a supported SDK; the anchor path is fully decoupled from both the monolith and EMS; scales independently; the at-least-once + idempotent-composite-key design makes the seam crash-safe and replay-safe.
- **Negative / trade-off:** a new deployable unit (deploy on the same Alibaba Cloud K8s as EMS per the existing Helm pattern — **G-13**) with its own MSP identity, TLS, and cert-rotation to provision; introduces the eventual-consistency reconciliation obligation from ADR-0002; if G-10 flips, the fallback host is the talenta-core `ChangeDataService` approval hook (PHP-INTEGRATION §5).
- **Follow-ups:** append prototype-phase risk rows for **anchor-service identity/TLS/cert-expiry** (D13/D15) and **seam failure modes** to `10-risk/risk-register.md` at G3+; identity provisioning, TLS, HSM, cert rotation are owned by `fabric-identity-security` (not this ADR); Gateway connect/endorse/submit mechanics live in `fabric-chaincode-dev/references/gateway-client.md`.

## Related
- Relates to: **ADR-0002** (the overlay this service hosts — isolation is the shared rationale), **ADR-0001** (the service builds and submits the salted commitment, never plaintext), **ADR-0010** (the service also drives the off-chain salt deletion for erasure).
- Knowledge-graph: Layer C (talenta-core / EMS / Kafka / gateway client), Layer D capability **D6**; traceability spine chain #1 (anchor). Grounding gap(s): **G-10** (and G-13, G-16, G-19). Context doc(s): `context/BLOCKCHAIN-INTEGRATION.md`, `context/PHP-INTEGRATION.md`, `context/GO-ARCHITECTURE.md`.

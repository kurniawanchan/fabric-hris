# ADR-0014: Record profile-section integrity in-band from the HRIS write path — no Kafka, no anchor-service

- **Status:** **Accepted**
- **Date:** 2026-08-06
- **Deciders:** fabric-engineer (formalization)
- **Rests on assumption(s):** **G-10 (reopened)** — which part of the HRIS application process
  hosts the Fabric Gateway client is deliberately left open by this ADR (see Decision). Grounded
  in `[prd: §4]`, `[prd: §11.3]` (ADR-0014 line), `rencana-rekonsiliasi.md` Gelombang 3 #17.
- **Supersedes:** ADR-0006
- **Superseded-by:** none
- **Ratifying authority / date:** `rencana-rekonsiliasi.md` Gelombang 3 #17, consistent with
  `[prd: §4]` (anchoring unit = profile section) and the STOP-LIST in `project-context.md`
  (Kafka `employee_info` consumer + standalone anchor-service retired).

## Context

ADR-0006 hosted the anchor path in a **new, thin Go anchor-service** that consumed a metadata-only
Kafka topic and read back the changed values through an authenticated app-layer call before
computing a commitment. That design was sized for a **per-event, changed-fields-only** commitment
(ADR-0001, superseded by ADR-0011) fed by a topic that, by its own producer contract, carries
**identity/org metadata only — no raw sensitive PII** (grounding gap **G-19**, carried unchanged
into this ADR as the reason the topic is unusable, not as an open question).

Two rulings make that architecture impossible to keep, not merely sub-optimal:

1. **The anchoring unit is now the whole profile section** (`[prd: §4]`), and computing
   `DataHash = SHA-256(salt ‖ JCS(section))` (ADR-0011) requires the **complete current section
   payload** at digest time — every field of `PERSONAL`, `EMPLOYMENT`, `EDUCATION`, `ADDITIONAL`,
   or `PAYROLL`, not a named subset of changed field values.
2. **The one topic this design could have reused physically cannot carry that payload.** Its
   producer contract is metadata-only by construction (`G-19`) for four of the five sections —
   `EDUCATION`, `ADDITIONAL`, and `PAYROLL` in particular have no field of that topic's schema that
   was ever meant to carry a formal/informal education record, a family/dependents record, or a
   salary/bank-account record. This is not a schema gap that a field addition closes cheaply: it
   would mean widening a shared producer's contract, consumed by other subscribers, to carry
   exactly the class of section content (financial, familial) that gap **G-19**'s own review
   exists to keep off that bus in the first place. Enlarging the topic to fit the digest input
   would recreate, on the wire, the confidentiality question ADR-0011/ADR-0020 exist to close.

Because the value needed to compute the digest already exists, in full, at the moment each
section's write path commits it to the operational database — and nowhere else, cheaply — the
anchor write belongs at that moment, not behind an asynchronous relay that was never able to carry
the value across in the first place.

## Decision

We will record profile-section integrity **in-band**: at the moment a section's write path in the
HRIS application commits that section to the operational database, the **same write path**
computes the off-chain digest (ADR-0011) and submits `RecordProfileSection` (ADR-0020) — as part
of that write's own request lifecycle, not via a message queue and not via a separately-deployed
consumer service.

**What this ADR decides:**
- **Kafka topic count dedicated to anchoring: 0.** No topic is introduced or widened to carry
  section content for this purpose.
- **Anchor-service deployable units: 0.** No new, separately-deployed Go service is introduced to
  host the Gateway client and a consume→anchor loop.
- **Hook points: 5** — one per profile-section write path, each responsible for computing its own
  digest and calling `RecordProfileSection` at commit time.

**What this ADR deliberately leaves open:**
- **Which part of the HRIS application process holds the Fabric Gateway client** is **not**
  decided here. Three shapes are all consistent with "in-band": (a) the client lives directly in
  each write path's own process; (b) the five write paths share one internal library that itself
  holds the retained Gateway connection; (c) a lightweight in-request call to a co-located sidecar
  that still blocks the same request (as opposed to an out-of-band queue consumer, which this ADR
  rules out). This reopens grounding gap **G-10** rather than closing it, and is left to a
  follow-on decision by `architect`/`backend-engineer` once the concrete write-path shape is
  scoped (see Consequences).

## Alternatives considered

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. In-band, synchronous, from each of the 5 write paths; Gateway-client host TBD (chosen)** | The digest input (full section payload) is available for free at the one place it already exists — the write path's own memory, at commit time; eliminates an entire failure-mode catalogue (offset management, at-least-once dedupe, reconciliation job) that existed only to transport a value that never needed transporting in the first place; removes one deployable unit and its identity/TLS/cert lifecycle | Five write paths each need a hook into digest-compute-and-submit logic instead of one centralized consumer; a Fabric outage now sits inside the HRIS write-path's own blocking window unless a local fallback is designed (flagged, not solved, here) | **Chosen** |
| B. Keep Kafka `employee_info` + anchor-service (ADR-0006, superseded) | Reuses an existing seam; keeps the ledger dependency out of the HRIS request path entirely | **Rejected** — the topic is metadata-only by producer contract (G-19) and cannot carry `EDUCATION`/`ADDITIONAL`/`PAYROLL` section payloads without widening a shared contract other consumers rely on; the anchoring unit itself changed underneath this design (§4) | Superseded by this ADR |
| C. Add one new Kafka topic/event per section, still relayed through an anchor-service | Keeps the write path decoupled from the ledger, mirroring B's isolation intent | **Rejected** — three of the five sections have no existing event producer wired at all today, so this option requires adding a new producer to those write paths regardless; once a write path must be touched to emit the section payload, the marginal cost of computing and submitting the digest **at that same point** is lower than standing up a new event-sourcing layer purely to relay it asynchronously one hop later; also reintroduces at-least-once/offset-management complexity the in-band design has no need for | Rejected |
| D. Change-data-capture (CDC) tailing the five section tables into an anchor-service | Row-level change does contain the needed values, unlike option B's topic; keeps write paths unmodified | **Rejected for the prototype** — reopens the "no shared-schema/CDC-infra change" boundary (grounding gap **G-16**) and adds a new operational dependency (a CDC connector with its own lag and backpressure characteristics) purely to re-derive a value the write path already held in memory a moment earlier; deferred as a future scale option if hook-point coupling (Consequences, below) proves costlier than expected, not chosen now | Rejected |

## Consequences

- **Positive:** eliminates the Kafka consumer, the `anchor_map` idempotency/offset state machine,
  and the periodic reconciliation job that ADR-0002/0006's overlay design needed solely to close
  an eventual-consistency gap that in-band recording does not open in the first place; removes one
  deployable unit (no anchor-service) and its own MSP identity/TLS/cert-rotation lifecycle; the
  write-path latency budget added by anchoring (NFR-9, target **< 500 ms** added to the write path)
  becomes directly attributable to one endorse-and-commit round trip per section write, which is
  measurable once the Gateway-client host is fixed.
- **Negative / trade-off:** five write paths are now coupled to the ledger instead of one consumer
  — whichever host is chosen for the Gateway client (open, see Decision), every one of the five
  write paths must reach it; if any one write path's request now blocks on Fabric endorsement and
  ordering with no fallback, a ledger outage becomes an HRIS write-path outage for that section,
  which the prior overlay design (ADR-0002's "approval never blocks on the ledger" posture)
  explicitly avoided — this ADR does **not** yet specify a circuit-breaker/local-queue fallback for
  that case; it is flagged as an open design question for whoever fixes the Gateway-client host.
- **Follow-ups:** **G-10 is reopened**, not closed — a follow-on ADR (owner: `architect`, in
  consultation with `backend-engineer`) must fix the Gateway-client host and, if a Fabric outage
  must not block the HRIS write path, its fallback/retry behavior; NFR-1 (< 3 s @ 500 TPS write)
  and NFR-9 (< 500 ms added latency) become measurable only once that host is fixed — carried to
  the Caliper benchmarking item (`rencana-rekonsiliasi.md` Gelombang 3 #19 / ADR-0018, not written
  here); a risk-register row for "write-path availability now coupled to ledger availability"
  belongs to security-architect/architect's risk register.

## Related

- Supersedes: **ADR-0006** (this ADR's Status flip is the only edit made to ADR-0006's body per
  the immutability rule in `_TEMPLATE.adr.md`).
- Relates to: **ADR-0011** (supplies the digest this ADR's write paths compute in-band);
  **ADR-0020** (the `RecordProfileSection` contract this ADR's write paths call); the pending
  ADR that will fix the Gateway-client host (G-10 reopened, owner `architect`).
- Knowledge-graph: Layer C (HRIS write paths / integration seam), traceability spine chain #1
  (anchor). Grounding gap(s): **G-10** (reopened), **G-19** (cited as the reason, not reopened),
  **G-16** (cited against Alternative D). Context doc(s): `context/BLOCKCHAIN-INTEGRATION.md`.
  Ratifying source: `rencana-rekonsiliasi.md` Gelombang 3 #17, `[prd: §4]`.

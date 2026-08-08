# ADR-0022: A new, standalone Go bridge service — sibling to EMS, not embedded in talenta-core or EMS — hosts the Fabric Gateway client and is called synchronously from talenta-core's real write-path commit sites

- **Status:** **Accepted**
- **Date:** 2026-08-08
- **Deciders:** architect, backend-engineer (ownership assigned by ADR-0014 itself)
- **Rests on assumption(s):** **G-10 (closed by this ADR)**. Re-confirms ADR-0006's no-PHP-SDK
  fact and extends it (no PHP gRPC client either) via a live-repo check `[docs: gateway.md#writing-client-applications]`,
  `context/REAL-INTEGRATION-TRIGGER-FLOW.md` §3.
- **Supersedes:** none
- **Superseded-by:** none
- **Ratifying authority / date:** Direct user instruction, 2026-08-08, informed by
  `context/REAL-INTEGRATION-TRIGGER-FLOW.md`'s 2026-08-07 live-repo verification (every
  file:line citation below is inherited from that document, not independently re-checked here).

## Context

`ADR-0014` decided profile-section integrity is recorded **in-band** — synchronously, from
whichever HRIS write path commits a section — and explicitly **reopened G-10**: *which part of
the application process hosts the Fabric Gateway client* was left to a follow-on ADR, owned by
`architect` in consultation with `backend-engineer`. Three shapes were named as all consistent
with "in-band": (a) embedded directly in each write path's own process, (b) a shared internal
library the write paths link against, (c) a co-located call that still blocks the same request
(ruling out only an out-of-band queue consumer).

`ADR-0006` (superseded by `ADR-0014`, but its underlying SDK fact still holds) already
established there is **no first-class Fabric Gateway SDK for PHP** — only Go, Node, and Java
`[docs: gateway.md#writing-client-applications]`. That alone eliminates shape (a) for a PHP write
path. `context/REAL-INTEGRATION-TRIGGER-FLOW.md` §3 re-checked this against the live repo rather
than re-asserting it, and additionally **ruled out a PHP gRPC client precedent**: `composer.json`
has `google/protobuf` but no gRPC client library, and the one `.proto` file in the codebase
(`proto/notification-service/event.proto`) is used only to serialize a message for **Kafka**
publishing (`helpers/NotificationServiceHelper.php:45-47`), not synchronous gRPC. There is
nothing in this codebase that could host a Fabric client from inside a PHP process, by any
mechanism, today.

The same document names a **real, already-production precedent** for exactly this shape of
problem: `services/ems/BaseEmsService.php`. It calls a separate Go service synchronously —
`GuzzleHttp\Client` (`send()`, lines 23-58), a shared `X-Api-Key` header
(`companyLevelHeaders()`, lines 98-105), config via `env("EMS_SERVICE_URL")` /
`env("EMS_API_KEY")` (`config/params.php:319-321`, live in `.env:387`). Whatever hosts the
Gateway client, talenta-core already knows how to call it this way — this ADR's job is to decide
*whether* to reuse that convention, not to invent a new one.

Two further facts from the same document bound the decision:

1. **Five logical hooks map to four real commit points, not five.** `UpdatePersonalData`
   (PERSONAL) and `UpdatePayrollBankAccount` (PAYROLL) are the *same* commit call —
   `InboxController::updateChangeData()`'s `$updateDataUser->save();`
   (`controllers/InboxController.php:8219`/`:8224`) — bank fields are three more `case` branches
   in the same `switch`. Whatever hosts the client is called from **4** real sites, not 5.
2. **The approver/actor identity is not a passed parameter at 3 of those 4 sites.**
   `InboxController::updateChangeData($id, $idUser)`, `UserRepository::transfer(...)`, and
   `BaseComponent::GetDataToApplyChangeFamilyData(...)` all lack an actor parameter in scope at
   the exact save point, even though `Yii::$app->user->identity` is reliably populated
   application-wide (`config/web.php:163-166`). Only `FormalEducationServices` carries it cleanly.

## Decision

We will build a **new, standalone Go HTTP bridge service** — a sibling to EMS, embedded in
neither talenta-core nor EMS — that wraps `write-path-integration/writepaths.Hooks` behind one
HTTP endpoint per profile section. talenta-core calls it **synchronously, blocking the same
request**, from each of its real write-path commit sites, using a Guzzle client modeled directly
on `BaseEmsService`'s own pattern: plain HTTP/JSON, a shared `X-Api-Key` header, configuration via
environment variables. This closes **G-10** within `ADR-0014`'s in-band constraint (no queue, no
async relay) while keeping the Gateway client in a language with first-class SDK support.

**What this ADR decides:**
- **Gateway-client host: a new, standalone Go service**, not talenta-core, not EMS.
- **Calling convention: reuse, not invent** — Guzzle + `X-Api-Key`, mirroring `BaseEmsService`
  exactly, so talenta-core's own engineers already know this idiom.
- **Call sites: the 4 real commit points**, not the 5 logical hooks — PERSONAL and PAYROLL share
  one call from `InboxController::updateChangeData()`.
- **Actor identity: read at the call site itself.** At the 3 sites without an actor parameter in
  scope, the inserted call reads `Yii::$app->user->identity->id` directly, not a passed variable.

**What this ADR deliberately leaves open** (see Consequences):
- The bridge's exact module layout / deployment path.
- Transport security beyond the API-key precedent (mTLS vs. TLS+API-key-only) — `BaseEmsService`
  does not itself confirm mTLS; don't assume the precedent satisfies this design's confidentiality
  bar without `security-architect` checking it.
- The 3 PAYROLL write surfaces this bridge, as scoped, does not cover (§4 of
  `REAL-INTEGRATION-TRIGGER-FLOW.md`) — a separate, disclosed gap, not resolved here.
- A circuit-breaker/local-queue fallback for a bridge or Fabric outage — `ADR-0014` flagged this
  risk and left it open; this ADR does not close it either.

**Numeric constraints (set by this ADR):**
- First-class Fabric Gateway SDKs usable from a PHP process: **0** — re-confirmed, not just
  re-asserted, including the gRPC-client check (`composer.json` has no gRPC library).
- Real PHP commit sites the bridge is called from: **4**, covering 5 profile sections (PERSONAL
  and PAYROLL share one).
- Commit sites with actor identity already in scope at the save point: **1 of 4**
  (`FormalEducationServices`); the other 3 read it directly at the call site.
- PAYROLL write surfaces NOT reached by this bridge as scoped: **3** (`MyInfoController::actionEditPayrollInfo()`,
  bulk import, a cron job, and a third-party integration importer) — named, not fixed, here.

## Alternatives considered

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. New standalone Go bridge, sibling to EMS (chosen)** | Reuses a real, already-production calling convention (`BaseEmsService`) — zero new integration idiom for talenta-core; keeps the ledger dependency out of talenta-core's request path and out of EMS's own deploy cadence; can hold one warm Gateway connection reused across every write, restoring the connection-reuse property `ADR-0006` originally required | One more deployable unit, with its own MSP identity/TLS/cert lifecycle to provision; talenta-core's write-path request now blocks on this bridge's own availability | **Chosen** |
| B. Embed the Gateway client directly in the talenta-core PHP process | Closest to the real commit sites; no new service | **Rejected** — no first-class Fabric SDK for PHP, and (newly re-confirmed here) no PHP gRPC client library either; would require a bespoke FFI bridge with no corpus support, and would put a blockchain dependency inside the monolith system-of-record | Rejected |
| C. Fold the bridge into EMS itself, rather than standing it up standalone | One fewer deployable unit | **Rejected** — couples the ledger dependency's deploy cadence, blast radius, and identity/TLS/cert lifecycle to an already-live production service, for a purpose (profile-section anchoring) EMS's own Kafka topic cannot even carry the data for (`G-19`, restated in `ADR-0014`'s Context); reproduces the coupling `ADR-0006` already rejected once, for a different reason that still applies | Rejected |
| D. No bridge process; each write path opens its own connection at call time | Simplest to reason about, if it were possible | **Rejected, even hypothetically** — PHP-FPM's per-request process model cannot hold a long-lived connection reused across events; `ADR-0006`'s "one connection per replica, opened once, reused for every event" constraint requires a real persistent process to hold it, regardless of which language ends up hosting the client | Rejected |

## Consequences

- **Positive:** `G-10` is closed. The integration reuses an idiom talenta-core's own engineers
  already maintain (`BaseEmsService`), rather than introducing a new one to learn and review. The
  ledger dependency stays isolated from both the monolith's request path and EMS's deploy cadence,
  preserving the isolation rationale `ADR-0002`/`ADR-0006` already established. The bridge can
  hold one warm, reused Gateway connection, making the write-path latency budget (`NFR-9`, target
  **< 500 ms** added) attributable to one endorse-and-commit round trip once this host is real.
- **Negative / trade-off:** a new deployable unit needs its own MSP identity, TLS, and
  cert-rotation lifecycle (owner: `fabric-identity-security`, not this ADR). talenta-core's
  write-path request now blocks on the bridge's own availability — a bridge or Fabric outage
  becomes a write-path outage for that section, exactly the risk `ADR-0014` flagged and left open;
  this ADR does **not** specify a circuit-breaker or local-queue fallback either.
- **Follow-ups:**
  - The bridge's concrete module/deployment layout (e.g. a new `write-path-integration` sibling
    module, or a separate repository/deployable) is an implementation task for `backend-engineer`
    — not decided here.
  - Transport security for the bridge (mTLS vs. the `BaseEmsService` precedent's TLS+API-key-only)
    needs `security-architect`'s explicit confirmation before build — do not assume the precedent
    is sufficient by default.
  - The 3 uncovered PAYROLL write surfaces need their own hook-and-bridge-call decision, flagged
    to `architect`/`security-architect` — matches this project's standing discipline of naming a
    gap rather than silently narrowing what "PAYROLL is anchored" means.
  - Append a risk-register row for "write-path availability now coupled to bridge/ledger
    availability" to `10-risk/risk-register.md` (owner: `security-architect`/`architect`).

## Related

- Relates to: **ADR-0006** (source of the no-PHP-SDK fact and the connection-reuse constraint,
  both reused here, even though ADR-0006 itself is superseded); **ADR-0014** (reopened `G-10`;
  this ADR closes it within ADR-0014's in-band constraint); **ADR-0011**/**ADR-0020** (the digest
  and contract this bridge's calls ultimately reach via `writepaths.Hooks`).
- Knowledge-graph: Layer C (talenta-core / integration seam), capability **D6** (MSP identity for
  the client — now applicable to this bridge, owned by `fabric-identity-security`). Grounding
  gap: **G-10 (closed)**. Context doc(s): `context/REAL-INTEGRATION-TRIGGER-FLOW.md` (real-specifics
  grounding — the disclosed exception file this ADR's citations are inherited from),
  `context/PHP-INTEGRATION.md` §5 (generic-register companion), `context/BLOCKCHAIN-INTEGRATION.md`.

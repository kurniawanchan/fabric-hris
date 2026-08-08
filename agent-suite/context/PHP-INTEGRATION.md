# Core HRIS Service — Integration Seams (where Fabric attaches)

> **Scope.** How the platform's core HRIS service integrates outward, and specifically the seams where
> the Fabric anchoring design attaches: the **five profile-section write paths** ADR-0014 anchors
> in-band, the external/integrator-facing API surface, and the central SSO seam. This is the concrete
> backing for gaps **G-05 (closed)**, **G-10 (reopened)**, **G-11**, **G-19** (pending sponsor
> confirmation S-4).
>
> **Rewritten 2026-08-06** (`rencana-rekonsiliasi.md` Gelombang 5 #33). The pre-2026-08-06 version of
> this document described a single Kafka producer/topic and a single `EVENT_UPDATE_PERSONAL` approval
> hook as *the* Fabric seam — that surface is **retired outright** (ADR-0014), not narrowed. The
> ratified design anchors **five independent write paths**, one per profile section, each in-band.
>
> **Register note.** No product name, codebase, file path, or class name appears in this document — per
> `project-context.md` AUTHORITY NOTICE, every written artifact stays in the generic register even where
> the underlying grounding is a real, internally-consulted repository. That internal grounding is
> preserved without being reproduced verbatim here — see grounding gap **G-29**. **Recommendation:** =
> engineering judgement. **[ASSUMPTION]** = a requirement-shaped specific tied to a gap ID, not yet
> ratified.

## 1. The five profile-section write paths (ADR-0014's anchor points)

Employee-initiated and HR-initiated profile edits do not, in general, write straight through to the
operational database — most route through an approval workflow that already produces an approver
identity, a diff or a new value, a timestamp, and an audit-log entry, before the write actually commits.
This existing approval/audit shape is exactly what each of the five anchor points below builds on; the
generic description of each path (not a literal file/class name) is the authoritative version in
[`../00-architecture/solution/data-model.md`](../00-architecture/solution/data-model.md) §4 and
[`../00-architecture/solution/integration-design.md`](../00-architecture/solution/integration-design.md)
§1/§3 — cross-referenced here, not restated.

| `profileSection` | Generic description of the triggering write path | What that path already has in hand before an anchor call |
|---|---|---|
| `PERSONAL` | A personal-data change-approval workflow. | Approver identity, before/after diff, timestamp, audit-log entry. |
| `EMPLOYMENT` | A transfer/mutation approval action. | Approver identity, the new employment state, timestamp. |
| `EDUCATION` | An education-history create/update write action. | Actor identity, the new record, timestamp. |
| `ADDITIONAL` | A family-data (marital status / dependents) change-approval workflow. | Approver identity, before/after diff, timestamp. |
| `PAYROLL` | A payroll/bank-account update action. | Approver identity, the new value, timestamp. |

Each of these five paths, at the point it commits its section to the operational database, is where
**ADR-0014** hooks the in-band digest computation and the `RecordProfileSection` Submit call — in the
same request, not via any message bus. **Internal grounding note:** which concrete service/controller
implements each of the five paths above was verified against the real codebase during this
reconciliation and is not reproduced literally on this page (gap **G-29**); the generic description in
the table is sufficient for any agent designing against this seam.

**Why this replaces, rather than extends, the pre-2026-08-06 single-hook design.** The retired design's
metadata-only event topic could carry identity/org metadata but **cannot** carry the
`EDUCATION`/`ADDITIONAL`/`PAYROLL` section content this design needs to hash — this is why the anchor
point changed shape from "one Kafka event" to "five in-band write-path hooks," not merely why it moved
(`[prd: §11.3 ADR-0014]`).

## 2. External/integrator-facing API surface (audit source, not an anchor point)

The platform exposes an external/integrator-facing API surface, gated by a per-endpoint shared-secret
header scheme, that authenticates and authorizes traffic separately from the internal write paths above.
This is the highest-exposure surface for OWASP API Top-10 purposes (BOLA/BFLA), but it is an **audit
source to anchor**, not a place to embed the in-band recording component — any mutation that surface
performs still lands on one of the five write paths in §1, which is where the anchor call actually lives.

## 3. Central identity / SSO

Human identity is federated to a central SSO provider; the employee record links to that identity via an
internal identifier that also rides the platform's existing metadata event stream. **This SSO identity is
a separate concern from the Fabric MSP/X.509 identity used by the in-band recording component or by any
verifier org** (gap **G-11**) — do not conflate a human's SSO identity with a Fabric client identity, and
do not derive one from the other (key-domain separation, ADR-0019/ADR-0021, gap **G-15**).

## 4. Attachment summary: the Fabric anchor seam

| Seam | Signal it already carries | Fabric role | Gap |
|---|---|---|---|
| The five profile-section write paths (§1) | Actor identity, new/changed value, timestamp, (often) an audit-log entry | In-band digest computation + `RecordProfileSection` Submit, per section, per write | G-05 (closed) |
| External/integrator-facing API surface (§2) | External mutations under per-endpoint shared-secret auth | Audit source only — mutations still land on §1's write paths | G-18 |
| Central SSO (§3) | Federated human identity | Kept separate from the Fabric MSP identity used for Submit/Evaluate | G-11 / G-15 |

> **[ASSUMPTION] (gap G-10, reopened):** which existing process hosts the in-band recording component's
> Fabric Gateway client — one of the write paths' own host process, a shared internal library called from
> all five, or a co-located synchronous helper — is **not decided** by this document or by ADR-0014.
> Do not assume a new standalone service; ADR-0014 retired that shape outright. If this question is
> answered, the answer belongs in an ADR (`architect`), not silently assumed here.

**Do not alter the shared operational-database schema from this design** (gap **G-16**, unaffected by
this reconciliation wave): new state lives on-ledger or in the off-chain salt/`employeeKey_i` store, so
the five write paths above stay emit-only/anchor-only integration boundaries, exactly as before.

## 5. Cross-refs

- **Real-specifics companion, deliberate register exception (2026-08-07, explicit user
  instruction)**: [`REAL-INTEGRATION-TRIGGER-FLOW.md`](REAL-INTEGRATION-TRIGGER-FLOW.md) names the
  real product, real file/class/line citations for §1's five write paths, re-verified against the
  live repo — including a real finding this generic table's "five independent write paths" framing
  glosses over (PERSONAL and PAYROLL share one real commit function; PAYROLL has additional
  write surfaces that path doesn't cover) and a confirmed calling-convention precedent
  (synchronous HTTP, not gRPC — no PHP gRPC client exists in the real repo). This file
  (`PHP-INTEGRATION.md`) is authoritative for the generic register and names nothing real, by
  design, even in this cross-reference; the companion is authoritative for real specifics.
- The record shape each write path anchors: [`BLOCKCHAIN-DATA-MODEL.md`](BLOCKCHAIN-DATA-MODEL.md).
- The in-band call path, the Gateway client, and the seam's open outage-handling question:
  [`BLOCKCHAIN-INTEGRATION.md`](BLOCKCHAIN-INTEGRATION.md).
- Layout / components / where code lives generically: `PHP-ARCHITECTURE.md`.
- Encryption trait + RBAC that these seams inherit: `PHP-CONVENTIONS.md`.
- Fabric gateway client, contract mechanics: `fabric-chaincode-dev/references/`.
- MSP / identity mapping, TLS: `fabric-identity-security/references/`.
- Full write-path-to-anchor traceability: `../00-architecture/solution/integration-design.md` §3/§4.

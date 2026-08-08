# ADR — Architecture Decision Records (project convention)

> **Scope.** What an ADR is and how *this* project writes them. The ADRs themselves are a **G4**
> deliverable and live in [`../05-adr/`](../05-adr/), not here. This stub is the pointer + the
> local convention; it does not restate the template.

## What an ADR is

An **Architecture Decision Record** captures one architecturally-significant decision: the context
and forces, the choice made, the alternatives considered and *why* they were rejected, and the
consequences. It is the durable answer to "why is it built this way?" — read by anyone who later
questions the decision.

## How this project writes them

- **Template:** copy [`../05-adr/_TEMPLATE.adr.md`](../05-adr/_TEMPLATE.adr.md) to
  `../05-adr/ADR-<NNNN>-<slug>.md` (zero-padded; next number = `1 + max existing`).
- **Authoring mode:** produced via the **architecture-design** skill's **ADR mode** — run its
  *Decision Q&A* first, then write the body. One decision per ADR (if you write "and we also
  decided…", split it).
- **Status is one of three terminal values — `Accepted`, `Rejected`, or `Superseded by ADR-NNNN`.**
  No `Proposed` or `Deprecated`. The status is *derived* — from the decision (the Q&A verdict) for
  Accepted/Rejected, or from the act of superseding for the third value — never asserted freely.
- **Body immutable once written; the Status field is the one exception.** If the situation changes,
  write a **new** ADR that supersedes the old one via its `Related` field and `Supersedes` header,
  then flip the OLD ADR's `Status` to `Superseded by ADR-NNNN` and fill its `Superseded-by` header —
  do not edit the old ADR's Context/Decision/Alternatives/Consequences in place. This is how a
  decision gets correctly retired without erasing the historical record of what was decided and why.
- **Every ADR names ≥1 rejected alternative** with its reason, and states numeric constraints as
  numbers, not adjectives.
- **Every ADR states the assumption(s) it rests on** — the gap ID(s) `G-01..G-29` from
  [`../11-execution/grounding-gaps.md`](../11-execution/grounding-gaps.md) — because the project runs
  in proceed-with-assumptions mode. That is how an ADR gets revisited when a real requirement lands.

## Decisions this project has recorded (G4) — historical note, verified current 2026-08-06

**Status-vocabulary check (this section's actual assignment):** the three-terminal-value status
convention above — `Accepted` / `Rejected` / `Superseded by ADR-NNNN` — **is already in force in
this document**; this stub does **not** say "binary, nothing else" anywhere. The vocabulary was
widened 3 Aug 2026 (`rencana-rekonsiliasi.md` Gelombang 0 #2) and this file, the template, and
[`../05-adr/README.md`](../05-adr/README.md) all agree. No edit was needed to the vocabulary itself.

G4 has already run once, and partially re-run: the original ten ADRs (0001–0010) were authored
against the pre-2026-08-02 assumption set, and **six of the ten** are now `Superseded by ADR-NNNN`
following the 2026-08-02/03 thesis ratification plus the 2026-08-06 real-repo grounding pass. The
list this stub used to present as "decisions this project **expects** to record" is stale in that
framing — those decisions have been recorded, and several have already been superseded. Kept here
**for historical orientation only**, not as a live backlog:

- Fabric as **audit/anchor overlay** vs source-of-truth (G-09) — recorded as **ADR-0002**,
  `Accepted`, unaffected by the reconciliation.
- **No sensitive PII plaintext on-chain**; anchor salted commitments only (G-02) — recorded as
  **ADR-0001**, now `Superseded by ADR-0011` (per-section salted digest, `[prd: §5.2]`).
- **Single channel + PDC** vs channel-per-tenant (G-03) — recorded as **ADR-0004**, now
  `Superseded by ADR-0013` (channel-per-tenant is the ratified topology; PDC is retired entirely
  from the design, not merely deferred).
- **Two-org** consortium (HR + Audit) for the prototype (G-04) — recorded as **ADR-0003**, now
  `Superseded by ADR-0012` (three organizations: platform, enterprise-client, auditor).
- Anchor client hosted in a **new thin Go service** consuming `employee_info` (G-10) — recorded as
  **ADR-0006**, now `Superseded by ADR-0014` (in-band recording from the HRIS profile-write path;
  zero Kafka topics, zero standalone anchor-service deployables).
- Fabric MSP/HSM keys **separate** from the app-level AES PII keys (G-15) — recorded as **ADR-0009**,
  now `Superseded by ADR-0019` (four key domains; the identifier-derivation domain is further
  amended in kind by **ADR-0021**, which removes the `pseudonymKey` master key entirely).

> **Do not use this list to decide what a *new* ADR should say** — it is a record of what was once
> assumed, not a current candidate set. For the current, load-bearing decision set (21 ADRs as of
> 2026-08-06), read [`../05-adr/README.md`](../05-adr/README.md) directly — that index, not this
> stub, is authoritative for ADR status.

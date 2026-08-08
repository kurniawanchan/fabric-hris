# Threat Modeling — method stub

> **Status: stub.** This doc names the *method* the project uses to threat-model the Fabric 2.5 + HRIS
> system. It does **not** contain the threat model itself. The full, system-specific threat model — a
> completed STRIDE pass over the real topology with rated findings — is a **gate G6** deliverable and
> lives in `../08-security/`. Do not author findings here.

## Method

Threat modeling for this system uses **STRIDE** (Spoofing, Tampering, Repudiation, Information
disclosure, Denial of service, Elevation of privilege), driven by two existing skills — reference
them, do not restate their content:

- **`fabric-security-review` → `references/stride-threat-model.md`** — the Fabric-specific STRIDE
  catalog: the trust-boundary data-flow diagram (org / channel / TLS / ordering), the documented
  control that already answers each threat per component (peer, orderer, operations service, HSM,
  client app), and the review question that surfaces a gap. Fill in its
  `assets/threat-model-worksheet.md` and emit findings with the FACT-vs-RECOMMENDATION finding model.
- **`security-review` skill** — the general AppSec method for the HTTP/API tier (STRIDE + OWASP,
  trust-boundary mapping, severity scoring with CWE refs). Use it for the EMS/Kong/public-API surface;
  use `fabric-security-review` for the ledger/chaincode/config surface.

## Scope inputs (already prepared)

The G6 threat model consumes, rather than re-derives, these context docs:

- **Actors, PII entities, services, capabilities** → `../11-execution/knowledge-graph.md` (Layers A–D)
  — the trust boundaries and data flows are enumerated there, including the Kafka `employee_info` →
  anchor-service → Fabric gateway seam.
- **Trust boundary to stress first** → the Kong header boundary `[ASSUMPTION G-18]`
  (`../11-execution/grounding-gaps.md`). If `X-Company-ID` / `X-Scope` are spoofable, Spoofing +
  Elevation-of-privilege dominate the model.
- **Frame-specific control mappings** → `SECURITY-BY-DESIGN.md`, `OWASP-ASVS.md`, `OWASP-API.md`
  (this directory). The threat model rates *whether* those controls hold; it does not re-list them.

## What the G6 deliverable must produce (not done here)

- A completed STRIDE table per component/data flow over the *working* topology (HR-org + Audit-org,
  single channel + PDC) `[ASSUMPTION G-03, G-04]`, using the real code seams.
- Rated findings (severity by blast radius) separating documented FACT from engineering
  RECOMMENDATION, each with evidence — per the `fabric-security-review` finding model.
- Explicit residual-risk notes for every open gap the model touches, cross-linked to the gap register.

> **Citations convention** (as elsewhere in this package): `[docs: …]` pinned Fabric corpus ·
> `[code: …]` real repo · `[ASSUMPTION]` + gap ID. See `../11-execution/knowledge-graph.md`.

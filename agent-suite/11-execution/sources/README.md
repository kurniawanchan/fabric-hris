# Requirement sources

> ⚠️ **Superseded 2026-08-06.** HLF-6/Confluence is **retired** as the closing authority for the
> grounding gaps below — it was never reachable, and the requirement layer has since been ratified
> by a real, in-hand source instead: the author's own thesis (human decision, not an external doc
> export). Gaps **G-01** and **G-05** are closed against this source per the closing procedure in
> [`../grounding-gaps.md`](../grounding-gaps.md).
>
> **Current ratifying sources (read these, not the Jira/Confluence links below):**
> - `_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-02/prd.md` (status: **final**) —
>   the PRD, ratifying PB-4/PB-5, the crypto scheme (§5.2), FRs/NFRs, the corrected UU PDP matrix.
> - `_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-02/source-thesis-extract.txt` —
>   the thesis extract the PRD ratifies from.
> - `_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-02/errata-tesis.md` +
>   `rencana-rekonsiliasi.md` — the defects found and the 40-step execution plan.
> - The real `talenta-core` repo (path supplied by the user at planning time) — grounds the concrete
>   Employee-module integration seam. **Internal grounding only** — written artifacts stay in the
>   generic register per the AUTHORITY NOTICE in `_bmad-output/planning-artifacts/project-context.md`.

## Historical note (pre-2026-08-02 — no longer actionable)

The brief ([`../../../agents-guide.md`](../../../agents-guide.md)) originally mandated grounding
against:

- **Jira Epic:** `HLF-6` — <https://chandrakurniawan.atlassian.net/browse/HLF-6>
- **Confluence space:** <https://chandrakurniawan.atlassian.net/wiki/x/jwIB>

These live on `chandrakurniawan.atlassian.net`, which was **not reachable** from this environment.
The package proceeded in **proceed-with-assumptions mode** instead (see `grounding-gaps.md`), and
that assumption set has now been ratified by the real source above — this Jira/Confluence pointer
is kept only as a historical record of the original (unmet) grounding intent.

## What to drop, and suggested filenames

Any readable format works — Markdown, plain-text, PDF, or Confluence "Export → Word/PDF/HTML". Paste
raw text into `.md` files if that's easiest. Suggested names (so ingestion can label provenance cleanly):

| File | Contents |
|------|----------|
| `HLF-6-epic.md` | The Jira epic HLF-6 (description + child issues / links) |
| `prd.md` | Product Requirements Document |
| `user-stories.md` | User stories / acceptance criteria |
| `api-contracts.md` (or `openapi.yaml`) | API contracts between HRIS ↔ Fabric ↔ services |
| `architecture.md` | Existing architecture / target architecture notes |
| `research.md` | Research documents (DSRM, ZKP, threat landscape, related work) |
| `technical-specs.md` | Technical specifications |

Sub-folders and extra files are fine — I'll read everything in this directory tree (this `README.md`
is ignored).

## What happens after you drop them

At **Gate G0** I will:
1. Read every file here and extract requirements, entities (employee/PII fields), API shapes, and
   non-functional targets.
2. Build [`../knowledge-graph.md`](../knowledge-graph.md) linking requirements → design decisions.
3. Log anything unclear or missing to [`../grounding-gaps.md`](../grounding-gaps.md) as `[ASSUMPTION]`
   (I never silently invent requirements).

## If you'd rather not export

Tell me to **"proceed with assumptions"** and I'll ground the technical layers (Fabric design, Go/PHP
integration, security controls) in the local repos + Fabric corpus, and mark every
requirement-derived specific as `[ASSUMPTION]` in `grounding-gaps.md`. The product-requirement layer
will then carry documented uncertainty until the real docs arrive.

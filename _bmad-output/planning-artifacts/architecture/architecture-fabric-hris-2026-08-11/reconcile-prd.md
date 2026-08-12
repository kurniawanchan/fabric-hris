# Reconcile: ARCHITECTURE-SPINE.md vs. PRD + Addendum

Source PRD: `_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-10/prd.md` + `addendum.md`
Spine checked: `_bmad-output/planning-artifacts/architecture/architecture-fabric-hris-2026-08-11/ARCHITECTURE-SPINE.md`

Bar applied: not "does the spine repeat the PRD" but "does anything load-bearing get dropped or
contradicted." Three findings below meet that bar; several other candidates (FR-2's explicit "0"
display, FR-1's few-seconds lag threshold, the recorded-fallback switchover, SM-4's thesis
screenshot) were considered and excluded — they are correctly left to UI/operational layers below
this spine's altitude.

## Gap 1 — FR-3 (topology) has no defined transport contract; AD-4 silently excludes it

AD-4 ("One relay channel; REST for everything static") is the *only* AD in the spine that defines
how any data reaches the frontend — SSE event types for live data, or REST for static data. Its
`Binds:` line lists FR-1, FR-2, FR-4, FR-5, FR-6, FR-7, FR-8. **FR-3 is missing from that list.**
The Capability → Architecture Map then assigns FR-3 to `backend/topology.ts` + the Network Profile
page, governed only by AD-1 and AD-7 — neither of which says *how* the frontend obtains the
topology data. The Consistency Conventions' REST route list (`/api/scenarios`, `/api/history`,
`/api/config`) also has no topology route.

Net effect: the PRD requires FR-3 to render "as a diagram or equivalent structured view" (§4.2),
and the addendum says it's "hand-modeled, not auto-discovered" — but the spine never states the
mechanism (REST fetch of a static JSON, bundled into the frontend build, etc.) by which that
hand-modeled data crosses the AD-1 backend/frontend boundary at all. This is a real contract gap,
not a stylistic omission, since AD-1 elsewhere insists the frontend "never touches ... any Fabric
operations endpoint" and reaches everything "only through the documented contract (AD-4)" — but
AD-4 doesn't cover FR-3, so FR-3 currently has no documented contract to reach the frontend through.

## Gap 2 — Addendum's plaintext-operations-endpoint safeguard question is not actually closed

The addendum (Concrete data sources, FR-4/FR-5) explicitly flags that peer/orderer `/metrics`
endpoints run with no TLS/client-cert auth, and asks architecture to decide "whether the
dashboard's read-only consumption of them needs any safeguard of its own, or whether
'localhost-only, defense-day-only' is a sufficient boundary." The spine's Consistency Conventions
answer this with `Auth: None` — "no login, no token, no access control anywhere in this product" —
justified by "runs on the operator's own laptop, for the operator alone."

That justification is only true if the backend actually binds to loopback. Express's default
`app.listen(port)` binds `0.0.0.0`, which would expose the relay (including the plaintext
metrics data it re-serves, and the full benchmark-config YAML from FR-8) to any device on the
defense room's Wi-Fi — not just the operator's laptop. The spine never states the concrete
containment rule ("bind to `127.0.0.1` only") that would make its own "no auth needed" reasoning
hold. This is the direct architectural answer the addendum asked for, and it's currently implied
but not written down as a rule — an AD or a Consistency Conventions row should say it explicitly,
the same way AD-8 pins down run-mode discipline.

## Gap 3 — PRD §10's generic-register constraint has no corresponding architectural invariant

PRD §10 (Project & Methodology Constraints) states the dashboard's own UI copy "must never name
the real HRIS platform/company" — this is a project-wide hard rule (see root `CLAUDE.md`'s
confidentiality register, which lists `fabric-network/`, `qa-tests/`, `docs/`, etc. as
hard-clean, verified-zero-hits zones). The spine's Structural Seed places this whole new
`caliper-dashboard/` tree inside `fabric-hris` and pulls live content directly from real
project files (Caliper CLI stdout, `RESULTS.md`, `benchconfigs/*.yaml`) into the UI verbatim
(FR-6, FR-7, FR-8 all require byte-for-byte / unaltered pass-through). None of the eight ADs, the
Consistency Conventions table, or the Structural Seed's comments carry any rule constraining what
those pass-through surfaces may contain, even though FR-7's live CLI tail and FR-8's raw YAML
dump are exactly the kind of unfiltered, verbatim-passthrough surfaces where a leaked real name
has previously slipped through in this project (per `CLAUDE.md`'s own account of two prior
`integration-bridge/` leaks caught only by grep). This is a dropped constraint, not a stylistic
gap — the spine's insistence on verbatim/byte-for-byte fidelity for FR-6/7/8 is in mild tension
with §10's confidentiality requirement, and nothing in the spine reconciles the two.

## Not flagged (considered and excluded)

- FR-2's explicit "0" vs. blank display — UI-layer detail, correctly below spine altitude.
- FR-1's "a few seconds" lag NFR — addressed structurally by AD-6; exact number is rightly deferred.
- UJ-1's recorded-fallback switchover — operational/rehearsal concern, not part of the dashboard's
  own architecture.
- SM-4 (thesis screenshot) — a usage/output concern, not an architectural one.
- Non-goals (no live config editing, no new persistence layer, no code reuse) — all correctly
  honored; the spine adds no config-write path and no database.

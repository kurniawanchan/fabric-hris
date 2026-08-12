# Adversarial Review — Caliper Dashboard Architecture Spine

Reviewed: `ARCHITECTURE-SPINE.md` (Caliper Performance-Benchmark GUI Dashboard, status: final, 2026-08-11)
Method: for each AD, construct two independent builders who each satisfy the AD's literal text but
make a different, unforced choice at the gap the AD leaves open. Report only pairs where the
resulting artifacts are incompatible at runtime — not style disagreements.

**8 incompatibility pairs found.** Ranked by severity below (P1 = breaks the live demo or corrupts
displayed data; P2 = degrades one feed silently; P3 = latent/cosmetic).

---

## P1 — `resourceUsage` null-shape ambiguity (AD-6)

**AD-6's text:** "A node reported `unreachable` sends `null` for its resource-usage fields."

**Builder A (backend/poller):** reads this as "the whole `resourceUsage` object becomes `null`" —
emits `{ type: "node-status", status: "unreachable", resourceUsage: null }`.

**Builder B (frontend/NetworkProfile):** reads the *same sentence* as "the fields inside
`resourceUsage` are individually null" (matching the Consistency Convention's "explicit `null`,
never an omitted key" applied field-by-field) — writes
`const { cpuPct, memPct } = event.resourceUsage;` assuming the object always exists.

**Failure:** the moment a real node goes `unreachable` mid-demo, Builder B's destructure throws
`Cannot destructure property 'cpuPct' of null` in front of the examination committee — the exact
failure mode AD-6 exists to prevent, reintroduced by an ambiguity inside AD-6 itself. Both builders
followed AD-6 to the letter; neither violated AD-4's schema (which never specifies the nested
shape at all).

**Fix:** AD-6 (or AD-4's event schema) must pick one shape explicitly, e.g. "`resourceUsage` is
always a non-null object; each field inside it is `null` when unknown," and give the frontend
contract a literal TypeScript type, not prose.

---

## P1 — "is-live" trigger point disagreement (AD-5 vs AD-3)

**AD-5's text:** is-live is "derived from watcher/heartbeat activity, not operator input."
AD-3 says the backend "tails" the redirected log file. Neither AD says *which* watcher event
flips the flag.

**Builder A (backend/watchers):** flips `isLive = true` the instant `fs.watch`/chokidar fires
`add` on `caliper-run.log` (the file appears the moment the operator's shell redirection opens
it — before Caliper has written a single byte).

**Builder B (backend singleton-state owner, possibly the same file but written at a different
time, or a second contributor finishing AD-5's state module):** flips `isLive` only on the first
parsed, non-empty line/JSONL record — reasoning that "activity" means data, not an empty file
handle.

**Failure:** these two are two different plausible readings of one AD implemented by two people
(or the same AD read twice, weeks apart, by the same author — the spine gives no way to tell which
is "right"). Under Builder A's rule, FR-1/FR-2's frontend "no data yet" idle state (driven by
*absence of metric-update events*) and the mode indicator (driven by `isLive`) visibly disagree
for however long it takes Caliper to actually start emitting — the mode badge says "Live" over an
empty chart. Under Builder B's rule there's a window where the log file plainly exists and is
growing but the indicator still reads "Idle." Either way, whichever half of the UI was built
assuming the other rule shows a contradiction on stage.

**Fix:** AD-5 needs a literal trigger definition: "is-live flips on the first successfully parsed
data line/record from either watched source, not on file-existence/open." Cross-reference it from
AD-3.

---

## P1 — scenario switch mid-write races the watcher rebuild against in-flight SSE fan-out (AD-4 + AD-6 + AD-3)

**AD-4:** `POST /api/scenario` "updates the backend's singleton state (AD-5)." **AD-5:** which
scenario is selected determines "which file paths are watched as a result." **AD-6:** guarantees
per-source failure isolation and heartbeat-driven staleness, but says nothing about a *live*
scenario swap while the old scenario's files are still being actively appended to.

**Builder A (backend/watchers, taking AD-5 literally):** on `POST /api/scenario`, synchronously
closes the old `fs.watch` handles and opens new ones for the new scenario's paths, *before*
responding 200 to the REST call — believes this is what "which file paths are watched as a
result" requires.

**Builder B (backend/stream.ts, taking AD-5's "reflected in every SSE event's context" literally):**
tags every in-flight event with the singleton's *current* scenario value read at emit time, not at
read time — i.e., an event whose bytes were tailed from the *old* scenario's log file, but not yet
flushed through the fan-out queue when the POST landed, gets stamped with the *new* scenario's
name.

**Failure:** the two behaviors compose into: a tail-callback for old-scenario data fires after the
old watcher was torn down (a use-after-close on some watcher libraries, or a dangling callback
closure that still has a reference to the old file descriptor) racing a stream.ts that relabels
whatever event does get through as belonging to the new scenario. The frontend Dashboard, filtering
by "does this event's context match the selected scenario," either drops real final data from the
old run or silently attributes stale old-run numbers to the newly-selected scenario's chart —
exactly the kind of clash the spine's own words ("no session model," "singleton state") make look
safe but don't actually rule out, because AD-4/AD-5/AD-6 never say whose responsibility teardown
ordering is, nor whether events are stamped at capture-time or emit-time.

**Fix:** add an explicit rule (AD-6 addendum or new AD) — event context must be captured at
*read/capture* time, not read from the live singleton at emit time; and scenario-switch watcher
teardown must drain (or explicitly discard) in-flight events for the old scenario before accepting
new-scenario events, with a documented order of operations.

---

## P2 — heartbeat payload shape unspecified beyond `type` (AD-4 + AD-6)

**AD-4** only guarantees a `type` discriminant exists; **AD-6** says heartbeat exists so "the
frontend can detect silence... without guessing a timeout on its own," implying heartbeat carries
per-source liveness, but never says so explicitly.

**Builder A (backend/stream.ts):** emits a bare `{ type: "heartbeat", timestamp }` — a pure
"I'm still alive" ping, nothing about the three individual sources.

**Builder B (frontend/Notifications or NetworkProfile):** builds its per-feed staleness-muting UI
(per the UX spine, referenced by AD-6) assuming heartbeat carries something like
`{ type: "heartbeat", sources: { log: <ts>, jsonl: <ts>, metrics: <ts> } }` so each page can mute
only the feed that's actually gone quiet, not the whole app.

**Failure:** Builder B's per-source staleness logic has nothing to read; either it silently no-ops
(feature looks built, isn't) or it throws on `event.sources` being undefined. Two frontend page
owners could each guess differently — one builds "global heartbeat = grey out everything," another
"per-source heartbeat = grey out just the dead one" — and only one matches whatever the backend
author decided, unrecorded anywhere in the spine.

**Fix:** AD-4's event-shape table should give heartbeat's exact payload, including whether/how it
reports individual-source liveness.

---

## P2 — SSE event `context` field presence is inconsistent across event types (AD-4 + AD-5)

**AD-5:** singleton state is "reflected in every SSE event's context." **AD-4:** defines only
`type` and (for `node-status`) `status` as guaranteed fields; says nothing about a `context` field
existing on the wire at all.

**Builder A (backend/stream.ts):** attaches `context: { scenario, isLive }` only to
`metric-update` and `node-status` events (the ones that are meaningfully scenario-scoped),
reasoning `log-line` and `heartbeat` are scenario-agnostic infrastructure signals.

**Builder B (frontend/Notifications page, consuming `log-line`):** filters incoming log lines by
`event.context.scenario === selectedScenario` to avoid showing a stale scenario's tail output after
a switch — because AD-5 says "every SSE event," taken literally.

**Failure:** `event.context` is `undefined` on every `log-line` event Builder A emits; Builder B's
filter throws or (with optional chaining) silently passes everything through unfiltered — after a
scenario switch, the Notifications tab keeps showing the *old* scenario's CLI output tail forever,
contradicting the mode/scenario indicator elsewhere on the same page.

**Fix:** AD-5's "every SSE event" needs to be either walked back ("context is attached to
`metric-update` and `node-status` only") or enforced literally with the event schema in AD-4
listing `context` as a mandatory field on all four types.

---

## P2 — topology node identifiers vs poller's endpoint-map identifiers can diverge (AD-7 + FR-4/5 poller)

**AD-7:** topology is hand-modeled to exactly `Org1`, `OrgClient-tenant01`, `Org3` in
`topology.ts`. The capability map assigns FR-4/FR-5 (`backend/poller`) to the *same* AD-7 but does
not say the poller must import its node list from `topology.ts` rather than maintaining its own
config of which operations endpoints to poll.

**Builder A (`topology.ts` owner):** models nodes with ids like `"org1-peer0"`,
`"orgclient-tenant01-peer0"`.

**Builder B (`backend/poller` owner, working from the PRD/network's real container names since
that's what the actual `/metrics` URLs are keyed by):** configures polling targets keyed as
`"peer0.org1.example.com"`, `"peer0.tenant01.example.com"` (matching real hostnames, not the
topology's display ids).

**Failure:** `node-status` events arrive keyed by the poller's hostname-style id; the frontend
Network Profile page, built against `topology.ts`'s ids (fetched once via `GET /api/topology`),
can't match incoming `node-status` events to the topology nodes it rendered — every node shows
"never received a status" (staleness) forever, even though the poller is working correctly.

**Fix:** AD-7 (or a new AD) should mandate a single canonical node-id source — `topology.ts` — that
`backend/poller` must import and key its polling/emission by, rather than each independently
choosing an id scheme.

---

## P3 — `GET /api/scenario` bootstrap vs SSE context ordering race (AD-4 + AD-5)

**Builder A (Dashboard page):** on mount, calls `GET /api/scenario` once to seed initial UI state,
then treats every subsequent SSE `context.scenario` as authoritative and never re-polls REST.

**Builder B (NetworkProfile page, built independently):** never calls `GET /api/scenario` at all —
relies solely on the first SSE event's `context` to learn the current scenario, since AD-5 says
SSE always reflects it.

**Failure:** if the operator POSTs a scenario change in the small window between page load and the
first SSE event/heartbeat arriving, Dashboard (which already rendered off its one-shot GET) shows
the old scenario until the next metric-update; NetworkProfile (which waited for SSE) shows the new
one immediately. The two tabs of the same single-operator dashboard disagree about "which scenario
is active" for a few seconds — small, but visible if the operator switches scenarios and clicks
between tabs during a defense.

**Fix:** AD-5 should state explicitly that REST `GET /api/scenario` and the SSE context are to be
treated as equally authoritative and always consistent (i.e., GET must return the live singleton
value at request time, and pages must not cache it past the next SSE context update) — currently
implied but not stated as a rule two independent page-builders would both apply the same way.

---

## P3 — historical-run parse shape for FR-6 is unspecified between routes and TableList (AD-4)

**AD-4/structural seed:** `backend/routes` serves FR-6 by reading `RESULTS.md`/`QA6-RESULTS.md`;
frontend TableList consumes it over REST. Neither the AD nor the capability map specifies whether
the REST response is pre-parsed structured JSON (rows/columns) or the raw Markdown text for the
frontend to parse client-side.

**Builder A (backend/routes):** treats "REST for everything static" (AD-4) as license to just
proxy the file's raw bytes (`{ content: "<markdown>" }`), since "passthrough-content
confidentiality" convention already frames FR-6 as pass-through.

**Builder B (frontend TableList):** built off DESIGN.md's table mockup, expects
`{ runs: [{ id, timestamp, tps, latencyP99, ... }] }` — structured rows ready to render in a table
component.

**Failure:** TableList renders nothing (or crashes mapping over `.runs` that doesn't exist) because
the backend sent Markdown prose, not rows — a straightforward but real contract gap since AD-4
governs *transport*, not *payload shape*, for this one FR.

**Fix:** extend AD-4 (or add a companion data-contract note) to state whether FR-6/FR-8's REST
payloads are structured JSON or raw-passthrough text — both are legitimate "REST," but a builder
of the route and a builder of the page need the same answer.

---

## Summary of required spine changes

1. Give `resourceUsage`'s null-shape a literal type, not prose (closes the P1 pair).
2. Define is-live's exact trigger event (first parsed data, not file-open) and cross-reference it
   from AD-3 into AD-5.
3. Add a capture-time-vs-emit-time rule for SSE event context, plus a teardown-ordering rule for
   scenario switches (new AD or AD-6 addendum).
4. Specify heartbeat's exact payload (bare ping vs per-source timestamps).
5. State explicitly which event types carry the `context` field — all four, or a named subset.
6. Mandate `backend/poller` import node ids from `topology.ts` rather than maintaining its own
   scheme (AD-7 addendum).
7. State that `GET /api/scenario` and SSE context are always mutually consistent.
8. Specify FR-6/FR-8's REST payload shape (structured JSON vs raw passthrough) per endpoint.

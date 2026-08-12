# Tech Verification Review — ARCHITECTURE-SPINE.md Stack Table

**Reviewed:** 2026-08-11
**Target:** `_bmad-output/planning-artifacts/architecture/architecture-fabric-hris-2026-08-11/ARCHITECTURE-SPINE.md`, `## Stack` table + AD-4/AD-2's Express/SSE reliance.
**Method:** Each claim independently checked via live web search on 2026-08-11 (not asserted from training data).

## Verdict: **All confirmed.**

Every version and behavioral claim in the Stack table checks out against current (2026-08-11) reality. No stale or invented entries found.

## Findings

| Claim in spine | Verification | Status |
| --- | --- | --- |
| Node.js 24.x (Active LTS), 22.x (Maintenance LTS) | Confirmed. Node 24 became Active LTS 2025-10-28 (EOL 2028-04-30); Node 22 is Maintenance LTS (EOL 2027-04-30). Node 26 is the current Current release, becoming Active LTS 2026-10-28 — not yet relevant. | ✅ Accurate as of 2026-08-11 |
| Express 5.2.1 | Confirmed as latest npm release (published ~Dec 2025; 5.2.0 had an extended-query-parser regression, fully reverted in 5.2.1). | ✅ Accurate |
| React 19.2.7 | Confirmed via GitHub release tag `v19.2.7` (June 1, 2026) and npm `react`/`react-dom`. Fixed a Server Actions FormData regression from 19.2.6. | ✅ Accurate |
| Vite 8.x, 8.2 patch line | Confirmed. Vite 8.0.0 released 2026-03-12; npm latest is 8.2.1 (published within days of this review). 8.2.x and 8.1.x both receive backported patches. | ✅ Accurate |
| shadcn/ui CLI v4, Vite template, `npx shadcn@latest init -t vite` | Confirmed. shadcn's March 2026 changelog ("shadcn/cli v4") states `init` now scaffolds full project templates for Next.js, **Vite**, Laravel, React Router, Astro, and TanStack Start, with dark mode included for Next.js and Vite. The exact flag syntax in current docs is `--template vite` (`-t` is the short alias); functionally equivalent to what the spine cites. | ✅ Accurate, minor flag-spelling note (see below) |
| TypeScript via the Vite React-TS template | Confirmed. `npm create vite@latest -- --template react-ts` remains the current, documented way to scaffold a TypeScript+React Vite project; this is unchanged in 2026 guidance. | ✅ Accurate |
| Express 5.x Router/middleware still supports the long-lived streaming/SSE pattern AD-4 needs | Confirmed. Setting `Content-Type: text/event-stream` and using `res.write()` (never `res.send()`/`res.end()`) to keep the connection open is unchanged in Express 5 — no special configuration is required; Node's chunked transfer-encoding kicks in automatically. This is the same idiom used pre-5.x. | ✅ Supported, no regression |
| shadcn Vite template still defaults to React + TypeScript | Confirmed. Current shadcn Vite installation docs and the March 2026 CLI v4 changelog both describe the Vite template flow as React+TypeScript-based (Tailwind + `@` path alias + `tsconfig.json` paths); no indication the default flipped to JS-only or another framework. | ✅ Accurate |

## Minor note (not a defect, worth a follow-up edit)

- The spine's literal invocation `npx shadcn@latest init -t vite` uses the short flag `-t`. Current official docs primarily show the long form `--template vite` (with `-t` documented as its alias in the CLI v4 changelog). This is not a stale/invented claim — the short flag is real and functional — but the architecture doc could cite the long form to match what a reader will see first in current shadcn docs. Cosmetic; does not require a design decision.

## What was actually checked (vs. what would have been asserted from training data)

- Node.js LTS status/schedule: web-verified against `nodejs/release` GitHub schedule and current EOL trackers — training data alone would not have known Node 24 promoted to Active LTS in Oct 2025 or Node 26's Oct 2026 promotion date.
- Express exact patch (5.2.1) and its fix history: web-verified against the GitHub releases page and npm — a training-data guess would likely have stopped at 5.1.0 (the "default on npm" milestone) and missed the 5.2.0 regression/5.2.1 revert.
- React exact patch (19.2.7) and its release date: web-verified against the `facebook/react` GitHub release tag — well past training cutoff.
- Vite 8.x existence and the 8.2 patch line: web-verified — Vite 8.0.0 (March 2026) postdates training data; this was a real risk area and is now independently confirmed live rather than assumed.
- shadcn CLI v4's Vite full-template scaffolding (a March 2026 feature): web-verified against shadcn's own changelog — this is a materially new CLI behavior since training and could easily have been invented or misremembered; it was not.
- Express 5's SSE/streaming-response compatibility: web-verified against current Express/SSE tutorials dated 2026 — confirms no `res.write()`/streaming API break landed in the 5.x line that would threaten AD-4.

## Conclusion

No stack entry in this spine was found to be stale, deprecated, non-existent, or contradicted by current library behavior. AD-4's SSE-based long-lived response pattern remains fully supported by Express 5.2.1's Router/middleware model. The spine's technology decisions pass live reality-check as of 2026-08-11.

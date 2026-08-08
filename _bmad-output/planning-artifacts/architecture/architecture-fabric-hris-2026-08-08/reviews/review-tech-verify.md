# Tech-Verification Review — Integration Bridge Architecture Spine

- **Target:** `_bmad-output/planning-artifacts/architecture/architecture-fabric-hris-2026-08-08/ARCHITECTURE-SPINE.md`
- **Lens:** every committed technology/version claim checked against the real repo state and/or current external documentation, not asserted from training-data recall alone.
- **Date:** 2026-08-08

## 1. Go 1.25.9 pin (Stack table)

**Claim:** "Go 1.25.9 (matches every other module in this repo except the chaincode)."

**Verified TRUE.**

- `write-path-integration/go.work`: `go 1.25.9`, `use (./gateway-client ./ipfsclient ./keystore ./writepaths)`.
- All four `write-path-integration/*` module `go.mod` files independently declare `go 1.25.9` (gatewayclient, keystore, writepaths, ipfsclient).
- All six `fabric-network/tools/*` module `go.mod` files declare `go 1.25.9`: `ccdeploy`, `pilscan`, `jcsverify`, `netjoin`, `raftfaulttest`, `tenantprovision`.
- The one real exception, exactly as the spine states: `fabric-network/chaincode/employeeprofilerecord/go.mod` declares `go 1.21` (vendored, per CLAUDE.md's documented gotcha).

No mismatches found. The claim is fully grounded in the live repo, not asserted.

## 2. AD-6's Go 1.22+ enhanced-`ServeMux` claim

**Claim:** "Go 1.22+'s enhanced `ServeMux` (method+path patterns) covers all five routes" — used to justify zero third-party HTTP router/framework dependency.

**Verified TRUE, but the framing is broader than what this design actually needs — worth tightening.**

Confirmed via the official Go blog (`go.dev/blog/routing-enhancements`) and current `pkg.go.dev/net/http` docs (fetched at Go 1.26.5, i.e. after 1.25.9 — nothing regressed):

- Go 1.22 (Feb 2024) added method-prefixed patterns (`"POST /items"`), which restrict a handler to one HTTP method and return `405 Method Not Allowed` with an `Allow` header on a mismatch — this did not exist pre-1.22 (handlers previously had to check `r.Method` manually).
- Go 1.22 also added `{id}`-style single-segment wildcards, `{path...}` multi-segment catch-alls, and `{$}` exact-trailing-slash matching, with values read via `Request.PathValue(key)`.
- Precedence is specificity-based (literal > method-specific wildcard > wildcard > catch-all), is registration-order-independent, and genuinely conflicting patterns **panic at registration time** — a real, sharp-edged limitation worth knowing before wiring five routes, though not one that bites a fixed, non-overlapping route set.
- This capability set is present unchanged through at least Go 1.26 docs — nothing about it was removed or altered in later releases. The claim is not stale.
- The Go team's **own** caveat, stated directly in that blog post: *"Third-party web frameworks remain a fine choice for current users or programs with advanced routing needs."* `ServeMux` is deliberately minimal — no regex, no query-param matching, no built-in per-route middleware chaining (AD-1's own pipeline supplies that instead via plain `http.Handler` wrapping, which is unaffected by this and works with any mux).

**The oversell to flag:** AD-6 bundles "method+path patterns" as one justification, but per `ADR-0022` and the spine's own Consistency Conventions table, the five routes carry `employeeInternalID` and every other identifier **in the JSON body**, not the URL path — the route set is five *static* method+literal-path pairs (one per `ProfileSection` enum value), not routes with `{id}`-style path parameters. The wildcard/`PathValue()` half of the Go 1.22 feature set — including its one real ergonomic caveat (values come back as untyped strings, no automatic int/UUID conversion) — is not actually exercised by this design at all. The only feature genuinely load-bearing here is the **method-restricted literal-path registration**, which is real, current, and sufficient. Recommend AD-6 either drop "path patterns" from the justification or explicitly note that path wildcards are unused so a future reader doesn't infer this design needs, or is exercising, parameterized routes.

## 3. AD-2's `fabric-network/tools/*` precedent claim

**Claim:** `integration-bridge/`'s standalone-module-with-a-`replace`-directive layout matches "the `fabric-network/tools/*` precedent of a standalone module depending on, not sharing a workspace with, what it needs."

**Partially true — one half confirmed, the other half not actually demonstrated by any existing code.**

Checked all six `fabric-network/tools/*/go.mod` files and their `.go` sources:

- **Standalone-module half: TRUE.** None of the six tools (`ccdeploy`, `pilscan`, `jcsverify`, `netjoin`, `raftfaulttest`, `tenantprovision`) appears in any `use` block of any `go.work` — the only `go.work` in the repo is `write-path-integration/go.work`, which lists only its own four library modules. Each tool is genuinely its own module with its own `go.mod`, exactly as CLAUDE.md's Go-workspace-gotcha section describes.
- **"Depending on... what it needs" half: NOT demonstrated.** `grep`-ing all six `go.mod` files found **zero `replace` directives anywhere**. Three tools (`ccdeploy`, `netjoin`, `raftfaulttest`, `tenantprovision`) have no `require` block at all beyond the `go` directive — they are pure-stdlib, shelling out to `docker exec`/`peer`/`osnadmin` binaries per CLAUDE.md's documented architecture, not importing local Go packages from elsewhere in the repo. `pilscan` and `jcsverify` do have `require`s, but only of **public, versioned, third-party** packages (`github.com/hyperledger/fabric-protos-go-apiv2`, `github.com/cyberphone/json-canonicalization`) — not of any sibling directory in this repo via `replace`.
  - `tenantprovision/main.go` does reference `const networkDir = "../../network"`, but that is a filesystem path used for shelling out to real binaries against `fabric-network/network/` config, not a Go-level module/import dependency — a different mechanism entirely from the `replace ... => ../write-path-integration/writepaths` pattern AD-2 proposes for `integration-bridge`.
  - `pilscan/main.go`'s comments mention `write-path-integration/gateway-client` only to note that *no new library was introduced* and that some field-name strings mirror write-path-integration's own test fixtures — again, not a code dependency.

**Conclusion:** there is no existing tool in this repo that actually imports or `replace`-depends on code living in another directory/module the way `integration-bridge` is being asked to depend on `write-path-integration/writepaths`. AD-2's chosen pattern (standalone module + `replace` pointing at a sibling module) is a **reasonable, idiomatic Go choice on its own merits**, but it is not precedented by any of the six existing tools — it would be a new pattern for this repo, not a repetition of an established one. The rule's phrasing overstates its own precedent; it should either cite this as a novel-but-sound pattern, or drop the "matching the ... precedent" framing for the dependency mechanism specifically (the "standalone, not in the workspace" framing is fine to keep — that part is real).

## Summary table

| # | Claim | Verdict |
|---|---|---|
| 1 | Go 1.25.9 pinned everywhere except chaincode | Confirmed — checked go.work + 4 write-path-integration go.mod + 6 fabric-network/tools go.mod + chaincode go.mod (1.21) |
| 2 | Go 1.22+ ServeMux method+path patterns justify no router | Confirmed accurate and not stale, but oversold: only the method-restriction half is actually used; no route in this design uses path wildcards |
| 3 | `fabric-network/tools/*` precedent for standalone-module-depending-on-sibling-via-replace | Half confirmed (standalone, not in go.work is real), half not demonstrated (no existing tool uses a `replace` or any cross-directory Go dependency — this would be a new pattern, not a repeated one) |

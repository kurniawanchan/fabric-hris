# Web/Reality Verification Review — ARCHITECTURE-SPINE.md

**Target:** `architecture-fabric-hris-talenta-2026-08-11/ARCHITECTURE-SPINE.md`
**Method:** checked the Stack table's pins against the actual `go.mod`/`go.sum`/module-cache state
of this repo, and against live web search; grepped for a `composer.json` for the PHP/Yii2 claim;
read every AD for other version/technology/existence claims.

## Verdict

**Pass, with one process gap to flag, not a factual error.** Both pinned versions in the Stack
table are real and match the existing repo exactly. Nothing in the spine asserts a plausible-but-
wrong version from training data. The one gap is that the spine's Stack table doesn't *cite* the
files it's actually drawing from — it states the versions as bare facts, which is indistinguishable
at a glance from an asserted-from-memory claim, even though (as verified below) the underlying
numbers are correct.

## Stack table line-by-line

| Claim | Verification | Result |
| --- | --- | --- |
| `integration-bridge` runtime: Go 1.25.9 | `integration-bridge/go.mod:3` reads `go 1.25.9` verbatim. Also matches every other Go module in the repo (`write-path-integration/{gateway-client,writepaths,ipfsclient,keystore}/go.mod`, all `fabric-network/tools/*/go.mod`) except `chaincode/employeeprofilerecord` (deliberately pinned to Go 1.21 per CLAUDE.md). Web search separately confirms Go 1.25.9 is a real, currently-shipping point release (security release paired with 1.26.2, per the golang-announce list). | **Confirmed — matches existing go.mod, and independently real per web search.** |
| `hyperledger/fabric-gateway`: v1.12.0 | `integration-bridge/go.mod` lists it as an indirect require at exactly `v1.12.0`; `write-path-integration/gateway-client/go.mod` requires it directly at `v1.12.0`. `gateway-client/go.sum` has a valid hash line for it, and the module is physically present, already downloaded, in the local Go module cache at `github.com/hyperledger/fabric-gateway@v1.12.0` — i.e., this version was actually resolved and built against, not just typed into a doc. Web search alone did not turn up v1.12.0 (indexed results topped out around v1.10.0/v1.11.0 on npm) — that's a lag in what's been crawled, not evidence the version is wrong; the go.sum + module-cache evidence is stronger than a search index for "does this exact version exist and resolve." | **Confirmed via existing go.sum/module cache — real version, correctly matches existing pin.** |
| `talenta-core` new job class: PHP / Yii2 (existing framework, no new language) | No `composer.json` exists anywhere in this repo (`talenta-core` is an external brownfield repo not vendored here), so there is no local lockfile to cite. The claim is qualitative ("existing framework, no new language introduced"), not a version pin, so there's nothing numeric to falsify — but it also means this line is **not verifiable from this repo alone**, unlike the two Go pins above. | **Unverifiable locally — see finding below.** |

## Findings

1. **The two version pins are correct and traceable to already-resolved dependencies, not asserted from training data.** Both `go.mod` and `go.sum` (plus the populated module cache) predate this spine and match it exactly — this is a case where citing the existing repo state would have been trivial and the spine's numbers happen to agree with it.

2. **Process gap: the Stack table doesn't cite its source.** Per the task framing, this spine binds an existing brownfield stack rather than choosing a new one — the correct move is to *cite* `go.mod`/`composer.json` rather than *assert* a version number. The spine does the former correctly in substance (the numbers are right) but not in form (no file:line citation, e.g. `integration-bridge/go.mod:3`). A future reader/reviewer has no way to tell, from the document alone, whether "Go 1.25.9" was read off the file or recalled from memory — it was the former, but the spine doesn't show its work. Recommend adding inline citations to the Stack table (e.g. `Go 1.25.9 (integration-bridge/go.mod:3)`, `fabric-gateway v1.12.0 (gateway-client/go.mod, go.sum)`).

3. **The PHP/Yii2 line has no lockfile to cite in this repo**, because `talenta-core` isn't vendored here — it's the one Stack-table row that's genuinely unverifiable against this codebase and wasn't checked against a live copy of `talenta-core` or any composer.json/composer.lock. This isn't necessarily wrong (Yii2 is a real, still-maintained PHP framework, and the spine explicitly says "no new language introduced" rather than pinning a version), but if a specific Yii2 version or PHP version ever gets asserted in a downstream doc, it should be sourced from `talenta-core`'s own `composer.json`/`composer.lock`, not asserted.

4. **No other technology-existence or version claims appear in the spine that need independent verification.** The AD sections (AD-1 through AD-6) and Structural Seed reference only internal repo paths and existing code constructs (`writepaths.Hooks`, `gatewayclient.SubmitRecordProfileSection`, `classify.go`, `validate.go`, the `ProfileSection` enum) — these are architectural/design decisions about an existing codebase, not external library/framework choices, so they fall outside this verification's scope (existence of named files/symbols wasn't part of this task, but nothing in the AD text implies an unfamiliar or exotic external dependency that would need a plausibility check).

5. **No confidentiality-register violation found in the reviewed content** — the spine correctly uses "Talenta"/"talenta-core" (research-substrate register, this document lives under `_bmad-output/planning-artifacts/`, which is explicitly exempt from the generic-placeholder rule), consistent with CLAUDE.md's confidentiality register.

## Recommendation

Treat this as **pass** on substance (no false or outdated version claims), but add a one-line
citation per Stack-table row pointing at the actual `go.mod`/`go.sum` path, and add a note that the
PHP/Yii2 row is unverified against `talenta-core`'s own manifest (not available in this repo) rather
than silently implying it was checked the same way the Go rows were.

# QA-1 — Unit test coverage audit (U-1..U-8)

> Audits existing unit-test coverage against `agent-suite/06-roadmap/test-strategy.md` §1's `U-1..U-8`
> table. Every row below is a **real test I ran myself** (fresh, this pass) — file:line, PASS/FAIL,
> and what it actually proves, not what the spec text merely says it should prove. Where a `U-#`'s
> own text no longer matches the ratified design, that is called out explicitly rather than silently
> reinterpreted (per this item's instruction and ADR-0021's supersession of the `pseudonymKey`
> hierarchy).

## How this was run

```
cd fabric-network/chaincode/employeeprofilerecord && go test -v ./...
cd write-path-integration/gateway-client        && go test -v ./...
cd write-path-integration/keystore              && go test -v ./...
cd write-path-integration/writepaths            && go test -v ./...
cd write-path-integration/ipfsclient            && go test -v ./...
```

All five commands were run from inside the specific module directory (the `go.work` quirk — running
from `write-path-integration/` itself fails with "directory prefix . does not contain modules listed
in go.work"). All five modules' `go test ./...` runs are **unit-only**: every `*_integration_test.go`
file in `gateway-client` and `writepaths` (and `ipfsclient`) carries `//go:build integration` and is
excluded by a plain `go test ./...` — confirmed by reading each file's build tag, not assumed. Result:
every test named below ran against mocks/in-memory stores only, no live Fabric/IPFS network touched.

Additionally, `fabric-network/tools/jcsverify` (a standalone CLI, its own `go.mod`, **not** part of
`go.work` and **not** invoked by any module's `go test ./...`) was built and run directly against the
pinned `github.com/cyberphone/json-canonicalization` library's own RFC 8785 test vectors, because its
own doc comment states it "verifies (test U-4)" and it is the only place `unicode.json`/`weird.json`/
`structures.json` vectors are exercised anywhere in this repo:

`jcsverify` has its own `go.mod` and is not reachable as a build target from the repo root or from
`write-path-integration/` — `cd` into its own module directory first (the same go.work-module-
boundary quirk documented elsewhere in this repo, confirmed to also apply here after an earlier
draft of this report gave a command that fails with "go.mod file not found" when run from the
repo root):

```
$ cd fabric-network/tools/jcsverify
$ go build -o /tmp/jcsverify_bin .
$ /tmp/jcsverify_bin "$(go env GOMODCACHE)/github.com/cyberphone/json-canonicalization@v0.0.0-20241213102144-19d51d7fe467/testdata"
arrays.json      PASS (32 bytes)
french.json      PASS (130 bytes)
structures.json  PASS (98 bytes)
unicode.json     PASS (30 bytes)
values.json      PASS (118 bytes)
weird.json       PASS (214 bytes)

SUCCESS: all 6 RFC 8785 test vectors byte-exact-match, idempotent on re-canonicalization — PB-3/G-24 closes on github.com/cyberphone/json-canonicalization@v0.0.0-20241213102144-19d51d7fe467
```

## Coverage matrix

| U-# | test-strategy.md target | Real covering test(s) | What it actually proves | Status |
|-----|--------------------------|------------------------|--------------------------|--------|
| **U-1** | `DataHash` builder: `SHA-256(salt‖JCS(section))`; salt ≥128-bit CSPRNG, new per record, never returned/logged | `TestComputeDataHash_Deterministic` (gateway-client/digestbuilder_test.go:31), `TestComputeDataHash_KeyOrderIndependent` (:56), `TestComputeDataHash_DifferentSaltDifferentHash` (:77), `TestComputeDataHash_RejectsShortSalt` (:91), `TestComputeDataHash_MatchesManualConstruction` (:104); salt freshness/floor: `TestSaltStore_FreshPerRecordVersion` (keystore/keystore_test.go:9) | Hash is deterministic for fixed inputs, salt is actually mixed in (not ignored), the exact byte layout (`salt ‖ canonical-JSON`, no separator) matches ADR-0011, salt below the 128-bit/`MinSaltBytes` floor is rejected, and two different record versions of the same employee/section get different salts | **PASS** (all 5 + 1 ran fresh, green) |
| **U-2** | `employeeKey_i` derivation: `HMAC-SHA256(pseudonymKey, tenantId‖employeeInternalId)`, deterministic, not recoverable without `pseudonymKey` | `TestEmployeeKeyStore_OneKeyPerEmployeeGeneratedOnce` (keystore/keystore_test.go:64), `TestEmployeeKeyStore_DifferentEmployeesDifferentKeys` (:83) | The **actually-shipped** derivation: `employeeKey_i` is a CSPRNG value generated **once** per employee and reused (not regenerated on second call), and independently random across employees | **PASS**, but **U-2's own text is STALE** — see "Stale premises" below. No test anywhere derives `employeeKey_i` from a `pseudonymKey` via HMAC, because that code path does not exist in this codebase |
| **U-3** | `EmployeeID`/`UpdatedBy` builder: `HMAC-SHA256(employeeKey_i, "id"/"actor"‖user_id)`; regression guard against deriving directly from `pseudonymKey` | `TestComputeEmployeeID_DeterministicAndKeyed` (gateway-client/digestbuilder_test.go:128), `TestComputeEmployeeID_RejectsEmptyKey` (:154), `TestComputeUpdatedBy_DifferentActorsDifferentPseudonyms` (:164), `TestComputeUpdatedBy_DistinctFromEmployeeID` (:183) | `EmployeeID`/`UpdatedBy` are computed from `employeeKey_i` via HMAC with domain-separated messages (`"id"` vs. `"actor"‖user_id`), deterministic per key, different actors/keys produce different pseudonyms, empty key rejected | **PASS** — the *functional* assertion (computed from `employeeKey_i`, domain-separated, deterministic) holds against current code. The "regression guard against ever deriving directly from `pseudonymKey`" framing is stale (see below): there is no `pseudonymKey` left in the codebase to regress into |
| **U-4** | RFC-8785 JCS canonicalization: byte-identical output for equivalent JSON (key order, unicode, numbers) | `TestJCS_ReordersKeysAndNormalizesNumbers` (gateway-client/digestbuilder_test.go:199) — key order + ECMAScript number normalization only; **`jcsverify`** (fabric-network/tools/jcsverify/main.go) — all 6 RFC 8785 reference vectors including `unicode.json`, `weird.json`, `structures.json`, run against the pinned library, byte-exact + idempotency-checked | Between the two: key-order independence, number normalization, **and** unicode/structural edge cases are all exercised against the RFC's own reference vectors, not just hand-picked examples | **PASS** (digestbuilder_test.go test ran fresh via `go test`; `jcsverify` built and run fresh, see output above). Note: `jcsverify` is a separate binary with its own `go.mod`, **not** reachable via any module's `go test ./...` — a CI pipeline that only runs `go test ./...` per module would silently stop re-verifying the unicode/structures/weird vectors. Flagged as a process gap, not a code gap |
| **U-5** | Chaincode ABAC: CID `GetAttributeValue`(role/tenant) gates read/write; wrong role denied; explicit reject on MSP-verification failure | `TestAuthorizeSubmit_PerOrg` (chaincode/identity_test.go:14), `TestAuthorizeEvaluate_PerOrg` (:44), `TestCallerMSPID_FailsClosedWhenIdentityUndeterminable` (:74), `TestIsOrgClientMSP` (:90), `TestRecordProfileSection_RejectsCallerThatFailsMSPVerification` (chaincode/record_profile_section_test.go:137), `TestRecordProfileSection_PerOrgWriteAllowDeny` (:150), `TestQueryFunctions_PerOrgReadAllowDeny` (chaincode/queries_test.go:145) | Deny-by-default ABAC actually gates every write/read function, per-org, and a caller with no determinable identity at all is rejected before any state read/write | **PASS** (all 7 ran fresh, green). **U-5's own text is a PARTIAL mismatch** — see "Stale premises" below: the mechanism actually implemented is **MSP-ID org-allow-list**, not `cid.GetAttributeValue(role/tenant)` attribute-based ABAC as ADR-0005 (which U-5 cites) specifies |
| **U-6** | No-duplicate-on-identical-content: resubmitting byte-identical canonical JSON produces no new record | `TestRecordProfileSection_NoOpOnIdenticalDataHash` (chaincode/record_profile_section_test.go:51) | Same `dataHash` for the same `(employeeID, profileSection)` → RecordID/Version/Timestamp unchanged, no new world-state key written | **PASS** |
| **U-7** | `PrevHash` chaining per section per employee; `Version` increments; a different section starts its own independent chain | `TestRecordProfileSection_ChainsCorrectlyAcrossVersions` (chaincode/record_profile_section_test.go:103), `TestRecordProfileSection_RejectsStalePrevHash` (:75), `TestGetProfileHistory_AscendingChronologicalOrder` (chaincode/queries_test.go:50), `TestGetEmployeeProfileSummary_FiveEntriesSomeAnchored` (chaincode/queries_test.go:111) | Version increments and `PrevHash(N+1) == DataHash(N)` within one `(employeeID, section)`; a stale/mismatched `PrevHash` is rejected in both the "head exists" and "no head yet" shapes; the SAME employee writing to two different sections (`PAYROLL`, `PERSONAL`) tracks independent `DataHash`/`Version` per section | **PASS** (all 4 ran fresh). Minor note: no single test explicitly asserts "chain independence" by comparing `PrevHash` values across two sections for the same employee in one assertion block — coverage is adequate but assembled from several tests' side effects rather than one dedicated test. Not flagged as a gap worth a new test (the property is exercised, just not narrated in one place) |
| **U-8** | Record identity: `RecordID` is a fresh UUID v4; `Timestamp` present; `ProfileSection` validated against the five allowed values, others rejected | Timestamp: `TestRecordProfileSection_TimestampIsLedgerTime` (chaincode/record_profile_section_test.go:36); enum validation: `TestIsValidProfileSection` (chaincode/asset_test.go:10), `TestRecordProfileSection_RejectsInvalidProfileSection` (chaincode/record_profile_section_test.go:124); RecordID presence only: `TestRecordProfileSection_SuccessfulWrite` (:10, `require.NotEmpty(t, res.RecordID)`) | Timestamp comes from ledger time (`GetTxTimestamp`), never the caller-supplied value; the five-value enum is enforced case-sensitively; a record gets *some* non-empty RecordID | **PASS** for Timestamp/enum. **RecordID "fresh UUID v4" was NOT covered by any existing test before this audit** — `require.NotEmpty` proves presence, not freshness or v4-ness, and (see "Stale premises" below) the implementation does not generate a v4 UUID at all. Closed a narrow slice of this with one new test (see below); the literal "fresh UUID v4" wording remains a genuine, disclosed mismatch, not something a test can "pass" |

**Summary: 8/8 `U-#` have at least one real, passing covering test.** Two (`U-2`, `U-8`) have covering
tests that verify the *actually-shipped* behavior rather than the literal spec text, because the spec
text is stale against a later, ratified decision (`U-2`) or an explicitly-flagged, deliberate
implementation deviation (`U-8`). One (`U-5`) has full behavioral coverage of a *different* mechanism
than the one the spec text names. None of the 8 has zero coverage.

## New test added (one, per this item's ceiling)

**File:** `fabric-network/chaincode/employeeprofilerecord/chaincode/u8_recordid_coverage_gap_test.go`
(new file only — no existing `*_test.go` touched)

**Gap it closes:** `deterministicRecordID` (`record_profile_section.go:34`) had **zero direct unit
test** anywhere in the repo before this. Every existing caller-side test only asserted
`res.RecordID` is non-empty. This new test (`TestDeterministicRecordID_U8CoverageGap`) asserts,
directly against the function: (a) the same `txID` always yields the same `RecordID` — the actual
property Fabric endorsement-matching depends on; (b) two different `txID`s never collide; (c) the
output parses as a syntactically valid UUID and is **version 5**, not version 4.

**Why it does not "close" U-8 as originally worded:** it deliberately does **not** assert "UUID v4" —
that would be asserting something false about the shipped code. `record_profile_section.go:15-33`
already self-documents this as a deliberate, flagged deviation from `data-model.md §2.1`'s "UUID v4"
wording, required by Fabric's determinism constraint (`chaincode4ade.rst#technical-problem`:
independent endorsing peers must produce byte-identical write sets, which a `crypto/rand`-based UUID
v4 cannot do across independent peer simulations). This test pins the *actual* behavior as a
regression guard — if a future refactor reintroduces true randomness, this test fails locally instead
of only breaking cross-peer endorsement on a live network — while leaving the wording mismatch
explicitly recorded rather than silently making the test agree with the (incorrect) spec text.

Fresh run:
```
$ cd fabric-network/chaincode/employeeprofilerecord && go test -v -run TestDeterministicRecordID_U8CoverageGap ./...
=== RUN   TestDeterministicRecordID_U8CoverageGap
=== RUN   TestDeterministicRecordID_U8CoverageGap/same_txID_always_yields_the_same_RecordID_(required_for_cross-peer_endorsement_matching)
=== RUN   TestDeterministicRecordID_U8CoverageGap/different_txIDs_yield_different_RecordIDs
=== RUN   TestDeterministicRecordID_U8CoverageGap/output_is_UUID-shaped_RFC_4122,_version_5_--_NOT_version_4,_confirming_the_documented_deviation_from_U-8's_literal_text_rather_than_silently_matching_it
--- PASS: TestDeterministicRecordID_U8CoverageGap (0.00s)
    ... (3 subtests) PASS
PASS
ok  	employeeprofilerecord/chaincode	1.009s
```
Full-suite regression check after adding it: `go test ./...` in the same module → `ok
employeeprofilerecord/chaincode 0.421s` (24 top-level test functions total — 23 pre-existing + the
1 new one — confirmed by `grep -c '^func Test' chaincode/*_test.go`, not just the PASS line count;
an earlier draft of this report miscounted this as 25 pre-existing/26 total, corrected here after
independent re-verification).

## Stale / mismatched premises found (test-strategy.md vs. the ratified/built design)

1. **U-2 is stale against ADR-0021.** U-2's text ("`HMAC-SHA256(pseudonymKey, tenantId‖employeeInternalId)`
   ... not recoverable from its outputs without `pseudonymKey`") describes the two-tier
   `pseudonymKey → employeeKey_i` hierarchy from ADR-0011/ADR-0019. **ADR-0021** (Accepted,
   2026-08-06 — the *same date* `test-strategy.md`'s header claims re-derivation against) explicitly
   **removes `pseudonymKey` from the design entirely**: `employeeKey_i` is now generated once per
   employee by a CSPRNG, independently random, with "no master secret that can regenerate it"
   (ADR-0021 Decision). Grep confirms `pseudonymKey` does not appear in any gateway-client/keystore
   source file (`grep -rn "pseudonymKey" write-path-integration/gateway-client/*.go
   write-path-integration/keystore/*.go` → no matches outside comments explaining its *absence*, e.g.
   `digestbuilder.go:16` — "there is no pseudonymKey/master key anywhere in this package"). **No test
   can or should exist for the HMAC-from-pseudonymKey derivation U-2 literally describes, because
   that code path was deliberately deleted, not merely untested.** This is exactly the kind of drift
   this item's instructions asked to flag rather than silently reinterpret.

2. **U-3's "regression guard" framing is stale for the same reason.** U-3's assertion that
   `EmployeeID`/`UpdatedBy` are "computed from `employeeKey_i`, never directly from `pseudonymKey`"
   made sense as a regression guard against the 2026-08-02→03 defect *while the hierarchy still
   existed*. Post-ADR-0021, there is no `pseudonymKey` anywhere in the codebase to regress into, so
   the guard's stated rationale is moot even though the tests that exist
   (`TestComputeEmployeeID_DeterministicAndKeyed` etc.) still correctly verify the *current*
   `employeeKey_i → EmployeeID/UpdatedBy` HMAC construction. The tests are fine; the spec's framing of
   why they matter needs updating.

3. **U-5 names a different access-control mechanism than what was built.** U-5 cites "CID
   `GetAttributeValue`(role/tenant)" and ADR-0005/D9 — i.e., **attribute-based** access control using
   enrolled X.509 attributes (`hris.role`, `hris.company`) per ADR-0005's Decision ("HRIS business
   role and company scope are carried as enrolled X.509 attributes ... Chaincode reads these via the
   CID API (`GetMSPID`, `GetID`, `GetAttributeValue`, `AssertAttributeValue`)"). The **actual shipped
   chaincode** (`identity.go`) uses only `cid.GetMSPID`/`cid.GetID` and a **hardcoded per-org
   allow-list** (`authorizeSubmit`/`authorizeEvaluate`, `isOrgClientMSP`) — coarse **org-level MSP
   membership**, which is exactly ADR-0005's rejected **Option B** ("Coarse per-org identity only ...
   cannot enforce role-level ABAC"). `grep -rn "GetAttributeValue"
   fabric-network/chaincode/employeeprofilerecord/chaincode/*.go` (excluding `vendor/`) returns **zero
   matches** — no role/tenant attribute is ever read anywhere in this chaincode. The tests that exist
   are real and pass, and they correctly verify the org-level mechanism that WAS built; they do not
   verify, and cannot verify, the attribute-based mechanism U-5's text names, because it was not
   built. This is a design-vs-spec gap for `fabric-architect`/`security-architect` to reconcile
   (either ADR-0005 needs a superseding ADR accepting Option B for this prototype, or role/tenant
   attribute ABAC is still outstanding implementation work) — not something QA-1 can or should paper
   over.

4. **U-8's "fresh UUID v4" does not match the shipped `deterministicRecordID`.** Covered in detail in
   the U-8 row and the new-test section above: the implementation deliberately produces a
   deterministic, txID-derived, version-5-shaped identifier, not a random version-4 UUID, because
   Fabric endorsement determinism requires it. This is **already self-flagged in the source**
   (`record_profile_section.go:15` — "`[FLAGGED DEVIATION — confirm before treating as final]`"), so
   this audit is corroborating an already-known, already-disclosed issue, not discovering a new one —
   but no test previously locked down the *actual* behavior, which is now closed (see above).

None of these four are reported as "coverage exists" for the literal U-# wording where it doesn't;
each is reported as coverage of the *actual* behavior, with the wording mismatch called out
separately, per this item's explicit instruction not to silently reinterpret a stale assertion as if
it were still accurate.

## What is genuinely NOT covered (no real test exists, not claimed otherwise)

- **U-1's "salt ... never returned/logged"** — the "never returned" half is structurally guaranteed
  by `ComputeDataHash`'s signature (`(string, error)`, no salt in the return), not something a test
  needs to exercise. The "never logged" half has **no test** anywhere, but I also found **no log call
  site** in the salt's code path (`gateway-client/digestbuilder.go`, `keystore/*.go`) that could leak
  it — `grep -n "log\."` across those files' non-test `.go` files returns no matches. I did not add a
  test for this: there is nothing to regress against (no logging statements exist to accidentally
  start including the salt), so a test asserting "logs don't contain X" would be exercising an absent
  code path rather than a real one. Flagged here as an honest "not tested, and there is currently
  nothing to test" rather than silently counted as covered.
- **U-7's "different section starts its own independent chain," narrated as one assertion** — the
  property holds (see U-7 row) but is assembled from several tests' side effects, not stated in one
  place. Not treated as a gap worth a new test under this item's "genuine, small gap" bar.

## Confidentiality check

Per this repo's standing confidentiality register, the two forbidden real-company/product terms are
intentionally **not spelled out literally in this file** (this file itself is a scan target, so
writing the terms here would defeat the point of the scan). The check run was a case-insensitive
recursive grep for those two terms (the same pair named in the register) against every file this
audit created or edited — this file and the new chaincode test file — and it returned **zero
matches** on both. The exact command and its (empty) output are reported through this task's
required structured-output channel, not reproduced here.

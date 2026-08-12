# Reconciliation: PRD (prd-fabric-hris-2026-08-11) vs. ratified fabric-hris design

Verified against: `fabric-network/chaincode/employeeprofilerecord/chaincode/asset.go`,
`agent-suite/05-adr/ADR-0013-channel-per-tenant.md`, `CLAUDE.md`.

## Facts confirmed accurate (no gap)

1. **12-field schema claim** — `EmployeeProfileRecord` in `asset.go` (lines 76-89) has exactly 12
   JSON fields (`canonicalizationVersion, dataHash, employeeID, hashAlgo, ipfsCIDs, prevHash,
   profileSection, recordID, tenantID, timestamp, updatedBy, version`). The struct's own comment
   (lines 63-69) explicitly says adding a field "re-opens the STRIDE T9 regression risk... Do not
   add a field without routing it through architect/security-architect first." PRD §2 and the
   `[RISK]` item in §13 both state this correctly and defer the mapping decision downstream — accurate.
2. **ADR-0013 channel-per-tenant** — `ADR-0013-channel-per-tenant.md` exists, Status: **Accepted**,
   supersedes ADR-0004. PRD's "already ratified (ADR-0013)" (§2, G3) is accurate. PRD Non-Goals
   (§4) correctly notes `tenant02` is deliberately abandoned per `tenantprovision`'s danger warning
   in CLAUDE.md, and does not contradict this — consistent.
3. **JCS+HMAC-SHA256, Domain C (SaltStore+EmployeeKeyStore), KEY_EMPLOYEE** — all three names match
   CLAUDE.md's write-path-integration architecture section verbatim: `gateway-client` implements
   the JCS+HMAC-SHA256 digest; `keystore` implements `SaltStore`+`EmployeeKeyStore` for Domain C
   (per-record salt/`employeeKey_i`); `DocumentKeyStore` uses `KEY_EMPLOYEE` for IPFS document
   encryption (Domain B'). PRD §6 and §14 use these names correctly.
4. **No typed error crosses the Gateway RPC boundary** — CLAUDE.md states this explicitly ("every
   cross-process chaincode business error has no typed representation on the client side... string-
   matching `err.Error()`"). PRD §11 table restates this accurately as an "established, deliberate
   limitation, not new to this PRD."

## Gaps / imprecisions found

1. **PRD's own header contradicts the confidentiality register it invokes.** The PRD front-matter
   (line 7) says `register: 'real-names (research substrate...)'` and the file lives under
   `_bmad-output/planning-artifacts/prds/`, which CLAUDE.md's rule 1 does list as part of the
   real-names research substrate — so the register choice itself is fine. However, the PRD body
   freely names real internal file paths and services (`BaseFabricBridgeService`,
   `MyInfoBlockchainHistoryService`, `TransactionHistory.vue`, `EmploymentUpdateWebhookService`,
   `PayrollComponentController`, `InformalEducationController`, `/Users/chan/Downloads/employee-
   profile-api-documentation.md`) — none of which appear anywhere in `agent-suite/` or existing
   `_bmad-output/` grounding files today. CLAUDE.md's real-names exception is scoped to "~25 files
   under `agent-suite/`... and all of `_bmad-output/planning-artifacts/`" — so this PRD is
   technically inside the allowed zone, but it is the *first* document to name Talenta-specific
   controller/service internals this concretely, meaningfully growing what CLAUDE.md calls a
   "grows as new ADRs close real-integration gaps" register. No `G-#` grounding-gap entry exists
   yet acknowledging this PRD as a new real-integration-naming source, unlike the disclosure
   CLAUDE.md requires (rule 1: "Only `agent-suite/context/REAL-INTEGRATION-TRIGGER-FLOW.md`
   discloses this at its own top; the others not doing so is a documentation gap"). **Recommend**
   adding a `G-#` row in `agent-suite/11-execution/grounding-gaps.md` and/or a top-of-file
   disclosure banner, consistent with how the one existing real-specifics-exception file is
   handled — otherwise this PRD repeats the "documentation gap" CLAUDE.md already flags for older
   files, at a new location.

2. **PRD asserts fields "extended/mapped to real fields" without confronting the ratified 12-field
   ceiling's practical consequence.** PRD §2 table row "On-chain data model" says the schema will
   be "Extended/mapped to the 5 domains' real fields" — but the schema is fixed at exactly 12
   generic, non-domain-specific fields (no per-domain field slots; `dataHash` is a single opaque
   digest per section, not per-field). "Extended" is the wrong verb: nothing in `asset.go` supports
   adding fields without an ADR/architect-routed change, and the actual mechanism the PRD's own
   FR-2 describes (digest the "post-write field set" into one `dataHash`) requires no schema
   extension at all — it fits the existing 12 fields with zero changes, using `profileSection` (an
   existing enum of exactly the 5 domains already, per `asset.go` lines 15-26/30-36) as the
   discriminator. The PRD's Overview (§2) and the `[RISK]` note in §13 are in tension: §13 correctly
   flags schema change as an architect decision, but §2's table cell reads as if mapping requires
   schema extension when the existing design likely already accommodates it via `ProfileSection`
   + `dataHash` without touching the struct. This is a clarity gap rather than a factual error, but
   worth flagging since it could send the downstream `fabric-engineer` agent looking for a schema
   change that examination of `asset.go` suggests is unnecessary.

3. **PRD implies chaincode-level support for the 5-domain model already exists, which is true but
   uncredited.** `asset.go` already defines `ProfileSection` as exactly `PERSONAL, EMPLOYMENT,
   EDUCATION, ADDITIONAL, PAYROLL` (lines 20-26) — a near-exact match to the PRD's 5 domains
   (Personal, Employment, Education & Experience, Additional Info, Payroll). The PRD's Current-
   State row ("generically designed, not yet wired to any real HRIS field set") undersells this:
   the domain *taxonomy* is already ratified and already matches the PRD's 5 domains one-for-one
   (only "Education & Experience" vs. chaincode's "EDUCATION" differs cosmetically). This is a
   positive finding for the PRD's feasibility, not a defect in the PRD, but the PRD does not cite
   this existing alignment anywhere (§2, §6, §12 Phase 1-2 discuss field mapping but never mention
   that the 5-way section split is already the chaincode's structuring dimension) — an
   **incomplete-but-not-wrong** gap: the PRD should credit `ProfileSection`'s existing 5-value enum
   explicitly, since it materially de-risks Phase 1/2 rollout (§12) and strengthens G3's "no new
   Fabric design" claim.

4. **No contradiction found on tenant02/ADR-0013, error-boundary, or key-domain naming** — checked
   explicitly per the task's concern list and all held up; no gap to report here beyond noting the
   verification was performed (see "Facts confirmed accurate" above).

## Summary

4 items examined as gaps; items 1-3 are real (documentation-completeness / precision) gaps, item 4
is a confirmation of no contradiction. No factual claim in the PRD about the ratified fabric-hris
design was found to be **wrong** — the two substantive gaps are (a) an undisclosed new real-name
grounding source not yet logged as a `G-#`, and (b) imprecise language around what "extending" the
12-field schema actually requires, given `ProfileSection`'s existing 5-value enum already matches
the PRD's domain split almost exactly.

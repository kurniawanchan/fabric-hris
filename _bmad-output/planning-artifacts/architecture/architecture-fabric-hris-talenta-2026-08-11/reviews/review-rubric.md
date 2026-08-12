# Architecture Spine Review — Talenta HRIS × Hyperledger Fabric Write-Path Integration

**Subject:** `ARCHITECTURE-SPINE.md` (architecture-fabric-hris-talenta-2026-08-11)
**Source PRD:** `_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-11/prd.md`
**Reviewed:** 2026-08-12

## Overall verdict: CONDITIONAL PASS

The spine correctly diagnoses the real seam (Talenta has no trigger; the bridge already has working
routes) and its six ADs are each individually enforceable and grounded in real, verified code. It
ratifies the brownfield codebase accurately (one line-citation slip aside) and its Stack section
names real, current versions. However, two of its own **bound** FRs (FR-10, FR-11) are pushed to
Deferred with zero interim invariant, and AD-6 states a rule that does not actually deliver the
idempotency guarantee it claims to protect (FR-5) — the dedup mechanism itself is left open. Both
are exactly the kind of gap that lets two independently-built units diverge, which is the spine's
core job to prevent. Not a rewrite; a follow-up pass to tighten AD-6 and give FR-10/FR-11 at least a
boundary-level invariant would close it out.

## Verdict per checklist item

1. **Fixes the real divergence points for the level below, misses none.**
   Mostly yes. AD-1–AD-5 cover the real seams found by cross-checking the PRD and code: retry
   authority (FR-1a/12/13), single write path (FR-2), hash-chain key shape (FR-3), the
   `ADDITIONAL`/Family route relabeling (FR-4, verified against `writepaths.go`), and operation-type
   as caller metadata (FR-4). **Miss:** FR-10 (verification) and FR-11 (reconciliation) are both in
   the spine's own `binds` list but get no AD at all — see Findings F1.

2. **Every AD's Rule is enforceable and actually prevents its stated divergence.**
   AD-1 through AD-5: yes — each names a concrete file/behavior constraint a reviewer could check
   by inspection (no queue in the bridge, no second caller of `SubmitRecordProfileSection`, no
   operation-type branching, etc.). **AD-6 is the exception:** its Rule only says the correlation ID
   "travels with the request" — it does not say where or how dedup is enforced, and that mechanism
   is explicitly kicked to Deferred. So the Rule as written does not prevent the divergence it
   claims to prevent (a retried job producing a duplicate chain entry) — see Findings F2.

3. **Nothing under Deferred could let two independently-built units diverge.**
   Fails on two of the four Deferred items:
   - "Idempotency dedup mechanism" (AD-6) — if the Talenta-job author assumes the bridge dedups and
     the bridge author assumes the job's correlation ID alone is enough, FR-5's idempotency
     guarantee silently doesn't exist. (F2)
   - "Reconciliation job's placement" (FR-11) — deferred with literally no boundary contract (not
     even which repo owns it), so a later contributor on either side could build it assuming the
     other side owns cadence/state. (F1)
   The other two Deferred items (persistent salt/key store, operator DLQ UI) are lower-risk: they're
   existing-component hardening / UI-only, and don't create two independent implementers of the
   same contract — acceptable to defer as-is, though the DLQ one would benefit from at least a data
   shape for what a dead-lettered entry must carry (F4, minor).

4. **Named tech is verified-current.**
   `Go 1.25.9` — confirmed against `integration-bridge/go.mod` (`go 1.25.9`) and independently a real,
   current Go security-patch release (April 2026). `hyperledger/fabric-gateway v1.12.0` — confirmed
   present verbatim in `integration-bridge/go.mod` as `// indirect`; web search could only confirm
   v1.10.0 as a known release and did not surface v1.12.0, but the repo's own `go.mod` is the more
   authoritative source here per checklist item 5, and the spine matches it exactly. No contradiction
   found. PHP/Yii2 for `talenta-core` is asserted as "existing app framework, no new language" — not
   independently checkable from this repo (talenta-core is not vendored here), consistent with the
   confidentiality register.

5. **Ratifies rather than contradicts the brownfield codebase.**
   Spot-checked and confirmed:
   - `write-path-integration/writepaths/writepaths.go` — `ApproveFamilyDataChange` exists exactly as
     described (SaveSection under `"ADDITIONAL"`, doc comment says family/dependents) at lines
     286–293, matching AD-4's claim (cited as 285–293 — off by one line, immaterial).
   - `integration-bridge/cmd/integrationbridge/main.go` — the five `POST /v1/profile-sections/{DOMAIN}`
     routes exist exactly as the Structural Seed and Consistency Conventions describe.
   - `integration-bridge/internal/pipeline/classify.go` — the auth/validation/timeout/partial-failure/
     success vocabulary exists exactly as referenced.
   - **One inaccurate citation:** AD-5 cites `validate.go:44` for the "`newValue` must be a JSON
     object" check; line 44 is the `validateRequest` function signature — the actual check is at
     line 61. Doesn't change the Rule's correctness, but would send an implementer to the wrong line.
     (F3, minor.)

6. **If a spec/PRD drove it, it covers that spec's capabilities.**
   Partial. Of the 11 bound FRs, 8 (FR-1, FR-1a, FR-2, FR-3, FR-4, FR-5*, FR-9, FR-12, FR-13*) get a
   concrete AD or Consistency Convention. FR-5's idempotency is only partially covered (F2). FR-13's
   "visible to operators, re-driveable" is covered only as "the Talenta job owns it" — ownership, not
   the contract itself (F4). FR-10 and FR-11 get none (F1). NFRs (latency, throughput) are correctly
   out of scope since they're not in `binds`.

7. **Dimension-silence check (scope frontmatter).**
   The spine's `scope` field explicitly excludes Fabric network topology, chaincode internals beyond
   the ProfileSection/key mapping, and the IPFS document path — correctly, and it stays silent on
   exactly those and only those. It does **not** explicitly exclude deployment/rollout or
   observability for the new `AnchorTriggerJob` / bridge changes, and says nothing about either — a
   silent dimension not covered by the scope exclusion (F5, minor; may be acceptable at "feature"
   altitude but isn't declared as out-of-scope, so it reads as an oversight rather than a decision).

## Findings (prioritized)

| # | Severity | Finding | Recommendation |
| --- | --- | --- | --- |
| F1 | **Major** | FR-10 (verification) and FR-11 (reconciliation) are in `binds` but the Capability→Architecture Map shows "—" and Deferred gives no boundary contract (not even which repo/component owns cadence or state). Two future implementers could build incompatible assumptions. | Either drop FR-10/FR-11 from `binds` (defer them to a future spine explicitly), or add a minimal AD fixing at least: which side owns the reconciliation job, and what "independent change-detection source" (DB timestamp) means as a cross-boundary contract. |
| F2 | **Major** | AD-6's Rule ("correlation ID travels with the request") does not itself deliver FR-5's idempotency guarantee — the actual dedup enforcement point is Deferred. As written, the AD's "Prevents" claim is not backed by its "Rule." | Either narrow AD-6's "Prevents" claim to "loses correlation trail across retries" only (drop the duplicate-entry claim), or pick a default dedup owner now (e.g., "the bridge's operational store enforces first-write-wins on correlation ID unless a future ADR says otherwise") so FR-5 has *some* enforced guarantee today. |
| F3 | Minor | AD-4 cites `internal/pipeline/validate.go:44` for the "newValue must be a JSON object" check; actual line is 61 (44 is the function signature). | Fix the citation to line 61 (or a line range) before this spine is used as an implementation reference. |
| F4 | Minor | FR-13 ("dead-lettered job visible to operators, re-driveable") is covered only by ownership ("lives in the Talenta job"), not by a data contract a future admin surface could rely on. | Add one line to AD-1 or Deferred specifying the minimum fields a dead-lettered entry must carry (e.g., correlation ID, last error, retry count, original payload) so the future DLQ UI and the job class don't diverge on shape. |
| F5 | Minor | Deployment/rollout and observability for the new Talenta job and the changed bridge routes are neither designed nor explicitly scoped out — a silent dimension rather than a declared exclusion. | Add a one-line scope exclusion (e.g., "does not govern deployment or observability for the new job — assumed to follow talenta-core's existing job-deployment conventions") or add a minimal AD if it matters at this altitude. |

**Counts:** 2 Major, 3 Minor, 0 Critical, 0 Info-only.

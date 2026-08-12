---
title: Adversarial Security/Privacy/Integrity Review — Talenta × Fabric PRD
status: draft
reviewed: '2026-08-11'
target: prd-fabric-hris-2026-08-11/prd.md
---

# Adversarial Review: Talenta HRIS × Hyperledger Fabric Integration PRD

## Verdict

The "detective, not preventive" model is coherent and honestly labeled as a trade-off, but the PRD
does not follow that trade-off to its actual conclusion for the threat actor it matters most
against: a privileged insider or anyone with direct DB access. For that actor, the anchoring
pipeline described (app-triggered webhook → queue → job) is not adversarially independent of the
write path it's supposed to police, so the "any out-of-band change is provably detectable" claim
in §1/§9/AC-4 is **not backed by the mechanism described** for the specific case the PRD's own
opening paragraph names as the motivating threat ("a privileged insider... leaves no tamper-evident
trail"). Combined with three pre-existing, unfixed Talenta auth gaps that the PRD itself says
"weaken the... assumption this whole design rests on," this reads as launch-blocking for the
Payroll domain specifically, not a footnote-level residual risk.

## Findings

### F1 — [CRITICAL] The "detective" guarantee assumes the attacker respects the trigger it doesn't gate

**Claim under test:** §1 "Any change made outside the application is provably detectable"; AC-4
"Employee A's payroll data is modified directly in the DB, bypassing Talenta's app entirely... a
hash-chain mismatch is reported."

**Attack:** AC-4's own scenario — direct DB write — is exactly the case where FR-10 (on-demand
verification) is the *only* backstop, because FR-1's enqueue trigger (`EmploymentUpdateWebhookService
::publish()`) lives in the PHP application layer, not in the database. A direct DB write, by
definition, never executes that PHP code path, so no anchoring job is ever enqueued for it — this
isn't a bug to fix, it's structurally what "bypassing the app" means. The PRD is correct that this
*specific* tamper is eventually caught, but only by FR-10 being run on-demand, or by FR-11's daily
reconciliation diffing Talenta's write log against Fabric's anchor log. Two failure modes the PRD
doesn't address:

1. **Same-privilege pipeline suppression.** Anyone with DB write access sufficient to tamper
   payroll rows directly, in most real deployments, also has access to the job queue table/backend
   (Redis/DB-backed Yii2 queue), the reconciliation job's own state (last-checked watermark,
   diff source), or can simply also update whatever log/table FR-11 diffs against. The PRD never
   states that the reconciliation job's source-of-truth (the "Talenta write log" in §10) is
   independent of the same DB the attacker already has write access to. If reconciliation diffs
   against another table in the *same* database the attacker just tampered, the attacker can erase
   their own trace from that log in the same transaction — nothing chain-based stops this, because
   the mismatch depends on Talenta's own write log being trustworthy, which is precisely the
   assumption being attacked.
2. **No independent enqueue signal.** There's exactly one code path (`publish()`) that can produce
   an anchor. FR-1 already documents 4 of 33 actions where this call is simply *missing* today —
   proof this single-path model is fragile even for the honest, non-adversarial case. For the
   adversarial case, an attacker with DB access doesn't even need to suppress a queue message: they
   just never trigger the one and only code path that would have created one. Detection then relies
   100% on FR-10/FR-11 comparing DB state to on-chain state — which the PRD does describe — but the
   PRD frames this as "detective, not preventive" (implying tampering is always eventually seen)
   without stating the corollary: it is **not detective at all** for a domain/employee pair that
   was *never anchored in the first place*, e.g. a brand-new record inserted directly via DB access
   with no prior anchor to chain-mismatch against. FR-10 compares current-state digest to "the
   latest on-chain anchor" — if there is no anchor (record created out-of-band, never touched the
   app), there is nothing to mismatch against, and nothing in FR-10/FR-11 as specified would flag an
   entirely-fabricated record with no baseline. This is a gap in the mechanism, not just lag.

**Why this is Critical, not Medium:** it directly undermines the PRD's stated raison d'être (§1,
first paragraph) for the exact attacker profile named there ("a privileged insider... leaves no
tamper-evident trail") on the domain the PRD itself calls highest-sensitivity (Payroll, §6, and the
Phase-1 pilot domain, §12). A design whose core promise fails against its own headline threat model
for its own pilot domain should block a "production/launch grade" claim.

**What's missing from the PRD, concretely:** (a) a statement of what FR-11's reconciliation
actually diffs against and why that source is attacker-independent from the DB itself (e.g. an
append-only audit log outside the attacker's writable scope, WAL/binlog shipped elsewhere,
etc.) — none is named; (b) a story for records with *no* prior anchor (net-new out-of-band rows);
(c) acknowledgment that queue/job suppression by a DB-privileged actor is in-scope-but-unaddressed
rather than silently assumed away.

### F2 — [HIGH] Payroll gets the same 24h detection lag as everything else, despite being flagged as highest-sensitivity

**Claim under test:** NFR-7 "reconciliation job... SHALL run daily"; §12 recommends Payroll as the
Phase-1 pilot "as the highest-sensitivity domain."

The PRD explicitly elevates Payroll's sensitivity (§6: "Highest-sensitivity domain") and yet applies
a uniform, undifferentiated 24-hour worst-case detection lag (FR-11/NFR-7) across all five domains.
For a bank-account or salary tamper, a day of undetected exposure is a materially different risk
than the same lag on, say, an Education & Experience `certification` field. §13 lists NFR-7's
cadence as `[OPEN]` (unconfirmed), which is honest, but the PRD never poses the question of
*domain-differentiated* cadence at all — it frames NFR-7 as a single global number to be confirmed,
not as a question of whether Payroll needs tighter reconciliation (e.g. hourly) than the other four
domains. Given the PRD calls itself "production/launch grade" and specifically sequences Payroll
first (§12) "even though all 5 are in scope," this is a real gap, not just an unresolved constant.
On-demand verification (FR-10) partially mitigates this if actually run proactively for Payroll, but
the PRD assigns no owner, cadence, or trigger for *when* FR-10 gets invoked outside of ad hoc
operator action — it's described only as an available action, never a scheduled/proactive one for
high-sensitivity domains.

**Severity: High** (not Critical) — this doesn't break the mechanism, it just leaves an
acknowledged-sensitive gap unsized and untargeted for the one domain the PRD itself singles out.

### F3 — [HIGH] Pre-existing Talenta auth gaps are treated as a footnote risk, but at least one directly falsifies "authorized write"

**Claim under test:** §9's premise that Fabric anchors changes "authorized or not," with detection
covering only *out-of-band* (non-app) changes, on the assumption that in-app changes are gated by
Talenta's existing authorization.

§9 discloses three gaps and is explicit that they "weaken the 'authorized write' assumption this
whole design rests on" — that phrasing is the PRD grading its own finding correctly, then filing it
under §13 as a `[RISK]` "tracked as a linked defect," not gated on before launch. Look at the third
gap specifically: `POST /my-info/update-identity-address` "does not check `canRequestChangeData` at
all — it always writes directly, bypassing the approval flow." This is not a hypothetical
out-of-band bypass requiring elevated DB access — it is an *in-app*, ordinary-user-privilege path
that already produces an "authorized" (i.e., app-mediated, correctly-anchored) write despite not
actually being approved per the domain's own intended control. Because this write goes through the
normal app path, it *will* be anchored (correctly, per FR-1/FR-2/FR-3) — meaning the hash chain
will faithfully record an unapproved change as if it were a legitimate, undisputed history entry.
The anchoring feature doesn't just fail to catch this; it actively launders it into "verified"
history that AC-4/FR-10 will never flag as compromised, because from Fabric's perspective nothing
about this write looks out-of-band. This is worse than a silent gap — it's the system's core value
proposition (an "undeniable... verified change history," §1) actively certifying a change that
bypassed an intended approval gate. This should block launch for the Personal domain specifically,
or at minimum be re-classified from `[RISK]` (tracked, not blocking) to a launch gate, since it
falsifies the "authorized" half of "any change — authorized or not" in §9's own framing.

**Severity: High** (borderline Critical) — downgraded slightly from Critical only because it's a
pre-existing Talenta defect the PRD is transparent about, rather than a defect newly introduced by
the Fabric integration itself; but its interaction with this PRD's specific "verified history"
marketing claim (§1) is a new, integration-specific harm this PRD introduces by giving that
unapproved write a permanent, cryptographically-attested-looking record.

### F4 — [MEDIUM] Key-management / re-derivation attack surface on the anchoring job is unaddressed, not merely under-specified

**Claim under test:** implicit throughout §6/§14/FR-2 — that salted HMAC digests can't be forged
without the salt/key material, and that this makes the chain trustworthy.

The PRD leans entirely on `fabric-hris`'s existing `SaltStore`/`EmployeeKeyStore` (Domain C) via
"the existing gateway-client digest construction" (FR-2) without ever discussing where the new
"Blockchain Integration Service" (§5) — a *new* service identity/process this PRD introduces —
obtains its key material, how it's scoped, or whether it's the same runtime as the anchoring job
that also has queue access. A privileged attacker who can read that job's runtime memory or config
(the same class of attacker F1 already establishes has DB-level access, plausibly adjacent
infrastructure access too) could, if they can also reach the salt/key material the job must hold to
compute digests, recompute valid digests for fabricated history entries — the digest scheme's
unforgeability is only as strong as the isolation between "can tamper with data" and "can read the
anchoring job's key access," and the PRD asserts zero-PII-on-chain (verified, NFR-5) but never
asserts or designs for key-material isolation between the write-tampering attacker and the anchoring
job. It cites `fabric-hris` CLAUDE.md's own disclosed limitation that chaincode-side MSP checks stop
a *wrong Fabric identity* from forging an anchor for a company/employee it doesn't own (§9) — but
that's orthogonal: it protects cross-tenant/cross-company forgery, not a compromised *correct*
identity forging plausible-looking history for records it's legitimately scoped to. This is silently
assumed safe rather than addressed.

**Severity: Medium** — real gap, but it inherits from `fabric-hris`'s existing (separately reviewed)
key-domain design rather than being newly introduced by this PRD; flagging for cross-reference to
whatever ADR governs Domain C's operational deployment/isolation, since this PRD doesn't reproduce
or extend that analysis for the new service identity it creates.

### F5 — [MEDIUM] "Right-to-delete... known, disclosed limitation" reads as more resolved than it is

**Claim under test:** §14 "This PRD does not claim deletion of anchored digests is possible... known,
disclosed limitation, not a compliance claim."

This is honestly framed as a non-claim, which is good — but it elides a live operational
consequence: because the hash chain is per-`(employee id, domain)` and FR-3 chains each anchor to
"the previous digest for that pair," a genuinely-deleted record (e.g. GDPR/PDP erasure request
honored in Talenta's DB) leaves a *dangling* chain — the latest known digest now corresponds to
data that provably no longer exists to recompute against. FR-10's verification action would then
either falsely report "compromised/inconsistent" (no current DB state to hash and compare) or
silently skip verification for deleted records — the PRD specifies neither behavior. This isn't
just a philosophical "chains are hard to erase" caveat; it's an unspecified failure mode in FR-10's
own logic for a case §14 acknowledges will happen (a "delete" in Talenta's DB). Should be resolved
before Phase 3 (FR-10 build), not left implicit.

**Severity: Medium.**

## Summary Table

| # | Finding | Severity |
|---|---|---|
| F1 | Detective model has no independent-of-DB anchoring/reconciliation signal; net-new out-of-band records have no baseline to mismatch against | **Critical** |
| F2 | Uniform 24h reconciliation lag despite Payroll being singled out as highest-sensitivity and the Phase-1 pilot | High |
| F3 | Pre-existing `update-identity-address` gap causes an unapproved write to be anchored as "verified" history — laundering, not just an unfixed gap | High |
| F4 | No key-isolation analysis for the new "Blockchain Integration Service" identity against the same privileged attacker class assumed in F1 | Medium |
| F5 | FR-10's behavior against a legitimately-deleted record (dangling chain tail) is unspecified | Medium |

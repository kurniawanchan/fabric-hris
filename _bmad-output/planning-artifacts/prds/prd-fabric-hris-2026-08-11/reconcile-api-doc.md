# Reconciliation: PRD (prd-fabric-hris-2026-08-11) vs. source API doc

Source API doc: `/Users/chan/Downloads/employee-profile-api-documentation.md`
PRD reviewed: `/Users/chan/www/hyperledger/fabric-hris/_bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-11/prd.md`

Overall: the PRD's field extraction and the two auth gaps it does cite (§9) are accurate
quotes from the API doc — no fabricated endpoint or field name was found. The gaps below are
about omission and one dangling reference, not misquotation.

## Gaps

1. **Dangling reference to "the addendum" (FR-1, PRD line ~110-113).**
   FR-1 says the anchored-write trigger list is "the full endpoint list extracted in the
   addendum," but no `addendum.md` exists in this PRD's own directory
   (`prds/prd-fabric-hris-2026-08-11/`). An `addendum.md` only exists under the sibling,
   earlier `prds/prd-fabric-hris-2026-08-10/` folder — a different PRD version. If FR-1 means
   that older addendum, the PRD should say so explicitly and re-verify it still matches this
   version's field/endpoint set; as written it's an unresolved reference.

2. **Omits a third pre-existing auth gap the doc flags at the same severity as the two it does cite.**
   PRD §9 lists two "known pre-existing auth gaps": `InformalEducationController` skipping the
   login guard, and `/additional-info/index` not checking company ownership — both accurately
   quoted (API doc Appendix items #2 and #8). But the API doc's Appendix item #1 / §1.1 note is
   just as directly relevant to this PRD's core premise ("authorized write" integrity): *"`POST
   /my-info/update-identity-address`... does **not** check `canRequestChangeData` at all — it
   always writes directly, regardless of whether the employee would otherwise need to go through
   the approval flow."* This is a write-path approval bypass on the same Personal domain the PRD
   anchors (§6) — it belongs in the §9 gap list and §13 risk register alongside the other two but
   is currently absent.

3. **§6 Data Classification table only covers the "Basic info" sub-tab of the Personal domain,
   silently dropping Family and Emergency Contact — leaving an ID-number-class field unclassified.**
   The API doc's Personal domain (§1) explicitly has three sub-tabs: Basic info, Family, Emergency
   contact (doc lines 51-52, endpoint table lines 57-74). The PRD's Personal row in §6
   (`full_name, phone, email, birth_date, nik/citizen_id, passport, address`) reflects only Basic
   info. It omits the Family sub-tab's `no_ktp` field (doc line 147-150 —
   `GET /my-info/family-data` response) — a family member's national-ID number, the same
   sensitivity class as the employee's own `nik`/`citizen_id`/`passport` that §6 does flag as
   "highest-sensitivity... never on-chain in any form" — and omits Emergency Contact's
   `phone_number` (doc line 189-190). Since §6 is the basis for what gets salted-digest treatment
   per domain, an unclassified `no_ktp` is a real completeness gap, not just a documentation nit.

4. **FR-4's operation-type derivation assumes a server-enforced HTTP verb the doc says doesn't exist for most of these endpoints.**
   FR-4 (PRD line ~119-121) says each anchor carries an "operation type (CREATE/UPDATE/DELETE)...
   source endpoint." The API doc's own "Verb enforcement caveat" (doc lines 24-30) states plainly:
   *"Most controllers in this module... define **no `behaviors()`/`VerbFilter`** at all. The HTTP
   method shown for each endpoint is the **frontend convention**... not mechanically enforced by
   the server unless explicitly noted."* (Doc line 485-486 repeats this for Additional Info's
   GET/PUT specifically.) The PRD doesn't address how the anchoring job derives a reliable
   CREATE/UPDATE/DELETE operation type when the transport-level verb isn't a guaranteed signal —
   worth a design note (likely: derive from the webhook payload's own action/diff, not the HTTP verb).

## Not flagged as gaps (checked and found accurate)
- §6 Payroll fields (`new_salary, npwp, bpjstk, bank_account, bank_account_holder,
  payment_account[]`) match the doc's `/my-info/payroll-info` and `update-payroll` shapes.
- §6's `feature_hash_bank_account` reference matches doc line 618-619 verbatim in spirit.
- §9's `status_employee` masking-precedent claim matches doc's employment-data masking note
  (doc lines 261-263).
- §4 Non-Goals' exclusion of `PayrollComponentController`'s ~28 other actions and custom-field
  *definition* CRUD matches the doc's own out-of-scope note (doc lines 682-688, 540-544).
</content>

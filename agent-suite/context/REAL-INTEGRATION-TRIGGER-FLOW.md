# Real Integration Trigger Flow — talenta-core → Fabric

> **Register exception, deliberate and disclosed.** Every other artifact in this repo — including
> this directory's own `PHP-INTEGRATION.md` — stays in the generic register even where the
> underlying grounding is a real, internally-consulted repository (`PHP-INTEGRATION.md`'s own
> "Register note", citing grounding gap `G-29`). **This document is the one deliberate exception**,
> per an explicit user instruction (2026-08-07): it names `talenta-core`, real file paths, and real
> function names, because an integration guide with the names washed out would not be actionable.
> Treat this file as internal engineering reference, not something to fold into the generic-register
> deliverable set (`qa-tests/`, `deploy/`, the anonymized thesis-facing reports) — those stay
> generic. `PHP-INTEGRATION.md` is not rewritten or contradicted by this file; it is cross-referenced
> from it (see that file's own §5 note pointing here).
>
> **Verified against the live repo 2026-08-07** (not just the pre-existing grounding docs, which
> predate this check and — as detailed in §1 — turn out to have glossed over some real structural
> messiness). Every file:line citation below was read directly from `/Users/chan/www/talenta-core`.

## 0. What this document is for, and what it is not

This is the concrete answer to "where, exactly, would a call into
`write-path-integration/writepaths.Hooks` be inserted in the real application, and what would that
call actually look like." It does **not** decide grounding gap `G-10` (which process hosts the
Fabric Gateway client) — `ADR-0014` leaves that open on purpose, and this document does not close
it. It documents the real trigger points and the real, already-established calling-convention
precedent in this codebase, so that whoever does make the `G-10` decision has accurate ground truth
to decide against.

## 1. The five `writepaths.go` hooks do not map onto five equally-shaped real code paths

`write-path-integration/writepaths/writepaths.go`'s five functions (`UpdatePersonalData`,
`ApproveEmploymentTransfer`, `RecordEducationHistory`, `ApproveFamilyDataChange`,
`UpdatePayrollBankAccount`) are a clean **logical** model. The real code is messier — confirmed by
reading the actual commit sites, not assumed from the generic table in `PHP-INTEGRATION.md` §1:

| `writepaths.go` hook | Real trigger chain | Real commit line | Shares a commit function with... |
|---|---|---|---|
| `UpdatePersonalData` (PERSONAL) | `EmployeeController::actionApproveAllChangeData()` → `ChangeDataService::approveAllChangeData()` → `ChangeDataService::approveChangeData()` (`services/ChangeDataService.php:244-366`) → `InboxController::updateChangeData($id, $idUser)` | `controllers/InboxController.php:8219` (SSO branch) or `:8224` (non-SSO branch), both `$updateDataUser->save();` | **`UpdatePayrollBankAccount`** — see next row |
| `UpdatePayrollBankAccount` (PAYROLL) | Same `ChangeDataService::approveChangeData()` chain as PERSONAL — bank fields are just three more `case` branches (`DATA_TYPE_BANK_NAME`/`BANK_ACCOUNT`/`BANK_ACCOUNT_HOLDER`, constants `21`/`22`/`23` in `models/EmployeeDataRequestDetail.php:39-41`) inside the SAME `switch($row['data_type'])` in `InboxController::updateChangeData()` (`controllers/InboxController.php:8068-8080`) | **The identical `$updateDataUser->save();` call as PERSONAL** | `UpdatePersonalData` — **this is one real commit point, not two** |
| `ApproveFamilyDataChange` (ADDITIONAL) | `ChangeDataService::approveChangeData()` (family branch, line 308-310) → `ChangeDataService::applyChangeFamilyData()` (private, lines 394-397) → `BaseComponent::GetDataToApplyChangeFamilyData($userId, $familyDataRequestId)` | `components/BaseComponent.php:10232` (raw SQL `UPDATE` via `Yii::$app->db->createCommand(...)->execute()`) or `:10277`/`:10286` (`->save()` for new dependents) | none — its own mechanism, and notably **raw SQL, not ActiveRecord**, for the edit case |
| `ApproveEmploymentTransfer` (EMPLOYMENT) | `EmployeeTransferServices::processApproval()` (`services/EmployeeTransferServices.php:784-820`) → private `approveTransfer()` (lines 1412-1588) → `UserRepository::transfer()` | `repositories/UserRepository.php:1678`, `$user->save();` | none — **entirely separate from `ChangeDataService`**, never touches it |
| `RecordEducationHistory` (EDUCATION) | `FormalEducationController::actionSave()` (`controllers/api/web/FormalEducationController.php:99-118`) → `FormalEducationServices::saveFormalEducation()` | `services/FormalEducationServices.php:119`, `$this->formalEducationRepository->save($params)` | none — **entirely separate, and not even approval-gated** (no `EmployeeDataRequest` envelope at all — a direct write) |

**Practical consequence**: a real integration needs at minimum **four** distinct hook insertion
points, not five, since PERSONAL and PAYROLL genuinely share one — but PAYROLL is also
**incompletely covered by that one hook** (§4).

## 2. Actor/approver identity is a global read at 3 of 4 real commit sites, not a passed parameter

`PHP-INTEGRATION.md`'s generic table says each path has the approver identity "already in hand."
True in the sense that `Yii::$app->user->identity` is reliably populated for the whole request
(Yii2's standard session-identity resolution, `config/web.php:163-166`,
`CustomLoginController::beforeAction()`), but **not literally true** at the exact save-point in 3 of
the 4 real commit functions:

- `InboxController::updateChangeData($id, $idUser)` — `public static`, **no `$actor`/identity
  parameter in scope at all** in the main write loop (only re-derived in a few unrelated
  sub-branches, e.g. line 8194's SSO check).
- `UserRepository::transfer(EmployeeTransferTransaction $employeeTransferTransaction)` — no actor
  parameter, even though the caller (`approveTransfer`) has `$actor` in scope one frame up.
- `BaseComponent::GetDataToApplyChangeFamilyData($userId, $familyDataRequestId)` — same gap.

Only `FormalEducationServices` carries an actor cleanly, via `BaseServices::__construct()`
(`services/BaseServices.php:23-28`): `$this->user = Yii::$app->user->identity ?? $user ?? null;`.

**Implication for the hook**: at the three gap sites, the inserted anchor call should read
`Yii::$app->user->identity->id` directly at the hook's own call site, not assume a variable is
already there — this is a small, mechanical fact worth getting right the first time rather than
discovering it mid-implementation.

## 3. The calling mechanism — confirmed precedent, not invented here

`ADR-0006` already established there is no first-class Fabric Gateway SDK for PHP. The real
codebase's own precedent for calling a synchronous external service from inside a PHP request is
**`services/ems/BaseEmsService.php`** — confirmed still active, wired to a real deployed service:

- Plain HTTP/JSON via `GuzzleHttp\Client` (`send()`, lines 23-58): `Yii::$app->params['ems-service-url'] . $path`, blocking request/response, JSON-decoded on return.
- Auth via a shared header scheme (`companyLevelHeaders()`, lines 98-105): `X-Api-Key` from
  `Yii::$app->params['ems-api-key']`, plus `X-Scope`/`X-Company-ID`.
- Config wiring: `config/params.php:319-321` (`env("EMS_SERVICE_URL")`, `env("EMS_API_KEY")`), live
  value in `.env:387`.

**Checked and ruled out**: a PHP gRPC client precedent. `composer.json` has `google/protobuf` but no
gRPC client library. The one `.proto` file in this repo
(`proto/notification-service/event.proto`) is used only to serialize a message to bytes for
**Kafka** publishing (`helpers/NotificationServiceHelper.php:45-47`,
`$dataNotif->serializeToString()` → `AttendanceKafkaService::sendKafkaNotification()`) — protobuf-
over-Kafka, not synchronous gRPC.

**Conclusion, not a decision (`G-10` stays open)**: the `BaseEmsService` pattern — a small Go HTTP
service, called synchronously via Guzzle with an API-key header, wrapping
`write-path-integration/writepaths.Hooks` — is the natural fit for this codebase's own existing
idiom and satisfies `ADR-0014`'s "synchronous, in-band, same request" requirement exactly the way
the existing EMS calls already do. The alternative already in this codebase (protobuf-over-Kafka)
is async and would reproduce the shape `ADR-0014` explicitly rejected. Whoever makes the `G-10`
decision should treat `BaseEmsService` as the concrete reference implementation to model against,
not invent a new calling convention.

## 4. A real, newly-found gap: PAYROLL has write surfaces this hook would not cover

Beyond the approval-flow path in §1, bank-account fields have at least two more independent write
surfaces that never touch `ChangeDataService` and therefore would be **invisible** to a hook placed
only at `InboxController::updateChangeData()`:

- `MyInfoController::actionEditPayrollInfo()` (~line 2897; direct field assignment at
  `controllers/MyInfoController.php:3378-3381`) — an HR-admin direct-edit form, no approval request
  created at all.
- Bulk import (`controllers/EmployeeController.php:26805`), a cron job
  (`CronServices.php:449`), and a third-party HR integration importer
  (`services/integration/bizzy/BizzyEmployee.php:144,168`).

**This is a real, disclosed scope gap, not resolved here**: if PAYROLL's anchoring is meant to
cover every bank-account change (not just employee-initiated change requests), these three
surfaces need their own hooks too. Flagging for `architect`/`security-architect` — matches this
whole project's standing discipline of naming a gap rather than silently narrowing what "PAYROLL is
anchored" actually means.

## 5. `EmployeeInfoProducerService`'s payload — re-confirmed metadata-only, from the live field list

`services/kafka/EmployeeInfoProducerService.php` is still active, wired from `ChangeDataService`
(PERSONAL) and `EmployeeTransferServices` (EMPLOYMENT). Its `getMessage()` (lines 68-145) builds
exactly this field set: `user_id, company_id, id_employee, first_name, last_name, join_date, job_id,
job_name, branch_id, job_level_id, organization_id, employment_status, religion, sbu_list` (plus a
few optional name fields behind a feature toggle). **Zero fields for EDUCATION, ADDITIONAL, or
PAYROLL content anywhere in the payload.** This re-confirms, from the live code rather than from
`ADR-0006`'s original prose claim, exactly why this topic cannot be the anchor's data source for
three of the five sections.

## 6. Cross-refs

- Generic (register-compliant) version of this same seam: `PHP-INTEGRATION.md` §1/§4.
- The record shape being anchored: `BLOCKCHAIN-DATA-MODEL.md`.
- The in-band call path and Gateway client mechanics: `BLOCKCHAIN-INTEGRATION.md`.
- The actual Go-side hooks this document's calls would reach: `write-path-integration/writepaths/writepaths.go`.
- Grounding gap this document partially informs but does not close: `G-10` (Gateway-client host).

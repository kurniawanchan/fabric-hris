# GO-ERROR-HANDLING — EMS error model (stub)

> **Read first:** `GO-CONVENTIONS.md` §2 (the full error walkthrough). Language-agnostic guidance:
> `> See coding-standards-backend/references/error-handling.md`. This stub is a quick reference to the
> `pkg/errs` type and its HTTP mapping.

## The `errs.Error` type
EMS returns a custom `errs.Error` (impl `errCustom`), not bare `error`, from repo/service/handler.
It carries `Message`, wrapped `Err`, an `ErrorType`, and captured `File` + `Stack`
`[code: ems/pkg/errs/error.go]`. It implements `Unwrap()` so `errors.Is/As` interoperate
`[code: ems/pkg/errs/error.go]`.

## ErrorType == HTTP status
The type constants double as HTTP codes `[code: ems/pkg/errs/factory.go]`:
`400 BadRequest · 401 Unauthorized · 403 Forbidden · 404 NotFound · 409 Conflict ·
422 UnprocessableEntity · 500 InternalError`, plus internal kinds `Missing/Parse/Sql/File/Invalid`.

## Factories (pick by intent)
`errs.New`, `errs.Wrap` (defaults to 422), `errs.WrapWithErrorType`, `errs.WrapWithMessage`,
`errs.NewNotFoundError`, `errs.NewBadRequest`, `errs.NewConflictError`,
`errs.NewUnprocessableEntityError`, `errs.NewUnautorizedError` (sic), `errs.NewForbiddenError`,
`errs.PanicError` `[code: ems/pkg/errs/factory.go]`.

## Where errors are created / mapped
- **Repository:** wrap raw GORM errors — `errs.Wrap(result.Error)`; return `(nil, nil)` when empty is
  a valid state `[code: ems/internal/education/repository/repository.go]`.
- **Duplicate keys:** `errs.IsDuplicateKey(err)` detects both `gorm.ErrDuplicatedKey` and MySQL 1062
  `[code: ems/pkg/errs/factory.go]`.
- **Validation:** parse helpers on `*app.Context` append to `_errors`; caller checks `ctx.HasError()`
  and reads `ctx.GetFirstError()` `[code: ems/internal/base/app/context.go]`.
- **HTTP mapping:** the route guard's `guardError` switches on `err.GetErrorType()` to the right
  responder (404 mask must NOT go through the generic 500/Flock path)
  `[code: ems/app/api/router.go]`.

## Panics
Two safety nets: `RunAction`'s `defer recover()` returns a 500 and raises a Flock alert
`[code: ems/internal/base/handler/base.go]`; `api.RecoveryWithFlock` is the gin-level backstop
(handles broken-pipe specially) `[code: ems/app/api/middleware.go]`. Async workers recover per job
in `workerpool.runJob` (see `GO-CONCURRENCY.md`) `[code: ems/pkg/workerpool/pool.go]`.

## Production hygiene
`errCustom` keeps `File`/`Stack` for developers but `GetErrorDebugResponse` omits line numbers, and
`CheckStatus` deliberately prints no secrets `[code: ems/pkg/errs/error.go]`,
`[code: ems/internal/base/handler/check.go]`. Sensitive fields are masked in request logs (`GO-SECURITY.md`).

## Recommendation
Fabric gateway/SDK failures → `errs.WrapWithErrorType(err, errs.ErrorTypeInternalError)`; reserve 4xx
for caller-input problems. For anchoring, prefer idempotent retry over surfacing transient ledger
errors to the user: `> See coding-standards-backend/references/idempotency.md`.

## Cross-references
Convention deep-dive: `GO-CONVENTIONS.md` §2. HTTP surface: `GO-APIS.md`.

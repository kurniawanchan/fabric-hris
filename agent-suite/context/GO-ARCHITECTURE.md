# GO-ARCHITECTURE — employee-management-service (EMS) layering & request flow

> **Scope.** How the real Go service `employee-management-service` (EMS) is structured: its
> packages, the handler → service → repository layering, request lifecycle, and where the Fabric
> anchor hook would attach. Grounded in the actual repo at `/Users/chan/www/employee-management-service`.
> **Module:** `bitbucket.org/mid-kelola-indonesia/employee-management-service`, **Go 1.21**
> `[code: ems/go.mod]`. EMS is the strangler-fig Go extraction of the talenta-core (PHP/Yii2) HRIS,
> reading/writing the **same shared MySQL**; see the knowledge-graph Layer C.
>
> This is a context doc, not a rulebook. For language-agnostic backend principles this repo is
> measured against, read the skill: **`> See coding-standards-backend/references/boundaries.md`** and
> **`> See coding-standards-backend/references/service-boundaries.md`**. Conventions and testing get
> their own docs (`GO-CONVENTIONS.md`, `GO-TESTING.md`).

## 1. Top-level layout

```
main.go                 # thin entrypoint → cmd.Execute()
cmd/                    # cobra commands + composition root (boot.go wires everything)
app/
  appconf/              # Config struct + InitAppConfig() (env → typed config)
  api/                  # gin HTTP server: routers, middleware, route guards
internal/
  base/                 # cross-cutting: app.Context, BaseHTTPHandler, authz, ULMS, mail
  <domain>/             # ~140 feature packages (employee, company, education, resign, …)
  datasource/model      # gorm/gen generated structs  (DO NOT EDIT)
  datasource/models     # gorm/gen type-safe query DSL (DO NOT EDIT)
  mocks/                # shared generated mocks
pkg/                    # reusable libs: db, errs, pii, bankhasher, workerpool, dbresolver, …
external/               # outbound clients (ssoserv, billingserv, talentaserv, httpreq, …)
deploy-alicloud/        # Helm chart for Alibaba Cloud K8s
qa-tests/ · specs/      # security matrices · spec-kit feature specs
```

`main.go` calls into cobra (`cmd/root.go`); the `http serve` subcommand (`cmd/http.go`) runs
`initHTTP()` in `cmd/boot.go`. `Makefile` `run-http` target: `go run main.go http serve`
`[code: ems/Makefile]`; the built binary is `bin/ems http` `[code: ems/main.go]`,
`[code: ems/cmd/http.go]`.

## 2. Feature-package anatomy (the repeating unit)

Every `internal/<domain>/` package repeats the same sub-layer shape. `internal/education/` is a
minimal, representative example:

```
internal/education/
  domain/education.go            # request/response structs + params (the domain types)
  repository/irepository.go      # Repository interface  ← consumers depend on this
  repository/repository.go       # repo struct implementing it (GORM)
  repository/mocks/Repository.go # generated mock
```

Fuller packages (`internal/user/`, `internal/employee/`) add `service/` (`iservice.go` +
`service.go` + `mocks/`), `handler/`, and `presenter/` `[code: ems/internal/user/]`. The dependency
direction is always **handler → service → repository → datasource/model**, and each layer depends on
the *interface* of the layer beneath (`repository.Repository`, `service.Service`), never the struct
`[code: ems/internal/education/repository/irepository.go]`,
`[code: ems/internal/user/service/iservice.go]`.

## 3. The three layers

### Handler layer (`internal/<domain>/handler`, `internal/base/handler`)
Handlers are functions of type `HandlerFn func(ctx *app.Context) *server.Response`
`[code: ems/internal/base/handler/base.go]`. They parse/validate input from `*app.Context`, call a
service, and return a `*server.Response` built by `BaseHTTPHandler` responders (`AsJson`,
`AsJsonError`) `[code: ems/internal/base/handler/base.go]`. Handlers do **not** talk to GORM
directly. `BaseHTTPHandler` is the shared handler base carrying DB, config, validator, and the
injected services (`UserService`, `ULMSService`, `Guard`, …) `[code: ems/internal/base/handler/base.go]`.

### Service layer (`internal/<domain>/service`)
Business logic. A service is an interface (`iservice.go`) plus a private `service` struct holding
injected repository and sibling-service interfaces, built by `NewService(...)`
`[code: ems/internal/user/service/service.go]`. Methods take `context.Context` or `*app.Context`
and return `(T, errs.Error)` `[code: ems/internal/user/service/iservice.go]`. Services orchestrate
multiple repositories and cross-domain rules (e.g. RBAC role derivation in `setRuleFromRoleId`).

### Repository layer (`internal/<domain>/repository`)
Data access over GORM. The `repo` struct holds `db` (**replica**, reads), `masterDB` (**writes**),
a `base *pkg/db.MySQLClientRepository`, and a generated `GetQuery *models.Query`
`[code: ems/internal/education/repository/repository.go]`. Every query is context-scoped
(`r.db.WithContext(ctx)`), company-filtered (`Where("company_id = ?", …)`), and DB errors are
wrapped with `errs.Wrap(result.Error)` `[code: ems/internal/education/repository/repository.go]`.
Reads run against `r.db` (replica); creates/updates run against `r.masterDB`
`[code: ems/internal/education/repository/repository.go]`. See `GO-DATABASE.md`.

## 4. Cross-cutting `internal/base`

- **`base/app/context.go`** — `app.Context` embeds `*gin.Context` and carries the per-request
  identity: `scope`, `_identity (*user.User)`, `company`, validation `_errors`, `ssoID`, `APIReqID`,
  and `ignoreAccessRoleValidation`. Accessors: `GetAuthenticatedCompanyID()`, `GetScope()`,
  `GetIdentity()`, `HasError()/AppendError()` `[code: ems/internal/base/app/context.go]`.
- **`base/handler/base.go`** — `BaseHTTPHandler`, `RunAction` (the gin adapter), `Authentication`
  (gateway-header → identity), the ULMS logging hook, panic recovery.
- **`base/authz`** — the authorization surface: `Guard` (object-level BOLA defense, resolver
  `Registry`) and `employee_rbac.go` (the shared VIEW/EDIT role rule)
  `[code: ems/internal/base/authz/guard.go]`, `[code: ems/internal/base/authz/employee_rbac.go]`.
  Detailed in `GO-CONVENTIONS.md` and `GO-SECURITY.md`.
- **`base/service`** — `ulms` (audit log), `mailservice`, `mixpanel`, `redisser`.

## 5. Request lifecycle (`app/api`)

```
gin.Engine
  └─ middleware: RecoveryWithFlock → gintrace(DataDog) → cors → RequestLogger
       [per group: ValidatePaginationParams]
  └─ RunAction(handlerFn)  ── internal/base/handler/base.go
        1. Authentication(c)        → build *app.Context from gateway headers, validate X-Api-Key
        2. defer recover()          → 500 + Flock alert on panic
        3. resp := handler(ctx)     → *server.Response
        4. if resp.Log != nil       → ULMSService.SendLog (audit, non-GET 2xx)
        5. c.JSON(status, resp)
```

`RunAction` wraps a `HandlerFn` into a `gin.HandlerFunc`, running authentication, panic recovery,
the handler, ULMS logging, then serialization `[code: ems/internal/base/handler/base.go]`. Routes
are registered by `HttpServe.RouteWithGroup(group, method, path, fn)`; guarded routes use
`RouteWithGuard(..., RouteGuard{ResourceType, IDParam, Scopes})`, which inserts scope + resource-
ownership checks before the handler `[code: ems/app/api/router.go]`. Route groups map to caller
classes: `/api/web` (talenta-core web forwards), `/api/v1` (mobile/generic), `/api/v1/public`
(Launchpad/external via Kong, gated by `FeaturePublicAPIExpose`), `/api/v1/internal` (S2S)
`[code: ems/app/api/router.go]`, `[code: ems/app/api/public_router.go]`. See `GO-APIS.md`.

## 6. Composition root (`cmd/boot.go`)

There is **no DI framework** — `initHTTP()` is a hand-written composition root. It initializes
infrastructure (`initMasterDB` / `initReplicaDB` / `initArchiveDB`, `initCache`), then constructs
repositories, then services, then handlers, then `api.New(...)` and `setupGuard()`
`[code: ems/cmd/boot.go]`, `[code: ems/app/api/server.go]`. Repository constructors receive the
replica `*gorm.DB`, the master `*gorm.DB`, and the shared `empQuery` DSL, e.g.
`empRepo.NewRepository(replicaDBRepo.DB, replicaDBRepo, masterDBRepo.DB, empQuery, encryptionSvc, archiveDB)`
`[code: ems/cmd/boot.go]`. A single shared `encryptionSvc` pointer is threaded into every repo and
GORM plugin so all layers see the same PII-encryption config `[code: ems/cmd/boot.go]`.

## 7. Generated datasource layer

`internal/datasource/model` and `internal/datasource/models` are produced by **gorm.io/gen** via
`cmd/orm/generate.go` `[code: ems/cmd/orm/generate.go]`, `[code: ems/go.mod]`. Files carry
`// Code generated by gorm.io/gen. DO NOT EDIT.` `[code: ems/internal/datasource/model/tbl_user.gen.go]`.
`model` holds table structs (`TblUser`, `TableNameTblUser = "tbl_user"`); `models` holds the
type-safe query DSL activated per connection with `models.Use(db)` (aliased `empQuery` in boot)
`[code: ems/cmd/boot.go]`. This layer is **excluded from tests** (`go list ... | grep -v ./internal/datasource`)
`[code: ems/Makefile]`.

## 8. Integration seam (where Fabric attaches)

> **Superseded 2026-08-06 (`ADR-0014`).** The paragraph below described the pre-reconciliation design —
> a Kafka `employee_info` consumer feeding a new standalone anchor-service. **That design is retired
> outright, not narrowed.** The ratified design anchors **in-band**, from each of five profile-section
> write paths directly, with **0 Kafka topics and 0 standalone anchor-service deployables** — see
> `context/BLOCKCHAIN-INTEGRATION.md` and `context/PHP-INTEGRATION.md` §1 for the current seam, and
> `ADR-0014`/`ADR-0020` for the ratifying decisions. This repo's own layering is unaffected by the
> reconciliation (that is what this file otherwise documents); only the *sentence about what consumes
> its Kafka output* is stale, and is corrected here rather than left to mislead a reader.

EMS integrates with the rest of Talenta over **HTTP/JSON and Kafka only — no gRPC** (knowledge-graph
Layer C, gap G-20 — moot for the Fabric anchor design specifically, per `grounding-gaps.md` Part 2). It
**emits/publishes** the `employee_info` Kafka topic
`[code: ems/internal/employee/domain/sync_kafka.go]`,
`[code: ems/internal/employee/service/update_employment_data.go]` and shares MySQL with talenta-core
`[code: ems/dbconfig.yml]` — this topic remains **identity/org metadata only** and is **not** a Fabric
anchor trigger in the ratified design (it structurally cannot carry the `EDUCATION`/`ADDITIONAL`/
`PAYROLL` section content the anchor needs to hash, `[prd: §11.3 ADR-0014]`). **[ASSUMPTION] (gap G-09,
unaffected):** MySQL stays the system-of-record. **[ASSUMPTION] (gap G-10, reopened):** whether the
in-band recording component (the off-chain digest builder + Fabric Gateway client, `ADR-0014`) is hosted
inside this service, inside the talenta-core PHP service, or elsewhere is **not decided** — this repo's
existing layering (handler → service → repository) is a plausible host for the write paths it already
owns (e.g. the employment-data update path), but that is not the same claim as "this repo hosts the
Gateway client," which remains open. Fabric SDK/topology concerns live outside Go:
`> See fabric-network-architect` and `> See fabric-chaincode-dev`.

## 9. Cross-references

- Language-agnostic layering rationale: `> See coding-standards-backend/references/boundaries.md`,
  `> See coding-standards-backend/references/service-boundaries.md`.
- Naming, error, context, auth conventions: `GO-CONVENTIONS.md`.
- Data-access split (master/replica/archive, gorm/gen): `GO-DATABASE.md`.
- HTTP surface and route guards: `GO-APIS.md`.
- Async worker pools: `GO-CONCURRENCY.md`.
- Fabric capability mapping (D1–D15) and gaps G-01..G-22: `11-execution/knowledge-graph.md`,
  `11-execution/grounding-gaps.md`.

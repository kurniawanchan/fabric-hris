# GO-APIS — EMS HTTP surface (stub)

> **Read the depth docs first:** `GO-ARCHITECTURE.md` (§5 request lifecycle, §6 composition) and
> `GO-CONVENTIONS.md` (§4 auth). Language-agnostic API design:
> `> See coding-standards-backend/references/api-design.md`. This stub only records EMS-specific
> facts. **No gRPC** — EMS speaks HTTP/JSON + Kafka only (knowledge-graph Layer C, gap G-20).

## Transport & framework
- **gin** (`github.com/gin-gonic/gin`) is the HTTP framework `[code: ems/go.mod]`.
- Handlers are `type HandlerFn func(ctx *app.Context) *server.Response`; `BaseHTTPHandler.RunAction`
  adapts them into `gin.HandlerFunc` and owns auth → recover → handle → ULMS → serialize
  `[code: ems/internal/base/handler/base.go]`.
- Responses go through `BaseHTTPHandler.AsJson` / `AsJsonError`, yielding a `server.Response`
  `{status, message, data, errors, request_id}` `[code: ems/internal/base/handler/base.go]`.

## Route groups = caller classes
Registered in `HttpServe.setupRouter` `[code: ems/app/api/router.go]`:
| Prefix | Caller | Notes |
|---|---|---|
| `/api/web` | talenta-core web forwards | My-Info, resign, dashboard |
| `/api/v1` | mobile / generic | `ValidatePaginationParams` applied to the group |
| `/api/v1/public` | Launchpad / external via Kong | gated by `FeaturePublicAPIExpose`; `LaunchpadBillingGuard` `[code: ems/app/api/public_router.go]` |
| `/api/v1/internal` | service-to-service | forced INTERNAL scope in `Authentication` `[code: ems/internal/base/handler/base.go]` |

## Registering routes
- `h.RouteWithGroup(group, method, path, fn)` — plain route `[code: ems/app/api/router.go]`.
- `h.RouteWithGuard(group, method, path, fn, RouteGuard{ResourceType, IDParam, Scopes})` — inserts
  scope + object-ownership checks before the handler; a guarded route with no `Guard` wired **fails
  closed** (500) `[code: ems/app/api/router.go]`. See `GO-SECURITY.md`.
- Only `GET/POST/PUT/PATCH/DELETE` are accepted; anything else panics at registration
  `[code: ems/app/api/router.go]`.

## Input & validation
- Path/query parsing lives on `*app.Context` (`GetVar`, `GetQueryInt64`, `ParamAsInt64`), appending
  to `_errors` on failure; check `ctx.HasError()` `[code: ems/internal/base/app/context.go]`.
- Body binding + struct validation uses `go-playground/validator/v10`
  (`validate:"required"`, `datetime=2006-01-02`) `[code: ems/internal/education/domain/education.go]`.
- Pagination: `pkg/pagination.Paginator`, seeded by `ctx.NewPaginator()`; the count-before-limit
  idiom is standard in repos `[code: ems/internal/education/repository/repository.go]`.

## Health
`/health-check`, `/ready-check` (DB + Redis ping), `/status-check` (non-sensitive env dump)
`[code: ems/internal/base/handler/check.go]`.

## Rate limiting
`app/api/middleware_rate_limit.go` + `golang.org/x/time/rate` guard select routes
`[code: ems/app/api/public_router.go]`.

## Cross-references
- Full request flow: `GO-ARCHITECTURE.md` §5. Error→HTTP status mapping: `GO-ERROR-HANDLING.md`.
- API-layer auth/BOLA: `GO-SECURITY.md` and `> See coding-standards-backend/references/auth-and-authz.md`.
- For documenting/exercising the surface, use the `api-documentation` / `api-test-automation` skills.

# GO-PERFORMANCE — EMS performance levers (stub)

> **Read first:** `GO-DATABASE.md`, `GO-CONCURRENCY.md`, `GO-DEPLOYMENT.md`. Language-agnostic
> guidance: `> See coding-standards-backend/references/database-access.md`. EMS specifics only below.
> This is app/service performance — Fabric ledger throughput (block size, endorsement, state DB) is a
> separate domain: `> See fabric-performance`.

## Read scaling — replica + archive
Reads hit the **replica** connection and writes the **master** (`GO-DATABASE.md`); cold data is
offloaded to an **archive DB** selected by context flag via `pkg/dbresolver`, avoiding load on the
primary `[code: ems/pkg/dbresolver/dbresolver.go]`, `[code: ems/cmd/boot.go]`.

## Connection pool
Tuned per env: `SetConnMaxLifetime`, `SetConnMaxIdleTime`, `SetMaxIdleConns`, `SetMaxOpenConns` from
`DB_CONNECTION_*` `[code: ems/pkg/db/mysql.go]`, `[code: ems/app/appconf/config.go]`. Pool health is
sampled by `pkg/monitor` (dbstats, `DB_STATS_INTERVAL`, default 5m) `[code: ems/pkg/monitor/dbstats.go]`,
`[code: ems/app/appconf/config.go]`.

## Caching — Redis
`attis/cache` client + `pkg/cache` wrapper; feature caches layer on top (e.g. `empService.NewCache`)
`[code: ems/cmd/boot.go]`. Key/value compression is toggleable (`CACHE_COMPRESS_KEY`,
`CACHE_COMPRESS_VALUE`, `CACHE_COMPRESS_ALGO`) `[code: ems/app/appconf/config.go]`.

## Async offload
Long-running work (bulk import, invitations) is dispatched to bounded `workerpool.Pool[T]`s so the
request returns immediately and a full queue yields 429 rather than back-pressuring the server
`[code: ems/pkg/workerpool/pool.go]`. See `GO-CONCURRENCY.md`. Submit-endpoint budget in specs:
`< 500ms`; 1,000-row import `< 10 min` `[code: ems/specs/002-bulk-import-employee-go/plan.md]`.

## Query shape
- **Count-before-limit** pagination: total count then `Limit/Offset` on the same builder
  `[code: ems/internal/education/repository/repository.go]`.
- **Slim generated models** — `TblUser` intentionally comments out unused columns to avoid selecting
  more than needed `[code: ems/internal/datasource/model/tbl_user.gen.go]`.
- Select explicit columns, not `*`, in hot reads `[code: ems/internal/education/repository/repository.go]`.

## Horizontal scale
HPA autoscaling **min 6 / max 50** replicas in production, with a PDB to bound disruption
`[code: ems/deploy-alicloud/chart/values-production.yaml]`. Stateless HTTP + external MySQL/Redis
makes horizontal scaling the primary lever.

## Observability for perf
DataDog APM: gin + gorm + sql tracing (`DB_ENABLE_TRACING`), spans tagged with scope/company/sso;
StatsD counters via `pkg/metric`; one structured request line with `latency_ms`
`[code: ems/pkg/db/mysql.go]`, `[code: ems/internal/base/handler/base.go]`,
`[code: ems/app/api/middleware_logger.go]`.

## PII-encryption cost note
The encryption plugin rewrites SELECTs into `AES_DECRYPT(...)`/`COALESCE(...)` CASE expressions per
versioned key (`GO-SECURITY.md`); this adds per-column SQL work only for companies in the rollout
(gated), so cost scales with rollout, not globally `[code: ems/pkg/db/encryption_plugins.go]`.

## Cross-references
Scaling infra: `GO-DEPLOYMENT.md`. Async internals: `GO-CONCURRENCY.md`. Non-functional targets are
otherwise undefined (gap **G-08**): `11-execution/grounding-gaps.md`.

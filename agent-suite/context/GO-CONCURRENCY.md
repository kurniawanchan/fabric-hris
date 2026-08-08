# GO-CONCURRENCY — EMS async patterns (stub)

> **Read first:** `GO-ARCHITECTURE.md` (§6 composition root). Language-agnostic background-job
> guidance: `> See coding-standards-backend/references/background-jobs.md` and
> `> See coding-standards-backend/references/concurrency.md`. This stub records only EMS specifics.

## The bounded worker pool — `pkg/workerpool`
The single sanctioned async primitive is the generic `workerpool.Pool[T]`
`[code: ems/pkg/workerpool/pool.go]`:
- `NewPool[T](maxWorkers, bufferSize, fn)` starts `maxWorkers` goroutines reading a buffered channel
  of capacity `bufferSize` `[code: ems/pkg/workerpool/pool.go]`.
- **`Dispatch(job) bool` is non-blocking** — returns `false` when the queue is full so the caller can
  respond **HTTP 429**; never blocks the request goroutine `[code: ems/pkg/workerpool/pool.go]`.
- **Per-job panic recovery** (`runJob`): a panic in one job is logged with a stack trace and the
  worker continues. This is deliberate — an unrecovered panic in a detached goroutine would crash the
  process (the HTTP request-scoped recovery cannot catch it), which previously surfaced as nginx 502s
  on bulk-import `[code: ems/pkg/workerpool/pool.go]`.
- `Close()` is idempotent: it stops intake, drains buffered jobs, and `wg.Wait()`s for in-flight jobs
  — call it on graceful shutdown, after the HTTP server stops `[code: ems/pkg/workerpool/pool.go]`.

## Where it is used
Feature pipelines instantiate their own typed pool and close it on shutdown, held as package-level
vars in the composition root: `jlImportPool` (job-level import), `empImportPool` (employee bulk-add,
strangler-fig from talenta-core `EmployeeImportAddWorker`), `empInvitationPool`
`[code: ems/cmd/boot.go]`. Sizes come from config with validated defaults, e.g.
`EmployeeImportMaxWorkers = 2`, `EmployeeImportBufferSize = 8`, tunable via
`EMPLOYEE_IMPORT_MAX_WORKERS` / `EMPLOYEE_IMPORT_BUFFER_SIZE`
`[code: ems/app/appconf/config.go]`.

## Concurrency-safety expectations
- The `ProcessFn[T]` must be safe for concurrent execution (documented on the type)
  `[code: ems/pkg/workerpool/pool.go]`.
- Pool state uses `sync.WaitGroup`, `sync.Once`, and `atomic.Bool` (the `closed` flag) — no ad-hoc
  mutexes around the channel `[code: ems/pkg/workerpool/pool.go]`.
- The authz resolver `Registry` is the other concurrent structure: `sync.RWMutex`, written once at
  startup, read per request `[code: ems/internal/base/authz/registry.go]`.

## Context & cancellation
Async jobs carry their own derived context, not the request's (which is cancelled when the HTTP
response returns). Company scoping for DB/encryption inside a worker must be set explicitly via
`db.SetCompanyContext` since there is no inbound gateway header `[code: ems/pkg/db/company_context.go]`.

## Recommendation for a Fabric anchor-service
A Kafka-consumer anchor-service — **[ASSUMPTION] (gap G-10)**, the assumed future consumer of
`employee_info` (EMS only emits/publishes that topic; no in-repo consumer exists) — fits this pattern:
consume `employee_info`, dispatch each anchor job to a `workerpool.Pool[AnchorJob]`, return fast, and
let the pool serialize Fabric gateway submits with bounded concurrency and per-job panic isolation. Fabric-side ordering/endorsement
concerns are out of scope here: `> See fabric-chaincode-dev`, `> See fabric-performance`.

## Cross-references
- Graceful shutdown wiring: `GO-DEPLOYMENT.md`. Job status persistence (`tbl_resque_status`):
  `GO-DATABASE.md`. `> See coding-standards-backend/references/idempotency.md` for retry-safe jobs.

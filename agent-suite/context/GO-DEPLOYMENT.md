# GO-DEPLOYMENT — EMS build, image, and deploy (stub)

> **Read first:** `GO-ARCHITECTURE.md` (§1 entrypoint) and `GO-TESTING.md` (§7 CI). Language-agnostic
> guidance: `> See coding-standards-backend/references/dependencies.md`. EMS specifics only below.
> **[ASSUMPTION] (gap G-13):** the deployment target for a Fabric prototype is the **same Alibaba
> Cloud K8s** as EMS.

## Build
- `Makefile`: `build` → `go mod tidy && go build -o bin/ems main.go`; `engine` cross-compiles
  `GOOS=linux GOARCH=amd64 CGO_ENABLED=0`; `image` builds the Docker image `[code: ems/Makefile]`.
- Binary runs as `bin/ems http` (cobra `http serve`) `[code: ems/cmd/http.go]`, `[code: ems/main.go]`.
- Local hot-reload via `air` (`.air.toml`) `[code: ems/.air.toml]`; toolchain pinned in `mise.toml`
  `[code: ems/mise.toml]`.

## Container — multi-stage Dockerfile
`[code: ems/Dockerfile]`:
- Builder: `golang:1.21-alpine`, `CGO_ENABLED=0`, `GOPRIVATE=bitbucket.org` (SSH key injected as a
  build arg to fetch private `bitbucket.org/mid-kelola-indonesia/*` modules).
- Runtime: `alpine:latest` with `ca-certificates`, `tzdata`; **`TZ=Asia/Jakarta`**.
- **Runs as non-root** user `talenta` (uid/gid 1001) — a security baseline (see `GO-SECURITY.md`).
- Entrypoint `["./employee-management-service", "http"]`.

## CI/CD — Bitbucket Pipelines + mkrctl
`bitbucket-pipelines.yml` `[code: ems/bitbucket-pipelines.yml]`:
- **Unit test** step: `golang:1.21` self-hosted alicloud runner, `go mod vendor`, `APP_ENV=test go test`
  (excludes `internal/datasource`), coverage func.
- **Build/deploy** steps run only on `master`/`deployment-manifest`, using **`mkrctl`**
  (`/app/mkrctl app buildv3 …`, image tag `staging-<shortsha>`) on the `talenta.staging.alicloud`
  runner; config in `mkrctl.yaml` `[code: ems/mkrctl.yaml]`.

## Kubernetes — Helm chart
`deploy-alicloud/chart` (Helm) with per-env values and CD manifests
`[code: ems/deploy-alicloud/]`:
- `values-production.yaml`: node/pod affinity, **HPA autoscaling min 6 / max 50 replicas**
  `[code: ems/deploy-alicloud/chart/values-production.yaml]`.
- Templates include `deployment.yaml`, `hpa.yaml`, `pdb.yaml` (PodDisruptionBudget), F5 ingresses
  (public + canary), and a `cronjob-tbs.yaml` `[code: ems/deploy-alicloud/chart/templates/]`.
- CD entrypoints per env: `deploy-alicloud/cd/{staging,production,ppe,sandbox,debug}.yaml`
  `[code: ems/deploy-alicloud/cd/]`.

## Configuration at runtime
All config is env-driven (`InitAppConfig` reads `os.Getenv`); `.env.example` documents the keys
`[code: ems/app/appconf/config.go]`, `[code: ems/.env.example]`. `APP_ENV` selects behaviour:
`production` / `development` (staging) / `test`.

## Recommendation
A Fabric anchor-service can reuse this chart pattern (own Helm sub-chart on the same K8s), but
**[ASSUMPTION] (gap G-15):** its crypto material (MSP/TLS/HSM) is a separate concern from EMS env
config and app AES keys — `> See fabric-identity-security` and `> See fabric-operations`.

## Cross-references
- CI test details: `GO-TESTING.md` §7. Non-root/secret handling: `GO-SECURITY.md`. Autoscaling &
  pool sizing under load: `GO-PERFORMANCE.md`, `GO-CONCURRENCY.md`.

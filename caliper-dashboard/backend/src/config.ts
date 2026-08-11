import path from "node:path";

/**
 * Paths read from env vars, never hardcoded absolute paths (Consistency
 * Conventions) -- this repo can be checked out anywhere. Defaults assume
 * caliper-dashboard/backend is run with its own directory as cwd, one level
 * below fabric-hris root, so ../../ reaches the repo root from there.
 */
const repoRoot = process.env.FABRIC_HRIS_ROOT ?? path.resolve(import.meta.dirname, "../../..");

export const config = {
  port: Number(process.env.PORT ?? 4000),
  host: "127.0.0.1", // AD-8: never 0.0.0.0
  benchconfigsDir: path.join(repoRoot, "qa-tests/performance/benchconfigs"),
  resultsMdPath: path.join(repoRoot, "qa-tests/performance/RESULTS.md"),
  qa6ResultsMdPath: path.join(repoRoot, "qa-tests/performance/QA6-RESULTS.md"),
  rawLatenciesDir: path.join(repoRoot, "qa-tests/performance/results/raw-latencies"),
  callerLogPath: process.env.CALIPER_LOG_PATH ?? path.join(repoRoot, "caliper-run.log"),
};

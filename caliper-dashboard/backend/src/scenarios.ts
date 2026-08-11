import { readdirSync } from "node:fs";
import path from "node:path";
import { config } from "./config.js";

/** Alphabetically sorted benchconfig scenario names (without the .yaml extension). */
export function listScenarios(): string[] {
  return readdirSync(config.benchconfigsDir)
    .filter((f) => f.endsWith(".yaml") || f.endsWith(".yml"))
    .map((f) => path.basename(f, path.extname(f)))
    .sort((a, b) => a.localeCompare(b));
}

/** The first scenario alphabetically -- Story 1.1's first-load default. */
export function defaultScenario(): string {
  const scenarios = listScenarios();
  const first = scenarios[0];
  if (!first) {
    throw new Error(`no benchconfig scenarios found in ${config.benchconfigsDir}`);
  }
  return first;
}

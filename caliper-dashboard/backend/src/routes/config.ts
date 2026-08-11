import { Router } from "express";
import { readFileSync, existsSync } from "node:fs";
import path from "node:path";
import { config } from "../config.js";
import { getState } from "../state.js";
import { listScenarios } from "../scenarios.js";

export const configRouter = Router();

/**
 * GET /api/config -- the current scenario's benchconfig YAML, byte-for-byte
 * (FR8, AD-4). No transformation of any kind: what the operator sees is
 * exactly what Caliper itself reads for this run, so it can stand as
 * evidence the demo isn't rigged.
 */
configRouter.get("/config", (_req, res) => {
  const { scenario } = getState();

  if (!listScenarios().includes(scenario)) {
    res.status(404).json({ error: `unknown scenario: ${scenario}` });
    return;
  }

  const yamlPath = path.join(config.benchconfigsDir, `${scenario}.yaml`);
  const ymlPath = path.join(config.benchconfigsDir, `${scenario}.yml`);
  const resolved = existsSync(yamlPath) ? yamlPath : ymlPath;

  res.type("text/plain").send(readFileSync(resolved, "utf8"));
});

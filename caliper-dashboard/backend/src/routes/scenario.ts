import { Router } from "express";
import { listScenarios } from "../scenarios.js";
import { getState, setScenario } from "../state.js";

export const scenarioRouter = Router();

/** GET /api/scenarios -- list of available benchconfig scenarios (AD-4). */
scenarioRouter.get("/scenarios", (_req, res) => {
  res.json({ scenarios: listScenarios() });
});

/** GET /api/scenario -- current singleton { scenario, isLive } (AD-5). */
scenarioRouter.get("/scenario", (_req, res) => {
  res.json(getState());
});

/**
 * POST /api/scenario -- the one mutating action in this product (AD-4).
 * Never touches isLive (AD-5: independent axes).
 */
scenarioRouter.post("/scenario", (req, res) => {
  const { scenario } = req.body as { scenario?: unknown };
  if (typeof scenario !== "string" || scenario.trim() === "") {
    res.status(400).json({ error: "scenario must be a non-empty string" });
    return;
  }

  const available = listScenarios();
  if (!available.includes(scenario)) {
    res.status(400).json({ error: `unknown scenario: ${scenario}` });
    return;
  }

  setScenario(scenario);
  res.json(getState());
});

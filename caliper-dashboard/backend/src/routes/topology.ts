import { Router } from "express";
import { orgs, orderers } from "../topology.js";

export const topologyRouter = Router();

/**
 * GET /api/topology -- FR3. Hand-modeled, static (AD-7). Never returns
 * OrgClient-tenant02/peer0.tenant02, even though that container is running.
 */
topologyRouter.get("/topology", (_req, res) => {
  res.json({ orgs, orderers });
});

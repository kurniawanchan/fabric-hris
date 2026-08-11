import { Router } from "express";
import { loadHistory } from "../history.js";

export const historyRouter = Router();

/**
 * GET /api/history -- FR6. Pre-parsed structured JSON, never raw Markdown
 * (AD-4). NFR6: no filtering/scrubbing of its own -- whatever text
 * RESULTS.md/QA6-RESULTS.md already contain passes through unchanged.
 */
historyRouter.get("/history", (_req, res) => {
  try {
    res.json({ runs: loadHistory() });
  } catch (err) {
    res.status(500).json({ error: err instanceof Error ? err.message : String(err) });
  }
});

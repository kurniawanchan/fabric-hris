import cors from "cors";
import express from "express";
import { config } from "./config.js";
import { initScenario } from "./state.js";
import { defaultScenario } from "./scenarios.js";
import { scenarioRouter } from "./routes/scenario.js";
import { topologyRouter } from "./routes/topology.js";
import { historyRouter } from "./routes/history.js";
import { streamRouter } from "./stream.js";
import { startRawLatenciesWatcher } from "./watchers/rawLatencies.js";
import { startHeartbeat } from "./watchers/heartbeat.js";
import { startNodeHealthPoller } from "./poller/nodeHealth.js";

const app = express();
// Backend and frontend are two separate processes (AD-2), both localhost-only
// (AD-8) -- CORS is scoped to localhost origins only, never a wildcard.
app.use(cors({ origin: /^https?:\/\/(127\.0\.0\.1|localhost):\d+$/ }));
app.use(express.json());
app.use("/api", scenarioRouter);
app.use("/api", topologyRouter);
app.use("/api", historyRouter);
app.use("/api", streamRouter);

initScenario(defaultScenario());
startRawLatenciesWatcher();
startNodeHealthPoller();
startHeartbeat();

// AD-8: localhost-only binding -- never 0.0.0.0. host is a fixed constant in
// config.ts, not overridable via env var, so this can't regress by accident.
app.listen(config.port, config.host, () => {
  console.log(`caliper-dashboard backend listening on http://${config.host}:${config.port}`);
});

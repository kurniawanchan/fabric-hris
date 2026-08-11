import { broadcast, captureContext } from "../stream.js";
import { getJsonlLastActivity } from "./rawLatencies.js";
import { getMetricsLastActivity } from "../poller/nodeHealth.js";

const HEARTBEAT_MS = 3000;

/**
 * AD-6: a fixed-interval heartbeat carrying each real source's own
 * last-activity timestamp independently, so the frontend can mute one dead
 * feed without muting the others. `log` stays null until Story 2.2 builds
 * that watcher -- a source that has never run is "no data," not a false
 * staleness signal for a feed this build doesn't have yet.
 */
function tick(): void {
  broadcast({
    type: "heartbeat",
    context: captureContext(),
    sources: {
      log: null,
      jsonl: getJsonlLastActivity(),
      metrics: getMetricsLastActivity(),
    },
  });
}

let intervalHandle: NodeJS.Timeout | undefined;

export function startHeartbeat(): void {
  if (intervalHandle) return;
  intervalHandle = setInterval(tick, HEARTBEAT_MS);
}

export function stopHeartbeat(): void {
  if (intervalHandle) clearInterval(intervalHandle);
  intervalHandle = undefined;
}

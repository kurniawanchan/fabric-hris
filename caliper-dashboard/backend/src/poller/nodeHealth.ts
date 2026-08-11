import { allNodes, type TopologyNode } from "../topology.js";
import { broadcast, captureContext } from "../stream.js";

const POLL_INTERVAL_MS = 5000;
const POLL_TIMEOUT_MS = 2000;

/**
 * "Slow" threshold, keyed on operations-endpoint poll response latency (AD-7)
 * -- a placeholder pending real tuning against the live network under real
 * benchmark load (PRD/UX spine's own Deferred item), not a load-bearing
 * number.
 */
const SLOW_THRESHOLD_MS = 500;

/**
 * KNOWN GAP: every peer/orderer in network-docker-compose.yaml sets
 * CORE_METRICS_PROVIDER=disabled (ORDERER_METRICS_PROVIDER=disabled for
 * orderers) -- confirmed across all 7 real nodes. Fabric's own /metrics
 * Prometheus endpoint returns nothing useful with the provider disabled, so
 * there is currently no real CPU/mem data to poll. Per an explicit decision
 * during this story (not enabling metrics on the live network -- a config
 * change + restart this project's own history shows real risk in, this
 * close to the defense), resourceUsage is reported as null for every node,
 * every status, until a real metrics source exists. Health/reachability
 * (via /healthz, which works regardless of the metrics-provider setting) is
 * fully real.
 */
let lastActivity: string | null = null;
export function getMetricsLastActivity(): string | null {
  return lastActivity;
}

/**
 * KNOWN GAP #2: reachability is judged by "did the server respond at all,"
 * not by HTTP status. Two real, permanent (not transient) reasons found
 * against the live network, neither fixable without touching its config:
 *  - Every peer's /healthz always returns 503 -- Fabric's own health check
 *    pings the Docker daemon (for CCaaS builder support) and this network
 *    deliberately mounts no docker.sock (network-docker-compose.yaml's own
 *    "NOT IN SCOPE HERE" comment) -- a real, correctly-reported, PERMANENT
 *    sub-check failure, not evidence the peer itself is down.
 *  - Every orderer's operations port unexpectedly requires TLS (contradicts
 *    grounding-gaps.md G-37's plaintext assumption for orderers
 *    specifically -- a refinement to flag there separately) -- plain HTTP
 *    gets back a real, fast "Client sent an HTTP request to an HTTPS
 *    server" response, which still proves the port is open and the process
 *    is alive and responsive.
 * Given neither is fixable without a live-network config change + restart
 * (explicitly out of scope for this story), any RESOLVED fetch --
 * regardless of status code -- counts as reachable; only a THROWN fetch
 * (timeout, connection refused, DNS failure) counts as unreachable.
 */
async function pollOne(node: TopologyNode): Promise<void> {
  const url = `http://127.0.0.1:${node.operationsPort}/healthz`;
  const started = performance.now();

  try {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), POLL_TIMEOUT_MS);
    await fetch(url, { signal: controller.signal });
    clearTimeout(timer);
    const latencyMs = performance.now() - started;

    lastActivity = new Date().toISOString();
    broadcast({
      type: "node-status",
      context: captureContext(),
      nodeId: node.nodeId,
      status: latencyMs > SLOW_THRESHOLD_MS ? "slow" : "healthy",
      resourceUsage: null, // see KNOWN GAP #1 above
    });
  } catch {
    // NFR3: this one node's poll failure (timeout, connection refused, DNS)
    // never affects any other node's status or the rest of the app.
    broadcast({
      type: "node-status",
      context: captureContext(),
      nodeId: node.nodeId,
      status: "unreachable",
      resourceUsage: null,
    });
  }
}

function tick(): void {
  // Every node is polled independently and concurrently -- one node's
  // failure/timeout never blocks or delays another's poll (NFR3).
  for (const node of allNodes()) {
    void pollOne(node);
  }
}

let intervalHandle: NodeJS.Timeout | undefined;

export function startNodeHealthPoller(): void {
  if (intervalHandle) return;
  tick(); // first poll immediately, don't wait a full interval to show status
  intervalHandle = setInterval(tick, POLL_INTERVAL_MS);
}

export function stopNodeHealthPoller(): void {
  if (intervalHandle) clearInterval(intervalHandle);
  intervalHandle = undefined;
}

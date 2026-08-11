import type { Request, Response } from "express";
import { Router } from "express";
import { getState } from "./state.js";

/**
 * AD-4: the one relay channel. Every pushed event is exactly one of the four
 * shapes the spine defines -- no others, no optional fields beyond what's
 * listed. `context` is present on every event type, always, stamped with the
 * singleton's value at the moment the backend CAPTURED the underlying data,
 * never re-read later at emit/flush time (this is what keeps a mid-demo
 * scenario switch from mislabeling in-flight data).
 */
export interface EventContext {
  scenario: string;
  isLive: boolean;
}

export type StreamEvent =
  | {
      type: "metric-update";
      context: EventContext;
      throughput: number;
      latency: number;
      successCount: number;
      failureCount: number;
    }
  | {
      type: "node-status";
      context: EventContext;
      nodeId: string;
      status: "healthy" | "slow" | "unreachable";
      resourceUsage: { cpuPct: number; memPct: number } | null;
    }
  | {
      type: "log-line";
      context: EventContext;
      line: string;
      timestamp: string;
    }
  | {
      type: "heartbeat";
      context: EventContext;
      sources: { log: string | null; jsonl: string | null; metrics: string | null };
    };

/** captureContext stamps an event's context at capture time (AD-4/AD-5). */
export function captureContext(): EventContext {
  const { scenario, isLive } = getState();
  return { scenario, isLive };
}

const clients = new Set<Response>();

/** broadcast fans one event out to every currently-connected SSE client. */
export function broadcast(event: StreamEvent): void {
  const payload = `data: ${JSON.stringify(event)}\n\n`;
  for (const client of clients) {
    client.write(payload);
  }
}

export const streamRouter = Router();

streamRouter.get("/stream", (req: Request, res: Response) => {
  res.setHeader("Content-Type", "text/event-stream");
  res.setHeader("Cache-Control", "no-cache");
  res.setHeader("Connection", "keep-alive");
  res.flushHeaders();

  // AD-6: on any new/reconnected connection, push current/future state only
  // -- no backlog replay. The client's next heartbeat/event arrives on the
  // normal cadence; nothing here re-sends history.
  clients.add(res);

  req.on("close", () => {
    clients.delete(res);
  });
});

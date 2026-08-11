import { useEffect, useRef, useState } from "react";

const API_BASE = import.meta.env.VITE_API_BASE ?? "http://127.0.0.1:4000";

/** Mirrors backend/src/stream.ts's StreamEvent union exactly (AD-4). */
type StreamEvent =
  | {
      type: "metric-update";
      context: { scenario: string; isLive: boolean };
      throughput: number;
      latency: number;
      successCount: number;
      failureCount: number;
    }
  | {
      type: "heartbeat";
      context: { scenario: string; isLive: boolean };
      sources: { log: string | null; jsonl: string | null; metrics: string | null };
    };

export interface LiveMetricsState {
  isLive: boolean;
  throughput: number | null;
  latency: number | null;
  successCount: number | null;
  failureCount: number | null;
  /** true once the jsonl source has gone quiet longer than the staleness window (NFR1). */
  jsonlStale: boolean;
}

const STALE_WINDOW_MS = 8000; // a few heartbeat intervals (AD-6 heartbeat = 3s)

export function useLiveMetrics(): LiveMetricsState {
  const [state, setState] = useState<LiveMetricsState>({
    isLive: false,
    throughput: null,
    latency: null,
    successCount: null,
    failureCount: null,
    jsonlStale: false,
  });
  const lastJsonlActivity = useRef<number | null>(null);

  useEffect(() => {
    const source = new EventSource(`${API_BASE}/api/stream`);

    source.onmessage = (evt) => {
      const parsed = JSON.parse(evt.data) as StreamEvent;

      if (parsed.type === "metric-update") {
        setState((prev) => ({
          ...prev,
          isLive: parsed.context.isLive,
          throughput: parsed.throughput,
          latency: parsed.latency,
          successCount: parsed.successCount,
          failureCount: parsed.failureCount,
        }));
      } else if (parsed.type === "heartbeat") {
        if (parsed.sources.jsonl) {
          lastJsonlActivity.current = new Date(parsed.sources.jsonl).getTime();
        }
        const stale = lastJsonlActivity.current === null
          ? false
          : Date.now() - lastJsonlActivity.current > STALE_WINDOW_MS;
        setState((prev) => ({ ...prev, isLive: parsed.context.isLive, jsonlStale: stale }));
      }
    };

    return () => source.close();
  }, []);

  return state;
}

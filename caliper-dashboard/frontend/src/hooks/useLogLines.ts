import { useEffect, useRef, useState } from "react";

const API_BASE = import.meta.env.VITE_API_BASE ?? "http://127.0.0.1:4000";
const MAX_LINES = 500;

const STALE_WINDOW_MS = 8000; // matches useLiveMetrics' jsonl staleness window

type StreamEvent =
  | { type: "log-line"; line: string; timestamp: string }
  | { type: "heartbeat"; sources: { log: string | null; jsonl: string | null; metrics: string | null } }
  | { type: "metric-update" }
  | { type: "node-status" };

export interface LogEntry {
  line: string;
  timestamp: string;
}

export interface LogLinesState {
  lines: LogEntry[];
  /** AD-6: log source's own staleness, independent of jsonl/metrics (NFR3). */
  logStale: boolean;
}

/** FR7: the Notifications tab's live CLI tail. */
export function useLogLines(): LogLinesState {
  const [lines, setLines] = useState<LogEntry[]>([]);
  const [logStale, setLogStale] = useState(false);
  const lastLogActivity = useRef<number | null>(null);

  useEffect(() => {
    const source = new EventSource(`${API_BASE}/api/stream`);
    source.onmessage = (evt) => {
      const parsed = JSON.parse(evt.data) as StreamEvent;

      if (parsed.type === "log-line") {
        lastLogActivity.current = Date.now();
        setLogStale(false);
        setLines((prev) => {
          const next = [...prev, { line: parsed.line, timestamp: parsed.timestamp }];
          // Bound memory -- this is a live tail, not a full-history archive.
          return next.length > MAX_LINES ? next.slice(next.length - MAX_LINES) : next;
        });
      } else if (parsed.type === "heartbeat") {
        if (parsed.sources.log) {
          lastLogActivity.current = new Date(parsed.sources.log).getTime();
        }
        const stale =
          lastLogActivity.current !== null && Date.now() - lastLogActivity.current > STALE_WINDOW_MS;
        setLogStale(stale);
      }
    };
    return () => source.close();
  }, []);

  return { lines, logStale };
}

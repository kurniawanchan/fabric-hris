import { useEffect, useState } from "react";

const API_BASE = import.meta.env.VITE_API_BASE ?? "http://127.0.0.1:4000";

export interface NodeStatusEntry {
  status: "healthy" | "slow" | "unreachable";
  resourceUsage: { cpuPct: number; memPct: number } | null;
}

type StreamEvent =
  | { type: "node-status"; nodeId: string; status: NodeStatusEntry["status"]; resourceUsage: NodeStatusEntry["resourceUsage"] }
  | { type: "metric-update" }
  | { type: "heartbeat" };

/** Per-node live status, keyed by the canonical nodeId (AD-7). */
export function useNodeStatus(): Record<string, NodeStatusEntry> {
  const [status, setStatus] = useState<Record<string, NodeStatusEntry>>({});

  useEffect(() => {
    const source = new EventSource(`${API_BASE}/api/stream`);
    source.onmessage = (evt) => {
      const parsed = JSON.parse(evt.data) as StreamEvent;
      if (parsed.type === "node-status") {
        setStatus((prev) => ({
          ...prev,
          [parsed.nodeId]: { status: parsed.status, resourceUsage: parsed.resourceUsage },
        }));
      }
    };
    return () => source.close();
  }, []);

  return status;
}

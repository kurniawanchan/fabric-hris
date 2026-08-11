export interface HistoryRun {
  id: string;
  source: string;
  section: string;
  headers: string[];
  rows: Record<string, string>[];
}

const API_BASE = import.meta.env.VITE_API_BASE ?? "http://127.0.0.1:4000";

export async function fetchHistory(): Promise<HistoryRun[]> {
  const res = await fetch(`${API_BASE}/api/history`);
  const data = (await res.json()) as { runs: HistoryRun[] };
  return data.runs;
}

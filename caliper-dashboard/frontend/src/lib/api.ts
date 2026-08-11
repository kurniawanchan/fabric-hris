const API_BASE = import.meta.env.VITE_API_BASE ?? "http://127.0.0.1:4000";

export interface DashboardState {
  scenario: string;
  isLive: boolean;
}

export async function fetchScenarios(): Promise<string[]> {
  const res = await fetch(`${API_BASE}/api/scenarios`);
  const data = (await res.json()) as { scenarios: string[] };
  return data.scenarios;
}

export async function fetchState(): Promise<DashboardState> {
  const res = await fetch(`${API_BASE}/api/scenario`);
  return (await res.json()) as DashboardState;
}

/** FR8: the active scenario's benchconfig YAML, byte-for-byte, as raw text. */
export async function fetchConfig(): Promise<string> {
  const res = await fetch(`${API_BASE}/api/config`);
  if (!res.ok) {
    const body = (await res.json()) as { error?: string };
    throw new Error(body.error ?? `request failed: ${res.status}`);
  }
  return res.text();
}

export async function selectScenario(scenario: string): Promise<DashboardState> {
  const res = await fetch(`${API_BASE}/api/scenario`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ scenario }),
  });
  if (!res.ok) {
    const body = (await res.json()) as { error?: string };
    throw new Error(body.error ?? `request failed: ${res.status}`);
  }
  return (await res.json()) as DashboardState;
}

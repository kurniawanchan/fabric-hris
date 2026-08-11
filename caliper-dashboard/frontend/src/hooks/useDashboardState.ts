import { useCallback, useEffect, useState } from "react";
import { fetchScenarios, fetchState, selectScenario, type DashboardState } from "@/lib/api";

/**
 * Story 1.1's slice: scenario list + the singleton { scenario, isLive } read
 * from the backend. Later stories (1.2+) layer the SSE connection on top of
 * this same state shape (AD-5: REST and SSE context must always agree).
 */
export function useDashboardState() {
  const [scenarios, setScenarios] = useState<string[]>([]);
  const [state, setState] = useState<DashboardState | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      try {
        const [list, current] = await Promise.all([fetchScenarios(), fetchState()]);
        setScenarios(list);
        setState(current);
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e));
      }
    })();
  }, []);

  const changeScenario = useCallback(async (scenario: string) => {
    try {
      const next = await selectScenario(scenario);
      setState(next);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, []);

  return { scenarios, state, error, changeScenario };
}

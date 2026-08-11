import { ModeIndicator } from "@/components/ModeIndicator";
import { ScenarioSelector } from "@/components/ScenarioSelector";
import { useDashboardState } from "@/hooks/useDashboardState";

/**
 * Story 1.1: project + relay foundation. Persistent chrome (mode indicator,
 * scenario selector) is wired end-to-end against the real backend; the five
 * tabs themselves (Dashboard, Network Profile, Table List, Notifications,
 * Configuration -- EXPERIENCE.md's Information Architecture) land in later
 * stories.
 */
export function App() {
  const { scenarios, state, error, changeScenario } = useDashboardState();

  return (
    <div className="min-h-svh">
      <header className="flex items-center justify-between border-b px-6 py-4">
        <h1 className="font-heading text-lg font-semibold">Caliper Benchmark Dashboard</h1>
        <div className="flex items-center gap-4">
          {state && (
            <ScenarioSelector
              scenarios={scenarios}
              selected={state.scenario}
              onChange={(scenario) => void changeScenario(scenario)}
            />
          )}
          <ModeIndicator isLive={state?.isLive ?? false} />
        </div>
      </header>

      <main className="p-6">
        {error ? (
          <p className="text-status-down">{error}</p>
        ) : (
          <p className="text-muted-foreground">No benchmark running.</p>
        )}
      </main>
    </div>
  );
}

export default App;

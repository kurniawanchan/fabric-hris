import { ModeIndicator } from "@/components/ModeIndicator";
import { ScenarioSelector } from "@/components/ScenarioSelector";
import { MetricTile } from "@/components/MetricTile";
import { useDashboardState } from "@/hooks/useDashboardState";
import { useLiveMetrics } from "@/hooks/useLiveMetrics";

/**
 * Story 1.1: project + relay foundation. Persistent chrome (mode indicator,
 * scenario selector) is wired end-to-end against the real backend; the five
 * tabs themselves (Dashboard, Network Profile, Table List, Notifications,
 * Configuration -- EXPERIENCE.md's Information Architecture) land in later
 * stories.
 */
export function App() {
  const { scenarios, state, error, changeScenario } = useDashboardState();
  const live = useLiveMetrics();

  // AD-5: REST and SSE context must always agree -- once the stream has
  // reported anything, prefer it; before that first event arrives, fall
  // back to the initial REST-fetched value.
  const isLive = live.isLive || (state?.isLive ?? false);
  const hasData = live.throughput !== null;

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
          <ModeIndicator isLive={isLive} />
        </div>
      </header>

      <main className="p-6">
        {error && <p className="mb-4 text-status-down">{error}</p>}

        {hasData ? (
          <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
            <MetricTile
              label="Throughput (tx/s)"
              value={live.throughput}
              formatValue={(v) => v.toFixed(1)}
              stale={live.jsonlStale}
            />
            <MetricTile
              label="Latency (ms)"
              value={live.latency}
              formatValue={(v) => v.toFixed(1)}
              stale={live.jsonlStale}
            />
            <MetricTile label="Success" value={live.successCount} stale={live.jsonlStale} />
            <MetricTile label="Failure" value={live.failureCount} stale={live.jsonlStale} />
          </div>
        ) : (
          <p className="text-muted-foreground">No benchmark running.</p>
        )}
      </main>
    </div>
  );
}

export default App;

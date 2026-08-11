import { ModeIndicator } from "@/components/ModeIndicator";
import { ScenarioSelector } from "@/components/ScenarioSelector";
import { MetricTile } from "@/components/MetricTile";
import { TopologyDiagram } from "@/components/TopologyDiagram";
import { TableList } from "@/components/TableList";
import { Notifications } from "@/components/Notifications";
import { ConfigView } from "@/components/ConfigView";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useDashboardState } from "@/hooks/useDashboardState";
import { useLiveMetrics } from "@/hooks/useLiveMetrics";

function DashboardTab() {
  const live = useLiveMetrics();
  const hasData = live.throughput !== null;

  if (!hasData) {
    return <p className="text-muted-foreground">No benchmark running.</p>;
  }

  return (
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
  );
}

/**
 * The full five-tab shell per EXPERIENCE.md's Information Architecture:
 * Dashboard + Network Profile (Epic 1), Table List + Notifications +
 * Configuration (Epic 2).
 */
export function App() {
  const { scenarios, state, error, changeScenario } = useDashboardState();
  const live = useLiveMetrics();

  // AD-5: REST and SSE context must always agree -- once the stream has
  // reported anything, prefer it; before that first event arrives, fall
  // back to the initial REST-fetched value.
  const isLive = live.isLive || (state?.isLive ?? false);

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

        <Tabs defaultValue="dashboard">
          <TabsList>
            <TabsTrigger value="dashboard">Dashboard</TabsTrigger>
            <TabsTrigger value="network-profile">Network Profile</TabsTrigger>
            <TabsTrigger value="table-list">Table List</TabsTrigger>
            <TabsTrigger value="notifications">Notifications</TabsTrigger>
            <TabsTrigger value="configuration">Configuration</TabsTrigger>
          </TabsList>
          <TabsContent value="dashboard">
            <DashboardTab />
          </TabsContent>
          <TabsContent value="network-profile">
            <TopologyDiagram />
          </TabsContent>
          <TabsContent value="table-list">
            <TableList />
          </TabsContent>
          <TabsContent value="notifications">
            <Notifications />
          </TabsContent>
          <TabsContent value="configuration">
            {state && <ConfigView scenario={state.scenario} />}
          </TabsContent>
        </Tabs>
      </main>
    </div>
  );
}

export default App;

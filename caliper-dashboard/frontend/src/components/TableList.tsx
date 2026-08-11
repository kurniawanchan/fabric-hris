import { useEffect, useState } from "react";
import { fetchHistory, type HistoryRun } from "@/lib/history";

/**
 * FR6: already-collected benchmark runs, each's key headline numbers
 * visible without navigating elsewhere. Each source table (RESULTS.md /
 * QA6-RESULTS.md) keeps its own real columns -- write throughput, read
 * latency, and MVCC contention are genuinely different measurements, so
 * this never forces one fixed schema onto them.
 */
export function TableList() {
  const [runs, setRuns] = useState<HistoryRun[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchHistory()
      .then(setRuns)
      .catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)));
  }, []);

  if (error) {
    return <p className="text-status-down">{error}</p>;
  }
  if (!runs) {
    return <p className="text-muted-foreground">Loading historical runs…</p>;
  }

  return (
    <div className="space-y-8">
      {runs.map((run) => (
        <div key={run.id}>
          <div className="mb-2 flex items-baseline gap-2">
            <h3 className="font-heading text-sm font-semibold">{run.section}</h3>
            <span className="font-mono text-xs text-muted-foreground">{run.source}</span>
          </div>
          <div className="overflow-x-auto rounded-lg border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/50">
                  {run.headers.map((h) => (
                    <th key={h} className="px-3 py-2 text-left font-medium whitespace-nowrap">
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {run.rows.map((row, i) => (
                  <tr key={i} className="border-b last:border-0">
                    {run.headers.map((h) => (
                      <td key={h} className="px-3 py-2 whitespace-nowrap tabular-nums">
                        {row[h]}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ))}
    </div>
  );
}

import { useEffect, useState } from "react";
import { fetchTopology, type TopologyResponse, type TopologyNode } from "@/lib/topology";

/**
 * DESIGN.md's topology-diagram component: nodes as cards (orderers as
 * circles), grouped into three org columns, the shared orderer set shown
 * once -- never duplicated per org column (Story 1.3's second AC).
 */
function NodeCard({ node }: { node: TopologyNode }) {
  return (
    <div
      className={`flex items-center justify-center border bg-card p-3 text-center text-sm font-medium ${
        node.kind === "orderer" ? "rounded-full aspect-square" : "rounded-lg"
      }`}
    >
      {node.hostname}
    </div>
  );
}

export function TopologyDiagram() {
  const [topology, setTopology] = useState<TopologyResponse | null>(null);

  useEffect(() => {
    void fetchTopology().then(setTopology);
  }, []);

  if (!topology) {
    return <p className="text-muted-foreground">Loading topology…</p>;
  }

  return (
    <div className="space-y-6">
      <div>
        <div className="mb-2 font-mono text-[13px] font-medium tracking-wide text-muted-foreground uppercase">
          Orderers (shared, shown once)
        </div>
        <div className="flex gap-3">
          {topology.orderers.map((o) => (
            <div key={o.nodeId} className="w-24">
              <NodeCard node={o} />
            </div>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-3 gap-4">
        {topology.orgs.map((org) => (
          <div key={org.mspId} className="rounded-lg border p-4">
            <div className="mb-3 font-heading text-sm font-semibold">{org.displayName}</div>
            <div className="flex flex-col gap-2">
              {org.peers.map((peer) => (
                <NodeCard key={peer.nodeId} node={peer} />
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

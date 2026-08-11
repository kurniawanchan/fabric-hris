import { useEffect, useState } from "react";
import { fetchTopology, type TopologyResponse, type TopologyNode } from "@/lib/topology";
import { useNodeStatus } from "@/hooks/useNodeStatus";
import { StatusBadge } from "@/components/StatusBadge";

/**
 * DESIGN.md's topology-diagram component: nodes as cards (orderers as
 * circles), grouped into three org columns, the shared orderer set shown
 * once -- never duplicated per org column (Story 1.3's second AC). Each
 * node carries its own live status badge, positioned top-right (Story 1.4).
 */
function NodeCard({
  node,
  status,
}: {
  node: TopologyNode;
  status: ReturnType<typeof useNodeStatus>[string] | undefined;
}) {
  return (
    <div
      className={`relative flex items-center justify-center border bg-card p-3 text-center text-sm font-medium ${
        node.kind === "orderer" ? "aspect-square rounded-full" : "rounded-lg"
      }`}
    >
      <span
        className={node.kind === "orderer" ? "absolute -top-2 right-1/2 translate-x-1/2" : "absolute -top-2 right-2"}
      >
        <StatusBadge status={status?.status} />
      </span>
      {node.hostname}
    </div>
  );
}

export function TopologyDiagram() {
  const [topology, setTopology] = useState<TopologyResponse | null>(null);
  const nodeStatus = useNodeStatus();

  useEffect(() => {
    void fetchTopology().then(setTopology);
  }, []);

  if (!topology) {
    return <p className="text-muted-foreground">Loading topology…</p>;
  }

  return (
    <div className="space-y-8">
      <div>
        <div className="mb-4 font-mono text-[13px] font-medium tracking-wide text-muted-foreground uppercase">
          Orderers (shared, shown once)
        </div>
        <div className="flex gap-4">
          {topology.orderers.map((o) => (
            <div key={o.nodeId} className="w-24">
              <NodeCard node={o} status={nodeStatus[o.nodeId]} />
            </div>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-3 gap-4">
        {topology.orgs.map((org) => (
          <div key={org.mspId} className="rounded-lg border p-4">
            <div className="mb-3 font-heading text-sm font-semibold">{org.displayName}</div>
            <div className="flex flex-col gap-3">
              {org.peers.map((peer) => (
                <NodeCard key={peer.nodeId} node={peer} status={nodeStatus[peer.nodeId]} />
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

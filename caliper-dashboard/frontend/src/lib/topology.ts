export interface TopologyNode {
  nodeId: string;
  kind: "peer" | "orderer";
  hostname: string;
}

export interface TopologyOrg {
  mspId: string;
  displayName: string;
  peers: TopologyNode[];
}

export interface TopologyResponse {
  orgs: TopologyOrg[];
  orderers: TopologyNode[];
}

const API_BASE = import.meta.env.VITE_API_BASE ?? "http://127.0.0.1:4000";

export async function fetchTopology(): Promise<TopologyResponse> {
  const res = await fetch(`${API_BASE}/api/topology`);
  return (await res.json()) as TopologyResponse;
}

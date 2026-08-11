/**
 * AD-7: hand-modeled topology for exactly the three `TenantChannelGenesis`
 * orgs -- never derived from a live discovery call against the Fabric
 * network, which could return a fourth org (`OrgClient-tenant02`). Verified
 * directly against fabric-network/network/configtx/configtx.yaml's
 * `TenantChannelGenesis` profile and its `&Org1`/`&OrgClientTenant01`/`&Org3`
 * anchor-peer definitions, and cross-checked against
 * network-docker-compose.yaml's real container names -- 4 peers (2 + 1 + 1),
 * not 5 as an earlier design-doc draft stated before this implementation
 * caught the discrepancy against ground truth.
 *
 * `nodeId` here is this product's single canonical node-identity source
 * (AD-7) -- backend/poller (Story 1.4) must import and key every poll/emit
 * by these exact ids, never inventing its own hostname-based scheme.
 */

export type NodeKind = "peer" | "orderer";

export interface TopologyNode {
  nodeId: string;
  kind: NodeKind;
  /** The real container hostname -- also the operations-endpoint host (Story 1.4). */
  hostname: string;
  /**
   * The HOST-published port for this node's operations service (Story 1.4's
   * poller target) -- verified directly against network-docker-compose.yaml's
   * real `ports:` mappings, not carried over from grounding-gaps.md's G-37
   * disclosure, which cites different orderer ports (7071/8070/9070) than
   * the real file (7060/8060/9060) -- a second doc/reality drift found
   * during this story, corrected separately.
   */
  operationsPort: number;
}

export interface TopologyOrg {
  mspId: string;
  displayName: string;
  peers: TopologyNode[];
}

export const orgs: TopologyOrg[] = [
  {
    mspId: "Org1MSP",
    displayName: "Org1",
    peers: [
      { nodeId: "org1-peer0", kind: "peer", hostname: "peer0.org1", operationsPort: 9444 },
      { nodeId: "org1-peer1", kind: "peer", hostname: "peer1.org1", operationsPort: 9445 },
    ],
  },
  {
    mspId: "OrgClient-tenant01MSP",
    displayName: "OrgClient-tenant01",
    peers: [
      { nodeId: "tenant01-peer0", kind: "peer", hostname: "peer0.tenant01", operationsPort: 9446 },
    ],
  },
  {
    mspId: "Org3MSP",
    displayName: "Org3",
    peers: [{ nodeId: "org3-peer0", kind: "peer", hostname: "peer0.org3", operationsPort: 9447 }],
  },
];

// The 3-node Raft consenter set -- shared across the channel, shown once,
// never duplicated per org column (Story 1.3's second AC).
export const orderers: TopologyNode[] = [
  { nodeId: "orderer0", kind: "orderer", hostname: "orderer0.org1", operationsPort: 7060 },
  { nodeId: "orderer1", kind: "orderer", hostname: "orderer1.org1", operationsPort: 8060 },
  { nodeId: "orderer2", kind: "orderer", hostname: "orderer2.org1", operationsPort: 9060 },
];

/** Every node this product ever polls/displays -- the canonical set (AD-7). */
export function allNodes(): TopologyNode[] {
  return [...orgs.flatMap((o) => o.peers), ...orderers];
}

export function allNodeIds(): string[] {
  return allNodes().map((n) => n.nodeId);
}

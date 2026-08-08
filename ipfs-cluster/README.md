# IPFS Private Cluster (REC-5, ADR-0016)

Two `ipfs/kubo` nodes on a swarm-key-gated private libp2p network
(`LIBP2P_FORCE_PNET=1` + `IPFS_SWARM_KEY_FILE`), plus two `ipfs/ipfs-cluster`
peers intended to run CRDT-based pin-set consensus on top of them. Brought up
by `docker-compose.yaml`.

## What actually works, verified live

- `ipfs0` and `ipfs1` form the private swarm correctly (`ipfs swarm connect`
  succeeds; cross-node fetch confirmed — content added on one node, pinned
  and read back correctly on the other, over the swarm, via each node's own
  HTTP API).
- `write-path-integration/ipfsclient` uses this private swarm directly:
  encrypts a document, adds the ciphertext to the primary node, and
  explicitly pins the same CID on the replica node — the same 2-peer
  durability property ADR-0016 asks for, achieved without going through the
  cluster orchestration layer below.

## Disclosed limitation: `cluster0`/`cluster1` do not peer with each other

Three real defects in the cluster bring-up were found and fixed along the
way:

1. kubo 0.43's AutoConf feature refuses to start on a private network unless
   explicitly disabled (fixed via the `/container-init.d/*.sh` hook —
   `init-hooks/disable-autoconf.sh`).
2. `CLUSTER_PEERADDRESSES` only seeds `service.json`'s `peer_addresses`
   field at init time; it is not an active CRDT bootstrap trigger. Fixed by
   passing `--bootstrap` on the daemon's own `command:` (see `cluster1` in
   `docker-compose.yaml`).
3. A stale peerstore file (left over from earlier failed bootstrap attempts)
   caused "dial to self attempted" errors even after fix (2). Fixed by
   clearing `/data/ipfs-cluster/peerstore` on both containers.

After all three fixes, `cluster0` and `cluster1` still fail to complete a
libp2p security handshake with each other:

```
failed to negotiate security protocol: incoming message was too large
```

Both peers' `CLUSTER_SECRET_FILE` contents were confirmed byte-identical
(MD5 match) — this is not a mismatched-secret problem. The root cause was
not found after a genuine debugging effort and remains open.

**Pragmatic decision:** REC-5 does not depend on the cluster orchestration
layer being reachable — ADR-0016's actual requirement is a 2-peer,
swarm-key-gated private network with pinning on both peers, which the two
kubo nodes already deliver directly. `write-path-integration/ipfsclient`
pins explicitly on both nodes instead of relying on CRDT consensus to
replicate the pin. Getting `cluster0`/`cluster1` peered is left as future
work under `DEP-3` (production-grade cluster deployment), not silently
dropped.

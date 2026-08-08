# ADR-0016: Private, swarm-key-gated IPFS cluster for encrypted supporting documents

- **Status:** **Accepted**
- **Date:** 2026-08-06
- **Deciders:** fabric-architect
- **Rests on assumption(s):** G-02 (partially — CID-only on-chain scope for supporting documents is
  ratified by FR-31); cluster operator identity and replication-node count are **new** topology
  assumptions not yet covered by any existing gap ID — tentatively grouped under **G-04**'s
  operator-topology extension, flagged below for addition to the gap register (not this ADR's file
  to edit).
- **Supersedes:** none (new — no predecessor ADR covers off-chain document storage)
- **Superseded-by:** none
- **Ratifying authority / date:** PRD `prd-fabric-hris-2026-08-02/prd.md` §5.1 (Batasan Teknologi),
  §7 Kelompok F (FR-30…FR-33), §11.4 (temuan arsitektural IPFS), §11.3 (ADR manifest, row ADR-0016) —
  human decision, Chandra Kurniawan, 2026-08-02/03.

## Context

Supporting documents attached to a profile section (identity documents for PERSONAL, diplomas for
EDUCATION, pay slips for PAYROLL, etc.) cannot go on-chain (INV-1) and cannot go untracked (FR-32
requires every document be linkable to its section). The PRD ratifies **IPFS Private Cluster**,
per-employee-encrypted before upload, as the answer (§5.1, FR-30/31).

Three properties of IPFS the prior design package had never modeled surfaced during reconciliation
(PRD §11.4) and are load-bearing for this decision, not incidental:

1. **A CID is a retrieval capability, not just an identifier.** Content-addressing means anyone
   holding a CID can fetch the object. The CID is written **on-chain** (FR-32), so it is readable by
   every org holding a ledger replica — under ADR-0012/0013 that is Org1, every tenant's
   `OrgClient-<tenantID>`, and Org3. The **only** thing standing between "holds the CID" and "can
   fetch the ciphertext" is cluster membership, gated by a **swarm key** — not the CID's secrecy,
   which does not exist once it is on-chain.
2. **`unpin` is not `delete`.** Any node that has ever fetched a block may retain a local copy
   indefinitely; there is no cluster-side operation that guarantees an object's bytes are gone from
   every node that ever touched it.
3. **IPFS objects are immutable at their address.** Re-encrypting a document (key rotation, suspected
   compromise) changes its ciphertext bytes, which changes its CID — the old CID cannot be updated in
   place to point at new ciphertext.

FR-33 ("objek IPFS hanya direplikasi di dalam *private cluster*") and FR-30/31/32 ground the decision
directly; §11.4's three findings above are the reasoning this ADR must carry forward honestly rather
than gloss over.

## Decision

We will operate a **private, permissioned IPFS cluster** — not the public IPFS network — gated by a
pre-shared libp2p **swarm key** distributed only to the cluster's own node operators. Every
supporting document is encrypted client-side with `KEY_EMPLOYEE` (key custody and rotation policy
owned by `security-architect`, out of this ADR's lane) **before** it is added to the cluster; only the
resulting CID, linked to its profile section (FR-32), is written on-chain.

**Pinning/replication across the cluster is a *durability* control on the ciphertext blob — explicitly
NOT an erasure mechanism.** We state this precisely because it is the crux of why this design remains
compliant after erasure: `unpin` does not guarantee deletion (finding 2 above), so **erasure is
achieved entirely by destroying `KEY_EMPLOYEE`** (and the operational-DB plaintext/salts, per the
crypto-shred design) — never by attempting to delete, unpin, or otherwise remove the IPFS object. This
is the load-bearing reason Pasal 26/44's crypto-shredding argument (PRD §9.1 row 4) remains sound even
though the ciphertext blob may physically persist somewhere in the cluster: the object is *retained but
inert* — unopenable without a key that no longer exists anywhere.

**Re-encryption produces a new CID.** Any event that requires re-encrypting a document (key rotation,
compromise response) therefore requires anchoring a **new** `EmployeeProfileRecord` version pointing
at the new CID — the old record's CID reference becomes historical, not corrected in place. This is a
`data-model.md` consequence, noted here, not redesigned here.

**Cluster operator and node count — stated as a recommendation, not a ratified fact** (no requirement
document fixes either number): recommend a minimum of **2** cluster peers for the encrypted-blob
durability property to mean anything, both **operated by Org1** in this iteration. `**TBD`: should
`OrgClient-<tenantID>` (the enterprise client, per ADR-0013) also operate a pinning peer, mirroring why
it holds its own Fabric ledger replica — extending the "independent copy" property from the ledger to
the document tier? The PRD does not settle this; flagged here rather than decided silently.

**The cluster's node operator is a fourth trust boundary**, distinct from the three Fabric MSP-credentialed
boundaries (Org1/Org2/Org3) fixed by ADR-0012. Root/cluster-API access to a pinning node is governed
by the cluster's own access-control plane (swarm key + cluster REST API auth), entirely outside
Fabric's X.509/MSP layer — an operator with node access can retain, inspect at the ciphertext level,
or exfiltrate the swarm key itself, independent of any Fabric identity. This boundary is **not yet
represented** in this package's threat model and must be added (`security-architect`'s STOP-LIST/T-
numbering, not this ADR's job to enumerate).

## Alternatives considered

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. Private IPFS cluster, swarm-key gated (chosen)** | CID-only on-chain footprint keeps INV-1/FR-31 clean; swarm-key gating closes the "public retrievability" gap a leaked CID would otherwise open; content-addressed dedup for identical documents | Adds a 4th trust boundary (cluster operator) that must be separately secured and modeled; `unpin` ≠ `delete` means the design's erasure story rests entirely on key destruction, not storage-side action | **Chosen** |
| B. Public IPFS network | No private infrastructure to operate | The CID is on-chain and therefore not secret; on public IPFS, **anyone** globally who learns the CID (a leaked ledger export, a compromised org node, or simply a member reading their own valid channel) can fetch the ciphertext from any public gateway with zero additional control — this removes the *fetch-side* control this design relies on entirely, and gives no operator accountability for an HRIS PII pipeline | Rejected |
| C. Store documents as encrypted BLOBs directly in the operational relational DB instead of IPFS | One fewer subsystem to operate | Re-creates, for supporting documents, exactly the "single mutable operator-controlled store" problem this platform anchors *profile sections* to solve; no content-addressing/dedup benefit; foreclosed by ratification (§5.1 names IPFS Private Cluster specifically) — listed for completeness only | Rejected |
| D. Traditional object storage (S3-compatible bucket) with per-object server-side encryption | Native delete semantics (no "unpin ≠ delete" caveat); mature access-control tooling | Foreclosed by ratification (§5.1); noted because it would have offered a simpler *erasure* story (an actual delete API) at the cost of losing content-addressing — not evaluated further since it is out of scope | Rejected (out of scope by ratification) |

## Consequences

- **Positive:** FR-31/INV-1 hold — zero document bytes ever reach the ledger, only CIDs. Swarm-key
  gating closes the public-retrievability gap that Option B would have left open. Because retrieval
  already requires cluster membership, an outside attacker who obtains a leaked CID cannot even
  attempt offline brute-force against ciphertext they cannot fetch — a defense-in-depth layer on top
  of `KEY_EMPLOYEE` encryption alone, not a replacement for it.

- **Negative / trade-off — stated explicitly, per instruction, not glossed over.** `unpin` is **not**
  `delete`. Any cluster peer that has ever fetched a block may retain it indefinitely. This design's
  erasure guarantee therefore rests **entirely** on `KEY_EMPLOYEE` destruction (and the corresponding
  salt/off-chain-value destruction in the crypto-shred design) being complete and irreversible — **not**
  on any IPFS-side deletion action, because no such action is relied upon or attempted. FR-27's
  principle ("TIDAK BOLEH mengubah/menghapus state on-chain untuk memenuhi penghapusan") extends
  naturally here: this design **also** does not rely on deleting the IPFS object. Erasure is a
  key-management property of a different subsystem (`security-architect`'s domain), not a
  storage-layer property of this one.

- **Negative / trade-off:** re-encryption (rotation or compromise response) produces a new CID for
  the same logical document — `EmployeeProfileRecord`'s CID reference is not stable across a
  re-encryption event. Every re-encryption requires a new anchored record version. This is a
  `data-model.md` consequence, flagged here, not redesigned here.

- **Negative / trade-off:** the cluster operator is a newly-identified, not-yet-modeled 4th trust
  boundary. Its access-control plane (swarm key + cluster API) is entirely independent of Fabric's
  MSP layer — a compromise there does not require compromising any Org1/Org2/Org3 Fabric identity.

- **Follow-ups:**
  - `security-architect`: add the cluster-operator trust boundary to the threat model; own swarm-key
    custody/rotation policy and `KEY_EMPLOYEE` lifecycle (this ADR only names the boundary and states
    the erasure dependency, it does not design either control).
  - `fabric-engineer`/data-model owner: model CID-versioning-on-re-encryption as a new
    `EmployeeProfileRecord` version, per the third IPFS finding above.
  - Cluster operator scope and node/replication count (recommendation above) should be escalated to a
    tracked gap — this ADR proceeds on a **recommendation**, not a ratified requirement.
  - Risk register (reported, not authored here): add "IPFS cluster operator as unmodeled 4th trust
    boundary" and "erasure completeness depends entirely on key destruction, not object deletion."

## Related

- Supersedes: none.
- Relates to: **ADR-0012** (Org1 as the assumed, not-yet-ratified cluster operator), **ADR-0013**
  (per-tenant client org as a candidate additional pinning operator — open `TBD`), forthcoming
  **ADR-0011** (digest scheme — CID is one of the fields the per-section record anchors), forthcoming
  **ADR-0015**/**ADR-0019** (crypto-shred erasure and key-domain custody — this ADR is the IPFS-side
  justification for why crypto-shred, not object deletion, is sufficient; owned by `security-architect`).
- Knowledge-graph: **D1** (confidentiality boundary reasoning, applied off-chain here), **D12**
  (immutable audit — the CID reference, not the document, is what the ledger keeps immutable).
  Grounding gap: **G-02** (on-chain scope, partial); cluster-operator topology not yet gap-tracked —
  flagged for the grounding-gaps.md owner to add. Context: `context/FABRIC-PRIVATE-DATA.md` (as the
  nearest confidentiality-boundary analogue; IPFS itself is outside the pinned Fabric corpus, cited
  per PRD §11.4 findings and the errata/rencana-rekonsiliasi record rather than `[docs:]`).

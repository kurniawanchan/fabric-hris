# ADR-0010: Satisfy right-to-erasure via v2.5 PurgePrivateData / blockToLive, leaving only the immutable salted hash

- **Status:** **Superseded by ADR-0015**
- **Date:** 2026-07-13
- **Deciders:** fabric-engineer, security-architect
- **Rests on assumption(s):** G-06 (erasure is required; regulatory basis — Indonesia PDP Law / GDPR / other — and the exact retention window are open). Purge/`blockToLive` mechanics grounded in `[docs: private-data-arch.rst#private-data-purging]`, `[docs: private-data/private-data.md]`.

## Context
A blockchain is append-only, yet Privacy-by-Design (a mandated frame, `[brief: agents-guide.md]`) and almost certainly G-06's regulation demand a right-to-erasure. This is the core tension and it must be resolved at the **private-data layer, not the block layer** — you cannot rewrite committed blocks.

Corpus facts that resolve it:
- Deleting private data from state still leaves its **history in the peer's private database**; to remove it from **all** peers use **`PurgePrivateData`**, not `DelPrivateData` `[docs: private-data/private-data.md]`.
- Collections can auto-purge private data after it has been unmodified for a configurable number of blocks — the **`blockToLive`** property; `blockToLive: 0` keeps it indefinitely `[docs: private-data-arch.rst]`.
- Purging leaves **only the hash on the blockchain as immutable evidence** that the data once existed `[docs: private-data/private-data.md]`.
- Explicit `PurgePrivateData` requires the **`V2_5`** channel capability `[docs: private-data-arch.rst]`.

Because ADR-0001 keeps sensitive PII plaintext entirely off-chain (only salted commitments are anchored), the *baseline* prototype has **no private data to purge** — erasure there is achieved by deleting the off-chain **salt** (and, per policy, the source PII in MySQL), which renders the surviving commitment un-openable and, being salted, un-brute-forceable ("crypto-shredding"). Purge/`blockToLive` become the required mechanism the moment any actual field *value* is ever placed in a PDC (data-model doc §7). Relevant capability: **D4** (purge/`blockToLive`/retention & erasure). See context `context/PRIVACY-BY-DESIGN.md` §4, `context/FABRIC-PRIVATE-DATA.md`, `context/BLOCKCHAIN-DATA-MODEL.md` §6.

## Decision
We will satisfy right-to-erasure so that **only the immutable salted commitment survives**, via a two-tier mechanism:
1. **Baseline (commitments only):** erase by **crypto-shredding** — delete the off-chain salt (anchor-service `anchor_map`) and the authoritative MySQL plaintext. The on-chain commitment then cannot be opened or brute-forced (salt defeats guessing), and no ledger operation is needed.
2. **Whenever a field value lives in a PDC:** use **Fabric-native `PurgePrivateData`** (requires the `V2_5` capability) to remove the value and its history from all peers, and set a finite **`blockToLive`** on such collections for automatic time-boxed purge. In both tiers the salted hash remains as immutable audit evidence (D12).

**Numeric constraints (set by this ADR):**
- Channel capability required for explicit purge: **`V2_5`** `[docs: private-data-arch.rst]`.
- PDC private data purged from **all** peers via `PurgePrivateData` (not just current state, unlike `DelPrivateData` which leaves history) `[docs: private-data/private-data.md]`.
- `blockToLive`: a finite **N > 0** blocks for any erasable PDC (**0** = keep forever, used only for the non-erasable commitment tier). `blockToLive` and a collection `name` **cannot be changed** after definition, and collections **cannot be deleted** `[docs: private-data-arch.rst]` — therefore the retention window is sized **up front**.
- What survives erasure: **1** artifact — the salted commitment (non-reversible, non-erasable, by design).

## Alternatives considered
| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. Crypto-shred off-chain salt + `PurgePrivateData`/`blockToLive` for any PDC (chosen)** | Honours erasure with no block rewrite; leaves immutable proof a change occurred without disclosing the value; baseline needs no ledger op at all; Fabric-native purge available the moment a value goes into a PDC | Salt store becomes an erasure-critical asset (deletion must be reliable and irreversible); `blockToLive`/collection immutability forces retention sizing up front; requires `V2_5` capability on the channel | **Chosen** |
| B. Keep everything on-chain forever | Simplest; maximal audit history | Rejected — directly violates Privacy-by-Design and the right-to-erasure; permanently un-erasable PII (or an openable commitment whose salt is retained) is non-compliant with any plausible G-06 basis | Rejected |
| C. Store nothing erasable on-chain at all | No purge machinery ever needed | **Partially adopted** — it *is* the baseline (only non-reversible commitments are anchored, ADR-0001), but rejected as the *complete* answer because a subset of orgs may need to exchange a real field value via a PDC (data-model §7), and that case needs `PurgePrivateData`/`blockToLive`; so C is the floor, not the ceiling | Partially adopted |

## Consequences
- **Positive:** the erasure story is complete and law-agnostic (holds whichever G-06 basis applies); the baseline path is trivial (delete a salt) and needs no ledger rewrite; the PDC path uses a corpus-documented v2.5 primitive; the surviving salted hash preserves tamper-evidence without disclosing the erased value.
- **Negative / trade-off:** if the salt is ever co-located with the commitment, erasing the value but keeping salt+hash may still allow brute-forcing small domains — **salt must be stored off-ledger and purged with the value**; `blockToLive` and collection names being immutable means a mis-sized retention window cannot be corrected in place (only superseded by a new collection); the `V2_5` capability is a hard channel prerequisite.
- **Follow-ups:** append a prototype-phase risk row for **salt co-location / incomplete crypto-shred** and **retention-window mis-sizing (immutable `blockToLive`)** to `10-risk/risk-register.md` at G3+; the concrete retention window stays **open** until G-06's regulatory basis is fixed (the crypto-shred design holds regardless); erasure-as-a-control review is owned by `fabric-security-review/references/config-hardening.md`.

## Related
- Relates to: **ADR-0001** (only salted commitments are on-chain, which is what makes baseline erasure a crypto-shred), **ADR-0002** (erasure on an audit overlay never rewrites the system-of-record), **ADR-0006** (the anchor-service drives the off-chain salt deletion).
- Knowledge-graph: Layer D capabilities **D4** (and D1/D12), Layer E frame **Privacy-by-Design (E3)**; traceability spine chain #4 (erasure). Grounding gap(s): **G-06** (and G-04 for the two-org PDC case). Context doc(s): `context/PRIVACY-BY-DESIGN.md`, `context/FABRIC-PRIVATE-DATA.md`, `context/BLOCKCHAIN-DATA-MODEL.md`.

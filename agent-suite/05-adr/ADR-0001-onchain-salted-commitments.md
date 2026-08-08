# ADR-0001: Anchor only salted commitments of PII-change events on-chain

- **Status:** **Superseded by ADR-0011**
- **Date:** 2026-07-13
- **Deciders:** fabric-engineer, security-architect
- **Rests on assumption(s):** G-02 (which PII goes on-chain vs off-chain). Confidentiality mechanics grounded in `[docs: private-data/private-data.md]`, `[docs: private-data-arch.rst#protecting-private-data-content]`.
- **Supersedes:** none
- **Superseded-by:** ADR-0011
- **Ratifying authority / date:** `prd-fabric-hris-2026-08-02/prd.md` §5.2, human decision 2026-08-02/03 (anchoring unit changed to profile section; identifier construction corrected to a two-tier keyed hierarchy — see `errata-tesis.md` E-1).

## Context
This is the core confidentiality decision for the HRIS-on-Fabric overlay. The employee master is highly sensitive PII — national ID (`citizen_id`/KTP), tax id (`npwp_16`), phone, salary, bank, family `no_ktp` — and a ledger is append-only and replicated. Anything written to a peer's ledger is effectively permanent and cannot be selectively rewritten, which collides head-on with Privacy-by-Design and right-to-erasure.

The corpus gives the escape: a private data collection writes the *actual data* peer-to-peer to authorized orgs and writes **a hash of that data** — endorsed, ordered, on every peer's ledger — which "serves as evidence of the transaction and is used for state validation and can be used for audit purposes"; a third party can later recompute the hash to prove the data existed `[docs: private-data/private-data.md]`. Predictable private data (the corpus example is a dollar amount) is brute-force-guessable against a bare hash and must therefore include a **random salt** `[docs: private-data-arch.rst#protecting-private-data-content]`. HRIS relevance is acute: `citizen_id`, phone, and salary-band have small/enumerable domains, so an *unsalted* commitment of any of them is effectively reversible.

Relevant capabilities: **D2** (on-chain hash vs off-chain data), **D3** (salt predictable PII), **D1** (PDC field-level confidentiality). See context `context/BLOCKCHAIN-DATA-MODEL.md` §§1-4, `context/FABRIC-PRIVATE-DATA.md`, `context/CRYPTOGRAPHY.md` §3. Existing app-level AES-256 at-rest encryption and the versioned `PII_ENCRYPTION_KEY` model are reused unchanged `[code: ems/pkg/db/encryption_plugins.go]` — Fabric anchors on top of them, it does not replace them.

## Decision
We will anchor **only salted one-way commitments of PII-change events** on the ledger; **no sensitive PII plaintext (nor any reversible derivative such as a masked or AES-ciphertext value) is ever written on-chain, in any form.** Each on-chain `AnchorRecord` carries `commitment = SHA-256( salt ‖ canonical(changedFields) )` (a 256-bit digest) plus non-PII metadata; the plaintext and the salt stay off-chain (MySQL system-of-record + the anchor-service `anchor_map` store). Field *names* (`changedFields`) are treated as low-sensitivity metadata and, if a threat review deems the name-set sensitive, are folded into the commitment. `employeeRef`/`actorRef` are themselves salted commitments so the public ledger is not a linkable directory of who-edited-whom.

**Numeric constraints (set by this ADR):**
- Commitment algorithm: **SHA-256** (256-bit output).
- Per-event salt: **≥ 128 bits** drawn from a CSPRNG, unique per change event, stored off-chain only.
- PII plaintext on-chain: **0 bytes** — enforced at the anchor-service boundary (only whitelisted metadata + the commitment become chaincode arguments).

## Alternatives considered
| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. Salted commitments only (chosen)** | PII never leaves the encrypted off-chain store; on-chain record is non-reversible yet verifiable; smallest possible blast radius; erasure = crypto-shred the off-chain salt (ADR-0010) with no ledger rewrite | Verification needs off-chain salt + source data; loses the salt → commitment un-openable (a feature for erasure, a risk otherwise — mitigated by treating the salt store as critical infra) | **Chosen** |
| B. PII values in Private Data Collections (D1) | Field-level confidentiality without a separate channel; native purge available | Rejected because the actual value is still **replicated to every authorized peer's private DB** = a materially larger blast radius than off-chain-only, and erasure is harder (must `PurgePrivateData` across all member peers vs deleting one off-chain salt). Reserved only for the narrow case where two orgs must exchange a real field value (data-model doc §7) | Rejected |
| C. PII plaintext on-chain | Trivial to implement; directly queryable | Rejected outright — immutable, replicated, permanently un-erasable PII is a privacy catastrophe and violates Privacy-by-Design principle 2 and right-to-erasure irreparably | Rejected |

## Consequences
- **Positive:** confidentiality becomes a data-model invariant, not a bolt-on; the ledger cannot leak a PII value because it never holds one; salt defeats small-domain brute-forcing (D3); the design composes cleanly with erasure (ADR-0010) and the audit-overlay posture (ADR-0002).
- **Negative / trade-off:** the ledger alone cannot answer "what was the value" — verification is a two-sided recompute against off-chain inputs; the off-chain salt store becomes security-critical infrastructure (loss = un-openable commitment); `changedFields` name leakage must be watched (recommendation: fold into commitment if sensitive).
- **Follow-ups:** append a prototype-phase risk row for **on-chain low-entropy PII brute-forcing** (mitigated here by mandatory ≥128-bit salt) to `10-risk/risk-register.md` when G3+ prototype risks are added; confirm G-02/G-17 (PII inventory is representative, not exhaustive — treat any unlisted column as sensitive); salt-store integrity/backup is carried as a seam failure mode in `context/BLOCKCHAIN-INTEGRATION.md` §5.

## Related
- Relates to: **ADR-0002** (audit/anchor overlay — why off-chain stays authoritative), **ADR-0010** (erasure by crypto-shredding the off-chain salt / `PurgePrivateData`).
- Knowledge-graph: Layer D capabilities **D1/D2/D3**, Layer E frames **CIA-Confidentiality (E1)**, **Privacy-by-Design (E3)**; traceability spine chains #1 (anchor) and #2 (confidentiality). Grounding gap(s): **G-02** (and G-17). Context doc(s): `context/BLOCKCHAIN-DATA-MODEL.md`, `context/FABRIC-PRIVATE-DATA.md`, `context/CRYPTOGRAPHY.md`.

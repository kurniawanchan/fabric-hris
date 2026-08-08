# ADR-0004: Single channel + private data collections for the prototype

- **Status:** **Superseded by ADR-0013**
- **Date:** 2026-07-13
- **Deciders:** fabric-architect
- **Rests on assumption(s):** G-03 (multi-tenancy model not fixed by any requirement doc)
- **Superseded-by:** ADR-0013
- **Ratifying authority / date (of supersession):** PRD `prd-fabric-hris-2026-08-02/prd.md` §5.1, §7
  Kelompok D, INV-5, §11.3 — human decision, Chandra Kurniawan, 2026-08-02/03. Single-channel +
  composite-key tenancy is replaced by channel-per-tenant because isolation must be cryptographic
  (channel/MSP-enforced), not the logical composite-key scoping this ADR accepted as a trade-off —
  see ADR-0013.

## Context
Fabric offers two isolation axes: **channels** (hard ledger separation) and **private data
collections** (field-level confidentiality within a channel). The prototype targets **one pilot
tenant** ([ASSUMPTION] G-01/G-08), so heavyweight per-tenant isolation is not yet warranted, but the
two-org design (ADR-0003) needs a way to exchange a field value + salt between HR and Audit during
endorsement without broadcasting it. Capabilities **D1** (PDC), **D7** (channels).
`[docs: channels.rst]`, `[docs: private-data/private-data.md#when-to-use-a-collection-within-a-channel-vs-a-separate-channel]`.

## Decision
A **single channel** shared by the two orgs. Tenant scoping is by **composite key** (`companyId`
prefix on every anchor key), not by separate channels. **Private data collections** are used **only**
for the narrow case of exchanging a field value + its salt between the HR and Audit orgs during
endorsement — never as a bulk PII store (consistent with ADR-0001).

## Alternatives considered
| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. Single channel + narrow PDC** | Simplest; sufficient isolation for 1 pilot tenant via key-scoping; supports the two-org field-exchange | No hard cross-tenant isolation | **Chosen** |
| B. Channel-per-tenant | Hard ledger isolation + data-residency story | N channels × N configs/policies to operate; premature for a single-tenant pilot | Deferred (multi-tenant scale option) |
| C. Single channel, no PDC | Least moving parts | No mechanism for the HR↔Audit field-value exchange without putting it in the open transaction | Rejected |

## Consequences
- **Positive:** low operational cost; composite-key scoping is enough for a single-tenant demonstration.
- **Trade-off:** no hard tenant isolation — acceptable while single-tenant, **must be revisited before
  any multi-tenant production use** (reopen via G-03). PDC scope is deliberately narrow so it does not
  contradict ADR-0001's "no PII bulk store on-chain/off-peer" rule.
- **Follow-up:** the composite-key scheme is specified in the G5 data model; PDC endorsement/member
  policy in G5/G6.

## Related
- Relates to: **ADR-0003** (two orgs share this channel), **ADR-0001** (PDC reserved for narrow field exchange), **ADR-0007** (state DB).
- Knowledge-graph: D1/D7. Grounding gap: **G-03**. Context: `context/FABRIC-CHANNELS.md`, `context/FABRIC-PRIVATE-DATA.md`.

# Zero-Knowledge Proof in the HRIS-on-Fabric System

> **Verified 2026-08-06 against the reconciled design** (`prd-fabric-hris-2026-08-02/prd.md`,
> ADR-0011/ADR-0012/ADR-0013/ADR-0021). **This doc's core position is unaffected: ZKP/Idemix remains
> deferred (ADR-0008 is not reopened by this reconciliation).** Only two topology-adjacent details
> below (the anonymity-set bound and the assumed anchor-service language) referenced the
> pre-reconciliation two-org/Kafka-anchor-service model as fact and have been corrected — the ZKP
> necessity gate, the feasibility analysis, and the recommendation are otherwise unchanged.

> **What this doc is.** A context/theory map of zero-knowledge proofs (ZKP) and the *one* concrete
> ZKP mechanism Fabric 2.5 ships — **Idemix**. It is a REFERENCE doc, not a skill: Idemix mechanics
> are owned by `fabric-identity-security` and are **cross-referenced, not restated**. Its job is to
> (a) state the ZKP theory plainly, (b) bind it to what Fabric can actually do today, and (c) be
> honest that ZKP is an *exploration*, not a confirmed requirement.

> **Necessity is an [ASSUMPTION].** No user story is known to require anonymous credentials — see
> gap **G-07** / assumption **A-KG-7** in `../11-execution/`. This doc is the theory basis for the
> **future `zkp-designer` skill (roadmap G2)**; it does not commit the prototype to building ZKP.

---

## 1. ZKP theory in one screen

A zero-knowledge proof lets a **prover** convince a **verifier** that a statement is true **without
revealing anything beyond the truth of the statement**. A ZKP must satisfy three properties:

- **Completeness** — if the statement is true, an honest prover convinces an honest verifier.
- **Soundness** — if the statement is false, no cheating prover can convince the verifier (except
  with negligible probability).
- **Zero-knowledge** — the verifier learns *nothing* except that the statement is true; the proof
  reveals no additional information.

Two axes matter for design:

- **Interactive vs. non-interactive** — whether prover and verifier exchange challenges, or the
  proof is a single self-contained message. Idemix credential proofs are used non-interactively at
  transaction time.
- **Proof-of-knowledge vs. commitment** — a full ZKP proves *knowledge of a secret satisfying a
  predicate*; a **commitment** (a salted hash) only proves *a value existed and matches later*. Both
  appear in this system; §4 draws the line.

The generic ZKP properties above are standard cryptographic theory (textbook), stated here as the
lens the mandated **ZKP frame** applies `[brief: agents-guide.md]` (`knowledge-graph.md` §Layer E).
Everything Fabric-specific below is corpus-grounded.

---

## 2. Idemix — the concrete Fabric mechanism (D10)

Idemix ("Identity Mixer") is a cryptographic protocol suite that provides **anonymity** (transact
without revealing the transactor's identity) and **unlinkability** (one identity sends multiple
transactions without revealing they came from the same identity) `[docs: idemix.rst]`.

**Three actors** `[docs: idemix.rst]`:

- **Issuer** — certifies a set of user attributes as a *credential* (Fabric CA ≥ 1.3, or `idemixgen`
  for dev).
- **User** — generates a **zero-knowledge proof** of possession of the credential and **selectively
  discloses** only the attributes chosen to reveal (Fabric Java SDK is the user API).
- **Verifier** — an Idemix MSP in Fabric that checks the proof.

The canonical example from the docs: Alice proves to a store clerk she holds a valid driver's
license **without revealing name, address, or exact age** — the proof reveals only license validity,
and repeated visits are not linkable to each other `[docs: idemix.rst]`.

**How it differs from X.509.** Both certify signed attributes bound to a secret key. The difference
is the signature scheme: Idemix uses a scheme allowing efficient **proofs of signature possession
without revealing the signature or the unselected attribute values**. With plain X.509 **all
attributes must be revealed to verify the signature**, so every use is linkable; avoiding that would
require fresh certificates each time. Idemix defeats linkability against **both verifiers and the
issuing CA** `[docs: idemix.rst]`.

> Depth (do not restate): enrollment flow, `idemixEnroll`, MSP `msptype: idemix` config, and CID
> `GetAttributeValue` are in `fabric-identity-security/references/advanced-identity.md`.

---

## 3. HRIS use cases (candidate, not committed)

The minimal-disclosure chain in `knowledge-graph.md` §3 (spine #5) is:
*prove "active employee / band X" → Idemix selective disclosure (D10) → ZKP frame (E2).*

Plausible HRIS statements a data subject or service might want to prove **without exposing
identity**:

1. *"The bearer is an active employee of company C"* — e.g. to an external benefits/partner portal
   (the highest-exposure surface, Layer A "external / partner integrators").
2. *"The bearer's salary is in band X"* — e.g. for a loan-eligibility check, without disclosing the
   exact salary value (Layer B, the `PAYROLL` profile section — `[prd: §4]`).
3. *"The requester holds HR-admin role in company C"* — an anonymous authorization for a bulk
   operation.

Each is an *attribute predicate* — exactly Idemix's shape. **But see §4: Fabric today cannot carry
these as custom attributes.**

---

## 4. Hard limitations (the honesty section)

These are **documented facts**, and they are decisive for whether Idemix can serve the use cases in
§3 `[docs: idemix.rst]`:

- **Fixed attribute set.** You **cannot** issue an Idemix credential with **custom attributes**
  today. Only four attributes exist; only **`ou`** (affiliation) and **`role`** (member/admin) are
  ever revealed to chaincode via CID; enrollment-ID and revocation-handle are never revealed in the
  signature. → *"salary band X"* and *"active employee"* are **not expressible** as Idemix attributes
  now; only org-affiliation and member/admin role are.
- **No revocation yet.** The revocation handle exists but **credential revocation is not supported**.
  → A terminated employee's anonymous credential cannot be cleanly revoked — a serious gap for HRIS.
- **Idemix orgs cannot endorse.** An Idemix MSP cannot endorse a chaincode transaction or approve a
  chaincode definition; too many Idemix orgs can break the default majority endorsement policy. →
  Idemix is a **client-side, verify-only** identity here, not an endorser.
- **One Idemix MSP per channel (recommended).** Multiple Idemix MSPs on a channel leak the signer's
  MSP-ID, breaking cross-org anonymity; Idemix gives anonymity **only among members of the same
  MSP**. → Under the ratified **three-org**, channel-per-tenant topology (ADR-0012/ADR-0013, gap
  **G-04 closed** on org count), the anonymity set is still bounded to a single org's membership on
  one tenant's channel — and for the two smaller orgs (a tenant's own client org, or the auditor org)
  that membership is typically **one peer/operator**, an even narrower anonymity set than the
  pre-reconciliation two-org framing implied. Only the platform org (2 peers, shared across every
  tenant channel) offers more than a trivial anonymity set.
- **SDK support.** The user API is the **Java SDK**. **There is no anchor-service in this design to
  compare it against** — recording is in-band from the HRIS write path (ADR-0014), and which process
  hosts that in-band caller is unresolved (gap **G-10, reopened**); a Go component is the more likely
  shape (`project-context.md`), but this is not yet decided, so the Go-vs-Java SDK mismatch remains a
  live consideration for whichever process eventually calls Idemix-issued credentials, not a fact
  about a specific service.

**Recommendation.** Given the fixed-attribute and no-revocation limits, Idemix in Fabric 2.5 fits
*"prove org-affiliation / admin-role anonymously"* but **not** *"prove salary band / active-status"*.
For value-band predicates, prefer a **salted commitment + range statement** approach (see §5) or defer
to a purpose-built ZK system in the `zkp-designer` skill rather than forcing Idemix.

---

## 5. Commitments: the ZK-adjacent path this system already has

Where a full anonymous credential is overkill, a **salted commitment** gives a weaker but sufficient
property for audit: prove *a specific value existed and matches later* without publishing the value.
Fabric's own guidance to **salt predictable private data against brute force**
`[docs: private-data-arch.rst]` is exactly this, and the repo already ships a keyed, per-record-salted
hash in `bankhasher` `[code: ems/pkg/bankhasher/bankhasher.go]`.

- Commitment (salted hash) → proves existence/match, **not** a predicate like `salary < N`. See
  `CRYPTOGRAPHY.md` §3.
- True ZKP (Idemix / future range proofs) → proves the *predicate* without the value.

**Design line:** use commitments for tamper-evident audit (the confirmed prototype goal, G-01); reserve
true ZKP for the evaluation-phase exploration (G-07).

---

## 6. Fact vs. recommendation (summary)

| # | Statement | Type |
|---|---|---|
| Idemix gives anonymity + unlinkability via ZK proof of credential possession | `[docs: idemix.rst]` | Fact |
| Only `ou` and `role` are revealed to chaincode; no custom attributes | `[docs: idemix.rst]` | Fact |
| Revocation not yet supported; Idemix orgs cannot endorse | `[docs: idemix.rst]` | Fact |
| Idemix fits org/role anonymity, not salary-band predicates today | derived from limits | Recommendation |
| Use salted commitments for audit; reserve ZKP for evaluation phase | G-01 / G-07 | Recommendation |
| ZKP is not a confirmed requirement | gap G-07 / A-KG-7 | Assumption |

---

## 7. Traceability

- **Capability:** D10 (Idemix ZKP), with D3 (salt/commitment) as the lighter-weight fallback —
  `knowledge-graph.md` §Layer D.
- **Frame:** ZKP (E2), supporting Privacy-by-Design (E3) minimal disclosure.
- **Spine:** minimal-disclosure chain #5 in `knowledge-graph.md` §3.
- **Gaps:** G-07 / A-KG-7 (is ZKP required), G-04 (closed on org count — three orgs; still bounds the
  anonymity set to one org's channel membership), G-10 (reopened — in-band caller's host/language
  undecided), G-11 (role→attribute map).
- **Roadmap:** basis for the future `zkp-designer` skill (G2).
- **Sibling docs:** commitment crypto → `CRYPTOGRAPHY.md`; privacy rationale → `PRIVACY-BY-DESIGN.md`.

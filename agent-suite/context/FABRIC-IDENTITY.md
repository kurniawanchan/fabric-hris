# Fabric Identity (PKI, X.509, Idemix) — context stub

**What it is.** Every actor in Fabric — peer, orderer, client, admin — has a digital **identity** in
the form of an X.509 certificate issued by a trusted CA and interpreted by an MSP; identity plus MSP
membership is what authorizes any action ("principal"). Fabric also supports **Idemix** (Identity
Mixer), a zero-knowledge anonymous-credential scheme that lets a client prove membership/attributes
(e.g. "is an employee") without revealing the specific certificate — enabling unlinkable, minimal-
disclosure interactions.

> **Depth: see fabric-identity-security/references/pki-hierarchy.md and
> fabric-identity-security/references/advanced-identity.md — do not restate here.** Those own the
> PKI trust chain, principals, and the Idemix issuer/credential model (MSP structure is in
> FABRIC-MSP; enrollment in FABRIC-CA).

**Corpus citations (verified).**
- `[docs: identity/identity.md]` — digital identity, X.509, principals, MSP role
- `[docs: idemix.rst]` — Idemix anonymous, unlinkable zero-knowledge credentials

**HRIS hooks (knowledge-graph Layer D).**
- **D6** (MSP + CA X.509 identity), **D9** (chaincode ABAC via CID/OUs), **D10** (Idemix ZKP) —
  the identity + minimal-disclosure toolkit behind the Zero-Trust (E6) and ZKP (E2) frames.
- Gap **G-07** — whether Idemix/ZKP is actually required (and for which story) is unresolved
  (evaluation-phase exploration); **G-11** — HRIS-role → OU/attribute mapping still to be defined.

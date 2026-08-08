# Fabric MSP (Membership Service Provider) — context stub

**What it is.** The MSP is the component that turns raw PKI material into Fabric *identities with
membership*: it defines which root/intermediate CAs are trusted for an organization, how X.509 certs
map to roles (client/peer/admin) and Organizational Units, and it holds the CRLs and validity rules
used at validation time. Every peer, orderer, and client acts under an MSP identity, so the MSP is
the anchor for all endorsement and access decisions.

> **Depth: see fabric-identity-security/references/msp-structure.md — do not restate here.** It owns
> the MSP folder layout, local vs channel MSP, NodeOUs, and identity-validity rules
> (cross-refs pki-hierarchy.md and cert-lifecycle.md; CA enrollment is in FABRIC-CA).

**Corpus citations (verified).**
- `[docs: msp.rst]` — MSP structure, trusted CAs, OUs, admins, CRLs
- `[docs: membership/membership.md]` — membership-service concept

**HRIS hooks (knowledge-graph Layer D).**
- **D6** (MSP + Fabric CA identity / PKI / X.509) — the identity foundation every actor authenticates
  and endorses under; basis for NIST Zero Trust (frame E6).
- Gap **G-11** — the HRIS-role → MSP-OU/attribute mapping is unresolved. **G-04** (closed, ADR-0012)
  — three MSPs: `Org1MSP` (platform), `OrgClient-<tenantID>MSP` (enterprise client, one per tenant),
  `Org3MSP` (auditor, read-only); the regulator-org line remains `[ASSUMPTION]`.

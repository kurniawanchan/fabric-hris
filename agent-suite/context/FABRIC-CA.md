# Fabric CA (Certificate Authority) & Enrollment — context stub

**What it is.** Fabric CA is the default certificate authority for a Fabric network: it registers
identities (name, secret, type, affiliation, attributes) and then *enrolls* them, issuing the X.509
enrollment certificates (ECerts) that MSPs consume. The `fabric-ca-client` drives register/enroll/
reenroll/revoke, and issued attributes are what chaincode later reads for attribute-based access
control. A CA is optional — any compliant X.509 PKI works — but it is the standard enrollment path.

> **Depth: see fabric-identity-security/references/fabric-ca.md — do not restate here.** It owns
> register/enroll/reenroll/revoke flows, attribute issuance, and CA-vs-external-PKI choices
> (MSP consumption of the resulting certs is in FABRIC-MSP; rotation in cert-lifecycle.md).

**Corpus citations (verified).**
- `[docs: commands/fabric-ca-commands.rst]` — `fabric-ca-client` register/enroll/reenroll/revoke
- `[docs: msp.rst]` — how enrolled certs populate an MSP

**HRIS hooks (knowledge-graph Layer D).**
- **D6** (MSP + Fabric CA identity) — enrollment is where an HRIS actor is minted a Fabric identity;
  **D15** (certificate lifecycle — expiry, rotation, CRL) is the operational continuation.
- Gap **G-11** — attributes issued at enrollment must encode the HRIS-role mapping that chaincode
  reads (still to be defined).

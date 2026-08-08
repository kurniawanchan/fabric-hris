<!--
Immutable once Accepted. Supersede with a new ADR; do not edit in place.
-->

# ADR-0009: Keep Fabric signing/TLS keys separate from the application-level AES PII keys

- **Status:** **Superseded by ADR-0019**
- **Date:** 2026-07-13
- **Deciders:** security-architect
- **Rests on assumption(s):** G-15 (whether Fabric key management integrates with the existing `PII_ENCRYPTION_KEY` versioning is not fixed by a requirement doc). Grounded technically in `[docs: hsm.md]` and `[code: ems/pkg/db/encryption_plugins.go]`.

## Context
Two cryptographic subsystems exist and must not be conflated (context `CRYPTOGRAPHY.md` §1). **Domain A — Fabric node crypto:** X.509/PKI signing keys (D6), TLS transport keys (D13), optionally HSM-held private keys (D14), cert lifecycle/CRL (D15); its purpose is to prove *who signed / who may connect*. Keys are enrolled via Fabric CA (context `FABRIC-CA`). **Domain B — application PII crypto:** AES-256-CBC envelope encryption at rest (a 64-hex-char key + 32-hex-char/16-byte IV per version), a `key_version` column, and a three-phase per-tenant rollout (`encrypt` → `read-from-encrypted` → `write-only`) `[code: ems/pkg/db/encryption_plugins.go]`, `[code: talenta-core/traits/EncryptableFieldsTrait.php]`; its purpose is to keep *plaintext unreadable at rest*.

Two documented constraints force the design. First, an HSM can protect peer/orderer **MSP signing** keys via PKCS11 (FIPS 140-2 available), but **TLS must use file-based keys** — so even within Domain A the key material splits `[docs: hsm.md]`. Second, the two domains use categorically different primitives (asymmetric X.509 signing/TLS vs symmetric AES-256 data-at-rest) with different lifecycles (cert rotation/CRL vs versioned key rollout). This is a design decision only — no prototype code.

## Decision
We will keep the **two key domains fully separate**: Fabric MSP/HSM signing keys and TLS keys (Domain A) are **not derived from, do not wrap, and are not wrapped by** the app-level AES PII keys (Domain B). The two models **coexist** — Fabric anchors tamper-evident commitments while MySQL keeps PII plaintext unreadable. As a secure default: **never anchor a value on-chain using a key that also decrypts PII**, so a compromise of one domain cannot cascade into the other (Security-by-Design least privilege, E4).

## Alternatives considered
| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. Two independent key domains; no derivation/wrapping between them** | Compromise of one domain does not cascade; each key follows its own lifecycle; matches documented HSM/TLS split | Two key-management surfaces to operate | **Chosen** |
| B. Unify into one key model / single key hierarchy for Fabric signing and AES PII | Fewer key stores to manage | **Different purposes and lifecycles**: Fabric needs asymmetric signing keys and **file-based TLS keys** `[docs: hsm.md]`, the app uses versioned symmetric **AES-256-CBC** with a phased per-tenant rollout — no single lifecycle fits both; a shared key makes one compromise cascade across domains | Rejected because no single key model serves both purposes and it maximizes blast radius |
| C. Anchor on-chain commitments using the existing app AES PII key | Reuses an existing key | A signing-key compromise would then also leak PII; violates domain separation and least privilege | Rejected because it couples confidentiality to integrity keys |
| D. Fold Fabric key management into the existing `PII_ENCRYPTION_KEY[_Vn]` versioning | One versioning scheme | `PII_ENCRYPTION_KEY` is a symmetric data-at-rest key; Fabric needs asymmetric X.509 signing/TLS keys — categorically different primitives that cannot share a versioning scheme | Rejected because the primitives are incompatible |

## Consequences
- **Positive:** Blast radius is contained per domain; each key hierarchy rotates on its own schedule; the existing AES-at-rest rollout (G-21) is untouched by the Fabric adoption; aligns with the documented HSM-signing / file-based-TLS split.
- **Negative / trade-off:** Two distinct key-management surfaces to provision, monitor, and rotate (Fabric CA/HSM/CRL for Domain A; `PII_ENCRYPTION_KEY` versioning for Domain B); operators must not "simplify" by merging them. Expired Fabric TLS/signing certs halt anchoring — an operational risk independent of PII-key rotation.
- **Follow-ups:** Ratify against a real requirement when G-15 closes. Provision HSM/PKCS11 for MSP signing keys and file-based TLS keys per `[docs: hsm.md]` in the deployment design (G-13). Add "Fabric cert/key expiry halts anchoring" and "operator merges the two key domains" as risks in `10-risk/risk-register.md`.

## Related
- Relates to: ADR-0005 (the X.509 identities whose signing keys this ADR governs); the audit-overlay decision (G-09, MySQL stays system-of-record) and the no-plaintext-PII-on-chain decision (G-02) authored separately.
- Knowledge-graph: Layer D — **D6** (X.509 signing identity), **D13** (TLS in transit), **D14** (HSM key protection), **D15** (cert lifecycle); frames **E1** (CIA — Confidentiality & Integrity), **E4** (Security-by-Design — key separation, secure defaults). Grounding gap(s): **G-15** (key-model integration), with **G-21** (partial per-tenant AES rollout) and **G-13** (deployment/HSM provisioning). Context doc(s): `context/CRYPTOGRAPHY.md` §1–§4, `context/FABRIC-CA.md`.

# GO-SECURITY — EMS security controls & PII crypto (stub)

> **Read first:** `GO-CONVENTIONS.md` §4–5 (auth + authz). Language-agnostic guidance:
> `> See coding-standards-backend/references/auth-and-authz.md`. This stub anchors the **real,
> reusable PII crypto** the Fabric design must build on rather than reinvent (knowledge-graph Layer B;
> gaps G-02/G-15). Frames referenced (E1 CIA, E3 PbD, E4 SbD, E5 OWASP-API, E6 Zero-Trust) and
> capabilities (D2/D3) are defined in `11-execution/knowledge-graph.md`.

## PII encryption at rest — `pkg/db/encryption_plugins.go`
The **`EncryptionPluginV2`** GORM plugin transparently encrypts sensitive columns using MySQL-native
**`AES_ENCRYPT`/`AES_DECRYPT` (AES-256-CBC)** with `UNHEX(key)`/`UNHEX(iv)`
`[code: ems/pkg/db/encryption_plugins.go]`:
- Callbacks: `beforeCreate`/`beforeUpdate` encrypt into a shadow `*_encrypted` VARBINARY column;
  `beforeQuery` rewrites the SELECT to decrypt `[code: ems/pkg/db/encryption_plugins.go]`.
- **Versioned keys:** a `key_version` column + a `CASE WHEN key_version = n THEN AES_ENCRYPT(…)`
  expression lets multiple key versions coexist and rotate `[code: ems/pkg/db/encryption_plugins.go]`.
- **Phased rollout:** fallback `COALESCE(decrypt, plain)` → read-from-encrypted-only → write-only
  (NULL the plain column). Gated per company via `ResolveCompanyID` + `IsEncryptionEnabledForCompany`
  `[code: ems/pkg/db/encryption_plugins.go]`, `[code: ems/pkg/db/company_context.go]`.
- Fields are declared by struct tag: `encrypt:"nik_encrypted"` on `TblUser.Nik`; encrypted columns
  are `json:"-"` so ciphertext never serializes `[code: ems/internal/datasource/model/tbl_user.gen.go]`.
- Keys/IVs are env-configured and format-validated (`PII_ENCRYPTION_KEY` 64 hex, `_IV` 32 hex,
  `_V2`, `_ACTIVE_VERSION`, `_TOGGLE_ENABLED` disabled/all/specific-CIDs, `_READ_ENABLED`,
  `_WRITE_ONLY`) `[code: ems/app/appconf/config.go]`.
Encrypted fields today: `citizen_id`, `nik`, `mobile_phone`, `phone`, `npwp_16`
`[code: ems/internal/datasource/model/tbl_user.gen.go]`.

## Deterministic lookup hashing — `pkg/bankhasher`
Bank fields are stored as **PBKDF2-SHA256** hashes so they stay searchable without plaintext.
`Create` produces `"sha256:{iterations}:{base64_iv}:{base64_hash}"` over `userID:companyID:iv:value`,
byte-compatible with talenta-core's PHP `HashService::create`
`[code: ems/pkg/bankhasher/bankhasher.go]`. Secret + iterations from `BANK_HASH_SECRET_KEY` /
`BANK_HASH_ITERATIONS` `[code: ems/app/appconf/config.go]`.

## Display masking — `pkg/pii/mask.go`
`MaskNIK` (2 + stars + 2) and `MaskPhone` (prefix + `****` + last 3) define the presentation FORMAT,
mirroring talenta-core `MaskHelper.php` byte-for-byte; callers decide *when* to mask
`[code: ems/pkg/pii/mask.go]`, `[code: ems/pkg/pii/mask_test.go]`.

> **These three (AES-at-rest, PBKDF2 hashing, masking) are the existing crypto assets the Fabric
> design reuses, not reinvents.** Fabric anchors only salted commitments/hashes off-chain (D2/D3);
> authoritative PII stays in this AES-encrypted MySQL (gaps G-02/G-09). **[ASSUMPTION] (gap G-15):**
> Fabric MSP/HSM signing keys are separate from these app AES keys — `> See fabric-identity-security`.

## Access control (frames E5/E6)
- **AuthN:** gateway-header trust + `X-Api-Key` validation; consultant cross-tenant handling; scopes
  user/company/internal `[code: ems/internal/base/handler/base.go]`. (`GO-CONVENTIONS.md` §4.)
- **Object-level authz / BOLA:** `authz.Guard.ValidateResourceOwnership` masks cross-tenant and
  not-found as **404**, fails closed on unregistered resources
  `[code: ems/internal/base/authz/guard.go]`, `[code: ems/internal/base/authz/registry.go]`.
- **Role RBAC:** `CanAccessEmployeesData` / `CanEditEmployeesData` exclude Employee & Finance roles
  `[code: ems/internal/base/authz/employee_rbac.go]`.
- **Auth hardening:** shadow/enforce closes the `X-Scope`/`X-Company-ID`/ignore-role escalation
  paths; verified by the IDOR matrix `[code: ems/internal/base/handler/base.go]`,
  `[code: ems/qa-tests/security/launchpad-idor.http]`. Gateway trust end-to-end is gap **G-18**.

## Defense-in-depth extras
- **Log masking:** `LOG_MASK_FIELDS` → `*****` in request/response logs
  `[code: ems/app/api/middleware_logger.go]`; `CheckStatus` prints no secrets
  `[code: ems/internal/base/handler/check.go]`.
- **Non-root container:** runs as user `talenta` (uid 1001) `[code: ems/Dockerfile]`.
- **Public API kill-switch:** `FeaturePublicAPIExpose` (a hotfix gate) — never enable in prod
  `[code: ems/app/appconf/config.go]`.

## Cross-references
- Deeper security-audit lens & Fabric threat frames: `> See fabric-security-review`,
  `> See fabric-identity-security`. Error-status hygiene: `GO-ERROR-HANDLING.md`.
- For a security review of new Go code, use the `security-review` skill.

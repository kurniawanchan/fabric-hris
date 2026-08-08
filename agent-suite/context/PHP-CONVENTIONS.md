# talenta-core Conventions (PHP / Yii2)

> **Scope.** The coding conventions and the two security-critical patterns an agent must respect before
> touching talenta-core: the `EncryptableFieldsTrait` / `ENCRYPTED_COLUMNS` PII-at-rest mechanism, and
> the `AccessRole` RBAC model. Structural layout is in `PHP-ARCHITECTURE.md`; outward integration is in
> `PHP-INTEGRATION.md`.
>
> **Grounding.** `[code: talenta-core/<path>]` = read from `/Users/chan/www/talenta-core`. Anything
> marked **Recommendation:** is engineering judgement, not a documented rule.

## 1. Coding conventions (fact)

From the repo's own AI-assistant contract `[code: talenta-core/.github/copilot-instructions.md]` and
`[code: talenta-core/CLAUDE.md]`:

- **Naming**: classes `PascalCase` (`PayrollServices`, `UserRepository`); methods/vars `camelCase`;
  constants `SCREAMING_SNAKE_CASE`; DB tables `snake_case` with a `tbl_` prefix; file name matches class
  name `[code: talenta-core/.github/copilot-instructions.md:64-70]`.
- **PHP 7.4+**: always use parameter and return type hints; `?Type` for nullables; PSR-4 autoloading;
  PHPDoc on public methods `[code: talenta-core/.github/copilot-instructions.md:71-77]`.
- **Layering**: services extend `BaseServices` and implement an interface; data access goes through
  repositories with interface contracts; controllers stay thin; models use ActiveRecord relations, not
  raw SQL `[code: talenta-core/.github/copilot-instructions.md:78-83]`.
- **Commits**: `<type>(<scope>): (<jira-ticket>): Short description [FULL_COPILOT]`; never add an
  AI co-author signature `[code: talenta-core/CLAUDE.md:30-31]`.
- **Prose/comments**: no em dash (use a plain `-`); comments only for non-obvious logic, max ~3 lines
  per block `[code: talenta-core/CLAUDE.md:26,38]`.
- **Remove dev artifacts**: `var_dump`/`dd`/`console.log`, stray TODOs, commented-out code
  `[code: talenta-core/.github/copilot-instructions.md:158-164]`.

**Recommendation:** these are the same rules a review agent should enforce on any Fabric-adjacent PHP
change. For general server-side rubric beyond this repo, defer to the `coding-standards-backend` skill.

## 2. ActiveRecord patterns (fact)

- `tableName()` returns the physical table, e.g. `tbl_user`, `tbl_family_data`
  `[code: talenta-core/models/User.php:825-828]`, `[code: talenta-core/models/FamilyData.php:71-74]`.
- **Scenario-scoped validation**: `rules()` gate validators with `'on' => '<scenario>'`, so the same
  model validates differently for API vs mass-import vs normal input
  `[code: talenta-core/models/FamilyData.php:82-123]`, `[code: talenta-core/models/User.php:847-859]`.
- **Lifecycle hooks**: models normalize and sanitize input in `beforeSave()` (e.g. `filter_var` +
  `strip_tags` on name/address fields; digit-only normalization of `npwp_16`)
  `[code: talenta-core/models/User.php:2568-2599]`, and `__set()` strips tags on a defined attribute
  allowlist `[code: talenta-core/models/User.php:833-841]`.
- **Feature toggles** gate behaviour per tenant: `Yii::$app->featureToggle->unleash('<flag>', $companyId)`
  / `->enabled('<flag>', $companyId)` `[code: talenta-core/models/FamilyData.php:159]`,
  `[code: talenta-core/models/User.php:2584]`. Toggles are `company_id`-scoped - never assume a flag is
  globally on.

## 3. PII encryption: `EncryptableFieldsTrait` + `ENCRYPTED_COLUMNS` (fact)

This is the reused crypto-at-rest asset the Fabric design must not reinvent (knowledge-graph Layer B,
capability D2/D3; gap **G-15** keeps Fabric keys separate from these app keys).

**Opt-in per model.** A model `use`s the trait and declares an `ENCRYPTED_COLUMNS` map of
`plain => shadow` columns. On `User`:

```php
const ENCRYPTED_COLUMNS = [
    'nik' => 'nik_encrypted', 'citizen_id' => 'citizen_id_encrypted',
    'mobile_phone' => 'mobile_phone_encrypted', 'phone' => 'phone_encrypted',
    'npwp_16' => 'npwp_16_encrypted',
];
```

`[code: talenta-core/models/User.php:41,595-601]`. A companion `WRITE_ONLY_DEFAULTS` supplies placeholder
values for NOT-NULL plain columns in the write-only phase `[code: talenta-core/models/User.php:604-610]`.

**How it works** `[code: talenta-core/traits/EncryptableFieldsTrait.php]`:

- **Encrypt on save**: the model calls `$this->encryptFields(self::tableName(), $insert)` inside
  `beforeSave()` `[code: talenta-core/models/User.php:2601-2602]`. The trait builds MySQL
  `AES_ENCRYPT(...)` `Expression`s with an IV, binding key/IV via PDO, and stamps a `key_version`
  `[code: talenta-core/traits/EncryptableFieldsTrait.php:196-259]`.
- **Decrypt on read**: `find()` is overridden to inject `AES_DECRYPT` into the SELECT so encrypted fields
  decrypt at the DB layer; `__get()` provides a per-row fallback decrypt for records not loaded through
  `find()` `[code: talenta-core/traits/EncryptableFieldsTrait.php:550-566,381-436]`.
  `findWithoutDecryption()` opts out for performance `[code: talenta-core/traits/EncryptableFieldsTrait.php:644-647]`.
- **Key-leak guard**: `afterSave()` scrubs any leftover `Expression` object off the encrypted attribute,
  because a serialized `Expression` would expose the raw key in an HTTP/Kafka/queue payload
  `[code: talenta-core/traits/EncryptableFieldsTrait.php:261-326]`. **A model overriding `afterSave()`
  must call `scrubEncryptedExpressionAttributes()` itself** - trait methods lose to class methods in PHP.
- **Phased, tenant-scoped rollout** via `EncryptionServices` toggles checked against `company_id`:
  encryption-enabled -> read-from-encrypted -> write-only (plain columns nulled). When a company is not
  yet enabled, reads fall back to `COALESCE(decrypt, plain_column)`
  `[code: talenta-core/traits/EncryptableFieldsTrait.php:196-259,583-636]`. Key-rotation is supported by
  re-encrypting existing rows under their original `key_version`.

**Consequence for the Fabric design (Recommendation, tied to gaps G-02 / G-21):** anchor logic must read
PII **through the model/app layer**, never off raw `*_encrypted` columns, because for non-pilot tenants
the encrypted columns may be empty and plaintext is still authoritative
`[code: talenta-core/traits/EncryptableFieldsTrait.php:583-636]`. Salting/commitment work belongs
off-chain per capability D3 - see `fabric-chaincode-dev/references/private-data.md`.

## 4. AccessRole RBAC (fact)

**Coarse role constants** on `User` `[code: talenta-core/models/User.php:91-110]`:

| Const | Value | Actor |
|---|---|---|
| `ROLE_SUPER` | 1 | Super admin |
| `ROLE_EMPLOYEE` | 2 | Employee (self-service) |
| `ROLE_ADMIN` | 3 | HR admin |
| `ROLE_CONSULT` | 4 | Consultant (multi-company) |
| `ROLE_FINANCE` | 5 | Finance |
| `ROLE_ADMIN_CONSULT` | 6 | Consultant admin |

**Fine-grained permissions** are resolved from access-role tables, not from the coarse role alone.
`setRuleFromRoleId()` loads the `AccessRole` for a user's `role_id`, then hydrates per-module
view/edit/request flags (`employee_view`/`employee_edit`, `payroll_view`/`payroll_edit`,
`attendance_*`, `approval_list_*`, `mpp_*`) from `AccessRoleEss` / `AccessRoleUser`
`[code: talenta-core/models/User.php:2073-2112]`. The RBAC entity itself is `tbl_access_role`
`[code: talenta-core/models/AccessRole.php:18-21]`, with `ROLE_TYPE_DEFAULT` / `ROLE_TYPE_CUSTOM`
distinguishing built-in from customer-defined roles `[code: talenta-core/models/AccessRole.php:9-16]`.

**Consultant cross-tenant switch.** `findIdentity()` reads `selected_company_id` from the session; if a
matching `Consultant` row exists, it swaps the user's `company_id` and re-derives the role
(super-admin -> role 4, else consultant-admin -> role 6), otherwise it revokes access (`role = -1`)
`[code: talenta-core/models/User.php:1204-1252]`. This is a privileged cross-tenant actor
(knowledge-graph Layer A) and gap **G-18** flags the header-trust boundary that backs it.

**Mapping to Fabric (Recommendation, gap G-11):** these HRIS roles and per-module flags are the source
truth that a chaincode ABAC / MSP-OU mapping must mirror (capability D6/D9). Do not invent a parallel
role model; project this one. Identity mechanics live in
`fabric-identity-security/references/msp-structure.md` and `.../policies-and-acl.md`.

## 5. Cross-refs

- Layout, components, Vue seam: `PHP-ARCHITECTURE.md`.
- Kafka producer, approval hook, public API, SSO: `PHP-INTEGRATION.md`.
- PII inventory, crypto assets, capability IDs, gap register: `../11-execution/knowledge-graph.md`,
  `../11-execution/grounding-gaps.md`.

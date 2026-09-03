INTERNAL

# integration-bridge Bruno collection

Executable API documentation for `integration-bridge`, the off-chain REST
bridge in front of the Fabric write paths (`write-path-integration/`) and
chaincode (`fabric-network/chaincode/employeeprofilerecord/`). This is the
only HTTP surface in this repository — `write-path-integration`'s
`gateway-client` and the chaincode itself are gRPC/Fabric-Gateway-internal
and are not separately documented here (see "Scope" below).

Open this folder as a collection in Bruno (`qa-tests/api/integration-bridge/`),
select the `local` environment, and run requests against a live bridge
instance (see `docs/QUICKSTART.md` for bringing up the network + bridge).

## Scope

Documented here (7 endpoints):

- `POST /v1/profile-sections/PERSONAL`
- `POST /v1/profile-sections/EMPLOYMENT`
- `POST /v1/profile-sections/EDUCATION`
- `POST /v1/profile-sections/ADDITIONAL`
- `POST /v1/profile-sections/PAYROLL`
- `GET  /v1/profile-sections/history`
- `POST /v1/profile-sections/verify`

Not documented here, by design:

- **`write-path-integration/gateway-client`** — a Go library wrapping the
  Fabric Gateway SDK, not an HTTP API. It has no independent transport
  surface; it is what `integration-bridge` calls internally.
- **Chaincode transactions** (`RecordProfileSection`, `GetProfileSectionRecord`,
  `GetProfileHistory`, `GetEmployeeProfileSummary`) — invoked over the
  Fabric Gateway gRPC protocol, not HTTP, and only ever from inside
  `gateway-client`/`writepaths`. `integration-bridge`'s `history` and
  `verify` routes are the HTTP-reachable surface over `GetProfileHistory`;
  the write routes are the HTTP-reachable surface over
  `RecordProfileSection`. A gRPC-native reference (e.g. a `.proto`-driven
  doc) is a reasonable follow-up but out of scope for a Bruno (HTTP)
  collection.

## Authentication

Every route requires two headers, checked with a constant-time comparison:

| Header | Source |
|---|---|
| `X-Api-Key` | `BRIDGE_API_KEY` env var on the bridge process |
| `X-Company-ID` | `BRIDGE_COMPANY_ID` env var — one bridge deployment serves exactly one tenant |

A missing or incorrect value on either header produces the identical error
message, by design (`integration-bridge/internal/pipeline/auth.go`) — this
prevents a caller from using the response to distinguish "wrong API key"
from "right key, wrong company ID."

Set `apiKey` and `companyId` in the `local` environment (`environments/local.bru`)
before running any request. **Never commit a real API key into this
collection or its environment files** — `apiKey` ships as a placeholder
value; treat any real value that ends up in one of these files as a
credential exposure requiring rotation (org data-handling rule 2).

## Response envelope (write routes)

`PERSONAL` / `EMPLOYMENT` / `EDUCATION` / `ADDITIONAL` / `PAYROLL` share one
response shape (AD-3):

```json
{ "status": "committed", "recordID": "...", "detail": "" }
```

| status | HTTP | Meaning |
|---|---|---|
| `committed` | 200 | Anchored successfully; `recordID` is populated. |
| `partial_failure` | 200 | Dispatch timed out or the write path returned a partial-failure error — outcome uncertain, not necessarily failed. Not the same as `error`. |
| `rejected` | 400 | Request failed `[validate]` (missing field, bad base64, non-object `newValue`) or an unknown route pattern. |
| `error` | 500 (default) | `[auth]` failure or any other bridge/dependency fault. Note: `error` deliberately collapses two different origins (auth vs. dependency fault) — see `agent-suite/11-execution/grounding-gaps.md` G-33 for why, and don't infer a 401 from an `error` status; the bridge always returns 500 for it. |

`detail` never echoes request body content (`newValue`/`document` bytes are
never reflected back) — every constructed error message uses fixed strings
and pseudonymous identifiers only.

`history` and `verify` use their own, narrower status vocabularies — see
`history.bru` and `verify.bru`.

## Config reference (for standing up a bridge instance to test against)

Every value below is a required env var on the bridge process
(`integration-bridge/cmd/integrationbridge/config.go`); the process fails
fast, naming every unset one, if any are missing:

`BRIDGE_PEER_ENDPOINT`, `BRIDGE_TLS_SERVER_NAME`, `BRIDGE_TLS_CA_CERT_PATH`,
`BRIDGE_CLIENT_TLS_CERT_PATH`, `BRIDGE_CLIENT_TLS_KEY_PATH`, `BRIDGE_MSP_ID`,
`BRIDGE_SIGN_CERT_PATH`, `BRIDGE_SIGN_KEY_PATH`, `BRIDGE_CHANNEL_NAME`,
`BRIDGE_CHAINCODE_NAME`, `BRIDGE_TENANT_ID`, `BRIDGE_IPFS_PRIMARY_API`,
`BRIDGE_IPFS_REPLICA_API`, `BRIDGE_API_KEY`, `BRIDGE_COMPANY_ID`,
`BRIDGE_KEYSTORE_DIR`, `BRIDGE_KEYSTORE_ENCRYPTION_KEY_HEX`.

Optional, with defaults: `BRIDGE_HTTP_ADDR` (default `:8080`),
`BRIDGE_DISPATCH_TIMEOUT` (default `30s`, unverified/placeholder per the
source's own comment — no ratified Caliper-derived number exists yet).

`BRIDGE_KEYSTORE_ENCRYPTION_KEY_HEX` and `BRIDGE_API_KEY` are secret
material — never place real values for these in this collection, its
environment files, or any request example.

## Confidentiality note

This collection lives under `qa-tests/`, one of this repo's hard-clean
zones (see root `CLAUDE.md`) — no real company/product name, and no PII, in
any request body, example, or comment here. All example values above are
placeholders (`tenant01`, `EMP-0001`, `REDACTED-FOR-DOCS-EXAMPLE`, etc.).

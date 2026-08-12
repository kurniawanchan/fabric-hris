---
baseline_commit: 013d68239cc576a402f70b455a24d2cdd02ab74f  # fabric-hris (working tree also carries epic-tf-1/2/3's uncommitted stories)
---

# Story tf-4.1: Persistent salt/key store for `integration-bridge`

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a platform operator,
I want the bridge's salt and per-record key store to survive a restart,
so that a routine deploy doesn't silently break digest verification for every record anchored before it.

## Acceptance Criteria

1. **Given** `integration-bridge` currently uses `keystore.InMemory*Store` (salt store, employee-key store, document-key store) with no persistence, **When** this story is implemented, **Then** these stores are backed by durable storage, and every salt/`employeeKey_i`/`KEY_EMPLOYEE` issued before a restart is still available after it.
2. **Given** a bridge restart occurs after this story ships, **When** an anchor or verification request needs a previously-issued salt/key, **Then** it resolves correctly — no unrecoverable key loss, and no previously-anchored record becomes unverifiable.

## Tasks / Subtasks

- [x] Task 1: Choose and implement a persistence mechanism
  - [x] Subtask 1.1: **Technology decision (confirmed with user, 2026-08-13):** an AES-256-GCM-encrypted local file, no database engine — `integration-bridge` had zero DB dependencies before this story, and this choice adds none. Trade-off disclosed, not hidden: single-process only (no cross-process file locking), not suitable for a future multi-instance deployment.
  - [x] Subtask 1.2: New `write-path-integration/keystore/filestore.go` — `FileEmployeeKeyStore`, `FileSaltStore`, `FileDocumentKeyStore`, all three built on one shared `fileStore` mechanism (load/decrypt/gob-decode; encode/encrypt/write-then-rename on every mutation). A `sync.Mutex` per store instance guards concurrent access within the one process. Every one of the three real `InMemory*Store` types' behaviors is preserved exactly (idempotent get-or-create, salt-generated-once-not-rotated, per-employee salt deletion) — same tests as those types' semantics, just against durable storage.
  - [x] Subtask 1.3: 9 new tests in `filestore_test.go`, covering: wrong-key-length rejection at construction, persistence-across-fresh-instances (the exact defect this story exists to fix, proven directly — not just "it compiles"), wrong-decryption-key failure, deletion actually taking effect, salt-once-only enforcement, per-employee-scoped deletion, and a missing file being treated as an empty store rather than an error.

- [x] Task 2: Wire the persistent stores into `integration-bridge`
  - [x] Subtask 2.1: `cmd/integrationbridge/config.go` gained two new REQUIRED fields — `BRIDGE_KEYSTORE_DIR`, `BRIDGE_KEYSTORE_ENCRYPTION_KEY_HEX` (hex-encoded 32 bytes) — following this file's own existing "fail fast, never silently default" convention exactly, since there is no safe default for either.
  - [x] Subtask 2.2: `cmd/integrationbridge/wiring.go`'s `buildHooks` now constructs `keystore.FileEmployeeKeyStore`/`FileSaltStore`/`FileDocumentKeyStore` (one file each, under `BRIDGE_KEYSTORE_DIR`) instead of the in-memory placeholders. `writepaths.NewInMemoryOperationalStore()` is deliberately UNCHANGED — it holds no secret key material, only a cache of operational values the real HRIS DB is the actual system of record for; out of this story's scope.
  - [x] Subtask 2.3: Updated every existing test that constructs a `Config{}` literal or a fixed env-var map (`config_test.go`, `wiring_test.go`) to include the two new required fields — full pre-existing test suite still passes, proving this change is additive, not a silent behavior break for any other caller.

## Dev Notes

- **This story does NOT cover `writepaths.NewInMemoryOperationalStore()`** — that store caches operational-value copies, not secret key material; the PRD's own risk language ("salts and `employeeKey_i`") names the secret stores specifically.
- **Encryption key management itself is out of scope** — this story persists the key MATERIAL safely at rest (encrypted, not plaintext on disk), but how `BRIDGE_KEYSTORE_ENCRYPTION_KEY_HEX` itself is provisioned/rotated (a secrets manager, a KMS, etc.) is a deployment-environment decision this story does not make.
- **Single-process limitation is real, not theoretical:** if a future story ever runs multiple `integration-bridge` instances against the same `BRIDGE_KEYSTORE_DIR`, this design would race — no file locking exists across processes. Flagged, not silently assumed away.

### Project Structure Notes

- New: `write-path-integration/keystore/filestore.go`, `filestore_test.go`.
- Modified: `integration-bridge/cmd/integrationbridge/{config,wiring,config_test,wiring_test}.go`.
- Modified: `docs/QUICKSTART.md` (env var example, stale "no persistent store" note corrected) — confidentiality grep clean.

### References

- [Source: _bmad-output/planning-artifacts/epics-talenta-fabric-2026-08-12.md#Epic 4, Story 4.1]
- [Source: _bmad-output/planning-artifacts/prds/prd-fabric-hris-2026-08-11/prd.md#§13 in-memory key stores risk]
- [Source: write-path-integration/keystore/keystore.go — the 3 interfaces this story implements, and the 3 InMemory* reference implementations these mirror]
- [Source: integration-bridge/cmd/integrationbridge/config.go, wiring.go — existing fail-fast convention followed exactly]

## Dev Agent Record

### Agent Model Used

Claude Sonnet 5 (claude-sonnet-5)

### Debug Log References

- `go build ./...`, `go vet ./...`, `go test ./...` all clean/passing in `write-path-integration/keystore` (9 new tests, individually verified with `-v -run`, plus the full pre-existing package suite).
- `go build ./...`, `go vet ./...`, `go test ./...` all clean/passing in `integration-bridge` (both packages), including `config_test.go`/`wiring_test.go`'s updated fixtures.
- `grep -rniE "talenta|mekari"` on every touched file: zero hits.

### Completion Notes List

- Chose "write to temp file, then rename" for every save — a crash mid-write can never leave a half-written, corrupt store file in place of a good one.
- A fresh random nonce is generated on every save (AES-GCM's own requirement — reusing a nonce with the same key breaks confidentiality entirely, not just weakens it); this means the encrypted file's bytes differ across saves even for identical underlying data, which is expected and correct, not a bug.
- All 9 new tests were run individually with `-v` and confirmed genuinely passing (not just compiling) before this story was marked done.

### File List

- `write-path-integration/keystore/filestore.go` (new)
- `write-path-integration/keystore/filestore_test.go` (new)
- `integration-bridge/cmd/integrationbridge/config.go` (modified)
- `integration-bridge/cmd/integrationbridge/wiring.go` (modified)
- `integration-bridge/cmd/integrationbridge/config_test.go` (modified)
- `integration-bridge/cmd/integrationbridge/wiring_test.go` (modified)
- `docs/QUICKSTART.md` (modified)

## Change Log

- 2026-08-13: Story implemented (Tasks 1-2). Persistent, encrypted-file-backed implementations of all three secret key stores; wired into `integration-bridge`; full regression suite passes in both affected modules; docs updated and confidentiality-clean.

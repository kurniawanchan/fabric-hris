// Package keystore fixes the INTERFACE SHAPE for REC-2's two off-chain
// secret stores (ADR-0011/ADR-0019/ADR-0021, data-model.md §7) and the
// non-mixing invariant between them. Custody, rotation, and backup POLICY
// for these stores is security-architect's to fix (ADR-0019's own
// Follow-ups) — consumed here, not designed. The in-memory implementations
// below exist to prove the interface shape is exercisable and testable;
// they are NOT the production store (no persistence, no encryption-at-rest,
// no backup discipline — all explicitly out of this item's scope).
//
// Non-mixing invariant (ADR-0009, restated by ADR-0019 across all four key
// domains): neither store below may ever derive its output from, or be
// derived from, Domain A (Fabric MSP/TLS signing keys) or Domain B (the
// application's AES-at-rest PII key). Both stores here generate key
// material via crypto/rand exclusively — never via HKDF/HMAC over anything
// sourced from another domain — enforced by construction, not by a runtime
// check (there is nothing in this package's own scope to check against;
// the discipline is "never import or reference Domain A/B material here").
package keystore

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
)

// MinKeyBytes/MinSaltBytes: the ADR-0011 floor (>=128 bits) for both the
// per-record salt and the per-employee key.
const (
	MinSaltBytes = 16
	MinKeyBytes  = 16
)

// recordKey identifies one record's salt slot: one chain per
// (employeeID, profileSection), one salt per version within that chain
// (data-model.md §5/§7 — the salt is per-RECORD, not per-employee).
type recordKey struct {
	employeeID     string
	profileSection string
	version        int
}

// SaltStore persists the per-record CSPRNG content salt (ADR-0011 §1) — a
// fresh, unique value per write, never derived from the section's own
// content (that would defeat the point of salting against low-entropy
// data, errata E-1's original defect class).
type SaltStore interface {
	// PutSalt generates and persists a FRESH salt for this exact record
	// version. Fails if a salt already exists for this key — a record's
	// salt is generated once, at first write of that version, never
	// rotated in place.
	PutSalt(ctx context.Context, employeeID, profileSection string, version int) ([]byte, error)
	// GetSalt returns the previously-stored salt for a record version —
	// needed by any party (writer's own resubmit path, or a verifier) that
	// must recompute DataHash against the same input.
	GetSalt(ctx context.Context, employeeID, profileSection string, version int) ([]byte, error)
	// DeleteAllSaltsForEmployee destroys EVERY salt this employee has, across
	// every section and version — REC-6's crypto-shred trigger. Irreversible.
	// Added for REC-6; not exercised by REC-2's own original scope, which
	// only needed per-record Put/Get.
	DeleteAllSaltsForEmployee(ctx context.Context, employeeID string) error
}

// DocumentKeyStore persists each employee's KEY_EMPLOYEE (ADR-0019 Domain
// B′) — encrypts IPFS-hosted supporting documents (REC-5). A separate
// secret from employeeKey_i (Domain C, EmployeeKeyStore below): a
// compromised document key must not open on-chain identifier
// pseudonymization, and vice versa (ADR-0019's founding cross-domain rule).
// Interface-only here (REC-6 needs to be ABLE to destroy this key; REC-5's
// own document-encryption use of it is a separate, not-yet-built item).
type DocumentKeyStore interface {
	GetOrCreateDocumentKey(ctx context.Context, employeeInternalID string) ([]byte, error)
	DeleteDocumentKey(ctx context.Context, employeeInternalID string) error
}

// EmployeeKeyStore persists each employee's independently-random
// employeeKey_i (ADR-0021 — no pseudonymKey/master key; generated once,
// off-chain, per employee, never re-derived).
type EmployeeKeyStore interface {
	// GetOrCreateEmployeeKey returns the employee's existing key if one
	// exists, or generates and persists a fresh, independently-random one
	// if this is the employee's first anchored write ever (any section).
	GetOrCreateEmployeeKey(ctx context.Context, employeeInternalID string) ([]byte, error)
	// DeleteEmployeeKey destroys an employee's key — the erasure mechanism
	// for on-chain pseudonym unlinkability (ADR-0015 steps 3-4). This is
	// intentionally irreversible: there is no "undelete."
	DeleteEmployeeKey(ctx context.Context, employeeInternalID string) error
}

// InMemorySaltStore is a reference SaltStore — proves the interface shape,
// not a production store (no persistence across process restarts, no
// encryption at rest). Safe for concurrent use.
type InMemorySaltStore struct {
	mu    sync.Mutex
	salts map[recordKey][]byte
}

func NewInMemorySaltStore() *InMemorySaltStore {
	return &InMemorySaltStore{salts: make(map[recordKey][]byte)}
}

func (s *InMemorySaltStore) PutSalt(_ context.Context, employeeID, profileSection string, version int) ([]byte, error) {
	key := recordKey{employeeID, profileSection, version}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.salts[key]; exists {
		return nil, fmt.Errorf("keystore: salt already exists for %s/%s v%d — a record's salt is generated once, not rotated in place", employeeID, profileSection, version)
	}
	salt := make([]byte, MinSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("keystore: generating salt: %w", err)
	}
	s.salts[key] = salt
	return salt, nil
}

func (s *InMemorySaltStore) GetSalt(_ context.Context, employeeID, profileSection string, version int) ([]byte, error) {
	key := recordKey{employeeID, profileSection, version}
	s.mu.Lock()
	defer s.mu.Unlock()
	salt, exists := s.salts[key]
	if !exists {
		return nil, fmt.Errorf("keystore: no salt stored for %s/%s v%d", employeeID, profileSection, version)
	}
	return salt, nil
}

func (s *InMemorySaltStore) DeleteAllSaltsForEmployee(_ context.Context, employeeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key := range s.salts {
		if key.employeeID == employeeID {
			delete(s.salts, key)
		}
	}
	return nil
}

// InMemoryDocumentKeyStore is a reference DocumentKeyStore — same caveats
// as the other in-memory stores in this file (not persistent, not
// production-grade custody).
type InMemoryDocumentKeyStore struct {
	mu   sync.Mutex
	keys map[string][]byte
}

func NewInMemoryDocumentKeyStore() *InMemoryDocumentKeyStore {
	return &InMemoryDocumentKeyStore{keys: make(map[string][]byte)}
}

func (s *InMemoryDocumentKeyStore) GetOrCreateDocumentKey(_ context.Context, employeeInternalID string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key, exists := s.keys[employeeInternalID]; exists {
		return key, nil
	}
	key := make([]byte, MinKeyBytes)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("keystore: generating KEY_EMPLOYEE: %w", err)
	}
	s.keys[employeeInternalID] = key
	return key, nil
}

func (s *InMemoryDocumentKeyStore) DeleteDocumentKey(_ context.Context, employeeInternalID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.keys, employeeInternalID)
	return nil
}

// InMemoryEmployeeKeyStore is a reference EmployeeKeyStore — same caveats
// as InMemorySaltStore above.
type InMemoryEmployeeKeyStore struct {
	mu   sync.Mutex
	keys map[string][]byte
}

func NewInMemoryEmployeeKeyStore() *InMemoryEmployeeKeyStore {
	return &InMemoryEmployeeKeyStore{keys: make(map[string][]byte)}
}

func (s *InMemoryEmployeeKeyStore) GetOrCreateEmployeeKey(_ context.Context, employeeInternalID string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key, exists := s.keys[employeeInternalID]; exists {
		return key, nil
	}
	key := make([]byte, MinKeyBytes)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("keystore: generating employeeKey_i: %w", err)
	}
	s.keys[employeeInternalID] = key
	return key, nil
}

func (s *InMemoryEmployeeKeyStore) DeleteEmployeeKey(_ context.Context, employeeInternalID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.keys, employeeInternalID)
	return nil
}

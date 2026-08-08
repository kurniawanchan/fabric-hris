// Package gatewayclient computes the off-chain values a write path must
// have ready before calling RecordProfileSection (REC-1: digest & identifier
// builder). Every function here runs BEFORE any chaincode call — none of
// its inputs (profile-section plaintext, salt, employeeKey_i) may ever
// leave this process as a chaincode argument (ADR-0020's confidentiality
// boundary; enforced on the chaincode side by CC-2/CC-3's own argument
// shape, not repeated here).
//
// Construction (ADR-0011, identifier derivation amended by ADR-0021):
//
//	DataHash   = SHA-256(salt ‖ JCS(section))
//	EmployeeID = HMAC-SHA256(employeeKey_i, "id")
//	UpdatedBy  = HMAC-SHA256(employeeKey_i, "actor" ‖ user_id)
//
// employeeKey_i is an independently-random, per-employee secret (ADR-0021 —
// there is no pseudonymKey/master key anywhere in this package; each
// employee's key is generated once, off-chain, and never derived from a
// tenant-wide root). Sourcing/storing employeeKey_i and the per-record salt
// is REC-2's scope, not this file's — callers pass both in already.
package gatewayclient

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
)

// CanonicalizationVersion and HashAlgo are the versioned interface tags
// ADR-0020 requires be recorded per write, so a future canonicalizer/digest
// change never silently invalidates already-anchored records' verifiability
// (data-model.md §2.1/§3).
const (
	CanonicalizationVersion = "JCS-RFC8785-v1"
	HashAlgo                = "SHA-256"
)

// MinSaltBytes is the ADR-0011 floor (>=128 bits) for the per-record content
// salt. Generating/storing the salt is REC-2's scope — this is enforced
// here only as an input-shape guard against an obviously-too-short salt
// being passed in by mistake.
const MinSaltBytes = 16

var (
	// ErrSaltTooShort signals a salt below ADR-0011's >=128-bit floor.
	ErrSaltTooShort = errors.New("gatewayclient: salt must be at least 128 bits (16 bytes)")
	// ErrEmptyEmployeeKey signals a missing employeeKey_i — this is always
	// a caller bug (REC-2 must have already provisioned the key before any
	// write for that employee can be built), never a runtime user input.
	ErrEmptyEmployeeKey = errors.New("gatewayclient: employeeKey_i must not be empty")
)

// ComputeDataHash canonicalizes sectionJSON per RFC 8785 (the pinned
// library, PB-3/G-24) and returns "sha256:<hex>" over salt‖JCS(section) —
// matching data-model.md §2's on-chain DataHash field format exactly.
// sectionJSON must already be the section's COMPLETE current value, not a
// delta (REC-1's own DoD) — this function does not itself enforce that; the
// write-path hook (REC-4) is responsible for passing the full section.
func ComputeDataHash(salt []byte, sectionJSON []byte) (string, error) {
	if len(salt) < MinSaltBytes {
		return "", ErrSaltTooShort
	}
	canonical, err := jsoncanonicalizer.Transform(sectionJSON)
	if err != nil {
		return "", fmt.Errorf("gatewayclient: canonicalizing section: %w", err)
	}
	h := sha256.New()
	h.Write(salt)
	h.Write(canonical)
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// ComputeEmployeeID derives the pseudonymous, deterministic on-chain
// EmployeeID from employeeKey_i. Deterministic by construction (ADR-0011's
// Option E rejection: identifiers need a key, not a salt, so the same
// employee always maps to the same EmployeeID across every write).
func ComputeEmployeeID(employeeKeyI []byte) (string, error) {
	if len(employeeKeyI) == 0 {
		return "", ErrEmptyEmployeeKey
	}
	mac := hmac.New(sha256.New, employeeKeyI)
	mac.Write([]byte("id"))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// ComputeUpdatedBy derives the pseudonymous actor identifier for one write.
// userID is the human actor's real internal user ID (HR admin, employee, or
// system user who made the change) — never the platform's own submitting
// Fabric client identity (data-model.md §2.2 draws that distinction; the
// submitting identity is checked separately, via CID, on the chaincode
// side — CC-4's scope, not this function's).
func ComputeUpdatedBy(employeeKeyI []byte, userID string) (string, error) {
	if len(employeeKeyI) == 0 {
		return "", ErrEmptyEmployeeKey
	}
	mac := hmac.New(sha256.New, employeeKeyI)
	mac.Write([]byte("actor" + userID))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

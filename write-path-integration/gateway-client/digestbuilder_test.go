package gatewayclient

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
)

func testSalt() []byte {
	s := make([]byte, MinSaltBytes)
	for i := range s {
		s[i] = byte(i + 1)
	}
	return s
}

func testEmployeeKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(0xA0 + i)
	}
	return k
}

// TestComputeDataHash_Deterministic — U-1/U-6 style: the same salt+section
// must always produce the same DataHash, and the "sha256:" prefix must
// match data-model.md §2's on-chain field format exactly.
func TestComputeDataHash_Deterministic(t *testing.T) {
	salt := testSalt()
	section := []byte(`{"fullName":"Test Employee","department":"Engineering"}`)

	got1, err := ComputeDataHash(salt, section)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got2, err := ComputeDataHash(salt, section)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got1 != got2 {
		t.Fatalf("DataHash not deterministic: %q vs %q", got1, got2)
	}
	if len(got1) != len("sha256:")+64 || got1[:7] != "sha256:" {
		t.Fatalf("unexpected DataHash format: %q", got1)
	}
}

// TestComputeDataHash_KeyOrderIndependent — RFC 8785's whole point: two
// JSON objects with the SAME keys/values in a DIFFERENT source order must
// canonicalize to the identical byte sequence, hence the identical hash.
// This is the actual property REC-1 needs from JCS — not just "some
// canonicalizer ran" but "key order in the writer's JSON does not matter."
func TestComputeDataHash_KeyOrderIndependent(t *testing.T) {
	salt := testSalt()
	a := []byte(`{"fullName":"Test Employee","department":"Engineering"}`)
	b := []byte(`{"department":"Engineering","fullName":"Test Employee"}`)

	hashA, err := ComputeDataHash(salt, a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hashB, err := ComputeDataHash(salt, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hashA != hashB {
		t.Fatalf("key-order dependence leaked through: %q vs %q", hashA, hashB)
	}
}

// TestComputeDataHash_DifferentSaltDifferentHash — proves the salt is
// actually mixed in (not silently ignored), the property that defeats the
// dictionary-attack class of defect this design replaced (errata E-1).
func TestComputeDataHash_DifferentSaltDifferentHash(t *testing.T) {
	section := []byte(`{"fullName":"Test Employee"}`)
	salt1 := testSalt()
	salt2 := append([]byte{}, testSalt()...)
	salt2[0] ^= 0xFF

	hash1, _ := ComputeDataHash(salt1, section)
	hash2, _ := ComputeDataHash(salt2, section)
	if hash1 == hash2 {
		t.Fatal("different salts produced the same DataHash — salt is not being mixed in")
	}
}

// TestComputeDataHash_RejectsShortSalt — ADR-0011's >=128-bit floor.
func TestComputeDataHash_RejectsShortSalt(t *testing.T) {
	shortSalt := make([]byte, MinSaltBytes-1)
	_, err := ComputeDataHash(shortSalt, []byte(`{}`))
	if err != ErrSaltTooShort {
		t.Fatalf("expected ErrSaltTooShort, got %v", err)
	}
}

// TestComputeDataHash_MatchesManualConstruction — pins the EXACT byte
// layout (salt directly prepended to the canonicalized JSON, no separator,
// no length prefix) against a hand-computed SHA-256, so a future refactor
// that changes concatenation order/adds a separator fails loudly here
// instead of silently producing verifier-incompatible hashes.
func TestComputeDataHash_MatchesManualConstruction(t *testing.T) {
	salt := testSalt()
	section := []byte(`{"z":1,"a":2}`)   // deliberately out-of-order keys
	canonical := []byte(`{"a":2,"z":1}`) // RFC 8785: lexicographic key order

	got, err := ComputeDataHash(salt, section)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	h := sha256.New()
	h.Write(salt)
	h.Write(canonical)
	want := "sha256:" + hex.EncodeToString(h.Sum(nil))

	if got != want {
		t.Fatalf("DataHash construction drifted from ADR-0011's byte layout:\n got  %q\n want %q", got, want)
	}
}

// TestComputeEmployeeID_DeterministicAndKeyed — same employeeKey_i always
// yields the same EmployeeID (needed for the (EmployeeID, ProfileSection)
// lookup key and PrevHash chain, data-model.md §5); different keys must
// yield different IDs (or the pseudonym isn't doing its job).
func TestComputeEmployeeID_DeterministicAndKeyed(t *testing.T) {
	key1 := testEmployeeKey()
	key2 := append([]byte{}, testEmployeeKey()...)
	key2[0] ^= 0xFF

	id1a, err := ComputeEmployeeID(key1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	id1b, err := ComputeEmployeeID(key1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id1a != id1b {
		t.Fatalf("EmployeeID not deterministic for the same key: %q vs %q", id1a, id1b)
	}

	id2, err := ComputeEmployeeID(key2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id1a == id2 {
		t.Fatal("different employeeKey_i values produced the same EmployeeID")
	}
}

func TestComputeEmployeeID_RejectsEmptyKey(t *testing.T) {
	if _, err := ComputeEmployeeID(nil); err != ErrEmptyEmployeeKey {
		t.Fatalf("expected ErrEmptyEmployeeKey, got %v", err)
	}
}

// TestComputeUpdatedBy_DifferentActorsDifferentPseudonyms — the same
// employeeKey_i but two different acting users must produce two different
// UpdatedBy values (otherwise "who made this change" can't be distinguished
// even pseudonymously).
func TestComputeUpdatedBy_DifferentActorsDifferentPseudonyms(t *testing.T) {
	key := testEmployeeKey()
	byHR, err := ComputeUpdatedBy(key, "hr-admin-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	byEmployee, err := ComputeUpdatedBy(key, "employee-self-002")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if byHR == byEmployee {
		t.Fatal("different actors produced the same UpdatedBy pseudonym")
	}
}

// TestComputeUpdatedBy_DistinctFromEmployeeID — EmployeeID and UpdatedBy
// use different HMAC messages ("id" vs "actor"+userID) under the SAME key;
// confirms the domain-separation actually changes the output, not just in
// theory.
func TestComputeUpdatedBy_DistinctFromEmployeeID(t *testing.T) {
	key := testEmployeeKey()
	employeeID, _ := ComputeEmployeeID(key)
	updatedBy, _ := ComputeUpdatedBy(key, "id") // deliberately pass userID="id" to stress the separator

	if employeeID == updatedBy {
		t.Fatal("EmployeeID and UpdatedBy collided — HMAC domain separation is not working")
	}
}

// TestJCS_ReordersKeysAndNormalizesNumbers — re-verifies the PINNED library
// from within THIS package's own import (not just the standalone jcsverify
// tool, which checks the library in isolation) against the two properties
// REC-1 actually depends on: key reordering and ECMAScript-style number
// normalization. A future dependency swap that silently changes either
// behavior fails here too, not just in jcsverify.
func TestJCS_ReordersKeysAndNormalizesNumbers(t *testing.T) {
	input := []byte(`{"z":1,"a":1E1}`)
	canonical, err := jsoncanonicalizer.Transform(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"a":10,"z":1}`
	if !bytes.Equal(canonical, []byte(want)) {
		t.Fatalf("canonicalization drifted:\n got  %q\n want %q", canonical, want)
	}
}

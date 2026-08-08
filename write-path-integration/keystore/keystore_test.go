package keystore

import (
	"bytes"
	"context"
	"testing"
)

func TestSaltStore_FreshPerRecordVersion(t *testing.T) {
	ctx := context.Background()
	s := NewInMemorySaltStore()

	salt1, err := s.PutSalt(ctx, "emp-1", "PERSONAL", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	salt2, err := s.PutSalt(ctx, "emp-1", "PERSONAL", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bytes.Equal(salt1, salt2) {
		t.Fatal("two different record versions got the same salt — not fresh per record")
	}
	if len(salt1) < MinSaltBytes {
		t.Fatalf("salt shorter than the ADR-0011 floor: %d bytes", len(salt1))
	}
}

func TestSaltStore_RejectsDuplicatePut(t *testing.T) {
	ctx := context.Background()
	s := NewInMemorySaltStore()
	if _, err := s.PutSalt(ctx, "emp-1", "PERSONAL", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := s.PutSalt(ctx, "emp-1", "PERSONAL", 1); err == nil {
		t.Fatal("expected an error rotating an already-generated record's salt in place, got nil")
	}
}

func TestSaltStore_GetMatchesPut(t *testing.T) {
	ctx := context.Background()
	s := NewInMemorySaltStore()
	put, err := s.PutSalt(ctx, "emp-1", "PAYROLL", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := s.GetSalt(ctx, "emp-1", "PAYROLL", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(put, got) {
		t.Fatal("GetSalt did not return the same bytes PutSalt generated")
	}
}

func TestSaltStore_GetMissingFails(t *testing.T) {
	ctx := context.Background()
	s := NewInMemorySaltStore()
	if _, err := s.GetSalt(ctx, "emp-nonexistent", "PERSONAL", 1); err == nil {
		t.Fatal("expected an error reading a salt that was never stored, got nil")
	}
}

func TestEmployeeKeyStore_OneKeyPerEmployeeGeneratedOnce(t *testing.T) {
	ctx := context.Background()
	s := NewInMemoryEmployeeKeyStore()
	key1, err := s.GetOrCreateEmployeeKey(ctx, "emp-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	key2, err := s.GetOrCreateEmployeeKey(ctx, "emp-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(key1, key2) {
		t.Fatal("GetOrCreateEmployeeKey generated a NEW key on the second call — employeeKey_i must be generated once and reused")
	}
	if len(key1) < MinKeyBytes {
		t.Fatalf("employeeKey_i shorter than the ADR-0011 floor: %d bytes", len(key1))
	}
}

func TestEmployeeKeyStore_DifferentEmployeesDifferentKeys(t *testing.T) {
	ctx := context.Background()
	s := NewInMemoryEmployeeKeyStore()
	key1, _ := s.GetOrCreateEmployeeKey(ctx, "emp-1")
	key2, _ := s.GetOrCreateEmployeeKey(ctx, "emp-2")
	if bytes.Equal(key1, key2) {
		t.Fatal("two different employees got the same employeeKey_i — independently-random generation is not independent")
	}
}

// TestEmployeeKeyStore_DeleteIsIrreversible proves destruction actually
// happens (a fresh, DIFFERENT key is generated on next use) rather than
// merely marking the old key hidden while still recoverable — this is the
// property ADR-0015's erasure mechanism and ADR-0021's accidental-loss
// trade-off both depend on.
func TestEmployeeKeyStore_DeleteIsIrreversible(t *testing.T) {
	ctx := context.Background()
	s := NewInMemoryEmployeeKeyStore()
	original, _ := s.GetOrCreateEmployeeKey(ctx, "emp-1")

	if err := s.DeleteEmployeeKey(ctx, "emp-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	regenerated, err := s.GetOrCreateEmployeeKey(ctx, "emp-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bytes.Equal(original, regenerated) {
		t.Fatal("the original key survived deletion — erasure is not irreversible")
	}
}

// TestNonMixingInvariant_IndependentGeneration is a sanity check, not a
// proof (a unit test cannot inspect Domain A/B, which this package never
// imports or references at all — that absence, checkable by code review,
// is the actual invariant). This just confirms the two stores' outputs for
// the SAME logical subject are not trivially identical or one derived from
// the other by an obvious transform.
func TestNonMixingInvariant_IndependentGeneration(t *testing.T) {
	ctx := context.Background()
	salts := NewInMemorySaltStore()
	keys := NewInMemoryEmployeeKeyStore()

	salt, _ := salts.PutSalt(ctx, "emp-1", "PERSONAL", 1)
	key, _ := keys.GetOrCreateEmployeeKey(ctx, "emp-1")

	if bytes.Equal(salt, key) {
		t.Fatal("salt and employeeKey_i for the same employee are byte-identical — extremely unlikely by chance, suggests a shared source")
	}
}

package keystore

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

// TestSaltHandoff_GrantsForRecordOwner: FR-36's core positive case — "the
// employee over their own record."
func TestSaltHandoff_GrantsForRecordOwner(t *testing.T) {
	ctx := context.Background()
	salts := NewInMemorySaltStore()
	want, err := salts.PutSalt(ctx, "emp-1", "PERSONAL", 1)
	if err != nil {
		t.Fatalf("unexpected error priming salt: %v", err)
	}
	audit := NewInMemoryAuditLog()
	h := &SaltHandoff{
		Salts: salts,
		Authorize: func(_ context.Context, requesterID, employeeID, _ string, _ int) (bool, error) {
			return requesterID == employeeID, nil // stand-in policy: "employee over own record"
		},
		Audit: audit,
	}

	got, err := h.RequestSalt(ctx, "emp-1", "emp-1", "PERSONAL", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("RequestSalt did not return the salt PutSalt generated")
	}

	entries := audit.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected exactly one audit entry, got %d: %+v", len(entries), entries)
	}
	if !entries[0].Granted {
		t.Fatalf("expected a granted entry, got %+v", entries[0])
	}
}

// TestSaltHandoff_DeniesUnauthorizedThirdParty: FR-36's core negative case
// — a requester who is neither the record's employee nor an in-scope
// auditor.
func TestSaltHandoff_DeniesUnauthorizedThirdParty(t *testing.T) {
	ctx := context.Background()
	salts := NewInMemorySaltStore()
	if _, err := salts.PutSalt(ctx, "emp-1", "PERSONAL", 1); err != nil {
		t.Fatalf("unexpected error priming salt: %v", err)
	}
	audit := NewInMemoryAuditLog()
	h := &SaltHandoff{
		Salts: salts,
		Authorize: func(_ context.Context, requesterID, employeeID, _ string, _ int) (bool, error) {
			return requesterID == employeeID, nil
		},
		Audit: audit,
	}

	_, err := h.RequestSalt(ctx, "emp-2", "emp-1", "PERSONAL", 1)
	if !errors.Is(err, ErrSaltAccessDenied) {
		t.Fatalf("expected ErrSaltAccessDenied, got %v", err)
	}

	entries := audit.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected exactly one audit entry, got %d: %+v", len(entries), entries)
	}
	if entries[0].Granted {
		t.Fatalf("expected a denied entry, got %+v", entries[0])
	}
	if entries[0].Reason == "" {
		t.Fatal("expected a non-empty denial reason")
	}
}

// TestSaltHandoff_FailsClosedOnAuthorizeError: a broken policy backend must
// never be treated as an implicit grant.
func TestSaltHandoff_FailsClosedOnAuthorizeError(t *testing.T) {
	ctx := context.Background()
	salts := NewInMemorySaltStore()
	if _, err := salts.PutSalt(ctx, "emp-1", "PERSONAL", 1); err != nil {
		t.Fatalf("unexpected error priming salt: %v", err)
	}
	audit := NewInMemoryAuditLog()
	policyErr := errors.New("policy backend unreachable")
	h := &SaltHandoff{
		Salts: salts,
		Authorize: func(context.Context, string, string, string, int) (bool, error) {
			// The bool is irrelevant here — even a stray "true" alongside a
			// non-nil error must not leak through as a grant.
			return true, policyErr
		},
		Audit: audit,
	}

	salt, err := h.RequestSalt(ctx, "emp-1", "emp-1", "PERSONAL", 1)
	if err == nil {
		t.Fatal("expected an error when the authorization backend itself fails")
	}
	if salt != nil {
		t.Fatal("expected no salt returned when failing closed")
	}

	entries := audit.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected exactly one audit entry, got %d: %+v", len(entries), entries)
	}
	if entries[0].Granted {
		t.Fatalf("expected a denied entry when failing closed, got %+v", entries[0])
	}
	if entries[0].Reason == "" {
		t.Fatal("expected a non-empty reason recording the backend failure")
	}
}

// TestSaltHandoff_GetSaltMissNotRecordedAsDenial: a genuinely-nonexistent
// salt (never anchored) is a "not found," not a "denied" — the audit log
// must not report it as a refused access attempt.
func TestSaltHandoff_GetSaltMissNotRecordedAsDenial(t *testing.T) {
	ctx := context.Background()
	salts := NewInMemorySaltStore() // deliberately empty: nothing was ever anchored
	audit := NewInMemoryAuditLog()
	h := &SaltHandoff{
		Salts: salts,
		Authorize: func(context.Context, string, string, string, int) (bool, error) {
			return true, nil // authorized; the record simply doesn't exist
		},
		Audit: audit,
	}

	_, err := h.RequestSalt(ctx, "emp-1", "emp-1", "PERSONAL", 1)
	if err == nil {
		t.Fatal("expected an error for a salt that was never anchored")
	}
	if errors.Is(err, ErrSaltAccessDenied) {
		t.Fatal("a genuine not-found must not be reported as an access denial")
	}

	entries := audit.Entries()
	if len(entries) != 0 {
		t.Fatalf("expected zero audit entries for a not-found (not a denial), got %d: %+v", len(entries), entries)
	}
}

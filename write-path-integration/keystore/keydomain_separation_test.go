// ST-10 regression guard: proves, for the SAME employeeInternalID, that
// DocumentKeyStore (KEY_EMPLOYEE, ADR-0019 Domain B') and EmployeeKeyStore
// (employeeKey_i, ADR-0021 Domain C) produce DIFFERENT, independently-random
// key material — neither is derived from, nor derivable into, the other.
// keystore_test.go's own TestNonMixingInvariant_IndependentGeneration only
// checks SaltStore vs EmployeeKeyStore; this file covers the DocumentKeyStore
// vs EmployeeKeyStore pair for the same subject, which was not previously
// exercised by any test in this package.
//
// As keystore.go's own package doc-comment notes, a unit test cannot prove
// non-derivation in general (it can't inspect Domain A/B material this
// package never imports at all — that absence is a code-review-checkable
// property, not something Go code can assert against). What IS testable and
// asserted here: (1) the two stores never hand back byte-identical material
// for the same subject, across many independent subjects (astronomically
// unlikely by chance for a shared crypto/rand source, so equality would be a
// real signal of accidental sharing/derivation); and (2) destroying one
// domain's key for a subject leaves the other domain's key for that SAME
// subject completely untouched — the two stores are backed by genuinely
// separate state, not merely separate accessor methods over one shared map.
package keystore

import (
	"bytes"
	"context"
	"testing"
)

// TestKeyDomainSeparation_DocumentKeyAndEmployeeKeyDifferForSameSubject is
// the ST-10 invariant the task calls out explicitly: two calls — one to
// GetOrCreateDocumentKey, one to GetOrCreateEmployeeKey — for the SAME
// employeeInternalID must produce different, independent key material.
func TestKeyDomainSeparation_DocumentKeyAndEmployeeKeyDifferForSameSubject(t *testing.T) {
	ctx := context.Background()
	docKeys := NewInMemoryDocumentKeyStore()
	empKeys := NewInMemoryEmployeeKeyStore()

	subjects := []string{"emp-1", "emp-2", "emp-3", "same-id-edge-case"}
	for _, subject := range subjects {
		docKey, err := docKeys.GetOrCreateDocumentKey(ctx, subject)
		if err != nil {
			t.Fatalf("subject %s: unexpected error from GetOrCreateDocumentKey: %v", subject, err)
		}
		empKey, err := empKeys.GetOrCreateEmployeeKey(ctx, subject)
		if err != nil {
			t.Fatalf("subject %s: unexpected error from GetOrCreateEmployeeKey: %v", subject, err)
		}

		if len(docKey) < MinKeyBytes {
			t.Fatalf("subject %s: KEY_EMPLOYEE shorter than the ADR-0011 floor: %d bytes", subject, len(docKey))
		}
		if len(empKey) < MinKeyBytes {
			t.Fatalf("subject %s: employeeKey_i shorter than the ADR-0011 floor: %d bytes", subject, len(empKey))
		}
		if bytes.Equal(docKey, empKey) {
			t.Fatalf("subject %s: DocumentKeyStore and EmployeeKeyStore returned BYTE-IDENTICAL material for the same subject — extremely unlikely from two independent crypto/rand.Read calls, suggests one was derived from (or shares a source with) the other, violating ADR-0019's non-mixing invariant", subject)
		}

		// Idempotent-per-domain, and re-confirms the two domains stay distinct
		// on a second read, not just at first generation.
		docKeyAgain, err := docKeys.GetOrCreateDocumentKey(ctx, subject)
		if err != nil {
			t.Fatalf("subject %s: unexpected error on second GetOrCreateDocumentKey: %v", subject, err)
		}
		if !bytes.Equal(docKey, docKeyAgain) {
			t.Fatalf("subject %s: GetOrCreateDocumentKey returned a DIFFERENT key on the second call — KEY_EMPLOYEE must be generated once and reused", subject)
		}
		empKeyAgain, err := empKeys.GetOrCreateEmployeeKey(ctx, subject)
		if err != nil {
			t.Fatalf("subject %s: unexpected error on second GetOrCreateEmployeeKey: %v", subject, err)
		}
		if !bytes.Equal(empKey, empKeyAgain) {
			t.Fatalf("subject %s: GetOrCreateEmployeeKey returned a DIFFERENT key on the second call — employeeKey_i must be generated once and reused", subject)
		}
	}
}

// TestKeyDomainSeparation_DeletingOneDomainLeavesTheOtherIntact proves the
// two stores are backed by genuinely separate state for the same subject,
// not merely separate method names over one shared map: destroying
// DocumentKeyStore's key for a subject must not touch EmployeeKeyStore's key
// for that same subject (and vice versa) — required for ADR-0015/ADR-0019's
// per-domain, independently-triggerable erasure model (REC-6 destroys
// Domain C material; a document-key rotation/erasure must not accidentally
// erase Domain C, and vice versa).
func TestKeyDomainSeparation_DeletingOneDomainLeavesTheOtherIntact(t *testing.T) {
	ctx := context.Background()
	docKeys := NewInMemoryDocumentKeyStore()
	empKeys := NewInMemoryEmployeeKeyStore()
	const subject = "emp-1"

	docKey, err := docKeys.GetOrCreateDocumentKey(ctx, subject)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	empKey, err := empKeys.GetOrCreateEmployeeKey(ctx, subject)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := docKeys.DeleteDocumentKey(ctx, subject); err != nil {
		t.Fatalf("unexpected error deleting document key: %v", err)
	}

	// EmployeeKeyStore for the SAME subject must be completely unaffected:
	// re-reading it must return the ORIGINAL key, not a freshly-generated
	// one — if deleting Domain B' had somehow also cleared Domain C, this
	// would silently regenerate a new employeeKey_i here instead.
	empKeyAfter, err := empKeys.GetOrCreateEmployeeKey(ctx, subject)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(empKey, empKeyAfter) {
		t.Fatal("deleting DocumentKeyStore's key affected EmployeeKeyStore's key for the same subject — the two domains are not independently erasable")
	}

	// And the document key really is gone (regenerating produces a NEW,
	// different value) — confirms DeleteDocumentKey did its own job too,
	// not a no-op that would make the isolation check above vacuous.
	docKeyAfter, err := docKeys.GetOrCreateDocumentKey(ctx, subject)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bytes.Equal(docKey, docKeyAfter) {
		t.Fatal("the original document key survived deletion — DeleteDocumentKey erasure is not irreversible")
	}
}

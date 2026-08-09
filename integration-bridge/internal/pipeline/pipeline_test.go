package pipeline

import (
	"context"
	"errors"
	"testing"
)

var errTestSentinel = errors.New("pipeline_test sentinel error")

func TestBridgeCtx_RoundTripsThroughContext(t *testing.T) {
	bc := &bridgectx{EmployeeInternalID: "emp-1", ProfileSection: "PERSONAL"}
	ctx := withBridgeCtx(context.Background(), bc)

	got := bridgeCtxFrom(ctx)
	if got != bc {
		t.Fatalf("bridgeCtxFrom() returned a different pointer than was attached")
	}
	if got.EmployeeInternalID != "emp-1" || got.ProfileSection != "PERSONAL" {
		t.Errorf("bridgeCtxFrom() = %+v, want EmployeeInternalID=emp-1 ProfileSection=PERSONAL", got)
	}
}

func TestBridgeCtx_MutationsThroughRetrievedPointerArePersistentAcrossStages(t *testing.T) {
	bc := &bridgectx{}
	ctx := withBridgeCtx(context.Background(), bc)

	// Simulate [validate] setting identifiers, then [dispatch] setting a
	// result, then [map-error] reading both back -- three independent
	// retrievals of the same context, as the real pipeline stages will do.
	bridgeCtxFrom(ctx).EmployeeInternalID = "emp-2"
	bridgeCtxFrom(ctx).ProfileSection = "EMPLOYMENT"
	bridgeCtxFrom(ctx).Err = errTestSentinel

	final := bridgeCtxFrom(ctx)
	if final.EmployeeInternalID != "emp-2" {
		t.Errorf("EmployeeInternalID = %q, want %q", final.EmployeeInternalID, "emp-2")
	}
	if final.ProfileSection != "EMPLOYMENT" {
		t.Errorf("ProfileSection = %q, want %q", final.ProfileSection, "EMPLOYMENT")
	}
	if final.Err != errTestSentinel {
		t.Errorf("Err = %v, want %v", final.Err, errTestSentinel)
	}
}

func TestBridgeCtxFrom_PanicsIfNeverAttached(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("bridgeCtxFrom() on a context with no attached bridgectx: expected a panic, got none")
		}
	}()
	bridgeCtxFrom(context.Background())
}

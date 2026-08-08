package main

import (
	"context"
	"errors"
	"fmt"
	"testing"

	writepaths "writepaths"
)

func TestClassify_NilError_Committed(t *testing.T) {
	if got := classify(nil); got != "committed" {
		t.Errorf("classify(nil) = %q, want %q", got, "committed")
	}
}

func TestClassify_AuthError_Error(t *testing.T) {
	err := &authError{errors.New("bad api key")}
	if got := classify(err); got != "error" {
		t.Errorf("classify(authError) = %q, want %q", got, "error")
	}
}

func TestClassify_ValidationError_Rejected(t *testing.T) {
	err := &validationError{errors.New("missing employeeInternalID")}
	if got := classify(err); got != "rejected" {
		t.Errorf("classify(validationError) = %q, want %q", got, "rejected")
	}
}

func TestClassify_PartialFailureError_PartialFailure(t *testing.T) {
	err := &writepaths.PartialFailureError{
		EmployeeInternalID: "emp-1",
		ProfileSection:     "PERSONAL",
		Version:            1,
		Err:                errors.New("gateway unreachable"),
	}
	if got := classify(err); got != "partial_failure" {
		t.Errorf("classify(*writepaths.PartialFailureError) = %q, want %q", got, "partial_failure")
	}
}

func TestClassify_WrappedPartialFailureError_PartialFailure(t *testing.T) {
	pfErr := &writepaths.PartialFailureError{Err: errors.New("x")}
	wrapped := fmt.Errorf("dispatch: %w", pfErr)
	if got := classify(wrapped); got != "partial_failure" {
		t.Errorf("classify(wrapped *PartialFailureError) = %q, want %q", got, "partial_failure")
	}
}

func TestClassify_DeadlineExceeded_PartialFailure(t *testing.T) {
	if got := classify(context.DeadlineExceeded); got != "partial_failure" {
		t.Errorf("classify(context.DeadlineExceeded) = %q, want %q", got, "partial_failure")
	}
}

func TestClassify_WrappedDeadlineExceeded_PartialFailure(t *testing.T) {
	wrapped := fmt.Errorf("integrationbridge: dispatch exceeded configured timeout: %w", context.DeadlineExceeded)
	if got := classify(wrapped); got != "partial_failure" {
		t.Errorf("classify(wrapped context.DeadlineExceeded) = %q, want %q", got, "partial_failure")
	}
}

func TestClassify_PlainUntypedError_Error(t *testing.T) {
	// e.g. a Hooks.SaveSection failure, pre-anchor() -- nothing committed,
	// not a PartialFailureError, not a timeout, not auth/validate.
	err := errors.New("operational store unavailable")
	if got := classify(err); got != "error" {
		t.Errorf("classify(plain error) = %q, want %q", got, "error")
	}
}

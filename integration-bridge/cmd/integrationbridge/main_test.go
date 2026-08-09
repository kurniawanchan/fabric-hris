package main

import (
	"errors"
	"testing"
	"time"
)

func TestCloseBoth_CallsBothEvenWhenFirstErrors(t *testing.T) {
	wantErrA := errors.New("a failed")
	var bCalled bool

	err := closeBoth(
		func() error { return wantErrA },
		func() error { bCalled = true; return nil },
	)

	if !bCalled {
		t.Error("second close function was not called after the first errored")
	}
	if !errors.Is(err, wantErrA) {
		t.Errorf("closeBoth() error = %v, want it to wrap %v", err, wantErrA)
	}
}

func TestCloseBoth_CallsBothEvenWhenSecondErrors(t *testing.T) {
	wantErrB := errors.New("b failed")
	var aCalled bool

	err := closeBoth(
		func() error { aCalled = true; return nil },
		func() error { return wantErrB },
	)

	if !aCalled {
		t.Error("first close function was not called")
	}
	if !errors.Is(err, wantErrB) {
		t.Errorf("closeBoth() error = %v, want it to wrap %v", err, wantErrB)
	}
}

func TestCloseBoth_NilWhenBothSucceed(t *testing.T) {
	err := closeBoth(
		func() error { return nil },
		func() error { return nil },
	)
	if err != nil {
		t.Errorf("closeBoth() = %v, want nil", err)
	}
}

func TestCloseBoth_JoinsBothErrorsWhenBothFail(t *testing.T) {
	wantErrA := errors.New("a failed")
	wantErrB := errors.New("b failed")

	err := closeBoth(
		func() error { return wantErrA },
		func() error { return wantErrB },
	)

	if !errors.Is(err, wantErrA) {
		t.Errorf("closeBoth() error = %v, want it to wrap %v", err, wantErrA)
	}
	if !errors.Is(err, wantErrB) {
		t.Errorf("closeBoth() error = %v, want it to wrap %v", err, wantErrB)
	}
}

// TestServerWriteTimeout_AlwaysExceedsDispatchTimeout is the fix for a code
// review finding: srv.WriteTimeout was hardcoded to 30s independently of
// the configurable dispatchTimeout -- raising BRIDGE_DISPATCH_TIMEOUT above
// 30s would let the server force-close the connection before the handler
// could ever write a response. serverWriteTimeout must always leave enough
// room for the dispatch call itself to finish before the server's own
// timeout could fire.
func TestServerWriteTimeout_AlwaysExceedsDispatchTimeout(t *testing.T) {
	cases := []time.Duration{5 * time.Second, 30 * time.Second, 90 * time.Second}
	for _, dispatchTimeout := range cases {
		got := serverWriteTimeout(dispatchTimeout)
		if got <= dispatchTimeout {
			t.Errorf("serverWriteTimeout(%v) = %v, want it to exceed dispatchTimeout", dispatchTimeout, got)
		}
	}
}

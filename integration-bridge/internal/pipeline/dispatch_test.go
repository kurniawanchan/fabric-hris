package pipeline

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestDispatch_Success_ReturnsResultAndNilError(t *testing.T) {
	want := []byte(`{"recordID":"abc"}`)
	fn := func(ctx context.Context, employeeInternalID, userID, recordIdentity string, newValue, document []byte) ([]byte, error) {
		return want, nil
	}

	result, err := dispatch(context.Background(), time.Second, fn, "emp-1", "user-1", "", []byte(`{}`), nil)
	if err != nil {
		t.Fatalf("dispatch() unexpected error: %v", err)
	}
	if string(result) != string(want) {
		t.Errorf("result = %s, want %s", result, want)
	}
}

func TestDispatch_FnError_PropagatesWhenDeadlineNotExceeded(t *testing.T) {
	wantErr := errors.New("boom")
	fn := func(ctx context.Context, employeeInternalID, userID, recordIdentity string, newValue, document []byte) ([]byte, error) {
		return nil, wantErr
	}

	_, err := dispatch(context.Background(), time.Second, fn, "emp-1", "user-1", "", []byte(`{}`), nil)
	if !errors.Is(err, wantErr) {
		t.Errorf("dispatch() error = %v, want it to wrap/equal %v", err, wantErr)
	}
}

func TestDispatch_TimeoutFires_FnAlsoErrors_ReturnsDeadlineExceeded(t *testing.T) {
	fn := func(ctx context.Context, employeeInternalID, userID, recordIdentity string, newValue, document []byte) ([]byte, error) {
		<-ctx.Done()
		return nil, errors.New("gateway: context deadline exceeded")
	}

	_, err := dispatch(context.Background(), 10*time.Millisecond, fn, "emp-1", "user-1", "", []byte(`{}`), nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("dispatch() error = %v, want context.DeadlineExceeded", err)
	}
}

// TestDispatch_TimeoutFires_FnStillSucceeds_ReturnsRealSuccess is the fix for
// a review finding (Story 1.2 code review, Decision 1): dispatch must not
// discard a genuinely successful result just because fn took longer than
// budget. Since dispatch blocks synchronously the entire time (AD-4), the
// caller is on the same open connection either way -- there is no
// duplicate-response risk in trusting a real success, however long it took.
// Only classify as a timeout when fn ALSO returns an error.
func TestDispatch_TimeoutFires_FnStillSucceeds_ReturnsRealSuccess(t *testing.T) {
	want := []byte(`{"recordID":"rec-slow-success"}`)
	fn := func(ctx context.Context, employeeInternalID, userID, recordIdentity string, newValue, document []byte) ([]byte, error) {
		<-ctx.Done()
		return want, nil
	}

	result, err := dispatch(context.Background(), 10*time.Millisecond, fn, "emp-1", "user-1", "", []byte(`{}`), nil)
	if err != nil {
		t.Errorf("dispatch() error = %v, want nil -- a genuine success must survive even after the budget elapsed", err)
	}
	if string(result) != string(want) {
		t.Errorf("result = %s, want %s -- a real recordID must never be discarded", result, want)
	}
}

func TestDispatch_NeverRacesGoroutine_BlocksUntilFnReturns(t *testing.T) {
	var fnFinished atomic.Bool
	fn := func(ctx context.Context, employeeInternalID, userID, recordIdentity string, newValue, document []byte) ([]byte, error) {
		<-ctx.Done()
		time.Sleep(30 * time.Millisecond)
		fnFinished.Store(true)
		return nil, nil
	}

	_, _ = dispatch(context.Background(), 5*time.Millisecond, fn, "emp-1", "user-1", "", []byte(`{}`), nil)

	if !fnFinished.Load() {
		t.Error("dispatch() returned before fn finished -- it must block synchronously, never race a goroutine against the timeout (AD-4)")
	}
}

func TestDispatch_PassesArgsThroughUnchanged(t *testing.T) {
	wantEmployeeInternalID := "emp-42"
	wantUserID := "user-42"
	wantNewValue := []byte(`{"fullName":"Test"}`)
	wantDocument := []byte("doc bytes")

	var gotEmployeeInternalID, gotUserID string
	var gotNewValue, gotDocument []byte
	var gotCtx context.Context

	fn := func(ctx context.Context, employeeInternalID, userID, recordIdentity string, newValue, document []byte) ([]byte, error) {
		gotCtx = ctx
		gotEmployeeInternalID = employeeInternalID
		gotUserID = userID
		gotNewValue = newValue
		gotDocument = document
		return nil, nil
	}

	_, _ = dispatch(context.Background(), time.Second, fn, wantEmployeeInternalID, wantUserID, "", wantNewValue, wantDocument)

	if gotEmployeeInternalID != wantEmployeeInternalID {
		t.Errorf("employeeInternalID = %q, want %q", gotEmployeeInternalID, wantEmployeeInternalID)
	}
	if gotUserID != wantUserID {
		t.Errorf("userID = %q, want %q", gotUserID, wantUserID)
	}
	if string(gotNewValue) != string(wantNewValue) {
		t.Errorf("newValue = %s, want %s", gotNewValue, wantNewValue)
	}
	if string(gotDocument) != string(wantDocument) {
		t.Errorf("document = %s, want %s", gotDocument, wantDocument)
	}
	if _, ok := gotCtx.Deadline(); !ok {
		t.Error("fn's ctx has no deadline -- dispatch must scope a context.WithTimeout around the call (AD-4)")
	}
}

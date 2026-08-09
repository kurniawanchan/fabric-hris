package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type stubCloser struct {
	calls int
	err   error
	mu    sync.Mutex
}

func (s *stubCloser) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return s.err
}

func (s *stubCloser) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestOnceCloser_MultipleCallsCloseUnderlyingExactlyOnce(t *testing.T) {
	stub := &stubCloser{}
	oc := newOnceCloser(stub)

	for i := 0; i < 5; i++ {
		if err := oc.Close(); err != nil {
			t.Fatalf("Close() call #%d: unexpected error: %v", i+1, err)
		}
	}

	if got := stub.callCount(); got != 1 {
		t.Errorf("underlying closer.Close() called %d times, want exactly 1", got)
	}
}

func TestOnceCloser_ConcurrentCallsCloseUnderlyingExactlyOnce(t *testing.T) {
	stub := &stubCloser{}
	oc := newOnceCloser(stub)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = oc.Close()
		}()
	}
	wg.Wait()

	if got := stub.callCount(); got != 1 {
		t.Errorf("underlying closer.Close() called %d times under concurrent shutdown signals, want exactly 1", got)
	}
}

func TestOnceCloser_PropagatesUnderlyingError(t *testing.T) {
	wantErr := errors.New("close failed")
	stub := &stubCloser{err: wantErr}
	oc := newOnceCloser(stub)

	if err := oc.Close(); err != wantErr {
		t.Errorf("Close() = %v, want %v", err, wantErr)
	}
	// A second call must still return the original error, not nil --
	// sync.Once doesn't re-run the func, but the caller still needs the result.
	if err := oc.Close(); err != wantErr {
		t.Errorf("second Close() = %v, want the same %v (cached, not silently nil)", err, wantErr)
	}
	if got := stub.callCount(); got != 1 {
		t.Errorf("underlying closer.Close() called %d times, want exactly 1", got)
	}
}

func TestRunUntilShutdown_ClosesExactlyOnceAfterContextCancellation(t *testing.T) {
	stub := &stubCloser{}
	oc := newOnceCloser(stub)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runUntilShutdown(ctx, oc) }()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("runUntilShutdown() = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runUntilShutdown() did not return after context cancellation")
	}

	if got := stub.callCount(); got != 1 {
		t.Errorf("underlying closer.Close() called %d times, want exactly 1", got)
	}
}

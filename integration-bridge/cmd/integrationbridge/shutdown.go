package main

import (
	"context"
	"sync"
)

// closer abstracts *gatewayclient.GatewayClient's Close() method as a local
// interface, so shutdown-wiring is unit-testable without a live Fabric
// network — gatewayclient.GatewayClient is a concrete struct with private
// fields and can't be mocked by substitution otherwise.
type closer interface {
	Close() error
}

// onceCloser makes Close() idempotent: however many times, or however
// concurrently, Close() is called, the underlying closer's Close() runs
// exactly once. AC#3 requires this literally ("Close() is never called more
// than once") regardless of how shutdown ends up triggered more than once.
type onceCloser struct {
	once   sync.Once
	closer closer
	err    error
}

func newOnceCloser(c closer) *onceCloser {
	return &onceCloser{closer: c}
}

func (o *onceCloser) Close() error {
	o.once.Do(func() {
		o.err = o.closer.Close()
	})
	return o.err
}

// runUntilShutdown blocks until ctx is done (the caller wires ctx from
// signal.NotifyContext for SIGTERM/SIGINT), then closes c and returns its
// error. c is expected to be an *onceCloser wrapping the real
// *gatewayclient.GatewayClient, so this is safe even if something else also
// calls Close() independently.
func runUntilShutdown(ctx context.Context, c closer) error {
	<-ctx.Done()
	return c.Close()
}

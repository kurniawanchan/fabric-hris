// PLACEHOLDER STORES IN USE: writepaths.Hooks below is wired with in-memory
// stores only. Every process restart silently loses every salt and
// employeeKey_i ever created. See wiring.go's buildHooks doc comment and the
// story's Open Questions before this ships to any real deployment — no
// persistent implementation of these stores exists anywhere in this
// codebase yet.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	gatewayclient "gatewayclient"
	writepaths "writepaths"

	"integrationbridge/internal/pipeline"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// closeBoth calls both close functions unconditionally, regardless of
// whether the first one errors, and joins any resulting errors. This is
// what makes AC#3 ("calls GatewayClient.Close() exactly once") hold even
// when the HTTP server's own shutdown fails or times out — a naive
// early-return after the first failure would skip the second call entirely.
func closeBoth(a, b func() error) error {
	return errors.Join(a(), b())
}

// shutdownSequence composes "stop accepting new HTTP requests" and "close
// the retained Fabric connection" into a single closer, so wrapping it in
// onceCloser gives AC#3's exact ordering ("stops accepting new requests,
// calls GatewayClient.Close() exactly once") with the same idempotency
// guarantee runUntilShutdown/onceCloser already have tests for. closeBoth
// guarantees gw.Close() still runs even if srv.Shutdown fails.
type shutdownSequence struct {
	srv *http.Server
	gw  closer
}

func (s *shutdownSequence) Close() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return closeBoth(
		func() error { return s.srv.Shutdown(shutdownCtx) },
		s.gw.Close,
	)
}

// buildMux registers all five AD-5 profile-section routes onto a fresh
// ServeMux -- extracted out of run() (Story 1.6) so the complete, assembled
// route set is itself unit-testable (AC#2's "all five routes exist, a 6th
// name 404s" and AC#3's "PERSONAL and PAYROLL stay independent" are claims
// about the whole set, not about any single route in isolation). Every call
// shares the identical pipeline (AD-5); only ProfileSection and Dispatch
// vary per route.
func buildMux(hooks *writepaths.Hooks, authCfg pipeline.AuthConfig, dispatchTimeout time.Duration) *http.ServeMux {
	mux := http.NewServeMux()
	pipeline.RegisterProfileSectionRoute(mux, "POST /v1/profile-sections/PERSONAL",
		pipeline.RouteConfig{ProfileSection: "PERSONAL", Dispatch: hooks.UpdatePersonalData},
		authCfg, hooks.TenantID, dispatchTimeout)
	pipeline.RegisterProfileSectionRoute(mux, "POST /v1/profile-sections/EMPLOYMENT",
		pipeline.RouteConfig{ProfileSection: "EMPLOYMENT", Dispatch: hooks.ApproveEmploymentTransfer},
		authCfg, hooks.TenantID, dispatchTimeout)
	pipeline.RegisterProfileSectionRoute(mux, "POST /v1/profile-sections/EDUCATION",
		pipeline.RouteConfig{ProfileSection: "EDUCATION", Dispatch: hooks.RecordEducationHistory},
		authCfg, hooks.TenantID, dispatchTimeout)
	pipeline.RegisterProfileSectionRoute(mux, "POST /v1/profile-sections/ADDITIONAL",
		pipeline.RouteConfig{ProfileSection: "ADDITIONAL", Dispatch: hooks.ApproveFamilyDataChange},
		authCfg, hooks.TenantID, dispatchTimeout)
	pipeline.RegisterProfileSectionRoute(mux, "POST /v1/profile-sections/PAYROLL",
		pipeline.RouteConfig{ProfileSection: "PAYROLL", Dispatch: hooks.UpdatePayrollBankAccount},
		authCfg, hooks.TenantID, dispatchTimeout)
	return mux
}

// writeTimeoutBuffer covers everything srv.WriteTimeout must allow for
// beyond the [dispatch] call itself: [auth]/[validate] work plus writing
// the response. serverWriteTimeout derives from dispatchTimeout rather than
// a disconnected constant so raising BRIDGE_DISPATCH_TIMEOUT can never put
// it above srv.WriteTimeout, which would force-close the connection before
// the handler could ever write a response (code review finding).
const writeTimeoutBuffer = 10 * time.Second

func serverWriteTimeout(dispatchTimeout time.Duration) time.Duration {
	return dispatchTimeout + writeTimeoutBuffer
}

func run() error {
	cfg, err := LoadConfig(os.Getenv)
	if err != nil {
		return fmt.Errorf("integrationbridge: %w", err)
	}

	// buildHooks constructs the *gatewayclient.GatewayClient and Hooks
	// exactly once, here, before the HTTP listener starts (AD-2, AC#2).
	hooks, gw, err := buildHooks(cfg, gatewayclient.NewGatewayClient)
	if err != nil {
		return fmt.Errorf("integrationbridge: %w", err)
	}
	log.Printf("integrationbridge: ready (tenant=%s)", hooks.TenantID)

	authCfg := pipeline.AuthConfig{APIKey: cfg.APIKey, CompanyID: cfg.CompanyID}
	dispatchTimeout := ResolveDispatchTimeout(os.Getenv)
	mux := buildMux(hooks, authCfg, dispatchTimeout)

	srv := &http.Server{
		Addr:              ResolveHTTPAddr(os.Getenv),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      serverWriteTimeout(dispatchTimeout),
		IdleTimeout:       120 * time.Second,
	}
	shutdown := newOnceCloser(&shutdownSequence{srv: srv, gw: gw})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	serveErrCh := make(chan error, 1)
	go func() { serveErrCh <- srv.ListenAndServe() }()

	select {
	case err := <-serveErrCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("integrationbridge: http server: %w", errors.Join(err, shutdown.Close()))
		}
		return nil
	case <-ctx.Done():
		return runUntilShutdown(ctx, shutdown)
	}
}

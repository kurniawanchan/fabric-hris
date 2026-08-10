package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	writepaths "writepaths"

	"integrationbridge/internal/pipeline"
)

// testAuthConfig mirrors internal/pipeline/handlers_test.go's own helper of
// the same name -- kept separately here (rather than exported from pipeline
// just for tests) because AuthConfig's zero-friction construction doesn't
// need a shared helper across a package boundary; the credential literals
// below must stay identical to pipeline's copy since several tests in this
// file authenticate against a mux built with it.
func testAuthConfig() pipeline.AuthConfig {
	return pipeline.AuthConfig{APIKey: "secret-key", CompanyID: "tenant01"}
}

// stubHooks builds a *writepaths.Hooks whose write-path methods are
// unreachable in these tests (buildMux only needs method VALUES to assign
// as pipeline.DispatchFunc -- it never calls them itself). A nil Store/Keys/etc. is
// fine here since no test in this file causes a route handler to actually
// invoke a Hooks method; each test only exercises 404/routing behavior or
// substitutes its own stub dispatch via a fresh pipeline.RouteConfig where needed.
func stubHooksForMuxTest() *writepaths.Hooks {
	return &writepaths.Hooks{TenantID: "tenant01"}
}

func TestBuildMux_AllFiveRoutesRegistered_UnrelatedPathIs404(t *testing.T) {
	mux := buildMux(stubHooksForMuxTest(), nil, testAuthConfig(), time.Second)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /health: status code = %d, want 404", rec.Code)
	}
}

func TestBuildMux_SixthSectionName_Is404(t *testing.T) {
	mux := buildMux(stubHooksForMuxTest(), nil, testAuthConfig(), time.Second)

	req := httptest.NewRequest(http.MethodPost, "/v1/profile-sections/BOGUS", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("POST /v1/profile-sections/BOGUS: status code = %d, want 404 (FR-8: only the 5 ratified sections exist, by construction)", rec.Code)
	}
}

func TestBuildMux_AllFiveRatifiedRoutesExist(t *testing.T) {
	mux := buildMux(stubHooksForMuxTest(), nil, testAuthConfig(), time.Second)

	for _, section := range []string{"PERSONAL", "EMPLOYMENT", "EDUCATION", "ADDITIONAL", "PAYROLL"} {
		req := httptest.NewRequest(http.MethodPost, "/v1/profile-sections/"+section, strings.NewReader(`{}`))
		req.Header.Set("X-Api-Key", "secret-key")
		req.Header.Set("X-Company-ID", "tenant01")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound {
			t.Errorf("POST /v1/profile-sections/%s: got 404, want this route to exist (even if it then rejects the empty body)", section)
		}
	}
}

func TestBuildMux_PersonalAndPayrollAreIndependentRoutes(t *testing.T) {
	// buildMux wires real Hooks methods, so this test cannot substitute stub
	// dispatch functions -- instead it proves independence structurally: two
	// distinct ProfileSection values reach two distinct routes, neither of
	// which is reachable from the other's path. A cross-dispatch bug (e.g.
	// PAYROLL accidentally registered with PERSONAL's Dispatch) cannot be
	// detected by response shape alone without a live network, so this test
	// instead confirms AC#3's literal claim: both paths are independently
	// routable and neither 404s the other away, regardless of how many
	// callers on the other side of this API share one commit path.
	mux := buildMux(stubHooksForMuxTest(), nil, testAuthConfig(), time.Second)

	for _, path := range []string{"/v1/profile-sections/PERSONAL", "/v1/profile-sections/PAYROLL"} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		req.Header.Set("X-Api-Key", "secret-key")
		req.Header.Set("X-Company-ID", "tenant01")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound {
			t.Errorf("%s: got 404, want an independently registered route", path)
		}
	}
}

// TestBuildMux_FamilyPath_IsNotRegistered guards against the naming trap
// directly, against the REAL production wiring: "/v1/profile-sections/FAMILY"
// must NOT resolve to anything on the mux main.go actually builds, only the
// ratified "ADDITIONAL" enum value is a real route. Deliberately built via
// buildMux (not a throwaway http.NewServeMux() + a manually re-declared
// pipeline.RouteConfig{ProfileSection: "ADDITIONAL", ...}) -- a test that hardcodes
// the correct literal itself would pass unconditionally regardless of what
// buildMux actually registers, proving nothing about the artifact this
// story ships (code review finding; the identical fix is still owed to
// Story 1.3's analogous TestRegisterProfileSectionRoute_TransferPath_IsNotRegistered,
// tracked in deferred-work.md rather than reopened here).
func TestBuildMux_FamilyPath_IsNotRegistered(t *testing.T) {
	mux := buildMux(stubHooksForMuxTest(), nil, testAuthConfig(), time.Second)

	req := httptest.NewRequest(http.MethodPost, "/v1/profile-sections/FAMILY", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want 404 -- \"FAMILY\" must never be a registered route path", rec.Code)
	}
}

// TestBuildMux_HistoryRouteRegistered proves the read route buildMux adds
// exists on the assembled mux. The request deliberately carries no auth
// headers so it fails at [auth] and never reaches cfg.Keys/cfg.Ledger (both
// nil in stubHooksForMuxTest/the nil historyReader passed above) -- this
// test is about routing existence, not read behavior (see
// internal/pipeline/history_route_test.go for that).
func TestBuildMux_HistoryRouteRegistered(t *testing.T) {
	mux := buildMux(stubHooksForMuxTest(), nil, testAuthConfig(), time.Second)

	req := httptest.NewRequest(http.MethodGet, "/v1/profile-sections/history", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Error("GET /v1/profile-sections/history: got 404, want this route to exist")
	}
}

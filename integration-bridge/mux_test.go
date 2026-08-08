package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	writepaths "writepaths"
)

// stubHooks builds a *writepaths.Hooks whose write-path methods are
// unreachable in these tests (buildMux only needs method VALUES to assign
// as dispatchFunc -- it never calls them itself). A nil Store/Keys/etc. is
// fine here since no test in this file causes a route handler to actually
// invoke a Hooks method; each test only exercises 404/routing behavior or
// substitutes its own stub dispatch via a fresh routeConfig where needed.
func stubHooksForMuxTest() *writepaths.Hooks {
	return &writepaths.Hooks{TenantID: "tenant01"}
}

func TestBuildMux_AllFiveRoutesRegistered_UnrelatedPathIs404(t *testing.T) {
	mux := buildMux(stubHooksForMuxTest(), testAuthConfig(), time.Second)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /health: status code = %d, want 404", rec.Code)
	}
}

func TestBuildMux_SixthSectionName_Is404(t *testing.T) {
	mux := buildMux(stubHooksForMuxTest(), testAuthConfig(), time.Second)

	req := httptest.NewRequest(http.MethodPost, "/v1/profile-sections/BOGUS", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("POST /v1/profile-sections/BOGUS: status code = %d, want 404 (FR-8: only the 5 ratified sections exist, by construction)", rec.Code)
	}
}

func TestBuildMux_AllFiveRatifiedRoutesExist(t *testing.T) {
	mux := buildMux(stubHooksForMuxTest(), testAuthConfig(), time.Second)

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
	mux := buildMux(stubHooksForMuxTest(), testAuthConfig(), time.Second)

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

// TestRegisterProfileSectionRoute_TwoRoutesOnOneMux_NeverCrossDispatch is the
// real cross-dispatch proof for AC#3, using registerProfileSectionRoute
// directly (not buildMux) so distinct stub Dispatch functions can be
// substituted per route -- proving PERSONAL's handler never reaches
// PAYROLL's dispatch target or vice versa, regardless of what the caller
// does on its own side.
func TestRegisterProfileSectionRoute_TwoRoutesOnOneMux_NeverCrossDispatch(t *testing.T) {
	var personalCalled, payrollCalled bool

	mux := http.NewServeMux()
	registerProfileSectionRoute(mux, "POST /v1/profile-sections/PERSONAL",
		routeConfig{ProfileSection: "PERSONAL", Dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
			personalCalled = true
			return []byte(`{"recordID":"rec-personal"}`), nil
		}},
		testAuthConfig(), "tenant01", time.Second)
	registerProfileSectionRoute(mux, "POST /v1/profile-sections/PAYROLL",
		routeConfig{ProfileSection: "PAYROLL", Dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
			payrollCalled = true
			return []byte(`{"recordID":"rec-payroll"}`), nil
		}},
		testAuthConfig(), "tenant01", time.Second)

	req := httptest.NewRequest(http.MethodPost, "/v1/profile-sections/PAYROLL",
		strings.NewReader(`{"employeeInternalID":"emp-1","userID":"user-1","newValue":{"bankAccountNumber":123}}`))
	req.Header.Set("X-Api-Key", "secret-key")
	req.Header.Set("X-Company-ID", "tenant01")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if payrollCalled == false {
		t.Error("PAYROLL request never reached the PAYROLL dispatch target")
	}
	if personalCalled {
		t.Error("PAYROLL request incorrectly reached the PERSONAL dispatch target -- routes are cross-dispatching")
	}
	if body["recordID"] != "rec-payroll" {
		t.Errorf("recordID = %v, want %q", body["recordID"], "rec-payroll")
	}
}

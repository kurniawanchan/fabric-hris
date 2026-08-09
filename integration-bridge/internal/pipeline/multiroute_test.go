package pipeline

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestRegisterProfileSectionRoute_TwoRoutesOnOneMux_NeverCrossDispatch is the
// real cross-dispatch proof for AC#3, using RegisterProfileSectionRoute
// directly (not buildMux, which lives in cmd/integrationbridge and is
// unreachable from this package) so distinct stub Dispatch functions can be
// substituted per route -- proving PERSONAL's handler never reaches
// PAYROLL's dispatch target or vice versa, regardless of what the caller
// does on its own side. testAuthConfig() is handlers_test.go's helper,
// reused here since both files are package pipeline.
func TestRegisterProfileSectionRoute_TwoRoutesOnOneMux_NeverCrossDispatch(t *testing.T) {
	var personalCalled, payrollCalled bool

	mux := http.NewServeMux()
	RegisterProfileSectionRoute(mux, "POST /v1/profile-sections/PERSONAL",
		RouteConfig{ProfileSection: "PERSONAL", Dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
			personalCalled = true
			return []byte(`{"recordID":"rec-personal"}`), nil
		}},
		testAuthConfig(), "tenant01", time.Second)
	RegisterProfileSectionRoute(mux, "POST /v1/profile-sections/PAYROLL",
		RouteConfig{ProfileSection: "PAYROLL", Dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
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

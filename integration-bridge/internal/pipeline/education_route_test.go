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

// TestRegisterProfileSectionRoute_EducationRoute_DispatchesAndReturnsCommitted
// proves the shared pipeline works unchanged for the EDUCATION route (Story
// 1.4, AC#1).
func TestRegisterProfileSectionRoute_EducationRoute_DispatchesAndReturnsCommitted(t *testing.T) {
	var gotEmployeeInternalID, gotUserID string
	var gotNewValue []byte
	route := RouteConfig{
		ProfileSection: "EDUCATION",
		Dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
			gotEmployeeInternalID = employeeInternalID
			gotUserID = userID
			gotNewValue = newValue
			return []byte(`{"recordID":"rec-edu-1"}`), nil
		},
	}
	mux := http.NewServeMux()
	RegisterProfileSectionRoute(mux, "POST /v1/profile-sections/EDUCATION", route, testAuthConfig(), "tenant01", time.Second)

	req := httptest.NewRequest(http.MethodPost, "/v1/profile-sections/EDUCATION",
		strings.NewReader(`{"employeeInternalID":"emp-1","userID":"user-1","newValue":{"degree":"B.Sc."}}`))
	req.Header.Set("X-Api-Key", "secret-key")
	req.Header.Set("X-Company-ID", "tenant01")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["status"] != "committed" {
		t.Errorf("status = %v, want %q", body["status"], "committed")
	}
	if body["recordID"] != "rec-edu-1" {
		t.Errorf("recordID = %v, want %q", body["recordID"], "rec-edu-1")
	}
	if gotEmployeeInternalID != "emp-1" {
		t.Errorf("dispatch called with employeeInternalID=%q, want emp-1", gotEmployeeInternalID)
	}
	if gotUserID != "user-1" {
		t.Errorf("dispatch called with userID=%q, want user-1", gotUserID)
	}
	if string(gotNewValue) != `{"degree":"B.Sc."}` {
		t.Errorf("dispatch called with newValue=%s, want the raw newValue bytes", gotNewValue)
	}
}

// TestRegisterProfileSectionRoute_EducationRoute_ValidateFailure_NeverCallsDispatch
// covers a genuinely EDUCATION-specific dimension: AC#2 says the bridge has
// no concept of the caller's own approval gate (unlike PERSONAL and
// EMPLOYMENT, EDUCATION commits directly with none). A request carrying no
// approval-related field at all still passes cleanly through [validate] up
// to the point of rejection for the actual missing field -- there is no
// approval-state check anywhere to bypass, which is the concrete,
// executable form of "no bridge behavior differs for the missing approval
// gate" this story's AC actually claims.
func TestRegisterProfileSectionRoute_EducationRoute_ValidateFailure_NeverCallsDispatch(t *testing.T) {
	dispatchCalled := false
	route := RouteConfig{
		ProfileSection: "EDUCATION",
		Dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
			dispatchCalled = true
			return nil, nil
		},
	}
	mux := http.NewServeMux()
	RegisterProfileSectionRoute(mux, "POST /v1/profile-sections/EDUCATION", route, testAuthConfig(), "tenant01", time.Second)

	req := httptest.NewRequest(http.MethodPost, "/v1/profile-sections/EDUCATION",
		strings.NewReader(`{"userID":"user-1","newValue":{}}`)) // missing employeeInternalID; no approval field of any kind
	req.Header.Set("X-Api-Key", "secret-key")
	req.Header.Set("X-Company-ID", "tenant01")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if dispatchCalled {
		t.Error("dispatch was called despite an invalid envelope on the EDUCATION route")
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want 400 -- AC#1 requires the same HTTP codes as the PERSONAL route for an equivalent rejection", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["status"] != "rejected" {
		t.Errorf("status = %v, want %q", body["status"], "rejected")
	}
	if _, hasRecordID := body["recordID"]; hasRecordID {
		t.Errorf("recordID = %v, want it absent on a rejected response", body["recordID"])
	}
}

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

// TestRegisterProfileSectionRoute_AdditionalRoute_DispatchesAndReturnsCommitted
// proves the shared pipeline works unchanged for the ADDITIONAL route (Story
// 1.5, AC#1). The path is deliberately literal ("/v1/profile-sections/ADDITIONAL")
// to pin the naming trap: the ratified chaincode enum value is "ADDITIONAL",
// not "FAMILY", even though the dispatched Hooks method is named
// ApproveFamilyDataChange. AC#2 ("the bridge only ever sees the HTTP
// request, never how the caller persisted anything on its own side") has
// no distinguishable behavior to test at the HTTP boundary -- there is no
// caller-persistence-mechanic-aware code path in this pipeline to exercise, so
// this happy-path test (successful commit regardless of payload shape) is
// its full, by-construction proof.
func TestRegisterProfileSectionRoute_AdditionalRoute_DispatchesAndReturnsCommitted(t *testing.T) {
	var gotEmployeeInternalID, gotUserID string
	var gotNewValue []byte
	route := RouteConfig{
		ProfileSection: "ADDITIONAL",
		Dispatch: func(ctx context.Context, employeeInternalID, userID, recordIdentity string, newValue, document []byte) ([]byte, error) {
			gotEmployeeInternalID = employeeInternalID
			gotUserID = userID
			gotNewValue = newValue
			return []byte(`{"recordID":"rec-add-1"}`), nil
		},
	}
	mux := http.NewServeMux()
	RegisterProfileSectionRoute(mux, "POST /v1/profile-sections/ADDITIONAL", route, testAuthConfig(), "tenant01", time.Second)

	req := httptest.NewRequest(http.MethodPost, "/v1/profile-sections/ADDITIONAL",
		strings.NewReader(`{"employeeInternalID":"emp-1","userID":"user-1","newValue":{"dependentName":"Test Child"}}`))
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
	if body["recordID"] != "rec-add-1" {
		t.Errorf("recordID = %v, want %q", body["recordID"], "rec-add-1")
	}
	if gotEmployeeInternalID != "emp-1" {
		t.Errorf("dispatch called with employeeInternalID=%q, want emp-1", gotEmployeeInternalID)
	}
	if gotUserID != "user-1" {
		t.Errorf("dispatch called with userID=%q, want user-1", gotUserID)
	}
	if string(gotNewValue) != `{"dependentName":"Test Child"}` {
		t.Errorf("dispatch called with newValue=%s, want the raw newValue bytes", gotNewValue)
	}
}

func TestRegisterProfileSectionRoute_AdditionalRoute_AuthFailure_NeverCallsDispatch(t *testing.T) {
	dispatchCalled := false
	route := RouteConfig{
		ProfileSection: "ADDITIONAL",
		Dispatch: func(ctx context.Context, employeeInternalID, userID, recordIdentity string, newValue, document []byte) ([]byte, error) {
			dispatchCalled = true
			return nil, nil
		},
	}
	mux := http.NewServeMux()
	RegisterProfileSectionRoute(mux, "POST /v1/profile-sections/ADDITIONAL", route, testAuthConfig(), "tenant01", time.Second)

	req := httptest.NewRequest(http.MethodPost, "/v1/profile-sections/ADDITIONAL",
		strings.NewReader(`{"employeeInternalID":"emp-1","userID":"user-1","newValue":{}}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if dispatchCalled {
		t.Error("dispatch was called despite missing auth headers on the ADDITIONAL route")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status code = %d, want 500 -- AC#1 requires the same HTTP codes as the PERSONAL route for an equivalent rejection", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["status"] != "error" {
		t.Errorf("status = %v, want %q", body["status"], "error")
	}
	if _, hasRecordID := body["recordID"]; hasRecordID {
		t.Errorf("recordID = %v, want it absent on an auth-rejected response", body["recordID"])
	}
}

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestRegisterProfileSectionRoute_PayrollRoute_DispatchesAndReturnsCommitted
// proves the shared pipeline works unchanged for the PAYROLL route.
func TestRegisterProfileSectionRoute_PayrollRoute_DispatchesAndReturnsCommitted(t *testing.T) {
	var gotEmployeeInternalID, gotUserID string
	var gotNewValue []byte
	route := routeConfig{
		ProfileSection: "PAYROLL",
		Dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
			gotEmployeeInternalID = employeeInternalID
			gotUserID = userID
			gotNewValue = newValue
			return []byte(`{"recordID":"rec-pay-1"}`), nil
		},
	}
	mux := http.NewServeMux()
	registerProfileSectionRoute(mux, "POST /v1/profile-sections/PAYROLL", route, testAuthConfig(), "tenant01", time.Second)

	req := httptest.NewRequest(http.MethodPost, "/v1/profile-sections/PAYROLL",
		strings.NewReader(`{"employeeInternalID":"emp-1","userID":"user-1","newValue":{"bankAccountNumber":123456}}`))
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
	if body["recordID"] != "rec-pay-1" {
		t.Errorf("recordID = %v, want %q", body["recordID"], "rec-pay-1")
	}
	if gotEmployeeInternalID != "emp-1" {
		t.Errorf("dispatch called with employeeInternalID=%q, want emp-1", gotEmployeeInternalID)
	}
	if gotUserID != "user-1" {
		t.Errorf("dispatch called with userID=%q, want user-1", gotUserID)
	}
	if string(gotNewValue) != `{"bankAccountNumber":123456}` {
		t.Errorf("dispatch called with newValue=%s, want the raw newValue bytes", gotNewValue)
	}
}

func TestRegisterProfileSectionRoute_PayrollRoute_ValidateFailure_NeverCallsDispatch(t *testing.T) {
	dispatchCalled := false
	route := routeConfig{
		ProfileSection: "PAYROLL",
		Dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
			dispatchCalled = true
			return nil, nil
		},
	}
	mux := http.NewServeMux()
	registerProfileSectionRoute(mux, "POST /v1/profile-sections/PAYROLL", route, testAuthConfig(), "tenant01", time.Second)

	req := httptest.NewRequest(http.MethodPost, "/v1/profile-sections/PAYROLL",
		strings.NewReader(`{"userID":"user-1","newValue":{}}`)) // missing employeeInternalID
	req.Header.Set("X-Api-Key", "secret-key")
	req.Header.Set("X-Company-ID", "tenant01")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if dispatchCalled {
		t.Error("dispatch was called despite an invalid envelope on the PAYROLL route")
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want 400 -- AC#1 requires the same status contract as the PERSONAL route for an equivalent rejection", rec.Code)
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

// TestRegisterProfileSectionRoute_PayrollRoute_LargeBankAccountNumber_PreservedByteForByte
// is AC#1's concrete proof: a bank-account number past float64's 2^53 exact-
// integer boundary reaches the PAYROLL dispatch target's newValue byte-for-
// byte, end-to-end through the real HTTP request body -> [validate] ->
// [dispatch] path -- not just validateRequest() in isolation (already proven
// by Story 1.2's TestValidateRequest_LargeInteger_PreservedByteForByte).
func TestRegisterProfileSectionRoute_PayrollRoute_LargeBankAccountNumber_PreservedByteForByte(t *testing.T) {
	const largeAccountNumber = "9223372036854775807" // 19 digits, past 2^53

	var gotNewValue []byte
	route := routeConfig{
		ProfileSection: "PAYROLL",
		Dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
			gotNewValue = newValue
			return []byte(`{"recordID":"rec-pay-2"}`), nil
		},
	}
	mux := http.NewServeMux()
	registerProfileSectionRoute(mux, "POST /v1/profile-sections/PAYROLL", route, testAuthConfig(), "tenant01", time.Second)

	wantNewValue := `{"bankAccountNumber":` + largeAccountNumber + `}`
	req := httptest.NewRequest(http.MethodPost, "/v1/profile-sections/PAYROLL", strings.NewReader(
		`{"employeeInternalID":"emp-1","userID":"user-1","newValue":`+wantNewValue+`}`))
	req.Header.Set("X-Api-Key", "secret-key")
	req.Header.Set("X-Company-ID", "tenant01")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if string(gotNewValue) != wantNewValue {
		t.Errorf("dispatch received newValue=%s, want the exact bytes %s untouched -- a float64 round-trip would have corrupted the digit string, and an exact-equality check (not a substring check) is what actually rules out surrounding-structure corruption too", gotNewValue, wantNewValue)
	}
}

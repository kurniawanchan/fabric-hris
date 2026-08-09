package pipeline

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	writepaths "writepaths"
)

func testAuthConfig() AuthConfig {
	return AuthConfig{APIKey: "secret-key", CompanyID: "tenant01"}
}

func newPersonalRequest(t *testing.T, body string, withAuth bool) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/profile-sections/PERSONAL", strings.NewReader(body))
	if withAuth {
		req.Header.Set("X-Api-Key", "secret-key")
		req.Header.Set("X-Company-ID", "tenant01")
	}
	return req
}

func TestRegisterProfileSectionRoute_ValidRequest_DispatchesAndReturnsCommitted(t *testing.T) {
	var gotEmployeeInternalID, gotUserID string
	var gotNewValue []byte
	route := RouteConfig{
		ProfileSection: "PERSONAL",
		Dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
			gotEmployeeInternalID = employeeInternalID
			gotUserID = userID
			gotNewValue = newValue
			return []byte(`{"recordID":"rec-1","timestamp":"t","version":1}`), nil
		},
	}
	mux := http.NewServeMux()
	RegisterProfileSectionRoute(mux, "POST /v1/profile-sections/PERSONAL", route, testAuthConfig(), "tenant01", time.Second)

	req := newPersonalRequest(t, `{"employeeInternalID":"emp-1","userID":"user-1","newValue":{"fullName":"Test"}}`, true)
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
	if body["recordID"] != "rec-1" {
		t.Errorf("recordID = %v, want %q", body["recordID"], "rec-1")
	}
	if gotEmployeeInternalID != "emp-1" || gotUserID != "user-1" {
		t.Errorf("dispatch called with employeeInternalID=%q userID=%q, want emp-1/user-1", gotEmployeeInternalID, gotUserID)
	}
	if string(gotNewValue) != `{"fullName":"Test"}` {
		t.Errorf("dispatch called with newValue=%s, want the raw newValue bytes", gotNewValue)
	}
}

func TestRegisterProfileSectionRoute_AuthFailure_NeverCallsDispatch(t *testing.T) {
	dispatchCalled := false
	route := RouteConfig{
		ProfileSection: "PERSONAL",
		Dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
			dispatchCalled = true
			return nil, nil
		},
	}
	mux := http.NewServeMux()
	RegisterProfileSectionRoute(mux, "POST /v1/profile-sections/PERSONAL", route, testAuthConfig(), "tenant01", time.Second)

	req := newPersonalRequest(t, `{"employeeInternalID":"emp-1","userID":"user-1","newValue":{}}`, false)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if dispatchCalled {
		t.Error("dispatch was called despite missing auth headers -- [auth] rejection must never reach [dispatch]")
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["status"] != "error" {
		t.Errorf("status = %v, want %q (AD-3: [auth] rejection -> error)", body["status"], "error")
	}
	if _, hasRecordID := body["recordID"]; hasRecordID {
		t.Error("recordID present on an auth-rejected response")
	}
}

func TestRegisterProfileSectionRoute_ValidateFailure_NeverCallsDispatch(t *testing.T) {
	dispatchCalled := false
	route := RouteConfig{
		ProfileSection: "PERSONAL",
		Dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
			dispatchCalled = true
			return nil, nil
		},
	}
	mux := http.NewServeMux()
	RegisterProfileSectionRoute(mux, "POST /v1/profile-sections/PERSONAL", route, testAuthConfig(), "tenant01", time.Second)

	req := newPersonalRequest(t, `{"userID":"user-1","newValue":{}}`, true) // missing employeeInternalID
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if dispatchCalled {
		t.Error("dispatch was called despite an invalid envelope -- [validate] rejection must never reach [dispatch]")
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["status"] != "rejected" {
		t.Errorf("status = %v, want %q", body["status"], "rejected")
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want 400", rec.Code)
	}
}

func TestRegisterProfileSectionRoute_PartialFailureFromDispatch_ReturnsPartialFailure(t *testing.T) {
	route := RouteConfig{
		ProfileSection: "PERSONAL",
		Dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
			return nil, &writepaths.PartialFailureError{
				EmployeeInternalID: employeeInternalID,
				ProfileSection:     "PERSONAL",
				Version:            1,
				Err:                errors.New("gateway unreachable"),
			}
		},
	}
	mux := http.NewServeMux()
	RegisterProfileSectionRoute(mux, "POST /v1/profile-sections/PERSONAL", route, testAuthConfig(), "tenant01", time.Second)

	req := newPersonalRequest(t, `{"employeeInternalID":"emp-1","userID":"user-1","newValue":{}}`, true)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["status"] != "partial_failure" {
		t.Errorf("status = %v, want %q", body["status"], "partial_failure")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want 200 (AD-3: partial_failure means the bridge processed the request)", rec.Code)
	}
}

func TestRegisterProfileSectionRoute_DispatchTimeout_ReturnsPartialFailureNotTimeout(t *testing.T) {
	route := RouteConfig{
		ProfileSection: "PERSONAL",
		Dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
			<-ctx.Done()
			return nil, errors.New("gateway: submit did not complete before the deadline")
		},
	}
	mux := http.NewServeMux()
	RegisterProfileSectionRoute(mux, "POST /v1/profile-sections/PERSONAL", route, testAuthConfig(), "tenant01", 10*time.Millisecond)

	req := newPersonalRequest(t, `{"employeeInternalID":"emp-1","userID":"user-1","newValue":{}}`, true)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["status"] != "partial_failure" {
		t.Errorf("status = %v, want %q (AC#6: never a distinct timeout status)", body["status"], "partial_failure")
	}
}

// TestRegisterProfileSectionRoute_SlowButSuccessfulDispatch_ReturnsCommitted
// is the full-pipeline regression test for the code review's Decision 1 fix:
// a Hooks call that runs past the configured budget but still returns a
// genuine success must reach the caller as "committed" with its real
// recordID -- not be discarded as "partial_failure" just because it took
// longer than the timeout.
func TestRegisterProfileSectionRoute_SlowButSuccessfulDispatch_ReturnsCommitted(t *testing.T) {
	route := RouteConfig{
		ProfileSection: "PERSONAL",
		Dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
			<-ctx.Done()
			return []byte(`{"recordID":"rec-slow-success"}`), nil
		},
	}
	mux := http.NewServeMux()
	RegisterProfileSectionRoute(mux, "POST /v1/profile-sections/PERSONAL", route, testAuthConfig(), "tenant01", 10*time.Millisecond)

	req := newPersonalRequest(t, `{"employeeInternalID":"emp-1","userID":"user-1","newValue":{}}`, true)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["status"] != "committed" {
		t.Errorf("status = %v, want %q -- a genuine success must survive even after the budget elapsed", body["status"], "committed")
	}
	if body["recordID"] != "rec-slow-success" {
		t.Errorf("recordID = %v, want %q", body["recordID"], "rec-slow-success")
	}
}

// TestRegisterProfileSectionRoute_NeverLeaksNewValueOrDocumentBytesInResponse
// is AC#7's concrete proof: across every reachable status outcome, the raw
// bytes of a request's newValue/document never appear in the HTTP response
// body. There is no logging sink anywhere in this codebase yet for a
// per-request event, so the response body is the only place a leak could
// surface today.
func TestRegisterProfileSectionRoute_NeverLeaksNewValueOrDocumentBytesInResponse(t *testing.T) {
	const newValueMarker = "SENSITIVE_FULLNAME_MARKER_998877"
	const documentMarker = "SENSITIVE_DOCUMENT_MARKER_665544"
	documentB64 := base64.StdEncoding.EncodeToString([]byte(documentMarker))

	cases := []struct {
		name     string
		body     string
		auth     bool
		dispatch DispatchFunc
	}{
		{
			name: "committed",
			body: fmt.Sprintf(`{"employeeInternalID":"emp-1","userID":"user-1","newValue":{"fullName":%q},"document":%q}`, newValueMarker, documentB64),
			auth: true,
			dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
				return []byte(`{"recordID":"rec-1"}`), nil
			},
		},
		{
			name: "rejected_missing_employeeInternalID",
			body: fmt.Sprintf(`{"userID":"user-1","newValue":{"fullName":%q},"document":%q}`, newValueMarker, documentB64),
			auth: true,
			dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
				t.Fatal("dispatch must not be called for a rejected request")
				return nil, nil
			},
		},
		{
			name: "auth_error",
			body: fmt.Sprintf(`{"employeeInternalID":"emp-1","userID":"user-1","newValue":{"fullName":%q},"document":%q}`, newValueMarker, documentB64),
			auth: false,
			dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
				t.Fatal("dispatch must not be called for an auth-rejected request")
				return nil, nil
			},
		},
		{
			name: "partial_failure",
			body: fmt.Sprintf(`{"employeeInternalID":"emp-1","userID":"user-1","newValue":{"fullName":%q},"document":%q}`, newValueMarker, documentB64),
			auth: true,
			dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
				return nil, &writepaths.PartialFailureError{
					EmployeeInternalID: employeeInternalID,
					ProfileSection:     "PERSONAL",
					Version:            1,
					Err:                errors.New("gateway unreachable"),
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			route := RouteConfig{ProfileSection: "PERSONAL", Dispatch: tc.dispatch}
			mux := http.NewServeMux()
			RegisterProfileSectionRoute(mux, "POST /v1/profile-sections/PERSONAL", route, testAuthConfig(), "tenant01", time.Second)

			req := newPersonalRequest(t, tc.body, tc.auth)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			respBody := rec.Body.String()
			if strings.Contains(respBody, newValueMarker) {
				t.Errorf("response body contains the raw newValue marker: %s", respBody)
			}
			if strings.Contains(respBody, documentMarker) {
				t.Errorf("response body contains the raw document marker: %s", respBody)
			}
			if strings.Contains(respBody, documentB64) {
				t.Errorf("response body contains the base64-encoded document: %s", respBody)
			}
		})
	}
}

func TestRegisterProfileSectionRoute_WrongMethod_Returns405(t *testing.T) {
	route := RouteConfig{
		ProfileSection: "PERSONAL",
		Dispatch: func(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
			return nil, nil
		},
	}
	mux := http.NewServeMux()
	RegisterProfileSectionRoute(mux, "POST /v1/profile-sections/PERSONAL", route, testAuthConfig(), "tenant01", time.Second)

	req := httptest.NewRequest(http.MethodGet, "/v1/profile-sections/PERSONAL", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want 405 (stdlib ServeMux's own method-restricted registration, AD-6)", rec.Code)
	}
}

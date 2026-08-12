package pipeline

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gatewayclient "gatewayclient"
)

// stubSaltReader is a minimal SaltReader double -- same dependency-inversion
// technique as stubKeyResolver/stubLedgerReader (history_route_test.go).
type stubSaltReader struct {
	salt  []byte
	err   error
	calls int
}

func (s *stubSaltReader) GetSalt(_ context.Context, employeeID, profileSection string, version int) ([]byte, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.salt, nil
}

func newVerifyMux(keys EmployeeKeyResolver, salts SaltReader, ledger LedgerHistoryReader) *http.ServeMux {
	mux := http.NewServeMux()
	RegisterProfileVerifyRoute(mux, "POST /v1/profile-sections/verify",
		VerifyRouteConfig{Keys: keys, Salts: salts, Ledger: ledger, TenantID: "tenant01"}, testAuthConfig())
	return mux
}

func newVerifyRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/v1/profile-sections/verify", strings.NewReader(body))
	req.Header.Set("X-Api-Key", "secret-key")
	req.Header.Set("X-Company-ID", "tenant01")
	return req
}

func TestRegisterProfileVerifyRoute_MatchingDigest_ReportsVerified(t *testing.T) {
	keys := &stubKeyResolver{key: []byte(testEmployeeKey)}
	salt := []byte("0123456789abcdef")
	salts := &stubSaltReader{salt: salt}

	currentValue := []byte(`{"fullName":"Test Employee"}`)
	dataHash, err := gatewayclient.ComputeDataHash(salt, currentValue)
	if err != nil {
		t.Fatalf("ComputeDataHash: %v", err)
	}
	historyJSON, _ := json.Marshal([]chaincodeHistoryEntry{
		{DataHash: dataHash, Version: 1},
	})
	ledger := &stubLedgerReader{sectionResults: map[string][]byte{"PERSONAL": historyJSON}}

	mux := newVerifyMux(keys, salts, ledger)
	req := newVerifyRequest(`{"employeeInternalID":"emp-1","profileSection":"PERSONAL","currentValue":` + string(currentValue) + `}`)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["status"] != "verified" {
		t.Errorf("status = %v, want %q", body["status"], "verified")
	}
}

func TestRegisterProfileVerifyRoute_MismatchedDigest_ReportsCompromised(t *testing.T) {
	keys := &stubKeyResolver{key: []byte(testEmployeeKey)}
	salts := &stubSaltReader{salt: []byte("0123456789abcdef")}

	historyJSON, _ := json.Marshal([]chaincodeHistoryEntry{
		{DataHash: "some-hash-that-will-never-match", Version: 1},
	})
	ledger := &stubLedgerReader{sectionResults: map[string][]byte{"PERSONAL": historyJSON}}

	mux := newVerifyMux(keys, salts, ledger)
	req := newVerifyRequest(`{"employeeInternalID":"emp-1","profileSection":"PERSONAL","currentValue":{"fullName":"Tampered"}}`)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["status"] != "compromised" {
		t.Errorf("status = %v, want %q", body["status"], "compromised")
	}
}

func TestRegisterProfileVerifyRoute_OmittedCurrentValue_ReportsDeletionVerified_NeverCallsSalts(t *testing.T) {
	keys := &stubKeyResolver{key: []byte(testEmployeeKey)}
	salts := &stubSaltReader{salt: []byte("0123456789abcdef")}

	historyJSON, _ := json.Marshal([]chaincodeHistoryEntry{
		{DataHash: "irrelevant", Version: 1},
	})
	ledger := &stubLedgerReader{sectionResults: map[string][]byte{"PERSONAL": historyJSON}}

	mux := newVerifyMux(keys, salts, ledger)
	req := newVerifyRequest(`{"employeeInternalID":"emp-1","profileSection":"PERSONAL"}`)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["status"] != "deletion_verified" {
		t.Errorf("status = %v, want %q", body["status"], "deletion_verified")
	}
	if salts.calls != 0 {
		t.Errorf("Salts.GetSalt called %d times, want 0 -- a deletion assertion must never attempt a hash-compare", salts.calls)
	}
}

func TestRegisterProfileVerifyRoute_NoAnchorExists_ReportsNoAnchorFound(t *testing.T) {
	keys := &stubKeyResolver{key: []byte(testEmployeeKey)}
	salts := &stubSaltReader{salt: []byte("0123456789abcdef")}
	ledger := &stubLedgerReader{} // no sectionResults configured -- empty history

	mux := newVerifyMux(keys, salts, ledger)
	req := newVerifyRequest(`{"employeeInternalID":"emp-1","profileSection":"PERSONAL","currentValue":{"fullName":"New Employee"}}`)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["status"] != "no_anchor_found" {
		t.Errorf("status = %v, want %q", body["status"], "no_anchor_found")
	}
}

func TestRegisterProfileVerifyRoute_AuthFailure_NeverCallsDependencies(t *testing.T) {
	keys := &stubKeyResolver{key: []byte(testEmployeeKey)}
	salts := &stubSaltReader{salt: []byte("0123456789abcdef")}
	ledger := &stubLedgerReader{}

	mux := newVerifyMux(keys, salts, ledger)
	req := httptest.NewRequest(http.MethodPost, "/v1/profile-sections/verify",
		strings.NewReader(`{"employeeInternalID":"emp-1","profileSection":"PERSONAL"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if len(keys.calls) != 0 || len(ledger.calledSections) != 0 || salts.calls != 0 {
		t.Error("[auth] rejection must never reach Keys/Salts/Ledger")
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["status"] != "error" {
		t.Errorf("status = %v, want %q", body["status"], "error")
	}
}

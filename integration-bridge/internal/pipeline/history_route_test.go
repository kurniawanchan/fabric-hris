package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	gatewayclient "gatewayclient"
	writepaths "writepaths"
)

// stubKeyResolver is a minimal EmployeeKeyResolver double -- mirrors the
// dependency-inversion technique already used throughout this package (e.g.
// handlers_test.go's route.Dispatch stubs).
type stubKeyResolver struct {
	key   []byte
	err   error
	calls []string
}

func (s *stubKeyResolver) GetOrCreateEmployeeKey(_ context.Context, employeeInternalID string) ([]byte, error) {
	s.calls = append(s.calls, employeeInternalID)
	if s.err != nil {
		return nil, s.err
	}
	return s.key, nil
}

// stubLedgerReader is a minimal LedgerHistoryReader double. sectionResults
// maps a profileSection to the raw ledger JSON EvaluateGetProfileHistory
// should return for it; an unlisted section returns an empty history (the
// real chaincode's own "never anchored yet" shape).
type stubLedgerReader struct {
	sectionResults map[string][]byte
	err            error
	calledSections []string
	calledEmployee []string
}

func (s *stubLedgerReader) EvaluateGetProfileHistory(_ context.Context, tenantID, employeeID, profileSection string) ([]byte, error) {
	s.calledSections = append(s.calledSections, profileSection)
	s.calledEmployee = append(s.calledEmployee, employeeID)
	if s.err != nil {
		return nil, s.err
	}
	if raw, ok := s.sectionResults[profileSection]; ok {
		return raw, nil
	}
	return []byte(`[]`), nil
}

const testEmployeeKey = "0123456789abcdef" // 16 bytes -- meets keystore.MinKeyBytes

func newHistoryMux(keys EmployeeKeyResolver, ledger LedgerHistoryReader) *http.ServeMux {
	mux := http.NewServeMux()
	RegisterProfileHistoryRoute(mux, "GET /v1/profile-sections/history",
		HistoryRouteConfig{Keys: keys, Ledger: ledger, TenantID: "tenant01"}, testAuthConfig())
	return mux
}

func newHistoryRequest(query string, withAuth bool) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/v1/profile-sections/history?"+query, nil)
	if withAuth {
		req.Header.Set("X-Api-Key", "secret-key")
		req.Header.Set("X-Company-ID", "tenant01")
	}
	return req
}

func TestRegisterProfileHistoryRoute_AuthFailure_NeverCallsKeysOrLedger(t *testing.T) {
	keys := &stubKeyResolver{key: []byte(testEmployeeKey)}
	ledger := &stubLedgerReader{}
	mux := newHistoryMux(keys, ledger)

	req := newHistoryRequest("employeeInternalID=emp-1", false)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if len(keys.calls) != 0 || len(ledger.calledSections) != 0 {
		t.Error("[auth] rejection must never reach Keys/Ledger")
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["status"] != "error" {
		t.Errorf("status = %v, want %q", body["status"], "error")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status code = %d, want 500", rec.Code)
	}
}

func TestRegisterProfileHistoryRoute_MissingEmployeeInternalID_Rejected(t *testing.T) {
	keys := &stubKeyResolver{key: []byte(testEmployeeKey)}
	ledger := &stubLedgerReader{}
	mux := newHistoryMux(keys, ledger)

	req := newHistoryRequest("", true)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if len(keys.calls) != 0 || len(ledger.calledSections) != 0 {
		t.Error("a rejected request must never reach Keys/Ledger")
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

func TestRegisterProfileHistoryRoute_UnknownProfileSection_Rejected(t *testing.T) {
	keys := &stubKeyResolver{key: []byte(testEmployeeKey)}
	ledger := &stubLedgerReader{}
	mux := newHistoryMux(keys, ledger)

	req := newHistoryRequest("employeeInternalID=emp-1&profileSection=BOGUS", true)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if len(keys.calls) != 0 || len(ledger.calledSections) != 0 {
		t.Error("a rejected request must never reach Keys/Ledger")
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["status"] != "rejected" {
		t.Errorf("status = %v, want %q", body["status"], "rejected")
	}
}

func TestRegisterProfileHistoryRoute_SingleSection_QueriesOnlyThatSection(t *testing.T) {
	keys := &stubKeyResolver{key: []byte(testEmployeeKey)}
	ledger := &stubLedgerReader{
		sectionResults: map[string][]byte{
			"PERSONAL": []byte(`[{"canonicalizationVersion":"JCS-1","dataHash":"sha256:aaa","hashAlgo":"SHA-256","ipfsCIDs":[],"prevHash":"","recordID":"rec-1","timestamp":"2026-08-01T00:00:00Z","updatedBy":"hmac-1","version":1}]`),
		},
	}
	mux := newHistoryMux(keys, ledger)

	req := newHistoryRequest("employeeInternalID=emp-1&profileSection=PERSONAL", true)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if len(ledger.calledSections) != 1 || ledger.calledSections[0] != "PERSONAL" {
		t.Errorf("Ledger called with sections %v, want exactly [PERSONAL]", ledger.calledSections)
	}
	if len(keys.calls) != 1 || keys.calls[0] != "emp-1" {
		t.Errorf("Keys called with %v, want exactly [emp-1]", keys.calls)
	}

	wantEmployeeID, err := gatewayclient.ComputeEmployeeID([]byte(testEmployeeKey))
	if err != nil {
		t.Fatalf("ComputeEmployeeID: %v", err)
	}
	if ledger.calledEmployee[0] != wantEmployeeID {
		t.Errorf("Ledger queried with employeeID=%q, want the pseudonym derived from the SAME employeeKey_i (%q) the write path would derive", ledger.calledEmployee[0], wantEmployeeID)
	}

	var body historyResponseBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("status = %q, want %q", body.Status, "ok")
	}
	if len(body.History) != 1 {
		t.Fatalf("History has %d entries, want 1", len(body.History))
	}
	if body.History[0].ProfileSection != "PERSONAL" {
		t.Errorf("History[0].ProfileSection = %q, want %q", body.History[0].ProfileSection, "PERSONAL")
	}
	if body.History[0].RecordID != "rec-1" {
		t.Errorf("History[0].RecordID = %q, want %q", body.History[0].RecordID, "rec-1")
	}
}

func TestRegisterProfileHistoryRoute_NoSectionRequested_FansOutToAllFiveAndMerges(t *testing.T) {
	keys := &stubKeyResolver{key: []byte(testEmployeeKey)}
	ledger := &stubLedgerReader{
		sectionResults: map[string][]byte{
			"PERSONAL": []byte(`[{"recordID":"rec-personal-1","timestamp":"2026-08-01T00:00:00Z","version":1}]`),
			"PAYROLL":  []byte(`[{"recordID":"rec-payroll-1","timestamp":"2026-08-03T00:00:00Z","version":1}]`),
			// EMPLOYMENT/EDUCATION/ADDITIONAL deliberately absent -- "never
			// anchored yet" must contribute zero entries, not an error.
		},
	}
	mux := newHistoryMux(keys, ledger)

	req := newHistoryRequest("employeeInternalID=emp-1", true)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if len(ledger.calledSections) != len(writepaths.AllProfileSections) {
		t.Errorf("Ledger called %d times, want once per ratified section (%d): %v", len(ledger.calledSections), len(writepaths.AllProfileSections), ledger.calledSections)
	}

	var body historyResponseBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if len(body.History) != 2 {
		t.Fatalf("History has %d entries, want 2 (only PERSONAL and PAYROLL ever anchored)", len(body.History))
	}
	// Most-recent-first: PAYROLL (2026-08-03) before PERSONAL (2026-08-01).
	if body.History[0].RecordID != "rec-payroll-1" || body.History[1].RecordID != "rec-personal-1" {
		t.Errorf("History order = [%s, %s], want most-recent-first [rec-payroll-1, rec-personal-1]", body.History[0].RecordID, body.History[1].RecordID)
	}
}

func TestRegisterProfileHistoryRoute_KeyResolverError_ReturnsError(t *testing.T) {
	keys := &stubKeyResolver{err: errors.New("keystore: unavailable")}
	ledger := &stubLedgerReader{}
	mux := newHistoryMux(keys, ledger)

	req := newHistoryRequest("employeeInternalID=emp-1", true)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if len(ledger.calledSections) != 0 {
		t.Error("a Keys failure must never reach Ledger")
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["status"] != "error" {
		t.Errorf("status = %v, want %q", body["status"], "error")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status code = %d, want 500", rec.Code)
	}
}

func TestRegisterProfileHistoryRoute_LedgerError_ReturnsError(t *testing.T) {
	keys := &stubKeyResolver{key: []byte(testEmployeeKey)}
	ledger := &stubLedgerReader{err: errors.New("gateway: unreachable")}
	mux := newHistoryMux(keys, ledger)

	req := newHistoryRequest("employeeInternalID=emp-1&profileSection=PERSONAL", true)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["status"] != "error" {
		t.Errorf("status = %v, want %q", body["status"], "error")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status code = %d, want 500", rec.Code)
	}
}

func TestRegisterProfileHistoryRoute_NeverAnchoredYet_ReturnsOkWithEmptyHistory(t *testing.T) {
	keys := &stubKeyResolver{key: []byte(testEmployeeKey)}
	ledger := &stubLedgerReader{} // no sectionResults configured -- every section returns "[]"
	mux := newHistoryMux(keys, ledger)

	req := newHistoryRequest("employeeInternalID=emp-1&profileSection=PERSONAL", true)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200 -- an employee with no anchored history yet is not an error", rec.Code)
	}
	var body historyResponseBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("status = %q, want %q", body.Status, "ok")
	}
	if len(body.History) != 0 {
		t.Errorf("History has %d entries, want 0", len(body.History))
	}
}

func TestRegisterProfileHistoryRoute_WrongMethod_Returns405(t *testing.T) {
	keys := &stubKeyResolver{key: []byte(testEmployeeKey)}
	ledger := &stubLedgerReader{}
	mux := newHistoryMux(keys, ledger)

	req := httptest.NewRequest(http.MethodPost, "/v1/profile-sections/history", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want 405 (stdlib ServeMux's own method-restricted registration)", rec.Code)
	}
}

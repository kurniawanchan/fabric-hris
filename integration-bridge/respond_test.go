package main

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
)

func TestRespond_Committed_IncludesRecordID(t *testing.T) {
	rec := httptest.NewRecorder()
	result := []byte(`{"recordID":"rec-123","timestamp":"2026-08-08T00:00:00Z","version":1}`)

	respond(rec, "committed", result, nil)

	if rec.Code != 200 {
		t.Errorf("status code = %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v (body=%s)", err, rec.Body.String())
	}
	if body["status"] != "committed" {
		t.Errorf("status = %v, want %q", body["status"], "committed")
	}
	if body["recordID"] != "rec-123" {
		t.Errorf("recordID = %v, want %q", body["recordID"], "rec-123")
	}
	if _, hasDetail := body["detail"]; hasDetail {
		t.Errorf("detail = %v, want it absent on a committed response with no error", body["detail"])
	}
}

// TestRespond_Committed_MalformedResultBytes_ReportsErrorNotCommitted is the
// fix for a code review finding: silently downgrading to
// {"status":"committed"} with no recordID and no signal hid a real
// invariant violation (fn succeeded but returned garbage). This can only
// happen from a bug in a Hooks/dispatchFunc implementation, never from live
// chaincode -- respond surfaces it as status "error" instead of pretending
// the write is fine.
func TestRespond_Committed_MalformedResultBytes_ReportsErrorNotCommitted(t *testing.T) {
	rec := httptest.NewRecorder()

	respond(rec, "committed", []byte("not json"), nil)

	if rec.Code != 500 {
		t.Errorf("status code = %d, want 500", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v (body=%s)", err, rec.Body.String())
	}
	if body["status"] != "error" {
		t.Errorf("status = %v, want %q -- an unparseable result on the committed path is an invariant violation, not a clean success", body["status"], "error")
	}
	if _, hasRecordID := body["recordID"]; hasRecordID {
		t.Errorf("recordID = %v, want it absent", body["recordID"])
	}
	if body["detail"] == "" {
		t.Error("detail is empty, want an explanation of what went wrong")
	}
}

func TestRespond_Committed_EmptyRecordID_ReportsErrorNotCommitted(t *testing.T) {
	rec := httptest.NewRecorder()

	respond(rec, "committed", []byte(`{"recordID":""}`), nil)

	if rec.Code != 500 {
		t.Errorf("status code = %d, want 500", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v (body=%s)", err, rec.Body.String())
	}
	if body["status"] != "error" {
		t.Errorf("status = %v, want %q -- a committed result with no recordID is an invariant violation", body["status"], "error")
	}
}

func TestRespond_Rejected_NoRecordID_DetailFromErr(t *testing.T) {
	rec := httptest.NewRecorder()
	wantErr := &validationError{errors.New("integrationbridge: employeeInternalID is required")}

	respond(rec, "rejected", nil, wantErr)

	if rec.Code != 400 {
		t.Errorf("status code = %d, want 400", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v (body=%s)", err, rec.Body.String())
	}
	if body["status"] != "rejected" {
		t.Errorf("status = %v, want %q", body["status"], "rejected")
	}
	if _, hasRecordID := body["recordID"]; hasRecordID {
		t.Errorf("recordID = %v, want it absent on a rejected response", body["recordID"])
	}
	if body["detail"] != wantErr.Error() {
		t.Errorf("detail = %v, want %q", body["detail"], wantErr.Error())
	}
}

func TestRespond_HTTPStatusCode_MapsFromBodyStatus(t *testing.T) {
	cases := []struct {
		status   string
		result   []byte
		wantCode int
	}{
		{"committed", []byte(`{"recordID":"rec-1"}`), 200},
		{"partial_failure", nil, 200},
		{"rejected", nil, 400},
		{"error", nil, 500},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		respond(rec, tc.status, tc.result, errors.New("x"))
		if rec.Code != tc.wantCode {
			t.Errorf("respond(%q) status code = %d, want %d", tc.status, rec.Code, tc.wantCode)
		}
	}
}

func TestRespond_SetsJSONContentType(t *testing.T) {
	rec := httptest.NewRecorder()
	respond(rec, "committed", []byte(`{"recordID":"r"}`), nil)

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

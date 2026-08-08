//go:build integration

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// TestIntegration_PayrollRoute_CommitsWithLargeBankAccountNumber proves
// main.go wires Hooks.UpdatePayrollBankAccount end-to-end and that a real
// RecordProfileSection submission with a bank-account number past 2^53
// still produces a real commit. This does NOT prove the digit string
// survived the real commit byte-for-byte -- confirming that would need a
// read-back path this bridge deliberately doesn't expose (grounding-gaps.md
// G-36, same pseudonym-opacity family as G-34); the byte-precision
// mechanism itself is proven at the unit level by
// TestRegisterProfileSectionRoute_PayrollRoute_LargeBankAccountNumber_PreservedByteForByte.
// Not executed this session (no live network, confirmed via `docker ps`).
func TestIntegration_PayrollRoute_CommitsWithLargeBankAccountNumber(t *testing.T) {
	httpAddr := "127.0.0.1:18087"
	startPersonalRouteTestBridge(t, httpAddr)

	status, body := postSection(t, httpAddr, "PAYROLL", map[string]any{
		"employeeInternalID": uniqueTestID("integration-test-emp"),
		"userID":             uniqueTestID("integration-test-user"),
		"newValue":           map[string]any{"bankAccountNumber": 9223372036854775807},
	})

	if status != http.StatusOK {
		t.Fatalf("status code = %d, want 200; body=%+v", status, body)
	}
	if body["status"] != "committed" {
		t.Fatalf("status = %v, want %q; body=%+v", body["status"], "committed", body)
	}
	if recordID, _ := body["recordID"].(string); recordID == "" {
		t.Errorf("recordID missing or empty in a committed response: %+v", body)
	}
}

// TestIntegration_UnregisteredSectionName_Is404 proves AC#2 against the
// real, fully-assembled binary (not just buildMux in isolation). Built
// without postSection since a real 404 response from net/http's default
// NotFoundHandler is plain text, not the JSON envelope postSection expects
// to decode.
func TestIntegration_UnregisteredSectionName_Is404(t *testing.T) {
	httpAddr := "127.0.0.1:18088"
	startPersonalRouteTestBridge(t, httpAddr)

	body, err := json.Marshal(map[string]any{
		"employeeInternalID": uniqueTestID("integration-test-emp"),
		"userID":             uniqueTestID("integration-test-user"),
		"newValue":           map[string]any{},
	})
	if err != nil {
		t.Fatalf("marshaling request payload: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, "http://"+httpAddr+"/v1/profile-sections/BOGUS", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	req.Header.Set("X-Api-Key", "integration-test-key")
	req.Header.Set("X-Company-ID", "tenant01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /v1/profile-sections/BOGUS: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("POST /v1/profile-sections/BOGUS: status code = %d, want 404", resp.StatusCode)
	}
}

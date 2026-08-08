//go:build integration

package main

import (
	"net/http"
	"testing"
)

// TestIntegration_EducationRoute_Commits proves main.go wires
// Hooks.RecordEducationHistory for the EDUCATION path end-to-end. Not
// executed this session (no live network, confirmed via `docker ps`).
func TestIntegration_EducationRoute_Commits(t *testing.T) {
	httpAddr := "127.0.0.1:18085"
	startPersonalRouteTestBridge(t, httpAddr)

	status, body := postSection(t, httpAddr, "EDUCATION", map[string]any{
		"employeeInternalID": uniqueTestID("integration-test-emp"),
		"userID":             uniqueTestID("integration-test-user"),
		"newValue":           map[string]any{"degree": "B.Sc. Integration Testing"},
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

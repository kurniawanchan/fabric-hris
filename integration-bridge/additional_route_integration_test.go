//go:build integration

package main

import (
	"net/http"
	"testing"
)

// TestIntegration_AdditionalRoute_Commits proves main.go wires
// Hooks.ApproveFamilyDataChange for the ADDITIONAL path end-to-end. Not
// executed this session (no live network, confirmed via `docker ps`).
func TestIntegration_AdditionalRoute_Commits(t *testing.T) {
	httpAddr := "127.0.0.1:18086"
	startPersonalRouteTestBridge(t, httpAddr)

	status, body := postSection(t, httpAddr, "ADDITIONAL", map[string]any{
		"employeeInternalID": uniqueTestID("integration-test-emp"),
		"userID":             uniqueTestID("integration-test-user"),
		"newValue":           map[string]any{"dependentName": "Integration Test Dependent"},
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

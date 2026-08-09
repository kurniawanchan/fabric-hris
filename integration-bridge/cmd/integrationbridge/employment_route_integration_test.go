//go:build integration

package main

import (
	"net/http"
	"testing"
)

// TestIntegration_EmploymentRoute_Commits proves main.go actually wires
// Hooks.ApproveEmploymentTransfer for the EMPLOYMENT path -- something no
// unit test can prove, since main.go's route table has no injectable seam
// (a deliberate choice, matching Story 1.2's own scope boundary against
// adding test-only seams for a one-line wiring addition). Not executed this
// session -- same disclosed reason as personal_route_integration_test.go
// (no live network; confirmed via `docker ps`).
func TestIntegration_EmploymentRoute_Commits(t *testing.T) {
	httpAddr := "127.0.0.1:18084"
	startPersonalRouteTestBridge(t, httpAddr)

	status, body := postSection(t, httpAddr, "EMPLOYMENT", map[string]any{
		"employeeInternalID": uniqueTestID("integration-test-emp"),
		"userID":             uniqueTestID("integration-test-user"),
		"newValue":           map[string]any{"newTitle": "Senior Integration Test Engineer"},
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

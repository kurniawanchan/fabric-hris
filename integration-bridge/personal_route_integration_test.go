//go:build integration

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// uniqueTestID appends a per-process-run suffix to base, so repeated runs
// against the live, append-only ledger (docs/QUICKSTART.md's own caution:
// re-running against it is not free) don't all write under the exact same
// identifiers indefinitely. Not a strong uniqueness guarantee (no
// concurrent-run dedup) -- just enough that two separate `go test` runs
// don't collide (code review finding).
func uniqueTestID(base string) string {
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
}

// startPersonalRouteTestBridge builds and runs the real integration-bridge
// binary against the live network from docs/QUICKSTART.md, waits for its
// HTTP listener, and registers cleanup. Deliberately not shared with
// main_integration_test.go's own startup block -- that test is already
// reviewed/finalized (Story 1.1); duplicating a few lines here is cheaper
// than risking a regression in it for a DRY refactor this story doesn't need.
func startPersonalRouteTestBridge(t *testing.T, httpAddr string) {
	t.Helper()
	binPath := filepath.Join(t.TempDir(), "integrationbridge")
	build := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	cmd := exec.Command(binPath)
	cmd.Env = append(os.Environ(),
		"BRIDGE_PEER_ENDPOINT=localhost:7051",
		"BRIDGE_TLS_SERVER_NAME=peer0.org1",
		"BRIDGE_TLS_CA_CERT_PATH="+netDir+"/crypto-config/peerOrganizations/org1/tlsca/tlsca.org1-cert.pem",
		"BRIDGE_CLIENT_TLS_CERT_PATH="+netDir+"/crypto-config/peerOrganizations/org1/users/Admin@org1/tls/client.crt",
		"BRIDGE_CLIENT_TLS_KEY_PATH="+netDir+"/crypto-config/peerOrganizations/org1/users/Admin@org1/tls/client.key",
		"BRIDGE_MSP_ID=Org1MSP",
		"BRIDGE_SIGN_CERT_PATH="+netDir+"/crypto-config/peerOrganizations/org1/users/Admin@org1/msp/signcerts/Admin@org1-cert.pem",
		"BRIDGE_SIGN_KEY_PATH="+netDir+"/crypto-config/peerOrganizations/org1/users/Admin@org1/msp/keystore/priv_sk",
		"BRIDGE_CHANNEL_NAME=tenant-tenant01",
		"BRIDGE_CHAINCODE_NAME=employeeprofilerecord",
		"BRIDGE_TENANT_ID=tenant01",
		"BRIDGE_IPFS_PRIMARY_API=127.0.0.1:5001",
		"BRIDGE_IPFS_REPLICA_API=127.0.0.1:5002",
		"BRIDGE_API_KEY=integration-test-key",
		"BRIDGE_COMPANY_ID=tenant01",
		"BRIDGE_HTTP_ADDR="+httpAddr,
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting integration-bridge: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", httpAddr, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("integration-bridge never opened its HTTP listener within 10s")
}

// postSection posts payload to the given AD-5 profile-section route
// (e.g. "PERSONAL", "EMPLOYMENT") and decodes the JSON response envelope --
// shared by every route's own `//go:build integration` test file so the
// request-building/decoding boilerplate isn't duplicated per route.
func postSection(t *testing.T, httpAddr, section string, payload map[string]any) (int, map[string]any) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshaling request payload: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, "http://"+httpAddr+"/v1/profile-sections/"+section, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	req.Header.Set("X-Api-Key", "integration-test-key")
	req.Header.Set("X-Company-ID", "tenant01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /v1/profile-sections/%s: %v", section, err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body (status %d): %v", resp.StatusCode, err)
	}

	var respBody map[string]any
	if err := json.Unmarshal(rawBody, &respBody); err != nil {
		// A non-2xx response (e.g. a plain-text 404 from net/http's default
		// NotFoundHandler) isn't JSON at all -- surfacing the status code and
		// raw body here is what tells a future caller "this wasn't ever going
		// to decode" instead of a bare, unexplained JSON-syntax error.
		t.Fatalf("decoding response body (status %d, body=%q): %v", resp.StatusCode, rawBody, err)
	}
	return resp.StatusCode, respBody
}

// TestIntegration_PersonalRoute_CommitsWithoutDocument exercises the real
// PERSONAL route end-to-end: a real RecordProfileSection submission
// producing a real recordID. No document is carried, so
// Hooks.IPFS.EncryptAndAdd is never invoked (writepaths.go's own
// `if len(document) > 0` guard) -- this is the AC#1 happy path.
func TestIntegration_PersonalRoute_CommitsWithoutDocument(t *testing.T) {
	httpAddr := "127.0.0.1:18082"
	startPersonalRouteTestBridge(t, httpAddr)

	status, body := postSection(t, httpAddr, "PERSONAL", map[string]any{
		"employeeInternalID": uniqueTestID("integration-test-emp"),
		"userID":             uniqueTestID("integration-test-user"),
		"newValue":           map[string]any{"fullName": "Integration Test Employee"},
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

// TestIntegration_PersonalRoute_CommitsWithDocument_NoPanic proves AC#8's
// safety property end-to-end: a request carrying a base64 document reaches
// doAnchor's h.IPFS.EncryptAndAdd call (Story 1.2's Dev Notes flagged this
// as a guaranteed nil-pointer panic before Task 1 wired a real
// ipfsclient.Client) and the request still commits successfully.
//
// What this test does NOT prove: that the resulting CID actually lands in
// the new version's ipfsCIDs field on the ledger. Confirming that requires
// querying GetProfileHistory with the on-chain `employeeID` pseudonym
// (gatewayclient.EvaluateGetProfileHistory's second argument) -- but that
// pseudonym is computed from employeeKey_i inside the bridge process's own
// in-memory keystore.EmployeeKeyStore and is never returned to any caller,
// by design (the entire point of the pseudonymization scheme is that an
// external caller cannot derive it). A black-box HTTP test -- which is
// exactly what a real caller of this API would be -- has no way to compute
// or obtain that value either. Recorded as grounding-gaps.md G-34 rather than
// silently claimed as covered.
func TestIntegration_PersonalRoute_CommitsWithDocument_NoPanic(t *testing.T) {
	httpAddr := "127.0.0.1:18083"
	startPersonalRouteTestBridge(t, httpAddr)

	documentB64 := base64.StdEncoding.EncodeToString([]byte("integration test supporting document"))
	status, body := postSection(t, httpAddr, "PERSONAL", map[string]any{
		"employeeInternalID": uniqueTestID("integration-test-emp"),
		"userID":             uniqueTestID("integration-test-user"),
		"newValue":           map[string]any{"fullName": "Integration Test Employee With Document"},
		"document":           documentB64,
	})

	if status != http.StatusOK {
		t.Fatalf("status code = %d, want 200; body=%+v", status, body)
	}
	if body["status"] != "committed" {
		t.Fatalf("status = %v, want %q; body=%+v", body["status"], "committed", body)
	}
}

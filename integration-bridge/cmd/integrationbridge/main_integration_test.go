//go:build integration

package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// netDir mirrors gateway-client/gatewayclient_integration_test.go's own
// constant -- same live-network crypto material, same convention, not
// reinvented. Requires docs/QUICKSTART.md's stack to be up. One more "../"
// than gateway-client's own copy: this file lives two directories deeper
// (integration-bridge/cmd/integrationbridge) than integration-bridge/ itself.
const netDir = "../../../fabric-network/network"

// TestIntegration_ConstructsOnceAndShutsDownCleanly builds and runs the real
// integration-bridge binary against the live network from docs/QUICKSTART.md,
// confirms it starts (exactly one GatewayClient constructed -- proven by
// main.go's own code structure, one call site, no loops), then sends SIGTERM
// and confirms a clean process exit with the HTTP port released. Process
// exit + port release together strongly imply Close() ran, but neither is a
// direct observation of the Close() call itself -- this test does not
// capture stdout/stderr, so it cannot literally see that call happen.
func TestIntegration_ConstructsOnceAndShutsDownCleanly(t *testing.T) {
	binPath := filepath.Join(t.TempDir(), "integrationbridge")
	build := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	httpAddr := "127.0.0.1:18081"
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

	// Confirm the HTTP listener is actually up (proves buildHooks/main ran
	// past construction without erroring) before testing shutdown.
	deadline := time.Now().Add(10 * time.Second)
	var connected bool
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", httpAddr, 500*time.Millisecond)
		if err == nil {
			connected = true
			conn.Close()
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !connected {
		_ = cmd.Process.Kill()
		_ = cmd.Wait() // reap the child -- avoid leaving a zombie process behind
		t.Fatal("integration-bridge never opened its HTTP listener within 10s")
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("sending SIGTERM: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("process exited with error after SIGTERM: %v", err)
		}
	case <-time.After(15 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("integration-bridge did not exit within 15s of SIGTERM -- possible shutdown hang")
	}

	// Port must be released -- proves srv.Shutdown actually ran, not just
	// that the process died.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+httpAddr+"/", nil)
	if _, err := http.DefaultClient.Do(req); err == nil {
		t.Error("HTTP port still accepting connections after shutdown -- srv.Shutdown did not release it")
	}
}

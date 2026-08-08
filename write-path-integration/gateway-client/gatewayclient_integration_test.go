//go:build integration

// Integration test against the LIVE network (run manually: `go test -tags
// integration -run TestIntegration -v ./...`). Not part of the normal `go
// test ./...` run — requires the actual 7-node network + committed
// employeeprofilerecord chaincode from NET-1..NET-6/CC-1..5 to be up.
package gatewayclient

import (
	"context"
	"fmt"
	"testing"
)

const netDir = "../../fabric-network/network"

func TestIntegration_EvaluateAndSubmitAgainstLiveNetwork(t *testing.T) {
	gw, err := NewGatewayClient(
		"localhost:7051", "peer0.org1",
		netDir+"/crypto-config/peerOrganizations/org1/tlsca/tlsca.org1-cert.pem",
		netDir+"/crypto-config/peerOrganizations/org1/users/Admin@org1/tls/client.crt",
		netDir+"/crypto-config/peerOrganizations/org1/users/Admin@org1/tls/client.key",
		"Org1MSP",
		netDir+"/crypto-config/peerOrganizations/org1/users/Admin@org1/msp/signcerts/Admin@org1-cert.pem",
		netDir+"/crypto-config/peerOrganizations/org1/users/Admin@org1/msp/keystore/priv_sk",
		"tenant-tenant01", "employeeprofilerecord",
	)
	if err != nil {
		t.Fatalf("NewGatewayClient: %v", err)
	}
	defer gw.Close()

	ctx := context.Background()

	t.Run("Evaluate CC-5's smoke-test record via the Gateway SDK", func(t *testing.T) {
		result, err := gw.EvaluateGetProfileSectionRecord(ctx, "tenant01", "cc5deadbeef00000000000000000000000000000000000000000000000001", "PERSONAL")
		if err != nil {
			t.Fatalf("Evaluate failed: %v", err)
		}
		t.Logf("result: %s", result)
	})

	t.Run("Submit a REC-3 test write with automatic prevHash handling", func(t *testing.T) {
		employeeID := "rec3gatewaytest00000000000000000000000000000000000000000000001"
		buildArgs := func(prevHash string) []string {
			return []string{
				"tenant01", employeeID, "PERSONAL",
				"sha256:rec3gatewaytest000000000000000000000000000000000000000000000",
				prevHash,
				"rec3actor00000000000000000000000000000000000000000000000000002",
				"[]", "JCS-RFC8785-v1", "SHA-256", "",
			}
		}
		result, err := gw.SubmitRecordProfileSection(ctx, "tenant01", employeeID, "PERSONAL", buildArgs)
		if err != nil {
			t.Fatalf("SubmitRecordProfileSection failed: %v", err)
		}
		t.Logf("submit result: %s", result)

		readBack, err := gw.EvaluateGetProfileSectionRecord(ctx, "tenant01", employeeID, "PERSONAL")
		if err != nil {
			t.Fatalf("read-back Evaluate failed: %v", err)
		}
		t.Logf("read-back: %s", readBack)
	})

	t.Run("Second submit to the SAME record exercises the retry-on-stale-prevHash path", func(t *testing.T) {
		employeeID := "rec3gatewaytest00000000000000000000000000000000000000000000001"
		// Deliberately supply a WRONG initial prevHash via a buildArgs that
		// ignores the fresh value on its first call, forcing the wrapper's
		// retry loop to kick in and prove it self-corrects.
		calls := 0
		buildArgs := func(prevHash string) []string {
			calls++
			usePrevHash := prevHash
			if calls == 1 {
				usePrevHash = "sha256:deliberately-wrong-prevhash-to-force-a-retry-0000000000000000"
			}
			return []string{
				"tenant01", employeeID, "PERSONAL",
				"sha256:rec3gatewaytest2ndwrite000000000000000000000000000000000000",
				usePrevHash,
				"rec3actor00000000000000000000000000000000000000000000000000002",
				"[]", "JCS-RFC8785-v1", "SHA-256", "",
			}
		}
		_, err := gw.SubmitRecordProfileSection(ctx, "tenant01", employeeID, "PERSONAL", buildArgs)
		if err != nil {
			t.Fatalf("expected the wrapper to recover from a deliberately-wrong prevHash via retry, got: %v", err)
		}
		if calls < 2 {
			t.Fatalf("expected at least 2 buildArgs calls (initial + retry), got %d — the retry path was not exercised", calls)
		}
		fmt.Printf("buildArgs invoked %d time(s) — retry-on-stale-prevHash path exercised\n", calls)
	})
}

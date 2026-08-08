//go:build integration

// Integration test against the LIVE network — REC-7. Verifies the SAME
// record from THREE independently-configured GatewayClients, one per
// verifier org (Org1, OrgClient-tenant01, Org3), each reading from its OWN
// peer — proving cross-org independent verification, not just one client
// reading its own write back.
package gatewayclient

import (
	"context"
	"testing"
)

func newOrgGatewayClient(t *testing.T, peerHostPort, tlsServerName, orgDomain, mspID string) *GatewayClient {
	adminDir := netDir + "/crypto-config/peerOrganizations/" + orgDomain + "/users/Admin@" + orgDomain
	gw, err := NewGatewayClient(
		peerHostPort, tlsServerName,
		netDir+"/crypto-config/peerOrganizations/"+orgDomain+"/tlsca/tlsca."+orgDomain+"-cert.pem",
		adminDir+"/tls/client.crt", adminDir+"/tls/client.key",
		mspID,
		adminDir+"/msp/signcerts/Admin@"+orgDomain+"-cert.pem",
		adminDir+"/msp/keystore/priv_sk",
		"tenant-tenant01", "employeeprofilerecord",
	)
	if err != nil {
		t.Fatalf("newOrgGatewayClient(%s): %v", orgDomain, err)
	}
	return gw
}

func TestIntegration_REC7_ThreeVerifierClassesAgainstTheirOwnPeers(t *testing.T) {
	writer := newOrgGatewayClient(t, "localhost:7051", "peer0.org1", "org1", "Org1MSP")
	defer writer.Close()

	employeeID := "rec7verifytest00000000000000000000000000000000000000000000001"
	sectionValue := []byte(`{"fullName":"REC-7 Verify Test","department":"QA"}`)
	salt := []byte("rec7-fixed-test-salt-16bytes!!!") // fixed, KNOWN salt so this test can recompute it — REC-2's real salt store is not used here on purpose, this test supplies its own known value to isolate REC-7's own logic.

	dataHash, err := ComputeDataHash(salt, sectionValue)
	if err != nil {
		t.Fatalf("ComputeDataHash: %v", err)
	}
	updatedBy, err := ComputeUpdatedBy([]byte("rec7-test-employee-key-32-bytes!!"), "rec7-test-actor")
	if err != nil {
		t.Fatalf("ComputeUpdatedBy: %v", err)
	}

	buildArgs := func(prevHash string) []string {
		return []string{"tenant01", employeeID, "PERSONAL", dataHash, prevHash, updatedBy, "[]", CanonicalizationVersion, HashAlgo, ""}
	}
	if _, err := writer.SubmitRecordProfileSection(context.Background(), "tenant01", employeeID, "PERSONAL", buildArgs); err != nil {
		t.Fatalf("seed SubmitRecordProfileSection: %v", err)
	}

	verifiers := []struct {
		name         string
		peerHostPort string
		tlsServer    string
		orgDomain    string
		mspID        string
	}{
		{"Org1 (platform, re-verifying its own write)", "localhost:7051", "peer0.org1", "org1", "Org1MSP"},
		{"OrgClient-tenant01 (enterprise-client auditor)", "localhost:9051", "peer0.tenant01", "tenant01", "OrgClient-tenant01MSP"},
		{"Org3 (platform auditor org)", "localhost:10051", "peer0.org3", "org3", "Org3MSP"},
	}

	for _, v := range verifiers {
		t.Run(v.name, func(t *testing.T) {
			gw := newOrgGatewayClient(t, v.peerHostPort, v.tlsServer, v.orgDomain, v.mspID)
			defer gw.Close()

			result, err := gw.Verify(context.Background(), "tenant01", employeeID, "PERSONAL", salt, sectionValue)
			if err != nil {
				t.Fatalf("Verify from %s failed: %v", v.name, err)
			}
			if !result.Matched {
				t.Fatalf("%s: recomputed hash %q does not match on-chain hash %q (own peer replica out of sync or logic bug)", v.name, result.RecomputedHash, result.OnChainHash)
			}
			t.Logf("%s: MATCHED (version=%d, updatedBy=%s)", v.name, result.OnChainVersion, result.UpdatedBy)
		})
	}

	t.Run("tampered value is correctly detected as a mismatch", func(t *testing.T) {
		gw := newOrgGatewayClient(t, "localhost:7051", "peer0.org1", "org1", "Org1MSP")
		defer gw.Close()

		tamperedValue := []byte(`{"fullName":"TAMPERED NAME","department":"QA"}`)
		result, err := gw.Verify(context.Background(), "tenant01", employeeID, "PERSONAL", salt, tamperedValue)
		if err != nil {
			t.Fatalf("Verify (tampered) failed: %v", err)
		}
		if result.Matched {
			t.Fatal("a tampered section value was reported as MATCHED — this defeats the entire tamper-detection purpose of the design (P1)")
		}
		t.Logf("tampered value correctly reported as NOT matched (on-chain=%s, recomputed=%s)", result.OnChainHash, result.RecomputedHash)
	})
}

//go:build integration

// INT-2: closes the gap REC-7 left open — Verify() must distinguish
// NotFound (never anchored, a legitimate outcome) from a local mismatch
// (tampered, only decidable client-side once a record IS found). Kept as
// its own self-contained file per this package's own established style
// (verify_integration_test.go / gatewayclient_integration_test.go each
// define their own small helpers rather than sharing one); the small
// duplication of newOrgGatewayClient here is intentional, not an oversight.
package gatewayclient

import (
	"context"
	"testing"
)

func newOrgGatewayClientForNotFoundTest(t *testing.T, peerHostPort, tlsServerName, orgDomain, mspID string) *GatewayClient {
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
		t.Fatalf("newOrgGatewayClientForNotFoundTest(%s): %v", orgDomain, err)
	}
	return gw
}

// TestIntegration_INT2_VerifyDistinguishesNotFoundFromMismatch is the whole
// point of INT-2: a caller must be able to tell "nothing was ever anchored
// here" apart from both a genuine error AND a tampered/mismatched record —
// the chaincode's ErrRecordNotFound crosses the Gateway boundary as an
// opaque error today, and it is Verify()'s job (not the caller's) to turn
// that back into a first-class, non-error VerificationResult.
func TestIntegration_INT2_VerifyDistinguishesNotFoundFromMismatch(t *testing.T) {
	// Random-ish, 64-char, never anchored by any other test in this suite —
	// deliberately distinct from every fixture employeeID used elsewhere in
	// this package (rec3gatewaytest..., rec7verifytest..., cc5deadbeef...).
	neverAnchoredEmployeeID := "int2notfoundtestc71ab26c227bad88eecdeb4e9d648f30e831345d957b8796"
	salt := []byte("int2-notfound-test-salt-16bytes!")
	sectionValue := []byte(`{"fullName":"INT-2 NotFound Test","department":"QA"}`)

	verifiers := []struct {
		name         string
		peerHostPort string
		tlsServer    string
		orgDomain    string
		mspID        string
	}{
		{"Org1 (platform)", "localhost:7051", "peer0.org1", "org1", "Org1MSP"},
		{"OrgClient-tenant01 (enterprise-client auditor)", "localhost:9051", "peer0.tenant01", "tenant01", "OrgClient-tenant01MSP"},
	}

	for _, v := range verifiers {
		t.Run(v.name, func(t *testing.T) {
			gw := newOrgGatewayClientForNotFoundTest(t, v.peerHostPort, v.tlsServer, v.orgDomain, v.mspID)
			defer gw.Close()

			result, err := gw.Verify(context.Background(), "tenant01", neverAnchoredEmployeeID, "PERSONAL", salt, sectionValue)
			if err != nil {
				t.Fatalf("%s: Verify against a never-anchored record returned an error (should be a legitimate NotFound outcome, not an error): %v", v.name, err)
			}
			if result.Found {
				t.Fatalf("%s: Verify reported Found: true for an employeeID/profileSection that was never anchored", v.name)
			}
			if result.Matched {
				t.Fatalf("%s: Verify reported Matched: true alongside Found: false — a NotFound result must not also claim a match", v.name)
			}
			t.Logf("%s: correctly reported Found: false, nil error for a never-anchored record", v.name)
		})
	}

	t.Run("regression: a record that DOES exist still reports Found: true with the correct Matched value", func(t *testing.T) {
		writer := newOrgGatewayClientForNotFoundTest(t, "localhost:7051", "peer0.org1", "org1", "Org1MSP")
		defer writer.Close()

		employeeID := "int2notfoundtestfixture0000000000000000000000000000000000000001"
		fixtureSalt := []byte("int2-fixture-test-salt-16bytes!!")
		fixtureValue := []byte(`{"fullName":"INT-2 Fixture","department":"QA"}`)

		dataHash, err := ComputeDataHash(fixtureSalt, fixtureValue)
		if err != nil {
			t.Fatalf("ComputeDataHash: %v", err)
		}
		updatedBy, err := ComputeUpdatedBy([]byte("int2-fixture-employee-key-32byte"), "int2-fixture-actor")
		if err != nil {
			t.Fatalf("ComputeUpdatedBy: %v", err)
		}
		buildArgs := func(prevHash string) []string {
			return []string{"tenant01", employeeID, "PERSONAL", dataHash, prevHash, updatedBy, "[]", CanonicalizationVersion, HashAlgo, ""}
		}
		if _, err := writer.SubmitRecordProfileSection(context.Background(), "tenant01", employeeID, "PERSONAL", buildArgs); err != nil {
			t.Fatalf("seed SubmitRecordProfileSection: %v", err)
		}

		result, err := writer.Verify(context.Background(), "tenant01", employeeID, "PERSONAL", fixtureSalt, fixtureValue)
		if err != nil {
			t.Fatalf("Verify against the freshly-anchored fixture record failed: %v", err)
		}
		if !result.Found {
			t.Fatal("Verify reported Found: false for a record that was just anchored")
		}
		if !result.Matched {
			t.Fatalf("Verify reported Matched: false for an untampered fixture record (on-chain=%s, recomputed=%s)", result.OnChainHash, result.RecomputedHash)
		}
		t.Logf("fixture record correctly reported Found: true, Matched: true (version=%d)", result.OnChainVersion)
	})
}

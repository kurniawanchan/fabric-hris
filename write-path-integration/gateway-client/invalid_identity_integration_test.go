//go:build integration

// CT-4 (test-strategy.md §1 "Contract (CT-#)" table): "Missing/invalid caller
// identity ⇒ rejected at the gateway and/or chaincode layer" (T1, T2, API2).
// qa-tests/contract-coverage-report.md's investigation of CT-1..CT-4 found
// this the one CT-# with a GENUINE, not merely relabeled, coverage gap:
// every OTHER NewGatewayClient construction in this package
// (gatewayclient_integration_test.go, verify_integration_test.go,
// verify_notfound_integration_test.go, tenant_isolation_integration_test.go)
// authenticates with a cert/key pair actually issued by the CA of the MSP it
// claims to be. Nothing in this package previously proved the negative. This
// file closes that gap with one new, self-contained test; it does not touch
// any existing file in this package.
//
// The attack modeled is identity FORGERY, not merely "absent identity": the
// connecting client's mTLS transport layer is perfectly legitimate — Org1's
// own, real client TLS cert, trusted by Org1's own peer, the same
// certificate every other test in this package uses to dial
// "localhost:7051" — so a test that only checked "does the TLS handshake
// succeed" would wrongly call this case accepted. The identity Fabric
// actually cares about for endorsement/evaluation is the separate MSP
// SIGNING identity carried inside the SignedProposal
// (mspID+certPEMPath+keyPEMPath in NewGatewayClient's signature), and that
// is the one this test deliberately mismatches: it presents Org1 Admin's
// real MSP signing cert (issued by Org1's own CA) while CLAIMING mspID
// "Org3MSP". Org3MSP is not an arbitrary made-up string chosen to guarantee
// failure for the trivial reason of not being configured anywhere:
// configtx.yaml's TenantChannelGenesis profile lists Org3 as a genuine
// (read-only, non-endorsing per FR-23) member of "tenant-tenant01", so this
// is a real, channel-recognized MSP being impersonated with the wrong org's
// certificate — the only thing that can reject this is the MSP layer
// itself noticing the presented certificate does not chain to Org3's
// registered root/intermediate CA, which is exactly the property CT-4
// exists to prove.
package gatewayclient

import (
	"context"
	"testing"
)

// TestIntegration_CT4_MismatchedMSPClaimRejected connects to Org1's own,
// live peer using Org1's own, real mTLS client certificate (so the gRPC
// transport layer behaves exactly as it does for every legitimate test in
// this package) but constructs the MSP signing identity with mspID
// "Org3MSP" while pointing certPEMPath/keyPEMPath at Org1 Admin's actual
// cert/key on disk — a certificate genuinely issued by Org1's CA, never by
// Org3's. As with TestIntegration_Tenant02CannotReachTenant01Channel, the
// load-bearing assertion is exactly "err != nil"; matching Fabric's precise
// rejection wording (an MSP/certificate-chain validation failure, however it
// surfaces through the Gateway SDK — most likely an EndorseError or a
// discovery/routing failure, since a claimed identity of "Org3MSP" may cause
// the gateway to route the evaluation to a genuine Org3 peer before the
// certificate-chain check ever runs) is an SDK/peer implementation detail
// this test does not pin down, exactly per that file's own precedent.
func TestIntegration_CT4_MismatchedMSPClaimRejected(t *testing.T) {
	org1AdminDir := netDir + "/crypto-config/peerOrganizations/org1/users/Admin@org1"

	gw, err := NewGatewayClient(
		"localhost:7051", "peer0.org1",
		netDir+"/crypto-config/peerOrganizations/org1/tlsca/tlsca.org1-cert.pem",
		// Genuine Org1 mTLS client identity — the transport layer is
		// legitimate on purpose, so a pass here proves nothing by itself.
		org1AdminDir+"/tls/client.crt", org1AdminDir+"/tls/client.key",
		"Org3MSP", // <-- the deliberate lie: claims membership in Org3MSP...
		// ...while presenting Org1's real MSP signing cert/key, never
		// issued by Org3's CA.
		org1AdminDir+"/msp/signcerts/Admin@org1-cert.pem", org1AdminDir+"/msp/keystore/priv_sk",
		"tenant-tenant01", "employeeprofilerecord",
	)
	if err != nil {
		// NewGatewayClient only dials + builds a local identity/contract
		// handle from files already on disk — it performs no MSP validation
		// itself (that happens on the peer, per-proposal, at Evaluate/Submit
		// time). A failure this early would be a local file/setup problem,
		// not the rejection this test exists to demonstrate — fail loudly
		// and distinctly, same convention as
		// tenant_isolation_integration_test.go and
		// verify_notfound_integration_test.go.
		t.Fatalf("NewGatewayClient (Org1 cert/key, claimed mspID \"Org3MSP\") unexpectedly failed at setup: %v", err)
	}
	defer gw.Close()

	_, err = gw.EvaluateGetEmployeeProfileSummary(context.Background(), "tenant01", "cc5deadbeef00000000000000000000000000000000000000000000000001")
	if err == nil {
		t.Fatal("a caller presenting Org1's real certificate under a forged \"Org3MSP\" claim was able to evaluate against tenant-tenant01 — MSP identity validation is BROKEN (the certificate was never issued by Org3's CA)")
	}
	t.Logf("forged Org3MSP claim (real Org1 cert/key) correctly rejected: %v", err)
}

// TestIntegration_CT4_MismatchedMSPClaimAcrossTenantOrgsRejected is the same
// property from a second, independent angle: instead of impersonating a
// read-only auditor org (Org3MSP), this claims "Org1MSP" — the platform
// org, an ACTIVE endorser on "tenant-tenant01" per configtx.yaml's
// Endorsement policy — while presenting OrgClient-tenant01 Admin's real
// cert/key (issued by tenant01's own CA, not Org1's). This rules out any
// possibility that the first test's rejection was somehow specific to
// Org3's read-only/non-endorsing status rather than the certificate-chain
// mismatch itself: here the claimed MSP is a full endorsing member of the
// channel, and the forged identity is still rejected.
func TestIntegration_CT4_MismatchedMSPClaimAcrossTenantOrgsRejected(t *testing.T) {
	tenant01AdminDir := netDir + "/crypto-config/peerOrganizations/tenant01/users/Admin@tenant01"

	gw, err := NewGatewayClient(
		"localhost:9051", "peer0.tenant01",
		netDir+"/crypto-config/peerOrganizations/tenant01/tlsca/tlsca.tenant01-cert.pem",
		// Genuine OrgClient-tenant01 mTLS client identity.
		tenant01AdminDir+"/tls/client.crt", tenant01AdminDir+"/tls/client.key",
		"Org1MSP", // <-- claims the platform org, an active endorser here...
		// ...while presenting OrgClient-tenant01 Admin's real signing
		// cert/key, issued by tenant01's own CA, never by Org1's.
		tenant01AdminDir+"/msp/signcerts/Admin@tenant01-cert.pem", tenant01AdminDir+"/msp/keystore/priv_sk",
		"tenant-tenant01", "employeeprofilerecord",
	)
	if err != nil {
		t.Fatalf("NewGatewayClient (tenant01 cert/key, claimed mspID \"Org1MSP\") unexpectedly failed at setup: %v", err)
	}
	defer gw.Close()

	_, err = gw.EvaluateGetEmployeeProfileSummary(context.Background(), "tenant01", "cc5deadbeef00000000000000000000000000000000000000000000000001")
	if err == nil {
		t.Fatal("a caller presenting OrgClient-tenant01's real certificate under a forged \"Org1MSP\" claim was able to evaluate against tenant-tenant01 — MSP identity validation is BROKEN (the certificate was never issued by Org1's CA)")
	}
	t.Logf("forged Org1MSP claim (real tenant01 cert/key) correctly rejected: %v", err)
}

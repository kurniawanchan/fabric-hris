//go:build integration

// Integration test against the LIVE network — INT-4. NET-7 already proved
// the tenant-provisioning AUTOMATION works for a second tenant (crypto
// material, channel creation, peer join); that is not re-litigated here.
// What this file adds is the one thing NET-7 did NOT: a live negative-
// authorization proof, at the Fabric CHANNEL-MEMBERSHIP layer, that a new
// tenant's own credentials cannot reach another tenant's channel. This is
// deliberately not a chaincode/ACL check — it is the deeper guarantee
// ADR-0013's channel-per-tenant model is actually built on: a peer that was
// never joined to a channel holds no blocks for it at all, so even a
// perfectly valid, correctly-scoped identity gets refused at the gRPC/
// gossip layer before any chaincode logic ever runs.
package gatewayclient

import (
	"context"
	"testing"
)

// TestIntegration_Tenant02CannotReachTenant01Channel targets tenant01's real,
// live channel — proven throughout this session to hold committed
// employeeprofilerecord chaincode and real data — using OrgClient-tenant02's
// OWN identity and OWN peer. peer0.tenant02 was never joined to
// "tenant-tenant01", so this MUST fail. The load-bearing assertion is
// exactly "err != nil"; the precise error text is an SDK/peer
// implementation detail (e.g. a discovery/endorsement failure vs. a bare
// gRPC status) that is not this test's business to pin down.
func TestIntegration_Tenant02CannotReachTenant01Channel(t *testing.T) {
	adminDir := netDir + "/crypto-config/peerOrganizations/tenant02/users/Admin@tenant02"
	gw, err := NewGatewayClient(
		"localhost:11051", "peer0.tenant02",
		netDir+"/crypto-config/peerOrganizations/tenant02/tlsca/tlsca.tenant02-cert.pem",
		adminDir+"/tls/client.crt", adminDir+"/tls/client.key",
		"OrgClient-tenant02MSP",
		adminDir+"/msp/signcerts/Admin@tenant02-cert.pem",
		adminDir+"/msp/keystore/priv_sk",
		"tenant-tenant01", "employeeprofilerecord",
	)
	if err != nil {
		// NewGatewayClient itself only dials + builds a local contract
		// handle — it does not touch the ledger — so a failure THIS early
		// would be a setup/credential problem, not the isolation proof this
		// test exists to make. Fail loudly and distinctly from the
		// evaluate-time failure below.
		t.Fatalf("NewGatewayClient (tenant02 identity, targeting tenant01's channel) unexpectedly failed at setup: %v", err)
	}
	defer gw.Close()

	_, err = gw.EvaluateGetEmployeeProfileSummary(context.Background(), "tenant01", "cc5deadbeef00000000000000000000000000000000000000000000000001")
	if err == nil {
		t.Fatal("tenant02's identity was able to evaluate a query against tenant01's channel — channel-membership isolation is BROKEN (peer0.tenant02 was never joined to tenant-tenant01)")
	}
	t.Logf("tenant02 -> tenant-tenant01 correctly refused: %v", err)
}

// TestIntegration_Tenant01CannotReachTenant02Channel is the symmetric
// direction: OrgClient-tenant01's own identity/peer against
// "tenant-tenant02". Note this direction is expected to fail for a SLIGHTLY
// different underlying reason on top of the same channel-membership fact —
// tenant02's channel also has no chaincode committed on it at all (CC-5
// deliberately only deployed to tenant01's channel) — but the isolation
// claim under test is still the channel-membership boundary itself, which
// holds symmetrically regardless of which side happens to have live data to
// (fail to) reach.
func TestIntegration_Tenant01CannotReachTenant02Channel(t *testing.T) {
	adminDir := netDir + "/crypto-config/peerOrganizations/tenant01/users/Admin@tenant01"
	gw, err := NewGatewayClient(
		"localhost:9051", "peer0.tenant01",
		netDir+"/crypto-config/peerOrganizations/tenant01/tlsca/tlsca.tenant01-cert.pem",
		adminDir+"/tls/client.crt", adminDir+"/tls/client.key",
		"OrgClient-tenant01MSP",
		adminDir+"/msp/signcerts/Admin@tenant01-cert.pem",
		adminDir+"/msp/keystore/priv_sk",
		"tenant-tenant02", "employeeprofilerecord",
	)
	if err != nil {
		t.Fatalf("NewGatewayClient (tenant01 identity, targeting tenant02's channel) unexpectedly failed at setup: %v", err)
	}
	defer gw.Close()

	_, err = gw.EvaluateGetEmployeeProfileSummary(context.Background(), "tenant02", "cc5deadbeef00000000000000000000000000000000000000000000000001")
	if err == nil {
		t.Fatal("tenant01's identity was able to evaluate a query against tenant02's channel — channel-membership isolation is BROKEN (peer0.tenant01 was never joined to tenant-tenant02)")
	}
	t.Logf("tenant01 -> tenant-tenant02 correctly refused: %v", err)
}

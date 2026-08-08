//go:build integration

// INT-1/INT-5: proves the provisional "retry-bounded, then degrade-and-
// flag" policy (see PartialFailureError's own doc comment in writepaths.go
// for why this is a labeled [ASSUMPTION], not a ratified ADR) against a
// REAL partial-failure state, not a simulated one. This deliberately does
// NOT touch a live container to induce the failure — pointing a otherwise-
// valid GatewayClient (real TLS, real identity, real committed channel) at
// a chaincode name that was never installed/committed on tenant01's
// channel fails cleanly at the Gateway layer (SubmitTransaction rejects
// before ever reaching chaincode logic), which is exactly the "genuine,
// non-retryable Fabric-layer failure" class this item's policy targets —
// while the operational-DB write (a real InMemoryOperationalStore, not a
// mock assertion) genuinely commits first, because Store.SaveSection runs
// before anchor() in every hook (see writepaths.go).
package writepaths

import (
	"context"
	"errors"
	"testing"

	gatewayclient "gatewayclient"
	keystore "keystore"
)

func TestIntegration_PartialFailure_AnchorFailsAfterOperationalWriteCommits(t *testing.T) {
	// tenant01's own org identity/peer, valid in every respect (TLS,
	// MSP, committed channel) EXCEPT the chaincode name — "doesnotexist"
	// was never installed or committed on tenant-tenant01, so Submit fails
	// at the Gateway layer without needing to touch any live container's
	// state.
	gw, err := gatewayclient.NewGatewayClient(
		"localhost:9051", "peer0.tenant01",
		netDir+"/crypto-config/peerOrganizations/tenant01/tlsca/tlsca.tenant01-cert.pem",
		netDir+"/crypto-config/peerOrganizations/tenant01/users/Admin@tenant01/tls/client.crt",
		netDir+"/crypto-config/peerOrganizations/tenant01/users/Admin@tenant01/tls/client.key",
		"OrgClient-tenant01MSP",
		netDir+"/crypto-config/peerOrganizations/tenant01/users/Admin@tenant01/msp/signcerts/Admin@tenant01-cert.pem",
		netDir+"/crypto-config/peerOrganizations/tenant01/users/Admin@tenant01/msp/keystore/priv_sk",
		"tenant-tenant01", "doesnotexist",
	)
	if err != nil {
		t.Fatalf("NewGatewayClient: %v", err)
	}
	defer gw.Close()

	type partialFailureCall struct {
		employeeInternalID string
		profileSection     string
		version            int
		err                error
	}
	var onPartialFailureCalls []partialFailureCall

	store := NewInMemoryOperationalStore()
	hooks := &Hooks{
		Store:        store,
		Keys:         keystore.NewInMemoryEmployeeKeyStore(),
		Salts:        keystore.NewInMemorySaltStore(),
		DocumentKeys: keystore.NewInMemoryDocumentKeyStore(),
		// IPFS is intentionally left nil — this test's call carries no
		// supporting document, so doAnchor's document branch (the only
		// path that dereferences h.IPFS) is never reached.
		Gateway:  gw,
		TenantID: "tenant01",
		OnPartialFailure: func(_ context.Context, employeeInternalID, profileSection string, version int, err error) {
			onPartialFailureCalls = append(onPartialFailureCalls, partialFailureCall{employeeInternalID, profileSection, version, err})
		},
	}

	ctx := context.Background()
	employeeInternalID := "int1demoemployee000000000000000000000000000000000000000001"
	userID := "hr-admin-int1-demo"
	newValue := []byte(`{"fullName":"INT-1 Partial-Failure Demo","address":"Generic City"}`)

	result, err := hooks.UpdatePersonalData(ctx, employeeInternalID, userID, newValue, nil)
	if err == nil {
		t.Fatalf("UpdatePersonalData: expected an error anchoring against a nonexistent chaincode, got nil (result=%s)", result)
	}

	// (a) The operational-DB write genuinely committed despite the anchor
	// failure — the actual partial-failure state occurred, not a simulated
	// one.
	if !store.HasValue(employeeInternalID, "PERSONAL") {
		t.Errorf("store.HasValue(%q, PERSONAL) = false; want true — SaveSection should have committed before anchor() ever ran", employeeInternalID)
	}

	// (b) The returned error is distinctly typed as *PartialFailureError,
	// carrying the exact identifying fields passed in.
	var pfErr *PartialFailureError
	if !errors.As(err, &pfErr) {
		t.Fatalf("errors.As(err, *PartialFailureError) failed; got err of type %T: %v", err, err)
	}
	if pfErr.EmployeeInternalID != employeeInternalID {
		t.Errorf("PartialFailureError.EmployeeInternalID = %q, want %q", pfErr.EmployeeInternalID, employeeInternalID)
	}
	if pfErr.ProfileSection != "PERSONAL" {
		t.Errorf("PartialFailureError.ProfileSection = %q, want %q", pfErr.ProfileSection, "PERSONAL")
	}
	if pfErr.Version != 1 {
		t.Errorf("PartialFailureError.Version = %d, want 1 (first SaveSection on a fresh InMemoryOperationalStore)", pfErr.Version)
	}
	if pfErr.Err == nil {
		t.Errorf("PartialFailureError.Err = nil, want the underlying Gateway/ledger error")
	}

	// (c) OnPartialFailure fired exactly once, with matching arguments.
	if len(onPartialFailureCalls) != 1 {
		t.Fatalf("OnPartialFailure called %d times, want exactly 1 (calls=%+v)", len(onPartialFailureCalls), onPartialFailureCalls)
	}
	call := onPartialFailureCalls[0]
	if call.employeeInternalID != employeeInternalID {
		t.Errorf("OnPartialFailure employeeInternalID = %q, want %q", call.employeeInternalID, employeeInternalID)
	}
	if call.profileSection != "PERSONAL" {
		t.Errorf("OnPartialFailure profileSection = %q, want %q", call.profileSection, "PERSONAL")
	}
	if call.version != 1 {
		t.Errorf("OnPartialFailure version = %d, want 1", call.version)
	}
	var callPfErr *PartialFailureError
	if !errors.As(call.err, &callPfErr) {
		t.Errorf("OnPartialFailure err is not a *PartialFailureError: %T: %v", call.err, call.err)
	}
}

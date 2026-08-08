//go:build integration

// Integration test against the LIVE network (run: `go test -tags
// integration -run TestIntegration -v ./...`). Exercises all FIVE
// write-path hooks for one demo employee, then confirms
// GetEmployeeProfileSummary fans out to all five anchored sections.
package writepaths

import (
	"context"
	"strings"
	"testing"

	gatewayclient "gatewayclient"
	ipfsclient "ipfsclient"
	keystore "keystore"
)

const netDir = "../../fabric-network/network"

func TestIntegration_AllFiveWritePathsAgainstLiveNetwork(t *testing.T) {
	gw, err := gatewayclient.NewGatewayClient(
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

	hooks := &Hooks{
		Store:        NewInMemoryOperationalStore(),
		Keys:         keystore.NewInMemoryEmployeeKeyStore(),
		Salts:        keystore.NewInMemorySaltStore(),
		DocumentKeys: keystore.NewInMemoryDocumentKeyStore(),
		IPFS:         ipfsclient.NewClient("http://localhost:5001", "http://localhost:5002"),
		Gateway:      gw,
		TenantID:     "tenant01",
	}

	ctx := context.Background()
	employeeInternalID := "rec4demoemployee00000000000000000000000000000000000000000001"
	userID := "hr-admin-rec4-demo"

	type call struct {
		name string
		fn   func() ([]byte, error)
	}
	// PERSONAL is written WITH a supporting document (REC-5) — the other four
	// carry none, the common case, proving `document` is genuinely optional.
	personalSupportingDoc := []byte("REC-5 integration fixture — a generic supporting document, never written in plaintext to IPFS")

	calls := []call{
		{"UpdatePersonalData", func() ([]byte, error) {
			return hooks.UpdatePersonalData(ctx, employeeInternalID, userID, []byte(`{"fullName":"REC-4 Demo Employee","address":"Generic City"}`), personalSupportingDoc)
		}},
		{"ApproveEmploymentTransfer", func() ([]byte, error) {
			return hooks.ApproveEmploymentTransfer(ctx, employeeInternalID, userID, []byte(`{"department":"Engineering","title":"Staff Engineer"}`), nil)
		}},
		{"RecordEducationHistory", func() ([]byte, error) {
			return hooks.RecordEducationHistory(ctx, employeeInternalID, userID, []byte(`{"degree":"B.Sc.","institution":"Generic University"}`), nil)
		}},
		{"ApproveFamilyDataChange", func() ([]byte, error) {
			return hooks.ApproveFamilyDataChange(ctx, employeeInternalID, userID, []byte(`{"maritalStatus":"married","dependents":1}`), nil)
		}},
		{"UpdatePayrollBankAccount", func() ([]byte, error) {
			return hooks.UpdatePayrollBankAccount(ctx, employeeInternalID, userID, []byte(`{"bankName":"Generic Bank","accountLast4":"1234"}`), nil)
		}},
	}

	for _, c := range calls {
		t.Run(c.name, func(t *testing.T) {
			result, err := c.fn()
			if err != nil {
				t.Fatalf("%s failed: %v", c.name, err)
			}
			t.Logf("%s result: %s", c.name, result)
		})
	}

	t.Run("GetEmployeeProfileSummary fans out to all five sections", func(t *testing.T) {
		employeeKey, _ := hooks.Keys.GetOrCreateEmployeeKey(ctx, employeeInternalID)
		employeeID, _ := gatewayclient.ComputeEmployeeID(employeeKey)

		summary, err := gw.EvaluateGetEmployeeProfileSummary(ctx, "tenant01", employeeID)
		if err != nil {
			t.Fatalf("GetEmployeeProfileSummary failed: %v", err)
		}
		t.Logf("summary: %s", summary)

		for _, section := range []string{"PERSONAL", "EMPLOYMENT", "EDUCATION", "ADDITIONAL", "PAYROLL"} {
			if !strings.Contains(string(summary), section) {
				t.Errorf("summary is missing section %s — not all five write-path hooks landed on-chain", section)
			}
		}
	})

	t.Run("REC-5: only the section written WITH a document carries a CID on-chain", func(t *testing.T) {
		// GetProfileSectionRecord deliberately returns only the reduced
		// ProfileSectionHead (dataHash/version/timestamp/updatedBy) per
		// CC-3's own contract (api-contracts.md "never the full asset") —
		// ipfsCIDs is only visible via GetProfileHistory's full
		// ProfileVersionEntry, so that is what this assertion must use.
		employeeKey, _ := hooks.Keys.GetOrCreateEmployeeKey(ctx, employeeInternalID)
		employeeID, _ := gatewayclient.ComputeEmployeeID(employeeKey)

		personalHistory, err := gw.EvaluateGetProfileHistory(ctx, "tenant01", employeeID, "PERSONAL")
		if err != nil {
			t.Fatalf("GetProfileHistory(PERSONAL): %v", err)
		}
		if strings.Contains(string(personalHistory), `"ipfsCIDs":[]`) {
			t.Errorf("PERSONAL was written WITH a supporting document but landed on-chain with an EMPTY ipfsCIDs — the CID never made it into the RecordProfileSection call: %s", personalHistory)
		}

		employmentHistory, err := gw.EvaluateGetProfileHistory(ctx, "tenant01", employeeID, "EMPLOYMENT")
		if err != nil {
			t.Fatalf("GetProfileHistory(EMPLOYMENT): %v", err)
		}
		if !strings.Contains(string(employmentHistory), `"ipfsCIDs":[]`) {
			t.Errorf("EMPLOYMENT was written WITHOUT a document but landed on-chain with a non-empty ipfsCIDs: %s", employmentHistory)
		}
	})
}

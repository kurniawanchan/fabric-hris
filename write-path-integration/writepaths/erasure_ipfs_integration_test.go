//go:build integration

// QA-2's actual new coverage work: IT-5 and IT-9's erasure-basis half do not
// yet exist as runnable proof anywhere in this suite. erasure_test.go (REC-6)
// already proves recompute-impossibility against MOCK stores, and erasure.go's
// own doc-comment already CLAIMS "zero ledger calls" — neither is the same as
// proving, against the LIVE ledger, that on-chain state survives Hooks.Erase
// byte-for-byte unchanged (IT-5), nor the same as proving the concrete
// old-ciphertext/new-key decryption failure IT-9/T18 wants. Deliberately its
// own file (QA-2's scope explicitly excludes editing erasure.go, erasure_test.go,
// writepaths.go, and writepaths_integration_test.go) — reuses
// writepaths_integration_test.go's own fixture/call-shape conventions (a
// fresh, distinctive employeeInternalID never used by any other test in this
// suite; PERSONAL carries the one supporting document, the other four carry
// none) rather than inventing a new pattern.
package writepaths

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	gatewayclient "gatewayclient"
	ipfsclient "ipfsclient"
	keystore "keystore"
)

// TestIntegration_QA2_ErasureLeavesOnChainStateUnchangedAndBreaksOldCiphertext
// covers both gaps in one fixture/run (they share the same write and Erase
// call, so splitting them into two separate tests would mean anchoring the
// same fixture twice for no benefit):
//
//   - IT-5: every on-chain read surface (GetEmployeeProfileSummary,
//     GetProfileHistory, GetProfileSectionRecord per section) is
//     byte-for-byte identical before and after Hooks.Erase.
//   - IT-9 (erasure-basis half): the CID anchored for the PERSONAL section's
//     supporting document decrypts correctly pre-erasure with the KEY_EMPLOYEE
//     that encrypted it, but a FRESH KEY_EMPLOYEE obtained for the SAME
//     employeeInternalID after Erase (which destroyed the old one via
//     DocumentKeys.DeleteDocumentKey) fails to decrypt that same ciphertext —
//     the concrete "unpin != delete, key-destruction is the real erasure
//     mechanism" proof, since this test never unpins the CID at all.
func TestIntegration_QA2_ErasureLeavesOnChainStateUnchangedAndBreaksOldCiphertext(t *testing.T) {
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

	// Kept as a local, concrete *ipfsclient.Client (not read back through
	// hooks.IPFS, which is now DocumentPinner-typed and deliberately omits
	// FetchAndDecrypt — writepaths' production code never reads a document
	// back) — mirroring how this same file already keeps a local `gw` for
	// its Evaluate* calls on the Gateway side.
	ipfsClient := ipfsclient.NewClient("http://localhost:5001", "http://localhost:5002")
	hooks := &Hooks{
		Store:        NewInMemoryOperationalStore(),
		Keys:         keystore.NewInMemoryEmployeeKeyStore(),
		Salts:        keystore.NewInMemorySaltStore(),
		DocumentKeys: keystore.NewInMemoryDocumentKeyStore(),
		IPFS:         ipfsClient,
		Gateway:      gw,
		TenantID:     "tenant01",
	}

	ctx := context.Background()
	employeeInternalID := "qa2erasuredemo00000000000000000000000000000000000000000001"
	userID := "hr-admin-qa2-erasure-demo"
	document := []byte("QA-2 IT-9 fixture — a generic supporting document, never written in plaintext to IPFS")

	if _, err := hooks.UpdatePersonalData(ctx, employeeInternalID, userID, []byte(`{"fullName":"QA-2 Erasure Demo","address":"Generic City"}`), document); err != nil {
		t.Fatalf("UpdatePersonalData: %v", err)
	}
	if _, err := hooks.ApproveEmploymentTransfer(ctx, employeeInternalID, userID, []byte(`{"department":"QA","title":"Erasure Fixture"}`), nil); err != nil {
		t.Fatalf("ApproveEmploymentTransfer: %v", err)
	}
	if _, err := hooks.RecordEducationHistory(ctx, employeeInternalID, userID, []byte(`{"degree":"B.Sc.","institution":"Generic University"}`), nil); err != nil {
		t.Fatalf("RecordEducationHistory: %v", err)
	}
	if _, err := hooks.ApproveFamilyDataChange(ctx, employeeInternalID, userID, []byte(`{"maritalStatus":"single","dependents":0}`), nil); err != nil {
		t.Fatalf("ApproveFamilyDataChange: %v", err)
	}
	if _, err := hooks.UpdatePayrollBankAccount(ctx, employeeInternalID, userID, []byte(`{"bankName":"Generic Bank","accountLast4":"5678"}`), nil); err != nil {
		t.Fatalf("UpdatePayrollBankAccount: %v", err)
	}

	employeeKey, err := hooks.Keys.GetOrCreateEmployeeKey(ctx, employeeInternalID)
	if err != nil {
		t.Fatalf("GetOrCreateEmployeeKey: %v", err)
	}
	employeeID, err := gatewayclient.ComputeEmployeeID(employeeKey)
	if err != nil {
		t.Fatalf("ComputeEmployeeID: %v", err)
	}

	// ---- snapshot every on-chain read surface BEFORE erasure ----
	preSummary, err := gw.EvaluateGetEmployeeProfileSummary(ctx, "tenant01", employeeID)
	if err != nil {
		t.Fatalf("pre-erase GetEmployeeProfileSummary: %v", err)
	}
	prePersonalHistory, err := gw.EvaluateGetProfileHistory(ctx, "tenant01", employeeID, "PERSONAL")
	if err != nil {
		t.Fatalf("pre-erase GetProfileHistory(PERSONAL): %v", err)
	}
	preRecords := map[string][]byte{}
	for _, section := range AllProfileSections {
		rec, err := gw.EvaluateGetProfileSectionRecord(ctx, "tenant01", employeeID, section)
		if err != nil {
			t.Fatalf("pre-erase GetProfileSectionRecord(%s): %v", section, err)
		}
		preRecords[section] = rec
	}

	// ---- IT-9, front half: fetch+decrypt succeeds PRE-erasure ----
	var cid string
	{
		var history []struct {
			IPFSCIDs []string `json:"ipfsCIDs"`
		}
		if err := json.Unmarshal(prePersonalHistory, &history); err != nil {
			t.Fatalf("parsing GetProfileHistory(PERSONAL) JSON: %v (raw=%s)", err, prePersonalHistory)
		}
		if len(history) == 0 || len(history[0].IPFSCIDs) == 0 {
			t.Fatalf("PERSONAL history carries no ipfsCIDs — the document write did not anchor a CID: %s", prePersonalHistory)
		}
		cid = history[0].IPFSCIDs[0]
	}
	preEraseDocumentKey, err := hooks.DocumentKeys.GetOrCreateDocumentKey(ctx, employeeInternalID)
	if err != nil {
		t.Fatalf("GetOrCreateDocumentKey (pre-erase): %v", err)
	}
	plaintext, err := ipfsClient.FetchAndDecrypt(ctx, preEraseDocumentKey, cid)
	if err != nil {
		t.Fatalf("FetchAndDecrypt pre-erasure with the correct KEY_EMPLOYEE unexpectedly failed: %v", err)
	}
	if !bytes.Equal(plaintext, document) {
		t.Fatalf("FetchAndDecrypt pre-erasure returned %q, want %q", plaintext, document)
	}

	// ---- the actual erasure ----
	if err := hooks.Erase(ctx, employeeInternalID); err != nil {
		t.Fatalf("Erase: %v", err)
	}

	// ---- IT-5: re-read every surface, assert byte-for-byte identity ----
	postSummary, err := gw.EvaluateGetEmployeeProfileSummary(ctx, "tenant01", employeeID)
	if err != nil {
		t.Fatalf("post-erase GetEmployeeProfileSummary: %v", err)
	}
	if !bytes.Equal(preSummary, postSummary) {
		t.Fatalf("GetEmployeeProfileSummary changed after Erase — on-chain state was NOT left untouched.\nbefore: %s\nafter:  %s", preSummary, postSummary)
	}
	postPersonalHistory, err := gw.EvaluateGetProfileHistory(ctx, "tenant01", employeeID, "PERSONAL")
	if err != nil {
		t.Fatalf("post-erase GetProfileHistory(PERSONAL): %v", err)
	}
	if !bytes.Equal(prePersonalHistory, postPersonalHistory) {
		t.Fatalf("GetProfileHistory(PERSONAL) changed after Erase.\nbefore: %s\nafter:  %s", prePersonalHistory, postPersonalHistory)
	}
	for _, section := range AllProfileSections {
		rec, err := gw.EvaluateGetProfileSectionRecord(ctx, "tenant01", employeeID, section)
		if err != nil {
			t.Fatalf("post-erase GetProfileSectionRecord(%s): %v", section, err)
		}
		if !bytes.Equal(preRecords[section], rec) {
			t.Fatalf("GetProfileSectionRecord(%s) changed after Erase.\nbefore: %s\nafter:  %s", section, preRecords[section], rec)
		}
	}
	t.Logf("IT-5 confirmed: GetEmployeeProfileSummary + GetProfileHistory(PERSONAL) + all %d GetProfileSectionRecord reads are byte-for-byte identical before/after Erase", len(AllProfileSections))

	// ---- IT-9, back half: fresh post-erasure KEY_EMPLOYEE cannot open the
	// old ciphertext. This is the erasure MECHANISM under test, not object
	// removal — the CID above was never unpinned, so if this decrypt
	// unexpectedly succeeded it would mean the ciphertext is still readable
	// by a party who only has a freshly-issued key, which would falsify the
	// whole crypto-shred design. ----
	postEraseDocumentKey, err := hooks.DocumentKeys.GetOrCreateDocumentKey(ctx, employeeInternalID)
	if err != nil {
		t.Fatalf("GetOrCreateDocumentKey (post-erase): %v", err)
	}
	if bytes.Equal(postEraseDocumentKey, preEraseDocumentKey) {
		t.Fatal("GetOrCreateDocumentKey returned the SAME key after Erase deleted it — DeleteDocumentKey did not actually remove the stored key")
	}
	if _, err := ipfsClient.FetchAndDecrypt(ctx, postEraseDocumentKey, cid); err == nil {
		t.Fatal("FetchAndDecrypt succeeded with a FRESH post-erasure KEY_EMPLOYEE against the OLD ciphertext — erasure basis is BROKEN: the old ciphertext should be permanently unopenable once the key that encrypted it is destroyed")
	} else {
		t.Logf("IT-9 confirmed: post-erasure FetchAndDecrypt with a fresh KEY_EMPLOYEE correctly failed to open the old ciphertext: %v", err)
	}
}

//go:build integration

// Integration test against the LIVE network — ST-5. REC-7's own
// verify_integration_test.go already proves tamper detection for the
// PERSONAL ProfileSection; this file replicates that exact pattern (anchor
// a real value via a GatewayClient, then Verify() against a DELIBERATELY
// DIFFERENT value than what was anchored, asserting Matched: false with
// both hashes shown differing) for the four remaining ProfileSection
// values: EMPLOYMENT, EDUCATION, ADDITIONAL, PAYROLL — closing errata E-2
// (the thesis claimed "100% across five sections" but only ever tested
// PERSONAL). Per test-strategy.md's own instruction, the ADDITIONAL
// scenario is a NAMED manipulation scenario labeled SC-F (avoiding the
// already-overloaded SC-A..E labels) — it tampers with a marital-status/
// dependent-count value.
package gatewayclient

import (
	"context"
	"testing"
)

func TestIntegration_ST5_TamperDetectionAllSections(t *testing.T) {
	writer := newOrgGatewayClient(t, "localhost:7051", "peer0.org1", "org1", "Org1MSP")
	defer writer.Close()

	salt := []byte("st5-fixed-test-salt-16bytes!!!!") // fixed, KNOWN salt so this test can recompute it locally — same rationale as REC-7's own verify_integration_test.go.
	employeeKeyI := []byte("st5-test-employee-key-32-bytes!!!")
	updatedBy, err := ComputeUpdatedBy(employeeKeyI, "st5-test-actor")
	if err != nil {
		t.Fatalf("ComputeUpdatedBy: %v", err)
	}

	cases := []struct {
		name           string
		profileSection string
		employeeID     string
		anchoredValue  []byte
		tamperedValue  []byte
	}{
		{
			name:           "EMPLOYMENT",
			profileSection: "EMPLOYMENT",
			employeeID:     "st5tampertest000000000000000000000000000000000000000000000001",
			anchoredValue:  []byte(`{"department":"Engineering","jobTitle":"Software Engineer"}`),
			tamperedValue:  []byte(`{"department":"Executive","jobTitle":"Chief Officer"}`),
		},
		{
			name:           "EDUCATION",
			profileSection: "EDUCATION",
			employeeID:     "st5tampertest000000000000000000000000000000000000000000000002",
			anchoredValue:  []byte(`{"degree":"Bachelor","institution":"State University"}`),
			tamperedValue:  []byte(`{"degree":"Doctorate","institution":"Diploma Mill University"}`),
		},
		{
			// SC-F (test-strategy.md ST-5): named manipulation scenario for
			// ADDITIONAL, distinct from the already-overloaded SC-A..E
			// labels — tampers with a marital-status/dependent-count value.
			name:           "ADDITIONAL (SC-F)",
			profileSection: "ADDITIONAL",
			employeeID:     "st5tampertest000000000000000000000000000000000000000000000003",
			anchoredValue:  []byte(`{"maritalStatus":"single","dependentCount":0}`),
			tamperedValue:  []byte(`{"maritalStatus":"married","dependentCount":4}`),
		},
		{
			name:           "PAYROLL",
			profileSection: "PAYROLL",
			employeeID:     "st5tampertest000000000000000000000000000000000000000000000004",
			anchoredValue:  []byte(`{"bankAccountNumber":"1234567890","bankName":"Generic Bank"}`),
			tamperedValue:  []byte(`{"bankAccountNumber":"9999999999","bankName":"Attacker Bank"}`),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dataHash, err := ComputeDataHash(salt, c.anchoredValue)
			if err != nil {
				t.Fatalf("ComputeDataHash: %v", err)
			}

			buildArgs := func(prevHash string) []string {
				return []string{"tenant01", c.employeeID, c.profileSection, dataHash, prevHash, updatedBy, "[]", CanonicalizationVersion, HashAlgo, ""}
			}
			if _, err := writer.SubmitRecordProfileSection(context.Background(), "tenant01", c.employeeID, c.profileSection, buildArgs); err != nil {
				t.Fatalf("seed SubmitRecordProfileSection: %v", err)
			}

			result, err := writer.Verify(context.Background(), "tenant01", c.employeeID, c.profileSection, salt, c.tamperedValue)
			if err != nil {
				t.Fatalf("Verify (tampered) failed: %v", err)
			}
			if result.Matched {
				t.Fatalf("%s: a tampered section value was reported as MATCHED — this defeats the entire tamper-detection purpose of the design (P1)", c.name)
			}
			if result.OnChainHash == result.RecomputedHash {
				t.Fatalf("%s: expected on-chain hash %q and recomputed hash %q to differ, they matched", c.name, result.OnChainHash, result.RecomputedHash)
			}
			t.Logf("%s: tampered value correctly reported as NOT matched (on-chain=%s, recomputed=%s)", c.name, result.OnChainHash, result.RecomputedHash)

			// Positive control, same record: verifying against the ORIGINAL
			// anchored value (not the tampered one) still matches — proving
			// the mismatch above is due to the tamper, not a broken Verify
			// path or a hashing bug that always disagrees.
			okResult, err := writer.Verify(context.Background(), "tenant01", c.employeeID, c.profileSection, salt, c.anchoredValue)
			if err != nil {
				t.Fatalf("Verify (original, positive control) failed: %v", err)
			}
			if !okResult.Matched {
				t.Fatalf("%s: positive control failed — the ORIGINAL anchored value did not match (on-chain=%s, recomputed=%s)", c.name, okResult.OnChainHash, okResult.RecomputedHash)
			}
		})
	}
}

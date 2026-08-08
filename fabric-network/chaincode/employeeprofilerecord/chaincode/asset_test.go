package chaincode

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidProfileSection(t *testing.T) {
	cases := map[string]bool{
		"PERSONAL":              true,
		"EMPLOYMENT":            true,
		"EDUCATION":             true,
		"ADDITIONAL":            true,
		"PAYROLL":               true,
		"personal":              false, // case-sensitive, exact match only (FR-8)
		"SALARY":                false, // not one of the five
		"":                      false,
		"EVENT_UPDATE_PERSONAL": false, // the retired event surface must not silently validate
	}
	for in, want := range cases {
		require.Equal(t, want, IsValidProfileSection(in), "IsValidProfileSection(%q)", in)
	}
}

// TestEmployeeProfileRecordFieldSetIsExact locks the asset shape to exactly
// the twelve fields data-model.md §2 lists — no more, no less (closes the
// STRIDE T9 regression risk the backlog cites: adding a field back, e.g. a
// per-field change list, would reopen it).
func TestEmployeeProfileRecordFieldSetIsExact(t *testing.T) {
	rec := EmployeeProfileRecord{
		CanonicalizationVersion: "JCS-RFC8785-v1",
		DataHash:                "sha256:aaaa",
		EmployeeID:              "emp-id",
		HashAlgo:                "SHA-256",
		IpfsCIDs:                []string{"bafy1"},
		PrevHash:                "sha256:bbbb",
		ProfileSection:          "PAYROLL",
		RecordID:                "rec-id",
		TenantID:                "tenant01",
		Timestamp:               "2026-08-06T00:00:00Z",
		UpdatedBy:               "actor-id",
		Version:                 3,
	}

	raw, err := json.Marshal(rec)
	require.NoError(t, err)

	var asMap map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &asMap))

	wantKeys := []string{
		"recordID", "employeeID", "tenantID", "profileSection", "dataHash",
		"prevHash", "version", "updatedBy", "timestamp", "ipfsCIDs",
		"canonicalizationVersion", "hashAlgo",
	}
	require.Len(t, asMap, len(wantKeys), "field count must be exactly the data-model.md §2 set")
	for _, k := range wantKeys {
		_, ok := asMap[k]
		require.True(t, ok, "missing expected field %q", k)
	}

	// No PII/plaintext-shaped field ever sneaks in under a different name —
	// a defensive regression guard, not a substitute for the security review.
	for _, forbidden := range []string{"salt", "values", "sectionData", "changedFieldNames", "pseudonymKey"} {
		_, ok := asMap[forbidden]
		require.False(t, ok, "forbidden field %q must never appear on the asset", forbidden)
	}
}

// TestEmployeeProfileRecordJSONIsDeterministic pins the exact field order
// Go's encoding/json produces (declaration order, not re-sorted) — the
// contract-api-go determinism convention this asset relies on
// [docs: chaincode4ade.rst#json-determinism]. If this ever fails, either the
// struct's field declaration order changed or the Go JSON encoder's
// behaviour changed — both are exactly the kind of drift that would silently
// break cross-peer endorsement matching, so this is pinned byte-for-byte.
func TestEmployeeProfileRecordJSONIsDeterministic(t *testing.T) {
	rec := EmployeeProfileRecord{
		CanonicalizationVersion: "JCS-RFC8785-v1",
		DataHash:                "sha256:aaaa",
		EmployeeID:              "emp-id",
		HashAlgo:                "SHA-256",
		IpfsCIDs:                []string{},
		PrevHash:                "",
		ProfileSection:          "PAYROLL",
		RecordID:                "rec-id",
		TenantID:                "tenant01",
		Timestamp:               "2026-08-06T00:00:00Z",
		UpdatedBy:               "actor-id",
		Version:                 1,
	}
	want := `{"canonicalizationVersion":"JCS-RFC8785-v1","dataHash":"sha256:aaaa","employeeID":"emp-id","hashAlgo":"SHA-256","ipfsCIDs":[],"prevHash":"","profileSection":"PAYROLL","recordID":"rec-id","tenantID":"tenant01","timestamp":"2026-08-06T00:00:00Z","updatedBy":"actor-id","version":1}`

	got, err := json.Marshal(rec)
	require.NoError(t, err)
	require.JSONEq(t, want, string(got))
	require.Equal(t, want, string(got), "byte-for-byte wire shape must match (not just JSON-equivalent) — this is what cross-peer endorsement matching depends on")
}

package chaincode

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecordProfileSection_SuccessfulWrite(t *testing.T) {
	sc := &SmartContract{}
	ctx, _ := newTestContext(t, mspOrg1)

	res, err := sc.RecordProfileSection(ctx, "tenant01", "emp-hmac-1", "PAYROLL",
		"sha256:aaaa", "", "actor-hmac-1", []string{}, "JCS-RFC8785-v1", "SHA-256", "")
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, 1, res.Version)
	require.NotEmpty(t, res.RecordID)
	require.NotEmpty(t, res.Timestamp)

	// The stored head must be readable back exactly, and via the Evaluate
	// side, as only {dataHash, version, timestamp, updatedBy}.
	head, err := sc.GetProfileSectionRecord(ctx, "tenant01", "emp-hmac-1", "PAYROLL")
	require.NoError(t, err)
	require.Equal(t, "sha256:aaaa", head.DataHash)
	require.Equal(t, 1, head.Version)
	require.Equal(t, "actor-hmac-1", head.UpdatedBy)
	require.Equal(t, res.Timestamp, head.Timestamp)
}

// TestRecordProfileSection_TimestampIsLedgerTime confirms the stored/returned
// timestamp comes from GetTxTimestamp(), never the caller-supplied
// clientTimestamp — the contract's explicit "never trust clientTimestamp as
// authoritative" requirement.
func TestRecordProfileSection_TimestampIsLedgerTime(t *testing.T) {
	sc := &SmartContract{}
	ctx, _ := newTestContext(t, mspOrg1)

	res, err := sc.RecordProfileSection(ctx, "tenant01", "emp-hmac-1", "PAYROLL",
		"sha256:aaaa", "", "actor-hmac-1", nil, "JCS-RFC8785-v1", "SHA-256",
		"2099-01-01T00:00:00Z" /* an obviously-wrong client-supplied value */)
	require.NoError(t, err)
	require.NotEqual(t, "2099-01-01T00:00:00Z", res.Timestamp, "clientTimestamp must never be echoed back as the authoritative timestamp")
}

// TestRecordProfileSection_NoOpOnIdenticalDataHash is FR-9: a second write
// with the SAME dataHash for the same (employeeID, profileSection) is a
// no-op — no new version, no PutState — and returns the existing head's
// result unchanged.
func TestRecordProfileSection_NoOpOnIdenticalDataHash(t *testing.T) {
	sc := &SmartContract{}
	ctx, stub := newTestContext(t, mspOrg1)

	first, err := sc.RecordProfileSection(ctx, "tenant01", "emp-hmac-1", "PAYROLL",
		"sha256:aaaa", "", "actor-hmac-1", nil, "JCS-RFC8785-v1", "SHA-256", "")
	require.NoError(t, err)
	require.Equal(t, 1, first.Version)

	stateLenAfterFirst := len(stub.State)

	second, err := sc.RecordProfileSection(ctx, "tenant01", "emp-hmac-1", "PAYROLL",
		"sha256:aaaa", "sha256:aaaa" /* even a caller-supplied prevHash equal to the (unchanged) head must not matter for a no-op */, "actor-hmac-1", nil, "JCS-RFC8785-v1", "SHA-256", "")
	require.NoError(t, err)

	require.Equal(t, first.RecordID, second.RecordID, "no-op must echo the existing head's RecordID, not mint a new one")
	require.Equal(t, first.Version, second.Version, "no-op must not increment version")
	require.Equal(t, first.Timestamp, second.Timestamp, "no-op must not advance the stored timestamp")
	require.Equal(t, stateLenAfterFirst, len(stub.State), "no-op must not add or change any world-state key")
}

// TestRecordProfileSection_RejectsStalePrevHash covers both stale-chain-
// reference shapes: (a) a head already exists and prevHash doesn't match it,
// and (b) no head exists yet but a non-empty prevHash was supplied anyway.
func TestRecordProfileSection_RejectsStalePrevHash(t *testing.T) {
	sc := &SmartContract{}

	t.Run("prevHash does not match an existing head", func(t *testing.T) {
		ctx, _ := newTestContext(t, mspOrg1)
		_, err := sc.RecordProfileSection(ctx, "tenant01", "emp-hmac-2", "PAYROLL",
			"sha256:v1", "", "actor-hmac-1", nil, "JCS-RFC8785-v1", "SHA-256", "")
		require.NoError(t, err)

		_, err = sc.RecordProfileSection(ctx, "tenant01", "emp-hmac-2", "PAYROLL",
			"sha256:v2", "sha256:WRONG-PREV-HASH", "actor-hmac-1", nil, "JCS-RFC8785-v1", "SHA-256", "")
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrStaleChainReference))
	})

	t.Run("non-empty prevHash supplied when no head exists yet", func(t *testing.T) {
		ctx, _ := newTestContext(t, mspOrg1)
		_, err := sc.RecordProfileSection(ctx, "tenant01", "emp-hmac-3", "EDUCATION",
			"sha256:v1", "sha256:should-have-been-empty", "actor-hmac-1", nil, "JCS-RFC8785-v1", "SHA-256", "")
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrStaleChainReference))
	})
}

// TestRecordProfileSection_ChainsCorrectlyAcrossVersions confirms the
// per-(employeeID, profileSection) hash-chain advances the way data-model.md
// §2.1/§5 requires: version increments, prevHash of version N+1 equals
// dataHash of version N.
func TestRecordProfileSection_ChainsCorrectlyAcrossVersions(t *testing.T) {
	sc := &SmartContract{}
	ctx, _ := newTestContext(t, mspOrg1)

	v1, err := sc.RecordProfileSection(ctx, "tenant01", "emp-hmac-4", "EMPLOYMENT",
		"sha256:v1", "", "actor-hmac-1", nil, "JCS-RFC8785-v1", "SHA-256", "")
	require.NoError(t, err)
	require.Equal(t, 1, v1.Version)

	v2, err := sc.RecordProfileSection(ctx, "tenant01", "emp-hmac-4", "EMPLOYMENT",
		"sha256:v2", "sha256:v1", "actor-hmac-1", nil, "JCS-RFC8785-v1", "SHA-256", "")
	require.NoError(t, err)
	require.Equal(t, 2, v2.Version)

	head, err := sc.GetProfileSectionRecord(ctx, "tenant01", "emp-hmac-4", "EMPLOYMENT")
	require.NoError(t, err)
	require.Equal(t, "sha256:v2", head.DataHash)
	require.Equal(t, 2, head.Version)
}

// TestRecordProfileSection_RejectsInvalidProfileSection is FR-8.
func TestRecordProfileSection_RejectsInvalidProfileSection(t *testing.T) {
	sc := &SmartContract{}
	ctx, _ := newTestContext(t, mspOrg1)

	_, err := sc.RecordProfileSection(ctx, "tenant01", "emp-hmac-5", "SALARY",
		"sha256:v1", "", "actor-hmac-1", nil, "JCS-RFC8785-v1", "SHA-256", "")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrInvalidArgument))
}

// TestRecordProfileSection_RejectsCallerThatFailsMSPVerification is FR-6 —
// a caller whose identity cannot be determined at all must be rejected
// before any state is read or written.
func TestRecordProfileSection_RejectsCallerThatFailsMSPVerification(t *testing.T) {
	sc := &SmartContract{}
	ctx, stub := newTestContext(t, "") // no creator set at all

	_, err := sc.RecordProfileSection(ctx, "tenant01", "emp-hmac-6", "PAYROLL",
		"sha256:v1", "", "actor-hmac-1", nil, "JCS-RFC8785-v1", "SHA-256", "")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrUnauthorized))
	require.Empty(t, stub.State, "a rejected caller must never reach PutState")
}

// TestRecordProfileSection_PerOrgWriteAllowDeny is the write-side allow/deny
// pair exercised through the real Submit function, per org (CC-4 DoD).
func TestRecordProfileSection_PerOrgWriteAllowDeny(t *testing.T) {
	sc := &SmartContract{}
	cases := []struct {
		name    string
		mspID   string
		allowed bool
	}{
		{"Org1MSP (platform) may write", mspOrg1, true},
		{"OrgClient-tenant01MSP (tenant enterprise-client org) may write", "OrgClient-tenant01MSP", true},
		{"Org3MSP (auditor, read-only) may NOT write", mspOrg3, false},
		{"an unrecognized MSP may NOT write", "Org4MSP", false},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, stub := newTestContext(t, tc.mspID)
			employeeID := "emp-hmac-org-" + string(rune('a'+i))
			_, err := sc.RecordProfileSection(ctx, "tenant01", employeeID, "PAYROLL",
				"sha256:v1", "", "actor-hmac-1", nil, "JCS-RFC8785-v1", "SHA-256", "")
			if tc.allowed {
				require.NoError(t, err)
				require.NotEmpty(t, stub.State)
			} else {
				require.Error(t, err)
				require.True(t, errors.Is(err, ErrUnauthorized))
				require.Empty(t, stub.State, "a denied write must never reach PutState")
			}
		})
	}
}

// TestRecordProfileSection_RequiredArgumentsValidated is a defensive-
// validation check beyond the enum (implementer's choice, api-contracts.md
// does not mandate the mechanism) — every required field rejects when empty.
func TestRecordProfileSection_RequiredArgumentsValidated(t *testing.T) {
	sc := &SmartContract{}
	base := func() (tenantId, employeeID, dataHash, updatedBy, canon, algo string) {
		return "tenant01", "emp-hmac-7", "sha256:v1", "actor-hmac-1", "JCS-RFC8785-v1", "SHA-256"
	}

	t.Run("empty employeeID rejected", func(t *testing.T) {
		ctx, _ := newTestContext(t, mspOrg1)
		tenantId, _, dataHash, updatedBy, canon, algo := base()
		_, err := sc.RecordProfileSection(ctx, tenantId, "", "PAYROLL", dataHash, "", updatedBy, nil, canon, algo, "")
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrInvalidArgument))
	})

	t.Run("empty dataHash rejected", func(t *testing.T) {
		ctx, _ := newTestContext(t, mspOrg1)
		tenantId, employeeID, _, updatedBy, canon, algo := base()
		_, err := sc.RecordProfileSection(ctx, tenantId, employeeID, "PAYROLL", "", "", updatedBy, nil, canon, algo, "")
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrInvalidArgument))
	})
}

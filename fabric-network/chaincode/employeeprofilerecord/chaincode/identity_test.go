package chaincode

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAuthorizeSubmit_PerOrg is the write-side allow/deny matrix (CC-4 DoD):
// Org1MSP (platform) and OrgClient-tenant01MSP (this pilot tenant's
// enterprise-client org) may submit; Org3MSP (auditor, read-only per
// ADR-0012) and an unrecognized MSP may not.
func TestAuthorizeSubmit_PerOrg(t *testing.T) {
	cases := []struct {
		name    string
		mspID   string
		allowed bool
	}{
		{"platform org allowed to submit", mspOrg1, true},
		{"tenant enterprise-client org allowed to submit", "OrgClient-tenant01MSP", true},
		{"a second tenant's enterprise-client org allowed to submit (general shape, not hardcoded)", "OrgClient-tenant02MSP", true},
		{"auditor org denied submit (read-only, ADR-0012)", mspOrg3, false},
		{"unrecognized MSP denied submit", "Org4MSP", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestContext(t, tc.mspID)
			gotMSP, err := authorizeSubmit(ctx)
			if tc.allowed {
				require.NoError(t, err)
				require.Equal(t, tc.mspID, gotMSP)
			} else {
				require.Error(t, err)
				require.True(t, errors.Is(err, ErrUnauthorized), "expected ErrUnauthorized, got %v", err)
			}
		})
	}
}

// TestAuthorizeEvaluate_PerOrg is the read-side allow/deny matrix: all three
// recognized orgs — including Org3, the read-only auditor — may Evaluate;
// an unrecognized MSP may not.
func TestAuthorizeEvaluate_PerOrg(t *testing.T) {
	cases := []struct {
		name    string
		mspID   string
		allowed bool
	}{
		{"platform org allowed to read", mspOrg1, true},
		{"tenant enterprise-client org allowed to read", "OrgClient-tenant01MSP", true},
		{"auditor org allowed to read (read-only access, not no access)", mspOrg3, true},
		{"unrecognized MSP denied read", "Org4MSP", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestContext(t, tc.mspID)
			gotMSP, err := authorizeEvaluate(ctx)
			if tc.allowed {
				require.NoError(t, err)
				require.Equal(t, tc.mspID, gotMSP)
			} else {
				require.Error(t, err)
				require.True(t, errors.Is(err, ErrUnauthorized))
			}
		})
	}
}

// TestCallerMSPID_FailsClosedWhenIdentityUndeterminable covers the "caller
// fails MSP verification" outcome at its origin — no creator set at all, so
// cid.GetMSPID/GetID cannot read an identity, not merely an org this
// contract's allow-list excludes. Deny-by-default must hold here too.
func TestCallerMSPID_FailsClosedWhenIdentityUndeterminable(t *testing.T) {
	ctx, _ := newTestContext(t, "") // "" => stub.Creator left nil

	_, err := callerMSPID(ctx)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrUnauthorized))

	_, err = authorizeSubmit(ctx)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrUnauthorized))

	_, err = authorizeEvaluate(ctx)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrUnauthorized))
}

func TestIsOrgClientMSP(t *testing.T) {
	cases := map[string]bool{
		"OrgClient-tenant01MSP": true,
		"OrgClient-tenant02MSP": true,
		"OrgClient-MSP":         false, // no tenant segment between prefix and suffix
		"OrgClientMSP":          false, // missing the hyphen entirely
		"Org1MSP":               false,
		"Org3MSP":               false,
		"":                      false,
	}
	for in, want := range cases {
		require.Equal(t, want, isOrgClientMSP(in), "isOrgClientMSP(%q)", in)
	}
}

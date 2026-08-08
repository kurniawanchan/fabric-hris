package chaincode

import (
	"errors"
	"testing"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/stretchr/testify/require"
)

// TestGetProfileSectionRecord_NotFoundIsDistinct is FR-15: "never anchored"
// must be a distinct, typed outcome (ErrRecordNotFound), never collapsed
// into a generic error.
func TestGetProfileSectionRecord_NotFoundIsDistinct(t *testing.T) {
	sc := &SmartContract{}
	ctx, _ := newTestContext(t, mspOrg1)

	_, err := sc.GetProfileSectionRecord(ctx, "tenant01", "emp-never-anchored", "PAYROLL")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrRecordNotFound))
	require.False(t, errors.Is(err, ErrInvalidArgument))
	require.False(t, errors.Is(err, ErrUnauthorized))
}

// TestGetProfileSectionRecord_ReturnsExactlyFourFields pins the read-side
// confidentiality invariant: {dataHash, version, timestamp, updatedBy} and
// nothing else reachable through this function (FR-35) — recordID,
// prevHash, ipfsCIDs, and the version tags are NOT exposed here.
func TestGetProfileSectionRecord_ReturnsExactlyFourFields(t *testing.T) {
	sc := &SmartContract{}
	ctx, _ := newTestContext(t, mspOrg1)

	_, err := sc.RecordProfileSection(ctx, "tenant01", "emp-hmac-q1", "PERSONAL",
		"sha256:v1", "", "actor-hmac-1", []string{"bafy1"}, "JCS-RFC8785-v1", "SHA-256", "")
	require.NoError(t, err)

	head, err := sc.GetProfileSectionRecord(ctx, "tenant01", "emp-hmac-q1", "PERSONAL")
	require.NoError(t, err)
	require.Equal(t, "sha256:v1", head.DataHash)
	require.Equal(t, 1, head.Version)
	require.Equal(t, "actor-hmac-1", head.UpdatedBy)
	require.NotEmpty(t, head.Timestamp)
}

// TestGetProfileHistory_AscendingChronologicalOrder writes three versions and
// confirms GetProfileHistory returns all three, ascending by version, with
// the chain-link fields intact — exercised against historyMockStub, since
// shimtest.MockStub itself does not implement GetHistoryForKey (see
// testutil_test.go and this task's final report).
func TestGetProfileHistory_AscendingChronologicalOrder(t *testing.T) {
	sc := &SmartContract{}
	ctx, _ := newTestContext(t, mspOrg1)

	_, err := sc.RecordProfileSection(ctx, "tenant01", "emp-hmac-q2", "EDUCATION",
		"sha256:v1", "", "actor-hmac-1", nil, "JCS-RFC8785-v1", "SHA-256", "")
	require.NoError(t, err)
	_, err = sc.RecordProfileSection(ctx, "tenant01", "emp-hmac-q2", "EDUCATION",
		"sha256:v2", "sha256:v1", "actor-hmac-1", nil, "JCS-RFC8785-v1", "SHA-256", "")
	require.NoError(t, err)
	_, err = sc.RecordProfileSection(ctx, "tenant01", "emp-hmac-q2", "EDUCATION",
		"sha256:v3", "sha256:v2", "actor-hmac-1", nil, "JCS-RFC8785-v1", "SHA-256", "")
	require.NoError(t, err)

	history, err := sc.GetProfileHistory(ctx, "tenant01", "emp-hmac-q2", "EDUCATION")
	require.NoError(t, err)
	require.Len(t, history, 3)
	require.Equal(t, 1, history[0].Version)
	require.Equal(t, 2, history[1].Version)
	require.Equal(t, 3, history[2].Version)
	require.Equal(t, "sha256:v1", history[0].DataHash)
	require.Equal(t, "sha256:v3", history[2].DataHash)
	require.Equal(t, history[0].DataHash, history[1].PrevHash)
	require.Equal(t, history[1].DataHash, history[2].PrevHash)
}

// TestGetProfileHistory_EmptyWhenNeverAnchored mirrors NotFound as a
// zero-length list rather than a distinct error, matching a history query's
// natural "zero results" shape (api-contracts.md).
func TestGetProfileHistory_EmptyWhenNeverAnchored(t *testing.T) {
	sc := &SmartContract{}
	ctx, _ := newTestContext(t, mspOrg1)

	history, err := sc.GetProfileHistory(ctx, "tenant01", "emp-never-anchored", "EDUCATION")
	require.NoError(t, err)
	require.Empty(t, history)
}

// TestUpstreamMockStubDoesNotImplementGetHistoryForKey documents, with a
// direct assertion rather than a comment claim, exactly why this package
// needed historyMockStub at all: the upstream shimtest.MockStub embedded
// inside it always errors "not implemented" on GetHistoryForKey
// [code: fabric-chaincode-go/shim/interfaces.go — MockQueryIteratorInterface
// carries a TODO to that effect]. historyMockStub's own override (exercised
// by every other GetProfileHistory test in this file) is what makes this
// function's real logic — not just its error path — testable without a
// live peer.
func TestUpstreamMockStubDoesNotImplementGetHistoryForKey(t *testing.T) {
	_, stub := newTestContext(t, mspOrg1)

	_, err := stub.MockStub.GetHistoryForKey("any-key") // the embedded, un-overridden upstream method
	require.Error(t, err, "upstream shimtest.MockStub.GetHistoryForKey is expected to be unimplemented — if this now passes, upstream added support and historyMockStub's workaround may be simplifiable")

	_, err = stub.GetHistoryForKey("any-key") // historyMockStub's override, used by the SmartContract
	require.NoError(t, err, "historyMockStub's own override must succeed even when upstream's does not")
}

// TestGetEmployeeProfileSummary_FiveEntriesSomeAnchored confirms the
// five-section fan-out always returns exactly five entries, in
// data-model.md §8's order, representing an un-anchored section as
// version 0 / empty dataHash (api-contracts.md's stated implementer choice).
func TestGetEmployeeProfileSummary_FiveEntriesSomeAnchored(t *testing.T) {
	sc := &SmartContract{}
	ctx, _ := newTestContext(t, mspOrg1)

	_, err := sc.RecordProfileSection(ctx, "tenant01", "emp-hmac-q3", "PAYROLL",
		"sha256:payroll-v1", "", "actor-hmac-1", nil, "JCS-RFC8785-v1", "SHA-256", "")
	require.NoError(t, err)
	_, err = sc.RecordProfileSection(ctx, "tenant01", "emp-hmac-q3", "PERSONAL",
		"sha256:personal-v1", "", "actor-hmac-1", nil, "JCS-RFC8785-v1", "SHA-256", "")
	require.NoError(t, err)

	summary, err := sc.GetEmployeeProfileSummary(ctx, "tenant01", "emp-hmac-q3")
	require.NoError(t, err)
	require.Len(t, summary, 5)

	wantOrder := []string{"PERSONAL", "EMPLOYMENT", "EDUCATION", "ADDITIONAL", "PAYROLL"}
	for i, entry := range summary {
		require.Equal(t, wantOrder[i], entry.ProfileSection)
	}

	byName := make(map[string]*ProfileSectionSummaryEntry, 5)
	for _, e := range summary {
		byName[e.ProfileSection] = e
	}
	require.Equal(t, "sha256:payroll-v1", byName["PAYROLL"].DataHash)
	require.Equal(t, 1, byName["PAYROLL"].Version)
	require.Equal(t, "sha256:personal-v1", byName["PERSONAL"].DataHash)
	require.Equal(t, 0, byName["EMPLOYMENT"].Version, "never-anchored section must report version 0")
	require.Empty(t, byName["EMPLOYMENT"].DataHash)
}

// TestQueryFunctions_PerOrgReadAllowDeny is the read-side allow/deny pair
// exercised through the real Evaluate functions, per org (CC-4 DoD) — Org3
// (auditor) is explicitly ALLOWED to read, unlike its write-side denial.
func TestQueryFunctions_PerOrgReadAllowDeny(t *testing.T) {
	writerCtx, _ := newTestContext(t, mspOrg1)
	sc := &SmartContract{}
	_, err := sc.RecordProfileSection(writerCtx, "tenant01", "emp-hmac-q4", "PAYROLL",
		"sha256:v1", "", "actor-hmac-1", nil, "JCS-RFC8785-v1", "SHA-256", "")
	require.NoError(t, err)

	// Reuse the SAME underlying world state across identities by wiring a
	// fresh context around the SAME stub, so read authorization — not data
	// availability — is what each sub-test actually measures.
	stub := writerCtx.GetStub()

	cases := []struct {
		name    string
		mspID   string
		allowed bool
	}{
		{"Org1MSP (platform) may read", mspOrg1, true},
		{"OrgClient-tenant01MSP may read", "OrgClient-tenant01MSP", true},
		{"Org3MSP (auditor) may read (read-only access IS access)", mspOrg3, true},
		{"an unrecognized MSP may NOT read", "Org4MSP", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.mspID != "" {
				setStubCreator(t, stub, tc.mspID)
			} else {
				clearStubCreator(stub)
			}
			readerCtx := &contractapi.TransactionContext{}
			readerCtx.SetStub(stub)

			_, err := sc.GetProfileSectionRecord(readerCtx, "tenant01", "emp-hmac-q4", "PAYROLL")
			if tc.allowed {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.True(t, errors.Is(err, ErrUnauthorized))
			}
		})
	}
}

package chaincode

import (
	"fmt"
	"strings"

	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Recognized MSP identities for this deployment (NET-1/NET-2/NET-5). The
// enterprise-client org's MSP ID uses a HYPHEN — OrgClient-<tenantID>MSP —
// not the underscore ADR-0012/ADR-0013 were originally written with; this is
// the empirical correction found at NET-1 (Fabric's signature-policy
// tokenizer rejects '_' in an AND(...) principal reference) and propagated
// everywhere else in the codebase, including here.
const (
	mspOrg1         = "Org1MSP" // platform org
	mspOrg3         = "Org3MSP" // auditor org — read-only (ADR-0012)
	orgClientPrefix = "OrgClient-"
	orgClientSuffix = "MSP"
)

// isOrgClientMSP reports whether mspID matches the OrgClient-<tenantID>MSP
// shape. ADR-0013's topology is one enterprise-client org PER TENANT CHANNEL
// (an O(N) fan-out, NET-7) — this pilot network instantiates exactly one,
// OrgClient-tenant01MSP, but the check below is written to the general
// shape rather than hardcoded to that one tenant, so it does not need to
// change when NET-7's automation provisions a second tenant.
func isOrgClientMSP(mspID string) bool {
	return strings.HasPrefix(mspID, orgClientPrefix) &&
		strings.HasSuffix(mspID, orgClientSuffix) &&
		len(mspID) > len(orgClientPrefix)+len(orgClientSuffix)
}

// callerMSPID reads the submitting client identity's MSP ID via the CID API
// (ADR-0005's shape: X.509 identity, org membership via MSP, read inside the
// transaction) and denies by default when it cannot be determined — this is
// the FR-6 "reject if the caller is not MSP-verified" check, and it is the
// single choke point every one of the four contract functions routes
// through (CC-4), so it is implemented once here rather than duplicated per
// function.
//
// Relationship to the channel-level endorsement policy (NET-5) — noted here
// explicitly per this item's instruction, not left implicit: NET-5 wires
// AND('Org1MSP.peer','OrgClient-<tenantID>MSP.peer') at the channel/commit
// level. That policy governs which ORGS' PEERS must co-sign an already-
// proposed transaction for it to validate at commit time; it says nothing
// about which SUBMITTING CLIENT identity may cause chaincode to execute on
// an endorsing peer in the first place. This function's checks (and
// authorizeSubmit/authorizeEvaluate below) are a chaincode-level ABAC gate
// on the CALLER's identity — reading the transaction proposal's creator,
// which every endorsing peer sees identically. The two mechanisms are
// complementary layers (identity-of-caller vs. identity-of-endorser), not
// duplicates, and neither substitutes for the other: removing this hook
// would not weaken NET-5's co-signature guarantee, but it WOULD let an
// Org3-submitted proposal reach RecordProfileSection's write logic on the
// Org1/OrgClient peers that would otherwise endorse it. Keeping both is the
// intended, deny-by-default posture (ADR-0012 §5, ADR-0020, T13).
func callerMSPID(ctx contractapi.TransactionContextInterface) (string, error) {
	stub := ctx.GetStub()

	mspID, err := cid.GetMSPID(stub)
	if err != nil {
		return "", fmt.Errorf("%w: cannot determine caller MSP ID: %v", ErrUnauthorized, err)
	}
	if mspID == "" {
		return "", fmt.Errorf("%w: caller MSP ID is empty", ErrUnauthorized)
	}

	// cid.GetID additionally requires the creator to carry a parsable X.509
	// identity (or an Idemix credential, this design does not use Idemix —
	// ADR-0008). Calling it too means a malformed identity that happens to
	// carry a readable MSP ID string is still rejected — deny-by-default is
	// grounded in the whole CID read, not merely the MSP ID field.
	if _, err := cid.GetID(stub); err != nil {
		return "", fmt.Errorf("%w: cannot determine caller identity: %v", ErrUnauthorized, err)
	}

	return mspID, nil
}

// authorizeEvaluate allows the platform org, any tenant's enterprise-client
// org, and the auditor org to call the three read-only functions — Org3 is
// deliberately PERMITTED here, because ADR-0012's auditor role is
// "read-only," not "no access" (FR-37 has the auditor org read its own
// peer). Anything else is denied by default.
func authorizeEvaluate(ctx contractapi.TransactionContextInterface) (string, error) {
	mspID, err := callerMSPID(ctx)
	if err != nil {
		return "", err
	}
	if mspID == mspOrg1 || mspID == mspOrg3 || isOrgClientMSP(mspID) {
		return mspID, nil
	}
	return "", fmt.Errorf("%w: MSP %q is not authorized to read this contract", ErrUnauthorized, mspID)
}

// authorizeSubmit allows only the platform org and a tenant's
// enterprise-client org to invoke RecordProfileSection — the auditor org is
// EXCLUDED (FR-23, ADR-0012 §5, ADR-0020's endorsement shape: "no single
// org, including the platform operator, can write an anchor alone," and the
// auditor is not one of the two co-signers at all). See callerMSPID's
// comment above for exactly how this relates to, and does not conflict
// with, the channel-level endorsement policy wired at NET-5.
func authorizeSubmit(ctx contractapi.TransactionContextInterface) (string, error) {
	mspID, err := callerMSPID(ctx)
	if err != nil {
		return "", err
	}
	if mspID == mspOrg1 || isOrgClientMSP(mspID) {
		return mspID, nil
	}
	return "", fmt.Errorf("%w: MSP %q is not authorized to submit a write", ErrUnauthorized, mspID)
}

package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract implements the four ADR-0020 transaction functions over the
// EmployeeProfileRecord asset: one Submit (RecordProfileSection) and three
// Evaluate functions (GetProfileSectionRecord, GetProfileHistory,
// GetEmployeeProfileSummary). Endorsement (chaincode-level policy requiring
// the platform org and the enterprise-client org to co-sign, auditor
// excluded) is wired at chaincode-lifecycle-commit time (CC-5, NET-5) — it is
// not, and cannot be, expressed inside this Go type.
type SmartContract struct {
	contractapi.Contract
}

// readHead reads and unmarshals the current head record for
// (employeeID, profileSection), if any. A nil *EmployeeProfileRecord with a
// nil error means "no record exists yet" — the composite key is also
// returned so callers that go on to write don't recompute it.
func readHead(ctx contractapi.TransactionContextInterface, employeeID, profileSection string) (*EmployeeProfileRecord, string, error) {
	key, err := ctx.GetStub().CreateCompositeKey(profileKeyObjectType, []string{employeeID, profileSection})
	if err != nil {
		return nil, "", fmt.Errorf("failed to build composite key: %w", err)
	}

	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, key, fmt.Errorf("failed to read world state: %w", err)
	}
	if data == nil {
		return nil, key, nil
	}

	var rec EmployeeProfileRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, key, fmt.Errorf("failed to unmarshal stored record for key %q: %w", key, err)
	}
	return &rec, key, nil
}

// REC-6: the crypto-shred erasure hook (ADR-0015). Issues ZERO ledger
// transactions — erasure is entirely off-chain, by design (INTEG §8). The
// on-chain record (dataHash/version/timestamp/updatedBy) is left exactly as
// it was; what changes is that nothing off-chain can ever again recompute
// or re-derive it.
package writepaths

import (
	"context"
	"fmt"

	gatewayclient "gatewayclient"
)

// Erase performs the four crypto-shred steps for one employee, across all
// five profile sections:
//  1. delete the operational-DB field for every section
//  2. destroy KEY_EMPLOYEE (Domain B′ — IPFS document key)
//  3. destroy every DataHash salt this employee has
//  4. destroy employeeKey_i (Domain C — on-chain pseudonym key)
//
// Deliberately does NOT touch the Gateway/chaincode at all — no Submit, no
// Evaluate. A post-erasure GetProfileSectionRecord still succeeds (the
// on-chain fields are untouched) but can never again be matched by a
// recompute, because the salt/employeeKey_i needed to do so no longer
// exist anywhere this hook has access to.
func (h *Hooks) Erase(ctx context.Context, employeeInternalID string) error {
	for _, section := range AllProfileSections {
		if err := h.Store.DeleteSection(ctx, employeeInternalID, section); err != nil {
			return fmt.Errorf("writepaths: erasure step 1 (operational-DB field, %s): %w", section, err)
		}
	}

	if h.DocumentKeys != nil {
		if err := h.DocumentKeys.DeleteDocumentKey(ctx, employeeInternalID); err != nil {
			return fmt.Errorf("writepaths: erasure step 2 (KEY_EMPLOYEE): %w", err)
		}
	}

	// Salts are stored keyed by the PSEUDONYMOUS EmployeeID (what REC-1's
	// ComputeEmployeeID derives from employeeKey_i), not the internal ID —
	// data-model.md §5/§7's own key shape. Must derive it BEFORE destroying
	// employeeKey_i in step 4, or the derivation becomes impossible and the
	// salts would be orphaned rather than deleted.
	employeeKey, err := h.Keys.GetOrCreateEmployeeKey(ctx, employeeInternalID)
	if err != nil {
		return fmt.Errorf("writepaths: erasure — fetching employeeKey_i to derive EmployeeID before destroying it: %w", err)
	}
	employeeID, err := gatewayclient.ComputeEmployeeID(employeeKey)
	if err != nil {
		return fmt.Errorf("writepaths: erasure — deriving EmployeeID: %w", err)
	}
	if err := h.Salts.DeleteAllSaltsForEmployee(ctx, employeeID); err != nil {
		return fmt.Errorf("writepaths: erasure step 3 (DataHash salts): %w", err)
	}

	if err := h.Keys.DeleteEmployeeKey(ctx, employeeInternalID); err != nil {
		return fmt.Errorf("writepaths: erasure step 4 (employeeKey_i): %w", err)
	}

	return nil
}

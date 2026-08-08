package chaincode

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// GetProfileSectionRecord returns exactly the stored head's hash-bearing
// fields for one employee's one section — never the full asset
// (api-contracts.md "GetProfileSectionRecord", FR-35). No candidate value or
// salt is ever a parameter here (FR-34); recomputation and comparison happen
// entirely client-side (api-contracts.md "Client-side verification
// protocol") — this function's role ends at returning what is stored.
//
// A genuine "no record exists yet" outcome is ErrRecordNotFound, distinct
// from every other error (FR-15) — callers should use errors.Is against it,
// never string-match the message.
func (s *SmartContract) GetProfileSectionRecord(
	ctx contractapi.TransactionContextInterface,
	tenantId string,
	employeeID string,
	profileSection string,
) (*ProfileSectionHead, error) {
	if _, err := authorizeEvaluate(ctx); err != nil {
		return nil, err
	}
	if !IsValidProfileSection(profileSection) {
		return nil, fmt.Errorf("%w: profileSection %q is not one of the five ratified values", ErrInvalidArgument, profileSection)
	}

	head, _, err := readHead(ctx, employeeID, profileSection)
	if err != nil {
		return nil, err
	}
	if head == nil {
		return nil, ErrRecordNotFound
	}

	return &ProfileSectionHead{
		DataHash:  head.DataHash,
		Version:   head.Version,
		Timestamp: head.Timestamp,
		UpdatedBy: head.UpdatedBy,
	}, nil
}

// GetProfileHistory returns the chronological version list for one
// employee's one section (api-contracts.md "GetProfileHistory"), sourced
// from the ledger's own per-key history — GetHistoryForKey — never a
// chaincode-maintained secondary index (data-model.md §5). An empty slice
// means the section has never been anchored (the same "not found" concept as
// GetProfileSectionRecord's ErrRecordNotFound, expressed as a zero-length
// list rather than a distinct error, matching a history query's natural
// "zero results" shape).
//
// [FLAGGED UNCERTAINTY] This function's ordering guarantee ("ascending
// chronological order," api-contracts.md) is enforced HERE by explicitly
// sorting the returned entries by Version, rather than assumed from
// GetHistoryForKey's own iteration order. This project's corpus does not
// pin that iteration order for this contract's benefit, and the
// shimtest.MockStub double used in this package's own unit tests does not
// implement GetHistoryForKey at all (returns "not implemented" —
// see queries_test.go and this task's final report). Sorting is safe here
// regardless of the real peer's actual order, because GetProfileHistory is
// Evaluate-only (single-peer query, not compared across endorsers for
// commit), so re-ordering client-visible results has no determinism
// consequence the way it would inside RecordProfileSection.
func (s *SmartContract) GetProfileHistory(
	ctx contractapi.TransactionContextInterface,
	tenantId string,
	employeeID string,
	profileSection string,
) ([]*ProfileVersionEntry, error) {
	if _, err := authorizeEvaluate(ctx); err != nil {
		return nil, err
	}
	if !IsValidProfileSection(profileSection) {
		return nil, fmt.Errorf("%w: profileSection %q is not one of the five ratified values", ErrInvalidArgument, profileSection)
	}

	key, err := ctx.GetStub().CreateCompositeKey(profileKeyObjectType, []string{employeeID, profileSection})
	if err != nil {
		return nil, fmt.Errorf("failed to build composite key: %w", err)
	}

	iterator, err := ctx.GetStub().GetHistoryForKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to read key history: %w", err)
	}
	defer iterator.Close()

	entries := make([]*ProfileVersionEntry, 0)
	for iterator.HasNext() {
		mod, err := iterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate key history: %w", err)
		}
		if mod.IsDelete || len(mod.Value) == 0 {
			// This design never DelStates a profile key (data-model.md §11 —
			// erasure is entirely off-chain), so this branch should not be
			// reachable in practice; skip defensively rather than fail the
			// whole read on an unexpected tombstone.
			continue
		}

		var rec EmployeeProfileRecord
		if err := json.Unmarshal(mod.Value, &rec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal history entry: %w", err)
		}

		entries = append(entries, &ProfileVersionEntry{
			CanonicalizationVersion: rec.CanonicalizationVersion,
			DataHash:                rec.DataHash,
			HashAlgo:                rec.HashAlgo,
			IpfsCIDs:                rec.IpfsCIDs,
			PrevHash:                rec.PrevHash,
			RecordID:                rec.RecordID,
			Timestamp:               rec.Timestamp,
			UpdatedBy:               rec.UpdatedBy,
			Version:                 rec.Version,
		})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Version < entries[j].Version })
	return entries, nil
}

// GetEmployeeProfileSummary fans out to all five sections in one call
// (api-contracts.md "GetEmployeeProfileSummary") — a convenience
// aggregation, not a requirement that every section exist. A section that
// has never been anchored is represented by an entry with Version: 0 and an
// empty DataHash (one of the two implementer choices api-contracts.md
// explicitly allows) so the response is always exactly five entries, one per
// ProfileSection value, in data-model.md §8's listed order.
func (s *SmartContract) GetEmployeeProfileSummary(
	ctx contractapi.TransactionContextInterface,
	tenantId string,
	employeeID string,
) ([]*ProfileSectionSummaryEntry, error) {
	if _, err := authorizeEvaluate(ctx); err != nil {
		return nil, err
	}

	summary := make([]*ProfileSectionSummaryEntry, 0, len(allProfileSections))
	for _, section := range allProfileSections {
		head, _, err := readHead(ctx, employeeID, string(section))
		if err != nil {
			return nil, err
		}
		if head == nil {
			summary = append(summary, &ProfileSectionSummaryEntry{
				ProfileSection: string(section),
				DataHash:       "",
				Version:        0,
				Timestamp:      "",
				UpdatedBy:      "",
			})
			continue
		}
		summary = append(summary, &ProfileSectionSummaryEntry{
			ProfileSection: string(section),
			DataHash:       head.DataHash,
			Version:        head.Version,
			Timestamp:      head.Timestamp,
			UpdatedBy:      head.UpdatedBy,
		})
	}
	return summary, nil
}

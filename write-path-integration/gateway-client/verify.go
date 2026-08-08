// REC-7: client-side verification helper. Used identically by all three
// verifier classes (employee, enterprise-client auditor org, platform
// auditor org) — the only difference between them is WHICH already-
// configured GatewayClient they call this with (their own org's identity,
// pointed at their own org's peer, per ADR-0020's per-org-replica
// verification model). This function itself never branches on verifier
// class.
//
// The entire chaincode footprint of a call to this function is ONE
// Evaluate carrying only (tenantID, employeeID, profileSection) — no
// section value, no salt, ever crosses to a peer (ADR-0020's
// confidentiality boundary, restated here for the read side as it was for
// the write side in REC-1/CC-2).
package gatewayclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// VerificationResult is what a caller needs to report "does this record
// still match" without exposing the section value or salt anywhere beyond
// this call. Found distinguishes "never anchored" (Found: false, every other
// field left zero-valued) from a genuine on-chain read (Found: true, Matched
// meaningful) — INT-2's own DoD: a caller must be able to tell "this record
// doesn't exist yet" apart from "it exists and was tampered with", since
// only the latter is a client-side-decidable mismatch.
type VerificationResult struct {
	Found          bool
	Matched        bool
	OnChainHash    string
	RecomputedHash string
	OnChainVersion int
	UpdatedBy      string
	Timestamp      string
}

type onChainRecord struct {
	DataHash  string `json:"dataHash"`
	Version   int    `json:"version"`
	UpdatedBy string `json:"updatedBy"`
	Timestamp string `json:"timestamp"`
}

// Verify recomputes SHA-256(salt‖JCS(currentValueJSON)) LOCALLY and
// compares it against the on-chain DataHash for (tenantID, employeeID,
// profileSection), read via ONE Evaluate call against g's own configured
// peer. salt must already be in the caller's possession, obtained via the
// SEPARATE hand-off channel INT-3 defines — this function never accepts,
// requests, or derives a salt itself; it is purely a parameter.
func (g *GatewayClient) Verify(ctx context.Context, tenantID, employeeID, profileSection string, salt []byte, currentValueJSON []byte) (*VerificationResult, error) {
	recomputed, err := ComputeDataHash(salt, currentValueJSON)
	if err != nil {
		return nil, fmt.Errorf("gatewayclient: recomputing DataHash locally: %w", err)
	}

	onChainJSON, err := g.EvaluateGetProfileSectionRecord(ctx, tenantID, employeeID, profileSection)
	if err != nil {
		// "Never anchored" is a LEGITIMATE outcome, not an error — the
		// chaincode's own GetProfileSectionRecord treats it as a distinct
		// case (ErrRecordNotFound, api-contracts.md), and callers on this
		// side need the same distinction: NotFound is client-decidable
		// ("nothing to compare against yet"), whereas every other Evaluate
		// failure here is a genuine ledger/gateway problem that must still
		// surface as a *LedgerError, unchanged from before this method
		// existed.
		if isNotFoundRejection(err) {
			return &VerificationResult{Found: false}, nil
		}
		return nil, fmt.Errorf("gatewayclient: reading on-chain record: %w", err)
	}
	var rec onChainRecord
	if err := json.Unmarshal(onChainJSON, &rec); err != nil {
		return nil, fmt.Errorf("gatewayclient: parsing on-chain record: %w", err)
	}

	return &VerificationResult{
		Found:          true,
		Matched:        recomputed == rec.DataHash,
		OnChainHash:    rec.DataHash,
		RecomputedHash: recomputed,
		OnChainVersion: rec.Version,
		UpdatedBy:      rec.UpdatedBy,
		Timestamp:      rec.Timestamp,
	}, nil
}

// isNotFoundRejection recognizes the chaincode's OWN "never anchored"
// sentinel (ErrRecordNotFound) inside an Evaluate error. Same
// situation as isExpectedRetryableRejection's "stale chain reference"
// match in gatewayclient.go: the Gateway RPC boundary erases Go error
// identity, so errors.Is/errors.As cannot see through it — the sentinel's
// exact text is the only thing that survives the round trip, and matching
// against it is this codebase's established (if unlovely) precedent for
// this entire class of cross-process business-error detection, not a new
// pattern invented here.
func isNotFoundRejection(err error) bool {
	return strings.Contains(err.Error(), "profile section record not found")
}

package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// deterministicRecordID derives RecordID from the transaction ID rather than
// from a locally generated random UUID v4.
//
// [FLAGGED DEVIATION — confirm before treating as final] data-model.md §2.1
// specifies recordID as "UUID v4 ... assigned at write time," which literally
// implies crypto/rand-based generation. That is incompatible with Fabric's
// determinism requirement: every endorsing peer independently simulates this
// transaction and must produce a byte-identical write set, or the
// AND(Org1MSP.peer, OrgClient-<tenantID>MSP.peer) endorsement policy's two
// responses will never match and the transaction will never reach ordering
// [docs: chaincode4ade.rst#technical-problem] ("endorsement requires that,
// given the same inputs, chaincode produces the same byte array on every
// endorsing peer"). GetTxID() is fixed on the proposal before simulation
// begins, so it is identical across every endorsing peer for one
// transaction; this derives a UUID-*shaped* (RFC 4122 §4.3, version-5,
// name-based/deterministic) identifier from it instead of calling a random
// generator. RecordID is therefore technically derivable from the
// transaction ID (data-model.md §2.1 says "not derivable from anything
// else") — a deliberate, necessary trade against the harder correctness
// constraint. This should be confirmed with fabric-engineer/architect before
// being relied on as literally UUID-v4-compliant; it is flagged here rather
// than silently asserted as spec-compliant.
func deterministicRecordID(txID string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(txID)).String()
}

// requiredStringArgs validates that every named argument is non-empty,
// returning ErrInvalidArgument (wrapped, naming the first empty field) on the
// first failure. This is an implementer's defensive validation choice
// (api-contracts.md marks per-field "required: yes" but does not mandate a
// specific rejection mechanism) — not itself a numbered step of the
// contract's Behavior list.
func requiredStringArgs(fields map[string]string) error {
	// Iterate a fixed, explicit order rather than Go's randomized map
	// iteration, so the reported field name is deterministic across runs —
	// map iteration order is one of the nondeterminism sources the
	// determinism checklist calls out, and even though this helper runs
	// before any state read/write (an argument-shape check, not ledger
	// logic), keeping it deterministic avoids surprising, run-dependent
	// error messages.
	order := []string{"tenantId", "employeeID", "dataHash", "updatedBy", "canonicalizationVersion", "hashAlgo"}
	for _, name := range order {
		if v, ok := fields[name]; ok && v == "" {
			return fmt.Errorf("%w: %s is required", ErrInvalidArgument, name)
		}
	}
	return nil
}

// RecordProfileSection anchors one profile section's current state
// (api-contracts.md "RecordProfileSection — Submit"). It is called in-band by
// the HRIS write path that just committed that section to the operational
// database (ADR-0014, REC-3/REC-4 — not implemented by this chaincode
// module).
//
// Zero bytes of profile-section plaintext or salt are ever accepted here —
// there is no parameter typed to carry either (ADR-0020's confidentiality
// boundary, data-model.md §1). The argument list below is exactly
// api-contracts.md's Request block, in the same order.
func (s *SmartContract) RecordProfileSection(
	ctx contractapi.TransactionContextInterface,
	tenantId string,
	employeeID string,
	profileSection string,
	dataHash string,
	prevHash string,
	updatedBy string,
	ipfsCIDs []string,
	canonicalizationVersion string,
	hashAlgo string,
	clientTimestamp string, // OPTIONAL (empty string = not supplied); informational only, NEVER authoritative (api-contracts.md "Conventions — Timestamps").
) (*RecordProfileSectionResult, error) {
	// Behavior step 1 (api-contracts.md): profileSection enum check, FR-8.
	if !IsValidProfileSection(profileSection) {
		return nil, fmt.Errorf("%w: profileSection %q is not one of the five ratified values", ErrInvalidArgument, profileSection)
	}

	// Behavior step 2 (api-contracts.md) / FR-6 / CC-4: caller identity must
	// pass MSP/ABAC verification before anything else is read or written —
	// deny-by-default, implemented together with the enum check rather than
	// as an unrelated add-on (this item's explicit instruction).
	if _, err := authorizeSubmit(ctx); err != nil {
		return nil, err
	}

	// Defensive argument-shape validation beyond the enum (implementer's
	// choice, api-contracts.md does not mandate the mechanism).
	//
	// [FLAGGED GAP — G-31, found at QA-5's ST-1 full-ledger scan, 2026-08-07]
	// requiredStringArgs below checks non-emptiness ONLY — it does not check
	// that employeeID/updatedBy are actually 64-hex-char HMAC-SHA256 output,
	// or that dataHash/prevHash are actually "sha256:<64 hex>", the shapes
	// api-contracts.md's own "digest format" convention describes. Nothing
	// here stops a caller from submitting a raw, human-readable identifier
	// instead of a real pseudonym — ST-1's scan found exactly this had
	// already happened 31,895 times on the live demo ledger (test suites
	// calling this function directly with fixture literals, bypassing
	// writepaths.Hooks.anchor()'s real pseudonymization). No real PII was
	// found in that scan (independently triple-checked), but the underlying
	// enforcement gap is real: this function currently trusts every caller
	// to have already pseudonymized correctly, with no defense-in-depth
	// check of its own. See grounding-gaps.md G-31 — not fixed here.
	if err := requiredStringArgs(map[string]string{
		"tenantId":                tenantId,
		"employeeID":              employeeID,
		"dataHash":                dataHash,
		"updatedBy":               updatedBy,
		"canonicalizationVersion": canonicalizationVersion,
		"hashAlgo":                hashAlgo,
	}); err != nil {
		return nil, err
	}
	if ipfsCIDs == nil {
		ipfsCIDs = []string{}
	}

	// Behavior step 3: read the current head for (employeeID, profileSection).
	head, key, err := readHead(ctx, employeeID, profileSection)
	if err != nil {
		return nil, err
	}

	// Behavior step 4 (FR-9): identical dataHash to the current head is a
	// no-op — return the existing head's result unchanged, write nothing.
	if head != nil && head.DataHash == dataHash {
		return &RecordProfileSectionResult{
			RecordID:  head.RecordID,
			Version:   head.Version,
			Timestamp: head.Timestamp,
		}, nil
	}

	// Behavior step 5: prevHash must chain to the current head (or be empty
	// iff no head exists yet) — otherwise this is a stale chain reference.
	var nextVersion int
	switch {
	case head == nil && prevHash != "":
		return nil, fmt.Errorf("%w: no existing record for this key, but prevHash %q was supplied (expected \"\")", ErrStaleChainReference, prevHash)
	case head == nil:
		nextVersion = 1
	case prevHash != head.DataHash:
		return nil, fmt.Errorf("%w: prevHash %q does not match current head dataHash %q", ErrStaleChainReference, prevHash, head.DataHash)
	default:
		nextVersion = head.Version + 1
	}

	// The AUTHORITATIVE timestamp is the ledger transaction timestamp — never
	// clientTimestamp, which is accepted above purely for the caller's own
	// bookkeeping and is not read again from this point on.
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return nil, fmt.Errorf("failed to read ledger transaction timestamp: %w", err)
	}
	timestamp := time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos)).UTC().Format(time.RFC3339)

	recordID := deterministicRecordID(ctx.GetStub().GetTxID())

	// Behavior step 6: write the new version.
	record := EmployeeProfileRecord{
		CanonicalizationVersion: canonicalizationVersion,
		DataHash:                dataHash,
		EmployeeID:              employeeID,
		HashAlgo:                hashAlgo,
		IpfsCIDs:                ipfsCIDs,
		PrevHash:                prevHash,
		ProfileSection:          profileSection,
		RecordID:                recordID,
		TenantID:                tenantId,
		Timestamp:               timestamp,
		UpdatedBy:               updatedBy,
		Version:                 nextVersion,
	}

	recordJSON, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record: %w", err)
	}
	if err := ctx.GetStub().PutState(key, recordJSON); err != nil {
		return nil, fmt.Errorf("failed to write world state: %w", err)
	}

	// A successful commit emits a chaincode event carrying the same non-PII
	// fields (ADR-0020 "should emit ... a recommendation, not a numeric
	// constraint"), so downstream audit projections can react on commit
	// [docs: gateway.md#listening-for-events].
	if err := ctx.GetStub().SetEvent("RecordProfileSection", recordJSON); err != nil {
		return nil, fmt.Errorf("failed to set chaincode event: %w", err)
	}

	return &RecordProfileSectionResult{
		RecordID:  recordID,
		Version:   nextVersion,
		Timestamp: timestamp,
	}, nil
}

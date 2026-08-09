// Package writepaths is REC-4's GENERIC DEMONSTRATION of the five
// write-path hook integrations — NOT the real integration into the actual
// external HRIS application's own write paths.
//
// Per the user's explicit, standing decision (2026-08-07): REC-4 as
// literally specified means inserting a RecordProfileSection call into a
// real, separate production codebase this workspace does not own, fork, or
// rehost (repository-structure.md: "the external HRIS application repo is
// not forked or rehosted here"). That real integration is out of THIS
// workspace's scope. What this package proves instead is that the HOOK
// PATTERN itself — read/write the operational value, compute the digest
// off-chain (REC-1), fetch the salt/employeeKey_i (REC-2), anchor in-band
// in the SAME request (REC-3, ADR-0014, no queue) — works end-to-end for
// all five profile sections, against a generic mock operational store, not
// the real one.
//
// Each of the five functions below simulates a DIFFERENT real-world
// business action per data-model.md §4 (a personal-data change-approval
// workflow, a transfer/mutation approval, an education-history write, a
// family-data change-approval, a payroll/bank-account update) — five
// distinct call sites, not one generic dispatcher, matching how the real
// application actually has five independent write paths, not one.
package writepaths

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	gatewayclient "gatewayclient"
	keystore "keystore"
)

// OperationalStore is a generic stand-in for the EXISTING relational
// database (ADR-0017: no migration, no new store — this mock exists only
// so this package is runnable without a real database dependency).
type OperationalStore interface {
	SaveSection(ctx context.Context, employeeInternalID, profileSection string, valueJSON []byte) (version int, err error)
	// DeleteSection deletes the operational-DB field for one section — the
	// first of REC-6's four crypto-shred steps. Deleting the OTHER four
	// sections for the same employee is the caller's job (REC-6 iterates).
	DeleteSection(ctx context.Context, employeeInternalID, profileSection string) error
}

// AllProfileSections lists the five ratified sections (PRD §4) — REC-6's
// erasure hook iterates this list to delete every section's operational-DB
// field for one employee, not just one.
var AllProfileSections = []string{"PERSONAL", "EMPLOYMENT", "EDUCATION", "ADDITIONAL", "PAYROLL"}

// InMemoryOperationalStore is the mock used by this package's own tests and
// the live demonstration — never a production store.
type InMemoryOperationalStore struct {
	mu       sync.Mutex
	versions map[string]int    // key: employeeInternalID+"/"+profileSection
	values   map[string][]byte // same key — the "field" REC-6 must delete
}

func NewInMemoryOperationalStore() *InMemoryOperationalStore {
	return &InMemoryOperationalStore{versions: make(map[string]int), values: make(map[string][]byte)}
}

func (s *InMemoryOperationalStore) SaveSection(_ context.Context, employeeInternalID, profileSection string, valueJSON []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := employeeInternalID + "/" + profileSection
	s.versions[key]++
	s.values[key] = valueJSON
	return s.versions[key], nil
}

func (s *InMemoryOperationalStore) DeleteSection(_ context.Context, employeeInternalID, profileSection string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := employeeInternalID + "/" + profileSection
	delete(s.values, key)
	return nil
}

// HasValue is a test/demo-only accessor proving DeleteSection actually
// removed the field, not just returned success — not part of the
// OperationalStore interface (a real DB would use its own existence check).
func (s *InMemoryOperationalStore) HasValue(employeeInternalID, profileSection string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.values[employeeInternalID+"/"+profileSection]
	return exists
}

// Hooks bundles everything a write-path hook needs: the (mock) operational
// store, the two off-chain secret stores (REC-2), and the retained Gateway
// client (REC-3). tenantID is fixed per Hooks instance — one HRIS process
// serves one tenant's write paths at a time in this design (channel-per-
// tenant, ADR-0013).
type Hooks struct {
	Store        OperationalStore
	Keys         keystore.EmployeeKeyStore
	Salts        keystore.SaltStore
	DocumentKeys keystore.DocumentKeyStore // KEY_EMPLOYEE (ADR-0019 Domain B'); nil is fine unless a hook call carries a document
	IPFS         DocumentPinner            // REC-5; nil is fine unless a hook call carries a document
	Gateway      LedgerAnchorer
	TenantID     string

	// OnPartialFailure is invoked (if non-nil), exactly once, at the point
	// anchor() is about to return a *PartialFailureError — INT-5's "at
	// minimum surfaced as an error/metric" floor: the returned error is the
	// "error" half, this callback is the "metric" half. Deliberately a
	// plain, caller-injectable Go function rather than a hardcoded metrics
	// dependency — this workspace has no metrics/observability stack wired
	// up yet (that is DEP-5's separate, later scope), and this package
	// picking one specific vendor/SDK here would bake in a policy decision
	// that belongs to whatever process actually embeds these hooks, not to
	// this demonstration. Matches this codebase's existing "interface-only,
	// policy/infra owned elsewhere" pattern (see keystore.go's own doc
	// comment on custody/rotation policy). nil is fine (the common case in
	// tests and the REC-4 demo) — anchor() only ever calls it, never
	// requires it.
	OnPartialFailure func(ctx context.Context, employeeInternalID, profileSection string, version int, err error)
}

// PartialFailureError signals the specific state INT-1's provisional
// "retry-bounded, then degrade-and-flag" policy exists to name: the
// operational-DB write for one profile-section version already committed,
// but its blockchain anchor did not. [ASSUMPTION — this is a provisional,
// clearly-labeled policy call filling ADR-0014's own Consequences section,
// which explicitly left the Fabric-outage fallback/retry policy for the
// write path OPEN as future work (grounding gap G-10 / SEC threat T6b); it
// is not a ratified policy and a real follow-on ADR from `architect` may
// supersede it.]
//
// The invariant that makes this type meaningful: every one of the five
// write-path hooks calls Store.SaveSection and returns early on its error
// BEFORE ever calling anchor() (see UpdatePersonalData et al. above) — so
// by construction, anchor() is only ever reached once the operational-DB
// write has already succeeded. That means ANY error anchor() itself
// produces — a genuine Fabric/network outage, a TLS failure, a
// non-retryable rejection that survived REC-3's own bounded retry
// (gatewayclient.MaxSubmitRetries), or an IPFS pin failure — necessarily
// means "the data change was saved, its anchor was not," never "nothing
// happened." A SaveSection failure, by contrast, is deliberately NOT
// wrapped in this type (nothing committed in that case) — the hooks'
// existing early-return keeps these two failure classes structurally
// apart; this type must never blur that line.
//
// This workspace has no distributed-transaction/dual-write-coordination
// mechanism, and ADR-0017 rules against inventing one — so this type does
// not attempt to roll back the operational-DB write it reports on. It only
// lets a caller distinguish this state from every other failure class
// (e.g. to flag the record for a later re-anchor pass — deciding what that
// pass looks like is out of this item's scope).
type PartialFailureError struct {
	EmployeeInternalID string
	ProfileSection     string
	Version            int
	Err                error
}

func (e *PartialFailureError) Error() string {
	return fmt.Sprintf(
		"writepaths: partial failure: operational-DB write committed (employeeInternalID=%s, profileSection=%s, version=%d) but its blockchain anchor did not: %v",
		e.EmployeeInternalID, e.ProfileSection, e.Version, e.Err,
	)
}

func (e *PartialFailureError) Unwrap() error { return e.Err }

// anchor is the shared "compute + submit" sequence every one of the five
// hooks below calls, in-band, in the SAME request as the operational-DB
// save above it (ADR-0014 — no queue, no standalone anchor-service). Every
// error path below is funneled through doAnchor and wrapped, once, in a
// *PartialFailureError here — see that type's doc comment for why this is
// always safe to do at this specific call site (SaveSection has already
// committed by the time any caller reaches this function).
func (h *Hooks) anchor(ctx context.Context, employeeInternalID, userID, profileSection string, sectionValueJSON []byte, version int, document []byte) ([]byte, error) {
	result, err := h.doAnchor(ctx, employeeInternalID, userID, profileSection, sectionValueJSON, version, document)
	if err != nil {
		pfErr := &PartialFailureError{
			EmployeeInternalID: employeeInternalID,
			ProfileSection:     profileSection,
			Version:            version,
			Err:                err,
		}
		if h.OnPartialFailure != nil {
			h.OnPartialFailure(ctx, employeeInternalID, profileSection, version, pfErr)
		}
		return nil, pfErr
	}
	return result, nil
}

// doAnchor is anchor's actual compute-and-submit body, split out only so
// anchor() has a single, unavoidable wrapping point for every error it can
// produce (see anchor's own doc comment) — this split changes no behavior
// beyond that wrapping.
//
// document is the OPTIONAL supporting document for this write (nil/empty =
// none, the common case). Per REC-5's DoD: when present, it is encrypted
// with the employee's own KEY_EMPLOYEE and pinned on the IPFS private
// cluster BEFORE this function ever builds the RecordProfileSection call —
// only the resulting CID, never document content, becomes part of that
// call's ipfsCIDs argument. A fresh call always produces a fresh CID (see
// ipfsclient's own re-encryption test), so this never mutates a prior
// version's ipfsCIDs in place — it only ever appears on the NEW version
// written by THIS call.
func (h *Hooks) doAnchor(ctx context.Context, employeeInternalID, userID, profileSection string, sectionValueJSON []byte, version int, document []byte) ([]byte, error) {
	employeeKey, err := h.Keys.GetOrCreateEmployeeKey(ctx, employeeInternalID)
	if err != nil {
		return nil, fmt.Errorf("writepaths: fetching employeeKey_i: %w", err)
	}
	employeeID, err := gatewayclient.ComputeEmployeeID(employeeKey)
	if err != nil {
		return nil, fmt.Errorf("writepaths: computing EmployeeID: %w", err)
	}

	salt, err := h.Salts.PutSalt(ctx, employeeID, profileSection, version)
	if err != nil {
		return nil, fmt.Errorf("writepaths: generating salt: %w", err)
	}
	dataHash, err := gatewayclient.ComputeDataHash(salt, sectionValueJSON)
	if err != nil {
		return nil, fmt.Errorf("writepaths: computing DataHash: %w", err)
	}
	updatedBy, err := gatewayclient.ComputeUpdatedBy(employeeKey, userID)
	if err != nil {
		return nil, fmt.Errorf("writepaths: computing UpdatedBy: %w", err)
	}

	ipfsCIDsJSON := "[]"
	if len(document) > 0 {
		documentKey, err := h.DocumentKeys.GetOrCreateDocumentKey(ctx, employeeInternalID)
		if err != nil {
			return nil, fmt.Errorf("writepaths: fetching KEY_EMPLOYEE: %w", err)
		}
		cid, err := h.IPFS.EncryptAndAdd(ctx, documentKey, document)
		if err != nil {
			return nil, fmt.Errorf("writepaths: encrypting+pinning supporting document: %w", err)
		}
		cidsJSON, err := json.Marshal([]string{cid})
		if err != nil {
			return nil, fmt.Errorf("writepaths: marshaling ipfsCIDs: %w", err)
		}
		ipfsCIDsJSON = string(cidsJSON)
	}

	tenantID := h.TenantID
	buildArgs := func(prevHash string) []string {
		return []string{
			tenantID, employeeID, profileSection, dataHash, prevHash, updatedBy,
			ipfsCIDsJSON, gatewayclient.CanonicalizationVersion, gatewayclient.HashAlgo, "",
		}
	}
	return h.Gateway.SubmitRecordProfileSection(ctx, tenantID, employeeID, profileSection, buildArgs)
}

// UpdatePersonalData — the personal-data change-approval workflow
// (data-model.md §4). document is the OPTIONAL supporting document for this
// change (nil/empty = none) — see anchor's own doc comment for REC-5's
// encrypt-before-pin handling.
func (h *Hooks) UpdatePersonalData(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
	version, err := h.Store.SaveSection(ctx, employeeInternalID, "PERSONAL", newValue)
	if err != nil {
		return nil, fmt.Errorf("writepaths: saving PERSONAL to operational store: %w", err)
	}
	return h.anchor(ctx, employeeInternalID, userID, "PERSONAL", newValue, version, document)
}

// ApproveEmploymentTransfer — the transfer/mutation approval action.
func (h *Hooks) ApproveEmploymentTransfer(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
	version, err := h.Store.SaveSection(ctx, employeeInternalID, "EMPLOYMENT", newValue)
	if err != nil {
		return nil, fmt.Errorf("writepaths: saving EMPLOYMENT to operational store: %w", err)
	}
	return h.anchor(ctx, employeeInternalID, userID, "EMPLOYMENT", newValue, version, document)
}

// RecordEducationHistory — the formal/informal education-history write action.
func (h *Hooks) RecordEducationHistory(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
	version, err := h.Store.SaveSection(ctx, employeeInternalID, "EDUCATION", newValue)
	if err != nil {
		return nil, fmt.Errorf("writepaths: saving EDUCATION to operational store: %w", err)
	}
	return h.anchor(ctx, employeeInternalID, userID, "EDUCATION", newValue, version, document)
}

// ApproveFamilyDataChange — the family-data (marital status / dependents)
// change-approval workflow (the ADDITIONAL section).
func (h *Hooks) ApproveFamilyDataChange(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
	version, err := h.Store.SaveSection(ctx, employeeInternalID, "ADDITIONAL", newValue)
	if err != nil {
		return nil, fmt.Errorf("writepaths: saving ADDITIONAL to operational store: %w", err)
	}
	return h.anchor(ctx, employeeInternalID, userID, "ADDITIONAL", newValue, version, document)
}

// UpdatePayrollBankAccount — the payroll/bank-account update action.
func (h *Hooks) UpdatePayrollBankAccount(ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error) {
	version, err := h.Store.SaveSection(ctx, employeeInternalID, "PAYROLL", newValue)
	if err != nil {
		return nil, fmt.Errorf("writepaths: saving PAYROLL to operational store: %w", err)
	}
	return h.anchor(ctx, employeeInternalID, userID, "PAYROLL", newValue, version, document)
}

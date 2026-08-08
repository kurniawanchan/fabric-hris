// Package chaincode implements the EmployeeProfileRecord contract (ADR-0020):
// one Submit function (RecordProfileSection) and three Evaluate functions
// (GetProfileSectionRecord, GetProfileHistory, GetEmployeeProfileSummary)
// over a per-(employeeID, profileSection) head key.
//
// Confidentiality invariant (data-model.md §1, ADR-0011/ADR-0020, restated
// here because it binds every file in this package without exception): no
// function in this contract — Submit or Evaluate — ever accepts a
// profile-section field value, or any other plaintext PII, as an argument, and
// no function ever returns a salt, in any response. Every value this package
// touches is a digest, a pseudonymous HMAC-derived identifier, an enum, a CID,
// or a non-PII metadata tag.
package chaincode

// ProfileSection enumerates the five anchoring units ratified by
// data-model.md §8 / §2.1 (FR-8) — superseding the pre-reconciliation
// EVENT_* surface entirely. No value beyond these five is ever valid.
type ProfileSection string

const (
	SectionPersonal   ProfileSection = "PERSONAL"
	SectionEmployment ProfileSection = "EMPLOYMENT"
	SectionEducation  ProfileSection = "EDUCATION"
	SectionAdditional ProfileSection = "ADDITIONAL"
	SectionPayroll    ProfileSection = "PAYROLL"
)

// allProfileSections fixes both the validation universe and the fan-out order
// GetEmployeeProfileSummary uses (data-model.md §8's listed order).
var allProfileSections = []ProfileSection{
	SectionPersonal,
	SectionEmployment,
	SectionEducation,
	SectionAdditional,
	SectionPayroll,
}

// IsValidProfileSection reports whether v is one of the exactly five ratified
// enum values (data-model.md §2.1/§8, FR-8). Comparison is exact-match,
// case-sensitive — no normalization, since the contract requires "exactly
// these five values, nothing else."
func IsValidProfileSection(v string) bool {
	for _, s := range allProfileSections {
		if string(s) == v {
			return true
		}
	}
	return false
}

// profileKeyObjectType is the fixed key namespace for the world-state
// composite key (data-model.md §5):
//
//	CreateCompositeKey("profile", []string{employeeID, profileSection})
//
// Deliberately NOT prefixed with tenantID/companyID — tenant isolation is a
// channel-membership property (ADR-0013), not a key-shape property. Adding a
// tenant prefix back here would reintroduce the composite-key-prefix model
// this session's earlier work retired; do not do it without routing the
// reversal through architect as a new ADR.
const profileKeyObjectType = "profile"

// EmployeeProfileRecord is the on-chain asset: a per-record digest, two
// HMAC-derived pseudonymous identifiers, and non-PII structural metadata —
// nothing else (data-model.md §2/§2.1). The field set below is EXACTLY the
// twelve fields data-model.md §2 lists; adding any field re-opens the
// STRIDE T9 regression risk (security-architecture.md T9 — a per-field/
// changed-content leak) that this shape was corrected to close. Do not add a
// field without routing it through architect/security-architect first.
//
// Fields are declared in alphabetic order (by Go identifier) so that Go's
// encoding/json — which preserves struct declaration order rather than
// sorting — produces the same canonical field order on every endorsing peer,
// per the contract-api-go determinism convention
// [docs: chaincode4ade.rst#json-determinism].
type EmployeeProfileRecord struct {
	CanonicalizationVersion string   `json:"canonicalizationVersion"`
	DataHash                string   `json:"dataHash"`
	EmployeeID              string   `json:"employeeID"`
	HashAlgo                string   `json:"hashAlgo"`
	IpfsCIDs                []string `json:"ipfsCIDs"`
	PrevHash                string   `json:"prevHash"`
	ProfileSection          string   `json:"profileSection"`
	RecordID                string   `json:"recordID"`
	TenantID                string   `json:"tenantID"`
	Timestamp               string   `json:"timestamp"`
	UpdatedBy               string   `json:"updatedBy"`
	Version                 int      `json:"version"`
}

// RecordProfileSectionResult is RecordProfileSection's return shape
// (api-contracts.md "Response — RecordProfileSectionResult"): an echo of
// ledger-assigned, non-PII metadata only. No section field value and no salt
// is ever echoed here — there is none to echo.
type RecordProfileSectionResult struct {
	RecordID  string `json:"recordID"`
	Timestamp string `json:"timestamp"`
	Version   int    `json:"version"`
}

// ProfileSectionHead is GetProfileSectionRecord's return shape
// (api-contracts.md, data-model.md §2.1) — exactly these four fields, never
// the full asset (FR-35). This is the read-side half of the confidentiality
// invariant: even a read-only query never has a way to leak recordID,
// prevHash, ipfsCIDs, or the version-tag fields through this function; those
// are reachable only via GetProfileHistory for a caller that needs them.
type ProfileSectionHead struct {
	DataHash  string `json:"dataHash"`
	Timestamp string `json:"timestamp"`
	UpdatedBy string `json:"updatedBy"`
	Version   int    `json:"version"`
}

// ProfileVersionEntry is one element of GetProfileHistory's return list
// (api-contracts.md "GetProfileHistory — Response").
type ProfileVersionEntry struct {
	CanonicalizationVersion string   `json:"canonicalizationVersion"`
	DataHash                string   `json:"dataHash"`
	HashAlgo                string   `json:"hashAlgo"`
	IpfsCIDs                []string `json:"ipfsCIDs"`
	PrevHash                string   `json:"prevHash"`
	RecordID                string   `json:"recordID"`
	Timestamp               string   `json:"timestamp"`
	UpdatedBy               string   `json:"updatedBy"`
	Version                 int      `json:"version"`
}

// ProfileSectionSummaryEntry is one element of GetEmployeeProfileSummary's
// return list (api-contracts.md "GetEmployeeProfileSummary — Response").
type ProfileSectionSummaryEntry struct {
	DataHash       string `json:"dataHash"`
	ProfileSection string `json:"profileSection"`
	Timestamp      string `json:"timestamp"`
	UpdatedBy      string `json:"updatedBy"`
	Version        int    `json:"version"`
}

package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	gatewayclient "gatewayclient"
	writepaths "writepaths"
)

// EmployeeKeyResolver is the one keystore.EmployeeKeyStore method this route
// needs -- narrowed the same way ports.go (write-path-integration/writepaths)
// narrows LedgerAnchorer/DocumentPinner, so this route is unit-testable
// against a stub without a real keystore. hooks.Keys (a
// keystore.EmployeeKeyStore) satisfies this automatically.
type EmployeeKeyResolver interface {
	GetOrCreateEmployeeKey(ctx context.Context, employeeInternalID string) ([]byte, error)
}

// LedgerHistoryReader is the one read-only ledger capability this route
// needs. writepaths.Hooks.Gateway is deliberately narrowed to LedgerAnchorer
// and does NOT expose it (writepaths/ports.go: "no Evaluate* reads, ...
// writepaths never reads back from the ledger") -- this route takes its own,
// separate reader (the full *gatewayclient.GatewayClient satisfies this)
// rather than widening writepaths' own port.
type LedgerHistoryReader interface {
	EvaluateGetProfileHistory(ctx context.Context, tenantID, employeeID, profileSection string) ([]byte, error)
}

// HistoryRouteConfig is what RegisterProfileHistoryRoute needs beyond the
// request itself -- one long-lived value per bridge process, exactly like
// RouteConfig's own Dispatch/ProfileSection pairing for the write routes.
type HistoryRouteConfig struct {
	Keys     EmployeeKeyResolver
	Ledger   LedgerHistoryReader
	TenantID string
}

// historyEntry is the wire shape of one returned transaction -- mirrors
// chaincode.ProfileVersionEntry's field set exactly (never re-derives or
// widens it -- no section value, no salt, per this contract's confidentiality
// invariant), tagged with the ProfileSection it belongs to since a caller
// requesting every section needs to know which section each entry is from
// (GetProfileHistory itself does not return that field -- it is implied by
// the query argument, not the ledger record).
type historyEntry struct {
	ProfileSection          string   `json:"profileSection"`
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

// chaincodeHistoryEntry decodes exactly one element of
// GetProfileHistory's raw JSON response
// (chaincode.ProfileVersionEntry's own field set) -- kept as a private,
// unexported type here rather than importing the chaincode package itself
// (this bridge talks to the chaincode only via the Gateway wire protocol,
// the same boundary writepaths/gatewayclient already established; it has no
// Go-level dependency on the chaincode module anywhere else either).
type chaincodeHistoryEntry struct {
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

// historyResponseBody is this route's own response envelope -- deliberately
// separate from responseBody (respond.go): that shape's recordID/committed
// vocabulary describes a WRITE outcome, and reusing it here would force a
// history read to lie about which of the two it is. Status is either "ok"
// (query succeeded, History may legitimately be empty -- "never anchored
// yet" is not an error) or "rejected"/"error", matching this bridge's
// existing AD-3 vocabulary split between a caller mistake and a bridge/
// dependency fault.
type historyResponseBody struct {
	Status  string         `json:"status"`
	Detail  string         `json:"detail,omitempty"`
	History []historyEntry `json:"history,omitempty"`
}

func isKnownProfileSection(section string) bool {
	for _, s := range writepaths.AllProfileSections {
		if s == section {
			return true
		}
	}
	return false
}

// RegisterProfileHistoryRoute wires the one read route this bridge exposes:
// GET /v1/profile-sections/history?employeeInternalID=...&profileSection=...
// (profileSection optional -- omitted means "every section," fanned out and
// merged). This closes grounding-gaps.md G-34's caller-facing half in a
// real, production way: employeeInternalID never leaves the caller's own
// request, and the on-chain pseudonym it maps to is derived HERE, from the
// same keystore.EmployeeKeyStore the write paths already use
// (GetOrCreateEmployeeKey is idempotent -- an employee's first read and
// first write derive the identical pseudonym), never returned in the
// response itself.
//
// path must already be a method-qualified pattern (e.g.
// "GET /v1/profile-sections/history"), same convention as
// RegisterProfileSectionRoute.
func RegisterProfileHistoryRoute(mux *http.ServeMux, path string, cfg HistoryRouteConfig, authCfg AuthConfig) {
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if err := authenticate(r, authCfg); err != nil {
			respondHistory(w, "error", err.Error(), nil)
			return
		}

		employeeInternalID := strings.TrimSpace(r.URL.Query().Get("employeeInternalID"))
		if employeeInternalID == "" {
			respondHistory(w, "rejected", "integrationbridge: employeeInternalID is required", nil)
			return
		}

		sections := writepaths.AllProfileSections
		if requested := strings.TrimSpace(r.URL.Query().Get("profileSection")); requested != "" {
			if !isKnownProfileSection(requested) {
				respondHistory(w, "rejected", fmt.Sprintf("integrationbridge: profileSection %q is not one of the five ratified values", requested), nil)
				return
			}
			sections = []string{requested}
		}

		ctx := r.Context()
		employeeKey, err := cfg.Keys.GetOrCreateEmployeeKey(ctx, employeeInternalID)
		if err != nil {
			respondHistory(w, "error", fmt.Sprintf("integrationbridge: resolving employee key: %v", err), nil)
			return
		}
		employeeID, err := gatewayclient.ComputeEmployeeID(employeeKey)
		if err != nil {
			respondHistory(w, "error", fmt.Sprintf("integrationbridge: computing EmployeeID: %v", err), nil)
			return
		}

		all := make([]historyEntry, 0)
		for _, section := range sections {
			raw, err := cfg.Ledger.EvaluateGetProfileHistory(ctx, cfg.TenantID, employeeID, section)
			if err != nil {
				respondHistory(w, "error", fmt.Sprintf("integrationbridge: querying ledger history: %v", err), nil)
				return
			}

			var chunk []chaincodeHistoryEntry
			if err := json.Unmarshal(raw, &chunk); err != nil {
				respondHistory(w, "error", fmt.Sprintf("integrationbridge: decoding ledger history: %v", err), nil)
				return
			}
			for _, e := range chunk {
				all = append(all, historyEntry{
					ProfileSection:          section,
					CanonicalizationVersion: e.CanonicalizationVersion,
					DataHash:                e.DataHash,
					HashAlgo:                e.HashAlgo,
					IpfsCIDs:                e.IpfsCIDs,
					PrevHash:                e.PrevHash,
					RecordID:                e.RecordID,
					Timestamp:               e.Timestamp,
					UpdatedBy:               e.UpdatedBy,
					Version:                 e.Version,
				})
			}
		}

		// Most-recent-first: a transaction feed reads naturally newest-on-top.
		// Timestamp is RFC3339 (chaincode's own txTimestamp formatting), so a
		// plain string comparison orders it correctly without parsing.
		sort.SliceStable(all, func(i, j int) bool { return all[i].Timestamp > all[j].Timestamp })

		respondHistory(w, "ok", "", all)
	})
}

// respondHistory writes this route's response envelope. Unlike respond.go's
// respond (which derives its HTTP status from classify()'s AD-3 vocabulary),
// this route has only three possible outcomes -- success, a caller mistake,
// or a bridge/dependency fault -- so the mapping is inlined rather than
// routed through classify(), which knows nothing about a read outcome.
func respondHistory(w http.ResponseWriter, status, detail string, history []historyEntry) {
	body := historyResponseBody{Status: status, Detail: detail, History: history}

	code := http.StatusOK
	switch status {
	case "rejected":
		code = http.StatusBadRequest
	case "error":
		code = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

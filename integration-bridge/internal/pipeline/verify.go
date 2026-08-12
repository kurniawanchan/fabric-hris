package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	gatewayclient "gatewayclient"
	writepaths "writepaths"
)

// SaltReader is the one keystore.SaltStore method this route needs — same
// narrowing technique as EmployeeKeyResolver (history.go) and writepaths'
// own ports.go: unit-testable against a stub, no dependency on a real
// keystore. hooks.Salts (a keystore.SaltStore) satisfies this automatically.
type SaltReader interface {
	GetSalt(ctx context.Context, employeeID, profileSection string, version int) ([]byte, error)
}

// VerifyRouteConfig is what RegisterProfileVerifyRoute needs beyond the
// request itself.
type VerifyRouteConfig struct {
	Keys     writepaths.EmployeeKeyGetter
	Salts    SaltReader
	Ledger   LedgerHistoryReader
	TenantID string
}

// verifyRequestEnvelope is the wire shape POST /v1/profile-sections/verify
// accepts. CurrentValue is a pointer so "field omitted" and "field present
// but null" are both distinguishable from "field present with a real JSON
// object" — either of the first two means "the caller asserts this record
// is deleted" (Story tf-3.1's resolution to the chaincode having no
// on-chain operationType, AD-5).
type verifyRequestEnvelope struct {
	EmployeeInternalID string          `json:"employeeInternalID"`
	ProfileSection     string          `json:"profileSection"`
	RecordIdentity     string          `json:"recordIdentity"`
	CurrentValue       json.RawMessage `json:"currentValue"`
}

type verifyResponseBody struct {
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// RegisterProfileVerifyRoute wires the on-demand integrity check
// (FR-10, Story tf-3.1): recompute the current value's digest and compare
// it to the latest on-chain anchor at the same pseudonym AD-3/AD-8 the
// write path uses. path must already be a method-qualified pattern (e.g.
// "POST /v1/profile-sections/verify").
func RegisterProfileVerifyRoute(mux *http.ServeMux, path string, cfg VerifyRouteConfig, authCfg AuthConfig) {
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if err := authenticate(r, authCfg); err != nil {
			respondVerify(w, "error", err.Error())
			return
		}

		var env verifyRequestEnvelope
		if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
			respondVerify(w, "rejected", fmt.Sprintf("integrationbridge: invalid request body: %v", err))
			return
		}

		employeeInternalID := strings.TrimSpace(env.EmployeeInternalID)
		if employeeInternalID == "" {
			respondVerify(w, "rejected", "integrationbridge: employeeInternalID is required")
			return
		}
		profileSection := strings.TrimSpace(env.ProfileSection)
		if !isKnownProfileSection(profileSection) {
			respondVerify(w, "rejected", fmt.Sprintf("integrationbridge: profileSection %q is not one of the five ratified values", profileSection))
			return
		}

		ctx := r.Context()
		employeeID, err := writepaths.ResolveEmployeeID(ctx, cfg.Keys, employeeInternalID, env.RecordIdentity)
		if err != nil {
			respondVerify(w, "error", fmt.Sprintf("integrationbridge: resolving employee key: %v", err))
			return
		}

		raw, err := cfg.Ledger.EvaluateGetProfileHistory(ctx, cfg.TenantID, employeeID, profileSection)
		if err != nil {
			respondVerify(w, "error", fmt.Sprintf("integrationbridge: querying ledger history: %v", err))
			return
		}
		var chunk []chaincodeHistoryEntry
		if err := json.Unmarshal(raw, &chunk); err != nil {
			respondVerify(w, "error", fmt.Sprintf("integrationbridge: decoding ledger history: %v", err))
			return
		}
		if len(chunk) == 0 {
			respondVerify(w, "no_anchor_found", "integrationbridge: no anchor exists yet for this employee/domain/recordIdentity")
			return
		}

		latest := chunk[0]
		for _, e := range chunk[1:] {
			if e.Version > latest.Version {
				latest = e
			}
		}

		// currentValue omitted or explicit JSON null both decode to a nil
		// json.RawMessage-backed pointer check below -- either means the
		// caller is asserting deletion (AC #3), never a hash-compare
		// attempt against an absent value.
		if len(env.CurrentValue) == 0 || string(env.CurrentValue) == "null" {
			respondVerify(w, "deletion_verified", "")
			return
		}

		salt, err := cfg.Salts.GetSalt(ctx, employeeID, profileSection, latest.Version)
		if err != nil {
			respondVerify(w, "error", fmt.Sprintf("integrationbridge: fetching salt: %v", err))
			return
		}
		dataHash, err := gatewayclient.ComputeDataHash(salt, []byte(env.CurrentValue))
		if err != nil {
			respondVerify(w, "error", fmt.Sprintf("integrationbridge: computing DataHash: %v", err))
			return
		}

		if dataHash != latest.DataHash {
			respondVerify(w, "compromised", "integrationbridge: current value's digest does not match the latest on-chain anchor")
			return
		}
		respondVerify(w, "verified", "")
	})
}

func respondVerify(w http.ResponseWriter, status, detail string) {
	body := verifyResponseBody{Status: status, Detail: detail}

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

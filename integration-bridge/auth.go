package main

import (
	"crypto/subtle"
	"errors"
	"net/http"
)

// authConfig holds the credentials [auth] checks every request against.
// One bridge deployment serves exactly one tenant (AD-2), so CompanyID is a
// single configured value, not a lookup.
type authConfig struct {
	APIKey    string
	CompanyID string
}

// authenticate checks X-Api-Key and X-Company-ID against cfg -- both
// headers are required, not X-Api-Key alone, matching the real integration
// precedent ADR-0022 cites for this shape of integration (see that ADR's
// own Context section for specifics -- not restated here, per this repo's
// confidentiality register; also see ARCHITECTURE-SPINE.md's Consistency
// Conventions). Comparisons are constant-time: this is a credential check,
// and a timing side-channel is a real category of vulnerability regardless
// of whether the story's ACs mention it explicitly.
//
// Any failure is wrapped in *authError -- classify() relies on that type to
// route it to status "error". Both branches below return the IDENTICAL
// message deliberately: a message naming which specific header failed would
// let a caller distinguish "wrong API key" from "right key, wrong company
// ID," creating a credential-validation oracle that undermines the
// constant-time comparison's whole purpose (code review finding).
var errMissingOrIncorrectCredentials = errors.New("integrationbridge: missing or incorrect credentials")

func authenticate(r *http.Request, cfg authConfig) error {
	apiKey := r.Header.Get("X-Api-Key")
	if apiKey == "" || subtle.ConstantTimeCompare([]byte(apiKey), []byte(cfg.APIKey)) != 1 {
		return &authError{errMissingOrIncorrectCredentials}
	}

	companyID := r.Header.Get("X-Company-ID")
	if companyID == "" || subtle.ConstantTimeCompare([]byte(companyID), []byte(cfg.CompanyID)) != 1 {
		return &authError{errMissingOrIncorrectCredentials}
	}

	return nil
}

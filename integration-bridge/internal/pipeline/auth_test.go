package pipeline

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newAuthRequest(t *testing.T, apiKey, companyID string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/profile-sections/PERSONAL", nil)
	if apiKey != "" {
		req.Header.Set("X-Api-Key", apiKey)
	}
	if companyID != "" {
		req.Header.Set("X-Company-ID", companyID)
	}
	return req
}

func TestAuthenticate_CorrectCredentials_Succeeds(t *testing.T) {
	cfg := AuthConfig{APIKey: "secret-key", CompanyID: "tenant01"}
	req := newAuthRequest(t, "secret-key", "tenant01")

	if err := authenticate(req, cfg); err != nil {
		t.Errorf("authenticate() unexpected error: %v", err)
	}
}

func TestAuthenticate_MissingAPIKey_Rejected(t *testing.T) {
	cfg := AuthConfig{APIKey: "secret-key", CompanyID: "tenant01"}
	req := newAuthRequest(t, "", "tenant01")

	assertAuthError(t, authenticate(req, cfg))
}

func TestAuthenticate_WrongAPIKey_Rejected(t *testing.T) {
	cfg := AuthConfig{APIKey: "secret-key", CompanyID: "tenant01"}
	req := newAuthRequest(t, "wrong-key", "tenant01")

	assertAuthError(t, authenticate(req, cfg))
}

func TestAuthenticate_MissingCompanyID_Rejected(t *testing.T) {
	cfg := AuthConfig{APIKey: "secret-key", CompanyID: "tenant01"}
	req := newAuthRequest(t, "secret-key", "")

	assertAuthError(t, authenticate(req, cfg))
}

func TestAuthenticate_WrongCompanyID_Rejected(t *testing.T) {
	cfg := AuthConfig{APIKey: "secret-key", CompanyID: "tenant01"}
	req := newAuthRequest(t, "secret-key", "tenant02")

	assertAuthError(t, authenticate(req, cfg))
}

// TestAuthenticate_APIKeyAndCompanyIDFailures_ProduceIdenticalMessage is the
// fix for a code review finding: distinguishable messages per failed header
// created a credential-validation oracle, undermining the constant-time
// comparison's purpose (a caller could tell "wrong API key" apart from
// "right key, wrong company ID"). Both failure branches must produce the
// exact same message.
func TestAuthenticate_APIKeyAndCompanyIDFailures_ProduceIdenticalMessage(t *testing.T) {
	cfg := AuthConfig{APIKey: "secret-key", CompanyID: "tenant01"}

	wrongAPIKeyErr := authenticate(newAuthRequest(t, "wrong-key", "tenant01"), cfg)
	wrongCompanyIDErr := authenticate(newAuthRequest(t, "secret-key", "tenant02"), cfg)

	if wrongAPIKeyErr == nil || wrongCompanyIDErr == nil {
		t.Fatal("expected both calls to return an error")
	}
	if wrongAPIKeyErr.Error() != wrongCompanyIDErr.Error() {
		t.Errorf("messages differ, creating a credential-validation oracle: %q vs %q", wrongAPIKeyErr.Error(), wrongCompanyIDErr.Error())
	}
}

func assertAuthError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("authenticate() got nil error, want an *authError")
	}
	var ae *authError
	if !errors.As(err, &ae) {
		t.Errorf("authenticate() error = %v, want it to be an *authError", err)
	}
}

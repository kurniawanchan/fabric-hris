package main

import (
	"fmt"
	"strings"
	"time"
)

// Config holds every value NewGatewayClient's 10 positional string parameters
// need, plus TenantID (AD-2: one bridge deployment serves exactly one
// tenant). No config-loading convention exists elsewhere in this repo's Go
// code to extend — env vars are the idiomatic default here, matching the
// spirit of the one real precedent ADR-0022 cites for this shape of
// integration (see that ADR's own Context section for specifics — not
// restated here, per this repo's confidentiality register).
type Config struct {
	PeerEndpoint      string
	TLSServerName     string
	TLSCACertPath     string
	ClientTLSCertPath string
	ClientTLSKeyPath  string
	MSPID             string
	SignCertPath      string
	SignKeyPath       string
	ChannelName       string
	ChaincodeName     string
	TenantID          string
	IPFSPrimaryAPI    string
	IPFSReplicaAPI    string
	APIKey            string
	CompanyID         string
}

// LoadConfig reads every required value via getenv (injected so tests never
// touch the real process environment) and fails fast, naming every unset
// variable at once — never silently defaulting. A whitespace-only value is
// treated as missing, not as a real (and later confusingly-invalid) value.
func LoadConfig(getenv func(string) string) (Config, error) {
	var missing []string
	get := func(key string) string {
		v := getenv(key)
		if strings.TrimSpace(v) == "" {
			missing = append(missing, key)
		}
		return v
	}

	cfg := Config{
		PeerEndpoint:      get("BRIDGE_PEER_ENDPOINT"),
		TLSServerName:     get("BRIDGE_TLS_SERVER_NAME"),
		TLSCACertPath:     get("BRIDGE_TLS_CA_CERT_PATH"),
		ClientTLSCertPath: get("BRIDGE_CLIENT_TLS_CERT_PATH"),
		ClientTLSKeyPath:  get("BRIDGE_CLIENT_TLS_KEY_PATH"),
		MSPID:             get("BRIDGE_MSP_ID"),
		SignCertPath:      get("BRIDGE_SIGN_CERT_PATH"),
		SignKeyPath:       get("BRIDGE_SIGN_KEY_PATH"),
		ChannelName:       get("BRIDGE_CHANNEL_NAME"),
		ChaincodeName:     get("BRIDGE_CHAINCODE_NAME"),
		TenantID:          get("BRIDGE_TENANT_ID"),
		IPFSPrimaryAPI:    get("BRIDGE_IPFS_PRIMARY_API"),
		IPFSReplicaAPI:    get("BRIDGE_IPFS_REPLICA_API"),
		APIKey:            get("BRIDGE_API_KEY"),
		CompanyID:         get("BRIDGE_COMPANY_ID"),
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("config: required environment variable(s) not set: %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}

// ResolveHTTPAddr reads BRIDGE_HTTP_ADDR via the same injected-getenv seam
// LoadConfig uses (so it's unit-testable without touching the real process
// environment), defaulting to ":8080" when unset or whitespace-only. This
// is genuinely optional — unlike LoadConfig's required set — so it is kept
// separate rather than folded into Config, whose contract is "every field
// fails fast if missing."
func ResolveHTTPAddr(getenv func(string) string) string {
	if addr := strings.TrimSpace(getenv("BRIDGE_HTTP_ADDR")); addr != "" {
		return addr
	}
	return ":8080"
}

// ResolveDispatchTimeout reads BRIDGE_DISPATCH_TIMEOUT (a time.ParseDuration
// string, e.g. "30s") via the same injected-getenv seam as ResolveHTTPAddr,
// defaulting to 30s when unset, unparseable, or non-positive. This default
// is a placeholder, not a ratified number -- ARCHITECTURE-SPINE.md's own
// Deferred item leaves the real [dispatch] budget open pending real Caliper
// data; it is configurable here so a real deployment can override it
// without a code change once that number exists. A non-positive value is
// syntactically valid to time.ParseDuration (e.g. "-5s") but would produce
// an already-expired context.WithTimeout deadline, silently misclassifying
// every request as partial_failure regardless of outcome -- treated the
// same as an invalid value rather than accepted (code review finding).
func ResolveDispatchTimeout(getenv func(string) string) time.Duration {
	const placeholderDefault = 30 * time.Second
	v := strings.TrimSpace(getenv("BRIDGE_DISPATCH_TIMEOUT"))
	if v == "" {
		return placeholderDefault
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return placeholderDefault
	}
	return d
}

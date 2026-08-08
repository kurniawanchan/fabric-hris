package main

import (
	"strings"
	"testing"
	"time"
)

func fakeEnvAllSet() map[string]string {
	return map[string]string{
		"BRIDGE_PEER_ENDPOINT":        "localhost:7051",
		"BRIDGE_TLS_SERVER_NAME":      "peer0.org1",
		"BRIDGE_TLS_CA_CERT_PATH":     "/tls/ca.pem",
		"BRIDGE_CLIENT_TLS_CERT_PATH": "/tls/client.crt",
		"BRIDGE_CLIENT_TLS_KEY_PATH":  "/tls/client.key",
		"BRIDGE_MSP_ID":               "Org1MSP",
		"BRIDGE_SIGN_CERT_PATH":       "/msp/signcerts/Admin.pem",
		"BRIDGE_SIGN_KEY_PATH":        "/msp/keystore/priv_sk",
		"BRIDGE_CHANNEL_NAME":         "tenant-tenant01",
		"BRIDGE_CHAINCODE_NAME":       "employeeprofilerecord",
		"BRIDGE_TENANT_ID":            "tenant01",
		"BRIDGE_IPFS_PRIMARY_API":     "127.0.0.1:5001",
		"BRIDGE_IPFS_REPLICA_API":     "127.0.0.1:5002",
		"BRIDGE_API_KEY":              "secret-key",
		"BRIDGE_COMPANY_ID":           "tenant01",
	}
}

func TestLoadConfig_AllVarsSet_Succeeds(t *testing.T) {
	env := fakeEnvAllSet()
	getenv := func(key string) string { return env[key] }

	cfg, err := LoadConfig(getenv)
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}
	if cfg.PeerEndpoint != "localhost:7051" {
		t.Errorf("PeerEndpoint = %q, want %q", cfg.PeerEndpoint, "localhost:7051")
	}
	if cfg.ChannelName != "tenant-tenant01" {
		t.Errorf("ChannelName = %q, want %q", cfg.ChannelName, "tenant-tenant01")
	}
	if cfg.TenantID != "tenant01" {
		t.Errorf("TenantID = %q, want %q", cfg.TenantID, "tenant01")
	}
	if cfg.IPFSPrimaryAPI != "127.0.0.1:5001" {
		t.Errorf("IPFSPrimaryAPI = %q, want %q", cfg.IPFSPrimaryAPI, "127.0.0.1:5001")
	}
	if cfg.IPFSReplicaAPI != "127.0.0.1:5002" {
		t.Errorf("IPFSReplicaAPI = %q, want %q", cfg.IPFSReplicaAPI, "127.0.0.1:5002")
	}
	if cfg.APIKey != "secret-key" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "secret-key")
	}
	if cfg.CompanyID != "tenant01" {
		t.Errorf("CompanyID = %q, want %q", cfg.CompanyID, "tenant01")
	}
}

func TestLoadConfig_AllVarsUnset_ListsEveryMissingVar(t *testing.T) {
	getenv := func(string) string { return "" }

	_, err := LoadConfig(getenv)
	if err == nil {
		t.Fatal("LoadConfig() with no env vars set: got nil error, want an error naming every missing var")
	}

	for _, want := range []string{
		"BRIDGE_PEER_ENDPOINT",
		"BRIDGE_TLS_SERVER_NAME",
		"BRIDGE_TLS_CA_CERT_PATH",
		"BRIDGE_CLIENT_TLS_CERT_PATH",
		"BRIDGE_CLIENT_TLS_KEY_PATH",
		"BRIDGE_MSP_ID",
		"BRIDGE_SIGN_CERT_PATH",
		"BRIDGE_SIGN_KEY_PATH",
		"BRIDGE_CHANNEL_NAME",
		"BRIDGE_CHAINCODE_NAME",
		"BRIDGE_TENANT_ID",
		"BRIDGE_IPFS_PRIMARY_API",
		"BRIDGE_IPFS_REPLICA_API",
		"BRIDGE_API_KEY",
		"BRIDGE_COMPANY_ID",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("LoadConfig() error %q does not mention missing var %q", err.Error(), want)
		}
	}
}

func TestLoadConfig_OneVarMissing_DoesNotSilentlyDefault(t *testing.T) {
	env := fakeEnvAllSet()
	delete(env, "BRIDGE_CHANNEL_NAME")
	getenv := func(key string) string { return env[key] }

	_, err := LoadConfig(getenv)
	if err == nil {
		t.Fatal("LoadConfig() with BRIDGE_CHANNEL_NAME unset: got nil error, want an error")
	}
	if !strings.Contains(err.Error(), "BRIDGE_CHANNEL_NAME") {
		t.Errorf("LoadConfig() error %q does not mention BRIDGE_CHANNEL_NAME", err.Error())
	}
}

func TestLoadConfig_WhitespaceOnlyValue_TreatedAsMissing(t *testing.T) {
	env := fakeEnvAllSet()
	env["BRIDGE_CHANNEL_NAME"] = "   "
	getenv := func(key string) string { return env[key] }

	_, err := LoadConfig(getenv)
	if err == nil {
		t.Fatal("LoadConfig() with BRIDGE_CHANNEL_NAME set to whitespace-only: got nil error, want an error")
	}
	if !strings.Contains(err.Error(), "BRIDGE_CHANNEL_NAME") {
		t.Errorf("LoadConfig() error %q does not mention BRIDGE_CHANNEL_NAME", err.Error())
	}
}

func TestResolveHTTPAddr_Unset_DefaultsToPort8080(t *testing.T) {
	got := ResolveHTTPAddr(func(string) string { return "" })
	if got != ":8080" {
		t.Errorf("ResolveHTTPAddr() = %q, want %q", got, ":8080")
	}
}

func TestResolveHTTPAddr_Set_UsesConfiguredValue(t *testing.T) {
	env := map[string]string{"BRIDGE_HTTP_ADDR": "127.0.0.1:9090"}
	got := ResolveHTTPAddr(func(key string) string { return env[key] })
	if got != "127.0.0.1:9090" {
		t.Errorf("ResolveHTTPAddr() = %q, want %q", got, "127.0.0.1:9090")
	}
}

func TestResolveHTTPAddr_WhitespaceOnlyValue_TreatedAsUnset(t *testing.T) {
	env := map[string]string{"BRIDGE_HTTP_ADDR": "   "}
	got := ResolveHTTPAddr(func(key string) string { return env[key] })
	if got != ":8080" {
		t.Errorf("ResolveHTTPAddr() = %q, want default %q for whitespace-only value", got, ":8080")
	}
}

func TestResolveDispatchTimeout_Unset_DefaultsToPlaceholder(t *testing.T) {
	got := ResolveDispatchTimeout(func(string) string { return "" })
	if got != 30*time.Second {
		t.Errorf("ResolveDispatchTimeout() = %v, want the documented placeholder default %v", got, 30*time.Second)
	}
}

func TestResolveDispatchTimeout_Set_UsesConfiguredValue(t *testing.T) {
	env := map[string]string{"BRIDGE_DISPATCH_TIMEOUT": "5s"}
	got := ResolveDispatchTimeout(func(key string) string { return env[key] })
	if got != 5*time.Second {
		t.Errorf("ResolveDispatchTimeout() = %v, want %v", got, 5*time.Second)
	}
}

func TestResolveDispatchTimeout_InvalidDuration_TreatedAsUnset(t *testing.T) {
	env := map[string]string{"BRIDGE_DISPATCH_TIMEOUT": "not-a-duration"}
	got := ResolveDispatchTimeout(func(key string) string { return env[key] })
	if got != 30*time.Second {
		t.Errorf("ResolveDispatchTimeout() = %v, want default %v for an invalid value", got, 30*time.Second)
	}
}

// TestResolveDispatchTimeout_NegativeDuration_TreatedAsUnset is the fix for
// a code review finding: "-5s" parses successfully (it's syntactically
// valid) but would produce an already-expired context.WithTimeout, silently
// misclassifying every request as partial_failure regardless of outcome.
func TestResolveDispatchTimeout_NegativeDuration_TreatedAsUnset(t *testing.T) {
	env := map[string]string{"BRIDGE_DISPATCH_TIMEOUT": "-5s"}
	got := ResolveDispatchTimeout(func(key string) string { return env[key] })
	if got != 30*time.Second {
		t.Errorf("ResolveDispatchTimeout() = %v, want default %v for a negative value", got, 30*time.Second)
	}
}

func TestResolveDispatchTimeout_ZeroDuration_TreatedAsUnset(t *testing.T) {
	env := map[string]string{"BRIDGE_DISPATCH_TIMEOUT": "0s"}
	got := ResolveDispatchTimeout(func(key string) string { return env[key] })
	if got != 30*time.Second {
		t.Errorf("ResolveDispatchTimeout() = %v, want default %v for a zero value", got, 30*time.Second)
	}
}

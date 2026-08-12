package main

import (
	"errors"
	"testing"

	gatewayclient "gatewayclient"
)

func TestBuildHooks_ConstructsGatewayClientExactlyOnce(t *testing.T) {
	cfg := Config{
		PeerEndpoint:             "localhost:7051",
		TLSServerName:            "peer0.org1",
		TLSCACertPath:            "/tls/ca.pem",
		ClientTLSCertPath:        "/tls/client.crt",
		ClientTLSKeyPath:         "/tls/client.key",
		MSPID:                    "Org1MSP",
		SignCertPath:             "/msp/signcerts/Admin.pem",
		SignKeyPath:              "/msp/keystore/priv_sk",
		ChannelName:              "tenant-tenant01",
		ChaincodeName:            "employeeprofilerecord",
		TenantID:                 "tenant01",
		IPFSPrimaryAPI:           "127.0.0.1:5001",
		IPFSReplicaAPI:           "127.0.0.1:5002",
		KeystoreDir:              t.TempDir(),
		KeystoreEncryptionKeyHex: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}

	callCount := 0
	var gotArgs [10]string
	stubNewGW := func(peerEndpoint, tlsServerNameOverride, tlsCACertPEMPath, clientTLSCertPEMPath, clientTLSKeyPEMPath, mspID, certPEMPath, keyPEMPath, channelName, chaincodeName string) (*gatewayclient.GatewayClient, error) {
		callCount++
		gotArgs = [10]string{peerEndpoint, tlsServerNameOverride, tlsCACertPEMPath, clientTLSCertPEMPath, clientTLSKeyPEMPath, mspID, certPEMPath, keyPEMPath, channelName, chaincodeName}
		return nil, nil // GatewayClient's own dial behavior is not under test here
	}

	hooks, gw, err := buildHooks(cfg, stubNewGW)
	if err != nil {
		t.Fatalf("buildHooks() unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("stub GatewayClient constructor called %d times, want exactly 1", callCount)
	}
	if gw != nil {
		t.Errorf("returned GatewayClient = %v, want nil (matches stub's nil return)", gw)
	}

	want := [10]string{cfg.PeerEndpoint, cfg.TLSServerName, cfg.TLSCACertPath, cfg.ClientTLSCertPath, cfg.ClientTLSKeyPath, cfg.MSPID, cfg.SignCertPath, cfg.SignKeyPath, cfg.ChannelName, cfg.ChaincodeName}
	if gotArgs != want {
		t.Errorf("constructor called with args %v, want %v (order must match NewGatewayClient's real positional signature)", gotArgs, want)
	}

	if hooks.TenantID != cfg.TenantID {
		t.Errorf("Hooks.TenantID = %q, want %q", hooks.TenantID, cfg.TenantID)
	}
	if hooks.Store == nil || hooks.Keys == nil || hooks.Salts == nil || hooks.DocumentKeys == nil {
		t.Error("Hooks must have non-nil Store/Keys/Salts/DocumentKeys (Story 1.1 wires the in-memory placeholders)")
	}
	if hooks.IPFS == nil {
		t.Error("Hooks.IPFS must be non-nil as of Story 1.2 -- doAnchor calls h.IPFS.EncryptAndAdd unconditionally whenever a document is present, with no nil-check; leaving this nil is a guaranteed nil-pointer panic on the first document-carrying request")
	}
}

func TestBuildHooks_ConstructorError_PropagatesAndReturnsNoHooks(t *testing.T) {
	cfg := Config{}
	wantErr := errors.New("dial failed")
	stubNewGW := func(string, string, string, string, string, string, string, string, string, string) (*gatewayclient.GatewayClient, error) {
		return nil, wantErr
	}

	hooks, gw, err := buildHooks(cfg, stubNewGW)
	if err == nil {
		t.Fatal("buildHooks() with a failing constructor: got nil error, want an error")
	}
	if hooks != nil {
		t.Errorf("buildHooks() returned non-nil Hooks on error: %v", hooks)
	}
	if gw != nil {
		t.Errorf("buildHooks() returned non-nil GatewayClient on error: %v", gw)
	}
}

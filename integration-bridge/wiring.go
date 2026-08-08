package main

import (
	"fmt"

	gatewayclient "gatewayclient"
	ipfsclient "ipfsclient"
	keystore "keystore"
	writepaths "writepaths"
)

// newGatewayClientFunc abstracts gatewayclient.NewGatewayClient's exact
// positional signature so buildHooks is unit-testable without dialing a real
// Fabric network — production code passes gatewayclient.NewGatewayClient
// itself; tests pass a stub. Mirrors the same dependency-inversion technique
// used for graceful shutdown (see closer in shutdown.go).
type newGatewayClientFunc func(peerEndpoint, tlsServerNameOverride, tlsCACertPEMPath, clientTLSCertPEMPath, clientTLSKeyPEMPath, mspID, certPEMPath, keyPEMPath, channelName, chaincodeName string) (*gatewayclient.GatewayClient, error)

// buildHooks constructs every writepaths.Hooks dependency exactly once and
// returns the wired Hooks plus the constructed GatewayClient (main needs the
// latter directly to call Close() on shutdown). Called exactly once, from
// main(), before the HTTP listener starts (AD-2).
//
// PLACEHOLDER STORES: Store/Keys/Salts/DocumentKeys are wired to the
// InMemory* implementations — the ONLY implementations of these interfaces
// that exist anywhere in this codebase today. Their own doc comments
// (writepaths.InMemoryOperationalStore, keystore.InMemory{Salt,DocumentKey,
// EmployeeKey}Store) call them test mocks with no persistence. Every process
// restart silently loses every salt and employeeKey_i ever created. This is
// correct and sufficient for Story 1.1's own ACs (compile, one-time
// construction, shutdown — nothing about surviving a restart), but is NOT
// production-ready. See the story's Open Questions before any real
// deployment: a persistent store is unbuilt work, not yet anyone's story.
//
// IPFS is a real *ipfsclient.Client as of Story 1.2 (Story 1.1 left it nil,
// correctly, since no route existed yet to carry a document). doAnchor
// (write-path-integration/writepaths/writepaths.go) calls
// h.IPFS.EncryptAndAdd(...) unconditionally whenever a document is present,
// with no nil-check on h.IPFS itself -- leaving this nil once a route can
// accept a document is a guaranteed nil-pointer panic, not a theoretical risk.
func buildHooks(cfg Config, newGW newGatewayClientFunc) (*writepaths.Hooks, *gatewayclient.GatewayClient, error) {
	gw, err := newGW(
		cfg.PeerEndpoint, cfg.TLSServerName, cfg.TLSCACertPath,
		cfg.ClientTLSCertPath, cfg.ClientTLSKeyPath, cfg.MSPID,
		cfg.SignCertPath, cfg.SignKeyPath, cfg.ChannelName, cfg.ChaincodeName,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("integrationbridge: constructing GatewayClient: %w", err)
	}

	hooks := &writepaths.Hooks{
		Store:        writepaths.NewInMemoryOperationalStore(),
		Keys:         keystore.NewInMemoryEmployeeKeyStore(),
		Salts:        keystore.NewInMemorySaltStore(),
		DocumentKeys: keystore.NewInMemoryDocumentKeyStore(),
		IPFS:         ipfsclient.NewClient(cfg.IPFSPrimaryAPI, cfg.IPFSReplicaAPI),
		Gateway:      gw,
		TenantID:     cfg.TenantID,
	}
	return hooks, gw, nil
}

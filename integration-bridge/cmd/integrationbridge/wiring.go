package main

import (
	"encoding/hex"
	"fmt"
	"path/filepath"

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
// PERSISTENT STORES (Story tf-4.1): Keys/Salts/DocumentKeys are wired to
// the AES-256-GCM-encrypted file implementations (keystore.File*Store) —
// a process restart no longer loses salts/employeeKey_i/KEY_EMPLOYEE.
// Store (the operational-value cache, distinct from these three secret
// stores) remains writepaths.NewInMemoryOperationalStore() — out of this
// story's scope, since it holds no secret key material, only a copy of
// data the real HRIS DB is the actual system of record for.
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

	encryptionKey, err := hex.DecodeString(cfg.KeystoreEncryptionKeyHex)
	if err != nil {
		return nil, nil, fmt.Errorf("integrationbridge: BRIDGE_KEYSTORE_ENCRYPTION_KEY_HEX is not valid hex: %w", err)
	}

	employeeKeys, err := keystore.NewFileEmployeeKeyStore(filepath.Join(cfg.KeystoreDir, "employee-keys.enc"), encryptionKey)
	if err != nil {
		return nil, nil, fmt.Errorf("integrationbridge: constructing employee key store: %w", err)
	}
	salts, err := keystore.NewFileSaltStore(filepath.Join(cfg.KeystoreDir, "salts.enc"), encryptionKey)
	if err != nil {
		return nil, nil, fmt.Errorf("integrationbridge: constructing salt store: %w", err)
	}
	documentKeys, err := keystore.NewFileDocumentKeyStore(filepath.Join(cfg.KeystoreDir, "document-keys.enc"), encryptionKey)
	if err != nil {
		return nil, nil, fmt.Errorf("integrationbridge: constructing document key store: %w", err)
	}

	hooks := &writepaths.Hooks{
		Store:        writepaths.NewInMemoryOperationalStore(),
		Keys:         employeeKeys,
		Salts:        salts,
		DocumentKeys: documentKeys,
		IPFS:         ipfsclient.NewClient(cfg.IPFSPrimaryAPI, cfg.IPFSReplicaAPI),
		Gateway:      gw,
		TenantID:     cfg.TenantID,
	}
	return hooks, gw, nil
}

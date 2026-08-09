// Consumer-owned ports for writepaths' two external capabilities: anchoring
// to the ledger (gateway-client) and pinning a supporting document (ipfsclient).
// Same dependency-inversion technique already established in this codebase
// for the identical reason — see integration-bridge/shutdown.go's closer
// interface doc comment: "gatewayclient.GatewayClient is a concrete struct
// with private fields and can't be mocked by substitution otherwise." Applied
// one layer down here, at the use-case boundary rather than the shutdown
// path, so anchor/doAnchor and the five write-path hooks become unit-testable
// without a live Fabric network or IPFS swarm.
//
// Both interfaces are narrowed to exactly what writepaths calls today — no
// Close, no Evaluate* reads, no FetchAndDecrypt. writepaths never reads back
// from the ledger or from IPFS, and never owns either connection's lifecycle
// (that's integration-bridge's main(), via buildHooks/shutdown.go).
package writepaths

import (
	"context"

	gatewayclient "gatewayclient"
)

// BuildArgsFunc MUST be a type alias, not a new defined type — Go interface
// satisfaction requires identical signatures, and a defined type is
// identical only to itself. Declaring this as
// `type BuildArgsFunc func(prevHash string) []string` instead would silently
// stop *gatewayclient.GatewayClient from satisfying LedgerAnchorer below.
type BuildArgsFunc = gatewayclient.BuildArgsFunc

// LedgerAnchorer is the only ledger capability the five write-path hooks
// need. Narrower than *gatewayclient.GatewayClient's full method set (Close,
// the three Evaluate* reads, Verify) — writepaths never reads back from the
// ledger and never owns the connection lifecycle (that's integration-bridge's
// main(), via its own `closer` interface in shutdown.go — the same pattern
// applied one layer down, for the same reason: a concrete struct with
// private fields can't be mocked by substitution otherwise).
type LedgerAnchorer interface {
	SubmitRecordProfileSection(ctx context.Context, tenantID, employeeID, profileSection string, buildArgs BuildArgsFunc) ([]byte, error)
}

// DocumentPinner is the only IPFS capability doAnchor needs. FetchAndDecrypt
// is deliberately absent — writepaths never reads a document back.
type DocumentPinner interface {
	EncryptAndAdd(ctx context.Context, key, plaintext []byte) (string, error)
}

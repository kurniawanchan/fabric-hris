// REC-3: Fabric Gateway client wrapper. One retained gRPC/Gateway
// connection per host process (ADR-0014's in-band model — the same process
// that just wrote the operational-DB section calls this, synchronously, in
// the same request; no queue, no standalone service). Wraps
// SubmitTransaction("RecordProfileSection", ...) and the three Evaluate
// functions; treats MVCC_READ_CONFLICT and a stale-prevHash rejection as
// the EXPECTED path (data-model.md §5.2), not an exceptional one — both
// trigger an automatic re-read-current-head-and-resubmit, not a bare error
// return to the caller.
package gatewayclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// MaxSubmitRetries bounds the "expected path" retry loop — DATA §5.2 calls
// re-read-and-resubmit expected, not that it should retry forever against a
// genuinely stuck/contended key.
const MaxSubmitRetries = 3

// GatewayClient holds ONE retained connection + contract handle, opened
// once by NewGatewayClient and reused for the lifetime of the host process
// (the backlog row's own DoD) — never re-dialed per call.
type GatewayClient struct {
	grpcConn *grpc.ClientConn
	gateway  *client.Gateway
	contract *client.Contract
}

// NewGatewayClient dials the peer's Gateway endpoint (served on the SAME
// port as the peer's main gRPC service since Fabric 2.4) exactly once.
// clientTLSCertPEMPath/clientTLSKeyPEMPath are the SEPARATE mTLS client
// identity (found at NET-6/CC-5: distinct from the MSP signing
// certPEMPath/keyPEMPath below — every peer in this network requires
// CORE_PEER_TLS_CLIENTAUTHREQUIRED=true, so presenting only a server-trust
// CA without a client certificate fails the handshake the same way the
// `peer` CLI failed until `--clientauth`/`--certfile`/`--keyfile` were
// added).
// tlsServerNameOverride matches the peer's cryptogen-issued cert SAN (e.g.
// "peer0.org1") even when peerEndpoint is reached via a different address
// (e.g. "localhost:7051" from a host process outside the docker network,
// which is not in that SAN list) — pass "" to use peerEndpoint's own
// hostname for verification (the in-network/normal case).
func NewGatewayClient(peerEndpoint, tlsServerNameOverride, tlsCACertPEMPath, clientTLSCertPEMPath, clientTLSKeyPEMPath, mspID, certPEMPath, keyPEMPath, channelName, chaincodeName string) (*GatewayClient, error) {
	tlsCreds, err := loadTLSCredentials(tlsCACertPEMPath, clientTLSCertPEMPath, clientTLSKeyPEMPath, tlsServerNameOverride)
	if err != nil {
		return nil, fmt.Errorf("gatewayclient: loading TLS credentials: %w", err)
	}

	conn, err := grpc.NewClient(peerEndpoint, grpc.WithTransportCredentials(tlsCreds))
	if err != nil {
		return nil, fmt.Errorf("gatewayclient: dialing %s: %w", peerEndpoint, err)
	}

	id, sign, err := loadIdentity(mspID, certPEMPath, keyPEMPath)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("gatewayclient: loading identity: %w", err)
	}

	gw, err := client.Connect(id,
		client.WithSign(sign),
		client.WithClientConnection(conn),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("gatewayclient: client.Connect: %w", err)
	}

	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	return &GatewayClient{grpcConn: conn, gateway: gw, contract: contract}, nil
}

// Close releases the retained connection. Call once, at host-process
// shutdown — never per transaction (that would defeat the "one retained
// connection" requirement this wrapper exists to satisfy).
func (g *GatewayClient) Close() error {
	g.gateway.Close()
	return g.grpcConn.Close()
}

// BuildArgsFunc constructs RecordProfileSection's argument list given the
// prevHash to use for this attempt. SubmitRecordProfileSection calls it
// once, and again on each expected-path retry with a freshly-read prevHash.
type BuildArgsFunc func(prevHash string) []string

// SubmitRecordProfileSection submits RecordProfileSection, automatically
// retrying up to MaxSubmitRetries times on the two EXPECTED rejection
// classes DATA §5.2 names — MVCC_READ_CONFLICT at commit time, and a
// stale-prevHash rejection at endorsement time — by re-reading the current
// head via Evaluate and calling buildArgs again with the fresh prevHash.
// Any OTHER error (a real ledger/gateway failure, or a non-retryable
// application rejection like a failed MSP/ABAC check) is returned
// immediately, distinctly, without masking it as a retry.
func (g *GatewayClient) SubmitRecordProfileSection(ctx context.Context, tenantID, employeeID, profileSection string, buildArgs BuildArgsFunc) ([]byte, error) {
	prevHash := ""
	// Best-effort: seed prevHash from the current head, if one exists, so
	// the FIRST attempt already has it right and does not need to fall
	// into the retry path just to discover the obvious case.
	if head, err := g.EvaluateGetProfileSectionRecord(ctx, tenantID, employeeID, profileSection); err == nil {
		prevHash = extractDataHash(head)
	}

	var lastErr error
	for attempt := 0; attempt <= MaxSubmitRetries; attempt++ {
		args := buildArgs(prevHash)
		result, err := g.contract.SubmitTransaction("RecordProfileSection", args...)
		if err == nil {
			return result, nil
		}
		lastErr = err

		if !isExpectedRetryableRejection(err) {
			return nil, &LedgerError{Err: err}
		}

		head, headErr := g.EvaluateGetProfileSectionRecord(ctx, tenantID, employeeID, profileSection)
		if headErr != nil {
			return nil, &LedgerError{Err: fmt.Errorf("re-read after retryable rejection failed: %w (original: %v)", headErr, err)}
		}
		prevHash = extractDataHash(head)
	}
	return nil, &LedgerError{Err: fmt.Errorf("exhausted %d retries on the expected MVCC/stale-prevHash path: %w", MaxSubmitRetries, lastErr)}
}

// EvaluateGetProfileSectionRecord/GetProfileHistory take tenantID FIRST —
// matching CC-3's actual argument order exactly (found the hard way: a
// 2-arg call missing tenantID fails with "Incorrect number of params.
// Expected 3, received 2", the same class of omission already hit once in
// this project's ccdeploy tool and not carried over automatically to this
// separate Go module).
func (g *GatewayClient) EvaluateGetProfileSectionRecord(ctx context.Context, tenantID, employeeID, profileSection string) ([]byte, error) {
	return g.evaluate(ctx, "GetProfileSectionRecord", tenantID, employeeID, profileSection)
}

func (g *GatewayClient) EvaluateGetProfileHistory(ctx context.Context, tenantID, employeeID, profileSection string) ([]byte, error) {
	return g.evaluate(ctx, "GetProfileHistory", tenantID, employeeID, profileSection)
}

func (g *GatewayClient) EvaluateGetEmployeeProfileSummary(ctx context.Context, tenantID, employeeID string) ([]byte, error) {
	return g.evaluate(ctx, "GetEmployeeProfileSummary", tenantID, employeeID)
}

func (g *GatewayClient) evaluate(ctx context.Context, fn string, args ...string) ([]byte, error) {
	result, err := g.contract.EvaluateWithContext(ctx, fn, client.WithArguments(args...))
	if err != nil {
		return nil, &LedgerError{Err: err}
	}
	return result, nil
}

// LedgerError distinguishes ledger/gateway-layer failures (network,
// endorsement, commit, or a non-retryable application rejection) from
// ordinary Go errors a caller might otherwise conflate with, e.g., a bad
// local argument — per this item's own DoD ("ledger/gateway errors
// surfaced distinctly from application-level rejections").
type LedgerError struct{ Err error }

func (e *LedgerError) Error() string { return "gatewayclient: ledger error: " + e.Err.Error() }
func (e *LedgerError) Unwrap() error { return e.Err }

// isExpectedRetryableRejection recognizes the two classes DATA §5.2 names
// as the expected, not exceptional, path.
func isExpectedRetryableRejection(err error) bool {
	var commitErr *client.CommitError
	if errors.As(err, &commitErr) && commitErr.Code == peer.TxValidationCode_MVCC_READ_CONFLICT {
		return true
	}
	// The stale-prevHash rejection is the chaincode's OWN business-logic
	// error (CC-2's ErrStaleChainReference), surfaced as an EndorseError
	// (rejected at simulation time, before ever reaching commit) — cross-
	// process, only the message text is available to match against.
	var endorseErr *client.EndorseError
	if errors.As(err, &endorseErr) && strings.Contains(endorseErr.Error(), "stale chain reference") {
		return true
	}
	return false
}

// extractDataHash pulls "dataHash" out of GetProfileSectionRecord's JSON
// result. A tiny hand-rolled extractor rather than a full JSON dependency
// here — this package already imports one JSON-adjacent library
// (jsoncanonicalizer) for a completely different purpose (REC-1); adding a
// second, general-purpose JSON decode path for this one field is scope
// callers can trivially do themselves with encoding/json if they need the
// other fields too. Returns "" if absent (e.g. NotFound — a legitimate
// first-write case, not an error).
func extractDataHash(recordJSON []byte) string {
	const marker = `"dataHash":"`
	s := string(recordJSON)
	idx := strings.Index(s, marker)
	if idx == -1 {
		return ""
	}
	rest := s[idx+len(marker):]
	end := strings.Index(rest, `"`)
	if end == -1 {
		return ""
	}
	return rest[:end]
}

func loadIdentity(mspID, certPEMPath, keyPEMPath string) (*identity.X509Identity, identity.Sign, error) {
	certPEM, err := os.ReadFile(certPEMPath)
	if err != nil {
		return nil, nil, err
	}
	cert, err := identity.CertificateFromPEM(certPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing certificate: %w", err)
	}
	id, err := identity.NewX509Identity(mspID, cert)
	if err != nil {
		return nil, nil, fmt.Errorf("creating X509 identity: %w", err)
	}

	keyPEM, err := os.ReadFile(keyPEMPath)
	if err != nil {
		return nil, nil, err
	}
	privateKey, err := identity.PrivateKeyFromPEM(keyPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing private key: %w", err)
	}
	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("creating signer: %w", err)
	}
	return id, sign, nil
}

func loadTLSCredentials(caCertPEMPath, clientCertPEMPath, clientKeyPEMPath, serverNameOverride string) (credentials.TransportCredentials, error) {
	caCertPEM, err := os.ReadFile(caCertPEMPath)
	if err != nil {
		return nil, err
	}
	caCert, err := identity.CertificateFromPEM(caCertPEM)
	if err != nil {
		return nil, fmt.Errorf("parsing TLS CA certificate: %w", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	clientCertPEM, err := os.ReadFile(clientCertPEMPath)
	if err != nil {
		return nil, err
	}
	clientKeyPEM, err := os.ReadFile(clientKeyPEMPath)
	if err != nil {
		return nil, err
	}
	clientCertPair, err := tls.X509KeyPair(clientCertPEM, clientKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parsing client TLS cert/key pair: %w", err)
	}

	return credentials.NewTLS(&tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{clientCertPair},
		ServerName:   serverNameOverride,
	}), nil
}

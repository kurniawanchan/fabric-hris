//go:build integration

// ST-4(a) (qa-tests/security transport-security sweep): every peer in this
// network sets CORE_PEER_TLS_CLIENTAUTHREQUIRED=true (confirmed via `docker
// exec <peer> env` against the live containers as part of this same sweep —
// peer0.org1, peer1.org1, peer0.tenant01, peer0.org3, peer0.tenant02 all show
// CORE_PEER_TLS_ENABLED=true and CORE_PEER_TLS_CLIENTAUTHREQUIRED=true). This
// file proves that holds in PRACTICE, not just in the container's env
// config: a gRPC client that never negotiates TLS at all (plain-text
// transport credentials, no certificate of any kind — not even a "wrong"
// one) must be refused by the peer's own listening socket, before any
// Fabric-level (MSP/identity/ABAC) check ever gets a chance to run.
//
// This deliberately does NOT reuse NewGatewayClient (every path through that
// constructor already negotiates real TLS via loadTLSCredentials) — it dials
// with google.golang.org/grpc/credentials/insecure directly, the same
// "transport" layer question CT-4's two tests in this package
// (invalid_identity_integration_test.go) explicitly did NOT cover (those
// both use a perfectly legitimate mTLS transport and forge only the
// higher-layer MSP claim). This is the complementary, lower-layer case.
package gatewayclient

import (
	"context"
	"testing"
	"time"

	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestIntegration_ST4_PlaintextConnectionToPeerRefused dials peer0.org1's
// live Gateway/gRPC endpoint (localhost:7051) with NO TLS negotiation
// whatsoever and attempts one RPC. grpc.NewClient itself is lazy (it never
// dials until the first RPC, per this package's own existing convention in
// gatewayclient.go's NewGatewayClient doc comment), so a real network
// round-trip only happens on the ProcessProposal call below — that call, not
// grpc.NewClient's return, is the load-bearing assertion. The exact error
// text/class (TLS handshake failure vs. connection reset vs. context
// deadline) is deliberately NOT pinned down, matching this package's own
// established precedent (invalid_identity_integration_test.go's two tests
// assert only err != nil for the same reason: it is an SDK/OS-transport
// implementation detail, not a stable contract).
func TestIntegration_ST4_PlaintextConnectionToPeerRefused(t *testing.T) {
	conn, err := grpc.NewClient("localhost:7051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		// grpc.NewClient only validates the target string and builds a lazy
		// connection object; it performs no network I/O itself. A failure
		// this early is a local construction problem, not the refusal this
		// test exists to demonstrate.
		t.Fatalf("grpc.NewClient (plaintext credentials, lazy dial) unexpectedly failed at construction: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	endorser := peer.NewEndorserClient(conn)
	// An empty SignedProposal is fine here: if the peer's listening socket
	// enforced TLS as configured, the connection itself must fail before
	// the (empty, otherwise-invalid) proposal content is ever inspected.
	_, err = endorser.ProcessProposal(ctx, &peer.SignedProposal{})
	if err == nil {
		t.Fatal("a PLAINTEXT (non-TLS) gRPC call to peer0.org1:7051 SUCCEEDED — CORE_PEER_TLS_CLIENTAUTHREQUIRED is NOT being enforced in practice, despite the container's own env config claiming it is")
	}
	t.Logf("plaintext (non-TLS) connection to peer0.org1:7051 correctly refused: %v", err)
}

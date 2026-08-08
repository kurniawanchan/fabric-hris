// Command netjoin executes backlog item NET-3: create the tenant channel
// (from the genesis block NET-1's configtxgen already produced) and join
// Org1's 2 peers, OrgClient-tenant01's peer, and Org3's peer to it.
//
// It runs on the host and drives the real `osnadmin`/`peer` binaries inside
// the `fabric-tools-net` helper container (started separately, attached to
// the `fabric-network-net` docker network NET-4 created, with
// fabric-network/network/ mounted read-only at /net) via `docker exec` — the
// orchestration/sequencing/verification logic is Go; the wire protocol work
// is delegated to Fabric's own official binaries, per this project's
// established pattern (NET-1/NET-2 also drove real binaries rather than
// reimplementing Fabric's protocols).
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	helperContainer = "fabric-tools-net"
	channelID       = "tenant-tenant01"
	blockPath       = "/net/channel-artifacts/tenant-tenant01.block"

	// osnadmin's own TLS client identity. Per Fabric's channel-participation
	// admin model, the Admin listener authorizes by mTLS trust chain alone
	// (Admin.TLS.ClientRootCAs), not by MSP role — reusing one orderer's own
	// TLS cert/key as the "admin caller" identity is the documented pattern
	// (matches fabric-samples' test-network scripts), not a workaround.
	adminTLSCert = "/net/crypto-config/ordererOrganizations/org1/orderers/orderer0.org1/tls/server.crt"
	adminTLSKey  = "/net/crypto-config/ordererOrganizations/org1/orderers/orderer0.org1/tls/server.key"
	adminTLSCA   = "/net/crypto-config/ordererOrganizations/org1/orderers/orderer0.org1/tls/ca.crt"
)

type ordererNode struct {
	name      string
	adminAddr string
}

type peerNode struct {
	name      string
	address   string
	mspID     string
	adminMSP  string
	tlsCAFile string
}

var orderers = []ordererNode{
	{"orderer0.org1", "orderer0.org1:9443"},
	{"orderer1.org1", "orderer1.org1:9443"},
	{"orderer2.org1", "orderer2.org1:9443"},
}

var peers = []peerNode{
	{"peer0.org1", "peer0.org1:7051", "Org1MSP",
		"/net/crypto-config/peerOrganizations/org1/users/Admin@org1/msp",
		"/net/crypto-config/peerOrganizations/org1/tlsca/tlsca.org1-cert.pem"},
	{"peer1.org1", "peer1.org1:7051", "Org1MSP",
		"/net/crypto-config/peerOrganizations/org1/users/Admin@org1/msp",
		"/net/crypto-config/peerOrganizations/org1/tlsca/tlsca.org1-cert.pem"},
	{"peer0.tenant01", "peer0.tenant01:7051", "OrgClient-tenant01MSP",
		"/net/crypto-config/peerOrganizations/tenant01/users/Admin@tenant01/msp",
		"/net/crypto-config/peerOrganizations/tenant01/tlsca/tlsca.tenant01-cert.pem"},
	{"peer0.org3", "peer0.org3:7051", "Org3MSP",
		"/net/crypto-config/peerOrganizations/org3/users/Admin@org3/msp",
		"/net/crypto-config/peerOrganizations/org3/tlsca/tlsca.org3-cert.pem"},
}

func dockerExec(args ...string) (string, error) {
	cmd := exec.Command("docker", append([]string{"exec", helperContainer}, args...)...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func joinOrderer(o ordererNode) error {
	// Idempotency: this tool may be re-run after a partial failure further
	// down the pipeline (e.g. the peer step). osnadmin channel join is not
	// itself idempotent (a second join attempt errors), so check first.
	listOut, _ := dockerExec(
		"osnadmin", "channel", "list", "-o", o.adminAddr,
		"--ca-file", adminTLSCA, "--client-cert", adminTLSCert, "--client-key", adminTLSKey,
	)
	if strings.Contains(listOut, channelID) {
		fmt.Printf("--- osnadmin channel join: %s --- already joined, skipping\n", o.name)
		return nil
	}

	out, err := dockerExec(
		"osnadmin", "channel", "join",
		"--channelID", channelID,
		"--config-block", blockPath,
		"-o", o.adminAddr,
		"--ca-file", adminTLSCA,
		"--client-cert", adminTLSCert,
		"--client-key", adminTLSKey,
	)
	fmt.Printf("--- osnadmin channel join: %s ---\n%s\n", o.name, out)
	if err != nil {
		return fmt.Errorf("joining orderer %s: %w", o.name, err)
	}
	return nil
}

func peerEnv(p peerNode) string {
	// The peer's gRPC server requires client TLS auth
	// (CORE_PEER_TLS_CLIENTAUTHREQUIRED=true, set at NET-4 bring-up) — the CLI
	// connecting to it must present its own client cert/key, not just trust
	// the server's. cryptogen generates a tls/client.{crt,key} pair for every
	// enrolled user (including Admin@<org>), reused here as the CLI's own
	// mTLS client identity. Omitting these caused the first real run to hang
	// until "context deadline exceeded" (TLS handshake never completed) —
	// found by running this tool, not by re-reading the design.
	adminTLSDir := strings.TrimSuffix(p.adminMSP, "/msp") + "/tls"
	return fmt.Sprintf(
		"CORE_PEER_LOCALMSPID=%s CORE_PEER_MSPCONFIGPATH=%s CORE_PEER_ADDRESS=%s "+
			"CORE_PEER_TLS_ENABLED=true CORE_PEER_TLS_ROOTCERT_FILE=%s "+
			"CORE_PEER_TLS_CLIENTAUTHREQUIRED=true "+
			"CORE_PEER_TLS_CLIENTCERT_FILE=%s/client.crt CORE_PEER_TLS_CLIENTKEY_FILE=%s/client.key",
		p.mspID, p.adminMSP, p.address, p.tlsCAFile, adminTLSDir, adminTLSDir,
	)
}

func joinPeer(p peerNode) error {
	shCmd := fmt.Sprintf("%s peer channel join -b %s", peerEnv(p), blockPath)
	out, err := dockerExec("sh", "-c", shCmd)
	fmt.Printf("--- peer channel join: %s ---\n%s\n", p.name, out)
	if err != nil && strings.Contains(out, "already exists") {
		fmt.Printf("(%s already joined, treating as success)\n", p.name)
		return nil
	}
	if err != nil {
		return fmt.Errorf("joining peer %s: %w", p.name, err)
	}
	return nil
}

func listChannels(p peerNode) (string, error) {
	shCmd := fmt.Sprintf("%s peer channel list", peerEnv(p))
	return dockerExec("sh", "-c", shCmd)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "FATAL:", err)
	os.Exit(1)
}

func main() {
	fmt.Println("=== NET-3: create channel", channelID, "and join Org1(x2)/OrgClient-tenant01/Org3 ===")

	fmt.Println("\n-- Step 1: osnadmin channel join on all 3 orderers --")
	for _, o := range orderers {
		if err := joinOrderer(o); err != nil {
			fail(err)
		}
		time.Sleep(1 * time.Second)
	}

	fmt.Println("\n-- Step 2: peer channel join on all 4 peers --")
	for _, p := range peers {
		if err := joinPeer(p); err != nil {
			fail(err)
		}
	}

	fmt.Println("\n-- Step 3: verify — peer channel list on every peer --")
	allJoined := true
	for _, p := range peers {
		out, err := listChannels(p)
		joined := err == nil && strings.Contains(out, channelID)
		fmt.Printf("%-16s joined=%v\n%s\n", p.name, joined, out)
		if !joined {
			allJoined = false
		}
	}

	if !allJoined {
		fail(fmt.Errorf("not all peers report channel %q joined", channelID))
	}
	fmt.Println("SUCCESS: all 4 peers report", channelID, "joined")
}

// Command ccdeploy executes backlog item CC-5: install the
// employeeprofilerecord chaincode package on Org1's 2 peers +
// OrgClient-tenant01's peer + Org3's peer, approve for Org1 and
// OrgClient-tenant01 with NET-5's AND-endorsement policy, commit, and run an
// invoke + query smoke test against the live tenant-tenant01 channel.
//
// Same pattern as netjoin (NET-3) and raftfaulttest (NET-6): drives the real
// `peer` binary inside the network-attached `fabric-tools-net` helper
// container via `docker exec`; the sequencing/verification is Go.
//
// PACKAGE FORMAT — Chaincode-as-a-Service (CCaaS), not `--lang golang`.
// `--lang golang` requires the peer to build the chaincode via
// Docker-in-Docker (mounting the host's /var/run/docker.sock into peer
// containers) — refused as a privilege-escalation-shaped grant; CCaaS avoids
// it entirely. The package at pkgPath is NOT produced by this tool — it is a
// hand-built tarball (`peer lifecycle chaincode package` does not support
// `--lang external` at all):
//
//	metadata.json:  {"type": "ccaas", "label": "employeeprofilerecord_1.0"}
//	connection.json: {"address": "employeeprofilerecord-ccaas:9999",
//	                  "dial_timeout": "10s", "tls_required": false}
//	code.tar.gz := tar(connection.json)      # at the package source ROOT,
//	                                           # NOT a chaincode/server/ subdir
//	package.tar.gz := tar(code.tar.gz, metadata.json)
//
// The peer image ships a working `ccaas_builder` pre-registered in its
// default core.yaml (`/opt/hyperledger/ccaas_builder`) — do NOT register a
// custom externalBuilders entry or bind-mount anything at that path; doing
// so silently REPLACES the correct built-in binaries with broken stand-ins
// (found the hard way — three separate packaging-format defects, documented
// in implementation-backlog.md's CC-5 row).
//
// The chaincode server itself (a separate, independently-started container,
// `employeeprofilerecord-ccaas`, built from
// fabric-network/chaincode/employeeprofilerecord/Dockerfile) must be started
// with CHAINCODE_ID set to the EXACT package ID `queryinstalled` reports —
// that ID changes whenever the package's byte content changes, so the
// server must be restarted with the new ID after any repackaging.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	helperContainer = "fabric-tools-net"
	channelID       = "tenant-tenant01"
	ordererAddr     = "orderer0.org1:7050"
	ordererTLSCA    = "/net/crypto-config/ordererOrganizations/org1/tlsca/tlsca.org1-cert.pem"
	pkgPath         = "/cc/employeeprofilerecord.tar.gz"
	ccName          = "employeeprofilerecord"
	ccVersion       = "1.0"
	ccSequence      = "2" // sequence 1 was the golang-package attempt, superseded; the CCaaS package actually committed at sequence 2
	signaturePolicy = "AND('Org1MSP.peer','OrgClient-tenant01MSP.peer')"

	org1TLSDir    = "/net/crypto-config/peerOrganizations/org1/users/Admin@org1/tls"
	org1TLSCAFile = "/net/crypto-config/peerOrganizations/org1/tlsca/tlsca.org1-cert.pem"
)

type peerNode struct {
	name      string
	address   string
	mspID     string
	adminMSP  string
	tlsCAFile string
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

func dockerExec(shCmd string) (string, error) {
	cmd := exec.Command("docker", "exec", helperContainer, "sh", "-c", shCmd)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func must(step, out string, err error) string {
	if err != nil {
		fmt.Printf("--- %s (FAILED) ---\n%s\n", step, out)
		fmt.Fprintf(os.Stderr, "FATAL at %s: %v\n", step, err)
		os.Exit(1)
	}
	fmt.Printf("--- %s ---\n%s\n", step, out)
	return out
}

func run(step, shCmd string) string {
	out, err := dockerExec(shCmd)
	return must(step, out, err)
}

// tryRun does not exit on failure — used where a non-zero exit is expected
// or merely informative (idempotency checks).
func tryRun(step, shCmd string) (string, error) {
	out, err := dockerExec(shCmd)
	fmt.Printf("--- %s ---\n%s\n", step, out)
	return out, err
}

func peerEnv(p peerNode) string {
	return fmt.Sprintf(
		"CORE_PEER_LOCALMSPID=%s CORE_PEER_MSPCONFIGPATH=%s CORE_PEER_ADDRESS=%s "+
			"CORE_PEER_TLS_ENABLED=true CORE_PEER_TLS_ROOTCERT_FILE=%s "+
			"CORE_PEER_TLS_CLIENTAUTHREQUIRED=true "+
			"CORE_PEER_TLS_CLIENTCERT_FILE=%s CORE_PEER_TLS_CLIENTKEY_FILE=%s",
		p.mspID, p.adminMSP, p.address, p.tlsCAFile,
		adminTLSClientCert(p), adminTLSClientKey(p),
	)
}

func adminTLSDir(p peerNode) string {
	return strings.TrimSuffix(p.adminMSP, "/msp") + "/tls"
}
func adminTLSClientCert(p peerNode) string { return adminTLSDir(p) + "/client.crt" }
func adminTLSClientKey(p peerNode) string  { return adminTLSDir(p) + "/client.key" }

// ordererFlags is required on every command that talks to the orderer
// (install does not; approve/commit/invoke/query do) — --clientauth is
// mandatory or the client silently sends no certificate (found at NET-6).
func ordererFlags() string {
	return fmt.Sprintf(
		"-o %s --tls --clientauth --cafile %s --certfile %s --keyfile %s",
		ordererAddr, ordererTLSCA, org1TLSDir+"/client.crt", org1TLSDir+"/client.key",
	)
}

func installOn(p peerNode) {
	out, err := tryRun("install on "+p.name, fmt.Sprintf(
		"%s peer lifecycle chaincode install %s", peerEnv(p), pkgPath))
	if err != nil && strings.Contains(out, "already successfully installed") {
		fmt.Printf("(%s: already installed, treating as success)\n", p.name)
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL installing on %s: %v\n", p.name, err)
		os.Exit(1)
	}
}

func queryPackageID(p peerNode) string {
	out := run("queryinstalled on "+p.name, fmt.Sprintf(
		"%s peer lifecycle chaincode queryinstalled", peerEnv(p)))
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, ccName+"_"+ccVersion) {
			// Line shape: "Package ID: employeeprofilerecord_1.0:<hash>, Label: employeeprofilerecord_1.0"
			parts := strings.Split(line, "Package ID: ")
			if len(parts) == 2 {
				idAndRest := strings.SplitN(parts[1], ",", 2)
				return strings.TrimSpace(idAndRest[0])
			}
		}
	}
	fmt.Fprintln(os.Stderr, "FATAL: could not find package ID in queryinstalled output")
	os.Exit(1)
	return ""
}

func approveFor(p peerNode, packageID string) {
	run("approveformyorg by "+p.mspID, fmt.Sprintf(
		"%s peer lifecycle chaincode approveformyorg %s --channelID %s --name %s --version %s "+
			"--package-id %s --sequence %s --signature-policy \"%s\"",
		peerEnv(p), ordererFlags(), channelID, ccName, ccVersion, packageID, ccSequence, signaturePolicy))
}

func commit(approvers []peerNode) {
	var peerFlags []string
	for _, p := range approvers {
		peerFlags = append(peerFlags, fmt.Sprintf("--peerAddresses %s --tlsRootCertFiles %s", p.address, p.tlsCAFile))
	}
	env := peerEnv(approvers[0]) // submit as the first approver (Org1)
	run("commit", fmt.Sprintf(
		"%s peer lifecycle chaincode commit %s --channelID %s --name %s --version %s "+
			"--sequence %s --signature-policy \"%s\" %s",
		env, ordererFlags(), channelID, ccName, ccVersion, ccSequence, signaturePolicy, strings.Join(peerFlags, " ")))
}

func invokeSmoke(writer, org1 peerNode) {
	args := `{"function":"RecordProfileSection","Args":["tenant01","cc5deadbeef00000000000000000000000000000000000000000000000001","PERSONAL","sha256:cc5smoketest0000000000000000000000000000000000000000000000000",` +
		`"","cc5actor000000000000000000000000000000000000000000000000000002","[]","JCS-RFC8785-v1","SHA-256",""]}`
	peerFlags := fmt.Sprintf("--peerAddresses %s --tlsRootCertFiles %s --peerAddresses %s --tlsRootCertFiles %s",
		writer.address, writer.tlsCAFile, org1.address, org1.tlsCAFile)
	run("invoke RecordProfileSection (smoke test)", fmt.Sprintf(
		"%s peer chaincode invoke %s -C %s -n %s -c '%s' %s --waitForEvent",
		peerEnv(writer), ordererFlags(), channelID, ccName, args, peerFlags))
}

func querySmoke(reader peerNode) string {
	// tenantId is the query functions' FIRST argument (queries.go) — omitting
	// it produces "Incorrect number of params. Expected 3, received 2" (a
	// real error hit once, not a hypothetical).
	args := `{"function":"GetProfileSectionRecord","Args":["tenant01","cc5deadbeef00000000000000000000000000000000000000000000000001","PERSONAL"]}`
	return run("query GetProfileSectionRecord (smoke test)", fmt.Sprintf(
		"%s peer chaincode query -C %s -n %s -c '%s'",
		peerEnv(reader), channelID, ccName, args))
}

func main() {
	fmt.Println("=== CC-5: install / approve / commit / smoke-test employeeprofilerecord ===")

	fmt.Println("\n-- Step 1: install on all 4 peers --")
	for _, p := range peers {
		installOn(p)
	}

	packageID := queryPackageID(peers[0])
	fmt.Println("\nPackage ID:", packageID)

	fmt.Println("\n-- Step 2: approve for Org1 and OrgClient-tenant01 --")
	approveFor(peers[0], packageID) // Org1, via peer0.org1
	approveFor(peers[2], packageID) // OrgClient-tenant01, via peer0.tenant01

	fmt.Println("\n-- Step 3: check commit readiness --")
	run("checkcommitreadiness", fmt.Sprintf(
		"%s peer lifecycle chaincode checkcommitreadiness %s --channelID %s --name %s --version %s "+
			"--sequence %s --signature-policy \"%s\" --output json",
		peerEnv(peers[0]), ordererFlags(), channelID, ccName, ccVersion, ccSequence, signaturePolicy))

	fmt.Println("\n-- Step 4: commit --")
	commit([]peerNode{peers[0], peers[2]})

	fmt.Println("\n-- Step 5: invoke smoke test (write) --")
	invokeSmoke(peers[0], peers[2]) // Org1 peer submits, endorsed by Org1 + tenant01

	fmt.Println("\n-- Step 6: query smoke test (read) --")
	out := querySmoke(peers[0])

	if !strings.Contains(out, "cc5smoketest") {
		fmt.Fprintln(os.Stderr, "FAIL: query did not return the written dataHash")
		os.Exit(1)
	}
	fmt.Println("\nSUCCESS: chaincode installed, approved, committed, and a real write+read round-trip verified on the live network")
}

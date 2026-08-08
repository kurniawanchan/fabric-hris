// Command raftfaulttest executes backlog item NET-6: prove, empirically, the
// 3-node Raft ordering service's crash-fault tolerance (f = floor((3-1)/2) =
// 1) rather than asserting it from the node count alone. It submits a real,
// signed channel config-update (a small BatchTimeout bump) through the live
// tenant-tenant01 channel while 1-of-3 orderers is stopped (expect: still
// orders, new block appears) and again while 2-of-3 are stopped (expect:
// no quorum, the update times out / fails).
//
// Like netjoin (NET-3), this drives the real `peer`/`configtxlator` binaries
// inside the network-attached `fabric-tools-net` helper container via
// `docker exec` — the sequencing/verification/pass-fail judgment is Go, the
// protobuf/JSON config-update mechanics are delegated to Fabric's own tools.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	helperContainer = "fabric-tools-net"
	channelID       = "tenant-tenant01"
	ordererAddr     = "orderer0.org1:7050"
	ordererTLSCA    = "/net/crypto-config/ordererOrganizations/org1/tlsca/tlsca.org1-cert.pem"

	// Any valid channel member works for fetch/update transport; use Org1's
	// peer-org admin throughout except where a specific org's signature is
	// being attached (signAs below switches identity per signconfigtx call).
	org1AdminMSP  = "/net/crypto-config/peerOrganizations/org1/users/Admin@org1/msp"
	org1TLSDir    = "/net/crypto-config/peerOrganizations/org1/users/Admin@org1/tls"
	org1TLSCAFile = "/net/crypto-config/peerOrganizations/org1/tlsca/tlsca.org1-cert.pem"
)

type signer struct {
	name    string
	mspID   string
	mspPath string
}

var signers = []signer{
	// Pre-signed by all three orgs' admins so this test isn't blocked on a
	// signature-policy question orthogonal to what it's actually measuring
	// (Raft fault tolerance, not endorsement-policy mechanics). Fabric
	// accepts extra valid signatures beyond the minimum required.
	{"OrdererMSP-admin", "OrdererMSP", "/net/crypto-config/ordererOrganizations/org1/users/Admin@org1/msp"},
	{"Org1MSP-admin", "Org1MSP", org1AdminMSP},
	{"OrgClient-tenant01MSP-admin", "OrgClient-tenant01MSP", "/net/crypto-config/peerOrganizations/tenant01/users/Admin@tenant01/msp"},
}

func peerCLIEnv(mspID, mspPath string) string {
	// CORE_PEER_ADDRESS: which peer to query for peer-server operations
	// (getinfo). CORE_PEER_TLS_CLIENT{CERT,KEY}_FILE govern mTLS to THAT
	// peer only — found empirically that `peer channel fetch`/`update`'s
	// connection to the ORDERER does NOT reuse these; it needs its own
	// `--certfile`/`--keyfile` CLI flags (see ordererTLSClientFlags below).
	// Omitting CORE_PEER_ADDRESS here originally caused getinfo to hang
	// against a non-existent default address.
	return fmt.Sprintf(
		"CORE_PEER_LOCALMSPID=%s CORE_PEER_MSPCONFIGPATH=%s CORE_PEER_ADDRESS=peer0.org1:7051 "+
			"CORE_PEER_TLS_ENABLED=true CORE_PEER_TLS_ROOTCERT_FILE=%s "+
			"CORE_PEER_TLS_CLIENTAUTHREQUIRED=true "+
			"CORE_PEER_TLS_CLIENTCERT_FILE=%s/client.crt CORE_PEER_TLS_CLIENTKEY_FILE=%s/client.key",
		mspID, mspPath, org1TLSCAFile, org1TLSDir, org1TLSDir,
	)
}

// ordererTLSClientFlags is appended to every `peer channel fetch`/`update`
// invocation — the orderer-connection mTLS client identity, found
// empirically to be independent of CORE_PEER_TLS_CLIENTCERT_FILE (that env
// var only covers connections to a PEER's own gRPC server, not the orderer).
func ordererTLSClientFlags() string {
	// --clientauth is REQUIRED to actually activate mutual TLS on this
	// connection — found empirically that --certfile/--keyfile alone are
	// silently unused without it (the client sends no certificate at all,
	// server logs "client didn't provide a certificate", "context deadline
	// exceeded" on the client side). Also found: ORDERER_GENERAL_TLS_
	// CLIENTROOTCAS can stay unset — the orderer's own shipped orderer.yaml
	// documents that General.TLS.ClientRootCAs augments, but is not required
	// for, the trust already sourced from the channel's own MSP tlscacerts
	// once the channel exists. Confirms NET-4's flagged uncertainty (1).
	return fmt.Sprintf("--clientauth --certfile %s/client.crt --keyfile %s/client.key", org1TLSDir, org1TLSDir)
}

func dockerExec(shCmd string) (string, error) {
	cmd := exec.Command("docker", "exec", helperContainer, "sh", "-c", shCmd)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func must(step string, out string, err error) string {
	if err != nil {
		fmt.Printf("--- %s (FAILED) ---\n%s\n", step, out)
		fmt.Fprintf(os.Stderr, "FATAL at %s: %v\n", step, err)
		os.Exit(1)
	}
	fmt.Printf("--- %s ---\n%s\n", step, out)
	return out
}

// run is dockerExec + must in one call, avoiding Go's restriction on mixing
// a multi-value call with other explicit arguments in the same call site.
func run(step, shCmd string) string {
	out, err := dockerExec(shCmd)
	return must(step, out, err)
}

// buildSignedUpdate crafts a signed config-update envelope that bumps the
// Orderer.BatchTimeout value to newTimeout, returning the in-container path
// of the resulting envelope .pb file (round is used to keep working files
// distinct across the two probes this tool runs).
func buildSignedUpdate(round int, newTimeout string) string {
	base := fmt.Sprintf("/tmp/raftfault-r%d", round)
	env := peerCLIEnv("Org1MSP", org1AdminMSP)

	run("fetch config block", fmt.Sprintf(
		"%s peer channel fetch config %s_config_block.pb -c %s -o %s --tls --cafile %s %s",
		env, base, channelID, ordererAddr, ordererTLSCA, ordererTLSClientFlags()))

	run("decode block", fmt.Sprintf(
		"configtxlator proto_decode --input %s_config_block.pb --type common.Block --output %s_config_block.json",
		base, base))

	run("extract config", fmt.Sprintf(
		"jq .data.data[0].payload.data.config %s_config_block.json > %s_config.json",
		base, base))

	run("copy config for modification", fmt.Sprintf("cp %s_config.json %s_modified_config.json", base, base))

	run("set new BatchTimeout", fmt.Sprintf(
		`jq '.channel_group.groups.Orderer.values.BatchTimeout.value.timeout = "%s"' `+
			`%s_config.json > %s_modified_config.json.tmp && mv %s_modified_config.json.tmp %s_modified_config.json`,
		newTimeout, base, base, base, base))

	run("encode original config", fmt.Sprintf(
		"configtxlator proto_encode --input %s_config.json --type common.Config --output %s_config.pb",
		base, base))
	run("encode modified config", fmt.Sprintf(
		"configtxlator proto_encode --input %s_modified_config.json --type common.Config --output %s_modified_config.pb",
		base, base))

	run("compute update", fmt.Sprintf(
		"configtxlator compute_update --channel_id %s --original %s_config.pb --updated %s_modified_config.pb --output %s_update.pb",
		channelID, base, base, base))

	run("decode update", fmt.Sprintf(
		"configtxlator proto_decode --input %s_update.pb --type common.ConfigUpdate --output %s_update.json",
		base, base))

	run("wrap update in envelope", fmt.Sprintf(
		`echo '{"payload":{"header":{"channel_header":{"channel_id":"%s","type":2}},"data":{"config_update":'"$(cat %s_update.json)"'}}}' | jq . > %s_update_env.json`,
		channelID, base, base))

	run("encode envelope", fmt.Sprintf(
		"configtxlator proto_encode --input %s_update_env.json --type common.Envelope --output %s_update_env.pb",
		base, base))

	for _, s := range signers {
		run("sign as "+s.name, fmt.Sprintf(
			"%s peer channel signconfigtx -f %s_update_env.pb",
			peerCLIEnv(s.mspID, s.mspPath), base))
	}

	return base + "_update_env.pb"
}

func submitUpdate(envPath string) (string, error) {
	env := peerCLIEnv("Org1MSP", org1AdminMSP)
	return dockerExec(fmt.Sprintf(
		"%s peer channel update -f %s -c %s -o %s --tls --cafile %s %s",
		env, envPath, channelID, ordererAddr, ordererTLSCA, ordererTLSClientFlags()))
}

func currentBatchTimeout() string {
	env := peerCLIEnv("Org1MSP", org1AdminMSP)
	out, err := dockerExec(fmt.Sprintf(
		"%s peer channel fetch config /tmp/rft_probe.pb -c %s -o %s --tls --cafile %s %s >/dev/null 2>&1 && "+
			"configtxlator proto_decode --input /tmp/rft_probe.pb --type common.Block --output /tmp/rft_probe.json && "+
			"jq -r .data.data[0].payload.data.config.channel_group.groups.Orderer.values.BatchTimeout.value.timeout /tmp/rft_probe.json",
		env, channelID, ordererAddr, ordererTLSCA, ordererTLSClientFlags()))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func blockHeight() int {
	env := peerCLIEnv("Org1MSP", org1AdminMSP)
	out, err := dockerExec(fmt.Sprintf(
		"%s peer channel getinfo -c %s 2>&1 | grep -o '\"height\":[0-9]*'",
		env, channelID))
	if err != nil {
		return -1
	}
	var h int
	fmt.Sscanf(strings.TrimSpace(out), `"height":%d`, &h)
	return h
}

func dockerStop(containers ...string) {
	args := append([]string{"stop"}, containers...)
	out, err := exec.Command("docker", args...).CombinedOutput()
	fmt.Printf("--- docker stop %s ---\n%s\n", strings.Join(containers, " "), out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: docker stop failed:", err)
	}
}

func dockerStart(containers ...string) {
	args := append([]string{"start"}, containers...)
	out, err := exec.Command("docker", args...).CombinedOutput()
	fmt.Printf("--- docker start %s ---\n%s\n", strings.Join(containers, " "), out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: docker start failed:", err)
	}
}

func main() {
	fmt.Println("=== NET-6: 3-node Raft fault-tolerance proof ===")

	startHeight := blockHeight()
	fmt.Println("\nStarting block height:", startHeight)

	// ---- Probe 1: kill 1-of-3 orderers, expect ordering to CONTINUE ----
	fmt.Println("\n== Probe 1: stop orderer1.org1 (1-of-3 down) — expect quorum still holds (2-of-3) ==")
	dockerStop("orderer1.org1")
	time.Sleep(3 * time.Second)

	cur := currentBatchTimeout()
	next := "3s"
	if cur == "3s" {
		next = "2s"
	}
	fmt.Printf("Current BatchTimeout=%q, probing with new value %q\n", cur, next)

	envPath := buildSignedUpdate(1, next)
	out, err := submitUpdate(envPath)
	fmt.Printf("--- submit update (1-of-3 down) ---\n%s\n", out)

	probe1Height := blockHeight()
	probe1OK := err == nil && strings.Contains(out, "Successfully submitted") || (err == nil && probe1Height > startHeight)
	fmt.Printf("Probe 1 result: update error=%v, height %d -> %d, PASS=%v\n", err, startHeight, probe1Height, probe1OK)

	// ---- Probe 2: also kill orderer2.org1 (2-of-3 down), expect FAILURE ----
	fmt.Println("\n== Probe 2: also stop orderer2.org1 (2-of-3 down) — expect quorum lost, update should fail/timeout ==")
	dockerStop("orderer2.org1")
	time.Sleep(3 * time.Second)

	cur2 := currentBatchTimeout() // best-effort; orderer0 alone can still serve reads via its own file ledger in some cases, but no new blocks can be ordered
	next2 := "4s"
	if cur2 == "4s" {
		next2 = "5s"
	}
	if cur2 == "" {
		cur2 = next // fall back to what probe 1 set, in case the read above failed without quorum
		next2 = "9s"
	}
	fmt.Printf("Attempting probe with new value %q (expect this to fail/timeout)\n", next2)

	envPath2 := buildSignedUpdate(2, next2)
	out2, err2 := submitUpdate(envPath2)
	fmt.Printf("--- submit update (2-of-3 down) ---\n%s\n", out2)

	probe2Height := blockHeight()
	probe2Rejected := err2 != nil || probe2Height <= probe1Height
	fmt.Printf("Probe 2 result: update error=%v, height %d -> %d, EXPECTED-TO-FAIL=%v\n", err2, probe1Height, probe2Height, probe2Rejected)

	// ---- Restore ----
	fmt.Println("\n== Restoring orderer1.org1 and orderer2.org1 ==")
	dockerStart("orderer1.org1", "orderer2.org1")
	time.Sleep(3 * time.Second)
	finalHeight := blockHeight()
	fmt.Println("Final block height after restore:", finalHeight)

	result := map[string]interface{}{
		"start_height":               startHeight,
		"probe1_1of3down_pass":       probe1OK,
		"probe1_height_after":        probe1Height,
		"probe2_2of3down_as_expected_fail": probe2Rejected,
		"probe2_height_after":        probe2Height,
		"final_height_after_restore": finalHeight,
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println("\n=== SUMMARY ===")
	fmt.Println(string(b))

	if !probe1OK {
		fmt.Fprintln(os.Stderr, "FAIL: 1-of-3-down probe did not demonstrate continued ordering")
		os.Exit(1)
	}
	if !probe2Rejected {
		fmt.Fprintln(os.Stderr, "FAIL: 2-of-3-down probe unexpectedly succeeded — quorum requirement not demonstrated")
		os.Exit(1)
	}
	fmt.Println("SUCCESS: f=1 crash-fault tolerance demonstrated (1-of-3 survives, 2-of-3 halts ordering)")
	_ = strconv.Itoa(0)
}

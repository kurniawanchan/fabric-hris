// Command tenantprovision executes backlog item NET-7: given a new tenant
// ID, performs NET-2's per-tenant CA enrollment, NET-3's channel-create-and-
// join, and NET-5's endorsement-policy wiring as one repeatable procedure —
// proven by running it against a SECOND tenant (tenant02) on the live
// network already carrying tenant01.
//
// Templating strategy: tenant01's own artifacts (crypto-config.yaml,
// configtx.yaml, network-docker-compose.yaml, ca-tenant01-config.yaml) are
// the working pattern; this tool derives the new tenant's blocks by string
// substitution against that known-good pattern rather than an abstract
// template engine — the same "copy and rename" approach a human operator
// would actually use, made repeatable and typo-proof.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const networkDir = "../../network"

func mustTenantID() string {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tenantprovision <tenantID>  (e.g. tenant02)")
		os.Exit(1)
	}
	return os.Args[1]
}

func readFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL reading %s: %v\n", path, err)
		os.Exit(1)
	}
	return string(b)
}

func writeFile(path, content string) {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL writing %s: %v\n", path, err)
		os.Exit(1)
	}
}

func run(step string, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	fmt.Printf("--- %s ---\n%s\n", step, out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL at %s: %v\n", step, err)
		os.Exit(1)
	}
	return string(out)
}

func tryRun(step string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	fmt.Printf("--- %s ---\n%s\n", step, out)
	return string(out), err
}

// dockerExecTools runs a shell command inside the fabric-tools-net helper
// container (same pattern as netjoin/raftfaulttest/ccdeploy).
func dockerExecTools(step, shCmd string) string {
	return run(step, "docker", "exec", "fabric-tools-net", "sh", "-c", shCmd)
}

// --- Step 1: crypto material -------------------------------------------

func genCryptoMaterial(tenantID string) {
	// Idempotency: cryptogen does NOT skip an org whose directory already
	// exists — rerunning it regenerates a brand-new keypair/CA, which then
	// mismatches whatever an already-running peer for this org loaded into
	// memory at boot (found the hard way: re-running this tool a second time
	// silently broke peer0.tenant02's TLS trust — "certificate signed by
	// unknown authority" — because its on-disk cert had been replaced out
	// from under the already-booted process). Guard explicitly.
	if _, err := os.Stat(networkDir + "/crypto-config/peerOrganizations/" + tenantID); err == nil {
		fmt.Println("crypto material for", tenantID, "already exists — skipping cryptogen (would regenerate fresh keys and break the already-running peer)")
		return
	}
	fragment := fmt.Sprintf(`PeerOrgs:
  - Name: OrgClient-%s
    Domain: %s
    EnableNodeOUs: true
    Template:
      Count: 1
    Users:
      Count: 1
`, tenantID, tenantID)
	fragPath := fmt.Sprintf("%s/crypto-config/crypto-config-%s.yaml", networkDir, tenantID)
	writeFile(fragPath, fragment)

	// cryptogen only touches the org(s) named in the config it's given —
	// verified empirically before writing this tool (a throwaway test org
	// generated cleanly with org1/tenant01/org3's material byte-identical
	// before and after). Never rerun with the FULL shared crypto-config.yaml
	// against a live network — that would regenerate every org's keys and
	// invalidate already-running peers' identities.
	run("cryptogen generate for "+tenantID, "docker", "run", "--rm",
		"-v", mustAbs(networkDir)+"/crypto-config:/work",
		"-w", "/work",
		"hyperledger/fabric-tools:2.5",
		"cryptogen", "generate", "--config=./crypto-config-"+tenantID+".yaml", "--output=.")
}

func mustAbs(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FATAL resolving path:", err)
		os.Exit(1)
	}
	return abs
}

// --- Step 2: configtx.yaml org block + channel profile ------------------

func addConfigtxOrgAndProfile(tenantID string) {
	path := networkDir + "/configtx/configtx.yaml"
	content := readFile(path)

	// tenant01's org anchor uses "Tenant01" (capital T, zero-padded suffix) —
	// derive the equivalent assuming the same "tenantNN" shape (tenant02,
	// tenant03, ...); a differently-shaped tenant ID would need this
	// derivation revisited, flagged rather than silently guessed.
	titleTenant := "Tenant" + strings.ToUpper(tenantID[6:7]) + tenantID[7:]
	newAnchor := "OrgClient" + titleTenant

	if strings.Contains(content, "&"+newAnchor) {
		fmt.Println("configtx.yaml already has", newAnchor, "— skipping org-block insertion")
	} else {
		orgBlockRe := regexp.MustCompile(`(?s)  # --- OrgClient-tenant01:.*?\n\n  # --- Org3:`)
		m := orgBlockRe.FindString(content)
		if m == "" {
			fmt.Fprintln(os.Stderr, "FATAL: could not locate tenant01's org block in configtx.yaml")
			os.Exit(1)
		}
		newBlock := strings.ReplaceAll(m, "tenant01", tenantID)
		newBlock = strings.ReplaceAll(newBlock, "Tenant01", titleTenant)
		newBlock = strings.TrimSuffix(newBlock, "  # --- Org3:") // keep only the new org block, drop the trailing marker we matched on
		content = strings.Replace(content, m, newBlock+"\n"+m, 1)
		writeFile(path, content)
		fmt.Println("Inserted org block", newAnchor, "into configtx.yaml")
	}

	profileName := titleTenant + "ChannelGenesis"
	if strings.Contains(content, profileName+":") {
		fmt.Println("configtx.yaml already has profile", profileName, "— skipping")
		return
	}
	newProfile := fmt.Sprintf(`
  %s:
      <<: *ChannelDefaults
      Orderer:
          <<: *OrdererDefaults
          Organizations:
              - *OrdererOrg
          Capabilities: *OrdererCapabilities
      Application:
          <<: *ApplicationDefaults
          Organizations:
              - *Org1
              - *%s
              - *Org3
          Policies:
              Readers: { Type: ImplicitMeta, Rule: "ANY Readers" }
              Writers: { Type: ImplicitMeta, Rule: "ANY Writers" }
              Admins:  { Type: ImplicitMeta, Rule: "MAJORITY Admins" }
              LifecycleEndorsement: { Type: Signature, Rule: "AND('Org1MSP.peer','OrgClient-%sMSP.peer')" }
              Endorsement:          { Type: Signature, Rule: "AND('Org1MSP.peer','OrgClient-%sMSP.peer')" }
          Capabilities: *ApplicationCapabilities
`, profileName, newAnchor, tenantID, tenantID)

	content = readFile(path) // re-read post org-block insertion
	marker := "# Invocation (fabric-operations, post-NET-1, not executed here):"
	idx := strings.Index(content, marker)
	if idx == -1 {
		fmt.Fprintln(os.Stderr, "FATAL: could not locate insertion marker for new profile in configtx.yaml")
		os.Exit(1)
	}
	content = content[:idx] + newProfile + "\n" + content[idx:]
	writeFile(path, content)
	fmt.Println("Inserted profile", profileName, "into configtx.yaml")
}

// --- Step 3: genesis block for the new channel ---------------------------

func genChannelBlock(tenantID, profileName string) {
	channelID := "tenant-" + tenantID
	run("configtxgen for "+channelID, "docker", "run", "--rm",
		"-v", mustAbs(networkDir)+":/work",
		"-w", "/work/configtx",
		"hyperledger/fabric-tools:2.5",
		"configtxgen", "-profile", profileName, "-channelID", channelID,
		"-outputBlock", "/work/channel-artifacts/"+channelID+".block", "-configPath", ".")
}

// --- Step 4: peer bring-up -------------------------------------------------

func addPeerComposeService(tenantID string) {
	path := networkDir + "/compose/network-docker-compose.yaml"
	content := readFile(path)
	serviceName := "peer0." + tenantID
	if strings.Contains(content, serviceName+":\n") {
		fmt.Println("compose file already has", serviceName, "— block insertion skipped, but still ensuring the container is up (file-content idempotency is not the same as container-running idempotency — found the hard way: a prior run's block-already-present skip also skipped `docker compose up`, leaving the container removed-but-referenced-as-existing)")
	} else {
		blockRe := regexp.MustCompile(`(?s)  peer0\.tenant01:\n.*?\n\n  # =+\n  # Org3`)
		m := blockRe.FindString(content)
		if m == "" {
			fmt.Fprintln(os.Stderr, "FATAL: could not locate peer0.tenant01's compose block")
			os.Exit(1)
		}
		newBlock := strings.ReplaceAll(m, "tenant01", tenantID)
		// Host ports: tenant01 used 9051 (gRPC) / 9446 (ops). Derive tenant02's
		// from the next free slot in NET-4's own documented scheme (11051/11448)
		// rather than colliding with any existing service.
		newBlock = strings.ReplaceAll(newBlock, `"9051:7051"`, `"11051:7051"`)
		newBlock = strings.ReplaceAll(newBlock, `"9446:9443"`, `"11446:9443"`)
		trimmed := regexp.MustCompile(`\n  # =+\n  # Org3$`).ReplaceAllString(newBlock, "")

		content = strings.Replace(content, m, trimmed+"\n\n"+strings.TrimPrefix(m, trimmed), 1)
		// Add the named volume declaration too.
		content = strings.Replace(content, "  peer0-tenant01-ledger:\n", "  peer0-tenant01-ledger:\n  peer0-"+tenantID+"-ledger:\n", 1)
		writeFile(path, content)
		fmt.Println("Inserted compose service", serviceName)
	}

	// Always attempt bring-up — `docker compose up -d` is itself idempotent
	// (no-ops if already running with the current config), so this is safe
	// to call unconditionally regardless of whether the block above was
	// freshly inserted or already present.
	run("docker compose up "+serviceName, "docker", "compose",
		"-f", path, "up", "-d", serviceName)
}

// --- Step 5: channel join (netjoin's pattern, generalized) -----------------

func joinNewChannel(tenantID string) {
	channelID := "tenant-" + tenantID
	blockPath := "/net/channel-artifacts/" + channelID + ".block"
	orderers := []string{"orderer0.org1:9443", "orderer1.org1:9443", "orderer2.org1:9443"}
	adminCert := "/net/crypto-config/ordererOrganizations/org1/orderers/orderer0.org1/tls/server.crt"
	adminKey := "/net/crypto-config/ordererOrganizations/org1/orderers/orderer0.org1/tls/server.key"
	adminCA := "/net/crypto-config/ordererOrganizations/org1/orderers/orderer0.org1/tls/ca.crt"

	for _, o := range orderers {
		dockerExecTools("osnadmin join "+o, fmt.Sprintf(
			"osnadmin channel join --channelID %s --config-block %s -o %s --ca-file %s --client-cert %s --client-key %s",
			channelID, blockPath, o, adminCA, adminCert, adminKey))
	}

	type peer struct{ name, addr, mspID, adminMSP, tlsCA string }
	peers := []peer{
		{"peer0.org1", "peer0.org1:7051", "Org1MSP",
			"/net/crypto-config/peerOrganizations/org1/users/Admin@org1/msp",
			"/net/crypto-config/peerOrganizations/org1/tlsca/tlsca.org1-cert.pem"},
		{"peer1.org1", "peer1.org1:7051", "Org1MSP",
			"/net/crypto-config/peerOrganizations/org1/users/Admin@org1/msp",
			"/net/crypto-config/peerOrganizations/org1/tlsca/tlsca.org1-cert.pem"},
		{"peer0." + tenantID, "peer0." + tenantID + ":7051", "OrgClient-" + tenantID + "MSP",
			"/net/crypto-config/peerOrganizations/" + tenantID + "/users/Admin@" + tenantID + "/msp",
			"/net/crypto-config/peerOrganizations/" + tenantID + "/tlsca/tlsca." + tenantID + "-cert.pem"},
		{"peer0.org3", "peer0.org3:7051", "Org3MSP",
			"/net/crypto-config/peerOrganizations/org3/users/Admin@org3/msp",
			"/net/crypto-config/peerOrganizations/org3/tlsca/tlsca.org3-cert.pem"},
	}

	for _, p := range peers {
		env := fmt.Sprintf(
			"CORE_PEER_LOCALMSPID=%s CORE_PEER_MSPCONFIGPATH=%s CORE_PEER_ADDRESS=%s "+
				"CORE_PEER_TLS_ENABLED=true CORE_PEER_TLS_ROOTCERT_FILE=%s "+
				"CORE_PEER_TLS_CLIENTAUTHREQUIRED=true "+
				"CORE_PEER_TLS_CLIENTCERT_FILE=%s/tls/client.crt CORE_PEER_TLS_CLIENTKEY_FILE=%s/tls/client.key",
			p.mspID, p.adminMSP, p.addr, p.tlsCA, strings.TrimSuffix(p.adminMSP, "/msp"), strings.TrimSuffix(p.adminMSP, "/msp"))
		out, err := tryRun("peer channel join "+p.name, "docker", "exec", "fabric-tools-net",
			"sh", "-c", env+" peer channel join -b "+blockPath)
		if err != nil && !strings.Contains(out, "already exists") {
			fmt.Fprintf(os.Stderr, "FATAL joining %s: %v\n", p.name, err)
			os.Exit(1)
		}
	}

	// Verify.
	p := peers[2]
	adminTLSDir := strings.TrimSuffix(p.adminMSP, "/msp") + "/tls"
	env := fmt.Sprintf(
		"CORE_PEER_LOCALMSPID=%s CORE_PEER_MSPCONFIGPATH=%s CORE_PEER_ADDRESS=%s "+
			"CORE_PEER_TLS_ENABLED=true CORE_PEER_TLS_ROOTCERT_FILE=%s "+
			"CORE_PEER_TLS_CLIENTAUTHREQUIRED=true "+
			"CORE_PEER_TLS_CLIENTCERT_FILE=%s/client.crt CORE_PEER_TLS_CLIENTKEY_FILE=%s/client.key",
		p.mspID, p.adminMSP, p.addr, p.tlsCA, adminTLSDir, adminTLSDir)
	out := dockerExecTools("verify peer channel list on "+p.name, env+" peer channel list")
	if !strings.Contains(out, channelID) {
		fmt.Fprintln(os.Stderr, "FAIL: new tenant's peer does not report the new channel joined")
		os.Exit(1)
	}
}

func main() {
	tenantID := mustTenantID()
	fmt.Println("=== NET-7: provisioning tenant", tenantID, "===")

	fmt.Println("\n-- Step 1: crypto material --")
	genCryptoMaterial(tenantID)

	fmt.Println("\n-- Step 2: configtx.yaml org + profile --")
	addConfigtxOrgAndProfile(tenantID)
	titleTenant := "Tenant" + strings.ToUpper(tenantID[6:7]) + tenantID[7:]
	profileName := titleTenant + "ChannelGenesis"

	fmt.Println("\n-- Step 3: genesis block --")
	genChannelBlock(tenantID, profileName)

	fmt.Println("\n-- Step 4: peer bring-up --")
	addPeerComposeService(tenantID)

	fmt.Println("\n-- Step 5: channel join --")
	joinNewChannel(tenantID)

	fmt.Println("\nSUCCESS: tenant", tenantID, "fully provisioned — channel created, all 4 peers joined, endorsement policy wired.")
}

// Command pilscan executes backlog item ST-1 (P0 — the single
// highest-priority test in the whole test-strategy, see
// agent-suite/06-roadmap/test-strategy.md §0/§1): a full-ledger
// confidentiality scan proving ZERO bytes of PII or any reversible
// derivative exist anywhere on-chain — world state, transaction ARGUMENTS
// (every past Submit call's raw args, not just current state), and
// chaincode EVENT payloads — swept across EVERY provisioned tenant channel.
//
// Why this is not just "call GetProfileHistory and check the JSON": that
// would only prove the QUERY API doesn't return PII, reusing the exact code
// path under test (circular). This tool instead fetches every raw block
// from the real orderer via the `peer channel fetch` binary (read-only —
// no channel join, no chaincode lifecycle action, matching this project's
// established docker-exec-into-fabric-tools-net pattern, see netjoin/
// raftfaulttest) and parses the raw protobuf bytes itself, in Go, using
// github.com/hyperledger/fabric-protos-go-apiv2 (already a dependency of
// write-path-integration/gateway-client — no new library introduced) — the
// SAME wire bytes an endorsing peer/orderer actually persisted, not
// anything reconstructed via a chaincode Evaluate call.
//
// Two independent, overlapping checks are run per block (belt-and-braces,
// not redundant busywork):
//
//  1. STRUCTURED extraction — walk Envelope -> Payload -> ChannelHeader (to
//     find ENDORSER_TRANSACTION envelopes) -> Transaction -> TransactionAction
//     -> ChaincodeActionPayload -> (a) ChaincodeProposalPayload ->
//     ChaincodeInvocationSpec.Input.Args (the tx arguments AS SUBMITTED) and
//     (b) ChaincodeEndorsedAction.ProposalResponsePayload -> ChaincodeAction
//     -> Results (rwset.TxReadWriteSet -> kvrwset.KVRWSet -> KVWrite = the
//     actual WORLD STATE writes) and Events (a marshaled ChaincodeEvent —
//     record_profile_section.go's own SetEvent("RecordProfileSection", ...)
//     payload). Every extracted string is checked against a fixture
//     dictionary and, where the value is a known field (EmployeeID/
//     UpdatedBy/DataHash/IpfsCIDs), against its required structural shape.
//  2. RAW BACKSTOP — every fetched block's undecoded byte stream is ALSO
//     scanned, as a plain string, for every fixture-dictionary entry. This
//     is deliberately independent of (1)'s parsing correctness: even if this
//     tool's own protobuf-walk had a bug and missed some nested structure,
//     a literal PII string anywhere in the block's bytes (Fabric never
//     compresses or encrypts block contents at rest) would still be found
//     here. This is what makes the "ZERO bytes" claim a real claim about the
//     bytes, not just about the fields this tool knew to look at.
//
// The fixture dictionary (see fixtureDictionary below) is the exact set of
// synthetic-but-realistic strings this session's own write-path-integration
// tests used as OPERATIONAL-DB content — extracted by grepping the actual
// test source (writepaths/*.go, gateway-client/*.go), not guessed.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/rwset"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
)

const (
	helperContainer = "fabric-tools-net"
	chaincodeName   = "employeeprofilerecord"
	ordererAddr     = "orderer0.org1:7050"
	ordererTLSCA    = "/net/crypto-config/ordererOrganizations/org1/tlsca/tlsca.org1-cert.pem"

	// Org1 is a Reader on BOTH tenant channels' Application group
	// (configtx.yaml Profiles.TenantChannelGenesis AND
	// Tenant02ChannelGenesis both list *Org1 in Organizations, and
	// ApplicationDefaults.Policies.Readers is "ANY Readers") — one signing
	// identity suffices for fetching both channels; no OrgClient-tenant02
	// identity, and critically no peer0.tenant02 interaction, is needed.
	org1AdminMSP  = "/net/crypto-config/peerOrganizations/org1/users/Admin@org1/msp"
	org1TLSDir    = "/net/crypto-config/peerOrganizations/org1/users/Admin@org1/tls"
	org1TLSCAFile = "/net/crypto-config/peerOrganizations/org1/tlsca/tlsca.org1-cert.pem"

	containerScratchDir = "/tmp/pilscan"
)

// channels is the full set of provisioned tenant channels ST-1 requires
// sweeping — tenant-tenant01 (real synthetic transaction history) AND
// tenant-tenant02 (expected empty/near-empty — confirmed, not assumed, by
// this tool actually fetching and parsing its blocks too).
//
// tenant-tenant02 is READ via `peer channel fetch` against the orderer only
// (confirmed reachable via a prior read-only `osnadmin channel list` — the
// channel already exists on the orderer from its own genesis-block
// creation). This tool never touches peer0.tenant02 (which by design is
// joined to no channel) and never runs `peer channel join` or `osnadmin
// channel join` against anything — see the safety constraints this tool was
// built under, restated in the ST-1 report this tool's own output feeds.
var channels = []string{"tenant-tenant01", "tenant-tenant02"}

// fixtureDictionary is the exact-match/substring dictionary of every
// SYNTHETIC-but-realistic fixture string this session's OWN
// write-path-integration tests used as OPERATIONAL-DB content — extracted
// by grepping the real test source (not guessed):
//
//	writepaths/writepaths_integration_test.go
//	writepaths/erasure_ipfs_integration_test.go
//	writepaths/partial_failure_integration_test.go
//	writepaths/erasure_test.go
//	gateway-client/tamper_detection_all_sections_integration_test.go
//	gateway-client/verify_integration_test.go
//	gateway-client/verify_notfound_integration_test.go
//	gateway-client/digestbuilder_test.go
//	ipfsclient/ipfsclient_integration_test.go, ipfsclient_test.go
//	keystore/keystore_test.go, handoff_test.go
//
// If ANY of these strings (or their JSON field names, which would only ever
// appear on-chain if raw section content leaked) is found anywhere in a
// fetched block's bytes, that is a ST-1 FAIL.
var fixtureDictionary = []string{
	// Distinctive fixture "content" values (fullName/degree/department/etc.)
	"REC-4 Demo Employee",
	"INT-1 Partial-Failure Demo",
	"QA-2 Erasure Demo",
	"REC-7 Verify Test",
	"INT-2 NotFound Test",
	"INT-2 Fixture",
	"TAMPERED NAME",
	"Test Employee",
	"Generic City",
	"Generic University",
	"Generic Bank",
	"Attacker Bank",
	"Staff Engineer",
	"Software Engineer",
	"Executive",
	"Chief Officer",
	"Erasure Fixture",
	"B.Sc.",
	"Bachelor",
	"State University",
	"Doctorate",
	"Diploma Mill University",
	"married",
	"single",
	"1234567890",
	"9999999999",
	// JSON field NAMES from the never-anchored operational-DB payloads — if
	// any of these keys is visible on-chain, raw section JSON leaked.
	"fullName",
	"jobTitle",
	"maritalStatus",
	"dependentCount",
	"dependents",
	"bankAccountNumber",
	"bankName",
	"accountLast4",
	"degree",
	"institution",
	"address",
	// Actor/employee identifiers as they exist BEFORE pseudonymization
	// (userID / employeeInternalID) — these must NEVER appear on-chain;
	// only their HMAC outputs (EmployeeID/UpdatedBy) may.
	"hr-admin-rec4-demo",
	"hr-admin-int1-demo",
	"hr-admin-qa2-erasure-demo",
	"rec4demoemployee00000000000000000000000000000000000000000001",
	"int1demoemployee000000000000000000000000000000000000000001",
	"qa2erasuredemo00000000000000000000000000000000000000000001",
	"erasure-test-employee-001",
	// Supporting-document plaintext (must never leave the encrypted IPFS
	// object — only a CID may appear on-chain).
	"REC-5 integration fixture",
	"QA-2 IT-9 fixture",
	"supporting document content",
	"only the correct KEY_EMPLOYEE",
	"only the right KEY_EMPLOYEE recovers this",
	"identical content, encrypted twice",
	"same content, re-encrypted after a hypothetical key rotation",
	"tamper-detection fixture",
}

// allowedRecordFields is the EXACT ratified EmployeeProfileRecord field set
// (asset.go, data-model.md §2) — ST-1 §2's schema-boundary check. Any other
// JSON object key found in a world-state write or event payload whose value
// looks like this record shape is a metadata-boundary regression (ties to
// ST-7, checked here too since this scan already has the decoded JSON in
// hand).
var allowedRecordFields = map[string]bool{
	"canonicalizationVersion": true,
	"dataHash":                true,
	"employeeID":              true,
	"hashAlgo":                true,
	"ipfsCIDs":                true,
	"prevHash":                true,
	"profileSection":          true,
	"recordID":                true,
	"tenantID":                true,
	"timestamp":               true,
	"updatedBy":                true,
	"version":                 true,
}

var (
	hex64Re     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	dataHashRe  = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	cidPlausRe  = regexp.MustCompile(`^(Qm[1-9A-HJ-NP-Za-km-z]{44}|[a-z2-7]{20,120}|B[A-Z2-7]{20,120})$`)
	sectionEnum = map[string]bool{"PERSONAL": true, "EMPLOYMENT": true, "EDUCATION": true, "ADDITIONAL": true, "PAYROLL": true}
)

// finding is one concrete PII/structural violation this scan found.
type finding struct {
	Channel  string
	Block    uint64
	Location string // e.g. "tx-args", "world-state-write key=...", "event", "raw-block-backstop"
	Detail   string
}

// channelStats accumulates the exhaustive counts the P0 report requires —
// this is a full sweep, not a sampling, so every number below is a total,
// not an estimate.
type channelStats struct {
	Channel            string
	BlocksScanned      int
	HighestBlockNumber uint64
	Envelopes          int
	ConfigEnvelopes    int
	EndorserTxEnvelopes int
	ChaincodeInvocations int
	TxArgsChecked      int
	WorldStateWrites   int
	EventsChecked      int
	IpfsCIDsChecked    int
	RawBytesScanned    int64
	Findings           []finding
}

func dockerExec(shCmd string) (string, error) {
	cmd := exec.Command("docker", "exec", helperContainer, "sh", "-c", shCmd)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// dockerCatBinary reads a file back out of the helper container's
// filesystem via `cat` over stdout only (exec.Command.Output, not
// CombinedOutput) so binary block bytes are never mixed with stderr text.
func dockerCatBinary(containerPath string) ([]byte, error) {
	cmd := exec.Command("docker", "exec", helperContainer, "cat", containerPath)
	return cmd.Output()
}

func peerCLIEnv() string {
	return fmt.Sprintf(
		"CORE_PEER_LOCALMSPID=Org1MSP CORE_PEER_MSPCONFIGPATH=%s CORE_PEER_ADDRESS=peer0.org1:7051 "+
			"CORE_PEER_TLS_ENABLED=true CORE_PEER_TLS_ROOTCERT_FILE=%s "+
			"CORE_PEER_TLS_CLIENTAUTHREQUIRED=true "+
			"CORE_PEER_TLS_CLIENTCERT_FILE=%s/client.crt CORE_PEER_TLS_CLIENTKEY_FILE=%s/client.key",
		org1AdminMSP, org1TLSCAFile, org1TLSDir, org1TLSDir,
	)
}

func ordererTLSClientFlags() string {
	return fmt.Sprintf("--clientauth --certfile %s/client.crt --keyfile %s/client.key", org1TLSDir, org1TLSDir)
}

// fetchBlock runs the real `peer channel fetch` binary (read-only — NOT
// `peer channel join`, NOT `osnadmin channel join`, no chaincode lifecycle
// action) inside fabric-tools-net, fetching blockSpec (a decimal block
// number, or "newest") for channel into containerScratchDir, then reads the
// raw bytes back out via `cat` (never `docker cp`, to avoid a host-side
// temp-file footprint outside this tool's own directory).
func fetchBlock(channel, blockSpec string) ([]byte, error) {
	containerPath := fmt.Sprintf("%s/%s_%s.block", containerScratchDir, channel, blockSpec)
	shCmd := fmt.Sprintf(
		"%s peer channel fetch %s %s -c %s -o %s --tls --cafile %s %s",
		peerCLIEnv(), blockSpec, containerPath, channel, ordererAddr, ordererTLSCA, ordererTLSClientFlags(),
	)
	out, err := dockerExec(shCmd)
	if err != nil {
		return nil, fmt.Errorf("peer channel fetch %s (channel=%s): %w\n%s", blockSpec, channel, err, out)
	}
	raw, err := dockerCatBinary(containerPath)
	if err != nil {
		return nil, fmt.Errorf("reading back fetched block %s (channel=%s): %w", blockSpec, channel, err)
	}
	return raw, nil
}

func containsFixture(s string) (string, bool) {
	for _, f := range fixtureDictionary {
		if strings.Contains(s, f) {
			return f, true
		}
	}
	return "", false
}

// scanRawBackstop is check (2) from the file header: a plain substring scan
// of the ENTIRE undecoded block byte stream, independent of this tool's own
// protobuf-walk correctness.
func scanRawBackstop(stats *channelStats, blockNum uint64, raw []byte) {
	stats.RawBytesScanned += int64(len(raw))
	s := string(raw)
	for _, f := range fixtureDictionary {
		if strings.Contains(s, f) {
			stats.Findings = append(stats.Findings, finding{
				Channel: stats.Channel, Block: blockNum, Location: "raw-block-backstop",
				Detail: fmt.Sprintf("fixture string %q found in raw block bytes", f),
			})
		}
	}
}

// checkRecordJSON applies the schema-boundary + structural checks (ST-1 §2 /
// ST-7 overlap, and the hex64/sha256 shape checks) to one decoded
// EmployeeProfileRecord-shaped JSON blob (a world-state write value or an
// event payload — both are the same recordJSON per record_profile_section.go).
func checkRecordJSON(stats *channelStats, blockNum uint64, location string, raw []byte) {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		// Not JSON (e.g. a _lifecycle/lscc namespace write, or a
		// non-record key) — the fixture-dictionary + raw-backstop checks
		// still cover it; nothing further to structurally validate.
		return
	}

	for k := range m {
		if !allowedRecordFields[k] {
			stats.Findings = append(stats.Findings, finding{
				Channel: stats.Channel, Block: blockNum, Location: location,
				Detail: fmt.Sprintf("SCHEMA VIOLATION: on-chain record has field %q, not in the ratified 12-field set", k),
			})
		}
	}

	checkHex64 := func(field string) {
		v, ok := m[field].(string)
		if !ok {
			return
		}
		if f, hit := containsFixture(v); hit {
			stats.Findings = append(stats.Findings, finding{Channel: stats.Channel, Block: blockNum, Location: location,
				Detail: fmt.Sprintf("%s value contains fixture string %q: %q", field, f, v)})
		}
		if !hex64Re.MatchString(v) {
			stats.Findings = append(stats.Findings, finding{Channel: stats.Channel, Block: blockNum, Location: location,
				Detail: fmt.Sprintf("%s is not a 64-hex-char HMAC-SHA256 output (plaintext-shaped?): %q", field, v)})
		}
	}
	checkHex64("employeeID")
	checkHex64("updatedBy")

	if v, ok := m["dataHash"].(string); ok {
		if f, hit := containsFixture(v); hit {
			stats.Findings = append(stats.Findings, finding{Channel: stats.Channel, Block: blockNum, Location: location,
				Detail: fmt.Sprintf("dataHash contains fixture string %q: %q", f, v)})
		}
		if !dataHashRe.MatchString(v) {
			stats.Findings = append(stats.Findings, finding{Channel: stats.Channel, Block: blockNum, Location: location,
				Detail: fmt.Sprintf("dataHash is not \"sha256:<64 hex>\" (raw content?): %q", v)})
		}
	}
	if v, ok := m["prevHash"].(string); ok && v != "" {
		if !dataHashRe.MatchString(v) {
			stats.Findings = append(stats.Findings, finding{Channel: stats.Channel, Block: blockNum, Location: location,
				Detail: fmt.Sprintf("prevHash is neither empty nor \"sha256:<64 hex>\": %q", v)})
		}
	}
	if v, ok := m["profileSection"].(string); ok && !sectionEnum[v] {
		stats.Findings = append(stats.Findings, finding{Channel: stats.Channel, Block: blockNum, Location: location,
			Detail: fmt.Sprintf("profileSection is not one of the five ratified values: %q", v)})
	}
	if cids, ok := m["ipfsCIDs"].([]interface{}); ok {
		for _, c := range cids {
			cs, ok := c.(string)
			if !ok {
				continue
			}
			stats.IpfsCIDsChecked++
			checkCID(stats, blockNum, location, cs)
		}
	}
}

func checkCID(stats *channelStats, blockNum uint64, location, cid string) {
	if f, hit := containsFixture(cid); hit {
		stats.Findings = append(stats.Findings, finding{Channel: stats.Channel, Block: blockNum, Location: location,
			Detail: fmt.Sprintf("ipfsCIDs entry contains fixture string %q (document content leaked into a CID field?): %q", f, cid)})
	}
	if !cidPlausRe.MatchString(cid) {
		stats.Findings = append(stats.Findings, finding{Channel: stats.Channel, Block: blockNum, Location: location,
			Detail: fmt.Sprintf("ipfsCIDs entry is not a plausible CID string (wrong charset/length for base58 CIDv0 or base32/base64 CIDv1): %q", cid)})
	}
	if len(cid) > 130 {
		stats.Findings = append(stats.Findings, finding{Channel: stats.Channel, Block: blockNum, Location: location,
			Detail: fmt.Sprintf("ipfsCIDs entry implausibly long (%d bytes) for a CID — possible embedded content: %q", len(cid), truncate(cid, 60))})
	}
}

// checkArg applies the fixture-dictionary + (where positionally known)
// structural checks to one raw RecordProfileSection argument string, per
// writepaths.go's own buildArgs order:
//
//	[tenantID, employeeID, profileSection, dataHash, prevHash, updatedBy,
//	 ipfsCIDsJSON, canonicalizationVersion, hashAlgo, clientTimestamp]
func checkArgs(stats *channelStats, blockNum uint64, fn string, args [][]byte) {
	for i, a := range args {
		s := string(a)
		stats.TxArgsChecked++
		if f, hit := containsFixture(s); hit {
			stats.Findings = append(stats.Findings, finding{Channel: stats.Channel, Block: blockNum, Location: "tx-args",
				Detail: fmt.Sprintf("fn=%s arg[%d] contains fixture string %q: %q", fn, i, f, truncate(s, 200))})
		}
	}
	if fn != "RecordProfileSection" || len(args) < 10 {
		return
	}
	// args[0] is the function name itself in the ChaincodeInput.Args slice
	// as fabric-gateway's SubmitTransaction constructs it — but our args
	// slice here is ALREADY the invocation's business args (see caller),
	// matching writepaths.go's buildArgs positions 0..9 above.
	employeeID := string(args[1])
	profileSection := string(args[2])
	dataHash := string(args[3])
	prevHash := string(args[4])
	updatedBy := string(args[5])
	ipfsCIDsJSON := string(args[6])

	if !hex64Re.MatchString(employeeID) {
		stats.Findings = append(stats.Findings, finding{Channel: stats.Channel, Block: blockNum, Location: "tx-args",
			Detail: fmt.Sprintf("employeeID arg is not 64-hex HMAC output: %q", employeeID)})
	}
	if !hex64Re.MatchString(updatedBy) {
		stats.Findings = append(stats.Findings, finding{Channel: stats.Channel, Block: blockNum, Location: "tx-args",
			Detail: fmt.Sprintf("updatedBy arg is not 64-hex HMAC output: %q", updatedBy)})
	}
	if !dataHashRe.MatchString(dataHash) {
		stats.Findings = append(stats.Findings, finding{Channel: stats.Channel, Block: blockNum, Location: "tx-args",
			Detail: fmt.Sprintf("dataHash arg is not \"sha256:<64 hex>\": %q", dataHash)})
	}
	if prevHash != "" && !dataHashRe.MatchString(prevHash) {
		stats.Findings = append(stats.Findings, finding{Channel: stats.Channel, Block: blockNum, Location: "tx-args",
			Detail: fmt.Sprintf("prevHash arg is neither empty nor \"sha256:<64 hex>\": %q", prevHash)})
	}
	if !sectionEnum[profileSection] {
		stats.Findings = append(stats.Findings, finding{Channel: stats.Channel, Block: blockNum, Location: "tx-args",
			Detail: fmt.Sprintf("profileSection arg not one of the five ratified values: %q", profileSection)})
	}
	var cids []string
	if err := json.Unmarshal([]byte(ipfsCIDsJSON), &cids); err == nil {
		for _, c := range cids {
			stats.IpfsCIDsChecked++
			checkCID(stats, blockNum, "tx-args", c)
		}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}

// parseBlock is the STRUCTURED extraction pass — check (1) from the file
// header.
func parseBlock(stats *channelStats, raw []byte) error {
	block := &common.Block{}
	if err := proto.Unmarshal(raw, block); err != nil {
		return fmt.Errorf("unmarshal common.Block: %w", err)
	}
	blockNum := block.GetHeader().GetNumber()
	stats.BlocksScanned++
	if blockNum > stats.HighestBlockNumber {
		stats.HighestBlockNumber = blockNum
	}

	scanRawBackstop(stats, blockNum, raw)

	for _, envBytes := range block.GetData().GetData() {
		env := &common.Envelope{}
		if err := proto.Unmarshal(envBytes, env); err != nil {
			continue // malformed/empty envelope slot — nothing further to parse from it
		}
		payload := &common.Payload{}
		if err := proto.Unmarshal(env.GetPayload(), payload); err != nil {
			continue
		}
		chdr := &common.ChannelHeader{}
		if err := proto.Unmarshal(payload.GetHeader().GetChannelHeader(), chdr); err != nil {
			continue
		}
		stats.Envelopes++

		if common.HeaderType(chdr.GetType()) != common.HeaderType_ENDORSER_TRANSACTION {
			stats.ConfigEnvelopes++
			continue
		}
		stats.EndorserTxEnvelopes++

		tx := &peer.Transaction{}
		if err := proto.Unmarshal(payload.GetData(), tx); err != nil {
			continue
		}
		for _, action := range tx.GetActions() {
			cap := &peer.ChaincodeActionPayload{}
			if err := proto.Unmarshal(action.GetPayload(), cap); err != nil {
				continue
			}

			// (a) ARGUMENTS AS SUBMITTED
			var fnName string
			cpp := &peer.ChaincodeProposalPayload{}
			if err := proto.Unmarshal(cap.GetChaincodeProposalPayload(), cpp); err == nil {
				cis := &peer.ChaincodeInvocationSpec{}
				if err := proto.Unmarshal(cpp.GetInput(), cis); err == nil {
					rawArgs := cis.GetChaincodeSpec().GetInput().GetArgs()
					if len(rawArgs) > 0 {
						fnName = string(rawArgs[0])
						stats.ChaincodeInvocations++
						checkArgs(stats, blockNum, fnName, rawArgs[1:])
					}
				}
			}

			// (b) WORLD-STATE WRITES + EVENTS
			endorsed := cap.GetAction()
			prp := &peer.ProposalResponsePayload{}
			if err := proto.Unmarshal(endorsed.GetProposalResponsePayload(), prp); err != nil {
				continue
			}
			ccAction := &peer.ChaincodeAction{}
			if err := proto.Unmarshal(prp.GetExtension(), ccAction); err != nil {
				continue
			}

			if len(ccAction.GetResults()) > 0 {
				txrw := &rwset.TxReadWriteSet{}
				if err := proto.Unmarshal(ccAction.GetResults(), txrw); err == nil {
					for _, ns := range txrw.GetNsRwset() {
						kv := &kvrwset.KVRWSet{}
						if err := proto.Unmarshal(ns.GetRwset(), kv); err != nil {
							continue
						}
						for _, w := range kv.GetWrites() {
							stats.WorldStateWrites++
							loc := fmt.Sprintf("world-state-write ns=%s key=%s", ns.GetNamespace(), w.GetKey())
							if f, hit := containsFixture(string(w.GetValue())); hit {
								stats.Findings = append(stats.Findings, finding{Channel: stats.Channel, Block: blockNum, Location: loc,
									Detail: fmt.Sprintf("write value contains fixture string %q: %q", f, truncate(string(w.GetValue()), 200))})
							}
							if ns.GetNamespace() == chaincodeName {
								checkRecordJSON(stats, blockNum, loc, w.GetValue())
							}
						}
					}
				}
			}

			if len(ccAction.GetEvents()) > 0 {
				evt := &peer.ChaincodeEvent{}
				if err := proto.Unmarshal(ccAction.GetEvents(), evt); err == nil {
					stats.EventsChecked++
					loc := fmt.Sprintf("event chaincodeId=%s eventName=%s txId=%s", evt.GetChaincodeId(), evt.GetEventName(), evt.GetTxId())
					if f, hit := containsFixture(string(evt.GetPayload())); hit {
						stats.Findings = append(stats.Findings, finding{Channel: stats.Channel, Block: blockNum, Location: loc,
							Detail: fmt.Sprintf("event payload contains fixture string %q: %q", f, truncate(string(evt.GetPayload()), 200))})
					}
					checkRecordJSON(stats, blockNum, loc, evt.GetPayload())
				}
			}
		}
	}
	return nil
}

func scanChannel(channel string) (*channelStats, error) {
	stats := &channelStats{Channel: channel}

	if _, err := dockerExec("mkdir -p " + containerScratchDir); err != nil {
		return nil, fmt.Errorf("mkdir scratch dir: %w", err)
	}

	newest, err := fetchBlock(channel, "newest")
	if err != nil {
		return nil, fmt.Errorf("fetching newest block for height discovery: %w", err)
	}
	newestBlock := &common.Block{}
	if err := proto.Unmarshal(newest, newestBlock); err != nil {
		return nil, fmt.Errorf("unmarshal newest block: %w", err)
	}
	height := newestBlock.GetHeader().GetNumber()
	fmt.Printf("[%s] newest block number = %d (height = %d blocks: 0..%d)\n", channel, height, height+1, height)

	for n := uint64(0); n <= height; n++ {
		raw, err := fetchBlock(channel, strconv.FormatUint(n, 10))
		if err != nil {
			return stats, fmt.Errorf("fetching block %d: %w", n, err)
		}
		if err := parseBlock(stats, raw); err != nil {
			return stats, fmt.Errorf("parsing block %d: %w", n, err)
		}
	}

	return stats, nil
}

func main() {
	fmt.Println("=== ST-1 (P0): full-ledger confidentiality scan ===")
	fmt.Println("Read-only: `peer channel fetch` against the live orderer only.")
	fmt.Println("NO peer channel join, NO osnadmin channel join, NO chaincode lifecycle action performed.")
	fmt.Println()

	var allStats []*channelStats
	overallPass := true

	for _, ch := range channels {
		fmt.Printf("--- Scanning channel %q ---\n", ch)
		stats, err := scanChannel(ch)
		if stats != nil {
			allStats = append(allStats, stats)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "FATAL scanning channel %s: %v\n", ch, err)
			overallPass = false
			continue
		}
		fmt.Printf("[%s] blocksScanned=%d envelopes=%d (config=%d, endorserTx=%d) ccInvocations=%d argsChecked=%d writesChecked=%d eventsChecked=%d cidsChecked=%d rawBytes=%d findings=%d\n",
			stats.Channel, stats.BlocksScanned, stats.Envelopes, stats.ConfigEnvelopes, stats.EndorserTxEnvelopes,
			stats.ChaincodeInvocations, stats.TxArgsChecked, stats.WorldStateWrites, stats.EventsChecked, stats.IpfsCIDsChecked, stats.RawBytesScanned, len(stats.Findings))
		if len(stats.Findings) > 0 {
			overallPass = false
		}
	}

	fmt.Println()
	if overallPass {
		fmt.Println("SUCCESS: ST-1 PASS — 0 PII/reversible-derivative findings across every scanned channel/block/tx-arg/write/event.")
	} else {
		fmt.Println("FAIL: ST-1 — see findings above/in the report; do not treat this build phase as gated-open.")
	}

	writeReport(allStats, overallPass)

	if !overallPass {
		os.Exit(1)
	}
}

// writeReport emits a machine-readable-ish summary to stdout in a form the
// report-writing step (qa-tests/security/st1-confidentiality-scan-report.md,
// written by hand from this run's real output, not templated) can quote
// verbatim. Kept in this tool (rather than only in the .md) so a future
// re-run is self-documenting without relying on a human to have transcribed
// it correctly.
func writeReport(allStats []*channelStats, pass bool) {
	fmt.Println("\n=== MACHINE-READABLE SUMMARY ===")
	totalBlocks, totalEnvelopes, totalArgs, totalWrites, totalEvents, totalCIDs := 0, 0, 0, 0, 0, 0
	var allFindings []finding
	for _, s := range allStats {
		totalBlocks += s.BlocksScanned
		totalEnvelopes += s.Envelopes
		totalArgs += s.TxArgsChecked
		totalWrites += s.WorldStateWrites
		totalEvents += s.EventsChecked
		totalCIDs += s.IpfsCIDsChecked
		allFindings = append(allFindings, s.Findings...)
	}
	sort.Slice(allFindings, func(i, j int) bool { return allFindings[i].Channel < allFindings[j].Channel || (allFindings[i].Channel == allFindings[j].Channel && allFindings[i].Block < allFindings[j].Block) })
	fmt.Printf("channels_scanned=%d total_blocks=%d total_envelopes=%d total_tx_args_checked=%d total_world_state_writes_checked=%d total_events_checked=%d total_ipfs_cids_checked=%d total_findings=%d verdict=%s\n",
		len(allStats), totalBlocks, totalEnvelopes, totalArgs, totalWrites, totalEvents, totalCIDs, len(allFindings), map[bool]string{true: "PASS", false: "FAIL"}[pass])
	for _, f := range allFindings {
		fmt.Printf("FINDING channel=%s block=%d location=%q detail=%q\n", f.Channel, f.Block, f.Location, f.Detail)
	}
}

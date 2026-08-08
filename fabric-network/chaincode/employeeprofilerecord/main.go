// Command employeeprofilerecord starts the EmployeeProfileRecord chaincode
// (ADR-0020) as a Fabric contract-api-go chaincode process. Contract logic
// lives in the chaincode package so it stays unit-testable without a running
// peer [docs: chaincode4ade.rst#chaincode-sample].
//
// Two start modes, selected by CHAINCODE_SERVER_ADDRESS:
//   - unset: classic Docker-managed lifecycle (peer launches this process).
//   - set: Chaincode-as-a-Service (CCaaS) — this process is a long-running
//     server the peer dials OUT to; the peer never launches it and never
//     needs access to the host's Docker socket. Chosen at CC-5 specifically
//     to avoid mounting /var/run/docker.sock into peer containers (a
//     privilege-escalation-shaped grant the harness correctly refused to
//     apply without explicit confirmation) — CCaaS is Fabric's own
//     documented alternative for exactly this reason, not a workaround.
package main

import (
	"log"
	"os"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"

	"employeeprofilerecord/chaincode"
)

func main() {
	cc, err := contractapi.NewChaincode(&chaincode.SmartContract{})
	if err != nil {
		log.Panicf("error creating employeeprofilerecord chaincode: %v", err)
	}

	address := os.Getenv("CHAINCODE_SERVER_ADDRESS")
	if address == "" {
		if err := cc.Start(); err != nil {
			log.Panicf("error starting employeeprofilerecord chaincode: %v", err)
		}
		return
	}

	server := &shim.ChaincodeServer{
		CCID:    os.Getenv("CHAINCODE_ID"),
		Address: address,
		CC:      cc,
		TLSProps: shim.TLSProperties{
			Disabled: true, // dev-grade prototype network; not a production posture.
		},
	}
	if err := server.Start(); err != nil {
		log.Panicf("error starting employeeprofilerecord chaincode server: %v", err)
	}
}

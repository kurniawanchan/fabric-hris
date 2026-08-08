module integrationbridge

go 1.25.9

require (
	gatewayclient v0.0.0-00010101000000-000000000000
	ipfsclient v0.0.0-00010101000000-000000000000
	keystore v0.0.0-00010101000000-000000000000
	writepaths v0.0.0-00010101000000-000000000000
)

require (
	github.com/cyberphone/json-canonicalization v0.0.0-20241213102144-19d51d7fe467 // indirect
	github.com/hyperledger/fabric-gateway v1.12.0 // indirect
	github.com/hyperledger/fabric-protos-go-apiv2 v0.3.7 // indirect
	github.com/miekg/pkcs11 v1.1.2 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478 // indirect
	google.golang.org/grpc v1.82.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	gatewayclient => ../write-path-integration/gateway-client
	ipfsclient => ../write-path-integration/ipfsclient
	keystore => ../write-path-integration/keystore
	writepaths => ../write-path-integration/writepaths
)

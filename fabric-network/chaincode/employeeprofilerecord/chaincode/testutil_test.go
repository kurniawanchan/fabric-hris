package chaincode

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/golang/protobuf/proto" //nolint:staticcheck // matches the legacy proto API fabric-chaincode-go/pkg/cid itself unmarshals with.
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/stretchr/testify/require"
)

// --- Test identity fixtures -------------------------------------------------
//
// cid.GetMSPID/cid.GetID (pkg/cid) require the stub's GetCreator() to return
// a protobuf-marshaled msp.SerializedIdentity whose IdBytes is a PEM-encoded
// X.509 certificate that actually parses [code: fabric-chaincode-go/pkg/cid/
// cid.go ClientID.init()] — the certificate's CONTENT is irrelevant to every
// check this package makes (only the Mspid field is read), so one self-signed
// certificate is generated once and reused, varying only Mspid, across every
// simulated identity.

var testCertPEM = generateTestCertPEM()

func generateTestCertPEM() []byte {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "employeeprofilerecord-test-fixture"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// serializedIdentity builds the raw creator bytes for a given MSP ID, reusing
// the shared test certificate.
func serializedIdentity(t *testing.T, mspID string) []byte {
	t.Helper()
	sid := &msp.SerializedIdentity{Mspid: mspID, IdBytes: testCertPEM}
	b, err := proto.Marshal(sid)
	require.NoError(t, err)
	return b
}

// --- fake history support ---------------------------------------------------
//
// shimtest.MockStub.GetHistoryForKey is not implemented upstream — it always
// returns an error [code: fabric-chaincode-go/shim/interfaces.go
// MockQueryIteratorInterface TODO: "Once the execute query and history query
// are implemented in MockStub"]. historyMockStub layers a minimal, in-memory,
// deterministic history log on top of MockStub's PutState so
// GetProfileHistory's own logic (mapping + sort) is exercised for real,
// rather than only its error-propagation path.

type historyMockStub struct {
	*shimtest.MockStub
	history map[string][]*queryresult.KeyModification
}

func newHistoryMockStub(name string) *historyMockStub {
	return &historyMockStub{
		MockStub: shimtest.NewMockStub(name, nil),
		history:  make(map[string][]*queryresult.KeyModification),
	}
}

func (s *historyMockStub) PutState(key string, value []byte) error {
	if err := s.MockStub.PutState(key, value); err != nil {
		return err
	}
	ts, err := s.MockStub.GetTxTimestamp()
	if err != nil {
		return err
	}
	valueCopy := append([]byte(nil), value...)
	s.history[key] = append(s.history[key], &queryresult.KeyModification{
		TxId:      s.MockStub.GetTxID(),
		Value:     valueCopy,
		Timestamp: ts,
		IsDelete:  false,
	})
	return nil
}

func (s *historyMockStub) GetHistoryForKey(key string) (shim.HistoryQueryIteratorInterface, error) {
	return &fakeHistoryIterator{mods: s.history[key]}, nil
}

type fakeHistoryIterator struct {
	mods []*queryresult.KeyModification
	pos  int
}

func (it *fakeHistoryIterator) HasNext() bool { return it.pos < len(it.mods) }
func (it *fakeHistoryIterator) Close() error  { return nil }
func (it *fakeHistoryIterator) Next() (*queryresult.KeyModification, error) {
	if !it.HasNext() {
		return nil, errors.New("fakeHistoryIterator: no more items")
	}
	m := it.mods[it.pos]
	it.pos++
	return m, nil
}

// --- context helpers ---------------------------------------------------------

// newTestContext wires a fresh historyMockStub into a contractapi
// TransactionContext and starts a mocked transaction (required before
// MockStub.PutState will succeed). mspID == "" leaves the stub's Creator
// unset, simulating a caller whose identity cannot be determined at all
// (cid.GetMSPID/GetID fail) — the "fails MSP verification" case.
func newTestContext(t *testing.T, mspID string) (*contractapi.TransactionContext, *historyMockStub) {
	t.Helper()
	stub := newHistoryMockStub("employeeprofilerecord")
	stub.MockTransactionStart("tx-" + t.Name())
	if mspID != "" {
		stub.Creator = serializedIdentity(t, mspID)
	}

	ctx := &contractapi.TransactionContext{}
	ctx.SetStub(stub)
	return ctx, stub
}

// setStubCreator swaps the simulated caller identity on an already-wired
// stub, so a test can drive several different identities against the SAME
// underlying world state (i.e. read-authorization behaviour, not data
// availability, is what varies between sub-tests).
func setStubCreator(t *testing.T, stub shim.ChaincodeStubInterface, mspID string) {
	t.Helper()
	hs, ok := stub.(*historyMockStub)
	require.True(t, ok, "setStubCreator expects a *historyMockStub")
	hs.Creator = serializedIdentity(t, mspID)
}

// clearStubCreator simulates a caller with no determinable identity at all.
func clearStubCreator(stub shim.ChaincodeStubInterface) {
	hs, ok := stub.(*historyMockStub)
	if !ok {
		return
	}
	hs.Creator = nil
}

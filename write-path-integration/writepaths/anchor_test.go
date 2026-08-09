package writepaths

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	gatewayclient "gatewayclient"
	ipfsclient "ipfsclient"
	keystore "keystore"
)

// Compile-time interface satisfaction pins — placed here rather than in
// ports.go so production code keeps zero ipfsclient dependency (ports.go
// only imports gatewayclient, for BuildArgsFunc's alias).
var _ LedgerAnchorer = (*gatewayclient.GatewayClient)(nil)
var _ DocumentPinner = (*ipfsclient.Client)(nil)

// ---- hand-written fakes (no mocking library, no testify — matches
// integration-bridge/shutdown_test.go's stubCloser convention). These tests
// never call either fake concurrently, so unlike stubCloser they don't need
// a mutex guarding their counters. ----

// fakeLedger implements LedgerAnchorer. It captures every argument
// SubmitRecordProfileSection was called with, including the buildArgs
// closure itself, so a test can invoke that closure directly to inspect the
// exact argument vector doAnchor would have sent to the chaincode.
type fakeLedger struct {
	calls          int
	ctx            context.Context
	tenantID       string
	employeeID     string
	profileSection string
	buildArgs      BuildArgsFunc

	result []byte
	err    error
}

func (f *fakeLedger) SubmitRecordProfileSection(ctx context.Context, tenantID, employeeID, profileSection string, buildArgs BuildArgsFunc) ([]byte, error) {
	f.calls++
	f.ctx = ctx
	f.tenantID = tenantID
	f.employeeID = employeeID
	f.profileSection = profileSection
	f.buildArgs = buildArgs
	return f.result, f.err
}

// fakeDocumentPinner implements DocumentPinner. It captures the key/
// plaintext EncryptAndAdd was called with — deliberately has no
// FetchAndDecrypt, matching DocumentPinner itself (writepaths never reads a
// document back).
type fakeDocumentPinner struct {
	calls     int
	key       []byte
	plaintext []byte

	cid string
	err error
}

func (f *fakeDocumentPinner) EncryptAndAdd(ctx context.Context, key, plaintext []byte) (string, error) {
	f.calls++
	f.key = key
	f.plaintext = plaintext
	return f.cid, f.err
}

// newTestHooksAndFakes wires a fresh Hooks against real (in-memory)
// operational-store/keystore implementations — matching how erasure_test.go
// already uses keystore.InMemory* rather than fakes for those — plus the two
// fakes above standing in for the ledger and IPFS, the only two dependencies
// this file exists to make fakeable.
func newTestHooksAndFakes() (*Hooks, *fakeLedger, *fakeDocumentPinner) {
	ledger := &fakeLedger{}
	pinner := &fakeDocumentPinner{}
	h := &Hooks{
		Store:        NewInMemoryOperationalStore(),
		Keys:         keystore.NewInMemoryEmployeeKeyStore(),
		Salts:        keystore.NewInMemorySaltStore(),
		DocumentKeys: keystore.NewInMemoryDocumentKeyStore(),
		IPFS:         pinner,
		Gateway:      ledger,
		TenantID:     "tenant01",
	}
	return h, ledger, pinner
}

// ---- failure-injection fakes for the three keystore-store-write errors
// doAnchor calls before submitting. Each embeds the real keystore.InMemory*
// implementation (so every method it doesn't override keeps working exactly
// like erasure_test.go's usage) and overrides only the one method that needs
// to fail. ----

type failingEmployeeKeyStore struct {
	*keystore.InMemoryEmployeeKeyStore
	err error
}

func newFailingEmployeeKeyStore(err error) *failingEmployeeKeyStore {
	return &failingEmployeeKeyStore{InMemoryEmployeeKeyStore: keystore.NewInMemoryEmployeeKeyStore(), err: err}
}

func (f *failingEmployeeKeyStore) GetOrCreateEmployeeKey(context.Context, string) ([]byte, error) {
	return nil, f.err
}

type failingSaltStore struct {
	*keystore.InMemorySaltStore
	err error
}

func newFailingSaltStore(err error) *failingSaltStore {
	return &failingSaltStore{InMemorySaltStore: keystore.NewInMemorySaltStore(), err: err}
}

func (f *failingSaltStore) PutSalt(context.Context, string, string, int) ([]byte, error) {
	return nil, f.err
}

type failingDocumentKeyStore struct {
	*keystore.InMemoryDocumentKeyStore
	err error
}

func newFailingDocumentKeyStore(err error) *failingDocumentKeyStore {
	return &failingDocumentKeyStore{InMemoryDocumentKeyStore: keystore.NewInMemoryDocumentKeyStore(), err: err}
}

func (f *failingDocumentKeyStore) GetOrCreateDocumentKey(context.Context, string) ([]byte, error) {
	return nil, f.err
}

// failingSaveSectionStore implements OperationalStore directly (it's a
// writepaths-owned interface with only two methods, so there's no
// InMemory* type to embed against — unlike the three keystore fakes above).
type failingSaveSectionStore struct{ err error }

func (f *failingSaveSectionStore) SaveSection(context.Context, string, string, []byte) (int, error) {
	return 0, f.err
}

func (f *failingSaveSectionStore) DeleteSection(context.Context, string, string) error {
	return nil
}

// ---- happy-path tests ----

func TestDoAnchor_HappyPath_NoDocument(t *testing.T) {
	h, ledger, pinner := newTestHooksAndFakes()
	ledger.result = []byte(`{"recordID":"fake-record-1"}`)

	ctx := context.Background()
	result, err := h.UpdatePersonalData(ctx, "emp-happy-nodoc", "user-1", []byte(`{"fullName":"No Document"}`), nil)
	if err != nil {
		t.Fatalf("UpdatePersonalData: unexpected error: %v", err)
	}
	if string(result) != string(ledger.result) {
		t.Errorf("result = %s, want %s", result, ledger.result)
	}

	if pinner.calls != 0 {
		t.Errorf("DocumentPinner.EncryptAndAdd called %d times, want 0 when no document is supplied", pinner.calls)
	}
	if ledger.calls != 1 {
		t.Fatalf("LedgerAnchorer.SubmitRecordProfileSection called %d times, want exactly 1", ledger.calls)
	}
	if ledger.ctx == nil {
		t.Error("SubmitRecordProfileSection was not passed the caller's ctx")
	}

	args := ledger.buildArgs("some-prev-hash")
	if len(args) != 10 {
		t.Fatalf("buildArgs returned %d args, want exactly 10 (RecordProfileSection's full argument list)", len(args))
	}
	if args[6] != "[]" {
		t.Errorf("ipfsCIDs arg (index 6) = %q, want \"[]\" when no document was supplied", args[6])
	}
}

func TestDoAnchor_HappyPath_WithDocument(t *testing.T) {
	h, ledger, pinner := newTestHooksAndFakes()
	ledger.result = []byte(`{"recordID":"fake-record-2"}`)
	pinner.cid = "QmFakeCIDForTestingOnly"

	ctx := context.Background()
	document := []byte("a generic supporting document, never written in plaintext to IPFS")
	if _, err := h.UpdatePersonalData(ctx, "emp-happy-doc", "user-1", []byte(`{"fullName":"With Document"}`), document); err != nil {
		t.Fatalf("UpdatePersonalData: unexpected error: %v", err)
	}

	if pinner.calls != 1 {
		t.Fatalf("DocumentPinner.EncryptAndAdd called %d times, want exactly 1", pinner.calls)
	}
	if string(pinner.plaintext) != string(document) {
		t.Errorf("EncryptAndAdd plaintext = %q, want %q", pinner.plaintext, document)
	}
	wantKey, err := h.DocumentKeys.GetOrCreateDocumentKey(ctx, "emp-happy-doc")
	if err != nil {
		t.Fatalf("GetOrCreateDocumentKey: %v", err)
	}
	if string(pinner.key) != string(wantKey) {
		t.Errorf("EncryptAndAdd key = %x, want the employee's KEY_EMPLOYEE %x", pinner.key, wantKey)
	}

	// Pin must happen before submit — doAnchor builds ipfsCIDs from the CID
	// returned by EncryptAndAdd, so it cannot call Submit first.
	if ledger.calls != 1 {
		t.Fatalf("LedgerAnchorer.SubmitRecordProfileSection called %d times, want exactly 1", ledger.calls)
	}

	args := ledger.buildArgs("some-prev-hash")
	wantIPFSCIDsJSON, err := json.Marshal([]string{pinner.cid})
	if err != nil {
		t.Fatalf("marshaling expected ipfsCIDs: %v", err)
	}
	if args[6] != string(wantIPFSCIDsJSON) {
		t.Errorf("ipfsCIDs arg (index 6) = %q, want %q (the CID EncryptAndAdd returned)", args[6], wantIPFSCIDsJSON)
	}
}

// TestDoAnchor_BuildArgsVectorIsCorrectAndOnlyPrevHashVaries proves the full
// 10-argument RecordProfileSection vector doAnchor builds — provable today
// only by round-tripping through a live chaincode before this test existed.
func TestDoAnchor_BuildArgsVectorIsCorrectAndOnlyPrevHashVaries(t *testing.T) {
	h, ledger, _ := newTestHooksAndFakes()
	ledger.result = []byte(`{"recordID":"fake-record-3"}`)

	ctx := context.Background()
	employeeInternalID := "emp-buildargs-vector"
	userID := "user-vector"
	sectionValue := []byte(`{"fullName":"Vector Check","address":"Generic City"}`)

	if _, err := h.UpdatePersonalData(ctx, employeeInternalID, userID, sectionValue, nil); err != nil {
		t.Fatalf("UpdatePersonalData: unexpected error: %v", err)
	}
	if ledger.buildArgs == nil {
		t.Fatal("buildArgs was never captured — SubmitRecordProfileSection was not called")
	}

	// Independently recompute every expected value from the same stores
	// doAnchor itself used, rather than hardcoding digests — the salt is
	// generated by crypto/rand inside doAnchor, so it can only be recovered
	// after the fact via Salts.GetSalt.
	employeeKey, err := h.Keys.GetOrCreateEmployeeKey(ctx, employeeInternalID)
	if err != nil {
		t.Fatalf("GetOrCreateEmployeeKey: %v", err)
	}
	wantEmployeeID, err := gatewayclient.ComputeEmployeeID(employeeKey)
	if err != nil {
		t.Fatalf("ComputeEmployeeID: %v", err)
	}
	salt, err := h.Salts.GetSalt(ctx, wantEmployeeID, "PERSONAL", 1)
	if err != nil {
		t.Fatalf("GetSalt: %v", err)
	}
	wantDataHash, err := gatewayclient.ComputeDataHash(salt, sectionValue)
	if err != nil {
		t.Fatalf("ComputeDataHash: %v", err)
	}
	wantUpdatedBy, err := gatewayclient.ComputeUpdatedBy(employeeKey, userID)
	if err != nil {
		t.Fatalf("ComputeUpdatedBy: %v", err)
	}

	argsA := ledger.buildArgs("prevHashA")
	argsB := ledger.buildArgs("prevHashB")

	wantFixed := []string{
		"tenant01",                            // [0] tenantID
		wantEmployeeID,                        // [1] employeeID
		"PERSONAL",                            // [2] profileSection
		wantDataHash,                          // [3] dataHash
		"",                                    // [4] prevHash — varies per invocation, checked separately below
		wantUpdatedBy,                         // [5] updatedBy
		"[]",                                  // [6] ipfsCIDs — no document in this test
		gatewayclient.CanonicalizationVersion, // [7]
		gatewayclient.HashAlgo,                // [8]
		"",                                    // [9] clientTimestamp — always empty; writepaths never supplies one
	}

	for i, want := range wantFixed {
		if i == 4 {
			continue // prevHash — checked below
		}
		if argsA[i] != want {
			t.Errorf("buildArgs()[%d] = %q, want %q", i, argsA[i], want)
		}
		if argsB[i] != argsA[i] {
			t.Errorf("buildArgs()[%d] changed between invocations with different prevHash (%q vs %q) — only index 4 (prevHash) should ever vary", i, argsA[i], argsB[i])
		}
	}
	if argsA[4] != "prevHashA" {
		t.Errorf("buildArgs(\"prevHashA\")[4] = %q, want \"prevHashA\"", argsA[4])
	}
	if argsB[4] != "prevHashB" {
		t.Errorf("buildArgs(\"prevHashB\")[4] = %q, want \"prevHashB\"", argsB[4])
	}
}

// TestDoAnchor_AllFiveHooksSubmitTheirOwnProfileSection is table-driven
// across all five write-path hooks — each must dispatch its own,
// distinct profileSection value.
func TestDoAnchor_AllFiveHooksSubmitTheirOwnProfileSection(t *testing.T) {
	type hookFunc func(h *Hooks, ctx context.Context, employeeInternalID, userID string, newValue, document []byte) ([]byte, error)

	tests := []struct {
		name    string
		section string
		call    hookFunc
	}{
		{"UpdatePersonalData", "PERSONAL", (*Hooks).UpdatePersonalData},
		{"ApproveEmploymentTransfer", "EMPLOYMENT", (*Hooks).ApproveEmploymentTransfer},
		{"RecordEducationHistory", "EDUCATION", (*Hooks).RecordEducationHistory},
		{"ApproveFamilyDataChange", "ADDITIONAL", (*Hooks).ApproveFamilyDataChange},
		{"UpdatePayrollBankAccount", "PAYROLL", (*Hooks).UpdatePayrollBankAccount},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, ledger, _ := newTestHooksAndFakes()
			ledger.result = []byte(`{"recordID":"fake-record"}`)

			if _, err := tt.call(h, context.Background(), "emp-"+tt.name, "user-1", []byte(`{"x":1}`), nil); err != nil {
				t.Fatalf("%s: unexpected error: %v", tt.name, err)
			}
			if ledger.calls != 1 {
				t.Fatalf("%s: SubmitRecordProfileSection called %d times, want exactly 1", tt.name, ledger.calls)
			}
			if ledger.profileSection != tt.section {
				t.Errorf("%s: submitted profileSection = %q, want %q", tt.name, ledger.profileSection, tt.section)
			}
		})
	}
}

// ---- PartialFailureError construction, one case per failure injection
// point doAnchor has before it returns. ----

func TestDoAnchor_EachFailurePointProducesPartialFailureError(t *testing.T) {
	tests := []struct {
		name         string
		withDocument bool
		configure    func(h *Hooks, ledger *fakeLedger, pinner *fakeDocumentPinner, sentinelErr error)
	}{
		{
			name: "Gateway.SubmitRecordProfileSection fails",
			configure: func(h *Hooks, ledger *fakeLedger, _ *fakeDocumentPinner, sentinelErr error) {
				ledger.err = sentinelErr
			},
		},
		{
			name:         "IPFS.EncryptAndAdd fails",
			withDocument: true,
			configure: func(_ *Hooks, _ *fakeLedger, pinner *fakeDocumentPinner, sentinelErr error) {
				pinner.err = sentinelErr
			},
		},
		{
			name: "Keys.GetOrCreateEmployeeKey fails",
			configure: func(h *Hooks, _ *fakeLedger, _ *fakeDocumentPinner, sentinelErr error) {
				h.Keys = newFailingEmployeeKeyStore(sentinelErr)
			},
		},
		{
			name: "Salts.PutSalt fails",
			configure: func(h *Hooks, _ *fakeLedger, _ *fakeDocumentPinner, sentinelErr error) {
				h.Salts = newFailingSaltStore(sentinelErr)
			},
		},
		{
			name:         "DocumentKeys.GetOrCreateDocumentKey fails",
			withDocument: true,
			configure: func(h *Hooks, _ *fakeLedger, _ *fakeDocumentPinner, sentinelErr error) {
				h.DocumentKeys = newFailingDocumentKeyStore(sentinelErr)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentinelErr := fmt.Errorf("boom: %s", tt.name)
			h, ledger, pinner := newTestHooksAndFakes()

			var onPartialFailureCalls int
			var onPartialFailureErr error
			h.OnPartialFailure = func(_ context.Context, employeeInternalID, profileSection string, version int, err error) {
				onPartialFailureCalls++
				onPartialFailureErr = err
				if employeeInternalID != "emp-partial-failure" {
					t.Errorf("OnPartialFailure employeeInternalID = %q, want %q", employeeInternalID, "emp-partial-failure")
				}
				if profileSection != "PERSONAL" {
					t.Errorf("OnPartialFailure profileSection = %q, want %q", profileSection, "PERSONAL")
				}
				if version != 1 {
					t.Errorf("OnPartialFailure version = %d, want 1", version)
				}
			}

			tt.configure(h, ledger, pinner, sentinelErr)

			var document []byte
			if tt.withDocument {
				document = []byte("a supporting document")
			}

			_, err := h.UpdatePersonalData(context.Background(), "emp-partial-failure", "user-1", []byte(`{"x":1}`), document)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}

			var pfErr *PartialFailureError
			if !errors.As(err, &pfErr) {
				t.Fatalf("errors.As(err, *PartialFailureError) failed; got %T: %v", err, err)
			}
			if pfErr.EmployeeInternalID != "emp-partial-failure" {
				t.Errorf("PartialFailureError.EmployeeInternalID = %q, want %q", pfErr.EmployeeInternalID, "emp-partial-failure")
			}
			if pfErr.ProfileSection != "PERSONAL" {
				t.Errorf("PartialFailureError.ProfileSection = %q, want %q", pfErr.ProfileSection, "PERSONAL")
			}
			if pfErr.Version != 1 {
				t.Errorf("PartialFailureError.Version = %d, want 1", pfErr.Version)
			}
			if !errors.Is(err, sentinelErr) {
				t.Errorf("errors.Is(err, sentinelErr) = false — Unwrap() chain does not reach the injected sentinel error: %v", err)
			}

			if onPartialFailureCalls != 1 {
				t.Fatalf("OnPartialFailure called %d times, want exactly 1", onPartialFailureCalls)
			}
			var onPartialFailurePfErr *PartialFailureError
			if !errors.As(onPartialFailureErr, &onPartialFailurePfErr) {
				t.Errorf("OnPartialFailure's err argument is not a *PartialFailureError: %T: %v", onPartialFailureErr, onPartialFailureErr)
			}
		})
	}
}

// TestDoAnchor_SaveSectionErrorDoesNotProducePartialFailure is the inverse
// invariant: SaveSection has not committed anything, so its own error must
// never be mistaken for the "operational write committed, anchor did not"
// state PartialFailureError exists to name (see that type's doc comment).
func TestDoAnchor_SaveSectionErrorDoesNotProducePartialFailure(t *testing.T) {
	h, ledger, pinner := newTestHooksAndFakes()
	sentinelErr := errors.New("boom: operational store write failed")
	h.Store = &failingSaveSectionStore{err: sentinelErr}

	var onPartialFailureCalls int
	h.OnPartialFailure = func(context.Context, string, string, int, error) { onPartialFailureCalls++ }

	_, err := h.UpdatePersonalData(context.Background(), "emp-save-failure", "user-1", []byte(`{"x":1}`), nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, sentinelErr) {
		t.Errorf("expected the SaveSection sentinel error to surface, got %v", err)
	}

	var pfErr *PartialFailureError
	if errors.As(err, &pfErr) {
		t.Fatalf("a SaveSection failure must NEVER be wrapped in *PartialFailureError (nothing committed yet) — got %+v", pfErr)
	}
	if onPartialFailureCalls != 0 {
		t.Errorf("OnPartialFailure called %d times, want 0 — anchor() must never run when SaveSection itself failed", onPartialFailureCalls)
	}
	if ledger.calls != 0 {
		t.Errorf("LedgerAnchorer.SubmitRecordProfileSection called %d times, want 0", ledger.calls)
	}
	if pinner.calls != 0 {
		t.Errorf("DocumentPinner.EncryptAndAdd called %d times, want 0", pinner.calls)
	}
}

// TestDoAnchor_OnPartialFailureNilSafe proves anchor() never dereferences a
// nil OnPartialFailure — the common case in every test/demo that doesn't set
// it (see Hooks.OnPartialFailure's own doc comment: "nil is fine").
func TestDoAnchor_OnPartialFailureNilSafe(t *testing.T) {
	h, ledger, _ := newTestHooksAndFakes()
	ledger.err = errors.New("boom: gateway unreachable")
	// OnPartialFailure deliberately left nil.

	_, err := h.UpdatePersonalData(context.Background(), "emp-nil-safe", "user-1", []byte(`{"x":1}`), nil)
	// Reaching this line without a panic is itself the assertion this test
	// exists to make.
	var pfErr *PartialFailureError
	if !errors.As(err, &pfErr) {
		t.Fatalf("errors.As(err, *PartialFailureError) failed; got %T: %v", err, err)
	}
}

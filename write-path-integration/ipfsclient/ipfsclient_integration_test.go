//go:build integration

// Integration tests against the LIVE two-node kubo private swarm brought up
// by ipfs-cluster/docker-compose.yaml (REC-5). Talks to plain kubo HTTP
// APIs directly, not the ipfs-cluster CRDT layer — that layer's own
// cluster0/cluster1 peers do not currently negotiate a security handshake
// with each other (an open, disclosed defect; see ipfs-cluster/README.md).
// The underlying private swarm the two kubo nodes share is independently
// verified working, which is the actual property these tests exercise:
// content added (as ciphertext) on one node, pinned on the other.
package ipfsclient

import (
	"context"
	"testing"
)

const (
	primaryAPI = "http://localhost:5001"
	replicaAPI = "http://localhost:5002"
)

func TestIntegration_EncryptAndAdd_RoundTrip(t *testing.T) {
	c := NewClient(primaryAPI, replicaAPI)
	key := testKey(0x55)
	plaintext := []byte("REC-5 integration fixture — supporting document content, never written in plaintext to IPFS")

	cid, err := c.EncryptAndAdd(context.Background(), key, plaintext)
	if err != nil {
		t.Fatalf("EncryptAndAdd: %v", err)
	}
	if cid == "" {
		t.Fatal("EncryptAndAdd returned an empty CID")
	}
	t.Logf("pinned on both nodes at CID %s", cid)

	got, err := c.FetchAndDecrypt(context.Background(), key, cid)
	if err != nil {
		t.Fatalf("FetchAndDecrypt: %v", err)
	}
	if string(got) != string(plaintext) {
		t.Fatalf("FetchAndDecrypt = %q, want %q", got, plaintext)
	}

	// The replica node must independently serve the SAME ciphertext bytes —
	// proving the explicit cross-node pin actually replicated content, not
	// just recorded a pin intent that happens to resolve via the primary.
	replicaOnly := &Client{PrimaryAPI: replicaAPI, ReplicaAPI: replicaAPI, HTTPClient: c.HTTPClient}
	gotFromReplica, err := replicaOnly.FetchAndDecrypt(context.Background(), key, cid)
	if err != nil {
		t.Fatalf("FetchAndDecrypt from replica node: %v", err)
	}
	if string(gotFromReplica) != string(plaintext) {
		t.Fatalf("replica node returned %q, want %q", gotFromReplica, plaintext)
	}
}

// REC-5 DoD, verified end-to-end against the live cluster: re-encrypting —
// even byte-identical plaintext, with the same key — produces a new CID.
// The caller (writepaths hooks) is therefore structurally unable to mutate
// an existing version's ipfsCIDs in place; each call is a brand new pin.
func TestIntegration_ReEncryptionProducesNewCID(t *testing.T) {
	c := NewClient(primaryAPI, replicaAPI)
	key := testKey(0x77)
	plaintext := []byte("same content, re-encrypted after a hypothetical key rotation")

	firstCID, err := c.EncryptAndAdd(context.Background(), key, plaintext)
	if err != nil {
		t.Fatalf("EncryptAndAdd (first): %v", err)
	}
	secondCID, err := c.EncryptAndAdd(context.Background(), key, plaintext)
	if err != nil {
		t.Fatalf("EncryptAndAdd (second): %v", err)
	}

	if firstCID == secondCID {
		t.Fatalf("re-encrypting identical plaintext with the same key produced the SAME CID (%s) — this would look like an in-place mutation instead of a new version", firstCID)
	}
	t.Logf("first CID %s, second CID %s — distinct, as required", firstCID, secondCID)
}

func TestIntegration_FetchAndDecryptWithWrongKeyFails(t *testing.T) {
	c := NewClient(primaryAPI, replicaAPI)
	correctKey := testKey(0x99)
	wrongKey := testKey(0x66)

	cid, err := c.EncryptAndAdd(context.Background(), correctKey, []byte("only the right KEY_EMPLOYEE recovers this"))
	if err != nil {
		t.Fatalf("EncryptAndAdd: %v", err)
	}

	if _, err := c.FetchAndDecrypt(context.Background(), wrongKey, cid); err == nil {
		t.Fatal("FetchAndDecrypt succeeded with the wrong key against real pinned ciphertext")
	}
}

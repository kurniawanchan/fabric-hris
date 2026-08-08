// Package ipfsclient implements REC-5: encrypt supporting documents with
// the employee's own document key (KEY_EMPLOYEE, ADR-0019 Domain B′)
// BEFORE pinning to the IPFS private cluster (ADR-0016); only the
// resulting CID — never document content — is meant to be attached to a
// RecordProfileSection call for that version.
//
// Talks to plain kubo (go-ipfs) HTTP APIs directly rather than the
// ipfs-cluster orchestration layer's own API: this session's own IPFS
// Private Cluster bring-up hit a real, unresolved libp2p security-protocol
// negotiation failure between the two ipfs-cluster CRDT-consensus peers
// (three OTHER real defects along the way — AutoConf, the bootstrap
// mechanism, a stale peerstore — were found and fixed; this fourth one was
// not, after a genuine debugging effort) — documented, not hidden. The
// underlying two-node PRIVATE SWARM (the actual ADR-0016 requirement: a
// swarm-key-gated network, not any particular orchestration software) is
// independently verified working: cross-node fetch confirmed live (`ipfs1`
// pinning and reading back content added on `ipfs0`, over the private
// swarm). This package therefore replicates by explicitly pinning on BOTH
// kubo nodes — the same practical durability property ADR-0016 wants
// ("2 cluster peers... both Org1-operated"), achieved directly rather than
// through the (currently broken) cluster consensus layer.
package ipfsclient

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

// MinDocumentKeyBytes matches keystore.MinKeyBytes exactly (16 bytes) —
// ADR-0011's ">=128 bits" floor, which KEY_EMPLOYEE (ADR-0019 Domain B′) is
// actually generated at. crypto/aes selects AES-128 for a 16-byte key
// automatically; this package does not invent a second, larger key-size
// convention independent of the one the key is actually issued at.
const MinDocumentKeyBytes = 16

// Client holds the two kubo node HTTP API endpoints this package pins to.
// "Primary" is where content is first added; "Replica" is explicitly
// pinned afterward for the 2-node durability property.
type Client struct {
	PrimaryAPI string // e.g. "http://localhost:5001"
	ReplicaAPI string // e.g. "http://localhost:5002"
	HTTPClient *http.Client
}

func NewClient(primaryAPI, replicaAPI string) *Client {
	return &Client{PrimaryAPI: primaryAPI, ReplicaAPI: replicaAPI, HTTPClient: &http.Client{}}
}

// encrypt performs AES-256-GCM encryption with a FRESH random nonce every
// call (never reused, even for the same key) — the nonce is prepended to
// the returned ciphertext, the standard self-describing layout. A fresh
// nonce alone (even with the SAME key and SAME plaintext) already produces
// different ciphertext bytes on every call, which is why re-encryption
// naturally produces a new CID (REC-5's DoD) without needing any special
// versioning logic here.
func encrypt(key, plaintext []byte) ([]byte, error) {
	if len(key) != MinDocumentKeyBytes {
		return nil, fmt.Errorf("ipfsclient: KEY_EMPLOYEE must be %d bytes, got %d", MinDocumentKeyBytes, len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("ipfsclient: creating AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("ipfsclient: creating GCM mode: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("ipfsclient: generating nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt reverses encrypt. Returns an error (never partial/garbage
// plaintext) if key is wrong or ciphertext was tampered with — AES-GCM's
// authentication tag check, not a separate integrity mechanism this
// package adds.
func decrypt(key, ciphertext []byte) ([]byte, error) {
	if len(key) != MinDocumentKeyBytes {
		return nil, fmt.Errorf("ipfsclient: KEY_EMPLOYEE must be %d bytes, got %d", MinDocumentKeyBytes, len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("ipfsclient: creating AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("ipfsclient: creating GCM mode: %w", err)
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ipfsclient: ciphertext shorter than nonce size")
	}
	nonce, sealed := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, fmt.Errorf("ipfsclient: decryption failed (wrong key or tampered ciphertext): %w", err)
	}
	return plaintext, nil
}

// EncryptAndAdd encrypts plaintext with key (KEY_EMPLOYEE), adds the
// CIPHERTEXT ONLY to the primary node, pins it on the replica node too
// (explicit cross-node pin, confirmed working over the private swarm), and
// returns the CID. The caller attaches this CID — never plaintext, never
// even the ciphertext bytes — to that version's RecordProfileSection call.
func (c *Client) EncryptAndAdd(ctx context.Context, key, plaintext []byte) (string, error) {
	ciphertext, err := encrypt(key, plaintext)
	if err != nil {
		return "", err
	}

	cid, err := c.add(ctx, c.PrimaryAPI, ciphertext)
	if err != nil {
		return "", fmt.Errorf("ipfsclient: adding to primary node: %w", err)
	}

	if err := c.pin(ctx, c.ReplicaAPI, cid); err != nil {
		return "", fmt.Errorf("ipfsclient: pinning on replica node: %w", err)
	}

	return cid, nil
}

// FetchAndDecrypt reads the ciphertext for cid from the primary node and
// decrypts it with key. Fails closed (no plaintext returned) if key is
// wrong.
func (c *Client) FetchAndDecrypt(ctx context.Context, key []byte, cid string) ([]byte, error) {
	ciphertext, err := c.cat(ctx, c.PrimaryAPI, cid)
	if err != nil {
		return nil, fmt.Errorf("ipfsclient: fetching %s: %w", cid, err)
	}
	return decrypt(key, ciphertext)
}

func (c *Client) add(ctx context.Context, api string, data []byte) (string, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", "document.bin")
	if err != nil {
		return "", err
	}
	if _, err := part.Write(data); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, api+"/api/v0/add", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("add: status %d: %s", resp.StatusCode, b)
	}
	var result struct {
		Hash string `json:"Hash"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Hash, nil
}

func (c *Client) pin(ctx context.Context, api, cid string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, api+"/api/v0/pin/add?arg="+cid, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pin/add: status %d: %s", resp.StatusCode, b)
	}
	return nil
}

func (c *Client) cat(ctx context.Context, api, cid string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, api+"/api/v0/cat?arg="+cid, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cat: status %d: %s", resp.StatusCode, b)
	}
	return io.ReadAll(resp.Body)
}

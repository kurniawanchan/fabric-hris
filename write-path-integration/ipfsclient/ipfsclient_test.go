package ipfsclient

import (
	"bytes"
	"strings"
	"testing"
)

func testKey(b byte) []byte {
	key := make([]byte, MinDocumentKeyBytes)
	for i := range key {
		key[i] = b
	}
	return key
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := testKey(0x42)
	plaintext := []byte("supporting document content — not real PII, this is a unit test fixture")

	ciphertext, err := encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if bytes.Contains(ciphertext, plaintext) {
		t.Fatal("ciphertext contains the plaintext verbatim — encryption did not actually transform the content")
	}

	got, err := decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("decrypt(encrypt(x)) = %q, want %q", got, plaintext)
	}
}

// REC-5's DoD: "re-encryption ... produces a new CID on a NEW version, never
// mutates an existing record's ipfsCIDs." The CID is a content hash of the
// ADDED BYTES (the ciphertext), so this property holds as long as encrypting
// the SAME plaintext twice, with the SAME key, never produces the SAME
// ciphertext bytes — which AES-GCM guarantees via a fresh random nonce per
// call. Verified here at the ciphertext level, independent of any live IPFS
// node.
func TestEncryptSamePlaintextSameKeyProducesDifferentCiphertext(t *testing.T) {
	key := testKey(0x07)
	plaintext := []byte("identical content, encrypted twice")

	first, err := encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt (first): %v", err)
	}
	second, err := encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt (second): %v", err)
	}

	if bytes.Equal(first, second) {
		t.Fatal("encrypting the same plaintext with the same key twice produced IDENTICAL ciphertext — nonce reuse would collapse to the same CID, violating REC-5's new-CID-per-write requirement")
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	correctKey := testKey(0xAA)
	wrongKey := testKey(0xBB)
	plaintext := []byte("only the correct KEY_EMPLOYEE should recover this")

	ciphertext, err := encrypt(correctKey, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if _, err := decrypt(wrongKey, ciphertext); err == nil {
		t.Fatal("decrypt with the wrong key succeeded — content would be recoverable without KEY_EMPLOYEE, defeating the whole point of encrypting before pinning")
	}
}

func TestDecryptTamperedCiphertextFails(t *testing.T) {
	key := testKey(0x11)
	ciphertext, err := encrypt(key, []byte("tamper-detection fixture"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	tampered := append([]byte{}, ciphertext...)
	tampered[len(tampered)-1] ^= 0xFF

	if _, err := decrypt(key, tampered); err == nil {
		t.Fatal("decrypt accepted tampered ciphertext without error — GCM's authentication tag should reject this")
	}
}

func TestEncryptRejectsWrongKeyLength(t *testing.T) {
	shortKey := []byte("too-short")
	if _, err := encrypt(shortKey, []byte("x")); err == nil {
		t.Fatal("encrypt accepted a key shorter than the required size")
	} else if !strings.Contains(err.Error(), "16 bytes") {
		t.Fatalf("error should name the expected key size, got: %v", err)
	}
}

func TestDecryptRejectsCiphertextShorterThanNonce(t *testing.T) {
	key := testKey(0x33)
	if _, err := decrypt(key, []byte("short")); err == nil {
		t.Fatal("decrypt accepted ciphertext shorter than a GCM nonce")
	}
}

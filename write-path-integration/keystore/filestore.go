// Story tf-4.1: persistent implementations of EmployeeKeyStore, SaltStore,
// and DocumentKeyStore — the PRD's own §13 risk ("a process restart
// silently loses every salt and employeeKey_i ever issued") applies to all
// three InMemory* stores identically, not just one.
//
// Persistence choice (confirmed with the user, 2026-08-13): an
// AES-256-GCM-encrypted local file, no database engine. Each store owns one
// file; the whole in-memory map is decrypted, mutated, and re-encrypted on
// every write (O(n) per write — acceptable for this deployment's scale, not
// optimized further). A sync.Mutex guards concurrent access WITHIN one
// process; this design does not support multiple integration-bridge
// instances sharing one file safely (no file locking across processes) —
// disclosed limitation, not silently assumed away.
package keystore

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"sync"
)

// fileStore is the shared encrypted-file mechanism all three Story tf-4.1
// stores below are built from. key must be exactly 32 bytes (AES-256).
type fileStore struct {
	path string
	key  []byte
	mu   sync.Mutex
}

// newFileStore validates the encryption key length up front — a wrong-size
// key would otherwise fail confusingly deep inside aes.NewCipher on the
// first read/write, not at construction time.
func newFileStore(path string, key []byte) (*fileStore, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("keystore: encryption key must be exactly 32 bytes (AES-256), got %d", len(key))
	}
	return &fileStore{path: path, key: key}, nil
}

// load decrypts and gob-decodes the whole file into a map. A missing file
// is treated as "empty store," not an error — the normal case on first
// startup, before anything has ever been written.
func (f *fileStore) load() (map[string][]byte, error) {
	raw, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string][]byte), nil
		}
		return nil, fmt.Errorf("keystore: reading %s: %w", f.path, err)
	}
	if len(raw) == 0 {
		return make(map[string][]byte), nil
	}

	block, err := aes.NewCipher(f.key)
	if err != nil {
		return nil, fmt.Errorf("keystore: constructing cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("keystore: constructing GCM: %w", err)
	}
	if len(raw) < gcm.NonceSize() {
		return nil, fmt.Errorf("keystore: %s is corrupt (shorter than one nonce)", f.path)
	}
	nonce, ciphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("keystore: decrypting %s (wrong key, or the file is corrupt/tampered): %w", f.path, err)
	}

	data := make(map[string][]byte)
	if err := gob.NewDecoder(bytes.NewReader(plaintext)).Decode(&data); err != nil {
		return nil, fmt.Errorf("keystore: decoding %s: %w", f.path, err)
	}
	return data, nil
}

// save gob-encodes and encrypts the whole map, writing it back to disk. A
// fresh random nonce is generated on every save (required for AES-GCM —
// reusing a nonce with the same key breaks its confidentiality guarantee
// entirely, not just weakens it).
func (f *fileStore) save(data map[string][]byte) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(data); err != nil {
		return fmt.Errorf("keystore: encoding: %w", err)
	}

	block, err := aes.NewCipher(f.key)
	if err != nil {
		return fmt.Errorf("keystore: constructing cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("keystore: constructing GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("keystore: generating nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, buf.Bytes(), nil)

	// Write to a temp file then rename, so a crash mid-write never leaves
	// a half-written, unreadable store file in place of a good one.
	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, ciphertext, 0o600); err != nil {
		return fmt.Errorf("keystore: writing %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, f.path); err != nil {
		return fmt.Errorf("keystore: renaming %s to %s: %w", tmp, f.path, err)
	}
	return nil
}

// FileEmployeeKeyStore is the persistent EmployeeKeyStore (Domain C,
// ADR-0021). One employeeKey_i per (possibly compound, per AD-3)
// employeeInternalID key.
type FileEmployeeKeyStore struct {
	fs *fileStore
}

func NewFileEmployeeKeyStore(path string, encryptionKey []byte) (*FileEmployeeKeyStore, error) {
	fs, err := newFileStore(path, encryptionKey)
	if err != nil {
		return nil, err
	}
	return &FileEmployeeKeyStore{fs: fs}, nil
}

func (s *FileEmployeeKeyStore) GetOrCreateEmployeeKey(_ context.Context, employeeInternalID string) ([]byte, error) {
	s.fs.mu.Lock()
	defer s.fs.mu.Unlock()

	data, err := s.fs.load()
	if err != nil {
		return nil, err
	}
	if key, exists := data[employeeInternalID]; exists {
		return key, nil
	}

	key := make([]byte, MinKeyBytes)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("keystore: generating employeeKey_i: %w", err)
	}
	data[employeeInternalID] = key
	if err := s.fs.save(data); err != nil {
		return nil, err
	}
	return key, nil
}

func (s *FileEmployeeKeyStore) DeleteEmployeeKey(_ context.Context, employeeInternalID string) error {
	s.fs.mu.Lock()
	defer s.fs.mu.Unlock()

	data, err := s.fs.load()
	if err != nil {
		return err
	}
	delete(data, employeeInternalID)
	return s.fs.save(data)
}

// FileDocumentKeyStore is the persistent DocumentKeyStore (Domain B',
// ADR-0019). One KEY_EMPLOYEE per employeeInternalID.
type FileDocumentKeyStore struct {
	fs *fileStore
}

func NewFileDocumentKeyStore(path string, encryptionKey []byte) (*FileDocumentKeyStore, error) {
	fs, err := newFileStore(path, encryptionKey)
	if err != nil {
		return nil, err
	}
	return &FileDocumentKeyStore{fs: fs}, nil
}

func (s *FileDocumentKeyStore) GetOrCreateDocumentKey(_ context.Context, employeeInternalID string) ([]byte, error) {
	s.fs.mu.Lock()
	defer s.fs.mu.Unlock()

	data, err := s.fs.load()
	if err != nil {
		return nil, err
	}
	if key, exists := data[employeeInternalID]; exists {
		return key, nil
	}

	key := make([]byte, MinKeyBytes)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("keystore: generating KEY_EMPLOYEE: %w", err)
	}
	data[employeeInternalID] = key
	if err := s.fs.save(data); err != nil {
		return nil, err
	}
	return key, nil
}

func (s *FileDocumentKeyStore) DeleteDocumentKey(_ context.Context, employeeInternalID string) error {
	s.fs.mu.Lock()
	defer s.fs.mu.Unlock()

	data, err := s.fs.load()
	if err != nil {
		return err
	}
	delete(data, employeeInternalID)
	return s.fs.save(data)
}

// FileSaltStore is the persistent SaltStore (ADR-0011). Keyed on the
// composite (employeeID, profileSection, version) — encoded as a single
// string key for this file's flat map, mirroring InMemorySaltStore's own
// recordKey struct.
type FileSaltStore struct {
	fs *fileStore
}

func NewFileSaltStore(path string, encryptionKey []byte) (*FileSaltStore, error) {
	fs, err := newFileStore(path, encryptionKey)
	if err != nil {
		return nil, err
	}
	return &FileSaltStore{fs: fs}, nil
}

func saltFileKey(employeeID, profileSection string, version int) string {
	return fmt.Sprintf("%s\x00%s\x00%d", employeeID, profileSection, version)
}

func (s *FileSaltStore) PutSalt(_ context.Context, employeeID, profileSection string, version int) ([]byte, error) {
	s.fs.mu.Lock()
	defer s.fs.mu.Unlock()

	data, err := s.fs.load()
	if err != nil {
		return nil, err
	}
	key := saltFileKey(employeeID, profileSection, version)
	if _, exists := data[key]; exists {
		return nil, fmt.Errorf("keystore: salt already exists for %s/%s v%d — a record's salt is generated once, not rotated in place", employeeID, profileSection, version)
	}

	salt := make([]byte, MinSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("keystore: generating salt: %w", err)
	}
	data[key] = salt
	if err := s.fs.save(data); err != nil {
		return nil, err
	}
	return salt, nil
}

func (s *FileSaltStore) GetSalt(_ context.Context, employeeID, profileSection string, version int) ([]byte, error) {
	s.fs.mu.Lock()
	defer s.fs.mu.Unlock()

	data, err := s.fs.load()
	if err != nil {
		return nil, err
	}
	salt, exists := data[saltFileKey(employeeID, profileSection, version)]
	if !exists {
		return nil, fmt.Errorf("keystore: no salt stored for %s/%s v%d", employeeID, profileSection, version)
	}
	return salt, nil
}

func (s *FileSaltStore) DeleteAllSaltsForEmployee(_ context.Context, employeeID string) error {
	s.fs.mu.Lock()
	defer s.fs.mu.Unlock()

	data, err := s.fs.load()
	if err != nil {
		return err
	}
	prefix := employeeID + "\x00"
	for key := range data {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(data, key)
		}
	}
	return s.fs.save(data)
}

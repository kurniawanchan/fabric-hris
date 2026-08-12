package keystore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func testEncryptionKey() []byte {
	return []byte("0123456789abcdef0123456789abcdef") // 32 bytes
}

func TestNewFileEmployeeKeyStore_WrongKeyLength_Errors(t *testing.T) {
	_, err := NewFileEmployeeKeyStore(filepath.Join(t.TempDir(), "keys.enc"), []byte("too-short"))
	if err == nil {
		t.Fatal("NewFileEmployeeKeyStore() with a 9-byte key: got nil error, want an error")
	}
}

func TestFileEmployeeKeyStore_GetOrCreate_PersistsAcrossNewInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys.enc")
	key := testEncryptionKey()
	ctx := context.Background()

	store1, err := NewFileEmployeeKeyStore(path, key)
	if err != nil {
		t.Fatalf("NewFileEmployeeKeyStore: %v", err)
	}
	firstKey, err := store1.GetOrCreateEmployeeKey(ctx, "emp-1")
	if err != nil {
		t.Fatalf("GetOrCreateEmployeeKey: %v", err)
	}

	// Simulate a process restart: a FRESH store instance reading the SAME
	// file must return the identical key, not generate a new one.
	store2, err := NewFileEmployeeKeyStore(path, key)
	if err != nil {
		t.Fatalf("NewFileEmployeeKeyStore (second instance): %v", err)
	}
	secondKey, err := store2.GetOrCreateEmployeeKey(ctx, "emp-1")
	if err != nil {
		t.Fatalf("GetOrCreateEmployeeKey (second instance): %v", err)
	}

	if string(firstKey) != string(secondKey) {
		t.Error("key did not survive a simulated restart -- this is the exact defect Story tf-4.1 exists to fix")
	}
}

func TestFileEmployeeKeyStore_WrongDecryptionKey_Errors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys.enc")
	ctx := context.Background()

	store1, _ := NewFileEmployeeKeyStore(path, testEncryptionKey())
	if _, err := store1.GetOrCreateEmployeeKey(ctx, "emp-1"); err != nil {
		t.Fatalf("GetOrCreateEmployeeKey: %v", err)
	}

	wrongKey := []byte("zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz")
	store2, _ := NewFileEmployeeKeyStore(path, wrongKey)
	if _, err := store2.GetOrCreateEmployeeKey(ctx, "emp-1"); err == nil {
		t.Fatal("GetOrCreateEmployeeKey() with the wrong encryption key: got nil error, want a decryption failure")
	}
}

func TestFileEmployeeKeyStore_DeleteEmployeeKey_RemovesIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys.enc")
	key := testEncryptionKey()
	ctx := context.Background()

	store, _ := NewFileEmployeeKeyStore(path, key)
	firstKey, _ := store.GetOrCreateEmployeeKey(ctx, "emp-1")
	if err := store.DeleteEmployeeKey(ctx, "emp-1"); err != nil {
		t.Fatalf("DeleteEmployeeKey: %v", err)
	}
	secondKey, err := store.GetOrCreateEmployeeKey(ctx, "emp-1")
	if err != nil {
		t.Fatalf("GetOrCreateEmployeeKey after delete: %v", err)
	}
	if string(firstKey) == string(secondKey) {
		t.Error("GetOrCreateEmployeeKey returned the same key after DeleteEmployeeKey -- deletion did not take effect")
	}
}

func TestFileSaltStore_PutThenGet_ReturnsSameSalt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "salts.enc")
	ctx := context.Background()
	store, _ := NewFileSaltStore(path, testEncryptionKey())

	putSalt, err := store.PutSalt(ctx, "emp-id", "PERSONAL", 1)
	if err != nil {
		t.Fatalf("PutSalt: %v", err)
	}
	gotSalt, err := store.GetSalt(ctx, "emp-id", "PERSONAL", 1)
	if err != nil {
		t.Fatalf("GetSalt: %v", err)
	}
	if string(putSalt) != string(gotSalt) {
		t.Error("GetSalt did not return the salt PutSalt stored")
	}
}

func TestFileSaltStore_PutSalt_Twice_SameVersion_Errors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "salts.enc")
	ctx := context.Background()
	store, _ := NewFileSaltStore(path, testEncryptionKey())

	if _, err := store.PutSalt(ctx, "emp-id", "PERSONAL", 1); err != nil {
		t.Fatalf("PutSalt (first): %v", err)
	}
	if _, err := store.PutSalt(ctx, "emp-id", "PERSONAL", 1); err == nil {
		t.Fatal("PutSalt() called twice for the same employeeID/profileSection/version: got nil error, want an error -- a salt is generated once, never rotated in place")
	}
}

func TestFileSaltStore_DeleteAllSaltsForEmployee_OnlyDeletesThatEmployeesSalts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "salts.enc")
	ctx := context.Background()
	store, _ := NewFileSaltStore(path, testEncryptionKey())

	if _, err := store.PutSalt(ctx, "emp-1", "PERSONAL", 1); err != nil {
		t.Fatalf("PutSalt emp-1: %v", err)
	}
	if _, err := store.PutSalt(ctx, "emp-2", "PERSONAL", 1); err != nil {
		t.Fatalf("PutSalt emp-2: %v", err)
	}

	if err := store.DeleteAllSaltsForEmployee(ctx, "emp-1"); err != nil {
		t.Fatalf("DeleteAllSaltsForEmployee: %v", err)
	}

	if _, err := store.GetSalt(ctx, "emp-1", "PERSONAL", 1); err == nil {
		t.Error("emp-1's salt still exists after DeleteAllSaltsForEmployee(emp-1)")
	}
	if _, err := store.GetSalt(ctx, "emp-2", "PERSONAL", 1); err != nil {
		t.Error("emp-2's salt was deleted by DeleteAllSaltsForEmployee(emp-1) -- must only affect the named employee")
	}
}

func TestFileDocumentKeyStore_GetOrCreate_PersistsAcrossNewInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doc-keys.enc")
	key := testEncryptionKey()
	ctx := context.Background()

	store1, _ := NewFileDocumentKeyStore(path, key)
	firstKey, err := store1.GetOrCreateDocumentKey(ctx, "emp-1")
	if err != nil {
		t.Fatalf("GetOrCreateDocumentKey: %v", err)
	}

	store2, _ := NewFileDocumentKeyStore(path, key)
	secondKey, err := store2.GetOrCreateDocumentKey(ctx, "emp-1")
	if err != nil {
		t.Fatalf("GetOrCreateDocumentKey (second instance): %v", err)
	}
	if string(firstKey) != string(secondKey) {
		t.Error("KEY_EMPLOYEE did not survive a simulated restart")
	}
}

func TestFileStore_MissingFile_TreatedAsEmptyNotError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist-yet.enc")
	store, _ := NewFileEmployeeKeyStore(path, testEncryptionKey())

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("test setup: file should not exist yet")
	}

	key, err := store.GetOrCreateEmployeeKey(context.Background(), "emp-1")
	if err != nil {
		t.Fatalf("GetOrCreateEmployeeKey() on a missing file: unexpected error: %v", err)
	}
	if len(key) != MinKeyBytes {
		t.Errorf("generated key is %d bytes, want %d", len(key), MinKeyBytes)
	}
}

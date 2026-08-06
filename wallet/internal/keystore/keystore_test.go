package keystore

import (
	"crypto/ed25519"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateAddressFormat(t *testing.T) {
	w, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Address()) != 64 {
		t.Errorf("address: want 64 hex chars, got %d", len(w.Address()))
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "wallet.json")
	w, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Save(path); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("wallet file permissions: want 0600, got %o", info.Mode().Perm())
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Address() != w.Address() {
		t.Errorf("address changed after roundtrip: %s != %s", loaded.Address(), w.Address())
	}
}

func TestSignatureVerifies(t *testing.T) {
	w, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("from|to|10|0|123")
	sig, err := hex.DecodeString(w.Sign(payload))
	if err != nil {
		t.Fatal(err)
	}
	pub, err := hex.DecodeString(w.Address())
	if err != nil {
		t.Fatal(err)
	}

	if !ed25519.Verify(ed25519.PublicKey(pub), payload, sig) {
		t.Error("signature does not verify against the address public key")
	}
	if ed25519.Verify(ed25519.PublicKey(pub), []byte("tampered"), sig) {
		t.Error("signature verified a different payload")
	}
}

func TestLoadRejectsMalformedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallet.json")
	if err := os.WriteFile(path, []byte(`{"seed":"zz"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("want error for malformed seed, got nil")
	}
}

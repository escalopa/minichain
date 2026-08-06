// Package keystore owns the ed25519 key pair: generation, disk
// persistence and signing. Private keys never leave this package.
package keystore

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Wallet wraps an ed25519 seed. The address is the hex of the public
// key — the same convention the node uses.
type Wallet struct {
	priv ed25519.PrivateKey
}

// Generate creates a fresh random key pair.
func Generate() (*Wallet, error) {
	seed := make([]byte, ed25519.SeedSize)
	if _, err := rand.Read(seed); err != nil {
		return nil, fmt.Errorf("generate seed: %w", err)
	}
	return &Wallet{priv: ed25519.NewKeyFromSeed(seed)}, nil
}

type walletFile struct {
	Seed    string `json:"seed"`
	Address string `json:"address"`
}

// Load reads a wallet file written by Save.
func Load(path string) (*Wallet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read wallet file: %w", err)
	}
	var wf walletFile
	if err := json.Unmarshal(raw, &wf); err != nil {
		return nil, fmt.Errorf("parse wallet file: %w", err)
	}
	seed, err := hex.DecodeString(wf.Seed)
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("wallet file has a malformed seed")
	}
	return &Wallet{priv: ed25519.NewKeyFromSeed(seed)}, nil
}

// Save writes the wallet as JSON with owner-only permissions.
// This is an educational project: a real wallet would encrypt the seed.
func (w *Wallet) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create wallet dir: %w", err)
	}
	raw, err := json.MarshalIndent(walletFile{
		Seed:    hex.EncodeToString(w.priv.Seed()),
		Address: w.Address(),
	}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("write wallet file: %w", err)
	}
	return nil
}

// Address is the hex representation of the public key.
func (w *Wallet) Address() string {
	return hex.EncodeToString(w.priv.Public().(ed25519.PublicKey))
}

// Sign returns the hex signature of the payload.
func (w *Wallet) Sign(payload []byte) string {
	return hex.EncodeToString(ed25519.Sign(w.priv, payload))
}

// Package keystore owns the ed25519 key pair: generation, disk
// persistence and signing. Private keys never leave this package.
//
// The containment is the point: `Sign` takes bytes and returns a
// signature, so no caller ever holds key material and no key can end
// up in a log line, an error message or an HTTP request by accident.
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
//
// Note what is absent: no server is contacted and nothing is
// registered anywhere. An "account" is 32 random bytes — which is
// also why losing the file means losing the coins, with nobody to
// appeal to.
func Generate() (*Wallet, error) {
	// crypto/rand, never math/rand: this is key material, and a
	// predictable seed would let anyone recompute the private key.
	seed := make([]byte, ed25519.SeedSize)
	if _, err := rand.Read(seed); err != nil {
		return nil, fmt.Errorf("generate seed: %w", err)
	}
	return &Wallet{priv: ed25519.NewKeyFromSeed(seed)}, nil
}

// walletFile is the on-disk shape. Only the 32-byte seed is stored:
// the private key, the public key and the address are all derived
// from it, so persisting anything else would just be a second copy
// that could disagree. `address` is written for human convenience and
// deliberately ignored on load — the seed is the source of truth.
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
	// Check the length as well as the encoding: ed25519.NewKeyFromSeed
	// panics on a wrong-sized seed, so a truncated or edited file must
	// be turned into an error here rather than crashing the CLI.
	seed, err := hex.DecodeString(wf.Seed)
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("wallet file has a malformed seed")
	}
	return &Wallet{priv: ed25519.NewKeyFromSeed(seed)}, nil
}

// Save writes the wallet as JSON with owner-only permissions.
//
// The seed is stored in PLAINTEXT — acceptable only because these
// coins are worthless. A real wallet derives an encryption key from a
// passphrase (scrypt/argon2) and stores the seed sealed; 0600 alone
// protects against other users on the box, not against anything that
// can already read your files.
func (w *Wallet) Save(path string) error {
	// 0700 on the directory as well: a world-readable parent would let
	// others at least enumerate which wallets exist.
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

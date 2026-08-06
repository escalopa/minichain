package cli

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/escalopa/minichain/wallet/internal/tx"
)

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := New()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func TestKeygenRefusesOverwriteWithoutForce(t *testing.T) {
	file := filepath.Join(t.TempDir(), "wallet.json")

	out, err := run(t, "keygen", "--file", file)
	if err != nil {
		t.Fatalf("first keygen: %v", err)
	}
	if !strings.Contains(out, "address: ") {
		t.Errorf("keygen output missing address: %q", out)
	}

	if _, err := run(t, "keygen", "--file", file); err == nil {
		t.Fatal("second keygen without --force: want error, got nil")
	}
	if _, err := run(t, "keygen", "--file", file, "--force"); err != nil {
		t.Fatalf("keygen --force: %v", err)
	}
}

// The full send flow: the CLI must fetch the nonce, sign the payload
// with the wallet key, and submit a verifiable transaction.
func TestSendSignsVerifiably(t *testing.T) {
	file := filepath.Join(t.TempDir(), "wallet.json")
	if _, err := run(t, "keygen", "--file", file); err != nil {
		t.Fatal(err)
	}

	var received tx.Transaction
	mux := http.NewServeMux()
	mux.HandleFunc("GET /nonce/{addr}", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"nonce": 5})
	})
	mux.HandleFunc("POST /tx", func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusCreated)
	})
	node := httptest.NewServer(mux)
	t.Cleanup(node.Close)

	out, err := run(t, "send", "--file", file, "--node", node.URL, "--to", "bob-addr", "--amount", "9")
	if err != nil {
		t.Fatalf("send: %v (output: %s)", err, out)
	}

	if received.Nonce != 5 || received.To != "bob-addr" || received.Amount != 9 {
		t.Errorf("node received wrong tx: %+v", received)
	}
	pub, _ := hex.DecodeString(received.From)
	sig, _ := hex.DecodeString(received.Signature)
	if !ed25519.Verify(ed25519.PublicKey(pub), received.Payload(), sig) {
		t.Error("submitted transaction signature does not verify")
	}
}

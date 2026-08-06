package nodeclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/escalopa/minichain/wallet/internal/tx"
)

// fakeNode imitates the Rust node's API surface.
func fakeNode(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /balance/{addr}", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"address": r.PathValue("addr"), "balance": 42})
	})
	mux.HandleFunc("GET /nonce/{addr}", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"address": r.PathValue("addr"), "nonce": 7})
	})
	mux.HandleFunc("POST /tx", func(w http.ResponseWriter, r *http.Request) {
		var got tx.Transaction
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("node received undecodable tx: %v", err)
		}
		if got.Signature == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid signature"})
			return
		}
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("POST /mine", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"index": 3, "hash": "000abc"})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestBalanceAndNonce(t *testing.T) {
	c := New(fakeNode(t).URL)

	balance, err := c.Balance(context.Background(), "alice")
	if err != nil || balance != 42 {
		t.Errorf("balance: want 42, got %d err=%v", balance, err)
	}
	nonce, err := c.Nonce(context.Background(), "alice")
	if err != nil || nonce != 7 {
		t.Errorf("nonce: want 7, got %d err=%v", nonce, err)
	}
}

func TestSubmitAcceptedAndRejected(t *testing.T) {
	c := New(fakeNode(t).URL)

	ok := &tx.Transaction{From: "a", To: "b", Amount: 1, Signature: "deadbeef"}
	if err := c.Submit(context.Background(), ok); err != nil {
		t.Errorf("valid tx: want accept, got %v", err)
	}

	bad := &tx.Transaction{From: "a", To: "b", Amount: 1}
	err := c.Submit(context.Background(), bad)
	if err == nil || !strings.Contains(err.Error(), "invalid signature") {
		t.Errorf("unsigned tx: want node error surfaced, got %v", err)
	}
}

func TestMine(t *testing.T) {
	c := New(fakeNode(t).URL)

	index, hash, err := c.Mine(context.Background(), "miner")
	if err != nil || index != 3 || hash != "000abc" {
		t.Errorf("mine: want block 3/000abc, got %d/%s err=%v", index, hash, err)
	}
}

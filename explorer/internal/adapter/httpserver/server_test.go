package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/escalopa/minichain/explorer/internal/adapter/memstore"
	"github.com/escalopa/minichain/explorer/internal/core/domain"
	"github.com/escalopa/minichain/explorer/internal/core/service"
)

func fixture() []domain.Block {
	return []domain.Block{
		{Index: 0, Hash: "aaa0"},
		{
			Index: 1, Hash: "aaa1", PrevHash: "aaa0",
			Transactions: []domain.Transaction{
				{From: domain.Coinbase, To: "alice", Amount: 50},
			},
		},
		{
			Index: 2, Hash: "aaa2", PrevHash: "aaa1",
			Transactions: []domain.Transaction{
				{From: domain.Coinbase, To: "alice", Amount: 50},
				{From: "alice", To: "bob", Amount: 30, Nonce: 0},
			},
		},
	}
}

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	repo := memstore.New()
	repo.Update(fixture())
	srv := httptest.NewServer(New(service.NewExplorer(repo)).Handler())
	t.Cleanup(srv.Close)
	return srv
}

func getJSON(t *testing.T, url string, wantStatus int) map[string]any {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		t.Fatalf("GET %s: want %d, got %d", url, wantStatus, resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body
}

func TestAPIBlocks(t *testing.T) {
	srv := testServer(t)

	body := getJSON(t, srv.URL+"/api/blocks?limit=2", http.StatusOK)
	if body["height"].(float64) != 3 {
		t.Errorf("want height 3, got %v", body["height"])
	}
	blocks := body["blocks"].([]any)
	if len(blocks) != 2 {
		t.Errorf("want 2 blocks with limit=2, got %d", len(blocks))
	}
}

func TestAPIBlockByRefAndNotFound(t *testing.T) {
	srv := testServer(t)

	resp, err := http.Get(srv.URL + "/api/blocks/aaa1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var block domain.Block
	if err := json.NewDecoder(resp.Body).Decode(&block); err != nil {
		t.Fatal(err)
	}
	if block.Index != 1 {
		t.Errorf("want block 1, got %d", block.Index)
	}

	getJSON(t, srv.URL+"/api/blocks/nope", http.StatusNotFound)
}

func TestAPIAddress(t *testing.T) {
	srv := testServer(t)

	body := getJSON(t, srv.URL+"/api/address/alice", http.StatusOK)
	if body["balance"].(float64) != 70 {
		t.Errorf("want balance 70, got %v", body["balance"])
	}
}

func TestSearchRedirects(t *testing.T) {
	srv := testServer(t)
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	for query, wantPrefix := range map[string]string{
		"2":     "/block/",
		"aaa1":  "/block/",
		"alice": "/address/",
	} {
		resp, err := client.Get(srv.URL + "/search?q=" + query)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		loc := resp.Header.Get("Location")
		if !strings.HasPrefix(loc, wantPrefix) {
			t.Errorf("search %q: want redirect to %s..., got %s", query, wantPrefix, loc)
		}
	}
}

func TestHomePageRenders(t *testing.T) {
	srv := testServer(t)

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	html := string(raw)
	for _, want := range []string{"minichain explorer", "height: 3", "aaa2"} {
		if !strings.Contains(html, want) {
			t.Errorf("home page missing %q", want)
		}
	}
}

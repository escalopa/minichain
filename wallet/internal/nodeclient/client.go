// Package nodeclient talks to the node's HTTP API.
package nodeclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/escalopa/minichain/wallet/internal/tx"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Balance returns the confirmed balance of an address.
func (c *Client) Balance(ctx context.Context, addr string) (uint64, error) {
	var out struct {
		Balance uint64 `json:"balance"`
	}
	if err := c.getJSON(ctx, "/balance/"+addr, &out); err != nil {
		return 0, err
	}
	return out.Balance, nil
}

// Nonce returns the next expected nonce for an address.
func (c *Client) Nonce(ctx context.Context, addr string) (uint64, error) {
	var out struct {
		Nonce uint64 `json:"nonce"`
	}
	if err := c.getJSON(ctx, "/nonce/"+addr, &out); err != nil {
		return 0, err
	}
	return out.Nonce, nil
}

// Submit sends a signed transaction to the node's mempool.
func (c *Client) Submit(ctx context.Context, t *tx.Transaction) error {
	resp, err := c.postJSON(ctx, "/tx", t)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var out struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err == nil && out.Error != "" {
			return fmt.Errorf("node rejected transaction: %s", out.Error)
		}
		return fmt.Errorf("node returned %s", resp.Status)
	}
	return nil
}

// Mine asks the node to mine the mempool, rewarding the miner address.
// Returns the index and hash of the new block.
func (c *Client) Mine(ctx context.Context, miner string) (uint64, string, error) {
	resp, err := c.postJSON(ctx, "/mine", map[string]string{"miner": miner})
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("node returned %s", resp.Status)
	}
	var out struct {
		Index uint64 `json:"index"`
		Hash  string `json:"hash"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, "", fmt.Errorf("decode block: %w", err)
	}
	return out.Index, out.Hash, nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("node returned %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) postJSON(ctx context.Context, path string, body any) (*http.Response, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.http.Do(req)
}

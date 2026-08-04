// Package nodeclient is the driven adapter that implements
// port.ChainSource over the node's HTTP API.
package nodeclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/escalopa/minichain/explorer/internal/core/domain"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Blocks fetches the full chain from the node.
func (c *Client) Blocks(ctx context.Context) ([]domain.Block, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/blocks", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("node returned %s", resp.Status)
	}
	var blocks []domain.Block
	if err := json.NewDecoder(resp.Body).Decode(&blocks); err != nil {
		return nil, fmt.Errorf("decode blocks: %w", err)
	}
	return blocks, nil
}

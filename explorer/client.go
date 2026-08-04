package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// NodeClient is a thin HTTP client for the Rust node's API.
type NodeClient struct {
	baseURL string
	http    *http.Client
}

func NewNodeClient(baseURL string) *NodeClient {
	return &NodeClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Blocks fetches the full chain from the node.
func (c *NodeClient) Blocks(ctx context.Context) ([]Block, error) {
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
	var blocks []Block
	if err := json.NewDecoder(resp.Body).Decode(&blocks); err != nil {
		return nil, fmt.Errorf("decode blocks: %w", err)
	}
	return blocks, nil
}

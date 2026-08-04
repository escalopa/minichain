// Package port defines the driven ports of the hexagon: interfaces
// the core needs the outside world to implement. The core depends on
// these abstractions, never on concrete adapters.
package port

import (
	"context"

	"github.com/escalopa/minichain/explorer/internal/core/domain"
)

// ChainSource fetches the chain from somewhere — in production, the
// node's HTTP API.
type ChainSource interface {
	Blocks(ctx context.Context) ([]domain.Block, error)
}

// ChainRepository stores a snapshot of the chain and answers lookups.
type ChainRepository interface {
	// Update atomically replaces the snapshot.
	Update(blocks []domain.Block)
	Height() int
	// Recent returns up to limit blocks, newest first.
	Recent(limit int) []domain.Block
	// ByRef finds a block by decimal index or by full hash.
	ByRef(ref string) (domain.Block, bool)
	// Snapshot returns the whole chain; callers must not mutate it.
	Snapshot() []domain.Block
}

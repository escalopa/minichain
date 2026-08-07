// Package port defines the driven ports of the hexagon: interfaces
// the core needs the outside world to implement. The core depends on
// these abstractions, never on concrete adapters.
//
// This is dependency inversion in practice: without it, the syncer
// would import the HTTP client, and testing it would mean standing up
// a fake server. With it, a struct with three fields is enough.
//
// Note the asymmetry with the *driving* side: the HTTP adapter
// declares the interface it consumes next to its own code
// (`httpserver.Explorer`), which is the idiomatic Go convention —
// interfaces belong to the consumer. Driven ports live here because
// the consumer in that direction is the core itself.
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

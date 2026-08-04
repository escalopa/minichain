// Package memstore is the driven adapter that implements
// port.ChainRepository as an in-memory snapshot with lookup indexes.
package memstore

import (
	"strconv"
	"strings"
	"sync"

	"github.com/escalopa/minichain/explorer/internal/core/domain"
)

type Store struct {
	mu     sync.RWMutex
	blocks []domain.Block
	byHash map[string]int // block hash -> position in blocks
}

func New() *Store {
	return &Store{byHash: map[string]int{}}
}

// Update atomically replaces the snapshot: readers always see either
// the old chain or the new one, never a half-updated mix.
func (s *Store) Update(blocks []domain.Block) {
	byHash := make(map[string]int, len(blocks))
	for i, b := range blocks {
		byHash[b.Hash] = i
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.blocks = blocks
	s.byHash = byHash
}

func (s *Store) Height() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.blocks)
}

// Recent returns up to limit blocks, newest first.
func (s *Store) Recent(limit int) []domain.Block {
	s.mu.RLock()
	defer s.mu.RUnlock()

	n := min(limit, len(s.blocks))
	out := make([]domain.Block, 0, n)
	for i := len(s.blocks) - 1; i >= len(s.blocks)-n; i-- {
		out = append(out, s.blocks[i])
	}
	return out
}

// ByRef finds a block by decimal index or by full hash.
func (s *Store) ByRef(ref string) (domain.Block, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if idx, err := strconv.ParseUint(ref, 10, 64); err == nil {
		if idx < uint64(len(s.blocks)) {
			return s.blocks[idx], true
		}
		return domain.Block{}, false
	}
	if i, ok := s.byHash[strings.ToLower(ref)]; ok {
		return s.blocks[i], true
	}
	return domain.Block{}, false
}

// Snapshot returns the current chain; callers must not mutate it.
func (s *Store) Snapshot() []domain.Block {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.blocks
}

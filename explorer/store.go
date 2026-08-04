package main

import (
	"strconv"
	"strings"
	"sync"
)

// Store is an in-memory snapshot of the chain with lookup indexes.
// The poller replaces the snapshot; readers see a consistent view.
type Store struct {
	mu     sync.RWMutex
	blocks []Block
	byHash map[string]int // block hash -> position in blocks
}

func NewStore() *Store {
	return &Store{byHash: map[string]int{}}
}

// Update replaces the snapshot with a fresh copy of the chain.
func (s *Store) Update(blocks []Block) {
	byHash := make(map[string]int, len(blocks))
	for i, b := range blocks {
		byHash[b.Hash] = i
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.blocks = blocks
	s.byHash = byHash
}

// Height is the number of blocks in the snapshot.
func (s *Store) Height() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.blocks)
}

// Recent returns up to limit blocks, newest first.
func (s *Store) Recent(limit int) []Block {
	s.mu.RLock()
	defer s.mu.RUnlock()

	n := min(limit, len(s.blocks))
	out := make([]Block, 0, n)
	for i := len(s.blocks) - 1; i >= len(s.blocks)-n; i-- {
		out = append(out, s.blocks[i])
	}
	return out
}

// BlockByRef finds a block by decimal index or by full hash.
func (s *Store) BlockByRef(ref string) (Block, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if idx, err := strconv.ParseUint(ref, 10, 64); err == nil {
		if idx < uint64(len(s.blocks)) {
			return s.blocks[idx], true
		}
		return Block{}, false
	}
	if i, ok := s.byHash[strings.ToLower(ref)]; ok {
		return s.blocks[i], true
	}
	return Block{}, false
}

// AddressInfo computes the balance and transaction history of an
// address from the cached chain — the same account-model arithmetic
// the node uses.
func (s *Store) AddressInfo(addr string) (balance uint64, history []TxRef) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var in, out uint64
	for _, b := range s.blocks {
		for _, tx := range b.Transactions {
			if tx.To != addr && tx.From != addr {
				continue
			}
			if tx.To == addr {
				in += tx.Amount
			}
			if tx.From == addr {
				out += tx.Amount
			}
			history = append(history, TxRef{
				Transaction: tx,
				BlockIndex:  b.Index,
				BlockHash:   b.Hash,
			})
		}
	}
	if in > out {
		balance = in - out
	}
	return balance, history
}

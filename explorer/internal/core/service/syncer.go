package service

import (
	"context"
	"log"
	"time"

	"github.com/escalopa/minichain/explorer/internal/core/port"
)

// Syncer keeps the repository in step with the chain source.
// Fetching the whole chain every tick is fine for an educational
// network; a real explorer would track the tip and backfill.
type Syncer struct {
	source   port.ChainSource
	repo     port.ChainRepository
	interval time.Duration
}

func NewSyncer(source port.ChainSource, repo port.ChainRepository, interval time.Duration) *Syncer {
	return &Syncer{source: source, repo: repo, interval: interval}
}

// SyncOnce fetches the chain and replaces the snapshot if the height
// changed. Returns whether an update happened.
//
// Split out from Run so the polling *policy* (a ticker, logging, a
// cancellable loop) can be tested separately from the *decision*
// (fetch, compare, replace) — the latter needs no clock at all.
func (s *Syncer) SyncOnce(ctx context.Context) (bool, error) {
	blocks, err := s.source.Blocks(ctx)
	if err != nil {
		return false, err
	}
	// Height is the cheap change detector. It misses a reorg that
	// swaps blocks without changing the length — harmless for a
	// read-only explorer, and a real one would compare tip hashes.
	if len(blocks) == s.repo.Height() {
		return false, nil
	}
	s.repo.Update(blocks)
	return true, nil
}

// Run polls until the context is cancelled.
func (s *Syncer) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		updated, err := s.SyncOnce(ctx)
		switch {
		case err != nil:
			log.Printf("sync chain: %v", err)
		case updated:
			log.Printf("synced %d blocks", s.repo.Height())
		}

		// Select at the *end* of the body, so the first sync happens
		// immediately on startup rather than one interval later; and
		// cancellation is checked between ticks, so shutdown does not
		// wait for the timer.
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

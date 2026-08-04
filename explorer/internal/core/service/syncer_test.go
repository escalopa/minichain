package service

import (
	"context"
	"errors"
	"testing"

	"github.com/escalopa/minichain/explorer/internal/core/domain"
)

// The core is tested against fakes of its own ports — no adapters,
// no HTTP, no locks.
type fakeSource struct {
	blocks []domain.Block
	err    error
	calls  int
}

func (f *fakeSource) Blocks(context.Context) ([]domain.Block, error) {
	f.calls++
	return f.blocks, f.err
}

type fakeRepo struct {
	blocks  []domain.Block
	updates int
}

func (f *fakeRepo) Update(blocks []domain.Block) {
	f.blocks = blocks
	f.updates++
}
func (f *fakeRepo) Height() int                       { return len(f.blocks) }
func (f *fakeRepo) Recent(int) []domain.Block         { return nil }
func (f *fakeRepo) ByRef(string) (domain.Block, bool) { return domain.Block{}, false }
func (f *fakeRepo) Snapshot() []domain.Block          { return f.blocks }

func TestSyncOnceUpdatesOnNewHeight(t *testing.T) {
	source := &fakeSource{blocks: []domain.Block{{Index: 0}, {Index: 1}}}
	repo := &fakeRepo{}
	syncer := NewSyncer(source, repo, 0)

	updated, err := syncer.SyncOnce(context.Background())
	if err != nil || !updated {
		t.Fatalf("want update, got updated=%v err=%v", updated, err)
	}
	if repo.updates != 1 || repo.Height() != 2 {
		t.Errorf("want 1 update to height 2, got %d updates, height %d", repo.updates, repo.Height())
	}
}

func TestSyncOnceSkipsWhenHeightUnchanged(t *testing.T) {
	source := &fakeSource{blocks: []domain.Block{{Index: 0}}}
	repo := &fakeRepo{blocks: []domain.Block{{Index: 0}}}
	syncer := NewSyncer(source, repo, 0)

	updated, err := syncer.SyncOnce(context.Background())
	if err != nil || updated {
		t.Fatalf("want no update, got updated=%v err=%v", updated, err)
	}
	if repo.updates != 0 {
		t.Errorf("want 0 updates, got %d", repo.updates)
	}
}

func TestSyncOnceReportsSourceError(t *testing.T) {
	source := &fakeSource{err: errors.New("node is down")}
	repo := &fakeRepo{}
	syncer := NewSyncer(source, repo, 0)

	if _, err := syncer.SyncOnce(context.Background()); err == nil {
		t.Fatal("want error from source, got nil")
	}
	if repo.updates != 0 {
		t.Errorf("want repo untouched on error, got %d updates", repo.updates)
	}
}

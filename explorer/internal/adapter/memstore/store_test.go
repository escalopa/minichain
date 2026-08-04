package memstore

import (
	"testing"

	"github.com/escalopa/minichain/explorer/internal/core/domain"
)

func fixture() []domain.Block {
	return []domain.Block{
		{Index: 0, Hash: "aaa0"},
		{Index: 1, Hash: "aaa1", PrevHash: "aaa0"},
		{Index: 2, Hash: "aaa2", PrevHash: "aaa1"},
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s := New()
	s.Update(fixture())
	return s
}

func TestRecentReturnsNewestFirst(t *testing.T) {
	s := newTestStore(t)

	recent := s.Recent(2)
	if len(recent) != 2 {
		t.Fatalf("want 2 blocks, got %d", len(recent))
	}
	if recent[0].Index != 2 || recent[1].Index != 1 {
		t.Errorf("want newest first [2, 1], got [%d, %d]", recent[0].Index, recent[1].Index)
	}
}

func TestByRef(t *testing.T) {
	s := newTestStore(t)

	if b, ok := s.ByRef("1"); !ok || b.Hash != "aaa1" {
		t.Errorf("by index: want aaa1, got %+v ok=%v", b, ok)
	}
	if b, ok := s.ByRef("aaa2"); !ok || b.Index != 2 {
		t.Errorf("by hash: want index 2, got %+v ok=%v", b, ok)
	}
	if _, ok := s.ByRef("99"); ok {
		t.Error("out-of-range index should not resolve")
	}
	if _, ok := s.ByRef("deadbeef"); ok {
		t.Error("unknown hash should not resolve")
	}
}

func TestUpdateReplacesSnapshot(t *testing.T) {
	s := newTestStore(t)

	s.Update(fixture()[:1])
	if s.Height() != 1 {
		t.Errorf("want height 1 after shrinking update, got %d", s.Height())
	}
	if _, ok := s.ByRef("aaa2"); ok {
		t.Error("stale hash index should be gone after update")
	}
}

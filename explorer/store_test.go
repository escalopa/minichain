package main

import "testing"

func chainFixture() []Block {
	return []Block{
		{Index: 0, Hash: "aaa0", Transactions: nil},
		{
			Index: 1, Hash: "aaa1", PrevHash: "aaa0",
			Transactions: []Transaction{
				{From: coinbase, To: "alice", Amount: 50},
			},
		},
		{
			Index: 2, Hash: "aaa2", PrevHash: "aaa1",
			Transactions: []Transaction{
				{From: coinbase, To: "alice", Amount: 50},
				{From: "alice", To: "bob", Amount: 30, Nonce: 0},
			},
		},
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s := NewStore()
	s.Update(chainFixture())
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

func TestBlockByRef(t *testing.T) {
	s := newTestStore(t)

	if b, ok := s.BlockByRef("1"); !ok || b.Hash != "aaa1" {
		t.Errorf("by index: want aaa1, got %+v ok=%v", b, ok)
	}
	if b, ok := s.BlockByRef("aaa2"); !ok || b.Index != 2 {
		t.Errorf("by hash: want index 2, got %+v ok=%v", b, ok)
	}
	if _, ok := s.BlockByRef("99"); ok {
		t.Error("out-of-range index should not resolve")
	}
	if _, ok := s.BlockByRef("deadbeef"); ok {
		t.Error("unknown hash should not resolve")
	}
}

func TestAddressInfo(t *testing.T) {
	s := newTestStore(t)

	balance, history := s.AddressInfo("alice")
	if balance != 70 { // 50 + 50 - 30
		t.Errorf("alice balance: want 70, got %d", balance)
	}
	if len(history) != 3 {
		t.Errorf("alice history: want 3 txs, got %d", len(history))
	}

	balance, history = s.AddressInfo("bob")
	if balance != 30 {
		t.Errorf("bob balance: want 30, got %d", balance)
	}
	if len(history) != 1 || history[0].BlockIndex != 2 {
		t.Errorf("bob history: want 1 tx in block 2, got %+v", history)
	}

	if balance, _ = s.AddressInfo("nobody"); balance != 0 {
		t.Errorf("unknown address balance: want 0, got %d", balance)
	}
}

func TestUpdateReplacesSnapshot(t *testing.T) {
	s := newTestStore(t)

	s.Update(chainFixture()[:1])
	if s.Height() != 1 {
		t.Errorf("want height 1 after shrinking update, got %d", s.Height())
	}
	if _, ok := s.BlockByRef("aaa2"); ok {
		t.Error("stale hash index should be gone after update")
	}
}

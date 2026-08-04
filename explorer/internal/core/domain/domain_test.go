package domain

import "testing"

func fixture() []Block {
	return []Block{
		{Index: 0, Hash: "aaa0"},
		{
			Index: 1, Hash: "aaa1", PrevHash: "aaa0",
			Transactions: []Transaction{
				{From: Coinbase, To: "alice", Amount: 50},
			},
		},
		{
			Index: 2, Hash: "aaa2", PrevHash: "aaa1",
			Transactions: []Transaction{
				{From: Coinbase, To: "alice", Amount: 50},
				{From: "alice", To: "bob", Amount: 30, Nonce: 0},
			},
		},
	}
}

func TestAddressInfo(t *testing.T) {
	balance, history := AddressInfo(fixture(), "alice")
	if balance != 70 { // 50 + 50 - 30
		t.Errorf("alice balance: want 70, got %d", balance)
	}
	if len(history) != 3 {
		t.Errorf("alice history: want 3 txs, got %d", len(history))
	}

	balance, history = AddressInfo(fixture(), "bob")
	if balance != 30 {
		t.Errorf("bob balance: want 30, got %d", balance)
	}
	if len(history) != 1 || history[0].BlockIndex != 2 {
		t.Errorf("bob history: want 1 tx in block 2, got %+v", history)
	}

	if balance, _ = AddressInfo(fixture(), "nobody"); balance != 0 {
		t.Errorf("unknown address balance: want 0, got %d", balance)
	}
}

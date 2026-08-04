// Package domain holds the pure model of the chain as the explorer
// sees it. No I/O, no locks — just data and arithmetic.
package domain

// Coinbase is the pseudo-sender of mining rewards.
const Coinbase = "COINBASE"

// Transaction mirrors the node's JSON representation.
type Transaction struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    uint64 `json:"amount"`
	Nonce     uint64 `json:"nonce"`
	Timestamp uint64 `json:"timestamp"`
	Signature string `json:"signature"`
}

// Block mirrors the node's JSON representation.
type Block struct {
	Index        uint64        `json:"index"`
	Timestamp    uint64        `json:"timestamp"`
	PrevHash     string        `json:"prev_hash"`
	Nonce        uint64        `json:"nonce"`
	Transactions []Transaction `json:"transactions"`
	Hash         string        `json:"hash"`
}

// TxRef is a transaction together with the block it was mined in.
type TxRef struct {
	Transaction
	BlockIndex uint64 `json:"block_index"`
	BlockHash  string `json:"block_hash"`
}

// AddressInfo computes the balance and transaction history of an
// address over a chain — the same account-model arithmetic the node
// uses: incoming minus outgoing across all blocks.
func AddressInfo(blocks []Block, addr string) (balance uint64, history []TxRef) {
	var in, out uint64
	for _, b := range blocks {
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

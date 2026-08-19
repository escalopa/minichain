// Package domain holds the pure model of the chain as the explorer
// sees it. No I/O, no locks — just data and arithmetic.
//
// This is the centre of the hexagon: it imports nothing from the rest
// of the application, which is what makes the rules below testable
// without a node, a store or an HTTP server anywhere in sight.
package domain

// Coinbase is the pseudo-sender of mining rewards.
const Coinbase = "COINBASE"

// Transaction mirrors the node's JSON representation.
//
// The json tags are a wire contract with the Rust node's serde output,
// not decoration: snake_case there, exported fields here. A rename on
// either side silently produces zero values rather than an error, so
// the fixtures in the tests double as a canary.
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
//
// Deliberately duplicating the node's rule rather than proxying to
// GET /balance: an explorer that recomputes from the blocks it holds
// can serve history and balance from one pass, and any divergence
// from the node is a real signal that something is wrong.
//
// A pure function over a slice — no receiver, no state — so the
// repository stays a dumb cache and this logic is trivially testable.
func AddressInfo(blocks []Block, addr string) (balance uint64, history []TxRef) {
	// Sum both directions separately, then subtract once at the end:
	// uint64 would underflow to a colossal number if a partial sum
	// ever went negative mid-loop.
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
	// A healthy chain can never spend more than it received, so this
	// guard should be dead code — it exists so that a corrupt chain
	// yields a zero balance instead of an absurd one.
	if in > out {
		balance = in - out
	}
	return balance, history
}

// Package tx builds transactions in the node's wire format.
package tx

import "fmt"

// Transaction mirrors the node's JSON representation.
type Transaction struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    uint64 `json:"amount"`
	Nonce     uint64 `json:"nonce"`
	Timestamp uint64 `json:"timestamp"`
	Signature string `json:"signature"`
}

// Payload is the exact byte sequence the node verifies the signature
// against; it must match the Rust side character for character:
// `format!("{}|{}|{}|{}|{}", from, to, amount, nonce, timestamp)`.
func (t *Transaction) Payload() []byte {
	return fmt.Appendf(nil, "%s|%s|%d|%d|%d", t.From, t.To, t.Amount, t.Nonce, t.Timestamp)
}

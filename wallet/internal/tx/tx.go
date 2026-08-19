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
//
// THE contract of this project — the one place where Go and Rust must
// agree exactly. Signing is over these bytes, not over the JSON, on
// purpose: JSON gives no ordering or whitespace guarantees, so two
// encoders could produce different bytes for the same transaction and
// every signature would break. A fixed string format has exactly one
// encoding.
//
// Consequences: the signature covers the amount, the recipient AND
// the nonce, so none of them can be altered in flight; and the
// tx_test.go case pins this string so a well-meaning refactor here
// fails loudly instead of silently invalidating every signature.
func (t *Transaction) Payload() []byte {
	return fmt.Appendf(nil, "%s|%s|%d|%d|%d", t.From, t.To, t.Amount, t.Nonce, t.Timestamp)
}

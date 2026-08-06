package tx

import "testing"

// The payload must match the Rust node byte for byte — this string is
// the contract.
func TestPayloadMatchesNodeFormat(t *testing.T) {
	tr := Transaction{
		From:      "aa",
		To:        "bb",
		Amount:    15,
		Nonce:     2,
		Timestamp: 1700000000000,
	}
	want := "aa|bb|15|2|1700000000000"
	if got := string(tr.Payload()); got != want {
		t.Errorf("payload: want %q, got %q", want, got)
	}
}

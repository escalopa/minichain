package main

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

const coinbase = "COINBASE"

// TxRef is a transaction together with the block it was mined in.
type TxRef struct {
	Transaction
	BlockIndex uint64 `json:"block_index"`
	BlockHash  string `json:"block_hash"`
}

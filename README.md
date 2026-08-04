# minichain

An educational blockchain from scratch: **Rust** core, **Go** tooling.

## Architecture

```
minichain/
├── node/       Rust — core: blocks, PoW, mempool, P2P, JSON-RPC API
├── explorer/   Go   — block explorer: REST API + web page on top of the node
└── wallet/     Go   — CLI wallet: keys, transaction signing, submission
```

The node is the single source of truth. Go services talk to it only
through the HTTP API, just like a real explorer talks to a real node.

## Roadmap

- [x] **Phase 1 — block chain** (Rust): block, SHA-256, Proof-of-Work,
      chain validation, tests
- [x] **Phase 2 — transactions** (Rust): ed25519 keys, transaction
      signing/verification, mempool, coinbase reward, balances (account model)
- [ ] **Phase 3 — HTTP API** (Rust, axum): `GET /blocks`, `GET /balance/:addr`,
      `POST /tx`, `POST /mine`
- [ ] **Phase 4 — explorer** (Go): polling the node, in-memory/SQLite cache,
      REST + a simple HTML page with block list and search
- [ ] **Phase 5 — wallet** (Go, cobra): `wallet keygen | balance | send`
- [ ] **Phase 6 — network** (Rust, libp2p): multiple nodes, block gossip,
      fork resolution via the longest-chain rule

## Running

```sh
cd node && cargo run       # mine a demo chain
cd node && cargo test      # core tests
```

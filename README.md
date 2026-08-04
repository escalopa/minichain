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
- [x] **Phase 3 — HTTP API** (Rust, axum): `GET /blocks`, `GET /balance/:addr`,
      `POST /tx`, `POST /mine`
- [x] **Phase 4 — explorer** (Go): polling the node, in-memory cache,
      REST + HTML pages with block list, address history and search
- [ ] **Phase 5 — wallet** (Go, cobra): `wallet keygen | balance | send`
- [ ] **Phase 6 — network** (Rust, libp2p): multiple nodes, block gossip,
      fork resolution via the longest-chain rule

## Running

```sh
cd node && cargo run       # start the node (PORT=3000, DIFFICULTY=4 by default)
cd node && cargo test      # core + API tests

cd explorer && go run .    # start the explorer (PORT=8080, NODE_URL=http://localhost:3000)
cd explorer && go test ./... # explorer tests
```

Open http://localhost:8080 for the explorer UI: recent blocks, block and
address pages, search by block index, hash or address. JSON mirror at
`/api/blocks`, `/api/blocks/{ref}`, `/api/address/{addr}`.

## Node HTTP API

| Method | Path                | Description                                      |
|--------|---------------------|--------------------------------------------------|
| GET    | `/blocks`           | full chain                                       |
| GET    | `/mempool`          | pending transactions                             |
| GET    | `/balance/{addr}`   | confirmed balance of an address                  |
| GET    | `/nonce/{addr}`     | next expected nonce (fetch before signing a tx)  |
| POST   | `/tx`               | submit a signed transaction (JSON body)          |
| POST   | `/mine`             | mine the mempool: `{"miner": "<address>"}`       |

```sh
curl localhost:3000/blocks
curl -X POST localhost:3000/mine \
  -H 'content-type: application/json' -d '{"miner":"<address>"}'
curl localhost:3000/balance/<address>
```

The node never sees private keys: clients fetch the nonce, sign locally,
and submit the ready-made transaction to `/tx`.

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
- [x] **Phase 5 — wallet** (Go, cobra): `wallet keygen | address | balance | send | mine`
- [x] **Phase 6 — network** (Rust, libp2p): multiple nodes, chain gossip,
      fork resolution via the longest-chain rule

## Running

```sh
cd node && cargo run       # start the node (PORT=3000, DIFFICULTY=4 by default)
cd node && cargo test      # core + API tests

cd explorer && go run ./cmd/explorer  # start the explorer (PORT=8080, NODE_URL=http://localhost:3000)
cd explorer && go test ./...          # explorer tests
```

Open http://localhost:8080 for the explorer UI: recent blocks, block and
address pages, search by block index, hash or address. JSON mirror at
`/api/blocks`, `/api/blocks/{ref}`, `/api/address/{addr}`.

The explorer follows hexagonal architecture (ports & adapters):

```
explorer/
├── cmd/explorer/            composition root — the only place that wires adapters
└── internal/
    ├── core/                the hexagon: no imports of adapters or net/http
    │   ├── domain/          Block, Transaction, pure balance arithmetic
    │   ├── port/            driven ports: ChainSource, ChainRepository
    │   └── service/         Syncer (polling) and Explorer (queries)
    └── adapter/
        ├── nodeclient/      driven:  node HTTP API  → port.ChainSource
        ├── memstore/        driven:  in-memory cache → port.ChainRepository
        └── httpserver/      driving: HTML + JSON API → core
```

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

### Concurrency model

Chain state lives behind a single `Arc<Mutex<Blockchain>>`, so every
operation is linearizable. Mining is **optimistic**: `POST /mine` takes
a snapshot of the tip under a short lock, runs the proof-of-work on a
blocking thread with no lock held, then re-takes the lock and appends —
but only if the tip has not moved. If another miner (or an adopted
longer chain from the network) won the race meanwhile, the block is
discarded and mining restarts on the new tip. Reads and transaction
submissions stay fast even while blocks are being mined.

## Wallet

```sh
cd wallet && go build -o wallet ./cmd/wallet

./wallet keygen                       # generate keys (~/.minichain/wallet.json, 0600)
./wallet address                      # print your address
./wallet mine                         # mine the mempool, take the 50-coin reward
./wallet balance [address]            # your balance, or anyone's
./wallet send --to <addr> --amount 15 # sign locally, submit to the node
```

`--node` / `NODE_URL` selects the node, `--file` / `WALLET_FILE` the key
file. Signing happens entirely client-side with Go's `crypto/ed25519`;
the node (Rust, ed25519-dalek) verifies the same payload byte for byte.

## Running a network

Every node runs a libp2p swarm: gossipsub for exchanging chains, mDNS
for discovering peers on the local network, plus explicit bootstrap
peers via `PEERS` for when multicast is unavailable.

```sh
# terminal 1 — first node
PORT=3901 P2P_PORT=4801 cargo run

# terminal 2 — second node, bootstrapped from the first
PORT=3902 P2P_PORT=4802 PEERS=/ip4/127.0.0.1/tcp/4801 cargo run
```

Mine on one node and watch the other adopt the chain within a couple
of seconds. The consensus is deliberately naive Nakamoto: whenever a
node's height grows it publishes its whole chain, and every node adopts
any incoming chain that is valid and strictly longer than its own
(`Blockchain::replace_if_longer`). Transactions already included in an
adopted chain are pruned from the mempool.

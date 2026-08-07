use std::sync::{Arc, Mutex};

use axum::{
    extract::{Path, State},
    http::StatusCode,
    routing::{get, post},
    Json, Router,
};
use serde::{Deserialize, Serialize};

use crate::block::Block;
use crate::chain::Blockchain;
use crate::transaction::Transaction;

/// The chain is shared between request handlers. A std Mutex is enough
/// here: we never hold the lock across an await point.
///
/// Two consequences worth knowing:
///
/// * Every operation on the chain is serialised, so the API is
///   linearizable — concurrent mines and submissions interleave in
///   some single global order, never partially.
/// * `std::sync::Mutex` blocks the OS thread. That is correct only
///   because no guard outlives an `.await`; holding one across a
///   suspension point could deadlock the runtime. The expensive
///   proof-of-work therefore runs on a blocking thread with the lock
///   released — see `mine`.
pub type SharedChain = Arc<Mutex<Blockchain>>;

#[derive(Serialize)]
struct ErrorBody {
    error: String,
}

#[derive(Serialize)]
struct BalanceBody {
    address: String,
    balance: u64,
}

#[derive(Serialize)]
struct NonceBody {
    address: String,
    nonce: u64,
}

#[derive(Deserialize)]
struct MineBody {
    miner: String,
}

pub fn router(chain: SharedChain) -> Router {
    Router::new()
        .route("/blocks", get(get_blocks))
        .route("/mempool", get(get_mempool))
        .route("/balance/{address}", get(get_balance))
        .route("/nonce/{address}", get(get_nonce))
        .route("/tx", post(submit_tx))
        .route("/mine", post(mine))
        .with_state(chain)
}

async fn get_blocks(State(chain): State<SharedChain>) -> Json<Vec<Block>> {
    Json(chain.lock().unwrap().blocks.clone())
}

async fn get_mempool(State(chain): State<SharedChain>) -> Json<Vec<Transaction>> {
    Json(chain.lock().unwrap().mempool.clone())
}

async fn get_balance(
    State(chain): State<SharedChain>,
    Path(address): Path<String>,
) -> Json<BalanceBody> {
    let balance = chain.lock().unwrap().balance_of(&address);
    Json(BalanceBody { address, balance })
}

/// Clients call this before signing: the transaction they build must
/// carry exactly this nonce to be accepted.
///
/// Inherently racy, and that is fine: between this call and `/tx`
/// another transfer from the same sender may claim the nonce, and the
/// submission is then rejected with a clear message. Ethereum behaves
/// the same way — the client, not the node, owns the retry.
async fn get_nonce(
    State(chain): State<SharedChain>,
    Path(address): Path<String>,
) -> Json<NonceBody> {
    let nonce = chain.lock().unwrap().next_nonce(&address);
    Json(NonceBody { address, nonce })
}

/// Accepts an already-signed transaction. The node never sees private
/// keys — signing happens client-side (in the Go wallet).
///
/// This is why the wallet can be written in another language and why
/// a node can be a stranger: it validates authority, it never holds
/// it. Rejections come back as 400 with the node's own reason, so the
/// wallet can print something actionable instead of "request failed".
async fn submit_tx(
    State(chain): State<SharedChain>,
    Json(tx): Json<Transaction>,
) -> Result<StatusCode, (StatusCode, Json<ErrorBody>)> {
    chain
        .lock()
        .unwrap()
        .submit_transaction(tx)
        .map(|()| StatusCode::CREATED)
        .map_err(|error| (StatusCode::BAD_REQUEST, Json(ErrorBody { error })))
}

/// Optimistic mining: take a snapshot of the tip under a short lock,
/// run the PoW on a blocking thread with NO lock held (reads and
/// submissions stay fast meanwhile), then re-take the lock and append.
/// If the tip moved mid-mine — another miner or an adopted longer
/// chain won the race — the block is discarded and mining restarts
/// on top of the new tip.
async fn mine(State(chain): State<SharedChain>, Json(body): Json<MineBody>) -> Json<Block> {
    loop {
        // Scoped block: the guard is dropped at the closing brace, so
        // the lock is definitely released before the mining starts.
        let (index, prev_hash, transactions, difficulty) = {
            let c = chain.lock().unwrap();
            let (index, prev_hash, transactions) = c.mining_job(&body.miner);
            (index, prev_hash, transactions, c.difficulty)
        };

        // PoW is CPU-bound and unbounded in time; running it directly
        // on a runtime worker would starve every other request on that
        // thread. `spawn_blocking` moves it to the blocking pool.
        let block = tokio::task::spawn_blocking(move || {
            Block::mine(index, prev_hash, transactions, difficulty)
        })
        .await
        .expect("mining task panicked");

        // The result is computed inside the scope but acted on outside
        // it, so the lock is not held while we log or return.
        let lost_at = {
            let mut c = chain.lock().unwrap();
            match c.try_append(block.clone()) {
                Ok(()) => None,
                Err(_) => Some(c.last_block().index),
            }
        };
        // Losing the race is normal, not an error: the work is thrown
        // away and we start over on the new tip, exactly as a real
        // miner does when a competitor's block arrives first. The loop
        // is unbounded — acceptable at this scale, but under permanent
        // contention a slow miner could in principle starve.
        match lost_at {
            None => return Json(block),
            Some(tip) => println!("mine: lost the race at height {tip}, remining"),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::wallet::Wallet;
    use http_body_util::BodyExt;
    use tower::ServiceExt;

    fn test_chain() -> SharedChain {
        Arc::new(Mutex::new(Blockchain::new(1)))
    }

    async fn body_json(response: axum::response::Response) -> serde_json::Value {
        let bytes = response.into_body().collect().await.unwrap().to_bytes();
        serde_json::from_slice(&bytes).unwrap()
    }

    fn get(uri: &str) -> axum::http::Request<axum::body::Body> {
        axum::http::Request::builder()
            .uri(uri)
            .body(axum::body::Body::empty())
            .unwrap()
    }

    fn post_json(uri: &str, json: &impl serde::Serialize) -> axum::http::Request<axum::body::Body> {
        axum::http::Request::builder()
            .method("POST")
            .uri(uri)
            .header("content-type", "application/json")
            .body(axum::body::Body::from(serde_json::to_vec(json).unwrap()))
            .unwrap()
    }

    #[tokio::test]
    async fn blocks_endpoint_returns_genesis() {
        let app = router(test_chain());
        let response = app.oneshot(get("/blocks")).await.unwrap();
        assert_eq!(response.status(), StatusCode::OK);
        let blocks = body_json(response).await;
        assert_eq!(blocks.as_array().unwrap().len(), 1);
        assert_eq!(blocks[0]["index"], 0);
    }

    #[tokio::test]
    async fn mining_credits_the_miner() {
        let chain = test_chain();
        let app = router(chain);
        let miner = Wallet::generate();

        let response = app
            .clone()
            .oneshot(post_json(
                "/mine",
                &serde_json::json!({"miner": miner.address()}),
            ))
            .await
            .unwrap();
        assert_eq!(response.status(), StatusCode::OK);

        let response = app
            .oneshot(get(&format!("/balance/{}", miner.address())))
            .await
            .unwrap();
        let body = body_json(response).await;
        assert_eq!(body["balance"], crate::transaction::BLOCK_REWARD);
    }

    #[tokio::test]
    async fn invalid_transaction_is_rejected_with_400() {
        let app = router(test_chain());
        let alice = Wallet::generate();
        // Alice has no funds — the transfer must be rejected.
        let tx = Transaction::new_signed(alice.signing_key(), "bob-addr", 10, 0);

        let response = app.oneshot(post_json("/tx", &tx)).await.unwrap();
        assert_eq!(response.status(), StatusCode::BAD_REQUEST);
        let body = body_json(response).await;
        assert!(body["error"].as_str().unwrap().contains("insufficient"));
    }

    #[tokio::test]
    async fn full_transfer_flow_over_http() {
        let app = router(test_chain());
        let alice = Wallet::generate();
        let bob = Wallet::generate();

        // Alice mines a block to get funds.
        app.clone()
            .oneshot(post_json(
                "/mine",
                &serde_json::json!({"miner": alice.address()}),
            ))
            .await
            .unwrap();

        // She asks the node for her next nonce, signs, and submits.
        let response = app
            .clone()
            .oneshot(get(&format!("/nonce/{}", alice.address())))
            .await
            .unwrap();
        let nonce = body_json(response).await["nonce"].as_u64().unwrap();

        let tx = Transaction::new_signed(alice.signing_key(), &bob.address(), 15, nonce);
        let response = app.clone().oneshot(post_json("/tx", &tx)).await.unwrap();
        assert_eq!(response.status(), StatusCode::CREATED);

        // Someone mines the mempool; Bob's balance appears.
        app.clone()
            .oneshot(post_json(
                "/mine",
                &serde_json::json!({"miner": alice.address()}),
            ))
            .await
            .unwrap();
        let response = app
            .oneshot(get(&format!("/balance/{}", bob.address())))
            .await
            .unwrap();
        assert_eq!(body_json(response).await["balance"], 15);
    }
}

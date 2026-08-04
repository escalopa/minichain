mod api;
mod block;
mod chain;
mod transaction;
mod wallet;

use std::sync::{Arc, Mutex};

use chain::Blockchain;

#[tokio::main]
async fn main() {
    let difficulty: usize = env_or("DIFFICULTY", 4);
    let port: u16 = env_or("PORT", 3000);

    let chain = Arc::new(Mutex::new(Blockchain::new(difficulty)));
    let app = api::router(chain);

    let addr = format!("0.0.0.0:{port}");
    let listener = tokio::net::TcpListener::bind(&addr)
        .await
        .expect("bind listener");
    println!("minichain node listening on {addr} (difficulty {difficulty})");
    axum::serve(listener, app).await.expect("server run");
}

fn env_or<T: std::str::FromStr>(name: &str, default: T) -> T {
    std::env::var(name)
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(default)
}

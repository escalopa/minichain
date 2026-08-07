use std::sync::{Arc, Mutex};

use minichain_node::{api, chain::Blockchain, p2p};

#[tokio::main]
async fn main() {
    let difficulty: usize = env_or("DIFFICULTY", 4);
    let port: u16 = env_or("PORT", 3000);
    let p2p_port: u16 = env_or("P2P_PORT", 0);

    let chain = Arc::new(Mutex::new(Blockchain::new(difficulty)));

    let peers = std::env::var("PEERS")
        .unwrap_or_default()
        .split(',')
        .filter(|s| !s.is_empty())
        .filter_map(|s| s.trim().parse().ok())
        .collect();

    let p2p_chain = chain.clone();
    tokio::spawn(async move {
        if let Err(e) = p2p::run(p2p_chain, p2p_port, peers).await {
            eprintln!("p2p error: {e}");
        }
    });

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

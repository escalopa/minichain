//! Peer-to-peer layer: gossipsub for exchanging chains, mDNS for
//! discovering peers on the local network.
//!
//! The protocol is deliberately naive: whenever our height grows, we
//! publish the whole chain; whoever receives a chain adopts it if it
//! is valid and longer (see `Blockchain::replace_if_longer`). Chains
//! are tiny here, and "longest valid chain wins, resync everything"
//! is the clearest possible statement of Nakamoto consensus.

use std::time::Duration;

use futures::StreamExt;
use libp2p::swarm::{NetworkBehaviour, SwarmEvent};
use libp2p::{gossipsub, mdns, noise, tcp, yamux, Multiaddr};

use crate::api::SharedChain;
use crate::block::Block;

/// Composing two protocols into one behaviour: the derive macro
/// generates a combined `BehaviourEvent` enum whose variants are the
/// events of each field, which is what the `select!` arm below matches.
#[derive(NetworkBehaviour)]
struct Behaviour {
    /// Publish/subscribe message routing — carries the chains.
    gossipsub: gossipsub::Behaviour,
    /// Zero-config discovery of peers on the same LAN via multicast.
    mdns: mdns::tokio::Behaviour,
}

/// `peers` are bootstrap multiaddrs (e.g. `/ip4/127.0.0.1/tcp/4801`)
/// dialed explicitly at startup — mDNS covers local discovery, but
/// multicast is often blocked, and real networks bootstrap anyway.
pub async fn run(
    chain: SharedChain,
    port: u16,
    peers: Vec<Multiaddr>,
) -> Result<(), Box<dyn std::error::Error>> {
    let mut swarm = libp2p::SwarmBuilder::with_new_identity()
        .with_tokio()
        .with_tcp(
            tcp::Config::default(),
            noise::Config::new,
            yamux::Config::default,
        )?
        .with_behaviour(|key| {
            let gossipsub = gossipsub::Behaviour::new(
                gossipsub::MessageAuthenticity::Signed(key.clone()),
                gossipsub::Config::default(),
            )?;
            let mdns =
                mdns::tokio::Behaviour::new(mdns::Config::default(), key.public().to_peer_id())?;
            Ok(Behaviour { gossipsub, mdns })
        })?
        .with_swarm_config(|c| c.with_idle_connection_timeout(Duration::from_secs(3600)))
        .build();

    let topic = gossipsub::IdentTopic::new("minichain/chain");
    swarm.behaviour_mut().gossipsub.subscribe(&topic)?;
    swarm.listen_on(format!("/ip4/0.0.0.0/tcp/{port}").parse::<Multiaddr>()?)?;

    for addr in peers {
        println!("p2p: dialing bootstrap peer {addr}");
        if let Err(e) = swarm.dial(addr) {
            eprintln!("p2p: dial failed: {e}");
        }
    }

    // Republish when our height exceeds what we last managed to
    // announce. Publishing fails while we have no peers — keeping
    // `announced` stale makes us retry once somebody shows up.
    //
    // Announcing on a timer rather than at the moment of mining keeps
    // the p2p task decoupled from the HTTP handlers: they only ever
    // touch the shared chain, and this loop notices the growth.
    let mut ticker = tokio::time::interval(Duration::from_secs(1));
    let mut announced = 0usize;

    loop {
        tokio::select! {
            _ = ticker.tick() => {
                // Serialise under the lock, publish outside it: the
                // gossip send must not block chain operations.
                let (height, payload) = {
                    let c = chain.lock().unwrap();
                    (c.blocks.len(), serde_json::to_vec(&c.blocks).expect("chain serializes"))
                };
                if height > announced
                    && swarm.behaviour_mut().gossipsub.publish(topic.clone(), payload).is_ok()
                {
                    announced = height;
                    println!("p2p: announced chain of {height} blocks");
                }
            }
            event = swarm.select_next_some() => match event {
                SwarmEvent::Behaviour(BehaviourEvent::Mdns(mdns::Event::Discovered(peers))) => {
                    for (peer, _addr) in peers {
                        println!("p2p: discovered peer {peer}");
                        swarm.behaviour_mut().gossipsub.add_explicit_peer(&peer);
                    }
                }
                SwarmEvent::Behaviour(BehaviourEvent::Mdns(mdns::Event::Expired(peers))) => {
                    for (peer, _addr) in peers {
                        swarm.behaviour_mut().gossipsub.remove_explicit_peer(&peer);
                    }
                }
                SwarmEvent::Behaviour(BehaviourEvent::Gossipsub(gossipsub::Event::Message {
                    message, ..
                })) => {
                    // Undecodable gossip is ignored rather than fatal —
                    // anyone can publish to a topic. `replace_if_longer`
                    // then re-verifies every hash, signature and nonce
                    // before anything is adopted: peers are untrusted.
                    if let Ok(blocks) = serde_json::from_slice::<Vec<Block>>(&message.data) {
                        let len = blocks.len();
                        if chain.lock().unwrap().replace_if_longer(blocks) {
                            println!("p2p: adopted longer chain of {len} blocks");
                        }
                    }
                }
                SwarmEvent::ConnectionEstablished { peer_id, .. } => {
                    println!("p2p: connected to {peer_id}");
                    swarm.behaviour_mut().gossipsub.add_explicit_peer(&peer_id);
                }
                SwarmEvent::NewListenAddr { address, .. } => {
                    println!("p2p: listening on {address}");
                }
                _ => {}
            }
        }
    }
}

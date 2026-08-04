mod block;
mod chain;

use chain::Blockchain;

fn main() {
    let difficulty = 4;
    println!("mining with difficulty {difficulty}...\n");

    let mut chain = Blockchain::new(difficulty);
    chain.add_block("tx: alice -> bob 10".into());
    chain.add_block("tx: bob -> carol 5".into());
    chain.add_block("tx: carol -> dave 2".into());

    for block in &chain.blocks {
        println!("{}", serde_json::to_string_pretty(block).unwrap());
    }

    println!("\nchain valid: {}", chain.is_valid());
}

mod block;
mod chain;
mod transaction;
mod wallet;

use chain::Blockchain;
use transaction::Transaction;
use wallet::Wallet;

fn main() {
    let difficulty = 4;
    println!("mining with difficulty {difficulty}...\n");

    let alice = Wallet::generate();
    let bob = Wallet::generate();
    println!("alice: {}", alice.address());
    println!("bob:   {}\n", bob.address());

    let mut chain = Blockchain::new(difficulty);

    // Алисе нужен стартовый капитал — она майнит первый блок.
    chain.mine_pending(&alice.address());

    let tx = Transaction::new_signed(alice.signing_key(), &bob.address(), 15);
    chain.submit_transaction(tx).expect("valid transfer");
    chain.mine_pending(&alice.address());

    for block in &chain.blocks {
        println!("{}", serde_json::to_string_pretty(block).unwrap());
    }

    println!("\nbalances:");
    for (address, balance) in chain.balances() {
        println!("  {}..{} = {balance}", &address[..8], &address[56..]);
    }
    println!("\nchain valid: {}", chain.is_valid());
}

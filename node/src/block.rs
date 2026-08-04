use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};

use crate::transaction::Transaction;

/// A single block in the chain: header + list of transactions.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Block {
    pub index: u64,
    pub timestamp: u64,
    pub prev_hash: String,
    pub nonce: u64,
    pub transactions: Vec<Transaction>,
    pub hash: String,
}

impl Block {
    /// Creates a block and immediately mines it to the given difficulty.
    pub fn mine(
        index: u64,
        prev_hash: String,
        transactions: Vec<Transaction>,
        difficulty: usize,
    ) -> Self {
        let timestamp = now_millis();
        let mut block = Block {
            index,
            timestamp,
            prev_hash,
            nonce: 0,
            transactions,
            hash: String::new(),
        };

        // Proof-of-Work: iterate the nonce until the hash starts
        // with `difficulty` zeros in its hex representation.
        let target = "0".repeat(difficulty);
        loop {
            block.hash = block.compute_hash();
            if block.hash.starts_with(&target) {
                return block;
            }
            block.nonce += 1;
        }
    }

    /// The hash covers every field EXCEPT the hash itself —
    /// otherwise we would have a circular dependency.
    pub fn compute_hash(&self) -> String {
        let txs = serde_json::to_string(&self.transactions).expect("transactions serialize");
        let payload = format!(
            "{}|{}|{}|{}|{}",
            self.index, self.timestamp, self.prev_hash, self.nonce, txs
        );
        let digest = Sha256::digest(payload.as_bytes());
        hex::encode(digest)
    }
}

pub fn now_millis() -> u64 {
    use std::time::{SystemTime, UNIX_EPOCH};
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .expect("system clock before unix epoch")
        .as_millis() as u64
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn mined_block_satisfies_difficulty() {
        let block = Block::mine(1, "genesis".into(), vec![], 2);
        assert!(block.hash.starts_with("00"));
        assert_eq!(block.hash, block.compute_hash());
    }

    #[test]
    fn tampering_with_transactions_changes_hash() {
        let mut block = Block::mine(1, "genesis".into(), vec![], 1);
        block
            .transactions
            .push(Transaction::coinbase("mallory-addr"));
        assert_ne!(block.hash, block.compute_hash());
    }
}

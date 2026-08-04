use std::collections::HashMap;

use crate::block::Block;
use crate::transaction::Transaction;

pub struct Blockchain {
    pub blocks: Vec<Block>,
    pub mempool: Vec<Transaction>,
    pub difficulty: usize,
}

impl Blockchain {
    /// A new chain always starts with a genesis block with no transactions.
    pub fn new(difficulty: usize) -> Self {
        let genesis = Block::mine(0, "0".repeat(64), vec![], difficulty);
        Blockchain {
            blocks: vec![genesis],
            mempool: Vec::new(),
            difficulty,
        }
    }

    pub fn last_block(&self) -> &Block {
        self.blocks.last().expect("chain always has genesis")
    }

    /// Accepts a transaction into the mempool: the signature must be
    /// valid and the sender's balance must cover the transfer, taking
    /// into account what they already promised to spend in the mempool.
    pub fn submit_transaction(&mut self, tx: Transaction) -> Result<(), String> {
        if tx.is_coinbase() {
            return Err("coinbase transactions are created only by mining".into());
        }
        if tx.amount == 0 {
            return Err("amount must be positive".into());
        }
        if !tx.verify() {
            return Err("invalid signature".into());
        }

        let pending_spend: u64 = self
            .mempool
            .iter()
            .filter(|p| p.from == tx.from)
            .map(|p| p.amount)
            .sum();
        let available = self.balance_of(&tx.from).saturating_sub(pending_spend);
        if available < tx.amount {
            return Err(format!(
                "insufficient funds: available {available}, needed {}",
                tx.amount
            ));
        }

        self.mempool.push(tx);
        Ok(())
    }

    /// Mines a block out of the whole mempool + a coinbase reward
    /// for the miner.
    pub fn mine_pending(&mut self, miner: &str) -> &Block {
        let mut transactions = vec![Transaction::coinbase(miner)];
        transactions.append(&mut self.mempool);

        let prev = self.last_block();
        let block = Block::mine(
            prev.index + 1,
            prev.hash.clone(),
            transactions,
            self.difficulty,
        );
        self.blocks.push(block);
        self.last_block()
    }

    /// Account-model balance: incoming minus outgoing across all blocks
    /// of the chain. The mempool is not counted — that is not money yet.
    pub fn balance_of(&self, address: &str) -> u64 {
        let mut balance: i128 = 0;
        for block in &self.blocks {
            for tx in &block.transactions {
                if tx.to == address {
                    balance += tx.amount as i128;
                }
                if tx.from == address {
                    balance -= tx.amount as i128;
                }
            }
        }
        balance.max(0) as u64
    }

    /// Balances of every address ever seen in the chain.
    pub fn balances(&self) -> HashMap<String, u64> {
        let mut addresses: Vec<&str> = Vec::new();
        for block in &self.blocks {
            for tx in &block.transactions {
                addresses.push(&tx.to);
                if !tx.is_coinbase() {
                    addresses.push(&tx.from);
                }
            }
        }
        addresses.sort();
        addresses.dedup();
        addresses
            .into_iter()
            .map(|a| (a.to_string(), self.balance_of(a)))
            .collect()
    }

    /// Full integrity check: hashes are correct, prev_hash links are
    /// unbroken, PoW holds, all signatures are valid, and each block
    /// has at most one coinbase transaction.
    pub fn is_valid(&self) -> bool {
        let target = "0".repeat(self.difficulty);
        for (i, block) in self.blocks.iter().enumerate() {
            if block.hash != block.compute_hash() || !block.hash.starts_with(&target) {
                return false;
            }
            if i > 0 && block.prev_hash != self.blocks[i - 1].hash {
                return false;
            }
            let coinbase_count = block
                .transactions
                .iter()
                .filter(|tx| tx.is_coinbase())
                .count();
            if coinbase_count > 1 {
                return false;
            }
            if !block.transactions.iter().all(Transaction::verify) {
                return false;
            }
        }
        true
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::wallet::Wallet;

    #[test]
    fn fresh_chain_is_valid() {
        let chain = Blockchain::new(1);
        assert!(chain.is_valid());
        assert_eq!(chain.blocks.len(), 1);
    }

    #[test]
    fn mining_pays_block_reward() {
        let mut chain = Blockchain::new(1);
        let miner = Wallet::generate();
        chain.mine_pending(&miner.address());
        assert_eq!(
            chain.balance_of(&miner.address()),
            crate::transaction::BLOCK_REWARD
        );
        assert!(chain.is_valid());
    }

    #[test]
    fn transfer_moves_funds_between_wallets() {
        let mut chain = Blockchain::new(1);
        let alice = Wallet::generate();
        let bob = Wallet::generate();

        chain.mine_pending(&alice.address());
        let tx = Transaction::new_signed(alice.signing_key(), &bob.address(), 20);
        chain.submit_transaction(tx).unwrap();
        chain.mine_pending(&alice.address());

        assert_eq!(chain.balance_of(&bob.address()), 20);
        assert_eq!(
            chain.balance_of(&alice.address()),
            2 * crate::transaction::BLOCK_REWARD - 20
        );
        assert!(chain.is_valid());
    }

    #[test]
    fn overspend_is_rejected() {
        let mut chain = Blockchain::new(1);
        let alice = Wallet::generate();
        let bob = Wallet::generate();

        chain.mine_pending(&alice.address());
        let tx = Transaction::new_signed(alice.signing_key(), &bob.address(), 999);
        assert!(chain.submit_transaction(tx).is_err());
    }

    #[test]
    fn double_spend_within_mempool_is_rejected() {
        let mut chain = Blockchain::new(1);
        let alice = Wallet::generate();
        let bob = Wallet::generate();

        chain.mine_pending(&alice.address());
        let tx1 = Transaction::new_signed(alice.signing_key(), &bob.address(), 40);
        let tx2 = Transaction::new_signed(alice.signing_key(), &bob.address(), 40);
        chain.submit_transaction(tx1).unwrap();
        assert!(chain.submit_transaction(tx2).is_err());
    }

    #[test]
    fn unsigned_transaction_is_rejected() {
        let mut chain = Blockchain::new(1);
        let alice = Wallet::generate();
        let mut tx = Transaction::new_signed(alice.signing_key(), "bob-addr", 10);
        tx.signature = String::new();
        assert!(chain.submit_transaction(tx).is_err());
    }

    #[test]
    fn chain_with_forged_transaction_is_invalid() {
        let mut chain = Blockchain::new(1);
        let alice = Wallet::generate();
        chain.mine_pending(&alice.address());

        // Forge a transaction directly inside a block, recomputing its
        // hash and PoW — the signature cannot be forged, and is_valid
        // catches that.
        let mut forged = Transaction::coinbase("mallory-addr");
        forged.from = alice.address();
        let prev_hash = chain.blocks[0].hash.clone();
        let forged_block = Block::mine(1, prev_hash, vec![forged], chain.difficulty);
        chain.blocks[1] = forged_block;
        assert!(!chain.is_valid());
    }
}

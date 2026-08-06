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

    /// The sender's next expected nonce: the number of their transactions
    /// in the chain plus the ones already pending in the mempool.
    pub fn next_nonce(&self, address: &str) -> u64 {
        let confirmed = self
            .blocks
            .iter()
            .flat_map(|b| &b.transactions)
            .filter(|tx| tx.from == address)
            .count() as u64;
        let pending = self.mempool.iter().filter(|tx| tx.from == address).count() as u64;
        confirmed + pending
    }

    /// Accepts a transaction into the mempool: the signature must be
    /// valid, the nonce must be exactly the next one in sequence
    /// (replay protection), and the sender's balance must cover the
    /// transfer, taking into account what they already promised to
    /// spend in the mempool.
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

        let expected = self.next_nonce(&tx.from);
        if tx.nonce != expected {
            return Err(format!(
                "bad nonce: expected {expected}, got {} (replay or gap)",
                tx.nonce
            ));
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

    /// Full integrity check of this chain.
    pub fn is_valid(&self) -> bool {
        Self::validate(&self.blocks, self.difficulty)
    }

    /// Longest-chain rule: adopt the candidate if it is valid and
    /// strictly longer than ours. Transactions that the new chain
    /// already includes are pruned from the mempool.
    pub fn replace_if_longer(&mut self, candidate: Vec<Block>) -> bool {
        if candidate.len() <= self.blocks.len() {
            return false;
        }
        if !Self::validate(&candidate, self.difficulty) {
            return false;
        }

        let included: std::collections::HashSet<(String, u64)> = candidate
            .iter()
            .flat_map(|b| &b.transactions)
            .filter(|tx| !tx.is_coinbase())
            .map(|tx| (tx.from.clone(), tx.nonce))
            .collect();
        self.mempool
            .retain(|tx| !included.contains(&(tx.from.clone(), tx.nonce)));

        self.blocks = candidate;
        true
    }

    /// Validates an arbitrary chain: hashes are correct, prev_hash
    /// links are unbroken, PoW holds, all signatures are valid, each
    /// block has at most one coinbase transaction, and every sender's
    /// nonces grow strictly sequentially (0, 1, 2, ...) across the
    /// whole chain.
    pub fn validate(blocks: &[Block], difficulty: usize) -> bool {
        let target = "0".repeat(difficulty);
        let mut nonces: HashMap<&str, u64> = HashMap::new();
        for (i, block) in blocks.iter().enumerate() {
            if block.hash != block.compute_hash() || !block.hash.starts_with(&target) {
                return false;
            }
            if i > 0 && block.prev_hash != blocks[i - 1].hash {
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
            for tx in &block.transactions {
                if !tx.verify() {
                    return false;
                }
                if tx.is_coinbase() {
                    continue;
                }
                let expected = nonces.entry(&tx.from).or_insert(0);
                if tx.nonce != *expected {
                    return false;
                }
                *expected += 1;
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
        let tx = Transaction::new_signed(alice.signing_key(), &bob.address(), 20, 0);
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
        let tx = Transaction::new_signed(alice.signing_key(), &bob.address(), 999, 0);
        assert!(chain.submit_transaction(tx).is_err());
    }

    #[test]
    fn double_spend_within_mempool_is_rejected() {
        let mut chain = Blockchain::new(1);
        let alice = Wallet::generate();
        let bob = Wallet::generate();

        chain.mine_pending(&alice.address());
        let tx1 = Transaction::new_signed(alice.signing_key(), &bob.address(), 40, 0);
        let tx2 = Transaction::new_signed(alice.signing_key(), &bob.address(), 40, 1);
        chain.submit_transaction(tx1).unwrap();
        assert!(chain.submit_transaction(tx2).is_err());
    }

    #[test]
    fn unsigned_transaction_is_rejected() {
        let mut chain = Blockchain::new(1);
        let alice = Wallet::generate();
        let mut tx = Transaction::new_signed(alice.signing_key(), "bob-addr", 10, 0);
        tx.signature = String::new();
        assert!(chain.submit_transaction(tx).is_err());
    }

    #[test]
    fn mined_transaction_cannot_be_replayed() {
        let mut chain = Blockchain::new(1);
        let alice = Wallet::generate();
        let bob = Wallet::generate();

        chain.mine_pending(&alice.address());
        let tx = Transaction::new_signed(alice.signing_key(), &bob.address(), 10, 0);
        chain.submit_transaction(tx.clone()).unwrap();
        chain.mine_pending(&alice.address());

        // The same signed transaction a second time: nonce 0 is already spent.
        let err = chain.submit_transaction(tx).unwrap_err();
        assert!(err.contains("bad nonce"));
        assert_eq!(chain.balance_of(&bob.address()), 10);
    }

    #[test]
    fn nonce_gap_is_rejected() {
        let mut chain = Blockchain::new(1);
        let alice = Wallet::generate();
        let bob = Wallet::generate();

        chain.mine_pending(&alice.address());
        let tx = Transaction::new_signed(alice.signing_key(), &bob.address(), 10, 5);
        assert!(chain.submit_transaction(tx).is_err());
    }

    #[test]
    fn longer_valid_chain_is_adopted() {
        let mut ours = Blockchain::new(1);
        let mut theirs = Blockchain::new(1);
        let miner = Wallet::generate();
        theirs.mine_pending(&miner.address());
        theirs.mine_pending(&miner.address());

        assert!(ours.replace_if_longer(theirs.blocks.clone()));
        assert_eq!(ours.blocks.len(), 3);
        assert!(ours.is_valid());
    }

    #[test]
    fn shorter_or_equal_chain_is_rejected() {
        let mut ours = Blockchain::new(1);
        let miner = Wallet::generate();
        ours.mine_pending(&miner.address());

        let equal = ours.blocks.clone();
        assert!(!ours.replace_if_longer(equal));
        assert!(!ours.replace_if_longer(vec![]));
        assert_eq!(ours.blocks.len(), 2);
    }

    #[test]
    fn longer_but_invalid_chain_is_rejected() {
        let mut ours = Blockchain::new(1);
        let mut theirs = Blockchain::new(1);
        let miner = Wallet::generate();
        theirs.mine_pending(&miner.address());
        theirs.mine_pending(&miner.address());
        theirs.blocks[1].transactions[0].amount = 9999; // break a hash

        assert!(!ours.replace_if_longer(theirs.blocks.clone()));
        assert_eq!(ours.blocks.len(), 1);
    }

    #[test]
    fn adoption_prunes_included_mempool_transactions() {
        let mut ours = Blockchain::new(1);
        let alice = Wallet::generate();
        let bob = Wallet::generate();
        ours.mine_pending(&alice.address());
        let tx = Transaction::new_signed(alice.signing_key(), &bob.address(), 10, 0);
        ours.submit_transaction(tx.clone()).unwrap();

        // The network mined a longer chain that already includes the tx.
        let mut theirs = Blockchain {
            blocks: ours.blocks.clone(),
            mempool: vec![tx],
            difficulty: 1,
        };
        theirs.mine_pending(&bob.address());
        theirs.mine_pending(&bob.address());

        assert!(ours.replace_if_longer(theirs.blocks.clone()));
        assert!(ours.mempool.is_empty(), "included tx must leave the mempool");
    }

    #[test]
    fn chain_with_replayed_transaction_is_invalid() {
        let mut chain = Blockchain::new(1);
        let alice = Wallet::generate();
        let bob = Wallet::generate();

        chain.mine_pending(&alice.address());
        let tx = Transaction::new_signed(alice.signing_key(), &bob.address(), 10, 0);
        chain.submit_transaction(tx.clone()).unwrap();
        chain.mine_pending(&alice.address());

        // An attacker manually mines a block with a copy of an already
        // spent transaction: PoW and signature are valid, but the nonce
        // repeats.
        let prev_hash = chain.last_block().hash.clone();
        let forged = Block::mine(2, prev_hash, vec![tx], chain.difficulty);
        chain.blocks.push(forged);
        assert!(!chain.is_valid());
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

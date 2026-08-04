use ed25519_dalek::{Signature, Signer, SigningKey, Verifier, VerifyingKey};
use serde::{Deserialize, Serialize};

use crate::block::now_millis;

/// Sender address of a coinbase transaction — the miner's reward.
/// It has no real sender and no signature.
pub const COINBASE: &str = "COINBASE";

/// Reward for a mined block.
pub const BLOCK_REWARD: u64 = 50;

/// A transfer of funds. An address is the hex representation of an
/// ed25519 public key (32 bytes); the signature covers every field
/// except itself.
///
/// `nonce` is the sender's sequential transaction number (0, 1, 2, ...).
/// It is part of the signed payload, so an already-mined transaction
/// cannot be submitted again: its nonce is spent (replay protection).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    pub from: String,
    pub to: String,
    pub amount: u64,
    pub nonce: u64,
    pub timestamp: u64,
    pub signature: String,
}

impl Transaction {
    /// Creates a transfer and immediately signs it with the sender's key.
    /// Ask the chain for the correct `nonce`: `Blockchain::next_nonce`.
    pub fn new_signed(key: &SigningKey, to: &str, amount: u64, nonce: u64) -> Self {
        let mut tx = Transaction {
            from: hex::encode(key.verifying_key().to_bytes()),
            to: to.to_string(),
            amount,
            nonce,
            timestamp: now_millis(),
            signature: String::new(),
        };
        let sig: Signature = key.sign(tx.payload().as_bytes());
        tx.signature = hex::encode(sig.to_bytes());
        tx
    }

    /// The miner's reward — the only transaction without a signature.
    pub fn coinbase(miner: &str) -> Self {
        Transaction {
            from: COINBASE.to_string(),
            to: miner.to_string(),
            amount: BLOCK_REWARD,
            nonce: 0,
            timestamp: now_millis(),
            signature: String::new(),
        }
    }

    pub fn is_coinbase(&self) -> bool {
        self.from == COINBASE
    }

    /// What gets signed: every field except the signature itself.
    fn payload(&self) -> String {
        format!(
            "{}|{}|{}|{}|{}",
            self.from, self.to, self.amount, self.nonce, self.timestamp
        )
    }

    /// Verifies the signature against the public key taken from `from`.
    /// Coinbase transactions are considered valid without a signature.
    pub fn verify(&self) -> bool {
        if self.is_coinbase() {
            return self.amount == BLOCK_REWARD;
        }
        let Ok(key_bytes) = hex::decode(&self.from) else {
            return false;
        };
        let Ok(key_bytes) = <[u8; 32]>::try_from(key_bytes.as_slice()) else {
            return false;
        };
        let Ok(key) = VerifyingKey::from_bytes(&key_bytes) else {
            return false;
        };
        let Ok(sig_bytes) = hex::decode(&self.signature) else {
            return false;
        };
        let Ok(sig) = Signature::from_slice(&sig_bytes) else {
            return false;
        };
        key.verify(self.payload().as_bytes(), &sig).is_ok()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::wallet::Wallet;

    #[test]
    fn signed_transaction_verifies() {
        let alice = Wallet::generate();
        let tx = Transaction::new_signed(alice.signing_key(), "bob-addr", 10, 0);
        assert!(tx.verify());
    }

    #[test]
    fn tampered_amount_fails_verification() {
        let alice = Wallet::generate();
        let mut tx = Transaction::new_signed(alice.signing_key(), "bob-addr", 10, 0);
        tx.amount = 1000;
        assert!(!tx.verify());
    }

    #[test]
    fn tampered_nonce_fails_verification() {
        let alice = Wallet::generate();
        let mut tx = Transaction::new_signed(alice.signing_key(), "bob-addr", 10, 0);
        tx.nonce = 7;
        assert!(!tx.verify());
    }

    #[test]
    fn forged_sender_fails_verification() {
        let alice = Wallet::generate();
        let mallory = Wallet::generate();
        let mut tx = Transaction::new_signed(alice.signing_key(), "bob-addr", 10, 0);
        tx.from = mallory.address();
        assert!(!tx.verify());
    }

    #[test]
    fn coinbase_is_valid_without_signature() {
        let tx = Transaction::coinbase("miner-addr");
        assert!(tx.verify());
        assert!(tx.is_coinbase());
    }
}

use ed25519_dalek::{Signature, Signer, SigningKey, Verifier, VerifyingKey};
use serde::{Deserialize, Serialize};

use crate::block::now_millis;

/// Адрес отправителя coinbase-транзакции — награды майнеру.
/// У неё нет реального отправителя и подписи.
pub const COINBASE: &str = "COINBASE";

/// Награда за смайненный блок.
pub const BLOCK_REWARD: u64 = 50;

/// Перевод средств. Адрес — это hex-представление публичного ключа
/// ed25519 (32 байта), подпись покрывает все поля, кроме неё самой.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    pub from: String,
    pub to: String,
    pub amount: u64,
    pub timestamp: u64,
    pub signature: String,
}

impl Transaction {
    /// Создаёт перевод и сразу подписывает его ключом отправителя.
    pub fn new_signed(key: &SigningKey, to: &str, amount: u64) -> Self {
        let mut tx = Transaction {
            from: hex::encode(key.verifying_key().to_bytes()),
            to: to.to_string(),
            amount,
            timestamp: now_millis(),
            signature: String::new(),
        };
        let sig: Signature = key.sign(tx.payload().as_bytes());
        tx.signature = hex::encode(sig.to_bytes());
        tx
    }

    /// Награда майнеру — единственная транзакция без подписи.
    pub fn coinbase(miner: &str) -> Self {
        Transaction {
            from: COINBASE.to_string(),
            to: miner.to_string(),
            amount: BLOCK_REWARD,
            timestamp: now_millis(),
            signature: String::new(),
        }
    }

    pub fn is_coinbase(&self) -> bool {
        self.from == COINBASE
    }

    /// То, что подписывается: все поля, кроме самой подписи.
    fn payload(&self) -> String {
        format!("{}|{}|{}|{}", self.from, self.to, self.amount, self.timestamp)
    }

    /// Проверяет подпись против публичного ключа из `from`.
    /// Coinbase-транзакции считаются валидными без подписи.
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
        let tx = Transaction::new_signed(alice.signing_key(), "bob-addr", 10);
        assert!(tx.verify());
    }

    #[test]
    fn tampered_amount_fails_verification() {
        let alice = Wallet::generate();
        let mut tx = Transaction::new_signed(alice.signing_key(), "bob-addr", 10);
        tx.amount = 1000;
        assert!(!tx.verify());
    }

    #[test]
    fn forged_sender_fails_verification() {
        let alice = Wallet::generate();
        let mallory = Wallet::generate();
        let mut tx = Transaction::new_signed(alice.signing_key(), "bob-addr", 10);
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

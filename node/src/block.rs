use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};

/// Один блок цепочки. В фазе 1 `data` — просто строка;
/// в фазе 3 здесь появится Vec<Transaction>.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Block {
    pub index: u64,
    pub timestamp: u64,
    pub prev_hash: String,
    pub nonce: u64,
    pub data: String,
    pub hash: String,
}

impl Block {
    /// Создаёт блок и сразу майнит его под нужную сложность.
    pub fn mine(index: u64, prev_hash: String, data: String, difficulty: usize) -> Self {
        let timestamp = now_millis();
        let mut block = Block {
            index,
            timestamp,
            prev_hash,
            nonce: 0,
            data,
            hash: String::new(),
        };

        // Proof-of-Work: перебираем nonce, пока хэш не начнётся
        // с `difficulty` нулей в hex-представлении.
        let target = "0".repeat(difficulty);
        loop {
            block.hash = block.compute_hash();
            if block.hash.starts_with(&target) {
                return block;
            }
            block.nonce += 1;
        }
    }

    /// Хэш считается по всем полям, КРОМЕ самого hash —
    /// иначе получилась бы циклическая зависимость.
    pub fn compute_hash(&self) -> String {
        let payload = format!(
            "{}|{}|{}|{}|{}",
            self.index, self.timestamp, self.prev_hash, self.nonce, self.data
        );
        let digest = Sha256::digest(payload.as_bytes());
        hex::encode(digest)
    }
}

fn now_millis() -> u64 {
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
        let block = Block::mine(1, "genesis".into(), "hello".into(), 2);
        assert!(block.hash.starts_with("00"));
        assert_eq!(block.hash, block.compute_hash());
    }

    #[test]
    fn tampering_with_data_changes_hash() {
        let mut block = Block::mine(1, "genesis".into(), "hello".into(), 1);
        block.data = "evil".into();
        assert_ne!(block.hash, block.compute_hash());
    }
}

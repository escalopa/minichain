use crate::block::Block;

pub struct Blockchain {
    pub blocks: Vec<Block>,
    pub difficulty: usize,
}

impl Blockchain {
    /// Новая цепочка всегда начинается с генезис-блока.
    pub fn new(difficulty: usize) -> Self {
        let genesis = Block::mine(0, "0".repeat(64), "genesis".into(), difficulty);
        Blockchain {
            blocks: vec![genesis],
            difficulty,
        }
    }

    pub fn last_block(&self) -> &Block {
        self.blocks.last().expect("chain always has genesis")
    }

    pub fn add_block(&mut self, data: String) -> &Block {
        let prev = self.last_block();
        let block = Block::mine(prev.index + 1, prev.hash.clone(), data, self.difficulty);
        self.blocks.push(block);
        self.last_block()
    }

    /// Полная проверка целостности: хэши корректны, ссылки prev_hash
    /// не разорваны, PoW соблюдён.
    pub fn is_valid(&self) -> bool {
        let target = "0".repeat(self.difficulty);
        for (i, block) in self.blocks.iter().enumerate() {
            if block.hash != block.compute_hash() || !block.hash.starts_with(&target) {
                return false;
            }
            if i > 0 && block.prev_hash != self.blocks[i - 1].hash {
                return false;
            }
        }
        true
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fresh_chain_is_valid() {
        let chain = Blockchain::new(1);
        assert!(chain.is_valid());
        assert_eq!(chain.blocks.len(), 1);
    }

    #[test]
    fn chain_grows_and_stays_valid() {
        let mut chain = Blockchain::new(2);
        chain.add_block("tx: alice -> bob 10".into());
        chain.add_block("tx: bob -> carol 5".into());
        assert_eq!(chain.blocks.len(), 3);
        assert!(chain.is_valid());
    }

    #[test]
    fn tampered_chain_is_detected() {
        let mut chain = Blockchain::new(1);
        chain.add_block("tx: alice -> bob 10".into());
        chain.blocks[1].data = "tx: alice -> mallory 1000".into();
        assert!(!chain.is_valid());
    }
}

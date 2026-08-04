use ed25519_dalek::SigningKey;
use rand::rngs::OsRng;

/// Пара ключей ed25519. Адрес кошелька = hex публичного ключа.
pub struct Wallet {
    signing_key: SigningKey,
}

impl Wallet {
    pub fn generate() -> Self {
        Wallet {
            signing_key: SigningKey::generate(&mut OsRng),
        }
    }

    pub fn address(&self) -> String {
        hex::encode(self.signing_key.verifying_key().to_bytes())
    }

    pub fn signing_key(&self) -> &SigningKey {
        &self.signing_key
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn address_is_64_hex_chars() {
        let wallet = Wallet::generate();
        assert_eq!(wallet.address().len(), 64);
        assert!(wallet.address().chars().all(|c| c.is_ascii_hexdigit()));
    }
}

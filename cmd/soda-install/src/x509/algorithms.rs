// ---------------------------------------------------------------------------
// Algorithm identifiers.
// ---------------------------------------------------------------------------

/// Go `x509.SignatureAlgorithm`, printing Go's table names.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SignatureAlgorithm {
    Unknown,
    Md5Rsa,
    Sha1Rsa,
    Sha256Rsa,
    Sha384Rsa,
    Sha512Rsa,
    Sha256Pss,
    Sha384Pss,
    Sha512Pss,
    DsaSha1,
    DsaSha256,
    EcdsaSha1,
    EcdsaSha256,
    EcdsaSha384,
    EcdsaSha512,
    Ed25519,
}

impl SignatureAlgorithm {
    pub(super) fn name(self) -> &'static str {
        match self {
            SignatureAlgorithm::Unknown => "0",
            SignatureAlgorithm::Md5Rsa => "MD5-RSA",
            SignatureAlgorithm::Sha1Rsa => "SHA1-RSA",
            SignatureAlgorithm::Sha256Rsa => "SHA256-RSA",
            SignatureAlgorithm::Sha384Rsa => "SHA384-RSA",
            SignatureAlgorithm::Sha512Rsa => "SHA512-RSA",
            SignatureAlgorithm::Sha256Pss => "SHA256-RSAPSS",
            SignatureAlgorithm::Sha384Pss => "SHA384-RSAPSS",
            SignatureAlgorithm::Sha512Pss => "SHA512-RSAPSS",
            SignatureAlgorithm::DsaSha1 => "DSA-SHA1",
            SignatureAlgorithm::DsaSha256 => "DSA-SHA256",
            SignatureAlgorithm::EcdsaSha1 => "ECDSA-SHA1",
            SignatureAlgorithm::EcdsaSha256 => "ECDSA-SHA256",
            SignatureAlgorithm::EcdsaSha384 => "ECDSA-SHA384",
            SignatureAlgorithm::EcdsaSha512 => "ECDSA-SHA512",
            SignatureAlgorithm::Ed25519 => "Ed25519",
        }
    }
}

/// Go `x509.PublicKeyAlgorithm`, printing Go's table names.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PublicKeyAlgorithm {
    Unknown,
    Rsa,
    Dsa,
    Ecdsa,
    Ed25519,
}

impl PublicKeyAlgorithm {
    pub(super) fn name(self) -> &'static str {
        match self {
            PublicKeyAlgorithm::Unknown => "0",
            PublicKeyAlgorithm::Rsa => "RSA",
            PublicKeyAlgorithm::Dsa => "DSA",
            PublicKeyAlgorithm::Ecdsa => "ECDSA",
            PublicKeyAlgorithm::Ed25519 => "Ed25519",
        }
    }
}

/// Go `x509.KeyUsage` bits (only bits 0-8 exist).
pub const KEY_USAGE_CERT_SIGN: u16 = 1 << 5;
#[cfg(test)]
pub const KEY_USAGE_DIGITAL_SIGNATURE: u16 = 1 << 0;
#[cfg(test)]
pub const KEY_USAGE_KEY_ENCIPHERMENT: u16 = 1 << 2;
#[cfg(test)]
pub const KEY_USAGE_DECIPHER_ONLY: u16 = 1 << 8;

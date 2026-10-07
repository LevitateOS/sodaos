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
            Self::Unknown => "0",
            Self::Md5Rsa => "MD5-RSA",
            Self::Sha1Rsa => "SHA1-RSA",
            Self::Sha256Rsa => "SHA256-RSA",
            Self::Sha384Rsa => "SHA384-RSA",
            Self::Sha512Rsa => "SHA512-RSA",
            Self::Sha256Pss => "SHA256-RSAPSS",
            Self::Sha384Pss => "SHA384-RSAPSS",
            Self::Sha512Pss => "SHA512-RSAPSS",
            Self::DsaSha1 => "DSA-SHA1",
            Self::DsaSha256 => "DSA-SHA256",
            Self::EcdsaSha1 => "ECDSA-SHA1",
            Self::EcdsaSha256 => "ECDSA-SHA256",
            Self::EcdsaSha384 => "ECDSA-SHA384",
            Self::EcdsaSha512 => "ECDSA-SHA512",
            Self::Ed25519 => "Ed25519",
        }
    }
}

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
            Self::Unknown => "0",
            Self::Rsa => "RSA",
            Self::Dsa => "DSA",
            Self::Ecdsa => "ECDSA",
            Self::Ed25519 => "Ed25519",
        }
    }
}

pub(super) fn signature_algorithm(
    algorithm: &x509_cert::spki::AlgorithmIdentifierOwned,
) -> SignatureAlgorithm {
    use x509_cert::der::{Decode, Encode};
    let parameters_absent = algorithm.parameters.is_none();
    let parameters_absent_or_null = algorithm
        .parameters
        .as_ref()
        .is_none_or(|parameters| x509_cert::der::asn1::AnyRef::from(parameters).is_null());
    let oid = algorithm.oid.to_string();
    if oid == "1.2.840.113549.1.1.10" {
        let Some(parameters) = &algorithm.parameters else {
            return SignatureAlgorithm::Unknown;
        };
        let Ok(encoded) = parameters.to_der() else {
            return SignatureAlgorithm::Unknown;
        };
        let Ok(pss) = rsa::pkcs1::RsaPssParams::from_der(&encoded) else {
            return SignatureAlgorithm::Unknown;
        };
        let hash_params_ok = pss
            .hash
            .parameters
            .is_none_or(|p| p == x509_cert::der::asn1::AnyRef::NULL);
        let mgf_params_ok = pss.mask_gen.parameters.is_some_and(|algorithm| {
            algorithm
                .parameters
                .is_none_or(|p| p == x509_cert::der::asn1::AnyRef::NULL)
        });
        if pss.mask_gen.oid.to_string() != "1.2.840.113549.1.1.8"
            || pss.mask_gen.parameters.map(|algorithm| algorithm.oid) != Some(pss.hash.oid)
            || !hash_params_ok
            || !mgf_params_ok
            || pss.trailer_field != rsa::pkcs1::TrailerField::BC
        {
            return SignatureAlgorithm::Unknown;
        }
        return if pss.hash.oid.to_string() == "2.16.840.1.101.3.4.2.1" && pss.salt_len == 32 {
            SignatureAlgorithm::Sha256Pss
        } else if pss.hash.oid.to_string() == "2.16.840.1.101.3.4.2.2" && pss.salt_len == 48 {
            SignatureAlgorithm::Sha384Pss
        } else if pss.hash.oid.to_string() == "2.16.840.1.101.3.4.2.3" && pss.salt_len == 64 {
            SignatureAlgorithm::Sha512Pss
        } else {
            SignatureAlgorithm::Unknown
        };
    }
    match oid.as_str() {
        "1.2.840.113549.1.1.4" if parameters_absent_or_null => SignatureAlgorithm::Md5Rsa,
        "1.2.840.113549.1.1.5" | "1.3.14.3.2.29" if parameters_absent_or_null => {
            SignatureAlgorithm::Sha1Rsa
        }
        "1.2.840.113549.1.1.11" if parameters_absent_or_null => SignatureAlgorithm::Sha256Rsa,
        "1.2.840.113549.1.1.12" if parameters_absent_or_null => SignatureAlgorithm::Sha384Rsa,
        "1.2.840.113549.1.1.13" if parameters_absent_or_null => SignatureAlgorithm::Sha512Rsa,
        "1.2.840.10040.4.3" if parameters_absent => SignatureAlgorithm::DsaSha1,
        "2.16.840.1.101.3.4.3.2" if parameters_absent => SignatureAlgorithm::DsaSha256,
        "1.2.840.10045.4.1" if parameters_absent => SignatureAlgorithm::EcdsaSha1,
        "1.2.840.10045.4.3.2" if parameters_absent => SignatureAlgorithm::EcdsaSha256,
        "1.2.840.10045.4.3.3" if parameters_absent => SignatureAlgorithm::EcdsaSha384,
        "1.2.840.10045.4.3.4" if parameters_absent => SignatureAlgorithm::EcdsaSha512,
        "1.3.101.112" if parameters_absent => SignatureAlgorithm::Ed25519,
        _ => SignatureAlgorithm::Unknown,
    }
}

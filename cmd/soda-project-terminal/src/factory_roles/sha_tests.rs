use super::*;

fn hex_of(data: &[u8]) -> String {
    hex(&Sha256::digest(data))
}

#[test]
fn nist_vectors() {
    assert_eq!(
        hex_of(b""),
        "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
    );
    assert_eq!(
        hex_of(b"abc"),
        "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
    );
    // Order-independent input: sorted inside, NUL-separated.
    let digest = approved_digest(&[
        ("check.sh".to_string(), b"true\n".to_vec()),
        ("setup.sh".to_string(), b"true\n".to_vec()),
    ]);
    let flipped = approved_digest(&[
        ("setup.sh".to_string(), b"true\n".to_vec()),
        ("check.sh".to_string(), b"true\n".to_vec()),
    ]);
    assert_eq!(digest, flipped);
    assert_eq!(digest.len(), 64);
    // Baked against hashlib.sha256 over the same framing.
    assert_eq!(
        digest,
        "61f93011cfadec13edfaf04218ba4f15df40b2151bd3771c2e4a73df641246d5"
    );
}

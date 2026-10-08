use super::*;
#[test]
fn bounded_reader_rejects_suffix_and_hashes_original_bytes() {
    let dir = std::env::temp_dir().join(format!("soda-jsonio-{}", std::process::id()));
    std::fs::create_dir_all(&dir).unwrap();
    let path = dir.join("input.json");
    std::fs::write(&path, b"{} trailing").unwrap();
    assert!(read_json_file(path.to_str().unwrap()).is_err());
    let bytes = br#"{"Revision":"x"}"#;
    std::fs::write(&path, bytes).unwrap();
    let (value, digest) = read_json_file(path.to_str().unwrap()).unwrap();
    assert_eq!(value.get(), std::str::from_utf8(bytes).unwrap());
    assert_eq!(
        digest,
        crate::sha256::hex_lower(&crate::sha256::digest(bytes))
    );
    std::fs::remove_dir_all(&dir).unwrap();
}

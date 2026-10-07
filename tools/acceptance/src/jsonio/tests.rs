use super::*;
use serde_json::value::RawValue;

#[test]
fn serde_formatter_matches_go_compact_and_indent() {
    let value = serde_json::json!({
        "b": [1, true, null, "x"],
        "a": {"n": -1250, "s": "a<b>&\"c\"\n\u{1}\u{2028}"},
    });
    let mut compact = String::new();
    write_compact(&mut compact, &value);
    assert_eq!(
        compact,
        r#"{"a":{"n":-1250,"s":"a\u003cb\u003e\u0026\"c\"\n\u0001\u2028"},"b":[1,true,null,"x"]}"#
    );
    let mut indent = String::new();
    write_indent(&mut indent, &serde_json::json!({"a":null,"b":[1,{}]}));
    assert_eq!(
        indent,
        "{\n  \"a\": null,\n  \"b\": [\n    1,\n    {}\n  ]\n}"
    );

    let raw = RawValue::from_string("1e2".to_string()).unwrap();
    let mut raw_number = String::new();
    write_compact(&mut raw_number, &raw);
    assert_eq!(raw_number, "1e2");
}

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

use super::*;

fn parse(text: &str) -> JsonValue {
    JsonValue::parse(text).unwrap()
}

#[test]
fn emitter_matches_go_byte_for_byte() {
    // Fixtures generated with Go's encoding/json during implementation;
    // see the oracle note in the commit message.
    let cases = [
        (
            "{\"b\":[1,true,null,\"x\"],\"a\":{\"n\":-12.5e2,\"s\":\"a<b>&\\\"c\\\"\\n\\u0001\\u2028\"}}",
            "{\"a\":{\"n\":-12.5e2,\"s\":\"a\\u003cb\\u003e\\u0026\\\"c\\\"\\n\\u0001\\u2028\"},\"b\":[1,true,null,\"x\"]}",
        ),
        ("{}", "{}"),
        ("[]", "[]"),
        ("{\"e\":{},\"l\":[]}", "{\"e\":{},\"l\":[]}"),
        ("{\"big\":9223372036854775807}", "{\"big\":9223372036854775807}"),
    ];
    for (input, want) in cases {
        let value = parse(input);
        // Go sorts map keys; emulate by sorting top-level object entries.
        let mut sorted = String::new();
        if let JsonValue::Object(entries) = &value {
            let mut ordered = entries.clone();
            ordered.sort_by(|a, b| a.0.cmp(&b.0));
            write_compact(&mut sorted, &JsonValue::Object(ordered));
        } else {
            write_compact(&mut sorted, &value);
        }
        assert_eq!(sorted, want, "compact {input}");
    }
    let mut indent = String::new();
    write_indent(&mut indent, &parse("{\"a\":null,\"b\":[1,{}]}"));
    assert_eq!(
        indent,
        "{\n  \"a\": null,\n  \"b\": [\n    1,\n    {}\n  ]\n}"
    );
}

#[test]
fn strict_read_rejects_unknown_fields_and_drift() {
    let dir = std::env::temp_dir().join(format!("soda-jsonio-{}", std::process::id()));
    std::fs::create_dir_all(&dir).unwrap();
    let path = dir.join("input.json");
    std::fs::write(&path, b"{\"Revision\":\"x\"} trailing").unwrap();
    assert!(read_json_file(path.to_str().unwrap()).is_err());
    std::fs::write(&path, b"{\"Revision\":\"x\",\"Extra\":1}").unwrap();
    let (value, digest) = read_json_file(path.to_str().unwrap()).unwrap();
    assert_eq!(digest.len(), 64);
    assert_eq!(require_string(&value, "Revision").unwrap(), "x");
    assert!(check_no_unknown(&value, &["Revision"]).is_err());
    assert!(check_no_unknown(&value, &["Revision", "Extra"]).is_ok());
    assert!(require_integer(&value, "Extra").is_ok());
    assert!(require_bool(&value, "Extra").is_err());
    std::fs::remove_dir_all(&dir).unwrap();
}

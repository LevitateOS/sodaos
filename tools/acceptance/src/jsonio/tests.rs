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

#[test]
fn civil_dates_cover_edges() {
    assert_eq!(format_unix_nano(0, 0), "1970-01-01T00:00:00Z");
    assert_eq!(format_unix_nano(0, 120_000_000), "1970-01-01T00:00:00.12Z");
    assert_eq!(format_unix_nano(1_704_067_200, 0), "2024-01-01T00:00:00Z");
    assert_eq!(
        format_unix_nano(1_893_456_000, 1),
        "2030-01-01T00:00:00.000000001Z"
    );
    assert_eq!(format_unix_nano(-1, 0), "1969-12-31T23:59:59Z");
    assert!(validate_rfc3339(&now_rfc3339_nano()).is_ok());
}

#[test]
fn timestamps_validate_strictly() {
    for good in [
        "2026-10-04T18:05:00Z",
        "2026-10-04T18:05:00.123456789Z",
        "2024-02-29T00:00:00+00:00",
        "0001-01-01T00:00:00Z",
    ] {
        assert!(validate_rfc3339(good).is_ok(), "{good}");
    }
    for bad in [
        "2026-10-04 18:05:00Z",
        "2026-13-01T00:00:00Z",
        "2023-02-29T00:00:00Z",
        "2026-10-04T24:00:00Z",
        "2026-10-04T00:00:00",
        "2026-10-04T00:00:00.",
        "2026-10-04T00:00:00.1234567890Z",
        "2026-10-04T00:00:61Z",
        "not-a-time",
    ] {
        assert!(validate_rfc3339(bad).is_err(), "{bad}");
    }
}

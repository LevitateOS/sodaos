use super::*;
use serde::Deserialize;

#[derive(Debug, PartialEq, Deserialize)]
struct Envelope {
    #[serde(default)]
    command_id: String,
    #[serde(
        default,
        rename = "type",
        deserialize_with = "crate::wire_scalars::null_tolerant::string"
    )]
    wire_type: String,
    #[serde(default)]
    target: String,
}

const FIELDS: &[&str] = &["command_id", "type", "target"];

fn decode_envelope(input: &str) -> Result<Envelope, String> {
    decode(input.as_bytes(), MAX_DOCUMENT, FIELDS, &[])
}

#[test]
fn strict_vectors_match_go() {
    // Mirrors scripts/fixtures/portcontracts/strictjson_vectors.json.
    for (input, ok) in [
        (r#"{"type":"status"}"#, true),
        (r#"{}"#, true),
        (r#"  {"type":"status","target":"r"}  "#, true),
        ("{\"type\":\"a\\\"b \\u00e9\"}", true),
        (r#"{"type":"status","project_id":"site"}"#, false),
        (r#"{"type":"status","type":"stop"}"#, false),
        (r#"{"type":{"a":1,"a":2}}"#, false),
        (r#"{"type":{"outer":{"key":"1","key":"2"}}}"#, false),
        (
            r#"{"type":"x","target":[{"k":"1"},{"k":"1","k":"2"}]}"#,
            false,
        ),
        (
            r#"{"type":"x","future":{"items":[{"\u006b":1,"k":2}]}}"#,
            false,
        ),
        (r#"[]"#, false),
        (r#"{"type":"one"}{"type":"two"}"#, false),
        (r#"{"type":"one""#, false),
    ] {
        let result = decode_envelope(input);
        assert_eq!(result.is_ok(), ok, "input {input}");
    }
    assert!(decode_envelope(r#"{"type":"status","project_id":"site"}"#)
        .unwrap_err()
        .contains("unknown field"));
    assert!(decode_envelope(r#"{"type":"status","type":"stop"}"#)
        .unwrap_err()
        .contains("duplicate request field"));
}

#[test]
fn limits_match_go() {
    let big = format!("{{\"type\":\"{}\"}}", "a".repeat(1 << 20));
    assert!(decode_envelope(&big).unwrap_err().contains("size limit"));
    assert!(
        decode::<Envelope>(&[b'{', b'"', 0xff], MAX_DOCUMENT, FIELDS, &[])
            .unwrap_err()
            .contains("valid UTF-8")
    );
    for (depth, ok) in [(99, true), (100, true), (101, false)] {
        let mut nested = String::from("{\"type\":");
        nested.push_str(&"[".repeat(depth));
        nested.push('0');
        nested.push_str(&"]".repeat(depth));
        nested.push('}');
        let result = decode::<serde_json::Value>(nested.as_bytes(), MAX_DOCUMENT, &["type"], &[]);
        assert_eq!(result.is_ok(), ok, "nesting depth {depth}");
        if !ok {
            assert!(result.unwrap_err().contains("nested too deeply"));
        }
    }
}

#[test]
fn case_fold_matches_go_fallback() {
    let envelope = decode_envelope(r#"{"TYPE":"status"}"#).unwrap();
    assert_eq!(envelope.wire_type, "status");
    // Exact wins; a colliding fold is rejected as unknown.
    assert!(decode_envelope(r#"{"TYPE":"a","type":"b"}"#).is_err());

    // Preserve the existing sorted-map last-wins remapping behavior.
    let envelope = decode_envelope(r#"{"TYPE":"upper","Type":"mixed"}"#).unwrap();
    assert_eq!(envelope.wire_type, "mixed");
    assert_eq!(decode_envelope(r#"{"type":null}"#).unwrap().wire_type, "");
}

#[test]
fn unicode_keys_compare_decoded() {
    assert!(decode_envelope("{\"type\":\"a\",\"\\u0074ype\":\"b\"}").is_err());
}

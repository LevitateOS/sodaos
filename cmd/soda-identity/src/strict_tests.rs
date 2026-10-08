use super::*;
use serde::Deserialize;

#[derive(Debug, PartialEq, Deserialize)]
#[serde(deny_unknown_fields)]
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

fn decode_envelope(input: &str) -> Result<Envelope, String> {
    decode_typed(input.as_bytes(), MAX_DOCUMENT)
}

#[test]
fn typed_http_request_preserves_wire_codecs_and_rejects_unknowns() {
    let request: crate::wire::Request = decode_typed(
        r#"{"owner_id":"42","id":"quoted \"id\" é","credential":"Zm8=","binding":{"child_id":null,"uid":17,"generation":4}}"#.as_bytes(),
        MAX_DOCUMENT,
    )
    .unwrap();
    assert_eq!(request.owner_id, 42);
    assert_eq!(request.id, "quoted \"id\" é");
    assert_eq!(request.credential.as_deref(), Some(b"fo".as_slice()));
    let binding = request.binding.unwrap();
    assert_eq!(binding.child_id, "");
    assert_eq!(binding.uid, 17);
    assert_eq!(binding.generation, 4);

    let nested: crate::wire::Request = decode_typed(
        br#"{"grant":{"connection_id":"conn","user_id":"8","project_id":"project","confirm_subscription":true,"confirm_credential_exposure":true},"acquire":{"repository_id":"7","provider_id":"codex","execution_id":"exec","actor_id":"8","connection_id":"conn","project_id":"project","kind":"factory","deadline":"2030-01-02T03:04:05Z"}}"#,
        MAX_DOCUMENT,
    )
    .unwrap();
    assert_eq!(nested.grant.unwrap().user_id, 8);
    assert_eq!(nested.acquire.unwrap().repository_id, 7);

    for input in [
        r#"{"Owner_id":"42"}"#,
        r#"{"binding":{"child_id":"x","future":true}}"#,
        r#"{"grant":{"connection_id":"x","future":true}}"#,
        r#"{"acquire":{"provider_id":"codex","future":true}}"#,
        r#"{"id":"a","id":"b"}"#,
        r#"{"binding":{"uid":1,"uid":2}}"#,
        r#"{"id":"ok"} {"id":"second"}"#,
    ] {
        assert!(
            decode_typed::<crate::wire::Request>(input.as_bytes(), MAX_DOCUMENT).is_err(),
            "accepted {input}"
        );
    }
    assert!(decode_typed::<crate::wire::Request>(&[b'{', b'"', 0xff], MAX_DOCUMENT).is_err());
    let too_deep = format!("{{\"id\":{}0{}}}", "[".repeat(101), "]".repeat(101));
    assert!(decode_typed::<crate::wire::Request>(too_deep.as_bytes(), MAX_DOCUMENT).is_err());
}

#[test]
fn typed_request_rejects_non_object_roots() {
    for input in ["[]", "null", "42"] {
        assert!(
            decode_typed::<crate::wire::Request>(input.as_bytes(), MAX_DOCUMENT).is_err(),
            "accepted non-object root {input}"
        );
    }
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
    assert!(decode_typed::<Envelope>(&[b'{', b'"', 0xff], MAX_DOCUMENT)
        .unwrap_err()
        .contains("valid UTF-8"));
    for (depth, ok) in [(99, true), (100, true), (101, false)] {
        let mut nested = String::from("{\"type\":");
        nested.push_str(&"[".repeat(depth));
        nested.push('0');
        nested.push_str(&"]".repeat(depth));
        nested.push('}');
        let result = decode_typed::<serde_json::Value>(nested.as_bytes(), MAX_DOCUMENT);
        assert_eq!(result.is_ok(), ok, "nesting depth {depth}");
        if !ok {
            assert!(result.unwrap_err().contains("nested too deeply"));
        }
    }
}

#[test]
fn typed_field_names_are_exact_and_null_codec_is_preserved() {
    assert!(decode_envelope(r#"{"TYPE":"status"}"#).is_err());
    assert!(decode_envelope(r#"{"TYPE":"a","type":"b"}"#).is_err());
    assert_eq!(decode_envelope(r#"{"type":null}"#).unwrap().wire_type, "");
}

#[test]
fn unicode_keys_compare_decoded() {
    assert!(decode_envelope("{\"type\":\"a\",\"\\u0074ype\":\"b\"}").is_err());
}

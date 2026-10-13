//! Current release-delivery behavior tests using captured original fixtures.
//!
//! Goldens were dumped from `internal/release/deliver` (since removed
//! temporary oracle test) and are embedded here. Fixture records are compared
//! semantically; original signed-input bytes and their signature/hash checks
//! remain byte-exact. Newly written OCI document layers are checked for
//! deterministic output and content round-trip, not the retired Go tar hash.

use serde_json::Value as JsonValue;
use soda_release_deliver::model::{Candidate, Trust};
use soda_release_deliver::native::merge_policy;
use soda_release_deliver::payload::Payload;

mod artifacts;

const GOLDENS: &str = include_str!("../goldens/deliver.json");

fn goldens() -> JsonValue {
    serde_json::from_str(GOLDENS).expect("goldens parse")
}

fn golden_str(g: &JsonValue, name: &str) -> String {
    g.get(name)
        .and_then(|v| v.as_str())
        .unwrap_or("")
        .to_string()
}

fn golden_ok(g: &JsonValue, name: &str) -> bool {
    let results = g.get("results").expect("results");
    let entry = results
        .get(name)
        .unwrap_or_else(|| panic!("missing result {name}"));
    entry.get("ok").and_then(|v| v.as_bool()).unwrap_or(false)
}

fn decode_trust(g: &JsonValue) -> Trust {
    let raw = golden_str(g, "trust");
    serde_json::from_str(&raw).expect("trust decode")
}

fn decode_payload(g: &JsonValue) -> (Payload, Vec<u8>) {
    let raw = golden_str(g, "payload");
    let payload = serde_json::from_str(&raw).expect("payload decode");
    (payload, raw.into_bytes())
}

fn decode_candidate(g: &JsonValue) -> (Candidate, Vec<u8>) {
    let raw = golden_str(g, "candidate");
    let candidate = serde_json::from_str(&raw).expect("candidate decode");
    (candidate, raw.into_bytes())
}

#[test]
fn merge_policy_matches_oracle_semantically() {
    let g = goldens();
    let trust = decode_trust(&g);
    let original = br#"{"default":[{"type":"insecureAcceptAnything"}],"transports":{"docker":{"registry.example.com/app":[{"type":"insecureAcceptAnything"}]}}}"#;
    let merged = merge_policy(&trust, original).expect("policy.merge");
    // Preserved scopes are re-emitted (not byte-kept), so compare as DOM.
    let got: JsonValue = serde_json::from_slice(&merged).unwrap();
    let repository = format!("{}-release", trust.prefix);
    let requirement = &got["transports"]["docker"][&repository][0];
    assert_eq!(requirement["signedIdentity"]["dockerRepository"], repository);
    assert_eq!(requirement["type"], "sigstoreSigned");

    let conflict = format!(
        r#"{{"default":[{{"type":"insecureAcceptAnything"}}],"transports":{{"docker":{{"{}-release:tag":[{{"type":"insecureAcceptAnything"}}]}}}}}}"#,
        trust.prefix
    );
    let ok = golden_ok(&g, "policy.override_refused");
    let result = merge_policy(&trust, conflict.as_bytes());
    assert_eq!(result.is_ok(), ok);
}

#[test]
fn merge_policy_preserves_arbitrary_object_order_and_number_tokens() {
    let trust = decode_trust(&goldens());
    let original = br#"{"default":{"z":1e2,"a":{"y":-0,"x":1e400}},"transports":{"z-transport":{"scope":[{"second":-0,"first":1e400}]}}}"#;
    let merged = merge_policy(&trust, original).expect("policy.merge");
    let text = String::from_utf8(merged).unwrap();

    let default_start = text.find("\"default\": {").unwrap();
    let transports_start = text.find("\"transports\": {").unwrap();
    let default = &text[default_start..transports_start];
    assert!(default.find("\"z\": 1e2").unwrap() < default.find("\"a\": {").unwrap());
    assert!(default.find("\"y\": -0").unwrap() < default.find("\"x\": 1e400").unwrap());

    let custom = &text[text.find("\"z-transport\"").unwrap()..];
    assert!(custom.find("\"second\": -0").unwrap() < custom.find("\"first\": 1e400").unwrap());
}

/// Canonicalize a JSON DOM by sorting every object level.
fn dom_sort(value: JsonValue) -> JsonValue {
    match value {
        JsonValue::Object(entries) => {
            JsonValue::Object(entries.into_iter().map(|(k, v)| (k, dom_sort(v))).collect())
        }
        JsonValue::Array(items) => JsonValue::Array(items.into_iter().map(dom_sort).collect()),
        scalar => scalar,
    }
}

fn json_escape(s: &str) -> String {
    let mut out = String::from("\"");
    for c in s.chars() {
        match c {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            c => out.push(c),
        }
    }
    out.push('"');
    out
}

fn hex_sha256(data: &[u8]) -> String {
    soda_release_deliver::hash_bytes(data)
        .trim_start_matches("sha256:")
        .to_string()
}

fn temp_dir(prefix: &str) -> String {
    let dir = std::env::temp_dir().join(format!(
        "{prefix}-{}-{}",
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos()
    ));
    std::fs::create_dir_all(&dir).unwrap();
    dir.to_string_lossy().into_owned()
}

fn private_dir(prefix: &str) -> String {
    let dir = temp_dir(prefix);
    use std::os::unix::fs::PermissionsExt;
    std::fs::set_permissions(&dir, std::fs::Permissions::from_mode(0o700)).unwrap();
    dir
}

//! Current release-delivery behavior tests using captured original fixtures.
//!
//! Goldens were dumped from `internal/release/deliver` (since removed
//! temporary oracle test) and are embedded here. Fixture records are compared
//! semantically; original signed-input bytes and their signature/hash checks
//! remain byte-exact. Newly written OCI document layers are checked for
//! deterministic output and content round-trip, not the retired Go tar hash.

use serde_json::Value as JsonValue;
use soda_release_deliver::model::Candidate;
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

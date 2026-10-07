//! Oracle-differential tests: Rust behavior vs the Go `deliver` owner.
//!
//! Goldens were dumped from `internal/release/deliver` (since removed
//! temporary oracle test) and are embedded here. Fixture records are compared
//! semantically; original signed-input bytes and their signature/hash checks
//! remain byte-exact. Newly written OCI document layers are checked for
//! deterministic output and content round-trip, not the retired Go tar hash.

use serde_json::Value as JsonValue;
use soda_release_deliver::model::{
    admit_channel, admit_release, empty_state, valid_candidate_content, Candidate, Channel,
    Highwater, Permit, Release, Trust,
};
use soda_release_deliver::native::merge_policy;
use soda_release_deliver::payload::Payload;
use soda_release_deliver::publish::Ledger;

mod artifacts;
mod fetch_state;

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

fn golden_result(g: &JsonValue, name: &str) -> (bool, String) {
    let results = g.get("results").expect("results");
    let entry = results
        .get(name)
        .unwrap_or_else(|| panic!("missing result {name}"));
    let ok = entry.get("ok").and_then(|v| v.as_bool()).unwrap_or(false);
    let err = entry
        .get("err")
        .and_then(|v| v.as_str())
        .unwrap_or("")
        .to_string();
    (ok, err)
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

fn decode_release(g: &JsonValue) -> Release {
    let raw = golden_str(g, "release");
    serde_json::from_str(&raw).expect("release decode")
}

fn decode_channel(g: &JsonValue) -> Channel {
    let raw = golden_str(g, "channel");
    serde_json::from_str(&raw).expect("channel decode")
}

fn assert_record_semantics<T: serde::Serialize>(value: &T, expected: &str) {
    let mut first = serde_json::to_vec_pretty(value).unwrap();
    first.push(b'\n');
    let mut second = serde_json::to_vec_pretty(value).unwrap();
    second.push(b'\n');
    assert!(first.ends_with(b"\n"));
    assert_eq!(first, second);
    assert_eq!(
        serde_json::from_slice::<JsonValue>(&first).unwrap(),
        serde_json::from_str::<JsonValue>(expected).unwrap()
    );
}

#[test]
fn fixtures_round_trip_as_semantic_records() {
    let g = goldens();
    let trust = decode_trust(&g);
    assert_record_semantics(&trust, &golden_str(&g, "trust"));
    let (payload, _) = decode_payload(&g);
    assert_record_semantics(&payload, &golden_str(&g, "payload"));
    let (candidate, _) = decode_candidate(&g);
    assert_record_semantics(&candidate, &golden_str(&g, "candidate"));
    let release = decode_release(&g);
    assert_record_semantics(&release, &golden_str(&g, "release"));
    let channel = decode_channel(&g);
    assert_record_semantics(&channel, &golden_str(&g, "channel"));
}

#[test]
fn validation_battery_matches_oracle() {
    let g = goldens();
    let trust = decode_trust(&g);
    let (payload, payload_bytes) = decode_payload(&g);
    let (candidate, _) = decode_candidate(&g);
    let release = decode_release(&g);

    let check = |name: &str, result: Result<(), soda_release_deliver::Error>| {
        let (ok, err) = golden_result(&g, name);
        assert_eq!(result.is_ok(), ok, "{name} ok");
        if !ok {
            assert_eq!(result.unwrap_err().0, err, "{name} err");
        }
    };

    check("trust.validate", trust.validate());
    let mut bad_trust = trust.clone();
    bad_trust.epoch = 0;
    check("trust.zero_epoch", bad_trust.validate());
    let mut shared = trust.clone();
    let artifact = shared.keys["artifact"].clone();
    shared.keys.insert("stable".to_string(), artifact);
    check("trust.shared_key", shared.validate());

    check("payload.validate", payload.validate());
    let mut bad = payload.clone();
    bad.format = 2;
    check("payload.bad_format", bad.validate());
    let mut bad = payload.clone();
    bad.upgrade_from = Some(vec!["x".to_string()]);
    check("payload.upgrade_from", bad.validate());
    let mut bad = payload.clone();
    bad.images.get_mut("proxy").unwrap().reference = "wrong".to_string();
    check("payload.bad_reference", bad.validate());

    check(
        "candidate.validate",
        candidate.validate(&payload, &payload_bytes),
    );
    let mut bad = candidate.clone();
    bad.architecture = "aarch64".to_string();
    check("candidate.bad_arch", bad.validate(&payload, &payload_bytes));
    let mut bad = candidate.clone();
    bad.content_sha256.clear();
    assert!(!valid_candidate_content(&bad.content_sha256));
    check(
        "candidate.empty_content",
        bad.validate(&payload, &payload_bytes),
    );

    check("release.validate", release.validate(&trust).map(|_| ()));
    let mut bad = release.clone();
    bad.qualification = "bogus".to_string();
    check(
        "release.bad_qualification",
        bad.validate(&trust).map(|_| ()),
    );
    let mut bad = release.clone();
    bad.provenance.clear();
    check("release.empty_provenance", bad.validate(&trust).map(|_| ()));

    let now = golden_now(&g);
    let permit = Permit {
        format: 1,
        repository: format!("{}-release", trust.prefix),
        digest: format!("sha256:{}", "3".repeat(64)),
        previous: String::new(),
        expires: now + 100,
    };
    check("permit.validate", permit.validate(&trust, now));
    let mut expired = permit.clone();
    expired.expires = now - 1;
    check("permit.expired", expired.validate(&trust, now));

    let ledger = Ledger {
        format: 1,
        repository: format!("{}-release", trust.prefix),
        phase: "idle".to_string(),
        state: empty_state(),
        ..Ledger::default()
    };
    check("ledger.validate", ledger.validate(&trust));
}

/// Recover the oracle's `now` from the admitted state's CheckedAt.
fn golden_now(g: &JsonValue) -> i64 {
    let raw = golden_str(g, "channel.admitted_state");
    serde_json::from_str::<Highwater>(&raw)
        .expect("admitted decode")
        .checked_at
}

#[test]
fn channel_progression_matches_oracle() {
    let g = goldens();
    let trust = decode_trust(&g);
    let channel = decode_channel(&g);
    let now = golden_now(&g);
    let digest = format!("sha256:{}", "7".repeat(64));

    let mut state = empty_state();
    state.trust_epoch = trust.epoch;
    state.checked_at = now - 20;
    let next =
        admit_channel(&trust, &state, &channel, &digest, "candidate", now).expect("channel.admit");
    assert_record_semantics(&next, &golden_str(&g, "channel.admitted_state"));
    let (ok, _) = golden_result(&g, "channel.readmit_same");
    assert_eq!(
        admit_channel(&trust, &next, &channel, &digest, "candidate", now).is_ok(),
        ok
    );
    let (ok, err) = golden_result(&g, "channel.substitute_same_seq");
    let result = admit_channel(
        &trust,
        &next,
        &channel,
        &format!("sha256:{}", "8".repeat(64)),
        "candidate",
        now,
    );
    assert_eq!(result.is_ok(), ok);
    assert_eq!(result.unwrap_err().0, err);
    let mut old = channel.clone();
    old.sequence = 0;
    let (ok, err) = golden_result(&g, "channel.rewind");
    let result = admit_channel(&trust, &next, &old, &digest, "candidate", now);
    assert_eq!(result.is_ok(), ok);
    assert_eq!(result.unwrap_err().0, err);
}

#[test]
fn release_admission_matches_oracle() {
    let g = goldens();
    let trust = decode_trust(&g);
    let release = decode_release(&g);
    let now = golden_now(&g);
    let admitted: Highwater = {
        let raw = golden_str(&g, "channel.admitted_state");
        serde_json::from_str(&raw).unwrap()
    };
    let arch_ref = format!("{}-release@sha256:{}", trust.prefix, "9".repeat(64));
    let offer = Channel {
        format: 1,
        name: "candidate".to_string(),
        sequence: 1,
        issued: now - 10,
        expires: now + 600,
        withdrawn: false,
        releases: [("x86_64".to_string(), arch_ref.clone())]
            .into_iter()
            .collect(),
    };
    let next = admit_release(&trust, &admitted, &offer, "x86_64", &arch_ref, &release)
        .expect("release.admit_candidate");
    assert_record_semantics(&next, &golden_str(&g, "release.admitted_state"));
    let mut stable = offer.clone();
    stable.name = "stable".to_string();
    let (ok, err) = golden_result(&g, "release.stable_needs_native");
    let result = admit_release(&trust, &admitted, &stable, "x86_64", &arch_ref, &release);
    assert_eq!(result.is_ok(), ok);
    assert_eq!(result.unwrap_err().0, err);
}

#[test]
fn merge_policy_matches_oracle_semantically() {
    let g = goldens();
    let trust = decode_trust(&g);
    let original = br#"{"default":[{"type":"insecureAcceptAnything"}],"transports":{"docker":{"registry.example.com/app":[{"type":"insecureAcceptAnything"}]}}}"#;
    let merged = merge_policy(&trust, original).expect("policy.merge");
    // Preserved scopes are re-emitted (not byte-kept), so compare as DOM.
    let got: JsonValue = serde_json::from_slice(&merged).unwrap();
    let want: JsonValue = serde_json::from_str(&golden_str(&g, "policy.merged")).unwrap();
    assert_eq!(dom_sort(got), dom_sort(want));
    // New Soda scopes carry the exact sigstore requirement shape.
    let text = String::from_utf8(merged).unwrap();
    assert!(text.contains(r#""dockerRepository": ""#));
    assert!(text.contains(&format!("{}-release", trust.prefix)));
    assert!(text.contains("sigstoreSigned"));

    let conflict = format!(
        r#"{{"default":[{{"type":"insecureAcceptAnything"}}],"transports":{{"docker":{{"{}-release:tag":[{{"type":"insecureAcceptAnything"}}]}}}}}}"#,
        trust.prefix
    );
    let (ok, err) = golden_result(&g, "policy.override_refused");
    let result = merge_policy(&trust, conflict.as_bytes());
    assert_eq!(result.is_ok(), ok);
    assert_eq!(result.unwrap_err().0, err);
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

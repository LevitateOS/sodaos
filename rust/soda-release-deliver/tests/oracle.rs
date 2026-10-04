//! Oracle-differential tests: Rust behavior vs the Go `deliver` owner.
//!
//! Goldens were dumped from `internal/release/deliver` (since removed
//! temporary oracle test) and are embedded here. Fixture bytes must
//! round-trip byte-identically; validation outcomes and error text must
//! match the owner exactly.

use soda_json::JsonValue;
use soda_release_deliver::admission::admit_qualification;
use soda_release_deliver::check::check_candidate;
use soda_release_deliver::document::{read_document, write_document};
use soda_release_deliver::fetch::{fetch, init_state};
use soda_release_deliver::finalize::Config;
use soda_release_deliver::jsonx::{base64_decode, marshal, parse_lenient, parse_strict};
use soda_release_deliver::model::{
    admit_channel, admit_release, empty_state, valid_candidate_content, Candidate, Channel,
    Highwater, Permit, Release, Trust,
};
use soda_release_deliver::native::{merge_policy, Runner};
use soda_release_deliver::oci::{inspect_oci, inspect_oci_content};
use soda_release_deliver::payload::{load, Payload};
use soda_release_deliver::publish::Ledger;

const GOLDENS: &str = include_str!("goldens/deliver.json");

fn goldens() -> JsonValue {
    JsonValue::parse(GOLDENS).expect("goldens parse")
}

fn golden_str(g: &JsonValue, name: &str) -> String {
    g.get(name).and_then(|v| v.as_str()).unwrap_or("").to_string()
}

fn golden_result(g: &JsonValue, name: &str) -> (bool, String) {
    let results = g.get("results").expect("results");
    let entry = results.get(name).unwrap_or_else(|| panic!("missing result {name}"));
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
    let value = parse_strict(raw.as_bytes()).expect("trust strict");
    Trust::decode(&value).expect("trust decode")
}

fn decode_payload(g: &JsonValue) -> (Payload, Vec<u8>) {
    let raw = golden_str(g, "payload");
    let value = parse_strict(raw.as_bytes()).expect("payload strict");
    let payload = Payload::decode(&value).expect("payload decode");
    (payload, raw.into_bytes())
}

fn decode_candidate(g: &JsonValue) -> (Candidate, Vec<u8>) {
    let raw = golden_str(g, "candidate");
    let value = parse_strict(raw.as_bytes()).expect("candidate strict");
    let candidate = Candidate::decode(&value).expect("candidate decode");
    (candidate, raw.into_bytes())
}

fn decode_release(g: &JsonValue) -> Release {
    let raw = golden_str(g, "release");
    let value = parse_strict(raw.as_bytes()).expect("release strict");
    Release::decode(&value).expect("release decode")
}

fn decode_channel(g: &JsonValue) -> Channel {
    let raw = golden_str(g, "channel");
    let value = parse_strict(raw.as_bytes()).expect("channel strict");
    Channel::decode(&value).expect("channel decode")
}

#[test]
fn fixtures_round_trip_byte_identical() {
    let g = goldens();
    let trust = decode_trust(&g);
    assert_eq!(marshal(&trust), golden_str(&g, "trust").into_bytes());
    let (payload, _) = decode_payload(&g);
    assert_eq!(marshal(&payload), golden_str(&g, "payload").into_bytes());
    let (candidate, _) = decode_candidate(&g);
    assert_eq!(marshal(&candidate), golden_str(&g, "candidate").into_bytes());
    let release = decode_release(&g);
    assert_eq!(marshal(&release), golden_str(&g, "release").into_bytes());
    let channel = decode_channel(&g);
    assert_eq!(marshal(&channel), golden_str(&g, "channel").into_bytes());
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
    check(
        "candidate.bad_arch",
        bad.validate(&payload, &payload_bytes),
    );
    let mut bad = candidate.clone();
    bad.content_sha256.clear();
    assert!(!valid_candidate_content(&bad.content_sha256));
    check(
        "candidate.empty_content",
        bad.validate(&payload, &payload_bytes),
    );

    check(
        "release.validate",
        release.validate(&trust).map(|_| ()),
    );
    let mut bad = release.clone();
    bad.qualification = "bogus".to_string();
    check("release.bad_qualification", bad.validate(&trust).map(|_| ()));
    let mut bad = release.clone();
    bad.provenance.clear();
    check(
        "release.empty_provenance",
        bad.validate(&trust).map(|_| ()),
    );

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
    let value = parse_strict(raw.as_bytes()).expect("admitted strict");
    Highwater::decode(&value).expect("admitted decode").checked_at
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
    let next = admit_channel(&trust, &state, &channel, &digest, "candidate", now)
        .expect("channel.admit");
    assert_eq!(
        marshal(&next),
        golden_str(&g, "channel.admitted_state").into_bytes()
    );
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
        let value = parse_strict(raw.as_bytes()).unwrap();
        Highwater::decode(&value).unwrap()
    };
    let arch_ref = format!(
        "{}-release@sha256:{}",
        trust.prefix,
        "9".repeat(64)
    );
    let offer = Channel {
        format: 1,
        name: "candidate".to_string(),
        sequence: 1,
        issued: now - 10,
        expires: now + 600,
        withdrawn: false,
        releases: [("x86_64".to_string(), arch_ref.clone())].into_iter().collect(),
    };
    let next = admit_release(&trust, &admitted, &offer, "x86_64", &arch_ref, &release)
        .expect("release.admit_candidate");
    assert_eq!(
        marshal(&next),
        golden_str(&g, "release.admitted_state").into_bytes()
    );
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
    let got = JsonValue::parse(std::str::from_utf8(&merged).unwrap()).unwrap();
    let want = JsonValue::parse(&golden_str(&g, "policy.merged")).unwrap();
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

/// Canonicalize a JSON DOM by sorting every object level.
fn dom_sort(value: JsonValue) -> JsonValue {
    match value {
        JsonValue::Object(entries) => {
            let mut entries: Vec<(String, JsonValue)> = entries
                .into_iter()
                .map(|(k, v)| (k, dom_sort(v)))
                .collect();
            entries.sort_by(|a, b| a.0.cmp(&b.0));
            JsonValue::Object(entries)
        }
        JsonValue::Array(items) => {
            JsonValue::Array(items.into_iter().map(dom_sort).collect())
        }
        scalar => scalar,
    }
}

#[test]
fn write_document_digests_match_oracle() {
    let g = goldens();
    let channel = decode_channel(&g);
    let release = decode_release(&g);
    let dir = temp_dir("srd-doc");
    let channel_out = format!("{dir}/ch");
    let digest = write_document(&channel_out, &channel).expect("write channel");
    assert_eq!(digest, golden_str(&g, "document.channel_digest"));
    let back: Channel = read_document(&copy_as_dir(&channel_out), &digest, Channel::decode)
        .expect("read channel");
    assert_eq!(back, channel);

    let release_out = format!("{dir}/rel");
    let digest = write_document(&release_out, &release).expect("write release");
    assert_eq!(digest, golden_str(&g, "document.release_digest"));
    let back: Release = read_document(&copy_as_dir(&release_out), &digest, Release::decode)
        .expect("read release");
    assert_eq!(back, release);
}

/// `ReadDocument` reads the skopeo `dir:` copy shape (`manifest.json` plus
/// bare blobs); adapt a `WriteDocument` OCI layout into that shape.
fn copy_as_dir(layout: &str) -> String {
    let out = format!("{layout}-copy");
    std::fs::create_dir_all(&out).unwrap();
    // manifest.json in dir: copies is the manifest blob bytes.
    let index = std::fs::read(format!("{layout}/index.json")).unwrap();
    let value = parse_lenient(&index).unwrap();
    let soft = soda_json::JsonValue::parse(std::str::from_utf8(&index).unwrap()).unwrap();
    let _ = value;
    let manifests = soft.get("manifests").unwrap();
    let first = match manifests {
        JsonValue::Array(items) => items[0].clone(),
        _ => panic!("manifests"),
    };
    let digest = first.get("digest").and_then(|v| v.as_str()).unwrap();
    let hex = digest.strip_prefix("sha256:").unwrap();
    let manifest = std::fs::read(format!("{layout}/blobs/sha256/{hex}")).unwrap();
    std::fs::write(format!("{out}/manifest.json"), &manifest).unwrap();
    for entry in std::fs::read_dir(format!("{layout}/blobs/sha256")).unwrap() {
        let entry = entry.unwrap();
        let name = entry.file_name();
        std::fs::copy(
            format!("{layout}/blobs/sha256/{}", name.to_string_lossy()),
            format!("{out}/{}", name.to_string_lossy()),
        )
        .unwrap();
    }
    out
}

#[test]
fn oci_inspection_matches_oracle() {
    let g = goldens();
    let dir = temp_dir("srd-oci");
    let revision = "d".repeat(40);

    let plain = base64_decode(&golden_str(&g, "oci.plain")).unwrap();
    let plain_path = format!("{dir}/plain.oci");
    std::fs::write(&plain_path, &plain).unwrap();
    let image = inspect_oci(&plain_path, "x86_64", &revision).expect("oci.inspect_plain");
    let want_image = JsonValue::parse(&golden_str(&g, "oci.plain_image")).unwrap();
    assert_eq!(image.manifest, want_image.get("Manifest").unwrap().as_str().unwrap());
    assert_eq!(image.config, want_image.get("Config").unwrap().as_str().unwrap());
    assert_eq!(
        image.architecture,
        want_image.get("Architecture").unwrap().as_str().unwrap()
    );
    assert_eq!(image.revision, revision);

    let (_, observed) = inspect_oci_content(
        &plain_path,
        "x86_64",
        &revision,
        &["/usr/bin/app".to_string(), "/etc/config".to_string()],
    )
    .expect("oci.content_plain");
    let want_content = JsonValue::parse(&golden_str(&g, "oci.plain_content")).unwrap();
    for (path, hash) in &observed {
        assert_eq!(
            hash,
            want_content.get(path).unwrap().as_str().unwrap(),
            "{path}"
        );
    }
    assert_eq!(observed.len(), 2);

    let gzip = base64_decode(&golden_str(&g, "oci.gzip")).unwrap();
    let gzip_path = format!("{dir}/gzip.oci");
    std::fs::write(&gzip_path, &gzip).unwrap();
    let (_, observed) = inspect_oci_content(
        &gzip_path,
        "x86_64",
        &revision,
        &["/usr/bin/app".to_string()],
    )
    .expect("oci.content_gzip");
    let want_gzip = JsonValue::parse(&golden_str(&g, "oci.gzip_content")).unwrap();
    assert_eq!(
        observed["/usr/bin/app"],
        want_gzip.get("/usr/bin/app").unwrap().as_str().unwrap()
    );

    let (ok, _) = golden_result(&g, "oci.inspect_empty_revision");
    assert_eq!(inspect_oci(&plain_path, "x86_64", "").is_ok(), ok);
    let (ok, err) = golden_result(&g, "oci.content_missing");
    let result = inspect_oci_content(&plain_path, "x86_64", &revision, &["/missing".to_string()]);
    assert_eq!(result.is_ok(), ok);
    assert_eq!(result.unwrap_err().0, err);
}

#[test]
fn strict_decode_edges_match_oracle() {
    let g = goldens();
    let (ok, _) = golden_result(&g, "strict.duplicate");
    assert_eq!(parse_strict(br#"{"Format":1,"Format":2}"#).is_ok(), ok);
    let (ok, _) = golden_result(&g, "strict.unknown");
    let value = parse_strict(br#"{"Format":1,"Bogus":true}"#).unwrap();
    assert_eq!(Channel::decode(&value).is_ok(), ok);
    let (ok, _) = golden_result(&g, "strict.trailing");
    assert_eq!(parse_strict(b"{\"Format\":1} ").is_ok(), ok);
    let (ok, _) = golden_result(&g, "strict.trailing_garbage");
    assert_eq!(parse_strict(b"{\"Format\":1}x").is_ok(), ok);
    let (ok, _) = golden_result(&g, "strict.nonobject");
    assert_eq!(parse_strict(b"[1]").is_ok(), ok);
}

#[test]
fn payload_load_matches_owner() {
    let g = goldens();
    let dir = temp_dir("srd-load");
    let path = format!("{dir}/payload.json");
    std::fs::write(&path, golden_str(&g, "payload")).unwrap();
    let payload = load(&path).expect("load");
    let (want, _) = decode_payload(&g);
    assert_eq!(payload, want);
    // Build-JSON sites use Go's unknown-field error text.
    let bad_path = format!("{dir}/bad.json");
    std::fs::write(&bad_path, r#"{"Format":3,"Bogus":1}"#).unwrap();
    let err = load(&bad_path).unwrap_err().0;
    assert!(err.contains(r#"json: unknown field "Bogus""#), "{err}");
}

#[test]
fn state_and_ledger_init_round_trip() {
    let g = goldens();
    let trust = decode_trust(&g);
    let dir = private_dir("srd-state");
    let state_path = format!("{dir}/state.json");
    init_state(&state_path, &trust).expect("init state");
    let raw = std::fs::read(&state_path).unwrap();
    let value = parse_strict(&raw).unwrap();
    let state = Highwater::decode(&value).unwrap();
    assert_eq!(state.format, 1);
    assert_eq!(state.trust_epoch, trust.epoch);
    assert!(state.checked_at > 0);
    let ledger_path = format!("{dir}/ledger.json");
    let repo = format!("{}-release", trust.prefix);
    soda_release_deliver::publish::init_ledger(&ledger_path, &trust, &repo).expect("init ledger");
    let raw = std::fs::read(&ledger_path).unwrap();
    let value = parse_strict(&raw).unwrap();
    let ledger = Ledger::decode(&value).unwrap();
    assert_eq!(ledger.phase, "idle");
    ledger.validate(&trust).unwrap();
}

#[test]
fn check_candidate_binds_before_archives() {
    let g = goldens();
    let dir = temp_dir("srd-check");
    std::fs::write(format!("{dir}/payload.json"), golden_str(&g, "payload")).unwrap();
    std::fs::write(format!("{dir}/candidate.json"), golden_str(&g, "candidate")).unwrap();
    let (payload, _) = decode_payload(&g);
    let (candidate, _) = decode_candidate(&g);
    let err = check_candidate(&dir, "aarch64", &payload.revision, &candidate.forgejo_revision)
        .unwrap_err()
        .0;
    assert_eq!(err, "candidate architecture: release authority or completeness refused");
    let err = check_candidate(&dir, "x86_64", &"0".repeat(40), &candidate.forgejo_revision)
        .unwrap_err()
        .0;
    assert_eq!(err, "candidate soda revision: release authority or completeness refused");
    // Correct bindings reach the archive check, which fails without .oci files.
    let err = check_candidate(&dir, "x86_64", &payload.revision, &candidate.forgejo_revision)
        .unwrap_err()
        .0;
    assert!(err.contains("host archive"), "{err}");
}

#[test]
fn admit_qualification_accepts_bound_evidence() {
    let g = goldens();
    let dir = temp_dir("srd-admit");
    let payload_bytes = golden_str(&g, "payload").into_bytes();
    let (payload, _) = decode_payload(&g);
    let (candidate, _) = decode_candidate(&g);
    std::fs::write(format!("{dir}/payload.json"), &payload_bytes).unwrap();
    std::fs::write(format!("{dir}/candidate.json"), golden_str(&g, "candidate")).unwrap();
    let media_path = format!("{dir}-media.json");
    std::fs::write(&media_path, golden_str(&g, "media")).unwrap();
    let evidence_path = format!("{dir}-evidence.json");
    let media: JsonValue = JsonValue::parse(&golden_str(&g, "media")).unwrap();
    let evidence = format!(
        r#"{{"Format":1,"Outcome":"passed","Scope":"native-install-upgrade-recovery","Revision":{},"Architecture":{},"PayloadSHA256":{},"HostManifest":{},"ISOSHA256":{},"RootfsSHA256":{},"Checks":[{{"Name":"install","Outcome":"passed","Detail":""}},{{"Name":"upgrade","Outcome":"passed","Detail":""}},{{"Name":"recovery","Outcome":"passed","Detail":""}},{{"Name":"preservation","Outcome":"passed","Detail":""}}],"Fixture":false}}"#,
        json_escape(&payload.revision),
        json_escape(&payload.architecture),
        json_escape(&hex_sha256(&payload_bytes)),
        json_escape(&candidate.host.manifest),
        json_escape(media.get("ISO").unwrap().get("SHA256").unwrap().as_str().unwrap()),
        json_escape(media.get("Rootfs").unwrap().get("SHA256").unwrap().as_str().unwrap()),
    );
    std::fs::write(&evidence_path, &evidence).unwrap();
    let config = Config {
        serial: 7,
        class: "normal".to_string(),
        notes: "admit me".to_string(),
        ..Config::default()
    };
    let qualification = admit_qualification(&config, &dir, &media_path, &evidence_path)
        .expect("admit");
    assert_eq!(qualification.serial, 7);
    assert_eq!(qualification.scope, "native-install-upgrade-recovery");
    assert!(qualification.evidence.contains_key("qualification.json"));

    // Fixture evidence is explicitly non-qualifying.
    let bad_path = format!("{dir}-evidence-bad.json");
    std::fs::write(&bad_path, evidence.replace(r#""Fixture":false"#, r#""Fixture":true"#)).unwrap();
    let err = admit_qualification(&config, &dir, &media_path, &bad_path).unwrap_err().0;
    assert_eq!(err, "fixture evidence is explicitly non-qualifying");
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

/// Scripted runner proving fetch wiring without a network.
struct ScriptRunner {
    calls: std::sync::Mutex<Vec<Vec<String>>>,
    inspect_raw: Vec<u8>,
    copy_manifest: Vec<u8>,
    copy_config: Vec<u8>,
}

impl Runner for ScriptRunner {
    fn run(&self, args: &[&str]) -> Result<Vec<u8>, soda_release_deliver::Error> {
        self.calls
            .lock()
            .unwrap()
            .push(args.iter().map(|s| s.to_string()).collect());
        if args.contains(&"inspect") && args.contains(&"--raw") {
            return Ok(self.inspect_raw.clone());
        }
        if args.contains(&"inspect") && args.contains(&"--config") {
            return Ok(self.copy_config.clone());
        }
        if args.contains(&"copy") {
            // Materialize the destination dir copy the verifier reads.
            for arg in args {
                if let Some(dest) = arg.strip_prefix("dir:") {
                    // Destination layout: <dest>/manifest.json plus blobs.
                    std::fs::create_dir_all(dest).unwrap();
                    std::fs::write(format!("{dest}/manifest.json"), &self.copy_manifest).unwrap();
                }
            }
            return Ok(Vec::new());
        }
        Err(soda_release_deliver::Error::unavailable())
    }
}

#[test]
fn fetch_rejects_bad_runner_output() {
    let g = goldens();
    let trust = decode_trust(&g);
    let dir = private_dir("srd-fetch");
    let state_path = format!("{dir}/state.json");
    init_state(&state_path, &trust).unwrap();
    let runner = ScriptRunner {
        calls: std::sync::Mutex::new(Vec::new()),
        inspect_raw: b"not-a-manifest".to_vec(),
        copy_manifest: b"{}".to_vec(),
        copy_config: b"{}".to_vec(),
    };
    let out = format!("{dir}/out");
    let result = fetch(&runner, &trust, "candidate", "x86_64", &state_path, &out, 0);
    assert!(result.is_err());
    assert!(!runner.calls.lock().unwrap().is_empty());
}

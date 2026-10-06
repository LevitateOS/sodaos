use std::collections::BTreeMap;

use super::*;

fn full_content() -> BTreeMap<String, String> {
    [
        "dashboard:/usr/local/bin/soda-dashboard",
        "forgejo:/usr/local/bin/gitea",
        "extension:/usr/local/bin/gitea",
        "extension:/usr/share/soda/extension/extension.json",
        "extension:/usr/share/soda/extension/backend",
        "extension:/usr/share/soda/extension/run",
        "extension:/usr/share/soda/extension/assets/entry.js",
        "host:/usr/share/containers/systemd/forgejo.container",
        "host:/usr/share/containers/systemd/soda-dashboard.container",
        "host:/usr/lib/systemd/system/soda-extension-install.service",
    ]
    .iter()
    .map(|n| (n.to_string(), "a".repeat(64)))
    .collect()
}

#[test]
fn candidate_content_rules() {
    assert!(valid_candidate_content(&full_content()));
    let mut missing_asset = full_content();
    missing_asset.remove("extension:/usr/share/soda/extension/assets/entry.js");
    assert!(!valid_candidate_content(&missing_asset));
    let mut unknown = full_content();
    unknown.insert("host:/etc/passwd".to_string(), "b".repeat(64));
    assert!(!valid_candidate_content(&unknown));
    let mut mismatch = full_content();
    mismatch.insert("extension:/usr/local/bin/gitea".to_string(), "b".repeat(64));
    assert!(!valid_candidate_content(&mismatch));
    let mut bad_digest = full_content();
    bad_digest.insert(
        "dashboard:/usr/local/bin/soda-dashboard".to_string(),
        "zzz".to_string(),
    );
    assert!(!valid_candidate_content(&bad_digest));
    let mut bad_asset = full_content();
    bad_asset.insert(
        "extension:/usr/share/soda/extension/assets/../escape".to_string(),
        "a".repeat(64),
    );
    assert!(!valid_candidate_content(&bad_asset));
}

#[test]
fn path_clean_matches_go() {
    assert_eq!(path_clean(""), ".");
    assert_eq!(path_clean("a/b"), "a/b");
    assert_eq!(path_clean("a//b/./c"), "a/b/c");
    assert_eq!(path_clean("a/../b"), "b");
    assert_eq!(path_clean("../a"), "../a");
    assert_eq!(path_clean("a/../../b"), "../b");
    assert_eq!(path_clean("/../a"), "/a");
    assert_eq!(path_clean("a/"), "a");
}

#[test]
fn trust_reference_shapes() {
    let trust = Trust {
        prefix: "ghcr.io/example/sodaos".to_string(),
        ..Trust::default()
    };
    assert_eq!(
        trust.role("ghcr.io/example/sodaos-release").unwrap(),
        "artifact"
    );
    assert_eq!(
        trust.role("ghcr.io/example/sodaos-channel-stable").unwrap(),
        "stable"
    );
    assert!(trust.role("ghcr.io/other/x").is_err());
    assert!(trust.reference("no-at-sign").is_err());
    assert!(trust
        .reference("ghcr.io/example/sodaos-release@bogus")
        .is_err());
}

#[test]
fn trust_key_admission_preserves_cross_role_separation() {
    let key = "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEaxfR8uEsQkf4vOblY6RA8ncDfYEt\n6zOg9KE5RdiYwpZP40Li/hp/m47n60p8D54WK84zV2sxXs7LtkBoN79R9Q==\n-----END PUBLIC KEY-----".to_string();
    let trust = Trust {
        format: 1,
        prefix: "ghcr.io/example/sodaos".to_string(),
        epoch: 1,
        keys: ["artifact", "candidate", "preview", "stable"]
            .into_iter()
            .map(|role| (role.to_string(), vec![key.clone()]))
            .collect(),
        not_before: 1,
        max_age_seconds: 60,
        clock_skew_seconds: 0,
        minimum_sequence: ["candidate", "preview", "stable"]
            .into_iter()
            .map(|role| (role.to_string(), 1))
            .collect(),
    };
    assert_eq!(
        trust.validate().unwrap_err().0,
        "signer roles must not share keys"
    );
}

#[test]
fn producer_trust_fixture_keeps_its_raw_der_fingerprint() {
    let value = crate::jsonx::parse_strict(include_bytes!(
        "../../../../system/host/trust/release-trust.json"
    ))
    .expect("producer trust fixture JSON");
    let trust = Trust::decode(&value).expect("producer trust fixture");
    trust.validate().expect("producer trust fixture admits");
    let pem = trust
        .keys
        .get("artifact")
        .and_then(|keys| keys.first())
        .expect("artifact trust key");
    let der = soda_build_tools::trust_key::parse_p256_public_key(pem).expect("admitted SPKI");
    assert_eq!(
        crate::hash_bytes(&der),
        "sha256:caf16f22be044cdd5cfb72570ba2204e09f13d1a26ff8bfb6ff7aa79d76416fd"
    );
}

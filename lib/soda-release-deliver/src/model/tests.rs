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

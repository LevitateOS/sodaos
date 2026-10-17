use std::collections::BTreeMap;

use super::*;

fn full_content() -> BTreeMap<String, String> {
    soda_build_tools::reader::candidate::REQUIRED_CONTENT
        .iter()
        .copied()
        .chain([
            "package:soda-extension-0.1.0-linux-x86_64.tar.gz",
            "package:tailnet-extension-0.1.0-linux-x86_64.tar.gz",
        ])
        .map(|name| (name.to_string(), "a".repeat(64)))
        .collect()
}

#[test]
fn candidate_content_rules() {
    assert!(valid_candidate_content(&full_content()));
    let mut missing_asset = full_content();
    missing_asset.remove("package:soda-extension-0.1.0-linux-x86_64.tar.gz");
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

/// R50 protected admission: the valid pair admits, and every mutated
/// binding refuses — forged payload digest or bytes, mismatched soda or
/// Forgejo revision, architecture, source, provenance, prefix and format.
#[test]
fn candidate_admission_matrix() {
    let (bytes, p, c) = Candidate::fixture_pair();
    assert!(c.validate(&p, &bytes).is_ok(), "valid pair must admit");
    let mut forged_digest = c.clone();
    forged_digest.payload_sha256 = "f".repeat(64);
    assert!(forged_digest.validate(&p, &bytes).is_err());
    let mut tampered = bytes.clone();
    let last = tampered.len() - 1;
    tampered[last] ^= 0x01;
    assert!(c.validate(&p, &tampered).is_err());
    let mut mismatched_revision = c.clone();
    mismatched_revision.host.revision = "f".repeat(40);
    assert!(mismatched_revision.validate(&p, &bytes).is_err());
    let mut mismatched_arch = c.clone();
    mismatched_arch.architecture = "aarch64".to_string();
    assert!(mismatched_arch.validate(&p, &bytes).is_err());
    let mut mismatched_source = c.clone();
    mismatched_source.host.source = "https://example.com/fork".to_string();
    assert!(mismatched_source.validate(&p, &bytes).is_err());
    let mut mismatched_forgejo = c.clone();
    mismatched_forgejo.forgejo_revision = "f".repeat(40);
    assert!(mismatched_forgejo.validate(&p, &bytes).is_err());
    let mut empty_provenance = c.clone();
    empty_provenance.migration.clear();
    assert!(empty_provenance.validate(&p, &bytes).is_err());
    let mut bad_prefix = c.clone();
    bad_prefix.repository_prefix = "docker.io/example/sodaos".to_string();
    bad_prefix.host_reference =
        format!("docker.io/example/sodaos-host@{}", bad_prefix.host.manifest);
    assert!(bad_prefix.validate(&p, &bytes).is_err());
    let mut bad_format = c.clone();
    bad_format.format = 2;
    assert!(bad_format.validate(&p, &bytes).is_err());
    let mut bad_content = c.clone();
    bad_content
        .content_sha256
        .remove("package:soda-extension-0.1.0-linux-x86_64.tar.gz");
    assert!(bad_content.validate(&p, &bytes).is_err());
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
    assert!(trust.role("ghcr.io/example/sodaos-channel-stable").is_err());
    assert!(trust.role("ghcr.io/other/x").is_err());
    assert!(trust.reference("no-at-sign").is_err());
    assert!(trust
        .reference("ghcr.io/example/sodaos-release@bogus")
        .is_err());
}

#[test]
fn producer_trust_fixture_keeps_its_raw_der_fingerprint() {
    let trust: Trust = crate::json_serde::strict(include_bytes!(
        "../../../../system/host/trust/release-trust.json"
    ))
    .expect("producer trust fixture JSON");
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

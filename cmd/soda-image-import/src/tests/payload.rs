use super::super::*;
use super::fixtures::*;

fn decode(json: &str) -> Result<Payload, String> {
    decode_payload(json.as_bytes())
}

#[test]
fn payload_decode_is_strict_like_disallow_unknown_fields() {
    let valid = payload_json_for(&HashMap::new());
    let payload = decode(&valid).expect("valid payload");
    assert!(payload.validate().is_ok());
    assert_eq!(payload.images.len(), 6);
    // Trailing data rejected, like the Go payload test's suffix.
    assert!(decode(&format!("{valid} {{\"unexpected\":true}}")).is_err());
    // Unknown top-level and nested fields rejected.
    assert!(decode(&valid.replace("\"Format\":3", "\"Format\":3,\"Bogus\":1")).is_err());
    assert!(decode(&valid.replace(
        "\"ArchiveSHA256\":\"",
        "\"ArchiveSHA256\":\"\", \"Extra\":\"x\", \"Ignored\":\""
    ))
    .is_err());
    // Wrong types rejected.
    assert!(decode(&valid.replace("\"Format\":3", "\"Format\":\"3\"")).is_err());
    assert!(decode(&valid.replace("\"Format\":3", "\"Format\":3.0")).is_err());
    assert!(decode(&valid.replace("\"Images\":{", "\"Images\":[]")).is_err());
    // Malformed JSON and non-object top level rejected.
    assert!(decode("{\"Format\":}").is_err());
    assert!(decode("[]").is_err());
    // Duplicates keep last value; null is a no-op; fold matches.
    let dup = valid.replacen("\"Format\":3", "\"Format\":2,\"Format\":3", 1);
    assert_eq!(decode(&dup).expect("dup").format, 3);
    let nul = valid.replacen("\"Format\":3", "\"Format\":3,\"Format\":null", 1);
    assert_eq!(decode(&nul).expect("null").format, 3);
    let folded = valid.replacen("\"Format\":3", "\"format\":3", 1);
    assert_eq!(decode(&folded).expect("fold").format, 3);
    let invalid_earlier_image = valid.replacen(
        "\"dashboard\":{",
        "\"dashboard\":{\"Reference\":false},\"dashboard\":{",
        1,
    );
    assert!(decode(&invalid_earlier_image).is_err());
    let negative_zero = valid.replacen("\"Format\":3", "\"Format\":-0", 1);
    assert_eq!(
        decode(&negative_zero)
            .expect("negative zero integer")
            .format,
        0
    );
    // Null and missing UpgradeFrom both decode as empty.
    let null_up = valid.replace("\"UpgradeFrom\":[]", "\"UpgradeFrom\":null");
    assert!(decode(&null_up).expect("null list").upgrade_from.is_empty());
    let missing_up = valid.replace(",\"UpgradeFrom\":[]", "");
    assert!(decode(&missing_up)
        .expect("missing list")
        .upgrade_from
        .is_empty());
}

#[test]
fn payload_validation_rejects_go_test_mutations() {
    let root = test_root("payload-validate");
    let (base, _) = full_fixture(&root);
    assert!(base.validate().is_ok());
    let mut bad_format = base.clone();
    bad_format.format = 2;
    assert!(bad_format.validate().is_err());
    let mut bad_revision = base.clone();
    bad_revision.revision = "dirty".to_string();
    assert!(bad_revision.validate().is_err());
    let mut bad_arch = base.clone();
    bad_arch.architecture = "armv7".to_string();
    assert!(bad_arch.validate().is_err());
    let mut bad_presentation = base.clone();
    bad_presentation.presentation_sha256.clear();
    assert!(bad_presentation.validate().is_err());
    let mut bad_upgrade = base.clone();
    bad_upgrade.upgrade_from = vec!["unproved".to_string()];
    assert_eq!(
        bad_upgrade.validate().unwrap_err(),
        "candidate has no qualified upgrade paths"
    );
    let mut bad_prefix = base.clone();
    bad_prefix.repository_prefix = "ghcr.io/example/soda\nImage=untrusted".to_string();
    assert!(bad_prefix.validate().is_err());
    let mut missing_proxy = base.clone();
    missing_proxy.images.remove("proxy");
    assert_eq!(
        missing_proxy.validate().unwrap_err(),
        "complete image set required"
    );
    let mut bad_reference = base.clone();
    bad_reference.images.get_mut("dashboard").unwrap().reference =
        "ghcr.io/example/dashboard:latest".to_string();
    assert_eq!(
        bad_reference.validate().unwrap_err(),
        "invalid dashboard image binding"
    );
    let mut reused = base.clone();
    let forgejo_config = reused.images["forgejo"].config.clone();
    reused.images.get_mut("extension").unwrap().config = forgejo_config;
    assert!(reused
        .validate()
        .unwrap_err()
        .contains("independent image identity"));
}

#[test]
fn payload_load_enforces_regular_bounded_input() {
    let root = test_root("payload-load");
    let (payload, _) = full_fixture(&root);
    // Round-trip through JSON text built from the validated struct.
    let mut entries = HashMap::new();
    for name in NAMES {
        let binding = &payload.images[name];
        entries.insert(
            name.to_string(),
            (binding.config.clone(), binding.manifest.clone()),
        );
    }
    let json = payload_json_for(&entries);
    let path = root.join("release.json");
    fs::write(&path, &json).expect("write release");
    let loaded = Payload::load(&path).expect("load");
    assert!(loaded.validate().is_ok());
    assert_eq!(loaded.images.len(), 6);
    assert!(Payload::load(&root.join("missing.json")).is_err());
    fs::write(&path, format!("{json} {{\"unexpected\":true}}")).expect("trailing");
    assert!(Payload::load(&path).is_err());
    // Symlinked input refused like the Go regular-file check.
    let outside = root.join("outside.json");
    fs::write(&outside, &json).expect("outside");
    let link = root.join("linked.json");
    std::os::unix::fs::symlink(&outside, &link).expect("symlink");
    assert_eq!(
        Payload::load(&link).unwrap_err(),
        "bounded regular JSON input required"
    );
    // Oversized input refused.
    let big = root.join("big.json");
    fs::write(&big, vec![b' '; (4 << 20) + 1]).expect("big");
    assert_eq!(
        Payload::load(&big).unwrap_err(),
        "bounded regular JSON input required"
    );
}

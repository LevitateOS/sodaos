use crate::config::Config;
use crate::test_support::{hex_string, TestDir};
use std::fs;

use super::{apply_release_images, load_release_payload};

fn valid_release_json() -> (String, String, String) {
    let mut value: serde_json::Value = serde_json::from_str(include_str!(
        "../../../lib/release-inputs/fixtures/payload-v4.json"
    ))
    .unwrap();
    let rev = hex_string('b', 40);
    value["Revision"] = rev.clone().into();
    value["ID"] = format!("41.20250101.3.0.soda-{}", &rev[..12]).into();
    value["CoreOS"] = "41.20250101.3.0".into();
    for name in ["dashboard", "forgejo", "project-os", "canlang-runtime"] {
        value["Images"][name]["Provenance"]["Revision"] = rev.clone().into();
    }
    let json = serde_json::to_string(&value).unwrap();
    let project_cfg = value["Images"]["project-os"]["Config"]
        .as_str()
        .unwrap()
        .to_string();
    let tailnet_cfg = value["OptionalExtensions"]["tailnet"]["Images"]["tailscale"]["Config"]
        .as_str()
        .unwrap()
        .to_string();
    (json, project_cfg, tailnet_cfg)
}

#[test]
fn release_payload_accept_and_reject() {
    let dir = TestDir::make("release");
    let (valid, project_cfg, _tailnet_cfg) = valid_release_json();
    let path = dir.path("release.json");
    fs::write(&path, &valid).unwrap();
    let payload = load_release_payload(&path).expect("valid payload refused");
    assert_eq!(payload.architecture, "x86_64");
    assert_eq!(payload.project_os, project_cfg);
    // Each mutation is rejected.
    let mutations: Vec<(&str, String)> = vec![
        ("format", valid.replacen("\"Format\":4", "\"Format\":2", 1)),
        (
            "revision",
            valid.replacen(&hex_string('b', 40), &hex_string('b', 39), 1),
        ),
        (
            "id",
            valid.replacen(".soda-bbbbbbbbbbbb", ".soda-cccccccccccc", 1),
        ),
        ("coreos", valid.replacen("41.20250101.3.0", "41.1", 1)),
        ("arch", valid.replacen("\"x86_64\"", "\"aarch64\"", 1)),
        (
            "release-tag",
            valid.replacen("\"ReleaseTag\":\"v", "\"ReleaseTag\":\"", 1),
        ),
        ("schema", valid.replacen("\"Schema\":1", "\"Schema\":0", 1)),
        (
            "presentation",
            valid.replacen("abcdef1234567890", "abcdefg234567890", 1),
        ),
        (
            "reference",
            valid.replacen("-project-os@sha256:", "-project-os@sha257:", 1),
        ),
        (
            "forgejo-base",
            valid.replacen(
                "registry.example.invalid/fountain",
                "registry.example.invalid/other",
                1,
            ),
        ),
        (
            "missing-name",
            valid.replacen("\"tailnet\":{", "\"tailnet2\":{", 1),
        ),
        (
            "seventh",
            valid.replacen("\"UpgradeFrom\"", "\"extra\":1,\"UpgradeFrom\"", 1),
        ),
        (
            "unknown",
            valid.replacen("\"Format\":4,", "\"Format\":4,\"Bogus\":1,", 1),
        ),
        ("trailing", format!("{valid} {{}}")),
        (
            "trailing-comma",
            valid.replacen("\"UpgradeFrom\":[]}", "\"UpgradeFrom\":[],}", 1),
        ),
        (
            "null-images",
            valid.replacen(
                &valid[valid.find("\"Images\":").unwrap()..valid.find(",\"UpgradeFrom\"").unwrap()],
                "\"Images\":null",
                1,
            ),
        ),
    ];
    for (tag, bad) in &mutations {
        let p = dir.path(&format!("bad-{tag}.json"));
        fs::write(&p, bad).unwrap();
        assert!(
            load_release_payload(&p).is_err(),
            "mutation accepted: {tag}"
        );
    }
    // Duplicates, case-folded aliases and null UpgradeFrom are rejected:
    // Payload v4 has no last-wins or null-defaulting branch.
    let dup = valid.replacen("\"Format\":4,", "\"Format\":2,\"Format\":4,", 1);
    let p = dir.path("dup.json");
    fs::write(&p, &dup).unwrap();
    assert!(load_release_payload(&p).is_err(), "duplicate accepted");
    let alias_last_valid = valid.replacen("\"Format\":4,", "\"format\":2,\"Format\":4,", 1);
    fs::write(&p, &alias_last_valid).unwrap();
    assert!(
        load_release_payload(&p).is_err(),
        "exact later alias accepted"
    );
    let dup_image = valid.replacen(
        "\"Images\":{\"canlang-runtime\":{",
        "\"Images\":{\"canlang-runtime\":{},\"canlang-runtime\":{",
        1,
    );
    let p = dir.path("dup-image.json");
    fs::write(&p, &dup_image).unwrap();
    assert!(
        load_release_payload(&p).is_err(),
        "duplicate image accepted"
    );
    let null_upgrade = valid.replacen("\"UpgradeFrom\":[]", "\"UpgradeFrom\":null", 1);
    let p = dir.path("null-upgrade.json");
    fs::write(&p, &null_upgrade).unwrap();
    assert!(
        load_release_payload(&p).is_err(),
        "null UpgradeFrom accepted"
    );
    // Shape rejections: symlink, missing, directory, oversize.
    let link = dir.path("link.json");
    std::os::unix::fs::symlink(&path, &link).unwrap();
    assert!(load_release_payload(&link).is_err(), "symlink accepted");
    assert!(load_release_payload(&dir.path("missing.json")).is_err());
    assert!(load_release_payload(&dir.dir_str()).is_err());
    let big = dir.path("big.json");
    fs::write(&big, vec![b' '; (4 << 20) + 1]).unwrap();
    assert!(load_release_payload(&big).is_err(), "oversize accepted");
}

#[test]
fn release_images_apply_and_conflict() {
    let dir = TestDir::make("relapply");
    let (valid, project_cfg, _tailnet_cfg) = valid_release_json();
    let path = dir.path("release.json");
    fs::write(&path, &valid).unwrap();
    let mut c = Config::default();
    apply_release_images(&mut c, &path).unwrap();
    assert_eq!(c.image, project_cfg);
    c.image = String::from("other");
    assert_eq!(
        apply_release_images(&mut c, &path).unwrap_err(),
        "saved image selection conflicts with appliance release; explicit migration required"
    );
    let mut c = Config::default();
    assert_eq!(
        apply_release_images(&mut c, &dir.path("missing.json")).unwrap_err(),
        "immutable appliance image defaults unavailable"
    );
}

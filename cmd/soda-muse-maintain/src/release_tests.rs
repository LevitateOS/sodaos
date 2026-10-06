use crate::config::Config;
use crate::test_support::{hex_string, TestDir};
use std::fs;

use super::{apply_release_images, load_release_payload};

fn valid_release_json() -> (String, String, String) {
    let rev = hex_string('b', 40);
    let coreos = "41.20250101.3.0";
    let id = format!("{coreos}.soda-{}", &rev[..12]);
    let prefix = "ghcr.io/test/soda";
    let mut images = String::from("{");
    let mut cfgs = std::collections::HashMap::new();
    for (i, name) in [
        "dashboard",
        "forgejo",
        "extension",
        "proxy",
        "project-os",
        "tailnet",
    ]
    .iter()
    .enumerate()
    {
        let c = char::from_digit(i as u32 + 1, 16).unwrap();
        let m = char::from_digit(i as u32 + 7, 16).unwrap();
        let cfg = format!("sha256:{}", hex_string(c, 64));
        let man = format!("sha256:{}", hex_string(m, 64));
        let arch = hex_string('d', 64);
        let reference = format!("{prefix}-{name}@{man}");
        images.push_str(&format!(
            "\"{name}\":{{\"Reference\":\"{reference}\",\"Config\":\"{cfg}\",\"Manifest\":\"{man}\",\"ArchiveSHA256\":\"{arch}\"}},"
        ));
        cfgs.insert(*name, cfg);
    }
    images.pop();
    images.push('}');
    let base = format!(
        "quay.io/fedora/fedora-coreos@sha256:{}",
        hex_string('0', 64)
    );
    let json = format!(
        "{{\"Format\":3,\"ID\":\"{id}\",\"Revision\":\"{rev}\",\"Architecture\":\"x86_64\",\"CoreOS\":\"{coreos}\",\"Base\":\"{base}\",\"RepositoryPrefix\":\"{prefix}\",\"Schema\":1,\"PresentationSHA256\":\"{}\",\"HostPackagesSHA256\":\"{}\",\"Images\":{images},\"UpgradeFrom\":[]}}",
        hex_string('e', 64),
        hex_string('f', 64),
    );
    (json, cfgs["project-os"].clone(), cfgs["tailnet"].clone())
}

#[test]
fn release_payload_accept_and_reject() {
    let dir = TestDir::make("release");
    let (valid, project_cfg, tailnet_cfg) = valid_release_json();
    let path = dir.path("release.json");
    fs::write(&path, &valid).unwrap();
    let payload = load_release_payload(&path).expect("valid payload refused");
    assert_eq!(payload.architecture, "x86_64");
    assert_eq!(
        payload.image_config("project-os").as_deref(),
        Some(project_cfg.as_str())
    );
    assert_eq!(
        payload.image_config("tailnet").as_deref(),
        Some(tailnet_cfg.as_str())
    );
    // Each mutation is rejected.
    let forgejo_cfg = format!("sha256:{}", hex_string('2', 64));
    let ext_cfg = format!("sha256:{}", hex_string('3', 64));
    let mutations: Vec<(&str, String)> = vec![
        ("format", valid.replacen("\"Format\":3", "\"Format\":2", 1)),
        (
            "revision",
            valid.replacen(&hex_string('b', 40), &hex_string('b', 39), 1),
        ),
        ("id", valid.replacen(".soda-bbbbbbbbbbbb", ".soda-cccccccccccc", 1)),
        ("coreos", valid.replacen("41.20250101.3.0", "41.1", 1)),
        ("arch", valid.replacen("\"x86_64\"", "\"aarch64\"", 1)),
        ("prefix", valid.replacen("ghcr.io/test/soda", "docker.io/test/soda", 1)),
        ("schema", valid.replacen("\"Schema\":1", "\"Schema\":0", 1)),
        (
            "presentation",
            valid.replacen(&hex_string('e', 64), &format!("g{}", hex_string('e', 63)), 1),
        ),
        ("reference", valid.replacen("-project-os@sha256:", "-project-os@sha257:", 1)),
        ("ext-forgejo", valid.replacen(&ext_cfg, &forgejo_cfg, 1)),
        ("missing-name", valid.replacen("\"tailnet\":{", "\"tailnet2\":{", 1)),
        (
            "seventh",
            valid.replacen("\"UpgradeFrom\"", "\"extra\":{\"Reference\":\"\",\"Config\":\"\",\"Manifest\":\"\",\"ArchiveSHA256\":\"\"},\"UpgradeFrom", 1),
        ),
        ("unknown", valid.replacen("\"Format\":3,", "\"Format\":3,\"Bogus\":1,", 1)),
        ("trailing", format!("{valid} {{}}")),
        ("trailing-comma", valid.replacen("\"UpgradeFrom\":[]}", "\"UpgradeFrom\":[],}", 1)),
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
    // Duplicates apply last-wins; null UpgradeFrom stays empty.
    let dup = valid.replacen("\"Format\":3,", "\"Format\":2,\"Format\":3,", 1);
    let p = dir.path("dup.json");
    fs::write(&p, &dup).unwrap();
    assert!(load_release_payload(&p).is_ok(), "duplicate rejected");
    let dup_image = valid.replacen(
        "\"Images\":{\"dashboard\":{",
        "\"Images\":{\"dashboard\":{\"Reference\":\"\",\"Config\":\"\",\"Manifest\":\"\",\"ArchiveSHA256\":\"\"},\"dashboard\":{",
        1,
    );
    let p = dir.path("dup-image.json");
    fs::write(&p, &dup_image).unwrap();
    assert!(load_release_payload(&p).is_ok(), "duplicate image rejected");
    let null_upgrade = valid.replacen("\"UpgradeFrom\":[]", "\"UpgradeFrom\":null", 1);
    let p = dir.path("null-upgrade.json");
    fs::write(&p, &null_upgrade).unwrap();
    assert!(
        load_release_payload(&p).is_ok(),
        "null UpgradeFrom rejected"
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
    let (valid, project_cfg, tailnet_cfg) = valid_release_json();
    let path = dir.path("release.json");
    fs::write(&path, &valid).unwrap();
    let mut c = Config::default();
    apply_release_images(&mut c, &path).unwrap();
    assert_eq!(c.image, project_cfg);
    assert_eq!(c.tailnet_image, "");
    c.tailnet_management = true;
    apply_release_images(&mut c, &path).unwrap();
    assert_eq!(c.tailnet_image, tailnet_cfg);
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

use super::*;

fn fixture() -> Payload {
    let mut payload = Payload {
        format: 3,
        id: String::new(),
        revision: "a".repeat(40),
        architecture: "x86_64".to_string(),
        core_os: "44.20260817.3.2".to_string(),
        base: format!("quay.io/fedora/fedora-coreos@sha256:{}", "b".repeat(64)),
        repository_prefix: "ghcr.io/example/sodaos".to_string(),
        schema: 10,
        presentation_sha256: "c".repeat(64),
        host_packages_sha256: "d".repeat(64),
        images: BTreeMap::new(),
        upgrade_from: Vec::new(),
    };
    payload.id = format!("{}.soda-{}", payload.core_os, &payload.revision[..12]);
    for (i, name) in NAMES.iter().enumerate() {
        let mut byte = [i as u8];
        let _ = &mut byte;
        payload.images.insert(
            name.to_string(),
            Image {
                reference: format!(
                    "{}-{name}@sha256:{}",
                    payload.repository_prefix,
                    "e".repeat(64)
                ),
                manifest: format!("sha256:{}", "e".repeat(64)),
                config: format!("sha256:{}", buildx::sha256_hex(&[i as u8])),
                archive_sha256: "1".repeat(64),
            },
        );
    }
    payload
}

#[test]
fn payload_validation_matrix() {
    fixture().validate().unwrap();
    let mut bad = fixture();
    bad.format = 2;
    assert!(bad.validate().is_err());
    let mut bad = fixture();
    bad.revision = "dirty".to_string();
    assert!(bad.validate().is_err());
    let mut bad = fixture();
    bad.architecture = "armv7".to_string();
    assert_eq!(bad.validate().unwrap_err().to_string(), "expected x86_64");
    let mut bad = fixture();
    bad.presentation_sha256 = String::new();
    assert!(bad.validate().is_err());
    let mut bad = fixture();
    bad.upgrade_from = vec!["unproved".to_string()];
    assert!(bad.validate().is_err());
    let mut bad = fixture();
    bad.repository_prefix = "ghcr.io/example/soda\nImage=untrusted".to_string();
    assert!(bad.validate().is_err());
    let mut bad = fixture();
    bad.images.remove("proxy");
    assert_eq!(
        bad.validate().unwrap_err().to_string(),
        "complete image set required"
    );
    let mut bad = fixture();
    bad.images.get_mut("dashboard").unwrap().reference =
        "ghcr.io/example/dashboard:latest".to_string();
    assert!(bad
        .validate()
        .unwrap_err()
        .to_string()
        .contains("invalid dashboard image binding"));
    let mut bad = fixture();
    let forgejo = bad.images["forgejo"].config.clone();
    bad.images.get_mut("extension").unwrap().config = forgejo;
    assert!(bad
        .validate()
        .unwrap_err()
        .to_string()
        .contains("independent image identity"));
}

#[test]
fn load_refuses_trailing_data() {
    let dir = std::env::temp_dir().join(format!("soda-deliver-{}", std::process::id()));
    std::fs::create_dir_all(&dir).unwrap();
    let path = dir.join("release.json");
    let mut raw = crate::jsongo::serialize(&payload_json(&fixture())).into_bytes();
    raw.extend_from_slice(br#" {"unexpected":true}"#);
    std::fs::write(&path, &raw).unwrap();
    assert!(load(path.to_str().unwrap()).is_err());
    std::fs::remove_dir_all(&dir).unwrap();
}

fn payload_json(payload: &Payload) -> JsonValue {
    let mut images = Vec::new();
    for (name, image) in &payload.images {
        images.push((
            name.clone(),
            JsonValue::Object(vec![
                (
                    "Reference".to_string(),
                    JsonValue::Str(image.reference.clone()),
                ),
                ("Config".to_string(), JsonValue::Str(image.config.clone())),
                (
                    "Manifest".to_string(),
                    JsonValue::Str(image.manifest.clone()),
                ),
                (
                    "ArchiveSHA256".to_string(),
                    JsonValue::Str(image.archive_sha256.clone()),
                ),
            ]),
        ));
    }
    JsonValue::Object(vec![
        (
            "Format".to_string(),
            JsonValue::Number(payload.format.to_string()),
        ),
        ("ID".to_string(), JsonValue::Str(payload.id.clone())),
        (
            "Revision".to_string(),
            JsonValue::Str(payload.revision.clone()),
        ),
        (
            "Architecture".to_string(),
            JsonValue::Str(payload.architecture.clone()),
        ),
        (
            "CoreOS".to_string(),
            JsonValue::Str(payload.core_os.clone()),
        ),
        ("Base".to_string(), JsonValue::Str(payload.base.clone())),
        (
            "RepositoryPrefix".to_string(),
            JsonValue::Str(payload.repository_prefix.clone()),
        ),
        (
            "Schema".to_string(),
            JsonValue::Number(payload.schema.to_string()),
        ),
        (
            "PresentationSHA256".to_string(),
            JsonValue::Str(payload.presentation_sha256.clone()),
        ),
        (
            "HostPackagesSHA256".to_string(),
            JsonValue::Str(payload.host_packages_sha256.clone()),
        ),
        ("Images".to_string(), JsonValue::Object(images)),
        ("UpgradeFrom".to_string(), JsonValue::Array(Vec::new())),
    ])
}

fn payload_fixture() -> (Payload, String) {
    let mut payload = fixture();
    let layout = crate::oci::test_support::temp_dir("soda-deliver");
    let mut configs = BTreeMap::new();
    for name in NAMES {
        let config = crate::oci::test_support::add_image(&layout, name, "amd64", &payload.revision);
        configs.insert(config, name.to_string());
    }
    // Fetch each manifest digest from the written index via the fixture
    // annotation add_image records.
    let index = std::fs::read(format!("{layout}/index.json")).unwrap();
    let value = parse(&index).unwrap();
    if let JsonValue::Object(entries) = &value {
        for (key, val) in entries {
            if key != "manifests" {
                continue;
            }
            if let JsonValue::Array(items) = val {
                for item in items {
                    if let JsonValue::Object(item_entries) = item {
                        let mut digest = String::new();
                        let mut config = String::new();
                        for (field, field_value) in item_entries {
                            if field == "digest" {
                                if let JsonValue::Str(text) = field_value {
                                    digest = text.clone();
                                }
                            }
                            if field == "annotations" {
                                if let JsonValue::Object(annotations) = field_value {
                                    for (name, text) in annotations {
                                        if name == "org.opencontainers.image.ref.name" {
                                            if let JsonValue::Str(text) = text {
                                                config = text.clone();
                                            }
                                        }
                                    }
                                }
                            }
                        }
                        if let Some(name) = configs.get(&config) {
                            let prefix = payload.repository_prefix.clone();
                            let image = payload.images.get_mut(name.as_str()).unwrap();
                            image.reference = format!("{prefix}-{name}@{digest}");
                            image.config = config.clone();
                            image.manifest = digest;
                        }
                    }
                }
            }
        }
    }
    payload.validate().unwrap();
    (payload, layout)
}

#[test]
fn verify_content_binds_layout() {
    let (payload, layout) = payload_fixture();
    let (files, bytes) = verify_content(&payload, &layout).unwrap();
    assert_eq!(files.len(), 2 * NAMES.len() + 3);
    assert!(bytes > 0);
    let mut bad = payload.clone();
    bad.images.get_mut("dashboard").unwrap().manifest = format!("sha256:{}", "9".repeat(64));
    let reference = format!(
        "{}-dashboard@{}",
        bad.repository_prefix, bad.images["dashboard"].manifest
    );
    bad.images.get_mut("dashboard").unwrap().reference = reference;
    assert!(verify_content(&bad, &layout).is_err());
    assert!(verify_content(&payload, "relative/layout").is_err());
    let mut bad = payload.clone();
    bad.format = 2;
    assert!(verify_content(&bad, &layout).is_err());
}

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
    let dir = crate::oci::test_support::temp_dir("soda-deliver-trailing");
    let path = std::path::PathBuf::from(format!("{dir}/release.json"));
    let mut raw = serde_json::to_vec(&payload_json(&fixture())).unwrap();
    raw.extend_from_slice(br#" {"unexpected":true}"#);
    std::fs::write(&path, &raw).unwrap();
    assert!(load(path.to_str().unwrap()).is_err());
    std::fs::remove_dir_all(&dir).unwrap();
}

#[test]
fn payload_chooses_raw_alias_before_type_conversion() {
    let encoded = serde_json::to_string(&payload_json(&fixture())).unwrap();
    let overridden = encoded.replacen("\"Format\":3", "\"Format\":\"bad\",\"format\":3", 1);
    let raw: Box<serde_json::value::RawValue> = serde_json::from_str(&overridden).unwrap();
    assert_eq!(decode_payload(&raw).unwrap().format, 3);
    let nulled = encoded.replacen("\"Format\":3", "\"Format\":3,\"format\":null", 1);
    let raw: Box<serde_json::value::RawValue> = serde_json::from_str(&nulled).unwrap();
    assert_eq!(decode_payload(&raw).unwrap().format, 0);
    for number in ["3.0", "3e0", "9223372036854775808"] {
        let invalid = encoded.replacen("\"Format\":3", &format!("\"Format\":{number}"), 1);
        let raw: Box<serde_json::value::RawValue> = serde_json::from_str(&invalid).unwrap();
        assert!(decode_payload(&raw).is_err(), "accepted Format {number}");
    }
    let mut value = payload_json(&fixture());
    value["Images"]["dashboard"] = serde_json::Value::Null;
    value["UpgradeFrom"] = serde_json::json!([null]);
    let raw: Box<serde_json::value::RawValue> = serde_json::from_str(&value.to_string()).unwrap();
    let decoded = decode_payload(&raw).unwrap();
    assert_eq!(decoded.images["dashboard"], Image::default());
    assert_eq!(decoded.upgrade_from, vec![String::new()]);
}

#[test]
fn payload_validates_each_duplicate_image_value() {
    for earlier in ["[]", "{\"Reference\":false}"] {
        let text = format!("{{\"Images\":{{\"dashboard\":{earlier},\"dashboard\":{{}}}}}}");
        let raw: Box<serde_json::value::RawValue> = serde_json::from_str(&text).unwrap();
        assert!(decode_payload(&raw).is_err());
    }
    let raw: Box<serde_json::value::RawValue> = serde_json::from_str(
        r#"{"Images":{"dashboard":{"Reference":"first"},"dashboard":{"Reference":"last"}},"Schema":-0}"#,
    ).unwrap();
    let payload = decode_payload(&raw).unwrap();
    assert_eq!(payload.images["dashboard"].reference, "last");
    assert_eq!(payload.schema, 0);
}

fn payload_json(payload: &Payload) -> serde_json::Value {
    let mut images = serde_json::Map::new();
    for (name, image) in &payload.images {
        images.insert(name.clone(), serde_json::json!({"Reference":image.reference,"Config":image.config,"Manifest":image.manifest,"ArchiveSHA256":image.archive_sha256}));
    }
    serde_json::json!({"Format":payload.format,"ID":payload.id,"Revision":payload.revision,"Architecture":payload.architecture,"CoreOS":payload.core_os,"Base":payload.base,"RepositoryPrefix":payload.repository_prefix,"Schema":payload.schema,"PresentationSHA256":payload.presentation_sha256,"HostPackagesSHA256":payload.host_packages_sha256,"Images":images,"UpgradeFrom":[]})
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
    let value: serde_json::Value = serde_json::from_slice(&index).unwrap();
    if let Some(items) = value.get("manifests").and_then(serde_json::Value::as_array) {
        for item in items {
            let digest = item
                .get("digest")
                .and_then(serde_json::Value::as_str)
                .unwrap_or_default();
            let config = item
                .get("annotations")
                .and_then(|v| v.get("org.opencontainers.image.ref.name"))
                .and_then(serde_json::Value::as_str)
                .unwrap_or_default();
            if let Some(name) = configs.get(config) {
                let prefix = payload.repository_prefix.clone();
                let image = payload.images.get_mut(name.as_str()).unwrap();
                image.reference = format!("{prefix}-{name}@{digest}");
                image.config = config.to_string();
                image.manifest = digest.to_string();
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

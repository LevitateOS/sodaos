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
    let mut raw = producer_payload_bytes(&fixture());
    raw.extend_from_slice(br#" {"unexpected":true}"#);
    std::fs::write(&path, &raw).unwrap();
    assert!(load(path.to_str().unwrap()).is_err());
    std::fs::remove_dir_all(&dir).unwrap();
}

#[test]
fn load_accepts_current_release_image_producer_output() {
    let dir = crate::oci::test_support::temp_dir("soda-deliver-current-producer");
    let path = std::path::PathBuf::from(format!("{dir}/release.json"));
    let expected = fixture();
    let bytes = producer_payload_bytes(&expected);
    std::fs::write(&path, &bytes).unwrap();

    assert_eq!(load(path.to_str().unwrap()).unwrap(), expected);
    assert!(bytes.ends_with(b"\n"));
    assert_eq!(std::fs::read(&path).unwrap(), bytes);
    assert_eq!(
        buildx::hash_file(path.to_str().unwrap()).unwrap(),
        buildx::sha256_hex(&bytes)
    );

    std::fs::remove_dir_all(&dir).unwrap();
}

#[test]
fn load_refuses_nonproducer_shapes_and_invalid_utf8() {
    let dir = crate::oci::test_support::temp_dir("soda-deliver-invalid-producer");
    let path = std::path::PathBuf::from(format!("{dir}/release.json"));
    let canonical = producer_payload_bytes(&fixture());
    let text = std::str::from_utf8(&canonical).unwrap();
    let cases = [
        text.replacen(r#""Format": 3,"#, r#""format": 3,"#, 1),
        text.replacen(r#""Format": 3,"#, r#""Format": null,"#, 1),
        text.replacen(
            r#"  "Schema": 10,
"#,
            "",
            1,
        ),
        text.replacen(r#""Schema": 10,"#, r#""Schema": "10","#, 1),
        text.replacen(r#""Format": 3,"#, "\"Format\": 3,\n  \"Unknown\": true,", 1),
        text.replacen(r#""Reference":"#, r#""reference":"#, 1),
    ];
    for invalid in cases {
        std::fs::write(&path, invalid.as_bytes()).unwrap();
        assert_eq!(
            load(path.to_str().unwrap()).unwrap_err().to_string(),
            "invalid payload document"
        );
    }

    let mut invalid_utf8 = canonical;
    let host = b"quay.io";
    let host_start = invalid_utf8
        .windows(host.len())
        .position(|window| window == host)
        .expect("producer Base host");
    invalid_utf8.splice(
        host_start..host_start + host.len(),
        b"quay\xff.io".iter().copied(),
    );
    std::fs::write(&path, invalid_utf8).unwrap();
    assert_eq!(
        load(path.to_str().unwrap()).unwrap_err().to_string(),
        "invalid payload document"
    );
    std::fs::remove_dir_all(&dir).unwrap();
}

fn producer_payload(payload: &Payload) -> soda_release_image::model::Payload {
    soda_release_image::model::Payload {
        format: payload.format,
        id: payload.id.clone(),
        revision: payload.revision.clone(),
        architecture: payload.architecture.clone(),
        core_os: payload.core_os.clone(),
        base: payload.base.clone(),
        repository_prefix: payload.repository_prefix.clone(),
        schema: payload.schema,
        presentation_sha256: payload.presentation_sha256.clone(),
        host_packages_sha256: payload.host_packages_sha256.clone(),
        images: payload
            .images
            .iter()
            .map(|(name, image)| {
                (
                    name.clone(),
                    soda_release_image::model::PayloadImage {
                        reference: image.reference.clone(),
                        config: image.config.clone(),
                        manifest: image.manifest.clone(),
                        archive_sha256: image.archive_sha256.clone(),
                    },
                )
            })
            .collect(),
        upgrade_from: payload.upgrade_from.clone(),
    }
}

fn producer_payload_bytes(payload: &Payload) -> Vec<u8> {
    let mut bytes = serde_json::to_vec_pretty(&producer_payload(payload)).unwrap();
    bytes.push(b'\n');
    bytes
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

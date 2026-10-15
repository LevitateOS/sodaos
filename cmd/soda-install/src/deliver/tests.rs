use super::*;

fn fixture() -> Payload {
    serde_json::from_str(include_str!(
        "../../../../lib/release-inputs/fixtures/payload-v4.json"
    ))
    .expect("shared v4 fixture")
}

#[test]
fn payload_validation_matrix() {
    fixture().validate().unwrap();
    let mut bad = fixture();
    bad.format = 3;
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
    Role::Proxy.binding_mut(&mut bad.images).reference = "not-a-reference".to_string();
    assert!(bad
        .validate()
        .unwrap_err()
        .to_string()
        .contains("invalid proxy image binding"));
    let mut bad = fixture();
    bad.images.forgejo.provenance = soda_build_tools::payload::ImageProvenance::Soda {
        revision: bad.revision.clone(),
        base: "quay.io/other@sha256:".to_string() + &"0".repeat(64),
    };
    assert!(bad
        .validate()
        .unwrap_err()
        .to_string()
        .contains("invalid forgejo image provenance"));
    let mut bad = fixture();
    bad.optional_extensions.tailnet.package_id = "soda".to_string();
    assert!(bad
        .validate()
        .unwrap_err()
        .to_string()
        .contains("invalid tailnet extension binding"));
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
        text.replacen(r#""Format": 4,"#, r#""format": 4,"#, 1),
        text.replacen(r#""Format": 4,"#, r#""Format": null,"#, 1),
        text.replacen(
            r#"  "Schema": 1,
"#,
            "",
            1,
        ),
        text.replacen(r#""Schema": 1,"#, r#""Schema": "1","#, 1),
        text.replacen(r#""Format": 4,"#, "\"Format\": 4,\n  \"Unknown\": true,", 1),
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

fn producer_payload(payload: &Payload) -> Payload {
    // The installer and the release-image producer share the canonical
    // Payload v4 definition; no conversion remains.
    payload.clone()
}

fn producer_payload_bytes(payload: &Payload) -> Vec<u8> {
    let mut bytes = serde_json::to_vec_pretty(&producer_payload(payload)).unwrap();
    bytes.push(b'\n');
    bytes
}

fn payload_fixture() -> (Payload, String) {
    use soda_build_tools::payload::ImageProvenance;
    let mut payload = fixture();
    let layout = crate::oci::test_support::temp_dir("soda-deliver");
    let mut configs = BTreeMap::new();
    for role in Role::iter() {
        let revision = match role.binding(&payload.images).provenance {
            ImageProvenance::Soda { .. } => payload.revision.clone(),
            ImageProvenance::Upstream { .. } => String::new(),
        };
        let config = crate::oci::test_support::add_image(&layout, role.name(), "amd64", &revision);
        configs.insert(config, role.name().to_string());
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
                let role = Role::from_name(name).expect("fixed role");
                let binding = role.binding_mut(&mut payload.images);
                let repo = binding
                    .reference
                    .rsplit_once('@')
                    .map(|(repo, _)| repo.to_string())
                    .unwrap_or_default();
                binding.reference = format!("{repo}@{digest}");
                binding.config = config.to_string();
                binding.manifest = digest.to_string();
            }
        }
    }
    payload.project_support.project_image = Role::ProjectOs.binding(&payload.images).config.clone();
    payload.validate().unwrap();
    (payload, layout)
}

#[test]
fn verify_content_binds_layout() {
    let (payload, layout) = payload_fixture();
    let (files, bytes) = verify_content(&payload, &layout).unwrap();
    assert_eq!(files.len(), 2 * Role::iter().count() + 3);
    assert!(bytes > 0);
    let mut bad = payload.clone();
    let dashboard = Role::Dashboard.binding_mut(&mut bad.images);
    dashboard.manifest = format!("sha256:{}", "9".repeat(64));
    let repo = dashboard
        .reference
        .rsplit_once('@')
        .map(|(repo, _)| repo.to_string())
        .unwrap_or_default();
    dashboard.reference = format!("{repo}@{}", dashboard.manifest.clone());
    assert!(verify_content(&bad, &layout).is_err());
    assert!(verify_content(&payload, "relative/layout").is_err());
    let mut bad = payload.clone();
    bad.format = 3;
    assert!(verify_content(&bad, &layout).is_err());
}

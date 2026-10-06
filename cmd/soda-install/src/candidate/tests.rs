use super::*;
use crate::deliver::NAMES;
use crate::jsongo::Soft;
use std::collections::BTreeMap;

pub fn candidate_root_fixture() -> (String, MediaIdentity, DiskInstallChoices) {
    let root = crate::oci::test_support::temp_dir("soda-candidate");
    let images = format!("{root}{}", deliver::IMAGES_PATH);
    std::fs::create_dir_all(&images).unwrap();
    let revision = "a".repeat(40);
    let hash = "b".repeat(64);
    let mut image_map = BTreeMap::new();
    for name in NAMES {
        let reference = crate::oci::test_support::add_image(&images, name, "amd64", &revision);
        // Fetch the manifest digest from the written index.
        let index = std::fs::read(format!("{images}/index.json")).unwrap();
        let value = parse(&index).unwrap();
        let mut manifest = String::new();
        if let JsonValue::Object(entries) = &value {
            for (key, val) in entries {
                if key == "manifests" {
                    if let JsonValue::Array(items) = val {
                        for item in items {
                            let soft = Soft::new(item).unwrap();
                            let annotations = soft.object("annotations").unwrap().unwrap();
                            let got = annotations
                                .string("org.opencontainers.image.ref.name")
                                .unwrap()
                                .unwrap();
                            if got == reference {
                                manifest = soft.string("digest").unwrap().unwrap();
                            }
                        }
                    }
                }
            }
        }
        image_map.insert(
            name.to_string(),
            deliver::Image {
                reference: format!("ghcr.io/example/sodaos-{name}@{manifest}"),
                config: reference,
                manifest,
                archive_sha256: "1".repeat(64),
            },
        );
    }
    let payload = deliver::Payload {
        format: 3,
        id: format!("44.20260817.3.2.soda-{}", &revision[..12]),
        revision: revision.clone(),
        architecture: crate::run::architecture(),
        core_os: "44.20260817.3.2".to_string(),
        base: format!("quay.io/fedora/fedora-coreos@sha256:{hash}"),
        repository_prefix: "ghcr.io/example/sodaos".to_string(),
        schema: 10,
        presentation_sha256: hash.clone(),
        host_packages_sha256: hash.clone(),
        images: image_map,
        upgrade_from: Vec::new(),
    };
    payload.validate().unwrap();
    // Serialize through the deliver test helper shape.
    let mut image_entries = Vec::new();
    for (name, image) in &payload.images {
        image_entries.push((
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
    let raw = serialize(&JsonValue::Object(vec![
        ("Format".to_string(), JsonValue::Number("3".to_string())),
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
        ("Images".to_string(), JsonValue::Object(image_entries)),
        ("UpgradeFrom".to_string(), JsonValue::Array(Vec::new())),
    ]));
    let release_path = format!("{root}{}", deliver::PATH);
    std::fs::create_dir_all(std::path::Path::new(&release_path).parent().unwrap()).unwrap();
    std::fs::write(&release_path, raw.as_bytes()).unwrap();
    let console_path = format!("{root}{CANDIDATE_INSTALLER_BINARY}");
    std::fs::create_dir_all(std::path::Path::new(&console_path).parent().unwrap()).unwrap();
    std::fs::write(&console_path, b"prebuilt fixture").unwrap();
    let media = MediaIdentity {
        architecture: payload.architecture.clone(),
        release: payload.core_os.clone(),
        revision,
        installer_version: "coreos-installer 0.26.0".to_string(),
        host_manifest: format!("sha256:{hash}"),
        payload_sha256: buildx::hash_file(&release_path).unwrap(),
        console_sha256: buildx::hash_file(&console_path).unwrap(),
    };
    let choices = DiskInstallChoices {
        hostname: "soda-tester".to_string(),
        password_hash: format!("$6$salt${}", "a".repeat(86)),
        subnet: "10.89.0.0/24".to_string(),
        ..DiskInstallChoices::default()
    };
    (root, media, choices)
}

#[test]
fn requirement_authenticates_all_images() {
    let (root, media, _) = candidate_root_fixture();
    let images = format!("{root}{}", deliver::IMAGES_PATH);
    let release_path = format!("{root}{}", deliver::PATH);
    let payload = deliver::load(&release_path).unwrap();
    let (_, unique_bytes) = deliver::verify_content(&payload, &images).unwrap();
    assert_eq!(candidate_requirement(&media, &root).unwrap(), unique_bytes);
    // Five media mutations refuse.
    let mut bad = media.clone();
    bad.host_manifest = "latest".to_string();
    assert!(candidate_requirement(&bad, &root).is_err());
    let mut bad = media.clone();
    bad.revision = "c".repeat(40);
    assert!(candidate_requirement(&bad, &root).is_err());
    let mut bad = media.clone();
    bad.payload_sha256 = "b".repeat(64);
    assert!(candidate_requirement(&bad, &root).is_err());
    let mut bad = media.clone();
    bad.console_sha256 = "b".repeat(64);
    assert!(candidate_requirement(&bad, &root).is_err());
    let mut bad = media.clone();
    bad.installer_version = "coreos-installer 0.25.0".to_string();
    assert!(candidate_requirement(&bad, &root).is_err());
    // A missing blob refuses.
    let (root2, media2, _) = candidate_root_fixture();
    let images2 = format!("{root2}{}", deliver::IMAGES_PATH);
    let payload2 = deliver::load(&format!("{root2}{}", deliver::PATH)).unwrap();
    let tailnet = payload2.images["tailnet"]
        .config
        .trim_start_matches("sha256:")
        .to_string();
    std::fs::remove_file(format!("{images2}/blobs/sha256/{tailnet}")).unwrap();
    assert!(candidate_requirement(&media2, &root2).is_err());
}

#[test]
fn destination_keeps_password_only_provisioning() {
    let template = br#"{"ignition":{"version":"3.5.0"},"storage":{"files":[]}}"#;
    let factory =
        br#"{"network":"soda-projects","bridge":"soda0","subnet":"","tailnet_management":false}"#;
    let choices = DiskInstallChoices {
        hostname: "soda-tester".to_string(),
        password_hash: format!("$6$salt${}", "a".repeat(86)),
        subnet: "10.89.0.0/24".to_string(),
        ..DiskInstallChoices::default()
    };
    let data = candidate_destination(template, factory, &choices).unwrap();
    let value = parse(&data).unwrap();
    let config = Soft::new(&value).unwrap();
    let passwd = config.object("passwd").unwrap().unwrap();
    let users = passwd.array("users").unwrap().unwrap();
    assert_eq!(users.len(), 1);
    let root = Soft::new(&users[0]).unwrap();
    assert_eq!(root.string("name").unwrap().unwrap(), "root");
    assert_eq!(
        root.string("passwordHash").unwrap().unwrap(),
        choices.password_hash
    );
    assert!(root.field("sshAuthorizedKeys").unwrap().is_none());
    let storage = config.object("storage").unwrap().unwrap();
    let files = storage.array("files").unwrap().unwrap();
    let mut seen = false;
    for file in files {
        let entry = Soft::new(file).unwrap();
        let path = entry.string("path").unwrap().unwrap();
        assert_ne!(path, "/etc/soda-installer/project-subnet");
        if path == "/etc/soda/host.json" {
            seen = true;
            assert_eq!(entry.integer("mode").unwrap().unwrap(), 0o600);
            let contents = entry.object("contents").unwrap().unwrap();
            let source = contents.string("source").unwrap().unwrap();
            let encoded = source.strip_prefix("data:;base64,").unwrap();
            let raw = crate::sshkey::b64_decode_go(encoded.as_bytes()).unwrap();
            let machine = parse(&raw).unwrap();
            let fields = Soft::new(&machine).unwrap();
            assert_eq!(fields.string("subnet").unwrap().unwrap(), choices.subnet);
            assert!(fields.field("image").unwrap().is_none());
            assert!(fields.field("tailnet_image").unwrap().is_none());
        }
    }
    assert!(seen);
    let text = String::from_utf8_lossy(&data);
    assert!(!text.contains("rpm-ostree install"));
    assert!(!text.contains("soda-install continue"));
    assert!(candidate_destination(
        br#"{"ignition":{"version":"3.5.0"},"storage":{"files":[{"path":"/etc/soda/host.json"}]}}"#,
        factory,
        &choices
    )
    .is_err());
    assert!(candidate_destination(template, br#"{"subnet":"10.0.0.0/24"}"#, &choices).is_err());
}

use super::*;
use crate::deliver::Role;

const FIXTURE: &str = include_str!("../../../../lib/release-inputs/fixtures/payload-v4.json");

pub fn candidate_root_fixture() -> (String, MediaIdentity, DiskInstallChoices) {
    let root = crate::oci::test_support::temp_dir("soda-candidate");
    let images = format!("{root}{}", deliver::IMAGES_PATH);
    std::fs::create_dir_all(&images).unwrap();
    let revision = "a".repeat(40);
    let hash = "b".repeat(64);
    let base = format!("quay.io/fedora/fedora-coreos@sha256:{hash}");
    let mut payload: deliver::Payload = serde_json::from_str(FIXTURE).unwrap();
    // Pin the fixture to this candidate's revision, release and base.
    payload.revision = revision.clone();
    payload.core_os = "44.20260817.3.2".to_string();
    payload.id = format!("{}.soda-{}", payload.core_os, &revision[..12]);
    payload.base = base.clone();
    for role in Role::iter() {
        let binding = role.binding_mut(&mut payload.images);
        if let soda_build_tools::payload::ImageProvenance::Soda {
            revision: revision_slot,
            base: base_slot,
        } = &mut binding.provenance
        {
            *revision_slot = revision.clone();
            if role != Role::Forgejo {
                *base_slot = base.clone();
            }
        }
        let built = crate::oci::test_support::add_image(&images, role.name(), "amd64", &revision);
        // Fetch the manifest digest from the written index.
        let index = std::fs::read(format!("{images}/index.json")).unwrap();
        let value: serde_json::Value = serde_json::from_slice(&index).unwrap();
        if let Some(items) = value.get("manifests").and_then(serde_json::Value::as_array) {
            for item in items {
                let got = item
                    .get("annotations")
                    .and_then(|v| v.get("org.opencontainers.image.ref.name"))
                    .and_then(serde_json::Value::as_str);
                if got == Some(built.as_str()) {
                    let manifest = item
                        .get("digest")
                        .and_then(serde_json::Value::as_str)
                        .unwrap_or_default()
                        .to_string();
                    let repo = binding
                        .reference
                        .rsplit_once('@')
                        .map(|(repo, _)| repo.to_string())
                        .unwrap_or_default();
                    binding.reference = format!("{repo}@{manifest}");
                    binding.config = built.clone();
                    binding.manifest = manifest;
                }
            }
        }
    }
    payload.project_support.sha256 = hash.clone();
    payload.project_support.project_image = Role::ProjectOs.binding(&payload.images).config.clone();
    payload.validate().unwrap();
    let mut raw = serde_json::to_vec_pretty(&payload).unwrap();
    raw.push(b'\n');
    let release_path = format!("{root}{}", deliver::PATH);
    std::fs::create_dir_all(std::path::Path::new(&release_path).parent().unwrap()).unwrap();
    std::fs::write(&release_path, &raw).unwrap();
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
    let original_payload = std::fs::read(&release_path).unwrap();
    let mut equivalent_payload = original_payload.clone();
    equivalent_payload.push(b' ');
    std::fs::write(&release_path, &equivalent_payload).unwrap();
    assert_eq!(deliver::load(&release_path).unwrap(), payload);
    assert!(candidate_requirement(&media, &root).is_err());
    std::fs::write(&release_path, original_payload).unwrap();
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
    let mut alternate_installer = media.clone();
    alternate_installer.installer_version = "coreos-installer 0.27.1".to_string();
    assert_eq!(
        candidate_requirement(&alternate_installer, &root).unwrap(),
        unique_bytes,
        "an authenticated recorded installer version is not restricted to one literal release"
    );
    let mut bad = media.clone();
    bad.installer_version = "coreos-installer ".to_string();
    assert!(candidate_requirement(&bad, &root).is_err());
    // A missing blob refuses.
    let (root2, media2, _) = candidate_root_fixture();
    let images2 = format!("{root2}{}", deliver::IMAGES_PATH);
    let payload2 = deliver::load(&format!("{root2}{}", deliver::PATH)).unwrap();
    let canlang = Role::CanlangRuntime
        .binding(&payload2.images)
        .config
        .trim_start_matches("sha256:")
        .to_string();
    std::fs::remove_file(format!("{images2}/blobs/sha256/{canlang}")).unwrap();
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
    let value: serde_json::Value = serde_json::from_slice(&data).unwrap();
    let passwd = value.get("passwd").unwrap();
    let users = passwd.get("users").unwrap().as_array().unwrap();
    assert_eq!(users.len(), 1);
    let root = users[0].as_object().unwrap();
    assert_eq!(
        root.get("name").and_then(serde_json::Value::as_str),
        Some("root")
    );
    assert_eq!(
        root.get("passwordHash").and_then(serde_json::Value::as_str),
        Some(choices.password_hash.as_str())
    );
    assert!(root.get("sshAuthorizedKeys").is_none());
    let storage = value.get("storage").unwrap();
    let files = storage.get("files").unwrap().as_array().unwrap();
    let mut seen = false;
    for file in files {
        let entry = file.as_object().unwrap();
        let path = entry
            .get("path")
            .and_then(serde_json::Value::as_str)
            .unwrap();
        assert_ne!(path, "/etc/soda-installer/project-subnet");
        if path == "/etc/soda/host.json" {
            seen = true;
            assert_eq!(
                entry.get("mode").and_then(serde_json::Value::as_u64),
                Some(0o600)
            );
            let contents = entry.get("contents").unwrap();
            let source = contents
                .get("source")
                .and_then(serde_json::Value::as_str)
                .unwrap();
            let encoded = source.strip_prefix("data:;base64,").unwrap();
            let raw = crate::sshkey::b64_decode_go(encoded.as_bytes()).unwrap();
            let machine: serde_json::Value = serde_json::from_slice(&raw).unwrap();
            assert_eq!(
                machine.get("subnet").and_then(serde_json::Value::as_str),
                Some(choices.subnet.as_str())
            );
            assert!(machine.get("image").is_none());
            assert!(machine.get("tailnet_image").is_none());
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
    assert!(candidate_destination(
        template,
        br#"{"network":"first","network":"last","bridge":"soda0","subnet":"","tailnet_management":false}"#,
        &choices,
    ).is_err());

    let escaped = candidate_destination(
        template,
        br#"{"network":"<>&","bridge":"soda0","subnet":"","tailnet_management":false}"#,
        &choices,
    )
    .unwrap();
    let value: serde_json::Value = serde_json::from_slice(&escaped).unwrap();
    let file = value["storage"]["files"]
        .as_array()
        .unwrap()
        .iter()
        .find(|file| file["path"] == "/etc/soda/host.json")
        .unwrap();
    let source = file["contents"]["source"].as_str().unwrap();
    let machine =
        crate::sshkey::b64_decode_go(source.strip_prefix("data:;base64,").unwrap().as_bytes())
            .unwrap();
    assert_eq!(
        String::from_utf8(machine).unwrap(),
        r#"{"bridge":"soda0","network":"\u003c\u003e\u0026","subnet":"10.89.0.0/24","tailnet_management":false}"#
    );
}

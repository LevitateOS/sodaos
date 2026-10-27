use serde_json::Value;
use std::collections::BTreeMap;
use std::fs;

use super::{candidate_check_bin, run, TempDir};

const DELIVERY_GOLDEN: &str =
    include_str!("../../../soda-release-deliver/tests/goldens/deliver.json");

#[test]
fn candidate_check_binary_binds_identity_before_archive_read() {
    let golden: Value = serde_json::from_str(DELIVERY_GOLDEN).expect("delivery golden");
    let payload = golden["payload"].as_str().expect("payload bytes");
    let candidate = golden["candidate"].as_str().expect("candidate bytes");
    let payload_value: Value = serde_json::from_str(payload).expect("payload record");
    let candidate_value: Value = serde_json::from_str(candidate).expect("candidate record");
    let arch = payload_value["Architecture"]
        .as_str()
        .expect("payload architecture");
    let soda_revision = payload_value["Revision"]
        .as_str()
        .expect("payload revision");
    let forgejo_revision = candidate_value["ForgejoRevision"]
        .as_str()
        .expect("candidate Forgejo revision");

    let dir = TempDir::new("candidate-check");
    let payload_path = dir.path.join("payload.json");
    let candidate_path = dir.path.join("candidate.json");
    fs::write(&payload_path, payload).expect("write payload");
    fs::write(&candidate_path, candidate).expect("write candidate");
    let payload_before = fs::read(&payload_path).expect("read payload");
    let candidate_before = fs::read(&candidate_path).expect("read candidate");
    assert!(!dir.path.join("host.oci").exists());

    let cases = [
        ("architecture", "aarch64", soda_revision, forgejo_revision),
        (
            "soda revision",
            arch,
            "0000000000000000000000000000000000000000",
            forgejo_revision,
        ),
        (
            "candidate forgejo revision",
            arch,
            soda_revision,
            "0000000000000000000000000000000000000000",
        ),
        ("host archive", arch, soda_revision, forgejo_revision),
    ];

    for (boundary, requested_arch, requested_soda, requested_forgejo) in cases {
        let args = [
            "--candidate".to_owned(),
            dir.path.to_string_lossy().into_owned(),
            "--arch".to_owned(),
            requested_arch.to_owned(),
            "--soda-revision".to_owned(),
            requested_soda.to_owned(),
            "--forgejo-revision".to_owned(),
            requested_forgejo.to_owned(),
        ];
        let (status, stdout, stderr) = run(&candidate_check_bin(), &dir.path, &args);
        assert_eq!(status, 1, "{boundary}: {stderr}");
        assert!(
            stdout.is_empty(),
            "{boundary}: unexpected stdout {stdout:?}"
        );
        assert_eq!(stderr.lines().count(), 1, "{boundary}: {stderr:?}");
        assert!(stderr.contains(boundary), "{boundary}: {stderr:?}");
    }

    assert_eq!(
        fs::read(&payload_path).expect("payload after check"),
        payload_before
    );
    assert_eq!(
        fs::read(&candidate_path).expect("candidate after check"),
        candidate_before
    );
    assert!(!dir.path.join("host.oci").exists());
}

fn sha256_hex(bytes: &[u8]) -> String {
    use sha2::Digest;
    format!("{:x}", sha2::Sha256::digest(bytes))
}

fn append_tar_file<W: std::io::Write>(archive: &mut tar::Builder<W>, path: &str, bytes: &[u8]) {
    let mut header = tar::Header::new_ustar();
    header.set_size(bytes.len() as u64);
    header.set_mode(0o644);
    header.set_uid(0);
    header.set_gid(0);
    header.set_mtime(0);
    header.set_cksum();
    archive
        .append_data(&mut header, path, bytes)
        .expect("append deterministic tar member");
}

fn oci_archive(revision: &str, variant: &str, base: &str, files: &[(&str, &[u8])]) -> Vec<u8> {
    let mut layer = tar::Builder::new(Vec::new());
    for (path, bytes) in files {
        append_tar_file(&mut layer, path.trim_start_matches('/'), bytes);
    }
    layer.finish().expect("finish layer tar");
    let layer = layer.into_inner().expect("read layer tar bytes");
    let layer_digest = sha256_hex(&layer);
    let base_digest = base.split_once('@').map(|(_, digest)| digest).unwrap_or("");
    let config = serde_json::json!({
        "architecture": "amd64",
        "os": "linux",
        "rootfs": {"type": "layers", "diff_ids": [format!("sha256:{layer_digest}")]},
        "config": {"Labels": {
            "org.opencontainers.image.revision": revision,
            "org.opencontainers.image.source": "https://github.com/LevitateOS/sodaos",
            "org.opencontainers.image.base.name": base,
            "org.opencontainers.image.base.digest": base_digest,
            "org.soda.fixture.variant": variant
        }}
    });
    let config = serde_json::to_vec(&config).expect("encode OCI config");
    let config_digest = sha256_hex(&config);
    let manifest = serde_json::json!({
        "schemaVersion": 2,
        "mediaType": "application/vnd.oci.image.manifest.v1+json",
        "config": {
            "mediaType": "application/vnd.oci.image.config.v1+json",
            "digest": format!("sha256:{config_digest}"),
            "size": config.len()
        },
        "layers": [{
            "mediaType": "application/vnd.oci.image.layer.v1.tar",
            "digest": format!("sha256:{layer_digest}"),
            "size": layer.len()
        }]
    });
    let manifest = serde_json::to_vec(&manifest).expect("encode OCI manifest");
    let manifest_digest = sha256_hex(&manifest);
    let index = serde_json::json!({
        "schemaVersion": 2,
        "manifests": [{
            "mediaType": "application/vnd.oci.image.manifest.v1+json",
            "digest": format!("sha256:{manifest_digest}"),
            "size": manifest.len()
        }]
    });
    let index = serde_json::to_vec(&index).expect("encode OCI index");
    let mut archive = tar::Builder::new(Vec::new());
    for (path, bytes) in [
        (format!("blobs/sha256/{layer_digest}"), layer),
        (format!("blobs/sha256/{config_digest}"), config),
        (format!("blobs/sha256/{manifest_digest}"), manifest),
        ("index.json".to_string(), index),
        (
            "oci-layout".to_string(),
            br#"{"imageLayoutVersion":"1.0.0"}"#.to_vec(),
        ),
    ] {
        append_tar_file(&mut archive, &path, &bytes);
    }
    archive.finish().expect("finish OCI archive");
    archive.into_inner().expect("read OCI archive bytes")
}

#[test]
fn candidate_check_binary_accepts_generated_full_archive_candidate() {
    let golden: Value = serde_json::from_str(DELIVERY_GOLDEN).expect("delivery golden");
    let mut payload: Value = serde_json::from_str(golden["payload"].as_str().unwrap()).unwrap();
    let mut candidate: Value = serde_json::from_str(golden["candidate"].as_str().unwrap()).unwrap();
    let source_release = "44.20260928.3.1.soda-bbbbbbbbbbbb";
    payload["UpgradeFrom"] = serde_json::json!([source_release]);
    let revision = payload["Revision"].as_str().unwrap().to_string();
    let arch = payload["Architecture"].as_str().unwrap().to_string();
    let base = payload["Base"].as_str().unwrap().to_string();
    let prefix = candidate["RepositoryPrefix"].as_str().unwrap().to_string();
    let fountain_image = payload["Fountain"]["Image"].as_str().unwrap().to_string();
    let dir = TempDir::new("candidate-check-valid");
    let images_dir = dir.path.join("images");
    fs::create_dir_all(&images_dir).unwrap();

    for package in payload["ExtensionCatalog"]["packages"]
        .as_array_mut()
        .unwrap()
    {
        package["sha256"] = sha256_hex(b"fixture component").into();
        package["size"] = serde_json::json!(b"fixture component".len());
    }
    let canonical: soda_build_tools::payload::PayloadV4 =
        serde_json::from_value(payload.clone()).unwrap();
    let catalog = canonical.projected_catalog_bytes().unwrap();
    let proof = serde_json::to_vec(&canonical.fountain).unwrap();
    fs::create_dir_all(dir.path.join("packages")).unwrap();
    fs::write(dir.path.join("packages/catalog.json"), &catalog).unwrap();
    let content_values: Vec<(&str, &[u8])> = soda_build_tools::reader::candidate::REQUIRED_CONTENT
        .iter()
        .copied()
        .chain([
            "package:soda-extension-0.0.0-golden-linux-x86_64.tar.gz",
            "package:tailnet-extension-0.0.0-golden-linux-x86_64.tar.gz",
        ])
        .map(|name| {
            (
                name,
                match name {
                    "host:/usr/share/soda/extensions/catalog.json" => catalog.as_slice(),
                    "release:fountain-release.json" => proof.as_slice(),
                    _ => b"fixture component".as_slice(),
                },
            )
        })
        .collect();
    for (name, bytes) in &content_values {
        if let Some(package) = name.strip_prefix("package:") {
            fs::create_dir_all(dir.path.join("packages")).unwrap();
            fs::write(dir.path.join("packages").join(package), bytes).unwrap();
        } else if let Some(release) = name.strip_prefix("release:") {
            fs::write(dir.path.join(release), bytes).unwrap();
        }
    }
    let content: BTreeMap<String, String> = content_values
        .iter()
        .map(|(path, bytes)| (path.to_string(), sha256_hex(bytes)))
        .collect();
    candidate["ContentSHA256"] = serde_json::to_value(&content).unwrap();

    let mut image_archives = serde_json::Map::new();
    for name in [
        "dashboard",
        "forgejo",
        "proxy",
        "postgres",
        "project-os",
        "canlang-runtime",
    ] {
        let mut files: Vec<(&str, &[u8])> = content_values
            .iter()
            .filter_map(|(path, bytes)| {
                path.split_once(':')
                    .filter(|(owner, _)| *owner == name)
                    .map(|(_, member)| (member, *bytes))
            })
            .collect();
        if name == "project-os" {
            files.push((
                "/usr/libexec/soda/project-terminal",
                b"fixture terminal support",
            ));
        }
        let soda = !matches!(name, "proxy" | "postgres");
        let image_revision = if soda { revision.as_str() } else { "" };
        let archive = oci_archive(image_revision, name, &base, &files);
        let archive_path = images_dir.join(format!("{name}.oci"));
        fs::write(&archive_path, &archive).unwrap();
        let inspected = soda_release_deliver::oci::inspect_oci(
            archive_path.to_str().unwrap(),
            &arch,
            image_revision,
        )
        .expect("inspect generated payload image");
        let reference = format!("{prefix}-{name}@{}", inspected.manifest);
        let provenance = if soda {
            let base = if name == "forgejo" {
                fountain_image.clone()
            } else {
                base.clone()
            };
            serde_json::json!({"Kind": "Soda", "Revision": revision, "Base": base})
        } else {
            serde_json::json!({"Kind": "Upstream", "Requested": reference})
        };
        payload["Images"][name] = serde_json::json!({
            "Reference": reference,
            "Config": inspected.config,
            "Manifest": inspected.manifest,
            "Platform": "linux/amd64",
            "Provenance": provenance
        });
        image_archives.insert(name.to_string(), sha256_hex(&archive).into());
    }
    candidate["ImageArchives"] = serde_json::Value::Object(image_archives);

    payload["ProjectSupport"]["SHA256"] = sha256_hex(b"fixture terminal support").into();
    payload["ProjectSupport"]["ProjectImage"] = payload["Images"]["project-os"]["Config"].clone();

    let inventory = serde_json::to_vec_pretty(&content).unwrap();
    let mut inventory = inventory;
    inventory.push(b'\n');
    let mut host_files: Vec<(&str, &[u8])> = content_values
        .iter()
        .filter_map(|(path, bytes)| {
            path.split_once(':')
                .filter(|(owner, _)| *owner == "host")
                .map(|(_, member)| (member, *bytes))
        })
        .collect();
    host_files.push(("/usr/share/soda/host-image/content.json", &inventory));
    host_files.push((
        "/usr/libexec/soda/project-terminal",
        b"fixture terminal support",
    ));
    host_files.push((
        "/usr/share/soda/project-support/project-terminal",
        b"fixture terminal support",
    ));
    let host_archive = oci_archive(&revision, "host", &base, &host_files);
    let host_path = dir.path.join("host.oci");
    fs::write(&host_path, &host_archive).unwrap();
    let host =
        soda_release_deliver::oci::inspect_oci(host_path.to_str().unwrap(), &arch, &revision)
            .expect("inspect generated host image");
    candidate["Host"] = serde_json::to_value(&host).unwrap();
    candidate["HostReference"] = format!("{prefix}-host@{}", host.manifest).into();
    candidate["HostArchiveSHA256"] = sha256_hex(&host_archive).into();

    let payload_bytes = serde_json::to_vec_pretty(&payload).unwrap();
    candidate["PayloadSHA256"] = sha256_hex(&payload_bytes).into();
    let candidate_bytes = serde_json::to_vec_pretty(&candidate).unwrap();
    let payload_path = dir.path.join("payload.json");
    let candidate_path = dir.path.join("candidate.json");
    fs::write(&payload_path, &payload_bytes).unwrap();
    fs::write(&candidate_path, &candidate_bytes).unwrap();
    let payload_before = fs::read(&payload_path).unwrap();
    let candidate_before = fs::read(&candidate_path).unwrap();

    let args = [
        "--candidate".to_string(),
        dir.path.to_string_lossy().into_owned(),
        "--arch".to_string(),
        arch,
        "--soda-revision".to_string(),
        revision,
        "--forgejo-revision".to_string(),
        candidate["ForgejoRevision"].as_str().unwrap().to_string(),
    ];
    let (status, stdout, stderr) = run(&candidate_check_bin(), &dir.path, &args);
    assert_eq!(status, 0, "{stderr}");
    assert!(stdout.is_empty(), "unexpected stdout: {stdout:?}");
    assert!(stderr.is_empty(), "unexpected stderr: {stderr:?}");

    // Candidate admission accepts a prospective upgrade before evidence exists.
    // The later authenticated promotion summary carries the exact receipt;
    // a caller's local receipt cannot replace that summary. This fixture checks
    // admission, not publication, boot or native transition qualification.
    use soda_release_deliver::upgrade::{self, EvidenceFile, PublishedQualification};
    let tag = "v9.9.9-upgrade-fixture";
    let rootfs = b"generated admission fixture, not bootable release media";
    let rootfs_sha = sha256_hex(rootfs);
    let rootfs_name = format!("{rootfs_sha}-rootfs.img");
    let media: soda_release_deliver::model::MediaBinding = serde_json::from_value(serde_json::json!({
        "Revision": payload["Revision"], "Architecture": payload["Architecture"],
        "HostManifest": candidate["Host"]["Manifest"],
        "PayloadSHA256": candidate["PayloadSHA256"], "DeploymentCommit": "c".repeat(64),
        "RootfsURL": format!("https://github.com/LevitateOS/sodaos/releases/download/{tag}/{rootfs_name}"),
        "ISO": {"Path":"installer.iso", "SHA256":sha256_hex(b"fixture iso"), "Bytes":11},
        "Rootfs": {"Path":rootfs_name, "SHA256":rootfs_sha, "Bytes":rootfs.len()}
    })).unwrap();
    fs::write(
        dir.path.join("media.json"),
        serde_json::to_vec(&media).unwrap(),
    )
    .unwrap();
    fs::write(dir.path.join(&media.rootfs.path), rootfs).unwrap();
    assert!(!dir.path.join(upgrade::QUALIFICATION_FILE).exists());
    let admitted = upgrade::admit_candidate_directory(&dir.path).unwrap();
    let qualification = upgrade::Qualification {
        format: 1,
        source_release: source_release.into(),
        source_support_sha256: "d".repeat(64),
        target_payload_sha256: admitted.candidate.payload_sha256.clone(),
        target_host_manifest: admitted.candidate.host.manifest.clone(),
        target_support_sha256: admitted.payload.project_support.sha256.clone(),
        target_project_image: admitted.payload.project_support.project_image.clone(),
        target_deployment_commit: media.deployment_commit.clone(),
        rootfs_sha256: media.rootfs.sha256.clone(),
        evidence: ["transition", "interrupted_recovery", "stopped_project"]
            .into_iter()
            .map(|kind| {
                (
                    kind.into(),
                    EvidenceFile {
                        sha256: sha256_hex(kind.as_bytes()),
                        bytes: kind.len() as u64,
                    },
                )
            })
            .collect(),
    };
    let published = PublishedQualification {
        format: 1,
        tag: tag.into(),
        soda_revision: admitted.payload.revision.clone(),
        candidate_sha256: sha256_hex(&candidate_before),
        payload_sha256: sha256_hex(&payload_before),
        qualification: qualification.clone(),
    };
    let summary = serde_json::json!({"upgrade": published});
    let notes = format!(
        "Fixture notes{}",
        upgrade::render_summary(summary.as_object().unwrap()).unwrap()
    );
    let parse = |text: &str| {
        upgrade::published_from_notes(
            text,
            tag,
            &published.soda_revision,
            &published.candidate_sha256,
            &published.payload_sha256,
        )
    };
    let parsed = parse(&notes).unwrap();
    assert_eq!(parsed, qualification);
    upgrade::admit_directory(
        &dir.path,
        source_release,
        &qualification.source_support_sha256,
        parsed,
    )
    .unwrap();
    // A local file exists in these refusal cases but is never a fallback.
    fs::write(
        dir.path.join(upgrade::QUALIFICATION_FILE),
        serde_json::to_vec(&qualification).unwrap(),
    )
    .unwrap();
    assert!(parse("no authenticated summary").is_err());
    assert!(parse(&format!("{notes}{notes}")).is_err());
    assert!(parse(&format!(
        "{}{}{}",
        upgrade::SUMMARY_START,
        "{}",
        upgrade::SUMMARY_END
    ))
    .is_err());
    assert!(parse(&"x".repeat((128 << 10) + 1)).is_err());
    let raw_summary = serde_json::to_string(&summary).unwrap();
    let duplicate = raw_summary.replacen(
        "{\"upgrade\":",
        &format!(
            "{{\"upgrade\":{},\"upgrade\":",
            serde_json::to_string(&published).unwrap()
        ),
        1,
    );
    assert!(parse(&format!(
        "{}{}{}",
        upgrade::SUMMARY_START,
        duplicate,
        upgrade::SUMMARY_END
    ))
    .is_err());
    for field in ["tag", "soda_revision", "candidate_sha256", "payload_sha256"] {
        let mut wrong = summary.clone();
        wrong["upgrade"][field] = "unrelated-input".into();
        assert!(
            parse(&format!(
                "{}{}{}",
                upgrade::SUMMARY_START,
                serde_json::to_string(&wrong).unwrap(),
                upgrade::SUMMARY_END
            ))
            .is_err(),
            "{field}"
        );
    }
    let mut wrong = qualification.clone();
    wrong.target_support_sha256 = "f".repeat(64);
    assert!(upgrade::admit_directory(
        &dir.path,
        source_release,
        &qualification.source_support_sha256,
        wrong
    )
    .is_err());
    let mut wrong = qualification.clone();
    wrong.target_deployment_commit = "f".repeat(64);
    assert!(upgrade::admit_directory(
        &dir.path,
        source_release,
        &qualification.source_support_sha256,
        wrong
    )
    .is_err());
    assert!(upgrade::admit_directory(
        &dir.path,
        source_release,
        &"f".repeat(64),
        qualification.clone()
    )
    .is_err());
    assert!(upgrade::admit_directory(
        &dir.path,
        "44.20260928.3.1.soda-cccccccccccc",
        &qualification.source_support_sha256,
        qualification.clone()
    )
    .is_err());

    fs::write(
        images_dir.join("dashboard.oci"),
        b"truncated fixture archive",
    )
    .unwrap();
    let (status, stdout, stderr) = run(&candidate_check_bin(), &dir.path, &args);
    assert_eq!(status, 1);
    assert!(stdout.is_empty(), "unexpected stdout: {stdout:?}");
    assert!(stderr.contains("dashboard archive"), "{stderr:?}");
    assert_eq!(fs::read(payload_path).unwrap(), payload_before);
    assert_eq!(fs::read(candidate_path).unwrap(), candidate_before);
}

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
    let revision = payload["Revision"].as_str().unwrap().to_string();
    let arch = payload["Architecture"].as_str().unwrap().to_string();
    let base = payload["Base"].as_str().unwrap().to_string();
    let prefix = payload["RepositoryPrefix"].as_str().unwrap().to_string();
    let dir = TempDir::new("candidate-check-valid");
    let images_dir = dir.path.join("images");
    fs::create_dir_all(&images_dir).unwrap();

    let content_values = [
        (
            "dashboard:/usr/local/bin/soda-dashboard",
            b"dashboard fixture".as_slice(),
        ),
        (
            "forgejo:/usr/local/bin/gitea",
            b"shared gitea fixture".as_slice(),
        ),
        ("extension:/usr/local/bin/gitea", b"shared gitea fixture"),
        ("extension:/usr/share/soda/extension/extension.json", b"{}"),
        ("extension:/usr/share/soda/extension/backend", b"backend"),
        (
            "extension:/usr/share/soda/extension/run",
            b"#!/bin/sh\nexit 0\n",
        ),
        (
            "extension:/usr/share/soda/extension/assets/app.js",
            b"fixture asset",
        ),
        (
            "host:/usr/share/containers/systemd/forgejo.container",
            b"[Container]\nImage=fixture\n",
        ),
        (
            "host:/usr/share/containers/systemd/soda-dashboard.container",
            b"[Container]\nImage=fixture\n",
        ),
        (
            "host:/usr/lib/systemd/system/soda-extension-install.service",
            b"[Service]\nType=oneshot\n",
        ),
    ];
    let content: BTreeMap<String, String> = content_values
        .iter()
        .map(|(path, bytes)| (path.to_string(), sha256_hex(bytes)))
        .collect();
    candidate["ContentSHA256"] = serde_json::to_value(&content).unwrap();

    for name in [
        "dashboard",
        "forgejo",
        "extension",
        "proxy",
        "project-os",
        "tailnet",
    ] {
        let files: Vec<(&str, &[u8])> = content_values
            .iter()
            .filter_map(|(path, bytes)| {
                path.split_once(':')
                    .filter(|(owner, _)| *owner == name)
                    .map(|(_, member)| (member, *bytes))
            })
            .collect();
        let image_revision = if name == "proxy" { "" } else { &revision };
        let archive = oci_archive(image_revision, name, &base, &files);
        let archive_path = images_dir.join(format!("{name}.oci"));
        fs::write(&archive_path, &archive).unwrap();
        let inspected = soda_release_deliver::oci::inspect_oci(
            archive_path.to_str().unwrap(),
            &arch,
            image_revision,
        )
        .expect("inspect generated payload image");
        payload["Images"][name] = serde_json::json!({
            "Reference": format!("{prefix}-{name}@{}", inspected.manifest),
            "Config": inspected.config,
            "Manifest": inspected.manifest,
            "ArchiveSHA256": sha256_hex(&archive)
        });
    }

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

    fs::write(images_dir.join("tailnet.oci"), b"truncated fixture archive").unwrap();
    let (status, stdout, stderr) = run(&candidate_check_bin(), &dir.path, &args);
    assert_eq!(status, 1);
    assert!(stdout.is_empty(), "unexpected stdout: {stdout:?}");
    assert!(stderr.contains("tailnet archive"), "{stderr:?}");
    assert_eq!(fs::read(payload_path).unwrap(), payload_before);
    assert_eq!(fs::read(candidate_path).unwrap(), candidate_before);
}

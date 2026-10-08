use super::*;
use std::sync::atomic::{AtomicU64, Ordering};

static COUNTER: AtomicU64 = AtomicU64::new(0);

fn temp_dir(tag: &str) -> std::path::PathBuf {
    let id = COUNTER.fetch_add(1, Ordering::SeqCst);
    let path = std::env::temp_dir().join(format!(
        "soda-reltools-artifacts-{tag}-{}-{id}",
        std::process::id()
    ));
    let _ = std::fs::remove_dir_all(&path);
    std::fs::create_dir_all(&path).unwrap();
    path
}

fn parse_list(args: &[&str]) -> Result<(String, ArtifactFlags), String> {
    parse_artifact_flags(&args.iter().map(|s| s.to_string()).collect::<Vec<_>>())
}

#[test]
fn flag_parse_matrix() {
    assert_eq!(parse_list(&[]).unwrap_err(), USAGE);
    let (action, f) = parse_list(&["inspect-oci", "--arch", "x86_64", "--revision", "r"]).unwrap();
    assert_eq!(action, "inspect-oci");
    assert_eq!(f.arch, "x86_64");
    assert_eq!(f.revision, "r");
    assert_eq!(
        parse_list(&["inspect-oci", "extra", "--arch", "x86_64"]).unwrap_err(),
        "invalid artifact command flags"
    );
    assert_eq!(
        parse_list(&["inspect-oci", "--bogus", "x"]).unwrap_err(),
        "invalid artifact command flags"
    );
    assert_eq!(
        parse_list(&["inspect-oci", "-h"]).unwrap_err(),
        HELP_REQUESTED
    );
    assert_eq!(
        parse_list(&["inspect-oci", "--help"]).unwrap_err(),
        HELP_REQUESTED
    );
    assert!(help_text().contains("--keyring"));
    assert!(run(&["inspect-oci".to_owned(), "--help".to_owned()]).is_ok());
}

#[test]
fn unknown_action_message() {
    let f = ArtifactFlags::default();
    assert_eq!(
        run_artifact_action("bogus", &f).unwrap_err(),
        "unknown artifact action; use fetch-coreos-iso for upstream ISO inputs; QCOW2 media delivery is not selected"
    );
}

#[test]
fn admission_order_matches_go() {
    let f = ArtifactFlags::default();
    assert_eq!(
        run_artifact_action("inspect-oci", &f).unwrap_err(),
        "expected x86_64"
    );
    assert_eq!(
        run_artifact_action("fetch-coreos", &f).unwrap_err(),
        "expected x86_64"
    );
    let f = ArtifactFlags {
        arch: "x86_64".to_owned(),
        signer: "deadbeef".to_owned(),
        ..f
    };
    assert_eq!(
        run_artifact_action("fetch-coreos", &f).unwrap_err(),
        "full trusted signer fingerprint required"
    );
    let f = ArtifactFlags {
        arch: "x86_64".to_owned(),
        source: "/nope".to_owned(),
        revision: "abc".to_owned(),
        ..ArtifactFlags::default()
    };
    assert_eq!(
        run_artifact_action("inspect-oci", &f).unwrap_err(),
        "full source revision required"
    );
}

#[test]
fn look_path_miss_message_matches_go() {
    assert_eq!(
        look_path("definitely-not-a-tool-xyz").unwrap_err(),
        "exec: \"definitely-not-a-tool-xyz\": executable file not found in $PATH"
    );
    assert!(look_path("sh").is_ok());
}

#[test]
fn private_destination_matrix() {
    let scratch = temp_dir("privdest");
    assert_eq!(
        private_destination("relative/out").unwrap_err(),
        "absolute private output required"
    );
    let world = scratch.join("world");
    std::fs::create_dir(&world).unwrap();
    assert_eq!(
        private_destination(world.join("o").to_str().unwrap()).unwrap_err(),
        "real private output parent required"
    );
    let private = scratch.join("p");
    std::fs::create_dir(&private).unwrap();
    std::fs::set_permissions(&private, std::fs::Permissions::from_mode(0o700)).unwrap();
    let dest = private.join("o.json");
    assert!(private_destination(dest.to_str().unwrap()).is_ok());
    std::fs::write(&dest, b"x").unwrap();
    assert_eq!(
        private_destination(dest.to_str().unwrap()).unwrap_err(),
        "output already exists or cannot be inspected"
    );
    let _ = std::fs::remove_dir_all(&scratch);
}

#[test]
fn fresh_directory_matrix() {
    let scratch = temp_dir("freshdir");
    assert!(fresh_directory("relative").is_err());
    let fresh = scratch.join("newdir");
    assert!(fresh_directory(fresh.to_str().unwrap()).is_ok());
    assert_eq!(
        std::fs::metadata(&fresh).unwrap().permissions().mode() & 0o777,
        0o700
    );
    assert!(fresh_directory(fresh.to_str().unwrap()).is_err());
    let _ = std::fs::remove_dir_all(&scratch);
}

#[test]
fn butane_conversion_lifecycle() {
    use std::io::Write;
    let scratch = temp_dir("butane");
    // Failure removes the partial output.
    let out = scratch.join("fail.json");
    let dest = std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(&out)
        .unwrap();
    let out_s = out.to_string_lossy().into_owned();
    let err = finish_butane_conversion(dest, &out_s, |dest| {
        let mut child_output = dest.try_clone().unwrap();
        let _ = child_output.write_all(b"partial");
        Err("butane boom".to_owned())
    })
    .unwrap_err();
    assert_eq!(err, "butane boom; partial output removed");
    assert!(std::fs::symlink_metadata(&out).is_err());
    // Success keeps the output.
    let out = scratch.join("ok.json");
    let dest = std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(&out)
        .unwrap();
    let out_s = out.to_string_lossy().into_owned();
    finish_butane_conversion(dest, &out_s, |dest| {
        let mut child_output = dest
            .try_clone()
            .map_err(|_| "output clone failed".to_owned())?;
        child_output
            .write_all(b"{}")
            .map_err(|_| "Butane write failed".to_owned())
    })
    .unwrap();
    assert!(out.is_file());
    let _ = std::fs::remove_dir_all(&scratch);
}

#[test]
fn butane_output_clone_failure_is_owned_and_cleanup_is_reported() {
    let scratch = temp_dir("butane-clone-failure");
    let out = scratch.join("failed.json");
    let dest = std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(&out)
        .unwrap();

    let error = finish_butane_conversion(dest, out.to_str().unwrap(), |_| {
        Err("Butane output descriptor clone failed".to_owned())
    })
    .unwrap_err();
    assert_eq!(
        error,
        "Butane output descriptor clone failed; partial output removed"
    );
    assert!(!out.exists());
    let _ = std::fs::remove_dir_all(&scratch);
}

#[test]
fn butane_cleanup_failure_preserves_primary_and_does_not_claim_removal() {
    let scratch = temp_dir("butane-unlink-failure");
    let out = scratch.join("failed.json");
    let moved = scratch.join("owned-output.json");
    let dest = std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(&out)
        .unwrap();

    let error = finish_butane_conversion(dest, out.to_str().unwrap(), |_| {
        std::fs::rename(&out, &moved).map_err(|_| "Butane conversion failed".to_owned())?;
        std::fs::create_dir(&out).map_err(|_| "Butane conversion failed".to_owned())?;
        Err("butane exited 2".to_owned())
    })
    .unwrap_err();
    assert_eq!(
        error,
        "butane exited 2; partial output removal failed; inspect output locally"
    );
    assert!(out.is_dir());
    assert!(moved.is_file());
    let _ = std::fs::remove_dir_all(&scratch);
}

fn fake_butane(scratch: &std::path::Path, name: &str, body: &str) -> std::path::PathBuf {
    use std::os::unix::fs::PermissionsExt;

    let path = scratch.join(name);
    std::fs::write(&path, format!("#!/bin/sh\n{body}\n")).unwrap();
    std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o700)).unwrap();
    path
}

fn private_output(path: &std::path::Path) -> std::fs::File {
    create_butane_output(path.to_str().unwrap()).unwrap()
}

#[test]
fn butane_output_is_private_and_exclusive() {
    use std::os::unix::fs::PermissionsExt;

    let scratch = temp_dir("butane-output-mode");
    let out = scratch.join("private.json");
    let file = private_output(&out);
    assert_eq!(file.metadata().unwrap().permissions().mode() & 0o777, 0o600);
    drop(file);
    assert!(create_butane_output(out.to_str().unwrap()).is_err());
    assert_eq!(
        std::fs::metadata(&out).unwrap().permissions().mode() & 0o777,
        0o600
    );
    let _ = std::fs::remove_dir_all(&scratch);
}

#[test]
fn butane_process_failure_and_timeout_are_reaped_before_output_cleanup() {
    use std::time::{Duration, Instant};

    let scratch = temp_dir("butane-process-cleanup");
    let input_path = scratch.join("source.yaml");
    std::fs::write(&input_path, b"source").unwrap();
    let failure = fake_butane(&scratch, "fail-butane", "printf partial; exit 7");
    let out = scratch.join("failed.json");
    let error = finish_butane_conversion(private_output(&out), out.to_str().unwrap(), |dest| {
        run_butane_with(
            std::fs::File::open(&input_path).unwrap(),
            dest,
            &failure,
            Duration::from_secs(2),
        )
    })
    .unwrap_err();
    assert_eq!(error, "butane exited 7; partial output removed");
    assert!(!out.exists());

    let timeout = fake_butane(
        &scratch,
        "timeout-butane",
        "printf '%s\\n' \"$$\" > \"$0.pid\"; exec sleep 60",
    );
    let timeout_pid = timeout.with_extension("pid");
    let out = scratch.join("timeout.json");
    let start = Instant::now();
    let error = finish_butane_conversion(private_output(&out), out.to_str().unwrap(), |dest| {
        run_butane_with(
            std::fs::File::open(&input_path).unwrap(),
            dest,
            &timeout,
            Duration::from_millis(500),
        )
    })
    .unwrap_err();
    assert_eq!(error, "Butane conversion timed out; partial output removed");
    assert!(start.elapsed() < Duration::from_secs(3));
    assert!(!out.exists());
    let pid: i32 = std::fs::read_to_string(timeout_pid)
        .unwrap()
        .trim()
        .parse()
        .unwrap();
    assert!(
        !std::path::Path::new(&format!("/proc/{pid}")).exists(),
        "timed-out Butane child {pid} was not reaped"
    );
    let _ = std::fs::remove_dir_all(&scratch);
}

#[test]
fn inspect_error_paths() {
    assert_eq!(
        inspect_artifact_oci("/nonexistent", "aarch64", "").unwrap_err(),
        "expected x86_64"
    );
    assert_eq!(
        inspect_artifact_oci("/nonexistent", "x86_64", "abc").unwrap_err(),
        "full source revision required"
    );
    assert!(inspect_artifact_oci("/nonexistent-oci-archive-xyz", "x86_64", "").is_err());
    // A non-archive regular file passes admission, then fails identity.
    let scratch = temp_dir("inspect-bad");
    let file = scratch.join("junk.oci");
    std::fs::write(&file, b"not a tar archive").unwrap();
    assert!(inspect_artifact_oci(file.to_str().unwrap(), "x86_64", "").is_err());
    let _ = std::fs::remove_dir_all(&scratch);
}

fn sha256_hex_test(data: &[u8]) -> String {
    use sha2::Digest;
    let mut hasher = sha2::Sha256::new();
    hasher.update(data);
    let mut out = String::with_capacity(64);
    for byte in hasher.finalize() {
        out.push_str(&format!("{byte:02x}"));
    }
    out
}

/// Single ustar file entry, 512-padded, without the end-of-archive
/// trailer (the caller appends the zero blocks once).
fn raw_tar_entry_test(name: &str, body: &[u8]) -> Vec<u8> {
    let mut header = [0u8; 512];
    header[..name.len()].copy_from_slice(name.as_bytes());
    header[100..108].copy_from_slice(b"0000644\0");
    header[108..116].copy_from_slice(b"0000000\0");
    header[116..124].copy_from_slice(b"0000000\0");
    let size = format!("{:011o}\0", body.len());
    header[124..136].copy_from_slice(size.as_bytes());
    header[136..148].copy_from_slice(b"00000000000\0");
    header[148..156].copy_from_slice(b"        ");
    header[156] = b'0';
    header[257..262].copy_from_slice(b"ustar");
    let sum: u32 = header.iter().map(|b| *b as u32).sum();
    let chksum = format!("{:06o}\0 ", sum);
    header[148..156].copy_from_slice(chksum.as_bytes());
    let mut out = Vec::new();
    out.extend_from_slice(&header);
    out.extend_from_slice(body);
    out.resize(out.len() + (512 - body.len() % 512) % 512, 0);
    out
}

#[test]
fn inspect_positive_returns_image_identity_json() {
    // Keep the CLI output assertion on a small archive fixture; delivery
    // owns OCI decoding.
    let revision = "a".repeat(40);
    let base = "b".repeat(64);
    let body = b"synthetic layer fixture; never executed";
    let mut layer = raw_tar_entry_test("fixture.txt", body);
    layer.extend_from_slice(&[0u8; 1024]);
    let layer_sum = sha256_hex_test(&layer);
    let config = format!(
        "{{\"architecture\":\"amd64\",\"os\":\"linux\",\"rootfs\":{{\"type\":\"layers\",\"diff_ids\":[\"sha256:{layer_sum}\"]}},\"config\":{{\"Labels\":{{\"org.opencontainers.image.revision\":\"{revision}\",\"org.opencontainers.image.source\":\"https://github.com/LevitateOS/sodaos\",\"org.opencontainers.image.base.name\":\"synthetic-base\",\"org.opencontainers.image.base.digest\":\"sha256:{base}\"}}}}}}"
    );
    let config_sum = sha256_hex_test(config.as_bytes());
    let manifest = format!(
        "{{\"schemaVersion\":2,\"config\":{{\"digest\":\"sha256:{config_sum}\",\"size\":{},\"mediaType\":\"application/vnd.oci.image.config.v1+json\"}},\"layers\":[{{\"digest\":\"sha256:{layer_sum}\",\"size\":{},\"mediaType\":\"application/vnd.oci.image.layer.v1.tar\"}}]}}",
        config.len(),
        layer.len()
    );
    let manifest_sum = sha256_hex_test(manifest.as_bytes());
    let index = format!(
        "{{\"schemaVersion\":2,\"manifests\":[{{\"digest\":\"sha256:{manifest_sum}\",\"size\":{},\"mediaType\":\"application/vnd.oci.image.manifest.v1+json\"}}]}}",
        manifest.len()
    );
    let mut archive = Vec::new();
    for (name, data) in [
        (format!("blobs/sha256/{layer_sum}"), layer),
        (format!("blobs/sha256/{config_sum}"), config.into_bytes()),
        (
            format!("blobs/sha256/{manifest_sum}"),
            manifest.into_bytes(),
        ),
        ("index.json".to_string(), index.into_bytes()),
        (
            "oci-layout".to_string(),
            br#"{"imageLayoutVersion":"1.0.0"}"#.to_vec(),
        ),
    ] {
        archive.extend_from_slice(&raw_tar_entry_test(&name, &data));
    }
    archive.extend_from_slice(&[0u8; 1024]);
    let scratch = temp_dir("inspect-ok");
    let file = scratch.join("image.oci");
    std::fs::write(&file, &archive).unwrap();
    let doc = inspect_artifact_oci(file.to_str().unwrap(), "x86_64", &revision).unwrap();
    let value: serde_json::Value = serde_json::from_str(&doc).unwrap();
    assert_eq!(value["Manifest"], format!("sha256:{manifest_sum}"));
    assert_eq!(value["Config"], format!("sha256:{config_sum}"));
    assert_eq!(value["Architecture"], "amd64");
    assert_eq!(value["Revision"], revision);
    assert_eq!(value["Source"], "https://github.com/LevitateOS/sodaos");
    assert_eq!(value["BaseName"], "synthetic-base");
    assert_eq!(value["BaseDigest"], format!("sha256:{base}"));
    assert_eq!(value.as_object().unwrap().len(), 7);
    assert!(!doc.ends_with('\n'));
    let _ = std::fs::remove_dir_all(&scratch);
}

#[test]
fn fetch_admission_errors_without_network() {
    let f = ArtifactFlags::default();
    assert_eq!(
        run_artifact_action("fetch-coreos", &f).unwrap_err(),
        "expected x86_64"
    );
    assert_eq!(
        run_artifact_action("fetch-coreos-iso", &f).unwrap_err(),
        "expected x86_64"
    );
    let f = ArtifactFlags {
        arch: "x86_64".to_owned(),
        signer: "short".to_owned(),
        ..ArtifactFlags::default()
    };
    assert_eq!(
        run_artifact_action("fetch-coreos", &f).unwrap_err(),
        "full trusted signer fingerprint required"
    );
    assert_eq!(
        run_artifact_action("fetch-coreos-iso", &f).unwrap_err(),
        "full trusted signer fingerprint required"
    );
    // Missing keyring fails hashing before any tool lookup or network.
    let f = ArtifactFlags {
        arch: "x86_64".to_owned(),
        signer: "f".repeat(40),
        keyring: "/nonexistent-keyring-xyz".to_owned(),
        ..ArtifactFlags::default()
    };
    assert!(run_artifact_action("fetch-coreos", &f).is_err());
    assert!(run_artifact_action("fetch-coreos-iso", &f).is_err());
}

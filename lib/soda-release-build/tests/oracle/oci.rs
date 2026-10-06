//! OCI oracle vectors: archive identity, tamper rejection, content
//! resolution, and layout inspection against Go ground truth.

use super::{data_path, oracle, scratch, FIXTURE_REVISION};
use soda_release_build::oci::{inspect_oci, inspect_oci_content};
use soda_release_build::oci_layout::inspect_oci_layout;
use std::collections::HashMap;

#[test]
fn oracle_go_fixture_identity() {
    let image = inspect_oci(&data_path("go-fixture.oci"), "x86_64", FIXTURE_REVISION).unwrap();
    assert_eq!(image.manifest, oracle::OCI_MANIFEST);
    assert_eq!(image.config, oracle::OCI_CONFIG);
    assert_eq!(image.architecture, oracle::OCI_ARCH);
    assert_eq!(image.revision, oracle::OCI_REVISION);
    assert_eq!(image.source, oracle::OCI_SOURCE);
    assert_eq!(image.base_name, oracle::OCI_BASENAME);
    assert_eq!(image.base_digest, oracle::OCI_BASEDIGEST);
}

#[test]
fn oracle_go_fixture_rejections() {
    let file = data_path("go-fixture.oci");
    assert_eq!(
        inspect_oci(&file, "aarch64", FIXTURE_REVISION)
            .unwrap_err()
            .message(),
        oracle::OCI_ERR_ARCH
    );
    assert_eq!(
        inspect_oci(&file, "x86_64", &"c".repeat(40))
            .unwrap_err()
            .message(),
        oracle::OCI_ERR_REVISION
    );
    let dir = scratch("tamper");
    let tampered = dir.join("tampered.oci");
    let bytes = std::fs::read(&file).unwrap();
    let needle = b"synthetic layer";
    let pos = bytes
        .windows(needle.len())
        .position(|w| w == needle)
        .unwrap();
    let mut tampered_bytes = bytes;
    tampered_bytes[pos..pos + needle.len()].copy_from_slice(b"tampered! layer");
    std::fs::write(&tampered, tampered_bytes).unwrap();
    assert_eq!(
        inspect_oci(&tampered, "x86_64", FIXTURE_REVISION)
            .unwrap_err()
            .message(),
        oracle::OCI_ERR_TAMPERED
    );
}

#[test]
fn oracle_go_fixture_content() {
    let file = data_path("go-fixture.oci");
    let (image, content) = inspect_oci_content(
        &file,
        "x86_64",
        FIXTURE_REVISION,
        &[String::from("/fixture.txt")],
    )
    .unwrap();
    assert_eq!(image.config, oracle::OCI_CONFIG);
    assert_eq!(
        content.get("/fixture.txt").unwrap(),
        oracle::OCI_FIXTURE_HASH
    );
    assert_eq!(
        inspect_oci_content(
            &file,
            "x86_64",
            FIXTURE_REVISION,
            &[String::from("/missing")]
        )
        .unwrap_err()
        .message(),
        oracle::OCI_ERR_MISSING
    );
}

#[test]
fn oracle_go_layout() {
    let dir = data_path("go-layout");
    let mut revisions = HashMap::new();
    revisions.insert(oracle::OCI_CONFIG.to_string(), FIXTURE_REVISION.to_string());
    let got = inspect_oci_layout(&dir, "x86_64", &revisions).unwrap();
    assert_eq!(got.images.len(), oracle::LAYOUT_IMAGES);
    assert_eq!(got.files.len(), oracle::LAYOUT_FILES);
    assert_eq!(got.bytes, oracle::LAYOUT_BYTES);
    assert_eq!(
        got.files.get("index.json").unwrap(),
        oracle::LAYOUT_INDEXHASH
    );
    let mut total = 0u64;
    for (name, hash) in &got.files {
        let bytes = std::fs::read(dir.join(name)).unwrap();
        assert_eq!(&soda_release_build::sha256_hex(&bytes), hash);
        total += bytes.len() as u64;
    }
    assert_eq!(total, got.bytes);
    assert_eq!(
        inspect_oci_layout(&dir, "aarch64", &revisions)
            .unwrap_err()
            .message(),
        oracle::LAYOUT_ERR_ARCH
    );
}

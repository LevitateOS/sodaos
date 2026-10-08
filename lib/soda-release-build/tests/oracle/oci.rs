//! OCI oracle vectors: archive identity, tamper rejection and content
//! resolution against retained ground truth.

use super::{data_path, oracle, scratch, FIXTURE_REVISION};
use soda_release_deliver::oci::{inspect_oci, inspect_oci_content};

#[test]
fn oracle_go_fixture_identity() {
    let image = inspect_oci(
        data_path("go-fixture.oci").to_str().unwrap(),
        "x86_64",
        FIXTURE_REVISION,
    )
    .unwrap();
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
    assert!(
        inspect_oci(file.to_str().unwrap(), "aarch64", FIXTURE_REVISION).is_err(),
        "oci-arch"
    );
    assert!(
        inspect_oci(file.to_str().unwrap(), "x86_64", &"c".repeat(40)).is_err(),
        "oci-revision"
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
    assert!(
        inspect_oci(tampered.to_str().unwrap(), "x86_64", FIXTURE_REVISION).is_err(),
        "oci-tampered"
    );
}

#[test]
fn oracle_go_fixture_content() {
    let file = data_path("go-fixture.oci");
    let (image, content) = inspect_oci_content(
        file.to_str().unwrap(),
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
    assert!(
        inspect_oci_content(
            file.to_str().unwrap(),
            "x86_64",
            FIXTURE_REVISION,
            &[String::from("/missing")]
        )
        .is_err(),
        "oci-content-missing"
    );
}

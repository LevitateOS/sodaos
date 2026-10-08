//! Focused checks for current release-build producer and consumer behavior.
//!
//! Some expectations and immutable OCI inputs were captured from an earlier
//! Go implementation. Their provenance does not define a general byte-parity
//! requirement; each assertion covers a current contract used by these tests.

mod oracle_vectors;

use oracle_vectors as oracle;
use soda_build_tools::reader::stream::{CoreOSImage, LiveInputs, ResolvedCoreOS, TailnetInputs};
use std::collections::BTreeMap;
use std::path::PathBuf;

#[path = "oracle/inputs.rs"]
mod inputs;
#[path = "oracle/oci.rs"]
mod oci;
#[path = "oracle/production.rs"]
mod production;

const FIXTURE_REVISION: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";

fn data_path(name: &str) -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("tests/data")
        .join(name)
}

fn scratch(tag: &str) -> PathBuf {
    let dir = std::env::temp_dir().join(format!(
        "soda-oracle-{}-{}-{}",
        tag,
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos()
    ));
    std::fs::create_dir_all(&dir).unwrap();
    dir
}

fn oracle_live_inputs() -> LiveInputs {
    let img = CoreOSImage {
        url: "https://builds.test/fedora-coreos-44.20260901.1.0-live.x86_64.iso".to_string(),
        signature_url: "https://builds.test/fedora-coreos-44.20260901.1.0-live.x86_64.iso.sig"
            .to_string(),
        sha256: "a".repeat(64),
        uncompressed_sha256: "b".repeat(64),
    };
    let mut container = BTreeMap::new();
    container.insert(
        "x86_64".to_string(),
        format!("quay.io/fedora/fedora-coreos@sha256:{}", "c".repeat(64)),
    );
    let mut iso = BTreeMap::new();
    iso.insert("x86_64".to_string(), img.clone());
    let mut qemu = BTreeMap::new();
    qemu.insert("x86_64".to_string(), img);
    LiveInputs {
        coreos: ResolvedCoreOS {
            release: "44.20260901.1.0".to_string(),
            metadata_url:
                "https://builds.test/prod/streams/stable/builds/44.20260901.1.0/release.json"
                    .to_string(),
            container,
            iso,
            qemu,
        },
        tailnet: TailnetInputs {
            version: "1.98.2".to_string(),
            sha256: "e".repeat(64),
            base: "docker.io/tailscale/alpine-base:3.22".to_string(),
        },
    }
}

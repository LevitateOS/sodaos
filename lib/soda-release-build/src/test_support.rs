//! Shared cfg(test) fixtures with production-test consumers in more
//! than one module. Each fixture keeps its current bytes and owner.

use crate::sha256_hex;
use soda_build_tools::reader::stream::{CoreOSImage, LiveInputs, ResolvedCoreOS, TailnetInputs};
use std::collections::BTreeMap;

pub const FIXTURE_REVISION: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";

const CONFIG_MEDIA_TYPE: &str = "application/vnd.oci.image.config.v1+json";
const LAYER_TAR: &str = "application/vnd.oci.image.layer.v1.tar";
const MANIFEST_MEDIA_TYPE: &str = "application/vnd.oci.image.manifest.v1+json";

/// Builds the Go `fixtureOCI` archive: one tar layer with fixture.txt,
/// config with soda attribution, manifest, index.
pub fn fixture_oci_bytes(arch: &str) -> Vec<u8> {
    let body = b"synthetic layer fixture; never executed";
    let mut layer = Vec::new();
    {
        let mut writer = tar::Builder::new(&mut layer);
        let mut header = tar::Header::new_ustar();
        header.set_size(body.len() as u64);
        header.set_mode(0o644);
        header.set_cksum();
        writer
            .append_data(&mut header, "fixture.txt", &body[..])
            .unwrap();
        writer.into_inner().unwrap();
    }
    let layer_sum = sha256_hex(&layer);
    let layer_digest = format!("sha256:{layer_sum}");
    let config = format!(
        "{{\"os\":\"linux\",\"architecture\":\"{arch}\",\"rootfs\":{{\"type\":\"layers\",\"diff_ids\":[\"{layer_digest}\"]}},\"config\":{{\"Labels\":{{\"org.opencontainers.image.revision\":\"{FIXTURE_REVISION}\",\"org.opencontainers.image.source\":\"https://github.com/LevitateOS/sodaos\",\"org.opencontainers.image.base.name\":\"synthetic-base\",\"org.opencontainers.image.base.digest\":\"sha256:{}\"}}}}}}",
        "b".repeat(64)
    );
    let config_sum = sha256_hex(config.as_bytes());
    let manifest = format!(
        "{{\"schemaVersion\":2,\"config\":{{\"digest\":\"sha256:{config_sum}\",\"size\":{},\"mediaType\":\"{CONFIG_MEDIA_TYPE}\"}},\"layers\":[{{\"digest\":\"{layer_digest}\",\"size\":{},\"mediaType\":\"{LAYER_TAR}\"}}]}}",
        config.len(),
        layer.len()
    );
    let manifest_sum = sha256_hex(manifest.as_bytes());
    let index = format!(
        "{{\"schemaVersion\":2,\"manifests\":[{{\"digest\":\"sha256:{manifest_sum}\",\"size\":{},\"mediaType\":\"{MANIFEST_MEDIA_TYPE}\"}}]}}",
        manifest.len()
    );
    let blobs: Vec<(String, Vec<u8>)> = vec![
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
    ];
    let mut archive = Vec::new();
    {
        let mut writer = tar::Builder::new(&mut archive);
        for (name, data) in &blobs {
            let mut header = tar::Header::new_ustar();
            header.set_size(data.len() as u64);
            header.set_mode(0o644);
            header.set_cksum();
            writer
                .append_data(&mut header, name.as_str(), &data[..])
                .unwrap();
        }
        writer.into_inner().unwrap();
    }
    archive
}

pub fn fixture_live_inputs() -> LiveInputs {
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

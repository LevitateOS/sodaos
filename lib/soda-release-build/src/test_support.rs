//! Shared cfg(test) fixtures with production-test consumers in more
//! than one module. Each fixture keeps its current bytes and owner.

use crate::oci::{CONFIG_MEDIA_TYPE, LAYER_TAR, MANIFEST_MEDIA_TYPE};
use crate::sha256_hex;

pub const FIXTURE_REVISION: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";

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

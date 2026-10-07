use super::*;
use std::collections::BTreeMap;
use std::sync::atomic::{AtomicU64, Ordering};

use crate::buildx;
use serde_json::Value;

static COUNTER: AtomicU64 = AtomicU64::new(0);

pub fn temp_dir(prefix: &str) -> String {
    let parent = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../.artifacts/l03-l04/tmp/installer");
    std::fs::create_dir_all(&parent).unwrap();
    loop {
        let id = COUNTER.fetch_add(1, Ordering::SeqCst);
        let dir = parent.join(format!("{prefix}-{}-{id}", std::process::id()));
        match std::fs::create_dir(&dir) {
            Ok(()) => return dir.to_str().unwrap().to_string(),
            Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => continue,
            Err(error) => panic!("cannot create test directory: {error}"),
        }
    }
}

fn write_blob(layout: &str, data: &[u8]) -> String {
    let sum = buildx::sha256_hex(data);
    let path = format!("{layout}/blobs/sha256/{sum}");
    std::fs::create_dir_all(format!("{layout}/blobs/sha256")).unwrap();
    if let Ok(previous) = std::fs::read(&path) {
        assert_eq!(previous, data, "conflicting blob {sum}");
    } else {
        std::fs::write(&path, data).unwrap();
    }
    format!("sha256:{sum}")
}

fn desc(digest: &str, size: usize, media: &str) -> Value {
    serde_json::json!({"mediaType":media,"digest":digest,"size":size})
}

/// Mirror of `testoci.Archive` + `Add`, written directly as a layout:
/// one shared layer, a per-name config, manifest, and index entry.
/// Returns the config digest reference.
pub fn add_image(layout: &str, name: &str, arch: &str, revision: &str) -> String {
    let layer = vec![0u8; 1024];
    let layer_digest = write_blob(layout, &layer);
    let ld = desc(&layer_digest, layer.len(), LAYER_TAR);
    let config_value = serde_json::json!({"architecture":arch,"config":{"Labels":{
        "org.opencontainers.image.revision":revision,
        "org.opencontainers.image.source":"https://github.com/LevitateOS/sodaos",
        "org.opencontainers.image.base.name":"synthetic-base",
        "org.opencontainers.image.base.digest":format!("sha256:{}", "b".repeat(64)),
        "io.soda.fixture":name
    }},"os":"linux","rootfs":{"type":"layers","diff_ids":[layer_digest]}});
    let config = serde_json::to_vec(&config_value).unwrap();
    let config_digest = write_blob(layout, &config);
    let cd = desc(&config_digest, config.len(), CONFIG_MEDIA_TYPE);
    let manifest_value = serde_json::json!({"schemaVersion":2,"mediaType":MANIFEST_MEDIA_TYPE,"config":cd,"layers":[ld]});
    let manifest = serde_json::to_vec(&manifest_value).unwrap();
    let manifest_digest = write_blob(layout, &manifest);
    let mut md = desc(&manifest_digest, manifest.len(), MANIFEST_MEDIA_TYPE);
    md.as_object_mut().unwrap().insert(
        "annotations".to_string(),
        serde_json::json!({"org.opencontainers.image.ref.name":config_digest}),
    );
    let index_path = format!("{layout}/index.json");
    let mut manifests: Vec<Value> = Vec::new();
    if let Ok(previous) = std::fs::read(&index_path) {
        let value: Value = serde_json::from_slice(&previous).unwrap();
        if let Some(items) = value.get("manifests").and_then(Value::as_array) {
            manifests.extend(items.iter().cloned());
        }
    } else {
        std::fs::write(
            format!("{layout}/oci-layout"),
            br#"{"imageLayoutVersion":"1.0.0"}"#,
        )
        .unwrap();
    }
    manifests.push(md);
    let index = serde_json::to_vec(
        &serde_json::json!({"schemaVersion":2,"mediaType":INDEX_MEDIA_TYPE,"manifests":manifests}),
    )
    .unwrap();
    std::fs::write(&index_path, &index).unwrap();
    config_digest
}

pub fn layout_fixture() -> (String, BTreeMap<String, String>) {
    let root = temp_dir("soda-oci-layout");
    let layout = format!("{root}/layout");
    std::fs::create_dir_all(&layout).unwrap();
    let mut revisions = BTreeMap::new();
    for name in ["first", "second"] {
        let reference = add_image(&layout, name, "amd64", &"a".repeat(40));
        revisions.insert(reference, "a".repeat(40));
    }
    (layout, revisions)
}

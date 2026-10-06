use super::*;
use std::collections::BTreeMap;
use std::sync::atomic::{AtomicU64, Ordering};

use soda_json::JsonValue;

use crate::buildx;
use crate::jsongo::parse;

static COUNTER: AtomicU64 = AtomicU64::new(0);

pub fn temp_dir(prefix: &str) -> String {
    let id = COUNTER.fetch_add(1, Ordering::SeqCst);
    let dir = std::env::temp_dir().join(format!("{prefix}-{}-{id}", std::process::id()));
    std::fs::create_dir_all(&dir).unwrap();
    dir.to_str().unwrap().to_string()
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

fn desc(digest: &str, size: usize, media: &str) -> JsonValue {
    JsonValue::Object(vec![
        ("mediaType".to_string(), JsonValue::Str(media.to_string())),
        ("digest".to_string(), JsonValue::Str(digest.to_string())),
        ("size".to_string(), JsonValue::Number(size.to_string())),
    ])
}

/// Mirror of `testoci.Archive` + `Add`, written directly as a layout:
/// one shared layer, a per-name config, manifest, and index entry.
/// Returns the config digest reference.
pub fn add_image(layout: &str, name: &str, arch: &str, revision: &str) -> String {
    let layer = vec![0u8; 1024];
    let layer_digest = write_blob(layout, &layer);
    let ld = desc(&layer_digest, layer.len(), LAYER_TAR);
    let config = crate::jsongo::serialize(&JsonValue::Object(vec![
        ("architecture".to_string(), JsonValue::Str(arch.to_string())),
        (
            "config".to_string(),
            JsonValue::Object(vec![(
                "Labels".to_string(),
                JsonValue::Object(vec![
                    (
                        "org.opencontainers.image.revision".to_string(),
                        JsonValue::Str(revision.to_string()),
                    ),
                    (
                        "org.opencontainers.image.source".to_string(),
                        JsonValue::Str("https://github.com/LevitateOS/sodaos".to_string()),
                    ),
                    (
                        "org.opencontainers.image.base.name".to_string(),
                        JsonValue::Str("synthetic-base".to_string()),
                    ),
                    (
                        "org.opencontainers.image.base.digest".to_string(),
                        JsonValue::Str(format!("sha256:{}", "b".repeat(64))),
                    ),
                    (
                        "io.soda.fixture".to_string(),
                        JsonValue::Str(name.to_string()),
                    ),
                ]),
            )]),
        ),
        ("os".to_string(), JsonValue::Str("linux".to_string())),
        (
            "rootfs".to_string(),
            JsonValue::Object(vec![
                ("type".to_string(), JsonValue::Str("layers".to_string())),
                (
                    "diff_ids".to_string(),
                    JsonValue::Array(vec![JsonValue::Str(layer_digest.clone())]),
                ),
            ]),
        ),
    ]));
    let config_digest = write_blob(layout, config.as_bytes());
    let cd = desc(&config_digest, config.len(), CONFIG_MEDIA_TYPE);
    let manifest = crate::jsongo::serialize(&JsonValue::Object(vec![
        (
            "schemaVersion".to_string(),
            JsonValue::Number("2".to_string()),
        ),
        (
            "mediaType".to_string(),
            JsonValue::Str(MANIFEST_MEDIA_TYPE.to_string()),
        ),
        ("config".to_string(), cd),
        ("layers".to_string(), JsonValue::Array(vec![ld])),
    ]));
    let manifest_digest = write_blob(layout, manifest.as_bytes());
    let mut md = desc(&manifest_digest, manifest.len(), MANIFEST_MEDIA_TYPE);
    if let JsonValue::Object(entries) = &mut md {
        entries.push((
            "annotations".to_string(),
            JsonValue::Object(vec![(
                "org.opencontainers.image.ref.name".to_string(),
                JsonValue::Str(config_digest.clone()),
            )]),
        ));
    }
    let index_path = format!("{layout}/index.json");
    let mut manifests: Vec<JsonValue> = Vec::new();
    if let Ok(previous) = std::fs::read(&index_path) {
        let value = parse(&previous).unwrap();
        if let JsonValue::Object(entries) = &value {
            for (key, val) in entries {
                if key == "manifests" {
                    if let JsonValue::Array(items) = val {
                        manifests.extend(items.iter().cloned());
                    }
                }
            }
        }
    } else {
        std::fs::write(
            format!("{layout}/oci-layout"),
            br#"{"imageLayoutVersion":"1.0.0"}"#,
        )
        .unwrap();
    }
    manifests.push(md);
    let index = crate::jsongo::serialize(&JsonValue::Object(vec![
        (
            "schemaVersion".to_string(),
            JsonValue::Number("2".to_string()),
        ),
        (
            "mediaType".to_string(),
            JsonValue::Str(INDEX_MEDIA_TYPE.to_string()),
        ),
        ("manifests".to_string(), JsonValue::Array(manifests)),
    ]));
    std::fs::write(&index_path, index.as_bytes()).unwrap();
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

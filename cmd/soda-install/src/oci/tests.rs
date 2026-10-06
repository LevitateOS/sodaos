use super::test_support::*;
use super::*;
use std::collections::BTreeMap;

use soda_json::JsonValue;

use crate::buildx;
use crate::jsongo::parse;

#[test]
fn layout_preserves_identities_and_counts_once() {
    let (dir, revisions) = layout_fixture();
    let got = inspect_oci_layout(&dir, "x86_64", &revisions).unwrap();
    assert_eq!(got.images.len(), 2);
    // index + layout + 2 configs + 2 manifests + 1 shared layer.
    assert_eq!(got.files.len(), 7);
    let mut total: u64 = 0;
    for (name, hash) in &got.files {
        let data = std::fs::read(format!("{dir}/{name}")).unwrap();
        assert_eq!(&buildx::sha256_hex(&data), hash);
        total += data.len() as u64;
    }
    assert_eq!(total, got.bytes);
    for (reference, image) in &got.images {
        assert_eq!(reference, &image.config);
        assert_eq!(image.architecture, "amd64");
        assert_eq!(&revisions[reference], &image.revision);
    }
    assert_eq!(
        inspect_oci_layout(&dir, "aarch64", &revisions)
            .unwrap_err()
            .to_string(),
        "expected x86_64"
    );
    let wrong: BTreeMap<String, String> = revisions
        .keys()
        .map(|k| (k.clone(), "b".repeat(40)))
        .collect();
    assert!(inspect_oci_layout(&dir, "x86_64", &wrong)
        .unwrap_err()
        .to_string()
        .contains("revision mismatch"));
}

#[test]
fn layout_refuses_substitution() {
    for kind in [
        "missing",
        "corrupt",
        "symlink-file",
        "symlink-dir",
        "symlink-root",
        "duplicate-ref",
        "wrong-ref",
        "wrong-size",
        "external-url",
        "empty-index",
        "nested-index",
    ] {
        let (mut dir, revisions) = layout_fixture();
        let before = inspect_oci_layout(&dir, "x86_64", &revisions).unwrap();
        let blob = before
            .files
            .keys()
            .find(|n| n.starts_with("blobs/"))
            .unwrap()
            .clone();
        match kind {
            "missing" => std::fs::remove_file(format!("{dir}/{blob}")).unwrap(),
            "corrupt" => std::fs::write(format!("{dir}/{blob}"), b"corrupted").unwrap(),
            "symlink-file" => {
                let outside = format!("{dir}-outside-blob");
                std::fs::rename(format!("{dir}/{blob}"), &outside).unwrap();
                std::os::unix::fs::symlink(&outside, format!("{dir}/{blob}")).unwrap();
            }
            "symlink-dir" => {
                let outside = format!("{dir}-outside");
                std::fs::rename(format!("{dir}/blobs"), &outside).unwrap();
                std::os::unix::fs::symlink(&outside, format!("{dir}/blobs")).unwrap();
            }
            "symlink-root" => {
                let link = format!("{dir}-link");
                std::os::unix::fs::symlink(&dir, &link).unwrap();
                dir = link;
            }
            _ => {
                let path = format!("{dir}/index.json");
                let data = std::fs::read(&path).unwrap();
                let value = parse(&data).unwrap();
                let mut entries = match value {
                    JsonValue::Object(entries) => entries,
                    _ => panic!("index shape"),
                };
                let manifests = entries.iter_mut().find(|(k, _)| k == "manifests").unwrap();
                let items = match &mut manifests.1 {
                    JsonValue::Array(items) => items,
                    _ => panic!("manifests shape"),
                };
                let first = items[0].clone();
                match kind {
                    "duplicate-ref" => items[1] = first,
                    "wrong-ref" => {
                        if let JsonValue::Object(fields) = &mut items[0] {
                            fields.retain(|(k, _)| k != "annotations");
                            fields.push((
                                "annotations".to_string(),
                                JsonValue::Object(vec![(
                                    "org.opencontainers.image.ref.name".to_string(),
                                    JsonValue::Str("latest".to_string()),
                                )]),
                            ));
                        }
                    }
                    "wrong-size" => {
                        if let JsonValue::Object(fields) = &mut items[0] {
                            for (k, v) in fields.iter_mut() {
                                if k == "size" {
                                    *v = JsonValue::Number("1".to_string());
                                }
                            }
                        }
                    }
                    "external-url" => {
                        if let JsonValue::Object(fields) = &mut items[0] {
                            fields.push((
                                "urls".to_string(),
                                JsonValue::Array(vec![JsonValue::Str(
                                    "https://example.invalid/layer".to_string(),
                                )]),
                            ));
                        }
                    }
                    "empty-index" => *items = Vec::new(),
                    "nested-index" => {
                        if let JsonValue::Object(fields) = &mut items[0] {
                            for (k, v) in fields.iter_mut() {
                                if k == "mediaType" {
                                    *v = JsonValue::Str(INDEX_MEDIA_TYPE.to_string());
                                }
                            }
                        }
                    }
                    _ => unreachable!(),
                }
                std::fs::write(
                    &path,
                    crate::jsongo::serialize(&JsonValue::Object(entries)).as_bytes(),
                )
                .unwrap();
            }
        }
        assert!(
            inspect_oci_layout(&dir, "x86_64", &revisions).is_err(),
            "{kind} accepted"
        );
    }
}

use super::test_support::*;
use super::*;
use std::collections::BTreeMap;

use crate::buildx;

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
                let mut value: serde_json::Value = serde_json::from_slice(&data).unwrap();
                let entries = value.as_object_mut().expect("index shape");
                let items = entries
                    .get_mut("manifests")
                    .and_then(serde_json::Value::as_array_mut)
                    .expect("manifests shape");
                let first = items[0].clone();
                match kind {
                    "duplicate-ref" => items[1] = first,
                    "wrong-ref" => {
                        let fields = items[0].as_object_mut().unwrap();
                        fields.insert(
                            "annotations".to_string(),
                            serde_json::json!({"org.opencontainers.image.ref.name":"latest"}),
                        );
                    }
                    "wrong-size" => {
                        items[0]
                            .as_object_mut()
                            .unwrap()
                            .insert("size".to_string(), serde_json::json!(1));
                    }
                    "external-url" => {
                        items[0].as_object_mut().unwrap().insert(
                            "urls".to_string(),
                            serde_json::json!(["https://example.invalid/layer"]),
                        );
                    }
                    "empty-index" => items.clear(),
                    "nested-index" => {
                        items[0]
                            .as_object_mut()
                            .unwrap()
                            .insert("mediaType".to_string(), serde_json::json!(INDEX_MEDIA_TYPE));
                    }
                    _ => unreachable!(),
                }
                std::fs::write(&path, serde_json::to_vec(&value).unwrap().as_slice()).unwrap();
            }
        }
        assert!(
            inspect_oci_layout(&dir, "x86_64", &revisions).is_err(),
            "{kind} accepted"
        );
    }
}

#[test]
fn layout_retains_original_hash_with_unknown_number_extension() {
    let (dir, revisions) = layout_fixture();
    let path = format!("{dir}/index.json");
    let mut data = std::fs::read(&path).unwrap();
    assert_eq!(data.pop(), Some(b'}'));
    data.extend_from_slice(br#", "x-extension":{"number":1e400}}"#);
    std::fs::write(&path, &data).unwrap();

    let layout = inspect_oci_layout(&dir, "x86_64", &revisions).unwrap();
    assert_eq!(layout.files["index.json"], buildx::sha256_hex(&data));
}

#[test]
fn layout_rejects_invalid_utf8_and_trailing_json() {
    for trailing in [false, true] {
        let (dir, revisions) = layout_fixture();
        let path = format!("{dir}/index.json");
        let mut data = std::fs::read(&path).unwrap();
        if trailing {
            data.extend_from_slice(b" {} ");
        } else {
            assert_eq!(data.pop(), Some(b'}'));
            data.extend_from_slice(b",\"x-extension\":\"\xff\"}");
        }
        std::fs::write(&path, data).unwrap();
        assert!(
            inspect_oci_layout(&dir, "x86_64", &revisions).is_err(),
            "trailing={trailing}"
        );
    }
}

#[test]
fn manifest_requires_fields_and_rejects_explicit_null_defaults() {
    let config = format!(
        r#"{{"mediaType":"{CONFIG_MEDIA_TYPE}","digest":"sha256:{}","size":1}}"#,
        "a".repeat(64)
    );
    let valid = format!(
        r#"{{"schemaVersion":2,"config":{config},"layers":[],"extension":{{"number":1e400}}}}"#
    );
    assert!(super::metadata::parse_oci_manifest(valid.as_bytes()).is_ok());

    for invalid in [
        format!(r#"{{"schemaVersion":2,"layers":[]}}"#),
        format!(r#"{{"schemaVersion":2,"mediaType":null,"config":{config},"layers":[]}}"#),
        format!(
            r#"{{"schemaVersion":2,"config":{{"mediaType":"{CONFIG_MEDIA_TYPE}","digest":"sha256:{}","size":1,"urls":null}},"layers":[]}}"#,
            "a".repeat(64)
        ),
    ] {
        assert!(
            super::metadata::parse_oci_manifest(invalid.as_bytes()).is_err(),
            "accepted {invalid}"
        );
    }
}

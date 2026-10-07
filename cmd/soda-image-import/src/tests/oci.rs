use super::super::*;
use super::fixtures::*;

#[test]
fn layout_preserves_identities_and_counts() {
    let root = test_root("layout-ok");
    let revision = repeat('a', 40);
    let (layout, digests) = build_layout(
        &root,
        &[("first", revision.clone()), ("second", revision.clone())],
    );
    let revisions: HashMap<String, String> = digests
        .values()
        .map(|image| (image.config.clone(), revision.clone()))
        .collect();
    let got = inspect_oci_layout(&layout, "x86_64", &revisions).expect("inspect");
    assert_eq!(got.len(), 2);
    for (config, image) in &got {
        assert_eq!(&image.config, config);
        assert_eq!(image.architecture, "amd64");
        assert_eq!(image.revision, revision);
    }
    assert_eq!(
        inspect_oci_layout(&layout, "aarch64", &revisions).unwrap_err(),
        "expected x86_64"
    );
    let wrong: HashMap<String, String> = digests
        .values()
        .map(|image| (image.config.clone(), repeat('b', 40)))
        .collect();
    assert!(inspect_oci_layout(&layout, "x86_64", &wrong)
        .unwrap_err()
        .contains("revision mismatch"));
}

fn mutate_layout(root: &Path, kind: &str) -> (PathBuf, HashMap<String, String>) {
    let revision = repeat('a', 40);
    let (layout, digests) = build_layout(
        root,
        &[("first", revision.clone()), ("second", revision.clone())],
    );
    let revisions: HashMap<String, String> = digests
        .values()
        .map(|image| (image.config.clone(), revision.clone()))
        .collect();
    assert!(inspect_oci_layout(&layout, "x86_64", &revisions).is_ok());
    let mut blobs = fs::read_dir(layout.join("blobs").join("sha256")).expect("blobs");
    let blob = blobs.next().expect("blob entry").expect("entry").path();
    match kind {
        "missing" => fs::remove_file(&blob).expect("remove"),
        "corrupt" => fs::write(&blob, b"corrupted").expect("corrupt"),
        "symlink-file" => {
            let outside = root.join("outside-blob");
            fs::rename(&blob, &outside).expect("move");
            std::os::unix::fs::symlink(&outside, &blob).expect("link file");
        }
        "symlink-dir" => {
            let moved = root.join("outside");
            fs::rename(layout.join("blobs"), &moved).expect("move dir");
            std::os::unix::fs::symlink(&moved, layout.join("blobs")).expect("link dir");
        }
        "symlink-root" => {}
        _ => {
            let path = layout.join("index.json");
            let raw = fs::read_to_string(&path).expect("index");
            let mut value: serde_json::Value = serde_json::from_str(&raw).expect("index json");
            let manifests = match &mut value {
                serde_json::Value::Object(entries) => {
                    entries.get_mut("manifests").expect("manifests")
                }
                _ => panic!("index object"),
            };
            let items = match manifests {
                serde_json::Value::Array(items) => items,
                _ => panic!("manifests array"),
            };
            match kind {
                "duplicate-ref" => {
                    items[1] = items[0].clone();
                }
                "wrong-ref" => {
                    set_annotation(&mut items[0], "latest");
                }
                "wrong-size" => {
                    set_size(&mut items[0], 1);
                }
                "external-url" => {
                    set_urls(&mut items[0]);
                }
                "empty-index" => {
                    *items = Vec::new();
                }
                "nested-index" => {
                    set_media(&mut items[0], "application/vnd.oci.image.index.v1+json");
                }
                _ => panic!("unknown kind"),
            }
            fs::write(&path, serde_json::to_vec(&value).expect("serialize index"))
                .expect("rewrite index");
        }
    }
    if kind == "symlink-root" {
        let link = root.join("layout-link");
        std::os::unix::fs::symlink(&layout, &link).expect("link root");
        return (link, revisions);
    }
    (layout, revisions)
}

fn set_field(item: &mut serde_json::Value, key: &str, value: serde_json::Value) {
    item.as_object_mut()
        .expect("descriptor object")
        .insert(key.to_string(), value);
}

fn set_annotation(item: &mut serde_json::Value, reference: &str) {
    set_field(
        item,
        "annotations",
        serde_json::json!({"org.opencontainers.image.ref.name": reference}),
    );
}

fn set_size(item: &mut serde_json::Value, size: i64) {
    set_field(item, "size", serde_json::json!(size));
}

fn set_urls(item: &mut serde_json::Value) {
    set_field(
        item,
        "urls",
        serde_json::json!(["https://example.invalid/layer"]),
    );
}

fn set_media(item: &mut serde_json::Value, media: &str) {
    set_field(item, "mediaType", serde_json::json!(media));
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
        let root = test_root(&format!("layout-{kind}"));
        let (layout, revisions) = mutate_layout(&root, kind);
        assert!(
            inspect_oci_layout(&layout, "x86_64", &revisions).is_err(),
            "{kind} accepted"
        );
    }
}

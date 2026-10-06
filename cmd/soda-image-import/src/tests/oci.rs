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
            let mut value = parse_json(raw.as_bytes()).expect("index json");
            let manifests = match &mut value {
                JsonValue::Object(entries) => entries
                    .iter_mut()
                    .find(|(k, _)| k == "manifests")
                    .map(|(_, v)| v)
                    .expect("manifests"),
                _ => panic!("index object"),
            };
            let items = match manifests {
                JsonValue::Array(items) => items,
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
            fs::write(&path, render_json(&value)).expect("rewrite index");
        }
    }
    if kind == "symlink-root" {
        let link = root.join("layout-link");
        std::os::unix::fs::symlink(&layout, &link).expect("link root");
        return (link, revisions);
    }
    (layout, revisions)
}

fn render_json(value: &JsonValue) -> String {
    match value {
        JsonValue::Null => "null".to_string(),
        JsonValue::Bool(true) => "true".to_string(),
        JsonValue::Bool(false) => "false".to_string(),
        JsonValue::Number(raw) => raw.clone(),
        JsonValue::Str(s) => render_string(s),
        JsonValue::Array(items) => {
            let parts: Vec<String> = items.iter().map(render_json).collect();
            format!("[{}]", parts.join(","))
        }
        JsonValue::Object(entries) => {
            let parts: Vec<String> = entries
                .iter()
                .map(|(k, v)| format!("{}:{}", render_string(k), render_json(v)))
                .collect();
            format!("{{{}}}", parts.join(","))
        }
    }
}

fn render_string(s: &str) -> String {
    let mut out = String::from("\"");
    for c in s.chars() {
        match c {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            c if (c as u32) < 0x20 => out.push_str(&format!("\\u{:04x}", c as u32)),
            c => out.push(c),
        }
    }
    out.push('"');
    out
}

fn set_field(item: &mut JsonValue, key: &str, value: JsonValue) {
    let entries = match item {
        JsonValue::Object(entries) => entries,
        _ => panic!("descriptor object"),
    };
    if let Some(slot) = entries.iter_mut().find(|(k, _)| k == key) {
        slot.1 = value;
    } else {
        entries.push((key.to_string(), value));
    }
}

fn set_annotation(item: &mut JsonValue, reference: &str) {
    set_field(
        item,
        "annotations",
        JsonValue::Object(vec![(
            "org.opencontainers.image.ref.name".to_string(),
            JsonValue::Str(reference.to_string()),
        )]),
    );
}

fn set_size(item: &mut JsonValue, size: i64) {
    set_field(item, "size", JsonValue::Number(size.to_string()));
}

fn set_urls(item: &mut JsonValue) {
    set_field(
        item,
        "urls",
        JsonValue::Array(vec![JsonValue::Str(
            "https://example.invalid/layer".to_string(),
        )]),
    );
}

fn set_media(item: &mut JsonValue, media: &str) {
    set_field(item, "mediaType", JsonValue::Str(media.to_string()));
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

use std::sync::atomic::AtomicU64;

use super::super::*;

static TEST_SEQ: AtomicU64 = AtomicU64::new(0);

pub(super) fn test_root(name: &str) -> PathBuf {
    let dir = std::env::temp_dir().join(format!(
        "soda-image-import-test-{}-{}-{name}",
        std::process::id(),
        TEST_SEQ.fetch_add(1, Ordering::SeqCst)
    ));
    let _ = fs::remove_dir_all(&dir);
    fs::create_dir_all(&dir).expect("test root");
    dir
}

pub(super) fn repeat(ch: char, n: usize) -> String {
    std::iter::repeat_n(ch, n).collect()
}

pub(super) fn test_ctx(cancelled: &AtomicBool) -> ImportCtx<'_> {
    ImportCtx {
        deadline: Instant::now() + Duration::from_secs(600),
        cancelled,
    }
}

// ----- OCI layout fixtures (inert; layer bytes are never executed) -----

pub(super) struct LayoutImage {
    pub(super) config: String,
    manifest: String,
}

fn write_blob(layout: &Path, bytes: &[u8]) -> (String, usize) {
    let sum = sha256_hex(bytes);
    let path = layout.join("blobs").join("sha256").join(&sum);
    fs::create_dir_all(path.parent().unwrap()).expect("blob dir");
    fs::write(&path, bytes).expect("blob write");
    (format!("sha256:{sum}"), bytes.len())
}

/// Build a shared layout directory plus per-tag identities, mirroring
/// what `testoci.Add` produces for the Go tests.
pub(super) fn build_layout(
    root: &Path,
    tags: &[(&str, String)],
) -> (PathBuf, HashMap<String, LayoutImage>) {
    let layout = root.join("layout");
    fs::create_dir_all(layout.join("blobs").join("sha256")).expect("layout dirs");
    let mut descriptors = Vec::new();
    let mut images = HashMap::new();
    for (tag, revision) in tags {
        let layer_bytes = format!("inert layer {tag}\n").repeat(64).into_bytes();
        let (layer_digest, layer_size) = write_blob(&layout, &layer_bytes);
        let config_json = [
            "{\"os\":\"linux\",\"architecture\":\"amd64\",\"rootfs\":{\"type\":\"layers\",\"diff_ids\":[\"",
            &layer_digest,
            "\"]},\"config\":{\"Labels\":{\"org.opencontainers.image.revision\":\"",
            revision,
            "\",\"org.opencontainers.image.source\":\"https://github.com/LevitateOS/sodaos\",\"org.opencontainers.image.base.name\":\"synthetic-base\",\"org.opencontainers.image.base.digest\":\"sha256:",
            &repeat('b', 64),
            "\",\"io.soda.fixture\":\"",
            tag,
            "\"}}}",
        ]
        .concat();
        let (config_digest, config_size) = write_blob(&layout, config_json.as_bytes());
        let manifest_json = [
            "{\"schemaVersion\":2,\"mediaType\":\"application/vnd.oci.image.manifest.v1+json\",\"config\":{\"mediaType\":\"application/vnd.oci.image.config.v1+json\",\"digest\":\"",
            &config_digest,
            "\",\"size\":",
            &config_size.to_string(),
            "},\"layers\":[{\"mediaType\":\"application/vnd.oci.image.layer.v1.tar\",\"digest\":\"",
            &layer_digest,
            "\",\"size\":",
            &layer_size.to_string(),
            "}]}",
        ]
        .concat();
        let (manifest_digest, manifest_size) = write_blob(&layout, manifest_json.as_bytes());
        descriptors.push(
            [
                "{\"mediaType\":\"application/vnd.oci.image.manifest.v1+json\",\"digest\":\"",
                &manifest_digest,
                "\",\"size\":",
                &manifest_size.to_string(),
                ",\"annotations\":{\"org.opencontainers.image.ref.name\":\"",
                &config_digest,
                "\"}}",
            ]
            .concat(),
        );
        images.insert(
            tag.to_string(),
            LayoutImage {
                config: config_digest,
                manifest: manifest_digest,
            },
        );
    }
    let index = [
        "{\"schemaVersion\":2,\"mediaType\":\"application/vnd.oci.image.index.v1+json\",\"manifests\":[",
        &descriptors.join(","),
        "]}",
    ]
    .concat();
    fs::write(layout.join("index.json"), index).expect("index write");
    fs::write(
        layout.join("oci-layout"),
        "{\"imageLayoutVersion\":\"1.0.0\"}",
    )
    .expect("layout marker");
    (layout, images)
}

pub(super) fn full_fixture(root: &Path) -> (Payload, PathBuf) {
    let revision = repeat('a', 40);
    let tags: Vec<(&str, String)> = NAMES
        .iter()
        .map(|name| {
            let rev = if *name == "proxy" {
                String::new()
            } else {
                revision.clone()
            };
            (*name, rev)
        })
        .collect();
    let (layout, digests) = build_layout(root, &tags);
    (test_payload(&digests), layout)
}

fn test_payload(digests: &HashMap<String, LayoutImage>) -> Payload {
    let revision = repeat('a', 40);
    let coreos = "44.20260817.3.2".to_string();
    let mut payload = Payload {
        format: 3,
        id: format!("{coreos}.soda-{}", &revision[..12]),
        revision,
        architecture: "x86_64".to_string(),
        coreos,
        base: format!("quay.io/fedora/fedora-coreos@sha256:{}", repeat('b', 64)),
        repository_prefix: "ghcr.io/example/sodaos".to_string(),
        schema: 10,
        presentation_sha256: repeat('c', 64),
        host_packages_sha256: repeat('d', 64),
        images: HashMap::new(),
        upgrade_from: Vec::new(),
    };
    for name in NAMES {
        let layout = digests.get(name).expect("fixture image");
        payload.images.insert(
            name.to_string(),
            ImageBinding {
                reference: format!("{}-{name}@{}", payload.repository_prefix, layout.manifest),
                config: layout.config.clone(),
                manifest: layout.manifest.clone(),
                archive_sha256: repeat('1', 64),
            },
        );
    }
    payload
}

/// Payload JSON with fixed-shape bindings (for decode rules, not layout).
pub(super) fn payload_json_for(entries: &HashMap<String, (String, String)>) -> String {
    let mut parts = Vec::new();
    for name in NAMES {
        let (config, manifest) = entries.get(name).cloned().unwrap_or_else(|| {
            // Distinct per-image identities; extension and forgejo must differ.
            let idx = NAMES.iter().position(|n| *n == name).unwrap_or(0);
            (
                format!("sha256:{}{idx:x}", repeat('e', 63)),
                format!("sha256:{}{idx:x}", repeat('d', 63)),
            )
        });
        parts.push(
            [
                "\"",
                name,
                "\":{\"Reference\":\"ghcr.io/example/sodaos-",
                name,
                "@",
                &manifest,
                "\",\"Config\":\"",
                &config,
                "\",\"Manifest\":\"",
                &manifest,
                "\",\"ArchiveSHA256\":\"",
                &repeat('1', 64),
                "\"}",
            ]
            .concat(),
        );
    }
    let revision = repeat('a', 40);
    [
        "{\"Format\":3,\"ID\":\"44.20260817.3.2.soda-",
        &revision[..12],
        "\",\"Revision\":\"",
        &revision,
        "\",\"Architecture\":\"x86_64\",\"CoreOS\":\"44.20260817.3.2\",\"Base\":\"quay.io/fedora/fedora-coreos@sha256:",
        &repeat('b', 64),
        "\",\"RepositoryPrefix\":\"ghcr.io/example/sodaos\",\"Schema\":10,\"PresentationSHA256\":\"",
        &repeat('c', 64),
        "\",\"HostPackagesSHA256\":\"",
        &repeat('d', 64),
        "\",\"Images\":{",
        &parts.join(","),
        "},\"UpgradeFrom\":[]}",
    ]
    .concat()
}

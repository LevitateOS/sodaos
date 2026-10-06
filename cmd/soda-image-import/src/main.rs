//! soda-image-import is the image-based appliance's fixed native import phase.
//!
//! Rust port of `cmd/soda-image-import` plus the release sliver it runs:
//! payload load/validate (`internal/release/deliver` payload + content +
//! import) over the shared OCI layout verifier (`internal/release/build`
//! OCI layout inspection). CLI surface, exit codes, stderr text, podman
//! argv, and verification rules match the Go implementation.
//!
//! Std + `soda-json` only (`cargo build --offline`); no new crates-io
//! dependencies.

use std::collections::HashMap;
use std::fs;
use std::io::Read;
use std::os::unix::fs::MetadataExt;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicBool, Ordering};
use std::time::{Duration, Instant};

use soda_json::JsonValue;

mod context;
mod json_binding;
mod oci;
mod payload;
mod platform;
mod sha256;

use context::{admit, run, ImportCtx};
use json_binding::{
    json_valid, obj_fields, parse_json, t_field, t_int, t_string, t_string_list, t_string_map,
    Binder,
};
use oci::{
    parse_oci_config, parse_oci_manifest, read_oci_index, OciBlobData, OciDescriptor, OciImage,
};
use payload::{decode_payload, ImageBinding, Payload};
use platform::{
    is_coreos_version, is_digest, is_prefixed_digest, is_revision, oci_architecture,
    require_native, valid_repository_prefix,
};
use sha256::{hex_lower, sha256_hex, Sha256};

#[link(name = "c")]
extern "C" {
    fn geteuid() -> u32;
    fn signal(signum: i32, handler: extern "C" fn(i32)) -> extern "C" fn(i32);
}

const RELEASE_PATH: &str = "/usr/share/soda/release.json";
const IMAGES_PATH: &str = "/usr/share/soda/images";
const PODMAN: &str = "/usr/bin/podman";
/// Go: `context.WithTimeout(ctx, 10*time.Minute)`; the unit allows 11.
const IMPORT_TIMEOUT: Duration = Duration::from_secs(10 * 60);
const SIGINT: i32 = 2;
const SIGTERM: i32 = 15;
const NAMES: [&str; 6] = [
    "dashboard",
    "forgejo",
    "extension",
    "proxy",
    "project-os",
    "tailnet",
];

fn main() {
    std::process::exit(run());
}

// ---------- shared layout loading (build/oci_layout.go) ----------

struct LayoutLoader {
    dir: PathBuf,
    entries: HashMap<String, OciBlobData>,
    json_bytes: usize,
}

impl LayoutLoader {
    fn load(&mut self, name: &str) -> Result<(), String> {
        if self.entries.contains_key(name) {
            return Ok(());
        }
        let (size, data) = read_layout_blob(&self.dir, name)?;
        self.json_bytes += data.as_ref().map(|d| d.len()).unwrap_or(0);
        if self.json_bytes > 32 << 20 {
            return Err("OCI JSON metadata limit exceeded".to_string());
        }
        self.entries
            .insert(name.to_string(), OciBlobData { size, data });
        Ok(())
    }

    fn fetch(&mut self, desc: &OciDescriptor) -> Result<OciBlobData, String> {
        if desc.size < 0 || !desc.urls.is_empty() {
            return Err("local bounded OCI descriptor required".to_string());
        }
        let hex = match desc.digest.strip_prefix("sha256:") {
            Some(hex) if is_digest(hex) => hex,
            _ => return Err("invalid OCI digest".to_string()),
        };
        let name = format!("blobs/sha256/{hex}");
        self.load(&name)?;
        match self.entries.get(&name) {
            Some(blob) if blob.size == desc.size => Ok(blob.clone()),
            _ => Err("missing or wrong-size OCI blob".to_string()),
        }
    }
}

fn open_layout_root(dir: &Path) -> Result<(), String> {
    match fs::symlink_metadata(dir) {
        Ok(st) if st.file_type().is_dir() => Ok(()),
        _ => Err("real OCI layout directory required".to_string()),
    }
}

/// Confined layout read: no absolute/parent segments, no symlink in any
/// prefix (mirroring `os.Root` confinement), regular final verified
/// unchanged across open, hashed while streaming.
fn read_layout_blob(dir: &Path, name: &str) -> Result<(i64, Option<Vec<u8>>), String> {
    if name.is_empty() || name.starts_with('/') {
        return Err("invalid OCI layout entry".to_string());
    }
    let parts: Vec<&str> = name.split('/').collect();
    let mut current = dir.to_path_buf();
    for (i, part) in parts.iter().enumerate() {
        if part.is_empty() || *part == "." || *part == ".." {
            return Err("invalid OCI layout entry".to_string());
        }
        current.push(part);
        let before = fs::symlink_metadata(&current)
            .map_err(|e| format!("cannot stat OCI layout entry {name}: {e}"))?;
        if i + 1 == parts.len() {
            if !before.file_type().is_file() {
                return Err("non-regular OCI layout entry".to_string());
            }
            return read_blob_bytes(&current, &before, name);
        }
        if !before.file_type().is_dir() {
            return Err(format!(
                "cannot stat OCI layout entry {name}: not a directory"
            ));
        }
    }
    Err("invalid OCI layout entry".to_string())
}

fn read_blob_bytes(
    path: &Path,
    before: &fs::Metadata,
    name: &str,
) -> Result<(i64, Option<Vec<u8>>), String> {
    let length = before.len();
    if length > i64::MAX as u64 {
        return Err("invalid OCI blob size".to_string());
    }
    let file =
        fs::File::open(path).map_err(|e| format!("cannot open OCI layout entry {name}: {e}"))?;
    let after = file
        .metadata()
        .map_err(|e| format!("cannot stat OCI layout entry {name}: {e}"))?;
    if !after.is_file() || after.dev() != before.dev() || after.ino() != before.ino() {
        return Err("OCI layout entry changed".to_string());
    }
    let mut hasher = Sha256::new();
    let buffered = length <= 4 << 20;
    let mut data = Vec::new();
    let mut size: u64 = 0;
    let mut reader = file.take(length + 1);
    let mut chunk = [0u8; 65536];
    loop {
        let n = reader
            .read(&mut chunk)
            .map_err(|e| format!("cannot read OCI layout entry {name}: {e}"))?;
        if n == 0 {
            break;
        }
        size += n as u64;
        hasher.update(&chunk[..n]);
        if buffered {
            data.extend_from_slice(&chunk[..n]);
        }
    }
    if size != length {
        return Err("OCI blob size changed".to_string());
    }
    let sum = hex_lower(&hasher.finish());
    if name.starts_with("blobs/") && name != format!("blobs/sha256/{sum}").as_str() {
        return Err("OCI blob checksum mismatch".to_string());
    }
    let body = if json_valid(&data) { Some(data) } else { None };
    Ok((length as i64, body))
}

fn validate_oci_layers(loader: &mut LayoutLoader, layers: &[OciDescriptor]) -> Result<(), String> {
    for layer in layers {
        match layer.media_type.as_str() {
            "application/vnd.oci.image.layer.v1.tar"
            | "application/vnd.oci.image.layer.v1.tar+gzip"
            | "application/vnd.oci.image.layer.v1.tar+zstd" => {}
            _ => return Err("unsupported OCI layer media type".to_string()),
        }
        loader.fetch(layer)?;
    }
    Ok(())
}

fn validate_oci_rootfs(diff_ids: &[String], layers: &[OciDescriptor]) -> Result<(), String> {
    if diff_ids.len() != layers.len() {
        return Err("OCI rootfs/layer count mismatch".to_string());
    }
    for (id, layer) in diff_ids.iter().zip(layers.iter()) {
        match id.strip_prefix("sha256:") {
            Some(hex) if is_digest(hex) => {}
            _ => return Err("invalid OCI diff ID".to_string()),
        }
        if layer.media_type == "application/vnd.oci.image.layer.v1.tar" && id != &layer.digest {
            return Err("uncompressed OCI layer identity mismatch".to_string());
        }
    }
    Ok(())
}

fn validate_oci_attribution(
    labels: &HashMap<String, String>,
    want_revision: &str,
) -> Result<String, String> {
    let revision = labels
        .get("org.opencontainers.image.revision")
        .cloned()
        .unwrap_or_default();
    if !want_revision.is_empty() && revision != want_revision {
        return Err("OCI source revision mismatch".to_string());
    }
    if want_revision.is_empty() {
        return Ok(revision);
    }
    let base_digest = labels
        .get("org.opencontainers.image.base.digest")
        .cloned()
        .unwrap_or_default();
    let base_hex = base_digest.strip_prefix("sha256:").unwrap_or("");
    let source_ok = labels
        .get("org.opencontainers.image.source")
        .map(|s| s.as_str())
        == Some("https://github.com/LevitateOS/sodaos");
    let base_name_ok = labels
        .get("org.opencontainers.image.base.name")
        .map(|s| !s.is_empty())
        .unwrap_or(false);
    if !source_ok || !base_name_ok || !base_digest.starts_with("sha256:") || !is_digest(base_hex) {
        return Err("soda image lacks source/base attribution".to_string());
    }
    Ok(revision)
}

fn inspect_oci_image(
    loader: &mut LayoutLoader,
    image: &OciDescriptor,
    want: &str,
    revision: &str,
) -> Result<OciImage, String> {
    let manifest_blob = loader.fetch(image)?;
    let manifest = parse_oci_manifest(manifest_blob.data.as_deref().unwrap_or(b""))?;
    validate_oci_layers(loader, &manifest.layers)?;
    let config_blob = loader.fetch(&manifest.config)?;
    let config = parse_oci_config(config_blob.data.as_deref().unwrap_or(b""))?;
    if config.rootfs_type != "layers" {
        return Err("OCI rootfs/layer count mismatch".to_string());
    }
    validate_oci_rootfs(&config.diff_ids, &manifest.layers)?;
    if config.os != "linux" || config.arch != want {
        return Err(format!("OCI must be linux/{want}"));
    }
    let found_revision = validate_oci_attribution(&config.labels, revision)?;
    Ok(OciImage {
        manifest: image.digest.clone(),
        config: manifest.config.digest.clone(),
        architecture: config.arch,
        revision: found_revision,
        source: config
            .labels
            .get("org.opencontainers.image.source")
            .cloned()
            .unwrap_or_default(),
        base_name: config
            .labels
            .get("org.opencontainers.image.base.name")
            .cloned()
            .unwrap_or_default(),
        base_digest: config
            .labels
            .get("org.opencontainers.image.base.digest")
            .cloned()
            .unwrap_or_default(),
    })
}

/// Verify the exact named image set and its local blobs without running
/// image code. References are immutable config IDs, as in our OCI exports.
fn inspect_oci_layout(
    dir: &Path,
    arch: &str,
    revisions: &HashMap<String, String>,
) -> Result<HashMap<String, OciImage>, String> {
    let want = oci_architecture(arch)?;
    if revisions.is_empty() {
        return Err("explicit OCI image set required".to_string());
    }
    for (reference, revision) in revisions {
        let hex = reference.strip_prefix("sha256:").unwrap_or("");
        if !reference.starts_with("sha256:")
            || !is_digest(hex)
            || (!revision.is_empty() && !is_revision(revision))
        {
            return Err("invalid OCI image selection".to_string());
        }
    }
    open_layout_root(dir)?;
    let mut loader = LayoutLoader {
        dir: dir.to_path_buf(),
        entries: HashMap::new(),
        json_bytes: 0,
    };
    loader.load("index.json")?;
    loader.load("oci-layout")?;
    let index = read_oci_index(&loader.entries)?;
    if index.len() != revisions.len() {
        return Err("exact OCI image set required".to_string());
    }
    let mut images: HashMap<String, OciImage> = HashMap::with_capacity(index.len());
    for descriptor in &index {
        let reference = descriptor
            .annotations
            .get("org.opencontainers.image.ref.name")
            .cloned()
            .unwrap_or_default();
        let revision = match revisions.get(&reference) {
            Some(revision) => revision.clone(),
            None => {
                return Err("unexpected or duplicate OCI image reference".to_string());
            }
        };
        if images.contains_key(&reference)
            || descriptor.media_type != "application/vnd.oci.image.manifest.v1+json"
        {
            return Err("unexpected or duplicate OCI image reference".to_string());
        }
        let image = inspect_oci_image(&mut loader, descriptor, want, &revision)?;
        if image.config != reference {
            return Err("OCI reference differs from config identity".to_string());
        }
        images.insert(reference, image);
    }
    Ok(images)
}

// ---------- content binding + native import (deliver/content.go, import.go) ----------

fn bind_image_revisions(payload: &Payload) -> Result<HashMap<String, String>, String> {
    let mut revisions = HashMap::with_capacity(NAMES.len());
    for name in NAMES {
        let revision = if name == "proxy" {
            String::new()
        } else {
            payload.revision.clone()
        };
        let reference = payload
            .images
            .get(name)
            .map(|image| image.config.clone())
            .unwrap_or_default();
        if let Some(previous) = revisions.get(&reference) {
            if *previous != revision {
                return Err("conflicting OCI source bindings".to_string());
            }
        }
        revisions.insert(reference, revision);
    }
    Ok(revisions)
}

fn match_layout_identities(
    payload: &Payload,
    images: &HashMap<String, OciImage>,
) -> Result<(), String> {
    for name in NAMES {
        let expected = match payload.images.get(name) {
            Some(image) => image,
            None => return Err(format!("{name} OCI identity mismatch")),
        };
        match images.get(&expected.config) {
            Some(got) if got.config == expected.config && got.manifest == expected.manifest => {}
            _ => return Err(format!("{name} OCI identity mismatch")),
        }
    }
    Ok(())
}

/// Bind the shared OCI content to the payload before import. Each shared
/// blob is hashed and counted once; export tar files are not embedded.
fn verify_content(payload: &Payload, images: &str) -> Result<(), String> {
    payload.validate()?;
    if !images.starts_with('/')
        || images.contains(':')
        || images.contains('\r')
        || images.contains('\n')
    {
        return Err("absolute local OCI directory required".to_string());
    }
    let revisions = bind_image_revisions(payload)?;
    let layout = inspect_oci_layout(Path::new(images), &payload.architecture, &revisions)?;
    match_layout_identities(payload, &layout)
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum PodmanOutcome {
    Code(i32),
    /// Spawn failure, signal death, or killed on cancel/timeout: never an
    /// exit-code match, exactly like a non-`ExitError` Go failure.
    Failed,
}

/// Verify release content and import missing exact images into ordinary
/// Podman. Never starts, replaces or deletes workloads.
fn import_missing_image(
    name: &str,
    image: &ImageBinding,
    images_dir: &str,
    podman: &str,
    ctx: &ImportCtx,
    run: &mut dyn FnMut(&str, &[String]) -> PodmanOutcome,
) -> Result<(), String> {
    ctx.check()?;
    let exists = vec![
        "--remote=false".to_string(),
        "image".to_string(),
        "exists".to_string(),
        image.config.clone(),
    ];
    match run(podman, &exists) {
        PodmanOutcome::Code(0) => return Ok(()),
        PodmanOutcome::Code(1) => {}
        _ => return Err(format!("{name} image observation failed")),
    }
    let pull = vec![
        "--remote=false".to_string(),
        "pull".to_string(),
        "--retry=0".to_string(),
        format!("oci:{images_dir}:{}", image.config),
    ];
    match run(podman, &pull) {
        PodmanOutcome::Code(0) => {}
        _ => return Err(format!("{name} image import unconfirmed")),
    }
    match run(podman, &exists) {
        PodmanOutcome::Code(0) => Ok(()),
        _ => Err(format!("{name} imported image unavailable")),
    }
}

fn import_images(
    payload: &Payload,
    images_dir: &str,
    podman: &str,
    ctx: &ImportCtx,
    run: &mut dyn FnMut(&str, &[String]) -> PodmanOutcome,
) -> Result<(), String> {
    ctx.check()?;
    verify_content(payload, images_dir)?;
    for name in NAMES {
        let image = payload.images.get(name).cloned().unwrap_or_default();
        import_missing_image(name, &image, images_dir, podman, ctx, run)?;
    }
    Ok(())
}

fn run_podman(podman: &str, args: &[String], ctx: &ImportCtx) -> PodmanOutcome {
    let mut child = match Command::new(podman)
        .args(args)
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .spawn()
    {
        Ok(child) => child,
        Err(_) => return PodmanOutcome::Failed,
    };
    loop {
        if ctx.cancelled.load(Ordering::SeqCst) || Instant::now() >= ctx.deadline {
            let _ = child.kill();
            let _ = child.wait();
            return PodmanOutcome::Failed;
        }
        match child.try_wait() {
            Ok(Some(status)) => {
                return match status.code() {
                    Some(code) => PodmanOutcome::Code(code),
                    None => PodmanOutcome::Failed,
                };
            }
            Ok(None) => std::thread::sleep(Duration::from_millis(10)),
            Err(_) => {
                let _ = child.kill();
                return PodmanOutcome::Failed;
            }
        }
    }
}

fn native_import(
    payload: &Payload,
    images_dir: &str,
    podman: &str,
    ctx: &ImportCtx,
) -> Result<(), String> {
    import_images(payload, images_dir, podman, ctx, &mut |cmd, args| {
        run_podman(cmd, args, ctx)
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::AtomicU64;

    static TEST_SEQ: AtomicU64 = AtomicU64::new(0);

    fn test_root(name: &str) -> PathBuf {
        let dir = std::env::temp_dir().join(format!(
            "soda-image-import-test-{}-{}-{name}",
            std::process::id(),
            TEST_SEQ.fetch_add(1, Ordering::SeqCst)
        ));
        let _ = fs::remove_dir_all(&dir);
        fs::create_dir_all(&dir).expect("test root");
        dir
    }

    fn repeat(ch: char, n: usize) -> String {
        std::iter::repeat_n(ch, n).collect()
    }

    fn test_ctx(cancelled: &AtomicBool) -> ImportCtx<'_> {
        ImportCtx {
            deadline: Instant::now() + Duration::from_secs(600),
            cancelled,
        }
    }

    // ----- OCI layout fixtures (inert; layer bytes are never executed) -----

    struct LayoutImage {
        config: String,
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
    fn build_layout(
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

    fn full_fixture(root: &Path) -> (Payload, PathBuf) {
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
    fn payload_json_for(entries: &HashMap<String, (String, String)>) -> String {
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

    #[test]
    fn admission_requires_root_and_no_arguments() {
        assert!(admit(0, 1).is_ok());
        assert_eq!(admit(1, 1).unwrap_err(), "root and no arguments required");
        assert_eq!(admit(0, 2).unwrap_err(), "root and no arguments required");
        assert_eq!(admit(0, 0).unwrap_err(), "root and no arguments required");
    }

    #[test]
    fn identifier_shapes_match_go_regexps() {
        assert!(is_digest(&repeat('a', 64)));
        assert!(is_digest(&repeat('9', 64)));
        assert!(!is_digest(&repeat('a', 63)));
        assert!(!is_digest(&repeat('a', 65)));
        assert!(!is_digest(&repeat('A', 64)));
        assert!(!is_digest(&repeat('g', 64)));
        assert!(is_revision(&repeat('f', 40)));
        assert!(!is_revision(&repeat('f', 39)));
        assert!(!is_revision(&repeat('F', 40)));
        assert!(is_prefixed_digest(&format!("sha256:{}", repeat('0', 64))));
        assert!(!is_prefixed_digest(&repeat('0', 64)));
        assert!(!is_prefixed_digest(&format!("sha256:{}", repeat('0', 63))));
        assert!(is_coreos_version("44.20260817.3.2"));
        assert!(!is_coreos_version("44.20260817.3"));
        assert!(!is_coreos_version("44.20260817.3.2.1"));
        assert!(!is_coreos_version("44.20260817.3.x"));
        assert!(!is_coreos_version(""));
        assert!(valid_repository_prefix("ghcr.io/example/sodaos"));
        assert!(valid_repository_prefix("ghcr.io/a-b/c_d.e-f"));
        assert!(!valid_repository_prefix(
            "ghcr.io/example/soda\nImage=untrusted"
        ));
        assert!(!valid_repository_prefix("quay.io/example/sodaos"));
        assert!(!valid_repository_prefix("ghcr.io/example"));
        assert!(!valid_repository_prefix("ghcr.io/example/a/b"));
        assert!(!valid_repository_prefix("ghcr.io/-bad/repo"));
        assert!(!valid_repository_prefix(&format!(
            "ghcr.io/example/{}",
            repeat('a', 200)
        )));
    }

    #[test]
    fn sha256_matches_fips_vectors_streamed_and_oneshot() {
        let vectors: &[(&[u8], &str)] = &[
            (
                b"",
                "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
            ),
            (
                b"abc",
                "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
            ),
            (
                b"abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq",
                "248d6a61d20638b8e5c026930c3e6039a33ce45964ff2167f6ecedd419db06c1",
            ),
        ];
        for (input, want) in vectors {
            assert_eq!(&sha256_hex(input), want);
            // Feed byte-by-byte plus odd chunks to exercise block splits.
            let mut h = Sha256::new();
            for byte in input.iter() {
                h.update(std::slice::from_ref(byte));
            }
            assert_eq!(&hex_lower(&h.finish()), want);
        }
        let million = vec![b'a'; 1_000_000];
        let mut h = Sha256::new();
        for chunk in million.chunks(333) {
            h.update(chunk);
        }
        assert_eq!(
            hex_lower(&h.finish()),
            "cdc76e5c9914fb9281a1c7e284d73e67f1809a48a497200e046d39ccc7112cd0"
        );
    }

    #[test]
    fn native_platform_matches_go_checks() {
        assert_eq!(oci_architecture("x86_64").unwrap(), "amd64");
        assert_eq!(oci_architecture("aarch64").unwrap_err(), "expected x86_64");
        assert_eq!(oci_architecture("").unwrap_err(), "expected x86_64");
        assert!(require_native("x86_64").is_ok());
        assert_eq!(require_native("armv7").unwrap_err(), "expected x86_64");
    }

    fn decode(json: &str) -> Result<Payload, String> {
        decode_payload(&parse_json(json.as_bytes())?)
    }

    #[test]
    fn payload_decode_is_strict_like_disallow_unknown_fields() {
        let valid = payload_json_for(&HashMap::new());
        let payload = decode(&valid).expect("valid payload");
        assert!(payload.validate().is_ok());
        assert_eq!(payload.images.len(), 6);
        // Trailing data rejected, like the Go payload test's suffix.
        assert!(decode(&format!("{valid} {{\"unexpected\":true}}")).is_err());
        // Unknown top-level and nested fields rejected.
        assert!(decode(&valid.replace("\"Format\":3", "\"Format\":3,\"Bogus\":1")).is_err());
        assert!(decode(&valid.replace(
            "\"ArchiveSHA256\":\"",
            "\"ArchiveSHA256\":\"\", \"Extra\":\"x\", \"Ignored\":\""
        ))
        .is_err());
        // Wrong types rejected.
        assert!(decode(&valid.replace("\"Format\":3", "\"Format\":\"3\"")).is_err());
        assert!(decode(&valid.replace("\"Format\":3", "\"Format\":3.0")).is_err());
        assert!(decode(&valid.replace("\"Images\":{", "\"Images\":[]")).is_err());
        // Malformed JSON and non-object top level rejected.
        assert!(decode("{\"Format\":}").is_err());
        assert!(decode("[]").is_err());
        // Duplicates keep last value; null is a no-op; fold matches.
        let dup = valid.replacen("\"Format\":3", "\"Format\":2,\"Format\":3", 1);
        assert_eq!(decode(&dup).expect("dup").format, 3);
        let nul = valid.replacen("\"Format\":3", "\"Format\":3,\"Format\":null", 1);
        assert_eq!(decode(&nul).expect("null").format, 3);
        let folded = valid.replacen("\"Format\":3", "\"format\":3", 1);
        assert_eq!(decode(&folded).expect("fold").format, 3);
        // Null and missing UpgradeFrom both decode as empty.
        let null_up = valid.replace("\"UpgradeFrom\":[]", "\"UpgradeFrom\":null");
        assert!(decode(&null_up).expect("null list").upgrade_from.is_empty());
        let missing_up = valid.replace(",\"UpgradeFrom\":[]", "");
        assert!(decode(&missing_up)
            .expect("missing list")
            .upgrade_from
            .is_empty());
    }

    #[test]
    fn payload_validation_rejects_go_test_mutations() {
        let root = test_root("payload-validate");
        let (base, _) = full_fixture(&root);
        assert!(base.validate().is_ok());
        let mut bad_format = base.clone();
        bad_format.format = 2;
        assert!(bad_format.validate().is_err());
        let mut bad_revision = base.clone();
        bad_revision.revision = "dirty".to_string();
        assert!(bad_revision.validate().is_err());
        let mut bad_arch = base.clone();
        bad_arch.architecture = "armv7".to_string();
        assert!(bad_arch.validate().is_err());
        let mut bad_presentation = base.clone();
        bad_presentation.presentation_sha256.clear();
        assert!(bad_presentation.validate().is_err());
        let mut bad_upgrade = base.clone();
        bad_upgrade.upgrade_from = vec!["unproved".to_string()];
        assert_eq!(
            bad_upgrade.validate().unwrap_err(),
            "candidate has no qualified upgrade paths"
        );
        let mut bad_prefix = base.clone();
        bad_prefix.repository_prefix = "ghcr.io/example/soda\nImage=untrusted".to_string();
        assert!(bad_prefix.validate().is_err());
        let mut missing_proxy = base.clone();
        missing_proxy.images.remove("proxy");
        assert_eq!(
            missing_proxy.validate().unwrap_err(),
            "complete image set required"
        );
        let mut bad_reference = base.clone();
        bad_reference.images.get_mut("dashboard").unwrap().reference =
            "ghcr.io/example/dashboard:latest".to_string();
        assert_eq!(
            bad_reference.validate().unwrap_err(),
            "invalid dashboard image binding"
        );
        let mut reused = base.clone();
        let forgejo_config = reused.images["forgejo"].config.clone();
        reused.images.get_mut("extension").unwrap().config = forgejo_config;
        assert!(reused
            .validate()
            .unwrap_err()
            .contains("independent image identity"));
    }

    #[test]
    fn payload_load_enforces_regular_bounded_input() {
        let root = test_root("payload-load");
        let (payload, _) = full_fixture(&root);
        // Round-trip through JSON text built from the validated struct.
        let mut entries = HashMap::new();
        for name in NAMES {
            let binding = &payload.images[name];
            entries.insert(
                name.to_string(),
                (binding.config.clone(), binding.manifest.clone()),
            );
        }
        let json = payload_json_for(&entries);
        let path = root.join("release.json");
        fs::write(&path, &json).expect("write release");
        let loaded = Payload::load(&path).expect("load");
        assert!(loaded.validate().is_ok());
        assert_eq!(loaded.images.len(), 6);
        assert!(Payload::load(&root.join("missing.json")).is_err());
        fs::write(&path, format!("{json} {{\"unexpected\":true}}")).expect("trailing");
        assert!(Payload::load(&path).is_err());
        // Symlinked input refused like the Go regular-file check.
        let outside = root.join("outside.json");
        fs::write(&outside, &json).expect("outside");
        let link = root.join("linked.json");
        std::os::unix::fs::symlink(&outside, &link).expect("symlink");
        assert_eq!(
            Payload::load(&link).unwrap_err(),
            "bounded regular JSON input required"
        );
        // Oversized input refused.
        let big = root.join("big.json");
        fs::write(&big, vec![b' '; (4 << 20) + 1]).expect("big");
        assert_eq!(
            Payload::load(&big).unwrap_err(),
            "bounded regular JSON input required"
        );
    }

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

    #[test]
    fn content_imports_exact_local_references() {
        for already_present in [false, true] {
            let root = test_root(&format!("content-{already_present}"));
            let (payload, layout) = full_fixture(&root);
            let dir = layout.to_str().unwrap().to_string();
            assert!(verify_content(&payload, &dir).is_ok());
            let cancelled = AtomicBool::new(false);
            let ctx = test_ctx(&cancelled);
            let mut present = std::collections::HashSet::new();
            let mut queries = 0;
            let mut pulls = 0;
            let mut pull_names: Vec<&str> = Vec::new();
            let mut run = |cmd: &str, args: &[String]| -> PodmanOutcome {
                assert_eq!(cmd, "/usr/bin/podman");
                assert_eq!(args[0], "--remote=false");
                match args[1].as_str() {
                    "image" => {
                        assert_eq!(args[2], "exists");
                        assert_eq!(args.len(), 4);
                        queries += 1;
                        if already_present || present.contains(&args[3]) {
                            PodmanOutcome::Code(0)
                        } else {
                            PodmanOutcome::Code(1)
                        }
                    }
                    "pull" => {
                        let expected = &payload.images[NAMES[pulls]];
                        assert_eq!(
                            args.to_vec(),
                            vec![
                                "--remote=false".to_string(),
                                "pull".to_string(),
                                "--retry=0".to_string(),
                                format!("oci:{dir}:{}", expected.config),
                            ]
                        );
                        present.insert(expected.config.clone());
                        pull_names.push(NAMES[pulls]);
                        pulls += 1;
                        PodmanOutcome::Code(0)
                    }
                    other => panic!("unexpected native operation: {other}"),
                }
            };
            import_images(&payload, &dir, "/usr/bin/podman", &ctx, &mut run).expect("import");
            if already_present {
                assert_eq!(queries, NAMES.len());
                assert_eq!(pulls, 0);
            } else {
                assert_eq!(queries, 2 * NAMES.len());
                assert_eq!(pulls, NAMES.len());
                assert_eq!(pull_names, NAMES.to_vec());
            }
        }
    }

    #[test]
    fn content_refuses_whole_layout_before_any_import() {
        for kind in [
            "bad-last-config",
            "wrong-manifest",
            "relative-path",
            "transport-separator",
            "wrong-format",
        ] {
            let root = test_root(&format!("refuse-{kind}"));
            let (mut payload, layout) = full_fixture(&root);
            let mut dir = layout.to_str().unwrap().to_string();
            match kind {
                "bad-last-config" => {
                    let last = NAMES[NAMES.len() - 1];
                    let hex = payload.images[last]
                        .config
                        .trim_start_matches("sha256:")
                        .to_string();
                    fs::write(layout.join("blobs").join("sha256").join(hex), b"bad")
                        .expect("clobber");
                }
                "wrong-manifest" => {
                    let manifest = format!("sha256:{}", repeat('9', 64));
                    let binding = payload.images.get_mut("dashboard").unwrap();
                    binding.manifest = manifest.clone();
                    binding.reference =
                        format!("{}-dashboard@{manifest}", payload.repository_prefix);
                }
                "relative-path" => dir = "relative/layout".to_string(),
                "transport-separator" => dir += "ignored:selector",
                "wrong-format" => payload.format = 2,
                _ => unreachable!(),
            }
            let cancelled = AtomicBool::new(false);
            let ctx = test_ctx(&cancelled);
            let mut calls = 0;
            let mut run = |_: &str, _: &[String]| -> PodmanOutcome {
                calls += 1;
                PodmanOutcome::Code(0)
            };
            assert!(
                import_images(&payload, &dir, "/usr/bin/podman", &ctx, &mut run).is_err(),
                "{kind} accepted"
            );
            assert_eq!(calls, 0, "{kind} ran podman");
        }
    }

    #[test]
    fn native_failures_remain_unconfirmed_without_replay() {
        for kind in ["observation", "pull", "postcheck", "cancelled", "expired"] {
            let root = test_root(&format!("fail-{kind}"));
            let (payload, layout) = full_fixture(&root);
            let dir = layout.to_str().unwrap().to_string();
            let cancelled = AtomicBool::new(kind == "cancelled");
            let mut ctx = test_ctx(&cancelled);
            if kind == "expired" {
                ctx.deadline = Instant::now() - Duration::from_secs(1);
            }
            let mut queries = 0;
            let mut pulls = 0;
            let mut run = |_: &str, args: &[String]| -> PodmanOutcome {
                if args[1] == "image" {
                    queries += 1;
                    if kind == "observation" {
                        return PodmanOutcome::Code(125);
                    }
                    if kind == "postcheck" && queries == 2 {
                        return PodmanOutcome::Code(1);
                    }
                    return PodmanOutcome::Code(if queries == 1 { 1 } else { 0 });
                }
                assert_eq!(args[1], "pull");
                pulls += 1;
                if kind == "pull" {
                    return PodmanOutcome::Code(3);
                }
                PodmanOutcome::Code(0)
            };
            let err = import_images(&payload, &dir, "/usr/bin/podman", &ctx, &mut run).unwrap_err();
            match kind {
                "observation" => {
                    assert!(err.contains("observation failed"), "{err}");
                    assert_eq!(queries, 1);
                    assert_eq!(pulls, 0);
                }
                "pull" => {
                    assert!(err.contains("import unconfirmed"), "{err}");
                    assert_eq!(queries, 1);
                    assert_eq!(pulls, 1);
                }
                "postcheck" => {
                    assert!(err.contains("unavailable"), "{err}");
                    assert_eq!(queries, 2);
                    assert_eq!(pulls, 1);
                }
                "cancelled" => {
                    assert_eq!(err, "context canceled");
                    assert_eq!(queries, 0);
                    assert_eq!(pulls, 0);
                }
                "expired" => {
                    assert_eq!(err, "context deadline exceeded");
                    assert_eq!(queries, 0);
                    assert_eq!(pulls, 0);
                }
                _ => unreachable!(),
            }
        }
    }

    fn fake_podman(root: &Path, name: &str, body: &str) -> String {
        let path = root.join(name);
        fs::write(&path, format!("#!/bin/sh\n{body}\n")).expect("fake podman");
        use std::os::unix::fs::PermissionsExt;
        fs::set_permissions(&path, fs::Permissions::from_mode(0o755)).expect("chmod");
        path.to_str().unwrap().to_string()
    }

    #[test]
    fn runner_maps_exits_and_kills_on_deadline() {
        let root = test_root("runner");
        let cancelled = AtomicBool::new(false);
        let ctx = test_ctx(&cancelled);
        let args = vec![
            "--remote=false".to_string(),
            "image".to_string(),
            "exists".to_string(),
            "x".to_string(),
        ];
        let ok = fake_podman(&root, "podman-ok", "exit 0");
        assert_eq!(run_podman(&ok, &args, &ctx), PodmanOutcome::Code(0));
        let missing = fake_podman(&root, "podman-missing", "exit 1");
        assert_eq!(run_podman(&missing, &args, &ctx), PodmanOutcome::Code(1));
        let bad = fake_podman(&root, "podman-bad", "exit 125");
        assert_eq!(run_podman(&bad, &args, &ctx), PodmanOutcome::Code(125));
        assert_eq!(
            run_podman("/nonexistent/podman-binary", &args, &ctx),
            PodmanOutcome::Failed
        );
        // A hung engine is killed once the deadline passes.
        let hung = fake_podman(&root, "podman-hung", "sleep 30");
        let tight = ImportCtx {
            deadline: Instant::now() + Duration::from_millis(200),
            cancelled: &cancelled,
        };
        let start = Instant::now();
        assert_eq!(run_podman(&hung, &args, &tight), PodmanOutcome::Failed);
        assert!(start.elapsed() < Duration::from_secs(10));
        // Cancellation mid-run kills the child too.
        let cancelling = AtomicBool::new(true);
        let cancel_ctx = ImportCtx {
            deadline: Instant::now() + Duration::from_secs(60),
            cancelled: &cancelling,
        };
        assert_eq!(run_podman(&hung, &args, &cancel_ctx), PodmanOutcome::Failed);
    }

    #[test]
    fn json_validity_matches_single_value_rule() {
        assert!(json_valid(br#"{"a":1}"#));
        assert!(json_valid(b"[1,2]"));
        assert!(json_valid(b"  null  "));
        assert!(!json_valid(b""));
        assert!(!json_valid(br#"{"a":1} {"b":2}"#));
        assert!(!json_valid(br#"{"a":}"#));
        assert!(!json_valid(b"\xff\xfe"));
    }
}

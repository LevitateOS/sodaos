//! `assemble.go`: candidate-derived native media assembly.

use std::collections::HashMap;
use std::fs;

use soda_json::JsonValue;

use crate::compression;
use crate::error::Error;
use crate::files;
use crate::foreign::Production;
use crate::ignition;
use crate::jsonio;
use crate::model;
use crate::request;
use crate::rootfs;
use crate::sys;

pub use crate::media_assembler::{
    builder_id, fetch_assembler_config, pin_assembler_build_args, prepare_assembler,
    verify_assembler_layers, wrap_assembler_image, ASSEMBLER_CONFIG_BRANCH, ASSEMBLER_IMAGE,
};

/// Container/run closure (`run func(name string, args ...string)`).
pub type RunFn<'a> = &'a (dyn Fn(&str, &[String]) -> Result<(), Error> + 'a);
/// Native capture closure (`native func(name string, args ...string)`).
pub type NativeFn<'a> = &'a (dyn Fn(&str, &[String]) -> Result<String, Error> + 'a);

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MediaLock {
    pub assembler: String,
    pub config: String,
    pub architecture: String,
    pub installer: String,
}

impl MediaLock {
    pub fn to_json(&self) -> JsonValue {
        JsonValue::Object(vec![
            (
                "Assembler".to_string(),
                JsonValue::Str(self.assembler.clone()),
            ),
            ("Config".to_string(), JsonValue::Str(self.config.clone())),
            (
                "Architecture".to_string(),
                JsonValue::Str(self.architecture.clone()),
            ),
            (
                "Installer".to_string(),
                JsonValue::Str(self.installer.clone()),
            ),
        ])
    }
}

/// Signing inputs belong to the invoking authority, never the source snapshot
/// or child build environment. Local fixture authority does not prove job
/// isolation; protected worker commissioning and final qualification remain
/// B4/B5 work.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MediaAuthority {
    pub trust: String,
    pub keys: model::SecretFiles,
}

impl MediaAuthority {
    pub fn parse(value: &JsonValue) -> Result<MediaAuthority, Error> {
        jsonio::check_no_unknown(value, &["Trust", "Keys"])
            .map_err(|_| Error::msg(sys::refused()))?;
        let keys_value =
            jsonio::require_object(value, "Keys").map_err(|_| Error::msg(sys::refused()))?;
        Ok(MediaAuthority {
            trust: jsonio::require_string(value, "Trust")
                .map_err(|_| Error::msg(sys::refused()))?,
            keys: model::SecretFiles::parse(keys_value)?,
        })
    }
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Media {
    pub scope: String,
    pub revision: String,
    pub architecture: String,
    pub host_manifest: String,
    pub payload_sha256: String,
    pub console_sha256: String,
    pub assembler_import_commit: String,
    pub rootfs_url: String,
    pub iso: files::MediaFile,
    pub rootfs: files::MediaFile,
    pub tools: MediaLock,
    pub compression_mode: String,
    pub rootfs_filesystem: String,
    pub rootfs_options: String,
}

impl Media {
    pub fn to_json(&self) -> JsonValue {
        JsonValue::Object(vec![
            ("Scope".to_string(), JsonValue::Str(self.scope.clone())),
            (
                "Revision".to_string(),
                JsonValue::Str(self.revision.clone()),
            ),
            (
                "Architecture".to_string(),
                JsonValue::Str(self.architecture.clone()),
            ),
            (
                "HostManifest".to_string(),
                JsonValue::Str(self.host_manifest.clone()),
            ),
            (
                "PayloadSHA256".to_string(),
                JsonValue::Str(self.payload_sha256.clone()),
            ),
            (
                "ConsoleSHA256".to_string(),
                JsonValue::Str(self.console_sha256.clone()),
            ),
            (
                "AssemblerImportCommit".to_string(),
                JsonValue::Str(self.assembler_import_commit.clone()),
            ),
            (
                "RootfsURL".to_string(),
                JsonValue::Str(self.rootfs_url.clone()),
            ),
            ("ISO".to_string(), self.iso.to_json()),
            ("Rootfs".to_string(), self.rootfs.to_json()),
            ("Tools".to_string(), self.tools.to_json()),
            (
                "CompressionMode".to_string(),
                JsonValue::Str(self.compression_mode.clone()),
            ),
            (
                "RootfsFilesystem".to_string(),
                JsonValue::Str(self.rootfs_filesystem.clone()),
            ),
            (
                "RootfsOptions".to_string(),
                JsonValue::Str(self.rootfs_options.clone()),
            ),
        ])
    }
}

pub fn media_base_url(value: &str) -> Result<(), Error> {
    let parsed = model::parse_url(value);
    match parsed {
        Some(u)
            if !u.host.is_empty()
                && (u.scheme == "https" || u.scheme == "http")
                && !u.user
                && u.raw_query.is_empty()
                && u.fragment.is_empty()
                && !value.chars().any(|c| c == '\r' || c == '\n' || c == ' ') => {}
        _ => {
            return Err(Error::msg(
                "explicit public HTTP(S) rootfs base URL required",
            ))
        }
    }
    // The installing machine fetches this address: a loopback URL always
    // points at the guest itself, never at the machine serving the rootfs.
    let host = model::parse_url(value)
        .map(|u| u.hostname.to_ascii_lowercase())
        .unwrap_or_default();
    if host == "localhost" {
        return Err(Error::msg(
            "rootfs base URL must be reachable from the installing machine, not loopback",
        ));
    }
    if model::is_loopback_addr(&host) {
        return Err(Error::msg(
            "rootfs base URL must be reachable from the installing machine, not loopback",
        ));
    }
    Ok(())
}

pub fn sign_media_input(
    production: &dyn Production,
    authority: &MediaAuthority,
    trust: &model::Trust,
    input: &str,
    transport: &str,
    repository: &str,
    digest: &str,
    out: &str,
) -> Result<(), Error> {
    let trust_home = sys::dir_name(&authority.trust);
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs() as i64)
        .unwrap_or(0);
    let permit = model::Permit {
        format: 1,
        repository: repository.to_string(),
        digest: digest.to_string(),
        expires: now + 3600,
        ..model::Permit::default()
    };
    production.sign_media(
        trust,
        &permit,
        transport,
        input,
        out,
        &authority.keys,
        &trust_home,
    )?;
    production.verify_copy(
        trust,
        &format!("{repository}@{digest}"),
        &format!("dir:{}", sys::join(&[out, "signed"])),
        &format!("{out}-admitted"),
        &trust_home,
    )
}

#[derive(Debug, Default)]
pub struct MediaInputs {
    pub authority: MediaAuthority,
    pub trust: model::Trust,
    pub candidate: model::Candidate,
    pub payload: model::Payload,
    pub observed: model::Image,
    pub archive: String,
    pub filesystem: String,
    pub fsoptions: String,
}

pub fn admit_media_authority(
    authority_path: &str,
    prefix: &str,
) -> Result<(MediaAuthority, model::Trust), Error> {
    let authority_value = sys::read_json_deliver(authority_path)?;
    let authority = MediaAuthority::parse(&authority_value)?;
    let trust_value = sys::read_json_deliver(&authority.trust)?;
    let trust = model::Trust::parse(&trust_value)?;
    if trust.validate().is_err() || trust.prefix != prefix {
        return Err(Error::msg(
            "media authority does not match intended repositories",
        ));
    }
    Ok((authority, trust))
}

pub fn admit_media_candidate(
    production: &dyn Production,
    artifacts: &str,
    arch: &str,
    revision: &str,
) -> Result<(model::Candidate, model::Payload, model::Image, String), Error> {
    let candidate_value = sys::read_json_deliver(&sys::join(&[artifacts, "candidate.json"]))?;
    let candidate = model::Candidate::parse(&candidate_value)?;
    let payload = model::Payload::load(&sys::join(&[artifacts, "payload.json"]))?;
    let payload_bytes = fs::read(sys::join(&[artifacts, "payload.json"]))?;
    candidate.validate(&payload, &payload_bytes)?;
    let archive = sys::join(&[artifacts, "host.oci"]);
    let observed = production.inspect_oci(&archive, arch, revision);
    match observed {
        Ok(observed) if observed.manifest == candidate.host.manifest => {
            let hash = sys::hash_file(&archive);
            match hash {
                Ok(hash) if hash == candidate.host_archive_sha256 => {
                    Ok((candidate, payload, observed, archive))
                }
                _ => Err(Error::msg("media candidate archive mismatch")),
            }
        }
        _ => Err(Error::msg("media candidate identity mismatch")),
    }
}

pub fn admit_media_compression(
    artifacts: &str,
    media_compression: &str,
) -> Result<(String, String), Error> {
    let (filesystem, fsoptions) = compression::rootfs_settings(artifacts)?;
    if media_compression == "fast"
        && (filesystem != "erofs" || fsoptions != compression::FAST_ROOTFS_OPTIONS)
    {
        return Err(Error::msg(
            "fast media metadata differs from selected setting",
        ));
    }
    Ok((filesystem, fsoptions))
}

pub fn admit_media_inputs(
    production: &dyn Production,
    request: &request::Request,
    artifacts: &str,
    arch: &str,
    revision: &str,
) -> Result<MediaInputs, Error> {
    let mut inputs = MediaInputs::default();
    let (authority, trust) =
        admit_media_authority(&request.media_authority, &request.repository_prefix)?;
    inputs.authority = authority;
    inputs.trust = trust;
    let (candidate, payload, observed, archive) =
        admit_media_candidate(production, artifacts, arch, revision)?;
    inputs.candidate = candidate;
    inputs.payload = payload;
    inputs.observed = observed;
    inputs.archive = archive;
    let (filesystem, fsoptions) = admit_media_compression(artifacts, &request.media_compression)?;
    inputs.filesystem = filesystem;
    inputs.fsoptions = fsoptions;
    Ok(inputs)
}

pub fn collect_media_inventory(
    root: &str,
    artifacts: &str,
    out_dir: &str,
) -> Result<HashMap<String, String>, Error> {
    let mut inventory: HashMap<String, String> = HashMap::new();
    for dir in [
        sys::join(&[root, "config"]),
        sys::join(&[artifacts, "tools"]),
    ] {
        sys::walk(&dir, |path, is_dir, is_link| {
            if is_dir || is_link {
                return Ok(());
            }
            let hash = sys::hash_file(path)?;
            let rel = sys::rel_path(out_dir, path).unwrap_or_else(|_| path.to_string());
            inventory.insert(rel, hash);
            Ok(())
        })?;
    }
    for path in [
        sys::join(&[root, "assembler-root.oci"]),
        sys::join(&[root, "config.tar"]),
        sys::join(&[artifacts, "live.ign"]),
        sys::join(&[artifacts, "destination.ign"]),
        sys::join(&[artifacts, "candidate.json"]),
        sys::join(&[artifacts, "payload.json"]),
        sys::join(&[artifacts, "image-config.json"]),
    ] {
        let hash = sys::hash_file(&path)?;
        let rel = sys::rel_path(out_dir, &path).unwrap_or_else(|_| path.clone());
        inventory.insert(rel, hash);
    }
    Ok(inventory)
}

pub fn verify_media_inventory(
    out_dir: &str,
    inventory: &HashMap<String, String>,
) -> Result<(), Error> {
    let mut paths: Vec<&String> = inventory.keys().collect();
    paths.sort();
    for path in paths {
        let want = &inventory[path];
        match sys::hash_file(&sys::join(&[out_dir, path])) {
            Ok(got) if &got == want => {}
            _ => return Err(Error::msg("packaging input changed after admission")),
        }
    }
    Ok(())
}

pub fn authenticate_packaging_inputs(
    production: &dyn Production,
    request: &request::Request,
    root: &str,
    artifacts: &str,
    archive: &str,
    manifest: &str,
    authority: &MediaAuthority,
    trust: &model::Trust,
    lock: &MediaLock,
) -> Result<(), Error> {
    production.check_native(&sys::dir_name(&authority.trust))?;
    sign_media_input(
        production,
        authority,
        trust,
        archive,
        "oci-archive",
        &format!("{}-host", request.repository_prefix),
        manifest,
        &sys::join(&[root, "host-signature"]),
    )?;
    // The signed public inventory authenticates exactly the auxiliary inputs
    // used by the native packager. No source script runs in this signing
    // operation.
    let inventory = collect_media_inventory(root, artifacts, &request.out)?;
    let mut files: Vec<(String, String)> = inventory.clone().into_iter().collect();
    files.sort_by(|a, b| a.0.cmp(&b.0));
    let document_value = JsonValue::Object(vec![
        ("Tools".to_string(), lock.to_json()),
        (
            "Files".to_string(),
            JsonValue::Object(
                files
                    .into_iter()
                    .map(|(k, v)| (k, JsonValue::Str(v)))
                    .collect(),
            ),
        ),
    ]);
    let document = sys::join(&[root, "input-document"]);
    let digest = production.write_document(&document, &document_value)?;
    sign_media_input(
        production,
        authority,
        trust,
        &document,
        "oci",
        &format!("{}-media", request.repository_prefix),
        &digest,
        &sys::join(&[root, "input-signature"]),
    )?;
    verify_media_inventory(&request.out, &inventory)
}

pub fn stop_packaging_container(root: &str) {
    let cid_path = sys::join(&[root, "packaging.cid"]);
    let Ok(cid_bytes) = fs::read(&cid_path) else {
        return;
    };
    let cid = String::from_utf8_lossy(&cid_bytes).trim().to_string();
    if !model::is_digest(&cid) {
        return;
    }
    let mut child = match std::process::Command::new("/usr/bin/podman")
        .args(["--remote=false", "stop", "--time=10", &cid])
        .env_clear()
        .envs(crate::build_runner::build_environment_pairs())
        .spawn()
    {
        Ok(child) => child,
        Err(_) => return,
    };
    let deadline = std::time::Instant::now() + std::time::Duration::from_secs(30);
    loop {
        match child.try_wait() {
            Ok(Some(_)) => break,
            Ok(None) if std::time::Instant::now() < deadline => {
                std::thread::sleep(std::time::Duration::from_millis(50));
            }
            _ => {
                let _ = child.kill();
                let _ = child.wait();
                break;
            }
        }
    }
}

pub fn build_media_container(
    production: &dyn Production,
    root: &str,
    artifacts: &str,
    payload_id: &str,
    work: &str,
    id: &str,
) -> Result<(), Error> {
    let mut base = vec![
        "--remote=false".to_string(),
        "run".to_string(),
        "--rm".to_string(),
        format!("--cidfile={}", sys::join(&[root, "packaging.cid"])),
        "--network=none".to_string(),
        "--privileged".to_string(),
        "--security-opt=label=disable".to_string(),
        "--device=/dev/kvm".to_string(),
        "--device=/dev/fuse".to_string(),
        "--cpus=4".to_string(),
        "--memory=16g".to_string(),
        "--env=COSA_SUPERMIN_MEMORY=12288".to_string(),
        "--env=RUNVM_NONET=1".to_string(),
        format!("--volume={work}:/srv:rw"),
        format!("--volume={}/config:/config:ro", root),
        format!("--volume={artifacts}:/inputs:ro"),
        format!(
            "--volume={}/assembler-root.oci:/assembler-root.oci:ro",
            root
        ),
        "--workdir=/srv".to_string(),
        "--entrypoint=/usr/bin/taskset".to_string(),
        id.to_string(),
        "-c".to_string(),
        "0-3".to_string(),
        "/bin/bash".to_string(),
        "-euc".to_string(),
    ];
    let script = "umask 0022; cosa init /config; cp /assembler-root.oci /srv/tmp/assembler-root.oci; cosa import --skip-prune oci-archive:/inputs/host.oci; cosa buildextend-live --build \"$1\"";
    base.push(script.to_string());
    base.push("assemble".to_string());
    base.push(payload_id.to_string());
    if let Err(e) = production.execute(root, "podman", &base) {
        // Podman owns the helper VM's namespace, outside the CLI process group.
        // Stop only this run's container even when the build context is cancelled.
        stop_packaging_container(root);
        return Err(e);
    }
    Ok(())
}

pub fn assemble_native_media(
    production: &dyn Production,
    root: &str,
    artifacts: &str,
    payload_id: &str,
) -> Result<(String, String), Error> {
    let work = sys::join(&[root, "work"]);
    sys::create_dir(&work, 0o755)?;
    let id = builder_id(root)?;
    build_media_container(production, root, artifacts, payload_id, &work, &id)?;
    Ok((
        sys::join(&[&work, "builds", payload_id, production.arch()]),
        id,
    ))
}

#[derive(Debug, Clone, Default)]
pub struct MediaMetaImage {
    pub path: String,
    pub sha256: String,
    pub size: i64,
}

#[derive(Debug, Clone, Default)]
pub struct MediaMeta {
    pub ostree_commit: String,
    pub images: HashMap<String, MediaMetaImage>,
}

impl MediaMeta {
    pub fn parse(value: &JsonValue) -> Result<MediaMeta, Error> {
        let mut meta = MediaMeta {
            ostree_commit: jsonio::require_string(value, "ostree-commit")?,
            ..MediaMeta::default()
        };
        if let JsonValue::Object(entries) = jsonio::require_object(value, "Images")? {
            for (name, image) in entries {
                meta.images.insert(
                    name.clone(),
                    MediaMetaImage {
                        path: jsonio::require_string(image, "Path")?,
                        sha256: jsonio::require_string(image, "SHA256")?,
                        size: jsonio::require_i64(image, "Size")?,
                    },
                );
            }
        }
        Ok(meta)
    }
}

pub fn verify_meta_images(
    build_dir: &str,
    images: &HashMap<String, MediaMetaImage>,
) -> Result<(), Error> {
    for name in [
        "ostree",
        "oci-manifest",
        "live-iso",
        "live-rootfs",
        "live-initramfs",
    ] {
        let file = images.get(name);
        match file {
            Some(file) if sys::base_name(&file.path) == file.path => {
                match sys::hash_file(&sys::join(&[build_dir, &file.path])) {
                    Ok(hash) if hash == file.sha256 => {}
                    _ => return Err(Error::msg("native output checksum mismatch")),
                }
            }
            _ => return Err(Error::msg("missing native media output")),
        }
    }
    Ok(())
}

pub fn verify_build_meta(
    build_dir: &str,
    candidate: &model::Candidate,
) -> Result<MediaMeta, Error> {
    let data = fs::read(sys::join(&[build_dir, "meta.json"]))?;
    let text = String::from_utf8_lossy(&data).into_owned();
    // Plain (non-strict) decode, like the Go owner.
    let value = jsonio::parse(&text)?;
    let meta = MediaMeta::parse(&value)?;
    verify_meta_images(build_dir, &meta.images)?;
    let ostree = meta.images.get("ostree").cloned().unwrap_or_default();
    let oci_manifest = meta.images.get("oci-manifest").cloned().unwrap_or_default();
    if ostree.sha256 != candidate.host_archive_sha256
        || oci_manifest.sha256 != candidate.host.manifest.trim_start_matches("sha256:")
    {
        return Err(Error::msg("native import changed candidate"));
    }
    Ok(meta)
}

pub fn setup_media_rootfs(
    artifacts: &str,
    build_dir: &str,
    rootfs_path: &str,
    rootfs_sha: &str,
) -> Result<(String, String), Error> {
    let media_dir = sys::join(&[artifacts, "media"]);
    sys::create_dir(&media_dir, 0o755)?;
    let rootfs_name = format!("{rootfs_sha}-rootfs.img");
    fs::hard_link(
        sys::join(&[build_dir, rootfs_path]),
        sys::join(&[&media_dir, &rootfs_name]),
    )
    .map_err(|e| Error::msg(e.to_string()))?;
    Ok((media_dir, rootfs_name))
}

pub fn verify_customized_iso(
    native: NativeFn<'_>,
    artifacts: &str,
    rootfs_url: &str,
) -> Result<(), Error> {
    let ignition = native(
        "/usr/bin/coreos-installer",
        &[
            "iso".to_string(),
            "ignition".to_string(),
            "show".to_string(),
            "/out/media/installer.iso".to_string(),
        ],
    )?;
    let expected = fs::read(sys::join(&[artifacts, "live.ign"]))?;
    // iso customize wraps the supplied fragment in a merge source. Require the
    // exact public fragment to survive native readback (checked below).
    ignition::verify_live_ignition(ignition.as_bytes(), &expected)?;
    let kargs = native(
        "/usr/bin/coreos-installer",
        &[
            "iso".to_string(),
            "kargs".to_string(),
            "show".to_string(),
            "/out/media/installer.iso".to_string(),
        ],
    )?;
    if !kargs.contains(&format!("coreos.live.rootfs_url={rootfs_url}"))
        || kargs.contains("coreos.liveiso")
    {
        return Err(Error::msg("minimal-media kernel arguments differ"));
    }
    Ok(())
}

pub fn customize_installer_iso(
    production: &dyn Production,
    root: &str,
    artifacts: &str,
    build_dir: &str,
    id: &str,
    rootfs_url: &str,
    live_iso_path: &str,
) -> Result<String, Error> {
    let native = |name: &str, args: &[String]| -> Result<String, Error> {
        let mut prefix = vec![
            "--remote=false".to_string(),
            "run".to_string(),
            "--rm".to_string(),
            "--pull=never".to_string(),
            "--network=none".to_string(),
            "--read-only".to_string(),
            "--cap-drop=all".to_string(),
            "--security-opt=label=disable".to_string(),
            format!("--volume={build_dir}:/build:ro"),
            format!("--volume={artifacts}:/out:rw"),
            format!("--entrypoint={name}"),
            id.to_string(),
        ];
        prefix.extend_from_slice(args);
        production.capture(root, "podman", &prefix)
    };
    let version = native("/usr/bin/coreos-installer", &["--version".to_string()])?;
    let version = version.trim().to_string();
    if !version.starts_with("coreos-installer ") || version.chars().any(|c| c == '\r' || c == '\n')
    {
        return Err(Error::msg("unrecognized installer version output"));
    }
    native(
        "/usr/bin/coreos-installer",
        &[
            "iso".to_string(),
            "extract".to_string(),
            "minimal-iso".to_string(),
            format!("/build/{live_iso_path}"),
            "/out/media/minimal.iso".to_string(),
        ],
    )?;
    native(
        "/usr/bin/coreos-installer",
        &[
            "iso".to_string(),
            "customize".to_string(),
            "--live-ignition".to_string(),
            "/out/live.ign".to_string(),
            "--live-karg-append".to_string(),
            format!("coreos.live.rootfs_url={rootfs_url}"),
            "--output".to_string(),
            "/out/media/installer.iso".to_string(),
            "/out/media/minimal.iso".to_string(),
        ],
    )?;
    verify_customized_iso(&native, artifacts, rootfs_url)?;
    Ok(version)
}

pub fn verify_media_readback(
    production: &dyn Production,
    root: &str,
    artifacts: &str,
    build_dir: &str,
    id: &str,
    media_dir: &str,
    rootfs_name: &str,
) -> Result<(), Error> {
    let native = |name: &str, args: &[String]| -> Result<String, Error> {
        let mut prefix = vec![
            "--remote=false".to_string(),
            "run".to_string(),
            "--rm".to_string(),
            "--pull=never".to_string(),
            "--network=none".to_string(),
            "--read-only".to_string(),
            "--cap-drop=all".to_string(),
            "--security-opt=label=disable".to_string(),
            format!("--volume={build_dir}:/build:ro"),
            format!("--volume={artifacts}:/out:rw"),
            format!("--entrypoint={name}"),
            id.to_string(),
        ];
        prefix.extend_from_slice(args);
        production.capture(root, "podman", &prefix)
    };
    let readback = sys::join(&[media_dir, "readback"]);
    sys::create_dir(&readback, 0o755)?;
    native(
        "/usr/bin/coreos-installer",
        &[
            "iso".to_string(),
            "extract".to_string(),
            "pxe".to_string(),
            "--output-dir".to_string(),
            "/out/media/readback".to_string(),
            "/out/media/installer.iso".to_string(),
        ],
    )?;
    let mut initrds: Vec<String> = Vec::new();
    for entry in fs::read_dir(&readback)? {
        let entry = entry?;
        let name = entry.file_name().to_string_lossy().into_owned();
        if name.ends_with("-initrd.img") {
            initrds.push(name);
        }
    }
    if initrds.len() != 1 {
        return Err(Error::msg("missing ISO initrd readback"));
    }
    native(
        "/usr/bin/coreos-installer",
        &[
            "dev".to_string(),
            "extract".to_string(),
            "initrd".to_string(),
            "--directory".to_string(),
            "/out/media/readback".to_string(),
            format!("/out/media/readback/{}", initrds[0]),
            "etc/coreos-live-want-rootfs".to_string(),
        ],
    )?;
    let chunks = fs::read(sys::join(&[&readback, "etc/coreos-live-want-rootfs"]))?;
    rootfs::verify_rootfs_chunks(
        &sys::join(&[media_dir, rootfs_name]),
        &String::from_utf8_lossy(&chunks),
    )
}

pub fn prepare_and_verify_media(
    production: &dyn Production,
    root: &str,
    artifacts: &str,
    build_dir: &str,
    id: &str,
    media_dir: &str,
    meta: &MediaMeta,
    rootfs_url: &str,
    rootfs_name: &str,
) -> Result<String, Error> {
    let live_iso = meta.images.get("live-iso").cloned().unwrap_or_default();
    let version = customize_installer_iso(
        production,
        root,
        artifacts,
        build_dir,
        id,
        rootfs_url,
        &live_iso.path,
    )?;
    verify_media_readback(
        production,
        root,
        artifacts,
        build_dir,
        id,
        media_dir,
        rootfs_name,
    )?;
    Ok(version)
}

pub fn seal_media(
    artifacts: &str,
    media_dir: &str,
    rootfs_name: &str,
    revision: &str,
    arch: &str,
    manifest: &str,
    payload_sha256: &str,
    ostree_commit: &str,
    rootfs_url: &str,
    compression: &str,
    filesystem: &str,
    fsoptions: &str,
    lock: &MediaLock,
) -> Result<Media, Error> {
    let iso = files::media_file(&sys::join(&[media_dir, "installer.iso"]))?;
    if iso.bytes >= 2_000_000_000 {
        return Err(Error::msg("minimal ISO exceeds distribution ceiling"));
    }
    let rootfs = files::media_file(&sys::join(&[media_dir, rootfs_name]))?;
    let console = sys::hash_file(&sys::join(&[artifacts, "tools/soda-installer"]))?;
    let result = Media {
        scope: "candidate-derived media; native qualification and final protected release signing pending".to_string(),
        revision: revision.to_string(),
        architecture: arch.to_string(),
        host_manifest: manifest.to_string(),
        payload_sha256: payload_sha256.to_string(),
        console_sha256: console,
        assembler_import_commit: ostree_commit.to_string(),
        rootfs_url: rootfs_url.to_string(),
        iso,
        rootfs,
        tools: lock.clone(),
        compression_mode: compression.to_string(),
        rootfs_filesystem: filesystem.to_string(),
        rootfs_options: fsoptions.to_string(),
    };
    let mut data = jsonio::to_indent(&result.to_json());
    data.push('\n');
    sys::write_new(
        &sys::join(&[media_dir, "media.json"]),
        data.as_bytes(),
        0o644,
    )?;
    Ok(result)
}

pub fn assemble_media(
    production: &dyn Production,
    request: &request::Request,
    lock: &mut MediaLock,
    next: &mut dyn FnMut(&str) -> Result<(), Error>,
) -> Result<Media, Error> {
    let root = sys::join(&[&request.out, "work/media"]);
    let artifacts = production.out().to_string();
    let inputs = admit_media_inputs(
        production,
        request,
        &artifacts,
        production.arch(),
        production.revision(),
    )?;
    next("P7 / Authenticate packaging inputs")?;
    authenticate_packaging_inputs(
        production,
        request,
        &root,
        &artifacts,
        &inputs.archive,
        &inputs.observed.manifest,
        &inputs.authority,
        &inputs.trust,
        lock,
    )?;
    next("P8 / Assemble candidate-derived native media")?;
    let (build_dir, id) = assemble_native_media(production, &root, &artifacts, &inputs.payload.id)?;
    let meta = verify_build_meta(&build_dir, &inputs.candidate)?;
    let live_rootfs = meta.images.get("live-rootfs").cloned().unwrap_or_default();
    let (media_dir, rootfs_name) = setup_media_rootfs(
        &artifacts,
        &build_dir,
        &live_rootfs.path,
        &live_rootfs.sha256,
    )?;
    let rootfs_url = format!(
        "{}/{}",
        request.rootfs_base_url.trim_end_matches('/'),
        rootfs_name
    );
    let installer_version = prepare_and_verify_media(
        production,
        &root,
        &artifacts,
        &build_dir,
        &id,
        &media_dir,
        &meta,
        &rootfs_url,
        &rootfs_name,
    )?;
    lock.installer = installer_version;
    seal_media(
        &artifacts,
        &media_dir,
        &rootfs_name,
        production.revision(),
        production.arch(),
        &inputs.candidate.host.manifest,
        &inputs.candidate.payload_sha256,
        &meta.ostree_commit,
        &rootfs_url,
        &request.media_compression,
        &inputs.filesystem,
        &inputs.fsoptions,
        lock,
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_media_url_has_no_credentials_or_mutable_query() {
        // Oracle: Go TestMediaURLHasNoCredentialsOrMutableQuery.
        assert!(media_base_url("https://example.invalid/rootfs").is_ok());
        assert!(media_base_url("http://example.invalid/rootfs").is_ok());
        for bad in [
            "https://user@example.invalid/rootfs",
            "https://example.invalid/rootfs?tag=latest",
            "https://example.invalid/rootfs#frag",
            "ftp://example.invalid/rootfs",
            "https:///noroot",
            "https://example.invalid/has space",
            "https://localhost/rootfs",
            "https://127.0.0.1/rootfs",
            "https://[::1]/rootfs",
            "not-a-url",
            "",
        ] {
            assert!(media_base_url(bad).is_err(), "accepted {bad:?}");
        }
    }

    #[test]
    fn oracle_assembler_pin_requires_buildroot_selection() {
        let dir = std::env::temp_dir().join(format!("sri-pin-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        fs::create_dir_all(dir.join("config")).unwrap();
        fs::write(dir.join("config/build-args.conf"), b"OTHER=1\n").unwrap();
        assert_eq!(
            pin_assembler_build_args(dir.to_str().unwrap())
                .unwrap_err()
                .0,
            "missing upstream buildroot selection"
        );
        fs::write(
            dir.join("config/build-args.conf"),
            b"BUILDER_IMG=upstream\n",
        )
        .unwrap();
        pin_assembler_build_args(dir.to_str().unwrap()).unwrap();
        assert_eq!(
            fs::read(dir.join("config/build-args.conf")).unwrap(),
            b"BUILDER_IMG=oci-archive:/srv/tmp/assembler-root.oci\n"
        );
        let _ = fs::remove_dir_all(&dir);
    }
}

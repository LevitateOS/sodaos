//! Native Fountain extension packages: deterministic staging, sealing,
//! and catalog recording for the required `soda` and optional `tailnet`
//! packages.
//!
//! Each archive is a Fountain archive-contract directory tree, not OCI:
//! `extension.json`, the executable entrypoint/backend, `assets/`,
//! `LICENSE` and `NOTICE` at the archive root, without a wrapping
//! directory. The staged tree is sealed once into a deterministic USTAR
//! archive compressed with gzip level 1: normalized timestamp (the Soda
//! source commit time), owner/group 0, preserved executable modes, sorted
//! entries, no links, devices, dot-prefixed top-level entries, PAX/GNU
//! extensions, or unsafe paths. Fountain validates the archive and manifest
//! again before installing it.
//!
//! The two package identities feed the Payload v4 extension catalog, whose
//! projected bytes are also staged as the host's trusted local catalog.
//! Final Tailnet package *contents* land with T03 at I01; this module owns
//! the staging contract and fails closed until that source directory exists.

use std::collections::BTreeSet;
use std::fs;

use crate::error::Error;
use crate::model;
use crate::sys;

/// Fountain archive caps: 512 MiB compressed and extracted regular-file
/// bytes, 10,000 total entries, 255 UTF-8 bytes per path, 1 MiB manifest.
const MAX_ARCHIVE_BYTES: u64 = 512 << 20;
const MAX_ARCHIVE_ENTRIES: usize = 10_000;
const MAX_PATH_BYTES: usize = 255;
const MAX_MANIFEST_BYTES: u64 = 1 << 20;

/// Conventional Tailnet package source directory inside the Soda checkout.
/// T03 supplies its contents; until then staging refuses closed.
pub const TAILNET_PACKAGE_SOURCE: &str = "system/containers/extension/tailnet-package";

/// The two sealed native packages plus the projected catalog hash bound
/// into the host: the same bytes Fountain reads from its trusted catalog.
#[derive(Debug, Clone)]
pub struct NativePackages {
    pub soda: StagedPackage,
    pub tailnet: StagedPackage,
    pub catalog_sha256: String,
}

/// One sealed native package: archive bytes plus its catalog identity.
#[derive(Debug, Clone)]
pub struct StagedPackage {
    pub id: String,
    pub version: String,
    pub archive: Vec<u8>,
    pub size: u64,
    pub sha256: String,
    /// Filename of the release asset, without any URL prefix.
    pub filename: String,
}

/// Frozen R02 producer inputs for the sealed Payload v4 document beyond
/// the image bindings: release tag, admitted Fountain release, upstream
/// requests, optional Tailnet binding, and source timestamp.
#[derive(Debug, Clone, Default)]
pub struct ReleaseInputs {
    pub release_tag: String,
    pub package_base_url: String,
    pub fountain: model::FountainRelease,
    pub proxy_requested: String,
    pub postgres_requested: String,
    pub tailscale: model::PayloadImage,
    pub source_timestamp: u64,
}

impl ReleaseInputs {
    pub fn validate(&self) -> Result<(), Error> {
        if self.release_tag.is_empty() {
            return Err(Error::msg(
                "explicit release tag required for native package URLs",
            ));
        }
        if !soda_build_tools::reader::url::https_url(&self.package_base_url)
            || !self.package_base_url.ends_with('/')
        {
            return Err(Error::msg("HTTPS native package asset base required"));
        }
        if self.fountain.version.is_empty()
            || self.fountain.image.is_empty()
            || self.fountain.image_config.is_empty()
            || self.fountain.source_revision.is_empty()
            || self.fountain.extension_protocol != model::EXTENSION_PROTOCOL
            || self.fountain.browser_contract != model::BROWSER_CONTRACT
        {
            return Err(Error::msg(
                "admitted Fountain release required for Payload v4",
            ));
        }
        if self.proxy_requested.is_empty() || self.postgres_requested.is_empty() {
            return Err(Error::msg(
                "admitted upstream requests required for proxy and postgres",
            ));
        }
        if self.source_timestamp == 0 {
            return Err(Error::msg(
                "Soda source commit timestamp required for deterministic packages",
            ));
        }
        Ok(())
    }
}

/// Manifest shape the stager reads from each package's `extension.json`.
#[derive(Debug, Clone)]
struct PackageManifest {
    id: String,
    version: String,
    executable: String,
    capabilities: Vec<String>,
}

fn read_package_manifest(
    dir: &str,
    release_tag: &str,
) -> Result<(PackageManifest, Vec<u8>), Error> {
    let data = fs::read(sys::join(&[dir, "extension.json"]))?;
    if data.len() as u64 > MAX_MANIFEST_BYTES {
        return Err(Error::msg("oversized package manifest"));
    }
    let mut value: serde_json::Value =
        serde_json::from_slice(&data).map_err(|_| Error::msg("invalid package manifest"))?;
    let version = release_tag.strip_prefix('v').unwrap_or(release_tag);
    if version.is_empty()
        || version.len() > 64
        || !version.as_bytes()[0].is_ascii_alphanumeric()
        || !version
            .bytes()
            .all(|b| b.is_ascii_alphanumeric() || matches!(b, b'.' | b'_' | b'+' | b'-'))
    {
        return Err(Error::msg("invalid package release version"));
    }
    value
        .as_object_mut()
        .ok_or_else(|| Error::msg("package manifest must be an object"))?
        .insert(
            "version".to_string(),
            serde_json::Value::String(version.to_string()),
        );
    let data = serde_json::to_vec_pretty(&value).map_err(|e| Error::msg(e.to_string()))?;
    let string = |name: &str| {
        value
            .get(name)
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .to_string()
    };
    let capabilities = value
        .get("capabilities")
        .and_then(|v| v.as_array())
        .map(|items| {
            items
                .iter()
                .filter_map(|v| v.as_str().map(str::to_string))
                .collect::<Vec<_>>()
        })
        .unwrap_or_default();
    let manifest = PackageManifest {
        id: string("id"),
        version: string("version"),
        executable: string("executable"),
        capabilities,
    };
    if manifest.id.is_empty()
        || manifest.version.is_empty()
        || manifest.executable.is_empty()
        || manifest.capabilities.is_empty()
    {
        return Err(Error::msg("incomplete package manifest"));
    }
    Ok((manifest, data))
}

/// Collect one staged package tree: manifest, entrypoint/backend,
/// assets, LICENSE and NOTICE. Returns sorted (path, mode, bytes).
fn collect_package_files(
    staged: &str,
    manifest: &PackageManifest,
    manifest_bytes: &[u8],
    license_dir: &str,
) -> Result<Vec<(String, u32, Vec<u8>)>, Error> {
    let mut files = vec![("extension.json".to_string(), 0o644, manifest_bytes.to_vec())];
    let entrypoint = sys::join(&[staged, &manifest.executable]);
    let entry_meta = fs::symlink_metadata(&entrypoint)?;
    if !entry_meta.file_type().is_file() {
        return Err(Error::msg("package entrypoint must be a regular file"));
    }
    files.push((manifest.executable.clone(), 0o755, fs::read(&entrypoint)?));
    if manifest.executable != "backend" {
        let backend_path = sys::join(&[staged, "backend"]);
        let backend_meta = fs::symlink_metadata(&backend_path)?;
        if !backend_meta.file_type().is_file() {
            return Err(Error::msg("package backend must be a regular file"));
        }
        files.push(("backend".to_string(), 0o755, fs::read(&backend_path)?));
    }
    let assets = sys::join(&[staged, "assets"]);
    let mut stack = vec![assets.clone()];
    let mut asset_paths = Vec::new();
    while let Some(dir) = stack.pop() {
        let mut entries: Vec<_> = fs::read_dir(&dir)
            .map_err(|_| Error::msg("package assets must stage as a directory"))?
            .collect::<Result<_, _>>()
            .map_err(|e| Error::msg(e.to_string()))?;
        entries.sort_by_key(|e| e.file_name());
        for entry in entries {
            let file_type = entry.file_type().map_err(|e| Error::msg(e.to_string()))?;
            if file_type.is_dir() {
                stack.push(entry.path().to_string_lossy().into_owned());
            } else if file_type.is_file() {
                asset_paths.push(entry.path().to_string_lossy().into_owned());
            } else {
                return Err(Error::msg("package assets must be regular files"));
            }
        }
    }
    asset_paths.sort();
    for path in asset_paths {
        let rel = path
            .strip_prefix(&format!("{assets}/"))
            .ok_or_else(|| Error::msg("unsafe package asset path"))?;
        files.push((
            format!("assets/{}", rel.replace('\\', "/")),
            0o644,
            fs::read(&path)?,
        ));
    }
    for name in ["LICENSE", "NOTICE"] {
        let data = fs::read(sys::join(&[license_dir, name]))?;
        files.push((name.to_string(), 0o644, data));
    }
    files.sort_by(|a, b| a.0.cmp(&b.0));
    Ok(files)
}

fn check_entry_path(path: &str, top_level_dot: bool) -> Result<(), Error> {
    if path.is_empty()
        || path.len() > MAX_PATH_BYTES
        || path.starts_with('/')
        || path.contains('\\')
        || path
            .split('/')
            .any(|c| c.is_empty() || c == "." || c == "..")
    {
        return Err(Error::msg(format!("unsafe package path: {path}")));
    }
    if top_level_dot && path.split('/').next().is_some_and(|c| c.starts_with('.')) {
        return Err(Error::msg(format!(
            "dot-prefixed top-level package entry: {path}"
        )));
    }
    Ok(())
}

/// Seal sorted package entries using the upstream USTAR writer. Header path
/// limits and prefix encoding are checked by tar, never silently truncated.
fn check_package_entries(files: &[(String, u32, Vec<u8>)]) -> Result<BTreeSet<String>, Error> {
    if files.is_empty() || files.len() > MAX_ARCHIVE_ENTRIES {
        return Err(Error::msg("invalid package entries or timestamp"));
    }
    let mut dirs = BTreeSet::new();
    let mut names = BTreeSet::new();
    let mut extracted = 0u64;
    for (path, _, data) in files {
        check_entry_path(path, true)?;
        if !names.insert(path.as_str()) {
            return Err(Error::msg("duplicate package path"));
        }
        extracted = extracted
            .checked_add(data.len() as u64)
            .ok_or_else(|| Error::msg("package size overflow"))?;
        if extracted > MAX_ARCHIVE_BYTES {
            return Err(Error::msg("package exceeds Fountain archive cap"));
        }
        let mut parent = std::path::Path::new(path).parent();
        while let Some(dir) = parent.filter(|p| !p.as_os_str().is_empty()) {
            let dir = dir
                .to_str()
                .ok_or_else(|| Error::msg("invalid package path"))?;
            dirs.insert(dir.to_string());
            parent = std::path::Path::new(dir).parent();
        }
    }
    if files.len() + dirs.len() > MAX_ARCHIVE_ENTRIES {
        return Err(Error::msg("package exceeds Fountain entry cap"));
    }
    if dirs.iter().any(|dir| names.contains(dir.as_str())) {
        return Err(Error::msg("package file conflicts with directory"));
    }
    Ok(dirs)
}

fn append_package_entry(
    archive: &mut tar::Builder<flate2::write::GzEncoder<Vec<u8>>>,
    timestamp: u64,
    path: &str,
    mode: u32,
    data: &[u8],
    directory: bool,
) -> Result<(), Error> {
    let mut header = tar::Header::new_ustar();
    header
        .set_path(path)
        .map_err(|e| Error::msg(e.to_string()))?;
    header.set_entry_type(if directory {
        tar::EntryType::Directory
    } else {
        tar::EntryType::Regular
    });
    header.set_mode(if directory {
        0o755
    } else if mode & 0o111 != 0 {
        0o755
    } else {
        0o644
    });
    header.set_uid(0);
    header.set_gid(0);
    header.set_mtime(timestamp);
    header.set_size(data.len() as u64);
    header.set_cksum();
    archive
        .append(&header, data)
        .map_err(|e| Error::msg(e.to_string()))
}

pub fn seal_package(files: &[(String, u32, Vec<u8>)], timestamp: u64) -> Result<Vec<u8>, Error> {
    if files.is_empty() || files.len() > MAX_ARCHIVE_ENTRIES || timestamp == 0 {
        return Err(Error::msg("invalid package entries or timestamp"));
    }
    let dirs = check_package_entries(files)?;
    let encoder = flate2::write::GzEncoder::new(Vec::new(), flate2::Compression::new(1));
    let mut archive = tar::Builder::new(encoder);
    for dir in &dirs {
        append_package_entry(&mut archive, timestamp, dir, 0o755, &[], true)?;
    }
    let mut sorted: Vec<_> = files.iter().collect();
    sorted.sort_by(|a, b| a.0.cmp(&b.0));
    for (path, mode, data) in sorted {
        append_package_entry(&mut archive, timestamp, path, *mode, data, false)?;
    }
    let encoder = archive
        .into_inner()
        .map_err(|e| Error::msg(e.to_string()))?;
    let bytes = encoder.finish().map_err(|e| Error::msg(e.to_string()))?;
    if bytes.len() as u64 > MAX_ARCHIVE_BYTES {
        return Err(Error::msg("package exceeds Fountain archive cap"));
    }
    Ok(bytes)
}

/// Hard-link the browser build's emitted assets into the staged package.
fn link_extension_assets(native: &str, staged: &str, package_id: &str) -> Result<(), Error> {
    let root = sys::join(&[native, &format!("{package_id}-extension-assets"), "assets"]);
    sys::walk(&root, |path, is_dir, is_link| {
        if is_dir {
            return Ok(());
        }
        if is_link {
            return Err(Error::msg("regular extension asset required"));
        }
        let to = sys::join(&[staged, "assets", &sys::rel_path(&root, path)?]);
        fs::create_dir_all(sys::dir_name(&to))?;
        fs::hard_link(path, &to).map_err(|e| Error::msg(e.to_string()))
    })
}

fn finish_staged_package(
    id: &str,
    version: &str,
    archive: Vec<u8>,
) -> Result<StagedPackage, Error> {
    let size = archive.len() as u64;
    if size == 0 || size > MAX_ARCHIVE_BYTES {
        return Err(Error::msg(format!("invalid sealed package size for {id}")));
    }
    Ok(StagedPackage {
        id: id.to_string(),
        version: version.to_string(),
        sha256: sys::hex_sha256(&archive),
        size,
        filename: format!("{id}-extension-{version}-linux-x86_64.tar.gz"),
        archive,
    })
}

/// Stage the required `soda` package from Soda-owned sources: the
/// extension manifest and entrypoint, the compiled backend, the browser
/// build assets, and the release LICENSE/NOTICE. Staging lands under the
/// attempt output; the source tree is never written. The manifest's
/// capabilities must equal the approved required list exactly.
pub fn stage_soda_package(
    source: &str,
    context: &str,
    native: &str,
    out: &str,
    timestamp: u64,
    release_tag: &str,
) -> Result<StagedPackage, Error> {
    stage_package("soda", source, context, native, out, timestamp, release_tag)
}

pub fn stage_tailnet_package(
    source: &str,
    context: &str,
    native: &str,
    out: &str,
    timestamp: u64,
    release_tag: &str,
) -> Result<StagedPackage, Error> {
    stage_package(
        "tailnet",
        source,
        context,
        native,
        out,
        timestamp,
        release_tag,
    )
}

fn admit_package_manifest(
    package_id: &str,
    source: &str,
    release_tag: &str,
) -> Result<(String, PackageManifest, Vec<u8>), Error> {
    let relative = match package_id {
        "soda" => "system/containers/extension",
        "tailnet" => TAILNET_PACKAGE_SOURCE,
        _ => return Err(Error::msg("unknown extension package")),
    };
    let origin = sys::join(&[source, relative]);
    if package_id == "tailnet" && fs::symlink_metadata(&origin).is_err() {
        return Err(Error::msg("tailnet package contents pending T03: no system/containers/extension/tailnet-package source"));
    }
    let (manifest, manifest_bytes) = read_package_manifest(&origin, release_tag)?;
    if manifest.id != package_id {
        return Err(Error::msg("soda package manifest carries the wrong id"));
    }
    let approved: Vec<String> = (if package_id == "soda" {
        &model::SODA_CAPABILITIES[..]
    } else {
        &model::TAILNET_CAPABILITIES[..]
    })
    .iter()
    .map(|s| s.to_string())
    .collect();
    if manifest.capabilities != approved {
        return Err(Error::msg(
            "soda package capabilities differ from the approved catalog",
        ));
    }
    Ok((origin, manifest, manifest_bytes))
}

fn stage_package(
    package_id: &str,
    source: &str,
    context: &str,
    native: &str,
    out: &str,
    timestamp: u64,
    release_tag: &str,
) -> Result<StagedPackage, Error> {
    let (origin, manifest, manifest_bytes) =
        admit_package_manifest(package_id, source, release_tag)?;
    let backend = sys::join(&[context, "rootfs/usr/libexec/soda/soda-extension"]);
    let backend_meta = fs::symlink_metadata(&backend)?;
    if !backend_meta.file_type().is_file() {
        return Err(Error::msg("soda package backend must be a regular file"));
    }
    let staged = sys::join(&[out, &format!("packages/stage/{package_id}")]);
    fs::create_dir_all(sys::join(&[&staged, "assets"]))?;
    sys::write_new(
        &sys::join(&[&staged, "extension.json"]),
        &manifest_bytes,
        0o644,
    )?;
    let entrypoint = fs::read(sys::join(&[&origin, &manifest.executable]))?;
    sys::write_new(
        &sys::join(&[&staged, &manifest.executable]),
        &entrypoint,
        0o755,
    )?;
    fs::hard_link(&backend, sys::join(&[&staged, "backend"]))
        .map_err(|e| Error::msg(e.to_string()))?;
    link_extension_assets(native, &staged, package_id)?;
    let files = collect_package_files(&staged, &manifest, &manifest_bytes, source)?;
    let archive = seal_package(&files, timestamp)?;
    let package = finish_staged_package(&manifest.id, &manifest.version, archive)?;
    sys::write_new(
        &sys::join(&[out, "packages", &package.filename]),
        &package.archive,
        0o644,
    )?;
    Ok(package)
}

/// Record the two sealed packages as the Payload v4 extension catalog and
/// return the projected catalog bytes for the trusted local catalog file.
/// Package manifest version, catalog version string, filename, and receipt
/// agree through finish_staged_package; the seal-time payload validation
/// rechecks every catalog rule.
pub fn record_extension_catalog(
    payload: &mut model::Payload,
    package_base_url: &str,
    soda: &StagedPackage,
    tailnet: &StagedPackage,
) -> Result<Vec<u8>, Error> {
    if !soda_build_tools::reader::url::https_url(package_base_url)
        || !package_base_url.ends_with('/')
    {
        return Err(Error::msg("HTTPS native package asset base required"));
    }
    let url = |package: &StagedPackage| format!("{package_base_url}{}", package.filename);
    payload.extension_catalog = model::FountainCatalog {
        version: model::CATALOG_VERSION,
        packages: vec![
            model::FountainPackage {
                id: soda.id.clone(),
                version: soda.version.clone(),
                url: url(soda),
                size: soda.size,
                sha256: soda.sha256.clone(),
                required: true,
                fountain_protocol: model::EXTENSION_PROTOCOL,
                browser_contract: model::BROWSER_CONTRACT,
                capabilities: model::SODA_CAPABILITIES
                    .iter()
                    .map(|s| s.to_string())
                    .collect(),
                lifecycle_controller: None,
            },
            model::FountainPackage {
                id: tailnet.id.clone(),
                version: tailnet.version.clone(),
                url: url(tailnet),
                size: tailnet.size,
                sha256: tailnet.sha256.clone(),
                required: false,
                fountain_protocol: model::EXTENSION_PROTOCOL,
                browser_contract: model::BROWSER_CONTRACT,
                capabilities: model::TAILNET_CAPABILITIES
                    .iter()
                    .map(|s| s.to_string())
                    .collect(),
                lifecycle_controller: Some(model::SODA_PACKAGE_ID.to_string()),
            },
        ],
    };
    payload
        .projected_catalog_bytes()
        .map_err(|e| Error::msg(e.to_string()))
}

/// Bind one upstream core role (proxy/postgres) from its frozen identity:
/// the runtime reference names the exact manifest digest, and the
/// provenance retains the pinned requested reference. Upstream bindings
/// carry no local archive checksum.
pub fn record_upstream_binding(
    payload: &mut model::Payload,
    role: model::Role,
    config: &str,
    manifest: &str,
    requested: &str,
) -> Result<(), Error> {
    match role {
        model::Role::Proxy | model::Role::Postgres => {}
        _ => return Err(Error::msg("only proxy and postgres bind upstream")),
    }
    if !model::prefixed_digest(config) || !model::prefixed_digest(manifest) {
        return Err(Error::msg("upstream binding requires digest identities"));
    }
    let repository = requested.split('@').next().unwrap_or(requested);
    let repository = match repository.rfind(':') {
        Some(colon) if colon > repository.rfind('/').unwrap_or(0) => &repository[..colon],
        _ => repository,
    };
    if repository.is_empty()
        || repository.contains(char::is_whitespace)
        || repository.contains("://")
    {
        return Err(Error::msg("invalid admitted upstream repository"));
    }
    let reference = format!("{repository}@{manifest}");
    *role.binding_mut(&mut payload.images) = model::PayloadImage {
        reference,
        manifest: manifest.to_string(),
        config: config.to_string(),
        platform: "linux/amd64".to_string(),
        provenance: model::ImageProvenance::Upstream {
            requested: requested.to_string(),
        },
    };
    Ok(())
}

/// Bind the optional Tailnet image from its frozen official identity. The
/// package itself is cataloged separately; this records only the OCI
/// dependency under its fixed role.
pub fn record_tailscale_binding(
    payload: &mut model::Payload,
    tailscale: &model::PayloadImage,
) -> Result<(), Error> {
    if !model::prefixed_digest(&tailscale.manifest)
        || !model::prefixed_digest(&tailscale.config)
        || tailscale.platform != "linux/amd64"
    {
        return Err(Error::msg("invalid tailscale image binding"));
    }
    match tailscale.reference.rsplit_once('@') {
        Some((repo, digest)) if !repo.is_empty() && digest == tailscale.manifest => {}
        _ => return Err(Error::msg("invalid tailscale image binding")),
    }
    if !matches!(
        tailscale.provenance,
        model::ImageProvenance::Upstream { .. }
    ) {
        return Err(Error::msg("invalid tailscale image provenance"));
    }
    payload.optional_extensions.tailnet = model::TailnetDependency {
        package_id: model::TAILNET_PACKAGE_ID.to_string(),
        images: model::TailnetImages {
            tailscale: tailscale.clone(),
        },
    };
    Ok(())
}

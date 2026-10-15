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
use crate::extension;
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
pub fn seal_package(files: &[(String, u32, Vec<u8>)], timestamp: u64) -> Result<Vec<u8>, Error> {
    if files.is_empty() || files.len() > MAX_ARCHIVE_ENTRIES || timestamp == 0 {
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
    let encoder = flate2::write::GzEncoder::new(Vec::new(), flate2::Compression::new(1));
    let mut archive = tar::Builder::new(encoder);
    let mut append = |path: &str, mode: u32, data: &[u8], directory: bool| -> Result<(), Error> {
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
    };
    for dir in &dirs {
        append(dir, 0o755, &[], true)?;
    }
    let mut sorted: Vec<_> = files.iter().collect();
    sorted.sort_by(|a, b| a.0.cmp(&b.0));
    for (path, mode, data) in sorted {
        append(path, *mode, data, false)?;
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

fn finish_staged_package(
    id: &str,
    version: &str,
    archive: Vec<u8>,
    release_tag: &str,
) -> Result<StagedPackage, Error> {
    let version_tag = release_tag.strip_prefix('v').unwrap_or(release_tag);
    if version != version_tag {
        return Err(Error::msg(format!(
            "package manifest version differs from release tag for {id}"
        )));
    }
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

fn stage_package(
    package_id: &str,
    source: &str,
    context: &str,
    native: &str,
    out: &str,
    timestamp: u64,
    release_tag: &str,
) -> Result<StagedPackage, Error> {
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
    extension::stage_extension_assets(native, &staged, package_id)?;
    let files = collect_package_files(&staged, &manifest, &manifest_bytes, source)?;
    let archive = seal_package(&files, timestamp)?;
    let package = finish_staged_package(&manifest.id, &manifest.version, archive, release_tag)?;
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

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Read;

    fn gunzip(archive: &[u8]) -> Vec<u8> {
        let mut decoder = flate2::read::GzDecoder::new(archive);
        let mut tar = Vec::new();
        decoder.read_to_end(&mut tar).unwrap();
        tar
    }

    /// Minimal independent USTAR reader for the fixture: names, modes,
    /// mtimes, checksums, and the end-of-archive marker.
    fn read_tar(tar: &[u8]) -> Vec<(String, u32, u64, Vec<u8>)> {
        let mut entries = Vec::new();
        let mut offset = 0;
        loop {
            let header = &tar[offset..offset + 512];
            if header.iter().all(|b| *b == 0) {
                let next = &tar[offset + 512..offset + 1024];
                assert!(
                    next.iter().all(|b| *b == 0),
                    "two zero blocks end the archive"
                );
                break;
            }
            let name = String::from_utf8(
                header[..100]
                    .iter()
                    .take_while(|b| **b != 0)
                    .copied()
                    .collect(),
            )
            .unwrap();
            let field = |start: usize, len: usize| {
                u64::from_str_radix(
                    std::str::from_utf8(&header[start..start + len])
                        .unwrap()
                        .trim_matches(|c| c == ' ' || c == '\0'),
                    8,
                )
                .unwrap()
            };
            let mode = field(100, 8) as u32;
            let size = field(124, 12);
            let mtime = field(136, 12);
            assert_eq!(&header[257..263], b"ustar\0", "USTAR magic, no GNU/PAX");
            assert_eq!(&header[263..265], b"00");
            assert_eq!(field(108, 8), 0, "uid 0");
            assert_eq!(field(116, 8), 0, "gid 0");
            let stored = field(148, 8);
            let computed: u64 = header.iter().map(|b| *b as u64).sum::<u64>() + 8 * 32
                - header[148..156].iter().map(|b| *b as u64).sum::<u64>();
            assert_eq!(stored, computed, "header checksum");
            let body = tar[offset + 512..offset + 512 + size as usize].to_vec();
            entries.push((name, mode, mtime, body));
            offset += 512 + (size as usize).div_ceil(512) * 512;
        }
        entries
    }

    #[test]
    fn sealer_preserves_ustar_prefix_and_counts_directories() {
        let path = format!("assets/{}/{}.js", "d".repeat(80), "f".repeat(40));
        let bytes = seal_package(&[(path.clone(), 0o644, b"asset".to_vec())], 7).unwrap();
        let decoded = gunzip(&bytes);
        let mut archive = tar::Archive::new(decoded.as_slice());
        let names: Vec<_> = archive
            .entries()
            .unwrap()
            .map(|entry| {
                let entry = entry.unwrap();
                assert_eq!(entry.header().as_bytes()[257..263], *b"ustar\0");
                entry.path().unwrap().to_string_lossy().into_owned()
            })
            .collect();
        assert!(names.contains(&path));
        let files: Vec<_> = (0..5001)
            .map(|n| (format!("assets/d{n}/f"), 0o644, vec![1]))
            .collect();
        assert!(
            seal_package(&files, 7).is_err(),
            "directories count toward the entry cap"
        );
        let collision = vec![
            ("assets".to_string(), 0o644, vec![]),
            ("assets/app.js".to_string(), 0o644, vec![]),
        ];
        assert!(seal_package(&collision, 7).is_err());
    }

    fn fixture_files() -> Vec<(String, u32, Vec<u8>)> {
        vec![
            ("LICENSE".to_string(), 0o644, b"license".to_vec()),
            ("NOTICE".to_string(), 0o644, b"notice".to_vec()),
            ("assets/app.js".to_string(), 0o644, b"asset".to_vec()),
            ("backend".to_string(), 0o755, b"backend".to_vec()),
            ("extension.json".to_string(), 0o644, b"{}".to_vec()),
            ("run".to_string(), 0o755, b"#!/bin/sh\n".to_vec()),
        ]
    }

    #[test]
    fn sealed_archives_are_deterministic_gzip_ustar() {
        let first = seal_package(&fixture_files(), 1_700_000_000).unwrap();
        let second = seal_package(&fixture_files(), 1_700_000_000).unwrap();
        assert_eq!(first, second, "same inputs seal identical bytes");
        assert_eq!(&first[..2], b"\x1f\x8b", "gzip envelope");
        let third = seal_package(&fixture_files(), 1_700_000_001).unwrap();
        assert_ne!(first, third, "timestamp feeds the seal");
        let entries = read_tar(&gunzip(&first));
        let names: Vec<&str> = entries.iter().map(|(n, _, _, _)| n.as_str()).collect();
        assert_eq!(
            names,
            [
                "assets",
                "LICENSE",
                "NOTICE",
                "assets/app.js",
                "backend",
                "extension.json",
                "run"
            ],
            "canonical directories first, then sorted files"
        );
        for (name, mode, mtime, _) in &entries {
            assert_eq!(*mtime, 1_700_000_000, "{name}");
            if name == "assets" {
                assert_eq!(*mode, 0o755);
            } else if *name == "backend" || *name == "run" {
                assert_eq!(*mode, 0o755, "{name} keeps its executable mode");
            } else {
                assert_eq!(*mode, 0o644, "{name}");
            }
        }
    }

    #[test]
    fn sealer_refuses_unsafe_entries() {
        for bad in [
            ("/abs".to_string(), 0o644, b"x".to_vec()),
            ("../escape".to_string(), 0o644, b"x".to_vec()),
            ("a/../../escape".to_string(), 0o644, b"x".to_vec()),
            (".hidden".to_string(), 0o644, b"x".to_vec()),
            (".hidden/file".to_string(), 0o644, b"x".to_vec()),
            ("back\\slash".to_string(), 0o644, b"x".to_vec()),
            ("".to_string(), 0o644, b"x".to_vec()),
        ] {
            assert!(seal_package(&[bad.clone()], 7).is_err(), "{bad:?}");
        }
        let dup = vec![
            ("a".to_string(), 0o644, b"x".to_vec()),
            ("a".to_string(), 0o644, b"y".to_vec()),
        ];
        assert!(seal_package(&dup, 7).is_err(), "duplicate path");
        assert!(seal_package(&[], 7).is_err(), "empty package");
        assert!(seal_package(&fixture_files(), 0).is_err(), "zero timestamp");
    }

    fn staged(id: &str, version: &str, seed: u8) -> StagedPackage {
        let archive = seal_package(&fixture_files(), 1_700_000_000).unwrap();
        let mut package =
            finish_staged_package(id, version, archive, &format!("v{version}")).unwrap();
        package.archive[10] = package.archive[10].wrapping_add(seed);
        package.sha256 = sys::hex_sha256(&package.archive);
        package.size = package.archive.len() as u64;
        package
    }

    #[test]
    fn catalog_binds_two_packages_with_exact_urls() {
        let mut payload = model::Payload::default();
        let soda = staged("soda", "0.1.0", 0);
        let tailnet = staged("tailnet", "0.1.0", 1);
        let bytes = record_extension_catalog(
            &mut payload,
            "https://github.com/LevitateOS/sodaos/releases/download/v0.1.0/",
            &soda,
            &tailnet,
        )
        .unwrap();
        assert!(bytes.ends_with(b"\n"));
        assert_eq!(bytes, payload.projected_catalog_bytes().unwrap());
        assert_eq!(payload.extension_catalog.version, 1);
        let (first, second) = (
            &payload.extension_catalog.packages[0],
            &payload.extension_catalog.packages[1],
        );
        assert_eq!(first.id, "soda");
        assert!(first.required && first.lifecycle_controller.is_none());
        assert_eq!(
            first.url,
            format!(
                "https://github.com/LevitateOS/sodaos/releases/download/v0.1.0/{}",
                soda.filename
            )
        );
        assert_eq!(
            (first.size, first.sha256.clone()),
            (soda.size, soda.sha256.clone())
        );
        assert_eq!(second.id, "tailnet");
        assert!(!second.required);
        assert_eq!(second.lifecycle_controller.as_deref(), Some("soda"));
        assert_eq!(
            second.url,
            format!(
                "https://github.com/LevitateOS/sodaos/releases/download/v0.1.0/{}",
                tailnet.filename
            )
        );
        // Manifest version, catalog version, filename, and tag agree.
        assert!(soda.filename.contains("-0.1.0-") && tailnet.filename.contains("-0.1.0-"));
    }

    #[test]
    fn package_version_must_match_release_tag() {
        let archive = seal_package(&fixture_files(), 7).unwrap();
        assert!(finish_staged_package("soda", "0.2.0", archive, "v0.1.0").is_err());
    }

    #[test]
    fn tailnet_staging_waits_for_t03_contents() {
        let dir = std::env::temp_dir().join(format!("sri-t03-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        fs::create_dir_all(&dir).unwrap();
        assert_eq!(
            stage_tailnet_package(dir.to_str().unwrap(), "unused", "unused", dir.to_str().unwrap(), 7, "v0.1.0")
                .unwrap_err()
                .0,
            "tailnet package contents pending T03: no system/containers/extension/tailnet-package source"
        );
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn release_inputs_fail_closed() {
        let empty = ReleaseInputs::default();
        assert!(empty.validate().is_err());
        let mut full = ReleaseInputs {
            release_tag: "v0.1.0".to_string(),
            package_base_url: "https://fixtures.example.test/v0.1.0/".to_string(),
            fountain: model::FountainRelease {
                version: "1.0".to_string(),
                image: "example.invalid/fountain@sha256:".to_string() + &"a".repeat(64),
                image_config: format!("sha256:{}", "b".repeat(64)),
                source_revision: "c".repeat(40),
                extension_protocol: 1,
                browser_contract: 1,
                ..Default::default()
            },
            proxy_requested: "example.invalid/proxy@sha256:".to_string() + &"d".repeat(64),
            postgres_requested: "example.invalid/postgres@sha256:".to_string() + &"e".repeat(64),
            source_timestamp: 7,
            ..Default::default()
        };
        full.release_tag.clear();
        assert!(full.validate().is_err(), "tag");
        full.release_tag = "v0.1.0".to_string();
        full.source_timestamp = 0;
        assert!(full.validate().is_err(), "timestamp");
        full.source_timestamp = 7;
        full.fountain.extension_protocol = 2;
        assert!(full.validate().is_err(), "protocol");
    }

    #[test]
    fn upstream_and_tailscale_bindings_refuse_guesses() {
        let mut payload = model::Payload::default();
        let config = format!("sha256:{}", "a".repeat(64));
        let manifest = format!("sha256:{}", "b".repeat(64));
        let requested = format!("example.invalid/proxy@{manifest}");
        record_upstream_binding(
            &mut payload,
            model::Role::Proxy,
            &config,
            &manifest,
            &requested,
        )
        .unwrap();
        let bound = model::Role::Proxy.binding(&payload.images);
        assert_eq!(bound.reference, format!("example.invalid/proxy@{manifest}"));
        assert_eq!(bound.platform, "linux/amd64");
        assert!(record_upstream_binding(
            &mut payload,
            model::Role::Dashboard,
            &config,
            &manifest,
            &requested,
        )
        .is_err());
        let tailscale = model::PayloadImage {
            reference: format!("example.invalid/tailscale@{manifest}"),
            manifest: manifest.clone(),
            config: config.clone(),
            platform: "linux/amd64".to_string(),
            provenance: model::ImageProvenance::Upstream {
                requested: format!("example.invalid/tailscale@{manifest}"),
            },
        };
        record_tailscale_binding(&mut payload, &tailscale).unwrap();
        assert_eq!(payload.optional_extensions.tailnet.package_id, "tailnet");
        let mut soda_provenance = tailscale.clone();
        soda_provenance.provenance = model::ImageProvenance::Soda {
            revision: "r".to_string(),
            base: "b".to_string(),
        };
        assert!(record_tailscale_binding(&mut payload, &soda_provenance).is_err());
        let mut wrong_ref = tailscale.clone();
        wrong_ref.reference = "example.invalid/other@sha256:".to_string() + &"c".repeat(64);
        assert!(record_tailscale_binding(&mut payload, &wrong_ref).is_err());
    }
}

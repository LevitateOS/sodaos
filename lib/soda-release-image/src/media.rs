//! `assemble.go`: candidate-derived native media assembly.

use std::fs;

use soda_json::JsonValue;

use crate::compression;
use crate::error::Error;
use crate::files;
use crate::foreign::Production;
use crate::jsonio;
use crate::model;
use crate::request;
use crate::sys;

pub use crate::media_assembler::{
    builder_id, fetch_assembler_config, pin_assembler_build_args, prepare_assembler,
    verify_assembler_layers, wrap_assembler_image, ASSEMBLER_CONFIG_BRANCH, ASSEMBLER_IMAGE,
};
pub use crate::media_authentication::{
    authenticate_packaging_inputs, collect_media_inventory, sign_media_input,
    verify_media_inventory,
};
pub use crate::media_container::{
    assemble_native_media, build_media_container, stop_packaging_container, verify_build_meta,
    verify_meta_images, MediaMeta, MediaMetaImage,
};
pub use crate::media_installer::{
    customize_installer_iso, prepare_and_verify_media, setup_media_rootfs, verify_customized_iso,
    verify_media_readback,
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
mod tests;

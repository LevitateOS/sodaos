//! `build_media.go`: media tooling admission + live-input generation.

use std::fs;

use crate::error::Error;
use crate::foreign::Production;
use crate::ignition;
use crate::media;
use crate::model;
use crate::request;
use crate::sys;

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MediaTools {
    pub butane: String,
    pub architecture: String,
}

/// butaneImage floats on the upstream tag; the observed version is recorded
/// per build. No pinned digest or version string precedes the pull.
pub const BUTANE_IMAGE: &str = "quay.io/coreos/butane:latest";

pub fn prepare_build_media(
    production: &dyn Production,
    request: &request::Request,
) -> Result<(MediaTools, media::MediaLock), Error> {
    if !request.wants_media() {
        return Ok((MediaTools::default(), media::MediaLock::default()));
    }
    let tools = admit_media_tools(
        production.source(),
        &sys::join(&[&request.out, "evidence"]),
        production,
    )?;
    let root = sys::join(&[&request.out, "work/media"]);
    let mut lock = media::prepare_assembler(production, &root)?;
    let id = media::builder_id(&root)?;
    lock.installer = media::observe_installer_version(production, production.source(), &id)?;
    Ok((tools, lock))
}

pub fn finish_build_media(
    production: &dyn Production,
    request: &request::Request,
    tools: &MediaTools,
    lock: &mut media::MediaLock,
    next: &mut dyn FnMut(&str) -> Result<(), Error>,
) -> Result<(), Error> {
    if !request.wants_media() {
        return Ok(());
    }
    media::validate_installer_version(&lock.installer)?;
    prepare_media_inputs(production.source(), production.out(), tools, production)?;
    media::assemble_media(production, request, lock, next)?;
    Ok(())
}

pub fn verify_butane_image(
    source: &str,
    evidence: &str,
    production: &dyn Production,
) -> Result<String, Error> {
    production.execute(
        source,
        "podman",
        &[
            "--remote=false".to_string(),
            "pull".to_string(),
            BUTANE_IMAGE.to_string(),
        ],
    )?;
    let platform = model::oci_architecture(production.arch()).unwrap_or("");
    let observed = production.capture(
        source,
        "podman",
        &[
            "--remote=false".to_string(),
            "image".to_string(),
            "inspect".to_string(),
            "--format".to_string(),
            "{{.Os}}/{{.Architecture}}".to_string(),
            BUTANE_IMAGE.to_string(),
        ],
    )?;
    if observed != format!("linux/{platform}") {
        return Err(Error::msg("native Butane platform mismatch"));
    }
    let version = production.capture(
        source,
        "podman",
        &[
            "--remote=false".to_string(),
            "run".to_string(),
            "--pull=never".to_string(),
            "--cidfile".to_string(),
            sys::join(&[evidence, "butane-version.cid"]),
            "--network=none".to_string(),
            "--read-only".to_string(),
            "--cap-drop=all".to_string(),
            "--security-opt=no-new-privileges".to_string(),
            BUTANE_IMAGE.to_string(),
            "--version".to_string(),
        ],
    )?;
    let version = version.trim().to_string();
    if !version.starts_with("Butane v") || version.chars().any(|c| c == '\r' || c == '\n') {
        return Err(Error::msg("unrecognized Butane version output"));
    }
    Ok(version)
}

pub fn admit_media_tools(
    source: &str,
    evidence: &str,
    production: &dyn Production,
) -> Result<MediaTools, Error> {
    let tools = MediaTools {
        butane: BUTANE_IMAGE.to_string(),
        architecture: production.arch().to_string(),
    };
    let version = verify_butane_image(source, evidence, production)?;
    sys::write_new(
        &sys::join(&[evidence, "butane-version.txt"]),
        format!("{version}\n").as_bytes(),
        0o600,
    )?;
    Ok(tools)
}

/// Generate the live handoff in the same source-to-candidate run, using the
/// candidate host's installer version and once-compiled console. This is
/// public build output, not protected media admission, ISO assembly or
/// installation authority.
pub fn prepare_media_inputs(
    source: &str,
    out: &str,
    tools: &MediaTools,
    production: &dyn Production,
) -> Result<(), Error> {
    let destination = production.capture(
        source,
        "podman",
        &[
            "--remote=false".to_string(),
            "run".to_string(),
            "--pull=never".to_string(),
            "--cidfile".to_string(),
            sys::join(&[out, "destination-convert.cid"]),
            "--network=none".to_string(),
            "--read-only".to_string(),
            "--cap-drop=all".to_string(),
            "--security-opt=label=disable".to_string(),
            format!(
                "--volume={}:{}",
                sys::join(&[source, "system/host/provisioning/candidate.json"]),
                "/input.json:ro"
            ),
            tools.butane.clone(),
            "--strict".to_string(),
            "/input.json".to_string(),
        ],
    )?;
    let candidate_text = sys::read_json_build_text(&sys::join(&[out, "candidate.json"]))?;
    let candidate = model::Candidate::parse(&candidate_text)?;
    if !model::prefixed_digest(&candidate.host.config) {
        return Err(Error::msg("candidate host image configuration required"));
    }
    let installer_version =
        media::observe_installer_version(production, source, &candidate.host.config)?;
    let payload = fs::read(sys::join(&[out, "payload.json"]))?;
    let console = sys::hash_file(&sys::join(&[out, "tools/soda-installer"]))?;
    let live = ignition::candidate_live_config(
        &payload,
        destination.as_bytes(),
        &candidate.host.manifest,
        &console,
        &installer_version,
    )?;
    sys::write_new(
        &sys::join(&[out, "destination.ign"]),
        format!("{destination}\n").as_bytes(),
        0o644,
    )?;
    let mut live_bytes = live;
    live_bytes.push(b'\n');
    sys::write_new(&sys::join(&[out, "live.ign"]), &live_bytes, 0o644)
}

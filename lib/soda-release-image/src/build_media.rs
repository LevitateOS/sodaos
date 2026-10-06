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
    production.next("P2 / Verify native media tooling")?;
    let tools = admit_media_tools(
        production.source(),
        &sys::join(&[&request.out, "evidence"]),
        production,
    )?;
    let lock = media::prepare_assembler(production, &sys::join(&[&request.out, "work/media"]))?;
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
    production.next("P6 / Prepare candidate live Ignition")?;
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
/// exact candidate and once-compiled console. This is public build output,
/// not protected media admission, ISO assembly or installation authority.
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
    let candidate_value = sys::read_json_build(&sys::join(&[out, "candidate.json"]))?;
    let candidate = model::Candidate::parse(&candidate_value)?;
    let payload = fs::read(sys::join(&[out, "payload.json"]))?;
    let console = sys::hash_file(&sys::join(&[out, "tools/soda-installer"]))?;
    let live = ignition::candidate_live_config(
        &payload,
        destination.as_bytes(),
        &candidate.host.manifest,
        &console,
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_media_boundary_skips_without_target() {
        // Oracle: Go TestCandidateBoundaryDoesNotAdmitOrDispatchMedia (half).
        struct Stub;
        impl Production for Stub {
            fn source(&self) -> &str {
                "/src"
            }
            fn forgejo_source(&self) -> &str {
                ""
            }
            fn forgejo_revision(&self) -> &str {
                ""
            }
            fn native(&self) -> &str {
                ""
            }
            fn out(&self) -> &str {
                "/out"
            }
            fn arch(&self) -> &str {
                "x86_64"
            }
            fn revision(&self) -> &str {
                ""
            }
            fn live_inputs(&self) -> &str {
                ""
            }
            fn execute(&self, _: &str, _: &str, _: &[String]) -> Result<(), Error> {
                panic!("dispatched")
            }
            fn capture(&self, _: &str, _: &str, _: &[String]) -> Result<String, Error> {
                panic!("dispatched")
            }
            fn next(&self, _: &str) -> Result<(), Error> {
                panic!("dispatched")
            }
            fn resolve_inputs(&mut self) -> Result<(), Error> {
                Ok(())
            }
            fn dependencies(&self) -> Result<(), Error> {
                Ok(())
            }
            fn compile(&self, _: &str, _: &str, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn compile_rust(&self, _: &str, _: &str, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn stage_fork_binary(&self, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn assets(&self, _: &str, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn images(
                &self,
                _: &str,
            ) -> Result<std::collections::HashMap<String, model::ProducedImage>, Error>
            {
                Ok(Default::default())
            }
            fn inspect_oci(&self, _: &str, _: &str, _: &str) -> Result<model::Image, Error> {
                Ok(Default::default())
            }
            fn verify_content(
                &self,
                _: &model::Payload,
                _: &str,
            ) -> Result<(std::collections::HashMap<String, String>, u64), Error> {
                Ok(Default::default())
            }
            fn resolve_core_os(&self) -> Result<model::ResolvedCoreOS, Error> {
                Ok(Default::default())
            }
            fn read_live_inputs(&self, _: &str) -> Result<model::LiveInputs, Error> {
                Ok(Default::default())
            }
            fn check_native(&self, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn sign_media(
                &self,
                _: &model::Trust,
                _: &model::Permit,
                _: &str,
                _: &str,
                _: &str,
                _: &model::SecretFiles,
                _: &str,
            ) -> Result<(), Error> {
                Ok(())
            }
            fn verify_copy(
                &self,
                _: &model::Trust,
                _: &str,
                _: &str,
                _: &str,
                _: &str,
            ) -> Result<(), Error> {
                Ok(())
            }
            fn write_document(&self, _: &str, _: &soda_json::JsonValue) -> Result<String, Error> {
                Ok(String::new())
            }
        }
        let stub = Stub;
        let candidate = request::Request {
            development: true,
            target: "candidate".to_string(),
            ..request::Request::default()
        };
        let (tools, lock) = prepare_build_media(&stub, &candidate).unwrap();
        assert_eq!(tools, MediaTools::default());
        assert_eq!(lock, media::MediaLock::default());
        finish_build_media(
            &stub,
            &candidate,
            &tools,
            &mut lock.clone(),
            &mut |_| Ok(()),
        )
        .unwrap();
    }
}

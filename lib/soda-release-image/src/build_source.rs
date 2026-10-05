//! Build input admission: clean-source, revision, compiler, and output checks.
use std::fs;

use crate::build::PINNED_GO_VERSION;
use crate::build_runner::Runner;
use crate::error::Error;
use crate::model;
use crate::request;
use crate::sys;

pub fn verify_checkout_source(requested_source: &str, runner: &Runner) -> Result<String, Error> {
    if !sys::is_abs(requested_source) || sys::clean_path(requested_source) != requested_source {
        return Err(Error::msg("explicit canonical checkout path required"));
    }
    let source = runner.capture(
        requested_source,
        "git",
        &[
            "-c".to_string(),
            format!("safe.directory={requested_source}"),
            "rev-parse".to_string(),
            "--show-toplevel".to_string(),
        ],
    )?;
    if source != requested_source || !sys::is_abs(&source) {
        return Err(Error::msg("canonical checkout root required"));
    }
    let info = fs::symlink_metadata(sys::join(&[&source, ".git"]));
    match info {
        Ok(info) if info.file_type().is_dir() => Ok(source),
        _ => Err(Error::msg("canonical checkout required; no worktree")),
    }
}

pub fn verify_committed_revision(
    source: &str,
    requested_revision: &str,
    runner: &Runner,
) -> Result<String, Error> {
    let status = runner.capture(
        source,
        "git",
        &[
            "-c".to_string(),
            format!("safe.directory={source}"),
            "status".to_string(),
            "--porcelain".to_string(),
            "--untracked-files=normal".to_string(),
        ],
    )?;
    if !status.is_empty() {
        return Err(Error::msg("clean committed source required"));
    }
    let revision = runner.capture(
        source,
        "git",
        &[
            "-c".to_string(),
            format!("safe.directory={source}"),
            "rev-parse".to_string(),
            "HEAD".to_string(),
        ],
    )?;
    if !model::is_revision(&revision) {
        return Err(Error::msg("exact committed revision required"));
    }
    if !requested_revision.is_empty() && requested_revision != revision {
        return Err(Error::msg("controller/source revision mismatch"));
    }
    Ok(revision)
}

pub fn verify_compiler(source: &str, runner: &Runner) -> Result<(), Error> {
    let version = runner.capture(source, "go", &["env".to_string(), "GOVERSION".to_string()])?;
    if version != PINNED_GO_VERSION {
        return Err(Error::msg(format!(
            "pinned Go {PINNED_GO_VERSION} compiler unavailable"
        )));
    }
    Ok(())
}

pub fn admit_build_output(
    out: &str,
    source: &str,
    wants_media: bool,
    authority: &str,
) -> Result<(), Error> {
    let below = sys::join(&[source, ".artifacts/releases"]);
    if !sys::is_abs(out) || !sys::clean_path(out).starts_with(&format!("{below}/")) {
        return Err(Error::msg(
            "fresh output must be below .artifacts/releases; parent must exist",
        ));
    }
    if wants_media && sys::private_file(authority).is_err() {
        return Err(Error::msg("restricted media authority file required"));
    }
    sys::fresh_directory(out)
}

pub fn admit_build_inputs(request: &request::Request, runner: &Runner) -> Result<String, Error> {
    request.validate_target()?;
    sys::require_native(&request.arch)?;
    if !model::valid_repository_prefix(&request.repository_prefix) {
        return Err(Error::msg("explicit intended repository prefix required"));
    }
    let source = verify_checkout_source(&request.source, runner)?;
    verify_compiler(&source, runner)?;
    let revision = verify_committed_revision(&source, &request.revision, runner)?;
    let fork = verify_checkout_source(&request.forgejo_source, runner)
        .map_err(|e| Error::msg(format!("forgejo source: {}", e.0)))?;
    verify_committed_revision(&fork, &request.forgejo_revision, runner)
        .map_err(|e| Error::msg(format!("forgejo source: {}", e.0)))?;
    admit_build_output(
        &request.out,
        &source,
        request.wants_media(),
        &request.media_authority,
    )?;
    Ok(revision)
}

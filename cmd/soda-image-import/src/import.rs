use std::collections::HashMap;
use std::path::Path;
use std::process::{Command, Stdio};
use std::sync::atomic::Ordering;
use std::time::{Duration, Instant};

use super::{inspect_oci_layout, ImageBinding, ImportCtx, OciImage, Payload, NAMES};

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
pub(super) fn verify_content(payload: &Payload, images: &str) -> Result<(), String> {
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
pub(super) enum PodmanOutcome {
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

pub(super) fn import_images(
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

pub(super) fn run_podman(podman: &str, args: &[String], ctx: &ImportCtx) -> PodmanOutcome {
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

pub(super) fn native_import(
    payload: &Payload,
    images_dir: &str,
    podman: &str,
    ctx: &ImportCtx,
) -> Result<(), String> {
    import_images(payload, images_dir, podman, ctx, &mut |cmd, args| {
        run_podman(cmd, args, ctx)
    })
}

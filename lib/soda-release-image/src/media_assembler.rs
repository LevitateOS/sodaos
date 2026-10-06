use std::fs;

use crate::error::Error;
use crate::foreign::Production;
use crate::media::{MediaLock, RunFn};
use crate::model;
use crate::sys;

/// assemblerImage and assemblerConfigBranch float on upstream: the latest
/// assembler release and the stable config branch matching the stable base.
/// (Upstream publishes no stable assembler tag.) The resolved digest and
/// fetched revision are recorded per build in media.json; no pinned digest,
/// revision or installer version precedes the run.
pub const ASSEMBLER_IMAGE: &str = "quay.io/coreos-assembler/coreos-assembler:latest";
pub const ASSEMBLER_CONFIG_BRANCH: &str = "stable";

pub fn fetch_assembler_config(
    production: &dyn Production,
    run: RunFn<'_>,
    root: &str,
) -> Result<String, Error> {
    for args in [
        vec!["init".to_string(), "config-repo".to_string()],
        vec![
            "-C".to_string(),
            "config-repo".to_string(),
            "fetch".to_string(),
            "--depth=1".to_string(),
            "https://github.com/coreos/fedora-coreos-config.git".to_string(),
            ASSEMBLER_CONFIG_BRANCH.to_string(),
        ],
    ] {
        run("git", &args)?;
    }
    let sha = production.capture(
        root,
        "git",
        &[
            "-C".to_string(),
            "config-repo".to_string(),
            "rev-parse".to_string(),
            "FETCH_HEAD".to_string(),
        ],
    )?;
    let sha = sha.trim().to_string();
    if !model::is_revision(&sha) {
        return Err(Error::msg("unresolved assembler config revision"));
    }
    run(
        "git",
        &[
            "-C".to_string(),
            "config-repo".to_string(),
            "archive".to_string(),
            "--format=tar".to_string(),
            "--output".to_string(),
            sys::join(&[root, "config.tar"]),
            sha.clone(),
        ],
    )?;
    let config = sys::join(&[root, "config"]);
    sys::create_dir(&config, 0o755)?;
    run(
        "tar",
        &[
            "-xf".to_string(),
            "config.tar".to_string(),
            "-C".to_string(),
            config,
            "--no-same-owner".to_string(),
        ],
    )?;
    Ok(sha)
}

pub fn pin_assembler_build_args(root: &str) -> Result<(), Error> {
    let args_file = sys::join(&[root, "config", "build-args.conf"]);
    let data = fs::read(&args_file)?;
    let mut lines: Vec<String> = String::from_utf8_lossy(&data)
        .split('\n')
        .map(|s| s.to_string())
        .collect();
    let mut found = false;
    for line in lines.iter_mut() {
        if line.starts_with("BUILDER_IMG=") {
            *line = "BUILDER_IMG=oci-archive:/srv/tmp/assembler-root.oci".to_string();
            found = true;
        }
    }
    if !found {
        return Err(Error::msg("missing upstream buildroot selection"));
    }
    fs::write(&args_file, lines.join("\n").as_bytes())?;
    Ok(())
}

pub fn verify_assembler_layers(
    production: &dyn Production,
    root: &str,
    assembler: &str,
    id: &str,
) -> Result<(), Error> {
    let original = production.capture(
        root,
        "podman",
        &[
            "--remote=false".to_string(),
            "image".to_string(),
            "inspect".to_string(),
            "--format".to_string(),
            "{{json .RootFS.Layers}}".to_string(),
            assembler.to_string(),
        ],
    )?;
    let wrapped = production.capture(
        root,
        "podman",
        &[
            "--remote=false".to_string(),
            "image".to_string(),
            "inspect".to_string(),
            "--format".to_string(),
            "{{json .RootFS.Layers}}".to_string(),
            id.to_string(),
        ],
    )?;
    if original != wrapped || original.is_empty() {
        return Err(Error::msg("assembler wrapper changed rootfs"));
    }
    Ok(())
}

pub fn wrap_assembler_image(
    production: &dyn Production,
    run: RunFn<'_>,
    root: &str,
) -> Result<String, Error> {
    // No --policy=missing: the tag floats, so every build re-resolves it.
    run(
        "podman",
        &[
            "--remote=false".to_string(),
            "pull".to_string(),
            ASSEMBLER_IMAGE.to_string(),
        ],
    )?;
    let digest = production.capture(
        root,
        "podman",
        &[
            "--remote=false".to_string(),
            "image".to_string(),
            "inspect".to_string(),
            "--format".to_string(),
            "{{.Digest}}".to_string(),
            ASSEMBLER_IMAGE.to_string(),
        ],
    )?;
    let digest = digest.trim().to_string();
    const PREFIX: &str = "quay.io/coreos-assembler/coreos-assembler@sha256:";
    let hex = match digest.strip_prefix("sha256:") {
        Some(hex) if model::is_digest(hex) => hex,
        _ => return Err(Error::msg("unresolved assembler digest")),
    };
    let reference = format!("{PREFIX}{hex}");
    sys::write_new(
        &sys::join(&[root, "Containerfile"]),
        format!("FROM {reference}\nUSER 0\n").as_bytes(),
        0o644,
    )?;
    run(
        "podman",
        &[
            "--remote=false".to_string(),
            "build".to_string(),
            "--pull=never".to_string(),
            "--network=none".to_string(),
            "--iidfile".to_string(),
            "builder.iid".to_string(),
            ".".to_string(),
        ],
    )?;
    let id = builder_id(root)?;
    verify_assembler_layers(production, root, &reference, &id)?;
    run(
        "podman",
        &[
            "--remote=false".to_string(),
            "save".to_string(),
            "--format=oci-archive".to_string(),
            "--output".to_string(),
            "assembler-root.oci".to_string(),
            id,
        ],
    )?;
    Ok(reference)
}

pub fn prepare_assembler(production: &dyn Production, root: &str) -> Result<MediaLock, Error> {
    let mut lock = MediaLock {
        architecture: production.arch().to_string(),
        ..MediaLock::default()
    };
    sys::create_dir(root, 0o700)?;
    let run =
        |cmd: &str, args: &[String]| -> Result<(), Error> { production.execute(root, cmd, args) };
    let sha = fetch_assembler_config(production, &run, root)?;
    lock.config = sha;
    pin_assembler_build_args(root)?;
    let digest = wrap_assembler_image(production, &run, root)?;
    lock.assembler = digest;
    Ok(lock)
}

pub fn builder_id(root: &str) -> Result<String, Error> {
    let data = fs::read(sys::join(&[root, "builder.iid"]));
    match data {
        Ok(data) => {
            let id = String::from_utf8_lossy(&data).trim().to_string();
            if model::prefixed_digest(&id) {
                return Ok(id);
            }
            Err(Error::msg("missing exact assembler image"))
        }
        Err(_) => Err(Error::msg("missing exact assembler image")),
    }
}

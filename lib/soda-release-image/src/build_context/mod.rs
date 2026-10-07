//! Host build-context preparation: freeze the selected base-image
//! config, prepare the host context, and link prepared frontend
//! assets for the Forgejo stage.

use std::fs;

use crate::compression;
use crate::error::Error;
use crate::files;
use crate::foreign::Production;
use crate::model;
use crate::prepare;
use crate::sys;

pub fn freeze_base_image_config(
    snapshot: &str,
    out: &str,
    context_dir: &str,
    arch: &str,
    prefix: &str,
    compression: &str,
    base: &prepare::Base,
    production: &dyn Production,
) -> Result<(), Error> {
    let platform = model::oci_architecture(arch).unwrap_or("");
    let pinned = base.image(arch);
    production.execute(
        snapshot,
        "podman",
        &[
            "--remote=false".to_string(),
            "pull".to_string(),
            format!("--platform=linux/{platform}"),
            pinned.clone(),
        ],
    )?;
    let metadata = production.capture(
        snapshot,
        "podman",
        &[
            "--remote=false".to_string(),
            "run".to_string(),
            "--cidfile".to_string(),
            sys::join(&[out, "evidence/base-config.cid"]),
            "--network=none".to_string(),
            "--read-only".to_string(),
            "--cap-drop=all".to_string(),
            "--security-opt=no-new-privileges".to_string(),
            "--entrypoint=/usr/bin/cat".to_string(),
            pinned,
            "/usr/share/coreos-assembler/image.json".to_string(),
        ],
    )?;
    let image_data = freeze_image_config(&metadata, prefix, compression)?;
    files::owned_write(
        &sys::join(&[context_dir, "rootfs/usr/share/coreos-assembler/image.json"]),
        image_data.as_bytes(),
        0o644,
    )
}

fn freeze_image_config(metadata: &str, prefix: &str, mode: &str) -> Result<String, Error> {
    let mut image_config = compression::ImageConfig::parse(metadata)?;
    if !image_config.is_nonempty_object() {
        return Err(Error::msg("missing upstream image configuration"));
    }
    if let Some(entries) = image_config.ordered_mut().object_mut() {
        entries.retain(|(key, _)| key != "container-imgref" && key != "bootc-install-to-fs");
        entries.push((
            "container-imgref".to_string(),
            crate::ordered_json::OrderedValue::String(format!(
                "ostree-image-signed:docker://{prefix}-host:candidate"
            )),
        ));
        entries.push((
            "bootc-install-to-fs".to_string(),
            crate::ordered_json::OrderedValue::Bool(false),
        ));
        entries.sort_by(|a, b| a.0.cmp(&b.0));
    }
    compression::set_media_compression(&mut image_config, mode)?;
    let mut output = image_config.to_pretty_json();
    output.push('\n');
    Ok(output)
}

pub fn prepare_build_host_context(
    snapshot: &str,
    out: &str,
    arch: &str,
    revision: &str,
    prefix: &str,
    compression: &str,
    live_inputs: &str,
    production: &mut dyn Production,
) -> Result<(String, prepare::Base), Error> {
    let context_dir = sys::join(&[out, "work/host-context"]);
    let base = if !live_inputs.is_empty() {
        prepare::prepare_resolved(
            production,
            snapshot,
            &context_dir,
            arch,
            revision,
            live_inputs,
        )?
    } else {
        prepare::prepare(production, snapshot, &context_dir, arch, revision)?
    };
    production.resolve_inputs()?;
    freeze_base_image_config(
        snapshot,
        out,
        &context_dir,
        arch,
        prefix,
        compression,
        &base,
        production,
    )?;
    Ok((context_dir, base))
}

/// linkPreparedAssets points the snapshot at the already-built frontend
/// outputs the Forgejo stage consumes. The language and presentation suites
/// stay runnable on their own (go test, bun run typecheck/test:*, unittest)
/// but never gate a development build: an ISO to test today must not fail
/// on unrelated suites.
pub fn link_prepared_assets(production: &dyn Production) -> Result<(), Error> {
    let mut pairs = [
        (
            sys::join(&[production.source(), ".artifacts/forgejo-js"]),
            sys::join(&[production.native(), "forgejo-js"]),
        ),
        (
            sys::join(&[production.source(), ".artifacts/browser-terminal/vendor"]),
            sys::join(&[production.native(), "terminal-assets"]),
        ),
    ];
    pairs.sort();
    for (link, target) in pairs {
        fs::create_dir_all(sys::dir_name(&link))?;
        std::os::unix::fs::symlink(&target, &link).map_err(|e| Error::msg(e.to_string()))?;
    }
    Ok(())
}

#[cfg(test)]
mod tests;

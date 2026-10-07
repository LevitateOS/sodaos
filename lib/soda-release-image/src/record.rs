//! `build_record.go`: candidate content inventory + build receipts.

use std::fs;

use crate::error::Error;
use crate::files;
use crate::foreign::Production;
use crate::jsonio;
use crate::model;
use crate::request;
use crate::sys;

/// These are the actual bytes staged into the host and component images. The
/// archive identities in the payload bind the images; this inventory names
/// the fork, independent package, and host service inputs within those images.
pub fn candidate_content(out: &str) -> Result<Vec<(String, String)>, Error> {
    let root = sys::dir_name(out);
    let mut paths: Vec<(String, String)> = [
        (
            "dashboard:/usr/local/bin/soda-dashboard",
            sys::join(&[
                &root,
                "work/host-context/rootfs/usr/libexec/soda/soda-dashboard",
            ]),
        ),
        (
            "forgejo:/usr/local/bin/gitea",
            sys::join(&[out, "forgejo-context/forgejo-bin"]),
        ),
        (
            "extension:/usr/local/bin/gitea",
            sys::join(&[out, "forgejo-context/forgejo-bin"]),
        ),
        (
            "extension:/usr/share/soda/extension/extension.json",
            sys::join(&[out, "extension-context/extension/extension.json"]),
        ),
        (
            "extension:/usr/share/soda/extension/backend",
            sys::join(&[out, "extension-context/extension/backend"]),
        ),
        (
            "extension:/usr/share/soda/extension/run",
            sys::join(&[out, "extension-context/extension/run"]),
        ),
        (
            "host:/usr/share/containers/systemd/forgejo.container",
            sys::join(&[
                &root,
                "work/host-context/rootfs/usr/share/containers/systemd/forgejo.container",
            ]),
        ),
        (
            "host:/usr/share/containers/systemd/soda-dashboard.container",
            sys::join(&[
                &root,
                "work/host-context/rootfs/usr/share/containers/systemd/soda-dashboard.container",
            ]),
        ),
        (
            "host:/usr/lib/systemd/system/soda-extension-install.service",
            sys::join(&[
                &root,
                "work/host-context/rootfs/usr/lib/systemd/system/soda-extension-install.service",
            ]),
        ),
    ]
    .into_iter()
    .map(|(a, b)| (a.to_string(), b))
    .collect();
    let assets = sys::join(&[out, "extension-context/extension/assets"]);
    sys::walk(&assets, |path, is_dir, _| {
        if is_dir {
            return Ok(());
        }
        let info = fs::symlink_metadata(path)?;
        if !info.file_type().is_file() {
            return Err(Error::msg("extension asset must be regular"));
        }
        let rel = sys::rel_path(&assets, path)?;
        paths.push((
            format!(
                "extension:/usr/share/soda/extension/assets/{}",
                sys::to_slash(&rel)
            ),
            path.to_string(),
        ));
        Ok(())
    })?;
    let mut files: Vec<(String, String)> = Vec::new();
    let mut sorted = paths;
    sorted.sort_by(|a, b| a.0.cmp(&b.0));
    for (name, path) in sorted {
        let hash = sys::hash_file(&path)?;
        files.push((name, hash));
    }
    if !model::valid_candidate_content(&files) {
        return Err(Error::msg("incomplete candidate content inventory"));
    }
    Ok(files)
}

pub fn content_inventory_bytes(files: &[(String, String)]) -> Result<Vec<u8>, Error> {
    let mut sorted = files.to_vec();
    sorted.sort_by(|a, b| a.0.cmp(&b.0));
    let mut data = jsonio::to_indent(&jsonio::SortedPairs(&sorted));
    data.push('\n');
    Ok(data.into_bytes())
}

pub fn write_content_inventory(context: &str, out: &str) -> Result<(), Error> {
    let files = candidate_content(out)?;
    let data = content_inventory_bytes(&files)?;
    files::owned_write(
        &sys::join(&[context, "rootfs/usr/share/soda/host-image/content.json"]),
        &data,
        0o644,
    )
}

/// Called only after the fixed candidate artifact checks and requested media
/// checks succeed. The candidate hash binds all app identities through
/// its payload hash; this producer receipt is never protected qualification
/// evidence. Language and presentation suites are not build gates, so they
/// are not listed here; run them directly when they matter.
pub fn record_build_result(
    production: &dyn Production,
    request: &request::Request,
) -> Result<request::Result, Error> {
    let mut result = request::Result {
        revision: production.revision().to_string(),
        architecture: production.arch().to_string(),
        candidate: sys::join(&[production.out(), "candidate.json"]),
        purpose: request.purpose(),
        requested_target: request.requested_target(),
        completed_target: "candidate".to_string(),
        scope: "P1-P6 verified candidate; not a qualified release".to_string(),
        checks: [
            "ELF architecture",
            "Application OCI identities",
            "Host identity, RPM inventory, shared layout and Quadlets",
            "Host OCI export",
        ]
        .into_iter()
        .map(|s| s.to_string())
        .collect(),
        ..request::Result::default()
    };
    let text = sys::read_json_build_text(&result.candidate)?;
    let candidate = model::Candidate::parse(&text)?;
    result.host_manifest = candidate.host.manifest.clone();
    result.payload_sha256 = candidate.payload_sha256.clone();
    result.candidate_sha256 = sys::hash_file(&result.candidate)?;
    if request.wants_media() {
        result.media_compression = request.media_compression.clone();
        result.media = sys::join(&[production.out(), "media/media.json"]);
        result.completed_target = "media".to_string();
        result.scope = "P1-P8 candidate-derived media; not a qualified release".to_string();
        result.checks.extend(
            [
                "Authenticated packaging inputs",
                "Native media identity, Ignition, kernel arguments and rootfs chunks",
            ]
            .into_iter()
            .map(|s| s.to_string()),
        );
    }
    if request.development {
        result.scope = "development-only; not release-qualified".to_string();
    }
    let mut data = jsonio::to_indent(&result);
    data.push('\n');
    sys::write_new(
        &sys::join(&[&request.out, "evidence/build.json"]),
        data.as_bytes(),
        0o600,
    )?;
    Ok(result)
}

/// The final host manifest cannot be embedded in itself. This detached local
/// receipt binds that manifest to the exact embedded payload bytes. It is NOT
/// a signed release/channel authority or a qualified upgrade declaration.
pub fn record_candidate(
    out: &str,
    prefix: &str,
    host: &model::Image,
    archive_hash: &str,
    forgejo_revision: &str,
    arch: &str,
) -> Result<(), Error> {
    let mut record = model::Candidate {
        format: 1,
        host: host.clone(),
        host_reference: format!("{prefix}-host@{}", host.manifest),
        host_archive_sha256: archive_hash.to_string(),
        forgejo_revision: forgejo_revision.to_string(),
        architecture: arch.to_string(),
        migration: "No upgrade or writable-install migration is qualified. Conflicting saved image selections are refused; machine settings/secrets require explicit first-install setup.".to_string(),
        notes: "Complete unsigned local appliance payload only. All app content is embedded in one shared OCI layout for ordinary Podman import; repository names are intended, not provisioned. No production, native boot, signature or recovery acceptance.".to_string(),
        ..model::Candidate::default()
    };
    record_candidate_inputs(out, &mut record)?;
    let mut data = jsonio::to_indent(&record);
    data.push('\n');
    sys::write_new(&sys::join(&[out, "candidate.json"]), data.as_bytes(), 0o600)
}

pub fn record_candidate_inputs(out: &str, record: &mut model::Candidate) -> Result<(), Error> {
    if !model::is_revision(&record.forgejo_revision) {
        return Err(Error::msg("exact Forgejo source revision required"));
    }
    let platform = model::oci_architecture(&record.architecture);
    match platform {
        Ok(platform) if record.host.architecture == platform => {}
        _ => {
            return Err(Error::msg(
                "native candidate architecture differs from host image",
            ))
        }
    }
    record.forgejo_source_sha256 = sys::hash_file(&sys::join(&[out, "forgejo-source.tar"]))?;
    let toolchain_text = sys::read_json_build_text(&sys::join(&[out, "forgejo-toolchain.json"]))?;
    record.forgejo_toolchain = model::ForgejoToolchain::parse(&toolchain_text)?;
    record.forgejo_toolchain.validate()?;
    record.content_sha256 = candidate_content(out)?;
    verify_embedded_content_inventory(out, &record.content_sha256)?;
    record.payload_sha256 = sys::hash_file(&sys::join(&[out, "payload.json"]))?;
    Ok(())
}

pub fn verify_embedded_content_inventory(
    out: &str,
    content: &[(String, String)],
) -> Result<(), Error> {
    let data = content_inventory_bytes(content)?;
    let embedded = fs::read(sys::join(&[
        &sys::dir_name(out),
        "work/host-context/rootfs/usr/share/soda/host-image/content.json",
    ]));
    match embedded {
        Ok(embedded) if embedded == data => Ok(()),
        _ => Err(Error::msg(
            "candidate content inventory differs from host image input",
        )),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_candidate_inputs_require_exact_fork_revision() {
        // Oracle: Go recordCandidateInputs revision gate.
        let mut record = model::Candidate {
            forgejo_revision: "short".to_string(),
            ..model::Candidate::default()
        };
        assert_eq!(
            record_candidate_inputs("/nonexistent", &mut record)
                .unwrap_err()
                .0,
            "exact Forgejo source revision required"
        );
    }
}

//! `build_payload.go`: application image staging + payload sealing.

use std::collections::HashMap;
use std::fs;

use crate::complete;
use crate::error::Error;
use crate::extension;
use crate::foreign::Production;
use crate::jsonio;
use crate::model;
use crate::sys;

pub fn link_candidate_commands(source: &str, context: &str, native: &str) -> Result<(), Error> {
    fs::create_dir_all(sys::join(&[native, "bin"]))?;
    let commands = sys::soda_commands(source)?;
    // Reuse the already compiled vendor binaries; never compile again for assets.
    for name in commands {
        fs::hard_link(
            sys::join(&[context, "rootfs/usr/libexec/soda", &name]),
            sys::join(&[native, "bin", &name]),
        )
        .map_err(|e| Error::msg(e.to_string()))?;
    }
    Ok(())
}

pub fn stage_candidate_forgejo(
    payload: &mut model::Payload,
    source: &str,
    context: &str,
    out: &str,
    production: &dyn Production,
) -> Result<String, Error> {
    let forgejo_context = sys::join(&[out, "forgejo-context"]);
    sys::create_dir(&forgejo_context, 0o700)?;
    production.stage_fork_binary(&forgejo_context)?;
    production.assets(context, &forgejo_context)?;
    production.next("Verify immutable Forgejo presentation")?;
    payload.presentation_sha256 = complete::stage_presentation(&forgejo_context, context)?;
    let recipe = fs::read(sys::join(&[
        source,
        "system/containers/forgejo/Containerfile",
    ]))?;
    sys::write_new(
        &sys::join(&[forgejo_context.as_str(), "Containerfile"]),
        &recipe,
        0o644,
    )?;
    stage_extension_package(source, context, out, production.native())?;
    crate::build::link_prepared_assets(production)?;
    Ok(forgejo_context)
}

pub fn stage_extension_package(
    source: &str,
    context: &str,
    out: &str,
    native: &str,
) -> Result<(), Error> {
    let root = sys::join(&[out, "extension-context"]);
    let package_dir = sys::join(&[&root, "extension"]);
    fs::create_dir_all(&package_dir)?;
    let mut pairs = [("extension.json", 0o644), ("run", 0o755)];
    pairs.sort();
    for (name, mode) in pairs {
        let data = fs::read(sys::join(&[source, "system/containers/extension", name]))?;
        sys::write_new(&sys::join(&[&package_dir, name]), &data, mode)?;
    }
    fs::hard_link(
        sys::join(&[context, "rootfs/usr/libexec/soda/soda-extension"]),
        sys::join(&[&package_dir, "backend"]),
    )
    .map_err(|e| Error::msg(e.to_string()))?;
    extension::stage_extension_assets(native, &package_dir)?;
    let recipe = fs::read(sys::join(&[
        source,
        "system/containers/extension/Containerfile",
    ]))?;
    sys::write_new(&sys::join(&[&root, "Containerfile"]), &recipe, 0o644)
}

pub fn record_candidate_images(
    payload: &mut model::Payload,
    prefix: &str,
    images: &HashMap<String, model::ProducedImage>,
) {
    let mut names: Vec<&String> = images.keys().collect();
    names.sort();
    for name in names {
        let image = &images[name];
        match payload.images.iter_mut().find(|(n, _)| n == name) {
            Some((_, slot)) => {
                *slot = model::PayloadImage {
                    reference: format!("{prefix}-{name}@{}", image.manifest),
                    manifest: image.manifest.clone(),
                    config: image.config.clone(),
                    archive_sha256: image.archive_sha256.clone(),
                };
            }
            None => payload.images.push((
                (*name).clone(),
                model::PayloadImage {
                    reference: format!("{prefix}-{name}@{}", image.manifest),
                    manifest: image.manifest.clone(),
                    config: image.config.clone(),
                    archive_sha256: image.archive_sha256.clone(),
                },
            )),
        }
    }
}

pub fn inspect_candidate_forgejo(
    source: &str,
    out: &str,
    images: &HashMap<String, model::ProducedImage>,
    production: &dyn Production,
) -> Result<(), Error> {
    production.next("Inspect immutable Forgejo presentation")?;
    let empty = model::ProducedImage::default();
    let config = images.get("forgejo").unwrap_or(&empty).config.clone();
    // Read-only upstream binary, not its database/bootstrap entrypoint.
    let script = "test \"$GITEA_CUSTOM\" = /usr/share/soda/forgejo\ntest \"$FORGEJO_CUSTOM\" = \"$GITEA_CUSTOM\"\ntest \"$(readlink \"$GITEA_CUSTOM/conf\")\" = /data/gitea/conf\ntest \"$(stat -c '%u:%g:%a' \"$GITEA_CUSTOM/templates/custom/header.tmpl\")\" = 0:0:444\n/usr/local/bin/gitea --version\n/usr/local/bin/gitea extensions --help >/dev/null";
    let result = production
        .capture(
            source,
            "podman",
            &[
                "--remote=false".to_string(),
                "run".to_string(),
                "--cidfile".to_string(),
                sys::join(&[out, "forgejo-inspect.cid"]),
                "--network=none".to_string(),
                "--read-only".to_string(),
                "--entrypoint=/bin/sh".to_string(),
                config,
                "-ec".to_string(),
                script.to_string(),
            ],
        )
        .map_err(|e| Error::msg(format!("forgejo payload image inspection failed: {}", e.0)))?;
    inspect_candidate_files(out, images, production)?;
    sys::write_new(
        &sys::join(&[out, "forgejo-inspection.txt"]),
        format!("{result}\n").as_bytes(),
        0o600,
    )
}

pub fn inspect_candidate_files(
    out: &str,
    images: &HashMap<String, model::ProducedImage>,
    production: &dyn Production,
) -> Result<(), Error> {
    let empty = model::ProducedImage::default();
    inspect_packaged_file(
        out,
        &images.get("forgejo").unwrap_or(&empty).config.clone(),
        "forgejo-context/forgejo-bin",
        "/usr/local/bin/gitea",
        production,
    )?;
    inspect_packaged_file(
        out,
        &images.get("extension").unwrap_or(&empty).config.clone(),
        "forgejo-context/forgejo-bin",
        "/usr/local/bin/gitea",
        production,
    )?;
    let mut pairs = [
        (
            "extension-context/extension/extension.json",
            "/usr/share/soda/extension/extension.json",
        ),
        (
            "extension-context/extension/backend",
            "/usr/share/soda/extension/backend",
        ),
        (
            "extension-context/extension/run",
            "/usr/share/soda/extension/run",
        ),
    ];
    pairs.sort();
    for (staged, installed) in pairs {
        inspect_packaged_file(
            out,
            &images.get("extension").unwrap_or(&empty).config.clone(),
            staged,
            installed,
            production,
        )?;
    }
    inspect_extension_assets(
        out,
        &images.get("extension").unwrap_or(&empty).config.clone(),
        production,
    )
}

pub fn inspect_extension_assets(
    out: &str,
    image_id: &str,
    production: &dyn Production,
) -> Result<(), Error> {
    let assets = sys::join(&[out, "extension-context/extension/assets"]);
    sys::walk(&assets, |path, is_dir, _| {
        if is_dir {
            return Ok(());
        }
        let rel = sys::rel_path(&assets, path)?;
        inspect_packaged_file(
            out,
            image_id,
            &format!("extension-context/extension/assets/{rel}"),
            &format!("/usr/share/soda/extension/assets/{}", sys::to_slash(&rel)),
            production,
        )
    })
}

pub fn inspect_packaged_file(
    out: &str,
    image_id: &str,
    staged: &str,
    installed: &str,
    production: &dyn Production,
) -> Result<(), Error> {
    let want = sys::hash_file(&sys::join(&[out, staged]))?;
    let observed = production.capture(
        out,
        "podman",
        &[
            "--remote=false".to_string(),
            "run".to_string(),
            "--rm".to_string(),
            "--network=none".to_string(),
            "--read-only".to_string(),
            "--cap-drop=all".to_string(),
            "--security-opt=no-new-privileges".to_string(),
            "--entrypoint=/usr/bin/sha256sum".to_string(),
            image_id.to_string(),
            installed.to_string(),
        ],
    );
    match observed {
        Ok(observed) if observed == format!("{want}  {installed}") => Ok(()),
        _ => Err(Error::msg(format!(
            "packaged content differs from staged source: {installed}"
        ))),
    }
}

pub fn seal_candidate_payload(
    payload: &model::Payload,
    source: &str,
    context: &str,
    out: &str,
    production: &dyn Production,
) -> Result<(), Error> {
    production.next("Assemble host payload and ordinary Podman image references")?;
    payload.validate()?;
    complete::complete(
        source,
        context,
        &sys::join(&[out, "images"]),
        payload,
        Some(production),
    )?;
    let mut record = jsonio::to_indent(&payload.to_json());
    record.push('\n');
    sys::write_new(&sys::join(&[out, "payload.json"]), record.as_bytes(), 0o600)
}

/// completeCandidate is image-layout assembly, not a second component
/// producer. The payload seals later: the host inventory floats, so its
/// fingerprint is recorded from the built image and set before sealing.
pub fn complete_candidate(
    source: &str,
    context: &str,
    out: &str,
    arch: &str,
    revision: &str,
    prefix: &str,
    base: &crate::prepare::Base,
    production: &dyn Production,
    phase: &mut dyn FnMut(&str) -> Result<(), Error>,
) -> Result<model::Payload, Error> {
    let mut payload = model::Payload {
        format: 3,
        id: format!(
            "{}.soda-{}",
            base.release,
            revision.get(..12).unwrap_or(revision)
        ),
        revision: revision.to_string(),
        architecture: arch.to_string(),
        core_os: base.release.clone(),
        base: base.image(arch),
        repository_prefix: prefix.to_string(),
        schema: model::SCHEMA_VERSION,
        images: Vec::new(),
        upgrade_from: Vec::new(),
        ..model::Payload::default()
    };
    link_candidate_commands(source, context, production.native())?;
    let forgejo_context = stage_candidate_forgejo(&mut payload, source, context, out, production)?;
    phase("P4 / Build application images")?;
    let images = production.images(&forgejo_context)?;
    record_candidate_images(&mut payload, prefix, &images);
    inspect_candidate_forgejo(source, out, &images, production)?;
    Ok(payload)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_record_candidate_images_binds_references() {
        // Oracle: Go recordCandidateImages reference shape.
        let mut payload = model::Payload::default();
        let mut images = HashMap::new();
        images.insert(
            "forgejo".to_string(),
            model::ProducedImage {
                manifest: "sha256:manifest".to_string(),
                config: "sha256:config".to_string(),
                archive_sha256: "archive".to_string(),
            },
        );
        record_candidate_images(&mut payload, "ghcr.io/e/sodaos", &images);
        let image = payload.image("forgejo");
        assert_eq!(image.reference, "ghcr.io/e/sodaos-forgejo@sha256:manifest");
        assert_eq!(image.config, "sha256:config");
    }

    #[test]
    fn extension_package_reads_new_layout() {
        // CORR-C-005: extension inputs live under system/containers/extension/.
        let dir = std::env::temp_dir().join(format!("sri-ext-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let source = dir.join("src");
        let ext = source.join("system/containers/extension");
        fs::create_dir_all(&ext).unwrap();
        fs::write(ext.join("extension.json"), b"{}").unwrap();
        fs::write(ext.join("run"), b"#!/bin/sh\n").unwrap();
        fs::write(ext.join("Containerfile"), b"FROM x\n").unwrap();
        let context = dir.join("ctx");
        let backend = context.join("rootfs/usr/libexec/soda/soda-extension");
        fs::create_dir_all(backend.parent().unwrap()).unwrap();
        fs::write(&backend, b"backend").unwrap();
        let native = dir.join("native");
        let assets = native.join("soda-extension-assets");
        fs::create_dir_all(assets.join("assets")).unwrap();
        fs::write(assets.join("files.json"), b"[\"a.bin\"]").unwrap();
        fs::write(assets.join("assets/a.bin"), b"asset").unwrap();
        let out = dir.join("out");
        fs::create_dir_all(&out).unwrap();
        stage_extension_package(
            source.to_str().unwrap(),
            context.to_str().unwrap(),
            out.to_str().unwrap(),
            native.to_str().unwrap(),
        )
        .unwrap();
        let package = out.join("extension-context/extension");
        assert!(package.join("extension.json").is_file());
        assert!(package.join("run").is_file());
        assert!(package.join("backend").is_file());
        assert!(package.join("assets/a.bin").is_file());
        assert!(out.join("extension-context/Containerfile").is_file());
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn cmd1_staging_links_resolve_without_terminal_identity() {
        // CORR-C-004-AMEND-1 (CODEX-A01-CMD-1): payload staging links only
        // real commands. Post-fold, context holds project-terminal (via
        // RUST_TOOLS) but no soda-project-terminal binary; staging must
        // succeed and must not link the bogus identity.
        let dir = std::env::temp_dir().join(format!("sri-cmd1c-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let source = dir.join("src");
        fs::create_dir_all(source.join("cmd/soda-fakego")).unwrap();
        let folded = source.join("cmd/soda-project-terminal");
        fs::create_dir_all(&folded).unwrap();
        fs::write(
            folded.join("Cargo.toml"),
            b"[package]\nname = \"soda-project-terminal\"\n[[bin]]\nname = \"project-terminal\"\n[[bin]]\nname = \"project-account\"\n",
        )
        .unwrap();
        let context = dir.join("ctx");
        let bindir = context.join("rootfs/usr/libexec/soda");
        fs::create_dir_all(&bindir).unwrap();
        fs::write(bindir.join("soda-fakego"), b"fake").unwrap();
        fs::write(bindir.join("project-terminal"), b"terminal").unwrap();
        fs::write(bindir.join("project-account"), b"account").unwrap();
        let native = dir.join("native");
        link_candidate_commands(
            source.to_str().unwrap(),
            context.to_str().unwrap(),
            native.to_str().unwrap(),
        )
        .unwrap();
        assert!(native.join("bin/soda-fakego").is_file());
        assert!(!native.join("bin/soda-project-terminal").exists());
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn cr02_staging_links_resolve_without_pg_identity() {
        // CODEX-CR02-001: payload staging links only real commands. Context
        // holds the three pg binaries (via RUST_TOOLS) but no
        // soda-pg-maintenance binary; staging must succeed and must not
        // link the bogus identity.
        let dir = std::env::temp_dir().join(format!("sri-cr02c-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let source = dir.join("src");
        fs::create_dir_all(source.join("cmd/soda-fakego")).unwrap();
        let pg = source.join("cmd/soda-pg-maintenance");
        fs::create_dir_all(&pg).unwrap();
        fs::write(
            pg.join("Cargo.toml"),
            b"[package]\nname = \"soda-pg-maintenance\"\n[[bin]]\nname = \"soda-pg-backup\"\n[[bin]]\nname = \"soda-pg-restore\"\n[[bin]]\nname = \"soda-pg-init-roles\"\n",
        )
        .unwrap();
        let context = dir.join("ctx");
        let bindir = context.join("rootfs/usr/libexec/soda");
        fs::create_dir_all(&bindir).unwrap();
        fs::write(bindir.join("soda-fakego"), b"fake").unwrap();
        fs::write(bindir.join("soda-pg-backup"), b"backup").unwrap();
        fs::write(bindir.join("soda-pg-restore"), b"restore").unwrap();
        fs::write(bindir.join("soda-pg-init-roles"), b"roles").unwrap();
        let native = dir.join("native");
        link_candidate_commands(
            source.to_str().unwrap(),
            context.to_str().unwrap(),
            native.to_str().unwrap(),
        )
        .unwrap();
        assert!(native.join("bin/soda-fakego").is_file());
        assert!(!native.join("bin/soda-pg-maintenance").exists());
        let _ = fs::remove_dir_all(&dir);
    }
}

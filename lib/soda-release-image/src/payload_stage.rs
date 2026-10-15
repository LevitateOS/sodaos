//! `build_payload.go`: application image staging + payload sealing.

use std::collections::HashMap;
use std::fs;

use crate::complete;
use crate::error::Error;
use crate::foreign::Production;
use crate::model;
use crate::native_package;
use crate::sys;

pub use native_package::ReleaseInputs;

pub fn link_candidate_commands(
    context: &str,
    native: &str,
    commands: &[String],
) -> Result<(), Error> {
    fs::create_dir_all(sys::join(&[native, "bin"]))?;
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
    release: &native_package::ReleaseInputs,
) -> Result<String, Error> {
    let forgejo_context = sys::join(&[out, "forgejo-context"]);
    sys::create_dir(&forgejo_context, 0o700)?;
    // Admission consumes the independently released Fountain image: the
    // producer never checks out, compiles, or stages a fork binary.
    production.assets(context, &forgejo_context)?;
    stage_project_support(context, production.native())?;
    payload.presentation_sha256 = complete::stage_presentation(&forgejo_context, context)?;
    stage_forgejo_container_inputs(source, &forgejo_context)?;
    stage_native_packages(payload, source, context, out, production.native(), release)?;
    crate::build::link_prepared_assets(production)?;
    remove_container_only_host_commands(context)?;
    Ok(forgejo_context)
}

/// Stage the two native extension packages, seal their catalog, and bind
/// the release tag plus the admitted Fountain release into the payload.
/// The projected catalog bytes are preserved as candidate evidence; the
/// host stages the same bytes read back from the payload.
pub fn stage_native_packages(
    payload: &mut model::Payload,
    source: &str,
    context: &str,
    out: &str,
    native: &str,
    release: &native_package::ReleaseInputs,
) -> Result<native_package::NativePackages, Error> {
    release.validate()?;
    fs::create_dir_all(sys::join(&[out, "packages"]))?;
    let soda = native_package::stage_soda_package(
        source,
        context,
        native,
        out,
        release.source_timestamp,
        &release.release_tag,
    )?;
    let tailnet = native_package::stage_tailnet_package(
        source,
        out,
        release.source_timestamp,
        &release.release_tag,
    )?;
    payload.release_tag = release.release_tag.clone();
    payload.fountain = release.fountain.clone();
    let catalog_bytes =
        native_package::record_extension_catalog(payload, &release.release_tag, &soda, &tailnet)?;
    sys::write_new(
        &sys::join(&[out, "packages/catalog.json"]),
        &catalog_bytes,
        0o644,
    )?;
    Ok(native_package::NativePackages {
        soda,
        tailnet,
        catalog_sha256: sys::hex_sha256(&catalog_bytes),
    })
}

/// Consume the single compilation owned by production_assets. Both host uses
/// are part of the complete release root; maintenance needs no extra download.
pub fn stage_project_support(context: &str, native: &str) -> Result<(), Error> {
    let source = sys::join(&[native, "project-tools/bin/project-terminal"]);
    let data = fs::read(&source)?;
    for installed in [
        soda_build_tools::project_support::DESTINATION,
        soda_build_tools::project_support::PAYLOAD_PATH,
    ] {
        let destination = sys::join(&[context, "rootfs", installed.trim_start_matches('/')]);
        fs::create_dir_all(sys::dir_name(&destination))?;
        sys::write_new(&destination, &data, 0o755)?;
    }
    Ok(())
}

pub fn bind_project_support(
    payload: &mut model::Payload,
    context: &str,
    native: &str,
    production: &dyn Production,
) -> Result<(), Error> {
    let sha256 = sys::hash_file(&sys::join(&[native, "project-tools/bin/project-terminal"]))?;
    let binding = model::ProjectSupport {
        sha256,
        destination: soda_build_tools::project_support::DESTINATION.to_string(),
        payload_path: soda_build_tools::project_support::PAYLOAD_PATH.to_string(),
        project_image: model::Role::ProjectOs
            .binding(&payload.images)
            .config
            .clone(),
    };
    for installed in [&binding.destination, &binding.payload_path] {
        let path = sys::join(&[context, "rootfs", installed.trim_start_matches('/')]);
        if sys::hash_file(&path)? != binding.sha256 {
            return Err(Error::msg("Project support staging mismatch"));
        }
    }
    inspect_packaged_digest(
        context,
        &binding.project_image,
        &binding.sha256,
        &binding.destination,
        production,
    )?;
    payload.project_support = binding;
    Ok(())
}

fn remove_container_only_host_commands(context: &str) -> Result<(), Error> {
    // Component contexts already hold hardlinks to these bytes. Their servers
    // execute inside the dashboard/extension images, not on the native host.
    for name in ["soda-dashboard", "soda-extension"] {
        fs::remove_file(sys::join(&[context, "rootfs/usr/libexec/soda", name]))?;
    }
    Ok(())
}

fn stage_forgejo_container_inputs(source: &str, context: &str) -> Result<(), Error> {
    for (name, mode) in [("Containerfile", 0o644), ("setup", 0o755)] {
        let data = fs::read(sys::join(&[source, "system/containers/forgejo", name]))?;
        sys::write_new(&sys::join(&[context, name]), &data, mode)?;
    }
    Ok(())
}

/// Record the four Soda-built roles with Soda provenance. The Rocky base
/// covers dashboard, project-os, and canlang-runtime; the Soda Fountain
/// derivative's base is the admitted Fountain image. Proxy and postgres
/// skip this binding: they bind upstream with their frozen requests.
pub fn record_candidate_images(
    payload: &mut model::Payload,
    prefix: &str,
    images: &HashMap<String, model::ProducedImage>,
    revision: &str,
    rocky_base: &str,
    fountain_image: &str,
    arch: &str,
) -> Result<(), Error> {
    let platform = match arch {
        "x86_64" => "linux/amd64",
        _ => return Err(Error::msg("payload images require the x86_64 platform")),
    };
    let mut names: Vec<&String> = images.keys().collect();
    names.sort();
    for name in names {
        let Some(role) = model::Role::from_name(name) else {
            return Err(Error::msg(format!(
                "unknown image role {name:?}: not a Payload v4 role"
            )));
        };
        if matches!(role, model::Role::Proxy | model::Role::Postgres) {
            continue;
        }
        let image = &images[name];
        let base = match role {
            model::Role::Forgejo => fountain_image,
            _ => rocky_base,
        };
        *role.binding_mut(&mut payload.images) = model::PayloadImage {
            reference: format!("{prefix}-{name}@{}", image.manifest),
            manifest: image.manifest.clone(),
            config: image.config.clone(),
            platform: platform.to_string(),
            provenance: model::ImageProvenance::Soda {
                revision: revision.to_string(),
                base: base.to_string(),
            },
        };
    }
    Ok(())
}

pub fn inspect_candidate_forgejo(
    source: &str,
    out: &str,
    images: &HashMap<String, model::ProducedImage>,
    production: &dyn Production,
) -> Result<(), Error> {
    let empty = model::ProducedImage::default();
    let config = images.get("forgejo").unwrap_or(&empty).config.clone();
    // The Soda Fountain derivative preserves the released upstream
    // executable, wrappers, and entrypoint: verify them plus Soda's
    // presentation overlay, never a compiled fork binary.
    let script = "test -x /app/gitea/gitea\ntest \"$GITEA_CUSTOM\" = /usr/share/soda/forgejo\ntest \"$FORGEJO_CUSTOM\" = \"$GITEA_CUSTOM\"\ntest \"$(readlink \"$GITEA_CUSTOM/conf\")\" = /data/gitea/conf\ntest \"$(stat -c '%u:%g:%a' \"$GITEA_CUSTOM/templates/custom/header.tmpl\")\" = 0:0:444\ntest -f /usr/share/soda/presentation.json\n/app/gitea/gitea --version\n/app/gitea/gitea extensions --help >/dev/null";
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
    for (name, staged) in crate::canlang::runtime_content(production.native())? {
        let installed = name
            .strip_prefix("canlang-runtime:")
            .expect("runtime content owner");
        inspect_packaged_digest(
            out,
            &images.get("canlang-runtime").unwrap_or(&empty).config,
            &sys::hash_file(&staged)?,
            installed,
            production,
        )?;
    }
    Ok(())
}

pub fn inspect_packaged_file(
    out: &str,
    image_id: &str,
    staged: &str,
    installed: &str,
    production: &dyn Production,
) -> Result<(), Error> {
    let want = sys::hash_file(&sys::join(&[out, staged]))?;
    inspect_packaged_digest(out, image_id, &want, installed, production)
}

pub(crate) fn inspect_packaged_digest(
    out: &str,
    image_id: &str,
    want: &str,
    installed: &str,
    production: &dyn Production,
) -> Result<(), Error> {
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
) -> Result<(), Error> {
    payload.validate().map_err(|e| Error::msg(e.to_string()))?;
    complete::complete(source, context, payload)?;
    let mut record =
        serde_json::to_string_pretty(payload).expect("serialization to String cannot fail");
    record.push('\n');
    sys::write_new(&sys::join(&[out, "payload.json"]), record.as_bytes(), 0o600)
}

/// completeCandidate is image-layout assembly, not a second component
/// producer. The payload seals later: the host inventory floats, so its
/// fingerprint is recorded from the built image and set before sealing.
/// R02 supplies the release tag, frozen Fountain release, native package
/// catalog, and optional Tailnet binding through `release`; each missing
/// input fails staging closed before anything seals.
pub fn complete_candidate(
    source: &str,
    context: &str,
    out: &str,
    arch: &str,
    revision: &str,
    prefix: &str,
    base: &crate::prepare::Base,
    production: &dyn Production,
    commands: &[String],
    release: &native_package::ReleaseInputs,
    phase: &mut dyn FnMut(&str) -> Result<(), Error>,
) -> Result<model::Payload, Error> {
    release.validate()?;
    let base_image = base.image(arch);
    let mut payload = model::Payload {
        format: 4,
        id: format!(
            "{}.soda-{}",
            base.release,
            revision.get(..12).unwrap_or(revision)
        ),
        revision: revision.to_string(),
        architecture: arch.to_string(),
        core_os: base.release.clone(),
        base: base_image.clone(),
        schema: model::SCHEMA_VERSION as u32,
        ..model::Payload::default()
    };
    link_candidate_commands(context, production.native(), commands)?;
    let forgejo_context =
        stage_candidate_forgejo(&mut payload, source, context, out, production, release)?;
    phase("P4 / Build application images")?;
    let images = production.images(&forgejo_context)?;
    record_candidate_images(
        &mut payload,
        prefix,
        &images,
        revision,
        &base_image,
        &release.fountain.image,
        arch,
    )?;
    for (role, requested) in [
        (model::Role::Proxy, &release.proxy_requested),
        (model::Role::Postgres, &release.postgres_requested),
    ] {
        let Some(image) = images.get(role.name()) else {
            return Err(Error::msg(format!(
                "missing frozen {} image binding",
                role.name()
            )));
        };
        native_package::record_upstream_binding(
            &mut payload,
            role,
            &image.config,
            &image.manifest,
            requested,
        )?;
    }
    native_package::record_tailscale_binding(&mut payload, &release.tailscale)?;
    bind_project_support(&mut payload, context, production.native(), production)?;
    inspect_candidate_forgejo(source, out, &images, production)?;
    Ok(payload)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::fs::PermissionsExt;

    #[test]
    fn forgejo_container_inputs_stage_from_one_source_snapshot_with_modes() {
        let dir = std::env::temp_dir().join(format!("sri-forgejo-stage-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let source = dir.join("source/system/containers/forgejo");
        let context = dir.join("context");
        fs::create_dir_all(&source).unwrap();
        fs::create_dir_all(&context).unwrap();
        fs::write(
            source.join("Containerfile"),
            include_bytes!("../../../system/containers/forgejo/Containerfile"),
        )
        .unwrap();
        fs::write(
            source.join("setup"),
            include_bytes!("../../../system/containers/forgejo/setup"),
        )
        .unwrap();

        stage_forgejo_container_inputs(
            dir.join("source").to_str().unwrap(),
            context.to_str().unwrap(),
        )
        .unwrap();

        assert_eq!(
            fs::read(context.join("Containerfile")).unwrap(),
            fs::read(source.join("Containerfile")).unwrap()
        );
        assert_eq!(
            fs::read(context.join("setup")).unwrap(),
            fs::read(source.join("setup")).unwrap()
        );
        assert_eq!(
            fs::metadata(context.join("Containerfile"))
                .unwrap()
                .permissions()
                .mode()
                & 0o777,
            0o644
        );
        assert_eq!(
            fs::metadata(context.join("setup"))
                .unwrap()
                .permissions()
                .mode()
                & 0o777,
            0o755
        );
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn oracle_record_candidate_images_binds_references() {
        // Oracle: Go recordCandidateImages reference shape. Soda-built roles
        // carry Soda provenance; the Fountain derivative's base is the
        // admitted Fountain image, not the Rocky base. Upstream roles skip
        // this binding and bind with their frozen requests instead.
        let mut payload = model::Payload::default();
        let mut images = HashMap::new();
        for name in ["forgejo", "dashboard", "proxy", "postgres"] {
            images.insert(
                name.to_string(),
                model::ProducedImage {
                    manifest: "sha256:manifest".to_string(),
                    config: "sha256:config".to_string(),
                    archive_sha256: "archive".to_string(),
                },
            );
        }
        let revision = "a".repeat(40);
        let rocky = "quay.io/fedora/fedora-coreos@sha256:".to_string() + &"b".repeat(64);
        let fountain = "example.invalid/fountain@sha256:".to_string() + &"c".repeat(64);
        record_candidate_images(
            &mut payload,
            "ghcr.io/e/sodaos",
            &images,
            &revision,
            &rocky,
            &fountain,
            "x86_64",
        )
        .unwrap();
        let image = model::Role::Forgejo.binding(&payload.images);
        assert_eq!(image.reference, "ghcr.io/e/sodaos-forgejo@sha256:manifest");
        assert_eq!(image.config, "sha256:config");
        assert_eq!(image.platform, "linux/amd64");
        match &image.provenance {
            model::ImageProvenance::Soda {
                revision: got,
                base: got_base,
            } => {
                assert_eq!(got, &revision);
                assert_eq!(got_base, &fountain);
            }
            other => panic!("unexpected provenance: {other:?}"),
        }
        match &model::Role::Dashboard.binding(&payload.images).provenance {
            model::ImageProvenance::Soda { base, .. } => assert_eq!(base, &rocky),
            other => panic!("unexpected provenance: {other:?}"),
        }
        // Upstream roles are untouched by the Soda binding.
        assert!(model::Role::Proxy
            .binding(&payload.images)
            .reference
            .is_empty());
        assert!(model::Role::Postgres
            .binding(&payload.images)
            .reference
            .is_empty());
    }

    #[test]
    fn record_candidate_images_rejects_non_v4_roles() {
        let mut payload = model::Payload::default();
        let mut images = HashMap::new();
        images.insert(
            "extension".to_string(),
            model::ProducedImage {
                manifest: "sha256:manifest".to_string(),
                config: "sha256:config".to_string(),
                archive_sha256: "archive".to_string(),
            },
        );
        let revision = "a".repeat(40);
        let base = "quay.io/fedora/fedora-coreos@sha256:".to_string() + &"b".repeat(64);
        assert_eq!(
            record_candidate_images(
                &mut payload,
                "ghcr.io/e/sodaos",
                &images,
                &revision,
                &base,
                &base,
                "x86_64"
            )
            .unwrap_err()
            .0,
            "unknown image role \"extension\": not a Payload v4 role"
        );
    }

    fn package_fixture_tree(
        dir: &std::path::Path,
        id: &str,
        capabilities: &[&str],
    ) -> (String, String, String) {
        // Synthetic T03-shaped package source: manifest, entrypoint,
        // backend, one asset, LICENSE/NOTICE. Capabilities mirror the
        // canonical approved lists so the fixture proves the contract,
        // not a divergent fixture shape.
        let source = dir.join("src");
        let origin = if id == "soda" {
            source.join("system/containers/extension")
        } else {
            source.join("system/containers/extension/tailnet-package")
        };
        fs::create_dir_all(&origin).unwrap();
        let caps: Vec<String> = capabilities.iter().map(|c| format!("\"{c}\"")).collect();
        fs::write(
            origin.join("extension.json"),
            format!(
                "{{\"protocol\":1,\"id\":\"{id}\",\"version\":\"0.1.0\",\"executable\":\"run\",\"capabilities\":[{}]}}",
                caps.join(",")
            ),
        )
        .unwrap();
        fs::write(origin.join("run"), b"#!/bin/sh\n").unwrap();
        fs::write(origin.join("backend"), b"backend").unwrap();
        fs::create_dir_all(origin.join("assets")).unwrap();
        fs::write(origin.join("assets/a.bin"), b"asset").unwrap();
        fs::write(source.join("LICENSE"), b"license").unwrap();
        fs::write(source.join("NOTICE"), b"notice").unwrap();
        let context = dir.join("ctx");
        let backend = context.join("rootfs/usr/libexec/soda/soda-extension");
        fs::create_dir_all(backend.parent().unwrap()).unwrap();
        fs::write(&backend, b"backend").unwrap();
        let native = dir.join("native");
        let assets = native.join("soda-extension-assets");
        fs::create_dir_all(assets.join("assets")).unwrap();
        fs::write(assets.join("files.json"), b"[\"a.bin\"]").unwrap();
        fs::write(assets.join("assets/a.bin"), b"asset").unwrap();
        (
            source.to_str().unwrap().to_string(),
            context.to_str().unwrap().to_string(),
            native.to_str().unwrap().to_string(),
        )
    }

    #[test]
    fn native_packages_stage_seal_and_catalog() {
        let dir = std::env::temp_dir().join(format!("sri-native-pkg-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let soda_caps: Vec<&str> = model::SODA_CAPABILITIES.to_vec();
        let (source, context, native) = package_fixture_tree(&dir, "soda", &soda_caps);
        let tail_caps: Vec<&str> = model::TAILNET_CAPABILITIES.to_vec();
        package_fixture_tree(&dir, "tailnet", &tail_caps);
        let out = dir.join("out");
        fs::create_dir_all(&out).unwrap();
        let mut payload = model::Payload::default();
        let release = native_package::ReleaseInputs {
            release_tag: "v0.0.0-dev-123456789abc".to_string(),
            fountain: model::FountainRelease {
                version: "9.9".to_string(),
                image: "example.invalid/fountain@sha256:".to_string() + &"a".repeat(64),
                image_config: format!("sha256:{}", "b".repeat(64)),
                source_revision: "c".repeat(40),
                extension_protocol: 1,
                browser_contract: 1,
                ..Default::default()
            },
            proxy_requested: "example.invalid/proxy@sha256:".to_string() + &"d".repeat(64),
            postgres_requested: "example.invalid/postgres@sha256:".to_string() + &"e".repeat(64),
            source_timestamp: 1_700_000_000,
            ..Default::default()
        };
        let packages = stage_native_packages(
            &mut payload,
            &source,
            &context,
            out.to_str().unwrap(),
            &native,
            &release,
        )
        .unwrap();
        assert_eq!(payload.release_tag, "v0.0.0-dev-123456789abc");
        for package in [&packages.soda, &packages.tailnet] {
            assert_eq!(package.version, "0.0.0-dev-123456789abc");
            let decoder = flate2::read::GzDecoder::new(package.archive.as_slice());
            let mut archive = tar::Archive::new(decoder);
            let mut manifest = archive
                .entries()
                .unwrap()
                .map(Result::unwrap)
                .find(|entry| entry.path().unwrap() == std::path::Path::new("extension.json"))
                .unwrap();
            let value: serde_json::Value = serde_json::from_reader(&mut manifest).unwrap();
            assert_eq!(value["version"], package.version);
        }
        let source_manifest: serde_json::Value = serde_json::from_slice(
            &fs::read(
                std::path::Path::new(&source).join("system/containers/extension/extension.json"),
            )
            .unwrap(),
        )
        .unwrap();
        assert_eq!(source_manifest["version"], "0.1.0");
        assert_eq!(payload.fountain.version, "9.9");
        assert_eq!(payload.extension_catalog.packages.len(), 2);
        assert!(out.join("packages").join(&packages.soda.filename).is_file());
        assert!(out
            .join("packages")
            .join(&packages.tailnet.filename)
            .is_file());
        let catalog = fs::read(out.join("packages/catalog.json")).unwrap();
        assert_eq!(catalog, payload.projected_catalog_bytes().unwrap());
        assert_eq!(packages.catalog_sha256, crate::sys::hex_sha256(&catalog));
        // The source tree is never written: no stage directory lands in it.
        assert!(!dir.join("src/system/containers/extension/package").exists());
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn native_packages_wait_for_tailnet_contents() {
        let dir = std::env::temp_dir().join(format!("sri-native-tail-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let soda_caps: Vec<&str> = model::SODA_CAPABILITIES.to_vec();
        let (source, context, native) = package_fixture_tree(&dir, "soda", &soda_caps);
        let out = dir.join("out");
        fs::create_dir_all(&out).unwrap();
        let mut payload = model::Payload::default();
        let release = native_package::ReleaseInputs {
            release_tag: "v0.1.0".to_string(),
            fountain: model::FountainRelease {
                version: "9.9".to_string(),
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
        assert_eq!(
            stage_native_packages(
                &mut payload,
                &source,
                &context,
                out.to_str().unwrap(),
                &native,
                &release
            )
            .unwrap_err()
            .0,
            "tailnet package contents pending T03: no system/containers/extension/tailnet-package source"
        );
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
            context.to_str().unwrap(),
            native.to_str().unwrap(),
            &["soda-fakego".to_string()],
        )
        .unwrap();
        assert!(native.join("bin/soda-fakego").is_file());
        assert!(!native.join("bin/soda-project-terminal").exists());
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn n07t3_staging_links_tailnet_only_before_go_retirement() {
        // N07-T3: candidate links follow discovery. Pre-retirement the
        // tailnet dir lists and links from rootfs/usr/libexec/soda/
        // soda-forgejo-tailnet — the same destination U1 compiles to.
        // Post-retirement the sourceless dir is excluded, staging succeeds,
        // and no link is produced (fixed-list-only binaries like soda-host
        // are likewise unlink candidates).
        let dir = std::env::temp_dir().join(format!("sri-n07t3c-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let source = dir.join("src");
        fs::create_dir_all(source.join("cmd/soda-fakego")).unwrap();
        let tailnet = source.join("cmd/soda-forgejo-tailnet");
        fs::create_dir_all(&tailnet).unwrap();
        fs::write(tailnet.join("main.go"), b"package main\n").unwrap();
        let context = dir.join("ctx");
        let bindir = context.join("rootfs/usr/libexec/soda");
        fs::create_dir_all(&bindir).unwrap();
        fs::write(bindir.join("soda-fakego"), b"fake").unwrap();
        fs::write(bindir.join("soda-forgejo-tailnet"), b"tailnet").unwrap();
        let native = dir.join("native");
        link_candidate_commands(
            context.to_str().unwrap(),
            native.to_str().unwrap(),
            &[
                "soda-fakego".to_string(),
                "soda-forgejo-tailnet".to_string(),
            ],
        )
        .unwrap();
        assert!(native.join("bin/soda-forgejo-tailnet").is_file());
        fs::remove_file(tailnet.join("main.go")).unwrap();
        fs::remove_dir_all(&native).unwrap();
        link_candidate_commands(
            context.to_str().unwrap(),
            native.to_str().unwrap(),
            &["soda-fakego".to_string()],
        )
        .unwrap();
        assert!(native.join("bin/soda-fakego").is_file());
        assert!(!native.join("bin/soda-forgejo-tailnet").exists());
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
            context.to_str().unwrap(),
            native.to_str().unwrap(),
            &["soda-fakego".to_string()],
        )
        .unwrap();
        assert!(native.join("bin/soda-fakego").is_file());
        assert!(!native.join("bin/soda-pg-maintenance").exists());
        let _ = fs::remove_dir_all(&dir);
    }
}

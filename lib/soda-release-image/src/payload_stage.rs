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
        context,
        native,
        out,
        release.source_timestamp,
        &release.release_tag,
    )?;
    payload.release_tag = release.release_tag.clone();
    payload.fountain = release.fountain.clone();
    let catalog_bytes = native_package::record_extension_catalog(
        payload,
        &release.package_base_url,
        &soda,
        &tailnet,
    )?;
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
        let base = &image.base_name;
        if !soda_build_tools::payload::valid_digest_reference(base)
            || base.rsplit_once('@').map(|(_, digest)| digest) != Some(image.base_digest.as_str())
            || !image
                .base_digest
                .strip_prefix("sha256:")
                .is_some_and(model::is_digest)
            || (role == model::Role::Forgejo && base != fountain_image)
        {
            return Err(Error::msg(
                "built image base differs from selected component",
            ));
        }
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

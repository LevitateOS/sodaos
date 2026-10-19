//! `build_record.go`: candidate content inventory + build receipts.

use soda_build_tools::reader::build::BuildResult;

use std::fs;

use crate::error::Error;
use crate::files;
use crate::foreign::Production;
use crate::jsonio;
use crate::model;
use crate::request;
use crate::sys;

/// These are the actual bytes staged into the host and component images.
/// The archive identities in the payload bind the images; this inventory
/// names Soda's own staged binaries, the generated service Quadlets, the
/// trusted extension catalog, the canlang runtime evidence, the selected
/// Fountain release proof, and the two sealed native packages. No fork
/// binary, extension OCI tree, or installer unit remains.
pub fn candidate_content(out: &str, native: &str) -> Result<Vec<(String, String)>, Error> {
    let root = sys::dir_name(out);
    let host = |installed: &str| {
        (
            format!("host:{installed}"),
            sys::join(&[
                &root,
                "work/host-context/rootfs",
                installed.trim_start_matches('/'),
            ]),
        )
    };
    let mut paths: Vec<(String, String)> = [
        (
            "dashboard:/usr/local/bin/soda-dashboard".to_string(),
            sys::join(&[native, "bin/soda-dashboard"]),
        ),
        host("/usr/share/containers/systemd/forgejo.container"),
        host("/usr/share/containers/systemd/soda-dashboard.container"),
        host("/usr/share/containers/systemd/soda-proxy.container"),
        host("/usr/share/containers/systemd/soda-postgres.container"),
        host("/usr/share/soda/extensions/catalog.json"),
        host("/usr/share/soda/canlang-tools/release.json"),
        host("/usr/share/soda/canlang-tools/canlang-tools-linux-x86_64.tar.gz"),
        (
            "release:fountain-release.json".to_string(),
            sys::join(&[out, "fountain-release.json"]),
        ),
    ]
    .into_iter()
    .collect();
    paths.extend(crate::canlang::runtime_content(native)?);
    let mut packages: Vec<String> = Vec::new();
    for entry in fs::read_dir(sys::join(&[out, "packages"]))? {
        let entry = entry.map_err(|e| Error::msg(e.to_string()))?;
        let name = entry.file_name().to_string_lossy().into_owned();
        if name.ends_with(".tar.gz") {
            packages.push(name);
        }
    }
    packages.sort();
    for name in packages {
        paths.push((
            format!("package:{name}"),
            sys::join(&[out, "packages", &name]),
        ));
    }
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
    let mut data = serde_json::to_string_pretty(&jsonio::SortedPairs(&sorted))
        .expect("serialization to String cannot fail");
    data.push('\n');
    Ok(data.into_bytes())
}

pub fn write_content_inventory(context: &str, out: &str, native: &str) -> Result<(), Error> {
    let files = candidate_content(out, native)?;
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
) -> Result<BuildResult, Error> {
    let mut result = BuildResult {
        revision: production.revision().to_string(),
        architecture: production.arch().to_string(),
        candidate: sys::join(&[production.out(), "candidate.json"]),
        purpose: request.purpose(),
        requested_target: request.requested_target(),
        distribution: request.distribution().to_string(),
        release_tag: request.release_tag.clone(),
        completed_target: "candidate".to_string(),
        scope: "P1-P6 verified candidate; not a qualified release".to_string(),
        checks: [
            "ELF architecture",
            "Application OCI identities",
            "Host identity, RPM inventory, thin content and Quadlets",
            "Host OCI export",
        ]
        .into_iter()
        .map(|s| s.to_string())
        .collect(),
        ..BuildResult::default()
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
    if request.wants_docker() {
        result.docker = sys::join(&[production.out(), "docker/docker.json"]);
        result.checks.push(
            "Docker OCI identity, candidate content, provenance and RPM inventory".to_string(),
        );
        if !request.development {
            result.scope = "candidate-derived packaging; not a qualified release".to_string();
        }
    }
    result.completed_target = request.completed_target().to_string();
    let mut data =
        serde_json::to_string_pretty(&result).expect("serialization to String cannot fail");
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
    native: &str,
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
        repository_prefix: prefix.to_string(),
        forgejo_revision: forgejo_revision.to_string(),
        architecture: arch.to_string(),
        migration: "No upgrade or writable-install migration is qualified. Conflicting saved image selections are refused; machine settings/secrets require explicit first-install setup.".to_string(),
        notes: "Complete unsigned local appliance payload only. Application images bind by digest for boot-time acquisition; native packages bind by catalog; repository names are intended, not provisioned. No production, native boot, signature or recovery acceptance.".to_string(),
        ..model::Candidate::default()
    };
    record_candidate_inputs(out, native, &mut record)?;
    let mut data =
        serde_json::to_string_pretty(&record).expect("serialization to String cannot fail");
    data.push('\n');
    sys::write_new(&sys::join(&[out, "candidate.json"]), data.as_bytes(), 0o600)
}

pub fn record_candidate_inputs(
    out: &str,
    native: &str,
    record: &mut model::Candidate,
) -> Result<(), Error> {
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
    // The selected Fountain release/source proof replaces the local fork
    // toolchain proof: the admitted revision must equal the proof's source
    // revision. The separate local source archive retains its own byte identity.
    let proof = sys::join(&[out, "fountain-release.json"]);
    let proof_text = fs::read(&proof)
        .map_err(|e| Error::msg(format!("missing selected Fountain release proof: {e}")))?;
    record.forgejo_source_sha256 = sys::hash_file(&sys::join(&[out, "forgejo-source.tar"]))?;
    let release: model::FountainRelease = serde_json::from_slice(&proof_text)
        .map_err(|e| Error::msg(format!("invalid Fountain release proof: {e}")))?;
    if release.source_revision != record.forgejo_revision {
        return Err(Error::msg(
            "selected Fountain source revision differs from the admitted revision",
        ));
    }
    record.content_sha256 = candidate_content(out, native)?;
    // The four built OCI archives plus the two sealed native packages ride
    // the candidate as transport metadata; upstream proxy/postgres/tailscale
    // bind by identity in the payload and carry no local archive.
    let mut archives = Vec::new();
    for name in ["dashboard", "project-os", "canlang-runtime", "forgejo"] {
        let archive = format!("images/{name}.oci");
        archives.push((
            name.to_string(),
            sys::hash_file(&sys::join(&[out, &archive]))?,
        ));
    }
    archives.sort_by(|a, b| a.0.cmp(&b.0));
    record.image_archives = archives;
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

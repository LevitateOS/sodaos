//! Docker artifact recording and read-only publisher admission. No image build,
//! container execution, credentials, registry writes or qualification authority.
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::io::{Read, Seek};

use crate::admission::load_admitted_candidate;
use crate::buildx::{open_layout_entry, read_at, write_new, Root};
use crate::model::{BuildPurpose, Candidate, Distribution, DockerMetadata, FileIdentity};
use crate::oci::{inspect_oci_file, inspect_oci_file_content, InspectedContent};
use crate::payload::Payload;
use crate::{hash_bytes, Error};

mod source;
#[cfg(test)]
mod tests;

pub const METADATA_PATH: &str = "docker/docker.json";
pub const IMAGE_PATH: &str = "docker/image.oci";
pub const PACKAGES_PATH: &str = "docker/packages.txt";
pub const COMPOSE_PATH: &str = "docker/compose.yaml";
const PACKAGES_INSTALLED: &str = "/usr/share/soda/docker-runtime/packages.txt";

/// Frozen producer inputs; the outer image must carry matching labels.
pub struct DockerRequest {
    pub release_tag: String,
    pub distribution: Distribution,
    pub purpose: BuildPurpose,
    pub fedora_base: String,
}

fn identity(bytes: &[u8]) -> FileIdentity {
    FileIdentity {
        sha256: hash_bytes(bytes)[7..].to_owned(),
        bytes: bytes.len() as u64,
    }
}

fn file_identity(root: &Root, path: &str) -> Result<FileIdentity, Error> {
    let (mut file, size) = open_layout_entry(root, path)?;
    let mut hash = Sha256::new();
    let mut buffer = [0; 64 * 1024];
    let mut bytes = 0;
    loop {
        let read = file.read(&mut buffer).map_err(|_| Error::refused())?;
        if read == 0 {
            break;
        }
        hash.update(&buffer[..read]);
        bytes += read as u64;
    }
    if bytes != size as u64 {
        return Err(Error::msg("Docker artifact size changed"));
    }
    let value = FileIdentity {
        sha256: format!("{:x}", hash.finalize()),
        bytes,
    };
    value.validate()?;
    Ok(value)
}

fn inspect(
    root: &Root,
    path: &str,
    revision: &str,
    paths: &[String],
) -> Result<InspectedContent, Error> {
    let (mut file, _) = open_layout_entry(root, path)?;
    file.rewind().map_err(|_| Error::refused())?;
    if paths.is_empty() {
        return inspect_oci_file(&mut file, "x86_64", revision);
    }
    inspect_oci_file_content(&mut file, "x86_64", revision, paths)
}

fn check_sources(
    root: &Root,
    p: &Payload,
    c: &Candidate,
) -> Result<BTreeMap<String, Vec<u8>>, Error> {
    source::verify_revision(root, "source.tar", &p.revision)?;
    source::verify_revision(root, "forgejo-source.tar", &c.forgejo_revision)?;
    if file_identity(root, "forgejo-source.tar")?.sha256 != c.forgejo_source_sha256 {
        return Err(Error::msg("Docker Fountain archive differs from candidate"));
    }
    let inputs = source::read_inputs(root)?;
    #[derive(serde::Deserialize)]
    #[serde(deny_unknown_fields)]
    struct Pin {
        repository: String,
        revision: String,
    }
    let pin: Pin = crate::json_serde::strict(&inputs["fountain-source.json"])
        .map_err(|_| Error::msg("Docker Fountain pin refused"))?;
    let parts: Vec<&str> = pin.repository.split('/').collect();
    if parts.len() != 2
        || parts.iter().any(|part| {
            part.is_empty()
                || !part
                    .bytes()
                    .all(|c| c.is_ascii_alphanumeric() || b"._-".contains(&c))
        })
        || pin.revision != c.forgejo_revision
    {
        return Err(Error::msg("Docker Fountain pin differs from candidate"));
    }
    Ok(inputs)
}

fn runtime_content(
    p: &Payload,
    c: &Candidate,
    inputs: &BTreeMap<String, Vec<u8>>,
    packages: &[u8],
) -> Result<BTreeMap<String, String>, Error> {
    let content = crate::prepare::candidate_content_with_embedded_inventory(&c.content_sha256)?;
    let mut expected: BTreeMap<String, String> = content
        .iter()
        .filter_map(|(path, hash)| {
            path.strip_prefix("host:")
                .map(|path| (path.to_owned(), hash.clone()))
        })
        .collect();
    expected.insert(
        "/usr/share/soda/release.json".into(),
        c.payload_sha256.clone(),
    );
    expected.insert(
        p.project_support.payload_path.clone(),
        p.project_support.sha256.clone(),
    );
    expected.insert(
        p.project_support.destination.clone(),
        p.project_support.sha256.clone(),
    );
    expected.insert(
        "/usr/share/soda/host-image/packages.txt".into(),
        p.host_packages_sha256.clone(),
    );
    for (member, input) in [
        ("/usr/share/licenses/soda/LICENSE", "LICENSE"),
        ("/usr/share/licenses/soda/NOTICE", "NOTICE"),
        (
            "/usr/lib/systemd/system/soda-docker-init.service",
            "system/docker/soda-docker-init.service",
        ),
        (
            "/usr/lib/systemd/system/soda-image-import.service.d/docker.conf",
            "system/docker/image-import.conf",
        ),
    ] {
        expected.insert(member.into(), identity(&inputs[input]).sha256);
    }
    expected.insert(PACKAGES_INSTALLED.into(), identity(packages).sha256);
    Ok(expected)
}

fn check_content(
    observed: &BTreeMap<String, String>,
    expected: &BTreeMap<String, String>,
) -> Result<(), Error> {
    if expected
        .iter()
        .any(|(path, hash)| observed.get(path) != Some(hash))
    {
        return Err(Error::msg(
            "Docker runtime content differs from candidate/source",
        ));
    }
    Ok(())
}

struct BundledBlobs {
    content: BTreeMap<String, String>,
    member_limits: BTreeMap<String, u64>,
}

fn bundled_blobs(root: &Root, p: &Payload) -> Result<BundledBlobs, Error> {
    let mut blobs = BTreeMap::new();
    let mut limits = BTreeMap::new();
    for (name, expected) in &p.images {
        let path = format!("images/{name}.oci");
        if file_identity(root, &path)?.sha256 != expected.archive_sha256 {
            return Err(Error::msg(format!(
                "Docker bundled {name} archive mismatch"
            )));
        }
        let revision = if name == "proxy" { "" } else { &p.revision };
        let observed = inspect(root, &path, revision, &[])?;
        if observed.image.manifest != expected.manifest || observed.image.config != expected.config
        {
            return Err(Error::msg(format!(
                "Docker bundled {name} identity mismatch"
            )));
        }
        for (path, hash) in observed.blobs {
            limits.insert(
                format!("/usr/share/soda/images/{path}"),
                observed.blob_sizes[&path],
            );
            blobs.insert(format!("/usr/share/soda/images/{path}"), hash);
        }
    }
    Ok(BundledBlobs {
        content: blobs,
        member_limits: limits,
    })
}

fn observed_metadata(candidate: &str, request: &DockerRequest) -> Result<DockerMetadata, Error> {
    if !crate::model::docker::valid_docker_tag(&request.release_tag)
        || request.distribution == Distribution::Appliance
    {
        return Err(Error::msg("Docker version and distribution required"));
    }
    crate::model::docker::fedora_digest(&request.fedora_base)?;
    let root = Root::open(candidate)?;
    let (p, c, _) = load_admitted_candidate(candidate)?;
    let inputs = check_sources(&root, &p, &c)?;
    let packages = read_at(&root, PACKAGES_PATH, 1 << 20)?;
    let text = std::str::from_utf8(&packages)
        .map_err(|_| Error::msg("Docker RPM inventory must be UTF-8"))?;
    let lines: Vec<String> = text.lines().map(str::to_owned).collect();
    if lines.is_empty() {
        return Err(Error::msg("Docker RPM inventory required"));
    }
    soda_build_tools::reader::rpm::valid_rpm_inventory(&lines)
        .map_err(|error| Error::msg(error.0))?;
    if read_at(&root, COMPOSE_PATH, 1 << 20)? != inputs["system/docker/compose.yaml"] {
        return Err(Error::msg("Docker Compose differs from archived source"));
    }
    if file_identity(&root, "host.oci")?.sha256 != c.host_archive_sha256 {
        return Err(Error::msg(
            "Docker source host archive differs from candidate",
        ));
    }
    let shared: Vec<String> = ["index.json", "oci-layout"]
        .iter()
        .map(|name| format!("/usr/share/soda/images/{name}"))
        .collect();
    let host = inspect(&root, "host.oci", &p.revision, &shared)?;
    if host.image != c.host {
        return Err(Error::msg(
            "Docker source host identity differs from candidate",
        ));
    }
    let mut expected = runtime_content(&p, &c, &inputs, &packages)?;
    expected.extend(host.content);
    let blobs = bundled_blobs(&root, &p)?;
    expected.extend(blobs.content);
    let paths = expected.keys().cloned().collect::<Vec<_>>();
    let (mut file, _) = open_layout_entry(&root, IMAGE_PATH)?;
    let image = crate::oci::inspect_oci_file_content_with_limits(
        &mut file,
        "x86_64",
        &p.revision,
        &paths,
        &blobs.member_limits,
    )?;
    check_content(&image.content, &expected)?;
    let value = DockerMetadata {
        format: 1,
        distribution: request.distribution,
        purpose: request.purpose,
        release_tag: request.release_tag.clone(),
        architecture: p.architecture.clone(),
        platform: "linux/amd64".into(),
        soda_revision: p.revision,
        forgejo_revision: c.forgejo_revision,
        soda_source: file_identity(&root, "source.tar")?,
        forgejo_source: file_identity(&root, "forgejo-source.tar")?,
        candidate_sha256: file_identity(&root, "candidate.json")?.sha256,
        payload_sha256: c.payload_sha256,
        host_manifest: c.host.manifest,
        host_archive_sha256: c.host_archive_sha256,
        fedora_base: request.fedora_base.clone(),
        recipe: identity(&inputs["system/docker/Dockerfile"]),
        image: image.image,
        image_archive: file_identity(&root, IMAGE_PATH)?,
        packages: identity(&packages),
        compose: identity(&inputs["system/docker/compose.yaml"]),
    };
    value.validate()?;
    check_labels(&image.labels, &value)?;
    Ok(value)
}

fn check_labels(labels: &BTreeMap<String, String>, value: &DockerMetadata) -> Result<(), Error> {
    let purpose = match value.purpose {
        BuildPurpose::Development => "development",
        BuildPurpose::Production => "production",
    };
    let distribution = match value.distribution {
        Distribution::Docker => "docker",
        Distribution::All => "all",
        Distribution::Appliance => "appliance",
    };
    for (label, expected) in [
        ("io.soda.runtime", "docker"),
        ("org.opencontainers.image.version", &value.release_tag),
        ("io.soda.build-purpose", purpose),
        ("io.soda.distribution", distribution),
        ("io.soda.candidate-sha256", &value.candidate_sha256),
        ("io.soda.payload-sha256", &value.payload_sha256),
        ("io.soda.docker.recipe-sha256", &value.recipe.sha256),
    ] {
        if labels.get(label).map(String::as_str) != Some(expected) {
            return Err(Error::msg(format!(
                "Docker provenance label mismatch: {label}"
            )));
        }
    }
    Ok(())
}

/// Producer handoff: derive identities from actual bytes and write once, only
/// after source, candidate, runtime-content and provenance-label checks pass.
pub fn record(candidate: &str, request: &DockerRequest) -> Result<DockerMetadata, Error> {
    let value = observed_metadata(candidate, request)?;
    let mut bytes = serde_json::to_vec_pretty(&value).map_err(|_| Error::refused())?;
    bytes.push(b'\n');
    write_new(&format!("{candidate}/{METADATA_PATH}"), &bytes, 0o644)?;
    Ok(value)
}

/// Recompute the complete binding; metadata declarations alone are not proof.
pub fn verify(candidate: &str) -> Result<DockerMetadata, Error> {
    let root = Root::open(candidate)?;
    let bytes = read_at(&root, METADATA_PATH, 1 << 20)?;
    let metadata: DockerMetadata =
        crate::json_serde::strict(&bytes).map_err(|_| Error::msg("Docker metadata refused"))?;
    metadata.validate()?;
    let observed = observed_metadata(
        candidate,
        &DockerRequest {
            release_tag: metadata.release_tag.clone(),
            distribution: metadata.distribution,
            purpose: metadata.purpose,
            fedora_base: metadata.fedora_base.clone(),
        },
    )?;
    if observed != metadata {
        return Err(Error::msg(
            "Docker metadata differs from observed artifacts",
        ));
    }
    Ok(metadata)
}

/// Publisher handoff before any registry write; runtime qualification is separate.
pub fn admit_publication(candidate: &str) -> Result<DockerMetadata, Error> {
    let metadata = verify(candidate)?;
    metadata.require_production()?;
    Ok(metadata)
}

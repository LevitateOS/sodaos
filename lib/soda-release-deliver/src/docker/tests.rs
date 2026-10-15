//! Small real tar/OCI fixtures; no engine, native build or registry required.
use super::*;
use crate::buildx::Image;
use crate::model::DOCKER_REPOSITORY;
use serde_json::{json, Value};
use std::io::Cursor;
use std::os::unix::fs::symlink;
use tempfile::TempDir;

const SODA_REVISION: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
const FOUNTAIN_REVISION: &str = "2222222222222222222222222222222222222222";

fn tar(files: &BTreeMap<String, Vec<u8>>, revision: Option<&str>) -> Vec<u8> {
    let mut archive = tar::Builder::new(Vec::new());
    if let Some(revision) = revision {
        let record = format!("52 comment={revision}\n");
        assert_eq!(record.len(), 52);
        let mut header = tar::Header::new_gnu();
        header.set_entry_type(tar::EntryType::XGlobalHeader);
        header.set_size(record.len() as u64);
        header.set_mode(0o644);
        header.set_cksum();
        archive
            .append_data(&mut header, "pax_global_header", record.as_bytes())
            .unwrap();
    }
    for (name, bytes) in files {
        let mut header = tar::Header::new_gnu();
        header.set_mode(0o644);
        header.set_size(bytes.len() as u64);
        header.set_cksum();
        archive
            .append_data(
                &mut header,
                name.trim_start_matches('/'),
                Cursor::new(bytes),
            )
            .unwrap();
    }
    archive.into_inner().unwrap()
}

struct FixtureImage {
    bytes: Vec<u8>,
    image: Image,
    blobs: BTreeMap<String, Vec<u8>>,
}

fn oci(
    files: &BTreeMap<String, Vec<u8>>,
    base: &str,
    labels: &BTreeMap<String, String>,
) -> FixtureImage {
    let layer = tar(files, None);
    let config = serde_json::to_vec(&json!({
        "os":"linux", "architecture":"amd64", "config":{"Labels":labels},
        "rootfs":{"type":"layers", "diff_ids":[hash_bytes(&layer)]}
    }))
    .unwrap();
    let manifest = serde_json::to_vec(&json!({
        "schemaVersion":2, "mediaType":"application/vnd.oci.image.manifest.v1+json",
        "config":{"mediaType":"application/vnd.oci.image.config.v1+json", "digest":hash_bytes(&config), "size":config.len()},
        "layers":[{"mediaType":"application/vnd.oci.image.layer.v1.tar", "digest":hash_bytes(&layer), "size":layer.len()}]
    })).unwrap();
    let blobs = [layer, config.clone(), manifest.clone()]
        .into_iter()
        .map(|bytes| (format!("blobs/sha256/{}", &hash_bytes(&bytes)[7..]), bytes))
        .collect::<BTreeMap<_, _>>();
    let mut entries = blobs.clone();
    entries.insert("index.json".into(), serde_json::to_vec(&json!({"schemaVersion":2, "manifests":[{
        "mediaType":"application/vnd.oci.image.manifest.v1+json", "digest":hash_bytes(&manifest), "size":manifest.len()
    }]})).unwrap());
    entries.insert(
        "oci-layout".into(),
        br#"{"imageLayoutVersion":"1.0.0"}"#.to_vec(),
    );
    FixtureImage {
        bytes: tar(&entries, None),
        blobs,
        image: Image {
            manifest: hash_bytes(&manifest),
            config: hash_bytes(&config),
            architecture: "amd64".into(),
            revision: labels["org.opencontainers.image.revision"].clone(),
            source: "https://github.com/LevitateOS/sodaos".into(),
            base_name: base.into(),
            base_digest: base.split('@').nth(1).unwrap().into(),
        },
    }
}

fn labels(base: &str, name: &str) -> BTreeMap<String, String> {
    BTreeMap::from([
        (
            "org.opencontainers.image.revision".into(),
            SODA_REVISION.into(),
        ),
        (
            "org.opencontainers.image.source".into(),
            "https://github.com/LevitateOS/sodaos".into(),
        ),
        ("org.opencontainers.image.base.name".into(), base.into()),
        (
            "org.opencontainers.image.base.digest".into(),
            base.split('@').nth(1).unwrap().into(),
        ),
        ("io.fixture.image".into(), name.into()),
    ])
}

struct Fixture {
    dir: TempDir,
    _build: TempDir,
    request: DockerRequest,
    runtime: BTreeMap<String, Vec<u8>>,
    runtime_labels: BTreeMap<String, String>,
}

impl Fixture {
    fn new(purpose: BuildPurpose) -> Self {
        let build = TempDir::new().unwrap();
        let dir = TempDir::new_in(build.path()).unwrap();
        std::fs::create_dir(dir.path().join("docker")).unwrap();
        std::fs::create_dir(dir.path().join("images")).unwrap();
        let golden: Value =
            serde_json::from_str(include_str!("../../tests/goldens/deliver.json")).unwrap();
        let mut p: Payload = serde_json::from_str(golden["payload"].as_str().unwrap()).unwrap();
        let mut c: Candidate = serde_json::from_str(golden["candidate"].as_str().unwrap()).unwrap();
        c.repository_prefix = crate::model::DOCKER_REPOSITORY.into();
        for role in [
            crate::payload::Role::Dashboard,
            crate::payload::Role::Forgejo,
            crate::payload::Role::ProjectOs,
            crate::payload::Role::CanlangRuntime,
        ] {
            let image = role.binding_mut(&mut p.images);
            image.reference = format!("{}-{}@{}", c.repository_prefix, role.name(), image.manifest);
        }
        p.release_tag = "v0.1.0".into();
        for package in &mut p.extension_catalog.packages {
            package.version = "0.1.0".into();
            package.url = format!("https://github.com/LevitateOS/sodaos/releases/download/v0.1.0/{}-extension-0.1.0-linux-x86_64.tar.gz", package.id);
        }
        c.content_sha256 = c
            .content_sha256
            .into_iter()
            .map(|(name, hash)| {
                (
                    name.replace("-extension-0.0.0-golden-", "-extension-0.1.0-"),
                    hash,
                )
            })
            .collect();
        let mut source = BTreeMap::new();
        for name in source::SOURCE_FILES {
            source.insert(name.into(), format!("fixture {name}\n").into_bytes());
        }
        source.insert(
            "fountain-source.json".into(),
            serde_json::to_vec(
                &json!({"repository":"fixture/fountain", "revision":FOUNTAIN_REVISION}),
            )
            .unwrap(),
        );
        std::fs::write(
            dir.path().join("source.tar"),
            tar(&source, Some(SODA_REVISION)),
        )
        .unwrap();
        let fountain = tar(
            &BTreeMap::from([("LICENSE".into(), b"fixture license".to_vec())]),
            Some(FOUNTAIN_REVISION),
        );
        std::fs::write(dir.path().join("forgejo-source.tar"), &fountain).unwrap();
        c.forgejo_source_sha256 = identity(&fountain).sha256;

        // Independently released upstream inputs also need real manifests for
        // anonymous publication readback, without publishing them as Soda images.
        for name in ["fountain", "tailscale"] {
            let image = oci(&BTreeMap::new(), &p.base, &labels(&p.base, name));
            if name == "fountain" {
                let repo = p.fountain.image.split('@').next().unwrap();
                p.fountain.image = format!("{repo}@{}", image.image.manifest);
                p.fountain.image_config = image.image.config.clone();
                if let soda_build_tools::payload::ImageProvenance::Soda { base, .. } =
                    &mut p.images.forgejo.provenance
                {
                    *base = p.fountain.image.clone();
                }
            } else {
                let binding = &mut p.optional_extensions.tailnet.images.tailscale;
                let repo = binding.reference.split('@').next().unwrap();
                binding.reference = format!("{repo}@{}", image.image.manifest);
                binding.manifest = image.image.manifest.clone();
                binding.config = image.image.config.clone();
            }
            std::fs::write(dir.path().join(format!("images/{name}.oci")), &image.bytes).unwrap();
        }

        let terminal = b"fixture terminal".to_vec();
        p.project_support.sha256 = identity(&terminal).sha256;
        let host_packages = b"coreos 0:1-1.x86_64\n".to_vec();
        p.host_packages_sha256 = identity(&host_packages).sha256;
        let package_bytes = b"fixture package";
        for package in &mut p.extension_catalog.packages {
            package.sha256 = identity(package_bytes).sha256;
            package.size = package_bytes.len() as u64;
        }
        let catalog = p.projected_catalog_bytes().unwrap();
        std::fs::create_dir_all(dir.path().join("packages")).unwrap();
        std::fs::write(dir.path().join("packages/catalog.json"), &catalog).unwrap();
        let mut all = BTreeMap::new();
        for (name, digest) in &mut c.content_sha256 {
            let bytes = if name.starts_with("package:") {
                package_bytes.to_vec()
            } else if name == "release:fountain-release.json" {
                serde_json::to_vec(&p.fountain).unwrap()
            } else if name == "host:/usr/share/soda/extensions/catalog.json" {
                catalog.clone()
            } else {
                format!("fixture {name}").into_bytes()
            };
            *digest = identity(&bytes).sha256;
            if let Some(package) = name.strip_prefix("package:") {
                std::fs::create_dir_all(dir.path().join("packages")).unwrap();
                std::fs::write(dir.path().join("packages").join(package), &bytes).unwrap();
            } else if let Some(release) = name.strip_prefix("release:") {
                std::fs::write(dir.path().join(release), &bytes).unwrap();
            }
            all.insert(name.clone(), bytes);
        }
        let mut host_files: BTreeMap<String, Vec<u8>> = all
            .iter()
            .filter_map(|(name, bytes)| {
                name.strip_prefix("host:")
                    .map(|name| (name.to_owned(), bytes.clone()))
            })
            .collect();
        host_files.insert(p.project_support.destination.clone(), terminal.clone());
        host_files.insert(p.project_support.payload_path.clone(), terminal.clone());
        host_files.insert(
            "/usr/share/soda/host-image/packages.txt".into(),
            host_packages,
        );

        for role in crate::payload::Role::iter() {
            let name = role.name();
            let binding = role.binding_mut(&mut p.images);
            let prefix = format!("{name}:");
            let mut files: BTreeMap<String, Vec<u8>> = all
                .iter()
                .filter_map(|(path, bytes)| {
                    path.strip_prefix(&prefix)
                        .map(|path| (path.to_owned(), bytes.clone()))
                })
                .collect();
            if name == "project-os" {
                files.insert(p.project_support.destination.clone(), terminal.clone());
            }
            let image = oci(&files, &p.base, &labels(&p.base, name));
            binding.config = image.image.config.clone();
            binding.manifest = image.image.manifest.clone();
            let repo = binding
                .reference
                .rsplit_once('@')
                .map(|(repo, _)| repo.to_string())
                .unwrap_or_else(|| format!("{}-{name}", c.repository_prefix));
            binding.reference = format!("{repo}@{}", image.image.manifest);
            c.image_archives
                .insert(name.to_string(), identity(&image.bytes).sha256);
            std::fs::write(dir.path().join(format!("images/{name}.oci")), &image.bytes).unwrap();
        }
        p.project_support.project_image = crate::payload::Role::ProjectOs
            .binding(&p.images)
            .config
            .clone();
        p.validate().unwrap();
        let mut payload = serde_json::to_vec_pretty(&p).unwrap();
        payload.push(b'\n');
        c.payload_sha256 = identity(&payload).sha256;
        host_files.insert("/usr/share/soda/release.json".into(), payload.clone());
        let mut inventory = serde_json::to_vec_pretty(&c.content_sha256).unwrap();
        inventory.push(b'\n');
        host_files.insert("/usr/share/soda/host-image/content.json".into(), inventory);
        let host = oci(&host_files, &p.base, &labels(&p.base, "host"));
        c.host = host.image;
        c.host_archive_sha256 = identity(&host.bytes).sha256;
        c.host_reference = format!("{}-host@{}", c.repository_prefix, c.host.manifest);
        c.validate(&p, &payload).unwrap();
        std::fs::write(dir.path().join("host.oci"), host.bytes).unwrap();
        std::fs::write(dir.path().join("payload.json"), payload).unwrap();
        let candidate = serde_json::to_vec_pretty(&c).unwrap();
        std::fs::write(dir.path().join("candidate.json"), &candidate).unwrap();
        for (installed, input) in [
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
            (
                "/usr/lib/systemd/system/libpod-.scope.d/soda-docker.conf",
                "system/docker/project-scope.conf",
            ),
            (
                "/usr/lib/systemd/system/run-.scope.d/soda-docker.conf",
                "system/docker/network-scope.conf",
            ),
        ] {
            host_files.insert(installed.into(), source[input].clone());
        }
        let packages = b"podman 0:5.4.0-1.x86_64\nsystemd 0:257-1.x86_64\n";
        host_files.insert(PACKAGES_INSTALLED.into(), packages.to_vec());
        std::fs::write(dir.path().join(PACKAGES_PATH), packages).unwrap();
        std::fs::write(
            dir.path().join(COMPOSE_PATH),
            &source["system/docker/compose.yaml"],
        )
        .unwrap();
        let base = format!(
            "registry.fedoraproject.org/fedora@sha256:{}",
            "4".repeat(64)
        );
        let mut runtime_labels = labels(&base, "docker");
        for (name, value) in [
            ("io.soda.runtime", "docker".into()),
            ("org.opencontainers.image.version", "v0.1.0".into()),
            (
                "io.soda.build-purpose",
                if purpose == BuildPurpose::Production {
                    "production"
                } else {
                    "development"
                }
                .into(),
            ),
            ("io.soda.distribution", "docker".into()),
            ("io.soda.candidate-sha256", identity(&candidate).sha256),
            ("io.soda.payload-sha256", c.payload_sha256),
            (
                "io.soda.docker.recipe-sha256",
                identity(&source["system/docker/Dockerfile"]).sha256,
            ),
        ] {
            runtime_labels.insert(name.into(), value);
        }
        let fixture = Self {
            dir,
            _build: build,
            request: DockerRequest {
                release_tag: "v0.1.0".into(),
                purpose,
                distribution: Distribution::Docker,
                fedora_base: base,
            },
            runtime: host_files,
            runtime_labels,
        };
        fixture.rebuild();
        fixture
    }
    fn path(&self) -> &str {
        self.dir.path().to_str().unwrap()
    }
    fn completion(&self, metadata: &DockerMetadata) {
        let output = self.dir.path().parent().unwrap().join("evidence");
        std::fs::create_dir(&output).unwrap();
        let result = soda_build_tools::reader::build::BuildResult {
            revision: metadata.soda_revision.clone(),
            architecture: "x86_64".into(),
            candidate: self
                .dir
                .path()
                .join("candidate.json")
                .to_string_lossy()
                .into_owned(),
            candidate_sha256: metadata.candidate_sha256.clone(),
            host_manifest: metadata.host_manifest.clone(),
            payload_sha256: metadata.payload_sha256.clone(),
            scope: "candidate-derived packaging; not a qualified release".into(),
            purpose: "production".into(),
            requested_target: "release".into(),
            completed_target: if self.request.distribution == Distribution::All {
                "all"
            } else {
                "docker"
            }
            .into(),
            distribution: if self.request.distribution == Distribution::All {
                "all"
            } else {
                "docker"
            }
            .into(),
            release_tag: metadata.release_tag.clone(),
            checks: vec![
                "Docker OCI identity, candidate content, provenance and RPM inventory".into(),
            ],
            docker: self
                .dir
                .path()
                .join(METADATA_PATH)
                .to_string_lossy()
                .into_owned(),
            ..Default::default()
        };
        let bytes = serde_json::to_vec(&result).unwrap();
        std::fs::write(output.join("build.json"), &bytes).unwrap();
        let completion = soda_build_tools::reader::build::BuildCompletion {
            build_sha256: identity(&bytes).sha256,
        };
        std::fs::write(
            output.join("completed.json"),
            serde_json::to_vec(&completion).unwrap(),
        )
        .unwrap();
    }
    fn rebuild(&self) {
        let image = oci(
            &self.runtime,
            &self.request.fedora_base,
            &self.runtime_labels,
        );
        std::fs::write(self.dir.path().join(IMAGE_PATH), image.bytes).unwrap();
    }
}

#[test]
fn record_and_publisher_admission_bind_real_archives_without_media() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let metadata = record(fixture.path(), &fixture.request).unwrap();
    assert_eq!(admit_publication(fixture.path()).unwrap(), metadata);
    assert_eq!(
        metadata.reference().unwrap(),
        format!("{DOCKER_REPOSITORY}@{}", metadata.image.manifest)
    );
    metadata.verify_registry(&metadata.image).unwrap();
    let mut wrong = metadata.image.clone();
    wrong.config = format!("sha256:{}", "0".repeat(64));
    assert!(metadata.verify_registry(&wrong).is_err());
    crate::check::check_candidate(fixture.path(), "x86_64", SODA_REVISION, FOUNTAIN_REVISION)
        .unwrap();
    let original = std::fs::read(fixture.dir.path().join(METADATA_PATH)).unwrap();
    assert!(record(fixture.path(), &fixture.request).is_err());
    assert_eq!(
        std::fs::read(fixture.dir.path().join(METADATA_PATH)).unwrap(),
        original
    );
    assert!(!fixture.dir.path().join("media").exists());
}

#[test]
fn changed_shipping_bytes_and_source_files_are_refused() {
    let fixture = Fixture::new(BuildPurpose::Production);
    record(fixture.path(), &fixture.request).unwrap();
    for file in [
        "source.tar",
        "forgejo-source.tar",
        "host.oci",
        "payload.json",
        "candidate.json",
        "images/dashboard.oci",
        "packages/catalog.json",
        "packages/soda-extension-0.1.0-linux-x86_64.tar.gz",
        "packages/tailnet-extension-0.1.0-linux-x86_64.tar.gz",
        "fountain-release.json",
        IMAGE_PATH,
        PACKAGES_PATH,
        COMPOSE_PATH,
    ] {
        let path = fixture.dir.path().join(file);
        let original = std::fs::read(&path).unwrap();
        std::fs::write(&path, b"changed").unwrap();
        assert!(verify(fixture.path()).is_err(), "accepted changed {file}");
        std::fs::write(path, original).unwrap();
    }
    verify(fixture.path()).unwrap();
}

#[test]
fn coherent_outer_image_with_changed_payload_is_not_accepted() {
    let mut fixture = Fixture::new(BuildPurpose::Production);
    fixture.runtime.insert(
        "/usr/share/soda/release.json".into(),
        b"different payload".to_vec(),
    );
    fixture.rebuild();
    assert!(record(fixture.path(), &fixture.request)
        .unwrap_err()
        .0
        .contains("content differs"));
    assert!(!fixture.dir.path().join(METADATA_PATH).exists());
}

#[test]
fn shutdown_scope_dropins_must_match_archived_source() {
    for path in [
        "/usr/lib/systemd/system/libpod-.scope.d/soda-docker.conf",
        "/usr/lib/systemd/system/run-.scope.d/soda-docker.conf",
    ] {
        let mut fixture = Fixture::new(BuildPurpose::Production);
        fixture
            .runtime
            .insert(path.into(), b"different ordering".to_vec());
        fixture.rebuild();
        assert!(record(fixture.path(), &fixture.request)
            .unwrap_err()
            .0
            .contains("content differs"));
        assert!(!fixture.dir.path().join(METADATA_PATH).exists());
    }
}

#[test]
fn changed_catalog_cannot_hide_behind_unchanged_candidate() {
    let mut fixture = Fixture::new(BuildPurpose::Production);
    assert!(!fixture
        .runtime
        .keys()
        .any(|path| path.starts_with("/usr/share/soda/images/")));
    fixture.runtime.insert(
        "/usr/share/soda/extensions/catalog.json".into(),
        b"changed catalog".to_vec(),
    );
    fixture.rebuild();
    assert!(record(fixture.path(), &fixture.request)
        .unwrap_err()
        .0
        .contains("content differs"));
}

#[test]
fn archived_source_revision_and_fountain_pin_are_checked_before_recording() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let root = Root::open(fixture.path()).unwrap();
    let mut inputs = source::read_inputs(&root).unwrap();
    let path = fixture.dir.path().join("source.tar");
    std::fs::write(&path, tar(&inputs, Some(FOUNTAIN_REVISION))).unwrap();
    assert!(record(fixture.path(), &fixture.request)
        .unwrap_err()
        .0
        .contains("source archive revision mismatch"));
    inputs.insert(
        "fountain-source.json".into(),
        serde_json::to_vec(&json!({"repository":"fixture/fountain","revision":SODA_REVISION}))
            .unwrap(),
    );
    std::fs::write(path, tar(&inputs, Some(SODA_REVISION))).unwrap();
    assert!(record(fixture.path(), &fixture.request)
        .unwrap_err()
        .0
        .contains("pin differs"));
    assert!(!fixture.dir.path().join(METADATA_PATH).exists());
}

#[test]
fn all_distribution_preserves_the_frozen_target_selection() {
    let mut fixture = Fixture::new(BuildPurpose::Production);
    fixture.request.distribution = Distribution::All;
    fixture
        .runtime_labels
        .insert("io.soda.distribution".into(), "all".into());
    fixture.rebuild();
    record(fixture.path(), &fixture.request).unwrap();
    assert_eq!(
        admit_publication(fixture.path()).unwrap().distribution,
        Distribution::All
    );
}

#[test]
fn strict_metadata_and_frozen_input_labels_are_required() {
    let mut fixture = Fixture::new(BuildPurpose::Production);
    let metadata = record(fixture.path(), &fixture.request).unwrap();
    let original = serde_json::to_vec(&metadata).unwrap();
    for field in [
        "Format",
        "SodaRevision",
        "CandidateSHA256",
        "FedoraBase",
        "ImageArchive",
        "Compose",
    ] {
        let mut value: Value = serde_json::from_slice(&original).unwrap();
        value.as_object_mut().unwrap().remove(field);
        std::fs::write(
            fixture.dir.path().join(METADATA_PATH),
            serde_json::to_vec(&value).unwrap(),
        )
        .unwrap();
        assert!(verify(fixture.path()).is_err(), "accepted missing {field}");
    }
    for bytes in [
        br#"{"Format":1,"Format":1}"#.as_slice(),
        br#"{"Format":1,"\u0046ormat":1}"#,
        br#"{} trailing"#,
    ] {
        std::fs::write(fixture.dir.path().join(METADATA_PATH), bytes).unwrap();
        assert!(verify(fixture.path()).is_err());
    }
    std::fs::write(fixture.dir.path().join(METADATA_PATH), &original).unwrap();
    let mut value: Value = serde_json::from_slice(&original).unwrap();
    value["Unrecognized"] = json!(true);
    std::fs::write(
        fixture.dir.path().join(METADATA_PATH),
        serde_json::to_vec(&value).unwrap(),
    )
    .unwrap();
    assert!(verify(fixture.path()).is_err());
    std::fs::remove_file(fixture.dir.path().join(METADATA_PATH)).unwrap();
    fixture
        .runtime_labels
        .insert("io.soda.docker.recipe-sha256".into(), "0".repeat(64));
    fixture.rebuild();
    assert!(record(fixture.path(), &fixture.request)
        .unwrap_err()
        .0
        .contains("provenance label mismatch"));
}

#[test]
fn development_outputs_cannot_be_relabelled_for_publication() {
    let fixture = Fixture::new(BuildPurpose::Development);
    let mut metadata = record(fixture.path(), &fixture.request).unwrap();
    verify(fixture.path()).unwrap();
    assert!(admit_publication(fixture.path())
        .unwrap_err()
        .0
        .contains("development"));
    metadata.purpose = BuildPurpose::Production;
    std::fs::write(
        fixture.dir.path().join(METADATA_PATH),
        serde_json::to_vec(&metadata).unwrap(),
    )
    .unwrap();
    assert!(admit_publication(fixture.path())
        .unwrap_err()
        .0
        .contains("provenance label mismatch"));
}

#[test]
fn linked_artifacts_and_conflicting_platforms_are_refused() {
    let mut fixture = Fixture::new(BuildPurpose::Production);
    let metadata = record(fixture.path(), &fixture.request).unwrap();
    let mut wrong = metadata.clone();
    wrong.platform = "linux/arm64".into();
    assert!(wrong.validate().is_err());
    wrong = metadata.clone();
    wrong.release_tag = "latest".into();
    assert!(wrong.validate().is_err());
    wrong = metadata.clone();
    wrong.fedora_base = "registry.fedoraproject.org/fedora:44".into();
    assert!(wrong.validate().is_err());
    wrong = metadata;
    wrong.distribution = Distribution::Appliance;
    assert!(wrong.validate().is_err());
    let packages = fixture.dir.path().join(PACKAGES_PATH);
    let retained = fixture.dir.path().join("retained-packages");
    std::fs::rename(&packages, &retained).unwrap();
    symlink(&retained, &packages).unwrap();
    assert!(verify(fixture.path()).is_err());
    std::fs::remove_file(&packages).unwrap();
    std::fs::rename(retained, packages).unwrap();
    fixture
        .runtime_labels
        .insert("org.opencontainers.image.revision".into(), "b".repeat(40));
    fixture.rebuild();
    assert!(verify(fixture.path())
        .unwrap_err()
        .0
        .contains("revision mismatch"));
}

#[path = "registry_tests.rs"]
mod registry_tests;

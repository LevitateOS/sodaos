use super::*;
use crate::publish::{registry, GitHub};
use std::cell::RefCell;
use std::collections::VecDeque;
use std::os::unix::fs::DirBuilderExt;
use std::path::Path;

struct Commands {
    replies: RefCell<VecDeque<Result<Vec<u8>, Error>>>,
    calls: RefCell<Vec<Vec<String>>>,
    digest: String,
    components: BTreeMap<String, (Vec<u8>, Vec<u8>)>,
}
impl Commands {
    fn new(replies: Vec<Result<Vec<u8>, Error>>, digest: &str) -> Self {
        Self {
            replies: RefCell::new(replies.into()),
            calls: RefCell::new(vec![]),
            digest: digest.into(),
            components: BTreeMap::new(),
        }
    }
    fn with_components(mut self, fixture: &Fixture) -> Self {
        let payload: Payload = serde_json::from_slice(
            &std::fs::read(fixture.dir.path().join("payload.json")).unwrap(),
        )
        .unwrap();
        let mut inputs: Vec<_> = crate::payload::Role::iter()
            .map(|role| {
                let image = role.binding(&payload.images);
                (
                    role.name().to_string(),
                    image.reference.clone(),
                    image.manifest.clone(),
                )
            })
            .collect();
        inputs.push((
            "fountain".into(),
            payload.fountain.image.clone(),
            payload.fountain.image.split('@').nth(1).unwrap().into(),
        ));
        let image = &payload.optional_extensions.tailnet.images.tailscale;
        inputs.push((
            "tailscale".into(),
            image.reference.clone(),
            image.manifest.clone(),
        ));
        for (name, reference, digest) in inputs {
            let file =
                std::fs::File::open(fixture.dir.path().join(format!("images/{name}.oci"))).unwrap();
            let mut archive = tar::Archive::new(file);
            let mut raw = vec![];
            for entry in archive.entries().unwrap() {
                let mut entry = entry.unwrap();
                if entry.path().unwrap().as_ref()
                    == Path::new(&format!("blobs/sha256/{}", &digest[7..]))
                {
                    entry.read_to_end(&mut raw).unwrap();
                }
            }
            assert!(!raw.is_empty());
            let observed =
                serde_json::to_vec(&json!({"Digest":digest,"Architecture":"amd64","Os":"linux"}))
                    .unwrap();
            if ["dashboard", "forgejo", "project-os", "canlang-runtime"].contains(&name.as_str()) {
                self.components.insert(
                    format!(
                        "docker://{}:{}",
                        reference.split('@').next().unwrap(),
                        payload.release_tag
                    ),
                    (raw.clone(), observed.clone()),
                );
            }
            self.components
                .insert(format!("docker://{reference}"), (raw, observed));
        }
        self
    }
    fn run(&self, args: &[String]) -> Result<Vec<u8>, Error> {
        self.calls.borrow_mut().push(args.to_vec());
        if args.first().map(String::as_str) == Some("inspect") {
            if let Some((raw, observed)) = self.components.get(args.last().unwrap()) {
                assert!(args.contains(&"--no-creds".into()));
                return Ok(if args.contains(&"--raw".into()) {
                    raw
                } else {
                    observed
                }
                .clone());
            }
        }
        if args.first().map(String::as_str) == Some("copy") {
            let index = args.iter().position(|arg| arg == "--digestfile").unwrap();
            if let Some((_, observed)) = self.components.get(args.last().unwrap()) {
                let identity: Value = serde_json::from_slice(observed).unwrap();
                std::fs::write(&args[index + 1], identity["Digest"].as_str().unwrap()).unwrap();
                return Ok(vec![]);
            }
            std::fs::write(&args[index + 1], &self.digest).unwrap();
        }
        self.replies
            .borrow_mut()
            .pop_front()
            .expect("unexpected command")
    }
}

#[test]
fn complete_staging_publishes_four_apps_and_outer_then_verifies_without_uploads() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let (output, manifest, metadata) = prepared(&fixture);
    let mut replies: Vec<_> = ci(&metadata, true)
        .replies
        .into_inner()
        .into_iter()
        .take(2)
        .collect();
    replies.extend([
        Ok(br#"{"enabled":true}"#.to_vec()),
        Err(Error::msg("GitHub resource not found")),
        Ok(serde_json::to_vec(&json!({"ref":format!("refs/tags/{}",manifest.tag),"object":{"type":"commit","sha":manifest.soda_revision}})).unwrap()),
    ]);
    for _ in 0..4 {
        replies.push(Err(Error::msg("GitHub resource not found")));
    }
    replies.extend(ci(&metadata, true).replies.into_inner());
    let mut inventory = manifest.assets.clone();
    for name in ["release.json", "SHA256SUMS"] {
        let bytes = std::fs::read(output.join(name)).unwrap();
        inventory.push(crate::publish::assets::Asset {
            name: name.into(),
            sha256: identity(&bytes).sha256,
            bytes: bytes.len() as u64,
        });
    }
    let remote = |draft: bool| {
        Ok(serde_json::to_vec(&json!({"id":17,"tag_name":manifest.tag,"draft":draft,"prerelease":true,"immutable":!draft,"assets":inventory.iter().map(|asset|json!({"name":asset.name,"size":asset.bytes,"digest":format!("sha256:{}",asset.sha256),"state":"uploaded"})).collect::<Vec<_>>()})).unwrap())
    };
    replies.extend([
        Ok(vec![]),
        Ok(vec![]),
        Ok(br#"{"databaseId":17}"#.to_vec()),
        remote(true),
        Ok(vec![]),
        Ok(br#"{"databaseId":17}"#.to_vec()),
        remote(false),
        Ok(vec![]),
    ]);
    let gh = Commands::new(replies, "");
    let mut observations = vec![Ok(vec![])]; // Outer upload; four app copies are handled independently.
    observations.extend(readback(&fixture, &metadata));
    let registry = Commands::new(observations, &metadata.image.manifest).with_components(&fixture);
    crate::publish::stage(&gh, &registry, &output, Some(&auth(fixture.dir.path()))).unwrap();
    assert!(gh.replies.borrow().is_empty());
    let copies: Vec<_> = registry
        .calls
        .borrow()
        .iter()
        .filter(|args| args[0] == "copy")
        .cloned()
        .collect();
    assert_eq!(copies.len(), 5);
    for role in ["dashboard", "forgejo", "project-os", "canlang-runtime"] {
        assert!(copies.iter().any(|args| args.last().unwrap()
            == &format!("docker://ghcr.io/levitateos/sodaos-{role}:v0.1.0")));
    }
    assert!(copies
        .iter()
        .all(|args| args.contains(&"--preserve-digests".into())));
    assert!(output.join("components/public.json").is_file());
    let mut ci_replies: Vec<_> = ci(&metadata, true)
        .replies
        .into_inner()
        .into_iter()
        .take(2)
        .collect();
    ci_replies.extend(ci(&metadata, true).replies.into_inner().into_iter().take(2));
    let gh = Commands::new(ci_replies, "");
    let registry = Commands::new(readback(&fixture, &metadata), "").with_components(&fixture);
    crate::publish::verify_registry(&gh, &registry, &output).unwrap();
    assert!(registry
        .calls
        .borrow()
        .iter()
        .all(|args| args[0] == "inspect"));
}

#[test]
fn component_staging_requires_auth_and_preserves_partial_attempt_without_remote_mutation() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let (output, _, _) = prepared(&fixture);
    let gh = Commands::new(vec![], "");
    let registry = Commands::new(vec![], "");
    assert!(crate::publish::stage(&gh, &registry, &output, None).is_err());
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(output.join("components"))
        .unwrap();
    let error = crate::publish::stage(&gh, &registry, &output, Some(&auth(fixture.dir.path())))
        .unwrap_err();
    assert!(error.0.contains("already attempted"));
    assert!(output.join("components").is_dir());
    assert!(gh.calls.borrow().is_empty());
    assert!(registry.calls.borrow().is_empty());
}
impl GitHub for Commands {
    fn run(&self, args: &[String]) -> Result<Vec<u8>, Error> {
        self.run(args)
    }
}
impl registry::Registry for Commands {
    fn run(&self, args: &[String]) -> Result<Vec<u8>, Error> {
        self.run(args)
    }
}
fn ci(metadata: &DockerMetadata, package_missing: bool) -> Commands {
    let mut replies = vec![
        Ok(serde_json::to_vec(&json!({"revision":metadata.forgejo_revision})).unwrap()),
        Ok(serde_json::to_vec(&json!({"workflow_runs":[{"id":23,"head_sha":metadata.soda_revision,"head_branch":"main","event":"push","path":".github/workflows/ci.yml","status":"completed","conclusion":"success"}]})).unwrap()),
    ];
    replies.push(if package_missing {
        Err(Error::msg("GitHub resource not found"))
    } else {
        Ok(b"{}".to_vec())
    });
    Commands::new(replies, "")
}
fn readback(fixture: &Fixture, metadata: &DockerMetadata) -> Vec<Result<Vec<u8>, Error>> {
    let image = oci(
        &fixture.runtime,
        &fixture.request.fedora_base,
        &fixture.runtime_labels,
    );
    let manifest = image.blobs[&format!("blobs/sha256/{}", &metadata.image.manifest[7..])].clone();
    vec![Ok(manifest), Ok(serde_json::to_vec(&json!({"Digest":metadata.image.manifest,"Architecture":"amd64","Os":"linux","Labels":fixture.runtime_labels})).unwrap())]
}
fn auth(dir: &Path) -> std::path::PathBuf {
    let path = dir.join("registry-auth.json");
    crate::buildx::write_new(path.to_str().unwrap(), b"{\"auths\":{}}", 0o600).unwrap();
    path
}

#[test]
fn candidate_upload_preserves_digest_and_records_anonymous_readback() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let metadata = record(fixture.path(), &fixture.request).unwrap();
    let gh = ci(&metadata, true);
    let mut replies = vec![Ok(vec![])];
    replies.extend(readback(&fixture, &metadata));
    let registry = Commands::new(replies, &metadata.image.manifest);
    let out = fixture.dir.path().join("publication");
    let result = registry::stage(
        &gh,
        &registry,
        fixture.dir.path(),
        &auth(fixture.dir.path()),
        &out,
    )
    .unwrap();
    assert_eq!(result.reference, metadata.reference().unwrap());
    assert!(result.anonymous && !result.qualified);
    assert_eq!(result.image, metadata.image);
    assert!(out.join("registry.json").is_file());
    let calls = registry.calls.borrow();
    assert!(calls[0].contains(&"--preserve-digests".into()));
    assert!(calls[0].contains(&"--dest-authfile".into()));
    assert!(calls[1].contains(&"--no-creds".into()));
    assert!(calls[2].contains(&"--no-creds".into()));
    assert!(!calls
        .iter()
        .flatten()
        .any(|arg| arg.contains(":latest") || arg == "--creds"));
}

#[test]
fn development_and_failed_main_ci_never_reach_registry() {
    for purpose in [BuildPurpose::Development, BuildPurpose::Production] {
        let fixture = Fixture::new(purpose);
        let metadata = record(fixture.path(), &fixture.request).unwrap();
        let gh = Commands::new(
            vec![
                Ok(serde_json::to_vec(&json!({"revision":metadata.forgejo_revision})).unwrap()),
                Ok(b"{\"workflow_runs\":[]}".to_vec()),
            ],
            "",
        );
        let registry = Commands::new(vec![], "");
        let out = fixture.dir.path().join("publication");
        assert!(registry::stage(
            &gh,
            &registry,
            fixture.dir.path(),
            &auth(fixture.dir.path()),
            &out
        )
        .is_err());
        assert!(registry.calls.borrow().is_empty());
        assert!(!out.exists());
    }
}

#[test]
fn existing_version_is_not_overwritten() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let metadata = record(fixture.path(), &fixture.request).unwrap();
    let registry = Commands::new(vec![Ok(vec![])], "");
    let out = fixture.dir.path().join("publication");
    assert!(registry::stage(
        &ci(&metadata, false),
        &registry,
        fixture.dir.path(),
        &auth(fixture.dir.path()),
        &out
    )
    .unwrap_err()
    .0
    .contains("already exists"));
    assert_eq!(registry.calls.borrow().len(), 1);
    assert_eq!(registry.calls.borrow()[0][0], "inspect");
    assert!(!out.exists());
}

#[test]
fn private_or_changed_public_image_cannot_receive_receipt() {
    for changed in [false, true] {
        let fixture = Fixture::new(BuildPurpose::Production);
        let metadata = record(fixture.path(), &fixture.request).unwrap();
        let registry = Commands::new(
            vec![
                Ok(vec![]),
                if changed {
                    Ok(b"{}".to_vec())
                } else {
                    Err(Error::msg("private package"))
                },
            ],
            &metadata.image.manifest,
        );
        let out = fixture.dir.path().join("publication");
        assert!(registry::stage(
            &ci(&metadata, true),
            &registry,
            fixture.dir.path(),
            &auth(fixture.dir.path()),
            &out
        )
        .is_err());
        assert!(!out.join("registry.json").exists());
        assert!(out.join("pushed.digest").exists());
    }
}

#[test]
fn independent_verification_never_repeats_upload() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let metadata = record(fixture.path(), &fixture.request).unwrap();
    let registry = Commands::new(readback(&fixture, &metadata), "");
    let out = fixture.dir.path().join("publication");
    std::fs::create_dir(&out).unwrap();
    use std::os::unix::fs::PermissionsExt;
    std::fs::set_permissions(&out, std::fs::Permissions::from_mode(0o700)).unwrap();
    registry::verify(&ci(&metadata, true), &registry, fixture.dir.path(), &out).unwrap();
    assert!(registry
        .calls
        .borrow()
        .iter()
        .all(|args| args[0] == "inspect" && args.contains(&"--no-creds".into())));
}

#[test]
fn registry_verification_finishes_partial_receipt_and_is_repeatable() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let metadata = record(fixture.path(), &fixture.request).unwrap();
    let out = fixture.dir.path().join("publication");
    std::fs::create_dir(&out).unwrap();
    use std::os::unix::fs::PermissionsExt;
    std::fs::set_permissions(&out, std::fs::Permissions::from_mode(0o700)).unwrap();
    let raw = readback(&fixture, &metadata).remove(0).unwrap();
    crate::buildx::write_new(
        out.join("public-manifest.json").to_str().unwrap(),
        &raw,
        0o600,
    )
    .unwrap();
    for _ in 0..2 {
        let registry = Commands::new(readback(&fixture, &metadata), "");
        registry::verify(&ci(&metadata, true), &registry, fixture.dir.path(), &out).unwrap();
        assert!(registry
            .calls
            .borrow()
            .iter()
            .all(|args| args[0] == "inspect"));
    }
    let receipt = std::fs::read(out.join("registry.json")).unwrap();
    std::fs::write(out.join("public-manifest.json"), b"different").unwrap();
    let registry = Commands::new(readback(&fixture, &metadata), "");
    assert!(registry::verify(&ci(&metadata, true), &registry, fixture.dir.path(), &out).is_err());
    assert_eq!(std::fs::read(out.join("registry.json")).unwrap(), receipt);
}

fn prepared(fixture: &Fixture) -> (std::path::PathBuf, crate::publish::Manifest, DockerMetadata) {
    let metadata = record(fixture.path(), &fixture.request).unwrap();
    fixture.completion(&metadata);
    let root = fixture.dir.path();
    std::fs::create_dir(root.join("forgejo-context")).unwrap();
    for (name, body) in [
        ("app-inputs.json", b"{}".as_slice()),
        ("packages.txt", b"coreos 0:1-1.x86_64\n".as_slice()),
        ("forgejo-context/presentation.json", b"{}".as_slice()),
        ("notes.md", b"Native candidate".as_slice()),
    ] {
        std::fs::write(root.join(name), body).unwrap();
    }
    if fixture.request.distribution == Distribution::All {
        std::fs::create_dir(root.join("media")).unwrap();
        let rootfs = format!("{}-rootfs.img", identity(b"rootfs").sha256);
        std::fs::write(root.join("media/installer.iso"), b"iso").unwrap();
        std::fs::write(root.join("media").join(&rootfs), b"rootfs").unwrap();
        let document = json!({"Revision":metadata.soda_revision,"Architecture":"x86_64","HostManifest":metadata.host_manifest,"PayloadSHA256":metadata.payload_sha256,"RootfsURL":format!("https://github.com/LevitateOS/sodaos/releases/download/v0.1.0/{rootfs}"),"ISO":{"Path":"installer.iso","SHA256":identity(b"iso").sha256,"Bytes":3},"Rootfs":{"Path":rootfs,"SHA256":identity(b"rootfs").sha256,"Bytes":6}});
        std::fs::write(
            root.join("media/media.json"),
            serde_json::to_vec(&document).unwrap(),
        )
        .unwrap();
    }
    let output = root.join("prepared");
    let manifest = crate::publish::prepare(
        root,
        "v0.1.0",
        fixture.request.distribution,
        &root.join("notes.md"),
        &output,
    )
    .unwrap();
    (output, manifest, metadata)
}
fn evidence(
    fixture: &Fixture,
    output: &Path,
    manifest: &crate::publish::Manifest,
    metadata: &DockerMetadata,
) -> std::path::PathBuf {
    let root = fixture.dir.path().join("evidence");
    std::fs::create_dir(&root).unwrap();
    std::fs::create_dir(root.join("downloads")).unwrap();
    let payload: Payload =
        serde_json::from_slice(&std::fs::read(output.join("payload.json")).unwrap()).unwrap();
    let native = json!({"target":"soda-qualification","tag":manifest.tag,"registry_manifest":metadata.image.manifest,"registry_config":metadata.image.config,"compose_sha256":metadata.compose.sha256,"installer_sha256":manifest.assets.iter().find(|asset|asset.name=="install-docker.sh").unwrap().sha256,"installer_execution_sha256":identity(b"native observation").sha256,"candidate_sha256":metadata.candidate_sha256,"payload_sha256":metadata.payload_sha256,"soda_revision":metadata.soda_revision,"forgejo_revision":metadata.forgejo_revision,"application_configs":soda_build_tools::payload::Role::iter().map(|role|(role.name().to_string(),role.binding(&payload.images).config.clone())).collect::<BTreeMap<_,_>>(),"os":"linux","architecture":"x86_64","rootful":true,"cgroup_version":2,"host_network":true,"fresh_volumes":true,"outer_id":"a".repeat(64)});
    let persistence = json!({"target":"soda-qualification","registry_manifest":metadata.image.manifest,"registry_config":metadata.image.config,"outer_before":"a".repeat(64),"outer_after":"b".repeat(64),"project_before":"c".repeat(64),"project_after":"c".repeat(64),"protected_before_sha256":"d".repeat(64),"protected_after_sha256":"d".repeat(64)});
    let factory = json!({"provider_execution_sha256":identity(b"native observation").sha256,"repository_id":1,"issue_id":2,"pull_request_id":3,"coder_actor":4,"reviewer_actor":5,"reviewed_head":"f".repeat(40),"merged_candidate_head":"f".repeat(40),"required_checks":{"CI":"success"},"elapsed_ms":1000,"funding_limit_ms":2000});
    let artifacts = manifest
        .assets
        .iter()
        .filter(|asset| {
            [
                "docker.json",
                "sodaos-docker-compose.yaml",
                "install-docker.sh",
                "candidate.json",
                "payload.json",
            ]
            .contains(&asset.name.as_str())
        })
        .map(|asset| (format!("/downloaded/{}", asset.name), asset.sha256.clone()))
        .collect::<BTreeMap<_, _>>();
    for asset in manifest.assets.iter().filter(|asset| {
        [
            "docker.json",
            "sodaos-docker-compose.yaml",
            "install-docker.sh",
            "candidate.json",
            "payload.json",
        ]
        .contains(&asset.name.as_str())
    }) {
        std::fs::copy(
            output.join(&asset.name),
            root.join("downloads").join(&asset.name),
        )
        .unwrap();
    }
    for (index, kind) in ["installation", "browser_project", "factory", "persistence"]
        .into_iter()
        .enumerate()
    {
        let folder = root.join(kind);
        std::fs::create_dir(&folder).unwrap();
        let mut files = BTreeMap::new();
        std::fs::write(folder.join("stdout.txt"), b"native observation").unwrap();
        files.insert("stdout.txt".into(), identity(b"native observation").sha256);
        if kind != "browser_project" {
            let body = serde_json::to_vec(match kind {
                "installation" => &native,
                "factory" => &factory,
                _ => &persistence,
            })
            .unwrap();
            std::fs::write(folder.join("native.json"), &body).unwrap();
            files.insert("native.json".into(), identity(&body).sha256);
        }
        let observation = soda_acceptance::report::Observation {
            owner: "P11".into(),
            requested_revision: manifest.soda_revision.clone(),
            tool_revision: manifest.soda_revision.clone(),
            requested_architecture: "x86_64".into(),
            target: "soda-qualification".into(),
            client_platform: "linux/amd64".into(),
            action: "exec".into(),
            invocation: Some(vec!["native-check".into(), kind.into()]),
            outcome: "completed".into(),
            execution: "completed".into(),
            evidence: "completed".into(),
            exit_code: Some(0),
            files: Some(files),
            artifacts: Some(artifacts.clone()),
            started: format!("2026-10-14T00:00:{:02}Z", index * 2),
            finished: format!("2026-10-14T00:00:{:02}Z", index * 2 + 1),
            ..Default::default()
        };
        std::fs::write(
            folder.join("observation.json"),
            serde_json::to_vec(&soda_acceptance::report::observation_json(&observation)).unwrap(),
        )
        .unwrap();
    }
    root
}
fn remote(
    manifest: &crate::publish::Manifest,
    output: &Path,
    prerelease: bool,
) -> Result<Vec<u8>, Error> {
    let mut assets = manifest.assets.clone();
    for name in ["release.json", "SHA256SUMS"] {
        let body = std::fs::read(output.join(name)).unwrap();
        assets.push(crate::publish::assets::Asset {
            name: name.into(),
            bytes: body.len() as u64,
            sha256: identity(&body).sha256,
        });
    }
    Ok(serde_json::to_vec(&json!({"id":17,"tag_name":manifest.tag,"draft":false,"prerelease":prerelease,"immutable":true,"published_at":"2026-10-13T00:00:00Z","assets":assets.iter().map(|asset|json!({"name":asset.name,"size":asset.bytes,"digest":format!("sha256:{}",asset.sha256),"state":"uploaded"})).collect::<Vec<_>>()})).unwrap())
}
fn promotion_gh(
    metadata: &DockerMetadata,
    manifest: &crate::publish::Manifest,
    output: &Path,
    fail_edit: bool,
) -> Commands {
    let source = ci(metadata, true);
    let mut replies = source.replies.into_inner();
    replies.pop_back();
    replies.push_front(Ok(serde_json::to_vec(&json!({"ref":format!("refs/tags/{}",metadata.release_tag),"object":{"type":"commit","sha":metadata.soda_revision}})).unwrap()));
    replies.extend([
        Ok(b"{\"databaseId\":17}".to_vec()),
        remote(manifest, output, true),
        Ok(vec![]),
    ]);
    if manifest.has_appliance() {
        replies.push_back(Ok(vec![]));
    } else {
        replies.push_back(Ok(b"{\"id\":11,\"tag_name\":\"v0.0.1\",\"draft\":false,\"prerelease\":false,\"immutable\":true,\"assets\":[]}".to_vec()));
    }
    replies.push_back(if fail_edit {
        Err(Error::msg("edit failed"))
    } else {
        Ok(vec![])
    });
    replies.extend([
        Ok(b"{\"databaseId\":17}".to_vec()),
        remote(manifest, output, false),
    ]);
    replies.push_back(if manifest.has_appliance(){remote(manifest,output,false)}else{Ok(b"{\"id\":11,\"tag_name\":\"v0.0.1\",\"draft\":false,\"prerelease\":false,\"immutable\":true,\"assets\":[]}".to_vec())});
    Commands::new(replies.into(), "")
}
#[test]
fn prepared_docker_has_exact_public_assets_and_qualification_refuses_missing_factory() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let (output, manifest, metadata) = prepared(&fixture);
    assert!(manifest.iso.is_empty());
    assert_eq!(manifest.assets.len(), 14);
    for name in [
        "install-docker.sh",
        "extensions-catalog.json",
        "soda-extension-0.1.0-linux-x86_64.tar.gz",
        "tailnet-extension-0.1.0-linux-x86_64.tar.gz",
    ] {
        assert!(manifest.assets.iter().any(|asset| asset.name == name));
    }
    assert!(manifest.docker.is_some());
    assert!(!manifest
        .assets
        .iter()
        .any(|asset| asset.name == "operator.json"));
    let root = evidence(&fixture, &output, &manifest, &metadata);
    std::fs::remove_file(root.join("factory/observation.json")).unwrap();
    assert!(crate::publish::qualification::stamp(&manifest, &output, &root).is_err());
    assert!(!output.join("docker-qualification.json").exists());
}
#[test]
fn qualified_docker_promotes_only_verified_digest_and_keeps_appliance_latest() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let (output, manifest, metadata) = prepared(&fixture);
    let root = evidence(&fixture, &output, &manifest, &metadata);
    crate::publish::qualification::stamp(&manifest, &output, &root).unwrap();
    let mut responses = readback(&fixture, &metadata);
    responses.push(Ok(vec![]));
    responses.push(readback(&fixture, &metadata).remove(0));
    let registry = Commands::new(responses, &metadata.image.manifest).with_components(&fixture);
    let gh = promotion_gh(&metadata, &manifest, &output, false);
    crate::publish::promote(&gh, &registry, &output, Some(&auth(fixture.dir.path()))).unwrap();
    let calls = registry.calls.borrow();
    let copy = calls.iter().find(|args| args[0] == "copy").unwrap();
    assert!(copy.contains(&format!("docker://{}", metadata.reference().unwrap())));
    assert!(copy.contains(&"--preserve-digests".into()));
    assert!(copy.last().unwrap().ends_with(":latest"));
    assert!(gh
        .calls
        .borrow()
        .iter()
        .any(|args| args.contains(&"--latest=false".into())));
    assert!(output.join("promotion-complete.json").exists());
    let notes = std::fs::read_to_string(output.join("qualified-release-notes.md")).unwrap();
    assert!(notes.lines().any(|line| line
        == format!(
            "<!-- soda-docker-qualified: {} {} {} -->",
            manifest.tag, manifest.soda_revision, metadata.image.manifest
        )));
}

#[test]
fn installer_is_rendered_from_bound_source_and_cannot_be_replaced_by_rehashing() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let (output, mut manifest, _) = prepared(&fixture);
    let script = std::fs::read_to_string(output.join("install-docker.sh")).unwrap();
    let metadata = manifest
        .assets
        .iter()
        .find(|asset| asset.name == "docker.json")
        .unwrap();
    assert_eq!(script, format!(
        "#!/usr/bin/env bash\nrelease='{}'\nrevision='{}'\nmetadata_sha256='{}'\nmetadata_bytes='{}'\n",
        manifest.tag, manifest.soda_revision, metadata.sha256, metadata.bytes
    ));
    crate::publish::assets::load(&output).unwrap();

    let replacement = b"#!/usr/bin/env bash\necho unbound replacement\n";
    std::fs::write(output.join("install-docker.sh"), replacement).unwrap();
    let asset = manifest
        .assets
        .iter_mut()
        .find(|asset| asset.name == "install-docker.sh")
        .unwrap();
    asset.sha256 = identity(replacement).sha256;
    asset.bytes = replacement.len() as u64;
    std::fs::write(
        output.join("release.json"),
        serde_json::to_vec(&manifest).unwrap(),
    )
    .unwrap();
    let mut sums = manifest.assets.clone();
    let release = std::fs::read(output.join("release.json")).unwrap();
    sums.push(crate::publish::assets::Asset {
        name: "release.json".into(),
        bytes: release.len() as u64,
        sha256: identity(&release).sha256,
    });
    std::fs::write(
        output.join("SHA256SUMS"),
        sums.iter()
            .map(|asset| format!("{}  {}\n", asset.sha256, asset.name))
            .collect::<String>(),
    )
    .unwrap();
    assert!(crate::publish::assets::load(&output)
        .unwrap_err()
        .0
        .contains("installer differs"));
}

#[test]
fn qualification_verifies_the_public_installer_asset_with_github() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let (output, manifest, metadata) = prepared(&fixture);
    let evidence = evidence(&fixture, &output, &manifest, &metadata);
    let mut replies = ci(&metadata, true).replies.into_inner();
    replies.pop_back();
    replies.push_front(Ok(serde_json::to_vec(&json!({"ref":format!("refs/tags/{}",metadata.release_tag),"object":{"type":"commit","sha":metadata.soda_revision}})).unwrap()));
    replies.extend([
        Ok(b"{\"databaseId\":17}".to_vec()),
        remote(&manifest, &output, true),
        Ok(vec![]),
    ]);
    replies.extend((0..5).map(|_| Ok(vec![])));
    let gh = Commands::new(replies.into(), "");
    let registry = Commands::new(readback(&fixture, &metadata), "");
    crate::publish::qualify_docker(&gh, &registry, &manifest, &output, &evidence).unwrap();
    assert!(gh.replies.borrow().is_empty());
    assert!(gh.calls.borrow().iter().any(|args| args[0] == "release"
        && args[1] == "verify-asset"
        && args[3]
            == evidence
                .join("downloads/install-docker.sh")
                .to_str()
                .unwrap()));
    assert!(output.join("docker-qualification.json").exists());
}

#[test]
fn qualification_requires_the_downloaded_installer_in_each_observation() {
    for case in ["download", "artifact", "identity", "transcript"] {
        let fixture = Fixture::new(BuildPurpose::Production);
        let (output, manifest, metadata) = prepared(&fixture);
        let root = evidence(&fixture, &output, &manifest, &metadata);
        if case == "download" {
            std::fs::write(root.join("downloads/install-docker.sh"), b"truncated").unwrap();
        } else {
            let path = root.join("installation/observation.json");
            let mut observation: Value =
                serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
            if case == "artifact" {
                observation["Artifacts"]
                    .as_object_mut()
                    .unwrap()
                    .remove("/downloaded/install-docker.sh");
            } else {
                let native_path = root.join("installation/native.json");
                let mut native: Value =
                    serde_json::from_slice(&std::fs::read(&native_path).unwrap()).unwrap();
                native[if case == "identity" {
                    "installer_sha256"
                } else {
                    "installer_execution_sha256"
                }] = json!("e".repeat(64));
                let body = serde_json::to_vec(&native).unwrap();
                std::fs::write(native_path, &body).unwrap();
                observation["Files"]["native.json"] = json!(identity(&body).sha256);
            }
            std::fs::write(path, serde_json::to_vec(&observation).unwrap()).unwrap();
        }
        assert!(
            crate::publish::qualification::stamp(&manifest, &output, &root).is_err(),
            "{case}"
        );
        assert!(!output.join("docker-qualification.json").exists());
    }
}

#[test]
fn docker_promotion_preserves_an_absent_appliance_recommendation() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let (output, manifest, metadata) = prepared(&fixture);
    let root = evidence(&fixture, &output, &manifest, &metadata);
    crate::publish::qualification::stamp(&manifest, &output, &root).unwrap();
    let mut responses = readback(&fixture, &metadata);
    responses.extend([Ok(vec![]), readback(&fixture, &metadata).remove(0)]);
    let registry = Commands::new(responses, &metadata.image.manifest).with_components(&fixture);
    let gh = promotion_gh(&metadata, &manifest, &output, false);
    for reply in gh.replies.borrow_mut().iter_mut() {
        if reply
            .as_ref()
            .is_ok_and(|bytes| String::from_utf8_lossy(bytes).contains("v0.0.1"))
        {
            *reply = Err(Error::msg("GitHub resource not found"));
        }
    }
    crate::publish::promote(&gh, &registry, &output, Some(&auth(fixture.dir.path()))).unwrap();
    assert!(output.join("promotion-complete.json").exists());
}

#[test]
fn docker_promotion_refuses_observations_from_before_publication() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let (output, manifest, metadata) = prepared(&fixture);
    let root = evidence(&fixture, &output, &manifest, &metadata);
    crate::publish::qualification::stamp(&manifest, &output, &root).unwrap();
    let gh = promotion_gh(&metadata, &manifest, &output, false);
    for reply in gh.replies.borrow_mut().iter_mut() {
        if let Ok(bytes) = reply {
            if let Ok(mut value) = serde_json::from_slice::<Value>(bytes) {
                if value.get("published_at").is_some() {
                    value["published_at"] = json!("2026-10-15T00:00:00Z");
                    *bytes = serde_json::to_vec(&value).unwrap();
                }
            }
        }
    }
    let registry = Commands::new(vec![], "");
    assert!(
        crate::publish::promote(&gh, &registry, &output, Some(&auth(fixture.dir.path()))).is_err()
    );
    assert!(registry.calls.borrow().is_empty());
    assert!(!output.join("promotion-started.json").exists());
}
#[test]
fn changed_evidence_and_half_qualified_all_refuse_before_remote_mutation() {
    let mut fixture = Fixture::new(BuildPurpose::Production);
    let (output, manifest, metadata) = prepared(&fixture);
    let root = evidence(&fixture, &output, &manifest, &metadata);
    crate::publish::qualification::stamp(&manifest, &output, &root).unwrap();
    std::fs::write(root.join("browser_project/stdout.txt"), b"changed").unwrap();
    let gh = Commands::new(vec![], "");
    let registry = Commands::new(vec![], "");
    assert!(
        crate::publish::promote(&gh, &registry, &output, Some(&auth(fixture.dir.path()))).is_err()
    );
    assert!(gh.calls.borrow().is_empty());
    assert!(registry.calls.borrow().is_empty());
    fixture = Fixture::new(BuildPurpose::Production);
    fixture.request.distribution = Distribution::All;
    fixture
        .runtime_labels
        .insert("io.soda.distribution".into(), "all".into());
    fixture.rebuild();
    let (output, manifest, metadata) = prepared(&fixture);
    let root = evidence(&fixture, &output, &manifest, &metadata);
    crate::publish::qualification::stamp(&manifest, &output, &root).unwrap();
    assert!(crate::publish::promote(&gh, &registry, &output, None).is_err());
    assert!(gh.calls.borrow().is_empty());
    assert!(registry.calls.borrow().is_empty());
}
#[test]
fn partial_promotion_is_retained_and_cannot_be_replayed() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let (output, manifest, metadata) = prepared(&fixture);
    let root = evidence(&fixture, &output, &manifest, &metadata);
    crate::publish::qualification::stamp(&manifest, &output, &root).unwrap();
    let mut replies = readback(&fixture, &metadata);
    replies.extend([Ok(vec![]), readback(&fixture, &metadata).remove(0)]);
    let registry = Commands::new(replies, &metadata.image.manifest).with_components(&fixture);
    let gh = promotion_gh(&metadata, &manifest, &output, true);
    let auth = auth(fixture.dir.path());
    assert!(
        crate::publish::promote(&gh, &registry, &output, Some(&auth))
            .unwrap_err()
            .0
            .contains("inspect status")
    );
    assert!(output.join("promotion-started.json").exists());
    assert!(!output.join("promotion-complete.json").exists());
    assert!(crate::publish::promote(
        &Commands::new(vec![], ""),
        &Commands::new(vec![], ""),
        &output,
        Some(&auth)
    )
    .unwrap_err()
    .0
    .contains("already attempted"));
}

#[test]
fn all_moves_both_recommendations_only_after_both_qualifications() {
    let mut fixture = Fixture::new(BuildPurpose::Production);
    fixture.request.distribution = Distribution::All;
    fixture
        .runtime_labels
        .insert("io.soda.distribution".into(), "all".into());
    fixture.rebuild();
    let (output, manifest, metadata) = prepared(&fixture);
    let root = evidence(&fixture, &output, &manifest, &metadata);
    crate::publish::qualification::stamp(&manifest, &output, &root).unwrap();
    let receipt = crate::publish::Qualification {
        format: 1,
        soda_revision: manifest.soda_revision.clone(),
        forgejo_revision: manifest.forgejo_revision.clone(),
        iso_sha256: identity(b"iso").sha256,
        rootfs_sha256: identity(b"rootfs").sha256,
        downloaded_iso_sha256: identity(b"iso").sha256,
        observed_soda_revision: manifest.soda_revision.clone(),
        install: "passed".into(),
        boot: "passed".into(),
        applications: "passed".into(),
    };
    std::fs::write(
        output.join("appliance-qualification.json"),
        serde_json::to_vec(&receipt).unwrap(),
    )
    .unwrap();
    std::fs::write(
        output.join("appliance-evidence.json"),
        serde_json::to_vec(&json!({"downloaded":output.join("sodaos-x86_64.iso")})).unwrap(),
    )
    .unwrap();
    let mut replies = readback(&fixture, &metadata);
    replies.extend([Ok(vec![]), readback(&fixture, &metadata).remove(0)]);
    let registry = Commands::new(replies, &metadata.image.manifest).with_components(&fixture);
    let gh = promotion_gh(&metadata, &manifest, &output, false);
    crate::publish::promote(&gh, &registry, &output, Some(&auth(fixture.dir.path()))).unwrap();
    assert!(gh
        .calls
        .borrow()
        .iter()
        .any(|args| args.contains(&"--latest=true".into())));
    assert!(output.join("promotion-complete.json").exists());
}
#[test]
fn coherent_but_unfunded_factory_evidence_is_not_qualification() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let (output, manifest, metadata) = prepared(&fixture);
    let root = evidence(&fixture, &output, &manifest, &metadata);
    let path = root.join("factory/native.json");
    let mut native: Value = serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
    native["funding_limit_ms"] = json!(0);
    let body = serde_json::to_vec(&native).unwrap();
    std::fs::write(path, &body).unwrap();
    let path = root.join("factory/observation.json");
    let mut observation: Value = serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
    observation["Files"]["native.json"] = json!(identity(&body).sha256);
    std::fs::write(path, serde_json::to_vec(&observation).unwrap()).unwrap();
    assert!(crate::publish::qualification::stamp(&manifest, &output, &root).is_err());
    assert!(!output.join("docker-qualification.json").exists());
}

#[test]
fn status_reports_partial_registry_readback_without_mutation() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let (output, _manifest, metadata) = prepared(&fixture);
    std::fs::create_dir(output.join("registry")).unwrap();
    let base = ci(&metadata, true);
    let mut replies = base.replies.into_inner();
    replies.push_back(Err(Error::msg("GitHub resource not found")));
    let gh = Commands::new(replies.into(), "");
    let registry = Commands::new(
        vec![
            Err(Error::msg("private registry package")),
            Err(Error::msg("latest absent")),
        ],
        "",
    )
    .with_components(&fixture);
    let report = crate::publish::status(&gh, &registry, &output).unwrap();
    assert_eq!(report["next_action"], "verify-images");
    assert_eq!(report["appliance_qualified"], Value::Null);
    assert_eq!(report["docker_qualified"], false);
    assert!(!output.join("docker-qualification.json").exists());
    assert!(registry
        .calls
        .borrow()
        .iter()
        .all(|args| args[0] == "inspect"));
    assert!(gh
        .calls
        .borrow()
        .iter()
        .all(|args| args[0] == "api" || args[1] == "view"));
}

#[test]
fn status_refuses_qualification_from_before_immutable_publication() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let (output, manifest, metadata) = prepared(&fixture);
    let evidence = evidence(&fixture, &output, &manifest, &metadata);
    crate::publish::qualification::stamp(&manifest, &output, &evidence).unwrap();
    let mut release: Value =
        serde_json::from_slice(&remote(&manifest, &output, true).unwrap()).unwrap();
    release["published_at"] = json!("2026-10-15T00:00:00Z");
    let mut replies = ci(&metadata, true).replies.into_inner();
    replies.pop_back();
    replies.extend([
        Ok(b"{\"databaseId\":17}".to_vec()),
        Ok(serde_json::to_vec(&release).unwrap()),
        Ok(serde_json::to_vec(&json!({"ref":format!("refs/tags/{}",metadata.release_tag),"object":{"type":"commit","sha":metadata.soda_revision}})).unwrap()),
        Ok(vec![]),
        Err(Error::msg("GitHub resource not found")),
    ]);
    let gh = Commands::new(replies.into(), "");
    let mut replies = readback(&fixture, &metadata);
    replies.push(readback(&fixture, &metadata).remove(0));
    let registry = Commands::new(replies, "").with_components(&fixture);
    let report = crate::publish::status(&gh, &registry, &output).unwrap();
    assert_eq!(report["docker_qualified"], false);
    assert_eq!(report["next_action"], "qualify");
    assert!(gh.replies.borrow().is_empty());
    assert!(registry
        .calls
        .borrow()
        .iter()
        .all(|args| args[0] == "inspect"));
}

#[test]
fn qualification_rejects_missing_stale_or_reordered_driver_records() {
    for (phase, key, value) in [
        ("installation", "ToolRevision", json!("unknown")),
        ("installation", "Invocation", Value::Null),
        ("installation", "Started", Value::Null),
        ("browser_project", "Started", json!("2026-10-14T00:00:00Z")),
        ("factory", "Finished", json!("2026-10-13T00:00:00Z")),
        ("persistence", "Finished", json!("2099-01-01T00:00:00Z")),
    ] {
        let fixture = Fixture::new(BuildPurpose::Production);
        let (output, manifest, metadata) = prepared(&fixture);
        let root = evidence(&fixture, &output, &manifest, &metadata);
        let file = root.join(phase).join("observation.json");
        let mut observation: Value =
            serde_json::from_slice(&std::fs::read(&file).unwrap()).unwrap();
        observation[key] = value;
        std::fs::write(file, serde_json::to_vec(&observation).unwrap()).unwrap();
        assert!(
            crate::publish::qualification::stamp(&manifest, &output, &root).is_err(),
            "{phase}/{key}"
        );
        assert!(!output.join("docker-qualification.json").exists());
    }
}

#[test]
fn qualification_requires_retained_execution_and_persistence_readbacks() {
    for phase in ["factory", "persistence"] {
        let fixture = Fixture::new(BuildPurpose::Production);
        let (output, manifest, metadata) = prepared(&fixture);
        let root = evidence(&fixture, &output, &manifest, &metadata);
        let file = root.join(phase).join("native.json");
        let mut native: Value = serde_json::from_slice(&std::fs::read(&file).unwrap()).unwrap();
        if phase == "factory" {
            native["provider_execution_sha256"] = json!("e".repeat(64));
        } else {
            native["outer_before"] = json!("e".repeat(64));
        }
        let bytes = serde_json::to_vec(&native).unwrap();
        std::fs::write(&file, &bytes).unwrap();
        let file = root.join(phase).join("observation.json");
        let mut observation: Value =
            serde_json::from_slice(&std::fs::read(&file).unwrap()).unwrap();
        observation["Files"]["native.json"] = json!(identity(&bytes).sha256);
        std::fs::write(file, serde_json::to_vec(&observation).unwrap()).unwrap();
        assert!(
            crate::publish::qualification::stamp(&manifest, &output, &root).is_err(),
            "{phase}"
        );
        assert!(!output.join("docker-qualification.json").exists());
    }
}

#[test]
fn qualification_atomically_records_original_evidence_and_rechecks_downloads() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let (output, manifest, metadata) = prepared(&fixture);
    let root = evidence(&fixture, &output, &manifest, &metadata);
    let expected = crate::publish::qualification::stamp(&manifest, &output, &root).unwrap();
    assert_eq!(
        crate::publish::qualification::stamp(&manifest, &output, &root).unwrap(),
        expected
    );
    let receipt = std::fs::read(output.join("docker-qualification.json")).unwrap();
    let retained: Value = serde_json::from_slice(&receipt).unwrap();
    assert_eq!(
        retained["evidence_directory"],
        json!(root.canonicalize().unwrap())
    );
    assert!(!output.join("qualification-evidence.json").exists());
    std::fs::write(
        root.join("downloads/sodaos-docker-compose.yaml"),
        b"changed compose",
    )
    .unwrap();
    assert!(crate::publish::qualification::stamp(&manifest, &output, &root).is_err());
    assert_eq!(
        std::fs::read(output.join("docker-qualification.json")).unwrap(),
        receipt
    );
}

#[test]
fn preparation_requires_completed_exact_production_build_and_valid_checksums() {
    let fixture = Fixture::new(BuildPurpose::Production);
    let (output, _, _) = prepared(&fixture);
    let receipt = fixture
        .dir
        .path()
        .parent()
        .unwrap()
        .join("evidence/build.json");
    let original = std::fs::read(&receipt).unwrap();
    let completion = receipt.parent().unwrap().join("completed.json");
    let completed = std::fs::read(&completion).unwrap();
    for (field, value) in [
        ("Purpose", "development"),
        ("RequestedTarget", "docker"),
        ("CompletedTarget", "candidate"),
        ("Distribution", "all"),
        ("CandidateSHA256", "changed"),
    ] {
        let mut result: Value = serde_json::from_slice(&original).unwrap();
        result[field] = json!(value);
        let bytes = serde_json::to_vec(&result).unwrap();
        std::fs::write(&receipt, &bytes).unwrap();
        std::fs::write(
            &completion,
            serde_json::to_vec(&json!({"BuildSHA256":identity(&bytes).sha256})).unwrap(),
        )
        .unwrap();
        let destination = fixture.dir.path().join("rejected");
        assert!(
            crate::publish::prepare(
                fixture.dir.path(),
                "v0.1.0",
                Distribution::Docker,
                &fixture.dir.path().join("notes.md"),
                &destination
            )
            .is_err(),
            "{field}"
        );
        assert!(!destination.exists());
    }
    std::fs::write(&receipt, &original).unwrap();
    // A completed producer alone cannot relabel a failed controller run.
    std::fs::remove_file(&completion).unwrap();
    let destination = fixture.dir.path().join("missing-completion");
    assert!(crate::publish::prepare(
        fixture.dir.path(),
        "v0.1.0",
        Distribution::Docker,
        &fixture.dir.path().join("notes.md"),
        &destination
    )
    .is_err());
    assert!(!destination.exists());
    std::fs::write(&completion, br#"{"BuildSHA256":"changed"}"#).unwrap();
    assert!(crate::publish::prepare(
        fixture.dir.path(),
        "v0.1.0",
        Distribution::Docker,
        &fixture.dir.path().join("notes.md"),
        &destination
    )
    .is_err());
    assert!(!destination.exists());
    std::fs::write(&completion, completed).unwrap();
    std::fs::remove_file(receipt).unwrap();
    assert!(crate::publish::prepare(
        fixture.dir.path(),
        "v0.1.0",
        Distribution::Docker,
        &fixture.dir.path().join("notes.md"),
        &fixture.dir.path().join("missing-result")
    )
    .is_err());
    std::fs::write(output.join("SHA256SUMS"), b"changed checksums").unwrap();
    assert!(crate::publish::assets::load(&output).is_err());
}

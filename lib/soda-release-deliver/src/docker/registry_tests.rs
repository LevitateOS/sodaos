use super::*;
use crate::publish::{registry, GitHub};
use std::cell::RefCell;
use std::collections::VecDeque;
use std::path::Path;

struct Commands {
    replies: RefCell<VecDeque<Result<Vec<u8>, Error>>>,
    calls: RefCell<Vec<Vec<String>>>,
    digest: String,
}
impl Commands {
    fn new(replies: Vec<Result<Vec<u8>, Error>>, digest: &str) -> Self {
        Self {
            replies: RefCell::new(replies.into()),
            calls: RefCell::new(vec![]),
            digest: digest.into(),
        }
    }
    fn run(&self, args: &[String]) -> Result<Vec<u8>, Error> {
        self.calls.borrow_mut().push(args.to_vec());
        if args.first().map(String::as_str) == Some("copy") {
            let index = args.iter().position(|arg| arg == "--digestfile").unwrap();
            std::fs::write(&args[index + 1], &self.digest).unwrap();
        }
        self.replies
            .borrow_mut()
            .pop_front()
            .expect("unexpected command")
    }
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

fn prepared(fixture: &Fixture) -> (std::path::PathBuf, crate::publish::Manifest, DockerMetadata) {
    let metadata = record(fixture.path(), &fixture.request).unwrap();
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
    let native = json!({"target":"soda-qualification","tag":manifest.tag,"registry_manifest":metadata.image.manifest,"registry_config":metadata.image.config,"compose_sha256":metadata.compose.sha256,"candidate_sha256":metadata.candidate_sha256,"payload_sha256":metadata.payload_sha256,"soda_revision":metadata.soda_revision,"forgejo_revision":metadata.forgejo_revision,"application_configs":payload.images.iter().map(|(name,image)|(name.clone(),image.config.clone())).collect::<BTreeMap<_,_>>(),"os":"linux","architecture":"x86_64","rootful":true,"cgroup_version":2,"host_network":true,"fresh_volumes":true,"outer_before":"a".repeat(64),"outer_after":"b".repeat(64),"project_before":"c".repeat(64),"project_after":"c".repeat(64),"protected_before_sha256":"d".repeat(64),"protected_after_sha256":"d".repeat(64)});
    let factory = json!({"provider_execution_sha256":"e".repeat(64),"repository_id":1,"issue_id":2,"pull_request_id":3,"coder_actor":4,"reviewer_actor":5,"reviewed_head":"f".repeat(40),"merged_candidate_head":"f".repeat(40),"required_checks":{"CI":"success"},"elapsed_ms":1000,"funding_limit_ms":2000});
    let artifacts = manifest
        .assets
        .iter()
        .filter(|asset| {
            [
                "docker.json",
                "sodaos-docker-compose.yaml",
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
    for kind in ["installation", "browser_project", "factory", "persistence"] {
        let folder = root.join(kind);
        std::fs::create_dir(&folder).unwrap();
        let mut files = BTreeMap::new();
        std::fs::write(folder.join("stdout.txt"), b"native observation").unwrap();
        files.insert("stdout.txt".into(), identity(b"native observation").sha256);
        if matches!(kind, "installation" | "factory") {
            let body = serde_json::to_vec(if kind == "installation" {
                &native
            } else {
                &factory
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
            outcome: "completed".into(),
            execution: "completed".into(),
            evidence: "completed".into(),
            exit_code: Some(0),
            files: Some(files),
            artifacts: Some(artifacts.clone()),
            started: "2026-10-14T00:00:00Z".into(),
            finished: "2026-10-14T00:00:01Z".into(),
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
    Ok(serde_json::to_vec(&json!({"id":17,"tag_name":manifest.tag,"draft":false,"prerelease":prerelease,"immutable":true,"assets":assets.iter().map(|asset|json!({"name":asset.name,"size":asset.bytes,"digest":format!("sha256:{}",asset.sha256),"state":"uploaded"})).collect::<Vec<_>>()})).unwrap())
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
    assert_eq!(manifest.assets.len(), 10);
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
    let registry = Commands::new(responses, &metadata.image.manifest);
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
    let registry = Commands::new(replies, &metadata.image.manifest);
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
    let registry = Commands::new(replies, &metadata.image.manifest);
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
    );
    let report = crate::publish::status(&gh, &registry, &output).unwrap();
    assert_eq!(report["next_action"], "verify-docker");
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

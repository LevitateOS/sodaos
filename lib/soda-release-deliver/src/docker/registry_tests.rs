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

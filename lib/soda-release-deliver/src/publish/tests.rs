use super::*;
fn stage(gh: &dyn GitHub, manifest: &Manifest, directory: &Path) -> Result<(), Error> {
    super::stage_github(gh, manifest, directory, |_| Ok(()))
}
use assets::{write_json, Qualification, ISO_NAME, MAX_ASSET_BYTES};
use std::cell::RefCell;
use std::collections::VecDeque;
use tempfile::TempDir;

struct Script {
    replies: RefCell<VecDeque<Result<Vec<u8>, Error>>>,
    calls: RefCell<Vec<Vec<String>>>,
}
impl Script {
    fn new(replies: Vec<Result<Vec<u8>, Error>>) -> Self {
        Self {
            replies: RefCell::new(replies.into()),
            calls: RefCell::new(vec![]),
        }
    }
}
impl GitHub for Script {
    fn run(&self, args: &[String]) -> Result<Vec<u8>, Error> {
        self.calls.borrow_mut().push(args.to_vec());
        self.replies
            .borrow_mut()
            .pop_front()
            .expect("unexpected GitHub call")
    }
}
fn json(v: serde_json::Value) -> Result<Vec<u8>, Error> {
    Ok(serde_json::to_vec(&v).unwrap())
}
fn fixture() -> (TempDir, Manifest) {
    let dir = TempDir::new().unwrap();
    std::fs::write(dir.path().join(ISO_NAME), b"iso").unwrap();
    let rootfs = format!(
        "{}-rootfs.img",
        crate::hash_bytes(b"rootfs").trim_start_matches("sha256:")
    );
    std::fs::write(dir.path().join(&rootfs), b"rootfs").unwrap();
    let mut entries = vec![
        inspect(&dir.path().join(ISO_NAME), ISO_NAME).unwrap(),
        inspect(&dir.path().join(&rootfs), &rootfs).unwrap(),
    ];
    for name in [
        "sodaos-source.tar",
        "fountain-source.tar",
        "payload.json",
        "candidate.json",
        "media.json",
        "app-inputs.json",
        "packages.txt",
        "presentation.json",
    ] {
        std::fs::write(dir.path().join(name), b"metadata").unwrap();
        entries.push(inspect(&dir.path().join(name), name).unwrap());
    }
    let m = Manifest {
        format: 1,
        tag: "v0.1.0".to_owned(),
        distribution: crate::model::Distribution::Appliance,
        docker: None,
        soda_revision: "a".repeat(40),
        forgejo_revision: "b".repeat(40),
        iso: ISO_NAME.to_owned(),
        rootfs,
        notes: "Release notes".to_owned(),
        assets: entries,
    };
    write_json(&dir.path().join("release.json"), &m).unwrap();
    std::fs::write(dir.path().join("SHA256SUMS"), b"test checksums").unwrap();
    (dir, m)
}
fn tag(m: &Manifest) -> Result<Vec<u8>, Error> {
    json(
        serde_json::json!({"ref":format!("refs/tags/{}",m.tag),"object":{"type":"commit","sha":m.soda_revision}}),
    )
}
fn fountain(m: &Manifest) -> Result<Vec<u8>, Error> {
    json(serde_json::json!({"revision":m.forgejo_revision}))
}
fn ci_run(m: &Manifest) -> serde_json::Value {
    serde_json::json!({"id":23,"head_sha":m.soda_revision,"head_branch":"main","event":"push","path":".github/workflows/ci.yml","status":"completed","conclusion":"success"})
}
fn ci(m: &Manifest) -> Result<Vec<u8>, Error> {
    json(serde_json::json!({"workflow_runs":[ci_run(m)]}))
}
fn remote(
    m: &Manifest,
    assets: &[Asset],
    draft: bool,
    prerelease: bool,
    immutable: bool,
) -> Result<Vec<u8>, Error> {
    json(
        serde_json::json!({"id":17,"tag_name":m.tag,"draft":draft,"prerelease":prerelease,"immutable":immutable,"assets":assets.iter().map(|a|serde_json::json!({"name":a.name,"size":a.bytes,"digest":format!("sha256:{}",a.sha256),"state":"uploaded"})).collect::<Vec<_>>() }),
    )
}
#[test]
fn stage_checks_exact_inventory_before_publishing() {
    let (dir, m) = fixture();
    let items = inventory(&m, dir.path()).unwrap();
    let gh = Script::new(vec![
        fountain(&m),
        ci(&m),
        json(serde_json::json!({"enabled":true})),
        Err(Error::msg("GitHub resource not found")),
        tag(&m),
        Ok(vec![]),
        Ok(vec![]),
        json(serde_json::json!({"databaseId":17})),
        remote(&m, &items, true, true, false),
        Ok(vec![]),
        json(serde_json::json!({"databaseId":17})),
        remote(&m, &items, false, true, true),
        Ok(vec![]),
    ]);
    stage(&gh, &m, dir.path()).unwrap();
    assert!(gh.replies.borrow().is_empty());
    let calls = gh.calls.borrow();
    assert_eq!(
        calls[3],
        [
            "release",
            "view",
            &m.tag,
            "--repo",
            REPOSITORY,
            "--json",
            "databaseId"
        ]
    );
    assert!(calls[5].contains(&"--draft".to_owned()));
    assert_eq!(calls[8].last().unwrap(), &format!("{API}/releases/17"));
    assert!(calls[9].contains(&"--latest=false".to_owned()));
    let mut bad = items.clone();
    bad[0].sha256 = "c".repeat(64);
    let gh = Script::new(vec![
        fountain(&m),
        ci(&m),
        json(serde_json::json!({"enabled":true})),
        Err(Error::msg("GitHub resource not found")),
        tag(&m),
        Ok(vec![]),
        Ok(vec![]),
        json(serde_json::json!({"databaseId":17})),
        remote(&m, &bad, true, true, false),
    ]);
    assert!(stage(&gh, &m, dir.path()).is_err());
    assert_eq!(gh.calls.borrow().len(), 9);
}
#[test]
fn stage_refuses_existing_release_and_wrong_tag() {
    let (dir, m) = fixture();
    let items = inventory(&m, dir.path()).unwrap();
    for draft in [true, false] {
        let gh = Script::new(vec![
            fountain(&m),
            ci(&m),
            json(serde_json::json!({"enabled":true})),
            json(serde_json::json!({"databaseId":17})),
            remote(&m, &items, draft, true, !draft),
        ]);
        assert!(stage(&gh, &m, dir.path()).is_err());
        assert_eq!(gh.calls.borrow().len(), 5);
    }
    let gh = Script::new(vec![
        fountain(&m),
        ci(&m),
        json(serde_json::json!({"enabled":true})),
        Err(Error::msg("GitHub resource not found")),
        json(
            serde_json::json!({"ref":"refs/tags/v0.1.0","object":{"type":"commit","sha":"c".repeat(40)}}),
        ),
    ]);
    assert!(stage(&gh, &m, dir.path()).is_err());
    assert_eq!(gh.calls.borrow().len(), 5);
}
#[test]
fn stage_creates_missing_tag_for_the_ci_qualified_revision() {
    let (dir, m) = fixture();
    let items = inventory(&m, dir.path()).unwrap();
    let gh = Script::new(vec![
        fountain(&m),
        ci(&m),
        json(serde_json::json!({"enabled":true})),
        Err(Error::msg("GitHub resource not found")),
        Err(Error::msg("GitHub resource not found")),
        tag(&m),
        Ok(vec![]),
        Ok(vec![]),
        json(serde_json::json!({"databaseId":17})),
        remote(&m, &items, true, true, false),
        Ok(vec![]),
        json(serde_json::json!({"databaseId":17})),
        remote(&m, &items, false, true, true),
        Ok(vec![]),
    ]);
    stage(&gh, &m, dir.path()).unwrap();
    assert!(gh.replies.borrow().is_empty());
    assert_eq!(
        gh.calls.borrow()[5],
        [
            "api",
            "--hostname",
            "github.com",
            "--method",
            "POST",
            &format!("{API}/git/refs"),
            "--raw-field",
            &format!("ref=refs/tags/{}", m.tag),
            "--raw-field",
            &format!("sha={}", m.soda_revision),
        ]
    );
}
#[test]
fn promotion_requires_observed_revision_smoke_confirmation_and_actual_download() {
    let (dir, m) = fixture();
    let items = inventory(&m, dir.path()).unwrap();
    let gh = Script::new(vec![
        tag(&m),
        fountain(&m),
        ci(&m),
        json(serde_json::json!({"databaseId":17})),
        remote(&m, &items, false, true, true),
        Ok(vec![]),
    ]);
    let downloaded = dir.path().join(ISO_NAME);
    qualify_appliance(&gh, dir.path(), &downloaded, &m.soda_revision, true).unwrap();
    assert_eq!(gh.calls.borrow()[5][1], "verify-asset");
    assert_eq!(gh.calls.borrow().len(), 6);
    let receipt: Qualification =
        read_json(&dir.path().join("appliance-qualification.json")).unwrap();
    assert_eq!(receipt.soda_revision, m.soda_revision);
    assert_eq!(receipt.forgejo_revision, m.forgejo_revision);
    assert_eq!(receipt.observed_soda_revision, m.soda_revision);
    assert_eq!(receipt.downloaded_iso_sha256, m.assets[0].sha256);
    assert_eq!(receipt.rootfs_sha256, m.assets[1].sha256);
    assert_eq!(receipt.install, "passed");
    assert_eq!(receipt.boot, "passed");
    assert_eq!(receipt.applications, "passed");
    let gh = Script::new(vec![]);
    assert!(qualify_appliance(&gh, dir.path(), &downloaded, &"c".repeat(40), true).is_err());
    assert!(gh.calls.borrow().is_empty());
    assert!(qualify_appliance(&gh, dir.path(), &downloaded, &m.soda_revision, false).is_err());
    assert!(gh.calls.borrow().is_empty());
    std::fs::write(dir.path().join(ISO_NAME), b"tampered").unwrap();
    assert!(qualify_appliance(&gh, dir.path(), &downloaded, &m.soda_revision, true).is_err());
    assert!(gh.calls.borrow().is_empty());
}
#[test]
fn publication_requires_current_ci_for_both_exact_sources() {
    let (dir, m) = fixture();
    for (field, value) in [
        ("head_sha", serde_json::json!("c".repeat(40))),
        ("head_branch", serde_json::json!("topic")),
        ("event", serde_json::json!("pull_request")),
        ("path", serde_json::json!(".github/workflows/other.yml")),
        ("status", serde_json::json!("in_progress")),
        ("conclusion", serde_json::json!("failure")),
        ("conclusion", serde_json::json!("cancelled")),
        ("conclusion", serde_json::json!("skipped")),
    ] {
        let mut run = ci_run(&m);
        run[field] = value;
        let gh = Script::new(vec![
            fountain(&m),
            json(serde_json::json!({"workflow_runs":[run]})),
        ]);
        assert!(stage(&gh, &m, dir.path()).is_err(), "{field}");
        assert_eq!(gh.calls.borrow().len(), 2);
    }
    let gh = Script::new(vec![
        fountain(&m),
        json(serde_json::json!({"workflow_runs":[]})),
    ]);
    assert!(stage(&gh, &m, dir.path()).is_err());
    let gh = Script::new(vec![json(serde_json::json!({"revision":"c".repeat(40)}))]);
    assert!(stage(&gh, &m, dir.path()).is_err());
    assert_eq!(gh.calls.borrow().len(), 1);
}
#[test]
fn asset_limit_and_tag_admission_refuse_before_transport() {
    let dir = TempDir::new().unwrap();
    let p = dir.path().join("large.iso");
    std::fs::File::create(&p)
        .unwrap()
        .set_len(MAX_ASSET_BYTES)
        .unwrap();
    assert!(inspect(&p, "large.iso").is_err());
    assert!(!assets::valid_tag("main^{commit}"));
    assert!(!assets::valid_tag("../tag"));
    assert!(assets::valid_tag("v0.1.0-rc.1"));
}

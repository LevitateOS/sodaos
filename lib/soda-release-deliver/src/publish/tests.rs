use super::*;
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
        tag(&m),
        json(serde_json::json!({"enabled":true})),
        Err(Error::msg("GitHub resource not found")),
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
        calls[2],
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
    assert!(calls[3].contains(&"--draft".to_owned()));
    assert_eq!(calls[6].last().unwrap(), &format!("{API}/releases/17"));
    assert!(calls[7].contains(&"--latest=false".to_owned()));
    let mut bad = items.clone();
    bad[0].sha256 = "c".repeat(64);
    let gh = Script::new(vec![
        tag(&m),
        json(serde_json::json!({"enabled":true})),
        Err(Error::msg("GitHub resource not found")),
        Ok(vec![]),
        Ok(vec![]),
        json(serde_json::json!({"databaseId":17})),
        remote(&m, &bad, true, true, false),
    ]);
    assert!(stage(&gh, &m, dir.path()).is_err());
    assert_eq!(gh.calls.borrow().len(), 7);
}
#[test]
fn stage_refuses_existing_release_and_wrong_tag() {
    let (dir, m) = fixture();
    let items = inventory(&m, dir.path()).unwrap();
    for draft in [true, false] {
        let gh = Script::new(vec![
            tag(&m),
            json(serde_json::json!({"enabled":true})),
            json(serde_json::json!({"databaseId":17})),
            remote(&m, &items, draft, true, !draft),
        ]);
        assert!(stage(&gh, &m, dir.path()).is_err());
        assert_eq!(gh.calls.borrow().len(), 4);
    }
    let gh = Script::new(vec![json(
        serde_json::json!({"ref":"refs/tags/v0.1.0","object":{"type":"commit","sha":"c".repeat(40)}}),
    )]);
    assert!(stage(&gh, &m, dir.path()).is_err());
    assert_eq!(gh.calls.borrow().len(), 1);
}
#[test]
fn promotion_requires_bound_native_receipt_and_actual_download() {
    let (dir, m) = fixture();
    let items = inventory(&m, dir.path()).unwrap();
    let mut q = Qualification {
        format: 1,
        soda_revision: m.soda_revision.clone(),
        forgejo_revision: m.forgejo_revision.clone(),
        iso_sha256: m.assets[0].sha256.clone(),
        rootfs_sha256: m.assets[1].sha256.clone(),
        downloaded_iso_sha256: m.assets[0].sha256.clone(),
        observed_soda_revision: m.soda_revision.clone(),
        install: "passed".into(),
        boot: "passed".into(),
        applications: "passed".into(),
    };
    let qp = dir.path().join("qualified.json");
    write_json(&qp, &q).unwrap();
    let gh = Script::new(vec![
        tag(&m),
        json(serde_json::json!({"databaseId":17})),
        remote(&m, &items, false, true, true),
        Ok(vec![]),
        Ok(vec![]),
        remote(&m, &items, false, false, true),
    ]);
    promote(&gh, dir.path(), &qp, &dir.path().join(ISO_NAME)).unwrap();
    assert_eq!(gh.calls.borrow()[3][1], "verify-asset");
    assert!(gh.calls.borrow()[4].contains(&"--prerelease=false".to_owned()));
    q.observed_soda_revision = "c".repeat(40);
    write_json(&qp, &q).unwrap();
    let gh = Script::new(vec![]);
    assert!(promote(&gh, dir.path(), &qp, &dir.path().join(ISO_NAME)).is_err());
    assert!(gh.calls.borrow().is_empty());
    q.observed_soda_revision = m.soda_revision.clone();
    write_json(&qp, &q).unwrap();
    std::fs::write(dir.path().join(ISO_NAME), b"tampered").unwrap();
    assert!(promote(&gh, dir.path(), &qp, &dir.path().join(ISO_NAME)).is_err());
    assert!(gh.calls.borrow().is_empty());
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

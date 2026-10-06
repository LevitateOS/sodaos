use super::*;
use crate::tailnet_domain::{ReadFile, RunTarget, StatFile, ERR_CONFLICT, ERR_UNCONFIRMED};
use crate::tailnet_runtime::ProjectRun;
use std::cell::RefCell;
use std::io::ErrorKind;
use std::os::unix::fs::{symlink, PermissionsExt};
use std::rc::Rc;
use std::sync::atomic::{AtomicU64, Ordering};

static COUNTER: AtomicU64 = AtomicU64::new(0);

struct TempDir {
    path: PathBuf,
}

impl TempDir {
    fn create() -> TempDir {
        let n = COUNTER.fetch_add(1, Ordering::SeqCst);
        let path =
            std::env::temp_dir().join(format!("soda-tailnet-files-{}-{n}", std::process::id()));
        fs::create_dir_all(&path).unwrap();
        TempDir { path }
    }

    fn path(&self) -> &Path {
        &self.path
    }
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.path);
    }
}

fn runtime_test_root() -> TempDir {
    let t = TempDir::create();
    fs::set_permissions(t.path(), fs::Permissions::from_mode(0o700)).unwrap();
    t
}

fn euid() -> u32 {
    unsafe { libc::getuid() }
}

fn egid() -> u32 {
    unsafe { libc::getgid() }
}

fn file_run() -> ProjectRun {
    ProjectRun {
        target: RunTarget {
            project: format!("p{}", "a".repeat(24)),
            container: "b".repeat(64),
            run: "c".repeat(64),
        },
        pid: 77,
        started: String::new(),
        userns: String::new(),
        netns: String::new(),
        uid: euid(),
        gid: egid(),
        resolver: String::new(),
    }
}

/// Non-root CI: keep the test UID; root CI: use an unprivileged-looking
/// shifted pair exactly like the Go tests.
fn shifted_run() -> ProjectRun {
    let mut run = file_run();
    if run.uid == 0 {
        run.uid = 100000;
        run.gid = 100000;
    }
    run
}

fn open(base: &str, project: &str) -> Result<RunFiles, String> {
    open_runtime_project_owned(
        base,
        project,
        euid(),
        egid(),
        Instant::now() + Duration::from_secs(60),
    )
}

#[test]
fn run_files_exclusive_secret_retirement_and_independent_locks() {
    let base_tmp = runtime_test_root();
    let base = base_tmp.path().to_str().unwrap().to_string();
    let run = shifted_run();
    let f = open(&base, &run.target.project).expect("open");
    let short = Instant::now() + Duration::from_millis(30);
    let e =
        open_runtime_project_owned(&base, &run.target.project, euid(), egid(), short).unwrap_err();
    assert_eq!(
        e, "context deadline exceeded",
        "lock waiter not cancellable"
    );
    let other = open(&base, &format!("p{}", "d".repeat(24))).expect("unrelated project blocked");
    drop(other);
    let root = f.prepare(&run, true).expect("prepare");
    let e = f.prepare(&run, true).unwrap_err();
    assert_eq!(e, ERR_CONFLICT, "run root recreated");
    f.save_current(&run).expect("save");
    let restored = f.current().expect("current");
    assert_eq!(restored, run);
    let cid = "e".repeat(64);
    write_companion_id(&root, &cid).expect("write companion id");
    let e = write_companion_id(&root, &"f".repeat(64)).unwrap_err();
    assert_eq!(e, ERR_CONFLICT, "companion identity replaced");
    let observed = read_companion_id(&base, &run, euid(), egid()).expect("read companion id");
    assert_eq!(observed, cid, "companion identity not retained");
    let key = write_run_key(&root, &run, "tskey-auth-synthetic-one-use-only").expect("write key");
    let info = key.metadata().expect("key stat");
    assert!(
        runtime_file(&info, run.uid, run.gid, 0o600),
        "unsafe key metadata"
    );
    let e = write_run_key(&root, &run, "tskey-auth-synthetic-another").unwrap_err();
    assert_eq!(e, ERR_CONFLICT, "key replaced");
    retire_run_key(&root, &key).expect("retire");
    let info = key.metadata().expect("key stat after retire");
    assert_eq!(info.len(), 0, "retired inode still contains key");
    let e = root.lstat("input/key").unwrap_err();
    assert_eq!(e.kind(), ErrorKind::NotFound, "key not retired");
    // No reusable credentials are stored with the runtime record.
    let b = fs::read(
        base_tmp
            .path()
            .join(&run.target.project)
            .join("current.json"),
    )
    .expect("read current.json");
    let s = String::from_utf8_lossy(&b);
    assert!(
        !s.contains("tskey") && !s.contains("secret"),
        "unsafe runtime projection"
    );
}

#[test]
fn runtime_root_refuses_symlink_without_writing_through_it() {
    let base_tmp = runtime_test_root();
    let link_tmp = TempDir::create();
    let link = link_tmp.path().join("runtime");
    symlink(base_tmp.path(), &link).unwrap();
    let project = file_run().target.project;
    let r = open_runtime_project_owned(
        link.to_str().unwrap(),
        &project,
        euid(),
        egid(),
        Instant::now() + Duration::from_secs(60),
    );
    assert!(r.is_err(), "runtime root link accepted");
    let entries = fs::read_dir(base_tmp.path()).expect("read base");
    assert_eq!(entries.count(), 0, "wrote through unsafe root");
}

#[test]
fn run_files_refuse_symlinks_modes_and_changed_key_inode() {
    for kind in ["project-link", "lock-link", "lock-hardlink", "unsafe-mode"] {
        let base_tmp = runtime_test_root();
        let outside = TempDir::create();
        let run = file_run();
        let project = base_tmp.path().join(&run.target.project);
        if kind == "project-link" {
            symlink(outside.path(), &project).unwrap();
        } else {
            fs::create_dir(&project).unwrap();
            fs::set_permissions(&project, fs::Permissions::from_mode(0o700)).unwrap();
            let target = outside.path().join("retained");
            fs::write(&target, b"preserve").unwrap();
            fs::set_permissions(&target, fs::Permissions::from_mode(0o600)).unwrap();
            match kind {
                "lock-link" => symlink(&target, project.join("lock")).unwrap(),
                "lock-hardlink" => fs::hard_link(&target, project.join("lock")).unwrap(),
                "unsafe-mode" => {
                    fs::set_permissions(&project, fs::Permissions::from_mode(0o755)).unwrap()
                }
                _ => unreachable!(),
            }
        }
        let base = base_tmp.path().to_str().unwrap().to_string();
        let r = open(&base, &run.target.project);
        assert!(r.is_err(), "unsafe path accepted: {kind}");
    }

    let base_tmp = runtime_test_root();
    let base = base_tmp.path().to_str().unwrap().to_string();
    let run = shifted_run();
    let f = open(&base, &run.target.project).expect("open");
    let root = f.prepare(&run, true).expect("prepare");
    let key = write_run_key(&root, &run, "tskey-auth-synthetic-secret").expect("write key");
    fs::rename(root.join("input/key"), root.join("input/retained")).expect("fixture");
    fs::write(root.join("input/key"), b"later write").expect("fixture");
    fs::set_permissions(root.join("input/key"), fs::Permissions::from_mode(0o600))
        .expect("fixture");
    let e = retire_run_key(&root, &key).unwrap_err();
    assert_eq!(e, ERR_UNCONFIRMED, "replaced input retired");
    let b = fs::read(root.join("input/key")).expect("read later write");
    assert_eq!(b, b"later write", "later write lost");
}

#[test]
fn resolver_requires_original_inode_and_no_conflicting_manager() {
    let mut run = file_run();
    run.resolver = format!(
        "/var/lib/containers/storage/overlay-containers/{}/userdata/resolv.conf",
        run.target.container
    );
    let dir = TempDir::create();
    let path = dir.path().join("resolver");
    fs::write(&path, b"nameserver 192.0.2.1\n").expect("fixture");
    let info = fs::metadata(&path).expect("fixture");
    let contents = Rc::new(RefCell::new("nameserver 192.0.2.1\n".to_string()));
    let devices = Rc::new(RefCell::new("eth0: 0 0\n".to_string()));
    // Fresh closures per call: ReadFile/StatFile are owned (Box) values.
    let make_read = || -> ReadFile {
        let resolver = run.resolver.clone();
        let contents = Rc::clone(&contents);
        let devices = Rc::clone(&devices);
        Box::new(move |path: &str| {
            if path == resolver {
                Ok(contents.borrow().as_bytes().to_vec())
            } else {
                Ok(devices.borrow().as_bytes().to_vec())
            }
        })
    };
    let make_stat_same = || -> StatFile {
        let info = info.clone();
        Box::new(move |_: &str| Ok(info.clone()))
    };
    validate_run_resolver(&run, make_read(), make_stat_same(), true).expect("baseline");
    for text in [
        "# Generated by systemd-resolved\n",
        "# resolvconf\n",
        "# resolv.conf(5) file generated by tailscale\n",
    ] {
        *contents.borrow_mut() = text.to_string();
        assert!(
            validate_run_resolver(&run, make_read(), make_stat_same(), true).is_err(),
            "conflicting resolver accepted"
        );
    }
    *contents.borrow_mut() = "nameserver 192.0.2.1\n".to_string();
    *devices.borrow_mut() = "tailscale0: 0 0\n".to_string();
    let e = validate_run_resolver(&run, make_read(), make_stat_same(), true).unwrap_err();
    assert_eq!(e, ERR_CONFLICT, "foreign TUN accepted");
    validate_run_resolver(&run, make_read(), make_stat_same(), false)
        .expect("owned same-run interface refused");
    let other = dir.path().join("other");
    fs::write(&other, b"nameserver 192.0.2.1\n").expect("fixture");
    let make_stat_split = || -> StatFile {
        let info = info.clone();
        let resolver = run.resolver.clone();
        let other = other.clone();
        Box::new(move |p: &str| {
            if p == resolver {
                Ok(info.clone())
            } else {
                // Fixture always exists; the fallback only avoids naming
                // the closure error type.
                match fs::metadata(&other) {
                    Ok(m) => Ok(m),
                    Err(_) => Ok(info.clone()),
                }
            }
        })
    };
    assert!(
        validate_run_resolver(&run, make_read(), make_stat_split(), false).is_err(),
        "different resolver inode accepted"
    );
}

#[test]
fn completed_key_input_can_be_retired_for_explicit_retry() {
    let base_tmp = runtime_test_root();
    let base = base_tmp.path().to_str().unwrap().to_string();
    let run = shifted_run();
    let f = open(&base, &run.target.project).expect("open");
    let root = f.prepare(&run, true).expect("prepare");
    let e = root.lstat("state").unwrap_err();
    assert_eq!(
        e.kind(),
        ErrorKind::NotFound,
        "node identity directory created"
    );
    let key =
        write_run_key(&root, &run, "tskey-auth-synthetic-completed-input").expect("write key");
    retire_pending_run_key(&root, &run).expect("retire pending");
    let info = key.metadata().expect("key stat after retire");
    assert_eq!(info.len(), 0, "old input not scrubbed");
    retire_pending_run_key(&root, &run).expect("absent input became a permanent retry veto");
    symlink("../companion-id", root.join("input/key")).expect("fixture");
    assert!(
        retire_pending_run_key(&root, &run).is_err(),
        "substituted input accepted"
    );
}

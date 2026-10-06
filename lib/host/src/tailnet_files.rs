//! Tailnet run files: rooted runtime state, locks, keys, resolver checks.
//! Lane C owns this file.
//!
//! Rust port of `internal/host/tailnet/files.go` plus `current`/`saveCurrent`
//! from `internal/host/tailnet/companion.go` (same Go type, owned by this lane).
//!
//! Containment note: Go uses `os.Root` (openat2-grade containment). This port
//! anchors every operation under an explicit base path and opens leaves with
//! `O_NOFOLLOW`, but joins are plain path joins, so containment under an
//! actively racing attacker is approximated, not proven. Every multi-component
//! relative path is built only from regex-validated IDs (no `.`, `..` or `/`
//! can occur), and tests run in private temp dirs with no attacker races.

use std::fs::{self, File, OpenOptions};
use std::io::{Read, Write};
use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
use std::os::unix::io::AsRawFd;
use std::path::{Path, PathBuf};
use std::time::{Duration, Instant};

use crate::tailnet_domain::{
    valid_container_id, valid_project_id, ReadFile, StatFile, ERR_CONFLICT, ERR_INVALID,
    ERR_UNAVAILABLE, ERR_UNCONFIRMED, ERR_UNSUPPORTED,
};
use crate::tailnet_runtime::{
    companion_create_args, decode_project_run, marshal_project_run, ProjectRun,
};

/// Text returned when `current.json` is absent. The companion lane matches on
/// this exact string the way Go matches `errors.Is(err, os.ErrNotExist)`.
const RUNTIME_RECORD_NOT_FOUND: &str = "runtime record not found";

/// Path-anchored root for runtime state. All operations join `rel` under the
/// anchored base path; leaves are opened with `O_NOFOLLOW`.
#[derive(Debug)]
pub struct Root {
    path: PathBuf,
}

/// Locked, owned runtime project directory. The lock releases when the
/// contained `File` is dropped (close), exactly like Go's `f.Close()`.
#[derive(Debug)]
pub struct RunFiles {
    pub(crate) root: Root,
    // Held for close-on-drop lock semantics; never read.
    #[allow(dead_code)]
    lock: File,
    pub(crate) uid: u32,
    pub(crate) gid: u32,
}

impl Root {
    pub(crate) fn join(&self, rel: &str) -> PathBuf {
        self.path.join(rel)
    }

    /// Lstat equivalent: never follows the final component.
    pub(crate) fn lstat(&self, rel: &str) -> std::io::Result<fs::Metadata> {
        fs::symlink_metadata(self.join(rel))
    }

    /// Stat equivalent: follows links, like Go `Root.Stat`.
    fn stat(&self, rel: &str) -> std::io::Result<fs::Metadata> {
        fs::metadata(self.join(rel))
    }
}

fn root_directory(info: &fs::Metadata, uid: u32) -> bool {
    info.is_dir() && info.uid() == uid && info.mode() & 0o777 == 0o700
}

fn runtime_file(info: &fs::Metadata, uid: u32, gid: u32, mode: u32) -> bool {
    info.is_file()
        && info.mode() & 0o777 == mode
        && info.uid() == uid
        && info.gid() == gid
        && info.nlink() == 1
}

fn owned_run_dir(info: &fs::Metadata, uid: u32, gid: u32) -> bool {
    info.is_dir() && info.mode() & 0o777 == 0o700 && info.uid() == uid && info.gid() == gid
}

fn same_file(a: &fs::Metadata, b: &fs::Metadata) -> bool {
    a.dev() == b.dev() && a.ino() == b.ino()
}

/// `mkdir(path, 0o700)`: Go passes the mode to mkdir(2); Rust `create_dir`
/// would use `0o777 & ~umask`, so call libc directly for exact semantics
/// (including `EEXIST` when the path already exists).
fn mkdir_700(path: &Path) -> std::io::Result<()> {
    use std::os::unix::ffi::OsStrExt;
    let c = std::ffi::CString::new(path.as_os_str().as_bytes())
        .map_err(|_| std::io::Error::new(std::io::ErrorKind::InvalidInput, "path contains nul"))?;
    if unsafe { libc::mkdir(c.as_ptr(), 0o700) } == 0 {
        Ok(())
    } else {
        Err(std::io::Error::last_os_error())
    }
}

fn chown_path(path: &Path, uid: u32, gid: u32) -> std::io::Result<()> {
    use std::os::unix::ffi::OsStrExt;
    let c = std::ffi::CString::new(path.as_os_str().as_bytes())
        .map_err(|_| std::io::Error::new(std::io::ErrorKind::InvalidInput, "path contains nul"))?;
    if unsafe { libc::chown(c.as_ptr(), uid as libc::uid_t, gid as libc::gid_t) } == 0 {
        Ok(())
    } else {
        Err(std::io::Error::last_os_error())
    }
}

fn open_runtime_root(base: &str, uid: u32) -> Result<Root, String> {
    let info = fs::symlink_metadata(base).map_err(|_| ERR_UNAVAILABLE.to_string())?;
    if !root_directory(&info, uid) {
        return Err(ERR_UNAVAILABLE.to_string());
    }
    let root = Root {
        path: PathBuf::from(base),
    };
    let opened = root.stat(".").map_err(|_| ERR_UNAVAILABLE.to_string())?;
    if !same_file(&info, &opened) {
        return Err(ERR_UNAVAILABLE.to_string());
    }
    Ok(root)
}

fn ensure_runtime_project_dir(parent: &Root, project: &str, uid: u32) -> Result<(), String> {
    let info = parent.stat(".").map_err(|_| ERR_UNAVAILABLE.to_string())?;
    if !root_directory(&info, uid) {
        return Err(ERR_UNAVAILABLE.to_string());
    }
    match mkdir_700(&parent.join(project)) {
        Ok(()) => {}
        Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => {}
        Err(_) => return Err(ERR_UNAVAILABLE.to_string()),
    }
    let info = parent
        .lstat(project)
        .map_err(|_| ERR_UNAVAILABLE.to_string())?;
    if !root_directory(&info, uid) {
        return Err(ERR_UNAVAILABLE.to_string());
    }
    Ok(())
}

/// Mirror of `filelock.Acquire` with `LOCK_EX`: poll `flock(LOCK_EX|LOCK_NB)`,
/// retry `EWOULDBLOCK`/`EINTR` on a 25ms tick, report other errors by message,
/// and return `"context deadline exceeded"` (Go `ctx.Err()` text) on expiry.
fn acquire_exclusive(lock: &File, deadline: Instant) -> Result<(), String> {
    loop {
        if Instant::now() >= deadline {
            return Err("context deadline exceeded".to_string());
        }
        let r = unsafe { libc::flock(lock.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) };
        if r == 0 {
            return Ok(());
        }
        let errno = std::io::Error::last_os_error().raw_os_error().unwrap_or(0);
        if errno != libc::EWOULDBLOCK && errno != libc::EINTR {
            return Err(std::io::Error::from_raw_os_error(errno).to_string());
        }
        let now = Instant::now();
        if now >= deadline {
            return Err("context deadline exceeded".to_string());
        }
        std::thread::sleep((deadline - now).min(Duration::from_millis(25)));
    }
}

fn lock_runtime_project(
    root: &Root,
    uid: u32,
    gid: u32,
    deadline: Instant,
) -> Result<File, String> {
    let mut opts = OpenOptions::new();
    opts.read(true)
        .write(true)
        .create(true)
        .mode(0o600)
        .custom_flags(libc::O_NOFOLLOW);
    let lock = opts
        .open(root.join("lock"))
        .map_err(|_| ERR_UNAVAILABLE.to_string())?;
    let info = lock.metadata().map_err(|_| ERR_UNAVAILABLE.to_string())?;
    if !runtime_file(&info, uid, gid, 0o600) {
        drop(lock);
        return Err(ERR_UNAVAILABLE.to_string());
    }
    if let Err(e) = acquire_exclusive(&lock, deadline) {
        drop(lock);
        return Err(e);
    }
    Ok(lock)
}

/// Open (creating if needed) and exclusively lock a runtime project directory.
/// Unsafe existing paths are refused, never repaired.
pub fn open_runtime_project_owned(
    base: &str,
    project: &str,
    uid: u32,
    gid: u32,
    deadline: Instant,
) -> Result<RunFiles, String> {
    if !valid_project_id(project) {
        return Err(ERR_INVALID.to_string());
    }
    let parent = open_runtime_root(base, uid)?;
    ensure_runtime_project_dir(&parent, project, uid)?;
    let root = Root {
        path: parent.join(project),
    };
    let lock = lock_runtime_project(&root, uid, gid, deadline)?;
    Ok(RunFiles {
        root,
        lock,
        uid,
        gid,
    })
}

/// Production open: the runtime tree is appliance-owned (`0:0`).
pub fn open_runtime_project(
    base: &str,
    project: &str,
    deadline: Instant,
) -> Result<RunFiles, String> {
    open_runtime_project_owned(base, project, 0, 0, deadline)
}

fn prepare_run_subdir(
    root: &Root,
    path: &str,
    run: &ProjectRun,
    fresh: bool,
) -> Result<(), String> {
    if fresh {
        match mkdir_700(&root.join(path)) {
            Ok(()) => {
                if chown_path(&root.join(path), run.uid, run.gid).is_err() {
                    return Err(ERR_UNAVAILABLE.to_string());
                }
            }
            Err(_) => return Err(ERR_UNAVAILABLE.to_string()),
        }
    }
    let info = root.lstat(path).map_err(|_| ERR_UNAVAILABLE.to_string())?;
    if !owned_run_dir(&info, run.uid, run.gid) {
        return Err(ERR_UNAVAILABLE.to_string());
    }
    Ok(())
}

impl RunFiles {
    /// Read the retained runtime record. A missing `current.json` returns
    /// exactly `"runtime record not found"` so callers can distinguish "no
    /// record yet" from confirmed failures, mirroring Go's raw `os.ErrNotExist`.
    pub fn current(&self) -> Result<ProjectRun, String> {
        let mut opts = OpenOptions::new();
        opts.read(true).custom_flags(libc::O_NOFOLLOW);
        let mut file = match opts.open(self.root.join("current.json")) {
            Ok(f) => f,
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
                return Err(RUNTIME_RECORD_NOT_FOUND.to_string())
            }
            Err(e) => return Err(e.to_string()),
        };
        let info = file.metadata().map_err(|_| ERR_UNAVAILABLE.to_string())?;
        if !runtime_file(&info, self.uid, self.gid, 0o600) || info.len() > 8192 {
            return Err(ERR_UNAVAILABLE.to_string());
        }
        let mut buf = Vec::new();
        file.read_to_end(&mut buf)
            .map_err(|_| ERR_UNAVAILABLE.to_string())?;
        let run = decode_project_run(&buf).map_err(|_| ERR_UNAVAILABLE.to_string())?;
        // The recipe validates all path-bearing identity fields. The caller
        // separately checks this record's project, parent CID and native
        // namespace incarnation.
        companion_create_args(&run, &format!("sha256:{}", "0".repeat(64)))?;
        Ok(run)
    }

    /// Durably install `run` as the current record via temp-file + rename.
    pub fn save_current(&self, run: &ProjectRun) -> Result<(), String> {
        let name = format!("current-{}.json", run.target.run);
        let b = marshal_project_run(run);
        let mut opts = OpenOptions::new();
        opts.write(true)
            .create_new(true)
            .mode(0o600)
            .custom_flags(libc::O_NOFOLLOW);
        let mut file = opts
            .open(self.root.join(&name))
            .map_err(|_| ERR_CONFLICT.to_string())?;
        if file.write_all(b.as_bytes()).is_err() {
            return Err(ERR_UNCONFIRMED.to_string());
        }
        if file.sync_all().is_err()
            || fs::rename(self.root.join(&name), self.root.join("current.json")).is_err()
        {
            return Err(ERR_UNCONFIRMED.to_string());
        }
        let dir = File::open(self.root.join(".")).map_err(|_| ERR_UNCONFIRMED.to_string())?;
        if dir.sync_all().is_err() {
            return Err(ERR_UNCONFIRMED.to_string());
        }
        Ok(())
    }

    /// Prepare (fresh: exclusively create) the run workspace `control`/`input`
    /// subdirectories. A pre-existing directory on a fresh prepare is an
    /// incomplete attempt, never an empty workspace.
    pub fn prepare(&self, run: &ProjectRun, fresh: bool) -> Result<Root, String> {
        let name = &run.target.run;
        if !valid_container_id(name) || run.uid == 0 || run.gid == 0 {
            return Err(ERR_INVALID.to_string());
        }
        if fresh && mkdir_700(&self.root.join(name)).is_err() {
            return Err(ERR_CONFLICT.to_string());
        }
        let info = self
            .root
            .lstat(name)
            .map_err(|_| ERR_UNAVAILABLE.to_string())?;
        if !root_directory(&info, self.uid) {
            return Err(ERR_UNAVAILABLE.to_string());
        }
        let root = Root {
            path: self.root.join(name),
        };
        for path in ["control", "input"] {
            prepare_run_subdir(&root, path, run, fresh)?;
        }
        Ok(root)
    }
}

#[path = "tailnet/files/keys.rs"]
mod keys;

pub use keys::{
    read_companion_id, retire_pending_run_key, retire_run_key, write_companion_id, write_run_key,
};

fn fresh_tailscale_conflict(devices: &str) -> bool {
    for line in devices.split('\n') {
        if let Some((name, _)) = line.split_once(':') {
            if name.trim() == "tailscale0" {
                return true;
            }
        }
    }
    false
}

fn validate_resolver_text(data: &[u8], fresh: bool) -> Result<(), String> {
    if data.len() > 16384 || data.is_empty() {
        return Err(ERR_UNSUPPORTED.to_string());
    }
    let text = String::from_utf8_lossy(data).to_lowercase();
    if text.contains("systemd-resolved")
        || text.contains("resolvconf")
        || (fresh && text.contains("tailscale"))
    {
        return Err(ERR_UNSUPPORTED.to_string());
    }
    Ok(())
}

fn resolver_path(container: &str) -> String {
    format!("/var/lib/containers/storage/overlay-containers/{container}/userdata/resolv.conf")
}

fn validate_resolver_inode(run: &ProjectRun, stat: &StatFile) -> Result<(), String> {
    let expected = resolver_path(&run.target.container);
    if run.resolver != expected {
        return Err(ERR_UNSUPPORTED.to_string());
    }
    let info = stat(&expected).map_err(|_| ERR_UNSUPPORTED.to_string())?;
    if !info.is_file() || info.len() > 16384 {
        return Err(ERR_UNSUPPORTED.to_string());
    }
    let actual = stat(&format!("/proc/{}/root/etc/resolv.conf", run.pid))
        .map_err(|_| ERR_UNSUPPORTED.to_string())?;
    if !same_file(&info, &actual) {
        return Err(ERR_UNSUPPORTED.to_string());
    }
    Ok(())
}

/// Validate the real shared resolver inode, not an arbitrary path obtained
/// from the project. Tailscale owns backup/restore in the retained companion
/// root. `stat` must follow links (like Go `os.Stat`).
pub fn validate_run_resolver(
    run: &ProjectRun,
    read: ReadFile,
    stat: StatFile,
    fresh: bool,
) -> Result<(), String> {
    validate_resolver_inode(run, &stat)?;
    let expected = resolver_path(&run.target.container);
    let data = read(&expected).map_err(|_| ERR_UNSUPPORTED.to_string())?;
    validate_resolver_text(&data, fresh)?;
    let devices =
        read(&format!("/proc/{}/net/dev", run.pid)).map_err(|_| ERR_UNAVAILABLE.to_string())?;
    if devices.len() > 65536 {
        return Err(ERR_UNAVAILABLE.to_string());
    }
    if fresh && fresh_tailscale_conflict(&String::from_utf8_lossy(&devices)) {
        return Err(ERR_CONFLICT.to_string());
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::tailnet_domain::RunTarget;
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
        let e = open_runtime_project_owned(&base, &run.target.project, euid(), egid(), short)
            .unwrap_err();
        assert_eq!(
            e, "context deadline exceeded",
            "lock waiter not cancellable"
        );
        let other =
            open(&base, &format!("p{}", "d".repeat(24))).expect("unrelated project blocked");
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
        let key =
            write_run_key(&root, &run, "tskey-auth-synthetic-one-use-only").expect("write key");
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
}

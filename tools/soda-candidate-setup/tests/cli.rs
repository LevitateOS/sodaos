// Integration tests for soda-candidate-setup. Every case fails before the
// first mutation (`sudo mkdir`), so nothing here touches protected state:
// no sudo, no services, no credentials. Deeper paths are covered by unit
// tests plus review; the privileged flow itself only runs on real operator
// hardware.

use std::env;
use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::sync::atomic::{AtomicU64, Ordering};

static COUNTER: AtomicU64 = AtomicU64::new(0);

// The two lease tests share the real per-operator lock path, so they must
// not run concurrently with each other.
static LEASE_SERIAL: std::sync::Mutex<()> = std::sync::Mutex::new(());

struct TempDir {
    path: PathBuf,
}

impl TempDir {
    fn new(tag: &str) -> TempDir {
        let id = COUNTER.fetch_add(1, Ordering::SeqCst);
        let path = env::temp_dir().join(format!("soda-setup-it-{tag}-{}-{id}", std::process::id()));
        let _ = fs::remove_dir_all(&path);
        fs::create_dir_all(&path).unwrap();
        TempDir { path }
    }
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.path);
    }
}

fn bin() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_soda-candidate-setup"))
}

/// `cmd` builds a hermetic invocation: the working directory is a fixture,
/// `$PWD` matches it (the binary reads `$PWD` like the script), ambient
/// `SODA_*` inputs are removed, and `SODA_FORGEJO_SOURCE` starts absent.
fn cmd(cwd: &Path) -> Command {
    let mut command = Command::new(bin());
    command.current_dir(cwd);
    command.env("PWD", cwd);
    for var in [
        "SODA_FORGEJO_SOURCE",
        "SODA_REPOSITORY_PREFIX",
        "SODA_REFRESH_AUTHORITY",
        "SODA_CANDIDATE_ROOT",
        "SODA_CANDIDATE_HOME",
        "SODA_CANDIDATE_RUN",
        "SODA_CANDIDATE_SCRATCH",
        "SODA_ROTATE_ACK",
        "SUDO_USER",
    ] {
        command.env_remove(var);
    }
    command
}

fn write_fake(dir: &Path, name: &str, body: &str) {
    let path = dir.join(name);
    fs::write(&path, format!("#!/bin/sh\n{body}\n")).unwrap();
    fs::set_permissions(&path, fs::Permissions::from_mode(0o755)).unwrap();
}

fn real_tool(name: &str) -> Option<PathBuf> {
    let raw = env::var_os("PATH")?;
    for dir in env::split_paths(&raw) {
        let candidate = dir.join(name);
        if let Ok(meta) = fs::metadata(&candidate) {
            if meta.is_file() && meta.permissions().mode() & 0o111 != 0 {
                return Some(candidate);
            }
        }
    }
    None
}

/// Every fake bin dir replaces `ip` with a noisy failure: the script
/// discards `ip` stderr (`2>/dev/null`), so the exact-stderr assertions in
/// every test below also prove the port suppresses it.
fn fake_ip(dir: &Path) {
    write_fake(dir, "ip", "echo ip-noise >&2; exit 1");
}

fn git_init(dir: &Path) -> bool {
    let git = match real_tool("git") {
        Some(path) => path,
        None => return false,
    };
    for args in [
        vec!["init", "-q"],
        vec!["-c", "user.name=t", "-c", "user.email=t@t", "add", "-A"],
        vec![
            "-c",
            "user.name=t",
            "-c",
            "user.email=t@t",
            "commit",
            "-qm",
            "init",
        ],
    ] {
        let status = Command::new(&git)
            .args(&args)
            .current_dir(dir)
            .env("GIT_CONFIG_NOSYSTEM", "1")
            .env("HOME", dir)
            .status();
        if !matches!(status, Ok(status) if status.success()) {
            return false;
        }
    }
    true
}

extern "C" {
    fn flock(fd: i32, op: i32) -> i32;
}

/// The holder keeps the lock in this process: no subprocess can be
/// orphaned with the file description still open.
struct HeldLock {
    _file: fs::File,
}

fn hold_flock_lock(path: &Path) -> Option<HeldLock> {
    let file = fs::OpenOptions::new()
        .write(true)
        .create(true)
        .truncate(true)
        .open(path)
        .ok()?;
    use std::os::unix::io::AsRawFd;
    if unsafe { flock(file.as_raw_fd(), 2 | 4) } != 0 {
        return None;
    }
    Some(HeldLock { _file: file })
}

#[test]
fn fails_outside_repo_root() {
    let root = TempDir::new("noroot");
    let fakes = TempDir::new("noroot-bin");
    fake_ip(&fakes.path);
    let mut command = cmd(&root.path);
    command.env(
        "PATH",
        format!("{}:{}", fakes.path.display(), env::var("PATH").unwrap()),
    );
    let out = command.output().unwrap();
    assert_eq!(out.status.code(), Some(1));
    assert!(out.stdout.is_empty());
    assert_eq!(
        String::from_utf8_lossy(&out.stderr),
        "setup-soda-candidate: run from the repository root\n"
    );
}

#[test]
fn requires_forgejo_source() {
    let root = TempDir::new("noforgejo");
    fs::write(root.path.join("go.mod"), "module fixture\n\ngo 1.26.7\n").unwrap();
    let fakes = TempDir::new("noforgejo-bin");
    fake_ip(&fakes.path);
    let mut command = cmd(&root.path);
    command.env(
        "PATH",
        format!("{}:{}", fakes.path.display(), env::var("PATH").unwrap()),
    );
    let out = command.output().unwrap();
    assert_eq!(out.status.code(), Some(1));
    assert!(out.stdout.is_empty());
    assert_eq!(
        String::from_utf8_lossy(&out.stderr),
        "setup-soda-candidate: set SODA_FORGEJO_SOURCE to the clean canonical Forgejo fork checkout\n"
    );
}

#[test]
fn rejects_relative_forgejo() {
    let root = TempDir::new("relforgejo");
    fs::write(root.path.join("go.mod"), "module fixture\n\ngo 1.26.7\n").unwrap();
    let fakes = TempDir::new("relforgejo-bin");
    fake_ip(&fakes.path);
    let mut command = cmd(&root.path);
    command.env(
        "PATH",
        format!("{}:{}", fakes.path.display(), env::var("PATH").unwrap()),
    );
    command.env("SODA_FORGEJO_SOURCE", "relative/path");
    let out = command.output().unwrap();
    assert_eq!(out.status.code(), Some(1));
    assert!(out.stdout.is_empty());
    assert_eq!(
        String::from_utf8_lossy(&out.stderr),
        "setup-soda-candidate: canonical Forgejo checkout required\n"
    );
}

#[test]
fn rejects_dirty_forgejo() {
    if real_tool("git").is_none() {
        eprintln!("skipping: git unavailable");
        return;
    }
    let root = TempDir::new("dirtyforgejo");
    fs::write(root.path.join("go.mod"), "module fixture\n\ngo 1.26.7\n").unwrap();
    let forgejo = TempDir::new("dirtyforgejo-src");
    fs::write(forgejo.path.join("file"), "v1\n").unwrap();
    assert!(git_init(&forgejo.path));
    fs::write(forgejo.path.join("file"), "v2-uncommitted\n").unwrap();
    let fakes = TempDir::new("dirtyforgejo-bin");
    fake_ip(&fakes.path);
    let mut command = cmd(&root.path);
    command.env(
        "PATH",
        format!("{}:{}", fakes.path.display(), env::var("PATH").unwrap()),
    );
    command.env("SODA_FORGEJO_SOURCE", &forgejo.path);
    let out = command.output().unwrap();
    assert_eq!(out.status.code(), Some(1));
    assert!(out.stdout.is_empty());
    assert_eq!(
        String::from_utf8_lossy(&out.stderr),
        "setup-soda-candidate: Forgejo source must be clean and committed\n"
    );
}

#[test]
fn reports_missing_tool() {
    let (Some(git), Some(realpath)) = (real_tool("git"), real_tool("realpath")) else {
        eprintln!("skipping: git or realpath unavailable");
        return;
    };
    let root = TempDir::new("missingtool");
    fs::write(root.path.join("go.mod"), "module fixture\n\ngo 1.26.7\n").unwrap();
    let forgejo = TempDir::new("missingtool-src");
    fs::write(forgejo.path.join("file"), "v1\n").unwrap();
    assert!(git_init(&forgejo.path));
    // Restricted PATH with every prerequisite except `go`; git and realpath
    // delegate to the real tools so the earlier checks pass.
    let fakes = TempDir::new("missingtool-bin");
    fake_ip(&fakes.path);
    write_fake(&fakes.path, "uname", "echo x86_64");
    write_fake(&fakes.path, "id", "exit 0");
    write_fake(
        &fakes.path,
        "git",
        &format!("exec {} \"$@\"", git.display()),
    );
    write_fake(
        &fakes.path,
        "realpath",
        &format!("exec {} \"$@\"", realpath.display()),
    );
    for tool in ["bun", "podman", "skopeo", "python3", "flock"] {
        write_fake(&fakes.path, tool, "exit 0");
    }
    let mut command = cmd(&root.path);
    command.env("PATH", &fakes.path);
    command.env("SODA_FORGEJO_SOURCE", &forgejo.path);
    let out = command.output().unwrap();
    assert_eq!(out.status.code(), Some(1));
    assert!(out.stdout.is_empty());
    assert_eq!(
        String::from_utf8_lossy(&out.stderr),
        "setup-soda-candidate: pinned go, bun, podman, skopeo, python3 and flock required (missing go)\n"
    );
}

fn full_fake_bin(tag: &str, smart_go: bool, delegating_id: bool) -> Option<TempDir> {
    let fakes = TempDir::new(tag);
    fake_ip(&fakes.path);
    write_fake(&fakes.path, "uname", "echo x86_64");
    if delegating_id {
        let id = real_tool("id")?;
        write_fake(
            &fakes.path,
            "id",
            &format!(
                "if [ \"$1\" = \"-un\" ]; then exec {} -un; else exit 0; fi",
                id.display()
            ),
        );
    } else {
        write_fake(&fakes.path, "id", "exit 0");
    }
    if smart_go {
        write_fake(
            &fakes.path,
            "go",
            "if [ \"$1\" = \"env\" ]; then echo \"/fake/goroot\"; fi\nexit 0",
        );
    } else {
        write_fake(&fakes.path, "go", "exit 0");
    }
    for tool in ["bun", "podman", "skopeo", "python3", "flock"] {
        write_fake(&fakes.path, tool, "exit 0");
    }
    Some(fakes)
}

fn clean_repo(tag: &str, go_mod: &str) -> Option<TempDir> {
    real_tool("git")?;
    let repo = TempDir::new(tag);
    fs::write(repo.path.join("go.mod"), go_mod).unwrap();
    if !git_init(&repo.path) {
        return None;
    }
    Some(repo)
}

#[test]
fn missing_go_pin_exits_silently() {
    // The script's `grep | awk` pipeline fails under `pipefail` before the
    // pin check runs, so a go.mod without a `go` line exits 1 with no
    // output at all. The port preserves the quirk.
    let (Some(root), Some(forgejo)) = (
        clean_repo("nopin", "module fixture\n"),
        clean_repo("nopin-src", "module forgejo\n"),
    ) else {
        eprintln!("skipping: git unavailable");
        return;
    };
    let fakes = full_fake_bin("nopin-bin", false, false).unwrap();
    let mut command = cmd(&root.path);
    command.env(
        "PATH",
        format!("{}:{}", fakes.path.display(), env::var("PATH").unwrap()),
    );
    command.env("SODA_FORGEJO_SOURCE", &forgejo.path);
    let out = command.output().unwrap();
    assert_eq!(out.status.code(), Some(1));
    assert!(out.stdout.is_empty());
    assert!(out.stderr.is_empty());
}

#[test]
fn refuses_held_lease() {
    let _serial = LEASE_SERIAL
        .lock()
        .unwrap_or_else(|poisoned| poisoned.into_inner());
    let (Some(root), Some(forgejo)) = (
        clean_repo("lease", "module fixture\n\ngo 1.26.7\n"),
        clean_repo("lease-src", "module forgejo\n"),
    ) else {
        eprintln!("skipping: git unavailable");
        return;
    };
    let user = Command::new("id").arg("-un").output().unwrap();
    assert!(user.status.success());
    let name = String::from_utf8_lossy(&user.stdout);
    let lock = PathBuf::from(format!("/tmp/soda-setup-{}.lock", name.trim()));
    let Some(_holder) = hold_flock_lock(&lock) else {
        eprintln!("skipping: lock already held by a live setup");
        return;
    };
    let fakes = full_fake_bin("lease-bin", true, true).unwrap();
    let mut command = cmd(&root.path);
    command.env(
        "PATH",
        format!("{}:{}", fakes.path.display(), env::var("PATH").unwrap()),
    );
    command.env("SODA_FORGEJO_SOURCE", &forgejo.path);
    let out = command.output().unwrap();
    assert_eq!(out.status.code(), Some(1));
    assert!(out.stdout.is_empty());
    assert_eq!(
        String::from_utf8_lossy(&out.stderr),
        "setup-soda-candidate: another setup is already running for this operator\n"
    );
}

#[test]
fn lists_units_quietly() {
    // `systemctl` stderr is discarded (`2>/dev/null`): only the fail
    // message surfaces when the listing itself errors.
    let _serial = LEASE_SERIAL.lock().unwrap_or_else(|poisoned| poisoned.into_inner());
    let (Some(root), Some(forgejo)) = (
        clean_repo("nols", "module fixture\n\ngo 1.26.7\n"),
        clean_repo("nols-src", "module forgejo\n"),
    ) else {
        eprintln!("skipping: git unavailable");
        return;
    };
    let fakes = full_fake_bin("nols-bin", true, true).unwrap();
    write_fake(&fakes.path, "systemctl", "echo boom >&2; exit 1");
    let mut command = cmd(&root.path);
    command.env(
        "PATH",
        format!("{}:{}", fakes.path.display(), env::var("PATH").unwrap()),
    );
    command.env("SODA_FORGEJO_SOURCE", &forgejo.path);
    let out = command.output().unwrap();
    assert_eq!(out.status.code(), Some(1));
    assert!(out.stdout.is_empty());
    assert_eq!(
        String::from_utf8_lossy(&out.stderr),
        "setup-soda-candidate: cannot list worker units; refusing to touch shared build state\n"
    );
}

#[test]
fn refuses_active_build() {
    let _serial = LEASE_SERIAL
        .lock()
        .unwrap_or_else(|poisoned| poisoned.into_inner());
    let (Some(root), Some(forgejo)) = (
        clean_repo("active", "module fixture\n\ngo 1.26.7\n"),
        clean_repo("active-src", "module forgejo\n"),
    ) else {
        eprintln!("skipping: git unavailable");
        return;
    };
    let fakes = full_fake_bin("active-bin", true, true).unwrap();
    write_fake(
        &fakes.path,
        "systemctl",
        "echo 'soda-build-x.service loaded active running desc'",
    );
    let mut command = cmd(&root.path);
    command.env(
        "PATH",
        format!("{}:{}", fakes.path.display(), env::var("PATH").unwrap()),
    );
    command.env("SODA_FORGEJO_SOURCE", &forgejo.path);
    let out = command.output().unwrap();
    assert_eq!(out.status.code(), Some(1));
    assert!(out.stdout.is_empty());
    assert_eq!(
        String::from_utf8_lossy(&out.stderr),
        "setup-soda-candidate: a candidate build is still active; finish it before rerunning setup\n"
    );
}

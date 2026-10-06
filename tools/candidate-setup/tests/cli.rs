// Integration tests for soda-candidate-setup. Every case fails before the
// first mutation (`sudo mkdir`), so nothing here touches protected state:
// no sudo, no services, no credentials. Deeper paths are covered by unit
// tests plus review; the privileged flow itself only runs on real operator
// hardware.

use std::env;
use std::fs;
use std::path::PathBuf;
use std::process::Command;

mod support;

use support::{cmd, fake_ip, git_init, hold_flock_lock, real_tool, write_fake, TempDir};

// The two lease tests share the real per-operator lock path, so they must
// not run concurrently with each other.
static LEASE_SERIAL: std::sync::Mutex<()> = std::sync::Mutex::new(());

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
    for tool in ["bun", "podman", "skopeo", "flock"] {
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
        "setup-soda-candidate: pinned go, bun, podman, skopeo and flock required (missing go)\n"
    );
}

#[test]
fn missing_ip_stays_silent() {
    // `ip` is entirely absent from PATH: the script's `$(ip ... 2>/dev/null)`
    // swallows even the shell's own "command not found", so the port must
    // print no spawn diagnostic either. Fails at the go.mod gate, before the
    // lease; no side effects.
    let root = TempDir::new("noip");
    let empty = TempDir::new("noip-bin");
    let mut command = cmd(&root.path);
    command.env("PATH", &empty.path);
    let out = command.output().unwrap();
    assert_eq!(out.status.code(), Some(1));
    assert!(out.stdout.is_empty());
    assert_eq!(
        String::from_utf8_lossy(&out.stderr),
        "setup-soda-candidate: run from the repository root\n"
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
    for tool in ["bun", "podman", "skopeo", "flock"] {
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
    let _serial = LEASE_SERIAL
        .lock()
        .unwrap_or_else(|poisoned| poisoned.into_inner());
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

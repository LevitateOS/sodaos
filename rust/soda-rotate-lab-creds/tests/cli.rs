// Integration tests for soda-rotate-lab-creds. Every case stays on the
// read-only side (inventory, runbooks, help, refusal paths), so nothing
// here touches credentials, services, or host state. The --execute success
// path is verified by review plus the differential harness, never here.

use std::env;
use std::fs;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::sync::atomic::{AtomicU64, Ordering};

static COUNTER: AtomicU64 = AtomicU64::new(0);

struct TempDir {
    path: PathBuf,
}

impl TempDir {
    fn new(tag: &str) -> TempDir {
        let id = COUNTER.fetch_add(1, Ordering::SeqCst);
        let path =
            env::temp_dir().join(format!("soda-rotate-it-{tag}-{}-{id}", std::process::id()));
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
    PathBuf::from(env!("CARGO_BIN_EXE_soda-rotate-lab-creds"))
}

/// `cmd` builds a hermetic invocation: fixture working directory and home,
/// ambient rotation inputs removed.
fn cmd(cwd: &Path, home: &Path) -> Command {
    let mut command = Command::new(bin());
    command.current_dir(cwd);
    command.env("PWD", cwd);
    command.env("HOME", home);
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

const HELP_HEADER: &str = "# rotate-lab-creds.sh — lab credential inventory and owner-gated rotation (B7).\n#\n# Context: the D3 rootfs HTTP exposure served live guest disks and images\n# world-readable over HTTP. Any secret that ever lived in those bytes must\n# be treated as exposed until the owner rotates it. Rotation itself is an\n# explicit owner decision: this script defaults to read-only inventory and\n# prints runbooks; only --execute with SODA_ROTATE_ACK=<class> mutates, and\n# only for fully scriptable classes. Secrets travel via 0600 files, never\n# argv, environment values in logs, or stdout.\n";

const RUNBOOK_FIXTURE: &str = "-- runbook: fixture-authority (fully scriptable with --execute)\nRegenerates the fixture-only media authority (never release keys):\n  SODA_ROTATE_ACK=fixture-authority cargo run -p soda-rotate-lab-creds --rotate fixture-authority --execute\nEffect: old fixture signatures stop verifying; in-flight development\nattempts using the old authority fail closed and must rerun setup.\n";

const RUNBOOK_CLOUDFLARED: &str = "-- runbook: cloudflared-token (owner-manual; dashboard-issued)\n1. Rotate the tunnel token in the Cloudflare dashboard.\n2. As root, install it with secret-file handling only:\n     install -m 0640 -o root -g cloudflared /path/to/new.token \\\n       /etc/cloudflared/dimensionlab-forgejo-https.token\n   Never pass the token on a command line; shred the staging copy after.\n3. Restart the tunnel service and re-run this script (inventory).\n";

const RUNBOOK_RUNNER: &str = "-- runbook: forgejo-runner (owner-manual; needs a Forgejo admin token)\n1. Revoke the runner registration in Forgejo and create a new token.\n2. Stop the runner, replace the secret file (0600) without argv exposure,\n   re-register, and restart the runner.\n3. Re-run this script (inventory) to confirm modes.\n";

const RUNBOOK_LAB_VM: &str = "-- runbook: lab-vm-operator (owner-manual; protected VMs)\nGuest operator credentials possibly baked into the served QCOW2s cannot be\naudited from the host (live disks are never mounted here). After the D3\nreplacement is installed, rotate operator/SSH material from the guest\nconsoles, then record the rotation date with the owner.\n";

#[test]
fn help_spellings_print_header_usage_and_classes() {
    let root = TempDir::new("help");
    let home = TempDir::new("help-home");
    for flag in ["-h", "--help", "help"] {
        let out = cmd(&root.path, &home.path).arg(flag).output().unwrap();
        assert_eq!(out.status.code(), Some(0));
        assert!(out.stderr.is_empty());
        let want = format!(
            "{HELP_HEADER}usage: {} [inventory|--rotate CLASS [--execute]]\nclasses: fixture-authority cloudflared-token forgejo-runner lab-vm-operator\n",
            bin().display()
        );
        assert_eq!(String::from_utf8_lossy(&out.stdout), want);
    }
}

#[test]
fn inventory_default_and_explicit_match_and_stay_structured() {
    let root = TempDir::new("inv");
    let home = TempDir::new("inv-home");
    let plain = cmd(&root.path, &home.path).output().unwrap();
    let explicit = cmd(&root.path, &home.path)
        .arg("inventory")
        .output()
        .unwrap();
    assert_eq!(plain.status.code(), Some(0));
    assert_eq!(explicit.status.code(), Some(0));
    assert!(plain.stderr.is_empty());
    assert_eq!(plain.stdout, explicit.stdout);
    let text = String::from_utf8_lossy(&plain.stdout);
    for section in [
        "-- fixture-only media authority (dev scope; regenerable)",
        "-- cloudflared tunnel credentials (dashboard-issued)",
        "-- forgejo runner registration (host service config)",
        "-- build metadata (public pins only; informational)",
        "-- D3 exposure reminder",
        "inventory complete; no values printed, nothing mutated.",
    ] {
        assert!(text.contains(section), "missing {section}:\n{text}");
    }
}

#[test]
fn inventory_ignores_extra_arguments() {
    let root = TempDir::new("invextra");
    let home = TempDir::new("invextra-home");
    let plain = cmd(&root.path, &home.path)
        .arg("inventory")
        .output()
        .unwrap();
    let extra = cmd(&root.path, &home.path)
        .args(["inventory", "extra", "--execute", "junk"])
        .output()
        .unwrap();
    assert_eq!(plain.status.code(), Some(0));
    assert_eq!(extra.status.code(), Some(0));
    assert_eq!(plain.stdout, extra.stdout);
    assert!(extra.stderr.is_empty());
}

#[test]
fn inventory_reports_live_inputs_when_present() {
    let root = TempDir::new("inputs");
    let home = TempDir::new("inputs-home");
    let dir = root.path.join(".artifacts/releases/isolated");
    fs::create_dir_all(&dir).unwrap();
    fs::write(dir.join("soda-live-inputs-01.json"), b"{}").unwrap();
    let out = cmd(&root.path, &home.path).output().unwrap();
    assert_eq!(out.status.code(), Some(0));
    let text = String::from_utf8_lossy(&out.stdout);
    let want = format!(
        "INFO    {}/.artifacts/releases/isolated/soda-live-inputs-01.json carries public URLs/hashes only; 0644 is expected",
        root.path.display()
    );
    assert!(text.contains(&want), "missing INFO line:\n{text}");
}

#[test]
fn rotate_runbooks_are_exact_with_or_without_execute() {
    let root = TempDir::new("runbooks");
    let home = TempDir::new("runbooks-home");
    for (class, want) in [
        ("fixture-authority", RUNBOOK_FIXTURE),
        ("cloudflared-token", RUNBOOK_CLOUDFLARED),
        ("forgejo-runner", RUNBOOK_RUNNER),
        ("lab-vm-operator", RUNBOOK_LAB_VM),
    ] {
        let out = cmd(&root.path, &home.path)
            .args(["--rotate", class])
            .output()
            .unwrap();
        assert_eq!(out.status.code(), Some(0), "{class}");
        assert!(out.stderr.is_empty(), "{class}");
        assert_eq!(String::from_utf8_lossy(&out.stdout), want, "{class}");
    }
    // Manual classes print runbooks even with --execute.
    for (class, want) in [
        ("cloudflared-token", RUNBOOK_CLOUDFLARED),
        ("forgejo-runner", RUNBOOK_RUNNER),
        ("lab-vm-operator", RUNBOOK_LAB_VM),
    ] {
        let out = cmd(&root.path, &home.path)
            .args(["--rotate", class, "--execute"])
            .output()
            .unwrap();
        assert_eq!(out.status.code(), Some(0), "{class}");
        assert_eq!(String::from_utf8_lossy(&out.stdout), want, "{class}");
    }
}

#[test]
fn unknown_command_and_class_fail_cleanly() {
    let root = TempDir::new("unknown");
    let home = TempDir::new("unknown-home");
    for (args, message) in [
        (vec!["bogus"], "unknown command 'bogus'; see --help"),
        (
            vec!["--rotate", "bogus"],
            "unknown class 'bogus'; see --help",
        ),
        (vec!["--rotate"], "unknown class ''; see --help"),
        (
            vec!["--rotate", "--execute", "fixture-authority"],
            "unknown class '--execute'; see --help",
        ),
    ] {
        let out = cmd(&root.path, &home.path).args(&args).output().unwrap();
        assert_eq!(out.status.code(), Some(1), "{args:?}");
        assert!(out.stdout.is_empty(), "{args:?}");
        assert_eq!(
            String::from_utf8_lossy(&out.stderr),
            format!("rotate-lab-creds: {message}\n"),
            "{args:?}"
        );
    }
}

#[test]
fn execute_requires_acknowledgement() {
    let root = TempDir::new("ack");
    let home = TempDir::new("ack-home");
    for ack in [None, Some("cloudflared-token"), Some("")] {
        let mut command = cmd(&root.path, &home.path);
        command.args(["--rotate", "fixture-authority", "--execute"]);
        if let Some(value) = ack {
            command.env("SODA_ROTATE_ACK", value);
        }
        let out = command.output().unwrap();
        assert_eq!(out.status.code(), Some(1), "{ack:?}");
        assert!(out.stdout.is_empty(), "{ack:?}");
        assert_eq!(
            String::from_utf8_lossy(&out.stderr),
            "rotate-lab-creds: refusing: set SODA_ROTATE_ACK=fixture-authority to execute\n",
            "{ack:?}"
        );
    }
}

#[test]
fn execute_requires_skopeo() {
    let root = TempDir::new("noskopeo");
    let home = TempDir::new("noskopeo-home");
    let empty = TempDir::new("noskopeo-bin");
    let out = cmd(&root.path, &home.path)
        .args(["--rotate", "fixture-authority", "--execute"])
        .env("SODA_ROTATE_ACK", "fixture-authority")
        .env("PATH", &empty.path)
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(1));
    assert!(out.stdout.is_empty());
    assert_eq!(
        String::from_utf8_lossy(&out.stderr),
        "rotate-lab-creds: skopeo required\n"
    );
}

#[test]
fn closed_stdout_dies_by_sigpipe_like_shell() {
    use std::os::fd::FromRawFd;
    use std::os::unix::process::ExitStatusExt;
    use std::process::Stdio;

    extern "C" {
        fn pipe(fds: *mut i32) -> i32;
        fn close(fd: i32) -> i32;
    }

    let root = TempDir::new("sigpipe");
    let home = TempDir::new("sigpipe-home");
    let mut fds = [0i32; 2];
    assert_eq!(unsafe { pipe(fds.as_mut_ptr()) }, 0);
    // Close the read end before spawn: the first inventory write to stdout
    // must raise SIGPIPE, which the binary leaves at SIG_DFL like the shell
    // instead of Rust's default SIG_IGN (which would panic with EPIPE).
    assert_eq!(unsafe { close(fds[0]) }, 0);
    let stdout = unsafe { Stdio::from_raw_fd(fds[1]) };
    let status = cmd(&root.path, &home.path)
        .arg("inventory")
        .env("USER", "testop")
        .stdout(stdout)
        .stderr(Stdio::null())
        .status()
        .unwrap();
    assert_eq!(
        status.signal(),
        Some(13),
        "expected death by SIGPIPE, got {status:?}"
    );
}

#[test]
fn unset_pwd_user_and_home_crash_like_set_u() {
    // The script expands $PWD/$USER/$HOME bare under `set -u`; the port
    // crashes with the same shape (fixed prefix instead of `$0`-and-line).
    // All three crash before any mutation: inventory is read-only.
    let root = TempDir::new("unbound");
    let home = TempDir::new("unbound-home");
    for (remove, set_user, message) in [
        ("PWD", false, "PWD: unbound variable"),
        ("USER", false, "USER: unbound variable"),
        ("HOME", true, "HOME: unbound variable"),
    ] {
        let mut command = cmd(&root.path, &home.path);
        command.arg("inventory").env_remove(remove);
        if set_user {
            // USER must be set so the earlier missing-path stats pass and
            // the crash lands on HOME instead.
            command.env("USER", "testop");
        }
        let out = command.output().unwrap();
        assert_eq!(out.status.code(), Some(1), "{message}");
        assert_eq!(
            String::from_utf8_lossy(&out.stderr),
            format!("rotate-lab-creds: {message}\n"),
            "{message}"
        );
    }
}

#[test]
fn missing_stat_stays_silent_like_redirected_shell() {
    // `stat` is entirely absent from PATH: the script's
    // `$(stat ... 2>/dev/null)` swallows even the shell's own "command not
    // found", so the port must print no spawn diagnostic either. A present
    // file forces the suppressed stat down the SpawnFailed path; absent
    // paths never spawn.
    let root = TempDir::new("nostat");
    let home = TempDir::new("nostat-home");
    let empty = TempDir::new("nostat-bin");
    let data = home.path.join("containers/forgejo-runner/data");
    fs::create_dir_all(&data).unwrap();
    fs::write(data.join("config.yaml"), "SECRET\n").unwrap();
    let out = cmd(&root.path, &home.path)
        .arg("inventory")
        .env("USER", "testop")
        .env("PATH", &empty.path)
        .output()
        .unwrap();
    assert_eq!(out.status.code(), Some(0));
    assert!(out.stderr.is_empty());
    assert!(String::from_utf8_lossy(&out.stdout).contains("unreadable (run with read access)"));
}

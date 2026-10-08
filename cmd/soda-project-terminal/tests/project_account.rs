//! Binary contract tests: exact stdin/stdout/stderr/exit bytes plus
//! filesystem effects, driven against the compiled `project-account` with
//! `SODA_PROJECT_ACCOUNT_TEST_ROOT` and PATH doubles for useradd/usermod.
//! Provisioning tests never touch real accounts; the status test performs a
//! read-only NSS query for the current process account/groups.

use std::ffi::{CStr, CString};
use std::io::Write;
use std::os::unix::fs::{MetadataExt as _, PermissionsExt};
use std::os::unix::io::AsRawFd;
use std::path::{Path, PathBuf};
use std::process::Stdio;
use std::sync::atomic::{AtomicU64, Ordering};

const FAILURE_LINE: &[u8] =
    b"account provisioning unconfirmed; inspect native account and managed files\n";
const STATUS_FAILURE_LINE: &[u8] =
    b"account privilege unconfirmed; inspect native account and managed files\n";

fn bin_path() -> PathBuf {
    if let Some(path) = option_env!("CARGO_BIN_EXE_project_account") {
        return PathBuf::from(path);
    }
    let mut dir = std::env::current_exe().expect("current exe");
    dir.pop();
    if dir.file_name().is_some_and(|n| n == "deps") {
        dir.pop();
    }
    dir.join("project-account")
}

static TEST_SEQ: AtomicU64 = AtomicU64::new(0);

/// One hermetic binary run: temp root, managed dirs, PATH doubles that log
/// argv and simulate user creation. Each test owns its env.
struct Env {
    root: PathBuf,
}

impl Env {
    fn setup(tag: &str) -> Env {
        let n = TEST_SEQ.fetch_add(1, Ordering::Relaxed);
        let root = std::env::temp_dir().join(format!(
            "soda-project-account-bin-{tag}-{}-{n}",
            std::process::id()
        ));
        std::fs::create_dir_all(root.join("accounts")).expect("accounts");
        std::fs::create_dir_all(root.join("keys")).expect("keys");
        std::fs::create_dir_all(root.join("bin")).expect("bin");
        std::fs::set_permissions(
            root.join("accounts"),
            std::fs::Permissions::from_mode(0o700),
        )
        .expect("accounts mode");
        std::fs::set_permissions(root.join("keys"), std::fs::Permissions::from_mode(0o755))
            .expect("keys mode");
        // Doubles log argv and simulate creation; stdout stays silent so the
        // binary's exact stdout contract is observable.
        std::fs::write(
            root.join("bin/useradd"),
            "#!/bin/sh\nprintf '%s\\n' \"useradd $*\" >>\"$SODA_PROJECT_ACCOUNT_TEST_ROOT/commands.log\"\nlogin=\nfor last in \"$@\"; do login=$last; done\nhome=$SODA_PROJECT_ACCOUNT_TEST_ROOT/home/$login\nmkdir -p \"$home\"\nprintf '%s:%s\\n' \"$login\" \"$home\" >>\"$SODA_PROJECT_ACCOUNT_TEST_ROOT/passwd\"\n",
        )
        .expect("useradd double");
        std::fs::write(
            root.join("bin/usermod"),
            "#!/bin/sh\nprintf '%s\\n' \"usermod $*\" >>\"$SODA_PROJECT_ACCOUNT_TEST_ROOT/commands.log\"\n",
        )
        .expect("usermod double");
        for tool in ["useradd", "usermod"] {
            std::fs::set_permissions(
                root.join("bin").join(tool),
                std::fs::Permissions::from_mode(0o755),
            )
            .expect("double mode");
        }
        Env { root }
    }

    fn run(
        &self,
        args: &[&str],
        stdin: &[u8],
        extra_env: &[(&str, &str)],
    ) -> (i32, Vec<u8>, Vec<u8>) {
        let path = format!(
            "{}:{}",
            self.root.join("bin").display(),
            std::env::var("PATH").unwrap_or_default()
        );
        let mut command = std::process::Command::new(bin_path());
        command
            .args(args)
            .env("SODA_PROJECT_ACCOUNT_TEST_ROOT", &self.root)
            .env("PATH", path)
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped());
        for (key, value) in extra_env {
            command.env(key, value);
        }
        let mut child = command.spawn().expect("spawn project-account");
        child
            .stdin
            .take()
            .expect("stdin")
            .write_all(stdin)
            .expect("write stdin");
        let output = child.wait_with_output().expect("wait");
        (
            output.status.code().expect("exit code"),
            output.stdout,
            output.stderr,
        )
    }

    fn commands(&self) -> Vec<String> {
        let log = self.root.join("commands.log");
        if !log.exists() {
            return Vec::new();
        }
        std::fs::read_to_string(log)
            .expect("commands log")
            .lines()
            .map(ToString::to_string)
            .collect()
    }

    fn mode(&self, rel: &str) -> u32 {
        std::fs::metadata(self.root.join(rel)).expect("meta").mode() & 0o777
    }

    fn read(&self, rel: &str) -> Vec<u8> {
        std::fs::read(self.root.join(rel)).expect("read")
    }
}

fn doc(login: &str, identity: i64, admin: bool, keys: &[&str]) -> Vec<u8> {
    let mut out = format!(
        "{{\"login\": \"{login}\", \"identity\": {identity}, \"admin\": {admin}, \"keys\": ["
    );
    for (i, key) in keys.iter().enumerate() {
        if i > 0 {
            out.push_str(", ");
        }
        out.push('"');
        for ch in key.chars() {
            match ch {
                '"' => out.push_str("\\\""),
                '\\' => out.push_str("\\\\"),
                '\n' => out.push_str("\\n"),
                '\r' => out.push_str("\\r"),
                '\t' => out.push_str("\\t"),
                c => out.push(c),
            }
        }
        out.push('"');
    }
    out.push_str("]}");
    out.into_bytes()
}

#[test]
fn provisions_locked_home_marker_shared_empty_keyfile() {
    let env = Env::setup("provisions");
    let (code, stdout, stderr) = env.run(&[], &doc("alice", 1, false, &[]), &[]);
    assert_eq!(code, 0);
    assert_eq!(stdout, b"{\"login\": \"alice\", \"identity\": 1}\n");
    assert!(stderr.is_empty());
    assert_eq!(env.read("keys/alice"), b"");
    assert_eq!(env.read("accounts/alice"), b"1");
    assert_eq!(env.mode("accounts/alice"), 0o600);
    assert_eq!(env.mode("keys/alice"), 0o644);
    assert_eq!(
        std::fs::read_link(env.root.join("home/alice/shared")).expect("shared"),
        Path::new("/srv/project/shared")
    );
    assert_eq!(
        env.commands(),
        ["useradd --create-home --shell /bin/bash --password ! --groups soda-project alice"]
    );
}

#[test]
fn selected_keys_admin_rerun_is_idempotent() {
    let env = Env::setup("idempotent");
    let body = doc("alice", 7, true, &["ssh-ed25519 YWJj\n"]);
    let (code, stdout, _) = env.run(&[], &body, &[]);
    assert_eq!(code, 0);
    assert_eq!(stdout, b"{\"login\": \"alice\", \"identity\": 7}\n");
    assert_eq!(env.read("keys/alice"), b"ssh-ed25519 YWJj\n");
    assert_eq!(
        env.commands(),
        [
            "useradd --create-home --shell /bin/bash --password ! --groups soda-project alice",
            "usermod --append --groups wheel alice",
        ]
    );
    std::fs::write(env.root.join("home/alice/work"), "later work").expect("work");
    let before = std::fs::metadata(env.root.join("keys/alice"))
        .expect("meta")
        .ino();
    let (code, _, _) = env.run(&[], &body, &[]);
    assert_eq!(code, 0);
    assert_eq!(
        std::fs::metadata(env.root.join("keys/alice"))
            .expect("meta")
            .ino(),
        before
    );
    assert_eq!(
        std::fs::read_to_string(env.root.join("home/alice/work")).expect("work"),
        "later work"
    );
    assert_eq!(
        env.commands()
            .iter()
            .filter(|c| c.starts_with("useradd"))
            .count(),
        1
    );
}

#[test]
fn retry_never_erases_existing_keys() {
    let env = Env::setup("retry");
    let (code, _, _) = env.run(&[], &doc("alice", 1, false, &["ssh-ed25519 YWJj"]), &[]);
    assert_eq!(code, 0);
    let (code, stdout, stderr) = env.run(&[], &doc("alice", 1, false, &[]), &[]);
    assert_eq!(code, 1);
    assert!(stdout.is_empty());
    assert_eq!(stderr, FAILURE_LINE);
    assert_eq!(env.read("keys/alice"), b"ssh-ed25519 YWJj\n");
    assert_eq!(env.commands().len(), 1);
}

#[test]
fn join_never_applies_keys_and_preserves_drift() {
    let env = Env::setup("join");
    let (code, _, _) = env.run(&[], &doc("alice", 1, false, &[]), &[]);
    assert_eq!(code, 0);
    let (code, stdout, stderr) = env.run(&[], &doc("alice", 1, false, &["ssh-ed25519 YWJj"]), &[]);
    assert_eq!(code, 1);
    assert!(stdout.is_empty());
    assert_eq!(stderr, FAILURE_LINE);
    assert_eq!(env.read("keys/alice"), b"");
    std::fs::write(env.root.join("accounts/alice"), "2").expect("drift");
    let (code, _, stderr) = env.run(&[], &doc("alice", 1, false, &[]), &[]);
    assert_eq!(code, 1);
    assert_eq!(stderr, FAILURE_LINE);
    assert_eq!(env.read("accounts/alice"), b"2");
}

#[test]
fn occupied_inputs_and_unassociated_users_refuse() {
    let env = Env::setup("occupied");
    std::os::unix::fs::symlink(env.root.join("absent"), env.root.join("keys/alice"))
        .expect("dangle");
    let (code, stdout, stderr) = env.run(&[], &doc("alice", 1, false, &[]), &[]);
    assert_eq!(code, 1);
    assert!(stdout.is_empty());
    assert_eq!(stderr, FAILURE_LINE);
    assert!(env.commands().is_empty());
    std::fs::remove_file(env.root.join("keys/alice")).expect("unlink");
    std::fs::create_dir_all(env.root.join("home/alice")).expect("home");
    std::fs::write(
        env.root.join("passwd"),
        format!("alice:{}/home/alice\n", env.root.display()),
    )
    .expect("passwd");
    let (code, _, stderr) = env.run(&[], &doc("alice", 1, false, &[]), &[]);
    assert_eq!(code, 1);
    assert_eq!(stderr, FAILURE_LINE);
    assert!(env.commands().is_empty());
}

#[test]
fn key_symlink_is_not_followed() {
    let env = Env::setup("symlink");
    let (code, _, _) = env.run(&[], &doc("alice", 1, false, &[]), &[]);
    assert_eq!(code, 0);
    std::fs::write(env.root.join("other"), "preserve").expect("other");
    std::fs::remove_file(env.root.join("keys/alice")).expect("unlink");
    std::os::unix::fs::symlink(env.root.join("other"), env.root.join("keys/alice")).expect("link");
    let (code, _, stderr) = env.run(&[], &doc("alice", 1, false, &[]), &[]);
    assert_eq!(code, 1);
    assert_eq!(stderr, FAILURE_LINE);
    assert_eq!(env.read("other"), b"preserve");
}

#[test]
fn validation_refuses_before_native_effects() {
    let env = Env::setup("validation");
    let too_many = doc("alice", 1, false, &["ssh-ed25519 YWJj"; 33]);
    for body in [
        too_many,
        br#"{"login":"alice","identity":1,"admin":false,"keys":null}"#.to_vec(),
        br#"{"login":"alice","identity":1,"admin":false,"keys":"key"}"#.to_vec(),
        br#"{"login":"alice","identity":1,"admin":false,"keys":["PRIVATE KEY"]}"#.to_vec(),
        br#"{"login":"alice","identity":1,"admin":false,"keys":["ssh-ed25519 YWJj\nssh-ed25519 ZGVm"]}"#.to_vec(),
        br#"{"login":"root","identity":1,"admin":false,"keys":[]}"#.to_vec(),
    ] {
        let (code, stdout, stderr) = env.run(&[], &body, &[]);
        assert_eq!(code, 1, "must refuse {}", String::from_utf8_lossy(&body));
        assert!(stdout.is_empty());
        assert_eq!(stderr, FAILURE_LINE);
    }
    assert!(env.commands().is_empty());
}

#[test]
fn lock_contention_refuses_before_effects() {
    let env = Env::setup("contend");
    let held = std::fs::File::open(env.root.join("keys")).expect("keys open");
    let rc = unsafe { libc::flock(held.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) };
    assert_eq!(rc, 0);
    let (code, stdout, stderr) = env.run(&[], &doc("alice", 1, false, &[]), &[]);
    assert_eq!(code, 1);
    assert!(stdout.is_empty());
    assert_eq!(stderr, FAILURE_LINE);
    assert!(env.commands().is_empty());
    assert!(std::fs::read_dir(env.root.join("keys"))
        .expect("readdir")
        .next()
        .is_none());
    assert!(std::fs::read_dir(env.root.join("accounts"))
        .expect("readdir")
        .next()
        .is_none());
    drop(held);
    let (code, stdout, _) = env.run(&[], &doc("alice", 1, false, &[]), &[]);
    assert_eq!(code, 0);
    assert_eq!(stdout, b"{\"login\": \"alice\", \"identity\": 1}\n");
}

#[test]
fn failed_sync_releases_lock_and_keeps_partial_files() {
    let env = Env::setup("failsync");
    let (code, _, stderr) = env.run(
        &[],
        &doc("alice", 1, false, &[]),
        &[("SODA_PROJECT_ACCOUNT_FAIL_SYNC", "1")],
    );
    assert_eq!(code, 1);
    assert_eq!(stderr, FAILURE_LINE);
    assert_eq!(env.read("accounts/alice"), b"1");
    assert!(env.root.join("home/alice").is_dir());
    let relock = std::fs::File::open(env.root.join("keys")).expect("keys open");
    let rc = unsafe { libc::flock(relock.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) };
    assert_eq!(rc, 0, "lock must not strand a failed provisioning");
}

#[test]
fn stdin_size_boundary_is_exact() {
    // A 65536-byte valid document succeeds; one byte more refuses.
    // Padding spreads over four keys so each stays under the 16384
    // character key limit.
    let env = Env::setup("size");
    let prefix = "{\"login\": \"alice\", \"identity\": 1, \"admin\": false, \"keys\": [";
    let suffix = "]}";
    let per = (65536 - prefix.len() - suffix.len() - 3 * 2 - 4 * 8) / 4;
    let mut lens = [per; 4];
    lens[3] += 65536 - (prefix.len() + suffix.len() + 6 + 4 * 8 + 4 * per);
    let mut edge = Vec::from(prefix.as_bytes());
    for (i, n) in lens.iter().enumerate() {
        if i > 0 {
            edge.extend_from_slice(b", ");
        }
        edge.extend_from_slice(b"\"ssh-x ");
        edge.extend(std::iter::repeat_n(b'A', *n));
        edge.push(b'"');
        assert!(6 + *n <= 16384, "key {i} exceeds the key limit");
    }
    edge.extend_from_slice(suffix.as_bytes());
    assert_eq!(edge.len(), 65536);
    let (code, stdout, _) = env.run(&[], &edge, &[]);
    assert_eq!(code, 0);
    assert_eq!(stdout, b"{\"login\": \"alice\", \"identity\": 1}\n");
    let env = Env::setup("size-over");
    let mut over = edge.clone();
    over.push(b' ');
    assert_eq!(over.len(), 65537);
    let (code, stdout, stderr) = env.run(&[], &over, &[]);
    assert_eq!(code, 1);
    assert!(stdout.is_empty());
    assert_eq!(stderr, FAILURE_LINE);
}

#[test]
fn malformed_stdin_refuses() {
    let env = Env::setup("malformed");
    for body in [
        b"".to_vec(),
        b"not json".to_vec(),
        b"null".to_vec(),
        b"[]".to_vec(),
        b"{}".to_vec(),
        b"\xff\xfe".to_vec(),
        b"{\"login\": \"alice\"}".to_vec(),
    ] {
        let (code, stdout, stderr) = env.run(&[], &body, &[]);
        assert_eq!(code, 1, "must refuse {body:?}");
        assert!(stdout.is_empty());
        assert_eq!(stderr, FAILURE_LINE);
    }
    assert!(env.commands().is_empty());
}

#[test]
fn unsupported_argument_tail_refuses_before_provisioning() {
    let env = Env::setup("argv");
    let (code, stdout, stderr) =
        env.run(&["--whatever", "else"], &doc("alice", 1, false, &[]), &[]);
    assert_eq!(code, 1);
    assert!(stdout.is_empty());
    assert_eq!(stderr, FAILURE_LINE);
    assert!(env.commands().is_empty());
    assert!(!env.root.join("accounts/alice").exists());
}

fn native_account_uid(login: &str) -> Option<u32> {
    let Ok(name) = CString::new(login) else {
        return None;
    };
    let mut entry: libc::passwd = unsafe { std::mem::zeroed() };
    let mut result: *mut libc::passwd = std::ptr::null_mut();
    let mut buffer = vec![0u8; 65536];
    let rc = unsafe {
        libc::getpwnam_r(
            name.as_ptr(),
            &mut entry,
            buffer.as_mut_ptr() as *mut libc::c_char,
            buffer.len(),
            &mut result,
        )
    };
    if rc == 0
        && !result.is_null()
        && !entry.pw_name.is_null()
        && unsafe { CStr::from_ptr(entry.pw_name).to_bytes() } == login.as_bytes()
    {
        Some(entry.pw_uid)
    } else {
        None
    }
}

fn wheel_group_resolves_through_nss() -> bool {
    let name = CString::new("wheel").expect("fixed group name");
    let mut entry: libc::group = unsafe { std::mem::zeroed() };
    let mut result: *mut libc::group = std::ptr::null_mut();
    let mut buffer = vec![0u8; 65536];
    unsafe {
        libc::getgrnam_r(
            name.as_ptr(),
            &mut entry,
            buffer.as_mut_ptr() as *mut libc::c_char,
            buffer.len(),
            &mut result,
        ) == 0
            && !result.is_null()
            && !entry.gr_name.is_null()
            && CStr::from_ptr(entry.gr_name).to_bytes() == b"wheel"
    }
}

#[test]
fn status_mode_reads_current_native_nss_without_provisioning() {
    // Use a fixed neutral system account, never the local runner's username.
    // Rocky supplies `nobody`; other systems may refuse if the account or
    // target `wheel` group is absent, which is distinct from guest qualification.
    let login = "nobody";
    let env = Env::setup("status-nss");
    let identity = 17;
    let marker = env.root.join("accounts").join(&login);
    std::fs::write(&marker, identity.to_string()).expect("write test identity marker");
    std::fs::set_permissions(&marker, std::fs::Permissions::from_mode(0o600))
        .expect("protect test identity marker");
    let request = format!(r#"{{"login":{login:?},"identity":{identity}}}"#);
    let (code, stdout, stderr) = env.run(&["--status"], request.as_bytes(), &[]);
    let native_uid = native_account_uid(login);
    let privilege_sources_resolve = match native_uid {
        Some(0) => true,
        Some(_) => wheel_group_resolves_through_nss(),
        None => false,
    };
    if native_uid.is_some() && privilege_sources_resolve {
        assert_eq!(
            code,
            0,
            "status failed: {}",
            String::from_utf8_lossy(&stderr)
        );
        assert!(stderr.is_empty());
        let response: serde_json::Value =
            serde_json::from_slice(&stdout).expect("typed status response");
        assert_eq!(response.as_object().map(|o| o.len()), Some(3));
        assert_eq!(response["login"], login);
        assert_eq!(response["identity"], identity);
        assert!(response["administrator"].is_boolean());
    } else {
        let reason = if native_uid.is_none() {
            "fixed nobody account is absent from host NSS"
        } else {
            "fixed wheel group is absent from host NSS"
        };
        assert_eq!(code, 1, "status did not fail closed: {reason}");
        assert!(stdout.is_empty());
        assert_eq!(stderr, STATUS_FAILURE_LINE);
    }
    assert!(
        env.commands().is_empty(),
        "status invoked a mutating command"
    );
    assert_eq!(
        std::fs::read(&marker).expect("marker unchanged"),
        identity.to_string().as_bytes()
    );
}

#[test]
fn status_mode_refuses_oversized_request_before_account_or_nss_effects() {
    let env = Env::setup("status-oversized");
    let body = vec![b' '; 4097];
    let (code, stdout, stderr) = env.run(&["--status"], &body, &[]);
    assert_eq!(code, 1);
    assert!(stdout.is_empty());
    assert_eq!(stderr, STATUS_FAILURE_LINE);
    assert!(env.commands().is_empty());
    assert!(std::fs::read_dir(env.root.join("accounts"))
        .expect("managed directory")
        .next()
        .is_none());
}

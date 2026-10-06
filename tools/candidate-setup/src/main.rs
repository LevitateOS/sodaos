// soda-candidate-setup prepares this machine so `sudo soda-candidate`
// runs without a coding agent: wrapper on sudo's PATH, admitted controller,
// worker directories, restricted worker config, and a fixture-only media
// authority. Development and fixture scope only: it never creates
// qualification or signing configs, and the generated keys must never
// stand in for release keys.
//
// Rust port of scripts/setup-soda-candidate.sh (plus the
// scripts/candidate-storage.sh defaults it sourced). Messages, exit codes,
// installed paths, file modes, and generated file bytes match the shell.
// Like the script, this binary takes no arguments and ignores any it is
// given. Run from the repository root.

use std::env;
use std::ffi::CString;
use std::fs;
use std::io::{self, Write};
use std::os::unix::ffi::OsStrExt;
use std::os::unix::fs::OpenOptionsExt;
use std::path::{Path, PathBuf};

mod config;
mod controller;
mod fixture_authority;
mod preflight;
mod process;
mod selinux;
mod storage;
mod worker_caches;
mod worker_tools;

#[cfg(test)]
use self::config::{config_json, json_escape, trust_json, worker_json};
#[cfg(test)]
use self::controller::controller_cargo_argv;
#[cfg(test)]
use self::fixture_authority::random_hex_passphrase;
#[cfg(test)]
use self::preflight::bridge_ip;
#[cfg(test)]
use self::process::{git_tree_clean, pipe2, run_in_dir, Captured};
use self::process::{id_un, run};
#[cfg(test)]
use self::storage::units_have_active;

const FAIL_PREFIX: &str = "setup-soda-candidate";
const PREFIX_DEFAULT: &str = "ghcr.io/levitateos/sodaos";
const STORAGE_ROOT_DEFAULT: &str = "/home/soda-candidate";
const ROOTFS_DIR: &str = "/home/soda-rootfs";
const ADMITTED: &str = "/usr/local/lib/soda/soda-build";
const PINNED_GO: &str = "/usr/local/lib/soda/pinned-go";
const WRAPPER: &str = "/usr/sbin/soda-candidate";
const TOOLS: &str = "/var/lib/soda-candidate-tools";
const WORKER_POLICY_SRC: &str = "system/host/selinux/soda-build-worker.te";
const AUTHORITY: &str = "/var/lib/soda-candidate-authority";
const LEGACY_HOME: &str = "/var/lib/soda-candidate-home";
const LEGACY_RUN: &str = "/var/lib/soda-candidate-run";
const WORKER_USER: &str = "soda-build-worker";

extern "C" {
    fn flock(fd: i32, op: i32) -> i32;
    fn faccessat(dirfd: i32, path: *const i8, mode: i32, flags: i32) -> i32;
    fn umask(mask: u32) -> u32;
    fn sigaction(signum: i32, act: *const Sigaction, oldact: *mut Sigaction) -> i32;
}

/// Matches glibc's `struct sigaction` on Linux (handler, signal mask, flags,
/// restorer). The glibc wrapper fills in the restorer itself.
#[repr(C)]
#[derive(Clone, Copy)]
struct Sigaction {
    handler: usize,
    mask: [u64; 16],
    flags: i32,
    restorer: usize,
}

const SIGPIPE: i32 = 13;
const SIG_DFL: usize = 0;
const LOCK_EX: i32 = 2;
const LOCK_NB: i32 = 4;
const AT_FDCWD: i32 = -100;
const X_OK: i32 = 1;

/// How the process ends: a `fail()` message with exit 1, or a propagated
/// child status with no extra output (the script's `set -e` behavior).
#[derive(Debug)]
enum Exit {
    Fail(String),
    Propagate(i32),
}

fn fail<T>(msg: impl Into<String>) -> Result<T, Exit> {
    Err(Exit::Fail(msg.into()))
}

/// `env_or` mirrors `${VAR:-default}`: unset or empty falls back.
fn env_or(key: &str, default: &str) -> String {
    match env::var(key) {
        Ok(v) if !v.is_empty() => v,
        _ => default.to_string(),
    }
}

/// `current_pwd` mirrors `$PWD`: the inherited logical path when present,
// otherwise the physical working directory.
fn current_pwd() -> String {
    match env::var("PWD") {
        Ok(v) => v,
        Err(_) => env::current_dir()
            .map(|p| p.to_string_lossy().into_owned())
            .unwrap_or_default(),
    }
}

fn is_dir(path: &str) -> bool {
    fs::metadata(path).map(|m| m.is_dir()).unwrap_or(false)
}

fn is_file(path: &str) -> bool {
    fs::metadata(path).map(|m| m.is_file()).unwrap_or(false)
}

/// `stripped` mirrors command substitution: all trailing newlines removed.
fn stripped(out: &[u8]) -> &[u8] {
    let mut end = out.len();
    while end > 0 && out[end - 1] == b'\n' {
        end -= 1;
    }
    &out[..end]
}

fn stripped_string(out: &[u8]) -> String {
    String::from_utf8_lossy(stripped(out)).into_owned()
}

fn current_umask() -> u32 {
    unsafe {
        let mask = umask(0);
        umask(mask);
        mask
    }
}

/// `write_staged` mirrors shell redirection into staging: bytes exact, mode
/// 0666 filtered by the operator umask like `>` and `open("w")`.
fn write_staged(path: &Path, bytes: &[u8]) -> io::Result<()> {
    let mode = 0o666 & !current_umask();
    let mut opts = fs::OpenOptions::new();
    opts.write(true).create(true).truncate(true).mode(mode);
    opts.open(path)?.write_all(bytes)
}

fn is_executable(path: &Path) -> bool {
    match CString::new(path.as_os_str().as_bytes()) {
        Ok(c) => unsafe { faccessat(AT_FDCWD, c.as_ptr(), X_OK, 0) == 0 },
        Err(_) => false,
    }
}

/// `command_v` mirrors `command -v`: a PATH search with execute permission.
fn command_v(tool: &str) -> Option<PathBuf> {
    let raw = env::var_os("PATH").unwrap_or_else(|| "/bin:/usr/bin".into());
    command_v_in(tool, &raw)
}

fn command_v_in(tool: &str, path_env: &std::ffi::OsStr) -> Option<PathBuf> {
    if tool.contains('/') {
        let path = PathBuf::from(tool);
        return is_executable(&path).then_some(path);
    }
    for dir in env::split_paths(path_env) {
        let candidate = dir.join(tool);
        if is_executable(&candidate) {
            return Some(candidate);
        }
    }
    None
}

fn run_setup(cleanup: &mut Vec<PathBuf>) -> Result<(), Exit> {
    let pre = preflight::preflight()?;
    let preflight::Preflight {
        prefix,
        refresh,
        forgejo_source,
        storage,
        rootfs_url,
        pwd,
        output_parent,
        worker_json_path,
        pinned,
        pinned_goroot,
        want,
    } = pre;

    storage::prepare_scratch(&storage)?;

    let ctl = controller::admit_controller(&storage, cleanup)?;
    let controller::ControllerPaths { bindir } = ctl;

    let tools = worker_tools::provision_worker_tools(
        &output_parent,
        &storage,
        &pinned,
        &pinned_goroot,
        &want,
    )?;
    let worker_tools::WorkerTools { bun_final, owned } = tools;

    let caches = worker_caches::warm_worker_caches(&storage, &pinned, &owned, &bun_final)?;
    let worker_caches::WorkerCaches {
        go_mod,
        go_build_cache,
    } = caches;

    selinux::install_worker_selinux(&bindir, &storage, &go_mod, &go_build_cache, &owned)?;

    fixture_authority::admit_fixture_authority(
        &pwd,
        &forgejo_source,
        &output_parent,
        &storage,
        &worker_json_path,
        &prefix,
        &refresh,
        &owned,
        cleanup,
    )?;

    run("sudo", &["mkdir", "-p", ROOTFS_DIR])?;
    let rootfs_owner = String::from_utf8_lossy(&id_un()).into_owned();
    run("sudo", &["chown", &rootfs_owner, ROOTFS_DIR])?;
    if command_v("restorecon").is_some() {
        run("sudo", &["restorecon", ROOTFS_DIR])?;
    }

    println!("-- ready. Development candidate command from {pwd}:");
    println!("  sudo {ADMITTED} --worker-config {worker_json_path} --arch x86_64 \\");
    println!("    --out {output_parent}/manual-01 --development --target candidate \\");
    println!("    --forgejo-source {forgejo_source}");
    println!("-- installer rootfs pickup: {ROOTFS_DIR} (served by soda-rootfs-server.service)");
    println!("  after a media build, copy its hash-named rootfs here:");
    println!("  sudo cp <out>/artifacts/media/*-rootfs.img {ROOTFS_DIR}/ && sudo chown root:root {ROOTFS_DIR}/*-rootfs.img && sudo chmod 0644 {ROOTFS_DIR}/*-rootfs.img");
    if !rootfs_url.is_empty() {
        println!("-- guests fetch the filed rootfs from: {rootfs_url}");
    }
    Ok(())
}

fn main() {
    // Rust ignores SIGPIPE at startup (a write to a closed pipe would return
    // EPIPE and println! would panic instead of dying like the shell);
    // restore the default disposition for byte parity.
    unsafe {
        let restore_pipe = Sigaction {
            handler: SIG_DFL,
            mask: [0; 16],
            flags: 0,
            restorer: 0,
        };
        sigaction(SIGPIPE, &restore_pipe, std::ptr::null_mut());
    }
    let mut cleanup: Vec<PathBuf> = Vec::new();
    let result = run_setup(&mut cleanup);
    for path in &cleanup {
        let _ = fs::remove_dir_all(path);
    }
    match result {
        Ok(()) => {}
        Err(Exit::Fail(msg)) => {
            eprintln!("{FAIL_PREFIX}: {msg}");
            std::process::exit(1);
        }
        Err(Exit::Propagate(code)) => std::process::exit(code),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::fs::PermissionsExt;

    #[test]
    fn json_escape_matches_python_dumps() {
        let input = "a\"b\\c\nd\re\tf\x08g\x0ch\x01i\x7fj\u{80}ké😀l\x1fm";
        assert_eq!(
            json_escape(input),
            "a\\\"b\\\\c\\nd\\re\\tf\\bg\\fh\\u0001i\\u007fj\\u0080k\\u00e9\\ud83d\\ude00l\\u001fm"
        );
        assert_eq!(json_escape("plain / path-_name.1"), "plain / path-_name.1");
        assert_eq!(json_escape(""), "");
    }

    #[test]
    fn worker_json_matches_python_dump_without_trailing_newline() {
        let got = worker_json(
            "/usr/local/lib/soda/soda-build",
            "/home/op/sodaos",
            "/home/op/forgejo",
            "/home/op/sodaos/.artifacts/releases/isolated",
            "/home/soda-candidate",
            "/home/soda-candidate/home",
            "/home/soda-candidate/run",
            "/var/lib/soda-candidate-tools",
            "/var/lib/soda-candidate-authority",
        );
        let want = "{\n  \"Executable\": \"/usr/local/lib/soda/soda-build\",\n  \"Source\": \"/home/op/sodaos\",\n  \"ForgejoSource\": \"/home/op/forgejo\",\n  \"OutputParent\": \"/home/op/sodaos/.artifacts/releases/isolated\",\n  \"StorageRoot\": \"/home/soda-candidate\",\n  \"BuildHome\": \"/home/soda-candidate/home\",\n  \"Runtime\": \"/home/soda-candidate/run\",\n  \"Tools\": \"/var/lib/soda-candidate-tools\",\n  \"MediaAuthorityDirectory\": \"/var/lib/soda-candidate-authority\"\n}";
        assert_eq!(got, want);
    }

    #[test]
    fn trust_json_matches_python_dump_with_trailing_newline() {
        let got = trust_json(
            "ghcr.io/levitateos/sodaos",
            424242,
            [
                "artifact-pub\n",
                "candidate-pub\n",
                "preview-pub\n",
                "stable-pub\n",
            ],
        );
        let want = "{\n  \"Format\": 1,\n  \"Prefix\": \"ghcr.io/levitateos/sodaos\",\n  \"Epoch\": 1,\n  \"Keys\": {\n    \"artifact\": [\n      \"artifact-pub\\n\"\n    ],\n    \"candidate\": [\n      \"candidate-pub\\n\"\n    ],\n    \"preview\": [\n      \"preview-pub\\n\"\n    ],\n    \"stable\": [\n      \"stable-pub\\n\"\n    ]\n  },\n  \"NotBefore\": 423642,\n  \"MaxAgeSeconds\": 3600,\n  \"ClockSkewSeconds\": 10,\n  \"MinimumSequence\": {\n    \"candidate\": 1,\n    \"preview\": 1,\n    \"stable\": 1\n  }\n}\n";
        assert_eq!(got, want);
    }

    #[test]
    fn config_json_matches_python_dump_with_trailing_newline() {
        let got = config_json();
        let want = "{\n  \"Trust\": \"/run/soda-media-authority/trust.json\",\n  \"Keys\": {\n    \"Key\": \"/run/soda-media-authority/artifact.private\",\n    \"Passphrase\": \"/run/soda-media-authority/passphrase\"\n  }\n}\n";
        assert_eq!(got, want);
    }

    #[test]
    fn units_have_active_matches_awk_third_field() {
        assert!(units_have_active(
            b"soda-build-x.service loaded active running desc\n"
        ));
        assert!(units_have_active(
            b"soda-build-x.service loaded activating start desc\n"
        ));
        assert!(units_have_active(
            b"soda-build-x.service loaded deactivating stop desc\n"
        ));
        assert!(!units_have_active(
            b"soda-build-x.service loaded failed failed desc\n"
        ));
        assert!(!units_have_active(b""));
        assert!(!units_have_active(b"short line\n"));
        assert!(!units_have_active(
            b"other.service loaded inactive dead desc\nsoda-build-y.service loaded failed failed desc\n"
        ));
        assert!(units_have_active(
            b"other.service loaded inactive dead desc\nsoda-build-y.service loaded active running desc\n"
        ));
    }

    #[test]
    fn bridge_ip_takes_first_line_fourth_field_before_slash() {
        assert_eq!(
            bridge_ip(
                b"2: virbr0    inet 192.168.122.1/24 brd 192.168.122.255 scope global virbr0\n"
            ),
            Some("192.168.122.1".to_string())
        );
        assert_eq!(bridge_ip(b""), None);
        assert_eq!(bridge_ip(b"2: virbr0\n"), None);
        assert_eq!(
            bridge_ip(b"2: virbr0    inet 10.0.0.1 scope global\n"),
            Some("10.0.0.1".to_string())
        );
    }

    #[test]
    fn env_or_falls_back_on_unset_or_empty() {
        let key = "SODA_CANDIDATE_SETUP_TEST_ENV_OR";
        env::remove_var(key);
        assert_eq!(env_or(key, "dflt"), "dflt");
        env::set_var(key, "");
        assert_eq!(env_or(key, "dflt"), "dflt");
        env::set_var(key, "value");
        assert_eq!(env_or(key, "dflt"), "value");
        env::remove_var(key);
    }

    #[test]
    fn stripped_removes_only_trailing_newlines() {
        assert_eq!(stripped(b"a\n"), b"a");
        assert_eq!(stripped(b"a\n\n\n"), b"a");
        assert_eq!(stripped(b"a"), b"a");
        assert_eq!(stripped(b""), b"");
        assert_eq!(stripped(b"a\nb\n"), b"a\nb");
        assert_eq!(stripped(b"\n"), b"");
    }

    #[test]
    fn command_v_searches_path_for_executables() {
        let dir = env::temp_dir().join(format!("soda-setup-cmdv-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        fs::create_dir_all(&dir).unwrap();
        let tool = dir.join("soda-test-tool");
        let flat = dir.join("soda-test-flat");
        fs::write(&tool, b"#!/bin/sh\nexit 0\n").unwrap();
        fs::write(&flat, b"data").unwrap();
        fs::set_permissions(&tool, fs::Permissions::from_mode(0o755)).unwrap();
        fs::set_permissions(&flat, fs::Permissions::from_mode(0o644)).unwrap();
        let found = command_v_in("soda-test-tool", dir.as_os_str());
        let missing = command_v_in("soda-test-flat", dir.as_os_str()).is_none()
            && command_v_in("soda-test-absent", dir.as_os_str()).is_none();
        let _ = fs::remove_dir_all(&dir);
        assert_eq!(found, Some(tool));
        assert!(missing);
    }

    #[test]
    fn pipe2_reports_last_nonzero_by_position() {
        assert_eq!(pipe2(("true", &[]), ("true", &[])), (0, Vec::new()));
        assert_eq!(pipe2(("true", &[]), ("false", &[])).0, 1);
        // pipefail: the first failure still fails the pipeline.
        assert_eq!(pipe2(("false", &[]), ("true", &[])).0, 1);
        let (code, out) = pipe2(("echo", &["hi"]), ("tr", &["a-z", "A-Z"]));
        assert_eq!((code, out), (0, b"HI\n".to_vec()));
    }

    #[test]
    fn write_staged_matches_redirection_bytes_and_mode() {
        let path = env::temp_dir().join(format!("soda-setup-stage-{}", std::process::id()));
        let _ = fs::remove_file(&path);
        write_staged(&path, b"bytes\n").unwrap();
        assert_eq!(fs::read(&path).unwrap(), b"bytes\n");
        let mode = fs::metadata(&path).unwrap().permissions().mode() & 0o777;
        assert_eq!(mode, 0o666 & !current_umask() & 0o777);
        let _ = fs::remove_file(&path);
    }

    #[test]
    fn passphrase_is_64_lowercase_hex_without_newline() {
        let passphrase = random_hex_passphrase().unwrap();
        assert_eq!(passphrase.len(), 64);
        assert!(passphrase
            .bytes()
            .all(|b| b.is_ascii_hexdigit() && !b.is_ascii_uppercase()));
        assert_ne!(
            random_hex_passphrase().unwrap(),
            random_hex_passphrase().unwrap()
        );
    }

    #[test]
    fn run_in_dir_maps_missing_directory_to_fail_message() {
        match run_in_dir(
            "true",
            &[],
            "/definitely/not/a/soda-setup-dir",
            true,
            "cannot warm Bun cache",
        ) {
            Err(Exit::Fail(msg)) => assert_eq!(msg, "cannot warm Bun cache"),
            other => panic!("expected Fail, got {other:?}"),
        }
    }

    #[test]
    fn git_tree_clean_requires_successful_empty_status() {
        // D01-F3: only a successful status with empty output proves clean.
        assert!(git_tree_clean(&Captured::Done(0, Vec::new())));
        assert!(git_tree_clean(&Captured::Done(0, b"\n".to_vec())));
        assert!(!git_tree_clean(&Captured::Done(
            0,
            b" M src/main.rs\n".to_vec()
        )));
        assert!(!git_tree_clean(&Captured::Done(1, Vec::new())));
        assert!(!git_tree_clean(&Captured::SpawnFailed(127)));
    }

    #[test]
    fn controller_build_selects_rust_release_tools() {
        // D01-F2: pinned Rust recipe, never the retired Go paths.
        let argv = controller_cargo_argv();
        assert_eq!(
            argv.as_slice(),
            &[
                "build",
                "--release",
                "--locked",
                "-p",
                "soda-release-tools",
                "--bin",
                "soda-build",
                "--bin",
                "soda-candidate",
            ]
        );
        assert!(!argv.iter().any(|a| a.contains("tools/soda-")));
    }

    #[test]
    fn worker_policy_input_resolves_in_checkout() {
        // CORR-C-005: the checkmodule input must resolve at its selected location.
        let manifest = Path::new(env!("CARGO_MANIFEST_DIR"));
        let root = manifest.ancestors().nth(2).unwrap();
        assert!(root.join(WORKER_POLICY_SRC).is_file());
    }
}

//! CLI-surface tests for the Forgejo locales port.
//!
//! Usage, help and refusal paths run without network or a checkout; the
//! `--lock` fetch path is covered through its source pin (refused before
//! any fetch) and its lock-document validation. Live-fetch byte checks
//! would need the upstream host, so the bounded-read plus SHA-256 rule is
//! pinned by the `--native` bound test and the crate's unit tests instead.

#![cfg(unix)]

use std::env;
use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};
use std::process::{Command, Output};
use std::sync::atomic::{AtomicU64, Ordering};

static COUNTER: AtomicU64 = AtomicU64::new(0);

const NATIVE: &str = "[common]\nhome = Home %s\n[settings]\nprofile = Profile\n";
const EXTRA: &str = "[soda]\nnav_personal = Personal\n";

struct TempDir {
    path: PathBuf,
}

impl TempDir {
    fn new(tag: &str) -> TempDir {
        let id = COUNTER.fetch_add(1, Ordering::SeqCst);
        let path = env::temp_dir().join(format!(
            "soda-forgejo-locales-{tag}-{}-{id}",
            std::process::id()
        ));
        let _ = fs::remove_dir_all(&path);
        fs::create_dir_all(&path).unwrap();
        fs::set_permissions(&path, fs::Permissions::from_mode(0o700)).unwrap();
        TempDir { path }
    }

    fn file(&self, name: &str, contents: &[u8]) -> PathBuf {
        let path = self.path.join(name);
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent).unwrap();
        }
        fs::write(&path, contents).unwrap();
        path
    }
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.path);
    }
}

fn bin() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_soda-forgejo-locales"))
}

fn run(dir: &Path, args: &[&str]) -> Output {
    Command::new(bin())
        .args(args)
        .current_dir(dir)
        .output()
        .expect("spawn soda-forgejo-locales")
}

fn status_of(output: &Output) -> i32 {
    output.status.code().expect("exit code")
}

#[test]
fn native_merge_is_byte_exact_and_silent() {
    let dir = TempDir::new("merge");
    let native = dir.file("native.ini", NATIVE.as_bytes());
    let additions = dir.file("additions.ini", EXTRA.as_bytes());
    let out = dir.path.join("nested/dir/locale.ini");
    let output = run(
        &dir.path,
        &[
            "--native",
            native.to_str().unwrap(),
            "--additions",
            additions.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(
        status_of(&output),
        0,
        "stderr: {}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert!(output.stdout.is_empty());
    assert!(output.stderr.is_empty());
    assert_eq!(
        fs::read(&out).unwrap(),
        format!("{NATIVE}\n{EXTRA}").into_bytes()
    );
}

#[test]
fn native_merge_supports_equals_flags() {
    let dir = TempDir::new("equals");
    let native = dir.file("native.ini", NATIVE.as_bytes());
    let additions = dir.file("additions.ini", EXTRA.as_bytes());
    let out = dir.path.join("locale.ini");
    let output = run(
        &dir.path,
        &[
            &format!("--native={}", native.display()),
            &format!("--additions={}", additions.display()),
            &format!("--out={}", out.display()),
        ],
    );
    assert_eq!(
        status_of(&output),
        0,
        "stderr: {}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert_eq!(
        fs::read(&out).unwrap(),
        format!("{NATIVE}\n{EXTRA}").into_bytes()
    );
}

#[test]
fn default_additions_resolve_against_the_working_directory() {
    let dir = TempDir::new("defaults");
    let native = dir.file("native.ini", NATIVE.as_bytes());
    dir.file("appliance/forgejo/i18n/en-US.ini", EXTRA.as_bytes());
    let out = dir.path.join("locale.ini");
    let output = run(
        &dir.path,
        &[
            "--native",
            native.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(
        status_of(&output),
        0,
        "stderr: {}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert_eq!(
        fs::read(&out).unwrap(),
        format!("{NATIVE}\n{EXTRA}").into_bytes()
    );
}

#[test]
fn native_over_one_mib_is_refused() {
    let dir = TempDir::new("bound");
    let mut big = NATIVE.as_bytes().to_vec();
    big.extend(std::iter::repeat_n(b'x', 1024 * 1024));
    let native = dir.file("native.ini", &big);
    let additions = dir.file("additions.ini", EXTRA.as_bytes());
    let out = dir.path.join("locale.ini");
    let output = run(
        &dir.path,
        &[
            "--native",
            native.to_str().unwrap(),
            "--additions",
            additions.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(status_of(&output), 2);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("native catalog exceeds the 1 MiB bound"),
        "{stderr}"
    );
    assert!(!out.exists());
}

#[test]
fn native_or_lock_is_required() {
    let dir = TempDir::new("required");
    let output = run(&dir.path, &["--out", "locale.ini"]);
    assert_eq!(status_of(&output), 2);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("one of the arguments --native --lock is required"),
        "{stderr}"
    );
}

#[test]
fn native_and_lock_are_mutually_exclusive() {
    let dir = TempDir::new("exclusive");
    let output = run(&dir.path, &["--native", "a", "--lock", "b", "--out", "c"]);
    assert_eq!(status_of(&output), 2);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("not allowed with argument --native"),
        "{stderr}"
    );
}

#[test]
fn out_is_required() {
    let dir = TempDir::new("out");
    let output = run(&dir.path, &["--native", "a"]);
    assert_eq!(status_of(&output), 2);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("the following arguments are required: --out"),
        "{stderr}"
    );
}

#[test]
fn stray_arguments_are_rejected() {
    let dir = TempDir::new("stray");
    let output = run(&dir.path, &["--native", "a", "--out", "b", "extra"]);
    assert_eq!(status_of(&output), 2);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("unrecognized arguments: extra"), "{stderr}");
}

#[test]
fn unknown_flags_are_rejected() {
    let dir = TempDir::new("unknown");
    let output = run(&dir.path, &["--native", "a", "--out", "b", "--bogus"]);
    assert_eq!(status_of(&output), 2);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("unrecognized arguments: --bogus"),
        "{stderr}"
    );
}

#[test]
fn help_exits_zero() {
    let dir = TempDir::new("help");
    for flag in ["--help", "-h"] {
        let output = run(&dir.path, &[flag]);
        assert_eq!(status_of(&output), 0);
        let stdout = String::from_utf8_lossy(&output.stdout);
        assert!(stdout.contains("--native FILE"), "{stdout}");
        assert!(stdout.contains("--lock FILE"), "{stdout}");
    }
}

#[test]
fn lock_with_unexpected_source_is_refused_without_fetch() {
    let dir = TempDir::new("source");
    let lock = dir.file(
        "lock.json",
        br#"{"url": "https://example.com/locale.ini", "sha256": "abc"}"#,
    );
    let additions = dir.file("additions.ini", EXTRA.as_bytes());
    let out = dir.path.join("locale.ini");
    let output = run(
        &dir.path,
        &[
            "--lock",
            lock.to_str().unwrap(),
            "--additions",
            additions.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(status_of(&output), 2);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("unexpected native catalog source"),
        "{stderr}"
    );
    assert!(!out.exists());
}

#[test]
fn lock_document_problems_fail_without_fetch() {
    let dir = TempDir::new("lockdoc");
    let additions = dir.file("additions.ini", EXTRA.as_bytes());
    for (tag, body) in [
        ("broken", b"{not json".as_slice()),
        (
            "keyless",
            br#"{"url": "https://codeberg.org/forgejo/forgejo/raw/tag/v1/x"}"#.as_slice(),
        ),
    ] {
        let lock = dir.file(&format!("{tag}.json"), body);
        let out = dir.path.join(format!("{tag}.ini"));
        let output = run(
            &dir.path,
            &[
                "--lock",
                lock.to_str().unwrap(),
                "--additions",
                additions.to_str().unwrap(),
                "--out",
                out.to_str().unwrap(),
            ],
        );
        assert_eq!(
            status_of(&output),
            1,
            "{tag}: {}",
            String::from_utf8_lossy(&output.stderr)
        );
        assert!(!out.exists());
    }
    let missing = dir.path.join("missing.json");
    let out = dir.path.join("missing.ini");
    let output = run(
        &dir.path,
        &[
            "--lock",
            missing.to_str().unwrap(),
            "--additions",
            additions.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(status_of(&output), 1);
    assert!(!out.exists());
}

#[test]
fn incomplete_native_catalog_is_rejected() {
    let dir = TempDir::new("incomplete");
    let native = dir.file("native.ini", b"[soda]\nx = y\n");
    let additions = dir.file("additions.ini", EXTRA.as_bytes());
    let out = dir.path.join("locale.ini");
    let output = run(
        &dir.path,
        &[
            "--native",
            native.to_str().unwrap(),
            "--additions",
            additions.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(status_of(&output), 1);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("Expected a complete native Forgejo English catalog"),
        "{stderr}"
    );
    assert!(!out.exists());
}

#[test]
fn duplicate_addition_keys_are_rejected() {
    let dir = TempDir::new("dupkey");
    let native = dir.file("native.ini", NATIVE.as_bytes());
    let additions = dir.file("additions.ini", b"[soda]\nx = one\nx = two\n");
    let out = dir.path.join("locale.ini");
    let output = run(
        &dir.path,
        &[
            "--native",
            native.to_str().unwrap(),
            "--additions",
            additions.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(
        status_of(&output),
        1,
        "stderr: {}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert!(!out.exists());
}

#[test]
fn non_soda_addition_namespaces_are_rejected() {
    let dir = TempDir::new("namespace");
    let native = dir.file("native.ini", NATIVE.as_bytes());
    let additions = dir.file("additions.ini", b"[settings]\nprofile = Wrong\n");
    let out = dir.path.join("locale.ini");
    let output = run(
        &dir.path,
        &[
            "--native",
            native.to_str().unwrap(),
            "--additions",
            additions.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(status_of(&output), 1);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("Additions must use only the Soda namespace"),
        "{stderr}"
    );
    assert!(!out.exists());
}

#[test]
fn native_soda_collisions_are_rejected() {
    let dir = TempDir::new("collision");
    let native = dir.file(
        "native.ini",
        b"[common]\na = b\n[settings]\nc = d\n[soda]\nx = y\n",
    );
    let additions = dir.file("additions.ini", EXTRA.as_bytes());
    let out = dir.path.join("locale.ini");
    let output = run(
        &dir.path,
        &[
            "--native",
            native.to_str().unwrap(),
            "--additions",
            additions.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(status_of(&output), 1);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("Native catalog already owns the Soda namespace"),
        "{stderr}"
    );
    assert!(!out.exists());
}

#[test]
fn existing_output_is_never_overwritten() {
    let dir = TempDir::new("exclusive");
    let native = dir.file("native.ini", NATIVE.as_bytes());
    let additions = dir.file("additions.ini", EXTRA.as_bytes());
    let out = dir.file("locale.ini", b"occupied");
    let output = run(
        &dir.path,
        &[
            "--native",
            native.to_str().unwrap(),
            "--additions",
            additions.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(
        status_of(&output),
        1,
        "stderr: {}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert_eq!(fs::read(&out).unwrap(), b"occupied");
}

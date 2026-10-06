// CLI-surface tests for the asset fetchers. Every case stays offline
// (usage, help, and refusal paths that precede any network fetch), so
// nothing here touches upstream releases. Success paths are covered by
// the lib unit tests against fixture servers.

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
        let path = env::temp_dir().join(format!(
            "soda-fetchers-cli-{tag}-{}-{id}",
            std::process::id()
        ));
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

fn tea_bin() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_soda-fetch-tea"))
}

fn terminal_bin() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_soda-fetch-terminal"))
}

fn muse_bin() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_soda-fetch-muse"))
}

fn run(bin: &Path, cwd: &Path, args: &[&str]) -> (i32, String, String) {
    let output = Command::new(bin)
        .current_dir(cwd)
        .args(args)
        .output()
        .expect("spawn fetcher");
    (
        output.status.code().unwrap_or(-1),
        String::from_utf8_lossy(&output.stdout).into_owned(),
        String::from_utf8_lossy(&output.stderr).into_owned(),
    )
}

#[test]
fn tea_help_and_usage() {
    let scratch = TempDir::new("tea-cli");
    let (code, out, _) = run(&tea_bin(), &scratch.path, &["--help"]);
    assert_eq!(code, 0);
    assert_eq!(out, "usage: soda-fetch-tea --arch x86_64 [--out DIR]\n");
    for args in [
        vec![],
        vec!["--arch"],
        vec!["--out", "x"],
        vec!["--arch", "x86_64", "extra"],
        vec!["--arch", "x86_64", "--bogus", "x"],
    ] {
        let (code, _, err) = run(&tea_bin(), &scratch.path, &args);
        assert_eq!(code, 2, "{args:?}");
        assert!(err.contains("usage: soda-fetch-tea"), "{args:?}: {err}");
    }
}

#[test]
fn tea_rejects_foreign_arch_offline() {
    let scratch = TempDir::new("tea-arch");
    let (code, _, err) = run(
        &tea_bin(),
        &scratch.path,
        &["--arch", "aarch64", "--out", "output"],
    );
    assert_eq!(code, 1);
    assert_eq!(err, "Tea staging supports x86_64 only\n");
    assert!(!scratch.path.join("output").exists());
}

#[test]
fn terminal_help_usage_and_missing_lock() {
    let scratch = TempDir::new("terminal-cli");
    let (code, out, _) = run(&terminal_bin(), &scratch.path, &["--help"]);
    assert_eq!(code, 0);
    assert_eq!(out, "usage: soda-fetch-terminal --out DIR\n");
    for args in [vec![], vec!["--out"], vec!["--out", "x", "extra"]] {
        let (code, _, err) = run(&terminal_bin(), &scratch.path, &args);
        assert_eq!(code, 2, "{args:?}");
        assert!(
            err.contains("usage: soda-fetch-terminal"),
            "{args:?}: {err}"
        );
    }
    // The output directory is created before the missing lock fails the run,
    // like the script's upfront mkdir.
    let (code, _, err) = run(&terminal_bin(), &scratch.path, &["--out", "vendor"]);
    assert_eq!(code, 1);
    assert!(err.contains("cannot read terminal lock"), "{err}");
    assert!(scratch.path.join("vendor").is_dir());
}

#[test]
fn muse_help_usage_and_offline_refusals() {
    let scratch = TempDir::new("muse-cli");
    let (code, out, _) = run(&muse_bin(), &scratch.path, &["--help"]);
    assert_eq!(code, 0);
    assert_eq!(
        out,
        "usage: soda-fetch-muse [--arch ARCH] [--manifest PATH] [--out PATH]\n"
    );
    let (code, _, err) = run(&muse_bin(), &scratch.path, &["--bogus"]);
    assert_eq!(code, 1);
    assert!(
        err.starts_with("soda-fetch-muse: usage: soda-fetch-muse"),
        "{err}"
    );
    // Relative destinations are refused before the manifest is even read.
    let (code, _, err) = run(&muse_bin(), &scratch.path, &["--out", "relative"]);
    assert_eq!(code, 1);
    assert_eq!(err, "soda-fetch-muse: absolute Muse destination required\n");
    // Unknown architectures are refused before any fetch.
    let dest = scratch.path.join("muse");
    let (code, _, err) = run(
        &muse_bin(),
        &scratch.path,
        &[
            "--arch",
            "aarch64",
            "--out",
            dest.to_str().unwrap(),
            "--manifest",
            "absent.json",
        ],
    );
    assert_eq!(code, 1);
    assert_eq!(err, "soda-fetch-muse: expected x86_64\n");
    assert!(!dest.exists());
    // Trailing positionals are ignored like the Go flag owner.
    let (code, _, err) = run(&muse_bin(), &scratch.path, &["stray", "--out", "relative"]);
    assert_eq!(code, 1);
    assert_eq!(err, "soda-fetch-muse: absolute Muse destination required\n");
}

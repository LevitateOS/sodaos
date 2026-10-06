//! CLI-surface parity tests: every `--help`, usage, admission, and
//! refusal path must match the Go owners byte-for-byte (exit code, stdout,
//! stderr). Expected strings were captured from the Go binaries built at
//! the base revision. Success paths that need root, the worker identity,
//! network, or external tools stop at explicit boundary errors instead.

use std::env;
use std::fs;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::sync::atomic::{AtomicU64, Ordering};

mod soda_artifacts;
mod soda_build;
mod soda_candidate;

static COUNTER: AtomicU64 = AtomicU64::new(0);

struct TempDir {
    path: PathBuf,
}

impl TempDir {
    fn new(tag: &str) -> TempDir {
        let id = COUNTER.fetch_add(1, Ordering::SeqCst);
        let path = env::temp_dir().join(format!("soda-reltools-{tag}-{}-{id}", std::process::id()));
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

fn build_bin() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_soda-build"))
}

fn candidate_bin() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_soda-candidate"))
}

fn artifacts_bin() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_soda-artifacts"))
}

fn run<S: AsRef<std::ffi::OsStr>>(bin: &Path, cwd: &Path, args: &[S]) -> (i32, String, String) {
    let output = Command::new(bin)
        .current_dir(cwd)
        .args(args)
        .output()
        .expect("spawn binary");
    (
        output.status.code().unwrap_or(-1),
        String::from_utf8_lossy(&output.stdout).into_owned(),
        String::from_utf8_lossy(&output.stderr).into_owned(),
    )
}

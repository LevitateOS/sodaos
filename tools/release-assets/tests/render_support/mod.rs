//! Shared CLI-test fixtures for the render suites.

use std::env;
use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::sync::atomic::{AtomicU64, Ordering};

pub static COUNTER: AtomicU64 = AtomicU64::new(0);

pub struct TempDir {
    pub path: PathBuf,
}

impl TempDir {
    pub fn new(tag: &str) -> TempDir {
        let id = COUNTER.fetch_add(1, Ordering::SeqCst);
        let path = env::temp_dir().join(format!(
            "soda-stage-render-{tag}-{}-{id}",
            std::process::id()
        ));
        let _ = fs::remove_dir_all(&path);
        fs::create_dir_all(&path).unwrap();
        fs::set_permissions(&path, fs::Permissions::from_mode(0o700)).unwrap();
        TempDir { path }
    }

    pub fn sub(&self, name: &str, mode: u32) -> PathBuf {
        let path = self.path.join(name);
        fs::create_dir_all(&path).unwrap();
        fs::set_permissions(&path, fs::Permissions::from_mode(mode)).unwrap();
        path
    }

    pub fn file(&self, name: &str, contents: &str, mode: u32) -> PathBuf {
        let path = self.path.join(name);
        fs::write(&path, contents).unwrap();
        fs::set_permissions(&path, fs::Permissions::from_mode(mode)).unwrap();
        path
    }
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.path);
    }
}

pub fn fixtures() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("tests/fixtures")
}

pub fn repo_root() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR"))
        .ancestors()
        .nth(2)
        .expect("crate sits two levels under the checkout root")
        .to_path_buf()
}

pub fn copy_dir(src: &Path, dst: &Path) {
    fs::create_dir_all(dst).unwrap();
    for entry in fs::read_dir(src).unwrap() {
        let entry = entry.unwrap();
        let from = entry.path();
        let to = dst.join(entry.file_name());
        if entry.file_type().unwrap().is_dir() {
            copy_dir(&from, &to);
        } else {
            fs::copy(&from, &to).unwrap();
        }
    }
}

pub fn stage_bin() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_soda-stage"))
}

pub fn provisioning_bin() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_soda-render-provisioning"))
}

pub fn logo_bin() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_soda-render-terminal-logo"))
}

pub fn run(bin: &Path, cwd: &Path, args: &[&str]) -> (i32, String, String) {
    run_env(bin, cwd, args, &[])
}

pub fn run_env(
    bin: &Path,
    cwd: &Path,
    args: &[&str],
    env: &[(&str, &str)],
) -> (i32, String, String) {
    let mut command = Command::new(bin);
    command.current_dir(cwd).args(args);
    for (key, value) in env {
        command.env(key, value);
    }
    let output = command.output().expect("spawn stage-render binary");
    (
        output.status.code().unwrap_or(-1),
        String::from_utf8_lossy(&output.stdout).into_owned(),
        String::from_utf8_lossy(&output.stderr).into_owned(),
    )
}

pub fn mode_of(path: &Path) -> u32 {
    fs::metadata(path).unwrap().permissions().mode() & 0o777
}

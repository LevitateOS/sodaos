// Rust port of the Soda identity subscription providers (PR23).
// Mirrors internal/identity/{codex,muse}: pinned native enrollment sessions,
// credential custody checks and the shared enrollment wire shapes. The Go
// broker keeps calling the Go providers until the PR28 broker cutover; this
// crate is additive and changes no Go behavior.
pub mod codex;
pub mod muse;
pub mod sha256;
pub mod types;

use std::fmt;
use std::fs::File;
use std::io::Read;
use std::path::{Path, PathBuf};
use std::process::{Child, Command};
use std::time::Duration;

/// Failure class. Provider error text stays out of client responses (the
/// broker maps unknown failures to `unavailable`), except for the fixed
/// `Enrollment.error` strings each session records, which are preserved
/// byte-identically in the provider modules.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Kind {
    /// Identity authority denied: invalid configuration, request or file.
    Denied,
    /// Subscription requires reconnection: teardown could not confirm.
    Uncertain,
    /// Any other enrollment failure.
    Failed,
}

#[derive(Debug)]
pub struct Error {
    kind: Kind,
    message: String,
}

impl Error {
    pub fn denied(message: impl Into<String>) -> Error {
        Error {
            kind: Kind::Denied,
            message: message.into(),
        }
    }
    pub fn uncertain(message: impl Into<String>) -> Error {
        Error {
            kind: Kind::Uncertain,
            message: message.into(),
        }
    }
    pub fn failed(message: impl Into<String>) -> Error {
        Error {
            kind: Kind::Failed,
            message: message.into(),
        }
    }
    pub fn kind(&self) -> Kind {
        self.kind
    }
    pub fn is_denied(&self) -> bool {
        self.kind == Kind::Denied
    }
    pub fn is_uncertain(&self) -> bool {
        self.kind == Kind::Uncertain
    }
}

impl fmt::Display for Error {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.message)
    }
}

impl std::error::Error for Error {}

impl From<std::io::Error> for Error {
    fn from(err: std::io::Error) -> Error {
        Error::failed(err.to_string())
    }
}

/// Unique `enrollment-*` directory under a provider root, mirroring
/// `os.MkdirTemp(root, "enrollment-")`.
pub(crate) fn enrollment_tempdir(root: &Path) -> std::io::Result<PathBuf> {
    for _ in 0..100 {
        let mut suffix = [0u8; 16];
        if !random_bytes(&mut suffix) {
            return Err(std::io::Error::other("enrollment randomness unavailable"));
        }
        let dir = root.join(format!("enrollment-{}", sha256::hex(&suffix)));
        match std::fs::create_dir(&dir) {
            Ok(()) => return Ok(dir),
            Err(err) if err.kind() == std::io::ErrorKind::AlreadyExists => continue,
            Err(err) => return Err(err),
        }
    }
    Err(std::io::Error::new(
        std::io::ErrorKind::AlreadyExists,
        "enrollment directory unavailable",
    ))
}

fn random_bytes(out: &mut [u8]) -> bool {
    if let Ok(mut f) = File::open("/dev/urandom") {
        if f.read_exact(out).is_ok() {
            return true;
        }
    }
    // Fallback when urandom is unavailable: pid mixed with a nanos clock.
    // Collisions only retry the directory creation loop above.
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0);
    let mut x = nanos ^ ((std::process::id() as u128) << 64) ^ 0x9e3779b97f4a7c15;
    for chunk in out.chunks_mut(8) {
        x ^= x >> 12;
        x ^= x << 25;
        x ^= x >> 27;
        let v = x.wrapping_mul(0x2545f4914f6cdd1d).to_le_bytes();
        let n = chunk.len().min(8);
        chunk[..n].copy_from_slice(&v[..n]);
    }
    true
}

/// Enrollment roots must be private tmpfs, mirroring `privateTmpfs`.
pub(crate) fn private_tmpfs(root: &Path) -> Result<(), Error> {
    #[cfg(target_os = "linux")]
    {
        let path = root.as_os_str().as_encoded_bytes();
        let mut buf = vec![0u8; path.len() + 1];
        buf[..path.len()].copy_from_slice(path);
        let mut fs: libc::statfs64 = unsafe { std::mem::zeroed() };
        // `statfs64` matches the kernel ABI on every supported x86_64 target.
        let ok = unsafe { libc::statfs64(buf.as_ptr() as *const libc::c_char, &mut fs) == 0 };
        if !ok {
            return Err(Error::from(std::io::Error::last_os_error()));
        }
        if fs.f_type as u32 != 0x01021994 {
            return Err(Error::denied("enrollment root must be tmpfs"));
        }
        Ok(())
    }
    #[cfg(not(target_os = "linux"))]
    {
        let _ = root;
        Err(Error::denied("identity enrollment requires Linux tmpfs"))
    }
}

#[cfg(test)]
pub(crate) struct TestDir {
    path: PathBuf,
}

#[cfg(test)]
impl TestDir {
    pub(crate) fn new() -> TestDir {
        let mut suffix = [0u8; 8];
        assert!(random_bytes(&mut suffix));
        let path = std::env::temp_dir().join(format!(
            "soda-idp-test-{}-{}",
            std::process::id(),
            sha256::hex(&suffix)
        ));
        std::fs::create_dir_all(&path).unwrap();
        TestDir { path }
    }

    pub(crate) fn path(&self) -> &Path {
        &self.path
    }
}

#[cfg(test)]
impl Drop for TestDir {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.path);
    }
}

/// Run a version probe with a bounded wait, mirroring the 10s
/// `exec.CommandContext` probes. Stdout before the deadline is returned;
/// anything else (timeout included) is a version mismatch at the call site.
pub(crate) fn run_capture(
    binary: &str,
    args: &[&str],
    env: &[String],
    dir: Option<&Path>,
    timeout: Duration,
) -> Result<String, Error> {
    let mut command = Command::new(binary);
    command
        .args(args)
        .env_clear()
        .envs(env.iter().map(|e| {
            let (k, v) = e.split_once('=').unwrap_or((e.as_str(), ""));
            (k, v)
        }))
        .stdout(std::process::Stdio::piped())
        .stderr(std::process::Stdio::null());
    if let Some(dir) = dir {
        command.current_dir(dir);
    }
    let mut child: Child = command.spawn()?;
    let stdout = child.stdout.take();
    let reader = std::thread::spawn(move || {
        let mut text = Vec::new();
        if let Some(mut out) = stdout {
            let _ = out.read_to_end(&mut text);
        }
        text
    });
    let deadline = std::time::Instant::now() + timeout;
    loop {
        match child.try_wait()? {
            Some(status) => {
                let text = reader.join().unwrap_or_default();
                if status.success() {
                    return Ok(String::from_utf8_lossy(&text).into_owned());
                }
                return Err(Error::failed("version probe failed"));
            }
            None => {
                if std::time::Instant::now() >= deadline {
                    let _ = child.kill();
                    let _ = child.wait();
                    let _ = reader.join();
                    return Err(Error::failed("version probe timed out"));
                }
                std::thread::sleep(Duration::from_millis(5));
            }
        }
    }
}

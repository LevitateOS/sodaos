// Shared provider support: errors, private enrollment roots and
// captured child execution. Folded from identity-providers (A06.M).
// Rust port of the Soda identity subscription providers (PR23).
// Mirrors internal/identity/{codex,muse}: pinned native enrollment sessions,
// credential custody checks and the shared enrollment wire shapes. The Go
// broker keeps calling the Go providers until the PR28 broker cutover; this

pub mod codex;
pub mod muse;
pub mod sha256;
pub mod types;

use std::fmt;
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
    enrollment_tempdir_with(root, |out| {
        getrandom::fill(out).map_err(std::io::Error::other)
    })
}

fn enrollment_tempdir_with(
    root: &Path,
    mut fill: impl FnMut(&mut [u8]) -> std::io::Result<()>,
) -> std::io::Result<PathBuf> {
    for _ in 0..100 {
        let mut suffix = [0u8; 16];
        fill(&mut suffix)
            .map_err(|_| std::io::Error::other("enrollment randomness unavailable"))?;
        let dir = root.join(format!("enrollment-{}", sha256::hex(&suffix)));
        use std::os::unix::fs::DirBuilderExt;
        match std::fs::DirBuilder::new().mode(0o700).create(&dir) {
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
        getrandom::fill(&mut suffix).expect("test entropy unavailable");
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
mod entropy_tests {
    use super::*;

    #[test]
    fn enrollment_tempdir_propagates_partial_entropy_failure() {
        let root = std::env::temp_dir().join(format!("soda-idp-entropy-{}", std::process::id()));
        std::fs::create_dir(&root).unwrap();
        let result = enrollment_tempdir_with(&root, |out| {
            out[0] = 1;
            Err(std::io::Error::other("injected entropy failure"))
        });
        assert!(result.is_err());
        assert_eq!(std::fs::read_dir(&root).unwrap().count(), 0);
        let made = enrollment_tempdir_with(&root, |out| {
            out.fill(1);
            Ok(())
        })
        .unwrap();
        use std::os::unix::fs::PermissionsExt;
        assert_eq!(
            std::fs::metadata(&made).unwrap().permissions().mode() & 0o777,
            0o700
        );
        std::fs::remove_dir_all(made).unwrap();
        std::fs::remove_dir_all(root).unwrap();
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

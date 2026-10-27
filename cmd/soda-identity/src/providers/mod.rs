// Shared provider errors and private enrollment roots.

pub mod codex;
pub mod muse;
pub mod resolution;
pub mod sha256;
pub mod types;

use std::fmt;
use std::path::Path;

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
pub(crate) fn enrollment_tempdir(root: &Path) -> std::io::Result<tempfile::TempDir> {
    enrollment_tempdir_with(root, |out| {
        getrandom::fill(out).map_err(std::io::Error::other)
    })
}

fn enrollment_tempdir_with(
    root: &Path,
    mut fill: impl FnMut(&mut [u8]) -> std::io::Result<()>,
) -> std::io::Result<tempfile::TempDir> {
    // Keep the provider's fail-closed entropy boundary explicit; tempfile's
    // own collision-safe suffix is used only after this required source passes.
    let mut prefix = [0u8; 16];
    fill(&mut prefix).map_err(|_| std::io::Error::other("enrollment randomness unavailable"))?;
    use std::os::unix::fs::PermissionsExt;
    tempfile::Builder::new()
        .prefix(&format!("enrollment-{}-", sha256::hex(&prefix)))
        .rand_bytes(8)
        .permissions(std::fs::Permissions::from_mode(0o700))
        .tempdir_in(root)
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

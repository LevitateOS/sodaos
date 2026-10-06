//! Rust port of `internal/release/build`: shared native build primitives
//! (OCI inspection, CoreOS inputs, production steps). It never signs,
//! publishes, or installs anything, matching the Go owner.
//!
//! Every pinned success output and every sentinel error message matches the
//! Go owner byte for byte. Only unpinned failure text differs: OS error
//! details, JSON-decode diagnostics, HTTP transport errors, and Go
//! toolchain/exec internals. Pure validators are reused from
//! `soda-build-tools`; this crate owns the IO, network, and pipeline logic.
//!
//! One deliberate deviation from the Go owner:
//!
//! - File modes passed to creators are applied with an explicit chmod, so
//!   outputs are exact under any umask (identical to Go under a standard
//!   umask, where the tests pin them).

pub mod confined_files;
pub mod coreos;
pub mod coreos_iso;
pub mod coreos_stream;
pub mod elf;
pub mod files;
pub mod forgejo;
pub mod http;
pub mod json_emit;
pub mod json_go;
pub mod json_input;
pub mod oci;
pub mod oci_layout;
pub mod production;

#[cfg(test)]
pub(crate) mod test_support;

use std::io;
use std::path::Path;

/// Pipeline failure. `message` matches the Go owner's error text; the
/// structured fields carry exit/signal/cancellation detail.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Error {
    message: String,
    exit_code: Option<i32>,
    signal: Option<i32>,
    cancelled: bool,
}

impl Error {
    pub fn msg(message: impl Into<String>) -> Error {
        Error {
            message: message.into(),
            exit_code: None,
            signal: None,
            cancelled: false,
        }
    }

    pub fn message(&self) -> &str {
        &self.message
    }

    pub fn exit_code(&self) -> Option<i32> {
        self.exit_code
    }

    pub fn signal(&self) -> Option<i32> {
        self.signal
    }

    pub fn is_cancelled(&self) -> bool {
        self.cancelled
    }

    pub fn with_exit_code(mut self, code: i32) -> Error {
        self.exit_code = Some(code);
        self
    }

    pub fn with_signal(mut self, signal: i32) -> Error {
        self.signal = Some(signal);
        self
    }

    pub fn cancelled(message: impl Into<String>) -> Error {
        Error {
            message: message.into(),
            exit_code: None,
            signal: None,
            cancelled: true,
        }
    }
}

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.message)
    }
}

impl std::error::Error for Error {}

impl From<soda_build_tools::reader::Error> for Error {
    fn from(err: soda_build_tools::reader::Error) -> Error {
        Error::msg(err.0)
    }
}

/// Go-shaped OS error: `<op> <path>: <detail>`. The detail text is the Rust
/// toolchain's, not Go's.
pub fn io_error(op: &str, path: &Path, err: io::Error) -> Error {
    Error::msg(format!("{op} {}: {err}", path.display()))
}

/// Lowercase-hex SHA-256 of bytes, like Go's `hex.EncodeToString(sum)`.
pub fn sha256_hex(data: &[u8]) -> String {
    use sha2::Digest;
    let mut hasher = sha2::Sha256::new();
    hasher.update(data);
    hex_lower(&hasher.finalize())
}

/// Lowercase-hex SHA-256 of a stream, like Go's `HashFile` core.
pub fn sha256_hex_stream(reader: &mut dyn io::Read) -> Result<String, io::Error> {
    use sha2::Digest;
    let mut hasher = sha2::Sha256::new();
    io::copy(reader, &mut hasher)?;
    Ok(hex_lower(&hasher.finalize()))
}

fn hex_lower(digest: &[u8]) -> String {
    let mut out = String::with_capacity(digest.len() * 2);
    for byte in digest {
        out.push_str(&format!("{byte:02x}"));
    }
    out
}

/// `exec.LookPath` equivalent. The not-found message matches Go's
/// `exec.Error` text exactly.
pub fn look_path(name: &str) -> Result<String, Error> {
    if name.contains('/') {
        if is_executable(Path::new(name)) {
            return Ok(name.to_string());
        }
        return Err(Error::msg(format!(
            "exec: {name:?}: executable file not found in $PATH"
        )));
    }
    if let Some(paths) = std::env::var_os("PATH") {
        for dir in std::env::split_paths(&paths) {
            let candidate = dir.join(name);
            if is_executable(&candidate) {
                return Ok(candidate.to_string_lossy().into_owned());
            }
        }
    }
    Err(Error::msg(format!(
        "exec: {name:?}: executable file not found in $PATH"
    )))
}

fn is_executable(path: &Path) -> bool {
    use std::os::unix::fs::PermissionsExt;
    match std::fs::metadata(path) {
        Ok(meta) => meta.is_file() && meta.permissions().mode() & 0o111 != 0,
        Err(_) => false,
    }
}

/// Go `path.Clean` (slash paths), byte-exact: used for OCI entry names and
/// lexical path comparisons.
pub fn path_clean(path: &str) -> String {
    if path.is_empty() {
        return ".".to_string();
    }
    let rooted = path.starts_with('/');
    let mut out: Vec<&str> = Vec::new();
    let mut dotdot = 0; // leading ".." run never pops (Go lazybuf rule)
    for component in path.split('/') {
        match component {
            "" | "." => {}
            ".." => {
                if out.len() > dotdot {
                    out.pop();
                } else if !rooted {
                    out.push("..");
                    dotdot += 1;
                }
            }
            name => out.push(name),
        }
    }
    let mut joined = out.join("/");
    if rooted {
        joined.insert(0, '/');
    }
    if joined.is_empty() {
        return ".".to_string();
    }
    joined
}

/// Like Go's `filepath.ToSlash` on unix: paths already use slashes.
pub fn to_slash(path: &std::path::Path) -> String {
    path.to_string_lossy().into_owned()
}

/// Serializes tests that mutate process environment (Rust runs tests in
/// threads sharing one environment).
#[cfg(test)]
pub(crate) fn test_env_lock() -> std::sync::MutexGuard<'static, ()> {
    static LOCK: std::sync::Mutex<()> = std::sync::Mutex::new(());
    match LOCK.lock() {
        Ok(guard) => guard,
        Err(poisoned) => poisoned.into_inner(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_path_clean_vectors() {
        // Oracle: Go path.Clean outputs for OCI entry shapes.
        for (input, want) in [
            ("", "."),
            (".", "."),
            ("./", "."),
            ("blobs/", "blobs"),
            ("blobs/sha256/x", "blobs/sha256/x"),
            ("a//b", "a/b"),
            ("a/./b", "a/b"),
            ("a/b/..", "a"),
            ("../escape", "../escape"),
            ("a/../../escape", "../escape"),
            ("/abs", "/abs"),
            ("/", "/"),
            ("/../x", "/x"),
            ("./wanted", "wanted"),
        ] {
            assert_eq!(path_clean(input), want, "clean {input:?}");
        }
    }

    #[test]
    fn look_path_matches_go_errors() {
        assert!(look_path("sh").is_ok());
        assert_eq!(
            look_path("definitely-not-a-soda-tool")
                .unwrap_err()
                .message(),
            "exec: \"definitely-not-a-soda-tool\": executable file not found in $PATH"
        );
    }
}

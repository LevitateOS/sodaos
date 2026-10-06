//! Command execution with separated results, mirroring `command.go`.
//!
//! [`execute`] runs one labeled command with redacted capture: the result
//! carries execution failure and the return value carries evidence
//! retention failure, and both must be checked. [`Remote`] builds pinned
//! SSH invocations; nothing here performs trust refresh or proxying.

use crate::error::Error;

mod execute;
mod ssh;

pub use self::execute::execute;
pub use self::ssh::{decode_remote, Remote};

/// Standard input wiring for a spawned command.
#[derive(Debug, Clone)]
pub enum StdinSpec {
    /// Inherit the caller's stdin, like Go's `os.Stdin`.
    Inherit,
    /// Empty stdin.
    Null,
    /// Pumped input bytes.
    Bytes(Vec<u8>),
}

/// Command description, mirroring Go's `Command`.
#[derive(Debug, Clone)]
pub struct CommandSpec {
    /// Program name or path.
    pub name: String,
    /// Arguments.
    pub args: Vec<String>,
    /// Working directory, when set.
    pub dir: Option<String>,
    /// Standard input wiring.
    pub stdin: StdinSpec,
    /// Extra `KEY=value` environment entries.
    pub env: Vec<String>,
}

/// Separated command outcome, mirroring Go's `Result`.
#[derive(Default)]
pub struct CommandResult {
    /// Redacted captured stdout.
    pub stdout: Vec<u8>,
    /// Redacted captured stderr.
    pub stderr: Vec<u8>,
    /// Execution failure, if any.
    pub err: Option<Error>,
    /// True once the process started.
    pub started: bool,
    /// Exit code, or `-1` for signal termination.
    pub exit_code: Option<i32>,
}

/// Shell-quote argv for one SSH remote command, byte-identical to Go's
/// `Quote`: single quotes with `'"'"'` splices.
pub fn quote(args: &[String]) -> String {
    args.iter()
        .map(|s| format!("'{}'", s.replace('\'', "'\"'\"'")))
        .collect::<Vec<_>>()
        .join(" ")
}

pub(crate) fn look_path(name: &str) -> Result<(), Error> {
    if let Some(paths) = std::env::var_os("PATH") {
        for dir in std::env::split_paths(&paths) {
            let candidate = dir.join(name);
            if let Ok(meta) = std::fs::symlink_metadata(&candidate) {
                use std::os::unix::fs::MetadataExt;
                if meta.is_file() && meta.mode() & 0o111 != 0 {
                    return Ok(());
                }
            }
        }
    }
    Err(Error::msg(format!(
        "exec: {name:?}: executable file not found in $PATH"
    )))
}

#[cfg(test)]
mod tests;

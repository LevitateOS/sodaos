use std::fs;

use super::process::{capture, is_file, os_error, Captured};
use super::{current_pwd, stripped, stripped_string, Exit, FAIL_PREFIX, SSH_PORT};

/// `pid_display` mirrors `$(<pidfile)`: content minus trailing newlines,
// printed verbatim by `status` (surrounding spaces preserved).
pub(super) fn pid_display(bytes: &[u8]) -> String {
    stripped_string(bytes)
}

/// `pid_alive` mirrors `kill -0 "$(<pidfile)"`: surrounding whitespace is
/// tolerated like strtol, anything unparseable or negative counts as dead,
// and pid 0 really probes the process group like the shell does.
pub(super) fn pid_alive(bytes: &[u8]) -> bool {
    let trimmed = stripped(bytes);
    let trimmed: &[u8] = {
        let start = trimmed
            .iter()
            .position(|b| !b.is_ascii_whitespace())
            .unwrap_or(trimmed.len());
        let end = trimmed
            .iter()
            .rposition(|b| !b.is_ascii_whitespace())
            .map(|pos| pos + 1)
            .unwrap_or(0);
        if start >= end {
            return false;
        }
        &trimmed[start..end]
    };
    let text = match std::str::from_utf8(trimmed) {
        Ok(text) => text,
        Err(_) => return false,
    };
    let pid: i32 = match text.parse() {
        Ok(pid) => pid,
        Err(_) => return false,
    };
    if pid < 0 {
        return false;
    }
    unsafe { libc::kill(pid, 0) == 0 }
}

/// `vm_running` mirrors `[[ -f pidfile ]] && kill -0 "$(<pidfile)"`: the
/// regular-file check short-circuits, and parse failures count as not
/// running, silently like `2>/dev/null`. A failed read is reported like
/// the shell's own substitution error (the redirect covers only `kill`)
/// and then aborts the action under `set -e` like the script does.
pub(super) fn vm_running(pid_path: &str) -> Result<bool, Exit> {
    if !is_file(pid_path) {
        return Ok(false);
    }
    match fs::read(pid_path) {
        Ok(bytes) => Ok(pid_alive(&bytes)),
        Err(err) => {
            eprintln!("{FAIL_PREFIX}: {pid_path}: {}", os_error(&err));
            Err(Exit::Propagate(1))
        }
    }
}

pub(super) fn vm_dir() -> String {
    format!("{}/.artifacts/test-vm", current_pwd())
}

/// `ssh_args` mirrors the script's `ssh_args` array: key and known-hosts
/// travel as paths, never content.
pub(super) fn ssh_args(vm: &str) -> Vec<String> {
    vec![
        "-p".to_string(),
        SSH_PORT.to_string(),
        "-i".to_string(),
        format!("{vm}/operator"),
        "-o".to_string(),
        "IdentitiesOnly=yes".to_string(),
        "-o".to_string(),
        "StrictHostKeyChecking=yes".to_string(),
        "-o".to_string(),
        format!("UserKnownHostsFile={vm}/known_hosts"),
    ]
}

/// `uname_is` mirrors `[[ $(uname -s) == Linux ]]`: the output bytes decide,
// like the shell the child exit status is ignored.
pub(super) fn uname_is(flag: &str, want: &str) -> bool {
    match capture("uname", &[flag], false) {
        Captured::SpawnFailed => false,
        Captured::CaptureFailed => false,
        Captured::Done(out) => stripped(&out) == want.as_bytes(),
    }
}

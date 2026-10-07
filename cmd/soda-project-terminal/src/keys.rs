//! Revision-checked SSH key replacement (port of `project_keys.py`).
//!
//! Reads one JSON request (`{"login","identity","apply","revision","keys"}`)
//! from stdin, previews or atomically replaces the Soda-managed key file at
//! `/etc/ssh/authorized_keys/<login>`, and prints
//! `{"revision":"<sha256>","keys":[...]}`. Any failure writes the fixed
//! `native key operation not confirmed` line to stderr and exits 1, with no
//! request bytes, key contents, paths, or error text in diagnostics.
//!
//! Deltas from the `.py`, all confined to unobservable or absurd inputs:
//! - stdin/object decoding is UTF-8 only (`.py` `json.loads(bytes)` also
//!   sniffs UTF-16/32); non-UTF-8 is a clean failure either way.
//! - `EPIPE` on the stdout result exits 0 silently instead of the `.py`
//!   stderr line plus exit 1 (the peer is gone; nothing can observe this).
//! - `identity` must fit `i64`: larger JSON ints fail closed here, where the
//!   `.py` would fail the marker comparison instead. Same exit, same output.
//! - `True`/`False` as `identity` fail here; the `.py` `type() is int`
//!   check rejects those too (this site uses a strict type check, unlike
//!   the broker's `int()` conversions).

use std::io::Read;

use crate::state_json::StateValue;

use crate::account;
use crate::fs;
use crate::key_lines::{canonical_lines, KEY_FILE_LIMIT, STDIN_LIMIT};
use crate::key_request::{decode_key_request, state_object, KeyRequest};
use crate::pyemit;
use crate::sha;
use crate::sys;

/// `/etc/ssh/authorized_keys` through held no-follow descriptors; every
/// level must be uid/gid 0 with no setgid/group-write/other-write bits.
fn keys_directory() -> Result<std::fs::File, String> {
    let mut fd = sys::open_root().map_err(|e| e.to_string())?;
    for part in ["etc", "ssh", "authorized_keys"] {
        let child = sys::open_child_dir(&fd, part).map_err(|e| e.to_string())?;
        drop(fd);
        fd = child;
        let info = fs::fstat_all(&fd).map_err(|e| e.to_string())?;
        if info.st_uid != 0 || info.st_gid != 0 || info.st_mode & 0o2022 != 0 {
            return Err("unsafe key directory".to_string());
        }
    }
    Ok(fd)
}

/// Device/inode/mtime identity for the replace-race check
/// (`(st_dev, st_ino, st_mtime_ns)` in the `.py`).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
struct FileId {
    dev: u64,
    ino: u64,
    mtime_sec: i64,
    mtime_nsec: i64,
}

fn file_id(info: &libc::stat) -> FileId {
    FileId {
        dev: info.st_dev,
        ino: info.st_ino,
        // `st_mtime`/`st_mtime_nsec`: POSIX.1 names for the nanosecond mtime.
        mtime_sec: info.st_mtime,
        mtime_nsec: info.st_mtime_nsec,
    }
}

/// Open plus safety check plus bounded single read of one managed key file:
/// regular, uid/gid 0, `nlink == 1`, mode exactly `0o644`.
fn read_keys(dir: &std::fs::File, login: &str) -> Result<(Vec<u8>, Vec<String>, FileId), String> {
    let key = sys::open_at(dir, login, libc::O_RDONLY | libc::O_NONBLOCK, 0)
        .map_err(|e| e.to_string())?;
    let info = fs::fstat_all(&key).map_err(|e| e.to_string())?;
    if info.st_mode & libc::S_IFMT != libc::S_IFREG
        || info.st_uid != 0
        || info.st_gid != 0
        || info.st_nlink != 1
        || info.st_mode & 0o7777 != 0o644
    {
        return Err("unsafe managed key file".to_string());
    }
    let raw = fs::read_up_to(&key, KEY_FILE_LIMIT + 1).map_err(|e| e.to_string())?;
    let keys = canonical_lines(&raw)?;
    Ok((raw, keys, file_id(&info)))
}

/// Candidate temp name: `.soda-keys-` plus the 32 lowercase uuid hex digits.
pub fn candidate_name(uuid: &str) -> Option<String> {
    let hex: String = uuid
        .bytes()
        .filter(|b| *b != b'-')
        .map(|b| b as char)
        .collect();
    if hex.len() == 32
        && hex
            .bytes()
            .all(|b| b.is_ascii_hexdigit() && (b.is_ascii_digit() || b.is_ascii_lowercase()))
    {
        return Some(format!(".soda-keys-{hex}"));
    }
    None
}

/// Preview or replace the managed key file (the `.py` `update`).
pub fn update(request: &KeyRequest) -> Result<StateValue, String> {
    if unsafe { libc::geteuid() } != 0 {
        return Err("project-local root required".to_string());
    }
    let dir = keys_directory()?;
    // All writers share this stable directory lock through publication.
    if !sys::flock_exclusive_nb(&dir).map_err(|e| e.to_string())? {
        return Err("managed keys busy".to_string());
    }
    // Validation only: the marker pins login to identity before any read.
    let _ = account::account_for(&request.login, request.identity)?;
    let (raw, installed, old) = read_keys(&dir, &request.login)?;
    let revision = sha::hex_digest(&raw);
    if !request.apply {
        if request.revision.is_truthy() || !request.keys.is_empty() {
            return Err("invalid preview".to_string());
        }
        return Ok(state_object(&revision, &installed));
    }
    if request.revision.as_string().as_deref() != Some(revision.as_str()) {
        return Err("key file changed since preview".to_string());
    }
    let uuid = sys::read_uuid().map_err(|e| e.to_string())?;
    let candidate =
        candidate_name(uuid.trim()).ok_or_else(|| "unusable key temp name".to_string())?;
    // Own the name only after exclusive creation succeeds; any failure below
    // unlinks this operation's unpublished file (post-rename the name is
    // gone, so the cleanup is a harmless no-op then).
    let out = sys::open_at(
        &dir,
        &candidate,
        libc::O_WRONLY | libc::O_CREAT | libc::O_EXCL,
        0o600,
    )
    .map_err(|e| e.to_string())?;
    let result = replace_inner(&dir, &out, request, &raw, &old, &candidate);
    drop(out);
    if result.is_err() {
        let _ = fs::unlink_at(&dir, &candidate);
    }
    result
}

fn replace_inner(
    dir: &std::fs::File,
    out: &std::fs::File,
    request: &KeyRequest,
    raw: &[u8],
    old: &FileId,
    candidate: &str,
) -> Result<StateValue, String> {
    use std::io::Write as _;
    let mut sink: &std::fs::File = out;
    sink.write_all(&request.desired)
        .map_err(|_| "short key write".to_string())?;
    fs::fchmod(out, 0o644).map_err(|e| e.to_string())?;
    fs::fsync_file(out).map_err(|e| e.to_string())?;
    let (current, _, info) = read_keys(dir, &request.login)?;
    if current != raw || info != *old {
        return Err("key file changed during update".to_string());
    }
    fs::replace_at(dir, candidate, &request.login).map_err(|e| e.to_string())?;
    fs::fsync_file(dir).map_err(|e| e.to_string())?;
    let (actual, installed, _) = read_keys(dir, &request.login)?;
    if actual != request.desired {
        return Err("key result unconfirmed".to_string());
    }
    Ok(state_object(&sha::hex_digest(&actual), &installed))
}

enum KeyFail {
    Error,
    Epipe,
}

/// Read at most `STDIN_LIMIT` bytes from stdin.
fn read_stdin() -> Result<Vec<u8>, KeyFail> {
    let mut body = Vec::new();
    std::io::stdin()
        .lock()
        .take(STDIN_LIMIT as u64)
        .read_to_end(&mut body)
        .map_err(|_| KeyFail::Error)?;
    Ok(body)
}

fn write_stdout_all(bytes: &[u8]) -> Result<(), KeyFail> {
    use std::io::Write as _;
    let mut out = std::io::stdout().lock();
    match out.write_all(bytes).and_then(|()| out.flush()) {
        Ok(()) => Ok(()),
        Err(err) if err.raw_os_error() == Some(libc::EPIPE) => Err(KeyFail::Epipe),
        Err(_) => Err(KeyFail::Error),
    }
}

fn write_stderr_all(bytes: &[u8]) {
    let mut rest = bytes;
    while !rest.is_empty() {
        let wrote = unsafe { libc::write(2, rest.as_ptr() as *const libc::c_void, rest.len()) };
        if wrote < 0 {
            if std::io::Error::last_os_error().raw_os_error() == Some(libc::EINTR) {
                continue;
            }
            break;
        }
        if wrote == 0 {
            break;
        }
        rest = &rest[wrote as usize..];
    }
}

/// `key_main`: decode, update, print. EPIPE on stdout exits 0 silently.
pub fn key_main() -> i32 {
    match key_inner() {
        Ok(()) => 0,
        Err(KeyFail::Epipe) => 0,
        Err(KeyFail::Error) => {
            write_stderr_all(b"native key operation not confirmed\n");
            1
        }
    }
}

fn key_inner() -> Result<(), KeyFail> {
    let body = read_stdin()?;
    let request = decode_key_request(&body).map_err(|_| KeyFail::Error)?;
    let result = update(&request).map_err(|_| KeyFail::Error)?;
    write_stdout_all(&pyemit::line(&result))
}

#[cfg(test)]
#[path = "keys_tests.rs"]
mod keys_tests;

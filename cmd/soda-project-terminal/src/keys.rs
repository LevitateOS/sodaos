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

use soda_json::JsonValue;

use crate::account;
use crate::fs;
use crate::pyemit;
use crate::sha;
use crate::sys;

/// Stdin read cap: `sys.stdin.buffer.read(65537)` reads at most this many
/// bytes (longer input is silently truncated, never an explicit error).
pub const STDIN_LIMIT: usize = 65537;
/// Managed key file bound mirrored in `canonical_lines`.
pub const KEY_FILE_LIMIT: usize = 65536;
/// Maximum managed keys per file.
pub const KEY_COUNT_LIMIT: usize = 32;

fn is_word(byte: u8) -> bool {
    byte.is_ascii_alphanumeric() || byte == b'_'
}

/// `[\w-]+` over ASCII (`\w` is ASCII-only here: the file decoded as ASCII).
fn word_dash_plus(text: &str) -> bool {
    !text.is_empty() && text.bytes().all(|b| is_word(b) || b == b'-')
}

/// `[\w@.-]+` over ASCII.
fn sk_plus(text: &str) -> bool {
    !text.is_empty()
        && text
            .bytes()
            .all(|b| is_word(b) || b == b'@' || b == b'.' || b == b'-')
}

/// One managed key line:
/// `(ssh-[\w-]+|ecdsa-[\w-]+|sk-[\w@.-]+) [A-Za-z0-9+/=]+` fullmatch.
/// Exactly one space; the blob is charset-only (padding position unchecked,
/// like the regex).
pub fn canonical_key_ok(key: &str) -> bool {
    let space = match key.find(' ') {
        Some(at) => at,
        None => return false,
    };
    if key.as_bytes()[space + 1..].contains(&b' ') {
        return false;
    }
    let (kind, blob) = (&key[..space], &key[space + 1..]);
    let kind_ok = kind
        .strip_prefix("ssh-")
        .map(word_dash_plus)
        .or_else(|| kind.strip_prefix("ecdsa-").map(word_dash_plus))
        .or_else(|| kind.strip_prefix("sk-").map(sk_plus))
        .unwrap_or(false);
    kind_ok
        && !blob.is_empty()
        && blob
            .bytes()
            .all(|b| b.is_ascii_alphanumeric() || b == b'+' || b == b'/' || b == b'=')
}

/// `canonical_lines`: empty file → no keys; otherwise at most 32 unique
/// ASCII lines with a trailing newline, each passing [`canonical_key_ok`].
pub fn canonical_lines(raw: &[u8]) -> Result<Vec<String>, String> {
    if raw.is_empty() {
        return Ok(Vec::new());
    }
    if raw.len() > KEY_FILE_LIMIT || !raw.ends_with(b"\n") {
        return Err("not a managed key file".to_string());
    }
    if !raw.is_ascii() {
        return Err("not a managed key file".to_string());
    }
    // SAFETY: ASCII checked above.
    let text = std::str::from_utf8(raw).map_err(|_| "not a managed key file".to_string())?;
    let keys: Vec<&str> = text[..text.len() - 1].split('\n').collect();
    if keys.len() > KEY_COUNT_LIMIT {
        return Err("not a managed key file".to_string());
    }
    for (i, key) in keys.iter().enumerate() {
        if keys[..i].contains(key) {
            return Err("not a managed key file".to_string());
        }
        if !canonical_key_ok(key) {
            return Err("not a managed key file".to_string());
        }
    }
    Ok(keys.into_iter().map(|k| k.to_string()).collect())
}

/// Python truthiness over JSON values (only used for the `revision` preview
/// gate: `if data['revision'] or keys`).
pub fn json_truthy(value: &JsonValue) -> bool {
    match value {
        JsonValue::Null => false,
        JsonValue::Bool(b) => *b,
        JsonValue::Number(raw) => {
            if let Some(n) = value.as_integer() {
                return n != 0;
            }
            raw.parse::<f64>().map(|n| n != 0.0).unwrap_or(true)
        }
        JsonValue::Str(s) => !s.is_empty(),
        JsonValue::Array(items) => !items.is_empty(),
        JsonValue::Object(entries) => !entries.is_empty(),
    }
}

/// Duplicate-key rejection at EVERY object level (the `.py`
/// `object_pairs_hook=unique` fires for nested objects too).
pub fn reject_duplicates(value: &JsonValue) -> Result<(), String> {
    match value {
        JsonValue::Object(entries) => {
            for i in 0..entries.len() {
                for other in entries.iter().skip(i + 1) {
                    if other.0 == entries[i].0 {
                        return Err("duplicate field".to_string());
                    }
                }
            }
            for (_, item) in entries {
                reject_duplicates(item)?;
            }
            Ok(())
        }
        JsonValue::Array(items) => {
            for item in items {
                reject_duplicates(item)?;
            }
            Ok(())
        }
        _ => Ok(()),
    }
}

/// Validated key request: exact shape, strict `apply`/`identity` types,
/// canonical desired bytes.
#[derive(Debug, Clone, PartialEq)]
pub struct KeyRequest {
    pub login: String,
    pub identity: i64,
    pub apply: bool,
    pub revision: JsonValue,
    pub keys: Vec<String>,
    pub desired: Vec<u8>,
}

/// Pure request decode: JSON parse plus `key_operation` shape validation.
pub fn decode_key_request(body: &[u8]) -> Result<KeyRequest, String> {
    let text = std::str::from_utf8(body).map_err(|_| "invalid key operation".to_string())?;
    let data = JsonValue::parse(text).map_err(|_| "invalid key operation".to_string())?;
    reject_duplicates(&data)?;
    let entries = match &data {
        JsonValue::Object(entries) => entries,
        _ => return Err("invalid key operation".to_string()),
    };
    let fields = pyemit::shape(entries, &["login", "identity", "apply", "revision", "keys"])
        .ok_or_else(|| "invalid key operation".to_string())?;
    let get = |key: &str| {
        fields
            .iter()
            .find(|(k, _)| k == key)
            .map(|(_, v)| *v)
            .ok_or_else(|| "invalid key operation".to_string())
    };
    // Strict `type(x) is bool` / `type(x) is int`: no float/str coercion.
    let apply = get("apply")?
        .as_bool()
        .ok_or_else(|| "invalid key operation".to_string())?;
    let identity =
        pyemit::as_int(get("identity")?).ok_or_else(|| "invalid key operation".to_string())?;
    let login = get("login")?
        .as_str()
        .ok_or_else(|| "invalid key operation".to_string())?;
    let items = match get("keys")? {
        JsonValue::Array(items) => items,
        _ => return Err("invalid keys".to_string()),
    };
    let mut keys = Vec::with_capacity(items.len());
    for item in items {
        keys.push(
            item.as_str()
                .ok_or_else(|| "invalid keys".to_string())?
                .to_string(),
        );
    }
    let mut desired = keys.join("\n").into_bytes();
    if !keys.is_empty() {
        desired.push(b'\n');
    }
    if !desired.is_ascii() {
        return Err("invalid keys".to_string());
    }
    if canonical_lines(&desired)? != keys {
        return Err("invalid keys".to_string());
    }
    Ok(KeyRequest {
        login: login.to_string(),
        identity,
        apply,
        revision: get("revision")?.clone(),
        keys,
        desired,
    })
}

/// `{"revision","keys"}` result object in `.py` key order.
pub fn state_object(revision: &str, keys: &[String]) -> JsonValue {
    JsonValue::Object(vec![
        ("revision".to_string(), JsonValue::Str(revision.to_string())),
        (
            "keys".to_string(),
            JsonValue::Array(keys.iter().map(|k| JsonValue::Str(k.clone())).collect()),
        ),
    ])
}

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
pub fn update(request: &KeyRequest) -> Result<JsonValue, String> {
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
        if json_truthy(&request.revision) || !request.keys.is_empty() {
            return Err("invalid preview".to_string());
        }
        return Ok(state_object(&revision, &installed));
    }
    if request.revision.as_str() != Some(revision.as_str()) {
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
) -> Result<JsonValue, String> {
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
mod tests {
    use super::*;

    const KEY_A: &str = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqu7mMCw5R";
    const KEY_B: &str = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY=";
    const KEY_SK: &str =
        "sk-ssh-ed25519@openssh.com AAAAGnNrLXNzaC1lZDI1NTE5QG9wZW5zc2guY29tAAAAIA==";

    #[test]
    fn key_line_matrix() {
        for good in [
            KEY_A,
            KEY_B,
            KEY_SK,
            "ssh-rsa AAAAB3Nza+Z/",
            "ssh-dss AAAAB3Nza+CD/==",
            "ecdsa-sha2-nistp521 AAAAE2VjZHNhLXNoYTItbmlzdHA1MjE=",
            "sk-ecdsa-sha2-nistp256@openssh.com AAAAGnNrLWVjZHNhLXNoYTItbmlzdHAyNTZAb3BlbnNzaC5jb20=",
            "ssh--x AAA", // subtype "-x" matches [\w-]+
        ] {
            assert!(canonical_key_ok(good), "{good}");
        }
        for bad in [
            "",
            "ssh-ed25519",
            "ssh-ed25519 ",
            " ssh-ed25519 AAA",
            "ssh-ed25519  AAA",
            "ssh-ed25519 AAA BBB",
            "ssh-ed25519 AAA\n",
            "ssh AAA",   // empty subtype
            "ecdsa AAA", // missing dash-subtype
            "sk AAA",    // missing dash-subtype
            "putty AAA",
            "ssh-rsaé AAA",
            "ssh-rsa AAAé",
            "ssh-rsa AAA-",
            "ssh-rsa AAA_",
            "ssh-rsa.AAA AAA",
            "SSH-RSA AAA",
        ] {
            assert!(!canonical_key_ok(bad), "{bad:?}");
        }
        // sk- class extras: @ . - allowed, but not space or plus.
        assert!(canonical_key_ok("sk-a@b.c-d_ AAA"));
        assert!(!canonical_key_ok("sk-a+b AAA"));
        assert!(!canonical_key_ok("ssh-a+b AAA"));
    }

    #[test]
    fn canonical_lines_matrix() {
        assert_eq!(canonical_lines(b"").unwrap(), Vec::<String>::new());
        assert_eq!(
            canonical_lines(format!("{KEY_A}\n").as_bytes()).unwrap(),
            vec![KEY_A]
        );
        let two = format!("{KEY_A}\n{KEY_B}\n");
        assert_eq!(canonical_lines(two.as_bytes()).unwrap(), vec![KEY_A, KEY_B]);
        // Violations.
        assert!(canonical_lines(KEY_A.as_bytes()).is_err()); // no newline
        assert!(canonical_lines(format!("{KEY_A}\n{KEY_A}\n").as_bytes()).is_err()); // dup
        assert!(canonical_lines("not-a-key\n".as_bytes()).is_err());
        assert!(canonical_lines("ssh-rsa AAA\ntrailing".as_bytes()).is_err());
        assert!(canonical_lines("\n".as_bytes()).is_err()); // empty line
        assert!(canonical_lines("ssh-rsa AAAé\n".as_bytes()).is_err()); // non-ascii
        let big = vec![b'A'; KEY_FILE_LIMIT + 1];
        assert!(canonical_lines(&big).is_err()); // oversize
                                                 // 33 unique keys rejected, 32 accepted.
        let many: Vec<String> = (0..33)
            .map(|i| format!("ssh-k{i} QUJDREVGR0U1NTE5{i:04}"))
            .collect();
        assert!(canonical_lines(format!("{}\n", many.join("\n")).as_bytes()).is_err());
        let ok32 = format!("{}\n", many[..32].join("\n"));
        assert_eq!(canonical_lines(ok32.as_bytes()).unwrap().len(), 32);
    }

    #[test]
    fn truthiness_matrix() {
        let parse = |t: &str| JsonValue::parse(t).unwrap();
        assert!(!json_truthy(&parse("null")));
        assert!(!json_truthy(&parse("false")));
        assert!(json_truthy(&parse("true")));
        assert!(!json_truthy(&parse("0")));
        assert!(!json_truthy(&parse("0.0")));
        assert!(json_truthy(&parse("1")));
        assert!(json_truthy(&parse("-2.5")));
        assert!(json_truthy(&parse("1e3")));
        assert!(json_truthy(&parse("99999999999999999999999")));
        assert!(!json_truthy(&parse("\"\"")));
        assert!(json_truthy(&parse("\"0\"")));
        assert!(!json_truthy(&parse("[]")));
        assert!(!json_truthy(&parse("{}")));
        assert!(json_truthy(&parse("[0]")));
        assert!(json_truthy(&parse("{\"a\":0}")));
    }

    fn request(login: &str, identity: &str, apply: &str, revision: &str, keys: &str) -> Vec<u8> {
        format!("{{\"login\":{login},\"identity\":{identity},\"apply\":{apply},\"revision\":{revision},\"keys\":{keys}}}")
            .into_bytes()
    }

    #[test]
    fn decode_matrix() {
        let body = request("\"op\"", "7", "true", "\"rev\"", &format!("[\"{KEY_A}\"]"));
        let req = decode_key_request(&body).unwrap();
        assert_eq!(req.login, "op");
        assert_eq!(req.identity, 7);
        assert!(req.apply);
        assert_eq!(req.desired, format!("{KEY_A}\n").into_bytes());
        // Empty set preview shape.
        let body = request("\"op\"", "7", "false", "\"\"", "[]");
        let req = decode_key_request(&body).unwrap();
        assert!(req.desired.is_empty());
        // Violations.
        assert!(decode_key_request(b"{}").is_err()); // missing keys
        assert!(decode_key_request(b"[]").is_err()); // not an object
        assert!(decode_key_request(b"null").is_err());
        assert!(decode_key_request(b"{oops").is_err());
        assert!(decode_key_request(&request("\"op\"", "7", "1", "\"\"", "[]")).is_err()); // apply not bool
        assert!(decode_key_request(&request("\"op\"", "7.0", "true", "\"\"", "[]")).is_err()); // identity not int
        assert!(decode_key_request(&request("\"op\"", "true", "true", "\"\"", "[]")).is_err());
        assert!(decode_key_request(&request("7", "7", "true", "\"\"", "[]")).is_err()); // login not str
        assert!(decode_key_request(&request("\"op\"", "7", "true", "\"\"", "[7]")).is_err()); // keys not strs
        assert!(decode_key_request(&request("\"op\"", "7", "true", "\"\"", "\"x\"")).is_err());
        let dup = format!("[\"{KEY_A}\",\"{KEY_A}\"]");
        assert!(decode_key_request(&request("\"op\"", "7", "true", "\"\"", &dup)).is_err()); // dup keys
        assert!(
            decode_key_request(&request("\"op\"", "7", "true", "\"\"", "[\"bad line\"]")).is_err()
        );
        assert!(decode_key_request(&request("\"op\"", "7", "true", "\"\"", "[\"é\"]")).is_err()); // non-ascii
                                                                                                  // Duplicate top-level field rejected (object_pairs_hook=unique).
        assert!(decode_key_request(b"{\"login\":\"op\",\"login\":\"x\",\"identity\":7,\"apply\":true,\"revision\":\"\",\"keys\":[]}").is_err());
        // Nested duplicates rejected too.
        assert!(decode_key_request(b"{\"login\":{\"a\":1,\"a\":2},\"identity\":7,\"apply\":true,\"revision\":\"\",\"keys\":[]}").is_err());
        // Extra field rejected (exact set).
        assert!(decode_key_request(b"{\"login\":\"op\",\"identity\":7,\"apply\":true,\"revision\":\"\",\"keys\":[],\"z\":1}").is_err());
    }

    #[test]
    fn candidate_matrix() {
        assert_eq!(
            candidate_name("12345678-1234-5678-9abc-1234567890ab").as_deref(),
            Some(".soda-keys-12345678123456789abc1234567890ab")
        );
        assert!(candidate_name("12345678-1234-5678-9abc-1234567890AB").is_none()); // uppercase
        assert!(candidate_name("short").is_none());
        assert!(candidate_name("").is_none());
    }

    #[test]
    fn state_shape_exact() {
        let value = state_object("abc123", &[KEY_A.to_string()]);
        assert_eq!(
            pyemit::dumps(&value),
            format!("{{\"revision\":\"abc123\",\"keys\":[\"{KEY_A}\"]}}")
        );
        assert_eq!(
            pyemit::dumps(&state_object("", &[])),
            "{\"revision\":\"\",\"keys\":[]}"
        );
    }

    #[test]
    fn update_refuses_unprivileged() {
        if unsafe { libc::geteuid() } == 0 {
            return;
        }
        let req = decode_key_request(&request("\"op\"", "7", "false", "\"\"", "[]")).unwrap();
        assert_eq!(update(&req).unwrap_err(), "project-local root required");
    }
}

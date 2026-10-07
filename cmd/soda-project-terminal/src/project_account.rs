//! Bounded project account setup (Rust port of
//! `project-os/rootfs/usr/libexec/soda/project-account`).
//!
//! [`provision`] mirrors the Python statement for statement: root gate,
//! exact request shape, login/key validation, lock-then-observe ordering,
//! `useradd`/`usermod` argv, `O_EXCL` creation with exact modes, parent-dir
//! fsyncs, and the single collapsed failure. Internal errors reuse the
//! Python messages for traceability; the binary only ever prints the one
//! unconfirmed line (see `main`).

use serde::de::{MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use serde_json::value::RawValue;
use std::ffi::{CStr, CString};
use std::fs::File;
use std::io::{Read as _, Write as _};
use std::os::unix::ffi::OsStrExt;
use std::os::unix::fs::MetadataExt;
use std::os::unix::io::{AsRawFd, FromRawFd};
use std::path::{Path, PathBuf};

/// Validated `{'login', 'identity', 'admin', 'keys'}` request.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Request {
    pub login: String,
    pub identity: i64,
    pub admin: bool,
    pub keys: Vec<String>,
}

/// Provisioned `{'login', 'identity'}` confirmation.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Response {
    pub login: String,
    pub identity: i64,
}

/// Native command runner (`useradd`/`usermod` argv, `check=True` semantics).
pub type RunCommand = dyn Fn(&[String]) -> std::io::Result<()>;
/// Durability sync for created files and their parent directories.
pub type SyncFile = dyn Fn(&File) -> std::io::Result<()>;

/// Execution environment. [`Config::production`] is the fixed guest
/// behavior; [`Config::test_root`] redirects paths, the passwd source, and
/// the ownership root under one directory so tests never touch real state.
pub struct Config {
    pub accounts: PathBuf,
    pub keys: PathBuf,
    /// `None` resolves homes via `getpwnam`; `Some` reads `login:dir` lines.
    pub passwd_file: Option<PathBuf>,
    pub expect_uid: u32,
    pub expect_gid: u32,
    pub require_root: bool,
    /// Test-only failure injection: every file/dir sync fails.
    pub fail_sync: bool,
    pub run_command: Box<RunCommand>,
    pub sync_file: Box<SyncFile>,
}

impl Config {
    /// Fixed guest behavior: `/var/lib/soda/accounts`,
    /// `/etc/ssh/authorized_keys`, system passwd, uid/gid 0, root required.
    pub fn production() -> Self {
        Config {
            accounts: PathBuf::from("/var/lib/soda/accounts"),
            keys: PathBuf::from("/etc/ssh/authorized_keys"),
            passwd_file: None,
            expect_uid: 0,
            expect_gid: 0,
            require_root: true,
            fail_sync: false,
            run_command: Box::new(run_command_inherit),
            sync_file: Box::new(|file| file.sync_all()),
        }
    }

    /// Test behavior rooted at `root`: `accounts`/`keys` dirs, a
    /// `login:dir` passwd file, current-uid ownership, no root gate.
    pub fn test_root(root: &Path) -> Self {
        Config {
            accounts: root.join("accounts"),
            keys: root.join("keys"),
            passwd_file: Some(root.join("passwd")),
            expect_uid: unsafe { libc::geteuid() },
            expect_gid: unsafe { libc::getegid() },
            require_root: false,
            fail_sync: false,
            run_command: Box::new(run_command_inherit),
            sync_file: Box::new(|file| file.sync_all()),
        }
    }
}

/// `re.fullmatch(r'[a-z][a-z0-9_-]{0,30}', login)` (the `root` refusal lives
/// in [`parse_request`], like the Python).
pub fn valid_login(login: &str) -> bool {
    let bytes = login.as_bytes();
    if bytes.is_empty() || bytes.len() > 31 || !bytes[0].is_ascii_lowercase() {
        return false;
    }
    bytes[1..]
        .iter()
        .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || *b == b'_' || *b == b'-')
}

/// Python `str.isspace`: Unicode `White_Space` plus FS/GS/RS/US.
fn is_py_space(ch: char) -> bool {
    ch.is_whitespace() || matches!(ch, '\u{1c}' | '\u{1d}' | '\u{1e}' | '\u{1f}')
}

/// Key-type head `(ssh-|ecdsa-|sk-)SUFFIX`, ASCII-only. This is a
/// deliberate, fail-closed deviation from the retired Python's `\w`
/// (which admitted non-ASCII word characters): SSH key types are ASCII
/// by protocol, so exotic heads are unusable keys sshd would reject
/// anyway; refusing them here keeps a regex engine (and its Unicode
/// tables) out of a root guest binary. Every ASCII behavior is exact.
fn valid_key_head(head: &str) -> bool {
    let rest = head
        .strip_prefix("ssh-")
        .or_else(|| head.strip_prefix("ecdsa-"))
        .or_else(|| head.strip_prefix("sk-"));
    let Some(rest) = rest else {
        return false;
    };
    if rest.is_empty() {
        return false;
    }
    // The `sk-` alternative additionally admits `@` and `.`.
    let sk = head.starts_with("sk-");
    rest.bytes().all(|b| {
        matches!(
            b,
            b'a'..=b'z' | b'A'..=b'Z' | b'0'..=b'9' | b'_' | b'-'
        ) || (sk && matches!(b, b'@' | b'.'))
    })
}

/// `[A-Za-z0-9+/=]+\s*` tail: a non-empty ASCII base64 run, then only
/// Python whitespace.
fn valid_key_body(rest: &str) -> bool {
    let bytes = rest.as_bytes();
    let mut len = 0;
    while len < bytes.len()
        && matches!(
            bytes[len],
            b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'+' | b'/' | b'='
        )
    {
        len += 1;
    }
    if len == 0 {
        return false;
    }
    rest[len..].chars().all(is_py_space)
}

/// One canonical public key: at most 16384 *characters*, no inner newline
/// after stripping, `type SP base64 SPACES*` shape.
pub fn valid_key(key: &str) -> bool {
    if key.chars().count() > 16384 {
        return false;
    }
    if key.trim_matches(is_py_space).contains('\n') {
        return false;
    }
    let Some((head, rest)) = key.split_once(' ') else {
        return false;
    };
    valid_key_head(head) && valid_key_body(rest)
}

/// Strict request decode: exact key set (duplicates collapse last-wins,
/// like `json.loads`), typed fields, validated values.
struct RequestFields(std::collections::HashMap<String, Box<RawValue>>);

impl<'de> Deserialize<'de> for RequestFields {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct FieldsVisitor;
        impl<'de> Visitor<'de> for FieldsVisitor {
            type Value = RequestFields;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("a project account request object")
            }
            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                let mut fields = std::collections::HashMap::new();
                while let Some(name) = map.next_key::<String>()? {
                    fields.insert(name, map.next_value::<Box<RawValue>>()?);
                }
                Ok(RequestFields(fields))
            }
        }
        deserializer.deserialize_map(FieldsVisitor)
    }
}

pub fn parse_request(text: &str) -> Option<Request> {
    let fields: RequestFields = serde_json::from_str(text).ok()?;
    if fields.0.len() != 4
        || !["login", "identity", "admin", "keys"]
            .iter()
            .all(|key| fields.0.contains_key(*key))
    {
        return None;
    }
    let login: String = serde_json::from_str(fields.0.get("login")?.get()).ok()?;
    let identity: i64 = fields.0.get("identity")?.get().parse().ok()?;
    let admin: bool = serde_json::from_str(fields.0.get("admin")?.get()).ok()?;
    let keys: Vec<String> = serde_json::from_str(fields.0.get("keys")?.get()).ok()?;
    let login = login.as_str();
    if login == "root" || !valid_login(login) {
        return None;
    }
    if identity < 1 {
        return None;
    }
    if keys.len() > 32 {
        return None;
    }
    let mut out = Vec::with_capacity(keys.len());
    for text in &keys {
        if !valid_key(text) {
            return None;
        }
        out.push(text.to_string());
    }
    Some(Request {
        login: login.to_string(),
        identity,
        admin,
        keys: out,
    })
}

/// `subprocess.run(argv, check=True)` with default inherited stdio.
fn run_command_inherit(argv: &[String]) -> std::io::Result<()> {
    let (head, rest) = argv
        .split_first()
        .ok_or_else(|| std::io::Error::new(std::io::ErrorKind::InvalidInput, "empty argv"))?;
    let status = std::process::Command::new(head).args(rest).status()?;
    if status.success() {
        Ok(())
    } else {
        Err(std::io::Error::other(format!("command failed: {status}")))
    }
}

fn useradd_argv(login: &str) -> Vec<String> {
    [
        "useradd",
        "--create-home",
        "--shell",
        "/bin/bash",
        "--password",
        "!",
        "--groups",
        "soda-project",
        login,
    ]
    .iter()
    .map(ToString::to_string)
    .collect()
}

fn usermod_argv(login: &str) -> Vec<String> {
    ["usermod", "--append", "--groups", "wheel", login]
        .iter()
        .map(ToString::to_string)
        .collect()
}

fn cstring(path: &Path) -> Option<CString> {
    CString::new(path.as_os_str().as_bytes()).ok()
}

/// `open(path, flags|O_CLOEXEC|O_NOFOLLOW, mode)`: every open in the
/// Python takes `O_NOFOLLOW`, and CPython fds are non-inheritable.
fn open_nofollow(path: &Path, flags: libc::c_int, mode: libc::mode_t) -> std::io::Result<File> {
    let name = cstring(path)
        .ok_or_else(|| std::io::Error::new(std::io::ErrorKind::InvalidInput, "bad path"))?;
    let fd = unsafe {
        libc::open(
            name.as_ptr(),
            flags | libc::O_CLOEXEC | libc::O_NOFOLLOW,
            mode,
        )
    };
    if fd < 0 {
        return Err(std::io::Error::last_os_error());
    }
    Ok(unsafe { File::from_raw_fd(fd) })
}

enum ReadError {
    Missing,
    Refused,
}

/// `owned_contents`: never follow a link, never accept drift; root-owned
/// single-link regular files within `limit` bytes only.
fn owned_contents(path: &Path, limit: usize, expect_uid: u32) -> Result<Vec<u8>, ReadError> {
    let mut file = match open_nofollow(path, libc::O_RDONLY | libc::O_NONBLOCK, 0) {
        Ok(file) => file,
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => return Err(ReadError::Missing),
        Err(_) => return Err(ReadError::Refused),
    };
    let meta = file.metadata().map_err(|_| ReadError::Refused)?;
    if meta.mode() & libc::S_IFMT != libc::S_IFREG
        || meta.uid() != expect_uid
        || meta.nlink() != 1
        || meta.mode() & 0o022 != 0
    {
        return Err(ReadError::Refused);
    }
    let mut contents = Vec::new();
    let mut buf = [0u8; 8192];
    loop {
        let got = file.read(&mut buf).map_err(|_| ReadError::Refused)?;
        if got == 0 {
            break;
        }
        contents.extend_from_slice(&buf[..got]);
        if contents.len() > limit {
            break;
        }
    }
    if contents.len() > limit {
        return Err(ReadError::Refused);
    }
    Ok(contents)
}

fn sync_one(config: &Config, file: &File) -> Result<(), String> {
    if config.fail_sync {
        return Err("synthetic sync failure".to_string());
    }
    (config.sync_file)(file).map_err(|err| format!("sync failed: {err}"))
}

/// `create_file`: `O_EXCL` at `0o600`, exact mode, content fsync, then a
/// parent-directory fsync -- in that order.
fn create_file(path: &Path, contents: &[u8], mode: u32, config: &Config) -> Result<(), String> {
    let mut file = open_nofollow(path, libc::O_WRONLY | libc::O_CREAT | libc::O_EXCL, 0o600)
        .map_err(|err| format!("create failed: {err}"))?;
    file.write_all(contents)
        .map_err(|err| format!("write failed: {err}"))?;
    if unsafe { libc::fchmod(file.as_raw_fd(), mode as libc::mode_t) } != 0 {
        return Err(format!("chmod failed: {}", std::io::Error::last_os_error()));
    }
    sync_one(config, &file)?;
    let parent = path
        .parent()
        .ok_or_else(|| "no parent directory".to_string())?;
    let dir = open_nofollow(parent, libc::O_RDONLY | libc::O_DIRECTORY, 0)
        .map_err(|err| format!("open parent failed: {err}"))?;
    sync_one(config, &dir)
}

/// `flock(fd, LOCK_EX|LOCK_NB)` with `EINTR` retry; contention is an error,
/// like the Python `BlockingIOError`.
fn lock_exclusive_nb(file: &File) -> Result<(), String> {
    loop {
        let rc = unsafe { libc::flock(file.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) };
        if rc == 0 {
            return Ok(());
        }
        let err = std::io::Error::last_os_error();
        if err.raw_os_error() == Some(libc::EINTR) {
            continue;
        }
        return Err("managed key directory is locked".to_string());
    }
}

/// `os.path.lexists`: a dangling symlink counts as present.
fn lexists(path: &Path) -> bool {
    std::fs::symlink_metadata(path).is_ok()
}

fn system_home(login: &str) -> Option<String> {
    let name = CString::new(login).ok()?;
    let mut pwd: libc::passwd = unsafe { std::mem::zeroed() };
    let mut buf = vec![0u8; 16384];
    let mut result: *mut libc::passwd = std::ptr::null_mut();
    for size in [16384usize, 65536] {
        buf.resize(size, 0);
        let rc = unsafe {
            libc::getpwnam_r(
                name.as_ptr(),
                &mut pwd,
                buf.as_mut_ptr() as *mut libc::c_char,
                buf.len(),
                &mut result,
            )
        };
        if rc == 0 && !result.is_null() {
            break;
        }
        if rc != libc::ERANGE {
            return None;
        }
        result = std::ptr::null_mut();
    }
    if result.is_null() || pwd.pw_dir.is_null() {
        return None;
    }
    unsafe { CStr::from_ptr(pwd.pw_dir) }
        .to_str()
        .ok()
        .map(str::to_string)
}

/// Test passwd source: first `login:dir` line wins; a missing or
/// unreadable file means no users.
fn test_passwd_home(path: &Path, login: &str) -> Option<String> {
    let text = std::fs::read_to_string(path).ok()?;
    for line in text.lines() {
        if let Some((name, home)) = line.split_once(':') {
            if name == login {
                return Some(home.to_string());
            }
        }
    }
    None
}

fn lookup_home(login: &str, config: &Config) -> Option<String> {
    match &config.passwd_file {
        Some(path) => test_passwd_home(path, login),
        None => system_home(login),
    }
}

/// Provision (or confirm) one project-local account. The `keys` directory
/// fd -- and with it the lock -- is held for the whole body, exactly like
/// the Python `try/finally`.
pub fn provision(config: &Config, request: &Request) -> Result<Response, String> {
    if config.require_root && unsafe { libc::geteuid() } != 0 {
        return Err("project-local root required".to_string());
    }
    let keys_fd = open_nofollow(&config.keys, libc::O_RDONLY | libc::O_DIRECTORY, 0)
        .map_err(|err| format!("open managed key directory failed: {err}"))?;
    let locked = keys_fd
        .metadata()
        .map_err(|err| format!("stat managed key directory failed: {err}"))?;
    if locked.uid() != config.expect_uid
        || locked.gid() != config.expect_gid
        || locked.mode() & 0o2022 != 0
    {
        return Err("unsafe managed key directory".to_string());
    }
    lock_exclusive_nb(&keys_fd)?;
    for directory in [&config.accounts, &config.keys] {
        let info = std::fs::symlink_metadata(directory)
            .map_err(|err| format!("stat managed account directory failed: {err}"))?;
        if info.mode() & libc::S_IFMT != libc::S_IFDIR
            || info.uid() != config.expect_uid
            || info.mode() & 0o022 != 0
        {
            return Err("unsafe managed account directory".to_string());
        }
    }
    let current = std::fs::symlink_metadata(&config.keys)
        .map_err(|err| format!("stat managed key directory failed: {err}"))?;
    if (locked.dev(), locked.ino()) != (current.dev(), current.ino()) {
        return Err("managed key directory changed".to_string());
    }
    let marker = config.accounts.join(&request.login);
    let expected = request.identity.to_string();
    let home = match lookup_home(&request.login, config) {
        None => {
            if lexists(&marker) || lexists(&config.keys.join(&request.login)) {
                return Err("unassociated account files already exist".to_string());
            }
            (config.run_command)(&useradd_argv(&request.login))
                .map_err(|err| format!("useradd failed: {err}"))?;
            let home = lookup_home(&request.login, config)
                .ok_or_else(|| "native account missing after useradd".to_string())?;
            create_file(marker.as_path(), expected.as_bytes(), 0o600, config)?;
            home
        }
        Some(home) => {
            match owned_contents(marker.as_path(), 64, config.expect_uid) {
                Ok(contents) if contents == expected.as_bytes() => {}
                Ok(_) => {
                    return Err(
                        "existing native account is not associated with this identity".to_string(),
                    );
                }
                Err(_) => return Err("account marker unreadable".to_string()),
            }
            home
        }
    };
    let keyfile = config.keys.join(&request.login);
    let mut desired = Vec::new();
    for key in &request.keys {
        desired.extend_from_slice(key.trim_matches(is_py_space).as_bytes());
        desired.push(b'\n');
    }
    match owned_contents(keyfile.as_path(), 65536, config.expect_uid) {
        Err(ReadError::Missing) => {
            create_file(keyfile.as_path(), &desired, 0o644, config)?;
        }
        Ok(existing) if existing == desired => {}
        Ok(_) => {
            return Err("existing key file differs; explicit key maintenance required".to_string());
        }
        Err(ReadError::Refused) => return Err("unsafe account metadata".to_string()),
    }
    let shared = Path::new(&home).join("shared");
    if !lexists(&shared) {
        std::os::unix::fs::symlink("/srv/project/shared", &shared)
            .map_err(|err| format!("shared link failed: {err}"))?;
    }
    if request.admin {
        (config.run_command)(&usermod_argv(&request.login))
            .map_err(|err| format!("usermod failed: {err}"))?;
    }
    Ok(Response {
        login: request.login.clone(),
        identity: request.identity,
    })
}

#[cfg(test)]
mod project_account_tests;

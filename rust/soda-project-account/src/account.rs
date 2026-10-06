//! Bounded project account setup (Rust port of
//! `system/project/rootfs/usr/libexec/soda/project-account`).
//!
//! [`provision`] mirrors the Python statement for statement: root gate,
//! exact request shape, login/key validation, lock-then-observe ordering,
//! `useradd`/`usermod` argv, `O_EXCL` creation with exact modes, parent-dir
//! fsyncs, and the single collapsed failure. Internal errors reuse the
//! Python messages for traceability; the binary only ever prints the one
//! unconfirmed line (see `main`).

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
pub fn parse_request(value: &soda_json::JsonValue) -> Option<Request> {
    let entries = match value {
        soda_json::JsonValue::Object(entries) => entries,
        _ => return None,
    };
    let mut seen = std::collections::BTreeSet::new();
    for (key, _) in entries {
        seen.insert(key.as_str());
    }
    if seen.len() != 4
        || !seen.contains("login")
        || !seen.contains("identity")
        || !seen.contains("admin")
        || !seen.contains("keys")
    {
        return None;
    }
    let login = value.get("login")?.as_str()?;
    if login == "root" || !valid_login(login) {
        return None;
    }
    let identity = value.get("identity")?.as_integer()?;
    if identity < 1 || identity > i64::MAX as i128 {
        return None;
    }
    let admin = value.get("admin")?.as_bool()?;
    let keys = match value.get("keys")? {
        soda_json::JsonValue::Array(keys) => keys,
        _ => return None,
    };
    if keys.len() > 32 {
        return None;
    }
    let mut out = Vec::with_capacity(keys.len());
    for key in keys {
        let text = key.as_str()?;
        if !valid_key(text) {
            return None;
        }
        out.push(text.to_string());
    }
    Some(Request {
        login: login.to_string(),
        identity: identity as i64,
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
mod tests {
    use super::*;
    use std::os::unix::fs::PermissionsExt;
    use std::sync::atomic::{AtomicU64, Ordering};
    use std::sync::{Arc, Mutex};

    static TEST_SEQ: AtomicU64 = AtomicU64::new(0);

    /// Unique per-test root with `accounts`/`keys` dirs (tests run in
    /// parallel threads). Never touches real accounts or the real passwd db.
    fn test_root(tag: &str) -> PathBuf {
        let n = TEST_SEQ.fetch_add(1, Ordering::Relaxed);
        let dir = std::env::temp_dir().join(format!(
            "soda-project-account-{tag}-{}-{n}",
            std::process::id()
        ));
        std::fs::create_dir_all(dir.join("accounts")).expect("accounts dir");
        std::fs::create_dir_all(dir.join("keys")).expect("keys dir");
        std::fs::set_permissions(dir.join("accounts"), std::fs::Permissions::from_mode(0o700))
            .expect("accounts mode");
        std::fs::set_permissions(dir.join("keys"), std::fs::Permissions::from_mode(0o755))
            .expect("keys mode");
        dir
    }

    fn request(login: &str, identity: i64, admin: bool, keys: &[&str]) -> Request {
        Request {
            login: login.to_string(),
            identity,
            admin,
            keys: keys.iter().map(ToString::to_string).collect(),
        }
    }

    /// `Config::test_root` with an in-process useradd/usermod double:
    /// records argv, simulates home creation plus the passwd entry.
    fn direct_config(root: &Path, log: &Arc<Mutex<Vec<Vec<String>>>>) -> Config {
        let mut config = Config::test_root(root);
        let home_base = root.join("home");
        std::fs::create_dir_all(&home_base).expect("home base");
        let passwd = root.join("passwd");
        let log = Arc::clone(log);
        config.run_command = Box::new(move |argv| {
            log.lock().expect("log").push(argv.to_vec());
            if argv[0] == "useradd" {
                let login = argv.last().expect("login argv").clone();
                let home = home_base.join(&login);
                std::fs::create_dir_all(&home).expect("fake home");
                let mut file = std::fs::OpenOptions::new()
                    .create(true)
                    .append(true)
                    .open(&passwd)
                    .expect("fake passwd");
                writeln!(file, "{login}:{}", home.display()).expect("passwd line");
            } else if argv[0] != "usermod" {
                panic!("unexpected native command: {argv:?}");
            }
            Ok(())
        });
        config
    }

    #[test]
    fn login_matrix() {
        for login in ["op", "a", "a0_-b", "alice", &format!("a{}", "b".repeat(30))] {
            assert!(valid_login(login), "{login:?}");
        }
        for login in [
            "",
            "Root",
            "ALICE",
            "0abc",
            "-abc",
            "_abc",
            "ab.cd",
            "ab/cd",
            "ab cd",
            "alice\n",
            "éclair",
            &format!("a{}", "b".repeat(31)),
        ] {
            assert!(!valid_login(login), "{login:?}");
        }
        // `root` passes the character class; the refusal is a separate gate.
        assert!(valid_login("root"));
    }

    /// Key oracle baked against CPython `re.fullmatch` plus the
    /// length/strip pre-checks (see the port notes for the probe matrix).
    #[test]
    fn key_oracle() {
        let valid = [
            "ssh-ed25519 YWJj",
            "ssh-ed25519 YWJj\n",
            "ssh-ed25519 YWJj\r\n",
            "ssh-ed25519 YWJj ",
            "ssh-ed25519 YWJj  \t ",
            "ssh-rsa AAAA",
            "ecdsa-sha2-nistp256 AAAA",
            "sk-ssh-ed25519@openssh.com AAAA",
            "sk-ecdsa-sha2-nistp256@openssh.com AAAA",
            "ssh-x AAAA\u{1c}",
            "ssh-x AAAA\u{85}",
            "ssh-x AAAA\u{a0}",
            "ssh-x AAAA\u{2003}",
            "ssh-x AAAA\n\n",
            "ecdsa-x AAAA",
            "sk-x AAAA",
            "sk-foo.bar@x AAAA",
            "sk-a_b-c.d@e AAAA",
            "ssh-x +/==",
        ];
        for key in valid {
            assert!(valid_key(key), "must accept {key:?}");
        }
        // Deliberate fail-closed deviation: the retired Python admitted
        // non-ASCII `\w` heads, but SSH key types are ASCII by protocol
        // (sshd would reject these), so the port refuses them rather
        // than ship a regex engine in a root guest binary.
        for key in [
            "ssh-é AAAA",
            "ssh-½ AAAA",
            "ssh-² AAAA",
            "ssh-Ⅷ AAAA",
            "ssh-中 AAAA",
        ] {
            assert!(!valid_key(key), "must refuse exotic {key:?}");
        }
        let invalid = [
            "PRIVATE KEY",
            "ssh-ed25519",
            "ssh-ed25519 ",
            "ssh-ed25519 YW Jj",
            "ssh-ed25519 YWJj\nssh-ed25519 ZGVm",
            "SSH-ED25519 YWJj",
            "ssh_YWJj AAAA",
            "ssh-a\u{301} AAAA",
            "ssh-a\u{200d} AAAA",
            "ssh-x \u{a0}AAAA",
            " ssh-x AAAA",
            "sk- AAAA",
            "ssh- AAAA",
            "ssh-x",
            "xssh-x AAAA",
            "ssh-foo.bar AAAA",
            "ecdsa-a.b AAAA",
            "ssh-ְ AAAA",
            "",
            " ",
            "ssh-x AAAA\u{200b}",
            "ssh-xAAAA",
            "ssh-x AA\nAA",
            "ssh-x é",
        ];
        for key in invalid {
            assert!(!valid_key(key), "must refuse {key:?}");
        }
    }

    #[test]
    fn key_length_counts_characters_not_bytes() {
        // Exactly at the character limit -- and a well-formed key
        // otherwise.
        let edge = format!("ssh-{} A", "a".repeat(16384 - 6));
        assert_eq!(edge.chars().count(), 16384);
        assert!(valid_key(&edge));
        let over = format!("ssh-{} A", "a".repeat(16384 - 5));
        assert_eq!(over.chars().count(), 16385);
        assert!(!valid_key(&over));
        assert!(!valid_key(&"a".repeat(16385)));
    }

    fn parse_doc(doc: &str) -> Option<Request> {
        parse_request(&soda_json::JsonValue::parse(doc).expect("test json"))
    }

    #[test]
    fn request_shape_matrix() {
        let good = parse_doc(r#"{"login":"alice","identity":1,"admin":false,"keys":[]}"#);
        assert_eq!(
            good,
            Some(Request {
                login: "alice".to_string(),
                identity: 1,
                admin: false,
                keys: vec![],
            })
        );
        // Field order is irrelevant; the set is what matters.
        assert!(parse_doc(r#"{"keys":[],"admin":true,"identity":7,"login":"op"}"#).is_some());
        // Duplicate keys collapse last-wins, like `json.loads`.
        let dup =
            parse_doc(r#"{"login":"alice","identity":1,"admin":false,"keys":[],"login":"op"}"#)
                .expect("dup login wins");
        assert_eq!(dup.login, "op");
        for doc in [
            r#"{"login":"alice","identity":1,"admin":false}"#,
            r#"{"login":"alice","identity":1,"admin":false,"keys":[],"extra":0}"#,
            r#"{}"#,
            r#"[]"#,
            r#"null"#,
            r#"42"#,
            "\"\"",
            r#"{"login":"root","identity":1,"admin":false,"keys":[]}"#,
            r#"{"login":"Op","identity":1,"admin":false,"keys":[]}"#,
            r#"{"login":"","identity":1,"admin":false,"keys":[]}"#,
            r#"{"login":7,"identity":1,"admin":false,"keys":[]}"#,
        ] {
            assert!(parse_doc(doc).is_none(), "must refuse {doc}");
        }
    }

    #[test]
    fn request_scalar_matrix() {
        assert!(parse_doc(
            r#"{"login":"op","identity":9223372036854775807,"admin":true,"keys":[]}"#
        )
        .is_some());
        for doc in [
            r#"{"login":"op","identity":0,"admin":false,"keys":[]}"#,
            r#"{"login":"op","identity":-1,"admin":false,"keys":[]}"#,
            r#"{"login":"op","identity":9223372036854775808,"admin":false,"keys":[]}"#,
            r#"{"login":"op","identity":99999999999999999999999999,"admin":false,"keys":[]}"#,
            r#"{"login":"op","identity":1.0,"admin":false,"keys":[]}"#,
            r#"{"login":"op","identity":1e3,"admin":false,"keys":[]}"#,
            r#"{"login":"op","identity":true,"admin":false,"keys":[]}"#,
            r#"{"login":"op","identity":"1","admin":false,"keys":[]}"#,
            r#"{"login":"op","identity":null,"admin":false,"keys":[]}"#,
            r#"{"login":"op","identity":1,"admin":1,"keys":[]}"#,
            r#"{"login":"op","identity":1,"admin":"true","keys":[]}"#,
            r#"{"login":"op","identity":1,"admin":null,"keys":[]}"#,
            r#"{"login":"op","identity":1,"admin":false,"keys":null}"#,
            r#"{"login":"op","identity":1,"admin":false,"keys":"ssh-x AAAA"}"#,
            r#"{"login":"op","identity":1,"admin":false,"keys":[42]}"#,
            r#"{"login":"op","identity":1,"admin":false,"keys":[null]}"#,
            r#"{"login":"op","identity":1,"admin":false,"keys":["PRIVATE KEY"]}"#,
        ] {
            assert!(parse_doc(doc).is_none(), "must refuse {doc}");
        }
        // 32 keys fit; 33 do not.
        let many: Vec<String> = (0..33).map(|_| r#""ssh-x AAAA""#.to_string()).collect();
        assert!(parse_doc(&format!(
            "{{\"login\":\"op\",\"identity\":1,\"admin\":false,\"keys\":[{}]}}",
            many[..32].join(",")
        ))
        .is_some());
        assert!(parse_doc(&format!(
            "{{\"login\":\"op\",\"identity\":1,\"admin\":false,\"keys\":[{}]}}",
            many.join(",")
        ))
        .is_none());
    }

    fn mode_of(path: &Path) -> u32 {
        std::fs::metadata(path).expect("meta").mode() & 0o777
    }

    #[test]
    fn provisions_locked_home_marker_shared_empty_keyfile() {
        let root = test_root("provisions");
        let log = Arc::new(Mutex::new(Vec::new()));
        let config = direct_config(&root, &log);
        let response = provision(&config, &request("alice", 1, false, &[])).expect("provision");
        assert_eq!(
            response,
            Response {
                login: "alice".to_string(),
                identity: 1,
            }
        );
        assert_eq!(std::fs::read(root.join("keys/alice")).expect("keys"), b"");
        assert_eq!(
            std::fs::read(root.join("accounts/alice")).expect("marker"),
            b"1"
        );
        assert_eq!(mode_of(&root.join("accounts/alice")), 0o600);
        assert_eq!(mode_of(&root.join("keys/alice")), 0o644);
        assert_eq!(
            std::fs::read_link(root.join("home/alice/shared")).expect("shared"),
            Path::new("/srv/project/shared")
        );
        let log = log.lock().expect("log");
        assert_eq!(log.len(), 1);
        assert_eq!(
            log[0],
            [
                "useradd",
                "--create-home",
                "--shell",
                "/bin/bash",
                "--password",
                "!",
                "--groups",
                "soda-project",
                "alice"
            ]
        );
    }

    #[test]
    fn selected_keys_rerun_is_idempotent() {
        let root = test_root("idempotent");
        let log = Arc::new(Mutex::new(Vec::new()));
        let config = direct_config(&root, &log);
        let req = request("alice", 1, true, &["ssh-ed25519 YWJj\n"]);
        provision(&config, &req).expect("first");
        assert_eq!(
            std::fs::read(root.join("keys/alice")).expect("keys"),
            b"ssh-ed25519 YWJj\n"
        );
        assert_eq!(
            log.lock().expect("log").last().expect("last").as_slice(),
            ["usermod", "--append", "--groups", "wheel", "alice"]
        );
        std::fs::write(root.join("home/alice/work"), "later work").expect("work");
        let before = std::fs::metadata(root.join("keys/alice"))
            .expect("meta")
            .ino();
        provision(&config, &req).expect("second");
        assert_eq!(
            std::fs::metadata(root.join("keys/alice"))
                .expect("meta")
                .ino(),
            before
        );
        assert_eq!(
            std::fs::read_to_string(root.join("home/alice/work")).expect("work"),
            "later work"
        );
        assert_eq!(
            log.lock()
                .expect("log")
                .iter()
                .filter(|argv| argv[0] == "useradd")
                .count(),
            1
        );
    }

    #[test]
    fn join_never_applies_keys_and_preserves_drift() {
        let root = test_root("join");
        let log = Arc::new(Mutex::new(Vec::new()));
        let config = direct_config(&root, &log);
        provision(&config, &request("alice", 1, false, &["ssh-ed25519 YWJj"])).expect("first");
        assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
        assert_eq!(
            std::fs::read(root.join("keys/alice")).expect("keys"),
            b"ssh-ed25519 YWJj\n"
        );
        assert_eq!(log.lock().expect("log").len(), 1);
        // Drifted marker refuses without being touched.
        std::fs::write(root.join("accounts/alice"), "2").expect("drift");
        assert!(provision(&config, &request("alice", 1, false, &["ssh-ed25519 YWJj"])).is_err());
        assert_eq!(
            std::fs::read(root.join("accounts/alice")).expect("marker"),
            b"2"
        );
    }

    #[test]
    fn occupied_inputs_and_unassociated_users_refuse() {
        let root = test_root("occupied");
        let log = Arc::new(Mutex::new(Vec::new()));
        let config = direct_config(&root, &log);
        std::os::unix::fs::symlink(root.join("absent"), root.join("keys/alice")).expect("dangle");
        assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
        assert!(log.lock().expect("log").is_empty());
        std::fs::remove_file(root.join("keys/alice")).expect("unlink");
        // Passwd entry without a marker: unassociated, refused silently.
        std::fs::create_dir_all(root.join("home/alice")).expect("home");
        std::fs::write(
            root.join("passwd"),
            format!("alice:{}/home/alice\n", root.display()),
        )
        .expect("passwd");
        assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
        assert!(log.lock().expect("log").is_empty());
    }

    #[test]
    fn key_symlink_is_not_followed() {
        let root = test_root("symlink");
        let log = Arc::new(Mutex::new(Vec::new()));
        let config = direct_config(&root, &log);
        provision(&config, &request("alice", 1, false, &[])).expect("first");
        std::fs::write(root.join("other"), "preserve").expect("other");
        std::fs::remove_file(root.join("keys/alice")).expect("unlink");
        std::os::unix::fs::symlink(root.join("other"), root.join("keys/alice")).expect("link");
        assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
        assert_eq!(
            std::fs::read(root.join("other")).expect("other"),
            b"preserve"
        );
    }

    #[test]
    fn existing_account_binding_matrix() {
        let root = test_root("binding");
        let log = Arc::new(Mutex::new(Vec::new()));
        let config = direct_config(&root, &log);
        std::fs::create_dir_all(root.join("home/alice")).expect("home");
        std::fs::write(
            root.join("passwd"),
            format!("alice:{}/home/alice\n", root.display()),
        )
        .expect("passwd");
        // Missing marker refuses.
        assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
        // Oversized marker refuses.
        std::fs::write(root.join("accounts/alice"), vec![b'1'; 65]).expect("marker");
        assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
        // Wrong identity refuses.
        std::fs::write(root.join("accounts/alice"), "2").expect("marker");
        assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
        // Matching marker provisions the (empty) keyfile.
        std::fs::write(root.join("accounts/alice"), "1").expect("marker");
        provision(&config, &request("alice", 1, false, &[])).expect("adopt");
        assert_eq!(std::fs::read(root.join("keys/alice")).expect("keys"), b"");
        assert!(log.lock().expect("log").is_empty());
    }

    #[test]
    fn lock_contention_refuses_before_effects() {
        let root = test_root("contend");
        let log = Arc::new(Mutex::new(Vec::new()));
        let config = direct_config(&root, &log);
        let held = std::fs::File::open(root.join("keys")).expect("keys open");
        let rc = unsafe { libc::flock(held.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) };
        assert_eq!(rc, 0);
        assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
        assert!(log.lock().expect("log").is_empty());
        assert!(std::fs::read_dir(root.join("keys"))
            .expect("readdir")
            .next()
            .is_none());
        assert!(std::fs::read_dir(root.join("accounts"))
            .expect("readdir")
            .next()
            .is_none());
        drop(held);
        provision(&config, &request("alice", 1, false, &[])).expect("after release");
    }

    #[test]
    fn fsync_order_holds_lock_and_orders_durably() {
        let root = test_root("fsyncorder");
        let log = Arc::new(Mutex::new(Vec::new()));
        let mut config = direct_config(&root, &log);
        let events = Arc::new(Mutex::new(Vec::new()));
        let contents = Arc::new(Mutex::new(Vec::new()));
        let keys_path = root.join("keys");
        let accounts_path = root.join("accounts");
        let marker_path = root.join("accounts/alice");
        let keyfile_path = root.join("keys/alice");
        let events_in = Arc::clone(&events);
        let contents_in = Arc::clone(&contents);
        config.sync_file = Box::new(move |file| {
            // Every sync runs under the directory lock.
            let probe = std::fs::File::open(&keys_path).expect("probe open");
            let rc = unsafe { libc::flock(probe.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) };
            assert_ne!(rc, 0, "sync ran without the directory lock");
            assert_eq!(
                std::io::Error::last_os_error().raw_os_error(),
                Some(libc::EWOULDBLOCK)
            );
            let synced = file.metadata().expect("synced meta").clone();
            let at = |path: &Path| {
                std::fs::metadata(path)
                    .ok()
                    .filter(|meta| (meta.dev(), meta.ino()) == (synced.dev(), synced.ino()))
                    .is_some()
            };
            if synced.mode() & libc::S_IFMT == libc::S_IFREG {
                let mut seen = None;
                for (path, want, name) in [
                    (&marker_path, b"1".as_slice(), "marker"),
                    (&keyfile_path, b"ssh-ed25519 YWJj\n".as_slice(), "keyfile"),
                ] {
                    if at(path) {
                        contents_in
                            .lock()
                            .expect("contents")
                            .push(std::fs::read(path).expect("read") == want);
                        seen = Some(name);
                        break;
                    }
                }
                events_in
                    .lock()
                    .expect("events")
                    .push(seen.expect("synced an unknown file").to_string());
            } else if at(&accounts_path) {
                events_in
                    .lock()
                    .expect("events")
                    .push("markers".to_string());
            } else if at(&keys_path) {
                events_in.lock().expect("events").push("keys".to_string());
            } else {
                panic!("synced an unknown directory");
            }
            file.sync_all()
        });
        provision(&config, &request("alice", 1, true, &["ssh-ed25519 YWJj"])).expect("provision");
        assert_eq!(
            *events.lock().expect("events"),
            ["marker", "markers", "keyfile", "keys"]
        );
        assert_eq!(*contents.lock().expect("contents"), [true, true]);
        assert_eq!(log.lock().expect("log").last().expect("last")[0], "usermod");
    }

    #[test]
    fn failed_sync_releases_lock_and_keeps_partial_files() {
        let root = test_root("failsync");
        let log = Arc::new(Mutex::new(Vec::new()));
        let mut config = direct_config(&root, &log);
        config.fail_sync = true;
        assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
        assert_eq!(
            std::fs::read(root.join("accounts/alice")).expect("marker"),
            b"1"
        );
        assert!(root.join("home/alice").is_dir());
        let relock = std::fs::File::open(root.join("keys")).expect("keys open");
        let rc = unsafe { libc::flock(relock.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) };
        assert_eq!(rc, 0, "lock must not strand a failed provisioning");
    }

    #[test]
    fn unsafe_directories_refuse() {
        let root = test_root("unsafedir");
        let log = Arc::new(Mutex::new(Vec::new()));
        let config = direct_config(&root, &log);
        std::fs::set_permissions(root.join("keys"), std::fs::Permissions::from_mode(0o775))
            .expect("relax keys");
        assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
        std::fs::set_permissions(root.join("keys"), std::fs::Permissions::from_mode(0o755))
            .expect("restore keys");
        std::fs::set_permissions(
            root.join("accounts"),
            std::fs::Permissions::from_mode(0o775),
        )
        .expect("relax accounts");
        assert!(provision(&config, &request("alice", 1, false, &[])).is_err());
        assert!(log.lock().expect("log").is_empty());
    }

    #[test]
    fn unprivileged_production_gate() {
        // The fixed guest config requires root before touching anything.
        if unsafe { libc::geteuid() } != 0 {
            assert!(provision(&Config::production(), &request("alice", 1, false, &[])).is_err());
        }
    }
}

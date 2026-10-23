//! Native account resolution (port of `account_for`, `account_binding`,
//! `user_environment`, `become_user`).

use std::fs::File;
use std::os::unix::io::AsRawFd;

use crate::fs;
use crate::sys;

fn s_isreg(mode: u32) -> bool {
    rustix::fs::FileType::from_raw_mode(mode).is_file()
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Account {
    pub pw_name: String,
    pub pw_uid: u32,
    pub pw_gid: u32,
    pub pw_dir: String,
    pub pw_shell: String,
}

/// `re.fullmatch(r'[a-z][a-z0-9_-]{0,30}', login)`.
pub fn valid_login(login: &str) -> bool {
    let bytes = login.as_bytes();
    if bytes.is_empty() || bytes.len() > 31 || !bytes[0].is_ascii_lowercase() {
        return false;
    }
    bytes[1..]
        .iter()
        .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || *b == b'_' || *b == b'-')
}

/// Shell basename refusal: `false`/`nologin` (mirrors `os.path.basename`).
pub fn login_shell_ok(shell: &str) -> bool {
    let base = shell.rsplit('/').next().unwrap_or(shell);
    base != "false" && base != "nologin"
}

/// Single-read marker comparison: the 33-byte read must equal the identity.
pub fn marker_matches(content: &[u8], identity: i64) -> bool {
    content == identity.to_string().as_bytes()
}

/// Held-descriptor walk to the account directory: every level is
/// root-owned and group/other write-free, like the `.py` chain.
fn open_account_chain() -> Result<std::fs::File, String> {
    // Refuse symlink/special-file/unsafe-ancestor adoption through held
    // no-follow descriptors; every level is root-owned and group/other
    // write-free, like the `.py` chain.
    let mut fd = sys::open_root().map_err(|e| format!("open /: {e}"))?;
    for part in ["var", "lib", "soda", "accounts"] {
        let child = sys::open_child_dir(&fd, part).map_err(|e| format!("open {part}: {e}"))?;
        drop(fd);
        fd = child;
        let (uid, mode) = fs::fstat_uid_mode(&fd).map_err(|e| format!("stat {part}: {e}"))?;
        if uid != 0 || mode & 0o022 != 0 {
            return Err("unsafe account directory".to_string());
        }
    }
    Ok(fd)
}

fn read_identity_marker(dir: std::fs::File, login: &str, identity: i64) -> Result<(), String> {
    let marker: File = sys::open_at(
        &dir,
        login,
        sys::OFlags::RDONLY | sys::OFlags::NOFOLLOW | sys::OFlags::NONBLOCK,
        rustix::fs::Mode::empty(),
    )
    .map_err(|e| format!("open marker: {e}"))?;
    drop(dir);
    let info = rustix::fs::fstat(&marker).map_err(|e| format!("stat marker: {e}"))?;
    if !s_isreg(info.st_mode) || info.st_uid != 0 || info.st_mode & 0o077 != 0 {
        return Err("unsafe identity marker".to_string());
    }
    let mut content = [0u8; 33];
    let got = loop {
        let got = unsafe {
            libc::read(
                marker.as_raw_fd(),
                content.as_mut_ptr() as *mut libc::c_void,
                content.len(),
            )
        };
        if got < 0 && std::io::Error::last_os_error().raw_os_error() == Some(libc::EINTR) {
            continue;
        }
        if got < 0 {
            return Err(format!("read marker: {}", std::io::Error::last_os_error()));
        }
        break got as usize;
    };
    if !marker_matches(&content[..got], identity) {
        return Err("identity mismatch".to_string());
    }
    drop(marker);
    Ok(())
}

fn validate_account(login: &str) -> Result<Account, String> {
    let account = lookup_passwd(login)?;
    if account.pw_name != login
        || account.pw_uid == 0
        || !account.pw_dir.starts_with('/')
        || !account.pw_shell.starts_with('/')
    {
        return Err("invalid native account".to_string());
    }
    if !login_shell_ok(&account.pw_shell) {
        return Err("login disabled".to_string());
    }
    Ok(account)
}

/// Resolve and validate the native account for a login/identity pair.
pub fn account_for(login: &str, identity: i64) -> Result<Account, String> {
    if unsafe { libc::geteuid() } != 0 || !valid_login(login) || login == "root" || identity <= 0 {
        return Err("invalid identity".to_string());
    }
    let fd = open_account_chain()?;
    read_identity_marker(fd, login, identity)?;
    // Restoration deliberately leaves login disabled until explicit Join.
    // Any retained access marker refuses managed terminals and key changes.
    match std::fs::symlink_metadata(format!("/var/lib/soda/accounts/{login}.access")) {
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => {}
        _ => return Err("Project access held; explicit restore and Join required".into()),
    }
    validate_account(login)
}

fn lookup_passwd(login: &str) -> Result<Account, String> {
    let name = std::ffi::CString::new(login).map_err(|_| "invalid login".to_string())?;
    let mut pwd: libc::passwd = unsafe { std::mem::zeroed() };
    let mut buf = vec![0u8; 16384];
    let mut result: *mut libc::passwd = std::ptr::null_mut();
    // Retry once on ERANGE with a larger buffer, like a careful getpwnam.
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
            return Err("unknown native account".to_string());
        }
        result = std::ptr::null_mut();
    }
    if result.is_null() {
        return Err("unknown native account".to_string());
    }
    let field = |ptr: *const libc::c_char| -> Result<String, String> {
        if ptr.is_null() {
            return Err("invalid native account".to_string());
        }
        unsafe { std::ffi::CStr::from_ptr(ptr) }
            .to_str()
            .map(|s| s.to_string())
            .map_err(|_| "invalid native account".to_string())
    };
    Ok(Account {
        pw_name: field(pwd.pw_name)?,
        pw_uid: pwd.pw_uid,
        pw_gid: pwd.pw_gid,
        pw_dir: field(pwd.pw_dir)?,
        pw_shell: field(pwd.pw_shell)?,
    })
}

/// `[name, uid, gid, dir, shell]` binding vector.
pub fn account_binding(a: &Account) -> crate::state_json::StateValue {
    crate::state_json::StateValue::Array(vec![
        crate::state_json::StateValue::Str(a.pw_name.clone()),
        crate::state_json::StateValue::Number(a.pw_uid.to_string()),
        crate::state_json::StateValue::Number(a.pw_gid.to_string()),
        crate::state_json::StateValue::Str(a.pw_dir.clone()),
        crate::state_json::StateValue::Str(a.pw_shell.clone()),
    ])
}

/// Exact environment for the user context (order matters: systemd `--setenv`
/// argv follows it).
pub fn user_environment(a: &Account) -> Vec<(String, String)> {
    vec![
        ("HOME".to_string(), a.pw_dir.clone()),
        ("USER".to_string(), a.pw_name.clone()),
        ("LOGNAME".to_string(), a.pw_name.clone()),
        ("SHELL".to_string(), a.pw_shell.clone()),
        (
            "PATH".to_string(),
            "/usr/local/bin:/usr/bin:/bin".to_string(),
        ),
        ("TERM".to_string(), "xterm-256color".to_string()),
        ("LANG".to_string(), "C.UTF-8".to_string()),
    ]
}

/// Drop real/effective/saved credentials, enter the home directory, umask 022.
pub fn become_user(a: &Account) -> std::io::Result<()> {
    let name = std::ffi::CString::new(a.pw_name.as_str())
        .map_err(|_| std::io::Error::new(std::io::ErrorKind::InvalidInput, "bad login"))?;
    let dir = std::ffi::CString::new(a.pw_dir.as_str())
        .map_err(|_| std::io::Error::new(std::io::ErrorKind::InvalidInput, "bad home"))?;
    // SAFETY: direct libc calls; checked one by one like the `.py` sequence.
    if unsafe { libc::initgroups(name.as_ptr(), a.pw_gid) } != 0 {
        return Err(std::io::Error::last_os_error());
    }
    if unsafe { libc::setresgid(a.pw_gid, a.pw_gid, a.pw_gid) } != 0 {
        return Err(std::io::Error::last_os_error());
    }
    if unsafe { libc::setresuid(a.pw_uid, a.pw_uid, a.pw_uid) } != 0 {
        return Err(std::io::Error::last_os_error());
    }
    if unsafe { libc::chdir(dir.as_ptr()) } != 0 {
        return Err(std::io::Error::last_os_error());
    }
    unsafe {
        libc::umask(0o022);
    }
    Ok(())
}

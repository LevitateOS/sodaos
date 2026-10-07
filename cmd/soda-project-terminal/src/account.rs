//! Native account resolution (port of `account_for`, `account_binding`,
//! `user_environment`, `become_user`).

use std::fs::File;
use std::os::unix::io::AsRawFd;

use crate::fs;
use crate::sys;

fn s_isreg(mode: u32) -> bool {
    mode & libc::S_IFMT == libc::S_IFREG
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

/// Resolve and validate the native account for a login/identity pair.
pub fn account_for(login: &str, identity: i64) -> Result<Account, String> {
    if unsafe { libc::geteuid() } != 0 || !valid_login(login) || login == "root" || identity <= 0 {
        return Err("invalid identity".to_string());
    }
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
    let marker: File = sys::open_at(
        &fd,
        login,
        libc::O_RDONLY | libc::O_NOFOLLOW | libc::O_NONBLOCK,
        0,
    )
    .map_err(|e| format!("open marker: {e}"))?;
    drop(fd);
    let mut info: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstat(marker.as_raw_fd(), &mut info) } != 0 {
        return Err(format!("stat marker: {}", std::io::Error::last_os_error()));
    }
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

#[cfg(test)]
mod tests {
    use super::*;

    fn sample() -> Account {
        Account {
            pw_name: "op".to_string(),
            pw_uid: 1001,
            pw_gid: 1001,
            pw_dir: "/home/op".to_string(),
            pw_shell: "/bin/bash".to_string(),
        }
    }

    #[test]
    fn login_matrix() {
        assert!(valid_login("op"));
        assert!(valid_login("a"));
        assert!(valid_login("a0_-b"));
        assert!(valid_login(&format!("a{}", "b".repeat(30)))); // 31 chars
        assert!(!valid_login(""));
        assert!(!valid_login("Root")); // uppercase
        assert!(!valid_login("0abc"));
        assert!(!valid_login("-abc"));
        assert!(!valid_login("_abc"));
        assert!(!valid_login("ab.cd"));
        assert!(!valid_login("ab/cd"));
        assert!(!valid_login("ab cd"));
        assert!(!valid_login(&format!("a{}", "b".repeat(31)))); // 32 chars
        assert!(!valid_login("éclair"));
    }

    #[test]
    fn shell_matrix() {
        assert!(login_shell_ok("/bin/bash"));
        assert!(login_shell_ok("/usr/bin/zsh"));
        assert!(!login_shell_ok("/usr/sbin/nologin"));
        assert!(!login_shell_ok("/bin/false"));
        assert!(!login_shell_ok("false"));
        assert!(login_shell_ok("/bin/falsex"));
    }

    #[test]
    fn marker_matrix() {
        assert!(marker_matches(b"42", 42));
        assert!(marker_matches(b"1", 1));
        assert!(!marker_matches(b"42\n", 42)); // trailing newline
        assert!(!marker_matches(b"042", 42)); // no zero padding
        assert!(!marker_matches(b"43", 42));
        assert!(!marker_matches(b"", 42));
        assert!(!marker_matches(b"4", 42)); // short read
    }

    #[test]
    fn binding_and_env_exact() {
        let account = sample();
        assert_eq!(
            crate::pyemit::dumps(&account_binding(&account)),
            r#"["op",1001,1001,"/home/op","/bin/bash"]"#
        );
        let env = user_environment(&account);
        let keys: Vec<&str> = env.iter().map(|(k, _)| k.as_str()).collect();
        assert_eq!(
            keys,
            vec!["HOME", "USER", "LOGNAME", "SHELL", "PATH", "TERM", "LANG"]
        );
        assert_eq!(env[0].1, "/home/op");
        assert_eq!(env[4].1, "/usr/local/bin:/usr/bin:/bin");
        assert_eq!(env[5].1, "xterm-256color");
        assert_eq!(env[6].1, "C.UTF-8");
    }

    #[test]
    fn account_for_refuses_unprivileged() {
        // Non-root test runner: euid gate fires before any filesystem touch.
        if unsafe { libc::geteuid() } != 0 {
            assert!(account_for("op", 1).is_err());
        }
        // Shape gates fire regardless of privilege.
        assert!(account_for("root", 1).is_err());
        assert!(account_for("Op", 1).is_err());
        assert!(account_for("op", 0).is_err());
        assert!(account_for("op", -3).is_err());
    }

    #[test]
    fn become_user_fails_unprivileged() {
        // initgroups/setresuid as non-root must fail, never partially apply.
        if unsafe { libc::geteuid() } != 0 {
            assert!(become_user(&sample()).is_err());
        }
    }
}

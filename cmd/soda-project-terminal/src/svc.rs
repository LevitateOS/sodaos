//! Supervisor/socket helpers (port of `service_state`, `stop_service`,
//! `cgroup_parent`, `cgroup_empty`, `socket_identity`, `tmux_control`).
//!
//! Merge-safe: uses only the brief-pinned `sys`/`fs` signatures plus `libc`.

use std::collections::HashMap;
use std::fs::File;
use std::io;
use std::os::unix::io::{AsRawFd, FromRawFd};

use crate::account::Account;
use crate::fs;
use crate::sys;

fn s_issock(mode: u32) -> bool {
    mode & libc::S_IFMT == libc::S_IFSOCK
}

pub fn unit_name(identifier: &str) -> String {
    format!("soda-terminal-{identifier}.service")
}

pub fn service_properties() -> &'static str {
    "LoadState,ActiveState,Description,FragmentPath,DropInPaths,Type,User,KillMode,Restart,SendSIGKILL,TimeoutStopUSec,StandardInput,StandardOutput,StandardError"
}

/// `systemctl show <unit> --property=<props>` argv.
pub fn systemctl_show_argv(unit: &str) -> Vec<String> {
    vec![
        "/usr/bin/systemctl".to_string(),
        "show".to_string(),
        unit.to_string(),
        format!("--property={}", service_properties()),
    ]
}

/// `systemctl stop <unit>` argv.
pub fn systemctl_stop_argv(unit: &str) -> Vec<String> {
    vec![
        "/usr/bin/systemctl".to_string(),
        "stop".to_string(),
        unit.to_string(),
    ]
}

/// `systemctl show --value --property=InvocationID <unit>` argv.
pub fn invocation_show_argv(unit: &str) -> Vec<String> {
    vec![
        "/usr/bin/systemctl".to_string(),
        "show".to_string(),
        "--value".to_string(),
        "--property=InvocationID".to_string(),
        unit.to_string(),
    ]
}

/// `stat -f -c %T /proc/self/fd/<fd>` argv.
pub fn stat_fs_argv(fd: i32) -> Vec<String> {
    vec![
        "/usr/bin/stat".to_string(),
        "-f".to_string(),
        "-c".to_string(),
        "%T".to_string(),
        format!("/proc/self/fd/{fd}"),
    ]
}

/// `infocmp <term>` argv.
pub fn infocmp_argv(term: &str) -> Vec<String> {
    vec!["/usr/bin/infocmp".to_string(), term.to_string()]
}

/// `tmux -N -S <sock> <args...>` argv.
pub fn tmux_argv(sock: &str, args: &[String]) -> Vec<String> {
    let mut argv = vec![
        "/usr/bin/tmux".to_string(),
        "-N".to_string(),
        "-S".to_string(),
        sock.to_string(),
    ];
    argv.extend(args.iter().cloned());
    argv
}

/// Parse `KEY=value` lines (last wins, like `dict(...)`); requires the exact
/// property key set and a ≤4096-byte ASCII body.
pub fn parse_service_fields(
    stdout: &[u8],
    properties: &str,
) -> Result<HashMap<String, String>, String> {
    if stdout.len() > 4096 {
        return Err("unit response size".to_string());
    }
    if !stdout.is_ascii() {
        return Err("unit inspection unavailable".to_string());
    }
    let text =
        std::str::from_utf8(stdout).map_err(|_| "unit inspection unavailable".to_string())?;
    // Mirror `splitlines`: drop one trailing newline, then every remaining
    // segment (including blank middles) must be a `KEY=value` line.
    let body = text.strip_suffix('\n').unwrap_or(text);
    let mut fields = HashMap::new();
    if !body.is_empty() {
        for line in body.split('\n') {
            let line = line.strip_suffix('\r').unwrap_or(line);
            let (key, value) = line
                .split_once('=')
                .ok_or_else(|| "unit inspection unavailable".to_string())?;
            fields.insert(key.to_string(), value.to_string());
        }
    }
    let expected: std::collections::HashSet<&str> = properties.split(',').collect();
    let got: std::collections::HashSet<&str> = fields.keys().map(|k| k.as_str()).collect();
    if got != expected {
        return Err("unit inspection unavailable".to_string());
    }
    Ok(fields)
}

/// Loaded-unit ownership and supervision-field checks.
pub fn verify_loaded_unit(
    fields: &HashMap<String, String>,
    identifier: &str,
    account: &Account,
) -> Result<(), String> {
    let unit = unit_name(identifier);
    if fields.get("Description").map(|s| s.as_str())
        != Some(&format!("Soda terminal {identifier}")[..])
        || fields.get("FragmentPath").map(|s| s.as_str())
            != Some(&format!("/run/systemd/transient/{unit}")[..])
    {
        return Err("not the owned terminal unit".to_string());
    }
    let selected = [
        ("DropInPaths", ""),
        ("Type", "exec"),
        ("User", account.pw_name.as_str()),
        ("KillMode", "control-group"),
        ("Restart", "no"),
        ("SendSIGKILL", "yes"),
        ("TimeoutStopUSec", "3s"),
        ("StandardInput", "null"),
        ("StandardOutput", "null"),
        ("StandardError", "null"),
    ];
    for (key, want) in selected {
        if fields.get(key).map(|s| s.as_str()) != Some(want) {
            return Err("terminal supervision changed".to_string());
        }
    }
    Ok(())
}

/// Inspect the transient unit; returns `ActiveState`.
pub fn service_state(identifier: &str, account: Option<&Account>) -> Result<String, String> {
    let unit = unit_name(identifier);
    let output = sys::run_output(&systemctl_show_argv(&unit), 3)
        .map_err(|e| format!("unit inspection: {e}"))?;
    let fields = parse_service_fields(&output.stdout, service_properties())?;
    if !output.status.success() && fields.get("LoadState").map(|s| s.as_str()) != Some("not-found")
    {
        return Err("unit inspection unavailable".to_string());
    }
    if fields.get("LoadState").map(|s| s.as_str()) != Some("not-found") {
        match account {
            None => {
                // Still refuse foreign units before reporting occupation.
                if fields.get("Description").map(|s| s.as_str())
                    != Some(&format!("Soda terminal {identifier}")[..])
                    || fields.get("FragmentPath").map(|s| s.as_str())
                        != Some(&format!("/run/systemd/transient/{unit}")[..])
                {
                    return Err("not the owned terminal unit".to_string());
                }
                return Err("terminal unit occupied".to_string());
            }
            Some(held) => verify_loaded_unit(&fields, identifier, held)?,
        }
    }
    fields
        .get("ActiveState")
        .cloned()
        .ok_or_else(|| "unit inspection unavailable".to_string())
}

/// Stop the unit and confirm native state plus cgroup emptiness.
pub fn stop_service(identifier: &str, account: &Account) -> Result<(), String> {
    let state = service_state(identifier, Some(account))?;
    if state != "inactive" && state != "failed" {
        // `check=False`: a collected unit may vanish after the observation;
        // native state and cgroup emptiness below are the real confirmation.
        let _ = sys::run_output(&systemctl_stop_argv(&unit_name(identifier)), 8);
    }
    let state = service_state(identifier, Some(account))?;
    if (state != "inactive" && state != "failed") || !cgroup_empty(identifier)? {
        return Err("terminal cleanup unconfirmed".to_string());
    }
    Ok(())
}

fn cstring(value: &str) -> io::Result<std::ffi::CString> {
    std::ffi::CString::new(value)
        .map_err(|_| io::Error::new(io::ErrorKind::InvalidInput, "nul byte in argument"))
}

fn wait_status_timeout(pid: i32, timeout_secs: u64) -> Result<i32, String> {
    let deadline = sys::monotonic() + timeout_secs as f64;
    let mut status = 0;
    loop {
        let waited = unsafe { libc::waitpid(pid, &mut status, libc::WNOHANG) };
        if waited == pid {
            return Ok(status);
        }
        if waited < 0 {
            let err = io::Error::last_os_error();
            if err.raw_os_error() == Some(libc::EINTR) {
                continue;
            }
            return Err(format!("wait: {err}"));
        }
        if sys::monotonic() >= deadline {
            unsafe {
                libc::kill(pid, libc::SIGKILL);
            }
            loop {
                let waited = unsafe { libc::waitpid(pid, &mut status, 0) };
                if waited == pid || waited < 0 {
                    break;
                }
            }
            return Err("command timed out".to_string());
        }
        std::thread::sleep(std::time::Duration::from_millis(10));
    }
}

fn exit_ok(status: i32) -> bool {
    libc::WIFEXITED(status) && libc::WEXITSTATUS(status) == 0
}

/// Run `stat -f -c %T` against a held fd (the `pass_fds` equivalent: the
/// CLOEXEC bit is cleared on that fd alone in the child).
fn run_stat_fs(fd: i32) -> Result<Vec<u8>, String> {
    let argv = stat_fs_argv(fd);
    let program = cstring(&argv[0]).map_err(|e| e.to_string())?;
    let args: Vec<std::ffi::CString> = argv
        .iter()
        .map(|a| cstring(a))
        .collect::<io::Result<_>>()
        .map_err(|e| e.to_string())?;
    let mut pipe = [0; 2];
    if unsafe { libc::pipe2(pipe.as_mut_ptr(), libc::O_CLOEXEC) } != 0 {
        return Err(io::Error::last_os_error().to_string());
    }
    // Nonblocking on the read end only: the child must see a blocking pipe.
    let flags = unsafe { libc::fcntl(pipe[0], libc::F_GETFL) };
    if flags < 0 || unsafe { libc::fcntl(pipe[0], libc::F_SETFL, flags | libc::O_NONBLOCK) } != 0 {
        unsafe {
            libc::close(pipe[0]);
            libc::close(pipe[1]);
        }
        return Err(io::Error::last_os_error().to_string());
    }
    let pid = unsafe { libc::fork() };
    if pid < 0 {
        unsafe {
            libc::close(pipe[0]);
            libc::close(pipe[1]);
        }
        return Err(io::Error::last_os_error().to_string());
    }
    if pid == 0 {
        unsafe {
            let null_read = libc::open(c"/dev/null".as_ptr(), libc::O_RDONLY);
            let null_write = libc::open(c"/dev/null".as_ptr(), libc::O_WRONLY);
            if null_read < 0 || null_write < 0 {
                libc::_exit(127);
            }
            // Admit only this fd through exec; everything else is CLOEXEC.
            let flags = libc::fcntl(fd, libc::F_GETFD);
            if flags < 0 || libc::fcntl(fd, libc::F_SETFD, flags & !libc::FD_CLOEXEC) != 0 {
                libc::_exit(127);
            }
            libc::dup2(null_read, 0);
            libc::dup2(pipe[1], 1);
            libc::dup2(null_write, 2);
            let mut pointers: Vec<*const libc::c_char> = args.iter().map(|a| a.as_ptr()).collect();
            pointers.push(std::ptr::null());
            libc::execv(program.as_ptr(), pointers.as_ptr());
            libc::_exit(127);
        }
    }
    unsafe {
        libc::close(pipe[1]);
    }
    let mut stdout = Vec::new();
    let mut chunk = [0u8; 1024];
    let deadline = sys::monotonic() + 2.0;
    let mut status = 0;
    loop {
        loop {
            let got = unsafe {
                libc::read(
                    pipe[0],
                    chunk.as_mut_ptr() as *mut libc::c_void,
                    chunk.len(),
                )
            };
            if got > 0 {
                stdout.extend_from_slice(&chunk[..got as usize]);
            } else {
                break;
            }
        }
        let waited = unsafe { libc::waitpid(pid, &mut status, libc::WNOHANG) };
        if waited == pid {
            loop {
                let got = unsafe {
                    libc::read(
                        pipe[0],
                        chunk.as_mut_ptr() as *mut libc::c_void,
                        chunk.len(),
                    )
                };
                if got > 0 {
                    stdout.extend_from_slice(&chunk[..got as usize]);
                } else {
                    break;
                }
            }
            break;
        }
        if waited < 0 && io::Error::last_os_error().raw_os_error() != Some(libc::EINTR) {
            unsafe {
                libc::close(pipe[0]);
            }
            return Err("stat wait failed".to_string());
        }
        if sys::monotonic() >= deadline {
            unsafe {
                libc::kill(pid, libc::SIGKILL);
                libc::close(pipe[0]);
            }
            loop {
                let waited = unsafe { libc::waitpid(pid, &mut status, 0) };
                if waited == pid || waited < 0 {
                    break;
                }
            }
            return Err("stat timed out".to_string());
        }
        std::thread::sleep(std::time::Duration::from_millis(10));
    }
    unsafe {
        libc::close(pipe[0]);
    }
    if !exit_ok(status) {
        return Err("stat failed".to_string());
    }
    Ok(stdout)
}

fn fstatvfs_readonly(file: &File) -> Result<bool, String> {
    let mut info: libc::statvfs = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstatvfs(file.as_raw_fd(), &mut info) } != 0 {
        return Err(io::Error::last_os_error().to_string());
    }
    Ok(info.f_flag & libc::ST_RDONLY != 0)
}

/// Held no-follow descriptors for `/sys/fs/cgroup/system.slice` with
/// per-level kernel-filesystem admission checks.
pub(crate) fn cgroup_parent() -> Result<File, String> {
    let mut fd = sys::open_root().map_err(|e| format!("open /: {e}"))?;
    for part in ["sys", "fs", "cgroup", "system.slice"] {
        let child = sys::open_child_dir(&fd, part).map_err(|e| format!("open {part}: {e}"))?;
        drop(fd);
        fd = child;
        let (uid, mode) = fs::fstat_uid_mode(&fd).map_err(|e| format!("stat {part}: {e}"))?;
        let fstype = run_stat_fs(fd.as_raw_fd())?;
        if part == "sys" || part == "fs" {
            if fstype != b"sysfs\n" || !fstatvfs_readonly(&fd)? {
                return Err("expected read-only kernel sysfs".to_string());
            }
        } else if fstype != b"cgroup2fs\n" || uid != 0 || mode & 0o022 != 0 {
            return Err("unsafe delegated cgroup directory".to_string());
        }
    }
    Ok(fd)
}

/// Pure `cgroup.events` rule: `populated 0` means empty.
pub fn parse_cgroup_populated(content: &[u8]) -> Result<bool, String> {
    if !content.is_ascii() {
        return Err("cgroup events encoding".to_string());
    }
    let text = std::str::from_utf8(content).map_err(|_| "cgroup events encoding".to_string())?;
    // Mirror `splitlines`: drop one trailing newline; blank middles raise.
    let body = text.strip_suffix('\n').unwrap_or(text);
    let mut values = HashMap::new();
    if !body.is_empty() {
        for row in body.split('\n') {
            let row = row.strip_suffix('\r').unwrap_or(row);
            // `dict(row.split() ...)`: anything but a pair raises.
            let parts: Vec<&str> = row.split_whitespace().collect();
            if parts.len() != 2 {
                return Err("cgroup events shape".to_string());
            }
            values.insert(parts[0].to_string(), parts[1].to_string());
        }
    }
    Ok(values.get("populated").map(|s| s.as_str()) == Some("0"))
}

/// True when the service cgroup is absent or unpopulated.
pub fn cgroup_empty(identifier: &str) -> Result<bool, String> {
    let parent = cgroup_parent()?;
    let name = format!("soda-terminal-{identifier}.service");
    let directory = match sys::open_child_dir(&parent, &name) {
        Ok(dir) => dir,
        Err(err) if err.kind() == io::ErrorKind::NotFound => return Ok(true),
        Err(err) => return Err(format!("open cgroup: {err}")),
    };
    drop(parent);
    let fd = sys::open_at(
        &directory,
        "cgroup.events",
        libc::O_RDONLY | libc::O_NOFOLLOW,
        0,
    )
    .map_err(|e| format!("open cgroup.events: {e}"))?;
    let mut content = vec![0u8; 4096];
    let got = loop {
        let got = unsafe {
            libc::read(
                fd.as_raw_fd(),
                content.as_mut_ptr() as *mut libc::c_void,
                content.len(),
            )
        };
        if got < 0 && io::Error::last_os_error().raw_os_error() == Some(libc::EINTR) {
            continue;
        }
        if got < 0 {
            return Err(format!(
                "read cgroup.events: {}",
                io::Error::last_os_error()
            ));
        }
        break got as usize;
    };
    content.truncate(got);
    parse_cgroup_populated(&content)
}

/// Socket check outcome with retry classification for the prepare poll loop:
/// missing/refused endpoints are transient, everything else is fatal.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) enum SocketCheck {
    Retryable(String),
    Fatal(String),
}

pub(crate) fn socket_identity_kinded(
    path: &str,
    account: &Account,
    pid: i32,
) -> Result<(u64, u64), SocketCheck> {
    let target = cstring(path).map_err(|_| SocketCheck::Fatal("bad socket path".to_string()))?;
    let mut info: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::lstat(target.as_ptr(), &mut info) } != 0 {
        let err = io::Error::last_os_error();
        if err.raw_os_error() == Some(libc::ENOENT) {
            return Err(SocketCheck::Retryable("socket absent".to_string()));
        }
        return Err(SocketCheck::Fatal(format!("stat socket: {err}")));
    }
    if !s_issock(info.st_mode) || info.st_uid != account.pw_uid || info.st_mode & 0o007 != 0 {
        return Err(SocketCheck::Fatal("unsafe tmux socket".to_string()));
    }
    if path.len() >= 108 {
        return Err(SocketCheck::Fatal("socket path length".to_string()));
    }
    let fd = unsafe {
        libc::socket(
            libc::AF_UNIX,
            libc::SOCK_STREAM | libc::SOCK_CLOEXEC | libc::SOCK_NONBLOCK,
            0,
        )
    };
    if fd < 0 {
        return Err(SocketCheck::Fatal(format!(
            "socket: {}",
            io::Error::last_os_error()
        )));
    }
    let peer = unsafe { File::from_raw_fd(fd) };
    let mut addr: libc::sockaddr_un = unsafe { std::mem::zeroed() };
    addr.sun_family = libc::AF_UNIX as libc::sa_family_t;
    for (i, b) in path.bytes().enumerate() {
        addr.sun_path[i] = b as libc::c_char;
    }
    let addr_len =
        (std::mem::offset_of!(libc::sockaddr_un, sun_path) + path.len() + 1) as libc::socklen_t;
    let rc = unsafe {
        libc::connect(
            peer.as_raw_fd(),
            &addr as *const _ as *const libc::sockaddr,
            addr_len,
        )
    };
    if rc != 0 {
        let err = io::Error::last_os_error();
        if err.raw_os_error() != Some(libc::EINPROGRESS) {
            return Err(classify_connect_err(&err));
        }
        // 1s connect budget, like `peer.settimeout(1)`.
        let mut write_set: libc::fd_set = unsafe { std::mem::zeroed() };
        unsafe {
            libc::FD_ZERO(&mut write_set);
            libc::FD_SET(peer.as_raw_fd(), &mut write_set);
        }
        let mut timeout = libc::timeval {
            tv_sec: 1,
            tv_usec: 0,
        };
        let ready = unsafe {
            libc::select(
                peer.as_raw_fd() + 1,
                std::ptr::null_mut(),
                &mut write_set,
                std::ptr::null_mut(),
                &mut timeout,
            )
        };
        if ready <= 0 {
            // Timeout (or select failure): fatal, like the `.py` timeout
            // which is neither FileNotFound nor ConnectionRefused.
            return Err(SocketCheck::Fatal("socket connect timeout".to_string()));
        }
        let mut so_error = 0;
        let mut so_len = std::mem::size_of::<libc::c_int>() as libc::socklen_t;
        if unsafe {
            libc::getsockopt(
                peer.as_raw_fd(),
                libc::SOL_SOCKET,
                libc::SO_ERROR,
                &mut so_error as *mut _ as *mut libc::c_void,
                &mut so_len,
            )
        } != 0
        {
            return Err(SocketCheck::Fatal(format!(
                "getsockopt: {}",
                io::Error::last_os_error()
            )));
        }
        if so_error != 0 {
            return Err(classify_connect_err(&io::Error::from_raw_os_error(
                so_error,
            )));
        }
    }
    let mut cred: libc::ucred = unsafe { std::mem::zeroed() };
    let mut cred_len = std::mem::size_of::<libc::ucred>() as libc::socklen_t;
    if unsafe {
        libc::getsockopt(
            peer.as_raw_fd(),
            libc::SOL_SOCKET,
            libc::SO_PEERCRED,
            &mut cred as *mut _ as *mut libc::c_void,
            &mut cred_len,
        )
    } != 0
    {
        return Err(SocketCheck::Fatal(format!(
            "peercred: {}",
            io::Error::last_os_error()
        )));
    }
    if cred.pid != pid || cred.uid != account.pw_uid {
        return Err(SocketCheck::Fatal("wrong tmux server".to_string()));
    }
    Ok((info.st_dev as u64, info.st_ino as u64))
}

fn classify_connect_err(err: &io::Error) -> SocketCheck {
    match err.raw_os_error() {
        // Mirror `.py`'s `(FileNotFoundError, ConnectionRefusedError)` retry.
        Some(code) if code == libc::ENOENT || code == libc::ECONNREFUSED => {
            SocketCheck::Retryable(format!("socket unavailable: {err}"))
        }
        _ => SocketCheck::Fatal(format!("socket connect: {err}")),
    }
}

/// Verify tmux socket ownership and server identity; returns `(dev, ino)`.
pub fn socket_identity(path: &str, account: &Account, pid: i32) -> Result<(u64, u64), String> {
    socket_identity_kinded(path, account, pid).map_err(|e| match e {
        SocketCheck::Retryable(message) | SocketCheck::Fatal(message) => message,
    })
}

/// Run a tmux control command as the account (stdio to /dev/null, 2s).
pub fn tmux_control(account: &Account, sock: &str, args: &[String]) -> Result<(), String> {
    let argv = tmux_argv(sock, args);
    let program = cstring(&argv[0]).map_err(|e| e.to_string())?;
    let params: Vec<std::ffi::CString> = argv
        .iter()
        .map(|a| cstring(a))
        .collect::<io::Result<_>>()
        .map_err(|e| e.to_string())?;
    let env: Vec<(std::ffi::CString, std::ffi::CString)> =
        crate::account::user_environment(account)
            .iter()
            .map(|(k, v)| (cstring(k).unwrap(), cstring(v).unwrap()))
            .collect();
    let name = cstring(&account.pw_name).map_err(|e| e.to_string())?;
    let home = cstring(&account.pw_dir).map_err(|e| e.to_string())?;
    let (uid, gid) = (account.pw_uid, account.pw_gid);
    let pid = unsafe { libc::fork() };
    if pid < 0 {
        return Err(io::Error::last_os_error().to_string());
    }
    if pid == 0 {
        unsafe {
            // `preexec_fn` equivalent: become the user, then a fixed env.
            if libc::initgroups(name.as_ptr(), gid) != 0
                || libc::setresgid(gid, gid, gid) != 0
                || libc::setresuid(uid, uid, uid) != 0
                || libc::chdir(home.as_ptr()) != 0
            {
                libc::_exit(127);
            }
            libc::umask(0o022);
            libc::clearenv();
            for (key, value) in &env {
                if libc::setenv(key.as_ptr(), value.as_ptr(), 1) != 0 {
                    libc::_exit(127);
                }
            }
            let null = libc::open(c"/dev/null".as_ptr(), libc::O_RDWR);
            if null < 0 {
                libc::_exit(127);
            }
            libc::dup2(null, 0);
            libc::dup2(null, 1);
            libc::dup2(null, 2);
            if null > 2 {
                libc::close(null);
            }
            let mut pointers: Vec<*const libc::c_char> =
                params.iter().map(|a| a.as_ptr()).collect();
            pointers.push(std::ptr::null());
            libc::execv(program.as_ptr(), pointers.as_ptr());
            libc::_exit(127);
        }
    }
    let status = wait_status_timeout(pid, 2)?;
    if !exit_ok(status) {
        return Err("tmux control failed".to_string());
    }
    Ok(())
}

#[cfg(test)]
#[path = "svc_tests.rs"]
mod svc_tests;

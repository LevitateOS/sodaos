use std::collections::HashMap;
use std::fs::File;
use std::io;
use std::os::unix::io::AsRawFd;

use crate::fs;
use crate::sys;

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

pub(crate) fn cstring(value: &str) -> io::Result<std::ffi::CString> {
    std::ffi::CString::new(value)
        .map_err(|_| io::Error::new(io::ErrorKind::InvalidInput, "nul byte in argument"))
}

pub(crate) fn wait_status_timeout(pid: i32, timeout_secs: u64) -> Result<i32, String> {
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

pub(crate) fn exit_ok(status: i32) -> bool {
    libc::WIFEXITED(status) && libc::WEXITSTATUS(status) == 0
}

/// Run `stat -f -c %T` against a held fd (the `pass_fds` equivalent: the
/// CLOEXEC bit is cleared on that fd alone in the child).
pub(crate) fn run_stat_fs(fd: i32) -> Result<Vec<u8>, String> {
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

pub(crate) fn fstatvfs_readonly(file: &File) -> Result<bool, String> {
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

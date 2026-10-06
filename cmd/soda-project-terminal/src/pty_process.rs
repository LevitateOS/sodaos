use std::io;

use crate::account::{become_user, user_environment, Account};
use crate::pty::is_eintr;
use crate::sys;

/// `tmux attach-session` argv for the login child.
pub(crate) fn tmux_attach_argv(sock: &str) -> Vec<String> {
    vec![
        "tmux".to_string(),
        "-N".to_string(),
        "-S".to_string(),
        sock.to_string(),
        "attach-session".to_string(),
        "-E".to_string(),
        "-t".to_string(),
        "=soda".to_string(),
    ]
}

pub(crate) fn cstring(value: &str) -> io::Result<std::ffi::CString> {
    std::ffi::CString::new(value)
        .map_err(|_| io::Error::new(io::ErrorKind::InvalidInput, "nul byte"))
}

/// Join key/value pairs into owned NUL-terminated `KEY=value` entries for
/// `execve`; the caller retains the vector so envp pointers stay valid.
pub(crate) fn env_entries(
    env: &[(std::ffi::CString, std::ffi::CString)],
) -> Vec<std::ffi::CString> {
    env.iter()
        .map(|(key, value)| {
            let mut bytes = key.as_bytes().to_vec();
            bytes.push(b'=');
            bytes.extend_from_slice(value.as_bytes());
            std::ffi::CString::new(bytes).unwrap()
        })
        .collect()
}

/// `TIOCSWINSZ` from validated dimensions.
pub fn set_size(master: i32, cols: i64, rows: i64) -> io::Result<()> {
    let size = libc::winsize {
        ws_row: rows as u16,
        ws_col: cols as u16,
        ws_xpixel: 0,
        ws_ypixel: 0,
    };
    if unsafe { libc::ioctl(master, libc::TIOCSWINSZ, &size) } != 0 {
        return Err(io::Error::last_os_error());
    }
    Ok(())
}

/// True once the child has exited, without reaping (`WNOWAIT` honesty: the
/// PID cannot be recycled while it stays our zombie).
pub fn child_exited(pid: i32) -> io::Result<bool> {
    loop {
        let mut info: libc::siginfo_t = unsafe { std::mem::zeroed() };
        let rc = unsafe {
            libc::waitid(
                libc::P_PID,
                pid as libc::id_t,
                &mut info,
                libc::WEXITED | libc::WNOHANG | libc::WNOWAIT,
            )
        };
        if rc != 0 {
            let err = io::Error::last_os_error();
            if is_eintr(&err) {
                continue;
            }
            return Err(err);
        }
        return Ok(unsafe { info.si_pid() } != 0);
    }
}

/// Close the master (kernel hangup targets this terminal only), then
/// TERM/KILL with 2s waits each; `None` on honest uncertainty.
pub fn end_child(pid: i32, master: i32) -> Option<i32> {
    unsafe {
        libc::close(master);
    }
    for signal in [libc::SIGTERM, libc::SIGKILL] {
        if child_exited(pid).unwrap_or(false) {
            break;
        }
        unsafe {
            libc::kill(pid, signal);
        }
        let until = sys::monotonic() + 2.0;
        while !child_exited(pid).unwrap_or(false) && sys::monotonic() < until {
            std::thread::sleep(std::time::Duration::from_millis(20));
        }
    }
    if !child_exited(pid).unwrap_or(false) {
        return None;
    }
    let mut status = 0;
    loop {
        let waited = unsafe { libc::waitpid(pid, &mut status, 0) };
        if waited == pid {
            return Some(status);
        }
        if waited < 0 && !is_eintr(&io::Error::last_os_error()) {
            return None;
        }
    }
}

/// Fork the login child behind the ready/gate pipes; returns
/// `(pid, master, ready_read, gate_write)`.
pub fn spawn_login_pty(account: &Account, sock: &str) -> io::Result<(i32, i32, i32, i32)> {
    let mut ready = [0; 2];
    if unsafe { libc::pipe2(ready.as_mut_ptr(), libc::O_CLOEXEC) } != 0 {
        return Err(io::Error::last_os_error());
    }
    let mut gate = [0; 2];
    if unsafe { libc::pipe2(gate.as_mut_ptr(), libc::O_CLOEXEC) } != 0 {
        unsafe {
            libc::close(ready[0]);
            libc::close(ready[1]);
        }
        return Err(io::Error::last_os_error());
    }
    // Environment and argv are built before the fork.
    let env: Vec<(std::ffi::CString, std::ffi::CString)> = user_environment(account)
        .iter()
        .map(|(k, v)| (cstring(k).unwrap(), cstring(v).unwrap()))
        .collect();
    let argv: Vec<std::ffi::CString> = tmux_attach_argv(sock)
        .iter()
        .map(|a| cstring(a).unwrap())
        .collect();
    let program = cstring("/usr/bin/tmux").unwrap();
    let mut master = 0;
    let pid = unsafe {
        libc::forkpty(
            &mut master,
            std::ptr::null_mut(),
            std::ptr::null_mut(),
            std::ptr::null_mut(),
        )
    };
    if pid < 0 {
        let err = io::Error::last_os_error();
        unsafe {
            libc::close(ready[0]);
            libc::close(ready[1]);
            libc::close(gate[0]);
            libc::close(gate[1]);
        }
        return Err(err);
    }
    if pid == 0 {
        unsafe {
            libc::close(ready[0]);
            libc::close(gate[1]);
            // Wait for the parent's dimensions gate.
            let mut byte = [0u8; 1];
            let ok = loop {
                let got = libc::read(gate[0], byte.as_mut_ptr() as *mut libc::c_void, 1);
                if got < 0 && io::Error::last_os_error().raw_os_error() == Some(libc::EINTR) {
                    continue;
                }
                break got == 1 && byte[0] == b'1';
            };
            libc::close(gate[0]);
            if ok && become_user(account).is_ok() {
                // Fixed argv plus the exact user environment, like `execve`.
                let mut pointers: Vec<*const libc::c_char> =
                    argv.iter().map(|a| a.as_ptr()).collect();
                pointers.push(std::ptr::null());
                // Owned entries retained through execve; envp points at them.
                let owned_env = env_entries(&env);
                let mut env_pointers: Vec<*const libc::c_char> =
                    owned_env.iter().map(|e| e.as_ptr()).collect();
                env_pointers.push(std::ptr::null());
                libc::execve(program.as_ptr(), pointers.as_ptr(), env_pointers.as_ptr());
            }
            let failed = b"failed";
            libc::write(
                ready[1],
                failed.as_ptr() as *const libc::c_void,
                failed.len(),
            );
            libc::_exit(1);
        }
    }
    unsafe {
        libc::close(ready[1]);
        libc::close(gate[0]);
        let flags = libc::fcntl(master, libc::F_GETFD);
        if flags >= 0 {
            libc::fcntl(master, libc::F_SETFD, flags | libc::FD_CLOEXEC);
        }
    }
    Ok((pid, master, ready[0], gate[1]))
}

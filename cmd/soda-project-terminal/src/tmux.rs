use std::io;

use crate::account::Account;
use crate::cgroup::{cstring, exit_ok, wait_status_timeout};

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

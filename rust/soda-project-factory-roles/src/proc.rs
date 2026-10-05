//! Supervised role execution: one fixed entry point per call as the role
//! with bounded captured output, plus the detached double-fork supervisor
//! so completion never depends on the caller's state lock.

use crate::account::Account;
use crate::emit::obj;
use crate::error::{fail, Error};
use crate::fsx;
use crate::ops_approve::ReqFields;
use soda_json::JsonValue;
use std::ffi::CString;
use std::fs::File;
use std::io::Write;
use std::os::unix::ffi::OsStrExt;
use std::path::Path;

/// Fixed child environment, in field order (visible in the child's
/// `/proc/PID/environ`, so the order is pinned).
pub fn child_environ(
    path_value: &str,
    home: &str,
    credential_file: &str,
    role: &str,
    pid: &str,
    snapshot: &str,
) -> Vec<(String, String)> {
    let mut environ = vec![
        ("PATH".to_string(), path_value.to_string()),
        ("HOME".to_string(), home.to_string()),
        ("TMPDIR".to_string(), format!("{home}/tmp")),
        ("SODA_ROLE".to_string(), role.to_string()),
        ("SODA_PREPARATION_ID".to_string(), pid.to_string()),
        ("SODA_SNAPSHOT".to_string(), snapshot.to_string()),
    ];
    if !credential_file.is_empty() {
        environ.push((
            "SODA_CREDENTIAL_FILE".to_string(),
            credential_file.to_string(),
        ));
    }
    environ
}

/// `Path(tool['path']).parent`, pinned for odd inputs (`""` is `"."`,
/// `"/"` stays `"/"`, trailing slashes collapse first).
pub fn parent_of(path: &str) -> String {
    let trimmed = path.trim_end_matches('/');
    if trimmed.is_empty() {
        if path.is_empty() {
            return ".".to_string();
        }
        return "/".to_string();
    }
    match trimmed.rfind('/') {
        None => ".".to_string(),
        Some(0) => "/".to_string(),
        Some(index) => trimmed[..index].to_string(),
    }
}

fn exit_code(status: libc::c_int) -> i32 {
    if libc::WIFEXITED(status) {
        libc::WEXITSTATUS(status)
    } else if libc::WIFSIGNALED(status) {
        -libc::WTERMSIG(status)
    } else {
        127
    }
}

fn wait_child(pid: libc::pid_t) -> libc::c_int {
    let mut status = 0;
    loop {
        let rc = unsafe { libc::waitpid(pid, &mut status, 0) };
        if rc < 0 {
            let errno = std::io::Error::last_os_error().raw_os_error();
            if errno == Some(libc::EINTR) {
                continue;
            }
            return -1;
        }
        return status;
    }
}

fn to_cstring(bytes: &[u8]) -> Option<CString> {
    CString::new(bytes).ok()
}

/// Run one fixed entry point as the role, capturing bounded output. The
/// supervisor stays privileged so role code cannot forge logs or
/// completion. Anything failing in the child exits it with 127.
pub fn run_as_role(
    argv: &[String],
    environ: &[(String, String)],
    workdir: &Path,
    logpath: &Path,
    uid: u32,
    gid: u32,
) -> Result<i32, Error> {
    let mut fds = [0 as libc::c_int; 2];
    if unsafe { libc::pipe(fds.as_mut_ptr()) } != 0 {
        return Err(Error::io_msg("pipe failed"));
    }
    let (reader, writer) = (fds[0], fds[1]);
    let pid = unsafe { libc::fork() };
    if pid < 0 {
        unsafe {
            libc::close(reader);
            libc::close(writer);
        }
        return Err(Error::io_msg("fork failed"));
    }
    if pid == 0 {
        unsafe {
            libc::close(reader);
            libc::dup2(writer, 1);
            libc::dup2(writer, 2);
            libc::close(writer);
            let dir = match to_cstring(workdir.as_os_str().as_bytes()) {
                Some(dir) => dir,
                None => libc::_exit(127),
            };
            if libc::chdir(dir.as_ptr()) != 0 {
                libc::_exit(127);
            }
            // Drop supplementary groups first so the role keeps exactly
            // its primary group. The privileged supervisor is always
            // root; the call is skipped only for unprivileged test
            // drivers of this function.
            if libc::geteuid() == 0 && libc::setgroups(0, std::ptr::null()) != 0 {
                libc::_exit(127);
            }
            if libc::setgid(gid) != 0 {
                libc::_exit(127);
            }
            if libc::setuid(uid) != 0 {
                libc::_exit(127);
            }
            let mut owned: Vec<CString> = Vec::with_capacity(argv.len() + environ.len());
            for word in argv {
                match to_cstring(word.as_bytes()) {
                    Some(word) => owned.push(word),
                    None => libc::_exit(127),
                }
            }
            let argv_count = owned.len();
            for (key, value) in environ {
                let mut pair = Vec::with_capacity(key.len() + value.len() + 1);
                pair.extend_from_slice(key.as_bytes());
                pair.push(b'=');
                pair.extend_from_slice(value.as_bytes());
                match to_cstring(&pair) {
                    Some(pair) => owned.push(pair),
                    None => libc::_exit(127),
                }
            }
            let mut argv_ptrs: Vec<*const libc::c_char> =
                owned[..argv_count].iter().map(|s| s.as_ptr()).collect();
            argv_ptrs.push(std::ptr::null());
            let mut env_ptrs: Vec<*const libc::c_char> =
                owned[argv_count..].iter().map(|s| s.as_ptr()).collect();
            env_ptrs.push(std::ptr::null());
            libc::execve(argv_ptrs[0], argv_ptrs.as_ptr(), env_ptrs.as_ptr());
            libc::_exit(127);
        }
    }
    unsafe {
        libc::close(writer);
    }
    let mut log = File::create(logpath).map_err(Error::classify)?;
    let mut kept = 0usize;
    loop {
        let mut chunk = [0u8; 65536];
        let got =
            unsafe { libc::read(reader, chunk.as_mut_ptr() as *mut libc::c_void, chunk.len()) };
        if got < 0 {
            if std::io::Error::last_os_error().raw_os_error() == Some(libc::EINTR) {
                continue;
            }
            unsafe {
                libc::close(reader);
            }
            return Err(Error::io_msg("log pipe failed"));
        }
        if got == 0 {
            break;
        }
        if kept < crate::LOG_CAP {
            let take = (got as usize).min(crate::LOG_CAP - kept);
            log.write_all(&chunk[..take])
                .map_err(|err| Error::io("write", &err))?;
            kept += take;
        }
    }
    if kept >= crate::LOG_CAP {
        log.write_all(b"\n[output truncated]\n")
            .map_err(|err| Error::io("write", &err))?;
    }
    drop(log);
    unsafe {
        libc::close(reader);
    }
    let status = wait_child(pid);
    if status < 0 {
        return Err(Error::io_msg("waitpid failed"));
    }
    Ok(exit_code(status))
}

/// `setup.sh`, then `check.sh` when setup passed and no stop tombstone
/// raced the launch; completion is always recorded.
pub fn run_preparation(
    ctx: &crate::Ctx,
    directory: &Path,
    fields: &ReqFields,
    tools: &[JsonValue],
    account: &Account,
) -> Result<(), Error> {
    let checkout = account.dir.join("checkouts").join(&fields.id);
    let home = checkout.join(".soda-home");
    fsx::mkdir_p(&home.join("tmp"), 0o700)?;
    fsx::chown(&home.join("tmp"), account.uid, account.gid)?;
    let mut dirs: Vec<String> = tools
        .iter()
        .map(|tool| parent_of(tool.get("path").and_then(|v| v.as_str()).unwrap_or("")))
        .collect();
    dirs.sort();
    dirs.dedup();
    dirs.push("/usr/bin".to_string());
    dirs.push("/bin".to_string());
    let credential_file = if fields.credential.is_empty() {
        String::new()
    } else {
        ctx.credentials
            .join(&fields.role)
            .join(&fields.credential)
            .to_string_lossy()
            .into_owned()
    };
    let snapshot = directory.join("snapshot");
    let home_text = home.to_string_lossy().into_owned();
    let environ = child_environ(
        &dirs.join(":"),
        &home_text,
        &credential_file,
        &fields.role,
        &fields.id,
        &snapshot.to_string_lossy(),
    );
    let setup = vec![
        crate::SHELL.to_string(),
        format!("{}/{}", snapshot.to_string_lossy(), crate::SETUP_ENTRY),
    ];
    let setup_exit = run_as_role(
        &setup,
        &environ,
        &checkout,
        &directory.join("setup.log"),
        account.uid,
        account.gid,
    )?;
    let mut check_exit: Option<i32> = None;
    if setup_exit == 0 && !fsx::lexists(&directory.join("stopped.json")) {
        let check = vec![
            crate::SHELL.to_string(),
            format!("{}/{}", snapshot.to_string_lossy(), crate::CHECK_ENTRY),
        ];
        check_exit = Some(run_as_role(
            &check,
            &environ,
            &checkout,
            &directory.join("check.log"),
            account.uid,
            account.gid,
        )?);
    }
    let finished = obj(vec![
        ("setup_exit", JsonValue::Number(setup_exit.to_string())),
        (
            "check_exit",
            check_exit
                .map(|code| JsonValue::Number(code.to_string()))
                .unwrap_or(JsonValue::Null),
        ),
    ]);
    let payload = crate::emit::dumps_default(&finished);
    fsx::write_new(&directory.join("finished.json"), payload.as_bytes(), 0o644)?;
    Ok(())
}

/// Double-fork supervisor: the intermediate records the leader before
/// exiting, the grandchild detaches fully (closing the inherited lock)
/// and runs the preparation. A stop tombstone racing the launch wins.
pub fn spawn_detached(
    ctx: &crate::Ctx,
    directory: &Path,
    fields: &ReqFields,
    tools: &[JsonValue],
) -> Result<JsonValue, Error> {
    let account = crate::account::role_record(ctx, &fields.role)?;
    let first = unsafe { libc::fork() };
    if first < 0 {
        return Err(Error::io_msg("fork failed"));
    }
    if first != 0 {
        let status = wait_child(first);
        if status != 0 {
            return fail("detached setup launch unconfirmed");
        }
        return fsx::read_json(ctx, &directory.join("started.json"), 1024);
    }
    unsafe {
        if libc::setsid() < 0 {
            libc::_exit(1);
        }
        let second = libc::fork();
        if second < 0 {
            libc::_exit(1);
        }
        if second != 0 {
            let payload = format!("{{\"pid\": {second}, \"pgid\": {second}}}");
            match fsx::write_new(&directory.join("started.json"), payload.as_bytes(), 0o644) {
                Ok(()) => libc::_exit(0),
                Err(_) => libc::_exit(1),
            }
        }
    }
    let outcome = grandchild_body(ctx, directory, fields, tools, account.as_ref());
    if outcome.is_err() {
        let _ = fsx::write_new(
            &directory.join("finished.json"),
            b"{\"setup_exit\": 127, \"check_exit\": null, \"launch_error\": true}",
            0o644,
        );
    }
    unsafe {
        libc::_exit(0);
    }
}

fn grandchild_body(
    ctx: &crate::Ctx,
    directory: &Path,
    fields: &ReqFields,
    tools: &[JsonValue],
    account: Option<&Account>,
) -> Result<(), Error> {
    unsafe {
        for fd in 3..1024 {
            libc::close(fd);
        }
        if libc::setpgid(0, 0) != 0 {
            return Err(Error::io_msg("setpgid failed"));
        }
        let root = CString::new("/").expect("root");
        if libc::chdir(root.as_ptr()) != 0 {
            return Err(Error::io_msg("chdir failed"));
        }
        let devnull = libc::open(c"/dev/null".as_ptr(), libc::O_RDWR);
        if devnull < 0 {
            return Err(Error::io_msg("devnull failed"));
        }
        libc::dup2(devnull, 0);
        libc::dup2(devnull, 1);
        libc::dup2(devnull, 2);
        if devnull > 2 {
            libc::close(devnull);
        }
    }
    if fsx::lexists(&directory.join("stopped.json")) {
        return Ok(());
    }
    match account {
        Some(account) => run_preparation(ctx, directory, fields, tools, account),
        None => Err(Error::fail("role account vanished before launch")),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parent_matrix() {
        assert_eq!(parent_of("/usr/bin/go"), "/usr/bin");
        assert_eq!(parent_of("go"), ".");
        assert_eq!(parent_of(""), ".");
        assert_eq!(parent_of("/"), "/");
        assert_eq!(parent_of("///"), "/");
        assert_eq!(parent_of("/go"), "/");
        assert_eq!(parent_of("a/"), ".");
        assert_eq!(parent_of("a/b/"), "a");
        assert_eq!(parent_of("/a/b"), "/a");
    }

    #[test]
    fn child_environment_is_fixed_and_ordered() {
        let environ = child_environ("/bin", "/h", "", "soda-coder", "f01", "/s");
        let keys: Vec<&str> = environ.iter().map(|(k, _)| k.as_str()).collect();
        assert_eq!(
            keys,
            vec![
                "PATH",
                "HOME",
                "TMPDIR",
                "SODA_ROLE",
                "SODA_PREPARATION_ID",
                "SODA_SNAPSHOT",
            ]
        );
        assert_eq!(environ[2].1, "/h/tmp");
        let with_cred = child_environ("/bin", "/h", "/c", "soda-coder", "f01", "/s");
        assert_eq!(with_cred.len(), 7);
        assert_eq!(
            with_cred[6],
            ("SODA_CREDENTIAL_FILE".to_string(), "/c".to_string())
        );
    }
}

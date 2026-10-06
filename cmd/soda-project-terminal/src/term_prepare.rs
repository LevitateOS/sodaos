use std::fs::File;
use std::io;
use std::os::unix::io::AsRawFd;

use soda_json::JsonValue;

use crate::account;
use crate::fs;
use crate::proto;
use crate::pyemit;
use crate::svc;
use crate::sys;
use crate::term_binding::binding_record;
use crate::term_paths::{checked_chain, closed_launch_failed, list_dir_names, terminal_path};
use crate::term_status::ready_object;

/// `tmux new-session` argv for the owned `soda` session. `-D` starts empty
/// and disables exit-empty; normal empty-server exit is restored AFTER
/// new-session in the same command queue (also supported by 3.2a).
pub fn tmux_new_session_args(cols: i64, rows: i64, home: &str, profile: &[String]) -> Vec<String> {
    let mut argv = vec![
        "new-session".to_string(),
        "-d".to_string(),
        "-s".to_string(),
        "soda".to_string(),
        "-x".to_string(),
        cols.to_string(),
        "-y".to_string(),
        rows.to_string(),
        "-c".to_string(),
        home.to_string(),
    ];
    argv.extend(profile.iter().cloned());
    argv.extend(
        [";", "set-option", "-s", "exit-empty", "on"]
            .into_iter()
            .map(|s| s.to_string()),
    );
    argv
}

fn subscription_session(directory: &File) -> Result<Vec<String>, String> {
    if !list_dir_names(directory)?
        .iter()
        .any(|n| n == "subscription")
    {
        return Ok(Vec::new());
    }
    let profile = fs::read_record(directory, "subscription")?;
    let id = profile
        .get("binding")
        .and_then(|v| v.get("id"))
        .and_then(|v| v.as_str())
        .ok_or_else(|| "terminal subscription".to_string())?;
    let unit = format!("soda-terminal-{id}.service");
    let output =
        sys::run_output(&svc::invocation_show_argv(&unit), 2).map_err(|e| e.to_string())?;
    if !output.status.success() {
        return Err("terminal incarnation".to_string());
    }
    let invocation = std::str::from_utf8(&output.stdout)
        .map_err(|_| "terminal incarnation".to_string())?
        .trim()
        .to_string();
    if !proto::valid_scope(&invocation) {
        return Err("missing terminal incarnation".to_string());
    }
    let record = JsonValue::Object(vec![(
        "invocation_id".to_string(),
        JsonValue::Str(invocation),
    )]);
    fs::new_file(
        directory,
        "subscription-unit",
        &pyemit::line(&record),
        0o600,
    )?;
    Ok(vec!["exec /usr/bin/sleep infinity".to_string()])
}

/// Privileged one-shot hook: verify the cgroup/socket, open the session,
/// publish `ready`. Silent success; `closed/launch_failed` + 1 on failure
/// (the `.py` `main` wrapper reports prepare failures exactly so).
pub fn prepare(identifier: &str) -> i32 {
    match prepare_inner(identifier) {
        Ok(()) => 0,
        Err(_) => {
            closed_launch_failed();
            1
        }
    }
}

fn prepare_inner(identifier: &str) -> Result<(), String> {
    let path = terminal_path(identifier)?;
    let parent = svc::cgroup_parent()?;
    let group =
        sys::open_child_dir(&parent, &svc::unit_name(identifier)).map_err(|e| e.to_string())?;
    drop(parent);
    let (uid, mode) = fs::fstat_uid_mode(&group).map_err(|e| e.to_string())?;
    if uid != 0 || mode & 0o022 != 0 {
        return Err("unsafe terminal cgroup".to_string());
    }
    let procs = sys::open_at(&group, "cgroup.procs", libc::O_RDONLY | libc::O_NOFOLLOW, 0)
        .map_err(|e| e.to_string())?;
    drop(group);
    let mut content = vec![0u8; 16384];
    let got = loop {
        let got = unsafe {
            libc::read(
                procs.as_raw_fd(),
                content.as_mut_ptr() as *mut libc::c_void,
                content.len(),
            )
        };
        if got < 0 && io::Error::last_os_error().raw_os_error() == Some(libc::EINTR) {
            continue;
        }
        if got < 0 {
            return Err(io::Error::last_os_error().to_string());
        }
        break got as usize;
    };
    content.truncate(got);
    if !content.is_ascii() {
        return Err("preparation outside owned cgroup".to_string());
    }
    let text = std::str::from_utf8(&content)
        .map_err(|_| "preparation outside owned cgroup".to_string())?;
    let here = std::process::id().to_string();
    if !text
        .split('\n')
        .any(|line| line.strip_suffix('\r').unwrap_or(line) == here)
    {
        return Err("preparation outside owned cgroup".to_string());
    }
    drop(procs);
    let directory = checked_chain(&path)?;
    let record = binding_record(&directory, None, 0)?;
    let account = account::account_for(&record.login, record.identity)?;
    binding_record(&directory, Some(&account), record.identity)?;
    let sock = format!("{path}/screen/socket");
    // MAINPID is supplied by systemd, never by the browser.
    let pid: i64 = std::env::var("MAINPID")
        .map_err(|_| "missing main process".to_string())?
        .parse()
        .map_err(|_| "missing main process".to_string())?;
    if pid <= 0 || pid > i64::from(i32::MAX) {
        return Err("missing main process".to_string());
    }
    let pid = pid as i32;
    let until = sys::monotonic() + 5.0;
    loop {
        if sys::monotonic() >= until {
            return Err("tmux startup failed".to_string());
        }
        if std::path::Path::new(&format!("{sock}.lock")).exists() {
            std::thread::sleep(std::time::Duration::from_millis(20));
            continue;
        }
        match svc::socket_identity_kinded(&sock, &account, pid) {
            Ok(_) => break,
            Err(svc::SocketCheck::Retryable(_)) => {
                std::thread::sleep(std::time::Duration::from_millis(20));
            }
            Err(svc::SocketCheck::Fatal(message)) => return Err(message),
        }
    }
    // Seal the socket's parent before publishing readiness: the project user
    // may connect but cannot replace this socket with another context.
    fs::chown_path(&format!("{path}/screen"), 0, 0, true).map_err(|e| e.to_string())?;
    fs::chmod_path(&format!("{path}/screen"), 0o711, true).map_err(|e| e.to_string())?;
    let (dev, ino) = svc::socket_identity(&sock, &account, pid)?;
    let profile = subscription_session(&directory)?;
    svc::tmux_control(
        &account,
        &sock,
        &tmux_new_session_args(record.cols, record.rows, &account.pw_dir, &profile),
    )?;
    fs::new_file(
        &directory,
        "ready",
        &pyemit::line(&ready_object(i64::from(pid), dev, ino)),
        0o600,
    )?;
    Ok(())
}

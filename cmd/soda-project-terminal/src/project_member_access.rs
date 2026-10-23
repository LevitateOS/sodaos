//! Native retained-account access effects. The host reuses this file against a
//! stopped container's mounted root; process operations run only inside its PID
//! namespace. No account, home or independent service identity is removed.
use serde::{Deserialize, Serialize};
use std::fs::{File, OpenOptions};
use std::io::{Read, Write};
use std::os::fd::AsRawFd;
use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
use std::path::{Path, PathBuf};
use std::time::{Duration, Instant};

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Request {
    pub project: String,
    pub login: String,
    pub identity: i64,
    pub action: String,
    pub revision: i64,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct Process {
    pub pid: i32,
    pub command: String,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct State {
    pub project: String,
    pub login: String,
    pub identity: i64,
    pub revision: i64,
    pub held: bool,
    pub requires_join: bool,
    pub login_disabled: bool,
    pub keys_removed: bool,
    pub processes_stopped: bool,
    pub keys: Vec<String>,
    pub processes: Vec<Process>,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Hold {
    identity: i64,
    revision: i64,
    phase: String,
}

fn valid_login(login: &str) -> bool {
    let b = login.as_bytes();
    !b.is_empty()
        && b.len() <= 31
        && b[0].is_ascii_lowercase()
        && login != "root"
        && b.iter()
            .all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || *c == b'_' || *c == b'-')
}

fn open(path: &Path, directory: bool) -> Result<File, String> {
    OpenOptions::new()
        .read(true)
        .custom_flags(
            libc::O_NOFOLLOW
                | libc::O_NONBLOCK
                | libc::O_CLOEXEC
                | if directory { libc::O_DIRECTORY } else { 0 },
        )
        .open(path)
        .map_err(|_| "member access file unavailable".to_string())
}

fn directory(root: &Path, parts: &[&str]) -> Result<PathBuf, String> {
    let mut path = root.to_path_buf();
    for part in parts {
        path.push(part);
        let info = open(&path, true)?
            .metadata()
            .map_err(|_| "member directory unavailable")?;
        if info.mode() & 0o022 != 0 {
            return Err("unsafe member directory".into());
        }
    }
    Ok(path)
}

fn read(path: &Path, owner: u32, limit: u64) -> Result<Vec<u8>, String> {
    let file = open(path, false)?;
    let info = file.metadata().map_err(|_| "member file unavailable")?;
    if !info.is_file()
        || info.uid() != owner
        || info.nlink() != 1
        || info.mode() & 0o022 != 0
        || info.len() > limit
    {
        return Err("unsafe member file".into());
    }
    let mut bytes = Vec::new();
    file.take(limit + 1)
        .read_to_end(&mut bytes)
        .map_err(|_| "member file unreadable")?;
    if bytes.len() as u64 > limit {
        return Err("oversized member file".into());
    }
    Ok(bytes)
}

fn atomic(path: &Path, bytes: &[u8], owner: u32, group: u32, mode: u32) -> Result<(), String> {
    let parent = path.parent().ok_or("member parent unavailable")?;
    let temp = parent.join(format!(".soda-access-{}", std::process::id()));
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(mode)
        .custom_flags(libc::O_NOFOLLOW | libc::O_CLOEXEC)
        .open(&temp)
        .map_err(|_| "member write unavailable")?;
    let result = (|| {
        if unsafe { libc::fchown(file.as_raw_fd(), owner, group) } != 0 {
            return Err("member ownership unconfirmed".into());
        }
        if unsafe { libc::fchmod(file.as_raw_fd(), mode) } != 0 {
            return Err("member permissions unconfirmed".into());
        }
        file.write_all(bytes).map_err(|_| "member write failed")?;
        file.sync_all().map_err(|_| "member sync failed")?;
        std::fs::rename(&temp, path).map_err(|_| "member replacement failed")?;
        open(parent, true)?
            .sync_all()
            .map_err(|_| "member directory sync failed")?;
        Ok(())
    })();
    if result.is_err() {
        let _ = std::fs::remove_file(&temp);
    }
    result
}

fn hold(accounts: &Path, login: &str, identity: i64, owner: u32) -> Result<Option<Hold>, String> {
    let path = accounts.join(format!("{login}.access"));
    match std::fs::symlink_metadata(&path) {
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(None),
        Err(_) => Err("member hold unavailable".into()),
        Ok(_) => {
            let h: Hold = serde_json::from_slice(&read(&path, owner, 1024)?)
                .map_err(|_| "invalid member hold")?;
            if h.identity != identity
                || h.revision <= 0
                || (h.phase != "revoked" && h.phase != "rejoin")
            {
                return Err("member hold identity mismatch".into());
            }
            Ok(Some(h))
        }
    }
}

fn passwd(root: &Path, login: &str, owner: u32) -> Result<(PathBuf, String, u32, String), String> {
    let etc = directory(root, &["etc"])?;
    let path = etc.join("passwd");
    let text = String::from_utf8(read(&path, owner, 1 << 20)?)
        .map_err(|_| "native account unavailable")?;
    let mut matching = text
        .lines()
        .filter(|line| line.split(':').next() == Some(login));
    let fields: Vec<_> = matching
        .next()
        .ok_or("native account absent")?
        .split(':')
        .collect();
    if fields.len() != 7 || matching.next().is_some() {
        return Err("invalid native account".into());
    }
    let uid: u32 = fields[2].parse().map_err(|_| "invalid account uid")?;
    if uid == 0 {
        return Err("root account refused".into());
    }
    let shell = fields[6].to_string();
    Ok((path, text, uid, shell))
}

fn set_shell(
    path: &Path,
    text: &str,
    login: &str,
    shell: &str,
    owner: u32,
    group: u32,
) -> Result<(), String> {
    let mut lines = Vec::new();
    for line in text.lines() {
        let mut fields: Vec<_> = line.split(':').collect();
        if fields.first() == Some(&login) {
            fields[6] = shell;
            lines.push(fields.join(":"));
        } else {
            lines.push(line.into());
        }
    }
    atomic(
        path,
        format!("{}\n", lines.join("\n")).as_bytes(),
        owner,
        group,
        0o644,
    )
}

fn processes(uid: u32) -> Result<Vec<Process>, String> {
    let mut out = Vec::new();
    for (count, entry) in std::fs::read_dir("/proc")
        .map_err(|_| "process observation unavailable")?
        .enumerate()
    {
        if count > 65536 {
            return Err("process observation limit exceeded".into());
        }
        let entry = entry.map_err(|_| "process observation incomplete")?;
        let Some(pid) = entry
            .file_name()
            .to_str()
            .and_then(|s| s.parse::<i32>().ok())
        else {
            continue;
        };
        let info = match entry.metadata() {
            Ok(v) => v,
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => continue,
            Err(_) => return Err("process identity unavailable".into()),
        };
        if info.uid() != uid {
            continue;
        }
        let status = match std::fs::read_to_string(entry.path().join("status")) {
            Ok(v) => v,
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => continue,
            Err(_) => return Err("process state unavailable".into()),
        };
        if status.lines().any(|s| s.starts_with("State:\tZ")) {
            continue;
        }
        let command = status
            .lines()
            .find_map(|s| s.strip_prefix("Name:\t"))
            .unwrap_or("process")
            .chars()
            .take(128)
            .collect();
        out.push(Process { pid, command });
        if out.len() > 4096 {
            return Err("member process limit exceeded".into());
        }
    }
    out.sort_by_key(|p| p.pid);
    Ok(out)
}

fn terminate(uid: u32) -> Result<(), String> {
    let deadline = Instant::now() + Duration::from_secs(3);
    loop {
        let current = processes(uid)?;
        if current.is_empty() {
            return Ok(());
        }
        for process in current {
            let fd = unsafe { libc::syscall(libc::SYS_pidfd_open, process.pid, 0) as i32 };
            if fd < 0 {
                if std::io::Error::last_os_error().raw_os_error() == Some(libc::ESRCH) {
                    continue;
                }
                return Err("member process handle unavailable".into());
            }
            // The pinned process must still belong to this UID. A recycled PID
            // cannot redirect the signal to an unrelated member.
            let matches =
                std::fs::metadata(format!("/proc/{}", process.pid)).is_ok_and(|m| m.uid() == uid);
            let rc = if matches {
                unsafe {
                    libc::syscall(
                        libc::SYS_pidfd_send_signal,
                        fd,
                        libc::SIGKILL,
                        std::ptr::null::<libc::siginfo_t>(),
                        0,
                    )
                }
            } else {
                0
            };
            let error = std::io::Error::last_os_error();
            unsafe { libc::close(fd) };
            if rc < 0 && error.raw_os_error() != Some(libc::ESRCH) {
                return Err("member termination unconfirmed".into());
            }
        }
        if Instant::now() >= deadline {
            return Err("member processes remain".into());
        }
        std::thread::sleep(Duration::from_millis(20));
    }
}

pub fn operate(root: &Path, request: &Request, live: bool) -> Result<State, String> {
    if !valid_login(&request.login)
        || request.identity <= 0
        || request.revision < 0
        || !["inspect", "revoke", "restore"].contains(&request.action.as_str())
    {
        return Err("invalid member operation".into());
    }
    let accounts = directory(root, &["var", "lib", "soda", "accounts"])?;
    let keys = directory(root, &["etc", "ssh", "authorized_keys"])?;
    let lock = open(&keys, true)?;
    if unsafe { libc::flock(lock.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) } != 0 {
        return Err("member keys busy".into());
    }
    let meta = lock
        .metadata()
        .map_err(|_| "member ownership unavailable")?;
    let owner = meta.uid();
    let group = meta.gid();
    if read(&accounts.join(&request.login), owner, 64)? != request.identity.to_string().as_bytes() {
        return Err("member identity mismatch".into());
    }
    let mut current = hold(&accounts, &request.login, request.identity, owner)?;
    let (path, text, uid, shell) = passwd(root, &request.login, owner)?;
    let keypath = keys.join(&request.login);
    let existing =
        String::from_utf8(read(&keypath, owner, 65536)?).map_err(|_| "member keys unavailable")?;
    if request.action != "inspect" {
        if request.revision <= 0
            || current
                .as_ref()
                .is_some_and(|h| h.revision > request.revision)
        {
            return Err("stale member operation".into());
        }
        let phase = if request.action == "revoke" {
            "revoked"
        } else {
            "rejoin"
        };
        if request.action == "restore" && current.is_none() {
            return Err("member has no revocation".into());
        }
        if current
            .as_ref()
            .is_some_and(|h| h.revision == request.revision && h.phase != phase)
        {
            return Err("member operation conflict".into());
        }
        let next = Hold {
            identity: request.identity,
            revision: request.revision,
            phase: phase.into(),
        };
        atomic(
            &accounts.join(format!("{}.access", request.login)),
            &serde_json::to_vec(&next).map_err(|_| "member hold encoding failed")?,
            owner,
            group,
            0o600,
        )?;
        atomic(&keypath, b"", owner, group, 0o644)?;
        set_shell(
            &path,
            &text,
            &request.login,
            "/usr/sbin/nologin",
            owner,
            group,
        )?;
        if live {
            terminate(uid)?
        }
        current = Some(next);
    }
    // Reopen the actual files after effects. Returning optimistic requested
    // values would turn a successful dispatch into a false native receipt.
    let changed = request.action != "inspect";
    let (observed_keys, observed_shell) = if changed {
        current = hold(&accounts, &request.login, request.identity, owner)?;
        let (_, _, _, shell) = passwd(root, &request.login, owner)?;
        (
            String::from_utf8(read(&keypath, owner, 65536)?)
                .map_err(|_| "member key readback unavailable")?,
            shell,
        )
    } else {
        (existing, shell)
    };
    let keys: Vec<String> = observed_keys
        .lines()
        .filter(|s| !s.is_empty())
        .map(str::to_owned)
        .collect();
    let process = if live { processes(uid)? } else { Vec::new() };
    Ok(State {
        project: request.project.clone(),
        login: request.login.clone(),
        identity: request.identity,
        revision: current.as_ref().map_or(0, |h| h.revision),
        held: current.as_ref().is_some_and(|h| h.phase == "revoked"),
        requires_join: current.as_ref().is_some_and(|h| h.phase == "rejoin"),
        login_disabled: observed_shell.ends_with("/nologin") || observed_shell.ends_with("/false"),
        keys_removed: keys.is_empty(),
        processes_stopped: process.is_empty(),
        keys,
        processes: process,
    })
}

/// Returns true only for an explicitly restored account waiting to Join.
pub fn join_state(root: &Path, login: &str, identity: i64) -> Result<bool, String> {
    let accounts = directory(root, &["var", "lib", "soda", "accounts"])?;
    let owner = open(&accounts, true)?
        .metadata()
        .map_err(|_| "account ownership unavailable")?
        .uid();
    match hold(&accounts, login, identity, owner)? {
        Some(h) if h.phase == "revoked" => Err("Project access revoked".into()),
        Some(_) => Ok(true),
        None => Ok(false),
    }
}

pub fn complete_join(root: &Path, login: &str, identity: i64) -> Result<(), String> {
    if !join_state(root, login, identity)? {
        return Ok(());
    }
    let accounts = directory(root, &["var", "lib", "soda", "accounts"])?;
    let meta = open(&accounts, true)?
        .metadata()
        .map_err(|_| "account ownership unavailable")?;
    let (path, text, _, _) = passwd(root, login, meta.uid())?;
    set_shell(&path, &text, login, "/bin/bash", meta.uid(), meta.gid())?;
    std::fs::remove_file(accounts.join(format!("{login}.access")))
        .map_err(|_| "rejoin unconfirmed")?;
    open(&accounts, true)?
        .sync_all()
        .map_err(|_| "rejoin sync failed".to_string())
}

pub fn replace_join_keys(root: &Path, login: &str, keys: &[String]) -> Result<(), String> {
    let dir = directory(root, &["etc", "ssh", "authorized_keys"])?;
    let meta = open(&dir, true)?
        .metadata()
        .map_err(|_| "key ownership unavailable")?;
    let bytes = if keys.is_empty() {
        String::new()
    } else {
        format!(
            "{}\n",
            keys.iter().map(|k| k.trim()).collect::<Vec<_>>().join("\n")
        )
    };
    atomic(
        &dir.join(login),
        bytes.as_bytes(),
        meta.uid(),
        meta.gid(),
        0o644,
    )
}

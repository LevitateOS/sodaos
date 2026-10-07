use std::fs::File;
use std::io;
use std::os::unix::io::AsRawFd;

use crate::state_json::StateValue;

use crate::account::{self, Account};
use crate::fs;
use crate::pyemit;
use crate::sha::{self, Digest};
use crate::svc;
use crate::sys;
use crate::term_attach::subscription_lifetime;
use crate::term_binding::{
    binding_object, binding_record, read_reservation, reservation_object, write_name, BindingRecord,
};
use crate::term_collect::{collect_finished, terminal_directories};
use crate::term_paths::{
    checked_chain, fstatat, s_isreg, terminal_path, PROGRAM, TERMINALS, TMUX_CONFIG,
};
use crate::term_status::terminal_status;
use crate::timex;

/// Pure `project-terminal` stat rule: regular, uid 0, `mode & 0o022 == 0`.
pub fn program_stat_ok(uid: u32, mode: u32, is_regular: bool) -> bool {
    is_regular && uid == 0 && mode & 0o022 == 0
}

/// Streaming SHA-256 hex over the whole file — DELIBERATE DELTA: the `.py`
/// caps at 64KB because it hashes a script, but this binary replaces it.
pub fn file_sha256_hex(file: &File) -> io::Result<String> {
    let mut hasher = sha::Sha256::new();
    let mut chunk = vec![0u8; 65536];
    loop {
        let got = unsafe {
            libc::read(
                file.as_raw_fd(),
                chunk.as_mut_ptr() as *mut libc::c_void,
                chunk.len(),
            )
        };
        if got < 0 {
            let err = io::Error::last_os_error();
            if err.raw_os_error() == Some(libc::EINTR) {
                continue;
            }
            return Err(err);
        }
        if got == 0 {
            break;
        }
        hasher.update(&chunk[..got as usize]);
    }
    Ok(sha::hex(&hasher.finalize()))
}

fn verify_program(source_hash: &str) -> Result<(), String> {
    // Missing/older project support refuses, never installs itself on Open.
    let parent = checked_chain("/usr/libexec/soda")?;
    let program = sys::open_at(
        &parent,
        "project-terminal",
        libc::O_RDONLY | libc::O_NOFOLLOW | libc::O_NONBLOCK,
        0,
    )
    .map_err(|e| format!("open program: {e}"))?;
    let mut info: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstat(program.as_raw_fd(), &mut info) } != 0 {
        return Err(io::Error::last_os_error().to_string());
    }
    if !program_stat_ok(info.st_uid, info.st_mode, s_isreg(info.st_mode)) {
        return Err("unsafe terminal program".to_string());
    }
    let digest = file_sha256_hex(&program).map_err(|e| e.to_string())?;
    if digest != source_hash {
        return Err("terminal support version differs".to_string());
    }
    Ok(())
}

/// Reserve a terminal locator; the caller holds the parent lock.
#[allow(clippy::too_many_arguments)] // 8-arg shape mandated by the PR25 brief
pub fn reserve_terminal(
    identifier: &str,
    account: &Account,
    identity: i64,
    cols: i64,
    rows: i64,
    name: &str,
    source_hash: &str,
    scope: &str,
) -> Result<StateValue, String> {
    verify_program(source_hash)?;
    for term in ["xterm-256color", "screen-256color"] {
        sys::run_checked(&svc::infocmp_argv(term), 2)
            .map_err(|e| format!("infocmp {term}: {e}"))?;
    }
    let path = terminal_path(identifier)?;
    // Refuse occupied locators before retiring any completed runtime files.
    // The caller holds the native parent lock across preparation/admission.
    let parent = checked_chain(TERMINALS)?;
    match fstatat(&parent, identifier) {
        Ok(_) => return Err("terminal identifier occupied".to_string()),
        Err(err) if err.kind() == io::ErrorKind::NotFound => {}
        Err(err) => return Err(err.to_string()),
    }
    if svc::service_state(identifier, None)? != "inactive" || !svc::cgroup_empty(identifier)? {
        return Err("terminal unit occupied".to_string());
    }
    collect_finished()?;
    if terminal_directories()?.len() >= 128 {
        return Err("native reservation capacity".to_string());
    }
    fs::mkdir_at(&parent, identifier, 0o711).map_err(|e| e.to_string())?;
    drop(parent);
    let directory = checked_chain(&path)?;
    fs::fchmod(&directory, 0o711).map_err(|e| e.to_string())?;
    let created = BindingRecord {
        login: account.pw_name.clone(),
        uid: account.pw_uid,
        gid: account.pw_gid,
        home: account.pw_dir.clone(),
        shell: account.pw_shell.clone(),
        identity,
        cols,
        rows,
        created_at: timex::now_secs(),
    };
    fs::new_file(
        &directory,
        "binding",
        &pyemit::line(&binding_object(&created)),
        0o600,
    )?;
    fs::new_file(
        &directory,
        "reservation",
        &pyemit::line(&reservation_object(timex::now_secs() + 120, scope)),
        0o600,
    )?;
    write_name(&directory, name)?;
    fs::new_file(&directory, "writer", b"", 0o600)?;
    fs::new_file(&directory, "tmux.conf", TMUX_CONFIG.as_bytes(), 0o644)?;
    fs::mkdir_at(&directory, "screen", 0o700).map_err(|e| e.to_string())?;
    fs::chown_path(
        &format!("{path}/screen"),
        account.pw_uid,
        account.pw_gid,
        false,
    )
    .map_err(|e| e.to_string())?;
    drop(directory);
    terminal_status(identifier, account, identity)?
        .ok_or_else(|| "terminal reservation".to_string())
}

// ---------------------------------------------------------------------------
// Create.
// ---------------------------------------------------------------------------

/// Exact `systemd-run` argv for the supervised tmux server. The privileged
/// hook runs the binary directly: `ExecStartPost=+…/project-terminal prepare
/// <id>` (no `python3`, no `-I`).
pub fn systemd_run_argv(
    identifier: &str,
    account: &Account,
    path: &str,
    lifetime: &[String],
) -> Vec<String> {
    let mut argv = vec![
        "/usr/bin/systemd-run".to_string(),
        "--quiet".to_string(),
        "--collect".to_string(),
        format!("--unit=soda-terminal-{identifier}"),
        format!("--description=Soda terminal {identifier}"),
        "--service-type=exec".to_string(),
        format!("--property=User={}", account.pw_name),
        format!("--property=Group={}", account.pw_gid),
        format!("--property=WorkingDirectory={}", account.pw_dir),
        "--slice=system.slice".to_string(),
        "--property=KillMode=control-group".to_string(),
        "--property=SendSIGKILL=yes".to_string(),
        "--property=Restart=no".to_string(),
        "--property=TimeoutStartSec=10s".to_string(),
        "--property=TimeoutStopSec=3s".to_string(),
        "--property=UMask=0077".to_string(),
        "--property=LimitCORE=0".to_string(),
    ];
    argv.extend(lifetime.iter().cloned());
    argv.extend(
        [
            "--property=StandardInput=null",
            "--property=StandardOutput=null",
            "--property=StandardError=null",
            &format!("--property=ExecStartPost=+{PROGRAM} prepare {identifier}"),
        ]
        .into_iter()
        .map(|s| s.to_string()),
    );
    for (key, value) in account::user_environment(account) {
        argv.push(format!("--setenv={key}={value}"));
    }
    argv.extend(
        [
            "/usr/bin/tmux",
            "-D",
            "-S",
            &format!("{path}/screen/socket"),
            "-f",
            &format!("{path}/tmux.conf"),
        ]
        .into_iter()
        .map(|s| s.to_string()),
    );
    argv
}

/// Consume the reservation and start the supervised server.
pub fn create_terminal(
    identifier: &str,
    account: &Account,
    identity: i64,
    cols: i64,
    rows: i64,
    name: &str,
    scope: &str,
) -> Result<StateValue, String> {
    let path = terminal_path(identifier)?;
    let directory = checked_chain(&path)?; // no creation on a missing/ended locator
    let record = binding_record(&directory, Some(account), identity)?;
    let permit = read_reservation(&directory)?;
    let usable = match permit.as_ref() {
        Some(permit) => {
            permit.expires > timex::now_secs()
                && permit.scope == scope
                && (cols, rows) == (record.cols, record.rows)
        }
        None => false,
    };
    if !usable {
        return Err("creation reservation expired, consumed or changed".to_string());
    }
    if collect_finished()? >= 64 {
        return Err("native terminal capacity".to_string());
    }
    // Same native lock as End: a late Create cannot run after End removed its
    // one-use permission, even across web/helper restarts or a lost reply.
    fs::unlink_at(&directory, "reservation").map_err(|e| e.to_string())?;
    write_name(&directory, name)?;
    drop(directory);
    if svc::service_state(identifier, None)? != "inactive" {
        return Err("terminal unit occupied".to_string());
    }
    let lifetime = subscription_lifetime(&path)?;
    sys::run_checked(&systemd_run_argv(identifier, account, &path, &lifetime), 15)
        .map_err(|e| format!("systemd-run: {e}"))?;
    // No owner stream or lifetime timer. systemd's start job includes the
    // bounded privileged preparation hook; it cleans the cgroup if it fails.
    terminal_status(identifier, account, identity)?.ok_or_else(|| "terminal creation".to_string())
}

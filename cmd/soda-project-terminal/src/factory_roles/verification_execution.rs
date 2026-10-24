//! Gated, pinned supervisor for one approved check in a disposable clone.

use super::{account::Account, execution, fsx, verification::Request};
use crate::emit::obj;
use crate::error::{fail, Error};
use crate::state_json::StateValue;
use std::fs::{File, OpenOptions};
use std::os::fd::{AsRawFd, FromRawFd};
use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
use std::path::Path;

pub(crate) fn sync_dir(path: &Path) -> Result<(), Error> {
    OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_DIRECTORY | libc::O_NOFOLLOW)
        .open(path)
        .map_err(Error::classify)?
        .sync_all()
        .map_err(Error::classify)
}

fn identity(ctx: &crate::Ctx, dir: &Path) -> Result<Option<(i32, u64)>, Error> {
    let started = match fsx::read_json(ctx, &dir.join("started.json"), 1024) {
        Ok(v) => v,
        Err(Error::Missing) => return Ok(None),
        Err(e) => return Err(e),
    };
    if !super::validate::as_object(&started)
        .is_some_and(|e| super::validate::key_set(e, &["pid", "pgid", "start_ticks"]))
    {
        return fail("unsafe verification supervisor record");
    }
    let number = |key| started.get(key).and_then(super::validate::as_i64);
    let pid = number("pid")
        .and_then(|v| i32::try_from(v).ok())
        .filter(|v| *v > 1)
        .ok_or_else(|| Error::fail("unsafe verification supervisor"))?;
    let ticks = number("start_ticks")
        .and_then(|v| u64::try_from(v).ok())
        .filter(|v| *v > 0)
        .ok_or_else(|| Error::fail("unsafe verification supervisor"))?;
    if number("pgid") != Some(i64::from(pid)) {
        return fail("unsafe verification supervisor group");
    }
    Ok(Some((pid, ticks)))
}

fn live_identity(ctx: &crate::Ctx, pid: i32, ticks: u64) -> Result<bool, Error> {
    let path = Path::new("/proc").join(pid.to_string());
    let meta = match std::fs::symlink_metadata(&path) {
        Ok(v) => v,
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(false),
        Err(e) => return Err(Error::classify(e)),
    };
    let (state, group, observed) = execution::proc_identity(&path);
    // A different start time proves that the original process has exited.
    // Never signal its replacement, or make retained terminal replay depend
    // on whether the operating system has subsequently reused this pid.
    if observed.is_some_and(|value| value != ticks) {
        return Ok(false);
    }
    if meta.uid() != ctx.priv_uid()
        || group.as_deref() != Some(pid.to_string().as_str())
        || observed != Some(ticks)
    {
        return fail("verification supervisor identity is uncertain");
    }
    Ok(state.as_deref() != Some("Z") && state.is_some())
}

pub(crate) fn supervisor_alive(ctx: &crate::Ctx, dir: &Path) -> Result<bool, Error> {
    match identity(ctx, dir)? {
        Some((pid, ticks)) => live_identity(ctx, pid, ticks),
        None => Ok(false),
    }
}

/// A pidfd pins the exact recorded process across validation and signalling.
pub(crate) fn stop_supervisor(ctx: &crate::Ctx, dir: &Path) -> Result<bool, Error> {
    let Some((pid, ticks)) = identity(ctx, dir)? else {
        return Ok(true);
    };
    if !live_identity(ctx, pid, ticks)? {
        return Ok(true);
    }
    let fd = unsafe { libc::syscall(libc::SYS_pidfd_open, pid, 0) } as i32;
    if fd < 0 {
        return Ok(!live_identity(ctx, pid, ticks)?);
    }
    let fd = unsafe { File::from_raw_fd(fd) };
    if !live_identity(ctx, pid, ticks)? {
        return Ok(true);
    }
    for signal in [libc::SIGTERM, libc::SIGKILL] {
        let rc = unsafe {
            libc::syscall(
                libc::SYS_pidfd_send_signal,
                fd.as_raw_fd(),
                signal,
                std::ptr::null::<libc::siginfo_t>(),
                0,
            )
        };
        if rc != 0 && std::io::Error::last_os_error().raw_os_error() != Some(libc::ESRCH) {
            return Ok(false);
        }
        let mut event = libc::pollfd {
            fd: fd.as_raw_fd(),
            events: libc::POLLIN,
            revents: 0,
        };
        if unsafe { libc::poll(&mut event, 1, 1000) } > 0 && event.revents & libc::POLLIN != 0 {
            return Ok(true);
        }
    }
    Ok(false)
}

pub(crate) fn record_fixture(ctx: &crate::Ctx, dir: &Path, account: &Account) -> Result<(), Error> {
    let mut fields = Vec::new();
    for name in ["work", "home"] {
        let meta = std::fs::symlink_metadata(dir.join(name)).map_err(Error::classify)?;
        if !meta.is_dir() || meta.uid() != account.uid || meta.gid() != account.gid {
            return fail("unsafe verification fixture");
        }
        fields.push((
            name,
            obj(vec![
                ("dev", StateValue::Number(meta.dev().to_string())),
                ("ino", StateValue::Number(meta.ino().to_string())),
                ("uid", StateValue::Number(account.uid.to_string())),
                ("gid", StateValue::Number(account.gid.to_string())),
            ]),
        ));
    }
    fsx::owned_dir(ctx, dir, 0o755)?;
    fsx::write_new(
        &dir.join("fixture.json"),
        crate::emit::dumps_default(&obj(fields)).as_bytes(),
        0o644,
    )?;
    sync_dir(dir)
}

/// The parent is helper-owned: a role cannot replace either top-level inode.
/// remove_dir_all does not follow symlinks inside the owned disposable tree.
pub(crate) fn cleanup(ctx: &crate::Ctx, dir: &Path) -> Result<(), Error> {
    fsx::owned_dir(ctx, dir, 0o755)?;
    let record = fsx::read_json(ctx, &dir.join("fixture.json"), 2048)?;
    for name in ["work", "home"] {
        let path = dir.join(name);
        let meta = match std::fs::symlink_metadata(&path) {
            Ok(v) => v,
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => continue,
            Err(e) => return Err(Error::classify(e)),
        };
        let expected = record
            .get(name)
            .ok_or_else(|| Error::fail("unknown verification fixture"))?;
        for (key, actual) in [
            ("dev", meta.dev()),
            ("ino", meta.ino()),
            ("uid", u64::from(meta.uid())),
            ("gid", u64::from(meta.gid())),
        ] {
            if expected
                .get(key)
                .and_then(super::validate::as_int_text)
                .and_then(|v| v.parse::<u64>().ok())
                != Some(actual)
                || !meta.is_dir()
            {
                return fail("verification fixture ownership is uncertain");
            }
        }
        std::fs::remove_dir_all(&path).map_err(Error::classify)?;
    }
    sync_dir(dir)
}

fn run(
    ctx: &crate::Ctx,
    dir: &Path,
    request: &Request,
    tools: &[StateValue],
    account: &Account,
) -> Result<(i32, Option<i32>), Error> {
    let source = account.dir.join("checkouts").join(&request.id);
    if std::fs::canonicalize(&source).map_err(Error::classify)? != source
        || !std::fs::symlink_metadata(&source)
            .map_err(Error::classify)?
            .is_dir()
    {
        return fail("unsafe verification source checkout");
    }
    let prep = fsx::prep_dir(ctx, &request.id)?;
    let snapshot = prep.join("snapshot");
    let home = dir.join("home");
    let work = dir.join("work");
    let checkout = work.join("checkout");
    let mut paths: Vec<String> = tools
        .iter()
        .map(|t| execution::parent_of(t.get("path").and_then(StateValue::as_str).unwrap_or("")))
        .collect();
    paths.sort();
    paths.dedup();
    paths.extend(["/usr/bin".into(), "/bin".into()]);
    let mut env = execution::child_environ(
        &paths.join(":"),
        &home.to_string_lossy(),
        "",
        "soda-reviewer",
        &request.id,
        &snapshot.to_string_lossy(),
    );
    env.extend([
        ("GIT_CONFIG_NOSYSTEM".into(), "1".into()),
        ("GIT_CONFIG_GLOBAL".into(), "/dev/null".into()),
        ("GIT_TERMINAL_PROMPT".into(), "0".into()),
        ("GIT_OPTIONAL_LOCKS".into(), "0".into()),
        ("GIT_NO_REPLACE_OBJECTS".into(), "1".into()),
    ]);
    env.push(("SODA_VERIFICATION_RUN_ID".into(), request.run_id.clone()));
    let deadline = Some(execution::deadline_instant(&request.deadline)?);
    let execute = |args: Vec<String>, cwd: &Path, log: &str| {
        execution::run_as_role(
            ctx,
            dir,
            &args,
            &env,
            cwd,
            &dir.join(log),
            account.uid,
            account.gid,
            deadline,
        )
    };
    let git = ctx.git().to_string_lossy().into_owned();
    let cloned = execute(
        vec![
            git.clone(),
            "-c".into(),
            format!("safe.directory={}", source.display()),
            "-c".into(),
            "core.hooksPath=/dev/null".into(),
            "clone".into(),
            "--no-hardlinks".into(),
            "--no-checkout".into(),
            "--".into(),
            source.to_string_lossy().into_owned(),
            checkout.to_string_lossy().into_owned(),
        ],
        &work,
        "clone.log",
    )?;
    if cloned != 0 {
        return Ok((cloned, None));
    }
    let detached = execute(
        vec![
            git,
            "-c".into(),
            "core.hooksPath=/dev/null".into(),
            "-C".into(),
            checkout.to_string_lossy().into_owned(),
            "checkout".into(),
            "--detach".into(),
            request.source_commit.clone(),
        ],
        &work,
        "checkout.log",
    )?;
    if detached != 0 {
        return Ok((detached, None));
    }
    let setup = execute(
        vec![
            crate::SHELL.into(),
            snapshot
                .join(crate::SETUP_ENTRY)
                .to_string_lossy()
                .into_owned(),
        ],
        &checkout,
        "setup.log",
    )?;
    if setup != 0 {
        return Ok((setup, None));
    }
    let checked = execute(
        vec![
            crate::SHELL.into(),
            snapshot
                .join(crate::CHECK_ENTRY)
                .to_string_lossy()
                .into_owned(),
        ],
        &checkout,
        "check.log",
    )?;
    Ok((setup, Some(checked)))
}

fn job(ctx: &crate::Ctx, dir: &Path, request: &Request, tools: &[StateValue], account: &Account) {
    let outcome = run(ctx, dir, request, tools, account);
    let Ok(_lock) = execution::state_lock(ctx) else {
        return;
    };
    // A failed process retirement retains its receipt and fixture for the
    // explicit stop owner. Never turn uncertainty into a finished receipt.
    if fsx::lexists(&dir.join("active-child.json")) || cleanup(ctx, dir).is_err() {
        return;
    }
    if fsx::lexists(&dir.join("stopped.json")) {
        return;
    }
    for name in ["clone.log", "checkout.log", "setup.log", "check.log"] {
        match fsx::open_ro(&dir.join(name), true) {
            Ok(file) if file.sync_all().is_ok() => {}
            Err(Error::Missing) => {}
            _ => return,
        }
    }
    let (setup, check) = outcome.unwrap_or((127, None));
    let finished = obj(vec![
        ("setup_exit", StateValue::Number(setup.to_string())),
        (
            "check_exit",
            check
                .map(|v| StateValue::Number(v.to_string()))
                .unwrap_or(StateValue::Null),
        ),
    ]);
    if fsx::write_new(
        &dir.join("finished.json"),
        crate::emit::dumps_default(&finished).as_bytes(),
        0o644,
    )
    .is_ok()
    {
        let _ = sync_dir(dir);
    }
}

pub(crate) fn launch(
    ctx: &crate::Ctx,
    dir: &Path,
    request: &Request,
    tools: &[StateValue],
    account: &Account,
) -> Result<(), Error> {
    record_fixture(ctx, dir, account)?;
    let mut gate = [0; 2];
    if unsafe { libc::pipe2(gate.as_mut_ptr(), libc::O_CLOEXEC) } != 0 {
        return Err(Error::io_msg("verification launch gate failed"));
    }
    let pid = unsafe { libc::fork() };
    if pid < 0 {
        unsafe {
            libc::close(gate[0]);
            libc::close(gate[1]);
        }
        return Err(Error::io_msg("verification fork failed"));
    }
    if pid == 0 {
        unsafe {
            for fd in 3..1024 {
                if fd != gate[0] {
                    libc::close(fd);
                }
            }
            let mut release = 0u8;
            let count = libc::read(gate[0], &mut release as *mut u8 as *mut libc::c_void, 1);
            libc::close(gate[0]);
            if count != 1 || release != 1 {
                libc::_exit(127);
            }
            libc::chdir(c"/".as_ptr());
            let null = libc::open(c"/dev/null".as_ptr(), libc::O_RDWR);
            if null < 0 {
                libc::_exit(127);
            }
            for fd in 0..3 {
                libc::dup2(null, fd);
            }
            if null > 2 {
                libc::close(null);
            }
        }
        job(ctx, dir, request, tools, account);
        unsafe {
            libc::_exit(0);
        }
    }
    unsafe {
        libc::close(gate[0]);
    }
    let admitted = (|| {
        if unsafe { libc::setpgid(pid, pid) } != 0 {
            return fail("verification group setup failed");
        }
        let (_, group, ticks) = execution::proc_identity(&Path::new("/proc").join(pid.to_string()));
        let ticks = ticks.ok_or_else(|| Error::fail("verification identity unavailable"))?;
        if group.as_deref() != Some(pid.to_string().as_str()) {
            return fail("verification group identity differs");
        }
        let started = obj(vec![
            ("pid", StateValue::Number(pid.to_string())),
            ("pgid", StateValue::Number(pid.to_string())),
            ("start_ticks", StateValue::Number(ticks.to_string())),
        ]);
        fsx::write_new(
            &dir.join("started.json"),
            crate::emit::dumps_default(&started).as_bytes(),
            0o644,
        )?;
        sync_dir(dir)?;
        let release = 1u8;
        if unsafe { libc::write(gate[1], &release as *const u8 as *const libc::c_void, 1) } != 1 {
            return fail("verification gate release failed");
        }
        Ok(())
    })();
    unsafe {
        libc::close(gate[1]);
    }
    // Before release the child can only observe EOF and exit. No commands
    // can run without the durable started identity and request above.
    admitted
}

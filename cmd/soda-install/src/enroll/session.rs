//! Enrollment session: unit publication, the service start, the result
//! wait, and the close that always runs, even after cancellation.

use std::ffi::CString;
use std::os::unix::io::{AsRawFd, FromRawFd};
use std::time::{Duration, Instant};

use crate::command::Runner;
use crate::console::Console;
use crate::errors::{self, Error};
use crate::fmtx::Arg;
use crate::signal::Ctx;

use super::arm::{
    enrollment_boot_seconds, enrollment_live_state, enrollment_write, EnrollmentAddress,
};
use super::keys::enrollment_safe_directory;
use super::{
    enrollment_client_command, enrollment_config, enrollment_socket_unit_config,
    enrollment_start_args, enrollment_template_unit_config, ENROLLMENT_CONFIG_PATH, ENROLLMENT_DIR,
    ENROLLMENT_SOCKET_UNIT, ENROLLMENT_TEMPLATE_UNIT, ENROLLMENT_UNIT, ENROLLMENT_UNIT_DIRECTORY,
};

pub struct EnrollmentSession<'a> {
    run: &'a dyn Runner,
    started: bool,
    published_units: Vec<String>,
}

impl<'a> EnrollmentSession<'a> {
    pub fn new(run: &'a dyn Runner) -> EnrollmentSession<'a> {
        EnrollmentSession {
            run,
            started: false,
            published_units: Vec::new(),
        }
    }

    fn stop_units(&self) -> Result<(), Error> {
        if !self.started {
            return Ok(());
        }
        let cleanup = Ctx::detached(Some(Instant::now() + Duration::from_secs(10)));
        // Stopping the socket first waits for its BindsTo/After connection
        // instances. Only these exact owned names are stopped, never a glob.
        let socket_err = self
            .run
            .run(
                &cleanup,
                "systemctl",
                &["stop".to_string(), ENROLLMENT_SOCKET_UNIT.to_string()],
                None,
            )
            .err();
        let stop_err = self
            .run
            .run(
                &cleanup,
                "systemctl",
                &["stop".to_string(), ENROLLMENT_UNIT.to_string()],
                None,
            )
            .err();
        let mut stopped = stop_err.is_none();
        if !stopped {
            // --collect may already have unloaded a successful unit. Confirm
            // both manager absence and cgroup removal before treating that
            // stop error as an already completed close.
            let load = self.run.run(
                &cleanup,
                "systemctl",
                &[
                    "show".to_string(),
                    "--property=LoadState".to_string(),
                    "--value".to_string(),
                    ENROLLMENT_UNIT.to_string(),
                ],
                None,
            );
            let group_gone = matches!(
                std::fs::symlink_metadata(format!("/sys/fs/cgroup/system.slice/{ENROLLMENT_UNIT}")),
                Err(err) if err.kind() == std::io::ErrorKind::NotFound
            );
            stopped = matches!(load, Ok(output) if String::from_utf8_lossy(&output).trim() == "not-found")
                && group_gone;
        }
        stopped = stopped && socket_err.is_none();
        if !stopped {
            return Err(Error::msg("enrollment close could not be confirmed; the five-minute native service limit still applies; preserve this attempt and inspect locally"));
        }
        Ok(())
    }

    fn cleanup_units(&mut self) -> Result<(), Error> {
        let mut result = Ok(());
        for name in &self.published_units {
            if std::fs::remove_file(format!("{ENROLLMENT_UNIT_DIRECTORY}/{name}")).is_err() {
                result = Err(Error::msg(
                    "owned enrollment unit cleanup failed; inspect locally before reopening",
                ));
            }
        }
        if !self.published_units.is_empty() {
            let cleanup = Ctx::detached(Some(Instant::now() + Duration::from_secs(10)));
            if self
                .run
                .run(&cleanup, "systemctl", &["daemon-reload".to_string()], None)
                .is_err()
            {
                result = Err(Error::msg(
                    "owned enrollment unit removal could not be reloaded; inspect locally before reopening",
                ));
            }
        }
        // Exact run-owned files only. Never remove another tree or stale evidence.
        for name in ["armed", "sshd_config", "receive.sock", "ready", "result"] {
            let _ = std::fs::remove_file(format!("{ENROLLMENT_DIR}/{name}"));
        }
        // Go's os.Remove removes the now-empty owned directory too.
        let _ = std::fs::remove_dir(ENROLLMENT_DIR);
        result
    }

    fn close(&mut self) -> Result<(), Error> {
        match self.stop_units() {
            Ok(()) => self.cleanup_units(),
            Err(err) => Err(err),
        }
    }
}

fn write_enrollment_config(
    ctx: &Ctx,
    run: &dyn Runner,
    selected: &EnrollmentAddress,
    enforcing: bool,
) -> Result<(), Error> {
    let now = enrollment_boot_seconds()?;
    enrollment_write(
        ENROLLMENT_DIR,
        "armed",
        &format!("{} {} {}\n", selected.name, selected.ip, now + 300),
    )?;
    enrollment_write(ENROLLMENT_DIR, "sshd_config", &enrollment_config())?;
    if run
        .run(
            ctx,
            "/usr/sbin/sshd",
            &[
                "-t".to_string(),
                "-f".to_string(),
                ENROLLMENT_CONFIG_PATH.to_string(),
            ],
            None,
        )
        .is_err()
    {
        return Err(Error::msg(
            "native SSH configuration check failed; no listener opened",
        ));
    }
    super::selinux::label_enrollment_config(ctx, run, enforcing)
}

fn publish_enrollment_units(
    ctx: &Ctx,
    session: &mut EnrollmentSession,
    ip: &str,
) -> Result<(), Error> {
    let path = CString::new(ENROLLMENT_UNIT_DIRECTORY).map_err(|_| Error::msg("invalid name"))?;
    let fd = unsafe {
        libc::open(
            path.as_ptr(),
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            0,
        )
    };
    if fd < 0 {
        return Err(Error::msg("native runtime unit directory unavailable"));
    }
    let dir = unsafe { std::os::unix::io::OwnedFd::from_raw_fd(fd) };
    enrollment_safe_directory(dir.as_raw_fd(), 0)?;
    let socket_config = enrollment_socket_unit_config(ip)?;
    for (name, contents) in [
        (ENROLLMENT_SOCKET_UNIT, socket_config),
        (ENROLLMENT_TEMPLATE_UNIT, enrollment_template_unit_config()),
    ] {
        if enrollment_write(ENROLLMENT_UNIT_DIRECTORY, name, &contents).is_err() {
            return Err(Error::msg(
                "runtime enrollment unit already exists or cannot be published; no replacement",
            ));
        }
        session.published_units.push(name.to_string());
    }
    if session
        .run
        .run(ctx, "systemctl", &["daemon-reload".to_string()], None)
        .is_err()
    {
        return Err(Error::msg("native enrollment units could not be loaded"));
    }
    Ok(())
}

fn start_enrollment_service(ctx: &Ctx, session: &mut EnrollmentSession) -> Result<(), Error> {
    // Mark before starting: a lost systemd-run reply is not permission to leave
    // a possibly started window running without the cleanup attempt.
    session.started = true;
    if session
        .run
        .run(ctx, "systemd-run", &enrollment_start_args(), None)
        .is_err()
    {
        return Err(Error::msg("temporary enrollment service could not start"));
    }
    Ok(())
}

fn publish_enrollment_state(
    ctx: &Ctx,
    session: &mut EnrollmentSession,
    selected: &EnrollmentAddress,
    enforcing: bool,
) -> Result<(), Error> {
    write_enrollment_config(ctx, session.run, selected, enforcing)?;
    publish_enrollment_units(ctx, session, &selected.ip)?;
    start_enrollment_service(ctx, session)
}

fn check_enrollment_result(console: &Console, ip: &str) -> Result<bool, Error> {
    let data = match crate::execute::read_regular(&format!("{ENROLLMENT_DIR}/result"), 128) {
        Ok(data) => data,
        Err(_) => return Ok(false),
    };
    if data == b"uncertain\n" {
        return Err(Error::EnrollUncertain);
    }
    if data != b"imported\n" {
        return Err(Error::msg(
            "key import failed; preserve existing access and inspect locally",
        ));
    }
    console.print("Public key installed. The import window is closing. Verify a NEW ordinary key-only SSH login from the laptop before continuing.", &[]);
    console.print(
        "Use the laptop private-key path matching the public .pub file you imported:",
        &[],
    );
    console.print("ssh -i ~/.ssh/id_ed25519 -o IdentitiesOnly=yes -o PreferredAuthentications=publickey -o PasswordAuthentication=no -o KbdInteractiveAuthentication=no -o ControlMaster=no -o ControlPath=none root@%s", &[Arg::Str(ip)]);
    Ok(true)
}

fn notify_enrollment_ready(console: &Console, selected: &EnrollmentAddress, ready: &mut bool) {
    if !*ready && std::fs::symlink_metadata(format!("{ENROLLMENT_DIR}/ready")).is_ok() {
        *ready = true;
        let laptop = enrollment_client_command(&selected.ip).unwrap_or_default();
        console.print(
            "On the laptop, check the host fingerprint above, then run (adjust only your PUBLIC .pub file path):\n%s",
            &[Arg::Str(&laptop)],
        );
        console.print(
            "Waiting for one public key. Press Enter to cancel; do not close this console while importing.",
            &[],
        );
    }
}

fn poll_console_cancel(console: &Console, cancel_ctx: &Ctx) -> Result<(), Error> {
    // A short poll keeps cancellation responsive without abandoning a reader
    // goroutine that could consume a subsequent wizard answer.
    let mut fd = libc::pollfd {
        fd: console.tty_fd(),
        events: libc::POLLIN,
        revents: 0,
    };
    if unsafe { libc::poll(&mut fd, 1, 200) } < 0 {
        let errno = unsafe { *libc::__errno_location() };
        if errno != libc::EINTR {
            return Err(errors::os_error(std::io::Error::from_raw_os_error(errno)));
        }
    }
    if fd.revents & (libc::POLLHUP | libc::POLLERR | libc::POLLNVAL) != 0 {
        return Err(Error::msg("local console disconnected; closing enrollment"));
    }
    if fd.revents & libc::POLLIN != 0 {
        let _ = console.line(cancel_ctx);
        return Err(Error::msg("key enrollment cancelled locally"));
    }
    Ok(())
}

fn check_enrollment_service_active(ctx: &Ctx, run: &dyn Runner) -> Result<(), Error> {
    let state = run.run(
        ctx,
        "systemctl",
        &[
            "show".to_string(),
            "--property=ActiveState".to_string(),
            "--value".to_string(),
            ENROLLMENT_UNIT.to_string(),
        ],
        None,
    );
    let active = match state {
        Ok(output) => String::from_utf8_lossy(&output).trim().to_string(),
        Err(_) => String::new(),
    };
    if active != "active" && active != "activating" {
        // The receipt may have appeared since the start of this loop.
        if std::fs::symlink_metadata(format!("{ENROLLMENT_DIR}/result")).is_ok() {
            return Ok(());
        }
        return Err(Error::msg("enrollment service closed without a receipt; a key may have been imported; verify native access locally before trying another import"));
    }
    Ok(())
}

fn wait_enrollment_result(
    ctx: &Ctx,
    console: &Console,
    run: &dyn Runner,
    selected: &EnrollmentAddress,
) -> Result<(), Error> {
    let wait_ctx = ctx.with_timeout(Instant::now() + Duration::from_secs(300));
    let mut ready = false;
    let mut last_state_check: Option<Instant> = None;
    loop {
        if check_enrollment_result(console, &selected.ip)? {
            return Ok(());
        }
        if wait_ctx.err().is_some() {
            return Err(Error::msg("key enrollment closed without confirmed import"));
        }
        if enrollment_live_state().is_err() {
            return Err(Error::msg("key enrollment window expired; closing it now"));
        }
        notify_enrollment_ready(console, selected, &mut ready);
        poll_console_cancel(console, &wait_ctx)?;
        let due = match last_state_check {
            None => true,
            Some(last) => last.elapsed() >= Duration::from_secs(1),
        };
        if due {
            last_state_check = Some(Instant::now());
            check_enrollment_service_active(&wait_ctx, run)?;
        }
    }
}

pub fn arm_enrollment(ctx: &Ctx, console: &Console, run: &dyn Runner) -> Result<(), Error> {
    let selected = super::arm::select_enrollment_target(ctx, console, run)?;
    // The admission helper reports enforcement as success; enrollment
    // branches on it instead of refusing to run.
    let enforcing = crate::hostadmit::selinux_enforcing().is_ok();
    super::selinux::ensure_enrollment_port_label(ctx, run, enforcing)?;
    super::arm::guard_existing_enrollment_state(ctx, run)?;
    let mut session = EnrollmentSession::new(run);
    let result = (|| -> Result<(), Error> {
        publish_enrollment_state(ctx, &mut session, &selected, enforcing)?;
        wait_enrollment_result(ctx, console, run, &selected)
    })();
    session.close()?;
    result
}

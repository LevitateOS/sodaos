//! Enrollment broker: the fixed service action that accepts one bounded
//! key over the private Unix socket and commits it to root's
//! authorized_keys, then closes the window.

use std::os::unix::io::AsRawFd;
use std::os::unix::net::{UnixListener, UnixStream};
use std::time::{Duration, Instant};

use crate::command::Runner;
use crate::errors::{self, Error};
use crate::signal::Ctx;

use super::arm::{enrollment_live_address, enrollment_state, enrollment_write};
use super::keys::{append_enrollment_key, enrollment_root_home};
use super::{
    enrollment_connection_unit, enrollment_public_key, enrollment_receiver_unit, read_bounded,
    ENROLLMENT_DIR, ENROLLMENT_KEY_LIMIT, ENROLLMENT_SOCKET, ENROLLMENT_SOCKET_UNIT, ENROLLMENT_UNIT,
};

fn validate_enrollment_server_environment() -> Result<Duration, Error> {
    let expected = format!("0::/system.slice/{ENROLLMENT_UNIT}");
    match std::fs::read("/proc/self/cgroup") {
        Ok(group) if String::from_utf8_lossy(&group).trim() == expected => {}
        _ => return Err(Error::msg("enrollment server requires its fixed native transient service")),
    }
    let (selected, remaining) = enrollment_state()?;
    if !enrollment_live_address(&selected) {
        return Err(Error::msg("selected private address is no longer live"));
    }
    Ok(remaining)
}

fn start_enrollment_broker(phase: &Ctx, run: &dyn Runner) -> Result<UnixListener, Error> {
    let broker = UnixListener::bind(ENROLLMENT_SOCKET).map_err(|e| {
        Error::msg(format!(
            "listen unix {ENROLLMENT_SOCKET}: bind: {}",
            errors::errno_text(e.raw_os_error().unwrap_or(0))
        ))
    })?;
    use std::os::unix::fs::PermissionsExt;
    if std::fs::set_permissions(ENROLLMENT_SOCKET, std::fs::Permissions::from_mode(0o600)).is_err() {
        let err = std::io::Error::last_os_error();
        return Err(errors::path_error("chmod", ENROLLMENT_SOCKET, err));
    }
    if run.run(
        phase,
        "systemctl",
        &["start".to_string(), ENROLLMENT_SOCKET_UNIT.to_string()],
        None,
    )
    .is_err()
    {
        return Err(Error::msg("native key-import socket could not start"));
    }
    enrollment_write(ENROLLMENT_DIR, "ready", "ready\n")?;
    Ok(broker)
}

fn set_enrollment_connection_deadline(phase: &Ctx, stream: &UnixStream) {
    let mut deadline = Instant::now() + Duration::from_secs(15);
    if let Some(window) = phase.deadline() {
        if window < deadline {
            deadline = window;
        }
    }
    let timeout = deadline.saturating_duration_since(Instant::now());
    let _ = stream.set_read_timeout(Some(timeout));
    let _ = stream.set_write_timeout(Some(timeout));
}

fn read_enrollment_key_from_connection(
    phase: &Ctx,
    stream: &UnixStream,
) -> Result<Option<String>, Error> {
    set_enrollment_connection_deadline(phase, stream);
    let unit = match enrollment_connection_unit(stream) {
        Ok(unit) => unit,
        Err(_) => return Ok(None),
    };
    if !enrollment_receiver_unit(&unit) {
        return Ok(None);
    }
    let key = (|| -> Result<String, Error> {
        let mut reader = stream;
        let data = read_bounded(&mut reader, ENROLLMENT_KEY_LIMIT)?;
        enrollment_public_key(&data)
    })();
    let key = match key {
        Ok(key) => key,
        Err(_) => {
            use std::io::Write;
            let _ = stream.write_all(b"refused\n");
            return Ok(None);
        }
    };
    if let Some(err) = phase.err() {
        return Err(err);
    }
    if let Err(err) = enrollment_state() {
        return Err(err);
    }
    Ok(Some(key))
}

fn enrollment_commit_status(err: Option<&Error>) -> &'static str {
    match err {
        None => "imported\n",
        Some(Error::EnrollUncertain) => "uncertain\n",
        Some(_) => "failed\n",
    }
}

fn commit_enrollment_key(
    phase: &Ctx,
    stream: &UnixStream,
    key: &str,
    run: &dyn Runner,
) -> Result<(), Error> {
    let mut err: Option<Error> = None;
    let mut home = String::new();
    match enrollment_root_home() {
        Ok(resolved) => {
            home = resolved.clone();
            if let Err(e) = append_enrollment_key(phase, &resolved, key, 0) {
                err = Some(e);
            }
        }
        Err(e) => err = Some(e),
    }
    if err.is_none() {
        let ssh = format!("{home}/.ssh");
        let keys = format!("{home}/.ssh/authorized_keys");
        if run.run(phase, "/usr/sbin/restorecon", &["--".to_string(), ssh, keys], None).is_err() {
            err = Some(Error::EnrollUncertain);
        }
    }
    let status = enrollment_commit_status(err.as_ref());
    if let Err(write_err) = enrollment_write(ENROLLMENT_DIR, "result", status) {
        err = Some(write_err);
    }
    {
        use std::io::Write;
        let _ = stream.write_all(status.as_bytes());
    }
    match err {
        Some(err) => Err(err),
        None => Ok(()),
    }
}

fn serve_enrollment_loop(
    phase: &Ctx,
    broker: &UnixListener,
    run: &dyn Runner,
) -> Result<(), Error> {
    // Go accepts in a goroutine and selects over phase expiry, a one-second
    // clock tick, and incoming requests. This port merges that into one
    // loop: poll the listener with the one-second clock tick; phase expiry,
    // the armed-window check, and request handling keep Go's order and
    // errors. A blocked Rust accept cannot be woken by closing the
    // listener, so the listener stays non-blocking under the poll.
    broker.set_nonblocking(true).map_err(errors::os_error)?;
    loop {
        if phase.err().is_some() {
            return Err(Error::msg("enrollment window closed"));
        }
        let mut fd = libc::pollfd { fd: broker.as_raw_fd(), events: libc::POLLIN, revents: 0 };
        if unsafe { libc::poll(&mut fd, 1, 1000) } < 0 {
            let errno = unsafe { *libc::__errno_location() };
            if errno != libc::EINTR {
                return Err(errors::os_error(std::io::Error::from_raw_os_error(errno)));
            }
        }
        if phase.err().is_some() {
            return Err(Error::msg("enrollment window closed"));
        }
        if fd.revents & libc::POLLIN == 0 {
            enrollment_state()?;
            continue;
        }
        let (stream, _) = match broker.accept() {
            Ok(accepted) => accepted,
            Err(_) => continue,
        };
        match read_enrollment_key_from_connection(phase, &stream)? {
            None => continue,
            Some(key) => return commit_enrollment_key(phase, &stream, &key, run),
        }
    }
}

// ServeEnrollment is an internal fixed systemd service action, not an arming
// entrypoint. Run dispatches it without requesting a terminal, after root and
// installed-CoreOS checks. The cgroup check prevents accidental direct execution.
pub fn serve_enrollment(ctx: &Ctx, run: &dyn Runner) -> Result<(), Error> {
    let remaining = validate_enrollment_server_environment()?;
    let phase = ctx.with_timeout(Instant::now() + remaining);
    let broker = start_enrollment_broker(&phase, run)?;
    serve_enrollment_loop(&phase, &broker, run)
}

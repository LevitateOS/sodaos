//! Enrollment receiver: the sole sshd ForceCommand. Client
//! commands/subsystems are refused, not interpreted. It never handles the
//! native password or a private key.

use std::os::unix::net::UnixStream;
use std::time::{Duration, Instant};

use crate::errors::{self, Error};
use crate::signal::Ctx;

use super::arm::enrollment_state;
use super::{
    enrollment_connection_unit, enrollment_public_key, ENROLLMENT_PORT, ENROLLMENT_SOCKET,
    ENROLLMENT_UNIT,
};

fn admit_receive_enrollment() -> Result<Duration, Error> {
    let present = |name: &str| std::env::var_os(name).is_some_and(|v| !v.is_empty());
    if unsafe { libc::geteuid() } != 0 || present("SSH_ORIGINAL_COMMAND") || present("SSH_TTY") {
        return Err(Error::msg("only public-key stdin enrollment is allowed"));
    }
    let (selected, remaining) = enrollment_state()?;
    let connection = std::env::var_os("SSH_CONNECTION");
    let text = connection
        .as_deref()
        .map(|v| v.to_string_lossy())
        .unwrap_or_default();
    let fields: Vec<&str> = text.split_whitespace().collect();
    if fields.len() != 4 || fields[2] != selected.ip || fields[3] != ENROLLMENT_PORT {
        return Err(Error::msg("dedicated key-import SSH connection required"));
    }
    if remaining > Duration::from_secs(30) {
        return Ok(Duration::from_secs(30));
    }
    Ok(remaining)
}

fn confirm_enrollment_import(result: &[u8], err: Option<Error>) -> Result<(), Error> {
    if err.is_none() && result == b"uncertain\n" {
        return Err(Error::EnrollUncertain);
    }
    if err.is_some() || result != b"imported\n" {
        return Err(Error::msg(
            "key import was not confirmed; inspect the local console before retrying",
        ));
    }
    Ok(())
}

fn submit_enrollment_key(ctx: &Ctx, key: &str) -> Result<(), Error> {
    let mut connection = UnixStream::connect(ENROLLMENT_SOCKET)
        .map_err(|_| Error::msg("key-import window is closed"))?;
    if let Some(deadline) = ctx.deadline() {
        let timeout = deadline.saturating_duration_since(Instant::now());
        let _ = connection.set_read_timeout(Some(timeout));
        let _ = connection.set_write_timeout(Some(timeout));
    }
    match enrollment_connection_unit(&connection) {
        Ok(unit) if unit == ENROLLMENT_UNIT.as_bytes() => {}
        _ => return Err(Error::msg("dedicated enrollment broker required")),
    }
    {
        use std::io::Write;
        if connection.write(format!("{key}\n").as_bytes()).is_err() {
            let err = std::io::Error::last_os_error();
            return Err(Error::msg(format!(
                "write unix {ENROLLMENT_SOCKET}: {}",
                errors::errno_text(err.raw_os_error().unwrap_or(0))
            )));
        }
    }
    if connection.shutdown(std::net::Shutdown::Write).is_err() {
        let err = std::io::Error::last_os_error();
        return Err(Error::msg(format!(
            "close unix {ENROLLMENT_SOCKET}: {}",
            errors::errno_text(err.raw_os_error().unwrap_or(0))
        )));
    }
    let mut result = Vec::new();
    let read_err = {
        use std::io::Read;
        match (&connection).take(129).read_to_end(&mut result) {
            Ok(_) => None,
            Err(err) => Some(errors::os_error(err)),
        }
    };
    confirm_enrollment_import(&result, read_err)
}

// ReceiveEnrollment is the sole ForceCommand. Client commands/subsystems are
// refused, not interpreted. It never handles the native password or a private key.
pub fn receive_enrollment(ctx: &Ctx) -> Result<(), Error> {
    let remaining = admit_receive_enrollment()?;
    let phase = ctx.with_timeout(Instant::now() + remaining);
    let data = enrollment_stdin(&phase)?;
    let key = enrollment_public_key(&data)?;
    submit_enrollment_key(&phase, &key)?;
    {
        use std::io::Write;
        let _ = writeln!(
            std::io::stdout(),
            "Public key imported. Verify a fresh ordinary key-only SSH login."
        );
    }
    Ok(())
}

fn enrollment_stdin(ctx: &Ctx) -> Result<Vec<u8>, Error> {
    let mut data = Vec::new();
    while data.len() <= super::ENROLLMENT_KEY_LIMIT {
        if let Some(err) = ctx.err() {
            return Err(err);
        }
        let mut fd = libc::pollfd {
            fd: 0,
            events: libc::POLLIN,
            revents: 0,
        };
        if unsafe { libc::poll(&mut fd, 1, 100) } < 0 {
            let errno = unsafe { *libc::__errno_location() };
            if errno != libc::EINTR {
                return Err(errors::os_error(std::io::Error::from_raw_os_error(errno)));
            }
        }
        if fd.revents & (libc::POLLERR | libc::POLLNVAL) != 0 {
            return Err(Error::msg("public-key input failed"));
        }
        if fd.revents & (libc::POLLIN | libc::POLLHUP) == 0 {
            continue;
        }
        let mut buffer = [0u8; 1024];
        let n = loop {
            let n =
                unsafe { libc::read(0, buffer.as_mut_ptr() as *mut libc::c_void, buffer.len()) };
            if n < 0 {
                let errno = unsafe { *libc::__errno_location() };
                if errno == libc::EINTR {
                    continue;
                }
                return Err(errors::os_error(std::io::Error::from_raw_os_error(errno)));
            }
            break n as usize;
        };
        data.extend_from_slice(&buffer[..n]);
        if n == 0 {
            return Ok(data);
        }
    }
    Err(Error::msg("public-key input exceeds limit"))
}

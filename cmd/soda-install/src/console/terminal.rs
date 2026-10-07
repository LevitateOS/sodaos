#[cfg(test)]
use std::fs::OpenOptions;
use std::io::{Read, Write};
use std::os::unix::io::AsRawFd;

use crate::errors::Error;
use crate::signal::Ctx;

use super::{errno_text, trim_space_bytes, Console};

impl Console {
    #[cfg(test)]
    pub fn open(path: &str) -> Result<Console, Error> {
        let tty = OpenOptions::new()
            .read(true)
            .write(true)
            .open(path)
            .map_err(|e| Error::msg(format!("cannot open operator terminal: {e}")))?;
        Ok(Console {
            tty_path: path.to_string(),
            tty,
        })
    }

    pub(super) fn fd(&self) -> libc::c_int {
        self.tty.as_raw_fd()
    }

    /// Raw terminal fd for enrollment's cancel poll.
    pub fn tty_fd(&self) -> libc::c_int {
        self.tty.as_raw_fd()
    }

    /// Write one line to the operator terminal; write failures are ignored.
    pub fn print(&self, message: impl std::fmt::Display) {
        let mut text = message.to_string();
        text.push('\n');
        let _ = self.tty_write(text.as_bytes());
    }

    fn tty_write(&self, bytes: &[u8]) -> Result<(), Error> {
        let mut view = bytes;
        // Go writes through unbuffered files; short writes retry.
        let mut tty = &self.tty;
        while !view.is_empty() {
            match tty.write(view) {
                Ok(0) => return Err(Error::msg(format!("write {}: short write", self.tty_path))),
                Ok(n) => view = &view[n..],
                Err(e) if e.kind() == std::io::ErrorKind::Interrupted => {}
                Err(e) => {
                    return Err(Error::msg(format!(
                        "write {}: {}",
                        self.tty_path,
                        errno_text(e)
                    )))
                }
            }
        }
        Ok(())
    }

    pub fn page(&self, title: &str) {
        self.print("\x1b[0m\x1b[2J\x1b[HSodaOS installation");
        self.print("");
        self.print(title);
        self.print("");
    }

    pub fn ask(&self, ctx: &Ctx, prompt: &str) -> Result<String, Error> {
        let _ = self.tty_write(format!("{prompt}: ").as_bytes());
        self.line(ctx)
    }

    fn poll(&self, timeout_ms: libc::c_int) -> Result<bool, Error> {
        let mut fd = libc::pollfd {
            fd: self.fd(),
            events: libc::POLLIN,
            revents: 0,
        };
        let result = unsafe { libc::poll(&mut fd, 1, timeout_ms) };
        if result < 0 {
            let errno = unsafe { *libc::__errno_location() };
            if errno != libc::EINTR {
                // Go returns the raw errno text, lowercase like its table.
                return Err(Error::msg(errno_text(std::io::Error::from_raw_os_error(
                    errno,
                ))));
            }
            return Ok(false);
        }
        if fd.revents & (libc::POLLHUP | libc::POLLERR | libc::POLLNVAL) != 0 {
            return Err(Error::msg("terminal disconnected"));
        }
        Ok(fd.revents & libc::POLLIN != 0)
    }

    fn read_byte(&self) -> Result<Option<u8>, Error> {
        let mut byte = [0u8; 1];
        let mut tty = &self.tty;
        loop {
            match tty.read(&mut byte) {
                Ok(0) => return Ok(None),
                Ok(_) => return Ok(Some(byte[0])),
                Err(e) if e.kind() == std::io::ErrorKind::Interrupted => {}
                Err(e) => {
                    return Err(Error::msg(format!(
                        "read {}: {}",
                        self.tty_path,
                        errno_text(e)
                    )))
                }
            }
        }
    }

    /// Go `line`: one trimmed line, refusing control bytes and EOF.
    pub fn line(&self, ctx: &Ctx) -> Result<String, Error> {
        let mut data = Vec::new();
        loop {
            if data.len() > 16384 {
                return Err(Error::msg("input exceeds limit"));
            }
            if let Some(err) = ctx.err() {
                return Err(err);
            }
            if !self.poll(100)? {
                continue;
            }
            match self.read_byte()? {
                None => return Err(Error::msg("EOF")),
                Some(b'\n') => {
                    // Byte-exact Go trim: invalid UTF-8 can never match an
                    // ASCII prompt, so lossy conversion stays equivalent.
                    return Ok(String::from_utf8_lossy(trim_space_bytes(&data)).to_string());
                }
                Some(b) if b < 32 && b != b'\t' => {
                    return Err(Error::msg("control character refused"));
                }
                Some(b) => data.push(b),
            }
        }
    }

    fn hide_echo(&self) -> Result<libc::termios, Error> {
        let mut state: libc::termios = unsafe { std::mem::zeroed() };
        if unsafe { libc::ioctl(self.fd(), libc::TCGETS, &mut state) } != 0 {
            return Err(Error::msg("password entry requires a terminal"));
        }
        let mut hidden = state;
        hidden.c_lflag &= !(libc::ECHO | libc::ECHONL);
        if unsafe { libc::ioctl(self.fd(), libc::TCSETS, &hidden) } != 0 {
            let errno = unsafe { *libc::__errno_location() };
            return Err(Error::msg(errno_text(std::io::Error::from_raw_os_error(
                errno,
            ))));
        }
        Ok(state)
    }

    fn restore_echo(&self, state: &libc::termios) {
        // TCSETSF flushes queued private input so it never becomes input to
        // a later prompt, exactly like the Go defer.
        unsafe {
            libc::ioctl(self.fd(), libc::TCSETSF, state);
        }
    }

    /// Go `secret`: hidden password entry without trimming. Raw bytes are
    /// returned, like Go's string: invalid UTF-8 stays observable so the
    /// password/token validators can refuse it exactly like Go.
    pub fn secret(&self, ctx: &Ctx, prompt: &str) -> Result<Vec<u8>, Error> {
        let state = self.hide_echo()?;
        let _ = self.tty_write(format!("{prompt}: ").as_bytes());
        let mut data = Vec::new();
        let result = loop {
            if data.len() > 1024 {
                break Err(Error::msg("password exceeds limit"));
            }
            if let Some(err) = ctx.err() {
                break Err(err);
            }
            // Go additionally drains a signal mailbox here, but every
            // secret() scope is already cancelled by those same signals
            // through its context, so the mailbox only fires in a delivery
            // race this synchronous port cannot observe.
            let readable = match self.poll(100) {
                Ok(readable) => readable,
                Err(err) => break Err(err),
            };
            if !readable {
                continue;
            }
            match self.read_byte() {
                Ok(Some(b'\n')) => {
                    self.print("");
                    break Ok(data);
                }
                Ok(Some(b)) if b < 32 || b == 127 => {
                    break Err(Error::msg("password contains control characters"));
                }
                Ok(Some(b)) => data.push(b),
                Ok(None) | Err(_) => break Err(Error::msg("password input ended")),
            }
        };
        self.restore_echo(&state);
        result
    }
}

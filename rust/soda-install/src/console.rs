//! Operator console: prompting, line/password entry with terminal echo
//! control, and the DHCP/`nmtui` network step.

use std::fs::{File, OpenOptions};
use std::io::{Read, Write};
use std::os::unix::io::AsRawFd;
use std::os::unix::process::ExitStatusExt;

use crate::command::Runner;
use crate::errors::Error;
use crate::fmtx::{go_lower, sprintf, Arg};
use crate::signal::Ctx;

pub struct Console {
    tty_path: String,
    tty: File,
}

impl Console {
    pub fn open(path: &str) -> Result<Console, Error> {
        let tty = OpenOptions::new()
            .read(true)
            .write(true)
            .open(path)
            .map_err(|e| Error::msg(format!("cannot open operator terminal: {e}")))?;
        Ok(Console { tty_path: path.to_string(), tty })
    }

    fn fd(&self) -> libc::c_int {
        self.tty.as_raw_fd()
    }

    /// Raw terminal fd for enrollment's cancel poll.
    pub fn tty_fd(&self) -> libc::c_int {
        self.tty.as_raw_fd()
    }

    /// Go `print`: format plus newline; write failures are ignored.
    pub fn print(&self, format: &str, args: &[Arg<'_>]) {
        let mut text = sprintf(format, args);
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
                Err(e) => return Err(Error::msg(format!("write {}: {}", self.tty_path, errno_text(e)))),
            }
        }
        Ok(())
    }

    pub fn page(&self, title: &str) {
        self.print("\x1b[0m\x1b[2J\x1b[HSodaOS installation", &[]);
        self.print("", &[]);
        self.print(title, &[]);
        self.print("", &[]);
    }

    pub fn ask(&self, ctx: &Ctx, prompt: &str) -> Result<String, Error> {
        let _ = self.tty_write(format!("{prompt}: ").as_bytes());
        self.line(ctx)
    }

    fn poll(&self, timeout_ms: libc::c_int) -> Result<bool, Error> {
        let mut fd = libc::pollfd { fd: self.fd(), events: libc::POLLIN, revents: 0 };
        let result = unsafe { libc::poll(&mut fd, 1, timeout_ms) };
        if result < 0 {
            let errno = unsafe { *libc::__errno_location() };
            if errno != libc::EINTR {
                // Go returns the raw errno text, lowercase like its table.
                return Err(Error::msg(errno_text(std::io::Error::from_raw_os_error(errno))));
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
                Err(e) => return Err(Error::msg(format!("read {}: {}", self.tty_path, errno_text(e)))),
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
            return Err(Error::msg(errno_text(std::io::Error::from_raw_os_error(errno))));
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
                    self.print("", &[]);
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

    pub fn choose_network_action(&self, ctx: &Ctx, open_editor: bool) -> Result<String, Error> {
        if open_editor {
            return Ok("edit".to_string());
        }
        self.ask(ctx, "Type keep, edit, back, restart, or cancel")
    }

    pub fn run_network_editor(&self, ctx: &Ctx) -> String {
        // Go CommandContext semantics: SIGKILL on cancellation, no timeout.
        let mut child = match std::process::Command::new("nmtui")
            .stdin(self.reopen_stdio())
            .stdout(self.reopen_stdio())
            .stderr(self.reopen_stdio())
            .spawn()
        {
            Ok(child) => child,
            Err(_) => {
                self.page("Step 1 of 5 — Network");
                return "NetworkManager editor failed. No disk installation started.".to_string();
            }
        };
        let pid = child.id() as libc::pid_t;
        let mut status_code: libc::c_int = 0;
        let reaped = loop {
            if ctx.err().is_some() {
                let _ = child.kill();
            }
            let waited = unsafe { libc::waitpid(pid, &mut status_code, libc::WNOHANG) };
            if waited == pid {
                break true;
            }
            if waited < 0 {
                let errno = unsafe { *libc::__errno_location() };
                if errno != libc::EINTR {
                    break false;
                }
            }
            std::thread::sleep(std::time::Duration::from_millis(5));
        };
        let status = if reaped {
            std::process::ExitStatus::from_raw(status_code)
        } else {
            child.wait().unwrap_or_else(|_| std::process::ExitStatus::from_raw(1 << 8))
        };
        self.page("Step 1 of 5 — Network");
        if !status.success() {
            return "NetworkManager editor failed. No disk installation started.".to_string();
        }
        String::new()
    }

    /// Duplicate the terminal fd for child stdio, like Go sharing `c.tty`.
    fn reopen_stdio(&self) -> std::process::Stdio {
        let fd = unsafe { libc::dup(self.fd()) };
        if fd < 0 {
            return std::process::Stdio::null();
        }
        unsafe { std::os::unix::io::FromRawFd::from_raw_fd(fd) }
    }

    /// Console over an already-open terminal or pipe.
    pub fn from_file(name: &str, tty: File) -> Console {
        Console { tty_path: name.to_string(), tty }
    }

    pub fn confirm_live_addresses(&self, ctx: &Ctx) -> Result<bool, Error> {
        loop {
            let answer = self.ask(ctx, "Type yes to use them, edit, back, restart, or cancel")?;
            match go_lower(&answer).as_str() {
                "yes" => return Ok(false),
                "edit" => return Ok(true),
                "back" => return Err(Error::Back),
                "restart" => return Err(Error::Restart),
                "cancel" => return Err(Error::Cancel),
                _ => self.print("Choose yes, edit, back, restart, or cancel.", &[]),
            }
        }
    }

    pub fn print_live_addresses(&self, data: &[u8]) {
        self.print("", &[]);
        self.print("Current live addresses:", &[]);
        // Quote native output bytes so a configured interface name cannot
        // inject terminal controls; invalid UTF-8 quotes per byte like Go.
        for line in trim_space_bytes(data).split(|b| *b == b'\n') {
            self.print("  %q", &[Arg::Bytes(line)]);
        }
    }

    pub fn apply_network_choice(&self, ctx: &Ctx, choice: &str) -> Result<(String, bool), Error> {
        match go_lower(choice).as_str() {
            "back" => Err(Error::Back),
            "restart" => Err(Error::Restart),
            "cancel" => Err(Error::Cancel),
            "edit" => {
                let feedback = self.run_network_editor(ctx);
                if !feedback.is_empty() {
                    return Ok((feedback, true));
                }
                Ok((String::new(), false))
            }
            "keep" => Ok((String::new(), false)),
            _ => Ok(("Choose keep, edit, back, restart, or cancel.".to_string(), true)),
        }
    }

    pub fn inspect_and_confirm_network(
        &self,
        ctx: &Ctx,
        run: &dyn Runner,
    ) -> Result<(bool, String), Error> {
        let data = match run.run(ctx, "ip", &["-brief".to_string(), "address".to_string()], None) {
            Ok(data) => data,
            Err(_) => return Ok((false, "Could not inspect live network addresses.".to_string())),
        };
        self.print_live_addresses(&data);
        let edit = self.confirm_live_addresses(ctx)?;
        Ok((edit, String::new()))
    }

    pub fn network_with(&self, ctx: &Ctx, run: &dyn Runner) -> Result<(), Error> {
        let mut feedback = String::new();
        let mut open_editor = false;
        loop {
            self.page("Step 1 of 5 — Network");
            if !feedback.is_empty() {
                self.print("%s", &[Arg::Str(&feedback)]);
                self.print("", &[]);
                feedback.clear();
            }
            self.print("DHCP is ready by default.", &[]);
            self.print("Use nmtui to set a static address, gateway, or DNS.", &[]);
            self.print("The installed system will receive the reviewed live settings.", &[]);
            let choice = self.choose_network_action(ctx, open_editor)?;
            open_editor = false;
            let (next, retry) = self.apply_network_choice(ctx, &choice)?;
            if retry {
                feedback = next;
                continue;
            }
            let (edit, inspect_feedback) = self.inspect_and_confirm_network(ctx, run)?;
            if !inspect_feedback.is_empty() {
                feedback = inspect_feedback;
                continue;
            }
            if !edit {
                return Ok(());
            }
            open_editor = true;
        }
    }
}

/// Length of one Go `unicode.IsSpace` rune opening the byte string, or
/// zero. Only exact UTF-8 sequences match; invalid bytes stop the trim,
/// exactly like Go decoding `RuneError` (never space).
fn go_space_len(prefix: &[u8]) -> usize {
    if prefix.is_empty() {
        return 0;
    }
    match prefix[0] {
        b'\t' | b'\n' | 0x0b | 0x0c | b'\r' | b' ' => return 1,
        0xc2 if prefix.len() >= 2 && (prefix[1] == 0x85 || prefix[1] == 0xa0) => return 2,
        0xe1 if prefix.len() >= 3 && prefix[1] == 0x9a && prefix[2] == 0x80 => return 3, // U+1680
        0xe2 if prefix.len() >= 3 && prefix[1] == 0x80 && (prefix[2] <= 0x8a || prefix[2] == 0xa8 || prefix[2] == 0xa9 || prefix[2] == 0xaf) => {
            return 3; // U+2000..U+200A, U+2028, U+2029, U+202F
        }
        0xe2 if prefix.len() >= 3 && prefix[1] == 0x81 && prefix[2] == 0x9f => return 3, // U+205F
        0xe3 if prefix.len() >= 3 && prefix[1] == 0x80 && prefix[2] == 0x80 => return 3, // U+3000
        _ => {}
    }
    0
}

fn trim_space_bytes(data: &[u8]) -> &[u8] {
    let mut start = 0;
    while start < data.len() {
        let len = go_space_len(&data[start..]);
        if len == 0 {
            break;
        }
        start += len;
    }
    let mut end = data.len();
    while end > start {
        // A trailing space run ends `end`; find its opening boundary by
        // scanning the longest candidate suffix first.
        let mut len = 0;
        for candidate in [3usize, 2, 1] {
            if end >= candidate && start <= end - candidate && go_space_len(&data[end - candidate..end]) == candidate {
                // The candidate must align with a rune boundary: it opens
                // either at `start` or right after a non-space byte run.
                // Mid-rune splits never match `go_space_len`, except a
                // 1-byte ASCII match inside a multibyte sequence, which
                // ASCII bytes cannot start.
                len = candidate;
                break;
            }
        }
        // ASCII spaces are single bytes, so a 1-byte match always aligns;
        // multibyte matches align because continuation bytes never match a
        // space opening.
        if len == 0 {
            break;
        }
        end -= len;
    }
    &data[start..end]
}

fn errno_name(errno: libc::c_int) -> String {
    unsafe {
        let text = libc::strerror(errno);
        if text.is_null() {
            return format!("errno {errno}");
        }
        String::from_utf8_lossy(std::ffi::CStr::from_ptr(text).to_bytes()).into_owned()
    }
}

fn errno_text(err: std::io::Error) -> String {
    match err.raw_os_error() {
        Some(errno) => {
            // Go spells errno text lowercase ("input/output error").
            let text = errno_name(errno);
            let mut chars = text.chars();
            match chars.next() {
                Some(first) => first.to_lowercase().collect::<String>() + chars.as_str(),
                None => text,
            }
        }
        None => err.to_string(),
    }
}

#[cfg(test)]
pub mod test_support {
    use std::fs::{File, OpenOptions};
    use std::io::{Read, Write};
    use std::os::unix::io::{AsRawFd, FromRawFd};

    pub struct Pty {
        pub master: File,
        pub slave_path: String,
    }

    pub fn open_pty() -> Pty {
        let mut master = 0;
        let mut slave = 0;
        assert_eq!(unsafe { libc::openpty(&mut master, &mut slave, std::ptr::null_mut(), std::ptr::null_mut(), std::ptr::null_mut()) }, 0);
        // ptsname_r: ptsname's static buffer races across test threads.
        let mut name = [0 as libc::c_char; 64];
        assert_eq!(unsafe { libc::ptsname_r(master, name.as_mut_ptr(), name.len()) }, 0);
        let slave_path = unsafe { std::ffi::CStr::from_ptr(name.as_ptr()).to_string_lossy().into_owned() };
        unsafe { libc::close(slave) };
        Pty { master: unsafe { File::from_raw_fd(master) }, slave_path }
    }

    fn poll_read(master: &mut File, timeout_ms: i32) -> Option<Vec<u8>> {
        let fd = master.as_raw_fd();
        let mut fdset: libc::pollfd = libc::pollfd { fd, events: libc::POLLIN, revents: 0 };
        let ready = unsafe { libc::poll(&mut fdset, 1, timeout_ms) };
        if ready > 0 && fdset.revents & libc::POLLIN != 0 {
            let mut chunk = [0u8; 4096];
            match master.read(&mut chunk) {
                Ok(0) => None,
                Ok(n) => Some(chunk[..n].to_vec()),
                Err(_) => None,
            }
        } else {
            None
        }
    }

    /// Drain until `needle` appears (panics on timeout), returning everything
    /// received. Synchronizes with the console before sending input.
    /// Wall-clock deadlines: an unopened slave reports instant HUP, which
    /// must not spin the wait budget.
    pub fn read_until(master: &mut File, needle: &[u8], timeout_ms: u64) -> Vec<u8> {
        let start = std::time::Instant::now();
        let budget = std::time::Duration::from_millis(timeout_ms.max(100));
        let mut output = Vec::new();
        while !output.windows(needle.len()).any(|w| w == needle) {
            match poll_read(master, 50) {
                Some(chunk) => output.extend(chunk),
                None => {
                    assert!(start.elapsed() < budget, "timed out waiting for console output");
                    std::thread::sleep(std::time::Duration::from_millis(10));
                }
            }
        }
        output
    }

    /// Drain until the master idles, returning everything received.
    pub fn drain_idle(master: &mut File, idle_ms: u64) -> Vec<u8> {
        let budget = std::time::Duration::from_millis(idle_ms.max(100));
        let mut output = Vec::new();
        let mut quiet_since = std::time::Instant::now();
        loop {
            match poll_read(master, 50) {
                Some(chunk) => {
                    output.extend(chunk);
                    quiet_since = std::time::Instant::now();
                }
                None => {
                    if quiet_since.elapsed() >= budget {
                        break;
                    }
                    std::thread::sleep(std::time::Duration::from_millis(10));
                }
            }
        }
        output
    }

    pub fn open_slave(path: &str) -> File {
        OpenOptions::new().read(true).write(true).open(path).unwrap()
    }
}

#[cfg(test)]
mod tests {
    use super::test_support::*;
    use super::*;
    use crate::command::FnRunner;

    #[test]
    fn ask_trims_and_refuses_controls() {
        let pty = open_pty();
        let mut master = pty.master;
        let slave = pty.slave_path.clone();
        let worker = std::thread::spawn(move || {
            let console = Console::open(&slave).unwrap();
            let (ctx, _flag) = Ctx::test();
            console.ask(&ctx, "Name")
        });
        let mut output = read_until(&mut master, b"Name: ", 2000);
        master.write_all(b"  soda-01  \n").unwrap();
        assert_eq!(worker.join().unwrap().unwrap(), "soda-01");
        output.extend(drain_idle(&mut master, 200));
        assert!(String::from_utf8_lossy(&output).contains("Name: "));

        let worker = std::thread::spawn(move || {
            let console = Console::open(&pty.slave_path).unwrap();
            let (ctx, _flag) = Ctx::test();
            console.ask(&ctx, "Name")
        });
        read_until(&mut master, b"Name: ", 2000);
        master.write_all(b"a\x01b\n").unwrap();
        assert_eq!(worker.join().unwrap().unwrap_err().to_string(), "control character refused");
    }

    #[test]
    fn secret_hides_input_and_restores_echo() {
        let pty = open_pty();
        let slave = pty.slave_path.clone();
        let mut master = pty.master;
        let worker = std::thread::spawn(move || {
            let console = Console::open(&slave).unwrap();
            let (ctx, _flag) = Ctx::test();
            console.secret(&ctx, "Password")
        });
        let mut output = read_until(&mut master, b"Password: ", 2000);
        master.write_all(b"s3cret!\n").unwrap();
        assert_eq!(worker.join().unwrap().unwrap(), b"s3cret!");
        output.extend(drain_idle(&mut master, 200));
        // Hidden input echoes nothing; only the prompt and newline show.
        assert!(!String::from_utf8_lossy(&output).contains("s3cret"));
        // Echo is restored for the next prompt.
        let worker = std::thread::spawn(move || {
            let console = Console::open(&pty.slave_path).unwrap();
            let (ctx, _flag) = Ctx::test();
            console.ask(&ctx, "Next")
        });
        let mut output = read_until(&mut master, b"Next: ", 2000);
        master.write_all(b"visible\n").unwrap();
        assert_eq!(worker.join().unwrap().unwrap(), "visible");
        output.extend(drain_idle(&mut master, 200));
        assert!(String::from_utf8_lossy(&output).contains("Next: "));
    }

    #[test]
    fn cancelled_line_returns_ctx_error() {
        let pty = open_pty();
        let console = Console::open(&pty.slave_path).unwrap();
        let (ctx, flag) = Ctx::test();
        flag.store(true, std::sync::atomic::Ordering::SeqCst);
        assert_eq!(console.line(&ctx).unwrap_err(), Error::Canceled);
    }

    #[test]
    fn network_step_flows() {
        let pty = open_pty();
        let slave = pty.slave_path.clone();
        let mut master = pty.master;
        let worker = std::thread::spawn(move || {
            let console = Console::open(&slave).unwrap();
            let (ctx, _flag) = Ctx::test();
            let runner = FnRunner::new(|_, name, _, _| {
                assert_eq!(name, "ip");
                Ok(b"lo UP 127.0.0.1/8\n".to_vec())
            });
            console.network_with(&ctx, &runner)
        });
        let mut output = read_until(&mut master, b"Type keep, edit, back, restart, or cancel: ", 2000);
        master.write_all(b"keep\n").unwrap();
        output.extend(read_until(&mut master, b"Type yes to use them, edit, back, restart, or cancel: ", 2000));
        master.write_all(b"yes\n").unwrap();
        assert!(worker.join().unwrap().is_ok());
        output.extend(drain_idle(&mut master, 200));
        let text = String::from_utf8_lossy(&output);
        assert!(text.contains("DHCP is ready by default."));
        assert!(text.contains("Current live addresses:"));
        assert!(text.contains("\"lo UP 127.0.0.1/8\""));
    }
}

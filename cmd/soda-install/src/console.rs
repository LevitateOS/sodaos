//! Operator console: prompting, line/password entry with terminal echo
//! control, and the DHCP/`nmtui` network step.

use std::fs::File;
use std::os::unix::process::ExitStatusExt;

use crate::command::Runner;
use crate::errors::Error;
use crate::fmtx::{go_lower, Arg};
use crate::signal::Ctx;

pub struct Console {
    tty_path: String,
    tty: File,
}

impl Console {
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
            child
                .wait()
                .unwrap_or_else(|_| std::process::ExitStatus::from_raw(1 << 8))
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
        Console {
            tty_path: name.to_string(),
            tty,
        }
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
            _ => Ok((
                "Choose keep, edit, back, restart, or cancel.".to_string(),
                true,
            )),
        }
    }

    pub fn inspect_and_confirm_network(
        &self,
        ctx: &Ctx,
        run: &dyn Runner,
    ) -> Result<(bool, String), Error> {
        let data = match run.run(
            ctx,
            "ip",
            &["-brief".to_string(), "address".to_string()],
            None,
        ) {
            Ok(data) => data,
            Err(_) => {
                return Ok((
                    false,
                    "Could not inspect live network addresses.".to_string(),
                ))
            }
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
            self.print(
                "The installed system will receive the reviewed live settings.",
                &[],
            );
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
        0xe2 if prefix.len() >= 3
            && prefix[1] == 0x80
            && (prefix[2] <= 0x8a
                || prefix[2] == 0xa8
                || prefix[2] == 0xa9
                || prefix[2] == 0xaf) =>
        {
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
            if end >= candidate
                && start <= end - candidate
                && go_space_len(&data[end - candidate..end]) == candidate
            {
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

mod terminal;

#[cfg(test)]
pub mod test_support;

#[cfg(test)]
mod tests;

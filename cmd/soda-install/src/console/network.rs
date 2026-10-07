use std::fs::File;
use std::os::unix::process::ExitStatusExt;

use crate::command::Runner;
use crate::errors::Error;
use crate::fmtx::{go_lower, Arg};
use crate::signal::Ctx;

use super::{trim_space_bytes, Console};

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
        let status = loop {
            if ctx.err().is_some() {
                let _ = child.kill();
            }
            match child.try_wait() {
                Ok(Some(status)) => break status,
                Ok(None) => {}
                Err(_) => {
                    let _ = child.kill();
                    break child
                        .wait()
                        .unwrap_or_else(|_| std::process::ExitStatus::from_raw(1 << 8));
                }
            }
            std::thread::sleep(std::time::Duration::from_millis(5));
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

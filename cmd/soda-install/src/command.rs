//! Privileged command execution: Go `command()` with its 8 MiB output
//! bound, 2 h timeout, cancellation kills, and `commandExit` taxonomy,
//! behind an injectable [`Runner`] for the console tests.

use std::io::{Read, Write};
use std::os::unix::process::ExitStatusExt;
use std::process::Stdio;
use std::sync::mpsc;
use std::time::{Duration, Instant};

use crate::errors::Error;
use crate::signal::Ctx;

/// 8 MiB stdout bound, mirroring `boundedOutput`.
const OUTPUT_BOUND: usize = 8 << 20;
/// 2 h command timeout, mirroring `commandTimeout`.
const COMMAND_TIMEOUT: Duration = Duration::from_secs(2 * 60 * 60);
/// 2 s post-exit output grace, mirroring `WaitDelay`.
const OUTPUT_GRACE: Duration = Duration::from_secs(2);

/// Command runner: `Ok(output)` on exit 0 within the output bound,
/// `Err(Error::CmdExit)` otherwise.
pub trait Runner {
    fn run(
        &self,
        ctx: &Ctx,
        name: &str,
        args: &[String],
        input: Option<&[u8]>,
    ) -> Result<Vec<u8>, Error>;
}

/// Closure-backed runner for tests.
#[cfg(test)]
pub struct FnRunner<F: Fn(&Ctx, &str, &[String], Option<&[u8]>) -> Result<Vec<u8>, Error>> {
    func: F,
}

#[cfg(test)]
impl<F: Fn(&Ctx, &str, &[String], Option<&[u8]>) -> Result<Vec<u8>, Error>> FnRunner<F> {
    pub fn new(func: F) -> FnRunner<F> {
        FnRunner { func }
    }
}

#[cfg(test)]
impl<F: Fn(&Ctx, &str, &[String], Option<&[u8]>) -> Result<Vec<u8>, Error>> Runner for FnRunner<F> {
    fn run(
        &self,
        ctx: &Ctx,
        name: &str,
        args: &[String],
        input: Option<&[u8]>,
    ) -> Result<Vec<u8>, Error> {
        (self.func)(ctx, name, args, input)
    }
}

/// `failureSummary`: exit failures keep their message, everything else is
/// suppressed behind the generic line.
pub fn failure_summary(err: &Error) -> String {
    match err {
        Error::CmdExit { .. } => err.to_string(),
        _ => "native command failed; raw diagnostics suppressed".to_string(),
    }
}

/// Real privileged runner: `Zero` stdin unless input is given, piped stdout
/// with the output bound, `/dev/null` stderr, SIGKILL on cancel/timeout.
pub struct RealRunner;

impl Runner for RealRunner {
    fn run(
        &self,
        ctx: &Ctx,
        name: &str,
        args: &[String],
        input: Option<&[u8]>,
    ) -> Result<Vec<u8>, Error> {
        if ctx.err().is_some() {
            // Go refuses to start cancelled commands; without a process
            // state the exit code is -1 and the phase is interrupted.
            return Err(Error::CmdExit {
                name: name.to_string(),
                code: -1,
                interrupted: true,
            });
        }
        let mut child = match std::process::Command::new(name)
            .args(args)
            .stdin(if input.is_some() {
                Stdio::piped()
            } else {
                Stdio::null()
            })
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn()
        {
            Ok(child) => child,
            Err(_) => {
                return Err(Error::CmdExit {
                    name: name.to_string(),
                    code: -1,
                    interrupted: ctx.err().is_some(),
                })
            }
        };
        if let Some(input) = input {
            // Installer stdin payloads are tiny (a password line); a single
            // write before the wait cannot block.
            let _ = child.stdin.take().unwrap().write_all(input);
        }
        let mut stdout = child.stdout.take().unwrap();
        let (output_tx, output_rx) = mpsc::channel();
        std::thread::spawn(move || {
            let mut output = Vec::new();
            let mut chunk = [0u8; 65536];
            let mut exceeded = false;
            loop {
                match stdout.read(&mut chunk) {
                    Ok(0) => break,
                    Ok(n) => {
                        if output.len() + n > OUTPUT_BOUND + 1 {
                            exceeded = true;
                            output.extend_from_slice(&chunk[..OUTPUT_BOUND + 1 - output.len()]);
                            // Drain the rest so the child never blocks.
                            while stdout.read(&mut chunk).unwrap_or(0) > 0 {}
                            break;
                        }
                        output.extend_from_slice(&chunk[..n]);
                    }
                    Err(_) => break,
                }
            }
            let _ = output_tx.send((output, exceeded));
        });
        let deadline = Instant::now() + COMMAND_TIMEOUT;
        let pid = child.id() as libc::pid_t;
        let mut killed = false;
        let mut status_code: libc::c_int = 0;
        let reaped = loop {
            if ctx.err().is_some() || Instant::now() >= deadline {
                let _ = child.kill();
                killed = true;
                break false;
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
            std::thread::sleep(Duration::from_millis(5));
        };
        // A kill we issued reaps through the handle; a poll that already
        // reaped the child converts the raw status. Dropping the handle
        // afterwards is safe: the child is already reaped.
        let status = if reaped {
            std::process::ExitStatus::from_raw(status_code)
        } else {
            match child.wait() {
                Ok(status) => status,
                Err(_) => {
                    return Err(Error::CmdExit {
                        name: name.to_string(),
                        code: -1,
                        interrupted: true,
                    })
                }
            }
        };
        let (output, exceeded) = output_rx
            .recv_timeout(OUTPUT_GRACE)
            .unwrap_or((Vec::new(), true));
        let code = status.code().unwrap_or(-1);
        // Go converts every failure (exit code, signal, output bound, or a
        // cancel that won the race) into commandExit; a kill we issued or a
        // cancelled phase marks the attempt interrupted.
        let interrupted = killed || ctx.err().is_some();
        if exceeded || !status.success() {
            return Err(Error::CmdExit {
                name: name.to_string(),
                code,
                interrupted,
            });
        }
        Ok(output)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn runner_reports_exit_codes() {
        let runner = RealRunner;
        let (ctx, _flag) = Ctx::test();
        let output = runner.run(&ctx, "echo", &["hi".to_string()], None).unwrap();
        assert_eq!(output, b"hi\n");
        let err = runner
            .run(&ctx, "sh", &["-c".to_string(), "exit 3".to_string()], None)
            .unwrap_err();
        assert_eq!(
            err,
            Error::CmdExit {
                name: "sh".to_string(),
                code: 3,
                interrupted: false
            }
        );
        assert_eq!(
            err.to_string(),
            "sh failed (exit 3, interrupted false); raw diagnostics suppressed"
        );
        let err = runner
            .run(&ctx, "no-such-soda-command", &[], None)
            .unwrap_err();
        assert_eq!(
            err,
            Error::CmdExit {
                name: "no-such-soda-command".to_string(),
                code: -1,
                interrupted: false
            }
        );
        assert_eq!(
            failure_summary(&err),
            "no-such-soda-command failed (exit -1, interrupted false); raw diagnostics suppressed"
        );
        assert_eq!(
            failure_summary(&Error::msg("boom")),
            "native command failed; raw diagnostics suppressed"
        );
    }

    #[test]
    fn cancelled_scope_marks_exit_interrupted() {
        let runner = RealRunner;
        let (ctx, flag) = Ctx::test();
        flag.store(true, std::sync::atomic::Ordering::SeqCst);
        let err = runner
            .run(&ctx, "echo", &["hi".to_string()], None)
            .unwrap_err();
        assert_eq!(
            err,
            Error::CmdExit {
                name: "echo".to_string(),
                code: -1,
                interrupted: true
            }
        );
    }

    #[test]
    fn stdin_round_trip() {
        let runner = RealRunner;
        let (ctx, _flag) = Ctx::test();
        let output = runner.run(&ctx, "cat", &[], Some(b"payload")).unwrap();
        assert_eq!(output, b"payload");
    }
}

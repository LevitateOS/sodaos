//! Privileged command execution: Go `command()` with its 8 MiB output
//! bound, 2 h timeout, cancellation kills, and `commandExit` taxonomy,
//! behind an injectable [`Runner`] for the console tests.

use std::io::{Read, Write};
use std::os::unix::io::AsRawFd;
use std::process::Stdio;
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

/// Take the piped stdio and force nonblocking mode. Either failure reaps
/// the child before any pump I/O, exactly like the inline setup it replaces.
fn take_pipes(
    child: &mut std::process::Child,
    name: &str,
) -> Result<(Option<std::process::ChildStdin>, std::process::ChildStdout), Error> {
    let stdin = child.stdin.take();
    if stdin
        .as_ref()
        .is_some_and(|pipe| !set_nonblocking(pipe.as_raw_fd()))
    {
        let _ = child.kill();
        let _ = child.wait();
        return Err(Error::CmdExit {
            name: name.to_string(),
            code: -1,
            interrupted: true,
        });
    }
    let stdout = child.stdout.take().unwrap();
    if !set_nonblocking(stdout.as_raw_fd()) {
        let _ = child.kill();
        let _ = child.wait();
        return Err(Error::CmdExit {
            name: name.to_string(),
            code: -1,
            interrupted: true,
        });
    }
    Ok((stdin, stdout))
}

/// Cancel/timeout prologue of one pump iteration: kill, reap and report
/// `true` so the caller breaks, mirroring the inline check it replaces.
fn check_cancel(
    child: &mut std::process::Child,
    ctx: &Ctx,
    deadline: Instant,
    killed: &mut bool,
    status: &mut Option<std::process::ExitStatus>,
) -> bool {
    if ctx.err().is_some() || Instant::now() >= deadline {
        let _ = child.kill();
        *killed = true;
        *status = child.wait().ok();
        true
    } else {
        false
    }
}

/// One stdin pump step. Returns `true` when the pipe is fully written and
/// the caller must close it, exactly like the inline `close_stdin` flag.
fn pump_stdin(
    stdin: &mut Option<std::process::ChildStdin>,
    input: &[u8],
    input_offset: &mut usize,
    ctx: &Ctx,
    child: &mut std::process::Child,
    name: &str,
) -> Result<bool, Error> {
    let Some(pipe) = stdin.as_mut() else {
        return Ok(false);
    };
    if *input_offset == input.len() {
        return Ok(true);
    }
    match pipe.write(&input[*input_offset..]) {
        Ok(0) => {
            let _ = child.kill();
            let _ = child.wait();
            Err(Error::CmdExit {
                name: name.to_string(),
                code: -1,
                interrupted: ctx.err().is_some(),
            })
        }
        Ok(n) => {
            *input_offset += n;
            Ok(false)
        }
        Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => Ok(false),
        Err(e) if e.kind() == std::io::ErrorKind::Interrupted => Ok(false),
        Err(_) => {
            let _ = child.kill();
            let _ = child.wait();
            Err(Error::CmdExit {
                name: name.to_string(),
                code: -1,
                interrupted: ctx.err().is_some(),
            })
        }
    }
}

/// One stdout pump step, enforcing the output bound. Sets `stdout_eof` on
/// clean EOF; every other outcome matches the inline reader it replaces.
fn pump_stdout(
    stdout: &mut std::process::ChildStdout,
    chunk: &mut [u8; 65536],
    output: &mut Vec<u8>,
    stdout_eof: &mut bool,
    ctx: &Ctx,
    child: &mut std::process::Child,
    name: &str,
) -> Result<(), Error> {
    match stdout.read(chunk) {
        Ok(0) => {
            *stdout_eof = true;
            Ok(())
        }
        Ok(n) => {
            if output.len().saturating_add(n) > OUTPUT_BOUND {
                let _ = child.kill();
                let _ = child.wait();
                return Err(Error::CmdExit {
                    name: name.to_string(),
                    code: -1,
                    interrupted: ctx.err().is_some(),
                });
            }
            output.extend_from_slice(&chunk[..n]);
            Ok(())
        }
        Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => Ok(()),
        Err(e) if e.kind() == std::io::ErrorKind::Interrupted => Ok(()),
        Err(_) => {
            let _ = child.kill();
            let _ = child.wait();
            Err(Error::CmdExit {
                name: name.to_string(),
                code: -1,
                interrupted: ctx.err().is_some(),
            })
        }
    }
}

/// Nonblocking reap poll. Records the exit instant on first observation,
/// exactly like the inline `try_wait` block it replaces.
fn poll_status(
    child: &mut std::process::Child,
    status: &mut Option<std::process::ExitStatus>,
    exited_at: &mut Option<Instant>,
    name: &str,
) -> Result<(), Error> {
    if status.is_none() {
        match child.try_wait() {
            Ok(Some(done)) => {
                *status = Some(done);
                *exited_at = Some(Instant::now());
            }
            Ok(None) => {}
            Err(_) => {
                let _ = child.kill();
                let _ = child.wait();
                return Err(Error::CmdExit {
                    name: name.to_string(),
                    code: -1,
                    interrupted: true,
                });
            }
        }
    }
    Ok(())
}

/// Post-reap checks of one pump iteration: an exit that won the race while
/// input is still pending fails, a drained exit breaks the loop, and an
/// exit older than the output grace fails. Returns `true` to break.
fn check_exit(
    status: Option<std::process::ExitStatus>,
    stdout_eof: bool,
    exited_at: Option<Instant>,
    input_pending: bool,
    ctx: &Ctx,
    child: &mut std::process::Child,
    name: &str,
) -> Result<bool, Error> {
    if status.is_some() && input_pending {
        let _ = child.kill();
        let _ = child.wait();
        return Err(Error::CmdExit {
            name: name.to_string(),
            code: -1,
            interrupted: ctx.err().is_some(),
        });
    }
    if status.is_some() {
        if stdout_eof {
            return Ok(true);
        }
        if exited_at.is_some_and(|at| at.elapsed() >= OUTPUT_GRACE) {
            return Err(Error::CmdExit {
                name: name.to_string(),
                code: -1,
                interrupted: ctx.err().is_some(),
            });
        }
    }
    Ok(false)
}

/// Classify the reaped status into the `commandExit` taxonomy: every
/// failure (exit code, signal, output bound, or a cancel that won the
/// race) becomes `CmdExit`; a kill we issued or a cancelled phase marks
/// the attempt interrupted.
fn finish_command(
    status: Option<std::process::ExitStatus>,
    killed: bool,
    ctx: &Ctx,
    name: &str,
    output: Vec<u8>,
) -> Result<Vec<u8>, Error> {
    let status = match status {
        Some(status) => status,
        None => {
            return Err(Error::CmdExit {
                name: name.to_string(),
                code: -1,
                interrupted: true,
            });
        }
    };
    let code = status.code().unwrap_or(-1);
    let interrupted = killed || ctx.err().is_some();
    if !status.success() {
        return Err(Error::CmdExit {
            name: name.to_string(),
            code,
            interrupted,
        });
    }
    Ok(output)
}

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
                });
            }
        };
        let (mut stdin, mut stdout) = take_pipes(&mut child, name)?;
        let deadline = Instant::now() + COMMAND_TIMEOUT;
        let mut killed = false;
        let mut output = Vec::new();
        let mut input_offset = 0;
        let input = input.unwrap_or(&[]);
        let mut status = None;
        let mut exited_at = None;
        let mut stdout_eof = false;
        let mut chunk = [0u8; 65536];
        loop {
            if check_cancel(&mut child, ctx, deadline, &mut killed, &mut status) {
                break;
            }
            if pump_stdin(&mut stdin, input, &mut input_offset, ctx, &mut child, name)? {
                stdin = None;
            }
            if !stdout_eof {
                pump_stdout(
                    &mut stdout,
                    &mut chunk,
                    &mut output,
                    &mut stdout_eof,
                    ctx,
                    &mut child,
                    name,
                )?;
            }
            poll_status(&mut child, &mut status, &mut exited_at, name)?;
            if check_exit(
                status,
                stdout_eof,
                exited_at,
                input_offset < input.len(),
                ctx,
                &mut child,
                name,
            )? {
                break;
            }
            std::thread::sleep(Duration::from_millis(5));
        }
        finish_command(status, killed, ctx, name, output)
    }
}

fn set_nonblocking(fd: libc::c_int) -> bool {
    let flags = unsafe { libc::fcntl(fd, libc::F_GETFL) };
    flags >= 0 && unsafe { libc::fcntl(fd, libc::F_SETFL, flags | libc::O_NONBLOCK) } >= 0
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

    #[test]
    fn successful_exit_cannot_drop_pending_input() {
        let runner = RealRunner;
        let (ctx, _) = Ctx::test();
        let input = vec![b'x'; 1024 * 1024];
        assert!(runner
            .run(
                &ctx,
                "sh",
                &["-c".to_string(), "exit 0".to_string()],
                Some(&input)
            )
            .is_err());
    }

    #[test]
    fn infinite_output_fails_at_bound() {
        let runner = RealRunner;
        let (ctx, _) = Ctx::test();
        assert!(runner.run(&ctx, "yes", &[], None).is_err());
    }

    #[test]
    fn cancellation_kills_and_reaps_direct_child() {
        let runner = RealRunner;
        let (ctx, flag) = Ctx::test();
        std::thread::scope(|scope| {
            scope.spawn(|| {
                std::thread::sleep(Duration::from_millis(50));
                flag.store(true, std::sync::atomic::Ordering::SeqCst);
            });
            let error = runner
                .run(
                    &ctx,
                    "sh",
                    &["-c".to_string(), "exec sleep 30".to_string()],
                    None,
                )
                .unwrap_err();
            assert!(matches!(
                error,
                Error::CmdExit {
                    interrupted: true,
                    ..
                }
            ));
        });
    }
}

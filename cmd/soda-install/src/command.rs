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
        let mut stdin = child.stdin.take();
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
        let mut stdout = child.stdout.take().unwrap();
        if !set_nonblocking(stdout.as_raw_fd()) {
            let _ = child.kill();
            let _ = child.wait();
            return Err(Error::CmdExit {
                name: name.to_string(),
                code: -1,
                interrupted: true,
            });
        }
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
            if ctx.err().is_some() || Instant::now() >= deadline {
                let _ = child.kill();
                killed = true;
                status = child.wait().ok();
                break;
            }
            let mut close_stdin = false;
            if let Some(pipe) = stdin.as_mut() {
                if input_offset == input.len() {
                    close_stdin = true;
                } else {
                    match pipe.write(&input[input_offset..]) {
                        Ok(0) => {
                            let _ = child.kill();
                            let _ = child.wait();
                            return Err(Error::CmdExit {
                                name: name.to_string(),
                                code: -1,
                                interrupted: ctx.err().is_some(),
                            });
                        }
                        Ok(n) => input_offset += n,
                        Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => {}
                        Err(e) if e.kind() == std::io::ErrorKind::Interrupted => {}
                        Err(_) => {
                            let _ = child.kill();
                            let _ = child.wait();
                            return Err(Error::CmdExit {
                                name: name.to_string(),
                                code: -1,
                                interrupted: ctx.err().is_some(),
                            });
                        }
                    }
                }
            }
            if close_stdin {
                stdin = None;
            }
            if !stdout_eof {
                match stdout.read(&mut chunk) {
                    Ok(0) => stdout_eof = true,
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
                    }
                    Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => {}
                    Err(e) if e.kind() == std::io::ErrorKind::Interrupted => {}
                    Err(_) => {
                        let _ = child.kill();
                        let _ = child.wait();
                        return Err(Error::CmdExit {
                            name: name.to_string(),
                            code: -1,
                            interrupted: ctx.err().is_some(),
                        });
                    }
                }
            }
            if status.is_none() {
                match child.try_wait() {
                    Ok(Some(done)) => {
                        status = Some(done);
                        exited_at = Some(Instant::now());
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
            if status.is_some() && input_offset < input.len() {
                let _ = child.kill();
                let _ = child.wait();
                return Err(Error::CmdExit {
                    name: name.to_string(),
                    code: -1,
                    interrupted: ctx.err().is_some(),
                });
            }
            if let Some(done) = status {
                if stdout_eof {
                    status = Some(done);
                    break;
                }
                if exited_at.is_some_and(|at| at.elapsed() >= OUTPUT_GRACE) {
                    return Err(Error::CmdExit {
                        name: name.to_string(),
                        code: -1,
                        interrupted: ctx.err().is_some(),
                    });
                }
            }
            std::thread::sleep(Duration::from_millis(5));
        }
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
        // Go converts every failure (exit code, signal, output bound, or a
        // cancel that won the race) into commandExit; a kill we issued or a
        // cancelled phase marks the attempt interrupted.
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

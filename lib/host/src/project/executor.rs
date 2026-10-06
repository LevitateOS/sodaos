use std::time::{Duration, Instant};

pub trait Executor {
    fn run(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String>;
    /// Mirrors Go's `HostNative()` marker assertion: the privileged host
    /// executor whose native protocols must never leak stderr text.
    fn is_host_native(&self) -> bool {
        false
    }
}

impl<E: Executor> Executor for &E {
    fn run(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        (*self).run(stdin, cmd, args, deadline)
    }

    fn is_host_native(&self) -> bool {
        (*self).is_host_native()
    }
}

/// Native process execution with deadline kill, mirroring
/// `exec.CommandContext` + `Output` (piped stdin, combined stderr in the
/// failure text).
pub struct Native;

impl Executor for Native {
    fn run(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        execute(stdin, cmd, args, deadline, false)
    }

    fn is_host_native(&self) -> bool {
        true
    }
}

/// Native process execution reporting status-only failures (Go `Run`
/// shape: `exit status N` / `signal: name`), for CLI diagnostics that
/// must not emit subprocess stderr. Shares Native's launch, concurrent
/// drain, and deadline kill/wait lifecycle; only the error format differs.
pub struct NativeStatusOnly;

impl Executor for NativeStatusOnly {
    fn run(
        &self,
        stdin: &[u8],
        cmd: &str,
        args: &[&str],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        execute(stdin, cmd, args, deadline, true)
    }

    fn is_host_native(&self) -> bool {
        true
    }
}

fn execute(
    stdin: &[u8],
    cmd: &str,
    args: &[&str],
    deadline: Instant,
    status_only: bool,
) -> Result<Vec<u8>, String> {
    use std::io::Write;
    use std::process::Stdio;
    // Refuse before launch: an expired deadline must never start an
    // effectful command. The error describes the refusal, never a kill
    // (no child exists), like Go's expired-context Start refusal.
    if Instant::now() >= deadline {
        return Err(refusal_text(cmd, status_only));
    }
    let mut child = std::process::Command::new(cmd)
        .args(args)
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|e| {
            if status_only {
                format!("fork/exec {cmd}: {e}")
            } else {
                format!("{cmd} failed: {e}")
            }
        })?;
    let mut input = child.stdin.take();
    let mut out_pipe = child.stdout.take();
    let mut err_pipe = child.stderr.take();
    let outcome = std::thread::scope(|scope| {
        let writer = scope.spawn(|| {
            if let Some(mut w) = input.take() {
                let _ = w.write_all(stdin);
            }
        });
        // Drain stdout/stderr concurrently: a child emitting beyond
        // pipe capacity would otherwise block forever while the poll
        // loop below waits for exit (H01-F2).
        let out_drain = scope.spawn(|| {
            use std::io::Read;
            let mut buf = Vec::new();
            if let Some(mut o) = out_pipe.take() {
                let _ = o.read_to_end(&mut buf);
            }
            buf
        });
        let err_drain = scope.spawn(|| {
            use std::io::Read;
            let mut buf = Vec::new();
            if let Some(mut e) = err_pipe.take() {
                let _ = e.read_to_end(&mut buf);
            }
            buf
        });
        let status = loop {
            match child.try_wait().map_err(|e| format!("{cmd} failed: {e}"))? {
                Some(status) => break status,
                None => {
                    if Instant::now() >= deadline {
                        let _ = child.kill();
                        let waited = child.wait();
                        let _ = writer.join();
                        if status_only {
                            // Go's deadline kill surfaces the wait
                            // status, not a deadline message.
                            return match waited {
                                Ok(status) => Err(exit_text(status)),
                                Err(e) => Err(format!("wait {cmd}: {e}")),
                            };
                        }
                        return Err(format!("{cmd} failed: deadline exceeded"));
                    }
                    std::thread::sleep(Duration::from_millis(5));
                }
            }
        };
        writer
            .join()
            .map_err(|_| format!("{cmd} failed: stdin writer panicked"))?;
        let stdout = out_drain.join().unwrap_or_default();
        let stderr = err_drain.join().unwrap_or_default();
        Ok((status, stdout, stderr))
    })?;
    let (status, stdout, stderr): (std::process::ExitStatus, Vec<u8>, Vec<u8>) = outcome;
    if status.success() {
        return Ok(stdout);
    }
    if status_only {
        return Err(exit_text(status));
    }
    Err(format!(
        "{cmd} failed: {}: {}",
        exit_text(status),
        String::from_utf8_lossy(&stderr)
    ))
}

/// Expired-deadline refusal text: no child was started, so the wording
/// reports the missed deadline and never claims a killed child.
fn refusal_text(cmd: &str, status_only: bool) -> String {
    if status_only {
        String::from("deadline exceeded")
    } else {
        format!("{cmd} failed: deadline exceeded")
    }
}

#[cfg(unix)]
fn exit_text(status: std::process::ExitStatus) -> String {
    use std::os::unix::process::ExitStatusExt;
    match status.code() {
        Some(code) => format!("exit status {code}"),
        None => match status.signal() {
            Some(1) => "signal: hangup".to_string(),
            Some(2) => "signal: interrupt".to_string(),
            Some(3) => "signal: quit".to_string(),
            Some(6) => "signal: aborted".to_string(),
            Some(9) => "signal: killed".to_string(),
            Some(15) => "signal: terminated".to_string(),
            Some(n) => format!("signal: {n}"),
            None => "signal: unknown".to_string(),
        },
    }
}

#[cfg(not(unix))]
fn exit_text(status: std::process::ExitStatus) -> String {
    match status.code() {
        Some(code) => format!("exit status {code}"),
        None => "signal: unknown".to_string(),
    }
}

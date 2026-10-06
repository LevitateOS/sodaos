//! Owned process-group execution, mirroring `process.go`.
//!
//! A started child leads its own process group. A reaper thread observes
//! the leader's exit without reaping it (`waitid` with `WNOWAIT`), then
//! terminates any surviving group members before reaping, so an exited
//! parent can neither leave descendants behind nor let cleanup signal a
//! reused PID. Cancellation and deadlines arrive through [`Phase`], the
//! port of Go's `context.Context` for this crate.

use std::sync::{Arc, Condvar, Mutex};
use std::thread::JoinHandle;
use std::time::{Duration, Instant};

use crate::error::Error;
use crate::evidence::RedactingWriter;

#[path = "process/launch.rs"]
mod launch;
#[path = "process/phase.rs"]
mod phase;

use self::launch::lock;
pub use self::launch::start_process;
pub use self::phase::Phase;

/// Shared redacting sink for pump threads.
pub type SharedWriter = Arc<Mutex<RedactingWriter>>;

/// Reaped outcome: exit code plus separated cleanup/wait failures.
#[derive(Debug, Clone)]
pub struct ProcessOutcome {
    /// Exit code, or `-1` for signal termination, like Go's `ExitCode`.
    pub exit_code: Option<i32>,
    /// Wait failure message (non-zero exit or signal).
    pub wait_message: Option<String>,
    /// Group cleanup failure message.
    pub cleanup_message: Option<String>,
}

impl ProcessOutcome {
    /// Combined failure like Go's joined process error.
    pub fn combined(&self) -> Option<Error> {
        Error::join(vec![
            self.cleanup_message.clone().map(Error::msg),
            self.wait_message.clone().map(Error::msg),
        ])
    }
}

struct ProcessState {
    finished: bool,
    outcome: Option<ProcessOutcome>,
}

/// Owned child process with group cleanup.
pub struct Process {
    // Manual `Debug` below: handles and guards have none.
    pid: i32,
    state: Mutex<ProcessState>,
    done: Condvar,
    sealed: Mutex<bool>,
    stop_result: Mutex<Option<Result<(), String>>>,
    pumps: Mutex<Option<(JoinHandle<()>, JoinHandle<()>)>>,
    pump_error: Arc<Mutex<Option<String>>>,
}

#[cfg(target_os = "linux")]
fn reaper_main(process: Arc<Process>, pid: i32) {
    let observed = wait_owned_exit(pid);
    {
        let mut sealed = lock(&process.sealed);
        let mut cleanup: Option<String> = observed;
        // The leader is observed but unreaped, so its PID/PGID cannot be
        // reused while remaining group members are terminated.
        let rc = unsafe { libc::kill(-pid, libc::SIGKILL) };
        if rc != 0 {
            let errno = unsafe { *libc::__errno_location() };
            if errno != libc::ESRCH {
                let extra = std::io::Error::from_raw_os_error(errno).to_string();
                cleanup = Some(match cleanup {
                    Some(message) => format!("{message}\n{extra}"),
                    None => extra,
                });
            }
        }
        *sealed = true;
        let (exit_code, wait_message) = reap_leader(pid);
        let reap_message = reap_owned_children(pid);
        let cleanup_message = match (cleanup, reap_message) {
            (Some(a), Some(b)) => Some(format!("{a}\n{b}")),
            (Some(a), None) => Some(a),
            (None, b) => b,
        };
        let mut state = lock(&process.state);
        state.finished = true;
        state.outcome = Some(ProcessOutcome {
            exit_code,
            wait_message,
            cleanup_message,
        });
    }
    process.done.notify_all();
}

#[cfg(target_os = "linux")]
fn wait_owned_exit(pid: i32) -> Option<String> {
    loop {
        let mut info: libc::siginfo_t = unsafe { std::mem::zeroed() };
        let rc = unsafe {
            libc::waitid(
                libc::P_PID,
                pid as u32,
                &mut info,
                libc::WEXITED | libc::WNOWAIT,
            )
        };
        if rc == 0 {
            return None;
        }
        let errno = unsafe { *libc::__errno_location() };
        if errno == libc::EINTR {
            continue;
        }
        if errno == libc::ECHILD {
            return None;
        }
        return Some(std::io::Error::from_raw_os_error(errno).to_string());
    }
}

#[cfg(target_os = "linux")]
fn reap_leader(pid: i32) -> (Option<i32>, Option<String>) {
    let mut status = 0;
    loop {
        let rc = unsafe { libc::waitpid(pid, &mut status, 0) };
        if rc >= 0 {
            break;
        }
        let errno = unsafe { *libc::__errno_location() };
        if errno == libc::EINTR {
            continue;
        }
        if errno == libc::ECHILD {
            return (None, None);
        }
        return (
            None,
            Some(std::io::Error::from_raw_os_error(errno).to_string()),
        );
    }
    if libc::WIFEXITED(status) {
        let code = libc::WEXITSTATUS(status);
        if code == 0 {
            (Some(0), None)
        } else {
            (Some(code), Some(format!("exit status {code}")))
        }
    } else if libc::WIFSIGNALED(status) {
        let signal = libc::WTERMSIG(status);
        let mut text = format!("signal: {}", signal_name(signal));
        if libc::WCOREDUMP(status) {
            text.push_str(" (core dumped)");
        }
        (Some(-1), Some(text))
    } else {
        (Some(-1), Some("stopped process reaped".to_string()))
    }
}

/// Go `syscall.Signal.String` names on Linux.
#[cfg(target_os = "linux")]
fn signal_name(signal: i32) -> String {
    match signal {
        libc::SIGHUP => "hangup",
        libc::SIGINT => "interrupt",
        libc::SIGQUIT => "quit",
        libc::SIGILL => "illegal instruction",
        libc::SIGTRAP => "trace/breakpoint trap",
        libc::SIGABRT => "aborted",
        libc::SIGBUS => "bus error",
        libc::SIGFPE => "floating point exception",
        libc::SIGKILL => "killed",
        libc::SIGUSR1 => "user defined signal 1",
        libc::SIGSEGV => "segmentation violation",
        libc::SIGUSR2 => "user defined signal 2",
        libc::SIGPIPE => "broken pipe",
        libc::SIGALRM => "alarm clock",
        libc::SIGTERM => "terminated",
        libc::SIGSTKFLT => "stack fault",
        libc::SIGCHLD => "child exited",
        libc::SIGCONT => "continued",
        libc::SIGSTOP => "stopped (signal)",
        libc::SIGTSTP => "stopped",
        libc::SIGTTIN => "stopped (tty input)",
        libc::SIGTTOU => "stopped (tty output)",
        libc::SIGURG => "urgent I/O condition",
        libc::SIGXCPU => "CPU time limit exceeded",
        libc::SIGXFSZ => "file size limit exceeded",
        libc::SIGVTALRM => "virtual timer expired",
        libc::SIGPROF => "profiling timer expired",
        libc::SIGWINCH => "window changed",
        libc::SIGIO => "I/O possible",
        libc::SIGPWR => "power failure",
        libc::SIGSYS => "bad system call",
        _ => return format!("signal {signal}"),
    }
    .to_string()
}

#[cfg(target_os = "linux")]
fn reap_owned_children(pgid: i32) -> Option<String> {
    loop {
        let mut status = 0;
        let rc = unsafe { libc::waitpid(-pgid, &mut status, 0) };
        if rc >= 0 {
            continue;
        }
        let errno = unsafe { *libc::__errno_location() };
        if errno == libc::ECHILD {
            return None;
        }
        if errno == libc::EINTR {
            continue;
        }
        return Some(std::io::Error::from_raw_os_error(errno).to_string());
    }
}

impl std::fmt::Debug for Process {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Process")
            .field("pid", &self.pid)
            .finish_non_exhaustive()
    }
}

impl Process {
    /// Leader PID.
    pub fn pid(&self) -> i32 {
        self.pid
    }

    /// True once the reaper has finished group cleanup.
    pub fn is_done(&self) -> bool {
        lock(&self.state).finished
    }

    /// Reaped outcome, once [`Process::is_done`].
    pub fn outcome(&self) -> Option<ProcessOutcome> {
        lock(&self.state).outcome.clone()
    }

    /// Wait for completion, phase cancellation, or phase expiry.
    pub fn wait(&self, phase: &Phase) -> Result<(), Error> {
        let mut state = lock(&self.state);
        loop {
            if state.finished {
                return match state.outcome.clone().and_then(|o| o.combined()) {
                    Some(err) => Err(err),
                    None => Ok(()),
                };
            }
            if phase.is_cancelled() {
                return Err(Error::Cancelled);
            }
            if phase.expired() {
                return Err(Error::msg("context deadline exceeded"));
            }
            let slice = match phase.deadline() {
                Some(deadline) => deadline.min(Instant::now() + Duration::from_millis(50)),
                None => Instant::now() + Duration::from_millis(50),
            };
            let now = Instant::now();
            if slice > now {
                let (guard, _) = self
                    .done
                    .wait_timeout(state, slice - now)
                    .unwrap_or_else(|e| e.into_inner());
                state = guard;
            }
        }
    }

    fn wait_done_timeout(&self, duration: Duration) -> bool {
        let deadline = Instant::now() + duration;
        let mut state = lock(&self.state);
        while !state.finished {
            let now = Instant::now();
            if now >= deadline {
                return false;
            }
            let (guard, _) = self
                .done
                .wait_timeout(state, deadline - now)
                .unwrap_or_else(|e| e.into_inner());
            state = guard;
        }
        true
    }

    fn signal(&self, signal: i32) -> Result<(), Error> {
        if *lock(&self.sealed) {
            return Ok(());
        }
        signal_group(self.pid, signal)
    }

    /// Stop the group: TERM, then KILL after ten seconds, mirroring `Stop`.
    /// The first call wins; later calls replay its outcome.
    pub fn stop(&self) -> Result<(), Error> {
        let mut slot = lock(&self.stop_result);
        if let Some(cached) = slot.clone() {
            return cached.map_err(Error::msg);
        }
        let outcome: Result<(), String> = self.stop_once();
        *slot = Some(outcome.clone());
        outcome.map_err(Error::msg)
    }

    #[cfg(unix)]
    fn stop_once(&self) -> Result<(), String> {
        if let Err(e) = self.signal(libc::SIGTERM) {
            return Err(e.to_string());
        }
        if self.wait_done_timeout(Duration::from_secs(10)) {
            return match self.outcome().and_then(|o| o.combined()) {
                Some(err) => Err(err.to_string()),
                None => Ok(()),
            };
        }
        let kill_err = self.signal(libc::SIGKILL).err().map(|e| e.to_string());
        if self.wait_done_timeout(Duration::from_secs(5)) {
            let outcome_err = self
                .outcome()
                .and_then(|o| o.combined())
                .map(|e| e.to_string());
            let mut parts = vec![
                Some(
                    "owned group required forced termination: context deadline exceeded"
                        .to_string(),
                ),
                kill_err,
                outcome_err,
            ];
            parts.retain(|p| p.is_some());
            let message = parts.into_iter().flatten().collect::<Vec<_>>().join("\n");
            return Err(message);
        }
        let mut parts = vec![
            kill_err,
            Some("owned group cleanup did not complete".to_string()),
        ];
        parts.retain(|p| p.is_some());
        Err(parts.into_iter().flatten().collect::<Vec<_>>().join("\n"))
    }

    #[cfg(not(unix))]
    fn stop_once(&self) -> Result<(), String> {
        Err("safe owned process execution requires Linux non-reaping wait support".to_string())
    }

    /// Join pump threads and report the first pump failure, if any.
    pub fn join_pumps(&self) -> Option<String> {
        let pumps = lock(&self.pumps).take();
        if let Some((out, err)) = pumps {
            let _ = out.join();
            let _ = err.join();
        }
        lock(&self.pump_error).clone()
    }
}

#[cfg(unix)]
fn signal_group(pid: i32, signal: i32) -> Result<(), Error> {
    let rc = unsafe { libc::kill(-pid, signal) };
    if rc != 0 {
        let errno = unsafe { *libc::__errno_location() };
        if errno != libc::ESRCH {
            return Err(Error::from(std::io::Error::from_raw_os_error(errno)));
        }
    }
    Ok(())
}

#[cfg(not(unix))]
fn signal_group(_pid: i32, _signal: i32) -> Result<(), Error> {
    Err(Error::msg(
        "safe owned process execution requires Linux non-reaping wait support",
    ))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::command::{CommandSpec, StdinSpec};

    fn discard_pair() -> (SharedWriter, SharedWriter) {
        (
            Arc::new(Mutex::new(RedactingWriter::discard())),
            Arc::new(Mutex::new(RedactingWriter::discard())),
        )
    }

    fn shell_command(script: &str) -> CommandSpec {
        CommandSpec {
            name: "/bin/sh".to_string(),
            args: vec!["-c".to_string(), script.to_string()],
            dir: None,
            stdin: StdinSpec::Null,
            env: Vec::new(),
        }
    }

    #[test]
    #[cfg(target_os = "linux")]
    fn cancellation_before_start() {
        let phase = Phase::background();
        phase.cancel();
        let (out, err) = discard_pair();
        let result = start_process(&phase, &shell_command("exit 0"), out, err);
        assert!(result.unwrap_err().is_cancelled());
    }

    #[test]
    fn child_cancel_stays_local_but_parent_flows_down() {
        let root = Phase::background();
        let child = root.child(Duration::from_secs(60));
        child.cancel();
        assert!(child.check().is_err());
        assert!(root.check().is_ok());
        assert!(!root.is_cancelled());
        let sibling = root.child(Duration::from_secs(60));
        assert!(sibling.check().is_ok());
        root.cancel();
        assert!(sibling.check().is_err());
        assert!(child.check().is_err());
    }

    #[test]
    #[cfg(target_os = "linux")]
    fn owned_process_wait_and_cleanup() {
        let (out, err) = discard_pair();
        let process =
            start_process(&Phase::background(), &shell_command("exit 0"), out, err).unwrap();
        process
            .wait(&Phase::timeout(Duration::from_secs(5)))
            .unwrap();
        process.stop().unwrap();
        process.stop().unwrap();
        assert!(process.is_done());
    }

    #[test]
    #[cfg(target_os = "linux")]
    fn leader_exit_and_resistant_descendants() {
        for mode in ["leader-exit", "leader-term", "leader-resistant"] {
            let mut dir = std::env::temp_dir();
            dir.push(format!("soda-owned-{}-{}", std::process::id(), mode));
            let _ = std::fs::remove_dir_all(&dir);
            std::fs::create_dir_all(&dir).unwrap();
            let pid_file = dir.join("descendant.pid");
            let script = match mode {
                "leader-exit" => format!("sleep 30 & echo $! > {}; exit 0", pid_file.display()),
                "leader-term" => format!("sleep 30 & echo $! > {}; sleep 30", pid_file.display()),
                _ => format!(
                    "trap '' TERM; sleep 30 & echo $! > {}; wait",
                    pid_file.display()
                ),
            };
            let (out, err) = discard_pair();
            let process =
                start_process(&Phase::background(), &shell_command(&script), out, err).unwrap();
            let deadline = Instant::now() + Duration::from_secs(5);
            let pid = loop {
                if let Ok(raw) = std::fs::read_to_string(&pid_file) {
                    let trimmed = raw.trim().to_string();
                    if !trimmed.is_empty() {
                        break trimmed;
                    }
                }
                assert!(Instant::now() < deadline, "child not ready ({mode})");
                std::thread::sleep(Duration::from_millis(10));
            };
            if mode != "leader-exit" {
                let stop = process.stop();
                if mode == "leader-resistant" {
                    let message = stop.unwrap_err().to_string();
                    assert!(
                        message.contains("owned group required forced termination"),
                        "{message}"
                    );
                }
            }
            let waited = process.wait(&Phase::timeout(Duration::from_secs(25)));
            if mode == "leader-exit" {
                waited.unwrap();
            }
            assert!(process.is_done(), "cleanup not complete ({mode})");
            // A killed orphan can remain a zombie until init reaps it; never
            // signal a PID read from disk. Assert it no longer executes.
            let deadline = Instant::now() + Duration::from_secs(2);
            loop {
                let stat = std::fs::read_to_string(format!("/proc/{pid}/stat"));
                match stat {
                    Err(e) if e.kind() == std::io::ErrorKind::NotFound => break,
                    Err(e) if e.raw_os_error() == Some(libc::ESRCH) => break,
                    Err(e) => panic!("{e}"),
                    Ok(raw) => {
                        let tail = raw.rsplit(')').next().unwrap_or("");
                        if tail.split_whitespace().next() == Some("Z") {
                            break;
                        }
                        assert!(
                            Instant::now() < deadline,
                            "TERM-resistant descendant survived ({mode})"
                        );
                        std::thread::sleep(Duration::from_millis(10));
                    }
                }
            }
            let _ = std::fs::remove_dir_all(&dir);
        }
    }
}

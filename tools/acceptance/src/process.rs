//! Owned process-group execution, mirroring `process.go`.
//!
//! A started child leads its own process group. A reaper thread observes
//! the leader's exit without reaping it (`waitid` with `WNOWAIT`), then
//! terminates any surviving group members before reaping, so an exited
//! parent can neither leave descendants behind nor let cleanup signal a
//! reused PID. Cancellation and deadlines arrive through [`Phase`], the
//! port of Go's `context.Context` for this crate.

use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

use crate::error::Error;
use crate::evidence::RedactingWriter;

#[path = "process/launch.rs"]
mod launch;
#[path = "process/owned_process.rs"]
mod owned_process;
#[path = "process/phase.rs"]
mod phase;

pub use self::launch::start_process;
pub use self::owned_process::Process;
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

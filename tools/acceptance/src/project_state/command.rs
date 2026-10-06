use std::time::Duration;

use crate::command::{CommandSpec, StdinSpec};
use crate::process::{start_raw_process, Phase};

use super::{SnapshotFailure, SnapshotKind};

/// Snapshot stdout bound, applied during capture.
const MAX_OUTPUT: usize = 4 * 1024 * 1024;

/// Run a snapshot command with piped output and a timeout, returning
/// stripped stdout. Stderr is drained and discarded, like the Python
/// owner capturing it and never reading it.
///
/// One lifecycle: the owned `Process` keeps the leader unreaped through
/// group retirement and reaps exactly once, so descendants holding pipes
/// can neither wedge the call nor survive it, and no signal ever names
/// a reaped PID. Raw capture keeps bounded machine bytes with the
/// existing failure taxonomy; no redaction is substituted. An
/// exited-before-deadline command wins over the deadline, like the
/// previous poll loop. Pump read errors stay tolerated like the
/// previous ignored `read_to_end` results.
///
/// Latency budget, reported separately: the phase deadline bounds the
/// wait plus pipe completion (phased pumps exit, marked cancelled, at
/// the deadline even when an escaped writer group retirement cannot
/// touch holds the pipes). Owned stop adds its own bounded grace past
/// the deadline on the timeout path only: TERM, then KILL after ten
/// seconds, then a five-second reap wait — fifteen seconds worst case.
pub fn command_with_timeout(
    argv: &[String],
    extra_env: &[(&str, &str)],
    timeout: Duration,
) -> Result<String, SnapshotFailure> {
    let phase = Phase::timeout(timeout);
    if phase.expired() {
        return Err(SnapshotFailure::bare(SnapshotKind::TimeoutExpired));
    }
    let spec = CommandSpec {
        name: argv[0].clone(),
        args: argv[1..].to_vec(),
        dir: None,
        stdin: StdinSpec::Inherit,
        env: extra_env
            .iter()
            .map(|(key, value)| format!("{key}={value}"))
            .collect(),
    };
    let (process, stdout, _) = start_raw_process(&phase, &spec, MAX_OUTPUT).map_err(map_launch)?;
    let clean = match process.wait(&phase) {
        Ok(()) => process.outcome().and_then(|o| o.exit_code) == Some(0),
        Err(_) if process.is_done() => {
            let failed = process.outcome().and_then(|o| o.exit_code) != Some(0);
            let _ = process.join_pumps();
            if failed {
                return Err(failed_command(argv));
            }
            // The command exited clean but group retirement reported a
            // failure: an OS-level fault with no detail slot.
            return Err(SnapshotFailure::bare(SnapshotKind::OSError));
        }
        Err(_) => {
            // The deadline passed with the leader still owned: owned stop
            // retires the group, then pumps join retired pipes.
            let _ = process.stop();
            let _ = process.join_pumps();
            return Err(SnapshotFailure::bare(SnapshotKind::TimeoutExpired));
        }
    };
    let _ = process.join_pumps();
    if !clean {
        return Err(failed_command(argv));
    }
    if stdout.cancelled() {
        // The command finished but its pipes did not: an escaped writer
        // held them past the deadline. The deadline covers pipe
        // completion, so this is a timeout, not partial success.
        return Err(SnapshotFailure::bare(SnapshotKind::TimeoutExpired));
    }
    let (stdout, overflow) = stdout.take();
    if overflow {
        return Err(SnapshotFailure::runtime("Snapshot output exceeded bound"));
    }
    let text = String::from_utf8(stdout)
        .map_err(|_| SnapshotFailure::bare(SnapshotKind::UnicodeDecodeError))?;
    Ok(text.trim().to_string())
}

/// Launch failures keep the previous spawn mapping: missing and
/// forbidden binaries keep their kinds, anything else is an OS fault.
fn map_launch(error: crate::error::Error) -> SnapshotFailure {
    use std::io::ErrorKind;
    match error.io_kind() {
        Some(ErrorKind::NotFound) => SnapshotFailure::bare(SnapshotKind::FileNotFoundError),
        Some(ErrorKind::PermissionDenied) => SnapshotFailure::bare(SnapshotKind::PermissionError),
        _ => SnapshotFailure::bare(SnapshotKind::OSError),
    }
}

fn failed_command(argv: &[String]) -> SnapshotFailure {
    let shown: Vec<&str> = argv.iter().take(5).map(String::as_str).collect();
    SnapshotFailure::runtime(format!(
        "Required snapshot command failed: {}",
        shown.join(" ")
    ))
}

/// Snapshot command with the retired probe's 30s timeout and inherited environment.
pub fn command(argv: &[String], extra_env: &[(&str, &str)]) -> Result<String, SnapshotFailure> {
    command_with_timeout(argv, extra_env, Duration::from_secs(30))
}

/// Split stripped command output into lines. Empty output yields no
/// lines, like the retired probe's `strip().splitlines()`.
pub fn output_lines(text: &str) -> Vec<String> {
    if text.is_empty() {
        return Vec::new();
    }
    text.lines().map(|line| line.to_string()).collect()
}

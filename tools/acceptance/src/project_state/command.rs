use std::io::Read;
use std::process::{Command, Stdio};
use std::time::{Duration, Instant};

use super::{SnapshotFailure, SnapshotKind};

/// Run a snapshot command with piped output and a timeout, returning
/// stripped stdout. Stderr is drained and discarded, like the Python
/// owner capturing it and never reading it.
pub fn command_with_timeout(
    argv: &[String],
    extra_env: &[(&str, &str)],
    timeout: Duration,
) -> Result<String, SnapshotFailure> {
    let mut child = Command::new(&argv[0])
        .args(&argv[1..])
        .envs(extra_env.iter().copied())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(SnapshotFailure::io)?;
    // Both pipes drain on threads while the parent polls, like
    // `communicate()`: waiting before reading would deadlock once a pipe
    // fills.
    let mut stdout_pipe = child.stdout.take();
    let out_reader = std::thread::spawn(move || {
        let mut stdout = Vec::new();
        if let Some(pipe) = stdout_pipe.take() {
            let _ = std::io::BufReader::new(pipe).read_to_end(&mut stdout);
        }
        stdout
    });
    let mut stderr = child.stderr.take();
    let drain = std::thread::spawn(move || {
        if let Some(stderr) = stderr.take() {
            let mut sink = Vec::new();
            let _ = std::io::BufReader::new(stderr).read_to_end(&mut sink);
        }
    });
    let deadline = Instant::now() + timeout;
    loop {
        match child.try_wait().map_err(SnapshotFailure::io)? {
            Some(status) => {
                let stdout = out_reader.join().unwrap_or_default();
                let _ = drain.join();
                if !status.success() {
                    let shown: Vec<&str> = argv.iter().take(5).map(String::as_str).collect();
                    return Err(SnapshotFailure::runtime(format!(
                        "Required snapshot command failed: {}",
                        shown.join(" ")
                    )));
                }
                if stdout.len() > 4 * 1024 * 1024 {
                    return Err(SnapshotFailure::runtime("Snapshot output exceeded bound"));
                }
                let text = String::from_utf8(stdout)
                    .map_err(|_| SnapshotFailure::bare(SnapshotKind::UnicodeDecodeError))?;
                return Ok(text.trim().to_string());
            }
            None if Instant::now() >= deadline => {
                let _ = child.kill();
                let _ = child.wait();
                let _ = drain.join();
                return Err(SnapshotFailure::bare(SnapshotKind::TimeoutExpired));
            }
            None => std::thread::sleep(Duration::from_millis(10)),
        }
    }
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

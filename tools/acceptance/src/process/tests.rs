use std::time::{Duration, Instant};

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
fn child_of_unbounded_parent_has_an_effective_timeout() {
    let parent = Phase::background();
    let child = parent.child(Duration::from_millis(10));
    assert!(child.deadline().is_some());
    std::thread::sleep(Duration::from_millis(20));
    assert_eq!(
        child.check().unwrap_err().to_string(),
        "context deadline exceeded"
    );
    assert!(parent.check().is_ok());
    assert!(parent.deadline().is_none());
}

#[test]
fn child_deadline_never_extends_a_bounded_parent() {
    let parent = Phase::timeout(Duration::from_secs(1));
    let later = parent.child(Duration::from_secs(60));
    assert_eq!(later.deadline(), parent.deadline());
    let earlier = parent.child(Duration::ZERO);
    assert!(earlier.deadline().unwrap() < parent.deadline().unwrap());
    assert!(earlier.expired());
    assert!(parent.check().is_ok());
}

#[test]
#[cfg(target_os = "linux")]
fn owned_process_wait_and_cleanup() {
    let (out, err) = discard_pair();
    let process = start_process(&Phase::background(), &shell_command("exit 0"), out, err).unwrap();
    process
        .wait(&Phase::timeout(Duration::from_secs(5)))
        .unwrap();
    process.stop().unwrap();
    process.stop().unwrap();
    assert!(process.is_done());
}

#[test]
#[cfg(target_os = "linux")]
fn raw_capture_keeps_bounded_bytes_and_discards_stderr() {
    // Raw mode keeps exact machine bytes (no redaction), flags overflow
    // past the cap while still draining, and discards stderr.
    let (process, out, err) = start_raw_process(
        &Phase::background(),
        &shell_command("echo hello; echo noise >&2"),
        4,
    )
    .unwrap();
    process
        .wait(&Phase::timeout(Duration::from_secs(5)))
        .unwrap();
    process.join_pumps();
    assert_eq!(out.take(), (b"hell".to_vec(), true));
    assert_eq!(err.take(), (Vec::new(), true));
    let (exact, bytes, _) =
        start_raw_process(&Phase::background(), &shell_command("printf 'a\\tb'"), 64).unwrap();
    exact.wait(&Phase::timeout(Duration::from_secs(5))).unwrap();
    exact.join_pumps();
    assert_eq!(bytes.take(), (b"a\tb".to_vec(), false));
}

#[test]
#[cfg(target_os = "linux")]
fn phased_pumps_exit_on_deadline_without_detaching() {
    // A setsid daemon escapes the group with inherited pipes and
    // outlives group retirement; phased pumps must exit at the deadline
    // (joined, marked cancelled, partial bytes kept) rather than hang
    // on the foreign-held pipes. No foreign signal is sent: the daemon
    // outlives this fixture on its own inside a five-second stray. The
    // ready file proves the escape completed before the leader exited;
    // without it retirement could reap the daemon mid-escape.
    let dir = crate::files::TempDir::new("escape").unwrap();
    let ready = dir.path().join("escaped");
    let script = format!(
        "setsid sh -c 'echo ready > {}; exec sleep 5' & while [ ! -f {} ]; do sleep 0.01; done; echo out",
        ready.display(),
        ready.display()
    );
    let start = std::time::Instant::now();
    let (process, out, _) = start_raw_process(
        &Phase::timeout(Duration::from_millis(300)),
        &shell_command(script.as_str()),
        64,
    )
    .unwrap();
    process
        .wait(&Phase::timeout(Duration::from_secs(60)))
        .unwrap();
    process.join_pumps();
    assert!(out.cancelled());
    assert_eq!(out.take(), (b"out\n".to_vec(), false));
    assert!(
        start.elapsed() < Duration::from_secs(5),
        "pumps outlived the deadline"
    );
}

#[cfg(target_os = "linux")]
struct DeadPipe {
    reads: usize,
}

#[cfg(target_os = "linux")]
impl std::io::Read for DeadPipe {
    fn read(&mut self, _buf: &mut [u8]) -> std::io::Result<usize> {
        self.reads += 1;
        Err(std::io::Error::from(std::io::ErrorKind::BrokenPipe))
    }
}

#[cfg(target_os = "linux")]
impl std::os::fd::AsRawFd for DeadPipe {
    fn as_raw_fd(&self) -> std::os::fd::RawFd {
        // Never a valid descriptor, so fcntl fails deterministically;
        // nothing is opened, so nothing is ever closed or signalled.
        -1
    }
}

#[cfg(target_os = "linux")]
struct ProbeSink {
    writes: usize,
    cancelled: bool,
}

#[cfg(target_os = "linux")]
impl launch::PumpSink for ProbeSink {
    fn pump_write(&mut self, _bytes: &[u8]) -> Result<(), String> {
        self.writes += 1;
        Ok(())
    }

    fn pump_cancelled(&mut self) {
        self.cancelled = true;
    }
}

#[test]
#[cfg(target_os = "linux")]
fn phased_pump_setup_failure_fails_without_fallback() {
    // ACCEPTANCE-21-002: failed nonblocking setup must fail the capture
    // promptly through the pump error result. Zero reads proves no
    // blocking fallback was entered (the fallback's defining behavior
    // is the read loop); the sink stays untouched — no partial writes
    // and no cancellation, which would be deadline fiction.
    let writer = Arc::new(Mutex::new(ProbeSink {
        writes: 0,
        cancelled: false,
    }));
    let pump_error = Arc::new(Mutex::new(None));
    let mut dead = DeadPipe { reads: 0 };
    launch::pump_phased(
        &mut dead,
        &writer,
        &pump_error,
        &Phase::timeout(Duration::from_secs(5)),
    );
    assert_eq!(dead.reads, 0, "setup failure entered the read loop");
    assert!(
        pump_error.lock().unwrap().is_some(),
        "setup failure left no error"
    );
    let sink = writer.lock().unwrap();
    assert_eq!(sink.writes, 0, "setup failure wrote partial bytes");
    assert!(!sink.cancelled, "setup failure marked deadline fiction");
}

#[test]
#[cfg(target_os = "linux")]
fn stop_is_safe_at_any_lifecycle_point() {
    // Sealed ownership: stop() must be safe racing the reaper, after
    // completion, and twice — signaling and sealing share one mutex
    // so they cannot interleave into an unpinned group. Outcomes vary
    // with the race (clean exit or signal termination); safety and
    // termination do not. This locks the contract; the resealed race
    // itself closes structurally and has no deterministic trigger.
    for _ in 0..25 {
        let process = start_process(
            &Phase::background(),
            &shell_command("echo done"),
            discard_pair().0,
            discard_pair().1,
        )
        .unwrap();
        let first = process.stop().map_err(|e| e.to_string());
        let second = process.stop().map_err(|e| e.to_string());
        assert_eq!(first, second, "later stops replay the first outcome");
        let _ = process.wait(&Phase::timeout(Duration::from_secs(5)));
        assert!(process.is_done());
        process.join_pumps();
    }
    let process = start_process(
        &Phase::background(),
        &shell_command("echo done"),
        discard_pair().0,
        discard_pair().1,
    )
    .unwrap();
    process
        .wait(&Phase::timeout(Duration::from_secs(5)))
        .unwrap();
    process.stop().unwrap();
    assert_eq!(process.outcome().and_then(|o| o.exit_code), Some(0));
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

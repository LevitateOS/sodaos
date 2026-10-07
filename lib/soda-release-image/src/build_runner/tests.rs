use super::*;

#[test]
fn metadata_deadline_is_scoped_to_the_metadata_operation() {
    assert_eq!(
        bounded_operation_deadline("cargo", &["metadata".to_string()]),
        Some(std::time::Duration::from_secs(120))
    );
    assert_eq!(
        bounded_operation_deadline("cargo", &["build".to_string()]),
        None
    );
    assert_eq!(
        bounded_operation_deadline("git", &["metadata".to_string()]),
        None
    );
}

#[test]
fn bounded_build_operation_expires_while_output_keeps_arriving() {
    let started = std::time::Instant::now();
    let error = run_build_command_with_budget(
        &Cancel::new(),
        &mut Vec::new(),
        None,
        env!("CARGO_MANIFEST_DIR"),
        "sh",
        &[
            "-c".to_string(),
            "while :; do printf progress; done".to_string(),
        ],
        Some(std::time::Duration::from_millis(50)),
    )
    .unwrap_err();
    assert!(
        error.0.contains("metadata deadline exceeded"),
        "{}",
        error.0
    );
    assert!(started.elapsed() < std::time::Duration::from_secs(3));
}

#[test]
fn oracle_build_command_capture_environment_and_failure() {
    // Oracle: Go TestBuildCommandCaptureEnvironmentAndFailure.
    let cancel = Cancel::new();
    let mut log: Vec<u8> = Vec::new();
    let out = run_build_command(
        &cancel,
        &mut log,
        None,
        env!("CARGO_MANIFEST_DIR"),
        "sh",
        &[
            "-c".to_string(),
            "echo captured; echo logged >&2".to_string(),
        ],
    )
    .unwrap();
    assert_eq!(out, "captured");
    let log_text = String::from_utf8_lossy(&log);
    assert!(log_text.contains("COMMAND sh"));
    assert!(log_text.contains("logged"));
    // Only the allowlisted environment is inherited.
    std::env::set_var("SODA_SECRET_TOKEN", "must-not-leak");
    let mut log: Vec<u8> = Vec::new();
    let out = run_build_command(
        &cancel,
        &mut log,
        None,
        env!("CARGO_MANIFEST_DIR"),
        "sh",
        &[
            "-c".to_string(),
            "echo \"token=$SODA_SECRET_TOKEN toolchain=$GOTOOLCHAIN\"".to_string(),
        ],
    )
    .unwrap();
    std::env::remove_var("SODA_SECRET_TOKEN");
    assert!(out.contains("token="));
    assert!(!out.contains("must-not-leak"));
    assert!(out.contains(&format!("toolchain={PINNED_GO_VERSION}")));
    // Failures name the tool and point at the build log.
    let mut log: Vec<u8> = Vec::new();
    let err = run_build_command(
        &cancel,
        &mut log,
        None,
        env!("CARGO_MANIFEST_DIR"),
        "sh",
        &["-c".to_string(), "exit 3".to_string()],
    )
    .unwrap_err();
    assert_eq!(
        err.0,
        "sh failed; retain attempt and inspect build.log: exit status 3"
    );
}

#[test]
fn run_build_command_drains_saturated_pipes() {
    // D03-F1: 256 KiB on each stream exceeds the 64 KiB pipe buffer;
    // the executor must drain concurrently instead of hanging forever.
    let cancel = Cancel::new();
    let mut log: Vec<u8> = Vec::new();
    let out = run_build_command(
        &cancel,
        &mut log,
        None,
        env!("CARGO_MANIFEST_DIR"),
        "sh",
        &[
            "-c".to_string(),
            "head -c 262144 /dev/zero >&2; head -c 262144 /dev/zero".to_string(),
        ],
    )
    .unwrap();
    assert_eq!(out.len(), 262144);
    assert!(log.len() >= 262144);
}

#[test]
fn run_build_command_cancel_kills_and_reports() {
    // D03-F1: cancellation during a slow child kills it and reports
    // promptly; the concurrent drains are reaped without hanging.
    let cancel = Cancel::new();
    let mut log: Vec<u8> = Vec::new();
    std::thread::scope(|s| {
        s.spawn(|| {
            std::thread::sleep(std::time::Duration::from_millis(100));
            cancel.cancel();
        });
        let err = run_build_command(
            &cancel,
            &mut log,
            None,
            env!("CARGO_MANIFEST_DIR"),
            "sh",
            &["-c".to_string(), "sleep 30".to_string()],
        )
        .unwrap_err();
        assert!(err.0.contains("build cancelled"), "unexpected: {}", err.0);
    });
}

#[test]
fn cloned_cancellation_reaches_a_running_command() {
    let cancel = Cancel::new();
    let observed = cancel.clone();
    let mut log = Vec::new();
    let mut output = CancellingWriter(observed);
    let error = run_build_command(
        &cancel,
        &mut log,
        Some(&mut output),
        env!("CARGO_MANIFEST_DIR"),
        "sh",
        &["-c".to_string(), "echo ready; sleep 30".to_string()],
    )
    .unwrap_err();
    assert!(error.0.contains("build cancelled"));
}

struct CancellingWriter(Cancel);

impl std::io::Write for CancellingWriter {
    fn write(&mut self, bytes: &[u8]) -> std::io::Result<usize> {
        self.0.cancel();
        Ok(bytes.len())
    }

    fn flush(&mut self) -> std::io::Result<()> {
        Ok(())
    }
}

#[test]
fn run_build_command_streams_before_child_exit() {
    let cancel = Cancel::new();
    let mut log = Vec::new();
    let mut output = Vec::new();
    run_build_command(
        &cancel,
        &mut log,
        Some(&mut output),
        env!("CARGO_MANIFEST_DIR"),
        "sh",
        &["-c".to_string(), "echo live; sleep 0.2".to_string()],
    )
    .unwrap();
    assert_eq!(String::from_utf8(output).unwrap(), "live\n");
}

#[test]
fn run_build_command_bounds_capture() {
    let cancel = Cancel::new();
    let mut log = Vec::new();
    let error = run_build_command(
        &cancel,
        &mut log,
        None,
        env!("CARGO_MANIFEST_DIR"),
        "sh",
        &["-c".to_string(), "head -c 16777217 /dev/zero".to_string()],
    )
    .unwrap_err();
    assert!(error.0.contains("capture exceeded"));
}

#[test]
fn descendant_pipe_drain_deadline_closes_owned_fds() {
    let cancel = Cancel::new();
    let mut log = Vec::new();
    let start = std::time::Instant::now();
    let error = run_build_command(
        &cancel,
        &mut log,
        None,
        env!("CARGO_MANIFEST_DIR"),
        "sh",
        &["-c".to_string(), "sleep 2.5 & exit 0".to_string()],
    )
    .unwrap_err();
    assert!(error.0.contains("output pipe remained open"));
    assert!(start.elapsed() < std::time::Duration::from_secs(5));
}

#[test]
fn output_read_error_is_returned() {
    struct FailingReader;
    impl std::io::Read for FailingReader {
        fn read(&mut self, _buf: &mut [u8]) -> std::io::Result<usize> {
            Err(std::io::Error::other("synthetic pipe read failure"))
        }
    }
    let error = read_available(&mut Some(FailingReader)).unwrap_err();
    assert!(error.0.contains("synthetic pipe read failure"));
}

use super::*;

#[test]
fn oracle_build_command_capture_environment_and_failure() {
    // Oracle: Go TestBuildCommandCaptureEnvironmentAndFailure.
    let cancel = Cancel::new();
    let mut log: Vec<u8> = Vec::new();
    let out = run_build_command(
        &cancel,
        &mut log,
        None,
        "/tmp",
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
        "/tmp",
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
        "/tmp",
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
        "/tmp",
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
            "/tmp",
            "sh",
            &["-c".to_string(), "sleep 30".to_string()],
        )
        .unwrap_err();
        assert!(err.0.contains("build cancelled"), "unexpected: {}", err.0);
    });
}

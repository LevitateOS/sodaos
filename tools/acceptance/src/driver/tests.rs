use super::*;
use crate::files::TempDir;

#[test]
fn durations_use_bounded_humantime_grammar() {
    assert!(parse_duration("0").is_err());
    assert_eq!(parse_duration("30m").unwrap(), Duration::from_secs(1800));
    assert_eq!(parse_duration("1h30m").unwrap(), Duration::from_secs(5400));
    assert_eq!(parse_duration("1.5h").unwrap(), Duration::from_secs(5400));
    assert_eq!(parse_duration("300ms").unwrap(), Duration::from_millis(300));
    assert_eq!(parse_duration("+2µs").unwrap(), Duration::from_micros(2));
    assert_eq!(parse_duration("2μs").unwrap(), Duration::from_micros(2));
    assert!(parse_duration("").is_err());
    assert!(parse_duration("1").is_err());
    assert!(parse_duration("1x").is_err());
    assert!(parse_duration("-5s").is_err());
    assert!(parse_duration("0.0000001h").is_err());
    assert!(parse_duration("s1s").is_err());
    assert!(parse_duration("99999999999999999999h").is_err());
}

#[test]
fn clap_schema_keeps_repeats_overrides_and_literal_exec_tail() {
    let revision = "a".repeat(40);
    let args = [
        "exec",
        "--owner",
        "P02",
        "--revision",
        revision.as_str(),
        "--arch",
        "x86_64",
        "--target",
        "first",
        "--target",
        "fixture",
        "--evidence",
        "/tmp/x",
        "--remote",
        "-opaque",
        "--hold=false",
        "--secret-file",
        "one",
        "--secret-file=two",
        "true",
        "--hold",
        "false",
        "--help",
    ]
    .iter()
    .map(|s| s.to_string())
    .collect::<Vec<_>>();
    let parsed = parse_run_options(&args).unwrap();
    assert_eq!(parsed.target, "fixture");
    assert!(!parsed.hold);
    assert_eq!(parsed.secret_files, vec!["one", "two"]);
    assert_eq!(parsed.cmd_args, vec!["true", "--hold", "false", "--help"]);
    assert_eq!(parsed.remote_file, "-opaque");
}

#[test]
fn generated_help_is_clean_and_does_not_consume_exec_tail_help() {
    let help = ["exec", "--help"].map(str::to_owned);
    assert_eq!(
        parse_run_options(&help).err().unwrap().to_string(),
        HELP_REQUESTED
    );
    assert!(run(&Phase::background(), &["--help".to_owned()]).is_ok());
    assert!(run(
        &Phase::background(),
        &["--help".to_owned(), "--bogus".to_owned()]
    )
    .is_err());
    assert!(requested_top_level_help(&["--help".to_owned()])
        .unwrap()
        .contains("probe-ssh"));
    assert!(options::help_text("exec").contains("--secret-file"));
    assert!(run(
        &Phase::background(),
        &["report".to_owned(), "--help".to_owned()]
    )
    .is_ok());
}

#[test]
fn option_validation_matches_go_messages() {
    let base = [
        "exec",
        "--owner",
        "P02",
        "--revision",
        &"a".repeat(40),
        "--arch",
        "x86_64",
        "--target",
        "fixture",
        "--evidence",
        "/tmp/x",
    ]
    .iter()
    .map(|s| s.to_string())
    .collect::<Vec<_>>();
    let mut with_cmd = base.clone();
    with_cmd.extend(["--".to_string(), "true".to_string()]);
    assert!(parse_run_options(&with_cmd).is_ok());
    assert_eq!(
        parse_run_options(&with_cmd).unwrap().timeout,
        Duration::from_secs(30 * 60)
    );
    assert_eq!(
        parse_run_options(&base).err().unwrap().to_string(),
        "an existing owned check command is required"
    );
    let mut bad_owner = with_cmd.clone();
    bad_owner[2] = "P07".to_string();
    assert_eq!(
        parse_run_options(&bad_owner).err().unwrap().to_string(),
        "explicit owner required; P07/P08 are not independent tasks"
    );
    let mut bad_timeout = base.clone();
    bad_timeout.extend([
        "--timeout".to_string(),
        "25h".to_string(),
        "--".to_string(),
        "true".to_string(),
    ]);
    assert_eq!(
        parse_run_options(&bad_timeout).err().unwrap().to_string(),
        "revision, non-secret target and bounded timeout required"
    );
    let mut max_timeout = base.clone();
    max_timeout.extend([
        "--timeout=24h".to_owned(),
        "--".to_owned(),
        "true".to_owned(),
    ]);
    assert_eq!(
        parse_run_options(&max_timeout).unwrap().timeout,
        Duration::from_secs(24 * 3600)
    );
    let publish = ["publish"]
        .iter()
        .map(|s| s.to_string())
        .collect::<Vec<_>>();
    assert_eq!(
        parse_run_options(&publish).err().unwrap().to_string(),
        "no product/media/release workflow is implemented by this support tool"
    );
}

fn run_args(action: &str, evidence: &str) -> Vec<String> {
    [
        action,
        "--owner",
        "P07",
        "--revision",
        &"a".repeat(40),
        "--arch",
        "x86_64",
        "--target",
        "fixture",
        "--evidence",
        evidence,
    ]
    .iter()
    .map(|s| s.to_string())
    .collect()
}

/// Port of `TestInvalidActionsDoNotCreateEvidence` from `main_test.go`.
#[test]
fn invalid_actions_do_not_create_evidence() {
    for action in ["publish", "exec", "native", "vm", "probe-ssh"] {
        let scratch = TempDir::new("driver").unwrap();
        let path = scratch.join("evidence").to_string_lossy().into_owned();
        let args = run_args(action, &path);
        assert!(run(&Phase::background(), &args).is_err(), "{action}");
        assert!(std::fs::symlink_metadata(&path).is_err(), "{action}");
    }
}

/// Port of `TestCancelledExecutionRecordsFailure` from `main_test.go`.
#[test]
fn cancelled_execution_records_failure() {
    let scratch = TempDir::new("driver").unwrap();
    let path = scratch.join("evidence").to_string_lossy().into_owned();
    let root = Phase::background();
    root.cancel();
    let args = [
        "exec",
        "--owner",
        "P02",
        "--revision",
        &"a".repeat(40),
        "--arch",
        "x86_64",
        "--target",
        "fixture",
        "--evidence",
        &path,
        "--",
        "must-not-be-started",
    ]
    .iter()
    .map(|s| s.to_string())
    .collect::<Vec<_>>();
    assert!(run(&root, &args).is_err());
    let raw = std::fs::read(format!("{path}/observation.json")).unwrap();
    let text = String::from_utf8(raw).unwrap();
    assert!(text.contains("\"Outcome\": \"cancelled\""), "{text}");
    assert!(text.contains("\"Execution\": \"not-started\""), "{text}");
}

#[test]
fn evidence_failure_records_failed_evidence_and_outcome() {
    let scratch = TempDir::new("driver-evidence-failure").unwrap();
    let path = scratch.join("evidence").to_string_lossy().into_owned();
    let evidence = crate::evidence::create_evidence(&path, &[]).unwrap();

    let result = finalize_observation(
        &evidence,
        Observation::default(),
        None,
        Some(crate::error::Error::msg("synthetic capture failure")),
    );

    assert!(result.is_err());
    let text = std::fs::read_to_string(format!("{path}/observation.json")).unwrap();
    assert!(text.contains("\"Evidence\": \"failed\""), "{text}");
    assert!(text.contains("\"Outcome\": \"failed\""), "{text}");
    assert!(!text.contains("\"Outcome\": \"completed\""), "{text}");
}

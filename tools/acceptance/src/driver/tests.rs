use super::*;
use crate::files::TempDir;

#[test]
fn durations_match_go_terms() {
    assert_eq!(parse_duration("0").unwrap(), 0);
    assert_eq!(parse_duration("30m").unwrap(), 1_800_000_000_000);
    assert_eq!(parse_duration("1h30m").unwrap(), 5_400_000_000_000);
    assert_eq!(parse_duration("1.5h").unwrap(), 5_400_000_000_000);
    assert_eq!(parse_duration("300ms").unwrap(), 300_000_000);
    assert_eq!(parse_duration("-5s").unwrap(), -5_000_000_000);
    assert_eq!(parse_duration("2µs").unwrap(), 2_000);
    assert_eq!(parse_duration("2μs").unwrap(), 2_000);
    assert!(parse_duration("").is_err());
    assert!(parse_duration("1").is_err());
    assert!(parse_duration("1x").is_err());
    assert!(parse_duration("99999999999999999999h").is_err());
}

#[test]
fn flags_mirror_go_forms() {
    let args = [
        "--owner",
        "P02",
        "-revision=a",
        "--hold",
        "-secret-file",
        "one",
        "--secret-file=two",
        "--",
        "pos",
    ]
    .iter()
    .map(|s| s.to_string())
    .collect::<Vec<_>>();
    let parsed = parse_flags(
        &args,
        &[
            ("owner", FlagKind::Str),
            ("revision", FlagKind::Str),
            ("hold", FlagKind::Bool),
            ("secret-file", FlagKind::Repeat),
        ],
    )
    .unwrap();
    assert_eq!(flag_string(&parsed, "owner"), "P02");
    assert_eq!(flag_string(&parsed, "revision"), "a");
    assert_eq!(parsed.bools.get("hold"), Some(&true));
    assert_eq!(
        parsed.repeats.get("secret-file").unwrap(),
        &vec!["one".to_string(), "two".to_string()]
    );
    assert_eq!(parsed.positionals, vec!["pos".to_string()]);
    // Bools take no separate value; unknown flags and missing values fail.
    let args = ["-hold", "false"]
        .iter()
        .map(|s| s.to_string())
        .collect::<Vec<_>>();
    let parsed = parse_flags(&args, &[("hold", FlagKind::Bool)]).unwrap();
    assert_eq!(parsed.positionals, vec!["false".to_string()]);
    let args = ["-bogus"].iter().map(|s| s.to_string()).collect::<Vec<_>>();
    assert!(parse_flags(&args, &[("hold", FlagKind::Bool)]).is_err());
    let args = ["-owner"].iter().map(|s| s.to_string()).collect::<Vec<_>>();
    assert!(parse_flags(&args, &[("owner", FlagKind::Str)]).is_err());
    let args = ["-hold=maybe"]
        .iter()
        .map(|s| s.to_string())
        .collect::<Vec<_>>();
    assert!(parse_flags(&args, &[("hold", FlagKind::Bool)]).is_err());
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

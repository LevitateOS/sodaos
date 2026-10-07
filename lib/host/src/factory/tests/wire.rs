use std::sync::atomic::Ordering;

use super::common::{dummy_factory, sample_launch, sample_run, test_state_dir, TAG_COUNTER};
use super::mocks::{FakeBroker, FakeExec, FakeTerminal};
use crate::factory::deadline::{deadline_is_zero, parse_deadline};
use crate::factory::*;

#[test]
fn validators_match_go() {
    assert!(valid_factory_phase("approved"));
    assert!(valid_factory_phase("uncertain"));
    assert!(!valid_factory_phase("waiting"));
    assert!(!valid_factory_phase(""));
    assert!(valid_factory_run_id(&"a".repeat(32)));
    assert!(valid_factory_run_id(&"09af".repeat(8)));
    assert!(!valid_factory_run_id(&"A".repeat(32)));
    assert!(!valid_factory_run_id(&"a".repeat(31)));
    assert!(!valid_factory_run_id(""));
    assert!(valid_harness_version("v1"));
    assert!(valid_harness_version(&"a".repeat(32)));
    assert!(valid_harness_version("1.2-3_4"));
    assert!(!valid_harness_version(""));
    assert!(!valid_harness_version(&"a".repeat(33)));
    assert!(!valid_harness_version("-v1"));
    assert!(!valid_harness_version("v 1"));
    assert_eq!(
        factory_unit_name(&"a".repeat(32)),
        format!("soda-factory-{}.service", "a".repeat(32))
    );
    assert_eq!(factory_unit_name("bogus"), "");
    let (checkout, run_dir, home, codex) = factory_run_paths(
        "soda-coder",
        &format!("f{}", "c".repeat(24)),
        &"a".repeat(32),
    );
    assert_eq!(
        checkout,
        format!("/home/soda-coder/checkouts/f{}", "c".repeat(24))
    );
    assert!(run_dir.ends_with(&format!("/.soda-home/runs/{}", "a".repeat(32))));
    assert!(home.ends_with("/home"));
    assert!(codex.ends_with("/home/.codex"));
    assert_eq!(
        factory_run_paths("root", "f", "g"),
        (String::new(), String::new(), String::new(), String::new())
    );
    assert_eq!(
        takeover_destination("alice", &"a".repeat(32)),
        format!("/home/alice/factory-takeover/{}", "a".repeat(32))
    );
    assert_eq!(takeover_destination("root", &"a".repeat(32)), "");
    assert!(takeover_source(
        &format!("/home/soda-coder/checkouts/f{}", "c".repeat(24)),
        "soda-coder",
        &format!("f{}", "c".repeat(24))
    ));
    assert!(!takeover_source(
        "/home/soda-coder/checkouts/f",
        "soda-coder",
        "f"
    ));
    assert!(!takeover_source(
        "/home/soda-coder/checkouts/x/",
        "soda-coder",
        &format!("f{}", "c".repeat(24))
    ));
}

#[test]
fn run_and_launch_validation_pins_every_error() {
    let good = sample_run();
    assert!(good.validate().is_ok());
    let mut bad = good.clone();
    bad.id = "short".to_string();
    assert_eq!(bad.validate().unwrap_err(), "invalid factory run identity");
    bad = good.clone();
    bad.role = "root".to_string();
    assert_eq!(bad.validate().unwrap_err(), "invalid factory run identity");
    bad = good.clone();
    bad.preparation = "bogus".to_string();
    assert_eq!(
        bad.validate().unwrap_err(),
        "invalid run preparation reference"
    );
    bad = good.clone();
    bad.harness = "Muse".to_string();
    assert_eq!(bad.validate().unwrap_err(), "unsupported factory harness");
    bad = good.clone();
    bad.harness_vers = String::new();
    assert_eq!(bad.validate().unwrap_err(), "unsupported factory harness");
    bad = good.clone();
    bad.model = "x".repeat(129);
    assert_eq!(bad.validate().unwrap_err(), "invalid run model selection");
    bad = good.clone();
    bad.model = "gpt-\u{7f}".to_string();
    assert_eq!(bad.validate().unwrap_err(), "invalid run model selection");
    bad = good.clone();
    bad.model = "line\nbreak".to_string();
    assert_eq!(bad.validate().unwrap_err(), "invalid run model selection");
    bad = good.clone();
    bad.model = "gpt-5".to_string();
    assert!(bad.validate().is_ok());
    bad = good.clone();
    bad.assignment = "zz".to_string();
    assert_eq!(
        bad.validate().unwrap_err(),
        "invalid run assignment or source identity"
    );
    bad = good.clone();
    bad.connection = String::new();
    assert_eq!(bad.validate().unwrap_err(), "invalid run sponsorship");
    bad = good.clone();
    bad.actor = 0;
    assert_eq!(bad.validate().unwrap_err(), "invalid run sponsorship");
    bad = good.clone();
    bad.deadline = String::new();
    assert_eq!(bad.validate().unwrap_err(), "run deadline is required");
    bad = good.clone();
    bad.deadline = "0001-01-01T00:00:00Z".to_string();
    assert_eq!(bad.validate().unwrap_err(), "run deadline is required");
    bad = good.clone();
    bad.deadline = "not-a-time".to_string();
    assert_eq!(bad.validate().unwrap_err(), "run deadline is required");

    let launch = sample_launch();
    assert!(launch.validate().is_ok());
    let mut bad_launch = launch.clone();
    bad_launch.prompt = Vec::new();
    assert_eq!(
        bad_launch.validate().unwrap_err(),
        "invalid run prompt size"
    );
    bad_launch = launch.clone();
    bad_launch.prompt = vec![b'x'; MAX_FACTORY_PROMPT + 1];
    assert_eq!(
        bad_launch.validate().unwrap_err(),
        "invalid run prompt size"
    );
    bad_launch = launch.clone();
    bad_launch.prompt = b"something else".to_vec();
    assert_eq!(
        bad_launch.validate().unwrap_err(),
        "prompt bytes do not match their digest"
    );
    bad_launch = launch.clone();
    bad_launch.harness_sha256 = "zz".to_string();
    assert_eq!(bad_launch.validate().unwrap_err(), "invalid harness pin");
}

#[test]
fn deadline_parser_uses_strict_shared_wire_shape() {
    assert!(parse_deadline("2030-01-02T03:04:05Z").is_some());
    assert!(parse_deadline("2030-01-02T03:04:05.123456789Z").is_some());
    assert!(parse_deadline("2030-01-02T03:04:05.1+02:00").is_some());
    assert!(parse_deadline("2030-01-02T03:04:05-05:30").is_some());
    // Same instant, different offsets.
    assert_eq!(
        parse_deadline("2030-01-02T03:04:05Z"),
        parse_deadline("2030-01-02T05:04:05+02:00")
    );
    // Fraction precision.
    let whole = parse_deadline("2030-01-02T03:04:05Z").unwrap();
    let frac = parse_deadline("2030-01-02T03:04:05.000000001Z").unwrap();
    assert_eq!(frac - whole, 1);
    // Rejections.
    for bad in [
        "",
        "2030-01-02",
        "2030-01-02T03:04:05",
        "2030-13-02T03:04:05Z",
        "2030-01-32T03:04:05Z",
        "2029-02-29T03:04:05Z",
        "2030-01-02T24:04:05Z",
        "2030-01-02T03:60:05Z",
        "2030-01-02T03:04:60Z",
        "2030-01-02T03:04:05.Z",
        "2030-01-02T03:04:05.1234567890Z",
        "2030-01-02T03:04:05+24:00",
        "2030-01-02T03:04:05+02:60",
        "2030-01-02T03:04:05+0200",
        "2030-01-02 03:04:05Z",
        "2030-01-02T3:04:05Z",
        "2030-01-02T03:04:05z",
    ] {
        assert!(parse_deadline(bad).is_none(), "accepted {bad:?}");
    }
    assert_eq!(parse_deadline("1969-12-31T23:59:59Z"), Some(-1_000_000_000));
    assert!(deadline_is_zero("0001-01-01T00:00:00Z"));
    assert!(!deadline_is_zero("0001-01-01T00:00:01Z"));
    assert!(!deadline_is_zero("garbage"));
}

#[test]
fn error_messages_are_exact() {
    assert_eq!(
        FactoryError::NotFound.message(),
        "identity execution missing"
    );
    assert_eq!(FactoryError::Stale.message(), "identity generation changed");
    assert_eq!(FactoryError::Busy.message(), "subscription is in use");
    assert_eq!(FactoryError::Denied.message(), "identity authority denied");
    assert_eq!(
        FactoryError::Uncertain.message(),
        "subscription requires reconnection"
    );
    assert_eq!(
        FactoryError::DeadlineExceeded.message(),
        "context deadline exceeded"
    );
    assert_eq!(
        FactoryError::ExportCandidate.message(),
        "export candidate is not recorded"
    );
    assert_eq!(
        FactoryError::ExportBounds.message(),
        "candidate export exceeds bounds"
    );
    assert_eq!(ERR_RUN_NOT_FOUND, "factory run not found");
    assert_eq!(ERR_RUN_STALE, "factory run incarnation changed");
    assert_eq!(FACTORY_STATE_ROOT, "/var/lib/soda/host/factory");
}

#[test]
fn open_factory_matrix() {
    use std::rc::Rc;
    let term = Rc::new(FakeTerminal::new());
    let broker = Rc::new(FakeBroker::new());
    let exec = Rc::new(FakeExec::new(Vec::new()));
    assert_eq!(
        Factory::open_factory("relative/path", exec.clone(), term.clone(), broker.clone())
            .err()
            .unwrap(),
        "factory receipt directory must be absolute"
    );
    assert!(Factory::open_factory(
        "/tmp/soda-pf26-definitely-missing",
        exec.clone(),
        term.clone(),
        broker.clone()
    )
    .is_err());
    // A regular file is not a directory.
    let file = test_state_dir("open-file").join("f");
    std::fs::write(&file, b"x").unwrap();
    assert_eq!(
        Factory::open_factory(
            file.to_str().unwrap(),
            exec.clone(),
            term.clone(),
            broker.clone()
        )
        .err()
        .unwrap(),
        "factory receipts must live in a private directory"
    );
    // Group/world-readable directories are refused.
    let open = test_state_dir("open-perms");
    std::fs::set_permissions(&open, std::fs::Permissions::from_mode(0o755)).unwrap();
    assert_eq!(
        Factory::open_factory(
            open.to_str().unwrap(),
            exec.clone(),
            term.clone(),
            broker.clone()
        )
        .err()
        .unwrap(),
        "factory receipts must live in a private directory"
    );
    // A symlink is not a directory (Lstat semantics).
    let target = test_state_dir("open-target");
    let link = std::env::temp_dir().join(format!(
        "soda-pf26-open-link-{}-{}",
        std::process::id(),
        TAG_COUNTER.fetch_add(1, Ordering::SeqCst)
    ));
    std::os::unix::fs::symlink(&target, &link).unwrap();
    assert_eq!(
        Factory::open_factory(
            link.to_str().unwrap(),
            exec.clone(),
            term.clone(),
            broker.clone()
        )
        .err()
        .unwrap(),
        "factory receipts must live in a private directory"
    );
    std::fs::remove_file(&link).unwrap();
    // Private directory opens.
    let dir = test_state_dir("open-ok");
    assert!(Factory::open_factory(dir.to_str().unwrap(), exec, term, broker).is_ok());
}

#[test]
fn harness_pin_shape_matches_go() {
    let dir = test_state_dir("harness");
    let factory = dummy_factory(&dir);
    let pin = factory.harness_pin();
    assert_eq!(pin.harness, "codex");
    assert_eq!(pin.version, "v1");
    assert_eq!(pin.sha256, "f".repeat(64));
    // The daemon route fills the image from its own configuration.
    assert_eq!(pin.image, "");
    assert_eq!(
        pin.encode(),
        format!(
            "{{\"harness\":\"codex\",\"version\":\"v1\",\"sha256\":\"{}\",\"image\":\"\"}}",
            "f".repeat(64)
        )
    );
    // Unpinned harness is unusable, like the Go client check.
    assert_eq!(
        confirm_factory_harness(&pin).unwrap_err(),
        "native factory harness pin is not usable"
    );
    let mut pinned = pin;
    pinned.image = format!("sha256:{}", "a".repeat(64));
    assert!(confirm_factory_harness(&pinned).is_ok());
}

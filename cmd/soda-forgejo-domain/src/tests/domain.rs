use std::fs;
use std::os::unix::fs::PermissionsExt;

use super::fixtures::*;

use crate::config::marker_path;

#[test]
fn stop_verifies_quiescence_before_returning() {
    let fx = valid_ini();
    let mut sys = FakeSys::new();
    let (result, stdout) = run_verb("stop", &fx, &mut sys);
    assert!(result.is_ok(), "{result:?}");
    assert!(
        stdout.contains("stopped: forgejo.service inactive"),
        "{stdout}"
    );
    assert!(sys
        .calls
        .iter()
        .any(|c| c == &["systemctl", "stop", "forgejo.service"]));
}

#[test]
fn stop_reports_survivors_after_timeout() {
    let fx = valid_ini();
    let mut sys = FakeSys::new();
    sys.active = true;
    sys.podman_out = "soda-forgejo\n".to_string();
    sys.now_values = vec![0.0, 61.0, 61.0, 61.0];
    let (result, _) = run_verb("stop", &fx, &mut sys);
    let err = result.expect_err("survivors");
    assert!(err.contains("unit forgejo.service still active"), "{err}");
    assert!(
        err.contains("container soda-forgejo still present"),
        "{err}"
    );
}

#[test]
fn inhibit_requires_quiescence_then_masks_and_marks() {
    let fx = valid_ini();
    let mut sys = FakeSys::new();
    sys.active = true;
    let (result, _) = run_verb("inhibit", &fx, &mut sys);
    assert!(result.expect_err("running").contains("still running"));
    let mut sys = FakeSys::new();
    let marker = marker_path(&fx.paths).expect("marker");
    fs::create_dir_all(marker.parent().expect("parent")).expect("parent");
    let (result, stdout) = run_verb("inhibit", &fx, &mut sys);
    assert!(result.is_ok(), "{result:?}");
    assert!(
        stdout.contains("inhibited: forgejo.service masked"),
        "{stdout}"
    );
    assert!(sys
        .calls
        .iter()
        .any(|c| c == &["systemctl", "mask", "--runtime", "forgejo.service"]));
    let marker = marker_path(&fx.paths).expect("marker");
    assert_eq!(
        fs::read_to_string(&marker).expect("marker"),
        "offline recovery\n"
    );
    assert_eq!(
        fs::metadata(&marker).expect("m").permissions().mode() & 0o777,
        0o600
    );
}

#[test]
fn status_reads_all_four_signals() {
    let fx = valid_ini();
    let mut sys = FakeSys::new();
    sys.enabled_out = "masked\n".to_string();
    sys.podman_out = "other\nsoda-forgejo\n".to_string();
    let (result, stdout) = run_verb("status", &fx, &mut sys);
    assert!(result.is_ok(), "{result:?}");
    assert!(
        stdout.contains("unit: forgejo.service inactive"),
        "{stdout}"
    );
    assert!(stdout.contains("masked: yes"), "{stdout}");
    assert!(
        stdout.contains("container: soda-forgejo present"),
        "{stdout}"
    );
    assert!(stdout.contains("marker: absent"), "{stdout}");
}

#[test]
fn lift_removes_marker_and_unmasks() {
    let fx = valid_ini();
    let marker = marker_path(&fx.paths).expect("marker");
    fs::create_dir_all(marker.parent().expect("parent")).expect("parent");
    fs::write(&marker, "offline recovery\n").expect("marker");
    let mut sys = FakeSys::new();
    let (result, stdout) = run_verb("lift", &fx, &mut sys);
    assert!(result.is_ok(), "{result:?}");
    assert!(stdout.contains("marker removed"), "{stdout}");
    assert!(stdout.contains("unmasked: forgejo.service"), "{stdout}");
    assert!(!marker.exists());
    // Unknown-marker refusal moved to lift_refuses_unknown_marker_before_unmask (O04-F1).
}

#[test]
fn start_refuses_inhibited_and_masked() {
    let fx = valid_ini();
    let marker = marker_path(&fx.paths).expect("marker");
    fs::create_dir_all(marker.parent().expect("parent")).expect("parent");
    fs::write(&marker, "offline recovery\n").expect("marker");
    let mut sys = FakeSys::new();
    let (result, _) = run_verb("start", &fx, &mut sys);
    assert!(result.expect_err("inhibited").contains("restart inhibited"));
    fs::remove_file(&marker).expect("remove");
    let mut sys = FakeSys::new();
    sys.enabled_out = "masked\nmasked-runtime\n".to_string();
    let (result, _) = run_verb("start", &fx, &mut sys);
    assert!(result.expect_err("masked").contains("is masked"));
    let mut sys = FakeSys::new();
    let (result, stdout) = run_verb("start", &fx, &mut sys);
    assert!(result.is_ok(), "{result:?}");
    assert!(
        stdout.contains("started: forgejo.service active"),
        "{stdout}"
    );
    // Start failure and never-active are distinct errors.
    let mut sys = FakeSys::new();
    sys.start_code = 1;
    let (result, _) = run_verb("start", &fx, &mut sys);
    assert!(result
        .expect_err("start-fail")
        .contains("systemctl start failed"));
}

#[test]
fn lift_refuses_unknown_marker_before_unmask() {
    // O04-F1: an unresolvable marker mapping must refuse before any
    // unmask mutation; it must not read as "already absent".
    let fx = fixture(Some("APP_NAME = Soda\n"), None);
    let mut sys = FakeSys::new();
    let (result, stdout) = run_verb("lift", &fx, &mut sys);
    let err = result.expect_err("lift refuses unknown marker");
    assert!(err.contains("app.ini"), "{err}");
    assert!(sys.calls.is_empty(), "no mutation: {:?}", sys.calls);
    assert!(!stdout.contains("already absent"), "{stdout}");
    assert!(!stdout.contains("unmasked"), "{stdout}");
}

#[test]
fn start_refuses_unknown_marker_before_start() {
    // O04-F1: an unresolvable marker mapping must refuse before any
    // start mutation; it must not skip the inhibited precondition.
    let fx = fixture(Some("APP_NAME = Soda\n"), None);
    let mut sys = FakeSys::new();
    let (result, stdout) = run_verb("start", &fx, &mut sys);
    let err = result.expect_err("start refuses unknown marker");
    assert!(err.contains("app.ini"), "{err}");
    assert!(sys.calls.is_empty(), "no mutation: {:?}", sys.calls);
    assert!(!stdout.contains("started:"), "{stdout}");
}

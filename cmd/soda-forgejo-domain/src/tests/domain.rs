use std::fs;
use std::os::unix::fs::{MetadataExt, PermissionsExt};

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
    let marker = marker_path(&fx.paths).expect("marker").path().to_path_buf();
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
    let marker = marker_path(&fx.paths).expect("marker").path().to_path_buf();
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
    let marker = marker_path(&fx.paths).expect("marker").path().to_path_buf();
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
    let marker = marker_path(&fx.paths).expect("marker").path().to_path_buf();
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

#[test]
fn inhibit_and_lift_refuse_marker_paths_that_escape_data_root() {
    let fx = fixture(Some("[server]\nAPP_DATA_PATH = /data/../outside\n"), None);
    let outside = fx.temp.join("outside");
    fs::create_dir_all(&outside).expect("outside directory");
    let marker = outside.join(crate::system::MARKER_NAME);
    fs::write(&marker, "preserve outside marker\n").expect("outside marker");

    let mut sys = FakeSys::new();
    let (result, _) = run_verb("inhibit", &fx, &mut sys);
    assert!(
        result.is_err(),
        "inhibit must refuse an escaping marker path"
    );
    assert_eq!(
        fs::read(&marker).expect("outside marker remains"),
        b"preserve outside marker\n"
    );

    let mut sys = FakeSys::new();
    let (result, _) = run_verb("lift", &fx, &mut sys);
    assert!(result.is_err(), "lift must refuse an escaping marker path");
    assert!(
        !sys.calls.iter().any(|call| {
            call.first().map(String::as_str) == Some("systemctl")
                && call.get(1).map(String::as_str) == Some("unmask")
        }),
        "lift must not unmask: {:?}",
        sys.calls
    );
    assert_eq!(
        fs::read(&marker).expect("outside marker remains"),
        b"preserve outside marker\n"
    );
}

#[test]
fn inhibit_and_lift_refuse_marker_paths_through_intermediate_symlink() {
    let fx = fixture(Some("[server]\nAPP_DATA_PATH = /data/link/subdir\n"), None);
    let outside = fx.temp.join("outside");
    fs::create_dir_all(outside.join("subdir")).expect("outside subdir");
    std::os::unix::fs::symlink(&outside, fx.paths.data_root.join("link")).expect("link");
    let marker = outside.join("subdir").join(crate::system::MARKER_NAME);
    fs::write(&marker, "preserve outside marker\n").expect("outside marker");

    let mut sys = FakeSys::new();
    let (result, _) = run_verb("inhibit", &fx, &mut sys);
    assert!(result.is_err(), "inhibit must refuse an ancestor symlink");
    assert_eq!(
        fs::read(&marker).expect("outside marker remains"),
        b"preserve outside marker\n"
    );

    let mut sys = FakeSys::new();
    let (result, _) = run_verb("lift", &fx, &mut sys);
    assert!(result.is_err(), "lift must refuse an ancestor symlink");
    assert!(
        !sys.calls.iter().any(|call| {
            call.first().map(String::as_str) == Some("systemctl")
                && call.get(1).map(String::as_str) == Some("unmask")
        }),
        "lift must not unmask: {:?}",
        sys.calls
    );
    assert_eq!(
        fs::read(&marker).expect("outside marker remains"),
        b"preserve outside marker\n"
    );
}

#[test]
fn marker_status_and_lifecycle_refuse_unobservable_parent() {
    let fx = fixture(
        Some("[server]\nAPP_DATA_PATH = /data/parent/subdir\n"),
        None,
    );
    fs::write(fx.paths.data_root.join("parent"), "not a directory").expect("file parent");

    let mut inhibit_sys = FakeSys::new();
    let (inhibit, _) = run_verb("inhibit", &fx, &mut inhibit_sys);
    assert!(
        inhibit.is_err(),
        "inhibit must refuse an unobservable marker parent"
    );

    let mut lift_sys = FakeSys::new();
    let (lift, _) = run_verb("lift", &fx, &mut lift_sys);
    assert!(lift.is_err(), "lift must not treat an I/O error as absent");
    assert!(
        !lift_sys.calls.iter().any(|call| {
            call.first().map(String::as_str) == Some("systemctl")
                && call.get(1).map(String::as_str) == Some("unmask")
        }),
        "lift must not unmask: {:?}",
        lift_sys.calls
    );

    let mut start_sys = FakeSys::new();
    let (start, _) = run_verb("start", &fx, &mut start_sys);
    assert!(
        start.is_err(),
        "start must not treat an I/O error as absent"
    );
    assert!(
        !start_sys.calls.iter().any(|call| {
            call.first().map(String::as_str) == Some("systemctl")
                && call.get(1).map(String::as_str) == Some("start")
        }),
        "start must not run: {:?}",
        start_sys.calls
    );

    let mut status_sys = FakeSys::new();
    let (status, stdout) = run_verb("status", &fx, &mut status_sys);
    assert!(status.is_err(), "status must report an unobservable marker");
    assert!(
        !stdout.contains("marker: absent"),
        "an I/O error is not known absence: {stdout}"
    );
}

#[test]
fn marker_symlink_refuses_all_marker_lifecycle_verbs() {
    let fx = fixture(Some("[server]\nAPP_DATA_PATH = /data/soda\n"), None);
    let data_dir = fx.paths.data_root.join("soda");
    fs::create_dir_all(&data_dir).expect("data directory");
    let outside = fx.temp.join("outside-marker");
    fs::write(&outside, "preserve symlink target\n").expect("outside marker target");
    std::os::unix::fs::symlink(&outside, data_dir.join(crate::system::MARKER_NAME))
        .expect("marker symlink");

    let mut inhibit_sys = FakeSys::new();
    let (inhibit, _) = run_verb("inhibit", &fx, &mut inhibit_sys);
    assert!(
        inhibit.is_err(),
        "inhibit must refuse a final marker symlink"
    );

    let mut status_sys = FakeSys::new();
    let (status, stdout) = run_verb("status", &fx, &mut status_sys);
    assert!(status.is_err(), "status must refuse a final marker symlink");
    assert!(stdout.contains("marker: unknown"), "{stdout}");
    assert!(!stdout.contains("marker: absent"), "{stdout}");

    let mut lift_sys = FakeSys::new();
    let (lift, _) = run_verb("lift", &fx, &mut lift_sys);
    assert!(lift.is_err(), "lift must refuse a final marker symlink");
    assert!(
        !lift_sys.calls.iter().any(|call| {
            call.first().map(String::as_str) == Some("systemctl")
                && call.get(1).map(String::as_str) == Some("unmask")
        }),
        "lift must not unmask: {:?}",
        lift_sys.calls
    );

    let mut start_sys = FakeSys::new();
    let (start, _) = run_verb("start", &fx, &mut start_sys);
    assert!(start.is_err(), "start must refuse a final marker symlink");
    assert!(
        !start_sys.calls.iter().any(|call| {
            call.first().map(String::as_str) == Some("systemctl")
                && call.get(1).map(String::as_str) == Some("start")
        }),
        "start must not run: {:?}",
        start_sys.calls
    );
    assert_eq!(
        fs::read(&outside).expect("outside target remains"),
        b"preserve symlink target\n"
    );
}

#[test]
fn nonregular_marker_refuses_lifecycle_verbs() {
    let fx = fixture(Some("[server]\nAPP_DATA_PATH = /data/soda\n"), None);
    let marker = fx
        .paths
        .data_root
        .join("soda")
        .join(crate::system::MARKER_NAME);
    fs::create_dir_all(&marker).expect("nonregular marker directory");

    let mut inhibit_sys = FakeSys::new();
    let (inhibit, _) = run_verb("inhibit", &fx, &mut inhibit_sys);
    assert!(inhibit.is_err(), "inhibit must refuse a nonregular marker");

    let mut status_sys = FakeSys::new();
    let (status, stdout) = run_verb("status", &fx, &mut status_sys);
    assert!(status.is_err(), "status must refuse a nonregular marker");
    assert!(stdout.contains("marker: unknown"), "{stdout}");

    let mut lift_sys = FakeSys::new();
    let (lift, _) = run_verb("lift", &fx, &mut lift_sys);
    assert!(lift.is_err(), "lift must refuse a nonregular marker");
    assert!(
        !lift_sys.calls.iter().any(|call| {
            call.first().map(String::as_str) == Some("systemctl")
                && call.get(1).map(String::as_str) == Some("unmask")
        }),
        "lift must not unmask: {:?}",
        lift_sys.calls
    );

    let mut start_sys = FakeSys::new();
    let (start, _) = run_verb("start", &fx, &mut start_sys);
    assert!(start.is_err(), "start must refuse a nonregular marker");
    assert!(
        !start_sys.calls.iter().any(|call| {
            call.first().map(String::as_str) == Some("systemctl")
                && call.get(1).map(String::as_str) == Some("start")
        }),
        "start must not run: {:?}",
        start_sys.calls
    );
}

#[test]
fn existing_regular_marker_is_left_untouched_by_inhibit() {
    let fx = valid_ini();
    let marker = fx
        .paths
        .data_root
        .join("soda")
        .join(crate::system::MARKER_NAME);
    fs::create_dir_all(marker.parent().expect("marker parent")).expect("parent");
    let outside = fx.temp.join("hardlink-target");
    fs::write(&outside, "existing marker bytes\n").expect("target");
    fs::set_permissions(&outside, fs::Permissions::from_mode(0o640)).expect("target mode");
    fs::hard_link(&outside, &marker).expect("marker hardlink");
    let before = fs::metadata(&outside).expect("before metadata");

    let mut sys = FakeSys::new();
    let (result, _) = run_verb("inhibit", &fx, &mut sys);
    assert!(
        result.is_ok(),
        "existing regular marker is already inhibited: {result:?}"
    );
    assert_eq!(
        fs::read(&outside).expect("outside bytes"),
        b"existing marker bytes\n"
    );
    assert_eq!(
        fs::metadata(&outside)
            .expect("after metadata")
            .permissions()
            .mode()
            & 0o777,
        0o640
    );
    let after = fs::metadata(&outside).expect("after metadata");
    assert_eq!(before.ino(), after.ino());
    assert_eq!(before.dev(), after.dev());
}

#[test]
fn missing_marker_parent_is_absent_for_readers_but_cannot_be_created() {
    let fx = valid_ini();
    let marker = fx
        .paths
        .data_root
        .join("soda")
        .join(crate::system::MARKER_NAME);

    let mut inhibit_sys = FakeSys::new();
    let (inhibit, _) = run_verb("inhibit", &fx, &mut inhibit_sys);
    assert!(
        inhibit.is_err(),
        "inhibit must not create missing parent directories"
    );
    assert!(inhibit_sys.calls.iter().any(|call| {
        call.first().map(String::as_str) == Some("systemctl")
            && call.get(1).map(String::as_str) == Some("mask")
    }));
    assert!(!marker.parent().expect("parent").exists());

    let mut status_sys = FakeSys::new();
    let (status, stdout) = run_verb("status", &fx, &mut status_sys);
    assert!(
        status.is_ok(),
        "known-missing parent means absent marker: {status:?}"
    );
    assert!(stdout.contains("marker: absent"), "{stdout}");

    let mut lift_sys = FakeSys::new();
    let (lift, stdout) = run_verb("lift", &fx, &mut lift_sys);
    assert!(
        lift.is_ok(),
        "known-missing parent means already absent: {lift:?}"
    );
    assert!(stdout.contains("marker already absent"), "{stdout}");

    let mut start_sys = FakeSys::new();
    let (start, _) = run_verb("start", &fx, &mut start_sys);
    assert!(
        start.is_ok(),
        "known-missing parent does not inhibit start: {start:?}"
    );
}

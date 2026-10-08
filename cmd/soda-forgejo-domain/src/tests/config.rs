use std::fs;
use std::os::unix::fs::PermissionsExt;

use super::fixtures::*;

use crate::config::{app_data_path, marker_path};
use crate::system::MARKER_NAME;

#[test]
fn bare_app_name_status_reports_clear_error() {
    let fx = fixture(Some("APP_NAME = Soda\n"), None);
    let mut sys = FakeSys::new();
    let (result, stdout) = run_verb("status", &fx, &mut sys);
    assert!(result.is_err());
    assert!(stdout.contains("marker: unknown"), "{stdout}");
    let err = result.expect_err("status fails");
    assert!(err.contains("app.ini"), "{err}");
    assert!(!stdout.contains("Traceback"));
    assert!(!err.contains("Traceback"));
}

#[test]
fn bare_app_name_start_has_no_crash() {
    let fx = fixture(Some("APP_NAME = Soda\n"), None);
    let mut sys = FakeSys::new();
    let (result, stdout) = run_verb("start", &fx, &mut sys);
    let _ = result;
    assert!(!stdout.contains("Traceback"));
}

#[test]
fn valid_app_ini_still_resolves() {
    let fx = valid_ini();
    assert_eq!(app_data_path(&fx.paths).expect("ini"), "/data/soda");
    fs::write(
        &fx.paths.env_file,
        "FORGEJO__server__APP_DATA_PATH=/data/override\n",
    )
    .expect("env");
    assert_eq!(
        app_data_path(&fx.paths).expect("override"),
        "/data/override"
    );
}

#[test]
fn parsed_but_missing_app_data_path_reports_clear_error() {
    let fx = fixture(Some("[server]\nAPP_NAME = Soda\n"), None);
    let err = app_data_path(&fx.paths).expect_err("missing");
    assert!(err.contains("not set"), "{err}");
}

#[test]
fn ini_shapes_match_configparser() {
    // (content, expected value or "parse-error"/"missing")
    for (content, expected) in [
        ("[server] trailing\nAPP_DATA_PATH = /data\n", Ok("/data")),
        ("[server]\nAPP_DATA_PATH : /data\n", Ok("/data")),
        ("[Server]\nAPP_DATA_PATH = /data\n", Err("nosection")),
        ("[server]\nApp_Data_Path = /data\n", Ok("/data")),
        ("[server]\njustwords\n", Err("parse")),
        (
            "[DEFAULT]\nAPP_DATA_PATH = /dflt\n[server]\nX = 1\n",
            Ok("/dflt"),
        ),
        ("[server]\nA = 1\nA = 2\n", Err("parse")),
        ("[server]\nA = 1\n[server]\nB = 2\n", Err("parse")),
        ("[server]\nAPP_DATA_PATH = /da\n ta\n", Ok("/da\nta")),
        ("[server]\nAPP_DATA_PATH = /da%ta\n", Err("parse")),
        (
            "[server]\nBASE = /d\nAPP_DATA_PATH = %(BASE)s/x\n",
            Ok("/d/x"),
        ),
        ("[server]\nAPP_DATA_PATH = %(NOPE)s/x\n", Err("parse")),
        ("[server]\n  # note\nAPP_DATA_PATH = /data\n", Ok("/data")),
        (
            "[server]\nAPP_DATA_PATH = /data ; note\n",
            Ok("/data ; note"),
        ),
        ("[server]\nAPP_DATA_PATH =\n", Err("missing")),
        ("[]\nA = 1\n", Err("parse")),
        ("[other]\nA = 1\n", Err("nosection")),
    ] {
        let fx = fixture(Some(content), None);
        match expected {
            Ok(want) => assert_eq!(
                app_data_path(&fx.paths).expect(content),
                want,
                "{content:?}"
            ),
            Err("missing") => assert!(app_data_path(&fx.paths)
                .expect_err(content)
                .contains("not set")),
            Err("nosection") => assert!(app_data_path(&fx.paths)
                .expect_err(content)
                .contains("No section")),
            _ => assert!(app_data_path(&fx.paths)
                .expect_err(content)
                .contains("cannot parse")),
        }
    }
    // Env override wins and strips one layer of quotes; last match wins.
    let fx = fixture(Some("[server]\nAPP_DATA_PATH = /data/ini\n"), Some("FORGEJO__server__APP_DATA_PATH=\"/data/first\"\nFORGEJO__server__APP_DATA_PATH='/data/second'\n"));
    assert_eq!(app_data_path(&fx.paths).expect("override"), "/data/second");
    // Relative values refuse to resolve.
    let fx = fixture(Some("[server]\nAPP_DATA_PATH = data/rel\n"), None);
    assert!(app_data_path(&fx.paths)
        .expect_err("relative")
        .contains("relative"));
    // Missing app.ini with no override is explicit.
    let fx = fixture(None, None);
    assert!(app_data_path(&fx.paths)
        .expect_err("absent")
        .contains("not found"));
}

#[test]
fn marker_mapping_rejects_outside_volume_and_symlinks() {
    let fx = fixture(Some("[server]\nAPP_DATA_PATH = /data/soda\n"), None);
    let marker = marker_path(&fx.paths).expect("marker");
    assert_eq!(
        marker.path(),
        fx.paths.data_root.join("soda").join(MARKER_NAME)
    );
    let fx = fixture(Some("[server]\nAPP_DATA_PATH = /etc/soda\n"), None);
    assert!(marker_path(&fx.paths)
        .expect_err("outside")
        .contains("outside"));
    // Symlinked marker parent is refused.
    let fx = fixture(Some("[server]\nAPP_DATA_PATH = /data/soda\n"), None);
    let target = fx.temp.join("target");
    fs::create_dir_all(&target).expect("target");
    std::os::unix::fs::symlink(&target, fx.paths.data_root.join("soda")).expect("link");
    assert!(
        marker_path(&fx.paths).is_err(),
        "symlinked parent must refuse"
    );
}

#[test]
fn marker_mapping_rejects_parent_traversal() {
    let fx = fixture(Some("[server]\nAPP_DATA_PATH = /data/../outside\n"), None);
    assert!(
        marker_path(&fx.paths).is_err(),
        "parent traversal must be refused"
    );
}

#[test]
fn marker_mapping_rejects_intermediate_symlink_ancestor() {
    let fx = fixture(Some("[server]\nAPP_DATA_PATH = /data/link/subdir\n"), None);
    let outside = fx.temp.join("outside");
    fs::create_dir_all(outside.join("subdir")).expect("outside subdir");
    std::os::unix::fs::symlink(&outside, fx.paths.data_root.join("link")).expect("link");

    assert!(
        marker_path(&fx.paths).is_err(),
        "intermediate symlink must be refused"
    );
}

#[test]
fn marker_mapping_rejects_symlinked_data_root() {
    let fx = fixture(Some("[server]\nAPP_DATA_PATH = /data/soda\n"), None);
    let outside = fx.temp.join("outside-root");
    fs::create_dir_all(outside.join("soda")).expect("outside marker parent");
    let held = fx.temp.join("held-data-root");
    fs::rename(&fx.paths.data_root, &held).expect("move data root");
    std::os::unix::fs::symlink(&outside, &fx.paths.data_root).expect("replace data root");

    assert!(
        marker_path(&fx.paths).is_err(),
        "symlinked data root must be refused"
    );
}

#[test]
fn retained_marker_parent_descriptor_survives_path_replacement() {
    let fx = fixture(Some("[server]\nAPP_DATA_PATH = /data/soda\n"), None);
    let admitted = fx.paths.data_root.join("soda");
    fs::create_dir_all(&admitted).expect("admitted directory");
    let outside = fx.temp.join("outside");
    fs::create_dir_all(&outside).expect("outside directory");
    let outside_marker = outside.join(MARKER_NAME);
    fs::write(&outside_marker, "outside stays intact\n").expect("outside marker");

    let marker = marker_path(&fx.paths).expect("admitted marker location");
    assert!(!marker.is_present().expect("initial marker state"));
    let held = fx.paths.data_root.join("held-soda");
    fs::rename(&admitted, &held).expect("rename admitted directory");
    std::os::unix::fs::symlink(&outside, &admitted).expect("replace old path with symlink");

    marker
        .create()
        .expect("create through retained directory descriptor");
    assert_eq!(
        fs::read(held.join(MARKER_NAME)).expect("marker in held directory"),
        b"offline recovery\n"
    );
    assert_eq!(
        fs::read(&outside_marker).expect("outside marker remains"),
        b"outside stays intact\n"
    );
    assert_eq!(
        fs::metadata(held.join(MARKER_NAME))
            .expect("created marker metadata")
            .permissions()
            .mode()
            & 0o777,
        0o600
    );
    assert!(marker
        .is_present()
        .expect("presence through retained descriptor"));
    assert!(marker.remove().expect("remove through retained descriptor"));
    assert!(!marker
        .is_present()
        .expect("absent through retained descriptor"));
    assert!(!held.join(MARKER_NAME).exists());
    assert_eq!(
        fs::read(&outside_marker).expect("outside marker remains after remove"),
        b"outside stays intact\n"
    );
}

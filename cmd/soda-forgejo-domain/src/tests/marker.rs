use std::fs;
use std::os::unix::fs::PermissionsExt;

use super::fixtures::*;

use crate::marker::{declared_app_data_path, marker_path, HOST_UNIT};
use crate::system::MARKER_NAME;

#[test]
fn fixed_data_path_comes_from_the_forgejo_unit() {
    let fx = fixture();
    assert_eq!(
        declared_app_data_path(HOST_UNIT).expect("unit declaration"),
        "/data/gitea"
    );
    assert_eq!(
        marker_path(&fx.paths).expect("marker").path(),
        fx.paths.data_root.join("gitea").join(MARKER_NAME)
    );
}

#[test]
fn data_path_declaration_must_be_unique_and_in_container_section() {
    assert!(declared_app_data_path("").is_err());
    assert!(declared_app_data_path(
        "[Service]\nEnvironment=FORGEJO__server__APP_DATA_PATH=/data/gitea\n"
    )
    .is_err());
    assert!(declared_app_data_path(
        "[Container]\nEnvironment=FORGEJO__server__APP_DATA_PATH=/data/gitea\nEnvironment=FORGEJO__server__APP_DATA_PATH=/data/other\n"
    )
    .is_err());
}

#[test]
fn marker_mapping_rejects_symlinked_data_root() {
    let fx = fixture();
    let target = fx.temp.join("target");
    fs::create_dir_all(&target).expect("target");
    let held = fx.temp.join("held-data-root");
    fs::rename(&fx.paths.data_root, &held).expect("move data root");
    std::os::unix::fs::symlink(&target, &fx.paths.data_root).expect("replace data root");
    assert!(
        marker_path(&fx.paths).is_err(),
        "symlinked data root must refuse"
    );
}

#[test]
fn retained_marker_parent_descriptor_survives_path_replacement() {
    let fx = fixture();
    let admitted = fx.paths.data_root.join("gitea");
    fs::create_dir_all(&admitted).expect("admitted directory");
    let outside = fx.temp.join("outside");
    fs::create_dir_all(&outside).expect("outside directory");
    let outside_marker = outside.join(MARKER_NAME);
    fs::write(&outside_marker, "outside stays intact\n").expect("outside marker");

    let marker = marker_path(&fx.paths).expect("admitted marker location");
    assert!(!marker.is_present().expect("initial marker state"));
    let held = fx.temp.join("held-gitea");
    fs::rename(&admitted, &held).expect("rename admitted directory");
    std::os::unix::fs::symlink(&outside, &admitted).expect("replace old path with symlink");
    assert!(
        marker_path(&fx.paths).is_err(),
        "new lookup must refuse symlink"
    );

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

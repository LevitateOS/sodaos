use super::super::payload::load as load_payload;
use super::super::*;
use super::fixtures::*;

#[test]
fn shared_fixture_loads_and_validates() {
    let root = test_root("payload-load");
    let path = write_release(&root);
    let loaded = load_payload(&path).expect("load");
    assert!(loaded.validate().is_ok());
    assert_eq!(loaded.format, 4);
}

#[test]
fn payload_load_enforces_regular_bounded_input() {
    let root = test_root("payload-bounds");
    let path = write_release(&root);
    assert!(load_payload(&path).is_ok());
    assert!(load_payload(&root.join("missing.json")).is_err());
    fs::write(&path, format!("{FIXTURE} {{\"unexpected\":true}}")).expect("trailing");
    assert!(load_payload(&path).is_err());
    // Symlinked input refused like the Go regular-file check.
    let outside = root.join("outside.json");
    fs::write(&outside, FIXTURE).expect("outside");
    let link = root.join("linked.json");
    std::os::unix::fs::symlink(&outside, &link).expect("symlink");
    assert_eq!(
        load_payload(&link).unwrap_err(),
        "bounded regular JSON input required"
    );
    // Oversized input refused.
    let big = root.join("big.json");
    fs::write(&big, vec![b' '; (4 << 20) + 1]).expect("big");
    assert_eq!(
        load_payload(&big).unwrap_err(),
        "bounded regular JSON input required"
    );
}

#[test]
fn old_payload_format_is_rejected_without_migration() {
    let root = test_root("payload-v3");
    let path = root.join("release.json");
    let v3 = FIXTURE.replacen("\"Format\":4", "\"Format\":3", 1);
    fs::write(&path, v3).expect("write v3");
    assert!(load_payload(&path).is_err());
}

//! Inputs oracle vectors: live inputs, CoreOS records, frozen input
//! marshalling, JSON strictness, and misc vectors. Fountain toolchain
//! provenance lives with its canonical owner (release-inputs reader);
//! the Soda producer records the selected release/source proof instead.

use super::{oracle, oracle_live_inputs, scratch};
use soda_build_tools::reader::settings::soda_commands;
use soda_build_tools::reader::stream::valid_live_inputs;
use soda_release_build::confined_files::Root;
use soda_release_build::coreos::https_url;
use soda_release_build::files::{is_digest, File};
use soda_release_build::json_input::{read_json, read_json_at};
use soda_release_build::live_inputs::{read_live_inputs, write_live_inputs};
use std::collections::BTreeMap;

#[test]
fn oracle_live_inputs_write_and_read() {
    let inputs = oracle_live_inputs();
    let dir = scratch("live");
    let path = dir.join("live-inputs.json");
    write_live_inputs(&path, &inputs).unwrap();
    assert!(std::fs::read(&path).unwrap().ends_with(b"\n"));
    assert_eq!(read_live_inputs(&path).unwrap(), inputs);
    let mut bad = inputs.clone();
    bad.fountain.extension_protocol = 2;
    assert!(valid_live_inputs(&bad).is_err(), "live-fountain");
    let mut bad = inputs.clone();
    bad.coreos.release = "tomorrow".to_string();
    assert!(valid_live_inputs(&bad).is_err(), "live-release");
    let mut bad = inputs;
    bad.coreos.container = BTreeMap::new();
    assert!(valid_live_inputs(&bad).is_err(), "live-container");
}

#[test]
fn oracle_misc_vectors() {
    assert!(
        soda_release_build::files::oci_architecture("aarch64").is_err(),
        "misc-oci-arch"
    );
    assert_eq!(
        https_url("http://example.test/base"),
        oracle::MISC_HTTPS_HTTP
    );
    assert!(soda_commands(&scratch("missing-cmds").join("cmd")).is_err());
}

#[test]
fn oracle_read_json_strictness() {
    let dir = scratch("readjson");
    let path = dir.join("strict.json");
    std::fs::write(
        &path,
        format!(
            "{{\"sha256\":\"{}\",\"mode\":420,\"directory\":true,\"bogus\":1}}",
            "a".repeat(64)
        ),
    )
    .unwrap();
    assert!(read_json::<File>(&path).is_err());
    std::fs::write(&path, "{\"mode\":420} {}").unwrap();
    assert!(read_json::<File>(&path).is_err(), "read-trailing");
    std::fs::write(&path, "{\"mode\":420}").unwrap();
    let file: File = read_json(&path).unwrap();
    assert_eq!(file.mode, oracle::READ_OK_MODE);
    assert!(is_digest(&"a".repeat(64)));
    let root = Root::open(&dir).unwrap();
    let (back, _digest): (File, String) = read_json_at::<File>(&root, "strict.json").unwrap();
    assert_eq!(back.mode, oracle::READ_OK_MODE);
}

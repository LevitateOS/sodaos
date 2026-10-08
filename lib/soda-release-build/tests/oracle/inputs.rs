//! Inputs oracle vectors: live inputs, CoreOS records, frozen input
//! marshalling, Forgejo toolchain semantics, JSON strictness, and misc vectors.

use super::{oracle, oracle_live_inputs, scratch, FIXTURE_REVISION};
use soda_build_tools::reader::settings::soda_commands;
use soda_build_tools::reader::stream::{valid_live_inputs, valid_tailnet_inputs, TailnetInputs};
use soda_release_build::confined_files::Root;
use soda_release_build::coreos::https_url;
use soda_release_build::files::{is_digest, File};
use soda_release_build::forgejo::{forgejo_build_args, ForgejoToolchain, FORGEJO_COMPILER_IMAGE};
use soda_release_build::json_input::{read_json, read_json_at};
use soda_release_build::live_inputs::{read_live_inputs, write_live_inputs};
use soda_release_build::production::Production;
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
    bad.tailnet.version = "yesterday".to_string();
    assert_eq!(
        valid_live_inputs(&bad).unwrap_err().0,
        oracle::LIVE_ERR_TAILNET
    );
    let mut bad = inputs.clone();
    bad.coreos.release = "tomorrow".to_string();
    assert_eq!(
        valid_live_inputs(&bad).unwrap_err().0,
        oracle::LIVE_ERR_RELEASE
    );
    let mut bad = inputs;
    bad.coreos.container = BTreeMap::new();
    assert_eq!(
        valid_live_inputs(&bad).unwrap_err().0,
        oracle::LIVE_ERR_CONTAINER
    );
}

#[test]
fn oracle_forgejo_argv_and_script() {
    let mut prod = Production::default();
    prod.native = "$ROOT/native".to_string();
    prod.forgejo_source = "$ROOT/frozen-fork".to_string();
    prod.forgejo_revision = FIXTURE_REVISION.to_string();
    let args = forgejo_build_args(&prod, "amd64");
    let (argv, script) = args.split_at(args.len() - 1);
    assert_eq!(argv.join("\n") + "\n", oracle::FORGEJO_ARGV);
    assert_eq!(script[0].clone() + "\n", oracle::FORGEJO_SCRIPT);
}

#[test]
fn oracle_forgejo_toolchain_semantics_and_determinism() {
    let toolchain = ForgejoToolchain {
        compiler_image: FORGEJO_COMPILER_IMAGE.to_string(),
        apk_packages: vec![
            "build-base-0.5-r4".to_string(),
            "gcc-14.2.0-r6".to_string(),
            "musl-dev-1.2.5-r10".to_string(),
        ],
    };
    toolchain.validate().unwrap();
    let rendered = toolchain.marshal();
    assert_eq!(rendered, toolchain.marshal());
    let readback: ForgejoToolchain = serde_json::from_str(&rendered).unwrap();
    assert_eq!(readback.compiler_image, FORGEJO_COMPILER_IMAGE);
    assert_eq!(readback.apk_packages, toolchain.apk_packages);
}

#[test]
fn oracle_misc_vectors() {
    assert_eq!(
        soda_release_build::files::oci_architecture("aarch64")
            .unwrap_err()
            .message(),
        oracle::MISC_OCI_ARCH
    );
    assert_eq!(
        https_url("http://example.test/base"),
        oracle::MISC_HTTPS_HTTP
    );
    assert_eq!(
        valid_tailnet_inputs(&TailnetInputs {
            version: "x".to_string(),
            sha256: "y".to_string(),
            base: "z".to_string(),
        })
        .unwrap_err()
        .0,
        oracle::MISC_TAILNET
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
    let err = read_json::<File>(&path).unwrap_err();
    assert_eq!(err.message(), "trailing JSON data");
    std::fs::write(&path, "{\"mode\":420}").unwrap();
    let file: File = read_json(&path).unwrap();
    assert_eq!(file.mode, oracle::READ_OK_MODE);
    assert!(is_digest(&"a".repeat(64)));
    let root = Root::open(&dir).unwrap();
    let (back, _digest): (File, String) = read_json_at::<File>(&root, "strict.json").unwrap();
    assert_eq!(back.mode, oracle::READ_OK_MODE);
}

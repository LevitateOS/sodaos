use soda_json::JsonValue;

use super::config::{preflight, rel_path, valid_signer, valid_vm_name};
use super::*;
use crate::evidence::create_evidence;
use crate::files::TempDir;
use crate::process::Phase;
use crate::qmp::QmpClient;

fn fixture_config() -> VmConfig {
    VmConfig {
        name: "soda-native-fixture".to_string(),
        architecture: "x86_64".to_string(),
        base_receipt: "/private/receipt.json".to_string(),
        ignition: "/private/input.ign".to_string(),
        qemu: "/private/qemu".to_string(),
        firmware: "/firmware/code".to_string(),
        variables: "/firmware/vars".to_string(),
        work: "/private/owned".to_string(),
        disk_gib: 64,
        ssh: RemoteConfig {
            user: "root".to_string(),
            host: "127.0.0.1".to_string(),
            key: "/private/key".to_string(),
            known_hosts: "/private/hosts".to_string(),
            port: 22222,
        },
    }
}

/// Port of `TestVMArgumentsRetainDiskAndNativeIsolation` from `remote_test.go`.
#[test]
fn vm_arguments_retain_disk_and_native_isolation() {
    let config = VmConfig {
        name: "soda-native-fixture".to_string(),
        architecture: "x86_64".to_string(),
        work: "/private/owned".to_string(),
        ignition: "/private/input.ign".to_string(),
        firmware: "/firmware/code".to_string(),
        ssh: RemoteConfig {
            port: 22222,
            ..RemoteConfig::default()
        },
        ..fixture_config()
    };
    let args = config.args().join(" ");
    for part in [
        "-machine q35,accel=kvm",
        "accel=kvm",
        "-cpu host",
        "hostfwd=tcp:127.0.0.1:22222-:22",
        "/private/owned/disk.qcow2",
        "/private/owned/vars.fd",
        "opt/com.coreos/config",
    ] {
        assert!(args.contains(part), "{args}");
    }
    assert!(
        !args.contains("-daemonize") && !args.contains("tap,"),
        "{args}"
    );
}

#[test]
fn name_and_signer_shapes() {
    assert!(valid_vm_name("soda-native-fixture"));
    assert!(valid_vm_name("soda-native-a-0-z-9"));
    assert!(!valid_vm_name("soda-native-"));
    assert!(!valid_vm_name("soda-native-FIXTURE"));
    assert!(!valid_vm_name("other-fixture"));
    assert!(valid_signer(&"A".repeat(40)));
    assert!(valid_signer(&"0123456789ABCDEF".repeat(4)));
    assert!(!valid_signer(&"a".repeat(40)));
    assert!(!valid_signer(&"A".repeat(41)));
}

#[test]
fn relative_paths_match_go_rel_cases() {
    assert_eq!(rel_path("/a/b", "/a/b"), ".");
    assert_eq!(rel_path("/a/b", "/a/b/c"), "c");
    assert_eq!(rel_path("/a/b", "/a"), "..");
    assert_eq!(rel_path("/a/b", "/c"), "../../c");
    assert_eq!(rel_path("/", "/a"), "a");
}

#[test]
fn vm_config_decode_rejects_unknown_and_mistyped() {
    let mut entries = vec![
        (
            "Name".to_string(),
            JsonValue::Str("soda-native-fixture".to_string()),
        ),
        ("DiskGiB".to_string(), JsonValue::Number("64".to_string())),
        (
            "SSH".to_string(),
            JsonValue::Object(vec![(
                "Port".to_string(),
                JsonValue::Number("22222".to_string()),
            )]),
        ),
    ];
    let config = decode_vm_config(&JsonValue::Object(entries.clone())).unwrap();
    assert_eq!(config.disk_gib, 64);
    assert_eq!(config.ssh.port, 22222);
    entries.push(("Extra".to_string(), JsonValue::Bool(true)));
    assert!(decode_vm_config(&JsonValue::Object(entries)).is_err());
    let bad = JsonValue::Object(vec![(
        "DiskGiB".to_string(),
        JsonValue::Str("64".to_string()),
    )]);
    assert!(decode_vm_config(&bad).is_err());
}

#[test]
fn preflight_rejects_bad_identity_and_paths() {
    let scratch = TempDir::new("vm").unwrap();
    let evidence_path = scratch.join("evidence").to_string_lossy().into_owned();
    let evidence = create_evidence(&evidence_path, &[]).unwrap();
    let mut config = fixture_config();
    config.name = "wrong".to_string();
    assert_eq!(
        preflight(&config, &evidence).err().unwrap().to_string(),
        "fresh soda-native-* fixture name required"
    );
    config = fixture_config();
    config.disk_gib = 0;
    assert_eq!(
        preflight(&config, &evidence).err().unwrap().to_string(),
        "fresh disk size must be 1..1024 GiB"
    );
    config = fixture_config();
    config.work = "relative".to_string();
    assert_eq!(
        preflight(&config, &evidence).err().unwrap().to_string(),
        "absolute paths without QEMU separators required"
    );
    config = fixture_config();
    config.work = format!("/{}/w", "d".repeat(90));
    assert_eq!(
        preflight(&config, &evidence).err().unwrap().to_string(),
        "select a shorter private work directory for QMP"
    );
    config = fixture_config();
    config.work = format!("{}/work", scratch.path().to_string_lossy());
    std::fs::create_dir_all(&config.work).unwrap();
    assert_eq!(
        preflight(&config, &evidence).err().unwrap().to_string(),
        "fresh unoccupied VM work path required"
    );
    config = fixture_config();
    config.work = evidence_path.clone();
    assert_eq!(
        preflight(&config, &evidence).err().unwrap().to_string(),
        "VM work and evidence must be disjoint"
    );
    config = fixture_config();
    config.work = format!("{}/fresh-work", scratch.path().to_string_lossy());
    config.ssh.host = "10.0.0.1".to_string();
    assert_eq!(
        preflight(&config, &evidence).err().unwrap().to_string(),
        "CoreOS fixture SSH must be root on loopback"
    );
}

#[test]
fn close_replays_first_outcome() {
    let scratch = TempDir::new("vm").unwrap();
    let evidence_path = scratch.join("evidence").to_string_lossy().into_owned();
    let evidence = create_evidence(&evidence_path, &[]).unwrap();
    let mut vm = Vm {
        config: fixture_config(),
        process: None,
        boot_args: None,
        wait_ssh: true,
        qmp: QmpClient {
            socket: scratch.join("missing.sock").to_string_lossy().into_owned(),
            dial: None,
        },
        outputs: Vec::new(),
        evidence: &evidence,
        attempt: 0,
        closed: None,
    };
    assert!(vm.close().is_ok());
    assert!(vm.close().is_ok());
    assert!(vm.wait(&Phase::background()).is_err());
}

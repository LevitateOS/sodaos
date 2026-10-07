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
    let value = serde_json::value::RawValue::from_string(
        r#"{"Name":false,"Name":"soda-native-fixture","DiskGiB":1e400,"DiskGiB":64,"SSH":{"Port":2.2,"Port":22222}}"#.to_string(),
    ).unwrap();
    let config = decode_vm_config(&value).unwrap();
    assert_eq!(config.disk_gib, 64);
    assert_eq!(config.ssh.port, 22222);
    let unknown =
        serde_json::value::RawValue::from_string(r#"{"Extra":true}"#.to_string()).unwrap();
    assert!(decode_vm_config(&unknown).is_err());
    for bad in [
        r#"{"DiskGiB":"64"}"#,
        r#"{"DiskGiB":6.4}"#,
        r#"{"DiskGiB":1e2}"#,
    ] {
        let raw = serde_json::value::RawValue::from_string(bad.to_string()).unwrap();
        assert!(decode_vm_config(&raw).is_err(), "{bad}");
    }
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

#[test]
#[cfg(target_os = "linux")]
fn close_joins_expired_capture_before_closing_and_replays_error() {
    use std::sync::{Arc, Mutex};
    use std::time::Duration;

    use crate::command::{CommandSpec, StdinSpec};
    use crate::evidence::RedactingWriter;
    use crate::process::{self, Phase, SharedWriter};

    let scratch = TempDir::new("vm-pump-close").unwrap();
    let evidence_path = scratch.join("evidence").to_string_lossy().into_owned();
    let evidence = create_evidence(&evidence_path, &[]).unwrap();
    let ready = scratch.path().join("escaped");
    let script = format!(
        "setsid sh -c 'echo ready > {}; exec sleep 5' & while [ ! -f {} ]; do sleep 0.01; done; echo out",
        ready.display(),
        ready.display()
    );
    let capture_phase = Phase::timeout(Duration::from_millis(300));
    let spec = CommandSpec {
        name: "/bin/sh".to_string(),
        args: vec!["-c".to_string(), script],
        dir: None,
        stdin: StdinSpec::Null,
        env: Vec::new(),
    };
    let out_path = scratch.join("stdout");
    let out_file = std::fs::File::create(out_path).unwrap();
    let out: SharedWriter = Arc::new(Mutex::new(RedactingWriter::tee(out_file, Vec::new())));
    let err: SharedWriter = Arc::new(Mutex::new(RedactingWriter::discard()));
    let process = process::start_process(&capture_phase, &spec, out.clone(), err.clone()).unwrap();
    process
        .wait(&Phase::timeout(Duration::from_secs(5)))
        .unwrap();
    std::thread::sleep(Duration::from_millis(350));

    let mut vm = Vm {
        config: fixture_config(),
        process: Some(process),
        boot_args: None,
        wait_ssh: false,
        qmp: QmpClient {
            socket: scratch.join("missing.sock").to_string_lossy().into_owned(),
            dial: None,
        },
        outputs: vec![out, err],
        evidence: &evidence,
        attempt: 1,
        closed: None,
    };
    let first = vm.close().unwrap_err().to_string();
    assert!(first.contains("capture incomplete"), "{first}");
    assert!(vm.process.is_none(), "completed process ownership retained");
    let second = vm.close().unwrap_err().to_string();
    assert_eq!(second, first, "close did not replay its cached error");
}

//! Owned QEMU fixture VMs, mirroring `internal/acceptance/vm.go`.
//!
//! CoreOS fixtures only: a fresh copy-on-write disk over a verified
//! read-only base, loopback management SSH, and QMP-driven power
//! control. Disks and provisioning stay retained for inspection,
//! even on failure; no enrollment is inferred or revoked.
//!
//! Cancellation arrives through [`Phase`] instead of contexts, and
//! process readiness is polled instead of channel-signalled. `Close`
//! replays its first outcome like the Go owner's `sync.Once`.

use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

use soda_json::JsonValue;

use crate::command::{CommandSpec, StdinSpec};
use crate::error::Error;
use crate::evidence::{Evidence, RedactingWriter};
use crate::process::{self, Phase, Process, SharedWriter};
use crate::qmp::QmpClient;

#[path = "vm/base.rs"]
mod base;
#[path = "vm/config.rs"]
mod config;
#[path = "vm/launch.rs"]
mod launch;

pub use self::base::VerifiedBase;
use self::base::{publish_launch_fixture, verify_launch_base_image};
pub use self::config::{decode_vm_config, RemoteConfig, VmConfig};
#[cfg(test)]
use self::config::{preflight, rel_path, valid_signer, valid_vm_name};
pub use self::launch::LaunchFailure;
use self::launch::{prepare_vm_work_directory, verify_base_image_format, verify_qemu_commands};

/// One owned fixture VM.
pub struct Vm<'a> {
    config: VmConfig,
    process: Option<Arc<Process>>,
    boot_args: Option<Vec<String>>,
    wait_ssh: bool,
    qmp: QmpClient,
    outputs: Vec<SharedWriter>,
    evidence: &'a Evidence,
    attempt: u32,
    closed: Option<(String, bool)>,
}

/// Launch a fresh fixture VM: preflight, verified base, fixture
/// record, QEMU checks, work directory, then boot to SSH readiness.
pub fn launch_vm<'a>(
    phase: &Phase,
    mut config: VmConfig,
    evidence: &'a Evidence,
) -> Result<Vm<'a>, Box<LaunchFailure<'a>>> {
    if config.disk_gib == 0 {
        config.disk_gib = 64;
    }
    if let Err(err) = preflight(&config, evidence) {
        return Err(Box::new(LaunchFailure { vm: None, err }));
    }
    let base = match verify_launch_base_image(phase, &config) {
        Ok(base) => base,
        Err(err) => return Err(Box::new(LaunchFailure { vm: None, err })),
    };
    if let Err(err) = publish_launch_fixture(&config, evidence, &base) {
        return Err(Box::new(LaunchFailure { vm: None, err }));
    }
    if let Err(err) = verify_qemu_commands(phase, evidence, &config.qemu) {
        return Err(Box::new(LaunchFailure { vm: None, err }));
    }
    if let Err(err) = verify_base_image_format(phase, evidence, &base.path, config.disk_gib) {
        return Err(Box::new(LaunchFailure { vm: None, err }));
    }
    if let Err(err) = prepare_vm_work_directory(phase, evidence, &config, &base.path) {
        return Err(Box::new(LaunchFailure { vm: None, err }));
    }
    let mut vm = Vm {
        config,
        process: None,
        boot_args: None,
        wait_ssh: true,
        qmp: QmpClient {
            socket: String::new(),
            dial: None,
        },
        outputs: Vec::new(),
        evidence,
        attempt: 0,
        closed: None,
    };
    if let Err(err) = vm.start(phase) {
        let joined = Error::join(vec![Some(err), vm.close().err()]).unwrap();
        return Err(Box::new(LaunchFailure {
            vm: Some(vm),
            err: joined,
        }));
    }
    Ok(vm)
}

impl<'a> Vm<'a> {
    fn open_boot_logs(&mut self) -> Result<(SharedWriter, SharedWriter), Error> {
        let label = format!("boot-{}", self.attempt);
        let out_file = self.evidence.open_file(&format!("{label}.serial"))?;
        let out: SharedWriter = Arc::new(Mutex::new(RedactingWriter::tee(
            out_file,
            self.evidence.secrets().to_vec(),
        )));
        self.outputs.push(out.clone());
        let err_file = self.evidence.open_file(&format!("{label}.stderr"))?;
        let err_writer: SharedWriter = Arc::new(Mutex::new(RedactingWriter::tee(
            err_file,
            self.evidence.secrets().to_vec(),
        )));
        self.outputs.push(err_writer.clone());
        Ok((out, err_writer))
    }

    fn wait_qemu_ready(&self, phase: &Phase) -> Result<(), Error> {
        let ready = phase.child(Duration::from_secs(30));
        let deadline = ready
            .deadline()
            .unwrap_or_else(|| Instant::now() + Duration::from_secs(30));
        let process = self.process.as_ref().unwrap();
        loop {
            let mut status = JsonValue::Null;
            // Any successful response is readiness, like the Go owner;
            // the payload content is not consulted.
            if self
                .qmp
                .execute("query-status", "status", None, Some(&mut status), deadline)
                .is_ok()
            {
                return Ok(());
            }
            if process.is_done() {
                return Err(Error::join(vec![
                    Some(Error::msg("QEMU exited before readiness")),
                    process.wait(phase).err(),
                ])
                .unwrap());
            }
            ready.check()?;
            std::thread::sleep(Duration::from_millis(200));
        }
    }

    fn wait_guest_ssh(&self, phase: &Phase) -> Result<(), Error> {
        if !self.wait_ssh {
            return Ok(());
        }
        let ssh_phase = phase.child(Duration::from_secs(10 * 60));
        let watcher_phase = ssh_phase.clone();
        let watcher_process = self.process.as_ref().unwrap().clone();
        std::thread::spawn(move || loop {
            if watcher_process.is_done() {
                watcher_phase.cancel();
                break;
            }
            if watcher_phase.is_cancelled() || watcher_phase.expired() {
                break;
            }
            std::thread::sleep(Duration::from_millis(50));
        });
        let process = self.process.as_ref().unwrap();
        match self.config.ssh.remote().wait_ready(&ssh_phase) {
            Ok(()) => Ok(()),
            Err(err) => {
                if process.is_done() {
                    return Err(Error::join(vec![
                        Some(Error::msg("QEMU exited during SSH readiness")),
                        process.wait(phase).err(),
                        Some(err),
                    ])
                    .unwrap());
                }
                Err(err)
            }
        }
    }

    fn start(&mut self, phase: &Phase) -> Result<(), Error> {
        self.attempt += 1;
        let (out, err_writer) = self.open_boot_logs()?;
        let args = self.boot_args.clone().unwrap_or_else(|| self.config.args());
        let spec = CommandSpec {
            name: self.config.qemu.clone(),
            args,
            dir: None,
            stdin: StdinSpec::Null,
            env: Vec::new(),
        };
        self.process = Some(process::start_process(phase, &spec, out, err_writer)?);
        self.qmp = QmpClient {
            socket: format!("{}/qmp.sock", self.config.work),
            dial: None,
        };
        self.wait_qemu_ready(phase)?;
        self.wait_guest_ssh(phase)
    }

    /// Restart reusing exactly this instance's disk and NVRAM. No data
    /// is removed.
    pub fn restart(&mut self, phase: &Phase) -> Result<(), Error> {
        self.power_down(phase)?;
        self.start(phase)
    }

    fn power_down(&mut self, phase: &Phase) -> Result<(), Error> {
        let Some(process) = self.process.clone() else {
            return Ok(());
        };
        let shutdown = phase.child(Duration::from_secs(2 * 60));
        let deadline = shutdown
            .deadline()
            .unwrap_or_else(|| Instant::now() + Duration::from_secs(2 * 60));
        self.qmp
            .execute("system_powerdown", "powerdown", None, None, deadline)?;
        process.wait(&shutdown)?;
        self.process = None;
        match std::fs::remove_file(&self.qmp.socket) {
            Ok(()) => {}
            Err(err) if err.kind() == std::io::ErrorKind::NotFound => {}
            Err(err) => return Err(Error::from(err)),
        }
        match self.close_outputs() {
            Some(err) => Err(err),
            None => Ok(()),
        }
    }

    fn close_outputs(&mut self) -> Option<Error> {
        let mut err = None;
        for writer in self.outputs.drain(..) {
            let mut slot = writer.lock().unwrap_or_else(|e| e.into_inner());
            err = Error::join(vec![err, slot.close().err()]);
        }
        err
    }

    /// Wait for guest exit or phase end. The Go owner panics on a
    /// closed VM; this port reports the guest as unexpectedly gone, a
    /// path the driver never takes.
    pub fn wait(&self, phase: &Phase) -> Result<(), Error> {
        let Some(process) = self.process.as_ref() else {
            return Err(Error::msg("guest exited unexpectedly"));
        };
        loop {
            phase.check()?;
            if process.is_done() {
                return Err(Error::join(vec![
                    Some(Error::msg("guest exited unexpectedly")),
                    process.wait(phase).err(),
                ])
                .unwrap());
            }
            std::thread::sleep(Duration::from_millis(50));
        }
    }

    fn close_once(&mut self) -> Result<(), Error> {
        let shutdown = Phase::background().child(Duration::from_secs(2 * 60));
        let mut err = self.power_down(&shutdown).err();
        if let Some(process) = self.process.clone() {
            err = Error::join(vec![err, process.stop().err()]);
            if !process.is_done() {
                // A kernel-stuck child still owns the capture writers. Do not
                // close them concurrently or describe their retention as complete.
                let outputs = std::mem::take(&mut self.outputs);
                std::thread::spawn(move || {
                    let _ = process.wait(&Phase::background());
                    for writer in outputs {
                        let mut slot = writer.lock().unwrap_or_else(|e| e.into_inner());
                        let _ = slot.close();
                    }
                });
                err = Error::join(vec![
                    err,
                    Some(Error::msg("VM capture still owned by incomplete cleanup")),
                ]);
                return err.map(Err).unwrap_or(Ok(()));
            }
        }
        err = Error::join(vec![err, self.close_outputs()]);
        err.map(Err).unwrap_or(Ok(()))
    }

    /// Keep disks/provisioning for inspection, even on failure. No
    /// enrollment is inferred or revoked. The first outcome replays on
    /// later calls, like the Go owner's `sync.Once`.
    pub fn close(&mut self) -> Result<(), Error> {
        if let Some((message, cancelled)) = &self.closed {
            if message.is_empty() && !cancelled {
                return Ok(());
            }
            let cause = if *cancelled {
                Error::Cancelled
            } else {
                Error::msg(String::new())
            };
            return Err(Error::wrap(message.clone(), cause));
        }
        let result = self.close_once();
        let (message, cancelled) = match &result {
            Ok(()) => (String::new(), false),
            Err(err) => (err.to_string(), err.is_cancelled()),
        };
        self.closed = Some((message, cancelled));
        result
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::evidence::create_evidence;
    use crate::files::TempDir;

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
}

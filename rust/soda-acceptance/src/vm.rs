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

use crate::command::{self, CommandSpec, Remote, StdinSpec};
use crate::coreos;
use crate::error::Error;
use crate::evidence::{Evidence, RedactingWriter};
use crate::files;
use crate::jsonio;
use crate::process::{self, Phase, Process, SharedWriter};
use crate::qmp::QmpClient;
use crate::report::require_native;
use crate::trust;

/// Fresh fixture VM configuration, mirroring Go's `VMConfig`.
#[derive(Debug, Clone, Default)]
pub struct VmConfig {
    /// Fixture name.
    pub name: String,
    /// Appliance architecture.
    pub architecture: String,
    /// Verified base receipt path.
    pub base_receipt: String,
    /// Ignition file path.
    pub ignition: String,
    /// QEMU binary path.
    pub qemu: String,
    /// Firmware code path.
    pub firmware: String,
    /// Firmware variables template path.
    pub variables: String,
    /// Fresh work directory path.
    pub work: String,
    /// Disk size in GiB (0 selects 64).
    pub disk_gib: i64,
    /// Loopback management SSH endpoint.
    pub ssh: RemoteConfig,
}

/// SSH endpoint inside a VM configuration.
#[derive(Debug, Clone, Default)]
pub struct RemoteConfig {
    /// Login user.
    pub user: String,
    /// Host address.
    pub host: String,
    /// Private identity file.
    pub key: String,
    /// Pinned known_hosts file.
    pub known_hosts: String,
    /// SSH port.
    pub port: i64,
}

impl RemoteConfig {
    fn remote(&self) -> Remote {
        Remote {
            user: self.user.clone(),
            host: self.host.clone(),
            key: self.key.clone(),
            known_hosts: self.known_hosts.clone(),
            port: self.port,
            timeout: Duration::ZERO,
        }
    }
}

/// Decode a VM configuration with Go field names and no unknown fields.
pub fn decode_vm_config(value: &JsonValue) -> Result<VmConfig, Error> {
    jsonio::check_no_unknown(
        value,
        &[
            "Name",
            "Architecture",
            "BaseReceipt",
            "Ignition",
            "QEMU",
            "Firmware",
            "Variables",
            "Work",
            "DiskGiB",
            "SSH",
        ],
    )?;
    let ssh_value = value.get("SSH").cloned().unwrap_or(JsonValue::Null);
    let ssh = command::decode_remote(&ssh_value)?;
    let disk_gib = jsonio::opt_integer(value, "DiskGiB")?;
    let disk_gib =
        i64::try_from(disk_gib).map_err(|_| Error::msg("invalid DiskGiB: integer required"))?;
    Ok(VmConfig {
        name: jsonio::opt_string(value, "Name")?,
        architecture: jsonio::opt_string(value, "Architecture")?,
        base_receipt: jsonio::opt_string(value, "BaseReceipt")?,
        ignition: jsonio::opt_string(value, "Ignition")?,
        qemu: jsonio::opt_string(value, "QEMU")?,
        firmware: jsonio::opt_string(value, "Firmware")?,
        variables: jsonio::opt_string(value, "Variables")?,
        work: jsonio::opt_string(value, "Work")?,
        disk_gib,
        ssh: RemoteConfig {
            user: ssh.user,
            host: ssh.host,
            key: ssh.key,
            known_hosts: ssh.known_hosts,
            port: ssh.port,
        },
    })
}

/// `^soda-native-[a-z0-9-]+$` fixture names.
fn valid_vm_name(name: &str) -> bool {
    match name.strip_prefix("soda-native-") {
        Some(rest) => {
            !rest.is_empty()
                && rest
                    .bytes()
                    .all(|b| matches!(b, b'a'..=b'z' | b'0'..=b'9' | b'-'))
        }
        None => false,
    }
}

/// `^(?:[A-F0-9]{40}|[A-F0-9]{64})$` signer fingerprints.
fn valid_signer(signer: &str) -> bool {
    (signer.len() == 40 || signer.len() == 64)
        && signer
            .bytes()
            .all(|b| matches!(b, b'0'..=b'9' | b'A'..=b'F'))
}

fn validate_vm_config_identity(config: &VmConfig) -> Result<(), Error> {
    require_native(&config.architecture)?;
    if config.disk_gib < 1 || config.disk_gib > 1024 {
        return Err(Error::msg("fresh disk size must be 1..1024 GiB"));
    }
    if !valid_vm_name(&config.name) {
        return Err(Error::msg("fresh soda-native-* fixture name required"));
    }
    Ok(())
}

/// Lexical relative path between two absolute paths, like Go's
/// `filepath.Rel` on POSIX.
fn rel_path(base: &str, target: &str) -> String {
    let base = files::lexical_clean(base);
    let target = files::lexical_clean(target);
    let b: Vec<&str> = base.split('/').filter(|s| !s.is_empty()).collect();
    let t: Vec<&str> = target.split('/').filter(|s| !s.is_empty()).collect();
    let mut shared = 0;
    while shared < b.len() && shared < t.len() && b[shared] == t[shared] {
        shared += 1;
    }
    let mut parts: Vec<&str> = vec![".."; b.len() - shared];
    parts.extend(t[shared..].iter().copied());
    if parts.is_empty() {
        return ".".to_string();
    }
    parts.join("/")
}

fn disjoint_vm_work(work: &str, evidence: &Evidence) -> Result<(), Error> {
    for (base, target) in [(work, evidence.path()), (evidence.path(), work)] {
        let rel = rel_path(base, target);
        if rel == "." || (!rel.starts_with("../") && rel != "..") {
            return Err(Error::msg("VM work and evidence must be disjoint"));
        }
    }
    Ok(())
}

fn validate_vm_paths(config: &VmConfig, evidence: &Evidence) -> Result<(), Error> {
    for path in [
        &config.base_receipt,
        &config.ignition,
        &config.qemu,
        &config.firmware,
        &config.variables,
        &config.work,
        &config.ssh.key,
        &config.ssh.known_hosts,
    ] {
        if !path.starts_with('/') || path.contains([',', '\n', '\r']) {
            return Err(Error::msg(
                "absolute paths without QEMU separators required",
            ));
        }
    }
    // `filepath.Join` cleans, so `/w/` and `/w` measure the same.
    if files::lexical_clean(&format!("{}/qmp.sock", config.work)).len() > 100 {
        return Err(Error::msg(
            "select a shorter private work directory for QMP",
        ));
    }
    disjoint_vm_work(&config.work, evidence)?;
    match std::fs::symlink_metadata(&config.work) {
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(()),
        _ => Err(Error::msg("fresh unoccupied VM work path required")),
    }
}

fn validate_vm_ssh_and_trust(config: &VmConfig) -> Result<(), Error> {
    if config.ssh.host != "127.0.0.1" || config.ssh.user != "root" {
        return Err(Error::msg("CoreOS fixture SSH must be root on loopback"));
    }
    config.ssh.remote().args()?;
    files::private_file(&config.ignition)?;
    trust::verify_fixture_trust(&config.ignition, &config.name, &config.ssh.remote())
}

fn validate_vm_inputs(config: &VmConfig) -> Result<(), Error> {
    for path in [&config.qemu, &config.firmware, &config.variables] {
        let meta = std::fs::symlink_metadata(path).map_err(Error::from)?;
        use std::os::unix::fs::MetadataExt;
        if !meta.is_file() || meta.size() == 0 || meta.size() > 128 << 20 {
            return Err(Error::msg("bounded regular QEMU/firmware input required"));
        }
    }
    Ok(())
}

fn check_vm_host_tools_and_kvm(qemu: &str, ssh_port: i64) -> Result<(), Error> {
    command::look_path(qemu)?;
    command::look_path("qemu-img")?;
    command::look_path("ssh")?;
    let kvm = std::fs::OpenOptions::new()
        .read(true)
        .write(true)
        .open("/dev/kvm")
        .map_err(Error::from)?;
    drop(kvm);
    let port = u16::try_from(ssh_port).map_err(|_| Error::msg("invalid SSH user/port"))?;
    let listener = std::net::TcpListener::bind(("127.0.0.1", port)).map_err(Error::from)?;
    drop(listener);
    Ok(())
}

fn preflight(config: &VmConfig, evidence: &Evidence) -> Result<(), Error> {
    validate_vm_config_identity(config)?;
    validate_vm_paths(config, evidence)?;
    validate_vm_ssh_and_trust(config)?;
    validate_vm_inputs(config)?;
    check_vm_host_tools_and_kvm(&config.qemu, config.ssh.port)
}

/// Verified base receipt, mirroring Go's `VerifiedBase`.
pub struct VerifiedBase {
    /// Base image path.
    pub path: String,
    /// Expected uncompressed SHA-256.
    pub sha256: String,
    /// Base architecture.
    pub architecture: String,
    /// Base release.
    pub release: String,
    /// Signer fingerprint.
    pub signer: String,
}

fn decode_verified_base(value: &JsonValue) -> Result<VerifiedBase, Error> {
    jsonio::check_no_unknown(
        value,
        &["Path", "SHA256", "Architecture", "Release", "Signer"],
    )?;
    Ok(VerifiedBase {
        path: jsonio::opt_string(value, "Path")?,
        sha256: jsonio::opt_string(value, "SHA256")?,
        architecture: jsonio::opt_string(value, "Architecture")?,
        release: jsonio::opt_string(value, "Release")?,
        signer: jsonio::opt_string(value, "Signer")?,
    })
}

fn verify_launch_base_image(phase: &Phase, config: &VmConfig) -> Result<VerifiedBase, Error> {
    let (value, _) = jsonio::read_json_file(&config.base_receipt)?;
    let base = decode_verified_base(&value)?;
    let (release, image) = coreos::resolve_qemu(phase, &config.architecture)?;
    if base.architecture != config.architecture
        || base.release != release
        || base.sha256 != image.uncompressed_sha256
    {
        return Err(Error::msg("base does not match resolved CoreOS input"));
    }
    if !base.path.starts_with('/') || base.path.contains([',', '\n', '\r']) {
        return Err(Error::msg("unsafe base path"));
    }
    let meta = std::fs::symlink_metadata(&base.path).map_err(Error::from)?;
    use std::os::unix::fs::MetadataExt;
    if meta.mode() & 0o222 != 0 {
        return Err(Error::msg(
            "verified base must be read-only; fetch a fresh cache, never chmod a live base",
        ));
    }
    if !valid_signer(&base.signer) {
        return Err(Error::msg("verified base receipt lacks selected signer"));
    }
    match files::hash_file(&base.path) {
        Ok(sum) if sum == base.sha256 => Ok(base),
        Ok(_) => Err(Error::msg("base checksum mismatch")),
        Err(err) => {
            Err(Error::join(vec![Some(err), Some(Error::msg("base checksum mismatch"))]).unwrap())
        }
    }
}

fn publish_launch_fixture(
    config: &VmConfig,
    evidence: &Evidence,
    base: &VerifiedBase,
) -> Result<(), Error> {
    let firmware_hash = files::hash_file(&config.firmware)?;
    let vars_hash = files::hash_file(&config.variables)?;
    let description = JsonValue::Object(vec![
        ("Name".to_string(), JsonValue::Str(config.name.clone())),
        (
            "Architecture".to_string(),
            JsonValue::Str(config.architecture.clone()),
        ),
        (
            "BaseSHA256".to_string(),
            JsonValue::Str(base.sha256.clone()),
        ),
        ("Release".to_string(), JsonValue::Str(base.release.clone())),
        ("FirmwareSHA256".to_string(), JsonValue::Str(firmware_hash)),
        ("VariablesSHA256".to_string(), JsonValue::Str(vars_hash)),
        ("Work".to_string(), JsonValue::Str(config.work.clone())),
    ]);
    evidence.write_json("fixture.json", &description)
}

fn verify_qemu_commands(phase: &Phase, evidence: &Evidence, qemu: &str) -> Result<(), Error> {
    for (label, name) in [("qemu-version", qemu), ("qemu-img-version", "qemu-img")] {
        let spec = CommandSpec {
            name: name.to_string(),
            args: vec!["--version".to_string()],
            dir: None,
            stdin: StdinSpec::Null,
            env: Vec::new(),
        };
        let (result, evidence_err) = command::execute(phase, evidence, label, &spec);
        if let Some(joined) = Error::join(vec![evidence_err, result.err]) {
            return Err(joined);
        }
    }
    Ok(())
}

fn verify_base_image_format(
    phase: &Phase,
    evidence: &Evidence,
    base_path: &str,
    disk_gib: i64,
) -> Result<(), Error> {
    // qemu-img obtains its normal image locks; never request unsafe -U access.
    let spec = CommandSpec {
        name: "qemu-img".to_string(),
        args: vec![
            "info".to_string(),
            "--output=json".to_string(),
            base_path.to_string(),
        ],
        dir: None,
        stdin: StdinSpec::Null,
        env: Vec::new(),
    };
    let (result, evidence_err) = command::execute(phase, evidence, "base-info", &spec);
    if let Some(joined) = Error::join(vec![evidence_err, result.err]) {
        return Err(joined);
    }
    let text =
        std::str::from_utf8(&result.stdout).map_err(|_| Error::msg("invalid base image info"))?;
    let info = JsonValue::parse(text).map_err(|_| Error::msg("invalid base image info"))?;
    let format =
        jsonio::opt_string(&info, "format").map_err(|_| Error::msg("invalid base image info"))?;
    let backing = jsonio::opt_string(&info, "backing-filename")
        .map_err(|_| Error::msg("invalid base image info"))?;
    let size = info
        .get("virtual-size")
        .and_then(|v| v.as_integer())
        .and_then(|v| i64::try_from(v).ok())
        .ok_or_else(|| Error::msg("invalid base image info"))?;
    if format != "qcow2" || !backing.is_empty() || size <= 0 || size > disk_gib << 30 {
        return Err(Error::msg("standalone qcow2 base required"));
    }
    Ok(())
}

fn prepare_vm_work_directory(
    phase: &Phase,
    evidence: &Evidence,
    config: &VmConfig,
    base_path: &str,
) -> Result<(), Error> {
    files::fresh_directory(&config.work)?;
    let disk = format!("{}/disk.qcow2", config.work);
    let spec = CommandSpec {
        name: "qemu-img".to_string(),
        args: vec![
            "create".to_string(),
            "-f".to_string(),
            "qcow2".to_string(),
            "-F".to_string(),
            "qcow2".to_string(),
            "-b".to_string(),
            base_path.to_string(),
            disk,
            format!("{}G", config.disk_gib),
        ],
        dir: None,
        stdin: StdinSpec::Null,
        env: Vec::new(),
    };
    let (result, evidence_err) = command::execute(phase, evidence, "disk-create", &spec);
    if let Some(joined) = Error::join(vec![evidence_err, result.err]) {
        return Err(joined);
    }
    let vars = std::fs::read(&config.variables).map_err(Error::from)?;
    files::write_new(&format!("{}/vars.fd", config.work), &vars, 0o600)
}

impl VmConfig {
    /// Owned QEMU arguments: KVM, host CPU, virtio disk/NIC, fw_cfg
    /// Ignition, loopback host forwarding, and a QMP socket.
    pub fn args(&self) -> Vec<String> {
        vec![
            "-name".to_string(),
            self.name.clone(),
            "-machine".to_string(),
            "q35,accel=kvm".to_string(),
            "-cpu".to_string(),
            "host".to_string(),
            "-smp".to_string(),
            "4".to_string(),
            "-m".to_string(),
            "8192".to_string(),
            "-display".to_string(),
            "none".to_string(),
            "-monitor".to_string(),
            "none".to_string(),
            "-serial".to_string(),
            "stdio".to_string(),
            "-drive".to_string(),
            format!("if=pflash,format=raw,readonly=on,file={}", self.firmware),
            "-drive".to_string(),
            format!("if=pflash,format=raw,file={}/vars.fd", self.work),
            "-drive".to_string(),
            format!("if=virtio,format=qcow2,file={}/disk.qcow2", self.work),
            "-fw_cfg".to_string(),
            format!("name=opt/com.coreos/config,file={}", self.ignition),
            "-nic".to_string(),
            format!(
                "user,model=virtio-net-pci,hostfwd=tcp:127.0.0.1:{}-:22",
                self.ssh.port
            ),
            "-qmp".to_string(),
            format!("unix:{}/qmp.sock,server=on,wait=off", self.work),
        ]
    }
}

/// Launch failure, carrying the VM when the guest started so the driver
/// can retain and close it.
pub struct LaunchFailure<'a> {
    /// Started VM, when the failure came after process start.
    pub vm: Option<Vm<'a>>,
    /// Launch error.
    pub err: Error,
}

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

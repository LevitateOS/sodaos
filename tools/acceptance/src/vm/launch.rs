use serde_json::value::RawValue;
use std::collections::BTreeMap;

use super::config::VmConfig;
use super::lifecycle::Vm;
use crate::command::{self, CommandSpec, StdinSpec};
use crate::error::Error;
use crate::evidence::Evidence;
use crate::files;
use crate::process::Phase;

pub(super) fn verify_qemu_commands(
    phase: &Phase,
    evidence: &Evidence,
    qemu: &str,
) -> Result<(), Error> {
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

pub(super) fn verify_base_image_format(
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
    let invalid = || Error::msg("invalid base image info");
    let info: BTreeMap<String, Box<RawValue>> =
        serde_json::from_str(text).map_err(|_| invalid())?;
    let string_field = |name: &str| -> Result<String, Error> {
        match info.get(name) {
            None => Ok(String::new()),
            Some(raw) => serde_json::from_str::<Option<String>>(raw.get())
                .map(|v| v.unwrap_or_default())
                .map_err(|_| invalid()),
        }
    };
    let format = string_field("format")?;
    let backing = string_field("backing-filename")?;
    let size = info
        .get("virtual-size")
        .ok_or_else(invalid)?
        .get()
        .parse::<i64>()
        .map_err(|_| invalid())?;
    if format != "qcow2" || !backing.is_empty() || size <= 0 || size > disk_gib << 30 {
        return Err(Error::msg("standalone qcow2 base required"));
    }
    Ok(())
}

pub(super) fn prepare_vm_work_directory(
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

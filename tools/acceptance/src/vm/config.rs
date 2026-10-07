use std::time::Duration;

use serde_json::value::RawValue;
use std::collections::BTreeMap;

use crate::command::{self, Remote};
use crate::error::Error;
use crate::evidence::Evidence;
use crate::files;
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
    pub(super) fn remote(&self) -> Remote {
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
pub fn decode_vm_config(value: &RawValue) -> Result<VmConfig, Error> {
    let fields = match serde_json::from_str::<BTreeMap<String, Box<RawValue>>>(value.get()) {
        Ok(fields) => fields,
        Err(_) if !value.get().trim_start().starts_with('{') => BTreeMap::new(),
        Err(_) => return Err(Error::msg("invalid VM configuration")),
    };
    for key in fields.keys() {
        if ![
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
        ]
        .contains(&key.as_str())
        {
            return Err(Error::msg("unknown JSON field"));
        }
    }
    let string_field = |name: &str| -> Result<String, Error> {
        match fields.get(name) {
            None => Ok(String::new()),
            Some(raw) => serde_json::from_str::<Option<String>>(raw.get())
                .map(|v| v.unwrap_or_default())
                .map_err(|_| Error::msg("invalid JSON string field")),
        }
    };
    let disk_gib = match fields.get("DiskGiB") {
        None => 0,
        Some(raw) if raw.get() == "null" => 0,
        Some(raw) => raw
            .get()
            .parse::<i128>()
            .map_err(|_| Error::msg("invalid DiskGiB: integer required"))?,
    };
    let disk_gib =
        i64::try_from(disk_gib).map_err(|_| Error::msg("invalid DiskGiB: integer required"))?;
    let absent_ssh = RawValue::from_string("null".to_string()).expect("valid JSON literal");
    let ssh_value = fields.get("SSH").map(Box::as_ref).unwrap_or(&absent_ssh);
    let ssh = command::decode_remote(ssh_value)?;
    Ok(VmConfig {
        name: string_field("Name")?,
        architecture: string_field("Architecture")?,
        base_receipt: string_field("BaseReceipt")?,
        ignition: string_field("Ignition")?,
        qemu: string_field("QEMU")?,
        firmware: string_field("Firmware")?,
        variables: string_field("Variables")?,
        work: string_field("Work")?,
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
pub(super) fn valid_vm_name(name: &str) -> bool {
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
pub(super) fn valid_signer(signer: &str) -> bool {
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
pub(super) fn rel_path(base: &str, target: &str) -> String {
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

pub(super) fn preflight(config: &VmConfig, evidence: &Evidence) -> Result<(), Error> {
    validate_vm_config_identity(config)?;
    validate_vm_paths(config, evidence)?;
    validate_vm_ssh_and_trust(config)?;
    validate_vm_inputs(config)?;
    check_vm_host_tools_and_kvm(&config.qemu, config.ssh.port)
}

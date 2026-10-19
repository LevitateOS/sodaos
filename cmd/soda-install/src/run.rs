//! Installer entrypoint: action dispatch, single-instance lock, and the
//! media identity every disk attempt authenticates.

use std::os::unix::fs::OpenOptionsExt;

use crate::buildx;
use crate::command::Runner;
use crate::console::Console;
use crate::errors::Error;
use crate::execute;
use crate::hostadmit;
use crate::signal::Ctx;
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use serde_json::value::RawValue;
use std::fmt;

pub const DATA_DIR: &str = "/usr/local/share/soda-installer";
pub const DISK_ATTEMPT_MARKER: &str = "/run/soda-installer-disk-started";

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MediaIdentity {
    pub architecture: String,
    pub release: String,
    pub installer_version: String,
    pub revision: String,
    pub host_manifest: String,
    pub payload_sha256: String,
    pub console_sha256: String,
}

impl MediaIdentity {
    pub fn validate(&self, image_version: &str, arch: &str) -> Result<(), Error> {
        if self.architecture != arch
            || self.release.is_empty()
            || self.release != image_version
            || !Self::valid_installer_version(&self.installer_version)
            || !buildx::revision(&self.revision)
            || !self.valid_content()
        {
            return Err(Error::msg(
                "media release, architecture, or included-payload mismatch",
            ));
        }
        Ok(())
    }

    pub fn valid_installer_version(version: &str) -> bool {
        version
            .strip_prefix("coreos-installer ")
            .is_some_and(|reported| {
                !reported.trim().is_empty() && !version.chars().any(char::is_control)
            })
    }

    pub fn valid_content(&self) -> bool {
        match self.host_manifest.strip_prefix("sha256:") {
            Some(hex) => {
                buildx::digest(hex)
                    && buildx::digest(&self.payload_sha256)
                    && buildx::digest(&self.console_sha256)
            }
            None => false,
        }
    }
}

pub fn decode_media_identity(data: &[u8]) -> Result<MediaIdentity, Error> {
    struct Fields(Vec<(String, Box<RawValue>)>);
    impl<'de> Deserialize<'de> for Fields {
        fn deserialize<D>(d: D) -> Result<Self, D::Error>
        where
            D: Deserializer<'de>,
        {
            struct FieldsVisitor;
            impl<'de> Visitor<'de> for FieldsVisitor {
                type Value = Fields;
                fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                    f.write_str("a media identity object")
                }
                fn visit_map<A>(self, mut map: A) -> Result<Fields, A::Error>
                where
                    A: MapAccess<'de>,
                {
                    let mut fields = Vec::new();
                    while let Some(key) = map.next_key::<String>()? {
                        if ![
                            "Architecture",
                            "Release",
                            "InstallerVersion",
                            "Revision",
                            "HostManifest",
                            "PayloadSHA256",
                            "ConsoleSHA256",
                        ]
                        .iter()
                        .any(|name| {
                            key == *name
                                || (key.len() == name.len() && key.eq_ignore_ascii_case(name))
                        }) {
                            return Err(de::Error::custom("unknown field"));
                        }
                        fields.push((key, map.next_value::<Box<RawValue>>()?));
                    }
                    Ok(Fields(fields))
                }
            }
            d.deserialize_map(FieldsVisitor)
        }
    }
    let invalid = || Error::msg("invalid media identity document");
    let text = String::from_utf8_lossy(data);
    let mut de = serde_json::Deserializer::from_str(&text);
    let Fields(fields) = Fields::deserialize(&mut de).map_err(|_| invalid())?;
    de.end().map_err(|_| invalid())?;
    fn get(fields: &[(String, Box<RawValue>)], name: &str) -> Result<String, ()> {
        let raw = fields
            .iter()
            .rev()
            .find(|(key, _)| {
                key == name || (key.len() == name.len() && key.eq_ignore_ascii_case(name))
            })
            .map(|(_, raw)| raw.get());
        match raw {
            None => Ok(String::new()),
            Some(raw) => serde_json::from_str::<Option<String>>(raw)
                .map(|s| s.unwrap_or_default())
                .map_err(|_| ()),
        }
    }
    let media = MediaIdentity {
        architecture: get(&fields, "Architecture").map_err(|_| invalid())?,
        release: get(&fields, "Release").map_err(|_| invalid())?,
        installer_version: get(&fields, "InstallerVersion").map_err(|_| invalid())?,
        revision: get(&fields, "Revision").map_err(|_| invalid())?,
        host_manifest: get(&fields, "HostManifest").map_err(|_| invalid())?,
        payload_sha256: get(&fields, "PayloadSHA256").map_err(|_| invalid())?,
        console_sha256: get(&fields, "ConsoleSHA256").map_err(|_| invalid())?,
    };
    Ok(media)
}

pub fn read_media_identity(path: &str) -> Result<MediaIdentity, Error> {
    let data = buildx::read_json_bytes(path)?;
    decode_media_identity(&data)
}

pub fn architecture() -> String {
    if cfg!(target_arch = "x86_64") {
        "x86_64".to_string()
    } else {
        "unsupported".to_string()
    }
}

/// Enrollment actions dispatch without requesting a terminal, after root and
/// installed-CoreOS checks.
fn run_enrollment_action(ctx: &Ctx, action: &str, run: &dyn Runner) -> Result<bool, Error> {
    match action {
        "enrollment-serve" => {
            hostadmit::core_os_host(false, run)?;
            return crate::enroll::serve_enrollment(ctx, run).map(|()| true);
        }
        "enrollment-receive" => {
            hostadmit::core_os_host(false, run)?;
            return crate::enroll::receive_enrollment(ctx).map(|()| true);
        }
        _ => {}
    }
    Ok(false)
}

struct InstallLock {
    file: std::fs::File,
}

impl Drop for InstallLock {
    fn drop(&mut self) {
        use std::os::unix::io::AsRawFd;
        unsafe {
            libc::flock(self.file.as_raw_fd(), libc::LOCK_UN);
        }
    }
}

fn lock_installer() -> Result<InstallLock, Error> {
    let file = std::fs::OpenOptions::new()
        .read(true)
        .write(true)
        .create(true)
        .mode(0o600)
        .custom_flags(libc::O_NOFOLLOW)
        .open("/run/soda-installer.lock")
        .map_err(|_| Error::msg("cannot open installer lock"))?;
    use std::os::unix::io::AsRawFd;
    if unsafe { libc::flock(file.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) } != 0 {
        return Err(Error::msg("another installer is active"));
    }
    Ok(InstallLock { file })
}

fn run_locked_install(
    ctx: &Ctx,
    console: &Console,
    run: &dyn Runner,
    action: &str,
) -> Result<(), Error> {
    match action {
        "configure" => crate::setup::configure_install(ctx, console, run),
        "enroll-key" => crate::enroll::arm_enrollment(ctx, console, run),
        _ => {
            let install_err = execute::install_disk(ctx, console, run).err();
            execute::land_diagnostic_console(ctx, console, run, install_err)
        }
    }
}

fn valid_install_action(action: &str) -> bool {
    matches!(action, "disk" | "configure" | "enroll-key")
}

/// `Run` requires a controlling terminal. The live service starts disk
/// review; installed-host actions are explicitly operator-started.
pub fn run(ctx: &Ctx, run: &dyn Runner, action: &str) -> Result<(), Error> {
    if unsafe { libc::geteuid() } != 0 || !cfg!(target_os = "linux") {
        return Err(Error::msg("native CoreOS root required"));
    }
    if run_enrollment_action(ctx, action, run)? {
        return Ok(());
    }
    if !valid_install_action(action) {
        return Err(Error::msg("usage: soda-install disk|configure|enroll-key"));
    }
    let tty = std::fs::OpenOptions::new()
        .read(true)
        .write(true)
        .open("/dev/tty")
        .map_err(|_| Error::msg("interactive operator terminal required"))?;
    let console = Console::from_file("/dev/tty", tty);
    hostadmit::core_os_host(action == "disk", run)?;
    // One local caller, including when different consoles are active. Lock
    // file is not a success marker and is never removed to pretend a
    // partial attempt is new.
    let _lock = lock_installer()?;
    run_locked_install(ctx, &console, run, action)
}

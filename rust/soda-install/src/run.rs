//! Installer entrypoint: action dispatch, single-instance lock, and the
//! media identity every disk attempt authenticates.

use std::os::unix::fs::OpenOptionsExt;

use crate::buildx;
use crate::command::Runner;
use crate::console::Console;
use crate::errors::Error;
use crate::execute;
use crate::hostadmit;
use crate::jsongo::{parse, Binder};
use crate::signal::Ctx;

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
            || !buildx::revision(&self.revision)
            || !self.valid_content()
        {
            return Err(Error::msg("media release, architecture, or included-payload mismatch"));
        }
        Ok(())
    }

    pub fn valid_content(&self) -> bool {
        match self.host_manifest.strip_prefix("sha256:") {
            Some(hex) => {
                buildx::digest(hex) && buildx::digest(&self.payload_sha256) && buildx::digest(&self.console_sha256)
            }
            None => false,
        }
    }
}

pub fn decode_media_identity(data: &[u8]) -> Result<MediaIdentity, Error> {
    let value = parse(data).map_err(|_| Error::msg("invalid media identity document"))?;
    let mut binder = Binder::new(&value).map_err(|_| Error::msg("invalid media identity document"))?;
    let media = MediaIdentity {
        architecture: binder.string("Architecture").map_err(|_| Error::msg("invalid media identity document"))?.unwrap_or_default(),
        release: binder.string("Release").map_err(|_| Error::msg("invalid media identity document"))?.unwrap_or_default(),
        installer_version: binder.string("InstallerVersion").map_err(|_| Error::msg("invalid media identity document"))?.unwrap_or_default(),
        revision: binder.string("Revision").map_err(|_| Error::msg("invalid media identity document"))?.unwrap_or_default(),
        host_manifest: binder.string("HostManifest").map_err(|_| Error::msg("invalid media identity document"))?.unwrap_or_default(),
        payload_sha256: binder.string("PayloadSHA256").map_err(|_| Error::msg("invalid media identity document"))?.unwrap_or_default(),
        console_sha256: binder.string("ConsoleSHA256").map_err(|_| Error::msg("invalid media identity document"))?.unwrap_or_default(),
    };
    binder.finish().map_err(|_| Error::msg("invalid media identity document"))?;
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

fn run_locked_install(ctx: &Ctx, console: &Console, run: &dyn Runner, action: &str) -> Result<(), Error> {
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn media_identity_matrix() {
        let media = MediaIdentity {
            architecture: "x86_64".to_string(),
            release: "44.20260817.3.2".to_string(),
            installer_version: "coreos-installer 0.26.0".to_string(),
            revision: "a".repeat(40),
            host_manifest: format!("sha256:{}", "b".repeat(64)),
            payload_sha256: "c".repeat(64),
            console_sha256: "d".repeat(64),
        };
        media.validate("44.20260817.3.2", "x86_64").unwrap();
        for version in ["", "44", "44.20260817.3.3"] {
            assert!(media.validate(version, "x86_64").is_err(), "{version:?}");
        }
        assert!(media.validate("44.20260817.3.2", "aarch64").is_err());
        let mut bad = media.clone();
        bad.revision = "unreviewed".to_string();
        assert!(bad.validate("44.20260817.3.2", "x86_64").is_err());
        let mut bad = media.clone();
        bad.release = String::new();
        assert!(bad.validate("", "x86_64").is_err());
        let mut bad = media.clone();
        bad.payload_sha256 = "unreviewed".to_string();
        assert!(bad.validate("44.20260817.3.2", "x86_64").is_err());
        // Strict document: unknown fields refused.
        let doc = format!(
            r#"{{"Architecture":"x86_64","Release":"{}","InstallerVersion":"coreos-installer 0.26.0","Revision":"{}","HostManifest":"{}","PayloadSHA256":"{}","ConsoleSHA256":"{}","Bogus":1}}"#,
            media.release, media.revision, media.host_manifest, media.payload_sha256, media.console_sha256
        );
        assert!(decode_media_identity(doc.as_bytes()).is_err());
        let doc = doc.replace(r#","Bogus":1"#, "");
        assert_eq!(decode_media_identity(doc.as_bytes()).unwrap(), media);
    }

    #[test]
    fn run_refuses_non_root_usage() {
        let (ctx, _flag) = Ctx::test();
        let runner = crate::command::FnRunner::new(|_, _, _, _| Ok(Vec::new()));
        // Effective UID decides; the assertion only pins the usage text for
        // the invalid-action path when root.
        if unsafe { libc::geteuid() } == 0 {
            assert_eq!(run(&ctx, &runner, "bogus").unwrap_err().to_string(), "usage: soda-install disk|configure|enroll-key");
        } else {
            assert_eq!(run(&ctx, &runner, "bogus").unwrap_err().to_string(), "native CoreOS root required");
        }
    }
}

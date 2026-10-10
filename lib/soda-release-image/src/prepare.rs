//! `prepare.go`: public-only host build context staging.
//!
//! Package image prepares a small, public-only host build context. It does
//! not install, migrate, sign or publish an appliance release.

use std::fs;
use std::os::unix::fs::MetadataExt;
use std::os::unix::fs::PermissionsExt;

use serde::Serialize;

use crate::error::Error;
use crate::foreign::Production;
use crate::jsonio;
use crate::model;
use crate::packages;
use crate::sys;
use soda_build_tools::reader::stream::{valid_resolved_coreos, ResolvedCoreOS};

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Base {
    pub release: String,
    pub metadata_url: String,
    pub images: Vec<(String, String)>,
}

impl Base {
    pub fn image(&self, arch: &str) -> String {
        self.images
            .iter()
            .find(|(a, _)| a == arch)
            .map(|(_, image)| image.clone())
            .unwrap_or_default()
    }
}

/// LoadBase resolves the current stable CoreOS build live: release and live
/// media locations from the stream, container digests from the registry.
/// Nothing is read from stored lock files; the resolved values are recorded
/// per build instead. Checks stay shape-strict but host-agnostic so local
/// stream fixtures exercise the same path as production.
pub fn load_base(production: &dyn Production, arch: &str) -> Result<Base, Error> {
    let resolved = production.resolve_core_os()?;
    base_from_resolved(arch, &resolved)
}

/// LoadBaseFromFile admits controller-resolved CoreOS inputs for the isolated
/// worker, which SELinux denies outbound HTTPS: same validation as the live
/// path, no network. The controller records these per build; the worker only
/// consumes its own attempt's file.
pub fn load_base_from_file(
    production: &dyn Production,
    path: &str,
    arch: &str,
) -> Result<Base, Error> {
    let inputs = production.read_live_inputs(path)?;
    base_from_resolved(arch, &inputs.coreos)
}

pub fn base_from_resolved(arch: &str, resolved: &ResolvedCoreOS) -> Result<Base, Error> {
    model::oci_architecture(arch)?;
    valid_resolved_coreos(resolved).map_err(|error| Error(error.0))?;
    Ok(Base {
        release: resolved.release.clone(),
        metadata_url: resolved.metadata_url.clone(),
        images: resolved
            .container
            .iter()
            .map(|(arch, image)| (arch.clone(), image.clone()))
            .collect(),
    })
}

pub struct PreparedWriter {
    pub source: String,
    pub out: String,
}

impl PreparedWriter {
    pub fn write(&self, name: &str, data: &[u8], mode: u32) -> Result<(), Error> {
        let dest = sys::join(&[&self.out, name]);
        fs::create_dir_all(sys::dir_name(&dest))?;
        sys::write_new(&dest, data, mode)?;
        fs::set_permissions(&dest, fs::Permissions::from_mode(mode))?;
        Ok(())
    }

    pub fn copy_file(&self, from: &str, to: &str, mode: u32, vendor: bool) -> Result<(), Error> {
        let path = sys::join(&[&self.source, from]);
        let info = fs::symlink_metadata(&path)?;
        if !info.file_type().is_file() {
            return Err(Error::msg(format!(
                "regular public source required: {from}"
            )));
        }
        let data = fs::read(&path)?;
        let mut text = String::from_utf8_lossy(&data).into_owned();
        if vendor {
            text = text.replace("/usr/local/libexec/soda/", "/usr/libexec/soda/");
        }
        self.write(to, text.as_bytes(), mode)
    }

    pub fn write_base_files(&self, packages: &[String], repo: &str) -> Result<(), Error> {
        self.copy_file("system/host/Containerfile", "Containerfile", 0o644, false)?;
        self.copy_file(
            "system/host/selinux/soda_dashboard.te",
            "selinux/soda_dashboard.te",
            0o644,
            false,
        )?;
        self.copy_file(
            "system/host/selinux/soda_dashboard.fc",
            "selinux/soda_dashboard.fc",
            0o644,
            false,
        )?;
        // Bare names only: the install floats on current repositories and the
        // built image's inventory is recorded as the bill-of-materials.
        // Nothing here pins versions.
        let mut entries = vec![
            ("packages.list".to_string(), packages.join("\n") + "\n"),
            ("tailscale-repo.url".to_string(), repo.to_string() + "\n"),
        ];
        entries.sort_by(|a, b| a.0.cmp(&b.0));
        for (name, text) in entries {
            self.write(&name, text.as_bytes(), 0o644)?;
        }
        Ok(())
    }

    pub fn stage_symlinks_and_extras(&self) -> Result<(), Error> {
        let mut links = vec![
            ("usr/bin/soda-tailnet", "../libexec/soda/soda-tailnet"),
            ("usr/bin/soda-setup", "../libexec/soda/soda-setup"),
            ("usr/bin/soda-install", "../libexec/soda/soda-install"),
        ];
        links.sort();
        for (name, target) in links {
            let dest = sys::join(&[&self.out, "rootfs", name]);
            fs::create_dir_all(sys::dir_name(&dest))?;
            std::os::unix::fs::symlink(target, &dest).map_err(|e| Error::msg(e.to_string()))?;
        }
        self.write("rootfs/etc/cockpit/disallowed-users", b"", 0o644)
    }

    pub fn stage_rootfs_files(&self) -> Result<(), Error> {
        let mut files = rootfs_file_map();
        files.sort();
        for (from, to) in files {
            self.copy_file(&from, &format!("rootfs/{to}"), 0o644, true)?;
        }
        // All appliance/bin operators are compiled by the Rust install table
        // now; no script files remain to stage from source.
        self.stage_symlinks_and_extras()
    }

    pub fn write_build_record(
        &self,
        base: &Base,
        revision: &str,
        arch: &str,
        packages: &[String],
    ) -> Result<(), Error> {
        #[derive(Serialize)]
        struct BuildRecord<'a> {
            #[serde(rename = "Scope")]
            scope: &'static str,
            #[serde(rename = "Revision")]
            revision: &'a str,
            #[serde(rename = "Architecture")]
            architecture: &'a str,
            #[serde(rename = "CoreOS")]
            core_os: &'a str,
            #[serde(rename = "Base")]
            base: String,
            #[serde(rename = "Packages")]
            packages: &'a [String],
        }
        let record = BuildRecord {
            scope: "host-content-only; not an installable or signed appliance release",
            revision,
            architecture: arch,
            core_os: &base.release,
            base: base.image(arch),
            packages,
        };
        let mut encoded =
            serde_json::to_string_pretty(&record).expect("serialization to String cannot fail");
        encoded.push('\n');
        self.write(
            "rootfs/usr/share/soda/host-image/build.json",
            encoded.as_bytes(),
            0o644,
        )?;
        sys::walk(&sys::join(&[&self.out, "rootfs"]), |path, is_dir, _| {
            if is_dir {
                fs::set_permissions(path, fs::Permissions::from_mode(0o755))?;
            }
            Ok(())
        })
    }
}

pub fn rootfs_file_map() -> Vec<(String, String)> {
    let mut files: Vec<(String, String)> = [
        (
            "system/host/config/soda.sysusers",
            "usr/lib/sysusers.d/soda.conf",
        ),
        (
            "system/host/image/packages.tmpfiles",
            "usr/lib/tmpfiles.d/soda-host-packages.conf",
        ),
        (
            "system/host/config/soda.tmpfiles",
            "usr/lib/tmpfiles.d/soda.conf",
        ),
        (
            "system/host/config/90-soda-routing.conf",
            "usr/lib/sysctl.d/90-soda-routing.conf",
        ),
        (
            "system/host/config/cockpit.socket.conf",
            "usr/lib/systemd/system/cockpit.socket.d/10-soda.conf",
        ),
        ("system/host/config/cockpit.pam", "etc/pam.d/cockpit"),
        (
            "system/host/config/cockpit.conf",
            "etc/cockpit/cockpit.conf",
        ),
        (
            "system/host/config/console-welcome.sh",
            "etc/profile.d/soda-console-welcome.sh",
        ),
        ("LICENSE", "usr/share/licenses/soda/LICENSE"),
        ("NOTICE", "usr/share/licenses/soda/NOTICE"),
    ]
    .into_iter()
    .map(|(a, b)| (a.to_string(), b.to_string()))
    .collect();
    for name in [
        "soda-host.service",
        "soda-host.socket",
        "soda-identity.service",
        "soda-identity.socket",
        "soda-identity-runtime.socket",
        "soda-project@.service",
        "soda-tailnet@.service",
        "soda-console.service",
        "soda-extension-install.service",
        "soda-postgres-backup.service",
        "soda-postgres-backup.timer",
        "soda-postgres-init.service",
        "soda-pg-provision.service",
        "soda-forgejo-migrate.service",
    ] {
        files.push((
            format!("system/host/services/{name}"),
            format!("usr/lib/systemd/system/{name}"),
        ));
    }
    for name in [
        "forgejo.container",
        "soda-dashboard.container",
        "soda-proxy.container",
        "soda-postgres.container",
        "soda.network",
    ] {
        files.push((
            format!("system/host/services/{name}"),
            format!("usr/share/containers/systemd/{name}"),
        ));
    }
    files
}

fn load_base_inputs(
    production: &dyn Production,
    source: &str,
    arch: &str,
    revision: &str,
) -> Result<(Base, Vec<String>, String), Error> {
    let base = load_base(production, arch)?;
    finish_base_inputs(source, revision, base)
}

fn load_base_inputs_resolved(
    production: &dyn Production,
    source: &str,
    arch: &str,
    revision: &str,
    inputs: &str,
) -> Result<(Base, Vec<String>, String), Error> {
    let base = load_base_from_file(production, inputs, arch)?;
    finish_base_inputs(source, revision, base)
}

fn finish_base_inputs(
    source: &str,
    revision: &str,
    base: Base,
) -> Result<(Base, Vec<String>, String), Error> {
    if !model::is_revision(revision) {
        return Err(Error::msg("exact source revision required"));
    }
    let data = fs::read(sys::join(&[source, "system/host/provisioning/base.json"]))?;
    let (packages, repo) = packages::package_inputs(&data)?;
    Ok((base, packages, repo))
}

/// Prepare consumes committed source, never a staged appliance, a /var tree
/// or a secret-file directory. Go binaries are compiled separately into the
/// returned context's usr/libexec/soda directory by the caller.
pub fn prepare(
    production: &dyn Production,
    source: &str,
    out: &str,
    arch: &str,
    revision: &str,
) -> Result<Base, Error> {
    let (base, packages, repo) = load_base_inputs(production, source, arch, revision)?;
    finish_prepare(source, out, arch, revision, base, packages, repo)
}

/// PrepareResolved stages the same context from controller-admitted CoreOS
/// inputs for the isolated worker: identical output, no network fetch.
pub fn prepare_resolved(
    production: &dyn Production,
    source: &str,
    out: &str,
    arch: &str,
    revision: &str,
    inputs: &str,
) -> Result<Base, Error> {
    let (base, packages, repo) =
        load_base_inputs_resolved(production, source, arch, revision, inputs)?;
    finish_prepare(source, out, arch, revision, base, packages, repo)
}

fn finish_prepare(
    source: &str,
    out: &str,
    arch: &str,
    revision: &str,
    base: Base,
    packages: Vec<String>,
    repo: String,
) -> Result<Base, Error> {
    sys::fresh_directory(out)?;
    let writer = PreparedWriter {
        source: source.to_string(),
        out: out.to_string(),
    };
    writer.write_base_files(&packages, &repo)?;
    writer.stage_rootfs_files()?;
    writer.write_build_record(&base, revision, arch, &packages)?;
    Ok(base)
}

/// Inventory records the context's exact files/modes and links. This is
/// integrity evidence, not signature verification or a native product
/// acceptance receipt.
pub fn inventory(context: &str) -> Result<(), Error> {
    let mut entries: Vec<(String, sys::File)> = Vec::new();
    sys::walk(context, |path, is_dir, _| {
        if is_dir {
            return Ok(());
        }
        let rel = sys::rel_path(context, path)?;
        let info = fs::symlink_metadata(path)?;
        let mut file = sys::File {
            mode: info.mode() & 0o7777,
            ..sys::File::default()
        };
        if info.file_type().is_symlink() {
            file.link = fs::read_link(path)?.to_string_lossy().into_owned();
        } else {
            file.sha256 = sys::hash_file(path)?;
        }
        entries.push((rel, file));
        Ok(())
    })?;
    entries.sort_by(|a, b| a.0.cmp(&b.0));
    let mut data = serde_json::to_string_pretty(&jsonio::SortedPairs(&entries))
        .expect("serialization to String cannot fail");
    data.push('\n');
    sys::write_new(
        &sys::join(&[&sys::dir_name(context), "context-inventory.json"]),
        data.as_bytes(),
        0o600,
    )
}

#[cfg(test)]
mod tests;

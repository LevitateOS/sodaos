//! `prepare.go`: public-only host build context staging.
//!
//! Package image prepares a small, public-only host build context. It does
//! not install, migrate, sign or publish an appliance release.

use std::fs;
use std::os::unix::fs::MetadataExt;
use std::os::unix::fs::PermissionsExt;

use soda_json::JsonValue;

use crate::error::Error;
use crate::foreign::Production;
use crate::jsonio;
use crate::model;
use crate::packages;
use crate::sys;

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
    base_from_resolved(arch, &inputs.core_os)
}

pub fn base_from_resolved(arch: &str, resolved: &model::ResolvedCoreOS) -> Result<Base, Error> {
    model::oci_architecture(arch)?;
    model::valid_resolved_core_os(resolved)?;
    Ok(Base {
        release: resolved.release.clone(),
        metadata_url: resolved.metadata_url.clone(),
        images: resolved.container.clone(),
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
        self.copy_file(
            "appliance/host.Containerfile",
            "Containerfile",
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
        let record = JsonValue::Object(vec![
            (
                "Scope".to_string(),
                JsonValue::Str(
                    "host-content-only; not an installable or signed appliance release".to_string(),
                ),
            ),
            ("Revision".to_string(), JsonValue::Str(revision.to_string())),
            ("Architecture".to_string(), JsonValue::Str(arch.to_string())),
            ("CoreOS".to_string(), JsonValue::Str(base.release.clone())),
            ("Base".to_string(), JsonValue::Str(base.image(arch))),
            (
                "Packages".to_string(),
                JsonValue::Array(packages.iter().map(|p| JsonValue::Str(p.clone())).collect()),
            ),
        ]);
        let mut encoded = jsonio::to_indent(&record);
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
            "appliance/config/soda.sysusers",
            "usr/lib/sysusers.d/soda.conf",
        ),
        (
            "appliance/host-image/packages.tmpfiles",
            "usr/lib/tmpfiles.d/soda-host-packages.conf",
        ),
        (
            "appliance/config/soda.tmpfiles",
            "usr/lib/tmpfiles.d/soda.conf",
        ),
        (
            "appliance/config/90-soda-routing.conf",
            "usr/lib/sysctl.d/90-soda-routing.conf",
        ),
        (
            "appliance/config/cockpit.socket.conf",
            "usr/lib/systemd/system/cockpit.socket.d/10-soda.conf",
        ),
        ("appliance/config/cockpit.pam", "etc/pam.d/cockpit"),
        ("appliance/config/cockpit.conf", "etc/cockpit/cockpit.conf"),
        (
            "appliance/config/console-welcome.sh",
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
            format!("appliance/services/{name}"),
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
            format!("appliance/services/{name}"),
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
    let data = fs::read(sys::join(&[source, "appliance/provisioning/base.json"]))?;
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
    let value = JsonValue::Object(
        entries
            .into_iter()
            .map(|(name, file)| (name, file.to_json()))
            .collect(),
    );
    let mut data = jsonio::to_indent(&value);
    data.push('\n');
    sys::write_new(
        &sys::join(&[&sys::dir_name(context), "context-inventory.json"]),
        data.as_bytes(),
        0o600,
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_base_inputs_require_exact_revision() {
        // Oracle: Go finishBaseInputs revision gate.
        struct Stub;
        impl Production for Stub {
            fn source(&self) -> &str {
                ""
            }
            fn forgejo_source(&self) -> &str {
                ""
            }
            fn forgejo_revision(&self) -> &str {
                ""
            }
            fn native(&self) -> &str {
                ""
            }
            fn out(&self) -> &str {
                ""
            }
            fn arch(&self) -> &str {
                "x86_64"
            }
            fn revision(&self) -> &str {
                ""
            }
            fn live_inputs(&self) -> &str {
                ""
            }
            fn execute(&self, _: &str, _: &str, _: &[String]) -> Result<(), Error> {
                Ok(())
            }
            fn capture(&self, _: &str, _: &str, _: &[String]) -> Result<String, Error> {
                Ok(String::new())
            }
            fn next(&self, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn resolve_inputs(&mut self) -> Result<(), Error> {
                Ok(())
            }
            fn dependencies(&self) -> Result<(), Error> {
                Ok(())
            }
            fn compile(&self, _: &str, _: &str, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn compile_rust(&self, _: &str, _: &str, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn stage_fork_binary(&self, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn assets(&self, _: &str, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn images(
                &self,
                _: &str,
            ) -> Result<std::collections::HashMap<String, model::ProducedImage>, Error>
            {
                Ok(Default::default())
            }
            fn inspect_oci(&self, _: &str, _: &str, _: &str) -> Result<model::Image, Error> {
                Ok(Default::default())
            }
            fn verify_content(
                &self,
                _: &model::Payload,
                _: &str,
            ) -> Result<(std::collections::HashMap<String, String>, u64), Error> {
                Ok(Default::default())
            }
            fn resolve_core_os(&self) -> Result<model::ResolvedCoreOS, Error> {
                Ok(Default::default())
            }
            fn read_live_inputs(&self, _: &str) -> Result<model::LiveInputs, Error> {
                Ok(Default::default())
            }
            fn check_native(&self, _: &str) -> Result<(), Error> {
                Ok(())
            }
            fn sign_media(
                &self,
                _: &model::Trust,
                _: &model::Permit,
                _: &str,
                _: &str,
                _: &str,
                _: &model::SecretFiles,
                _: &str,
            ) -> Result<(), Error> {
                Ok(())
            }
            fn verify_copy(
                &self,
                _: &model::Trust,
                _: &str,
                _: &str,
                _: &str,
                _: &str,
            ) -> Result<(), Error> {
                Ok(())
            }
            fn write_document(&self, _: &str, _: &JsonValue) -> Result<String, Error> {
                Ok(String::new())
            }
        }
        let stub = Stub;
        assert_eq!(
            load_base(&stub, "not-an-arch").unwrap_err().0,
            "expected x86_64"
        );
        assert_eq!(
            finish_base_inputs("/nonexistent", "short", Base::default())
                .unwrap_err()
                .0,
            "exact source revision required"
        );
    }
}

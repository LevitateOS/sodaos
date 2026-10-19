use crate::sha256::{Digest, Sha256};
use soda_build_tools::payload::{FountainPackage, PayloadV4, Role};
use std::collections::{BTreeMap, BTreeSet};
use std::io::Read;
use std::path::{Component, Path};

use crate::sha256;

use super::{podman, read_file, sha256_file, ProbeFailure};

fn image_config_id_matches(observed: &str, expected: &str) -> bool {
    let Some(expected_hex) = expected.strip_prefix("sha256:") else {
        return false;
    };
    let observed = observed.strip_suffix('\n').unwrap_or(observed);
    let observed_hex = observed.strip_prefix("sha256:").unwrap_or(observed);
    expected_hex.len() == 64
        && expected_hex.bytes().all(|byte| byte.is_ascii_hexdigit())
        && observed_hex == expected_hex
}

/// Read and validate the installed release payload.
fn read_release_payload() -> Result<PayloadV4, ProbeFailure> {
    let payload: PayloadV4 = serde_json::from_slice(&read_file(
        "/usr/share/soda/release.json",
        "read release.json",
    )?)
    .map_err(|_| ProbeFailure::failed("parse release.json"))?;
    payload
        .validate()
        .map_err(|_| ProbeFailure::failed("validate Payload v4"))?;
    Ok(payload)
}

/// Read the installed content inventory.
fn read_content_inventory() -> Result<BTreeMap<String, String>, ProbeFailure> {
    serde_json::from_slice(&read_file(
        "/usr/share/soda/host-image/content.json",
        "read content inventory",
    )?)
    .map_err(|_| ProbeFailure::failed("parse content inventory"))
}

/// Check inventory shape and architecture against the payload.
fn check_inventory_arch(
    payload: &PayloadV4,
    content: &BTreeMap<String, String>,
) -> Result<(), ProbeFailure> {
    if !soda_build_tools::reader::candidate::valid_content(content)
        || payload.architecture != std::env::consts::ARCH
    {
        return Err(ProbeFailure::exit(
            "installed inventory or architecture differs from release",
        ));
    }
    Ok(())
}

/// Check the installed RPM inventory hash against the payload.
fn check_package_inventory(payload: &PayloadV4) -> Result<(), ProbeFailure> {
    if sha256_file(
        Path::new("/usr/share/soda/host-image/packages.txt"),
        "read package inventory",
    )? != payload.host_packages_sha256
    {
        return Err(ProbeFailure::exit(
            "installed RPM inventory differs from release metadata",
        ));
    }
    Ok(())
}

/// Check the installed catalog bytes against the payload projection.
fn check_installed_catalog(payload: &PayloadV4) -> Result<(), ProbeFailure> {
    if read_file(
        "/usr/share/soda/extensions/catalog.json",
        "read installed catalog",
    )? != payload
        .projected_catalog_bytes()
        .map_err(|_| ProbeFailure::failed("project trusted catalog"))?
    {
        return Err(ProbeFailure::exit(
            "installed catalog differs from Payload v4",
        ));
    }
    Ok(())
}

/// Check every acquired image config digest against the payload.
fn check_acquired_images(payload: &PayloadV4) -> Result<(), ProbeFailure> {
    for role in Role::ALL {
        let config = &role.binding(&payload.images).config;
        let observed = podman(
            &["image", "inspect", "--format", "{{.Id}}", config],
            "inspect acquired image",
        )?;
        if !image_config_id_matches(&observed, config) {
            return Err(ProbeFailure::exit(format!(
                "{} image differs from installed release",
                role.name()
            )));
        }
    }
    Ok(())
}

/// Check one inventory entry from host bytes or its acquired image.
fn check_runtime_entry(
    payload: &PayloadV4,
    name: &str,
    expected: &str,
    component: &str,
    path: &str,
) -> Result<(), ProbeFailure> {
    let actual = if component == "host" {
        sha256_file(Path::new(path), "read installed host content")?
    } else if let Some(role) = Role::from_name(component) {
        let output = podman(
            &[
                "run",
                "--rm",
                "--pull=never",
                "--network=none",
                "--read-only",
                "--cap-drop=all",
                "--security-opt=no-new-privileges",
                "--entrypoint=/usr/bin/sha256sum",
                &role.binding(&payload.images).config,
                path,
            ],
            "read acquired image content",
        )?;
        output
            .trim()
            .strip_suffix(&format!("  {path}"))
            .unwrap_or("")
            .to_string()
    } else {
        return Ok(());
    };
    if actual != expected {
        return Err(ProbeFailure::exit(format!(
            "installed runtime content differs: {name}"
        )));
    }
    Ok(())
}

/// Check the runtime content inventory entry by entry.
fn check_runtime_content(
    payload: &PayloadV4,
    content: &BTreeMap<String, String>,
) -> Result<(), ProbeFailure> {
    for (name, expected) in content {
        let (component, path) = name
            .split_once(':')
            .ok_or_else(|| ProbeFailure::failed("parse content inventory"))?;
        check_runtime_entry(payload, name, expected, component, path)?;
    }
    Ok(())
}

/// Check catalog packages against the inventory, verifying required ones.
fn check_extension_packages(
    payload: &PayloadV4,
    content: &BTreeMap<String, String>,
) -> Result<(), ProbeFailure> {
    for package in &payload.extension_catalog.packages {
        let name = package
            .url
            .rsplit('/')
            .next()
            .ok_or_else(|| ProbeFailure::failed("read package URL"))?;
        if content.get(&format!("package:{name}")) != Some(&package.sha256) {
            return Err(ProbeFailure::exit("package inventory differs from catalog"));
        }
        // Optional absence is valid. Lifecycle cases check its effects separately.
        if package.required {
            check_required_package(Path::new("/var/lib/soda/forgejo/gitea/extensions"), package)?;
        }
    }
    Ok(())
}

/// Verify installed runtime content from the current thin-host contract.
/// Release-only proof/source files remain candidate-admission evidence.
pub fn host_content(phase: &str) -> Result<String, ProbeFailure> {
    if !matches!(phase, "activated" | "pre-activation") {
        return Err(ProbeFailure::failed("select host phase"));
    }
    let payload = read_release_payload()?;
    let content = read_content_inventory()?;
    check_inventory_arch(&payload, &content)?;
    check_package_inventory(&payload)?;
    check_installed_catalog(&payload)?;
    check_acquired_images(&payload)?;
    check_runtime_content(&payload, &content)?;
    check_extension_packages(&payload, &content)?;
    Ok("Installed Payload v4, six acquired image configs, runtime inventory and required Soda package bytes match. Optional lifecycle and release-only source evidence remain separate.\n".into())
}

/// Open the verified package cache after size and hash checks.
fn verify_package_cache(
    cache: &Path,
    package: &FountainPackage,
) -> Result<std::fs::File, ProbeFailure> {
    let file = std::fs::File::open(cache)
        .map_err(|_| ProbeFailure::failed("read verified package cache"))?;
    if file
        .metadata()
        .map_err(|_| ProbeFailure::failed("read package size"))?
        .len()
        != package.size
        || sha256_file(cache, "hash cached package")? != package.sha256
    {
        return Err(ProbeFailure::exit(
            "required package cache differs from catalog",
        ));
    }
    Ok(file)
}

/// Collect archive member hashes into the expected installed set.
fn collect_archive_members(
    file: std::fs::File,
    expected: &mut BTreeMap<String, String>,
) -> Result<(), ProbeFailure> {
    let mut archive = tar::Archive::new(flate2::read::GzDecoder::new(file));
    let mut total = 0u64;
    for entry in archive
        .entries()
        .map_err(|_| ProbeFailure::failed("read cached package entries"))?
    {
        let mut entry = entry.map_err(|_| ProbeFailure::failed("read cached package entry"))?;
        let path = entry
            .path()
            .map_err(|_| ProbeFailure::failed("read package path"))?
            .into_owned();
        if path
            .components()
            .any(|part| !matches!(part, Component::Normal(_)))
        {
            return Err(ProbeFailure::exit("unsafe cached package path"));
        }
        if entry.header().entry_type().is_dir() {
            continue;
        }
        total = total
            .checked_add(entry.size())
            .ok_or_else(|| ProbeFailure::failed("bound package contents"))?;
        if !entry.header().entry_type().is_file() || total > 512 << 20 {
            return Err(ProbeFailure::exit("invalid cached package contents"));
        }
        let mut digest = Sha256::new();
        let mut buffer = [0u8; 16384];
        loop {
            let count = entry
                .read(&mut buffer)
                .map_err(|_| ProbeFailure::failed("hash package entry"))?;
            if count == 0 {
                break;
            }
            digest.update(&buffer[..count]);
        }
        let path = path
            .to_str()
            .ok_or_else(|| ProbeFailure::failed("read package path"))?
            .to_string();
        if expected
            .insert(path, sha256::hex_lower(&digest.finalize()))
            .is_some()
        {
            return Err(ProbeFailure::exit("duplicate cached package member"));
        }
    }
    Ok(())
}

fn check_required_package(root: &Path, package: &FountainPackage) -> Result<(), ProbeFailure> {
    let installed = root.join(&package.id);
    let receipt_path = installed.join(".fountain-package.json");
    #[derive(serde::Deserialize)]
    #[serde(deny_unknown_fields)]
    struct Receipt {
        schema: u32,
        package_id: String,
        package_version: String,
        archive_sha256: String,
    }
    let receipt: Receipt = serde_json::from_slice(
        &std::fs::read(&receipt_path)
            .map_err(|_| ProbeFailure::failed("read native package receipt"))?,
    )
    .map_err(|_| ProbeFailure::failed("parse native package receipt"))?;
    if receipt.schema != 1
        || receipt.package_id != package.id
        || receipt.package_version != package.version
        || receipt.archive_sha256 != package.sha256
    {
        return Err(ProbeFailure::exit(
            "required package receipt differs from catalog",
        ));
    }
    let cache = root
        .join(".cache/packages")
        .join(format!("{}.tar.gz", package.sha256));
    let file = verify_package_cache(&cache, package)?;
    let mut expected = BTreeMap::from([(
        ".fountain-package.json".to_string(),
        sha256_file(&receipt_path, "hash package receipt")?,
    )]);
    collect_archive_members(file, &mut expected)?;
    check_extension_package(&installed, expected)
}

fn check_extension_package(
    root: &Path,
    expected: BTreeMap<String, String>,
) -> Result<(), ProbeFailure> {
    let ordered: Vec<String> = expected.keys().cloned().collect();
    let meta = std::fs::symlink_metadata(root).map_err(|_| {
        ProbeFailure::exit("installed Soda extension package is not a regular directory")
    })?;
    if !meta.is_dir() || meta.is_symlink() {
        return Err(ProbeFailure::exit(
            "installed Soda extension package is not a regular directory",
        ));
    }
    let mut observed = BTreeSet::new();
    let mut stack = vec![root.to_path_buf()];
    while let Some(dir) = stack.pop() {
        let mut children: Vec<std::path::PathBuf> = Vec::new();
        for entry in
            std::fs::read_dir(&dir).map_err(|_| ProbeFailure::failed("walk extension package"))?
        {
            children.push(
                entry
                    .map_err(|_| ProbeFailure::failed("walk extension package"))?
                    .path(),
            );
        }
        children.sort();
        for path in children {
            let file_type = std::fs::symlink_metadata(&path)
                .map_err(|_| ProbeFailure::failed("walk extension package"))?
                .file_type();
            if file_type.is_symlink() {
                return Err(ProbeFailure::exit(
                    "installed Soda extension package contains a symlink",
                ));
            }
            if file_type.is_dir() {
                stack.push(path);
            } else if file_type.is_file() {
                let relative = path
                    .strip_prefix(root)
                    .map_err(|_| ProbeFailure::failed("walk extension package"))?;
                let relative = relative.to_string_lossy().replace('\\', "/");
                observed.insert(relative);
            }
        }
    }
    let want: BTreeSet<String> = expected.keys().cloned().collect();
    if observed != want {
        return Err(ProbeFailure::exit(
            "installed Soda package file set differs from verified archive",
        ));
    }
    for relative in &ordered {
        let actual = sha256_file(&root.join(relative), "read extension package")?;
        if actual != expected[relative.as_str()] {
            return Err(ProbeFailure::exit(format!(
                "installed package differs from verified archive: {relative}"
            )));
        }
    }
    Ok(())
}

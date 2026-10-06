use std::collections::{BTreeSet, HashMap};

use soda_json::JsonValue;

use crate::sha256;

use super::{get_str, parse_json, podman, read_file, read_text, sha256_file, ProbeFailure};

/// Installed content inventory, images, and (when activated) the
/// extension package: the first `host.sh` Python block.
pub fn host_content(phase: &str) -> Result<String, ProbeFailure> {
    let payload = parse_json(
        &read_text("/usr/share/soda/release.json", "read release.json")?,
        "parse release.json",
    )?;
    let content = parse_json(
        &read_text(
            "/usr/share/soda/host-image/content.json",
            "read content inventory",
        )?,
        "parse content inventory",
    )?;
    let fixed = BTreeSet::from([
        "dashboard:/usr/local/bin/soda-dashboard",
        "forgejo:/usr/local/bin/gitea",
        "extension:/usr/local/bin/gitea",
        "extension:/usr/share/soda/extension/extension.json",
        "extension:/usr/share/soda/extension/backend",
        "extension:/usr/share/soda/extension/run",
        "host:/usr/share/containers/systemd/forgejo.container",
        "host:/usr/share/containers/systemd/soda-dashboard.container",
        "host:/usr/lib/systemd/system/soda-extension-install.service",
    ]);
    let asset_prefix = "extension:/usr/share/soda/extension/assets/";
    let JsonValue::Object(entries) = &content else {
        return Err(ProbeFailure::failed("parse content inventory"));
    };
    let names: BTreeSet<&str> = entries.iter().map(|(k, _)| k.as_str()).collect();
    let assets: BTreeSet<&str> = names
        .iter()
        .copied()
        .filter(|name| name.starts_with(asset_prefix))
        .collect();
    let mut want = fixed.clone();
    want.extend(assets.iter().copied());
    if assets.is_empty() || names != want {
        return Err(ProbeFailure::exit("incomplete installed content inventory"));
    }
    for name in &assets {
        let relative = name.strip_prefix(asset_prefix).unwrap_or("");
        if relative.is_empty()
            || relative.split('/').any(|part| part == "..")
            || relative.starts_with('/')
        {
            return Err(ProbeFailure::exit(
                "unsafe installed extension asset inventory",
            ));
        }
    }
    if get_str(&payload, "Architecture", "parse release.json")? != std::env::consts::ARCH {
        return Err(ProbeFailure::exit(
            "installed release architecture differs from native host",
        ));
    }
    let packages = read_file(
        "/usr/share/soda/host-image/packages.txt",
        "read package inventory",
    )?;
    if sha256::hex_lower(&sha256::digest(&packages))
        != get_str(&payload, "HostPackagesSHA256", "parse release.json")?
    {
        return Err(ProbeFailure::exit(
            "installed RPM inventory differs from release metadata",
        ));
    }
    let images = payload
        .get("Images")
        .ok_or_else(|| ProbeFailure::failed("parse release.json"))?;
    let config = |name: &str| -> Result<&str, ProbeFailure> {
        images
            .get(name)
            .and_then(|image| image.get("Config"))
            .and_then(|v| v.as_str())
            .ok_or_else(|| ProbeFailure::failed("parse release.json"))
    };
    if config("forgejo")? == config("extension")? {
        return Err(ProbeFailure::exit("independent extension image required"));
    }
    let content_digest = |name: &str| -> Result<&str, ProbeFailure> {
        content
            .get(name)
            .and_then(|v| v.as_str())
            .ok_or_else(|| ProbeFailure::failed("parse content inventory"))
    };
    if content_digest("forgejo:/usr/local/bin/gitea")?
        != content_digest("extension:/usr/local/bin/gitea")?
    {
        return Err(ProbeFailure::exit(
            "extension installer CLI differs from patched Forgejo",
        ));
    }
    for name in ["dashboard", "forgejo", "extension"] {
        let image = config(name)?;
        let observed = podman(
            &["image", "inspect", "--format", "{{.Id}}", image],
            "inspect installed image",
        )?;
        if observed.trim() != image {
            return Err(ProbeFailure::exit(format!(
                "{name} image identity differs from installed release"
            )));
        }
    }
    for (name, expected) in entries {
        let expected = expected
            .as_str()
            .ok_or_else(|| ProbeFailure::failed("parse content inventory"))?;
        let Some((component, path)) = name.split_once(':') else {
            return Err(ProbeFailure::failed("parse content inventory"));
        };
        let actual = if component == "host" {
            sha256_file(std::path::Path::new(path), "read installed content")?
        } else {
            let image = config(component)?;
            let output = podman(
                &[
                    "run",
                    "--rm",
                    "--network=none",
                    "--read-only",
                    "--cap-drop=all",
                    "--security-opt=no-new-privileges",
                    "--entrypoint=/usr/bin/sha256sum",
                    image,
                    path,
                ],
                "read installed image content",
            )?;
            let output = output.trim();
            match output.strip_suffix(&format!("  {path}")) {
                Some(hash) => hash.to_string(),
                None => String::new(),
            }
        };
        if actual != expected {
            return Err(ProbeFailure::exit(format!(
                "installed content differs from release inventory: {name}"
            )));
        }
    }
    let checked = if phase == "activated" {
        check_extension_package(&content)?;
        "installed package replacement"
    } else {
        "extension image package"
    };
    Ok(format!("Installed fork, {checked}, Soda service bytes and native architecture match the release inventory.\n"))
}

fn check_extension_package(content: &JsonValue) -> Result<(), ProbeFailure> {
    let package_prefix = "extension:/usr/share/soda/extension/";
    let JsonValue::Object(entries) = content else {
        return Err(ProbeFailure::failed("parse content inventory"));
    };
    let mut expected = HashMap::new();
    let mut ordered: Vec<String> = Vec::new();
    for (name, digest) in entries {
        if let Some(relative) = name.strip_prefix(package_prefix) {
            expected.insert(
                relative.to_string(),
                digest
                    .as_str()
                    .ok_or_else(|| ProbeFailure::failed("parse content inventory"))?,
            );
            ordered.push(relative.to_string());
        }
    }
    let root = std::path::Path::new("/var/lib/soda/forgejo/gitea/extensions/soda");
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
                if relative != ".disabled" {
                    observed.insert(relative);
                }
            }
        }
    }
    let want: BTreeSet<String> = expected.keys().cloned().collect();
    if observed != want {
        return Err(ProbeFailure::exit(
            "installed Soda extension package differs from candidate inventory",
        ));
    }
    for relative in &ordered {
        let actual = sha256_file(&root.join(relative), "read extension package")?;
        if actual != expected[relative.as_str()] {
            return Err(ProbeFailure::exit(format!(
                "installed package replacement differs from candidate inventory: {relative}"
            )));
        }
    }
    Ok(())
}

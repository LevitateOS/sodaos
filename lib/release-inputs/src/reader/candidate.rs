//! Candidate inventory shared by production and release admission.
use super::is_digest;
use std::collections::BTreeMap;

pub const REQUIRED_CONTENT: [&str; 15] = [
    "dashboard:/usr/local/bin/soda-dashboard",
    "host:/usr/share/containers/systemd/forgejo.container",
    "host:/usr/share/containers/systemd/soda-dashboard.container",
    "host:/usr/share/containers/systemd/soda-proxy.container",
    "host:/usr/share/containers/systemd/soda-postgres.container",
    "host:/usr/share/soda/extensions/catalog.json",
    "host:/usr/share/soda/canlang-tools/release.json",
    "host:/usr/share/soda/canlang-tools/canlang-tools-linux-x86_64.tar.gz",
    "canlang-runtime:/opt/canlang/tools.json",
    "canlang-runtime:/usr/share/licenses/canlang/Canlang-LICENSE",
    "canlang-runtime:/usr/share/licenses/canlang/Node-LICENSE",
    "canlang-runtime:/usr/share/licenses/canlang/workerd-LICENSE",
    "canlang-runtime:/usr/share/licenses/canlang/Miniflare-D1-LICENSE",
    "canlang-runtime:/usr/share/licenses/canlang/D1-NOTICE",
    "release:fountain-release.json",
];

fn package_id(name: &str) -> Option<&'static str> {
    let name = name
        .strip_prefix("package:")?
        .strip_suffix("-linux-x86_64.tar.gz")?;
    for id in ["soda", "tailnet"] {
        if let Some(version) = name.strip_prefix(&format!("{id}-extension-")) {
            return crate::payload::valid_release_tag(&format!("v{version}")).then_some(id);
        }
    }
    None
}

fn valid_asset_name(name: &str) -> bool {
    !name.is_empty()
        && !name.starts_with('/')
        && name
            .split('/')
            .all(|part| !part.is_empty() && part != "." && part != "..")
        && !name.bytes().any(|b| matches!(b, b'\\' | b'\n' | b'\r' | 0))
}

pub fn valid_content(files: &BTreeMap<String, String>) -> bool {
    if REQUIRED_CONTENT
        .iter()
        .any(|name| !files.get(*name).is_some_and(|hash| is_digest(hash)))
    {
        return false;
    }
    let mut soda = 0;
    let mut tailnet = 0;
    for (name, hash) in files {
        if !is_digest(hash) {
            return false;
        }
        match package_id(name) {
            Some("soda") => {
                soda += 1;
                continue;
            }
            Some("tailnet") => {
                tailnet += 1;
                continue;
            }
            _ => {}
        }
        if let Some(asset) = name
            .strip_prefix("release:")
            .or_else(|| name.strip_prefix("canlang-runtime:/usr/share/licenses/canlang/"))
        {
            if !valid_asset_name(asset) {
                return false;
            }
        } else if !REQUIRED_CONTENT.contains(&name.as_str()) {
            return false;
        }
    }
    soda == 1 && tailnet == 1
}

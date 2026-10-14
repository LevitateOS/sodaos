use super::release::ReleaseImage;

#[allow(clippy::too_many_arguments)]
pub(crate) fn validate_release_payload(
    format: i64,
    id: &str,
    revision: &str,
    architecture: &str,
    coreos: &str,
    base: &str,
    repository_prefix: &str,
    schema: i64,
    presentation: &str,
    host_packages: &str,
    images: &[(String, ReleaseImage)],
    upgrade_from: &[String],
) -> Result<(), ()> {
    if format != 3
        || !is_hex_string(revision, 40)
        || id != format!("{coreos}.soda-{}", &revision[..12])
        || !valid_coreos_version(coreos)
        || architecture != "x86_64"
        || !valid_repository_prefix(repository_prefix)
        || schema < 1
        || !is_hex_string(presentation, 64)
        || !is_hex_string(host_packages, 64)
        || !upgrade_from.is_empty()
    {
        return Err(());
    }
    let (host, rest) = match base.split_once("/fedora/fedora-coreos@sha256:") {
        Some(v) => v,
        None => return Err(()),
    };
    if host.is_empty() || host.contains('/') || !is_hex_string(rest, 64) {
        return Err(());
    }
    const NAMES: [&str; 6] = [
        "dashboard",
        "forgejo",
        "extension",
        "proxy",
        "project-os",
        "tailnet",
    ];
    if images.len() != NAMES.len() {
        return Err(());
    }
    for name in NAMES {
        let image = &images.iter().find(|(n, _)| n == name).ok_or(())?.1;
        if !is_digest(&image.config)
            || !is_digest(&image.manifest)
            || !is_hex_string(&image.archive_sha256, 64)
            || image.reference != format!("{repository_prefix}-{name}@{}", image.manifest)
        {
            return Err(());
        }
    }
    let ext = &images.iter().find(|(n, _)| n == "extension").ok_or(())?.1;
    let forgejo = &images.iter().find(|(n, _)| n == "forgejo").ok_or(())?.1;
    if ext.config == forgejo.config {
        return Err(());
    }
    Ok(())
}

pub(crate) fn is_hex_string(s: &str, len: usize) -> bool {
    // Go's [0-9a-f] classes: lowercase hex only.
    s.len() == len && s.bytes().all(super::is_lower_hex)
}

fn is_digest(s: &str) -> bool {
    match s.strip_prefix("sha256:") {
        Some(hex) => is_hex_string(hex, 64),
        None => false,
    }
}

fn valid_coreos_version(s: &str) -> bool {
    let parts: Vec<&str> = s.split('.').collect();
    parts.len() == 4
        && parts
            .iter()
            .all(|p| !p.is_empty() && p.bytes().all(|b| b.is_ascii_digit()))
}

fn valid_repository_prefix(s: &str) -> bool {
    if s.len() >= 200 {
        return false;
    }
    let rest = match s.strip_prefix("ghcr.io/") {
        Some(r) => r,
        None => return false,
    };
    let (org, name) = match rest.split_once('/') {
        Some(v) => v,
        None => return false,
    };
    !org.is_empty()
        && !name.is_empty()
        && org
            .bytes()
            .next()
            .is_some_and(|b| b.is_ascii_lowercase() || b.is_ascii_digit())
        && org
            .bytes()
            .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'-')
        && (name
            .bytes()
            .next()
            .is_some_and(|b| b.is_ascii_lowercase() || b.is_ascii_digit()))
        && name.bytes().all(|b| {
            b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'.' || b == b'_' || b == b'-'
        })
        && !rest[org.len() + 1..].contains('/')
}

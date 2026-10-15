use super::config::Config;

pub(crate) fn validate_runtime_config(c: &Config) -> Result<(), String> {
    // Order matches validateRuntimeConfig: muse, identity, subnet,
    // then network names.
    validate_muse_runtime(c)?;
    validate_identity_runtime(c)?;
    super::network::parse_prefix(&c.subnet)?;
    if !valid_network_names(&c.image, &c.network, &c.bridge) {
        return Err(String::from("invalid native runtime configuration"));
    }
    Ok(())
}

fn validate_muse_runtime(c: &Config) -> Result<(), String> {
    if c.muse_sha256.is_empty() {
        return Ok(());
    }
    if !c.muse_socket.starts_with('/')
        || !c.identity_socket.starts_with('/')
        || !super::release::is_hex_string(&c.muse_sha256, 64)
        || c.muse_version.is_empty()
        || super::filesystem::go_base(&c.muse_socket) != "launch.sock"
    {
        return Err(String::from(
            "explicit muse socket, broker socket and release digest required",
        ));
    }
    Ok(())
}

fn validate_identity_runtime(c: &Config) -> Result<(), String> {
    for (harness, digest, version) in [
        (
            &c.codex_harness,
            &c.codex_harness_sha256,
            &c.codex_harness_version,
        ),
        (
            &c.muse_harness,
            &c.muse_harness_sha256,
            &c.muse_harness_version,
        ),
    ] {
        if harness.is_empty() {
            continue;
        }
        if !is_clean_absolute_path(&c.identity_socket)
            || !is_clean_absolute_path(harness)
            || digest.len() != 64
        {
            return Err(String::from(
                "explicit identity runtime socket and verified harness required",
            ));
        }
        if !version.is_empty() && !valid_harness_version(version) {
            return Err(String::from("invalid staged harness version"));
        }
    }
    Ok(())
}

fn is_clean_absolute_path(path: &str) -> bool {
    use std::path::{Component, Path, PathBuf};

    if !Path::new(path).is_absolute() {
        return false;
    }
    let mut rebuilt = PathBuf::new();
    for component in Path::new(path).components() {
        match component {
            Component::RootDir => rebuilt.push("/"),
            Component::Normal(part) => rebuilt.push(part),
            Component::CurDir | Component::ParentDir | Component::Prefix(_) => return false,
        }
    }
    rebuilt.to_str() == Some(path)
}

fn valid_harness_version(s: &str) -> bool {
    // ^[A-Za-z0-9][A-Za-z0-9._-]{0,31}$
    let mut chars = s.bytes();
    match chars.next() {
        Some(b) if b.is_ascii_alphanumeric() => {}
        _ => return false,
    }
    let rest: Vec<u8> = chars.collect();
    rest.len() <= 31
        && rest
            .iter()
            .all(|b| b.is_ascii_alphanumeric() || matches!(b, b'.' | b'_' | b'-'))
}

fn valid_network_names(image: &str, network: &str, bridge: &str) -> bool {
    if image.is_empty() || image.starts_with('-') {
        return false;
    }
    valid_network_name(network) && valid_network_name(bridge)
}

fn valid_network_name(s: &str) -> bool {
    // ^[a-z][a-z0-9_-]{0,30}$
    let mut bytes = s.bytes();
    match bytes.next() {
        Some(b) if b.is_ascii_lowercase() => {}
        _ => return false,
    }
    let rest: Vec<u8> = bytes.collect();
    rest.len() <= 30
        && rest
            .iter()
            .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || matches!(b, b'_' | b'-'))
}

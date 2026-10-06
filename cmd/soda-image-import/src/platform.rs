// ---------- native platform + identifier shapes ----------

pub(super) fn oci_architecture(arch: &str) -> Result<&'static str, String> {
    if arch == "x86_64" {
        Ok("amd64")
    } else {
        Err("expected x86_64".to_string())
    }
}

pub(super) fn require_native(arch: &str) -> Result<(), String> {
    let want = oci_architecture(arch)?;
    // `std::env::consts::ARCH` uses rustc names; map to the Go values the
    // payload architecture implies.
    let native = match std::env::consts::ARCH {
        "x86_64" => "amd64",
        "aarch64" => "arm64",
        other => other,
    };
    if std::env::consts::OS != "linux" || native != want {
        return Err("matching-native Linux required".to_string());
    }
    Ok(())
}

fn is_lower_hex(s: &str, len: usize) -> bool {
    s.len() == len && s.bytes().all(|b| matches!(b, b'0'..=b'9' | b'a'..=b'f'))
}

pub(super) fn is_digest(s: &str) -> bool {
    is_lower_hex(s, 64)
}

pub(super) fn is_revision(s: &str) -> bool {
    is_lower_hex(s, 40)
}

pub(super) fn is_prefixed_digest(s: &str) -> bool {
    s.len() == "sha256:".len() + 64 && s.starts_with("sha256:") && is_digest(&s[7..])
}

/// `^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$` without regex.
pub(super) fn is_coreos_version(s: &str) -> bool {
    let mut parts = s.split('.');
    for _ in 0..4 {
        match parts.next() {
            Some(p) if !p.is_empty() && p.bytes().all(|b| b.is_ascii_digit()) => {}
            _ => return false,
        }
    }
    parts.next().is_none()
}

/// `^ghcr\.io/[a-z0-9][a-z0-9-]*/[a-z0-9][a-z0-9._-]*$`, byte length < 200.
pub(super) fn valid_repository_prefix(s: &str) -> bool {
    if s.len() >= 200 {
        return false;
    }
    let rest = match s.strip_prefix("ghcr.io/") {
        Some(r) => r,
        None => return false,
    };
    let (owner, repo) = match rest.split_once('/') {
        Some(p) => p,
        None => return false,
    };
    if repo.contains('/') {
        return false;
    }
    let mut owner_bytes = owner.bytes();
    match owner_bytes.next() {
        Some(b'a'..=b'z' | b'0'..=b'9') => {}
        _ => return false,
    }
    if !owner_bytes.all(|b| matches!(b, b'a'..=b'z' | b'0'..=b'9' | b'-')) {
        return false;
    }
    let mut repo_bytes = repo.bytes();
    match repo_bytes.next() {
        Some(b'a'..=b'z' | b'0'..=b'9') => {}
        _ => return false,
    }
    repo_bytes.all(|b| matches!(b, b'a'..=b'z' | b'0'..=b'9' | b'.' | b'_' | b'-'))
}

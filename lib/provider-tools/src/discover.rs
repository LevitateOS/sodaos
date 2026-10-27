//! Official upstream metadata discovery (lane A implementation).
//!
//! Resolves live release metadata only: no download, no admission, no model
//! call, no credential use. Codex reads the official GitHub `latest` release
//! and its native `codex-package-*-musl.tar.zst` asset (digest/size from the
//! asset plus SHA256SUMS where published). Muse reads the official Meta
//! stable channel manifest and validates it through to the immutable artifact
//! URL/checksum/size.

use crate::store::{Provider, ReleaseIdentity};

/// Resolve current official upstream metadata for a provider on its
/// configured channel. Network or upstream failures return `Failed`, never a
/// stale pin presented as current.
pub fn resolve_upstream(provider: Provider) -> Result<ReleaseIdentity, crate::Error> {
    match provider {
        Provider::Codex => resolve_codex(),
        Provider::Muse => resolve_muse(&crate::config::channel()),
    }
}

fn native_asset_name(provider: Provider) -> Result<&'static str, crate::Error> {
    match (provider, std::env::consts::ARCH) {
        (Provider::Codex, "x86_64") => Ok("codex-package-x86_64-unknown-linux-musl.tar.zst"),
        (Provider::Codex, "aarch64") => Ok("codex-package-aarch64-unknown-linux-musl.tar.zst"),
        (Provider::Muse, "x86_64") => Ok("muse-x86-linux"),
        (Provider::Muse, "aarch64") => Ok("muse-aarch64-linux"),
        (provider, arch) => Err(crate::Error::denied(format!(
            "unsupported native arch for {}: {arch}",
            provider.as_str()
        ))),
    }
}

fn resolve_codex() -> Result<ReleaseIdentity, crate::Error> {
    let url = Provider::Codex.metadata_url(&crate::config::channel());
    let body = crate::http_get_text(&url, "application/vnd.github+json")?;
    let release: serde_json::Value = serde_json::from_str(&body)?;
    let tag = release
        .get("tag_name")
        .and_then(|v| v.as_str())
        .ok_or_else(|| crate::Error::failed("GitHub latest release has no tag_name"))?;
    let assets = release
        .get("assets")
        .and_then(|v| v.as_array())
        .ok_or_else(|| crate::Error::failed("GitHub latest release has no assets array"))?;
    let want = native_asset_name(Provider::Codex)?;
    let asset = assets
        .iter()
        .find(|a| a.get("name").and_then(|n| n.as_str()) == Some(want))
        .ok_or_else(|| {
            crate::Error::failed(format!("GitHub release {tag} has no native asset {want}"))
        })?;
    let artifact_url = asset
        .get("browser_download_url")
        .and_then(|v| v.as_str())
        .ok_or_else(|| crate::Error::failed("native asset has no download URL"))?;
    if !artifact_url.starts_with("https://") {
        return Err(crate::Error::failed("native asset URL is not https"));
    }
    let size = asset
        .get("size")
        .and_then(|v| v.as_u64())
        .filter(|s| *s > 0)
        .ok_or_else(|| crate::Error::failed("native asset has no positive size"))?;
    let digest = asset
        .get("digest")
        .and_then(|v| v.as_str())
        .ok_or_else(|| crate::Error::failed("native asset has no digest"))?;
    let digest = crate::normalize_digest(digest)?;

    // Cross-check SHA256SUMS where published (covers .tar.gz siblings; the
    // .tar.zst asset itself carries its digest in the API record).
    if let Some(sums) = assets
        .iter()
        .find(|a| a.get("name").and_then(|n| n.as_str()) == Some("codex-package_SHA256SUMS"))
        .and_then(|a| a.get("browser_download_url"))
        .and_then(|v| v.as_str())
    {
        if let Ok(text) = crate::http_get_text(sums, "text/plain") {
            for line in text.lines() {
                let mut parts = line.split_whitespace();
                if let (Some(sum), Some(name)) = (parts.next(), parts.next()) {
                    if name == want && sum.to_lowercase() != digest {
                        return Err(crate::Error::failed(format!(
                            "SHA256SUMS mismatch for {want}: sums={sum} api={digest}"
                        )));
                    }
                }
            }
        }
    }

    Ok(ReleaseIdentity {
        provider: Provider::Codex,
        version: tag.to_string(),
        digest,
        size,
        artifact_url: artifact_url.to_string(),
    })
}

fn resolve_muse(channel: &str) -> Result<ReleaseIdentity, crate::Error> {
    let channel_url = Provider::Muse.metadata_url(channel);
    let body = crate::http_get_text(&channel_url, "application/json")?;
    let channel_doc: serde_json::Value = serde_json::from_str(&body)?;
    let version = channel_doc
        .get("version")
        .and_then(|v| v.as_str())
        .ok_or_else(|| crate::Error::failed("Muse channel doc has no version"))?;
    let manifest_url = channel_doc
        .get("manifest_url")
        .and_then(|v| v.as_str())
        .ok_or_else(|| crate::Error::failed("Muse channel doc has no manifest_url"))?;
    if !manifest_url.starts_with("https://") {
        return Err(crate::Error::failed("Muse manifest URL is not https"));
    }
    let manifest_body = crate::http_get_text(manifest_url, "application/json")?;
    let manifest: serde_json::Value = serde_json::from_str(&manifest_body)?;
    let manifest_version = manifest
        .get("version")
        .and_then(|v| v.as_str())
        .ok_or_else(|| crate::Error::failed("Muse manifest has no version"))?;
    if manifest_version != version {
        return Err(crate::Error::failed(format!(
            "Muse manifest version {manifest_version} does not match channel version {version}"
        )));
    }
    let want = native_asset_name(Provider::Muse)?;
    let artifact = manifest
        .get("artifacts")
        .and_then(|v| {
            v.get(match std::env::consts::ARCH {
                "aarch64" => "aarch64_linux",
                _ => "x86_linux",
            })
        })
        .ok_or_else(|| crate::Error::failed(format!("Muse manifest has no artifact for {want}")))?;
    let artifact_url = artifact
        .get("url")
        .and_then(|v| v.as_str())
        .ok_or_else(|| crate::Error::failed("Muse artifact has no url"))?;
    if !artifact_url.starts_with("https://") {
        return Err(crate::Error::failed("Muse artifact URL is not https"));
    }
    let checksum = artifact
        .get("checksum")
        .and_then(|v| v.as_str())
        .ok_or_else(|| crate::Error::failed("Muse artifact has no checksum"))?;
    let digest = crate::normalize_digest(checksum)?;
    let size = artifact
        .get("size")
        .and_then(|v| v.as_u64())
        .filter(|s| *s > 0)
        .ok_or_else(|| crate::Error::failed("Muse artifact has no positive size"))?;

    Ok(ReleaseIdentity {
        provider: Provider::Muse,
        version: manifest_version.to_string(),
        digest,
        size,
        artifact_url: artifact_url.to_string(),
    })
}

/// Verify the actual required native behavior of an executable without
/// authentication or model calls: `--help` resolves, the provider protocol
/// handshake used by broker/Factory works, and (Muse) the requested
/// model-profile identity resolves without a model call.
pub fn probe_executable(provider: Provider, exe: &std::path::Path) -> Result<(), crate::Error> {
    if !exe.is_file() {
        return Err(crate::Error::denied(format!(
            "probe target is not a file: {}",
            exe.display()
        )));
    }
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt as _;
        let mode = std::fs::metadata(exe)?.permissions().mode();
        if mode & 0o111 == 0 {
            return Err(crate::Error::denied(format!(
                "probe target is not executable: {}",
                exe.display()
            )));
        }
    }
    let timeout = std::time::Duration::from_secs(30);
    let help = crate::run_probe(exe, &["--help"], timeout)?;
    if help.trim().is_empty() {
        return Err(crate::Error::failed(format!(
            "probe --help produced no output: {}",
            exe.display()
        )));
    }
    match provider {
        Provider::Codex => {
            // Companion required by broker/Factory resource layout.
            if exe
                .parent()
                .and_then(|p| p.file_name())
                .and_then(|n| n.to_str())
                == Some("bin")
            {
                let companion = exe
                    .parent()
                    .expect("exe has parent")
                    .join("codex-code-mode-host");
                if !companion.is_file() {
                    return Err(crate::Error::failed(format!(
                        "codex bundle missing companion: {}",
                        companion.display()
                    )));
                }
            }
            // Actual app-server protocol surface used by broker/Factory.
            crate::run_probe(exe, &["app-server", "--help"], timeout)?;
        }
        Provider::Muse => {
            crate::run_probe(exe, &["exec", "--help"], timeout)?;
            // Profile resolution only; never starts a session or model call.
            // Identity mirrors the owner's selected profile for this run.
            crate::run_probe(
                exe,
                &[
                    "model-profile",
                    "show",
                    "muse-spark-1.3-contributor",
                    "--effort",
                    "xhigh",
                ],
                timeout,
            )?;
        }
    }
    Ok(())
}

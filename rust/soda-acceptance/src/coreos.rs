//! Stable CoreOS build resolution for VM launches, mirroring the
//! `ResolveCoreOSQEMU` closure of `internal/release/build/coreos_stream.go`.
//!
//! TLS is delegated to host `curl`, like SSH is delegated to OpenSSH:
//! no hand-rolled crypto. Fetch semantics mirror the Go owner (HTTPS
//! only, at most five redirects, 60s cap, bounded bodies), but
//! transport-failure details come from curl rather than Go's HTTP
//! client. Stream/registry document parsing and shape rules are exact.
//!
//! Only the QEMU closure is ported: the Tailnet/live-input/download
//! surface belongs to Tier-3 builders that stay in Go.

use soda_build_tools::reader::stream::{valid_stream_images, CoreOSImage};
use soda_build_tools::reader::url::https_url;
use soda_build_tools::reader::{is_digest, oci_architecture};
use soda_json::JsonValue;

use crate::command::{CommandSpec, StdinSpec};
use crate::error::Error;
use crate::files::TempDir;
use crate::process::{self, Phase, SharedWriter};

/// Default stable stream document.
const DEFAULT_STREAM_URL: &str = "https://builds.coreos.fedoraproject.org/streams/stable.json";
/// Default container registry.
const DEFAULT_REGISTRY: &str = "https://quay.io";
/// Streamed container repository and tag.
const CONTAINER_REPO: &str = "fedora/fedora-coreos";
/// Pinned container tag.
const CONTAINER_TAG: &str = "stable";

/// Stream endpoint, overridable for fixtures.
fn stream_url() -> String {
    match std::env::var("SODA_COREOS_STREAM_URL").map(|value| value.trim().to_string()) {
        Ok(url) if !url.is_empty() => url,
        _ => DEFAULT_STREAM_URL.to_string(),
    }
}

/// Registry endpoint, overridable for fixtures.
fn registry_url() -> String {
    match std::env::var("SODA_COREOS_REGISTRY").map(|value| value.trim().to_string()) {
        // `TrimSuffix` drops one trailing slash, not a run of them.
        Ok(url) if !url.is_empty() => url.strip_suffix('/').unwrap_or(&url).to_string(),
        _ => DEFAULT_REGISTRY.to_string(),
    }
}

/// Release metadata URL for one stream endpoint, like the Go owner's
/// `streamReleaseURL`. The QEMU closure discards the URL, but a stream
/// endpoint that names no stream still fails resolution.
fn stream_release_url(stream_url: &str, release: &str) -> Result<String, Error> {
    let Some((prefix, _)) = stream_url.split_once("/streams/") else {
        return Err(Error::msg("CoreOS stream URL names no stream"));
    };
    let meta = format!("{prefix}/prod/streams/stable/builds/{release}/release.json");
    if !https_url(&meta) {
        return Err(Error::msg("CoreOS release metadata URL is malformed"));
    }
    Ok(meta)
}

fn curl_spec(
    url: &str,
    body: &str,
    headers: &str,
    extra: &[String],
    max_filesize: u64,
) -> CommandSpec {
    let mut args = vec![
        "--silent".to_string(),
        "--show-error".to_string(),
        "--location".to_string(),
        "--max-redirs".to_string(),
        "5".to_string(),
        "--max-time".to_string(),
        "60".to_string(),
        "--proto".to_string(),
        "=https".to_string(),
        "--proto-redir".to_string(),
        "=https".to_string(),
        "--max-filesize".to_string(),
        max_filesize.to_string(),
        "-o".to_string(),
        body.to_string(),
        "-D".to_string(),
        headers.to_string(),
        "-w".to_string(),
        "%{http_code} %{url_effective}".to_string(),
    ];
    args.extend(extra.iter().cloned());
    args.push(url.to_string());
    CommandSpec {
        name: "curl".to_string(),
        args,
        dir: None,
        stdin: StdinSpec::Null,
        env: Vec::new(),
    }
}

/// Last HTTP status line wins across redirect header blocks.
fn status_text(headers: &str) -> String {
    let mut status = String::new();
    for line in headers.lines() {
        if line.starts_with("HTTP/") {
            if let Some((_, rest)) = line.split_once(' ') {
                status = rest.trim_end().to_string();
            }
        }
    }
    status
}

/// Bounded HTTPS fetch through host curl. Returns the status code, the
/// status text, and the body bytes.
fn fetch_capped(
    phase: &Phase,
    url: &str,
    max_bytes: u64,
    extra: &[String],
) -> Result<(u16, String, Vec<u8>), Error> {
    if !https_url(url) || max_bytes == 0 {
        return Err(Error::msg("bounded HTTPS fetch required"));
    }
    phase
        .check()
        .map_err(|err| Error::msg(format!("live input fetch failed: {err}")))?;
    let scratch = TempDir::new("coreos-fetch")
        .map_err(|err| Error::msg(format!("live input fetch failed: {err}")))?;
    let body_path = scratch.join("body").to_string_lossy().into_owned();
    let headers_path = scratch.join("headers").to_string_lossy().into_owned();
    let meta_path = scratch.join("meta").to_string_lossy().into_owned();
    let errors_path = scratch.join("errors").to_string_lossy().into_owned();
    let spec = curl_spec(url, &body_path, &headers_path, extra, max_bytes);
    let meta_file = std::fs::File::create(&meta_path)
        .map_err(|err| Error::msg(format!("live input fetch failed: {err}")))?;
    let errors_file = std::fs::File::create(&errors_path)
        .map_err(|err| Error::msg(format!("live input fetch failed: {err}")))?;
    let out: SharedWriter = std::sync::Arc::new(std::sync::Mutex::new(
        crate::evidence::RedactingWriter::tee(meta_file, Vec::new()),
    ));
    let err_writer: SharedWriter = std::sync::Arc::new(std::sync::Mutex::new(
        crate::evidence::RedactingWriter::tee(errors_file, Vec::new()),
    ));
    let process = process::start_process(phase, &spec, out.clone(), err_writer.clone())
        .map_err(|err| Error::msg(format!("live input fetch failed: {err}")))?;
    let wait_err = process.wait(phase).err();
    let _ = process.join_pumps();
    let detail = String::from_utf8_lossy(
        err_writer
            .lock()
            .unwrap_or_else(|e| e.into_inner())
            .buffer(),
    )
    .into_owned();
    if wait_err.is_some() {
        if phase.is_cancelled() {
            return Err(Error::msg("live input fetch failed: context canceled"));
        }
        if phase.expired() {
            return Err(Error::msg(
                "live input fetch failed: context deadline exceeded",
            ));
        }
        if detail.contains("not supported or disabled") {
            return Err(Error::msg("unsafe metadata redirect"));
        }
        if detail.contains("exceeds maximum file size")
            || detail.contains("Maximum file size exceeded")
        {
            return Err(Error::msg("live input exceeds size limit"));
        }
        if detail.trim().is_empty() {
            let cause = wait_err.map(|e| e.to_string()).unwrap_or_default();
            return Err(Error::msg(format!("live input fetch failed: {cause}")));
        }
        return Err(Error::msg(format!(
            "live input fetch failed: {}",
            detail.trim()
        )));
    }
    let meta = out
        .lock()
        .unwrap_or_else(|e| e.into_inner())
        .buffer()
        .to_vec();
    let meta = String::from_utf8_lossy(&meta);
    let (code_text, effective) = meta.trim().split_once(' ').unwrap_or(("", ""));
    let code: u16 = code_text.parse().unwrap_or(0);
    if !effective.starts_with("https://") {
        return Err(Error::msg("unsafe metadata redirect"));
    }
    let headers = std::fs::read_to_string(&headers_path).unwrap_or_default();
    let body = std::fs::read(&body_path).unwrap_or_default();
    if body.len() as u64 > max_bytes {
        return Err(Error::msg("live input exceeds size limit"));
    }
    Ok((code, status_text(&headers), body))
}

/// Parse `/builds/<release>/` out of an ISO location, like the Go
/// owner's stream release pattern.
fn release_from_location(location: &str) -> Option<String> {
    let rest = location.split("/builds/").nth(1)?;
    let (release, _) = rest.split_once('/')?;
    let mut parts = release.split('.');
    match (
        parts.next(),
        parts.next(),
        parts.next(),
        parts.next(),
        parts.next(),
    ) {
        (Some(a), Some(b), Some(c), Some(d), None)
            if !a.is_empty()
                && !b.is_empty()
                && !c.is_empty()
                && !d.is_empty()
                && a.bytes().all(|b| b.is_ascii_digit())
                && b.bytes().all(|b| b.is_ascii_digit())
                && c.bytes().all(|b| b.is_ascii_digit())
                && d.bytes().all(|b| b.is_ascii_digit()) =>
        {
            Some(release.to_string())
        }
        _ => None,
    }
}

fn image_at(doc: &JsonValue, artifact: &str, format: &str) -> Option<CoreOSImage> {
    let entry = doc.get("architectures")?.get("x86_64")?;
    let disk = entry
        .get("artifacts")?
        .get(artifact)?
        .get("formats")?
        .get(format)?
        .get("disk")?;
    Some(CoreOSImage {
        url: disk.get("location")?.as_str()?.to_string(),
        signature_url: disk
            .get("signature")
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .to_string(),
        sha256: disk
            .get("sha256")
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .to_string(),
        uncompressed_sha256: disk
            .get("uncompressed-sha256")
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .to_string(),
    })
}

/// Parse one stable-stream document into the release and the x86_64
/// ISO/QEMU triples. Mirrors `resolveStreamBuild`.
pub fn resolve_stream_build(data: &[u8]) -> Result<(String, CoreOSImage, CoreOSImage), Error> {
    let text =
        std::str::from_utf8(data).map_err(|_| Error::msg("invalid CoreOS stream document"))?;
    let doc = JsonValue::parse(text).map_err(|_| Error::msg("invalid CoreOS stream document"))?;
    if doc
        .get("architectures")
        .and_then(|v| v.get("x86_64"))
        .is_none()
    {
        return Err(Error::msg("stable stream lacks architecture x86_64"));
    }
    let Some(iso) = image_at(&doc, "metal", "iso") else {
        return Err(Error::msg("stable stream lacks x86_64 live ISO"));
    };
    let Some(release) = release_from_location(&iso.url) else {
        return Err(Error::msg(
            "stable stream x86_64 ISO location names no release",
        ));
    };
    let Some(qemu) = image_at(&doc, "qemu", "qcow2.xz") else {
        return Err(Error::msg("stable stream lacks x86_64 qemu image"));
    };
    valid_stream_images(&release, &iso, &qemu).map_err(|err| Error::msg(err.to_string()))?;
    Ok((release, iso, qemu))
}

/// Registry manifest endpoint and host for one registry base URL.
pub fn registry_endpoint(registry: &str) -> Result<(String, String), Error> {
    if !https_url(registry) {
        return Err(Error::msg("container registry URL must be HTTPS"));
    }
    let authority = registry["https://".len()..].split('/').next().unwrap_or("");
    if authority.is_empty() || authority.contains('@') {
        return Err(Error::msg("container registry URL is malformed"));
    }
    Ok((
        format!("{registry}/v2/{CONTAINER_REPO}/manifests/{CONTAINER_TAG}"),
        authority.to_string(),
    ))
}

/// Read the container manifest index and return the x86_64 digest-pinned
/// ref. Mirrors `resolveRegistryDigests`.
pub fn resolve_registry_digest(phase: &Phase) -> Result<String, Error> {
    let registry = registry_url();
    let (endpoint, host) = registry_endpoint(&registry)?;
    let extra = vec![
        "-H".to_string(),
        "Accept: application/vnd.oci.image.index.v1+json".to_string(),
    ];
    let (code, status, body) = match fetch_capped(phase, &endpoint, 1 << 20, &extra) {
        Ok(ok) => ok,
        Err(err) if err.to_string() == "live input exceeds size limit" => {
            return Err(Error::msg("container index exceeds size limit"));
        }
        Err(err) if err.to_string() == "unsafe metadata redirect" => return Err(err),
        Err(_) => return Err(Error::msg("container registry fetch failed")),
    };
    if code == 401 {
        return Err(Error::msg(
            "container registry refused anonymous manifest access",
        ));
    }
    if code != 200 {
        let shown = if status.is_empty() {
            code.to_string()
        } else {
            status
        };
        return Err(Error::msg(format!(
            "container registry HTTP failure: {shown}"
        )));
    }
    let text =
        std::str::from_utf8(&body).map_err(|_| Error::msg("container index is malformed"))?;
    let index = JsonValue::parse(text).map_err(|_| Error::msg("container index is malformed"))?;
    // Missing/null `manifests` decodes to no entries, like Go: the
    // lookup below then reports the lacking architecture.
    let manifests = match index.get("manifests") {
        None | Some(JsonValue::Null) => &[],
        Some(JsonValue::Array(manifests)) => manifests.as_slice(),
        Some(_) => return Err(Error::msg("container index is malformed")),
    };
    let oci_arch = oci_architecture("x86_64").map_err(|err| Error::msg(err.to_string()))?;
    let mut found = String::new();
    for manifest in manifests {
        let JsonValue::Object(_) = manifest else {
            return Err(Error::msg("container index is malformed"));
        };
        if let Some(platform) = manifest.get("platform") {
            if !matches!(platform, JsonValue::Object(_) | JsonValue::Null) {
                return Err(Error::msg("container index is malformed"));
            }
        }
        if manifest
            .get("platform")
            .and_then(|p| p.get("architecture"))
            .and_then(|v| v.as_str())
            != Some(oci_arch)
        {
            continue;
        }
        let Some(digest) = manifest.get("digest").and_then(|v| v.as_str()) else {
            return Err(Error::msg("container index x86_64 digest is malformed"));
        };
        let Some(hex) = digest.strip_prefix("sha256:") else {
            return Err(Error::msg("container index x86_64 digest is malformed"));
        };
        if !is_digest(hex) {
            return Err(Error::msg("container index x86_64 digest is malformed"));
        }
        found = digest.to_string();
    }
    if found.is_empty() {
        return Err(Error::msg("container index lacks architecture x86_64"));
    }
    Ok(format!("{host}/{CONTAINER_REPO}@{found}"))
}

/// Resolve the current stable QEMU image: release plus the verified
/// download triple. Mirrors `ResolveCoreOSQEMU`.
pub fn resolve_qemu(phase: &Phase, arch: &str) -> Result<(String, CoreOSImage), Error> {
    oci_architecture(arch).map_err(|err| Error::msg(err.to_string()))?;
    let url = stream_url();
    if !https_url(&url) {
        return Err(Error::msg("CoreOS stream URL must be HTTPS"));
    }
    let (code, status, body) = fetch_capped(phase, &url, 8 << 20, &[])?;
    if code != 200 {
        let shown = if status.is_empty() {
            code.to_string()
        } else {
            status
        };
        return Err(Error::msg(format!("live input HTTP failure: {shown}")));
    }
    let (release, _, qemu) = resolve_stream_build(&body)?;
    let _ = resolve_registry_digest(phase)?;
    let _ = stream_release_url(&url, &release)?;
    Ok((release, qemu))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn stream_doc(iso_location: &str) -> String {
        r#"{"architectures": {"x86_64": {"artifacts": {
            "metal": {"formats": {"iso": {"disk": {
                "location": "ISO_LOCATION",
                "signature": "ISO_LOCATION.sig",
                "sha256": "SHA_A"
            }}}},
            "qemu": {"formats": {"qcow2.xz": {"disk": {
                "location": "https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/41.20250101.3.0/x86_64/fedora-coreos-41.20250101.3.0-qemu.x86_64.qcow2.xz",
                "signature": "https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/41.20250101.3.0/x86_64/fedora-coreos-41.20250101.3.0-qemu.x86_64.qcow2.xz.sig",
                "sha256": "SHA_B",
                "uncompressed-sha256": "SHA_C"
            }}}}
        }}}}"#
            .replace("ISO_LOCATION", iso_location)
            .replace("SHA_A", &"a".repeat(64))
            .replace("SHA_B", &"b".repeat(64))
            .replace("SHA_C", &"c".repeat(64))
    }

    #[test]
    fn stream_build_parses_release_and_triples() {
        let location = "https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/41.20250101.3.0/x86_64/fedora-coreos-41.20250101.3.0-live.x86_64.iso";
        let (release, iso, qemu) = resolve_stream_build(stream_doc(location).as_bytes()).unwrap();
        assert_eq!(release, "41.20250101.3.0");
        assert_eq!(iso.url, location);
        assert_eq!(iso.signature_url, format!("{location}.sig"));
        assert_eq!(qemu.uncompressed_sha256, "c".repeat(64));
        assert!(resolve_stream_build(b"not json").is_err());
        assert!(resolve_stream_build(br#"{"architectures": {}}"#).is_err());
        let err = resolve_stream_build(
            stream_doc("https://example.test/no-builds/segment.iso").as_bytes(),
        )
        .err()
        .unwrap();
        assert_eq!(
            err.to_string(),
            "stable stream x86_64 ISO location names no release"
        );
        let err = resolve_stream_build(br#"{"architectures": {"x86_64": {"artifacts": {}}}}"#)
            .err()
            .unwrap();
        assert_eq!(err.to_string(), "stable stream lacks x86_64 live ISO");
    }

    #[test]
    fn registry_endpoint_gates() {
        let (endpoint, host) = registry_endpoint("https://quay.io").unwrap();
        assert_eq!(
            endpoint,
            "https://quay.io/v2/fedora/fedora-coreos/manifests/stable"
        );
        assert_eq!(host, "quay.io");
        assert_eq!(
            registry_endpoint("http://quay.io").unwrap_err().to_string(),
            "container registry URL must be HTTPS"
        );
        assert_eq!(
            registry_endpoint("https://").unwrap_err().to_string(),
            "container registry URL must be HTTPS"
        );
    }

    #[test]
    fn status_text_prefers_last_response() {
        let headers =
            "HTTP/1.1 301 Moved Permanently\r\nLocation: x\r\n\r\nHTTP/1.1 404 Not Found\r\n\r\n";
        assert_eq!(status_text(headers), "404 Not Found");
        assert_eq!(status_text("HTTP/2 200\r\n"), "200");
        assert_eq!(status_text("garbage"), "");
    }

    #[test]
    fn fetch_gate_rejects_plain_http_without_network() {
        // No request is issued: the gate fails first.
        let err = fetch_capped(
            &Phase::background(),
            "http://example.test/streams/stable.json",
            8 << 20,
            &[],
        )
        .unwrap_err();
        assert_eq!(err.to_string(), "bounded HTTPS fetch required");
    }

    #[test]
    fn resolve_gate_rejects_plain_http_stream() {
        std::env::set_var(
            "SODA_COREOS_STREAM_URL",
            "http://example.test/streams/stable.json",
        );
        let err = resolve_qemu(&Phase::background(), "x86_64").err().unwrap();
        std::env::remove_var("SODA_COREOS_STREAM_URL");
        assert_eq!(err.to_string(), "CoreOS stream URL must be HTTPS");
        assert!(resolve_qemu(&Phase::background(), "aarch64").is_err());
    }
}

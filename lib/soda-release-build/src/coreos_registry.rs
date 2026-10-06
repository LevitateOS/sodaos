//! CoreOS container-registry resolution: image-index request,
//! bounded body decode and matching x86_64 digest selection.

use crate::coreos::https_url;
use crate::files::{is_digest, oci_architecture};
use crate::http::{get_follow, HttpTransport};
use crate::json_go::Fields;
use crate::Error;
use soda_json::JsonValue;
use std::collections::HashMap;
use std::io::Read;
use std::time::Duration;

const COREOS_CONTAINER_REPO: &str = "fedora/fedora-coreos";
const COREOS_CONTAINER_TAG: &str = "stable";

pub(crate) fn resolve_registry_digests_with<T: HttpTransport>(
    transport: &T,
    registry: &str,
) -> Result<HashMap<String, String>, Error> {
    if !https_url(registry) {
        return Err(Error::msg("container registry URL must be HTTPS"));
    }
    let host = registry
        .split_once("://")
        .map(|(_, rest)| rest.split('/').next().unwrap_or(""))
        .unwrap_or("");
    if host.is_empty() || host.contains('/') {
        return Err(Error::msg("container registry URL is malformed"));
    }
    let endpoint =
        format!("{registry}/v2/{COREOS_CONTAINER_REPO}/manifests/{COREOS_CONTAINER_TAG}");
    let response = get_follow(
        transport,
        &endpoint,
        Some("application/vnd.oci.image.index.v1+json"),
        Duration::from_secs(60),
        "unsafe metadata redirect",
    )
    .map_err(|_| Error::msg("container registry fetch failed"))?;
    if response.status == 401 {
        return Err(Error::msg(
            "container registry refused anonymous manifest access",
        ));
    }
    if response.status != 200 {
        return Err(Error::msg(format!(
            "container registry HTTP failure: {}",
            response.status_line()
        )));
    }
    let mut data = Vec::new();
    response
        .body
        .take((1 << 20) + 1)
        .read_to_end(&mut data)
        .map_err(|e| Error::msg(e.to_string()))?;
    if data.len() > 1 << 20 {
        return Err(Error::msg("container index exceeds size limit"));
    }
    let text = String::from_utf8_lossy(&data);
    let value = JsonValue::parse(&text).map_err(|_| Error::msg("invalid container index"))?;
    let index = Fields::of(&value).ok_or_else(|| Error::msg("invalid container index"))?;
    let manifests = index
        .object_list("manifests")
        .map_err(|_| Error::msg("invalid container index"))?;
    let oci_arch = oci_architecture("x86_64")?;
    let mut found = String::new();
    for manifest in &manifests {
        let platform = manifest
            .object("platform")
            .map_err(|_| Error::msg("invalid container index"))?;
        let architecture = match platform {
            Some(p) => p
                .string("architecture")
                .map_err(|_| Error::msg("invalid container index"))?,
            None => String::new(),
        };
        if architecture != oci_arch {
            continue;
        }
        let digest = manifest
            .string("digest")
            .map_err(|_| Error::msg("invalid container index"))?;
        match digest.strip_prefix("sha256:") {
            Some(hex) if is_digest(hex) => {}
            _ => return Err(Error::msg("container index x86_64 digest is malformed")),
        }
        found = digest;
    }
    if found.is_empty() {
        return Err(Error::msg("container index lacks architecture x86_64"));
    }
    let mut digests = HashMap::new();
    digests.insert(
        "x86_64".to_string(),
        format!("{host}/{COREOS_CONTAINER_REPO}@{found}"),
    );
    Ok(digests)
}

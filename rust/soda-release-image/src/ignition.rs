//! `assemble.go` + `candidate_live.go`: live-Ignition readback and handoff.

use std::io::Read;

use base64::Engine;
use flate2::read::GzDecoder;
use soda_json::JsonValue;

use crate::error::Error;
use crate::jsonio;
use crate::model;

/// VerifyLiveIgnition checks the native customization readback against the
/// exact fragment supplied by this build. Ignition's serializer emits absent
/// optionals as null.
pub fn verify_live_ignition(data: &[u8], expected: &[u8]) -> Result<(), Error> {
    let raw = live_ignition_bytes(data)?;
    let raw_text =
        std::str::from_utf8(&raw).map_err(|_| Error::msg("invalid Ignition readback"))?;
    let expected_text =
        std::str::from_utf8(expected).map_err(|_| Error::msg("invalid Ignition readback"))?;
    let mut got = jsonio::parse(raw_text).map_err(|_| Error::msg("invalid Ignition readback"))?;
    let mut want =
        jsonio::parse(expected_text).map_err(|_| Error::msg("invalid Ignition readback"))?;
    omit_null_fields(&mut got);
    omit_null_fields(&mut want);
    if got != want {
        return Err(Error::msg("embedded live Ignition differs"));
    }
    Ok(())
}

fn live_ignition_bytes(data: &[u8]) -> Result<Vec<u8>, Error> {
    let text =
        std::str::from_utf8(data).map_err(|_| Error::msg("unexpected native live Ignition"))?;
    let wrapper = jsonio::parse(text).map_err(|_| Error::msg("unexpected native live Ignition"))?;
    let merge = wrapper
        .get("ignition")
        .and_then(|ignition| ignition.get("config"))
        .and_then(|config| config.get("merge"));
    let JsonValue::Array(entries) = merge.unwrap_or(&JsonValue::Null) else {
        return Err(Error::msg("unexpected native live Ignition"));
    };
    if entries.len() != 1 {
        return Err(Error::msg("unexpected native live Ignition"));
    }
    let source = entries[0]
        .get("source")
        .and_then(|v| v.as_str())
        .unwrap_or("");
    let compression = entries[0]
        .get("compression")
        .and_then(|v| v.as_str())
        .unwrap_or("");
    let encoded = match source.strip_prefix("data:;base64,") {
        Some(encoded) if compression == "gzip" => encoded,
        _ => return Err(Error::msg("unexpected native Ignition encoding")),
    };
    let compressed = base64::engine::general_purpose::STANDARD
        .decode(encoded.as_bytes())
        .map_err(|e| Error::msg(e.to_string()))?;
    let decoder = GzDecoder::new(compressed.as_slice());
    let mut raw = Vec::new();
    decoder
        .take(2 << 20)
        .read_to_end(&mut raw)
        .map_err(|e| Error::msg(e.to_string()))?;
    Ok(raw)
}

fn omit_null_fields(value: &mut JsonValue) {
    match value {
        JsonValue::Object(entries) => {
            entries.retain(|(_, child)| !matches!(child, JsonValue::Null));
            for (_, child) in entries.iter_mut() {
                omit_null_fields(child);
            }
        }
        JsonValue::Array(items) => {
            for child in items.iter_mut() {
                omit_null_fields(child);
            }
        }
        _ => {}
    }
}

// ---------------------------------------------------------------------------
// candidate_live.go
// ---------------------------------------------------------------------------

/// candidateInstallerBinary is the installed live-console path the generated
/// console unit executes. The Rust port owns the binary; this package owns
/// only the media handoff that references it.
pub const CANDIDATE_INSTALLER_BINARY: &str = "/usr/libexec/soda/soda-install";

#[derive(Debug, Clone, Default)]
pub struct MediaIdentity {
    pub architecture: String,
    pub release: String,
    pub installer_version: String,
    pub revision: String,
    pub host_manifest: String,
    pub payload_sha256: String,
    pub console_sha256: String,
}

impl MediaIdentity {
    fn validate(&self, image_version: &str, arch: &str) -> Result<(), Error> {
        if self.architecture != arch
            || self.release.is_empty()
            || self.release != image_version
            || !model::is_revision(&self.revision)
            || !self.valid_content()
        {
            return Err(Error::msg(
                "media release, architecture, or included-payload mismatch",
            ));
        }
        Ok(())
    }

    fn valid_content(&self) -> bool {
        model::prefixed_digest(&self.host_manifest)
            && model::is_digest(&self.payload_sha256)
            && model::is_digest(&self.console_sha256)
    }

    fn to_json(&self) -> JsonValue {
        JsonValue::Object(vec![
            (
                "Architecture".to_string(),
                JsonValue::Str(self.architecture.clone()),
            ),
            ("Release".to_string(), JsonValue::Str(self.release.clone())),
            (
                "InstallerVersion".to_string(),
                JsonValue::Str(self.installer_version.clone()),
            ),
            (
                "Revision".to_string(),
                JsonValue::Str(self.revision.clone()),
            ),
            (
                "HostManifest".to_string(),
                JsonValue::Str(self.host_manifest.clone()),
            ),
            (
                "PayloadSHA256".to_string(),
                JsonValue::Str(self.payload_sha256.clone()),
            ),
            (
                "ConsoleSHA256".to_string(),
                JsonValue::Str(self.console_sha256.clone()),
            ),
        ])
    }
}

/// candidateLiveConfig is a media-only leaf. Its protected caller
/// authenticates the candidate and supplies the already-compiled console's
/// digest. That binary is in the same native, stream-verified rootfs: no
/// separate executable download, compilation, disk selection, private input
/// or reboot is performed here.
pub fn candidate_live_config(
    payload: &[u8],
    destination: &[u8],
    manifest: &str,
    console_sha256: &str,
) -> Result<Vec<u8>, Error> {
    let payload_text = std::str::from_utf8(payload)
        .map_err(|_| Error::msg("complete ordinary-Podman candidate required"))?;
    // Plain (non-strict) decode, like the Go owner.
    let payload_value = jsonio::parse(payload_text)
        .map_err(|_| Error::msg("complete ordinary-Podman candidate required"))?;
    let parsed = model::Payload::parse(&payload_value)
        .map_err(|_| Error::msg("complete ordinary-Podman candidate required"))?;
    if parsed.validate().is_err() {
        return Err(Error::msg("complete ordinary-Podman candidate required"));
    }
    let destination_text = std::str::from_utf8(destination)
        .map_err(|_| Error::msg("public converted destination template required"))?;
    let destination_value = jsonio::parse(destination_text)
        .map_err(|_| Error::msg("public converted destination template required"))?;
    let version = destination_value
        .get("ignition")
        .and_then(|ignition| ignition.get("version"))
        .and_then(|v| v.as_str())
        .unwrap_or("");
    if version != "3.5.0" || jsonio::raw_present(&destination_value, "passwd") {
        return Err(Error::msg("public converted destination template required"));
    }
    let sum = crate::sys::hex_sha256(payload);
    let identity = MediaIdentity {
        architecture: parsed.architecture.clone(),
        release: parsed.core_os.clone(),
        installer_version: "coreos-installer 0.26.0".to_string(),
        revision: parsed.revision.clone(),
        host_manifest: manifest.to_string(),
        payload_sha256: sum,
        console_sha256: console_sha256.to_string(),
    };
    if identity
        .validate(&parsed.core_os, &parsed.architecture)
        .is_err()
    {
        return Err(Error::msg("candidate media identity required"));
    }
    let media = jsonio::to_compact(&identity.to_json());
    let inline = |path: &str, contents: &[u8]| -> JsonValue {
        JsonValue::Object(vec![
            (
                "contents".to_string(),
                JsonValue::Object(vec![(
                    "source".to_string(),
                    JsonValue::Str(format!(
                        "data:;base64,{}",
                        base64::engine::general_purpose::STANDARD.encode(contents)
                    )),
                )]),
            ),
            ("mode".to_string(), JsonValue::Number("420".to_string())),
            ("path".to_string(), JsonValue::Str(path.to_string())),
        ])
    };
    // Ignition writes the native physical /var path, not through /usr/local.
    const LIVE_DATA: &str = "/var/usrlocal/share/soda-installer";
    let files = JsonValue::Array(vec![
        inline(&format!("{LIVE_DATA}/media.json"), media.as_bytes()),
        inline(&format!("{LIVE_DATA}/destination.ign"), destination),
    ]);
    // These masks apply only to live Ignition, never to destination Ignition.
    let mut units: Vec<JsonValue> = Vec::new();
    for name in [
        "getty@tty1.service",
        "forgejo.service",
        "soda-dashboard.service",
        "soda-proxy.service",
        "soda-host.service",
        "soda-host.socket",
        "soda-image-import.service",
    ] {
        units.push(JsonValue::Object(vec![
            ("mask".to_string(), JsonValue::Bool(true)),
            ("name".to_string(), JsonValue::Str(name.to_string())),
        ]));
    }
    let body = [
        "[Unit]",
        "Description=SodaOS installation console",
        "After=systemd-user-sessions.service NetworkManager.service",
        "Conflicts=getty@tty1.service",
        "[Service]",
        "Type=simple",
        "PrivateMounts=yes",
        &format!("ExecStart={CANDIDATE_INSTALLER_BINARY} disk"),
        "StandardInput=tty-force",
        "StandardOutput=tty",
        "StandardError=tty",
        "TTYPath=/dev/tty1",
        "TTYReset=yes",
        "TTYVHangup=yes",
        "Restart=no",
        "[Install]",
        "WantedBy=multi-user.target",
        "",
    ]
    .join("\n");
    units.push(JsonValue::Object(vec![
        ("contents".to_string(), JsonValue::Str(body)),
        ("enabled".to_string(), JsonValue::Bool(true)),
        (
            "name".to_string(),
            JsonValue::Str("soda-installer-console.service".to_string()),
        ),
    ]));
    let document = JsonValue::Object(vec![
        (
            "ignition".to_string(),
            JsonValue::Object(vec![(
                "version".to_string(),
                JsonValue::Str("3.5.0".to_string()),
            )]),
        ),
        (
            "storage".to_string(),
            JsonValue::Object(vec![("files".to_string(), files)]),
        ),
        (
            "systemd".to_string(),
            JsonValue::Object(vec![("units".to_string(), JsonValue::Array(units))]),
        ),
    ]);
    Ok(jsonio::to_compact(&document).into_bytes())
}

#[cfg(test)]
mod tests {
    use super::*;
    use flate2::write::GzEncoder;
    use flate2::Compression;
    use std::io::Write;

    fn gzip_bytes(data: &[u8]) -> Vec<u8> {
        let mut encoder = GzEncoder::new(Vec::new(), Compression::default());
        encoder.write_all(data).unwrap();
        encoder.finish().unwrap()
    }

    #[test]
    fn oracle_live_ignition_readback_ignores_nulls() {
        // Oracle: Go TestMediaReadbackBindsNativeIgnitionAndRootfs (ignition half).
        let fragment = br#"{"ignition":{"version":"3.5.0"},"storage":{}}"#;
        let gzipped = gzip_bytes(fragment);
        let wrapped = format!(
            "{{\"ignition\":{{\"config\":{{\"merge\":[{{\"source\":\"data:;base64,{}\",\"compression\":\"gzip\"}}]}}}}}}",
            base64::engine::general_purpose::STANDARD.encode(&gzipped)
        );
        let expected =
            br#"{"ignition":{"version":"3.5.0","config":null},"storage":{},"systemd":null}"#;
        assert!(verify_live_ignition(wrapped.as_bytes(), expected).is_ok());
        assert_eq!(
            verify_live_ignition(wrapped.as_bytes(), br#"{"ignition":{"version":"3.4.0"}}"#)
                .unwrap_err()
                .0,
            "embedded live Ignition differs"
        );
        assert!(verify_live_ignition(b"{}", fragment).is_err());
    }
}

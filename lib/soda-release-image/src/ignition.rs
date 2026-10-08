//! `assemble.go` + `candidate_live.go`: live-Ignition readback and handoff.

use std::io::Read;

use base64::Engine;
use flate2::read::MultiGzDecoder;
use serde::Serialize;

use crate::error::Error;
use crate::jsonio;
use crate::model;

const LIVE_IGNITION_LIMIT: u64 = 2 << 20;
const LIVE_IGNITION_COMPRESSED_LIMIT: usize = 10 << 20;

/// VerifyLiveIgnition checks the native customization readback against the
/// exact fragment supplied by this build. Ignition's serializer emits absent
/// optionals as null.
pub fn verify_live_ignition(data: &[u8], expected: &[u8]) -> Result<(), Error> {
    let raw = live_ignition_bytes(data)?;
    let raw_text =
        std::str::from_utf8(&raw).map_err(|_| Error::msg("invalid Ignition readback"))?;
    let expected_text =
        std::str::from_utf8(expected).map_err(|_| Error::msg("invalid Ignition readback"))?;
    let mut got: serde_json::Value =
        serde_json::from_str(raw_text).map_err(|_| Error::msg("invalid Ignition readback"))?;
    let mut want: serde_json::Value =
        serde_json::from_str(expected_text).map_err(|_| Error::msg("invalid Ignition readback"))?;
    prune_null_object_members(&mut got);
    prune_null_object_members(&mut want);
    if got != want {
        return Err(Error::msg("embedded live Ignition differs"));
    }
    Ok(())
}

fn prune_null_object_members(value: &mut serde_json::Value) {
    match value {
        serde_json::Value::Object(object) => {
            object.retain(|_, value| !value.is_null());
            for value in object.values_mut() {
                prune_null_object_members(value);
            }
        }
        serde_json::Value::Array(values) => {
            for value in values {
                prune_null_object_members(value);
            }
        }
        _ => {}
    }
}

fn live_ignition_bytes(data: &[u8]) -> Result<Vec<u8>, Error> {
    let text =
        std::str::from_utf8(data).map_err(|_| Error::msg("unexpected native live Ignition"))?;
    let wrapper: serde_json::Value =
        serde_json::from_str(text).map_err(|_| Error::msg("unexpected native live Ignition"))?;
    let merge = wrapper
        .get("ignition")
        .and_then(|ignition| ignition.get("config"))
        .and_then(|config| config.get("merge"));
    let Some(serde_json::Value::Array(entries)) = merge else {
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
        Some(encoded)
            if compression == "gzip"
                && encoded.len() <= ((LIVE_IGNITION_COMPRESSED_LIMIT + 2) / 3) * 4 =>
        {
            encoded
        }
        _ => return Err(Error::msg("unexpected native Ignition encoding")),
    };
    let compressed = base64::engine::general_purpose::STANDARD
        .decode(encoded.as_bytes())
        .map_err(|e| Error::msg(e.to_string()))?;
    if compressed.len() > LIVE_IGNITION_COMPRESSED_LIMIT {
        return Err(Error::msg("unexpected native live Ignition"));
    }
    let mut decoder = MultiGzDecoder::new(compressed.as_slice());
    let mut raw = Vec::new();
    decoder
        .by_ref()
        .take(LIVE_IGNITION_LIMIT + 1)
        .read_to_end(&mut raw)
        .map_err(|e| Error::msg(e.to_string()))?;
    if raw.len() as u64 > LIVE_IGNITION_LIMIT {
        return Err(Error::msg("unexpected native live Ignition"));
    }
    Ok(raw)
}

// ---------------------------------------------------------------------------
// candidate_live.go
// ---------------------------------------------------------------------------

/// candidateInstallerBinary is the installed live-console path the generated
/// console unit executes. The Rust port owns the binary; this package owns
/// only the media handoff that references it.
pub const CANDIDATE_INSTALLER_BINARY: &str = "/usr/libexec/soda/soda-install";

#[derive(Debug, Clone, Default, Serialize)]
pub struct MediaIdentity {
    #[serde(rename = "Architecture")]
    pub architecture: String,
    #[serde(rename = "Release")]
    pub release: String,
    #[serde(rename = "InstallerVersion")]
    pub installer_version: String,
    #[serde(rename = "Revision")]
    pub revision: String,
    #[serde(rename = "HostManifest")]
    pub host_manifest: String,
    #[serde(rename = "PayloadSHA256")]
    pub payload_sha256: String,
    #[serde(rename = "ConsoleSHA256")]
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
    let parsed = model::Payload::parse(payload_text)
        .map_err(|_| Error::msg("complete ordinary-Podman candidate required"))?;
    if parsed.validate().is_err() {
        return Err(Error::msg("complete ordinary-Podman candidate required"));
    }
    let destination_text = std::str::from_utf8(destination)
        .map_err(|_| Error::msg("public converted destination template required"))?;
    let destination_value: serde_json::Value = jsonio::parse(destination_text)
        .map_err(|_| Error::msg("public converted destination template required"))?;
    let version = destination_value
        .get("ignition")
        .and_then(|ignition| ignition.get("version"))
        .and_then(|v| v.as_str())
        .unwrap_or("");
    if version != "3.5.0" || destination_value.get("passwd").is_some() {
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
    let media = serde_json::to_string(&identity).expect("serialization to String cannot fail");
    #[derive(Serialize)]
    struct InlineContents {
        source: String,
    }
    #[derive(Serialize)]
    struct InlineFile {
        contents: InlineContents,
        mode: u16,
        path: String,
    }
    #[derive(Serialize)]
    struct MaskUnit {
        mask: bool,
        name: String,
    }
    #[derive(Serialize)]
    struct ConsoleUnit {
        contents: String,
        enabled: bool,
        name: String,
    }
    #[derive(Serialize)]
    #[serde(untagged)]
    enum Unit {
        Mask(MaskUnit),
        Console(ConsoleUnit),
    }
    #[derive(Serialize)]
    struct IgnitionVersion {
        version: &'static str,
    }
    #[derive(Serialize)]
    struct Storage {
        files: Vec<InlineFile>,
    }
    #[derive(Serialize)]
    struct Systemd {
        units: Vec<Unit>,
    }
    #[derive(Serialize)]
    struct IgnitionDocument {
        ignition: IgnitionVersion,
        storage: Storage,
        systemd: Systemd,
    }
    let inline = |path: &str, contents: &[u8]| InlineFile {
        contents: InlineContents {
            source: format!(
                "data:;base64,{}",
                base64::engine::general_purpose::STANDARD.encode(contents)
            ),
        },
        mode: 420,
        path: path.to_string(),
    };
    // Ignition writes the native physical /var path, not through /usr/local.
    const LIVE_DATA: &str = "/var/usrlocal/share/soda-installer";
    let files = vec![
        inline(&format!("{LIVE_DATA}/media.json"), media.as_bytes()),
        inline(&format!("{LIVE_DATA}/destination.ign"), destination),
    ];
    // These masks apply only to live Ignition, never to destination Ignition.
    let mut units: Vec<Unit> = Vec::new();
    for name in [
        "getty@tty1.service",
        "forgejo.service",
        "soda-dashboard.service",
        "soda-proxy.service",
        "soda-host.service",
        "soda-host.socket",
        "soda-image-import.service",
    ] {
        units.push(Unit::Mask(MaskUnit {
            mask: true,
            name: name.to_string(),
        }));
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
    units.push(Unit::Console(ConsoleUnit {
        contents: body,
        enabled: true,
        name: "soda-installer-console.service".to_string(),
    }));
    let document = IgnitionDocument {
        ignition: IgnitionVersion { version: "3.5.0" },
        storage: Storage { files },
        systemd: Systemd { units },
    };
    Ok(serde_json::to_string(&document)
        .expect("serialization to String cannot fail")
        .into_bytes())
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
        let mut bad_trailer = gzipped;
        *bad_trailer.last_mut().unwrap() ^= 1;
        let corrupt = format!(
            "{{\"ignition\":{{\"config\":{{\"merge\":[{{\"source\":\"data:;base64,{}\",\"compression\":\"gzip\"}}]}}}}}}",
            base64::engine::general_purpose::STANDARD.encode(&bad_trailer)
        );
        assert!(verify_live_ignition(corrupt.as_bytes(), expected).is_err());
        assert_eq!(
            verify_live_ignition(wrapped.as_bytes(), br#"{"ignition":{"version":"3.4.0"}}"#)
                .unwrap_err()
                .0,
            "embedded live Ignition differs"
        );
        assert!(verify_live_ignition(b"{}", fragment).is_err());
    }

    #[test]
    fn live_ignition_equality_uses_object_values_and_preserves_array_order() {
        fn wrapped(fragment: &[u8]) -> String {
            let compressed = gzip_bytes(fragment);
            format!(
                "{{\"ignition\":{{\"config\":{{\"merge\":[{{\"source\":\"data:;base64,{}\",\"compression\":\"gzip\"}}]}}}}}}",
                base64::engine::general_purpose::STANDARD.encode(compressed)
            )
        }
        let expected =
            br#"{"a":100.0,"storage":{"nullable":null},"array":[null,{"field":null,"value":1}]}"#;
        let same = wrapped(expected);
        assert!(verify_live_ignition(same.as_bytes(), expected).is_ok());

        let reordered = wrapped(
            br#"{"array":[null,{"value":1,"field":null}],"storage":{"nullable":null},"a":100.0}"#,
        );
        assert!(verify_live_ignition(reordered.as_bytes(), expected).is_ok());
        let different_array_order = wrapped(
            br#"{"a":100.0,"storage":{"nullable":null},"array":[{"field":null,"value":1},null]}"#,
        );
        assert!(verify_live_ignition(different_array_order.as_bytes(), expected).is_err());
        let different_number = wrapped(br#"{"a":101.0,"storage":{},"array":[null,{"value":1}]}"#);
        assert!(verify_live_ignition(different_number.as_bytes(), expected).is_err());

        let nested =
            |depth: usize| format!("{{\"opaque\":{}0{}}}", "[".repeat(depth), "]".repeat(depth));
        let within_value_depth = nested(126);
        let wrapped_within = wrapped(within_value_depth.as_bytes());
        assert!(
            verify_live_ignition(wrapped_within.as_bytes(), within_value_depth.as_bytes()).is_ok()
        );
        let beyond_value_depth = nested(127);
        let wrapped_beyond = wrapped(beyond_value_depth.as_bytes());
        assert_eq!(
            verify_live_ignition(wrapped_beyond.as_bytes(), beyond_value_depth.as_bytes())
                .unwrap_err()
                .0,
            "invalid Ignition readback"
        );
    }
}

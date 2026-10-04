//! `media_compression.go`: development fast-rootfs admission + readback.

use std::fs;

use soda_json::JsonValue;

use crate::error::Error;
use crate::jsonio;
use crate::sys;

pub const DEFAULT_ROOTFS_OPTIONS: &str = "-zlzma,level=6 -Efragments -C1048576 --quiet";
pub const FAST_ROOTFS_OPTIONS: &str = "-zlzma,level=1 -Efragments -C1048576 --quiet";
pub const IMAGE_CONFIG_PATH: &str = "usr/share/coreos-assembler/image.json";

/// The caller has admitted the development/media request. Only this metadata
/// field changes; upstream Assembler reads it from the resulting distinct
/// host candidate.
pub fn set_media_compression(config: &mut JsonValue, mode: &str) -> Result<(), Error> {
    if mode.is_empty() {
        return Ok(()); // production and ordinary development preserve upstream defaults
    }
    let fs = config.get("live-rootfs-fstype").and_then(|v| v.as_str());
    let options = config.get("live-rootfs-fsoptions").and_then(|v| v.as_str());
    if mode != "fast" || fs != Some("erofs") || options != Some(DEFAULT_ROOTFS_OPTIONS) {
        return Err(Error::msg(
            "fast media requires the reviewed upstream EROFS/LZMA defaults",
        ));
    }
    if let JsonValue::Object(entries) = config {
        for (key, value) in entries.iter_mut() {
            if key == "live-rootfs-fsoptions" {
                *value = JsonValue::Str(FAST_ROOTFS_OPTIONS.to_string());
            }
        }
    }
    Ok(())
}

pub fn record_image_config(context: &str, out: &str, observed: &str) -> Result<(), Error> {
    let expected = fs::read(sys::join(&[context, "rootfs", IMAGE_CONFIG_PATH]))?;
    if String::from_utf8_lossy(&expected).trim() != observed {
        return Err(Error::msg(
            "host image configuration differs from admitted metadata",
        ));
    }
    sys::write_new(&sys::join(&[out, "image-config.json"]), &expected, 0o644)
}

pub fn rootfs_settings(out: &str) -> Result<(String, String), Error> {
    let data = fs::read(sys::join(&[out, "image-config.json"]))?;
    let text = std::str::from_utf8(&data).map_err(|e| Error::msg(e.to_string()))?;
    // Plain (non-strict) decode, like the Go owner.
    let value = jsonio::parse(text).map_err(|e| Error::msg(e.to_string()))?;
    Ok((
        jsonio::require_string(&value, "live-rootfs-fstype")?,
        jsonio::require_string(&value, "live-rootfs-fsoptions")?,
    ))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_media_compression_admission_and_metadata() {
        // Oracle: Go TestMediaCompressionAdmissionAndMetadata.
        let mut config = jsonio::parse(&format!(
            "{{\"live-rootfs-fstype\":\"erofs\",\"live-rootfs-fsoptions\":{}}}",
            jsonio::to_compact(&JsonValue::Str(DEFAULT_ROOTFS_OPTIONS.to_string()))
        ))
        .unwrap();
        set_media_compression(&mut config, "").unwrap();
        assert_eq!(
            config.get("live-rootfs-fsoptions").and_then(|v| v.as_str()),
            Some(DEFAULT_ROOTFS_OPTIONS)
        );
        set_media_compression(&mut config, "fast").unwrap();
        assert_eq!(
            config.get("live-rootfs-fsoptions").and_then(|v| v.as_str()),
            Some(FAST_ROOTFS_OPTIONS)
        );
        let mut config = jsonio::parse("{\"live-rootfs-fstype\":\"xfs\"}").unwrap();
        assert_eq!(
            set_media_compression(&mut config, "fast").unwrap_err().0,
            "fast media requires the reviewed upstream EROFS/LZMA defaults"
        );
        let mut config = jsonio::parse(&format!(
            "{{\"live-rootfs-fstype\":\"erofs\",\"live-rootfs-fsoptions\":{}}}",
            jsonio::to_compact(&JsonValue::Str(DEFAULT_ROOTFS_OPTIONS.to_string()))
        ))
        .unwrap();
        assert!(set_media_compression(&mut config, "turbo").is_err());
    }
}

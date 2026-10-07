//! `media_compression.go`: development fast-rootfs admission + readback.

use std::fs;

use crate::error::Error;
use crate::jsonio;
use crate::ordered_json::OrderedValue;
use crate::sys;

pub const DEFAULT_ROOTFS_OPTIONS: &str = "-zlzma,level=6 -Efragments -C1048576 --quiet";
pub const FAST_ROOTFS_OPTIONS: &str = "-zlzma,level=1 -Efragments -C1048576 --quiet";
pub const IMAGE_CONFIG_PATH: &str = "usr/share/coreos-assembler/image.json";

/// The assembler-owned configuration that this crate edits structurally.
/// Keeping this wrapper opaque prevents callers from losing duplicate members
/// and raw number spellings through `serde_json::Value`.
pub struct ImageConfig(OrderedValue);

impl ImageConfig {
    pub fn parse(text: &str) -> Result<Self, Error> {
        Ok(Self(OrderedValue::parse(text)?))
    }

    pub fn to_compact_json(&self) -> String {
        jsonio::to_compact(&self.0)
    }

    pub(crate) fn to_pretty_json(&self) -> String {
        jsonio::to_indent(&self.0)
    }

    pub(crate) fn is_nonempty_object(&self) -> bool {
        matches!(&self.0, OrderedValue::Object(entries) if !entries.is_empty())
    }

    pub fn settings(&self) -> Result<(String, String), Error> {
        Ok((
            self.0.string_or_empty("live-rootfs-fstype")?,
            self.0.string_or_empty("live-rootfs-fsoptions")?,
        ))
    }

    pub(crate) fn ordered_mut(&mut self) -> &mut OrderedValue {
        &mut self.0
    }
}

/// The caller has admitted the development/media request. Only this metadata
/// field changes; upstream Assembler reads it from the resulting distinct
/// host candidate.
pub fn set_media_compression(config: &mut ImageConfig, mode: &str) -> Result<(), Error> {
    if mode.is_empty() {
        return Ok(()); // production and ordinary development preserve upstream defaults
    }
    let fs = config.0.last_exact("live-rootfs-fstype");
    let options = config.0.last_exact("live-rootfs-fsoptions");
    if mode != "fast"
        || !matches!(fs, Some(OrderedValue::String(value)) if value == "erofs")
        || !matches!(options, Some(OrderedValue::String(value)) if value == DEFAULT_ROOTFS_OPTIONS)
    {
        return Err(Error::msg(
            "fast media requires the reviewed upstream EROFS/LZMA defaults",
        ));
    }
    config.0.set_all_exact(
        "live-rootfs-fsoptions",
        OrderedValue::String(FAST_ROOTFS_OPTIONS.to_string()),
    );
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
    ImageConfig::parse(text)?.settings()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_media_compression_admission_and_metadata() {
        // Oracle: Go TestMediaCompressionAdmissionAndMetadata.
        let mut config = ImageConfig::parse(&format!(
            "{{\"live-rootfs-fstype\":\"erofs\",\"live-rootfs-fsoptions\":{}}}",
            jsonio::to_compact(&serde_json::Value::String(
                DEFAULT_ROOTFS_OPTIONS.to_string()
            ))
        ))
        .unwrap();
        set_media_compression(&mut config, "").unwrap();
        assert_eq!(config.settings().unwrap().1, DEFAULT_ROOTFS_OPTIONS);
        set_media_compression(&mut config, "fast").unwrap();
        assert_eq!(config.settings().unwrap().1, FAST_ROOTFS_OPTIONS);
        let mut config = ImageConfig::parse("{\"live-rootfs-fstype\":\"xfs\"}").unwrap();
        assert_eq!(
            set_media_compression(&mut config, "fast").unwrap_err().0,
            "fast media requires the reviewed upstream EROFS/LZMA defaults"
        );
        let mut config = ImageConfig::parse(&format!(
            "{{\"live-rootfs-fstype\":\"erofs\",\"live-rootfs-fsoptions\":{}}}",
            jsonio::to_compact(&serde_json::Value::String(
                DEFAULT_ROOTFS_OPTIONS.to_string()
            ))
        ))
        .unwrap();
        assert!(set_media_compression(&mut config, "turbo").is_err());
    }

    #[test]
    fn settings_select_exact_then_first_folded_and_fast_updates_every_exact_pair() {
        let config = ImageConfig::parse(
            r#"{"LIVE-rootfs-fstype":"folded-first","live-rootfs-fstype":"erofs","live-rootfs-fstype":"xfs","Live-rootfs-fsoptions":"folded-options","live-rootfs-fsoptions":null,"live-rootfs-fsoptions":"later"}"#,
        )
        .unwrap();
        assert_eq!(config.settings().unwrap(), ("erofs".into(), String::new()));

        let config = ImageConfig::parse(
            r#"{"LIVE-ROOTFS-FSTYPE":"xfs","Live-rootfs-fstype":"erofs","LIVE-ROOTFS-FSOPTIONS":"first","Live-rootfs-fsoptions":"second"}"#,
        )
        .unwrap();
        assert_eq!(config.settings().unwrap(), ("xfs".into(), "first".into()));

        let mut config = ImageConfig::parse(&format!(
            r#"{{"live-rootfs-fstype":"erofs","live-rootfs-fsoptions":"{DEFAULT_ROOTFS_OPTIONS}","LIVE-rootfs-fsoptions":"keep","live-rootfs-fsoptions":"{DEFAULT_ROOTFS_OPTIONS}"}}"#
        ))
        .unwrap();
        set_media_compression(&mut config, "fast").unwrap();
        assert_eq!(
            config.to_compact_json(),
            format!(
                r#"{{"live-rootfs-fstype":"erofs","live-rootfs-fsoptions":"{FAST_ROOTFS_OPTIONS}","LIVE-rootfs-fsoptions":"keep","live-rootfs-fsoptions":"{FAST_ROOTFS_OPTIONS}"}}"#
            )
        );
    }
}

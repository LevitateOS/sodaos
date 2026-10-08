//! `media_compression.go`: development fast-rootfs admission + readback.

use serde_json::value::{to_raw_value, RawValue};
use std::collections::BTreeMap;
use std::fs;

use crate::error::Error;
use crate::sys;

pub const DEFAULT_ROOTFS_OPTIONS: &str = "-zlzma,level=6 -Efragments -C1048576 --quiet";
pub const FAST_ROOTFS_OPTIONS: &str = "-zlzma,level=1 -Efragments -C1048576 --quiet";
pub const IMAGE_CONFIG_PATH: &str = "usr/share/coreos-assembler/image.json";

fn parse_image_config(text: &str) -> Result<BTreeMap<String, Box<RawValue>>, Error> {
    serde_json::from_str(text).map_err(|_| Error::msg("invalid JSON"))
}

fn selected_string(config: &BTreeMap<String, Box<RawValue>>, name: &str) -> Result<String, Error> {
    let value = config.get(name).or_else(|| {
        config
            .iter()
            .find(|(key, _)| key.eq_ignore_ascii_case(name))
            .map(|(_, value)| value)
    });
    match value {
        None => Ok(String::new()),
        Some(value) => serde_json::from_str::<Option<String>>(value.get())
            .map(Option::unwrap_or_default)
            .map_err(|_| Error::msg(format!("missing or invalid {name}"))),
    }
}

fn exact_string(config: &BTreeMap<String, Box<RawValue>>, name: &str) -> Option<String> {
    serde_json::from_str::<String>(config.get(name)?.get()).ok()
}

fn settings(config: &BTreeMap<String, Box<RawValue>>) -> Result<(String, String), Error> {
    Ok((
        selected_string(config, "live-rootfs-fstype")?,
        selected_string(config, "live-rootfs-fsoptions")?,
    ))
}

/// The caller has admitted the development/media request. Only this metadata
/// field changes; upstream Assembler reads it from the resulting distinct
/// host candidate.
pub fn set_media_compression(
    config: &mut BTreeMap<String, Box<RawValue>>,
    mode: &str,
) -> Result<(), Error> {
    if mode.is_empty() {
        return Ok(()); // production and ordinary development preserve upstream defaults
    }
    if mode != "fast"
        || exact_string(config, "live-rootfs-fstype").as_deref() != Some("erofs")
        || exact_string(config, "live-rootfs-fsoptions").as_deref() != Some(DEFAULT_ROOTFS_OPTIONS)
    {
        return Err(Error::msg(
            "fast media requires the reviewed upstream EROFS/LZMA defaults",
        ));
    }
    config.insert(
        "live-rootfs-fsoptions".to_string(),
        to_raw_value(FAST_ROOTFS_OPTIONS).expect("serializing a string cannot fail"),
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
    settings(&parse_image_config(text)?)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn parsed_config(text: &str) -> BTreeMap<String, Box<RawValue>> {
        parse_image_config(text).unwrap()
    }

    #[test]
    fn oracle_media_compression_admission_and_metadata() {
        let mut config = parsed_config(&format!(
            "{{\"live-rootfs-fstype\":\"erofs\",\"live-rootfs-fsoptions\":{}}}",
            serde_json::to_string(DEFAULT_ROOTFS_OPTIONS).unwrap()
        ));
        set_media_compression(&mut config, "").unwrap();
        assert_eq!(settings(&config).unwrap().1, DEFAULT_ROOTFS_OPTIONS);
        set_media_compression(&mut config, "fast").unwrap();
        assert_eq!(settings(&config).unwrap().1, FAST_ROOTFS_OPTIONS);

        let mut config = parsed_config(r#"{"live-rootfs-fstype":"xfs"}"#);
        assert_eq!(
            set_media_compression(&mut config, "fast").unwrap_err().0,
            "fast media requires the reviewed upstream EROFS/LZMA defaults"
        );
        let mut config = parsed_config(&format!(
            "{{\"live-rootfs-fstype\":\"erofs\",\"live-rootfs-fsoptions\":{}}}",
            serde_json::to_string(DEFAULT_ROOTFS_OPTIONS).unwrap()
        ));
        assert!(set_media_compression(&mut config, "turbo").is_err());
    }

    #[test]
    fn settings_select_exact_then_first_folded_and_fast_updates_the_exact_pair() {
        let config = parsed_config(
            r#"{"LIVE-rootfs-fstype":"folded-first","live-rootfs-fstype":"erofs","live-rootfs-fstype":"xfs","Live-rootfs-fsoptions":"folded-options","live-rootfs-fsoptions":null,"live-rootfs-fsoptions":"later"}"#,
        );
        assert_eq!(settings(&config).unwrap(), ("xfs".into(), "later".into()));

        let config = parsed_config(
            r#"{"LIVE-ROOTFS-FSTYPE":"xfs","Live-rootfs-fstype":"erofs","LIVE-ROOTFS-FSOPTIONS":"first","Live-rootfs-fsoptions":"second"}"#,
        );
        assert_eq!(settings(&config).unwrap(), ("xfs".into(), "first".into()));

        let mut config = parsed_config(&format!(
            r#"{{"live-rootfs-fstype":"erofs","live-rootfs-fsoptions":"{DEFAULT_ROOTFS_OPTIONS}","LIVE-rootfs-fsoptions":"keep"}}"#
        ));
        set_media_compression(&mut config, "fast").unwrap();
        assert_eq!(config.len(), 3);
        assert_eq!(
            config["live-rootfs-fsoptions"].get(),
            serde_json::to_string(FAST_ROOTFS_OPTIONS).unwrap()
        );
        assert_eq!(config["LIVE-rootfs-fsoptions"].get(), r#""keep""#);
        assert_eq!(config["live-rootfs-fstype"].get(), r#""erofs""#);
    }
}

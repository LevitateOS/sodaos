//! Admit upstream media settings and select the native release profile.

use serde_json::value::{to_raw_value, RawValue};
use std::collections::BTreeMap;
use std::fs;

use crate::error::Error;
use crate::sys;

// Expected upstream input. The release freezes one profile: level-1 LZMA
// with fragments and a 1 MiB cluster, without global deduplication or
// compression hints. There is no fast-vs-release split and no hints file.
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
/// host candidate. Both admitted modes select the one frozen release profile;
/// the fast-vs-release split is removed, not retained as a hidden mode.
pub fn set_media_compression(
    config: &mut BTreeMap<String, Box<RawValue>>,
    mode: &str,
) -> Result<(), Error> {
    if !matches!(mode, "" | "fast")
        || exact_string(config, "live-rootfs-fstype").as_deref() != Some("erofs")
        || exact_string(config, "live-rootfs-fsoptions").as_deref() != Some(DEFAULT_ROOTFS_OPTIONS)
    {
        return Err(Error::msg(
            "media requires the reviewed upstream EROFS/LZMA defaults",
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

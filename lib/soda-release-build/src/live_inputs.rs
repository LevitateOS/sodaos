//! Admitted live-input records and validation: TailnetInputs,
//! ResolvedCoreOS and LiveInputs emit/decode, exact write/read and
//! admitted-input validation with the round-trip/refusal case.

use crate::coreos::{https_url, CoreOSImage};
use crate::files::{is_digest, write_new};
use crate::json_emit::{marshal_indent, Emit};
use crate::json_go::Strict;
use crate::json_input::read_json;
use crate::Error;
use std::collections::HashMap;
use std::path::Path;

/// Floating Tailnet toolchain for one attempt.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct TailnetInputs {
    pub version: String,
    pub sha256: String,
    pub base: String,
}

impl TailnetInputs {
    pub fn emit(&self) -> Emit {
        Emit::Object(vec![
            ("Version".to_string(), Emit::Str(self.version.clone())),
            ("SHA256".to_string(), Emit::Str(self.sha256.clone())),
            ("Base".to_string(), Emit::Str(self.base.clone())),
        ])
    }

    pub fn decode(binder: &mut Strict<'_>) -> Result<TailnetInputs, String> {
        Ok(TailnetInputs {
            version: binder.string("Version")?,
            sha256: binder.string("SHA256")?,
            base: binder.string("Base")?,
        })
    }
}

/// One stable build as found right now.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ResolvedCoreOS {
    pub release: String,
    pub metadata_url: String,
    pub container: HashMap<String, String>,
    pub iso: HashMap<String, CoreOSImage>,
    pub qemu: HashMap<String, CoreOSImage>,
}

impl ResolvedCoreOS {
    pub fn emit(&self) -> Emit {
        let mut container: Vec<(String, Emit)> = self
            .container
            .iter()
            .map(|(k, v)| (k.clone(), Emit::Str(v.clone())))
            .collect();
        container.sort_by(|a, b| a.0.cmp(&b.0));
        let mut iso: Vec<(String, Emit)> = self
            .iso
            .iter()
            .map(|(k, v)| (k.clone(), v.emit()))
            .collect();
        iso.sort_by(|a, b| a.0.cmp(&b.0));
        let mut qemu: Vec<(String, Emit)> = self
            .qemu
            .iter()
            .map(|(k, v)| (k.clone(), v.emit()))
            .collect();
        qemu.sort_by(|a, b| a.0.cmp(&b.0));
        Emit::Object(vec![
            ("Release".to_string(), Emit::Str(self.release.clone())),
            (
                "MetadataURL".to_string(),
                Emit::Str(self.metadata_url.clone()),
            ),
            ("Container".to_string(), Emit::Object(container)),
            ("ISO".to_string(), Emit::Object(iso)),
            ("QEMU".to_string(), Emit::Object(qemu)),
        ])
    }

    pub fn decode(binder: &mut Strict<'_>) -> Result<ResolvedCoreOS, String> {
        let container: Vec<(String, String)> = binder.string_map("Container")?;
        let iso = binder.object_map("ISO", "build.CoreOSImage", decode_coreos_image)?;
        let qemu = binder.object_map("QEMU", "build.CoreOSImage", decode_coreos_image)?;
        Ok(ResolvedCoreOS {
            release: binder.string("Release")?,
            metadata_url: binder.string("MetadataURL")?,
            container: container.into_iter().collect(),
            iso: iso.into_iter().collect(),
            qemu: qemu.into_iter().collect(),
        })
    }
}

fn decode_coreos_image(binder: &mut Strict<'_>) -> Result<CoreOSImage, String> {
    Ok(CoreOSImage {
        url: binder.string("URL")?,
        signature_url: binder.string("SignatureURL")?,
        sha256: binder.string("SHA256")?,
        uncompressed_sha256: binder.string("UncompressedSHA256")?,
    })
}

/// Controller-resolved live inputs for one isolated worker attempt.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct LiveInputs {
    pub coreos: ResolvedCoreOS,
    pub tailnet: TailnetInputs,
}

impl LiveInputs {
    pub fn marshal(&self) -> String {
        marshal_indent(&Emit::Object(vec![
            ("CoreOS".to_string(), self.coreos.emit()),
            ("Tailnet".to_string(), self.tailnet.emit()),
        ]))
    }

    pub fn decode(binder: &mut Strict<'_>) -> Result<LiveInputs, String> {
        let coreos = binder.nested("CoreOS", "build.ResolvedCoreOS", ResolvedCoreOS::decode)?;
        let tailnet = binder.nested("Tailnet", "build.TailnetInputs", TailnetInputs::decode)?;
        Ok(LiveInputs { coreos, tailnet })
    }
}

/// Records one attempt's live inputs. Validation mirrors the live path.
pub fn write_live_inputs(path: &Path, inputs: &LiveInputs) -> Result<(), Error> {
    valid_live_inputs(inputs)?;
    write_new(path, (inputs.marshal() + "\n").as_bytes(), 0o644)
}

/// Admits controller-resolved inputs for the isolated worker.
pub fn read_live_inputs(path: &Path) -> Result<LiveInputs, Error> {
    let inputs: LiveInputs = read_json(path, "build.LiveInputs", LiveInputs::decode)?;
    valid_live_inputs(&inputs)?;
    Ok(inputs)
}

/// Live resolution shape rules over admitted inputs.
pub fn valid_live_inputs(inputs: &LiveInputs) -> Result<(), Error> {
    valid_resolved_coreos(&inputs.coreos)?;
    valid_tailnet_inputs(&inputs.tailnet)
}

pub fn valid_tailnet_inputs(tailnet: &TailnetInputs) -> Result<(), Error> {
    let reader = soda_build_tools::reader::stream::TailnetInputs {
        version: tailnet.version.clone(),
        sha256: tailnet.sha256.clone(),
        base: tailnet.base.clone(),
    };
    soda_build_tools::reader::stream::valid_tailnet_inputs(&reader)?;
    Ok(())
}

pub fn valid_resolved_coreos(resolved: &ResolvedCoreOS) -> Result<(), Error> {
    let iso = match resolved.iso.get("x86_64") {
        Some(img) => img,
        None => return Err(Error::msg("stable stream x86_64 live ISO is malformed")),
    };
    let qemu = match resolved.qemu.get("x86_64") {
        Some(img) => img,
        None => return Err(Error::msg("stable stream x86_64 qemu image is malformed")),
    };
    let reader_iso = soda_build_tools::reader::stream::CoreOSImage {
        url: iso.url.clone(),
        signature_url: iso.signature_url.clone(),
        sha256: iso.sha256.clone(),
        uncompressed_sha256: iso.uncompressed_sha256.clone(),
    };
    let reader_qemu = soda_build_tools::reader::stream::CoreOSImage {
        url: qemu.url.clone(),
        signature_url: qemu.signature_url.clone(),
        sha256: qemu.sha256.clone(),
        uncompressed_sha256: qemu.uncompressed_sha256.clone(),
    };
    soda_build_tools::reader::stream::valid_stream_images(
        &resolved.release,
        &reader_iso,
        &reader_qemu,
    )?;
    // The shared reader takes single triples; container + metadata rules
    // apply here over the admitted maps.
    if !https_url(&resolved.metadata_url)
        || !resolved
            .metadata_url
            .ends_with(&format!("/builds/{}/release.json", resolved.release))
    {
        return Err(Error::msg("resolved CoreOS metadata URL is malformed"));
    }
    if resolved.container.len() != 1 {
        return Err(Error::msg("x86_64 base digest required"));
    }
    let entry = resolved
        .container
        .get("x86_64")
        .map(String::as_str)
        .unwrap_or("");
    match entry.split_once("/fedora/fedora-coreos@sha256:") {
        Some((host, digest)) if !host.is_empty() && !host.contains('/') && is_digest(digest) => {
            Ok(())
        }
        _ => Err(Error::msg("digest-pinned CoreOS base required")),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::coreos_stream::tests::fixture_live_inputs;

    #[test]
    fn oracle_live_inputs_round_trip() {
        // Oracle: WriteLiveInputs/ReadLiveInputs keep the resolved release.
        let dir = std::env::temp_dir().join(format!(
            "soda-live-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        std::fs::create_dir_all(&dir).unwrap();
        let path = dir.join("live-inputs.json");
        let inputs = fixture_live_inputs();
        write_live_inputs(&path, &inputs).unwrap();
        let back = read_live_inputs(&path).unwrap();
        assert_eq!(back.coreos.release, "44.20260901.1.0");
        assert_eq!(back.tailnet.version, "1.98.2");
        // Tampered digests and versions are refused on read.
        let mut bad = inputs.clone();
        bad.tailnet.version = "yesterday".to_string();
        assert!(write_live_inputs(&dir.join("bad.json"), &bad).is_err());
        let mut bad = inputs.clone();
        bad.coreos.release = "tomorrow".to_string();
        assert!(write_live_inputs(&dir.join("bad2.json"), &bad).is_err());
    }
}

//! Admit a locally produced Canlang tool archive; startup never fetches packages.
use flate2::read::GzDecoder;
use serde::Deserialize;
use std::collections::HashSet;
use std::fs::File;
use std::io::{Seek, SeekFrom};
use std::path::{Component, Path};

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct Release {
    pub source_revision: String,
    pub archive_filename: String,
    pub sha256: String,
    pub versions: Versions,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Versions {
    pub compiler: String,
    pub node: String,
    pub bun: String,
    pub miniflare: String,
    pub workerd: String,
}

pub fn stage(manifest: &Path, archive: &Path, out: &Path) -> Result<(), String> {
    let release: Release =
        serde_json::from_reader(File::open(manifest).map_err(|e| e.to_string())?)
            .map_err(|e| format!("invalid Canlang release: {e}"))?;
    if release.source_revision.len() != 40
        || !release
            .source_revision
            .bytes()
            .all(|b| b.is_ascii_hexdigit())
        || release.sha256.len() != 64
        || !release.sha256.bytes().all(|b| b.is_ascii_hexdigit())
        || Path::new(&release.archive_filename).components().count() != 1
        || archive.file_name().and_then(|s| s.to_str()) != Some(&release.archive_filename)
        || !release.versions.compiler.starts_with("can 0.1.0 ")
        || release.versions.node != "v24.21.0"
        || release.versions.bun != "1.4.2"
        || release.versions.miniflare != "4.20260730.0"
        || release.versions.workerd != "1.20260730.1"
    {
        return Err("invalid Canlang release identity".into());
    }
    let mut input = File::open(archive).map_err(|e| format!("Canlang archive unavailable: {e}"))?;
    if input.metadata().map_err(|e| e.to_string())?.len() > 1_000_000_000 {
        return Err("Canlang archive exceeds size limit".into());
    }
    if super::sha256_hex_stream(&mut input)? != release.sha256 {
        return Err("Canlang archive checksum mismatch".into());
    }
    std::fs::create_dir_all(out).map_err(|e| e.to_string())?;
    let destination = out.join("canlang");
    if destination.symlink_metadata().is_ok() {
        return Err("Canlang destination is occupied".into());
    }
    let staging = tempfile::Builder::new()
        .prefix(".canlang-stage-")
        .tempdir_in(out)
        .map_err(|e| e.to_string())?;
    input.seek(SeekFrom::Start(0)).map_err(|e| e.to_string())?;
    let mut tar = tar::Archive::new(GzDecoder::new(input));
    let mut seen = HashSet::new();
    let mut bytes = 0u64;
    for entry in tar.entries().map_err(|e| e.to_string())? {
        let mut entry = entry.map_err(|e| e.to_string())?;
        let path = entry.path().map_err(|e| e.to_string())?.into_owned();
        if path
            .components()
            .any(|c| !matches!(c, Component::Normal(_)))
            || !path.starts_with("canlang")
            || !seen.insert(path.clone())
            || seen.len() > 100_000
        {
            return Err("invalid Canlang archive path".into());
        }
        let kind = entry.header().entry_type();
        if !kind.is_dir() && !kind.is_file() {
            return Err("Canlang archive links and special files are forbidden".into());
        }
        bytes = bytes
            .checked_add(entry.size())
            .ok_or("Canlang archive size overflow")?;
        if bytes > 2_000_000_000 {
            return Err("Canlang expanded archive exceeds size limit".into());
        }
        let target = staging.path().join(&path);
        if let Some(parent) = target.parent() {
            std::fs::create_dir_all(parent).map_err(|e| e.to_string())?;
        }
        entry.unpack(&target).map_err(|e| e.to_string())?;
        let mode =
            if kind.is_dir() || entry.header().mode().map_err(|e| e.to_string())? & 0o111 != 0 {
                0o755
            } else {
                0o644
            };
        super::chmod(&target, mode)?;
    }
    let root = staging.path().join("canlang");
    for file in [
        "bin/can",
        "bin/can-platform",
        "bin/can-dev",
        "bin/can-soda",
        "libexec/node",
        "libexec/bun",
        "tools.json",
    ] {
        if !root.join(file).is_file() {
            return Err(format!("Canlang archive is missing {file}"));
        }
    }
    let tools: serde_json::Value =
        serde_json::from_reader(File::open(root.join("tools.json")).map_err(|e| e.to_string())?)
            .map_err(|e| e.to_string())?;
    if tools["schema"] != "canlang.tools.v1" || tools["sourceRevision"] != release.source_revision {
        return Err("Canlang producer identity mismatch".into());
    }
    std::fs::rename(root, destination).map_err(|e| e.to_string())?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use flate2::{write::GzEncoder, Compression};
    use std::io::Write;

    fn fixture(root: &Path, link: bool) -> (std::path::PathBuf, std::path::PathBuf) {
        let archive = root.join("canlang-tools.tar.gz");
        let mut builder = tar::Builder::new(GzEncoder::new(
            File::create(&archive).unwrap(),
            Compression::default(),
        ));
        for name in [
            "canlang/bin/can",
            "canlang/bin/can-platform",
            "canlang/bin/can-dev",
            "canlang/bin/can-soda",
            "canlang/libexec/node",
            "canlang/libexec/bun",
            "canlang/tools.json",
        ] {
            let content = if name.ends_with("tools.json") {
                format!(
                    "{{\"schema\":\"canlang.tools.v1\",\"sourceRevision\":\"{}\"}}",
                    "a".repeat(40)
                )
            } else {
                "fixture".to_string()
            };
            let mut header = tar::Header::new_gnu();
            header.set_size(content.len() as u64);
            header.set_mode(0o755);
            header.set_cksum();
            builder
                .append_data(&mut header, name, content.as_bytes())
                .unwrap();
        }
        if link {
            let mut header = tar::Header::new_gnu();
            header.set_size(0);
            header.set_mode(0o755);
            header.set_entry_type(tar::EntryType::Symlink);
            header.set_link_name("/home/foreign").unwrap();
            header.set_cksum();
            builder
                .append_data(&mut header, "canlang/escape", std::io::empty())
                .unwrap();
        }
        builder.into_inner().unwrap().finish().unwrap();
        let digest = super::super::sha256_hex(&std::fs::read(&archive).unwrap());
        let manifest = root.join("release.json");
        let release = serde_json::json!({"sourceRevision": "a".repeat(40), "archiveFilename": "canlang-tools.tar.gz", "sha256": digest,
            "versions": {"compiler": "can 0.1.0 (fixture)", "node": "v24.21.0", "bun": "1.4.2", "miniflare": "4.20260730.0", "workerd": "1.20260730.1"}});
        File::create(&manifest)
            .unwrap()
            .write_all(release.to_string().as_bytes())
            .unwrap();
        (manifest, archive)
    }

    #[test]
    fn admits_exact_archive_and_preserves_occupied_output() {
        let dir = tempfile::tempdir().unwrap();
        let (manifest, archive) = fixture(dir.path(), false);
        let out = dir.path().join("tools");
        stage(&manifest, &archive, &out).unwrap();
        assert!(stage(&manifest, &archive, &out)
            .unwrap_err()
            .contains("occupied"));
        assert_eq!(
            std::fs::read_to_string(out.join("canlang/bin/can")).unwrap(),
            "fixture"
        );
    }

    #[test]
    fn rejects_links_without_promoting_partial_output() {
        let dir = tempfile::tempdir().unwrap();
        let (manifest, archive) = fixture(dir.path(), true);
        let out = dir.path().join("tools");
        assert!(stage(&manifest, &archive, &out)
            .unwrap_err()
            .contains("links"));
        assert!(!out.join("canlang").exists());
    }

    #[test]
    fn rejects_changed_bytes_before_unpacking() {
        let dir = tempfile::tempdir().unwrap();
        let (manifest, archive) = fixture(dir.path(), false);
        std::fs::OpenOptions::new()
            .append(true)
            .open(&archive)
            .unwrap()
            .write_all(b"changed")
            .unwrap();
        assert!(stage(&manifest, &archive, &dir.path().join("tools"))
            .unwrap_err()
            .contains("checksum"));
    }
}

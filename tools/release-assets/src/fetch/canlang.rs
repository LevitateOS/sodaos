//! Admit a locally produced Canlang tool archive; startup never fetches packages.
use flate2::read::GzDecoder;
use serde::{Deserialize, Serialize};
use std::collections::HashSet;
use std::fs::File;
use std::io::{Read, Seek, SeekFrom};
use std::path::{Component, Path};

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct Release {
    pub source_revision: String,
    pub archive_filename: String,
    pub sha256: String,
    pub versions: Versions,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Versions {
    pub compiler: String,
    pub node: String,
    pub bun: String,
    pub miniflare: String,
    pub workerd: String,
}

pub fn stage(manifest: &Path, archive: &Path, out: &Path, arch: &str) -> Result<(), String> {
    let filename =
        soda_build_tools::reader::canlang::archive_filename(arch).map_err(|error| error.0)?;
    let release: Release =
        serde_json::from_reader(File::open(manifest).map_err(|error| {
            format!("Canlang {arch} release pin unavailable at {}: {error}; supply the verified native tools release before building", manifest.display())
        })?)
            .map_err(|e| format!("invalid Canlang release: {e}"))?;
    if release.source_revision.len() != 40
        || !release
            .source_revision
            .bytes()
            .all(|b| b.is_ascii_hexdigit())
        || release.sha256.len() != 64
        || !release.sha256.bytes().all(|b| b.is_ascii_hexdigit())
        || release.archive_filename != filename
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
        if kind.is_file() {
            verify_elf(&target, arch, false)?;
        }
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
        "compiler/target/release/can",
        "tools.json",
    ] {
        if !root.join(file).is_file() {
            return Err(format!("Canlang archive is missing {file}"));
        }
    }
    let tools: serde_json::Value =
        serde_json::from_reader(File::open(root.join("tools.json")).map_err(|e| e.to_string())?)
            .map_err(|e| e.to_string())?;
    if tools["schema"] != "canlang.tools.v1"
        || tools["sourceRevision"] != release.source_revision
        || tools["platform"] != format!("linux-{arch}")
        || tools["versions"]
            != serde_json::to_value(&release.versions).map_err(|error| error.to_string())?
    {
        return Err("Canlang producer identity mismatch".into());
    }
    for file in ["compiler/target/release/can", "libexec/node", "libexec/bun"] {
        verify_elf(&root.join(file), arch, true)?;
    }
    std::fs::rename(root, destination).map_err(|e| e.to_string())?;
    Ok(())
}

/// Inspect every delivered ELF (including workerd and native Node extensions),
/// and require the compiler/interpreters to actually be native executables.
fn verify_elf(path: &Path, arch: &str, required: bool) -> Result<(), String> {
    let mut header = Vec::new();
    File::open(path)
        .map_err(|error| error.to_string())?
        .take(soda_build_tools::elf::HEADER_LEN as u64)
        .read_to_end(&mut header)
        .map_err(|error| error.to_string())?;
    if !required && !header.starts_with(b"\x7fELF") {
        return Ok(());
    }
    let machine = match arch {
        "x86_64" => 62,
        "aarch64" => 183,
        _ => return Err("unsupported Canlang architecture".into()),
    };
    if !soda_build_tools::elf::elf64_le_header(&header)
        .is_some_and(|header| header.machine == machine && matches!(header.file_type, 2 | 3))
    {
        return Err(format!(
            "Canlang ELF does not match {arch}: {}",
            path.display()
        ));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use flate2::{write::GzEncoder, Compression};
    use std::io::Write;

    fn fixture(
        root: &Path,
        link: bool,
        arch: &str,
        machine: u16,
    ) -> (std::path::PathBuf, std::path::PathBuf) {
        let filename = soda_build_tools::reader::canlang::archive_filename(arch).unwrap();
        let archive = root.join(filename);
        let versions = serde_json::json!({"compiler": "can 0.1.0 (fixture)", "node": "v24.21.0", "bun": "1.4.2", "miniflare": "4.20260730.0", "workerd": "1.20260730.1"});
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
            "canlang/compiler/target/release/can",
            "canlang/lib/node_modules/workerd/bin/workerd",
            "canlang/tools.json",
        ] {
            let content = if name.ends_with("tools.json") {
                serde_json::to_vec(&serde_json::json!({"schema":"canlang.tools.v1", "sourceRevision":"a".repeat(40), "platform":format!("linux-{arch}"), "versions":versions})).unwrap()
            } else if name.contains("/libexec/")
                || name.contains("/target/")
                || name.ends_with("/workerd")
            {
                let mut bytes = vec![0; 64];
                bytes[..6].copy_from_slice(b"\x7fELF\x02\x01");
                bytes[16..18].copy_from_slice(&2u16.to_le_bytes());
                bytes[18..20].copy_from_slice(&machine.to_le_bytes());
                bytes
            } else {
                b"fixture".to_vec()
            };
            let mut header = tar::Header::new_gnu();
            header.set_size(content.len() as u64);
            header.set_mode(0o755);
            header.set_cksum();
            builder
                .append_data(&mut header, name, content.as_slice())
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
        let release = serde_json::json!({"sourceRevision": "a".repeat(40), "archiveFilename": filename, "sha256": digest, "versions": versions});
        File::create(&manifest)
            .unwrap()
            .write_all(release.to_string().as_bytes())
            .unwrap();
        (manifest, archive)
    }

    #[test]
    fn admits_exact_archive_and_preserves_occupied_output() {
        let dir = tempfile::tempdir().unwrap();
        let (manifest, archive) = fixture(dir.path(), false, "x86_64", 62);
        let out = dir.path().join("tools");
        stage(&manifest, &archive, &out, "x86_64").unwrap();
        assert!(stage(&manifest, &archive, &out, "x86_64")
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
        let (manifest, archive) = fixture(dir.path(), true, "x86_64", 62);
        let out = dir.path().join("tools");
        assert!(stage(&manifest, &archive, &out, "x86_64")
            .unwrap_err()
            .contains("links"));
        assert!(!out.join("canlang").exists());
    }

    #[test]
    fn rejects_changed_bytes_before_unpacking() {
        let dir = tempfile::tempdir().unwrap();
        let (manifest, archive) = fixture(dir.path(), false, "x86_64", 62);
        std::fs::OpenOptions::new()
            .append(true)
            .open(&archive)
            .unwrap()
            .write_all(b"changed")
            .unwrap();
        assert!(
            stage(&manifest, &archive, &dir.path().join("tools"), "x86_64")
                .unwrap_err()
                .contains("checksum")
        );
    }

    #[test]
    fn arm_release_requires_native_arm_bytes_before_promotion() {
        for machine in [62, 183] {
            let dir = tempfile::tempdir().unwrap();
            let (manifest, archive) = fixture(dir.path(), false, "aarch64", machine);
            let out = dir.path().join("tools");
            let result = stage(&manifest, &archive, &out, "aarch64");
            if machine == 183 {
                result.unwrap();
                assert!(out.join("canlang/compiler/target/release/can").is_file());
            } else {
                assert!(result.unwrap_err().contains("ELF does not match aarch64"));
                assert!(!out.join("canlang").exists());
            }
        }
    }

    #[test]
    fn absent_arm_lock_reports_the_required_dependency_without_writing() {
        let dir = tempfile::tempdir().unwrap();
        let out = dir.path().join("tools");
        let error = stage(
            &dir.path().join("absent.json"),
            &dir.path().join("archive"),
            &out,
            "aarch64",
        )
        .unwrap_err();
        assert!(error.contains("Canlang aarch64 release pin unavailable"));
        assert!(!out.exists());
    }
}

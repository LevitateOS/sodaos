//! Admit a locally produced Canlang tool archive; startup never fetches packages.
use flate2::read::GzDecoder;
use serde::Deserialize;
use std::collections::HashSet;
use std::fs::File;
use std::io::{Seek, SeekFrom};
use std::os::unix::fs::PermissionsExt;
use std::path::{Component, Path, PathBuf};

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct Release {
    pub source_revision: String,
    pub archive_filename: String,
    pub sha256: String,
    #[serde(rename = "versions", default)]
    pub _versions: serde_json::Value,
}

fn load_release(manifest: &Path) -> Result<Release, String> {
    serde_json::from_reader(File::open(manifest).map_err(|e| e.to_string())?)
        .map_err(|e| format!("invalid Canlang release: {e}"))
}

fn valid_hex(value: &str, len: usize) -> bool {
    value.len() == len && value.bytes().all(|b| b.is_ascii_hexdigit())
}

fn check_release_identity(release: &Release, archive: &Path) -> Result<(), String> {
    if !valid_hex(&release.source_revision, 40)
        || !valid_hex(&release.sha256, 64)
        || Path::new(&release.archive_filename).components().count() != 1
        || archive.file_name().and_then(|s| s.to_str()) != Some(&release.archive_filename)
    {
        return Err("invalid Canlang release identity".into());
    }
    Ok(())
}

fn open_verified_archive(archive: &Path, digest: &str) -> Result<File, String> {
    let mut input = File::open(archive).map_err(|e| format!("Canlang archive unavailable: {e}"))?;
    if input.metadata().map_err(|e| e.to_string())?.len() > 1_000_000_000 {
        return Err("Canlang archive exceeds size limit".into());
    }
    if super::sha256_hex_stream(&mut input)? != digest {
        return Err("Canlang archive checksum mismatch".into());
    }
    Ok(input)
}

fn check_entry_path(path: &Path, seen: &mut HashSet<PathBuf>) -> Result<(), String> {
    if path
        .components()
        .any(|c| !matches!(c, Component::Normal(_)))
        || !path.starts_with("canlang")
        || !seen.insert(path.to_path_buf())
        || seen.len() > 100_000
    {
        return Err("invalid Canlang archive path".into());
    }
    Ok(())
}

fn extract_entries(input: File, staging: &Path) -> Result<(), String> {
    let mut tar = tar::Archive::new(GzDecoder::new(input));
    let mut seen = HashSet::new();
    let mut bytes = 0u64;
    for entry in tar.entries().map_err(|e| e.to_string())? {
        let mut entry = entry.map_err(|e| e.to_string())?;
        let path = entry.path().map_err(|e| e.to_string())?.into_owned();
        check_entry_path(&path, &mut seen)?;
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
        let target = staging.join(&path);
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
    Ok(())
}

fn verify_staged_tree(root: &Path, source_revision: &str) -> Result<(), String> {
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
    if tools["schema"] != "canlang.tools.v1" || tools["sourceRevision"] != source_revision {
        return Err("Canlang producer identity mismatch".into());
    }
    verify_runtime(root, &tools)?;
    Ok(())
}

fn runtime_paths(
    root: &Path,
    directory: &Path,
    paths: &mut HashSet<PathBuf>,
) -> Result<(), String> {
    for entry in std::fs::read_dir(directory).map_err(|e| e.to_string())? {
        let entry = entry.map_err(|e| e.to_string())?;
        let path = entry.path();
        let kind = entry.file_type().map_err(|e| e.to_string())?;
        if kind.is_dir() {
            runtime_paths(root, &path, paths)?;
        } else if kind.is_file() {
            let relative = path.strip_prefix(root).map_err(|e| e.to_string())?;
            if relative != Path::new("runtime.json") {
                paths.insert(relative.to_path_buf());
            }
        } else {
            return Err("Canlang runtime contains a link or special file".into());
        }
    }
    Ok(())
}

fn verify_runtime(root: &Path, tools: &serde_json::Value) -> Result<(), String> {
    let runtime = root.join("runtime");
    let manifest: serde_json::Value = serde_json::from_reader(
        File::open(runtime.join("runtime.json"))
            .map_err(|e| format!("Canlang production runtime missing: {e}"))?,
    )
    .map_err(|e| e.to_string())?;
    if tools["runtime"]["path"] != "runtime"
        || tools["runtime"]["schema"] != "canlang.soda.runtime.v1"
        || manifest["schema"] != "canlang.soda.runtime.v1"
        || manifest["platform"] != "linux-x86_64"
        || manifest["command"] != "bin/can-soda"
        || manifest["workerd"] != tools["versions"]["workerd"]
        || manifest["files"] != tools["runtime"]["files"]
    {
        return Err("Canlang production runtime inventory differs from its tools archive".into());
    }
    let files = manifest["files"]
        .as_array()
        .filter(|files| !files.is_empty() && files.len() <= 100_000)
        .ok_or("Canlang runtime file inventory missing or exceeds bound")?;
    let mut expected = HashSet::new();
    for file in files {
        let path = Path::new(
            file["path"]
                .as_str()
                .ok_or("Canlang runtime file path missing")?,
        );
        if path.components().count() == 0
            || path
                .components()
                .any(|part| !matches!(part, Component::Normal(_)))
            || !expected.insert(path.to_path_buf())
        {
            return Err("Canlang runtime file path invalid or duplicated".into());
        }
        let digest = file["sha256"]
            .as_str()
            .filter(|digest| valid_hex(digest, 64))
            .ok_or("Canlang runtime file checksum invalid")?;
        let mut input = File::open(runtime.join(path)).map_err(|e| e.to_string())?;
        let metadata = input.metadata().map_err(|e| e.to_string())?;
        if !metadata.is_file()
            || file["bytes"].as_u64() != Some(metadata.len())
            || file["mode"].as_u64() != Some(u64::from(metadata.permissions().mode() & 0o777))
            || super::sha256_hex_stream(&mut input)? != digest
        {
            return Err(format!("Canlang runtime file differs: {}", path.display()));
        }
    }
    for path in [
        "bin/can-soda",
        "bin/node",
        "libexec/node",
        "lib/node_modules/@canlang/cloudflare/dist/cli/soda-runtime.js",
        "lib/node_modules/@canlang/cloudflare/dist/host/can-workerd-exec",
        "lib/node_modules/@canlang/cloudflare/dist/host/soda-admin.worker.js",
        "lib/node_modules/@cloudflare/workerd-linux-64/bin/workerd",
        "licenses/Node-LICENSE",
        "licenses/Miniflare-D1-LICENSE",
        "licenses/workerd-LICENSE",
    ] {
        if !expected.contains(Path::new(path)) {
            return Err(format!("Canlang production runtime is missing {path}"));
        }
    }
    let mut actual = HashSet::new();
    runtime_paths(&runtime, &runtime, &mut actual)?;
    if actual != expected {
        return Err("Canlang runtime tree has files outside its inventory".into());
    }
    Ok(())
}

fn prepare_destination(out: &Path) -> Result<PathBuf, String> {
    std::fs::create_dir_all(out).map_err(|e| e.to_string())?;
    let destination = out.join("canlang");
    if destination.symlink_metadata().is_ok() {
        return Err("Canlang destination is occupied".into());
    }
    Ok(destination)
}

fn populate_staging(
    mut input: File,
    out: &Path,
    destination: &Path,
    source_revision: &str,
) -> Result<(), String> {
    let staging = tempfile::Builder::new()
        .prefix(".canlang-stage-")
        .tempdir_in(out)
        .map_err(|e| e.to_string())?;
    input.seek(SeekFrom::Start(0)).map_err(|e| e.to_string())?;
    extract_entries(input, staging.path())?;
    let root = staging.path().join("canlang");
    verify_staged_tree(&root, source_revision)?;
    std::fs::rename(root, destination).map_err(|e| e.to_string())?;
    Ok(())
}

pub fn stage(manifest: &Path, archive: &Path, out: &Path) -> Result<(), String> {
    let release = load_release(manifest)?;
    check_release_identity(&release, archive)?;
    let input = open_verified_archive(archive, &release.sha256)?;
    let destination = prepare_destination(out)?;
    populate_staging(input, out, &destination, &release.source_revision)
}

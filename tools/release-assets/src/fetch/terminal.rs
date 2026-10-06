//! Pinned terminal distribution extraction (`scripts/fetch-terminal.py`).

use std::io::Read;
use std::path::{Path, PathBuf};
use std::time::Duration;

pub const USER_AGENT: &str = "SodaOS-build";
const TIMEOUT: Duration = Duration::from_secs(30);
const ARCHIVE_LIMIT: u64 = 10_000_000;
const MEMBER_LIMIT: u64 = 2_000_000;

struct Asset {
    member: String,
    file: String,
    sha256: String,
}

struct Item {
    url: String,
    integrity: String,
    files: Vec<Asset>,
}

fn lock_string<'a>(item: &'a soda_json::JsonValue, key: &str) -> Result<&'a str, String> {
    item.get(key)
        .and_then(|v| v.as_str())
        .ok_or_else(|| "terminal lock entry is malformed".to_string())
}

fn parse_lock(text: &str) -> Result<Vec<Item>, String> {
    let lock = soda_json::JsonValue::parse(text)
        .map_err(|_| "terminal lock is not valid JSON".to_string())?;
    let raw_items = match &lock {
        soda_json::JsonValue::Array(items) => items,
        _ => return Err("terminal lock is not valid JSON".to_string()),
    };
    let mut items = Vec::with_capacity(raw_items.len());
    for raw in raw_items {
        let files = match raw.get("files") {
            Some(soda_json::JsonValue::Array(files)) => files,
            _ => return Err("terminal lock entry is malformed".to_string()),
        };
        let mut assets = Vec::with_capacity(files.len());
        for file in files {
            assets.push(Asset {
                member: lock_string(file, "member")?.to_string(),
                file: lock_string(file, "file")?.to_string(),
                sha256: lock_string(file, "sha256")?.to_string(),
            });
        }
        items.push(Item {
            url: lock_string(raw, "url")?.to_string(),
            integrity: lock_string(raw, "integrity")?.to_string(),
            files: assets,
        });
    }
    Ok(items)
}

/// The owner reads `appliance/terminal-assets.lock.json` under the repo
/// root; every caller runs fetchers from the source root, so the working
/// directory resolves the same file.
pub fn default_lock() -> Result<PathBuf, String> {
    let cwd = std::env::current_dir().map_err(|e| e.to_string())?;
    Ok(cwd.join("appliance/terminal-assets.lock.json"))
}

/// Every listed file already staged with a matching digest. An item with
/// no files is trivially satisfied, like the owner's `all()`.
fn cached(out: &Path, files: &[Asset]) -> Result<bool, String> {
    for asset in files {
        let path = out.join(&asset.file);
        if !path.is_file() {
            return Ok(false);
        }
        let data = std::fs::read(&path).map_err(|e| e.to_string())?;
        if crate::fetch::sha256_hex(&data) != asset.sha256 {
            return Ok(false);
        }
    }
    Ok(true)
}

struct Member {
    name: String,
    is_file: bool,
    size: u64,
    data: Vec<u8>,
}

fn read_members(body: &[u8]) -> Result<Vec<Member>, String> {
    let gz = flate2::read::GzDecoder::new(body);
    let mut archive = tar::Archive::new(gz);
    let mut members = Vec::new();
    let entries = archive
        .entries()
        .map_err(|e| format!("terminal archive is not readable: {e}"))?;
    for entry in entries {
        let mut entry = entry.map_err(|e| format!("terminal archive is not readable: {e}"))?;
        let name = entry
            .path()
            .map_err(|e| format!("terminal archive is not readable: {e}"))?
            .to_str()
            .unwrap_or("")
            .to_string();
        let is_file = entry.header().entry_type().is_file();
        let size = entry.header().size().unwrap_or(u64::MAX);
        let mut data = Vec::new();
        entry
            .read_to_end(&mut data)
            .map_err(|e| format!("terminal archive is not readable: {e}"))?;
        members.push(Member {
            name,
            is_file,
            size,
            data,
        });
    }
    Ok(members)
}

fn extract(members: &[Member], asset: &Asset, out: &Path) -> Result<(), String> {
    // First exact-name match, like `getmember` (a missing member surfaces
    // as the invalid-member error rather than a bare key error).
    let member = members
        .iter()
        .find(|m| m.name == asset.member)
        .ok_or_else(|| "invalid terminal distribution member".to_string())?;
    if !member.is_file || member.size > MEMBER_LIMIT || asset.file.contains('/') {
        return Err("invalid terminal distribution member".to_string());
    }
    if crate::fetch::sha256_hex(&member.data) != asset.sha256 {
        return Err("terminal asset integrity mismatch".to_string());
    }
    std::fs::write(out.join(&asset.file), &member.data).map_err(|e| e.to_string())
}

pub fn fetch(lock_path: &Path, out: &Path) -> Result<(), String> {
    std::fs::create_dir_all(out).map_err(|e| e.to_string())?;
    let text = std::fs::read_to_string(lock_path)
        .map_err(|e| format!("cannot read terminal lock {}: {e}", lock_path.display()))?;
    let items = parse_lock(&text)?;
    for item in &items {
        if cached(out, &item.files)? {
            continue;
        }
        let mut response = crate::fetch::http_get(&item.url, USER_AGENT, TIMEOUT)
            .map_err(|e| format!("fetch {}: {e}", item.url))?;
        if !(200..300).contains(&response.status) {
            return Err(format!(
                "fetch {}: unexpected HTTP status {}",
                item.url, response.status
            ));
        }
        let body = crate::fetch::read_capped(&mut response.reader, ARCHIVE_LIMIT)?;
        if body.len() as u64 > ARCHIVE_LIMIT
            || format!("sha512-{}", crate::fetch::sha512_base64(&body)) != item.integrity
        {
            return Err("terminal archive integrity mismatch".to_string());
        }
        let members = read_members(&body)?;
        for asset in &item.files {
            extract(&members, asset, out)?;
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::fetch::test_server::{Server, TempDir};
    use std::collections::HashMap;
    use std::io::Write;

    fn tarball(entries: &[(&str, &[u8])]) -> Vec<u8> {
        let mut tar = tar::Builder::new(Vec::new());
        for (name, data) in entries {
            let mut header = tar::Header::new_gnu();
            header.set_size(data.len() as u64);
            header.set_mode(0o644);
            header.set_cksum();
            tar.append_data(&mut header, name, *data).unwrap();
        }
        let tarred = tar.into_inner().unwrap();
        let mut encoder = flate2::write::GzEncoder::new(Vec::new(), flate2::Compression::default());
        encoder.write_all(&tarred).unwrap();
        encoder.finish().unwrap()
    }

    fn integrity(body: &[u8]) -> String {
        format!("sha512-{}", crate::fetch::sha512_base64(body))
    }

    fn write_lock(dir: &Path, text: &str) -> PathBuf {
        let path = dir.join("terminal-assets.lock.json");
        std::fs::write(&path, text).unwrap();
        path
    }

    #[test]
    fn exact_members_checksums_and_cached_bytes() {
        let body = tarball(&[("package/lib/xterm.mjs", b"synthetic")]);
        let file_sha = crate::fetch::sha256_hex(b"synthetic");
        let mut routes = HashMap::new();
        routes.insert("/fixture.tgz".to_string(), (200, body.clone()));
        let server = Server::start(routes);
        let lock = format!(
            r#"[{{"url":"{}/fixture.tgz","integrity":"{}","files":[{{"member":"package/lib/xterm.mjs","file":"xterm.mjs","sha256":"{file_sha}"}}]}}]"#,
            server.base,
            integrity(&body),
        );
        let scratch = TempDir::new("terminal-ok");
        let lock_path = write_lock(&scratch.path, &lock);
        let out = scratch.path.join("out");
        fetch(&lock_path, &out).unwrap();
        assert_eq!(std::fs::read(out.join("xterm.mjs")).unwrap(), b"synthetic");
        // Second run is served from the verified cache: no new download.
        fetch(&lock_path, &out).unwrap();
        assert_eq!(server.seen().len(), 1);
        assert_eq!(server.seen()[0].user_agent, "SodaOS-build");
        // A changed cache re-downloads; the good archive repairs it.
        std::fs::write(out.join("xterm.mjs"), b"changed").unwrap();
        fetch(&lock_path, &out).unwrap();
        assert_eq!(std::fs::read(out.join("xterm.mjs")).unwrap(), b"synthetic");
        assert_eq!(server.seen().len(), 2);
    }

    #[test]
    fn integrity_failure_keeps_changed_cache_bytes() {
        let body = tarball(&[("package/lib/xterm.mjs", b"synthetic")]);
        let file_sha = crate::fetch::sha256_hex(b"synthetic");
        let mut routes = HashMap::new();
        routes.insert("/fixture.tgz".to_string(), (200, b"bad archive".to_vec()));
        let server = Server::start(routes);
        let lock = format!(
            r#"[{{"url":"{}/fixture.tgz","integrity":"{}","files":[{{"member":"package/lib/xterm.mjs","file":"xterm.mjs","sha256":"{file_sha}"}}]}}]"#,
            server.base,
            integrity(&body),
        );
        let scratch = TempDir::new("terminal-bad");
        let lock_path = write_lock(&scratch.path, &lock);
        let out = scratch.path.join("out");
        std::fs::create_dir_all(&out).unwrap();
        std::fs::write(out.join("xterm.mjs"), b"changed").unwrap();
        assert_eq!(
            fetch(&lock_path, &out).unwrap_err(),
            "terminal archive integrity mismatch"
        );
        assert_eq!(std::fs::read(out.join("xterm.mjs")).unwrap(), b"changed");
    }

    #[test]
    fn per_file_digest_mismatch_is_reported() {
        let body = tarball(&[("package/lib/xterm.mjs", b"synthetic")]);
        let mut routes = HashMap::new();
        routes.insert("/fixture.tgz".to_string(), (200, body.clone()));
        let server = Server::start(routes);
        let lock = format!(
            r#"[{{"url":"{}/fixture.tgz","integrity":"{}","files":[{{"member":"package/lib/xterm.mjs","file":"xterm.mjs","sha256":"{}"}}]}}]"#,
            server.base,
            integrity(&body),
            "0".repeat(64),
        );
        let scratch = TempDir::new("terminal-sha");
        let lock_path = write_lock(&scratch.path, &lock);
        assert_eq!(
            fetch(&lock_path, &scratch.path.join("out")).unwrap_err(),
            "terminal asset integrity mismatch"
        );
    }

    #[test]
    fn invalid_members_are_refused() {
        let body = tarball(&[("package/lib/xterm.mjs", b"synthetic")]);
        let good = crate::fetch::sha256_hex(b"synthetic");
        let cases: Vec<(&str, &str, String)> = vec![
            // Missing member.
            ("package/lib/absent.mjs", "absent.mjs", good.clone()),
            // Output escapes the directory.
            ("package/lib/xterm.mjs", "sub/xterm.mjs", good.clone()),
            ("package/lib/xterm.mjs", "/abs.mjs", good),
        ];
        for (index, (member, file, sha)) in cases.iter().enumerate() {
            let mut routes = HashMap::new();
            routes.insert("/fixture.tgz".to_string(), (200, body.clone()));
            let server = Server::start(routes);
            let lock = format!(
                r#"[{{"url":"{}/fixture.tgz","integrity":"{}","files":[{{"member":"{member}","file":"{file}","sha256":"{sha}"}}]}}]"#,
                server.base,
                integrity(&body),
            );
            let scratch = TempDir::new("terminal-member");
            let lock_path = write_lock(&scratch.path, &lock);
            assert_eq!(
                fetch(&lock_path, &scratch.path.join(format!("out-{index}"))).unwrap_err(),
                "invalid terminal distribution member",
                "case {member} -> {file}"
            );
        }
    }

    #[test]
    fn directory_member_is_refused() {
        let mut tar = tar::Builder::new(Vec::new());
        tar.append_dir("package/lib", ".").unwrap();
        let tarred = tar.into_inner().unwrap();
        let mut encoder = flate2::write::GzEncoder::new(Vec::new(), flate2::Compression::default());
        encoder.write_all(&tarred).unwrap();
        let body = encoder.finish().unwrap();
        let mut routes = HashMap::new();
        routes.insert("/fixture.tgz".to_string(), (200, body.clone()));
        let server = Server::start(routes);
        let lock = format!(
            r#"[{{"url":"{}/fixture.tgz","integrity":"{}","files":[{{"member":"package/lib","file":"lib","sha256":"{}"}}]}}]"#,
            server.base,
            integrity(&body),
            "0".repeat(64),
        );
        let scratch = TempDir::new("terminal-dir");
        let lock_path = write_lock(&scratch.path, &lock);
        assert_eq!(
            fetch(&lock_path, &scratch.path.join("out")).unwrap_err(),
            "invalid terminal distribution member"
        );
    }

    #[test]
    fn oversized_archive_is_refused() {
        let body = vec![0x41u8; 10_000_001];
        let mut routes = HashMap::new();
        routes.insert("/fixture.tgz".to_string(), (200, body));
        let server = Server::start(routes);
        // Any integrity pin: the size cap trips first either way. One file
        // forces the fetch (empty file lists never download).
        let lock = format!(
            r#"[{{"url":"{}/fixture.tgz","integrity":"sha512-{}","files":[{{"member":"m","file":"f","sha256":"s"}}]}}]"#,
            server.base,
            "A".repeat(88),
        );
        let scratch = TempDir::new("terminal-big");
        let lock_path = write_lock(&scratch.path, &lock);
        assert_eq!(
            fetch(&lock_path, &scratch.path.join("out")).unwrap_err(),
            "terminal archive integrity mismatch"
        );
    }

    #[test]
    fn empty_file_list_skips_the_download() {
        let routes = HashMap::new();
        let server = Server::start(routes);
        let lock = format!(
            r#"[{{"url":"{}/fixture.tgz","integrity":"sha512-{}","files":[]}}]"#,
            server.base,
            "A".repeat(88),
        );
        let scratch = TempDir::new("terminal-empty");
        let lock_path = write_lock(&scratch.path, &lock);
        let out = scratch.path.join("out");
        fetch(&lock_path, &out).unwrap();
        assert!(server.seen().is_empty());
        assert!(out.is_dir());
    }

    #[test]
    fn malformed_lock_and_http_errors_fail() {
        let scratch = TempDir::new("terminal-malformed");
        let bad = write_lock(&scratch.path, "not json");
        assert!(fetch(&bad, &scratch.path.join("out")).is_err());
        let bad = write_lock(&scratch.path, r#"{"url": 1}"#);
        assert!(fetch(&bad, &scratch.path.join("out")).is_err());
        let mut routes = HashMap::new();
        routes.insert("/fixture.tgz".to_string(), (404, Vec::new()));
        let server = Server::start(routes);
        let lock = format!(
            r#"[{{"url":"{}/fixture.tgz","integrity":"sha512-{}","files":[{{"member":"m","file":"f","sha256":"s"}}]}}]"#,
            server.base,
            "A".repeat(88),
        );
        let lock_path = write_lock(&scratch.path, &lock);
        let err = fetch(&lock_path, &scratch.path.join("out2")).unwrap_err();
        assert!(err.contains("404"), "{err}");
    }

    #[test]
    fn default_lock_is_repo_shaped() {
        let lock = default_lock().unwrap();
        assert!(lock.is_absolute());
        assert!(
            lock.ends_with("appliance/terminal-assets.lock.json"),
            "{}",
            lock.display()
        );
    }
}

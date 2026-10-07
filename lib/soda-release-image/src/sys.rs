//! Small exact mirrors of the foreign `release/build` + `release/deliver`
//! helpers the pipeline calls: path lexics, file hashing, exclusive writes,
//! fresh directories, bounded strict JSON reads, native admission, command
//! inventory, and the private-file check. Heavy foreign operations (OCI,
//! signing, live resolution) live behind [`crate::foreign::Production`].

use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::{Component, Path, PathBuf};

use crate::error::Error;
use crate::jsonio;
use serde::Serialize;
use sha2::{Digest, Sha256};

/// Shared inventory primitive (`build.File`).
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize)]
pub struct File {
    #[serde(rename = "sha256", skip_serializing_if = "String::is_empty")]
    pub sha256: String,
    #[serde(rename = "mode")]
    pub mode: u32,
    #[serde(rename = "link", skip_serializing_if = "String::is_empty")]
    pub link: String,
    #[serde(rename = "directory", skip_serializing_if = "is_false")]
    pub directory: bool,
}

fn is_false(value: &bool) -> bool {
    !value
}

impl File {
    pub fn parse(text: &str) -> Result<File, Error> {
        jsonio::parse(text)
    }
}

crate::jsonio::case_record!(File, {
    sha256: String => "sha256",
    mode: u32 => "mode",
    link: String => "link",
    directory: bool => "directory",
});

/// Go `path/filepath.Clean` lexics.
pub fn clean_path(path: &str) -> String {
    if path.is_empty() {
        return ".".to_string();
    }
    let rooted = path.starts_with('/');
    let mut out: Vec<&str> = Vec::new();
    for part in path.split('/') {
        match part {
            "" | "." => {}
            ".." => {
                if out.pop().is_none() && !rooted {
                    out.push("..");
                }
            }
            _ => out.push(part),
        }
    }
    let mut joined = out.join("/");
    if rooted {
        joined.insert(0, '/');
    }
    if joined.is_empty() {
        return if rooted {
            "/".to_string()
        } else {
            ".".to_string()
        };
    }
    joined
}

pub fn is_abs(path: &str) -> bool {
    path.starts_with('/')
}

pub fn join<S: AsRef<str>>(parts: &[S]) -> String {
    let mut buf = PathBuf::new();
    for part in parts {
        if part.as_ref().is_empty() {
            continue;
        }
        buf.push(part.as_ref());
    }
    clean_path(&buf.to_string_lossy())
}

pub fn dir_name(path: &str) -> String {
    let cleaned = clean_path(path);
    match cleaned.rfind('/') {
        None => ".".to_string(),
        Some(0) => "/".to_string(),
        Some(i) => cleaned[..i].to_string(),
    }
}

pub fn base_name(path: &str) -> String {
    let cleaned = clean_path(path);
    if cleaned == "/" {
        return "/".to_string();
    }
    cleaned.rsplit('/').next().unwrap_or("").to_string()
}

/// Go `filepath.Rel`: lexical relative path or an error.
pub fn rel_path(base: &str, target: &str) -> Result<String, Error> {
    let b = clean_path(base);
    let t = clean_path(target);
    let b_parts: Vec<&str> = b.split('/').filter(|p| !p.is_empty()).collect();
    let t_parts: Vec<&str> = t.split('/').filter(|p| !p.is_empty()).collect();
    let mut common = 0;
    while common < b_parts.len() && common < t_parts.len() && b_parts[common] == t_parts[common] {
        common += 1;
    }
    if common == 0 && (b.starts_with('/') || t.starts_with('/')) {
        return Err(Error::msg("cannot make relative path"));
    }
    let mut out = vec![".."; b_parts.len() - common];
    out.extend_from_slice(&t_parts[common..]);
    if out.is_empty() {
        return Ok(".".to_string());
    }
    Ok(out.join("/"))
}

pub fn to_slash(path: &str) -> String {
    path.to_string()
}

/// Lexical path components without `.`/`..` surprises, for walk inventories.
pub fn components(path: &Path) -> Vec<String> {
    path.components()
        .filter_map(|c| match c {
            Component::Normal(part) => Some(part.to_string_lossy().into_owned()),
            _ => None,
        })
        .collect()
}

/// `build.HashFile`: SHA-256 of a regular non-symlink file.
pub fn hash_file(path: &str) -> Result<String, Error> {
    let meta = fs::symlink_metadata(path)?;
    if !meta.file_type().is_file() {
        return Err(Error::msg("regular non-symlink file required"));
    }
    let data = fs::read(path)?;
    Ok(hex_sha256(&data))
}

pub fn hex_sha256(data: &[u8]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(data);
    hex_bytes(&hasher.finalize())
}

pub fn hex_bytes(bytes: &[u8]) -> String {
    let mut out = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        out.push_str(&format!("{b:02x}"));
    }
    out
}

/// `build.FreshDirectory`: create a new mode-700 directory below a real parent.
pub fn fresh_directory(path: &str) -> Result<(), Error> {
    if !is_abs(path) {
        return Err(Error::msg("absolute new directory required"));
    }
    let parent = dir_name(path);
    let resolved = fs::canonicalize(&parent).map_err(|e| Error::msg(e.to_string()))?;
    if resolved.to_string_lossy() != clean_path(&parent) {
        return Err(Error::msg("symlinked parent refused"));
    }
    fs::create_dir(path).map_err(|e| Error::msg(e.to_string()))?;
    fs::set_permissions(path, fs::Permissions::from_mode(0o700))?;
    Ok(())
}

/// `build.WriteNew`: exclusive-create write; refuses to overwrite.
pub fn write_new(path: &str, data: &[u8], mode: u32) -> Result<(), Error> {
    use std::fs::OpenOptions;
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(path)
        .map_err(|e| Error::msg(e.to_string()))?;
    use std::io::Write;
    let write_result = file.write_all(data).map_err(Error::from);
    drop(file);
    write_result?;
    fs::set_permissions(path, fs::Permissions::from_mode(mode))?;
    Ok(())
}

fn read_bounded(path: &str, maximum: usize, too_big: &str) -> Result<Vec<u8>, Error> {
    let meta = fs::symlink_metadata(path)?;
    if !meta.file_type().is_file() || meta.len() > maximum as u64 {
        return Err(Error::msg("bounded regular JSON input required"));
    }
    let data = fs::read(path)?;
    if data.len() > maximum {
        return Err(Error::msg(too_big));
    }
    Ok(data)
}

/// Keep the exact bounded input text available to the schema-specific DTO
/// visitor. The raw JSON bytes remain the authority for every consumer that
/// hashes or signs these files.
pub fn read_json_build_text(path: &str) -> Result<String, Error> {
    let data = read_bounded(path, 4 << 20, "JSON input exceeds limit")?;
    String::from_utf8(data).map_err(|_| Error::msg("invalid JSON"))
}

/// Deliver's bounded text counterpart for schema-specific raw-slot visitors.
pub fn read_json_deliver_text(path: &str) -> Result<String, Error> {
    let meta = fs::symlink_metadata(path).map_err(|_| Error::msg(refused()))?;
    if !meta.file_type().is_file() || meta.len() > (1 << 20) as u64 {
        return Err(Error::msg(refused()));
    }
    let data = fs::read(path).map_err(|_| Error::msg(refused()))?;
    if data.len() > 1 << 20 {
        return Err(Error::msg(refused()));
    }
    String::from_utf8(data).map_err(|_| Error::msg(refused()))
}

pub fn refused() -> String {
    "release authority or completeness refused".to_string()
}

/// `deliver.PrivateFile`: absolute, symlink-free, owned private regular file.
pub fn private_file(path: &str) -> Result<(), Error> {
    if !is_abs(path) {
        return Err(Error::msg(refused()));
    }
    let resolved = fs::canonicalize(path).map_err(|_| Error::msg(refused()))?;
    if resolved.to_string_lossy() != clean_path(path) {
        return Err(Error::msg(refused()));
    }
    let meta = fs::symlink_metadata(path).map_err(|_| Error::msg(refused()))?;
    if !meta.file_type().is_file() || meta.permissions().mode() & 0o077 != 0 {
        return Err(Error::msg(refused()));
    }
    Ok(())
}

#[cfg(test)]
mod json_tests {
    use super::File;

    #[test]
    fn file_mode_keeps_integer_token_and_minus_zero_policy() {
        assert_eq!(File::parse(r#"{"mode":-0}"#).unwrap().mode, 0);
        assert!(File::parse(r#"{"mode":1.0}"#).is_err());
        assert!(File::parse(r#"{"mode":1e0}"#).is_err());
        assert!(File::parse(r#"{"mode":4294967296}"#).is_err());
    }
}

/// `build.RequireNative`: x86_64 Linux only.
pub fn require_native(arch: &str) -> Result<(), Error> {
    let oci =
        soda_build_tools::reader::oci_architecture(arch).map_err(|e| Error::msg(e.to_string()))?;
    // D01-F1: Rust target arch and OCI arch are separate namespaces; each
    // is validated against its own supported value.
    if std::env::consts::OS != "linux" || std::env::consts::ARCH != "x86_64" || oci != "amd64" {
        return Err(Error::msg("matching-native Linux required"));
    }
    Ok(())
}

/// CORR-C-001: a cmd directory owned by Rust carries its own manifest. A
/// Go command directory never has a top-level Cargo.toml, so its presence
/// selects the Rust recipe; anything else stays on the Go recipe.
pub fn is_rust_command(cmd_dir: &str) -> bool {
    fs::metadata(join(&[cmd_dir, "Cargo.toml"]))
        .map(|meta| meta.is_file())
        .unwrap_or(false)
}

/// CODEX-CR02-001: `[[bin]]` names declared by a Rust command manifest.
/// Unreadable manifests yield no names so the caller keeps the previous
/// classification; the real workspace build owns those failures.
fn declared_bins(cmd_dir: &str) -> Vec<String> {
    let text = fs::read_to_string(join(&[cmd_dir, "Cargo.toml"])).unwrap_or_default();
    let mut names = Vec::new();
    let mut in_bin = false;
    for raw in text.lines() {
        let line = raw.trim();
        if line.starts_with('[') {
            let head = line.split('#').next().unwrap_or("").trim();
            in_bin = head == "[[bin]]";
            continue;
        }
        if !in_bin {
            continue;
        }
        let Some(rest) = line.strip_prefix("name") else {
            continue;
        };
        let Some(value) = rest.trim_start().strip_prefix('=') else {
            continue;
        };
        let mut value = value.trim();
        if value.len() >= 2 {
            let quote = value.as_bytes()[0];
            if (quote == b'"' || quote == b'\'') && value.as_bytes()[value.len() - 1] == quote {
                value = &value[1..value.len() - 1];
            } else if quote == b'"' || quote == b'\'' {
                // Trailing comment after the quoted name; take the quoted part.
                let bytes = value.as_bytes();
                if let Some(end) = bytes[1..].iter().position(|b| *b == quote) {
                    value = &value[1..1 + end];
                }
            }
        }
        names.push(value.to_string());
    }
    names
}

/// CODEX-CR02-001: true when a Rust-owned cmd directory must not ship
/// through the cmd identity — its manifest declares `[[bin]]` entries but
/// none of them is the command's own directory name. Such a crate ships
/// its real binaries through RUST_TOOLS tuples instead. Manifests with no
/// `[[bin]]` section keep the previous classification.
fn ships_no_command_bin(cmd_dir: &str, name: &str) -> bool {
    let bins = declared_bins(cmd_dir);
    !bins.is_empty() && !bins.iter().any(|bin| bin == name)
}

/// Packaging ownership: crates whose binaries RUST_TOOLS installs keep
/// their established destinations; shared discovery must not also produce
/// them as appliance commands. Kept in sync with
/// `build_compile::RUST_TOOLS` by `toolsown_members_match_rust_tools`.
const RUST_TOOLS_MEMBERS: &[&str] = &[
    "soda-acceptance",
    "soda-activate",
    "soda-console-welcome",
    "soda-forgejo-domain",
    "soda-forgejo-migrate",
    "soda-install",
    "soda-pg-maintenance",
    "soda-project-terminal",
];

fn is_tools_owned(name: &str) -> bool {
    RUST_TOOLS_MEMBERS.contains(&name)
}

/// N07-T3: true when a cmd directory still carries Go sources. Unreadable
/// directories report true so discovery keeps its previous listing; only a
/// readable directory with no `.go` files counts as retired.
fn has_go_sources(cmd_dir: &str) -> bool {
    fs::read_dir(cmd_dir)
        .map(|entries| {
            entries.filter_map(|entry| entry.ok()).any(|entry| {
                entry
                    .file_type()
                    .map(|kind| kind.is_file())
                    .unwrap_or(false)
                    && entry.file_name().to_string_lossy().ends_with(".go")
            })
        })
        .unwrap_or(true)
}

/// `build.SodaCommands`: sorted `cmd/soda-*` directories, tools excluded.
pub fn soda_commands(source: &str) -> Result<Vec<String>, Error> {
    let mut names = Vec::new();
    let entries = fs::read_dir(join(&[source, "cmd"]))?;
    let mut dirs: Vec<String> = entries
        .filter_map(|entry| entry.ok())
        .filter(|entry| entry.file_type().map(|t| t.is_dir()).unwrap_or(false))
        .map(|entry| entry.file_name().to_string_lossy().into_owned())
        .collect();
    dirs.sort();
    for name in dirs {
        // CODEX-CR02-001 (subsumes CODEX-A01-CMD-1): a Rust-owned cmd
        // directory ships as an appliance command only when its manifest
        // declares a binary of its own directory name. Multi-binary/lib
        // crates (soda-project-terminal, soda-pg-maintenance) ship their
        // real binaries via RUST_TOOLS; skip them here (not error) so
        // compile/link never reference the bogus identity.
        if is_rust_command(&join(&[source, "cmd", &name]))
            && ships_no_command_bin(&join(&[source, "cmd", &name]), &name)
        {
            continue;
        }
        // Packaging ownership: a RUST_TOOLS member crate keeps its
        // established install destinations; skip it here (not error) so
        // compile/link produce each binary exactly once.
        if is_rust_command(&join(&[source, "cmd", &name])) && is_tools_owned(&name) {
            continue;
        }
        // N07-T3: cmd/soda-forgejo-tailnet is Rust-built from the soda-host
        // package once its Go sources retire; skip it here (not error) so
        // discovery never attempts a Go build of a sourceless directory.
        // Pre-retirement trees (Go sources present) list it exactly as before.
        if name == "soda-forgejo-tailnet" && !has_go_sources(&join(&[source, "cmd", &name])) {
            continue;
        }
        if !is_soda_command(&name) || name == "soda-artifacts" || name == "soda-acceptance" {
            return Err(Error::msg(
                "support tools must remain outside appliance commands",
            ));
        }
        names.push(name);
    }
    if names.is_empty() {
        return Err(Error::msg("missing Soda commands"));
    }
    Ok(names)
}

fn is_soda_command(name: &str) -> bool {
    let rest = match name.strip_prefix("soda-") {
        Some(rest) => rest,
        None => return false,
    };
    !rest.is_empty()
        && rest
            .bytes()
            .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'-')
}

/// Mount mode bits for a freshly created directory (`os.MkdirAll` parity).
pub fn mkdir_all(path: &str, mode: u32) -> Result<(), Error> {
    fs::create_dir_all(path)?;
    fs::set_permissions(path, fs::Permissions::from_mode(mode))?;
    Ok(())
}

pub fn create_dir(path: &str, mode: u32) -> Result<(), Error> {
    fs::create_dir(path).map_err(|e| Error::msg(e.to_string()))?;
    fs::set_permissions(path, fs::Permissions::from_mode(mode))?;
    Ok(())
}

/// Recursive directory walk yielding (path, is_dir, is_symlink).
pub fn walk<F>(root: &str, mut visit: F) -> Result<(), Error>
where
    F: FnMut(&str, bool, bool) -> Result<(), Error>,
{
    let mut stack = vec![root.to_string()];
    while let Some(path) = stack.pop() {
        let meta = fs::symlink_metadata(&path)?;
        let is_link = meta.file_type().is_symlink();
        let is_dir = meta.file_type().is_dir();
        visit(&path, is_dir, is_link)?;
        if is_dir && !is_link {
            let mut entries: Vec<String> = fs::read_dir(&path)?
                .filter_map(|entry| entry.ok())
                .map(|entry| entry.path().to_string_lossy().into_owned())
                .collect();
            entries.sort();
            entries.reverse();
            stack.extend(entries);
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_clean_path_matches_go() {
        // Oracle: Go filepath.Clean vectors.
        assert_eq!(clean_path(""), ".");
        assert_eq!(clean_path("/a//b/./c/../d"), "/a/b/d");
        assert_eq!(clean_path("a/../../b"), "../b");
        assert_eq!(join(&["/a", "b", "c"]), "/a/b/c");
        assert_eq!(dir_name("/a/b"), "/a");
        assert_eq!(base_name("/a/b"), "b");
        assert_eq!(rel_path("/a/b", "/a/b/c/d").unwrap(), "c/d");
        assert_eq!(rel_path("/a/b/c", "/a/d").unwrap(), "../../d");
    }

    #[test]
    fn oracle_hash_file_refuses_symlink() {
        let dir = std::env::temp_dir().join(format!("sri-sys-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        fs::create_dir_all(&dir).unwrap();
        let target = dir.join("f");
        fs::write(&target, b"bytes").unwrap();
        let link = dir.join("l");
        std::os::unix::fs::symlink(&target, &link).unwrap();
        assert_eq!(
            hash_file(target.to_str().unwrap()).unwrap(),
            hex_sha256(b"bytes")
        );
        assert_eq!(
            hash_file(link.to_str().unwrap()).unwrap_err().0,
            "regular non-symlink file required"
        );
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn rust_command_follows_manifest_presence() {
        // CORR-C-001: top-level Cargo.toml means Rust-owned; anything
        // else (Go sources, empty, missing) stays on the Go recipe.
        let dir = std::env::temp_dir().join(format!("sri-own-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let rust_cmd = dir.join("soda-rust");
        let go_cmd = dir.join("soda-go");
        fs::create_dir_all(&rust_cmd).unwrap();
        fs::create_dir_all(&go_cmd).unwrap();
        fs::write(rust_cmd.join("Cargo.toml"), b"[package]\n").unwrap();
        fs::write(go_cmd.join("main.go"), b"package main\n").unwrap();
        assert!(is_rust_command(rust_cmd.to_str().unwrap()));
        assert!(!is_rust_command(go_cmd.to_str().unwrap()));
        assert!(!is_rust_command(dir.join("soda-missing").to_str().unwrap()));
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn cmd1_discovery_skips_folded_terminal_crate() {
        // CORR-C-004-AMEND-1 (CODEX-A01-CMD-1): post-A01 fold, cmd/
        // soda-project-terminal carries a manifest, but its package ships
        // project-terminal + project-account — never a soda-project-terminal
        // binary. Discovery must skip it (RUST_TOOLS owns those bins) while
        // real commands still list.
        let dir = std::env::temp_dir().join(format!("sri-cmd1a-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let snapshot = dir.join("snap");
        fs::create_dir_all(snapshot.join("cmd/soda-fakego")).unwrap();
        let folded = snapshot.join("cmd/soda-project-terminal");
        fs::create_dir_all(&folded).unwrap();
        fs::write(
            folded.join("Cargo.toml"),
            b"[package]\nname = \"soda-project-terminal\"\n[[bin]]\nname = \"project-terminal\"\n[[bin]]\nname = \"project-account\"\n",
        )
        .unwrap();
        let names = soda_commands(snapshot.to_str().unwrap()).unwrap();
        assert_eq!(names, vec!["soda-fakego".to_string()]);
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn cr02_discovery_skips_multibin_pg_crate() {
        // CODEX-CR02-001: cmd/soda-pg-maintenance is Rust-owned but declares
        // only backup/restore/init-roles bins — never a soda-pg-maintenance
        // binary. Discovery must skip it (RUST_TOOLS owns those bins) while
        // a same-name-bin crate and real Go commands still list.
        let dir = std::env::temp_dir().join(format!("sri-cr02a-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let snapshot = dir.join("snap");
        fs::create_dir_all(snapshot.join("cmd/soda-fakego")).unwrap();
        let identity = snapshot.join("cmd/soda-identity");
        fs::create_dir_all(&identity).unwrap();
        fs::write(
            identity.join("Cargo.toml"),
            b"[package]\nname = \"soda-identity\"\n[[bin]]\nname = \"soda-identity\"\n",
        )
        .unwrap();
        let pg = snapshot.join("cmd/soda-pg-maintenance");
        fs::create_dir_all(&pg).unwrap();
        fs::write(
            pg.join("Cargo.toml"),
            b"[package]\nname = \"soda-pg-maintenance\"\n[lib]\nname = \"soda_pg_maintenance\"\n[[bin]]\nname = \"soda-pg-backup\"\n[[bin]]\nname = \"soda-pg-restore\"\n[[bin]]\nname = \"soda-pg-init-roles\"\n",
        )
        .unwrap();
        let names = soda_commands(snapshot.to_str().unwrap()).unwrap();
        assert_eq!(
            names,
            vec!["soda-fakego".to_string(), "soda-identity".to_string()]
        );
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn toolsown_members_match_rust_tools() {
        // The discovery ownership list must name exactly the RUST_TOOLS
        // member crates: port PRs extend the table, and discovery skips
        // every member so each binary is produced exactly once.
        let mut members: Vec<&str> = crate::build_compile::RUST_TOOLS
            .iter()
            .map(|(m, _, _)| *m)
            .collect();
        members.sort();
        members.dedup();
        let mut listed: Vec<&str> = RUST_TOOLS_MEMBERS.to_vec();
        listed.sort();
        assert_eq!(listed, members);
    }

    #[test]
    fn toolsown_discovery_skips_tools_members() {
        // A RUST_TOOLS member with a same-name binary (e.g. activate) is
        // still skipped: RUST_TOOLS owns its established destinations. A
        // same-name crate outside the table and Go commands still list.
        let dir = std::env::temp_dir().join(format!("sri-toolsown-a-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let snapshot = dir.join("snap");
        fs::create_dir_all(snapshot.join("cmd/soda-fakego")).unwrap();
        let owned = snapshot.join("cmd/soda-activate");
        fs::create_dir_all(&owned).unwrap();
        fs::write(
            owned.join("Cargo.toml"),
            b"[package]\nname = \"soda-activate\"\n[[bin]]\nname = \"soda-activate\"\n",
        )
        .unwrap();
        let kept = snapshot.join("cmd/soda-identity");
        fs::create_dir_all(&kept).unwrap();
        fs::write(
            kept.join("Cargo.toml"),
            b"[package]\nname = \"soda-identity\"\n[[bin]]\nname = \"soda-identity\"\n",
        )
        .unwrap();
        let names = soda_commands(snapshot.to_str().unwrap()).unwrap();
        assert_eq!(
            names,
            vec!["soda-fakego".to_string(), "soda-identity".to_string()]
        );
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn n07t3_discovery_skips_tailnet_only_after_go_retirement() {
        // N07-T3: cmd/soda-forgejo-tailnet has no manifest of its own; it
        // leaves Go discovery only once its Go sources retire. A retired
        // directory (Rust sources only) is skipped so no Go build is
        // attempted, while a pre-retirement directory (main.go present)
        // lists exactly as before.
        let dir = std::env::temp_dir().join(format!("sri-n07t3b-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        let snapshot = dir.join("snap");
        fs::create_dir_all(snapshot.join("cmd/soda-fakego")).unwrap();
        let tailnet = snapshot.join("cmd/soda-forgejo-tailnet");
        fs::create_dir_all(&tailnet).unwrap();
        fs::write(tailnet.join("main.go"), b"package main\n").unwrap();
        fs::write(tailnet.join("main.rs"), b"fn main() {}\n").unwrap();
        let names = soda_commands(snapshot.to_str().unwrap()).unwrap();
        assert_eq!(
            names,
            vec![
                "soda-fakego".to_string(),
                "soda-forgejo-tailnet".to_string()
            ]
        );
        fs::remove_file(tailnet.join("main.go")).unwrap();
        let names = soda_commands(snapshot.to_str().unwrap()).unwrap();
        assert_eq!(names, vec!["soda-fakego".to_string()]);
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn require_native_accepts_matching_x86_64_linux() {
        // D01-F1: Rust target arch and OCI arch are separate namespaces;
        // matching-native x86_64 Linux must pass, anything else must fail.
        assert!(require_native("x86_64").is_ok());
        assert!(require_native("amd64").is_err());
        assert!(require_native("arm64").is_err());
        assert!(require_native("").is_err());
    }

    #[test]
    fn oracle_write_new_refuses_overwrite() {
        let dir = std::env::temp_dir().join(format!("sri-wn-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        fs::create_dir_all(&dir).unwrap();
        let path = dir.join("n");
        write_new(path.to_str().unwrap(), b"one", 0o600).unwrap();
        assert!(write_new(path.to_str().unwrap(), b"two", 0o600).is_err());
        assert_eq!(fs::read(&path).unwrap(), b"one");
        let _ = fs::remove_dir_all(&dir);
    }
}

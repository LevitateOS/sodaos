use std::fs::File;
use std::io::{Read, Write};

use soda_json::JsonValue;

use crate::error::Error;
use crate::files::{self, OwnedDir};
use crate::jsonio;

use super::RedactOut;
use super::{
    contains_slice, longest_secret, redact_urls, Evidence, RedactingWriter, EVIDENCE_LIMIT,
};

/// Secret-scan block size.
const SCAN_BLOCK: usize = 32_768;

/// Create a fresh private evidence directory that must not exist yet.
/// Mirrors `CreateEvidence`, including the JSON-escaped secret variants.
pub fn create_evidence(path: &str, secrets: &[Vec<u8>]) -> Result<Evidence, Error> {
    if !path.starts_with('/') {
        return Err(Error::msg("absolute fresh evidence directory required"));
    }
    let parent = std::path::Path::new(path)
        .parent()
        .unwrap_or(std::path::Path::new("/"));
    let parent_text = parent.to_string_lossy();
    let resolved = std::fs::canonicalize(parent)?;
    if resolved.to_string_lossy() != files::lexical_clean(&parent_text) {
        return Err(Error::msg("evidence parent must not contain symlinks"));
    }
    let raw_path = std::ffi::CString::new(path)
        .map_err(|_| Error::msg("absolute fresh evidence directory required"))?;
    let rc = unsafe { libc::mkdir(raw_path.as_ptr(), 0o700) };
    if rc != 0 {
        return Err(Error::from(std::io::Error::last_os_error()));
    }
    let root = OwnedDir::open(path)?;
    let mut kept: Vec<Vec<u8>> = Vec::new();
    for secret in secrets {
        if secret.is_empty() {
            continue;
        }
        kept.push(secret.clone());
        // Logs may contain JSON-escaped credentials rather than raw values.
        let mut encoded = String::new();
        crate::jsonio::escape_go(&mut encoded, &String::from_utf8_lossy(secret));
        let inner = &encoded[1..encoded.len() - 1];
        if inner.as_bytes() != secret.as_slice() {
            kept.push(inner.as_bytes().to_vec());
        }
    }
    kept.sort_by_key(|a| std::cmp::Reverse(a.len()));
    Ok(Evidence {
        root,
        secrets: kept,
    })
}

fn valid_evidence_name(name: &str) -> bool {
    !name.starts_with('/')
        && files::lexical_clean(name) == name
        && name != "."
        && name != ".."
        && !name.starts_with("../")
}

impl Evidence {
    /// Original root path, for display and disjointness checks.
    pub fn path(&self) -> &str {
        self.root.path()
    }

    /// Secret set, longest first. For command capture wiring.
    pub fn secrets(&self) -> &[Vec<u8>] {
        &self.secrets
    }

    fn mkdir_evidence_parent(&self, current: &str) -> Result<(), Error> {
        match self.root.mkdir_at(current, 0o700) {
            Ok(()) => {}
            Err(e) if e.io_kind() == Some(std::io::ErrorKind::AlreadyExists) => {}
            Err(e) => return Err(e),
        }
        let attr = self.root.lstat_at(current)?;
        if !attr.is_dir || attr.is_symlink {
            return Err(Error::msg("unsafe evidence parent"));
        }
        Ok(())
    }

    fn ensure_evidence_parents(&self, dir: &str) -> Result<(), Error> {
        if dir == "." {
            return Ok(());
        }
        let mut current = String::new();
        for part in dir.split('/') {
            if !current.is_empty() {
                current.push('/');
            }
            current.push_str(part);
            self.mkdir_evidence_parent(&current)?;
        }
        Ok(())
    }

    fn open(&self, name: &str) -> Result<File, Error> {
        if !valid_evidence_name(name) {
            return Err(Error::msg("invalid evidence name"));
        }
        let dir = match name.rsplit_once('/') {
            Some((parent, _)) => parent,
            None => ".",
        };
        self.ensure_evidence_parents(dir)?;
        self.root.create_new_at(name, 0o600)
    }

    /// Exclusive redacting stream for one evidence entry.
    pub fn writer(&self, name: &str) -> Result<RedactingWriter, Error> {
        Ok(RedactingWriter::new(
            RedactOut::file(self.open(name)?),
            self.secrets.clone(),
        ))
    }

    /// Exclusive raw file for command capture tees.
    pub(crate) fn open_file(&self, name: &str) -> Result<File, Error> {
        self.open(name)
    }

    /// Write one complete redacted entry.
    pub fn write(&self, name: &str, data: &[u8]) -> Result<(), Error> {
        let mut writer = self.writer(name)?;
        // Like Go, the entry always closes so retained failures join.
        let write_err = writer.write_all(data).err().map(Error::from);
        let close_err = writer.close().err();
        match Error::join(vec![write_err, close_err]) {
            Some(err) => Err(err),
            None => Ok(()),
        }
    }

    /// Scrub a decoded value: strings redacted, keys redacted with
    /// collision detection, numbers/booleans/null preserved. Mirrors
    /// `scrubJSON` (decoded with number identity, like `UseNumber`).
    pub fn scrub_json(&self, value: &JsonValue) -> Result<JsonValue, Error> {
        match value {
            JsonValue::Str(s) => Ok(JsonValue::Str(self.redact_string(s))),
            JsonValue::Array(items) => {
                let mut out = Vec::with_capacity(items.len());
                for item in items {
                    out.push(self.scrub_json(item)?);
                }
                Ok(JsonValue::Array(out))
            }
            JsonValue::Object(entries) => {
                let mut out = Vec::with_capacity(entries.len());
                for (key, item) in entries {
                    let scrubbed_key = self.redact_string(key);
                    if out.iter().any(|(k, _)| k == &scrubbed_key) {
                        return Err(Error::msg("redacted JSON key collision"));
                    }
                    out.push((scrubbed_key, self.scrub_json(item)?));
                }
                Ok(JsonValue::Object(out))
            }
            _ => Ok(value.clone()),
        }
    }

    /// Confined evidence root for hash walks over retained files.
    pub fn root(&self) -> &OwnedDir {
        &self.root
    }

    fn encode_scrubbed_json(&self, value: &JsonValue) -> Result<Vec<u8>, Error> {
        let mut compact = String::new();
        jsonio::write_compact(&mut compact, value);
        if compact.len() as u64 > EVIDENCE_LIMIT {
            return Err(Error::msg("structured evidence limit exceeded"));
        }
        let decoded = JsonValue::parse(&compact)
            .map_err(|_| Error::msg("structured evidence limit exceeded"))?;
        let scrubbed = self.scrub_json(&decoded)?;
        // Go re-encodes a decoded map, so keys render sorted.
        let sorted = sort_json_keys(scrubbed);
        let mut data = String::new();
        jsonio::write_indent(&mut data, &sorted);
        data.push('\n');
        if data.len() as u64 > EVIDENCE_LIMIT {
            return Err(Error::msg("structured evidence limit exceeded"));
        }
        Ok(data.into_bytes())
    }

    /// Write one structured value: scrubbed before encoding, then scanned
    /// for secrets before the bytes are retained. Mirrors `WriteJSON`.
    pub fn write_json(&self, name: &str, value: &JsonValue) -> Result<(), Error> {
        let data = self.encode_scrubbed_json(value)?;
        for secret in &self.secrets {
            if contains_slice(&data, secret) {
                return Err(Error::msg("secret reached structured evidence"));
            }
        }
        let file = self.open(name)?;
        write_and_sync(file, &data)
    }

    /// Finalize one observation: pending record first, leak scan, then an
    /// exclusive link to the final name. Mirrors `PublishObservation`.
    pub fn publish_observation(&self, observation: &JsonValue) -> Result<(), Error> {
        self.write_json("observation.pending.json", observation)?;
        self.check_secrets()?;
        self.root
            .link_at("observation.pending.json", "observation.json")?;
        Ok(())
    }

    /// Re-scan every retained byte for known secrets. Defense in depth,
    /// not a claim that unknown secrets are absent.
    pub fn check_secrets(&self) -> Result<(), Error> {
        self.root.walk_files(&mut |rel: &str, regular: bool| {
            if !regular {
                return Err(Error::msg("unexpected non-regular evidence entry"));
            }
            self.scan_regular_evidence(rel)
        })
    }

    fn scan_regular_evidence(&self, name: &str) -> Result<(), Error> {
        let file = self.root.open_file_at(name)?;
        let attr = fstat_attr(&file)?;
        if !attr.is_regular {
            return Err(Error::msg("evidence entry changed to a special file"));
        }
        scan_evidence_bytes(file, &self.secrets)
    }

    /// Redact known secrets plus URL queries/fragments/userinfo.
    pub fn redact_string(&self, text: &str) -> String {
        let mut out = text.to_string();
        for secret in &self.secrets {
            if let Ok(pattern) = std::str::from_utf8(secret) {
                out = out.replace(pattern, "[REDACTED]");
            }
        }
        redact_urls(&out)
    }

    /// Scrubbed error keeping the cause chain, like Go's `safeError`.
    pub fn redact_error(&self, err: Error) -> Error {
        Error::redacted(self.redact_string(&err.to_string()), err)
    }
}

/// Recursively sort object keys, like Go's map encoding.
fn sort_json_keys(value: JsonValue) -> JsonValue {
    match value {
        JsonValue::Object(entries) => {
            let mut entries: Vec<(String, JsonValue)> = entries
                .into_iter()
                .map(|(k, v)| (k, sort_json_keys(v)))
                .collect();
            entries.sort_by(|a, b| a.0.cmp(&b.0));
            // Duplicate keys keep last-wins, like Go's map decode.
            let mut unique: Vec<(String, JsonValue)> = Vec::with_capacity(entries.len());
            for (key, value) in entries.into_iter().rev() {
                if !unique.iter().any(|(k, _)| k == &key) {
                    unique.push((key, value));
                }
            }
            unique.reverse();
            JsonValue::Object(unique)
        }
        JsonValue::Array(items) => {
            JsonValue::Array(items.into_iter().map(sort_json_keys).collect())
        }
        other => other,
    }
}

fn write_and_sync(mut file: File, data: &[u8]) -> Result<(), Error> {
    // Go joins write, sync, and close errors; the file closes on drop,
    // so only the first two are observable here.
    let write_err = file.write_all(data).err().map(Error::from);
    let sync_err = file.sync_all().err().map(Error::from);
    drop(file);
    match Error::join(vec![write_err, sync_err]) {
        Some(err) => Err(err),
        None => Ok(()),
    }
}

fn fstat_attr(file: &File) -> Result<files::FileAttr, Error> {
    use std::os::fd::AsRawFd;
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    let rc = unsafe { libc::fstat(file.as_raw_fd(), &mut st) };
    if rc != 0 {
        return Err(Error::from(std::io::Error::last_os_error()));
    }
    let mode = st.st_mode;
    Ok(files::FileAttr {
        is_regular: mode & libc::S_IFMT == libc::S_IFREG,
        is_dir: mode & libc::S_IFMT == libc::S_IFDIR,
        is_symlink: mode & libc::S_IFMT == libc::S_IFLNK,
        perm: mode & 0o7777,
        size: st.st_size.max(0) as u64,
        uid: st.st_uid,
        gid: st.st_gid,
        dev: st.st_dev as u64,
        ino: st.st_ino as u64,
    })
}

fn scan_evidence_bytes(mut file: File, secrets: &[Vec<u8>]) -> Result<(), Error> {
    let max = longest_secret(secrets);
    let mut tail: Vec<u8> = Vec::new();
    let mut buf = [0u8; SCAN_BLOCK];
    loop {
        let n = file.read(&mut buf)?;
        let mut block = tail.clone();
        block.extend_from_slice(&buf[..n]);
        for secret in secrets {
            if contains_slice(&block, secret) {
                return Err(Error::msg("secret reached evidence"));
            }
        }
        tail = if block.len() >= max {
            block[block.len() - max + 1..].to_vec()
        } else {
            block
        };
        if n == 0 {
            return Ok(());
        }
    }
}

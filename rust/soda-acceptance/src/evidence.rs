//! Private exclusive evidence with streaming redaction, mirroring
//! `internal/acceptance/evidence.go`.
//!
//! An [`Evidence`] root is a fresh private directory held open; every entry
//! is created exclusively and never overwritten. Streamed bytes pass through
//! [`RedactingWriter`], which withholds trailing matches across writes so a
//! split secret never reaches the file, then strips URL queries/fragments.
//! Structured values are scrubbed before encoding, and
//! [`Evidence::check_secrets`] re-scans every retained byte as defense in
//! depth.

use std::fs::File;
use std::io::{Read, Write};

use soda_json::JsonValue;

use crate::error::Error;
use crate::files::{self, OwnedDir};
use crate::jsonio;

/// Structured/streamed evidence size bound: 16 MiB of input bytes.
pub const EVIDENCE_LIMIT: u64 = 16 << 20;
/// Secret-scan block size.
const SCAN_BLOCK: usize = 32_768;

/// Private exclusive evidence root with a redaction secret set.
pub struct Evidence {
    root: OwnedDir,
    secrets: Vec<Vec<u8>>,
}

/// Create a fresh private evidence directory that must not exist yet.
/// Mirrors `CreateEvidence`, including the JSON-escaped secret variants.
pub fn create_evidence(path: &str, secrets: &[Vec<u8>]) -> Result<Evidence, Error> {
    if !path.starts_with('/') {
        return Err(Error::msg("absolute fresh evidence directory required"));
    }
    let parent = std::path::Path::new(path).parent().unwrap_or(std::path::Path::new("/"));
    let parent_text = parent.to_string_lossy();
    let resolved = std::fs::canonicalize(parent)?;
    if resolved.to_string_lossy() != files::lexical_clean(&parent_text) {
        return Err(Error::msg("evidence parent must not contain symlinks"));
    }
    let raw_path = std::ffi::CString::new(path).map_err(|_| Error::msg("absolute fresh evidence directory required"))?;
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
    kept.sort_by(|a, b| b.len().cmp(&a.len()));
    Ok(Evidence { root, secrets: kept })
}

fn valid_evidence_name(name: &str) -> bool {
    !name.starts_with('/') && files::lexical_clean(name) == name && name != "." && name != ".." && !name.starts_with("../")
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
        Ok(RedactingWriter::new(RedactOut::file(self.open(name)?), self.secrets.clone()))
    }

    /// Write one complete redacted entry.
    pub fn write(&self, name: &str, data: &[u8]) -> Result<(), Error> {
        let mut writer = self.writer(name)?;
        writer.write_all(data).map_err(Error::from)?;
        writer.close()
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

    fn encode_scrubbed_json(&self, value: &JsonValue) -> Result<Vec<u8>, Error> {
        let mut compact = String::new();
        jsonio::write_compact(&mut compact, value);
        if compact.len() as u64 > EVIDENCE_LIMIT {
            return Err(Error::msg("structured evidence limit exceeded"));
        }
        let decoded = JsonValue::parse(&compact).map_err(|_| Error::msg("structured evidence limit exceeded"))?;
        let scrubbed = self.scrub_json(&decoded)?;
        let mut data = String::new();
        jsonio::write_indent(&mut data, &scrubbed);
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
        self.root.link_at("observation.pending.json", "observation.json")?;
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

fn write_and_sync(mut file: File, data: &[u8]) -> Result<(), Error> {
    let write_err = file.write_all(data).err().map(Error::from);
    let sync_err = file.sync_all().err().map(Error::from);
    drop(file);
    if let Some(e) = write_err {
        return Err(e);
    }
    if let Some(e) = sync_err {
        return Err(e);
    }
    Ok(())
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

fn contains_slice(haystack: &[u8], needle: &[u8]) -> bool {
    if needle.is_empty() || needle.len() > haystack.len() {
        return false;
    }
    haystack.windows(needle.len()).any(|w| w == needle)
}

fn longest_secret(secrets: &[Vec<u8>]) -> usize {
    secrets.iter().map(|s| s.len()).max().unwrap_or(0).max(1)
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

/// Strip userinfo, queries, and fragments from `http(s)` URLs, matching
/// Go's `redactURLs`. Malformed URLs keep a sanitized form with those parts
/// still stripped; only the whole-URL omission differs, and only when Go's
/// parser rejects the URL outright.
pub fn redact_urls(text: &str) -> String {
    let bytes = text.as_bytes();
    let mut out = String::with_capacity(text.len());
    let mut i = 0;
    while i < bytes.len() {
        let rest = &text[i..];
        let scheme = if rest.starts_with("http://") {
            Some("http://")
        } else if rest.starts_with("https://") {
            Some("https://")
        } else {
            None
        };
        if let Some(prefix) = scheme {
            let mut end = i + prefix.len();
            while end < bytes.len() && !is_url_break(bytes[end]) {
                end += 1;
            }
            out.push_str(&sanitize_url(prefix, &text[i + prefix.len()..end]));
            i = end;
        } else {
            let ch = text[i..].chars().next().unwrap_or('\u{FFFD}');
            out.push(ch);
            i += ch.len_utf8();
        }
    }
    out
}

fn is_url_break(byte: u8) -> bool {
    matches!(byte, b' ' | b'\t' | b'\n' | b'\r' | b'\x0b' | b'\x0c' | b'"' | b'\'' | b'<' | b'>' | b'\\')
}

fn sanitize_url(prefix: &str, rest: &str) -> String {
    if rest.bytes().any(|b| b < 0x20) || has_bad_escape(rest) || has_bad_brackets(rest) {
        return "[URL OMITTED]".to_string();
    }
    let (authority, path) = match rest.find('/') {
        Some(index) => (&rest[..index], &rest[index..]),
        None => (rest, ""),
    };
    let host = match authority.rfind('@') {
        Some(index) => &authority[index + 1..],
        None => authority,
    };
    let clean_path = path.split(['?', '#']).next().unwrap_or("");
    format!("{prefix}{host}{clean_path}")
}

fn has_bad_escape(text: &str) -> bool {
    let bytes = text.as_bytes();
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] == b'%' {
            if i + 2 >= bytes.len() || !bytes[i + 1].is_ascii_hexdigit() || !bytes[i + 2].is_ascii_hexdigit() {
                return true;
            }
            i += 3;
        } else {
            i += 1;
        }
    }
    false
}

fn has_bad_brackets(text: &str) -> bool {
    let authority = text.split('/').next().unwrap_or("");
    if let Some(open) = authority.find('[') {
        return !authority[open..].contains(']');
    }
    authority.contains(']')
}

/// Redaction sink: a lone file, or a file teed with a captured buffer for
/// command results. Both sides receive redacted bytes only.
enum RedactOut {
    File(File),
    Tee(File, Vec<u8>),
    Discard,
}

impl RedactOut {
    fn file(file: File) -> RedactOut {
        RedactOut::File(file)
    }
}

/// Streaming secret/URL redactor with split-match withholding. Mirrors
/// `redactingWriter`, including the 16 MiB input bound and the
/// line-buffered URL pass.
pub struct RedactingWriter {
    out: RedactOut,
    secrets: Vec<Vec<u8>>,
    pending: Vec<u8>,
    url_pending: Vec<u8>,
    count: u64,
    limit: Option<u64>,
    closed: bool,
    err: Option<String>,
}

impl RedactingWriter {
    fn new(out: RedactOut, secrets: Vec<Vec<u8>>) -> RedactingWriter {
        RedactingWriter {
            out,
            secrets,
            pending: Vec::new(),
            url_pending: Vec::new(),
            count: 0,
            limit: Some(EVIDENCE_LIMIT),
            closed: false,
            err: None,
        }
    }

    /// Unbounded secret-free sink for directly started test processes.
    pub fn discard() -> RedactingWriter {
        RedactingWriter {
            out: RedactOut::Discard,
            secrets: Vec::new(),
            pending: Vec::new(),
            url_pending: Vec::new(),
            count: 0,
            limit: None,
            closed: false,
            err: None,
        }
    }

    /// File teed with a captured buffer, for command results.
    pub fn tee(file: File, secrets: Vec<Vec<u8>>) -> RedactingWriter {
        RedactingWriter::new(RedactOut::Tee(file, Vec::new()), secrets)
    }

    /// Captured buffer, for tee writers.
    pub fn buffer(&self) -> &[u8] {
        match &self.out {
            RedactOut::Tee(_, buf) => buf,
            _ => &[],
        }
    }

    /// Take the captured buffer, for tee writers.
    pub fn into_buffer(self) -> Vec<u8> {
        match self.out {
            RedactOut::Tee(_, buf) => buf,
            _ => Vec::new(),
        }
    }

    fn emit(&mut self, bytes: &[u8]) -> Result<(), Error> {
        match &mut self.out {
            RedactOut::File(file) => file.write_all(bytes)?,
            RedactOut::Tee(file, buf) => {
                file.write_all(bytes)?;
                buf.extend_from_slice(bytes);
            }
            RedactOut::Discard => {}
        }
        Ok(())
    }

    fn flush(&mut self, final_flush: bool) -> Result<(), Error> {
        let max = longest_secret(&self.secrets);
        let mut produced: Vec<u8> = Vec::new();
        let pending = std::mem::take(&mut self.pending);
        let mut consumed = 0;
        while consumed < pending.len() && (final_flush || pending.len() - consumed >= max) {
            let window = &pending[consumed..];
            let mut match_len = 0;
            for secret in &self.secrets {
                if secret.len() > match_len && window.starts_with(secret) {
                    match_len = secret.len();
                }
            }
            if match_len > 0 {
                produced.extend_from_slice(b"[REDACTED]");
                consumed += match_len;
            } else {
                produced.push(window[0]);
                consumed += 1;
            }
        }
        self.pending = pending[consumed..].to_vec();
        self.url_pending.extend_from_slice(&produced);
        let end = if final_flush {
            self.url_pending.len()
        } else {
            self.url_pending.iter().rposition(|b| *b == b'\n').map(|i| i + 1).unwrap_or(0)
        };
        if end == 0 {
            return Ok(());
        }
        let chunk = String::from_utf8_lossy(&self.url_pending[..end]).into_owned();
        let rest = self.url_pending[end..].to_vec();
        self.url_pending = rest;
        let safe = redact_urls(&chunk);
        self.emit(safe.as_bytes())
    }

    /// Write bytes, retaining the first failure like the Go owner.
    pub fn write_bytes(&mut self, data: &[u8]) -> Result<usize, Error> {
        if self.closed {
            return Err(Error::msg("file already closed"));
        }
        if let Some(message) = &self.err {
            return Err(Error::msg(message.clone()));
        }
        if let Some(limit) = self.limit {
            if self.count + data.len() as u64 > limit {
                self.err = Some("evidence output limit exceeded".to_string());
                return Err(Error::msg("evidence output limit exceeded"));
            }
        }
        self.count += data.len() as u64;
        self.pending.extend_from_slice(data);
        if let Err(e) = self.flush(false) {
            self.err = Some(e.to_string());
            return Err(Error::msg(e.to_string()));
        }
        Ok(data.len())
    }

    /// Final flush and close, joining retained failures like the Go owner.
    pub fn close(&mut self) -> Result<(), Error> {
        if self.closed {
            return match &self.err {
                Some(message) => Err(Error::msg(message.clone())),
                None => Ok(()),
            };
        }
        self.closed = true;
        if let Err(e) = self.flush(true) {
            if self.err.is_none() {
                self.err = Some(e.to_string());
            }
        }
        match &self.err {
            Some(message) => Err(Error::msg(message.clone())),
            None => Ok(()),
        }
    }
}

impl Write for RedactingWriter {
    fn write(&mut self, data: &[u8]) -> std::io::Result<usize> {
        self.write_bytes(data).map_err(std::io::Error::other)
    }

    fn flush(&mut self) -> std::io::Result<()> {
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::fs::PermissionsExt;

    struct Fixture {
        dir: std::path::PathBuf,
        evidence: Evidence,
    }

    impl Fixture {
        fn new(secrets: &[&str]) -> Fixture {
            let mut dir = std::env::temp_dir();
            dir.push(format!("soda-evidence-{}-{}", std::process::id(), fresh_id()));
            std::fs::create_dir_all(&dir).unwrap();
            let path = dir.join("evidence").to_string_lossy().into_owned();
            let owned: Vec<Vec<u8>> = secrets.iter().map(|s| s.as_bytes().to_vec()).collect();
            let evidence = create_evidence(&path, &owned).unwrap();
            Fixture { dir, evidence }
        }
    }

    impl Drop for Fixture {
        fn drop(&mut self) {
            let _ = std::fs::remove_dir_all(&self.dir);
        }
    }

    fn fresh_id() -> u64 {
        use std::sync::atomic::{AtomicU64, Ordering};
        static NEXT: AtomicU64 = AtomicU64::new(0);
        NEXT.fetch_add(1, Ordering::SeqCst)
    }

    #[test]
    fn split_secrets_and_redirect_queries() {
        let fixture = Fixture::new(&["synthetic-password"]);
        let e = &fixture.evidence;
        let mut writer = e.writer("out").unwrap();
        for part in [
            "synthetic-",
            "pass",
            "word\nhttps://example.test/callback?co",
            "de=unknown-code&state=unknown-state\n",
        ] {
            writer.write_bytes(part.as_bytes()).unwrap();
        }
        writer.close().unwrap();
        let text = std::fs::read_to_string(format!("{}/out", e.path())).unwrap();
        for bad in ["synthetic-password", "unknown-code", "unknown-state"] {
            assert!(!text.contains(bad), "retained unsafe output: {text:?}");
        }
        assert!(text.contains("https://example.test/callback"), "{text:?}");
        e.check_secrets().unwrap();
    }

    #[test]
    fn exclusive_and_confined() {
        let fixture = Fixture::new(&[]);
        let e = &fixture.evidence;
        e.write("once", b"first").unwrap();
        let err = e.write("once", b"second").unwrap_err();
        assert_eq!(err.io_kind(), Some(std::io::ErrorKind::AlreadyExists));
        for name in ["../escape", "/absolute", "a/../b", "."] {
            assert!(e.writer(name).is_err(), "accepted {name:?}");
        }
        let mut link_dir = std::env::temp_dir();
        link_dir.push(format!("soda-evidence-link-{}-{}", std::process::id(), fresh_id()));
        std::fs::create_dir_all(&link_dir).unwrap();
        std::os::unix::fs::symlink(&link_dir, format!("{}/link", e.path())).unwrap();
        assert!(e.writer("link/out").is_err());
        let root_mode = std::fs::metadata(e.path()).unwrap().permissions().mode() & 0o777;
        assert_eq!(root_mode, 0o700);
        let file_mode = std::fs::metadata(format!("{}/once", e.path())).unwrap().permissions().mode() & 0o777;
        assert_eq!(file_mode, 0o600);
        std::fs::remove_dir_all(&link_dir).unwrap();
    }

    #[test]
    fn redacted_error_retains_identity() {
        let fixture = Fixture::new(&["synthetic-password"]);
        let e = &fixture.evidence;
        let sentinel = Error::msg("underlying");
        let wrapped = Error::wrap("synthetic-password https://example.test/?code=hidden", sentinel);
        let redacted = e.redact_error(wrapped);
        assert!(!redacted.to_string().contains("synthetic-password"), "{redacted}");
        assert!(!redacted.to_string().contains("code="), "{redacted}");
        let mut found = false;
        let mut current: Option<&(dyn std::error::Error + 'static)> = std::error::Error::source(&redacted);
        while let Some(cause) = current {
            if cause.to_string() == "underlying" {
                found = true;
            }
            current = cause.source();
        }
        assert!(found, "sentinel lost: {redacted}");
    }

    #[test]
    fn output_bound() {
        let fixture = Fixture::new(&[]);
        let e = &fixture.evidence;
        let mut writer = e.writer("large").unwrap();
        let big = vec![0u8; (EVIDENCE_LIMIT + 1) as usize];
        assert!(writer.write_bytes(&big).is_err());
        assert!(writer.close().is_err());
    }

    #[test]
    fn structured_evidence_escapes_and_numeric_identity() {
        let secret = "synthetic-\"credential\\with\nnewline";
        let fixture = Fixture::new(&[secret]);
        let e = &fixture.evidence;
        let input = JsonValue::Object(vec![
            ("id".to_string(), JsonValue::Number("9223372036854775807".to_string())),
            ("secret".to_string(), JsonValue::Str(secret.to_string())),
            (
                "url".to_string(),
                JsonValue::Str("https://example.test/path?code=hidden\"".to_string()),
            ),
        ]);
        e.write_json("metadata.json", &input).unwrap();
        let raw = std::fs::read(format!("{}/metadata.json", e.path())).unwrap();
        assert!(!contains_slice(&raw, b"hidden"), "redirect query retained");
        let result = JsonValue::parse(std::str::from_utf8(&raw).unwrap()).unwrap();
        match result.get("id") {
            Some(JsonValue::Number(digits)) => assert_eq!(digits, "9223372036854775807"),
            other => panic!("integer identity changed: {other:?}"),
        }
        assert_eq!(result.get("secret").and_then(|v| v.as_str()), Some("[REDACTED]"));
        e.check_secrets().unwrap();
    }

    #[test]
    fn escaped_credentials_in_raw_split_writes() {
        let secret = "synthetic-\"credential\\line\nend";
        let fixture = Fixture::new(&[secret]);
        let e = &fixture.evidence;
        let mut encoded = String::new();
        crate::jsonio::escape_go(&mut encoded, secret);
        let inner = &encoded[1..encoded.len() - 1];
        let mut writer = e.writer("raw").unwrap();
        for byte in encoded.bytes() {
            writer.write_bytes(&[byte]).unwrap();
        }
        writer.close().unwrap();
        let raw = std::fs::read(format!("{}/raw", e.path())).unwrap();
        assert!(!contains_slice(&raw, inner.as_bytes()), "encoded secret not redacted");
        assert!(contains_slice(&raw, b"[REDACTED]"), "encoded secret not redacted");
    }

    #[test]
    fn finalization_does_not_publish_failed_or_occupied_attempts() {
        for mode in ["pending-collision", "leak", "final-collision"] {
            let fixture = Fixture::new(&["synthetic-private-marker"]);
            let e = &fixture.evidence;
            match mode {
                "pending-collision" => {
                    e.root.mkdir_at("observation.pending.json", 0o700).unwrap();
                }
                "leak" => {
                    let mut file = e.root.create_new_at("unredacted", 0o600).unwrap();
                    file.write_all(b"synthetic-private-marker").unwrap();
                }
                _ => {
                    let mut file = e.root.create_new_at("observation.json", 0o600).unwrap();
                    file.write_all(b"earlier bytes").unwrap();
                }
            }
            let observation = JsonValue::Object(vec![("Outcome".to_string(), JsonValue::Str("completed".to_string()))]);
            assert!(e.publish_observation(&observation).is_err(), "finalized failed attempt ({mode})");
            let raw = std::fs::read(format!("{}/observation.json", e.path()));
            if mode == "final-collision" {
                assert_eq!(raw.unwrap(), b"earlier bytes", "overwrote previous record");
            } else {
                assert!(raw.is_err(), "published success-shaped record ({mode})");
            }
        }
    }

    #[test]
    fn url_shapes_match_go() {
        assert_eq!(redact_urls("no url here"), "no url here");
        assert_eq!(
            redact_urls("see https://example.test/callback?code=x&state=y done"),
            "see https://example.test/callback done"
        );
        assert_eq!(
            redact_urls("http://user@example.test:8080/p#frag"),
            "http://example.test:8080/p"
        );
        assert_eq!(redact_urls("http://[::1]/x"), "http://[::1]/x");
        assert_eq!(redact_urls("http://[::1/x"), "[URL OMITTED]");
        assert_eq!(redact_urls("https://h/%zz"), "[URL OMITTED]");
        assert_eq!(redact_urls("https://h/a%20b?x=1"), "https://h/a%20b");
    }
}

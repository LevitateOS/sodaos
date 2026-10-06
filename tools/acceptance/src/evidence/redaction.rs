use std::fs::File;
use std::io::Write;

use crate::error::Error;

use super::{longest_secret, EVIDENCE_LIMIT};

/// Strip userinfo, queries, and fragments from `http(s)` URLs, matching
/// Go's `redactURLs`. Malformed URLs keep a sanitized form with those parts
/// still stripped; only the whole-URL omission differs, and only when Go's
/// parser rejects the URL outright.
pub fn redact_urls(text: &str) -> String {
    // Byte-level sanitizing only removes ASCII spans or emits an ASCII
    // literal, so valid UTF-8 input stays valid UTF-8.
    String::from_utf8(redact_urls_bytes(text.as_bytes())).expect("url redaction preserves UTF-8")
}

/// Byte-level URL redaction, so binary command output keeps its exact
/// retained bytes: only ASCII URL spans are rewritten, everything else
/// passes through untouched.
fn redact_urls_bytes(text: &[u8]) -> Vec<u8> {
    let mut out: Vec<u8> = Vec::with_capacity(text.len());
    let mut i = 0;
    while i < text.len() {
        let rest = &text[i..];
        let prefix: &[u8] = if rest.starts_with(b"http://") {
            b"http://"
        } else if rest.starts_with(b"https://") {
            b"https://"
        } else {
            out.push(text[i]);
            i += 1;
            continue;
        };
        let mut end = i + prefix.len();
        while end < text.len() && !is_url_break(text[end]) {
            end += 1;
        }
        out.extend_from_slice(&sanitize_url_bytes(prefix, &text[i + prefix.len()..end]));
        i = end;
    }
    out
}

fn is_url_break(byte: u8) -> bool {
    // Go's `https?://[^\s"'<>\\]+`: single quotes stay inside the match.
    matches!(
        byte,
        b' ' | b'\t' | b'\n' | b'\r' | b'\x0b' | b'\x0c' | b'"' | b'<' | b'>' | b'\\'
    )
}

fn sanitize_url_bytes(prefix: &[u8], rest: &[u8]) -> Vec<u8> {
    if rest.iter().any(|b| *b < 0x20) || has_bad_escape(rest) || has_bad_brackets(rest) {
        return b"[URL OMITTED]".to_vec();
    }
    let (authority, path) = match rest.iter().position(|b| *b == b'/') {
        Some(index) => (&rest[..index], &rest[index..]),
        None => (rest, &[][..]),
    };
    let host = match authority.iter().rposition(|b| *b == b'@') {
        Some(index) => &authority[index + 1..],
        None => authority,
    };
    let clean_path = match path.iter().position(|b| *b == b'?' || *b == b'#') {
        Some(index) => &path[..index],
        None => path,
    };
    let mut out = Vec::with_capacity(prefix.len() + host.len() + clean_path.len());
    out.extend_from_slice(prefix);
    out.extend_from_slice(host);
    out.extend_from_slice(clean_path);
    out
}

fn has_bad_escape(text: &[u8]) -> bool {
    let mut i = 0;
    while i < text.len() {
        if text[i] == b'%' {
            if i + 2 >= text.len()
                || !text[i + 1].is_ascii_hexdigit()
                || !text[i + 2].is_ascii_hexdigit()
            {
                return true;
            }
            i += 3;
        } else {
            i += 1;
        }
    }
    false
}

fn has_bad_brackets(text: &[u8]) -> bool {
    let authority = match text.iter().position(|b| *b == b'/') {
        Some(index) => &text[..index],
        None => text,
    };
    if let Some(open) = authority.iter().position(|b| *b == b'[') {
        return !authority[open..].contains(&b']');
    }
    authority.contains(&b']')
}

/// Redaction sink: a lone file, or a file teed with a captured buffer for
/// command results. Both sides receive redacted bytes only.
pub(super) enum RedactOut {
    File(File),
    Tee(File, Vec<u8>),
    Discard,
}

impl RedactOut {
    pub(super) fn file(file: File) -> RedactOut {
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
    pub(super) fn new(out: RedactOut, secrets: Vec<Vec<u8>>) -> RedactingWriter {
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
            self.url_pending
                .iter()
                .rposition(|b| *b == b'\n')
                .map(|i| i + 1)
                .unwrap_or(0)
        };
        if end == 0 {
            return Ok(());
        }
        let chunk = self.url_pending[..end].to_vec();
        let rest = self.url_pending[end..].to_vec();
        self.url_pending = rest;
        let safe = redact_urls_bytes(&chunk);
        self.emit(&safe)
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
        // Like Go, retained and final-flush failures join; the file
        // itself closes on drop.
        let flush_err = self.flush(true).err();
        let joined = Error::join(vec![self.err.clone().map(Error::msg), flush_err]);
        self.err = joined.as_ref().map(|e| e.to_string());
        match joined {
            Some(err) => Err(err),
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

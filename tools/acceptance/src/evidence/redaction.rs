use std::fs::File;
use std::io::Write;

use crate::error::Error;

use super::{longest_secret, EVIDENCE_LIMIT};

/// Keep redaction metadata bounded independently of the capture size. The
/// pattern count cap reserves at most 384 KiB of Vec descriptors on 64-bit
/// targets; aggregate pattern bytes and all retained/transformed output use
/// the existing 16 MiB evidence budget.
const SECRET_PATTERN_COUNT_LIMIT: usize = 16_384;
const SECRET_PATTERN_BYTES_LIMIT: usize = EVIDENCE_LIMIT as usize;

/// Strip userinfo, queries, and fragments from `http(s)` URLs, matching
/// Go's `redactURLs`. Malformed URLs keep a sanitized form with those parts
/// still stripped; only the whole-URL omission differs, and only when Go's
/// parser rejects the URL outright.
pub fn redact_urls(text: &str) -> String {
    redact_urls_checked(text).unwrap_or_else(|_| "[REDACTED]".to_string())
}

pub(super) fn redact_urls_checked(text: &str) -> Result<String, Error> {
    if text.len() > EVIDENCE_LIMIT as usize {
        return Err(Error::msg("evidence redaction output limit exceeded"));
    }
    // Byte-level sanitizing only removes ASCII spans or emits an ASCII
    // literal, so valid UTF-8 input stays valid UTF-8.
    let output = redact_urls_bytes(text.as_bytes())
        .ok_or_else(|| Error::msg("evidence redaction output limit exceeded"))?;
    Ok(String::from_utf8(output).expect("url redaction preserves UTF-8"))
}

/// Byte-level URL redaction, so binary command output keeps its exact
/// retained bytes: only ASCII URL spans are rewritten, everything else
/// passes through untouched.
fn redact_urls_bytes(text: &[u8]) -> Option<Vec<u8>> {
    let mut out: Vec<u8> = Vec::with_capacity(text.len());
    let mut i = 0;
    while i < text.len() {
        let rest = &text[i..];
        let prefix: &[u8] = if rest
            .get(..7)
            .is_some_and(|p| p.eq_ignore_ascii_case(b"http://"))
        {
            b"http://"
        } else if rest
            .get(..8)
            .is_some_and(|p| p.eq_ignore_ascii_case(b"https://"))
        {
            b"https://"
        } else {
            if out.len() == EVIDENCE_LIMIT as usize {
                return None;
            }
            out.push(text[i]);
            i += 1;
            continue;
        };
        let mut end = i + prefix.len();
        while end < text.len() && !is_url_break(text[end]) {
            end += 1;
        }
        let safe = if text.get(end) == Some(&b'\\') {
            // A backslash can act as a path separator in downstream URL
            // parsers. Omit the malformed token through the next real boundary
            // so its apparent suffix cannot escape the match.
            while end < text.len() && !is_url_break_except_backslash(text[end]) {
                end += 1;
            }
            b"[URL OMITTED]".to_vec()
        } else {
            sanitize_url_bytes(&text[i..i + prefix.len()], &text[i + prefix.len()..end])
        };
        if out
            .len()
            .checked_add(safe.len())
            .is_none_or(|n| n > EVIDENCE_LIMIT as usize)
        {
            return None;
        }
        out.extend_from_slice(&safe);
        i = end;
    }
    Some(out)
}

fn is_url_break(byte: u8) -> bool {
    // Go's `https?://[^\s"'<>\\]+`: single quotes stay inside the match.
    matches!(
        byte,
        b' ' | b'\t' | b'\n' | b'\r' | b'\x0b' | b'\x0c' | b'"' | b'<' | b'>' | b'\\'
    )
}

fn is_url_break_except_backslash(byte: u8) -> bool {
    matches!(
        byte,
        b' ' | b'\t' | b'\n' | b'\r' | b'\x0b' | b'\x0c' | b'"' | b'<' | b'>'
    )
}

fn sanitize_url_bytes(prefix: &[u8], rest: &[u8]) -> Vec<u8> {
    // URL output is text-oriented; a non-UTF8 sequence inside a matched URL
    // is ambiguous and is omitted in full. Other binary output stays exact.
    if rest.iter().any(|b| *b < 0x20 || *b >= 0x80)
        || has_bad_escape(rest)
        || has_ambiguous_encoded_delimiter(rest)
        || has_bad_brackets(rest)
    {
        return b"[URL OMITTED]".to_vec();
    }
    // `?` and `#` can follow the authority directly (`https://host?token=x`)
    // without a slash. Find the authority boundary before stripping either
    // component so neither can accidentally be retained as part of the host.
    let authority_end = rest
        .iter()
        .position(|b| matches!(b, b'/' | b'?' | b'#'))
        .unwrap_or(rest.len());
    let authority = &rest[..authority_end];
    let path_and_suffix = &rest[authority_end..];
    let host_and_port = match authority.iter().rposition(|b| *b == b'@') {
        Some(index) => &authority[index + 1..],
        None => authority,
    };
    // Empty hosts and a dangling userinfo delimiter are malformed; omit the
    // matched URL rather than preserving an uncertain authority.
    if host_and_port.is_empty() || host_and_port.starts_with(b":") {
        return b"[URL OMITTED]".to_vec();
    }
    let clean_path = if path_and_suffix.first() == Some(&b'/') {
        match path_and_suffix
            .iter()
            .position(|b| *b == b'?' || *b == b'#')
        {
            Some(index) => &path_and_suffix[..index],
            None => path_and_suffix,
        }
    } else {
        &[][..]
    };
    let mut out = Vec::with_capacity(prefix.len() + host_and_port.len() + clean_path.len());
    out.extend_from_slice(prefix);
    out.extend_from_slice(host_and_port);
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
    let authority_end = text
        .iter()
        .position(|b| matches!(b, b'/' | b'?' | b'#'))
        .unwrap_or(text.len());
    let authority = &text[..authority_end];
    if let Some(open) = authority.iter().position(|b| *b == b'[') {
        return !authority[open..].contains(&b']');
    }
    authority.contains(&b']')
}

fn has_ambiguous_encoded_delimiter(text: &[u8]) -> bool {
    let authority_end = text
        .iter()
        .position(|b| matches!(b, b'/' | b'?' | b'#'))
        .unwrap_or(text.len());
    let mut i = 0;
    while i + 2 < text.len() {
        if text[i] == b'%' {
            let code = [
                text[i + 1].to_ascii_lowercase(),
                text[i + 2].to_ascii_lowercase(),
            ];
            let structural = matches!(
                &code,
                b"2f" | b"3f" | b"23" | b"5c" | b"40" | b"3a" | b"5b" | b"5d"
            );
            if structural && (i < authority_end || code == *b"5c") {
                return true;
            }
            i += 3;
        } else {
            i += 1;
        }
    }
    false
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
    emitted: u64,
    limit: Option<u64>,
    closed: bool,
    err: Option<String>,
}

impl RedactingWriter {
    pub(super) fn new(out: RedactOut, secrets: Vec<Vec<u8>>) -> RedactingWriter {
        let pattern_bytes = secrets
            .iter()
            .try_fold(0usize, |sum, secret| sum.checked_add(secret.len()));
        let pattern_error = if secrets.len() > SECRET_PATTERN_COUNT_LIMIT
            || pattern_bytes.is_none_or(|size| size > SECRET_PATTERN_BYTES_LIMIT)
        {
            Some("evidence redaction pattern limit exceeded".to_string())
        } else {
            None
        };
        RedactingWriter {
            out,
            secrets: if pattern_error.is_some() {
                Vec::new()
            } else {
                secrets
            },
            pending: Vec::new(),
            url_pending: Vec::new(),
            count: 0,
            emitted: 0,
            limit: Some(EVIDENCE_LIMIT),
            closed: false,
            err: pattern_error,
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
            emitted: 0,
            limit: None,
            closed: false,
            err: None,
        }
    }

    /// File teed with a captured buffer, for command results.
    pub fn tee(file: File, secrets: Vec<Vec<u8>>) -> RedactingWriter {
        RedactingWriter::new(RedactOut::Tee(file, Vec::new()), secrets)
    }

    /// Mark a capture that stopped before its stream completed. Keep any
    /// earlier writer failure as the primary cause.
    pub(crate) fn mark_incomplete_capture(&mut self) {
        if self.err.is_none() {
            self.err = Some("evidence capture incomplete: deadline or cancellation".to_string());
        }
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
        let next = self
            .emitted
            .checked_add(bytes.len() as u64)
            .filter(|size| *size <= EVIDENCE_LIMIT)
            .ok_or_else(|| Error::msg("evidence output limit exceeded"))?;
        match &mut self.out {
            RedactOut::File(file) => file.write_all(bytes)?,
            RedactOut::Tee(file, buf) => {
                file.write_all(bytes)?;
                buf.extend_from_slice(bytes);
            }
            RedactOut::Discard => {}
        }
        self.emitted = next;
        Ok(())
    }

    fn append_bounded(target: &mut Vec<u8>, bytes: &[u8]) -> Result<(), Error> {
        let next = target
            .len()
            .checked_add(bytes.len())
            .filter(|size| *size <= EVIDENCE_LIMIT as usize)
            .ok_or_else(|| Error::msg("evidence output limit exceeded"))?;
        target.reserve(next.saturating_sub(target.len()));
        target.extend_from_slice(bytes);
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
                Self::append_bounded(&mut produced, b"[REDACTED]")?;
                consumed += match_len;
            } else {
                Self::append_bounded(&mut produced, &window[..1])?;
                consumed += 1;
            }
        }
        self.pending = pending[consumed..].to_vec();
        Self::append_bounded(&mut self.url_pending, &produced)?;
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
        let safe = redact_urls_bytes(&chunk)
            .ok_or_else(|| Error::msg("evidence output limit exceeded"))?;
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
            if self
                .count
                .checked_add(data.len() as u64)
                .is_none_or(|next| next > limit)
            {
                self.err = Some("evidence output limit exceeded".to_string());
                return Err(Error::msg("evidence output limit exceeded"));
            }
        }
        self.count += data.len() as u64;
        if self
            .pending
            .len()
            .checked_add(data.len())
            .is_none_or(|size| size > EVIDENCE_LIMIT as usize)
        {
            self.err = Some("evidence output limit exceeded".to_string());
            return Err(Error::msg("evidence output limit exceeded"));
        }
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
        let flush_err = if self.err.is_some() {
            None
        } else {
            self.flush(true).err()
        };
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

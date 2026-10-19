//! `build.go`: recall ring log and failure attribution.

use crate::error::Error;

/// recallLog keeps bounded diagnostic context in memory and forwards all
/// bytes written after attachment to the build log. Admission commands run
/// before the log file exists; without the ring their recent context would
/// be lost and failures would surface as a bare exit status.
#[derive(Default)]
pub struct RecallLog {
    lines: Vec<String>,
    frag: Vec<u8>,
    dropping_fragment: bool,
    file: Option<Box<dyn std::io::Write>>,
}

const MAX_DIAGNOSTIC_LINE_BYTES: usize = 4 * 1024;
const MAX_DIAGNOSTIC_LINES: usize = 20;

fn bounded_utf8_prefix(text: &str) -> &str {
    if text.len() <= MAX_DIAGNOSTIC_LINE_BYTES {
        return text;
    }
    let mut end = MAX_DIAGNOSTIC_LINE_BYTES;
    while !text.is_char_boundary(end) {
        end -= 1;
    }
    &text[..end]
}

fn retained_line(bytes: &[u8]) -> String {
    let text = String::from_utf8_lossy(bytes);
    bounded_utf8_prefix(&text).to_owned()
}

impl RecallLog {
    pub fn new() -> RecallLog {
        RecallLog::default()
    }

    pub fn write_bytes(&mut self, data: &[u8]) -> Result<usize, Error> {
        let written = match self.file.as_mut() {
            Some(file) => {
                use std::io::Write;
                file.write(data).map_err(Error::from)?
            }
            None => data.len(),
        };
        self.retain_bytes(&data[..written]);
        Ok(written)
    }

    fn retain_bytes(&mut self, data: &[u8]) {
        let mut rest = data;
        while let Some(newline) = rest.iter().position(|byte| *byte == b'\n') {
            self.append_fragment(&rest[..newline]);
            if self.lines.len() == MAX_DIAGNOSTIC_LINES {
                self.lines.remove(0);
            }
            self.lines.push(retained_line(&self.frag));
            self.frag.clear();
            self.dropping_fragment = false;
            rest = &rest[newline + 1..];
        }
        self.append_fragment(rest);
    }

    fn append_fragment(&mut self, bytes: &[u8]) {
        if self.dropping_fragment || bytes.is_empty() {
            return;
        }
        let available = MAX_DIAGNOSTIC_LINE_BYTES - self.frag.len();
        let retained = available.min(bytes.len());
        self.frag.extend_from_slice(&bytes[..retained]);
        if retained < bytes.len() {
            self.dropping_fragment = true;
        }
    }

    /// Attach the final build-log writer after replaying bounded admission
    /// context. Raw command output is complete only after attachment.
    pub fn attach(&mut self, mut file: Box<dyn std::io::Write>) -> Result<(), Error> {
        use std::io::Write;
        for line in &self.lines {
            file.write_all(line.as_bytes()).map_err(Error::from)?;
            file.write_all(b"\n").map_err(Error::from)?;
        }
        file.write_all(&self.frag).map_err(Error::from)?;
        self.file = Some(file);
        Ok(())
    }

    /// reason returns the last tool error line: the likeliest one-line cause.
    /// Command markers and secret-adjacent shell echoes never qualify.
    pub fn reason(&self) -> String {
        let fragment = retained_line(&self.frag);
        let reason = qualify_reason(&fragment);
        if !reason.is_empty() {
            return reason;
        }
        for line in self.lines.iter().rev() {
            if !qualify_reason(line).is_empty() {
                return qualify_reason(line);
            }
        }
        String::new()
    }
}

pub fn qualify_reason(line: &str) -> String {
    let trimmed = line.trim();
    if trimmed.is_empty() || trimmed.starts_with("COMMAND ") || trimmed.starts_with("$ ") {
        return String::new();
    }
    trimmed.to_string()
}

/// failureReason prefers the recall ring for tool failures, where it holds
/// the failing command's own stderr. Local failures (decode, file,
/// validation) must surface their own error instead: the ring then holds
/// only progress noise from earlier, already successful commands.
pub fn failure_reason(ring: &RecallLog, err: &Error) -> String {
    if is_tool_failure(err) {
        let reason = ring.reason();
        if !reason.is_empty() {
            return reason;
        }
    }
    bounded_utf8_prefix(err.0.trim().split('\n').next().unwrap_or("")).to_string()
}

/// isToolFailure reports whether err came from a build command rather than
/// local admission or validation.
pub fn is_tool_failure(err: &Error) -> bool {
    err.0.contains("retain attempt and inspect build.log")
}

impl std::io::Write for RecallLog {
    fn write(&mut self, data: &[u8]) -> std::io::Result<usize> {
        self.write_bytes(data).map_err(std::io::Error::other)
    }

    fn flush(&mut self) -> std::io::Result<()> {
        match self.file.as_mut() {
            Some(file) => file.flush(),
            None => Ok(()),
        }
    }
}

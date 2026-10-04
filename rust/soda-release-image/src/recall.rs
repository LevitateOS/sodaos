//! `build.go`: recall ring log and failure attribution.

use crate::error::Error;

/// recallLog keeps the last lines of every build command in memory and
/// forwards everything to the build log once it opens. Admission commands
/// run before the log file exists; without the ring their voice would be
/// lost and failures would surface as a bare exit status.
#[derive(Default)]
pub struct RecallLog {
    lines: Vec<String>,
    frag: String,
    file: Option<Box<dyn std::io::Write>>,
}

impl RecallLog {
    pub fn new() -> RecallLog {
        RecallLog::default()
    }

    pub fn write_bytes(&mut self, data: &[u8]) -> Result<usize, Error> {
        let text = String::from_utf8_lossy(data);
        let mut parts: Vec<String> = (self.frag.clone() + &text)
            .split('\n')
            .map(|s| s.to_string())
            .collect();
        self.frag = parts.pop().unwrap_or_default();
        self.lines.extend(parts);
        if self.lines.len() > 20 {
            self.lines = self.lines[self.lines.len() - 20..].to_vec();
        }
        if let Some(file) = self.file.as_mut() {
            use std::io::Write;
            return file.write(data).map_err(Error::from);
        }
        Ok(data.len())
    }

    /// attach connects the build log file. Buffered admission output is
    /// flushed first so the file holds the whole attempt from the first
    /// command.
    pub fn attach(&mut self, mut file: Box<dyn std::io::Write>) {
        use std::io::Write;
        for line in &self.lines {
            // Best-effort admission replay; the build outcome never depends on it.
            let _ = writeln!(file, "{line}");
        }
        self.file = Some(file);
    }

    /// reason returns the last tool error line: the likeliest one-line cause.
    /// Command markers and secret-adjacent shell echoes never qualify.
    pub fn reason(&self) -> String {
        if !qualify_reason(&self.frag).is_empty() {
            return qualify_reason(&self.frag);
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
    let reason = ring.reason();
    if !reason.is_empty() && is_tool_failure(err) {
        return reason;
    }
    err.0.trim().split('\n').next().unwrap_or("").to_string()
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_reason_skips_markers() {
        // Oracle: Go TestRecallLogReasonSkipsMarkers.
        let mut ring = RecallLog::new();
        ring.write_bytes(b"\nCOMMAND podman\n$ echo secret\nboom\n")
            .unwrap();
        assert_eq!(ring.reason(), "boom");
        let mut ring = RecallLog::new();
        ring.write_bytes(b"COMMAND podman\n").unwrap();
        assert_eq!(ring.reason(), "");
    }

    #[test]
    fn oracle_failure_reason_prefers_local_error_over_stale_ring() {
        // Oracle: Go TestFailureReasonPrefersLocalErrorOverStaleRing.
        let mut ring = RecallLog::new();
        ring.write_bytes(b"old noise\n").unwrap();
        let local = Error::msg("local decode broke\nsecond line");
        assert_eq!(failure_reason(&ring, &local), "local decode broke");
        let tool = Error::msg("podman failed; retain attempt and inspect build.log: exit 1");
        assert_eq!(failure_reason(&ring, &tool), "old noise");
    }
}

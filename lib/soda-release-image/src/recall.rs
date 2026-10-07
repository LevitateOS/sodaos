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

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::{Arc, Mutex};

    #[derive(Clone)]
    struct Captured(Arc<Mutex<Vec<u8>>>);

    impl std::io::Write for Captured {
        fn write(&mut self, bytes: &[u8]) -> std::io::Result<usize> {
            self.0.lock().unwrap().extend_from_slice(bytes);
            Ok(bytes.len())
        }

        fn flush(&mut self) -> std::io::Result<()> {
            Ok(())
        }
    }

    struct Failing;

    impl std::io::Write for Failing {
        fn write(&mut self, _bytes: &[u8]) -> std::io::Result<usize> {
            Err(std::io::Error::new(
                std::io::ErrorKind::BrokenPipe,
                "synthetic log failure",
            ))
        }

        fn flush(&mut self) -> std::io::Result<()> {
            Ok(())
        }
    }

    struct Short(Arc<Mutex<Vec<u8>>>);

    impl std::io::Write for Short {
        fn write(&mut self, bytes: &[u8]) -> std::io::Result<usize> {
            let written = bytes.len().min(3);
            self.0.lock().unwrap().extend_from_slice(&bytes[..written]);
            Ok(written)
        }

        fn flush(&mut self) -> std::io::Result<()> {
            Ok(())
        }
    }

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

    #[test]
    fn diagnostic_retention_is_bounded_and_attached_log_stays_complete() {
        let captured = Arc::new(Mutex::new(Vec::new()));
        let mut ring = RecallLog::new();
        ring.attach(Box::new(Captured(captured.clone()))).unwrap();

        let mut input = Vec::new();
        for index in 0..22 {
            input.extend_from_slice(format!("line-{index:02}\n").as_bytes());
        }
        input.extend_from_slice(b"cause:");
        input.extend(std::iter::repeat_n(b'x', MAX_DIAGNOSTIC_LINE_BYTES * 3));
        input.push(b'\n');
        input.extend_from_slice(b"$ ");
        input.extend(std::iter::repeat_n(b's', MAX_DIAGNOSTIC_LINE_BYTES * 300));
        ring.write_bytes(&input).unwrap();

        assert_eq!(ring.lines.len(), MAX_DIAGNOSTIC_LINES);
        assert_eq!(ring.lines.first().unwrap(), "line-03");
        assert!(ring.frag.len() <= MAX_DIAGNOSTIC_LINE_BYTES);
        assert!(ring.dropping_fragment);
        assert_eq!(ring.reason().len(), MAX_DIAGNOSTIC_LINE_BYTES);
        assert!(ring.reason().starts_with("cause:"));
        assert_eq!(*captured.lock().unwrap(), input);

        let local = Error::msg(format!(
            "{}\nadditional detail",
            "e".repeat(MAX_DIAGNOSTIC_LINE_BYTES * 5)
        ));
        assert_eq!(
            failure_reason(&RecallLog::new(), &local).len(),
            MAX_DIAGNOSTIC_LINE_BYTES
        );
    }

    #[test]
    fn attached_log_write_errors_propagate() {
        let mut ring = RecallLog::new();
        ring.attach(Box::new(Failing)).unwrap();
        let err = ring.write_bytes(b"failure\n").unwrap_err();
        assert!(err.to_string().contains("synthetic log failure"));
    }

    #[test]
    fn attached_short_writes_follow_write_all_semantics() {
        use std::io::Write;

        let captured = Arc::new(Mutex::new(Vec::new()));
        let mut ring = RecallLog::new();
        ring.attach(Box::new(Short(captured.clone()))).unwrap();
        ring.write_all(b"cause\n").unwrap();

        assert_eq!(*captured.lock().unwrap(), b"cause\n");
        assert_eq!(ring.reason(), "cause");
    }

    #[test]
    fn attach_replay_is_bounded_short_write_safe_and_fallible() {
        use std::io::Write;

        let mut ring = RecallLog::new();
        ring.write_all(b"admission line\nunfinished").unwrap();
        let captured = Arc::new(Mutex::new(Vec::new()));
        ring.attach(Box::new(Short(captured.clone()))).unwrap();
        assert_eq!(*captured.lock().unwrap(), b"admission line\nunfinished");

        let mut ring = RecallLog::new();
        ring.write_all(b"prior admission").unwrap();
        assert!(ring.attach(Box::new(Failing)).is_err());
    }
}

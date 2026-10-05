//! `media_events.go`: observe upstream packager progress windows.

use std::io::Write;

use crate::error::Error;

/// Observe existing upstream progress without modifying the packager. These
/// are log-arrival windows, not CPU profiles: the rootfs window includes
/// CPIO/hashing, and osmet includes its checksum/readback. Never persist
/// arbitrary log text here.
pub struct MediaEventWriter<W: Write, E: Write> {
    log: W,
    events: E,
    start: std::time::Instant,
    line: Vec<u8>,
    dropped: bool,
}

impl<W: Write, E: Write> MediaEventWriter<W, E> {
    pub fn new(log: W, events: E) -> MediaEventWriter<W, E> {
        MediaEventWriter {
            log,
            events,
            start: std::time::Instant::now(),
            line: Vec::new(),
            dropped: false,
        }
    }

    pub fn media_log_event(line: &str) -> &str {
        if line.starts_with("Generating osmet file for ") {
            "osmet-start"
        } else if line == "Packing successful!" {
            "osmet-end"
        } else if line.starts_with("Creating erofs with ") {
            "rootfs-start"
        } else if line.starts_with("Substituting ISO kernel arguments:") {
            "rootfs-end"
        } else if line.starts_with("genisoimage ") {
            "iso-start"
        } else if line.contains(" extents written (") {
            "iso-end"
        } else {
            ""
        }
    }

    fn append_line_byte(&mut self, b: u8) {
        if self.line.len() == 8192 {
            self.line.clear();
            self.dropped = true;
        }
        if !self.dropped {
            self.line.push(b);
        }
    }

    fn finish_line(&mut self) -> String {
        let mut event = String::new();
        if !self.dropped {
            event = Self::media_log_event(&String::from_utf8_lossy(&self.line)).to_string();
        }
        self.line.clear();
        self.dropped = false;
        event
    }

    fn consume(&mut self, b: u8) -> Result<(), Error> {
        if b != b'\n' {
            self.append_line_byte(b);
            return Ok(());
        }
        let event = self.finish_line();
        if event.is_empty() {
            return Ok(());
        }
        let seconds = self.start.elapsed().as_secs_f64();
        let mut line = String::new();
        crate::jsonio::write_compact(
            &mut line,
            &soda_json::JsonValue::Object(vec![
                ("Event".to_string(), soda_json::JsonValue::Str(event)),
                (
                    "Seconds".to_string(),
                    soda_json::JsonValue::Number(crate::jsonio::format_float_go(seconds)),
                ),
            ]),
        );
        line.push('\n');
        self.events.write_all(line.as_bytes())?;
        Ok(())
    }

    pub fn write_data(&mut self, data: &[u8]) -> Result<usize, Error> {
        let written = self.log.write(data)?;
        for b in &data[..written] {
            self.consume(*b)?;
        }
        Ok(written)
    }
}

impl<W: Write, E: Write> Write for MediaEventWriter<W, E> {
    fn write(&mut self, data: &[u8]) -> std::io::Result<usize> {
        self.write_data(data)
            .map_err(|e| std::io::Error::other(e.0))
    }

    fn flush(&mut self) -> std::io::Result<()> {
        self.log.flush()?;
        self.events.flush()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_events_observe_bounded_public_windows() {
        // Oracle: Go TestMediaEventsObserveBoundedPublicWindows.
        let mut writer = MediaEventWriter::new(Vec::new(), Vec::new());
        writer
            .write_data(b"noise\nCreating erofs with mkfs\nmore\n")
            .unwrap();
        writer.write_data(b"Packing successful!\n").unwrap();
        let events = String::from_utf8(writer.events.clone()).unwrap();
        assert!(events.contains("\"Event\":\"rootfs-start\""));
        assert!(events.contains("\"Event\":\"osmet-end\""));
        assert!(!events.contains("noise"));
        // Over-long lines are dropped, never emitted.
        let mut writer = MediaEventWriter::new(Vec::new(), Vec::new());
        let mut long = vec![b'x'; 9000];
        long.extend_from_slice(b"Packing successful!\n");
        writer.write_data(&long).unwrap();
        assert!(writer.events.is_empty());
    }
}

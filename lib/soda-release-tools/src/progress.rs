//! Monotonic build progress (Go `internal/release/build` `progress.go`):
//! the `START`/`DONE`/`FAILED`/`CANCELLED` stderr event protocol the
//! `soda-candidate` display consumes.

use std::io::Write;
use std::os::unix::fs::OpenOptionsExt;
use std::sync::{Arc, Mutex};
use std::time::Duration;

use crate::exitcode::{build_exit_code, ToolError};

pub fn monotonic() -> Duration {
    let mut ts = libc::timespec {
        tv_sec: 0,
        tv_nsec: 0,
    };
    // CLOCK_MONOTONIC cannot fail with valid arguments.
    unsafe {
        libc::clock_gettime(libc::CLOCK_MONOTONIC, &mut ts);
    }
    Duration::new(ts.tv_sec.max(0) as u64, ts.tv_nsec.max(0) as u32)
}

fn format_duration(d: Duration) -> String {
    let s = d.as_secs() as i64;
    let s = s.max(0);
    format!("{:02}:{:02}:{:02}", s / 3600, s / 60 % 60, s % 60)
}

fn failed_text(label: &str, reason: &str) -> String {
    if reason.is_empty() {
        label.to_owned()
    } else {
        format!("{label} | reason {reason}")
    }
}

fn finish_kind(code: i32) -> &'static str {
    if code == 0 {
        "SUCCESS"
    } else if code == 130 || code == 143 {
        "CANCELLED"
    } else {
        "FAILED"
    }
}

pub struct BuildProgress {
    title: String,
    label: String,
    path: String,
    origin: Duration,
    started: Duration,
    now: Box<dyn Fn() -> Duration + Send>,
    stderr: Arc<Mutex<Vec<u8>>>,
    mirror_stderr: bool,
    finished: bool,
    phase: String,
    phase_started: Duration,
    reason: String,
}

impl BuildProgress {
    pub fn new(title: &str) -> Result<BuildProgress, String> {
        Self::new_with_clock(title, Box::new(monotonic))
    }

    pub fn new_with_clock(
        title: &str,
        now: Box<dyn Fn() -> Duration + Send>,
    ) -> Result<BuildProgress, String> {
        let origin_now = now();
        let origin = match std::env::var("SODA_BUILD_START_NS") {
            Ok(raw) if !raw.is_empty() => match raw.parse::<i64>() {
                Ok(n) if n >= 0 && Duration::from_nanos(n as u64) <= origin_now => {
                    Duration::from_nanos(n as u64)
                }
                _ => return Err("invalid inherited timing origin".to_owned()),
            },
            _ => {
                unsafe {
                    std::env::set_var("SODA_BUILD_START_NS", origin_now.as_nanos().to_string());
                }
                origin_now
            }
        };
        Ok(BuildProgress {
            title: title.to_owned(),
            label: String::new(),
            path: String::new(),
            origin,
            started: Duration::ZERO,
            now,
            stderr: Arc::new(Mutex::new(Vec::new())),
            mirror_stderr: true,
            finished: false,
            phase: String::new(),
            phase_started: Duration::ZERO,
            reason: String::new(),
        })
    }

    /// Redirect event lines into the returned buffer instead of stderr (tests).
    pub fn capture(&mut self) -> Arc<Mutex<Vec<u8>>> {
        self.mirror_stderr = false;
        Arc::clone(&self.stderr)
    }

    pub fn note_reason(&mut self, s: &str) {
        let mut s = match s.find(['\r', '\n']) {
            Some(i) => &s[..i],
            None => s,
        }
        .trim()
        .replace('|', "/");
        if s.chars().count() > 300 {
            s = s.chars().take(300).collect();
        }
        if s.is_empty() {
            return;
        }
        self.reason = s;
    }

    fn emit(&mut self, kind: &str, text: &str) -> Result<(), String> {
        let line = format!("{kind:<8} {text}\n");
        if self.mirror_stderr {
            if let Err(e) = std::io::stderr().write_all(line.as_bytes()) {
                return Err(e.to_string());
            }
        } else if let Ok(mut buf) = self.stderr.lock() {
            buf.extend_from_slice(line.as_bytes());
        }
        if !self.path.is_empty() {
            let file = std::fs::OpenOptions::new()
                .append(true)
                .custom_flags(libc::O_NOFOLLOW)
                .open(&self.path);
            match file {
                Ok(mut f) => {
                    if let Err(e) = f.write_all(line.as_bytes()) {
                        return Err(e.to_string());
                    }
                }
                Err(e) => return Err(e.to_string()),
            }
        }
        Ok(())
    }

    pub fn create_log(&mut self, path: &str) -> Result<(), String> {
        let inherited = std::env::var("SODA_BUILD_TIMING_LOG").unwrap_or_default();
        if !inherited.is_empty() && std::env::var("SODA_BUILD_CHILD").as_deref() == Ok("1") {
            self.path = inherited;
            return Ok(());
        }
        if let Err(e) = std::fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .open(path)
        {
            return Err(e.to_string());
        }
        self.path = path.to_owned();
        self.emit("LOG", path)
    }

    pub fn phase(&mut self, label: &str) -> Result<(), String> {
        if self.finished || label.contains(['\r', '\n']) {
            return Err("invalid progress transition".to_owned());
        }
        self.end_opt(None)?;
        self.end_phase_opt(None)?;
        self.phase = label.to_owned();
        self.phase_started = (self.now)();
        self.reason.clear();
        self.emit("START", label)
    }

    fn kind_for(err: Option<&ToolError>) -> &'static str {
        match err {
            None => "DONE",
            Some(e) => match build_exit_code(Some(e)) {
                130 | 143 => "CANCELLED",
                _ => "FAILED",
            },
        }
    }

    pub fn end_phase(&mut self, err: Option<&ToolError>) -> Result<(), String> {
        self.end_phase_opt(err)
    }

    fn end_phase_opt(&mut self, err: Option<&ToolError>) -> Result<(), String> {
        if self.phase.is_empty() {
            return Ok(());
        }
        let kind = Self::kind_for(err);
        let label = std::mem::take(&mut self.phase);
        let now = (self.now)();
        let text = format!(
            "{} | phase {} | total {}",
            label,
            format_duration(now.saturating_sub(self.phase_started)),
            format_duration(now.saturating_sub(self.origin))
        );
        let text = if kind == "FAILED" {
            failed_text(&text, &self.reason.clone())
        } else {
            text
        };
        self.emit(kind, &text)
    }

    pub fn next(&mut self, label: &str) -> Result<(), String> {
        if self.finished || label.contains(['\r', '\n']) {
            return Err("invalid progress transition".to_owned());
        }
        self.end_opt(None)?;
        self.label = label.to_owned();
        self.started = (self.now)();
        self.reason.clear();
        self.emit("START", label)
    }

    pub fn end(&mut self, err: Option<&ToolError>) -> Result<(), String> {
        self.end_opt(err)
    }

    fn end_opt(&mut self, err: Option<&ToolError>) -> Result<(), String> {
        if self.label.is_empty() {
            return Ok(());
        }
        let kind = Self::kind_for(err);
        let label = std::mem::take(&mut self.label);
        let now = (self.now)();
        let text = format!(
            "{} | section {} | total {}",
            label,
            format_duration(now.saturating_sub(self.started)),
            format_duration(now.saturating_sub(self.origin))
        );
        let text = if kind == "FAILED" {
            failed_text(&text, &self.reason.clone())
        } else {
            text
        };
        self.emit(kind, &text)
    }

    fn write_section_summary(&mut self) -> Result<(), String> {
        if self.path.is_empty() {
            return Ok(());
        }
        let data = std::fs::read(&self.path).map_err(|e| e.to_string())?;
        let text = String::from_utf8_lossy(&data);
        let mut summary = String::from("\nSECTION SUMMARY\n");
        for line in text.split('\n') {
            if line.starts_with("DONE ")
                || line.starts_with("FAILED ")
                || line.starts_with("CANCELLED ")
            {
                summary.push_str(&format!("  {line}\n"));
            }
        }
        if self.mirror_stderr {
            std::io::stderr()
                .write_all(summary.as_bytes())
                .map_err(|e| e.to_string())?;
        } else if let Ok(mut buf) = self.stderr.lock() {
            buf.extend_from_slice(summary.as_bytes());
        }
        Ok(())
    }

    pub fn finish(&mut self, err: Option<&ToolError>) -> Result<(), String> {
        if self.finished {
            return Ok(());
        }
        self.finished = true;
        let mut first_err: Option<String> = None;
        if let Err(e) = self.end_opt(err) {
            first_err = Some(e);
        }
        if let Err(e) = self.end_phase_opt(err) {
            if first_err.is_none() {
                first_err = Some(e);
            }
        }
        if std::env::var("SODA_BUILD_CHILD").as_deref() == Ok("1") {
            return first_err.map_or(Ok(()), Err);
        }
        if let Err(e) = self.write_section_summary() {
            if first_err.is_none() {
                first_err = Some(e);
            }
        }
        let code = match (&err, &first_err) {
            (None, None) => 0,
            (Some(e), _) => build_exit_code(Some(e)),
            (None, Some(_)) => 1,
        };
        let now = (self.now)();
        let text = format!(
            "{} | total {} | exit {code}",
            self.title,
            format_duration(now.saturating_sub(self.origin))
        );
        if let Err(e) = self.emit(finish_kind(code), &text) {
            if first_err.is_none() {
                first_err = Some(e);
            }
        }
        first_err.map_or(Ok(()), Err)
    }
}

#[cfg(test)]
mod tests;

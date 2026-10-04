//! `soda-candidate` progress display (Go `tools/soda-candidate`
//! `display.go`): controller event parsing and the terminal renderer.

use std::io::Write;
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

use crate::candidate_hints::failure_hint;

#[derive(Debug, Clone, Default)]
pub struct Event {
    pub kind: String,
    pub label: String,
    pub phase_dur: String,
    pub total_dur: String,
    pub path: String,
    pub reason: String,
}

pub fn parse_event(line: &str) -> Option<Event> {
    let (kind, rest) = match line.split_once(' ') {
        Some((k, r)) => (k, r),
        None => return None,
    };
    match kind {
        "START" => parse_start_event(kind, rest),
        "DONE" | "FAILED" | "CANCELLED" => parse_done_event(kind, rest),
        "CANDIDATE" | "MEDIA" | "FINAL" | "LOG" => parse_artifact_event(kind, rest),
        _ => None,
    }
}

fn parse_start_event(kind: &str, rest: &str) -> Option<Event> {
    if rest.is_empty() {
        return None;
    }
    Some(Event { kind: kind.to_owned(), label: rest.to_owned(), ..Event::default() })
}

fn parse_done_event(kind: &str, rest: &str) -> Option<Event> {
    let parts: Vec<&str> = rest.split(" | ").collect();
    if parts.is_empty() || parts[0].is_empty() {
        return None;
    }
    let mut e = Event { kind: kind.to_owned(), label: parts[0].to_owned(), ..Event::default() };
    for p in &parts[1..] {
        apply_duration_part(&mut e, p);
    }
    Some(e)
}

fn apply_duration_part(e: &mut Event, p: &str) {
    if let Some(d) = p.strip_prefix("phase ") {
        e.phase_dur = d.to_owned();
    } else if let Some(d) = p.strip_prefix("section ") {
        e.phase_dur = d.to_owned();
    } else if let Some(d) = p.strip_prefix("total ") {
        e.total_dur = d.to_owned();
    } else if let Some(r) = p.strip_prefix("reason ") {
        e.reason = r.to_owned();
    }
}

fn parse_artifact_event(kind: &str, rest: &str) -> Option<Event> {
    if rest.is_empty() {
        return None;
    }
    Some(Event { kind: kind.to_owned(), path: rest.to_owned(), ..Event::default() })
}

/// Map a worker sandbox path back onto the host output directory.
pub fn host_artifact_path(out_dir: &str, sandbox: &str) -> String {
    let base = out_dir.rsplit('/').next().unwrap_or_default();
    if out_dir.is_empty() || base.is_empty() || base == "/" || base == "." {
        return sandbox.to_owned();
    }
    match sandbox.rfind(base) {
        Some(i) => {
            let rest = &sandbox[i + base.len()..];
            let rest = rest.strip_prefix('/').unwrap_or(rest);
            if rest.is_empty() {
                out_dir.to_owned()
            } else {
                format!("{}/{}", out_dir.trim_end_matches('/'), rest)
            }
        }
        None => sandbox.to_owned(),
    }
}

#[derive(Debug, Clone)]
struct Phase {
    label: String,
    state: String,
    dur: String,
    started: Instant,
}

const SPINNER: [char; 4] = ['|', '/', '-', '\\'];
pub const LOG_VIEWPORT: usize = 8;
pub const CLOSED_PHASE_KEEP: usize = 5;

struct RendererInner {
    tty: bool,
    width: usize,
    start: Instant,
    phases: Vec<Phase>,
    log: Vec<String>,
    arts: Vec<String>,
    drawn: usize,
    out_dir: String,
    failed_label: String,
    failed_reason: String,
}

/// Terminal renderer. Output goes to the injected writer.
pub struct Renderer {
    inner: Arc<Mutex<RendererInner>>,
    writer: Arc<Mutex<Box<dyn Write + Send>>>,
    ticker: Mutex<Option<TickerHandle>>,
}

struct TickerHandle {
    stop: std::sync::mpsc::Sender<()>,
    thread: Option<std::thread::JoinHandle<()>>,
}

impl Renderer {
    pub fn new(writer: Box<dyn Write + Send>, tty: bool, width: usize) -> Renderer {
        let width = if width < 20 { 80 } else { width };
        Renderer {
            inner: Arc::new(Mutex::new(RendererInner {
                tty,
                width,
                start: Instant::now(),
                phases: Vec::new(),
                log: Vec::new(),
                arts: Vec::new(),
                drawn: 0,
                out_dir: String::new(),
                failed_label: String::new(),
                failed_reason: String::new(),
            })),
            writer: Arc::new(Mutex::new(writer)),
            ticker: Mutex::new(None),
        }
    }

    pub fn set_out_dir(&self, out_dir: &str) {
        self.inner.lock().unwrap().out_dir = out_dir.to_owned();
    }

    pub fn feed(&self, line: &str) -> Result<(), String> {
        let mut inner = self.inner.lock().unwrap();
        let event = parse_event(line);
        match event {
            None => Self::note_locked(&mut inner, &self.writer, line),
            Some(e) => {
                let Event { kind, label, phase_dur, path, reason, .. } = e;
                match kind.as_str() {
                    "START" => inner.phases.push(Phase {
                        label,
                        state: "run".to_owned(),
                        dur: String::new(),
                        started: Instant::now(),
                    }),
                    "DONE" | "FAILED" | "CANCELLED" => {
                        let state = match kind.as_str() {
                            "DONE" => "ok",
                            "FAILED" => "fail",
                            _ => "stop",
                        };
                        close_phase(&mut inner.phases, &label, state, &phase_dur);
                        if kind == "FAILED" && inner.failed_label.is_empty() {
                            inner.failed_label = label;
                            inner.failed_reason = reason;
                        }
                    }
                    _ => {
                        let translated = host_artifact_path(&inner.out_dir, &path);
                        inner.arts.push(format!("{kind} {translated}"));
                    }
                }
                if !inner.tty {
                    let mut shown = line.to_owned();
                    if matches!(kind.as_str(), "CANDIDATE" | "MEDIA" | "FINAL" | "LOG") {
                        let translated = host_artifact_path(&inner.out_dir, &path);
                        if translated != path {
                            shown = format!("{kind} {translated}");
                        }
                    }
                    let stamp = wall_since(inner.start);
                    let mut w = self.writer.lock().unwrap();
                    return w.write_all(format!("[{stamp}] {shown}\n").as_bytes()).map_err(|e| e.to_string());
                }
                Self::draw_locked(&mut inner, &self.writer)
            }
        }
    }

    pub fn note(&self, line: &str) -> Result<(), String> {
        let mut inner = self.inner.lock().unwrap();
        Self::note_locked(&mut inner, &self.writer, line)
    }

    fn note_locked(
        inner: &mut RendererInner,
        writer: &Mutex<Box<dyn Write + Send>>,
        line: &str,
    ) -> Result<(), String> {
        if !inner.tty {
            let stamp = wall_since(inner.start);
            let mut w = writer.lock().unwrap();
            return w.write_all(format!("[{stamp}] {line}\n").as_bytes()).map_err(|e| e.to_string());
        }
        inner.log.push(line.to_owned());
        if inner.log.len() > LOG_VIEWPORT {
            let excess = inner.log.len() - LOG_VIEWPORT;
            inner.log.drain(..excess);
        }
        Self::draw_locked(inner, writer)
    }

    fn draw_locked(
        inner: &mut RendererInner,
        writer: &Mutex<Box<dyn Write + Send>>,
    ) -> Result<(), String> {
        let lines = build_lines(inner, Instant::now());
        let mut first_err: Option<String> = None;
        let mut w = writer.lock().unwrap();
        if inner.drawn > 0 {
            if let Err(e) = write!(w, "\x1b[{}A", inner.drawn) {
                first_err = Some(e.to_string());
            }
        }
        for line in &lines {
            if let Err(e) = writeln!(w, "\x1b[K{}", truncate(line, inner.width)) {
                if first_err.is_none() {
                    first_err = Some(e.to_string());
                }
            }
        }
        let _ = w.flush();
        inner.drawn = lines.len();
        first_err.map_or(Ok(()), Err)
    }
}

fn close_phase(phases: &mut Vec<Phase>, label: &str, state: &str, dur: &str) {
    for p in phases.iter_mut().rev() {
        if p.label == label && p.state == "run" {
            p.state = state.to_owned();
            p.dur = dur.to_owned();
            return;
        }
    }
    phases.push(Phase {
        label: label.to_owned(),
        state: state.to_owned(),
        dur: dur.to_owned(),
        started: Instant::now(),
    });
}

fn phase_mark(state: &str) -> &'static str {
    match state {
        "ok" => "[ok]  ",
        "fail" => "[FAIL]",
        "stop" => "[stop]",
        _ => "[..]  ",
    }
}

fn build_lines(inner: &RendererInner, now: Instant) -> Vec<String> {
    let mut lines = vec![format!(
        "soda-candidate | elapsed {}",
        wall_duration(now.saturating_duration_since(inner.start))
    )];
    for line in &inner.log {
        lines.push(format!("  {line}"));
    }
    if !inner.phases.is_empty() {
        lines.push("  steps:".to_owned());
    }
    let mut keep_ok = vec![false; inner.phases.len()];
    let mut kept = 0;
    for (i, p) in inner.phases.iter().enumerate().rev() {
        if p.state != "ok" {
            continue;
        }
        kept += 1;
        if kept <= CLOSED_PHASE_KEEP {
            keep_ok[i] = true;
        }
    }
    let mut hidden = 0;
    for (i, p) in inner.phases.iter().enumerate() {
        if p.state == "ok" && !keep_ok[i] {
            hidden += 1;
            continue;
        }
        if hidden > 0 {
            lines.push(format!("  … {hidden} earlier steps done"));
            hidden = 0;
        }
        lines.push(phase_line(p, now));
    }
    if !inner.arts.is_empty() {
        lines.push("  outputs:".to_owned());
    }
    for art in &inner.arts {
        lines.push(format!("  {art}"));
    }
    lines
}

fn phase_line(p: &Phase, now: Instant) -> String {
    if p.state == "run" {
        let elapsed = now.saturating_duration_since(p.started);
        let frame = SPINNER[(elapsed.as_millis() / 250) as usize % SPINNER.len()];
        return format!("  [{frame}] {}  (live {})", p.label, wall_duration(elapsed));
    }
    let mut line = format!("  {} {}", phase_mark(&p.state), p.label);
    if !p.dur.is_empty() {
        line.push_str(&format!("  ({})", p.dur));
    }
    line
}

impl Renderer {
    /// Refresh the elapsed header while the controller runs.
    pub fn start_ticker(&self) {
        let tty = self.inner.lock().unwrap().tty;
        let mut slot = self.ticker.lock().unwrap();
        if !tty || slot.is_some() {
            return;
        }
        let (stop_tx, stop_rx) = std::sync::mpsc::channel::<()>();
        let inner = Arc::clone(&self.inner);
        let writer = Arc::clone(&self.writer);
        let thread = std::thread::spawn(move || {
            loop {
                std::thread::sleep(Duration::from_millis(250));
                if stop_rx.try_recv().is_ok() {
                    return;
                }
                let mut guard = match inner.lock() {
                    Ok(g) => g,
                    Err(_) => return,
                };
                let _ = Self::draw_locked(&mut guard, &writer);
            }
        });
        *slot = Some(TickerHandle { stop: stop_tx, thread: Some(thread) });
    }

    pub fn stop_ticker(&self) {
        let mut slot = self.ticker.lock().unwrap();
        if let Some(mut handle) = slot.take() {
            let _ = handle.stop.send(());
            if let Some(thread) = handle.thread.take() {
                let _ = thread.join();
            }
        }
    }

    pub fn finish(&self, code: i32) -> Result<(), String> {
        let mut inner = self.inner.lock().unwrap();
        if inner.tty {
            Self::draw_locked(&mut inner, &self.writer)?;
        }
        let summary = format!(
            "soda-candidate: finished in {} with exit {code}{}\n",
            wall_since(inner.start),
            exit_meaning(code)
        );
        {
            let mut w = self.writer.lock().unwrap();
            w.write_all(summary.as_bytes()).map_err(|e| e.to_string())?;
        }
        if code != 0 && !inner.failed_label.is_empty() {
            Self::print_why_panel_locked(&inner, &self.writer)?;
        }
        Ok(())
    }

    fn print_why_panel_locked(
        inner: &RendererInner,
        writer: &Mutex<Box<dyn Write + Send>>,
    ) -> Result<(), String> {
        let cause = if inner.failed_reason.is_empty() {
            "see the build log"
        } else {
            inner.failed_reason.as_str()
        };
        let mut panel = format!("  why: {}\n  cause: {cause}\n", inner.failed_label);
        if !inner.out_dir.is_empty() {
            panel.push_str(&format!(
                "  log: {}/logs/build.log\n",
                inner.out_dir.trim_end_matches('/')
            ));
        }
        let hint = failure_hint(&inner.failed_reason);
        if !hint.is_empty() {
            panel.push_str(&format!("  hint: {hint}\n"));
        }
        let mut w = writer.lock().unwrap();
        w.write_all(panel.as_bytes()).map_err(|e| e.to_string())
    }
}

pub fn exit_meaning(code: i32) -> &'static str {
    match code {
        0 => " (development output ready; not release-qualified)",
        2 => " (incomplete: qualification passed but final signing is not connected; no qualified release)",
        _ => "",
    }
}

pub fn wall_duration(d: Duration) -> String {
    let s = d.as_secs() as i64;
    format!("{:02}:{:02}:{:02}", s / 3600, s / 60 % 60, s % 60)
}

pub fn wall_since(start: Instant) -> String {
    wall_duration(start.elapsed())
}

pub fn truncate(s: &str, width: usize) -> String {
    let chars: Vec<char> = s.chars().collect();
    if chars.len() <= width {
        return s.to_owned();
    }
    chars[..width.saturating_sub(1)].iter().collect::<String>() + "…"
}

pub fn is_terminal(fd: libc::c_int) -> bool {
    unsafe {
        let mut termios: libc::termios = std::mem::zeroed();
        libc::tcgetattr(fd, &mut termios) == 0
    }
}

pub fn term_width(fd: libc::c_int) -> usize {
    unsafe {
        let mut winsize: libc::winsize = std::mem::zeroed();
        if libc::ioctl(fd, libc::TIOCGWINSZ, &mut winsize) == 0 && winsize.ws_col != 0 {
            return winsize.ws_col as usize;
        }
    }
    80
}

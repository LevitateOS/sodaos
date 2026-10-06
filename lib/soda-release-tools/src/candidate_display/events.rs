//! Candidate progress events: wire parsing and artifact paths.

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
    // D01-F5: the wire pads kinds to 8 columns; normalize the padding out
    // of label identity so DONE closes its START.
    Some(Event {
        kind: kind.to_owned(),
        label: rest.trim_start().to_owned(),
        ..Event::default()
    })
}

fn parse_done_event(kind: &str, rest: &str) -> Option<Event> {
    let parts: Vec<&str> = rest.split(" | ").collect();
    if parts.is_empty() || parts[0].is_empty() {
        return None;
    }
    let mut e = Event {
        kind: kind.to_owned(),
        label: parts[0].trim_start().to_owned(),
        ..Event::default()
    };
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
    Some(Event {
        kind: kind.to_owned(),
        path: rest.to_owned(),
        ..Event::default()
    })
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

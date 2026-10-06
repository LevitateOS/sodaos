use super::*;

use std::io::Write;

struct Shared {
    buf: Arc<Mutex<Vec<u8>>>,
}

impl Write for Shared {
    fn write(&mut self, data: &[u8]) -> std::io::Result<usize> {
        self.buf.lock().unwrap().extend_from_slice(data);
        Ok(data.len())
    }

    fn flush(&mut self) -> std::io::Result<()> {
        Ok(())
    }
}

fn renderer(tty: bool) -> (Renderer, Arc<Mutex<Vec<u8>>>) {
    let buf = Arc::new(Mutex::new(Vec::new()));
    let view = Renderer::new(
        Box::new(Shared {
            buf: Arc::clone(&buf),
        }),
        tty,
        80,
    );
    (view, buf)
}

fn text(buf: &Arc<Mutex<Vec<u8>>>) -> String {
    String::from_utf8(buf.lock().unwrap().clone()).unwrap()
}

#[test]
fn parse_controller_events() {
    let e = parse_event("START P1 / Build runtime").unwrap();
    assert_eq!(e.kind, "START");
    assert_eq!(e.label, "P1 / Build runtime");
    let e = parse_event("DONE P1 / Build runtime | phase 00:05:30 | total 00:05:30").unwrap();
    assert_eq!(e.kind, "DONE");
    assert_eq!(e.phase_dur, "00:05:30");
    assert_eq!(e.total_dur, "00:05:30");
    let e = parse_event("FAILED P8 / Media | section 00:18:35 | total 00:21:24").unwrap();
    assert_eq!(e.phase_dur, "00:18:35");
    let e = parse_event("CANDIDATE /out/artifacts/candidate.json").unwrap();
    assert_eq!(e.kind, "CANDIDATE");
    assert!(!e.path.is_empty());
    assert!(parse_event("some builder log line").is_none());
    assert!(parse_event("START").is_none());
}

#[test]
fn padded_wire_labels_normalize_for_identity() {
    // D01-F5: the real wire pads kinds to 8 columns; the parser keeps
    // the first-space split but normalizes label identity so DONE
    // closes its START instead of appending a padded duplicate.
    let e = parse_event("START    P1 / Build runtime").unwrap();
    assert_eq!(e.kind, "START");
    assert_eq!(e.label, "P1 / Build runtime");
    let e = parse_event("DONE     P1 / Build runtime | phase 00:05:30 | total 00:05:30").unwrap();
    assert_eq!(e.label, "P1 / Build runtime");
    assert_eq!(e.phase_dur, "00:05:30");
}

#[test]
fn padded_wire_lifecycle_closes_one_phase() {
    // D01-F5: real emitted START->DONE closes the same phase; the
    // original must not linger as running next to a duplicate.
    let (r, buf) = renderer(true);
    r.feed("START    P1 / Build runtime").unwrap();
    let drawn = text(&buf).len();
    r.feed("DONE     P1 / Build runtime | phase 00:05:30 | total 00:05:30")
        .unwrap();
    // Only the post-DONE redraw matters; the buffer retains history.
    let got = &text(&buf)[drawn..];
    assert!(got.contains("[ok]") && got.contains("00:05:30"), "{got}");
    assert!(!got.contains("(live "), "{got}");
}

#[test]
fn running_phase_shows_live_elapsed() {
    let (r, buf) = renderer(true);
    r.feed("START P1 / Build runtime").unwrap();
    let got = text(&buf);
    assert!(
        got.contains("P1 / Build runtime") && got.contains("(live "),
        "{got}"
    );
    r.feed("DONE P1 / Build runtime | phase 00:05:30 | total 00:05:30")
        .unwrap();
    let got = text(&buf);
    assert!(got.contains("[ok]") && got.contains("00:05:30"), "{got}");
}

#[test]
fn failed_run_prints_why_panel_with_host_paths() {
    for tty in [true, false] {
        let (r, buf) = renderer(tty);
        r.set_out_dir("/home/op/sodaos/.artifacts/releases/isolated/run-01");
        for line in [
            "START P3 / Compile shipping programs",
            "LOG /run/soda-build-source/.artifacts/releases/isolated/run-01/logs/timing.log",
            "FAILED Compile soda-dashboard | section 00:00:00 | total 00:00:00 | reason open /run/go/src/a.go: permission denied",
            "FAILED P3 / Compile shipping programs | phase 00:00:00 | total 00:00:00",
        ] {
            r.feed(line).unwrap();
        }
        r.finish(1).unwrap();
        let got = text(&buf);
        for want in [
            "why: Compile soda-dashboard",
            "cause: open /run/go/src/a.go: permission denied",
            "log: /home/op/sodaos/.artifacts/releases/isolated/run-01/logs/build.log",
        ] {
            assert!(got.contains(want), "tty={tty} misses {want}:\n{got}");
        }
        assert!(
            !got.contains(
                "/run/soda-build-source/.artifacts/releases/isolated/run-01/logs/timing.log"
            ),
            "tty={tty} leaks sandbox path:\n{got}"
        );
    }
}

#[test]
fn failed_run_without_reason_falls_back_to_log() {
    let (r, buf) = renderer(false);
    r.set_out_dir("/out/run-02");
    r.feed("FAILED P1 / Admit | phase 00:00:01 | total 00:00:01")
        .unwrap();
    r.finish(1).unwrap();
    let got = text(&buf);
    assert!(
        got.contains("why: P1 / Admit") && got.contains("cause: see the build log"),
        "{got}"
    );
}

#[test]
fn failed_panel_shows_hint_line() {
    let (r, buf) = renderer(false);
    r.set_out_dir("/out/run-03");
    r.feed("FAILED Compile soda-dashboard | section 00:00:00 | total 00:00:00 | reason x.go: permission denied").unwrap();
    r.finish(1).unwrap();
    let got = text(&buf);
    assert!(
        got.contains("  hint: ") && got.contains("soda-candidate-setup"),
        "{got}"
    );
}

#[test]
fn host_artifact_path_leaves_foreign_paths_alone() {
    let out = "/home/op/sodaos/.artifacts/releases/isolated/run-01";
    assert_eq!(
        host_artifact_path(
            out,
            "/run/soda-build-source/.artifacts/releases/isolated/run-01/artifacts/candidate.json"
        ),
        "/home/op/sodaos/.artifacts/releases/isolated/run-01/artifacts/candidate.json"
    );
    for (o, sandbox) in [
        ("", "/run/soda-build-source/x"),
        (out, "/somewhere/else/media.json"),
        (out, "relative/path.json"),
    ] {
        assert_eq!(host_artifact_path(o, sandbox), sandbox);
    }
}

#[test]
fn pipe_renderer_summarizes() {
    let (r, buf) = renderer(false);
    for line in [
        "START P1 / Build runtime",
        "DONE P1 / Build runtime | phase 00:05:30 | total 00:05:30",
        "CANDIDATE /out/artifacts/candidate.json",
    ] {
        r.feed(line).unwrap();
    }
    r.finish(0).unwrap();
    let got = text(&buf);
    for want in ["P1 / Build runtime", "CANDIDATE", "exit 0"] {
        assert!(got.contains(want), "{got}");
    }
}

#[test]
fn build_lines_collapse_old_successes() {
    let (r, _) = renderer(true);
    let now = Instant::now();
    {
        let mut inner = r.inner.lock().unwrap();
        for i in 0..8 {
            inner.phases.push(Phase {
                label: format!("step {i}"),
                state: "ok".to_owned(),
                dur: "00:00:01".to_owned(),
                started: now,
            });
        }
        inner.phases.push(Phase {
            label: "bad step".to_owned(),
            state: "fail".to_owned(),
            dur: "00:00:02".to_owned(),
            started: now,
        });
        inner.phases.push(Phase {
            label: "live step".to_owned(),
            state: "run".to_owned(),
            dur: String::new(),
            started: now,
        });
        let joined = build_lines(&inner, now).join("\n");
        for want in ["… 3 earlier steps done", "step 7", "bad step", "live step"] {
            assert!(joined.contains(want), "{joined}");
        }
        for hidden in ["step 0", "step 1", "step 2"] {
            assert!(!joined.contains(hidden), "{joined}");
        }
    }
}

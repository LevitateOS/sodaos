use super::*;

use crate::candidate::tests::base_options;
use std::io::Cursor;
use std::sync::{Arc, Mutex};

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

fn scripted(o: &Options, input: &str) -> (Options, String) {
    let mut o = o.clone();
    let owned: Vec<String> = input.lines().map(|l| format!("{l}\n")).collect();
    let joined = owned.join("");
    let reader: Box<dyn BufRead> = Box::new(Cursor::new(joined.into_bytes()));
    let buf = Arc::new(Mutex::new(Vec::new()));
    let writer: Box<dyn Write> = Box::new(Shared {
        buf: Arc::clone(&buf),
    });
    // Nothing on disk matches the standard paths in these scripts.
    let probe: Box<dyn Fn(&str) -> bool> = Box::new(|_| false);
    let mut p = Prompter::with_exists(reader, writer, probe);
    let err = match p.overview(&mut o, &|| "/suggested/out".to_owned()) {
        Ok(()) => "<nil>".to_owned(),
        Err(e) => e,
    };
    let screen = String::from_utf8(buf.lock().unwrap().clone()).unwrap();
    (o, format!("{screen}\x00{err}"))
}

fn scripted_exists(o: &Options, input: &str) -> (Options, String) {
    let mut o = o.clone();
    let owned: Vec<String> = input.lines().map(|l| format!("{l}\n")).collect();
    let joined = owned.join("");
    let reader: Box<dyn BufRead> = Box::new(Cursor::new(joined.into_bytes()));
    let buf = Arc::new(Mutex::new(Vec::new()));
    let writer: Box<dyn Write> = Box::new(Shared {
        buf: Arc::clone(&buf),
    });
    let probe: Box<dyn Fn(&str) -> bool> = Box::new(|p: &str| {
        p == "/usr/local/lib/soda/soda-build"
            || p == "/var/lib/soda-candidate-authority/worker.json"
    });
    let mut p = Prompter::with_exists(reader, writer, probe);
    let err = match p.overview(&mut o, &|| "/suggested/out".to_owned()) {
        Ok(()) => "<nil>".to_owned(),
        Err(e) => e,
    };
    let screen = String::from_utf8(buf.lock().unwrap().clone()).unwrap();
    (o, format!("{screen}\x00{err}"))
}

fn fresh_options() -> Options {
    let mut o = base_options();
    let dir = std::env::temp_dir().join(format!("soda-reltools-ov-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&dir);
    std::fs::create_dir_all(&dir).unwrap();
    o.out = format!("{}/fresh", dir.to_string_lossy());
    o
}

#[test]
fn controller_args_cover_modes() {
    let joined = controller_args(&base_options()).join(" ");
    for want in ["--development", "--target", "media", "--rootfs-base-url"] {
        assert!(joined.contains(want), "{joined}");
    }
    let mut o = base_options();
    o.mode.clear();
    let joined = controller_args(&o).join(" ");
    assert!(joined.contains("--target media"), "{joined}");
    assert!(
        joined.contains(&format!("--rootfs-base-url {}", o.rootfs_url)),
        "{joined}"
    );
}

#[test]
fn overview_starts_when_valid() {
    let (o, screen) = scripted(&fresh_options(), "go\n");
    assert!(screen.ends_with("<nil>"), "{screen}");
    assert!(validate_resolved(&o).is_ok());
}

#[test]
fn overview_edits_field() {
    let mut o = fresh_options();
    o.rootfs_url = "http://old.invalid".to_owned();
    // Field 5 is the media rootfs URL; then start.
    let (o, _) = scripted(&o, "5\nhttp://new.invalid\n\ngo\n");
    assert_eq!(o.rootfs_url, "http://new.invalid");
}

#[test]
fn overview_blocks_bad_start_without_losing_answers() {
    let mut o = fresh_options();
    o.mode = "candidate".to_owned();
    o.rootfs_url = "http://fixture:8080".to_owned();
    let (_, screen) = scripted(&o, "go\nquit\n");
    assert!(screen.contains("Cannot start:"), "{screen}");
    assert!(screen.contains("aborted by operator"), "{screen}");
}

#[test]
fn overview_prefills_standard_paths() {
    let mut o = Options {
        arch: "x86_64".to_owned(),
        mode: "media".to_owned(),
        out: format!("{}/fresh", std::env::temp_dir().to_string_lossy()),
        rootfs_url: "http://fixture:8080".to_owned(),
        ..Options::default()
    };
    // Point the output at a fresh temp leaf with an existing parent.
    let dir = std::env::temp_dir().join(format!("soda-reltools-ovp-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&dir);
    std::fs::create_dir_all(&dir).unwrap();
    o.out = format!("{}/fresh", dir.to_string_lossy());
    let (got, screen) = scripted_exists(&o, "go\n");
    assert!(screen.ends_with("<nil>"), "{screen}");
    assert_eq!(got.controller, "/usr/local/lib/soda/soda-build");
    assert_eq!(
        got.worker_config,
        "/var/lib/soda-candidate-authority/worker.json"
    );
}

#[test]
fn overview_defaults_fixture_rootfs_url() {
    let dir = std::env::temp_dir().join(format!("soda-reltools-ovd-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&dir);
    std::fs::create_dir_all(&dir).unwrap();
    let o = Options {
        arch: "x86_64".to_owned(),
        out: format!("{}/fresh", dir.to_string_lossy()),
        controller: "/a".to_owned(),
        worker_config: "/b".to_owned(),
        repo_prefix: "x".to_owned(),
        ..Options::default()
    };
    let (got, _) = scripted(&o, "quit\n");
    assert_eq!(got.mode, "media");
    assert_eq!(got.rootfs_url, FIXTURE_ROOTFS_URL);
}

#[test]
fn mode_switch_restores_fixture_url() {
    let dir = std::env::temp_dir().join(format!("soda-reltools-ovm-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&dir);
    std::fs::create_dir_all(&dir).unwrap();
    let mut o = Options {
        arch: "x86_64".to_owned(),
        out: format!("{}/fresh", dir.to_string_lossy()),
        controller: "/a".to_owned(),
        worker_config: "/b".to_owned(),
        repo_prefix: "x".to_owned(),
        ..Options::default()
    };
    o.mode = "candidate".to_owned();
    let (got, _) = scripted(&o, "1\n2\nquit\n");
    assert_eq!(got.mode, "media");
    assert_eq!(got.rootfs_url, FIXTURE_ROOTFS_URL);
}

#[test]
fn ask_out_rejects_then_accepts() {
    let dir = std::env::temp_dir().join(format!("soda-reltools-ask-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&dir);
    std::fs::create_dir_all(&dir).unwrap();
    let fresh = format!("{}/fresh", dir.to_string_lossy());
    let input = format!("rel\n{fresh}\n");
    let reader: Box<dyn BufRead> = Box::new(Cursor::new(input.into_bytes()));
    let buf = Arc::new(Mutex::new(Vec::new()));
    let writer: Box<dyn Write> = Box::new(Shared {
        buf: Arc::clone(&buf),
    });
    let mut p = Prompter::new(reader, writer);
    let got = p.ask_out("Fresh output directory", "rel").unwrap();
    assert_eq!(got, fresh);
    let screen = String::from_utf8(buf.lock().unwrap().clone()).unwrap();
    assert!(screen.contains("Absolute path required."), "{screen}");
}

#[test]
fn suggested_out_follows_worker_rule() {
    let leaf = suggest_out()
        .rsplit('/')
        .next()
        .unwrap_or_default()
        .to_owned();
    assert!(valid_out_leaf(&leaf), "{leaf}");
    assert_eq!(leaf.len(), 16);
}

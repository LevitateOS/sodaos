//! `soda-candidate` interactive prompts (Go `tools/soda-candidate`
//! `prompts.go` plus the `preflight.go` answer checks).

use std::io::{BufRead, Write};

use crate::candidate::{valid_out_leaf, validate_resolved, Options};

pub const FIXTURE_ROOTFS_URL: &str = "http://127.0.0.1:8080";

pub const STANDARD_CONTROLLER_PATHS: &[&str] = &["/usr/local/lib/soda/soda-build"];
pub const STANDARD_WORKER_CONFIG_PATHS: &[&str] =
    &["/var/lib/soda-candidate-authority/worker.json"];

pub const MODE_OPTIONS: &[&str] = &[
    "Development candidate (no media, no signing)",
    "Development media (installer ISO, fixture rootfs URL)",
];

pub fn mode_label(mode: &str) -> &'static str {
    if mode == "candidate" {
        MODE_OPTIONS[0]
    } else {
        MODE_OPTIONS[1]
    }
}

pub fn mode_index(mode: &str) -> usize {
    for (i, m) in ["candidate", "media"].iter().enumerate() {
        if mode == *m {
            return i;
        }
    }
    1
}

pub fn default_path(current: &str, exists: &dyn Fn(&str) -> bool, candidates: &[&str]) -> String {
    if !current.is_empty() {
        return current.to_owned();
    }
    for c in candidates {
        if exists(c) {
            return (*c).to_owned();
        }
    }
    String::new()
}

fn file_exists(path: &str) -> bool {
    std::fs::metadata(path).is_ok()
}

pub fn suggest_out() -> String {
    let cwd = std::env::current_dir()
        .map(|p| p.to_string_lossy().into_owned())
        .unwrap_or_default();
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap_or_default();
    let leaf = utc_stamp(now.as_secs());
    format!("{cwd}/.artifacts/releases/isolated/{leaf}")
}

fn utc_stamp(epoch_secs: u64) -> String {
    // Civil-date conversion (Howard Hinnant's algorithm) for the
    // `20060102t150405z` leaf; lowercase by the worker name rule.
    let days = (epoch_secs / 86_400) as i64;
    let secs = epoch_secs % 86_400;
    let z = days + 719_468;
    let era = z.div_euclid(146_097);
    let doe = z.rem_euclid(146_097);
    let yoe = (doe - doe / 1460 + doe / 36_524 - doe / 146_096) / 365;
    let mut year = yoe + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let day = (doy - (153 * mp + 2) / 5 + 1) as u64;
    let month = (if mp < 10 { mp + 3 } else { mp - 9 }) as u64;
    year += i64::from(mp >= 10);
    format!(
        "{year:04}{month:02}{day:02}t{:02}{:02}{:02}z",
        secs / 3600,
        secs / 60 % 60,
        secs % 60
    )
}

/// Translate answers into the admitted controller flags.
pub fn controller_args(o: &Options) -> Vec<String> {
    let mut args = vec![
        "--worker-config".to_owned(),
        o.worker_config.clone(),
        "--arch".to_owned(),
        o.arch.clone(),
        "--out".to_owned(),
        o.out.clone(),
        "--repository-prefix".to_owned(),
        o.repo_prefix.clone(),
    ];
    let mut target = o.mode.clone();
    if target.is_empty() {
        target = "media".to_owned();
    }
    args.push("--development".to_owned());
    args.push("--target".to_owned());
    args.push(target.clone());
    if !o.compression.is_empty() {
        args.push("--media-compression".to_owned());
        args.push(o.compression.clone());
    }
    if target == "media" {
        args.push("--rootfs-base-url".to_owned());
        args.push(o.rootfs_url.clone());
    }
    args
}

pub struct Prompter<'a> {
    input: Box<dyn BufRead + 'a>,
    output: Box<dyn Write + 'a>,
    exists: Box<dyn Fn(&str) -> bool + 'a>,
}

impl<'a> Prompter<'a> {
    pub fn new(input: Box<dyn BufRead + 'a>, output: Box<dyn Write + 'a>) -> Self {
        Prompter {
            input,
            output,
            exists: Box::new(file_exists),
        }
    }

    pub fn with_exists(
        input: Box<dyn BufRead + 'a>,
        output: Box<dyn Write + 'a>,
        exists: Box<dyn Fn(&str) -> bool + 'a>,
    ) -> Self {
        Prompter {
            input,
            output,
            exists,
        }
    }

    fn line(&mut self, prompt: &str, def: &str) -> Result<String, String> {
        if def.is_empty() {
            write!(self.output, "{prompt}: ").map_err(|e| e.to_string())?;
        } else {
            write!(self.output, "{prompt} [{def}]: ").map_err(|e| e.to_string())?;
        }
        self.output.flush().map_err(|e| e.to_string())?;
        let mut s = String::new();
        match self.input.read_line(&mut s) {
            Ok(0) | Err(_) => {
                return Err("input ended; rerun with flags or --non-interactive".to_owned());
            }
            Ok(_) => {}
        }
        let s = s.trim().to_owned();
        if s.is_empty() {
            return Ok(def.to_owned());
        }
        Ok(s)
    }

    fn choice(&mut self, prompt: &str, options: &[&str], def: usize) -> Result<usize, String> {
        for (i, o) in options.iter().enumerate() {
            writeln!(self.output, "  {}) {o}", i + 1).map_err(|e| e.to_string())?;
        }
        loop {
            let s = self.line(prompt, &(def + 1).to_string())?;
            match s.trim().parse::<usize>() {
                Ok(n) if n >= 1 && n <= options.len() => return Ok(n - 1),
                _ => {
                    writeln!(self.output, "Enter a number 1-{}.", options.len())
                        .map_err(|e| e.to_string())?;
                }
            }
        }
    }

    fn edit_mode(&mut self, o: &mut Options) -> Result<(), String> {
        let sel = self.choice("Build mode", MODE_OPTIONS, mode_index(&o.mode))?;
        o.mode = ["candidate", "media"][sel].to_owned();
        if o.mode == "media" && o.rootfs_url.is_empty() {
            o.rootfs_url = FIXTURE_ROOTFS_URL.to_owned();
        }
        Ok(())
    }

    fn edit_out(&mut self, o: &mut Options) -> Result<(), String> {
        o.out = self.ask_out("Fresh output directory", &o.out.clone())?;
        Ok(())
    }

    fn edit_controller(&mut self, o: &mut Options) -> Result<(), String> {
        o.controller =
            self.ask_absolute("Admitted soda-build executable", &o.controller.clone())?;
        Ok(())
    }

    fn edit_worker_config(&mut self, o: &mut Options) -> Result<(), String> {
        o.worker_config =
            self.ask_absolute("Restricted worker config", &o.worker_config.clone())?;
        Ok(())
    }

    fn edit_rootfs_url(&mut self, o: &mut Options) -> Result<(), String> {
        o.rootfs_url = self.ask_non_empty("Public rootfs base URL", &o.rootfs_url.clone())?;
        Ok(())
    }

    fn edit_fast_compress(&mut self, o: &mut Options) -> Result<(), String> {
        let def = if o.compression == "fast" { "y" } else { "n" };
        o.compression = self.ask_fast(def)?;
        Ok(())
    }

    fn edit_repo_prefix(&mut self, o: &mut Options) -> Result<(), String> {
        o.repo_prefix = self.ask_non_empty(
            "Image repository prefix (intent only; no publication)",
            &o.repo_prefix.clone(),
        )?;
        Ok(())
    }

    pub fn ask_absolute(&mut self, prompt: &str, def: &str) -> Result<String, String> {
        let mut def = def.to_owned();
        loop {
            let s = self.line(prompt, &def)?;
            def = s.clone();
            if !s.starts_with('/') {
                writeln!(self.output, "Absolute path required.").map_err(|e| e.to_string())?;
                continue;
            }
            return Ok(s);
        }
    }

    pub fn ask_non_empty(&mut self, prompt: &str, def: &str) -> Result<String, String> {
        let mut def = def.to_owned();
        loop {
            let s = self.line(prompt, &def)?;
            def = s.clone();
            if s.trim().is_empty() {
                writeln!(self.output, "A value is required here.").map_err(|e| e.to_string())?;
                continue;
            }
            return Ok(s);
        }
    }

    pub fn ask_out(&mut self, prompt: &str, def: &str) -> Result<String, String> {
        let mut def = def.to_owned();
        loop {
            let s = self.line(prompt, &def)?;
            def = s.clone();
            let leaf = s.rsplit('/').next().unwrap_or_default();
            let parent = match s.rfind('/') {
                Some(0) => "/".to_owned(),
                Some(i) => s[..i].to_owned(),
                None => ".".to_owned(),
            };
            let reason = if !s.starts_with('/') {
                Some("Absolute path required.".to_owned())
            } else if !valid_out_leaf(leaf) {
                Some("Lowercase letters, digits, or dashes only (worker name rule).".to_owned())
            } else if !parent_dir_exists(&parent) {
                Some(format!("Parent {parent} must already exist."))
            } else if !path_absent(&s) {
                Some("That output exists; each attempt needs a fresh directory.".to_owned())
            } else {
                None
            };
            match reason {
                Some(r) => {
                    writeln!(self.output, "{r}").map_err(|e| e.to_string())?;
                }
                None => return Ok(s),
            }
        }
    }

    fn ask_fast(&mut self, def: &str) -> Result<String, String> {
        let mut def = def.to_owned();
        loop {
            let s = self.line("Fast iteration compression (dev ISO only) [y/n]", &def)?;
            match s.trim().to_lowercase().as_str() {
                "y" | "yes" => return Ok("fast".to_owned()),
                "n" | "no" => return Ok(String::new()),
                _ => {
                    def = s;
                    writeln!(self.output, "Answer y or n.").map_err(|e| e.to_string())?;
                }
            }
        }
    }

    fn show_fast_compress(o: &Options) -> String {
        if o.compression == "fast" {
            "yes".to_owned()
        } else {
            "no".to_owned()
        }
    }

    fn render_overview(&mut self, o: &Options) -> Result<(), String> {
        writeln!(
            self.output,
            "\nsoda-candidate | {} | arch {} (this host)",
            mode_label(&o.mode),
            o.arch
        )
        .map_err(|e| e.to_string())?;
        for (i, (label, show, status)) in overview_rows(o).iter().enumerate() {
            let extra = match status {
                Some(s) => format!("  [{s}]"),
                None => String::new(),
            };
            writeln!(self.output, "  {}) {label:<13} {show}{extra}", i + 1)
                .map_err(|e| e.to_string())?;
        }
        Ok(())
    }

    /// Overview screen: edit fields by number, start with `go`.
    pub fn overview(
        &mut self,
        o: &mut Options,
        suggest: &dyn Fn() -> String,
    ) -> Result<(), String> {
        default_overview(o, suggest, &self.exists_as_fn());
        loop {
            self.render_overview(o)?;
            let s = self.line("Number to edit, 'go' to start, 'quit' to abort", "go")?;
            match s.trim().to_lowercase().as_str() {
                "go" | "run" | "" => {
                    if let Err(e) = validate_resolved(o) {
                        writeln!(self.output, "Cannot start: {e}").map_err(|e| e.to_string())?;
                        continue;
                    }
                    return Ok(());
                }
                "quit" | "q" | "abort" => return Err("aborted by operator".to_owned()),
                _ => {
                    let count = overview_rows(o).len();
                    match s.trim().parse::<usize>() {
                        Ok(n) if n >= 1 && n <= count => self.edit_field(o, n - 1)?,
                        _ => {
                            writeln!(self.output, "Type a field number, 'go', or 'quit'.")
                                .map_err(|e| e.to_string())?;
                        }
                    }
                }
            }
        }
    }

    fn exists_as_fn(&self) -> Box<dyn Fn(&str) -> bool + '_> {
        let exists: &dyn Fn(&str) -> bool = &self.exists;
        Box::new(move |p| exists(p))
    }

    fn edit_field(&mut self, o: &mut Options, index: usize) -> Result<(), String> {
        // Field order mirrors `overview_rows`.
        let media = o.mode == "media";
        match (index, media) {
            (0, _) => self.edit_mode(o),
            (1, _) => self.edit_out(o),
            (2, _) => self.edit_controller(o),
            (3, _) => self.edit_worker_config(o),
            (4, true) => self.edit_rootfs_url(o),
            (5, true) => self.edit_fast_compress(o),
            (4, false) | (6, true) => self.edit_repo_prefix(o),
            _ => Ok(()),
        }
    }
}

fn overview_rows(o: &Options) -> Vec<(String, String, Option<String>)> {
    let mut rows = vec![
        ("mode".to_owned(), o.mode.clone(), None),
        ("output".to_owned(), o.out.clone(), Some(out_status(&o.out))),
        ("controller".to_owned(), o.controller.clone(), None),
        ("worker config".to_owned(), o.worker_config.clone(), None),
    ];
    if o.mode == "media" {
        rows.push(("rootfs URL".to_owned(), o.rootfs_url.clone(), None));
        rows.push((
            "fast compress".to_owned(),
            Prompter::show_fast_compress(o),
            None,
        ));
    }
    rows.push(("repo prefix".to_owned(), o.repo_prefix.clone(), None));
    rows
}

fn out_status(out: &str) -> String {
    if !out.starts_with('/') {
        "✗ not absolute".to_owned()
    } else {
        let parent = match out.rfind('/') {
            Some(0) => "/",
            Some(i) => &out[..i],
            None => ".",
        };
        if !parent_dir_exists(parent) {
            "✗ parent missing".to_owned()
        } else if !path_absent(out) {
            "✗ already exists".to_owned()
        } else {
            "✓ fresh".to_owned()
        }
    }
}

fn parent_dir_exists(path: &str) -> bool {
    matches!(std::fs::metadata(path), Ok(st) if st.file_type().is_dir())
}

fn path_absent(path: &str) -> bool {
    std::fs::symlink_metadata(path).is_err()
}

fn default_overview(o: &mut Options, suggest: &dyn Fn() -> String, exists: &dyn Fn(&str) -> bool) {
    if o.mode.is_empty() {
        o.mode = "media".to_owned();
    }
    if o.mode == "media" && o.rootfs_url.is_empty() {
        o.rootfs_url = FIXTURE_ROOTFS_URL.to_owned();
    }
    if o.out.is_empty() {
        o.out = suggest();
    }
    o.controller = default_path(&o.controller.clone(), exists, STANDARD_CONTROLLER_PATHS);
    o.worker_config = default_path(
        &o.worker_config.clone(),
        exists,
        STANDARD_WORKER_CONFIG_PATHS,
    );
}

/// Real-terminal overview entry used by the binary.
pub fn prompter_overview(o: &mut Options, suggest: fn() -> String) -> Result<(), String> {
    let stdin = std::io::stdin();
    let input: Box<dyn BufRead> = Box::new(std::io::BufReader::new(stdin.lock()));
    // The Go owner prompts on stderr.
    let output: Box<dyn Write> = Box::new(std::io::stderr());
    let mut p = Prompter::new(input, output);
    p.overview(o, &suggest)
}

#[cfg(test)]
mod tests {
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
}

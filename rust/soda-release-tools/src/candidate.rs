//! `soda-candidate` wrapper core (Go `tools/soda-candidate` `main.go`):
//! options, flag parsing, answer validation, and checkout preflight.

use std::path::Path;
use std::process::Command;

use crate::candidate_controller::{file_built_rootfs, maybe_serve_fixture, start_controller_run};
use crate::candidate_display::is_terminal;
use crate::candidate_fixture::default_rootfs_dir;
use crate::candidate_prompts::{prompter_overview, suggest_out};
use crate::goflag::{self, FlagKind, FlagSpec};

#[derive(Debug, Clone, Default)]
pub struct Options {
    pub controller: String,
    pub worker_config: String,
    pub arch: String,
    pub out: String,
    pub mode: String,
    pub compression: String,
    pub rootfs_url: String,
    pub rootfs_dir: String,
    pub repo_prefix: String,
    pub non_interactive: bool,
}

pub fn flag_specs() -> Vec<FlagSpec> {
    vec![
        FlagSpec {
            name: "controller",
            kind: FlagKind::Text,
            usage: "admitted soda-build executable (asked when empty)",
            default_text: "",
        },
        FlagSpec {
            name: "worker-config",
            kind: FlagKind::Text,
            usage: "restricted worker configuration (asked when empty)",
            default_text: "",
        },
        FlagSpec {
            name: "arch",
            kind: FlagKind::Text,
            usage: "matching native x86_64",
            default_text: ARCH_DEFAULT,
        },
        FlagSpec {
            name: "out",
            kind: FlagKind::Text,
            usage: "fresh output below .artifacts/releases (asked when empty)",
            default_text: "",
        },
        FlagSpec {
            name: "mode",
            kind: FlagKind::Text,
            usage: "candidate or media (asked when empty)",
            default_text: "",
        },
        FlagSpec {
            name: "media-compression",
            kind: FlagKind::Text,
            usage: "fast: development media only",
            default_text: "",
        },
        FlagSpec {
            name: "rootfs-base-url",
            kind: FlagKind::Text,
            usage: "public base URL for the hash-named rootfs file",
            default_text: "",
        },
        FlagSpec {
            name: "rootfs-dir",
            kind: FlagKind::Text,
            usage:
                "pickup folder served for loopback development media (default .artifacts/rootfs)",
            default_text: "",
        },
        FlagSpec {
            name: "repository-prefix",
            kind: FlagKind::Text,
            usage: "intended image repositories; no publication",
            default_text: "ghcr.io/levitateos/sodaos",
        },
        FlagSpec {
            name: "non-interactive",
            kind: FlagKind::Bool,
            usage: "require all flags; timestamped log output",
            default_text: "",
        },
    ]
}

// The `--arch` default is the native arch, always `x86_64` on the only
// supported platform; parse fails before specs exist anywhere else.
const ARCH_DEFAULT: &str = "x86_64";

pub fn usage() -> String {
    goflag::print_defaults("soda-candidate", &flag_specs())
}

pub fn native_arch() -> Result<String, String> {
    if std::env::consts::ARCH == "x86_64" {
        Ok("x86_64".to_owned())
    } else {
        Err(format!(
            "unsupported native {}: build requires matching x86_64",
            std::env::consts::ARCH
        ))
    }
}

pub fn validate_mode_flag(mode: &str) -> Result<(), String> {
    if !mode.is_empty() && mode != "candidate" && mode != "media" {
        return Err("--mode accepts candidate or media".to_owned());
    }
    Ok(())
}

pub fn validate_arch_flag(arch: &str) -> Result<(), String> {
    if arch != "x86_64" {
        return Err("matching native x86_64 required".to_owned());
    }
    Ok(())
}

pub fn parse_options(args: &[String]) -> Result<Options, String> {
    let _native = native_arch()?;
    let specs = flag_specs();
    let outcome = goflag::parse(&specs, args)?;
    if !outcome.positionals.is_empty() {
        return Err("unexpected positional arguments".to_owned());
    }
    let o = Options {
        controller: outcome.text("controller"),
        worker_config: outcome.text("worker-config"),
        arch: outcome.text("arch"),
        out: outcome.text("out"),
        mode: outcome.text("mode"),
        compression: outcome.text("media-compression"),
        rootfs_url: outcome.text("rootfs-base-url"),
        rootfs_dir: outcome.text("rootfs-dir"),
        repo_prefix: outcome.text("repository-prefix"),
        non_interactive: outcome.boolean("non-interactive"),
    };
    validate_mode_flag(&o.mode)?;
    validate_arch_flag(&o.arch)?;
    Ok(o)
}

pub fn valid_out_leaf(leaf: &str) -> bool {
    if leaf.is_empty() || leaf.len() > 48 {
        return false;
    }
    leaf.bytes()
        .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'-')
}

fn validate_common_paths(o: &Options) -> Result<(), String> {
    if o.controller.is_empty() || !o.controller.starts_with('/') {
        return Err("absolute controller path required".to_owned());
    }
    if o.worker_config.is_empty() || !o.worker_config.starts_with('/') {
        return Err("absolute worker config path required".to_owned());
    }
    if o.out.is_empty() || !o.out.starts_with('/') {
        return Err("absolute --out required".to_owned());
    }
    let leaf = o.out.rsplit('/').next().unwrap_or_default();
    if !valid_out_leaf(leaf) {
        return Err(
            "output name must be lowercase letters, digits, or dashes (worker name rule)"
                .to_owned(),
        );
    }
    Ok(())
}

fn validate_candidate(o: &Options) -> Result<(), String> {
    if !o.rootfs_url.is_empty() || !o.compression.is_empty() {
        return Err("candidate refuses media-only inputs".to_owned());
    }
    Ok(())
}

fn validate_media(o: &Options) -> Result<(), String> {
    if !o.compression.is_empty() && o.compression != "fast" {
        return Err("--media-compression accepts only fast with development media".to_owned());
    }
    if o.rootfs_url.is_empty() {
        return Err("media requires the rootfs base URL".to_owned());
    }
    Ok(())
}

pub fn validate_resolved(o: &Options) -> Result<(), String> {
    validate_common_paths(o)?;
    match o.mode.as_str() {
        "candidate" => validate_candidate(o),
        "media" => validate_media(o),
        _ => Err("choose a build mode: candidate or media".to_owned()),
    }
}

pub fn check_checkout_root() -> Result<(), String> {
    match std::fs::metadata("go.mod") {
        Ok(st) if st.file_type().is_file() => Ok(()),
        _ => Err("run soda-candidate from the checkout root (~/Projects/sodaos)".to_owned()),
    }
}

pub fn dirty_files(out: &[u8]) -> Vec<String> {
    let mut lines = Vec::new();
    for line in String::from_utf8_lossy(out).split('\n') {
        let trimmed = line.trim();
        if !trimmed.is_empty() {
            lines.push(trimmed.to_owned());
        }
    }
    lines
}

pub fn check_clean_tree() -> Result<(), String> {
    let output = Command::new("git")
        .arg("status")
        .arg("--porcelain")
        .output();
    let output = match output {
        Ok(o) if o.status.success() => o,
        _ => return Err("source must be a clean git checkout".to_owned()),
    };
    let lines = dirty_files(&output.stdout);
    if lines.is_empty() {
        return Ok(());
    }
    let mut shown = lines
        .iter()
        .take(10)
        .cloned()
        .collect::<Vec<_>>()
        .join("\n");
    let mut suffix = String::new();
    if lines.len() > 10 {
        suffix = format!("\n... and {} more", lines.len() - 10);
    }
    shown.push_str(&suffix);
    Err(format!(
        "controller requires committed source ({} dirty file(s)); commit or stash first:\n{}",
        lines.len(),
        shown
    ))
}

pub fn check_fresh_out(out: &str) -> Result<(), String> {
    if Path::new(out).symlink_metadata().is_ok() {
        return Err(format!(
            "output {out} exists; choose a fresh --out per attempt"
        ));
    }
    let parent = match out.rfind('/') {
        Some(0) => "/",
        Some(i) => &out[..i],
        None => ".",
    };
    match std::fs::metadata(parent) {
        Ok(st) if st.file_type().is_dir() => Ok(()),
        _ => Err(format!("output parent {parent} must already exist")),
    }
}

pub fn preflight(o: &Options) -> Result<(), String> {
    check_checkout_root()?;
    let native = native_arch()?;
    if o.arch != native {
        return Err(format!("arch {} is not this native {native} host", o.arch));
    }
    check_clean_tree()?;
    check_fresh_out(&o.out)
}

pub fn monotonic_ns() -> Result<i64, String> {
    let mut ts = libc::timespec {
        tv_sec: 0,
        tv_nsec: 0,
    };
    if unsafe { libc::clock_gettime(libc::CLOCK_MONOTONIC, &mut ts) } != 0 {
        return Err(std::io::Error::last_os_error().to_string());
    }
    ts.tv_sec
        .checked_mul(1_000_000_000)
        .and_then(|s| s.checked_add(ts.tv_nsec))
        .ok_or_else(|| "monotonic clock overflow".to_owned())
}

pub fn describe(o: &Options) -> String {
    format!(
        "mode {} | arch {} | out {} | run {}",
        crate::candidate_prompts::mode_label(&o.mode),
        o.arch,
        o.out,
        o.controller
    )
}

pub fn resolve_options(
    args: &[String],
    stdin_is_terminal: bool,
    stderr_is_terminal: bool,
    prompt: &mut dyn FnMut(&mut Options) -> Result<(), String>,
) -> Result<Options, String> {
    let mut o = parse_options(args)?;
    if !stdin_is_terminal || !stderr_is_terminal || o.non_interactive {
        if o.mode.is_empty() {
            return Err("choose --mode candidate or media (or run on a terminal)".to_owned());
        }
        return Ok(o);
    }
    prompt(&mut o)?;
    Ok(o)
}

pub fn ready_run(o: &mut Options) -> Result<(), String> {
    preflight(o)?;
    default_rootfs_dir(o)
}

/// Exit-code-carrying controller failure.
#[derive(Debug, Clone)]
pub struct ExitError {
    pub code: i32,
}

impl std::fmt::Display for ExitError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        if self.code == 2 {
            write!(
                f,
                "controller incomplete: no qualified release without final signing"
            )
        } else {
            write!(f, "controller exited {}", self.code)
        }
    }
}

pub fn run_candidate() -> Result<(), CandidateError> {
    let argv: Vec<String> = std::env::args().collect();
    let args = if argv.len() > 1 { &argv[1..] } else { &[] };
    // ContinueOnError: flag failures print error + usage, then surface
    // through the run error like the Go owner.
    let specs = flag_specs();
    if native_arch().is_err() {
        return Err(CandidateError::Message(native_arch().unwrap_err()));
    }
    if let Err(e) = goflag::parse(&specs, args) {
        let mut text = String::new();
        if e != goflag::ERR_HELP {
            text.push_str(&format!("{e}\n"));
        }
        text.push_str(&usage());
        eprint!("{text}");
        return Err(CandidateError::Message(e));
    }
    let stdin_tty = is_terminal(libc::STDIN_FILENO);
    let stderr_tty = is_terminal(libc::STDERR_FILENO);
    let mut prompt = |o: &mut Options| prompter_overview(o, suggest_out);
    let mut o = resolve_options(args, stdin_tty, stderr_tty, &mut prompt)
        .map_err(CandidateError::Message)?;
    validate_resolved(&o).map_err(CandidateError::Message)?;
    ready_run(&mut o).map_err(CandidateError::Message)?;
    let stop = maybe_serve_fixture(&o).map_err(CandidateError::Message)?;
    let controller = start_controller_run(&o).map_err(CandidateError::Message)?;
    let code = controller.wait().map_err(CandidateError::Message)?;
    stop();
    if code != 0 {
        return Err(CandidateError::Exit(ExitError { code }));
    }
    let mut stderr = std::io::stderr().lock();
    file_built_rootfs(&o, &mut stderr).map_err(CandidateError::Message)
}

#[derive(Debug)]
pub enum CandidateError {
    Message(String),
    Exit(ExitError),
}

impl std::fmt::Display for CandidateError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            CandidateError::Message(m) => write!(f, "{m}"),
            CandidateError::Exit(e) => write!(f, "{e}"),
        }
    }
}

pub fn main() {
    if let Err(err) = run_candidate() {
        eprintln!("soda-candidate: {err}");
        match err {
            CandidateError::Exit(e) => std::process::exit(e.code),
            CandidateError::Message(_) => std::process::exit(1),
        }
    }
}

#[cfg(test)]
pub mod tests {
    use super::*;

    pub fn base_options() -> Options {
        Options {
            controller: "/admitted/soda-build".to_owned(),
            worker_config: "/restricted/worker.json".to_owned(),
            arch: "x86_64".to_owned(),
            out: "/source/.artifacts/releases/isolated/test".to_owned(),
            mode: "media".to_owned(),
            rootfs_url: "http://fixture:8080".to_owned(),
            repo_prefix: "ghcr.io/levitateos/sodaos".to_owned(),
            ..Options::default()
        }
    }

    #[test]
    fn arch_flag_admits_only_x86_64() {
        assert!(validate_arch_flag("x86_64").is_ok());
        for arch in ["", "aarch64", "amd64", "arm64"] {
            assert!(validate_arch_flag(arch).is_err(), "{arch}");
        }
    }

    #[test]
    fn resolved_boundaries() {
        let mut o = base_options();
        assert!(validate_resolved(&o).is_ok());
        o.mode = "candidate".to_owned();
        assert_eq!(
            validate_resolved(&o).unwrap_err(),
            "candidate refuses media-only inputs"
        );
        o = base_options();
        o.rootfs_url.clear();
        assert_eq!(
            validate_resolved(&o).unwrap_err(),
            "media requires the rootfs base URL"
        );
        o.mode = "production".to_owned();
        assert!(validate_resolved(&o).is_err());
    }

    #[test]
    fn out_leaf_follows_worker_name_rule() {
        for leaf in ["20260915t212541z", "manual-01", "a"] {
            assert!(valid_out_leaf(leaf), "{leaf}");
        }
        for leaf in [
            "",
            "20260915T212541Z",
            "has space",
            "UPPER",
            "under_score",
            &"a".repeat(49),
        ] {
            assert!(!valid_out_leaf(leaf), "{leaf}");
        }
        let mut o = base_options();
        o.out = "/source/.artifacts/releases/isolated/20260915T212541Z".to_owned();
        let err = validate_resolved(&o).unwrap_err();
        assert!(err.contains("lowercase"), "{err}");
    }

    #[test]
    fn dirty_files_skips_blanks() {
        let got = dirty_files(b" M tools/soda-candidate/main.go\n\n?? scratch\n");
        assert_eq!(got, vec!["M tools/soda-candidate/main.go", "?? scratch"]);
        assert!(dirty_files(b"").is_empty());
    }

    #[test]
    fn resolve_options_defers_worker_admission_to_controller() {
        let args = [
            "--controller",
            "/admitted/soda-build",
            "--worker-config",
            "/nonexistent/worker.json",
            "--arch",
            "x86_64",
            "--out",
            "/tmp/fresh-out-01",
            "--mode",
            "candidate",
            "--non-interactive",
        ]
        .into_iter()
        .map(str::to_owned)
        .collect::<Vec<_>>();
        let mut prompt = |_: &mut Options| -> Result<(), String> {
            panic!("must not prompt");
        };
        let o = resolve_options(&args, false, false, &mut prompt).unwrap();
        assert_eq!(o.worker_config, "/nonexistent/worker.json");
    }

    #[test]
    fn check_fresh_out_matrix() {
        let scratch =
            std::env::temp_dir().join(format!("soda-reltools-fresh-{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&scratch);
        std::fs::create_dir_all(&scratch).unwrap();
        let fresh = scratch.join("fresh-01");
        assert!(check_fresh_out(fresh.to_str().unwrap()).is_ok());
        std::fs::create_dir(&fresh).unwrap();
        assert!(check_fresh_out(fresh.to_str().unwrap())
            .unwrap_err()
            .contains("exists"));
        assert!(
            check_fresh_out(scratch.join("missing-parent/fresh").to_str().unwrap())
                .unwrap_err()
                .contains("must already exist")
        );
        let _ = std::fs::remove_dir_all(&scratch);
    }
}

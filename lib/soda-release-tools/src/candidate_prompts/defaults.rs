//! Candidate prompt defaults: modes, paths, and controller args.
use crate::candidate::Options;

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

pub(crate) fn file_exists(path: &str) -> bool {
    std::fs::metadata(path).is_ok()
}

pub fn suggest_out() -> String {
    let cwd = std::env::current_dir()
        .map(|p| p.to_string_lossy().into_owned())
        .unwrap_or_default();
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap_or_default();
    let epoch_secs = i64::try_from(now.as_secs()).expect("current clock exceeds supported epoch");
    let leaf = soda_wire_time::compact_utc(epoch_secs)
        .expect("current clock timestamp outside supported RFC3339 range")
        .to_ascii_lowercase();
    format!("{cwd}/.artifacts/releases/isolated/{leaf}")
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

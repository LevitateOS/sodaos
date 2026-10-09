// Captures the build-time VCS stamp the soda-build controller admission
// requires (the Go owner reads it from its build info instead).
use std::process::Command;

fn git(args: &[&str]) -> Option<String> {
    let root = format!("{}/../..", env!("CARGO_MANIFEST_DIR"));
    let output = Command::new("git")
        .arg("--no-optional-locks")
        .arg("-C")
        .arg(&root)
        .args(args)
        .output()
        .ok()?;
    if !output.status.success() {
        return None;
    }
    Some(String::from_utf8_lossy(&output.stdout).trim().to_owned())
}

fn watch_git_input(path: &str) {
    if let Some(path) = git(&["rev-parse", "--path-format=absolute", "--git-path", path]) {
        if std::path::Path::new(&path).exists() {
            println!("cargo:rerun-if-changed={path}");
        }
    }
}

fn watch_tracked_files() {
    let Some(files) = git(&["ls-files", "--full-name", "-z"]) else {
        return;
    };
    let Some(root) = git(&["rev-parse", "--show-toplevel"]) else {
        return;
    };
    for file in files.split('\0').filter(|file| !file.is_empty()) {
        println!("cargo:rerun-if-changed={}/{}", root, file);
    }
}

fn main() {
    // Re-run when Git's revision metadata or any tracked cleanliness input changes.
    watch_git_input("HEAD");
    watch_git_input("index");
    watch_git_input("packed-refs");
    watch_git_input("refs");
    if let Some(reference) = git(&["symbolic-ref", "-q", "HEAD"]) {
        watch_git_input(&reference);
    }
    watch_tracked_files();
    let revision = git(&["rev-parse", "HEAD"]).unwrap_or_default();
    let dirty = git(&["status", "--porcelain", "--untracked-files=normal"])
        .map(|s| !s.is_empty())
        .unwrap_or(true);
    println!("cargo:rustc-env=SODA_BUILD_VCS_REVISION={revision}");
    println!(
        "cargo:rustc-env=SODA_BUILD_VCS_MODIFIED={}",
        i32::from(dirty)
    );
}

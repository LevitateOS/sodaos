//! Build-time VCS stamping, the equivalent of Go's `-buildvcs`
//! `vcs.revision`/`vcs.modified` settings the driver records on every
//! observation. Read-only git inspection; missing git bakes "unknown".

fn git_output(manifest: &str, args: &[&str]) -> Option<String> {
    let output = std::process::Command::new("git")
        .arg("--no-optional-locks")
        .arg("-C")
        .arg(manifest)
        .args(args)
        .output()
        .ok()?;
    if !output.status.success() {
        return None;
    }
    Some(String::from_utf8_lossy(&output.stdout).trim().to_string())
}

fn watch_git_input(manifest: &str, path: &str) {
    if let Some(path) = git_output(
        manifest,
        &["rev-parse", "--path-format=absolute", "--git-path", path],
    ) {
        if std::path::Path::new(&path).exists() {
            println!("cargo:rerun-if-changed={path}");
        }
    }
}

fn watch_tracked_files(manifest: &str) {
    let Some(root) = git_output(manifest, &["rev-parse", "--show-toplevel"]) else {
        return;
    };
    let output = std::process::Command::new("git")
        .arg("--no-optional-locks")
        .arg("-C")
        .arg(&root)
        .args(["ls-files", "--full-name", "-z"])
        .output();
    let Ok(output) = output else { return };
    if !output.status.success() {
        return;
    }
    let files = String::from_utf8_lossy(&output.stdout);
    for file in files.split('\0').filter(|file| !file.is_empty()) {
        println!("cargo:rerun-if-changed={}/{}", root, file);
    }
}

fn main() {
    let manifest = std::env::var("CARGO_MANIFEST_DIR").unwrap_or_else(|_| ".".to_string());
    // Cargo otherwise watches only build.rs. Track metadata that can change the
    // revision and every tracked path whose status contributes to `dirty`.
    watch_git_input(&manifest, "HEAD");
    watch_git_input(&manifest, "index");
    watch_git_input(&manifest, "packed-refs");
    watch_git_input(&manifest, "refs");
    if let Some(reference) = git_output(&manifest, &["symbolic-ref", "-q", "HEAD"]) {
        watch_git_input(&manifest, &reference);
    }
    watch_tracked_files(&manifest);
    let revision =
        git_output(&manifest, &["rev-parse", "HEAD"]).unwrap_or_else(|| "unknown".to_string());
    let dirty = git_output(&manifest, &["status", "--porcelain"])
        .map(|status| !status.is_empty())
        .unwrap_or(true);
    println!("cargo:rustc-env=BUILD_REVISION={revision}");
    println!(
        "cargo:rustc-env=BUILD_MODIFIED={}",
        if dirty { "true" } else { "false" }
    );
}

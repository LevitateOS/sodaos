// Captures the build-time VCS stamp the soda-build controller admission
// requires (the Go owner reads it from its build info instead).
use std::process::Command;

fn git(args: &[&str]) -> Option<String> {
    let root = format!("{}/../..", env!("CARGO_MANIFEST_DIR"));
    let output = Command::new("git").arg("-C").arg(&root).args(args).output().ok()?;
    if !output.status.success() {
        return None;
    }
    Some(String::from_utf8_lossy(&output.stdout).trim().to_owned())
}

fn main() {
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

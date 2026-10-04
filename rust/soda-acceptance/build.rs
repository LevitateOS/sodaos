//! Build-time VCS stamping, the equivalent of Go's `-buildvcs`
//! `vcs.revision`/`vcs.modified` settings the driver records on every
//! observation. Read-only git inspection; missing git bakes "unknown".

fn git_output(manifest: &str, args: &[&str]) -> Option<String> {
    let output = std::process::Command::new("git").arg("-C").arg(manifest).args(args).output().ok()?;
    if !output.status.success() {
        return None;
    }
    Some(String::from_utf8_lossy(&output.stdout).trim().to_string())
}

fn main() {
    let manifest = std::env::var("CARGO_MANIFEST_DIR").unwrap_or_else(|_| ".".to_string());
    let revision = git_output(&manifest, &["rev-parse", "HEAD"]).unwrap_or_else(|| "unknown".to_string());
    let dirty = git_output(&manifest, &["status", "--porcelain"])
        .map(|status| !status.is_empty())
        .unwrap_or(false);
    println!("cargo:rustc-env=BUILD_REVISION={revision}");
    println!("cargo:rustc-env=BUILD_MODIFIED={}", if dirty { "true" } else { "false" });
}

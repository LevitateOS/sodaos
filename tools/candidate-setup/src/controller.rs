use std::env;
use std::fs;
use std::path::{Path, PathBuf};

use super::process::{capture, run, Captured};
use super::storage::Storage;
use super::{fail, is_executable, stripped, stripped_string, Exit, ADMITTED, WRAPPER};

/// Controller outputs the remaining stages still need.
pub(super) struct ControllerPaths {
    pub(super) bindir: String,
}

/// Admitted controller recipe (D01-F2): the existing Rust release-tools
/// package, never the retired ./tools Go paths.
pub(super) fn controller_cargo_argv() -> [&'static str; 9] {
    [
        "build",
        "--release",
        "--locked",
        "-p",
        "soda-release-tools",
        "--bin",
        "soda-build",
        "--bin",
        "soda-candidate",
    ]
}

/// Build, install and verify the admitted controller + wrapper.
pub(super) fn admit_controller(
    storage: &Storage,
    cleanup: &mut Vec<PathBuf>,
) -> Result<ControllerPaths, Exit> {
    println!("-- build tools from committed source");
    let bindir_template = format!("{}/setup-bindir.XXXXXXXX", storage.scratch);
    let bindir = match capture("mktemp", &["-d", &bindir_template], false) {
        Captured::SpawnFailed(code) => return Err(Exit::Propagate(code)),
        Captured::Done(code, out) => {
            if code != 0 {
                return Err(Exit::Propagate(code));
            }
            stripped_string(&out)
        }
    };
    cleanup.push(PathBuf::from(&bindir));
    let soda_build = format!("{bindir}/soda-build");
    let soda_candidate = format!("{bindir}/soda-candidate");
    // D01-F2: the admitted controller is the existing Rust release-tools
    // build; the toolchain-pinned cargo build runs from this
    // verified-clean checkout.
    run("cargo", &controller_cargo_argv())?;
    // Fresh bindir, populated only by this build's outputs: these bytes
    // are the current Rust producer provenance. Both must land as real
    // executables or setup refuses to admit them.
    let target_dir = env::var("CARGO_TARGET_DIR").unwrap_or_else(|_| "target".to_string());
    for (bin, dest) in [
        ("soda-build", &soda_build),
        ("soda-candidate", &soda_candidate),
    ] {
        let built = format!("{target_dir}/release/{bin}");
        if fs::copy(&built, dest).is_err() || !is_executable(Path::new(dest)) {
            return fail(format!("admitted controller binary missing from {built}"));
        }
    }

    println!("-- install wrapper and admitted controller");
    run("sudo", &["install", "-m", "0755", &soda_candidate, WRAPPER])?;
    run(
        "sudo",
        &["install", "-D", "-m", "0755", &soda_build, ADMITTED],
    )?;
    let which_out = match capture("sudo", &["which", "soda-candidate"], false) {
        Captured::SpawnFailed(_) => Vec::new(),
        Captured::Done(_, out) => stripped(&out).to_vec(),
    };
    let which_str = String::from_utf8_lossy(&which_out);
    let got_wrapper = match capture("sudo", &["readlink", "-f", which_str.as_ref()], false) {
        Captured::SpawnFailed(code) => return Err(Exit::Propagate(code)),
        Captured::Done(code, out) => {
            if code != 0 {
                return Err(Exit::Propagate(code));
            }
            stripped(&out).to_vec()
        }
    };
    let want_wrapper = match capture("readlink", &["-f", WRAPPER], false) {
        Captured::SpawnFailed(code) => return Err(Exit::Propagate(code)),
        Captured::Done(code, out) => {
            if code != 0 {
                return Err(Exit::Propagate(code));
            }
            stripped(&out).to_vec()
        }
    };
    if got_wrapper != want_wrapper {
        return fail(format!(
            "wrapper not visible on sudo secure_path (got {})",
            String::from_utf8_lossy(&got_wrapper)
        ));
    }
    Ok(ControllerPaths { bindir })
}

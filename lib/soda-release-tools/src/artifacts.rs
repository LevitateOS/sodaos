//! `soda-artifacts` (Go `tools/soda-artifacts` `main.go`): artifact
//! subcommand dispatch. OCI inspection and CoreOS fetches admit inputs
//! locally, then delegate to the `soda-release-build` pipeline.

use std::os::unix::fs::{OpenOptionsExt, PermissionsExt};
use std::path::Path;
use std::process::{Command, Stdio};

use crate::digest::{hash_file, is_revision, is_signer, require_native};
use clap::{Arg, ArgAction, Command as ClapCommand};

pub const USAGE: &str =
    "usage: soda-artifacts inspect-oci|fetch-coreos|fetch-coreos-iso|convert-butane [flags]";
const HELP_REQUESTED: &str = "artifact help requested";

fn command() -> ClapCommand {
    ClapCommand::new("soda-artifacts")
        .no_binary_name(true)
        .args_override_self(true)
        .arg(Arg::new("action").required(true).value_parser([
            "inspect-oci",
            "fetch-coreos",
            "fetch-coreos-iso",
            "convert-butane",
        ]))
        .arg(
            Arg::new("arch")
                .long("arch")
                .help("matching native architecture")
                .action(ArgAction::Set)
                .num_args(1)
                .allow_hyphen_values(true),
        )
        .arg(
            Arg::new("revision")
                .long("revision")
                .help("full source revision")
                .action(ArgAction::Set)
                .num_args(1)
                .allow_hyphen_values(true),
        )
        .arg(
            Arg::new("source")
                .long("source")
                .help("OCI archive or private Butane file")
                .action(ArgAction::Set)
                .num_args(1)
                .allow_hyphen_values(true),
        )
        .arg(
            Arg::new("out")
                .long("out")
                .help("new absolute output directory/file")
                .action(ArgAction::Set)
                .num_args(1)
                .allow_hyphen_values(true),
        )
        .arg(
            Arg::new("keyring")
                .long("keyring")
                .help("already trusted Fedora keyring")
                .action(ArgAction::Set)
                .num_args(1)
                .allow_hyphen_values(true),
        )
        .arg(
            Arg::new("signer")
                .long("signer")
                .help("full independently trusted signer fingerprint")
                .action(ArgAction::Set)
                .num_args(1)
                .allow_hyphen_values(true),
        )
}

fn help_text() -> String {
    command().render_help().to_string()
}

#[derive(Debug, Clone, Default)]
pub struct ArtifactFlags {
    pub arch: String,
    pub revision: String,
    pub source: String,
    pub out: String,
    pub keyring: String,
    pub signer: String,
}

pub fn parse_artifact_flags(args: &[String]) -> Result<(String, ArtifactFlags), String> {
    if args.is_empty() {
        return Err(USAGE.to_owned());
    }
    let matches = command()
        .try_get_matches_from(args.iter().map(String::as_str))
        .map_err(|error| {
            if error.kind() == clap::error::ErrorKind::DisplayHelp {
                HELP_REQUESTED.to_owned()
            } else {
                "invalid artifact command flags".to_owned()
            }
        })?;
    let action = matches
        .get_one::<String>("action")
        .cloned()
        .unwrap_or_default();
    Ok((
        action,
        ArtifactFlags {
            arch: matches
                .get_one::<String>("arch")
                .cloned()
                .unwrap_or_default(),
            revision: matches
                .get_one::<String>("revision")
                .cloned()
                .unwrap_or_default(),
            source: matches
                .get_one::<String>("source")
                .cloned()
                .unwrap_or_default(),
            out: matches
                .get_one::<String>("out")
                .cloned()
                .unwrap_or_default(),
            keyring: matches
                .get_one::<String>("keyring")
                .cloned()
                .unwrap_or_default(),
            signer: matches
                .get_one::<String>("signer")
                .cloned()
                .unwrap_or_default(),
        },
    ))
}

/// `exec.LookPath` with Go's exact miss message.
pub fn look_path(name: &str) -> Result<String, String> {
    if name.contains('/') {
        let path = std::path::Path::new(name);
        if is_executable_file(path) {
            return Ok(name.to_owned());
        }
        return Err(format!(
            "exec: {name:?}: executable file not found in $PATH"
        ));
    }
    if let Ok(path_var) = std::env::var("PATH") {
        for dir in path_var.split(':') {
            let dir = if dir.is_empty() { "." } else { dir };
            let candidate = std::path::Path::new(dir).join(name);
            if is_executable_file(&candidate) {
                return Ok(candidate.to_string_lossy().into_owned());
            }
        }
    }
    Err(format!(
        "exec: {name:?}: executable file not found in $PATH"
    ))
}

fn is_executable_file(path: &std::path::Path) -> bool {
    match std::fs::metadata(path) {
        Ok(st) => st.file_type().is_file() && st.permissions().mode() & 0o111 != 0,
        Err(_) => false,
    }
}

/// `build.PrivateDestination`: absolute output under a real private parent,
/// not already present.
pub fn private_destination(path: &str) -> Result<(), String> {
    if !path.starts_with('/') {
        return Err("absolute private output required".to_owned());
    }
    let parent = match path.rfind('/') {
        Some(0) => "/",
        Some(i) => &path[..i],
        None => ".",
    };
    let resolved = std::fs::canonicalize(parent).map_err(|e| e.to_string())?;
    let st = std::fs::metadata(parent).map_err(|e| e.to_string())?;
    if resolved.to_string_lossy() != parent || st.permissions().mode() & 0o077 != 0 {
        return Err("real private output parent required".to_owned());
    }
    match std::fs::symlink_metadata(path) {
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(()),
        _ => Err("output already exists or cannot be inspected".to_owned()),
    }
}

/// `build.FreshDirectory`: absolute path whose real parent is symlink-free.
pub fn fresh_directory(path: &str) -> Result<(), String> {
    if !path.starts_with('/') {
        return Err("absolute new directory required".to_owned());
    }
    let parent = match path.rfind('/') {
        Some(0) => "/",
        Some(i) => &path[..i],
        None => ".",
    };
    let resolved = std::fs::canonicalize(parent).map_err(|e| e.to_string())?;
    if resolved.to_string_lossy() != parent {
        return Err("symlinked parent refused".to_owned());
    }
    std::fs::create_dir(path).map_err(|e| e.to_string())?;
    std::fs::set_permissions(path, std::fs::Permissions::from_mode(0o700))
        .map_err(|e| e.to_string())
}

fn open_private_butane(source: &str) -> Result<std::fs::File, String> {
    let st = std::fs::symlink_metadata(source).map_err(|e| e.to_string())?;
    if !st.file_type().is_file() || st.permissions().mode() & 0o077 != 0 {
        return Err("private regular Butane input required".to_owned());
    }
    std::fs::File::open(source).map_err(|e| e.to_string())
}

fn run_butane(input: std::fs::File, dest: &std::fs::File) -> Result<(), String> {
    run_butane_with(
        input,
        dest,
        Path::new("butane"),
        std::time::Duration::from_secs(60),
    )
}

fn run_butane_with(
    input: std::fs::File,
    dest: &std::fs::File,
    program: &Path,
    timeout: std::time::Duration,
) -> Result<(), String> {
    let mut child = Command::new(program)
        .arg("--strict")
        .stdin(Stdio::from(input))
        .stdout(Stdio::from(dest.try_clone().map_err(|_| {
            "Butane output descriptor clone failed".to_owned()
        })?))
        .spawn()
        .map_err(|_| "Butane could not start".to_owned())?;
    // One-minute phase timeout like the Go owner's context deadline.
    let start = std::time::Instant::now();
    loop {
        match child.try_wait() {
            Ok(Some(status)) => {
                return if status.success() {
                    Ok(())
                } else {
                    Err(format!("butane exited {}", status.code().unwrap_or(-1)))
                };
            }
            Ok(None) => {
                if start.elapsed() >= timeout {
                    let _ = child.kill();
                    return match child.wait() {
                        Ok(_) => Err("Butane conversion timed out".to_owned()),
                        Err(_) => Err(
                            "Butane conversion timed out; child cleanup could not be confirmed"
                                .to_owned(),
                        ),
                    };
                }
                std::thread::sleep(std::time::Duration::from_millis(25));
            }
            Err(_) => {
                let _ = child.kill();
                return match child.wait() {
                    Ok(_) => Err("Butane process wait failed".to_owned()),
                    Err(_) => Err(
                        "Butane process wait failed; child cleanup could not be confirmed"
                            .to_owned(),
                    ),
                };
            }
        }
    }
}

fn finish_butane_conversion(
    dest: std::fs::File,
    out: &str,
    run: impl FnOnce(&std::fs::File) -> Result<(), String>,
) -> Result<(), String> {
    let result = run(&dest);
    drop(dest);
    match result {
        Ok(()) => Ok(()),
        Err(primary) => match std::fs::remove_file(out) {
            Ok(()) => Err(format!("{primary}; partial output removed")),
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
                match std::fs::symlink_metadata(out) {
                    Err(check) if check.kind() == std::io::ErrorKind::NotFound => {
                        Err(format!("{primary}; partial output removed"))
                    }
                    _ => Err(format!(
                        "{primary}; partial output removal failed; inspect output locally"
                    )),
                }
            }
            Err(_) => Err(format!(
                "{primary}; partial output removal failed; inspect output locally"
            )),
        },
    }
}

fn create_butane_output(out: &str) -> Result<std::fs::File, String> {
    std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(out)
        .map_err(|e| e.to_string())
}

pub fn convert_butane(source: &str, out: &str, arch: &str) -> Result<(), String> {
    require_native(arch)?;
    private_destination(out)?;
    look_path("butane")?;
    let input = open_private_butane(source)?;
    let dest = create_butane_output(out)?;
    // The finalizer owns the output before any descriptor cloning can fail.
    finish_butane_conversion(dest, out, |dest| {
        let input = input
            .try_clone()
            .map_err(|_| "Butane input descriptor clone failed".to_owned())?;
        run_butane(input, dest)
    })
}

fn admit_coreos_fetch(arch: &str, signer: &str, keyring: &str) -> Result<(), String> {
    require_native(arch)?;
    if !is_signer(signer) {
        return Err("full trusted signer fingerprint required".to_owned());
    }
    hash_file(keyring)?;
    for name in ["gpgv", "xz"] {
        look_path(name)?;
    }
    Ok(())
}

fn admit_coreos_iso_fetch(arch: &str, signer: &str, keyring: &str) -> Result<String, String> {
    require_native(arch)?;
    if !is_signer(signer) {
        return Err("full trusted signer fingerprint required".to_owned());
    }
    hash_file(keyring)?;
    // `filepath.Abs` joins the cwd without resolving symlinks.
    let absolute = if keyring.starts_with('/') {
        keyring.to_owned()
    } else {
        std::env::current_dir()
            .map_err(|e| e.to_string())?
            .join(keyring)
            .to_string_lossy()
            .into_owned()
    };
    look_path("gpgv")?;
    Ok(absolute)
}

fn open_oci_archive(file: &str, arch: &str, revision: &str) -> Result<(), String> {
    crate::digest::oci_architecture(arch)?;
    if !revision.is_empty() && !is_revision(revision) {
        return Err("full source revision required".to_owned());
    }
    let st = std::fs::symlink_metadata(file).map_err(|e| e.to_string())?;
    if !st.file_type().is_file() {
        return Err("OCI archive must be regular".to_owned());
    }
    std::fs::File::open(file).map_err(|e| e.to_string())?;
    Ok(())
}

pub fn inspect_artifact_oci(source: &str, arch: &str, revision: &str) -> Result<String, String> {
    open_oci_archive(source, arch, revision)?;
    let image = soda_release_build::oci::inspect_oci(std::path::Path::new(source), arch, revision)
        .map_err(|e| e.to_string())?;
    Ok(image.marshal_compact())
}

fn run_coreos_artifact(action: &str, f: &ArtifactFlags) -> Result<(), String> {
    match action {
        "fetch-coreos" => {
            admit_coreos_fetch(&f.arch, &f.signer, &f.keyring)?;
            soda_release_build::coreos::fetch_coreos(&f.arch, &f.keyring, &f.signer, &f.out)
                .map(|_| ())
                .map_err(|e| e.to_string())
        }
        "fetch-coreos-iso" => {
            admit_coreos_iso_fetch(&f.arch, &f.signer, &f.keyring)?;
            soda_release_build::coreos_iso::fetch_coreos_iso(
                &f.arch,
                &f.keyring,
                &f.signer,
                &f.out,
            )
            .map(|_| ())
            .map_err(|e| e.to_string())
        }
        _ => Err("unknown artifact action; use fetch-coreos-iso for upstream ISO inputs; QCOW2 media delivery is not selected".to_owned()),
    }
}

pub fn run_artifact_action(action: &str, f: &ArtifactFlags) -> Result<Option<String>, String> {
    match action {
        "inspect-oci" => inspect_artifact_oci(&f.source, &f.arch, &f.revision).map(Some),
        "convert-butane" => convert_butane(&f.source, &f.out, &f.arch).map(|()| None),
        _ => run_coreos_artifact(action, f).map(|()| None),
    }
}

pub fn run(args: &[String]) -> Result<Option<String>, String> {
    let (action, flags) = match parse_artifact_flags(args) {
        Ok(parsed) => parsed,
        Err(e) if e == HELP_REQUESTED => {
            print!("{}", help_text());
            return Ok(None);
        }
        Err(e) => return Err(e),
    };
    run_artifact_action(&action, &flags)
}

pub fn main() {
    // Signal disposition matches the Go owner (default terminate); no
    // handler is installed.
    let argv: Vec<String> = std::env::args().collect();
    let args = if argv.len() > 1 { &argv[1..] } else { &[] };
    match run(args) {
        Ok(Some(document)) => println!("{document}"),
        Ok(None) => {}
        Err(e) => {
            eprintln!("{e}");
            std::process::exit(1);
        }
    }
}

#[cfg(test)]
mod tests;

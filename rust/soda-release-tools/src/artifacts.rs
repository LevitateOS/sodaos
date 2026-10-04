//! `soda-artifacts` (Go `tools/soda-artifacts` `main.go`): artifact
//! subcommand dispatch. Butane conversion is fully ported; OCI inspection
//! and CoreOS fetches admit inputs locally, then stop at explicit
//! release-pipeline boundary errors.

use std::os::unix::fs::{OpenOptionsExt, PermissionsExt};
use std::process::{Command, Stdio};

use crate::digest::{hash_file, is_revision, is_signer, require_native};
use crate::goflag::{self, FlagKind, FlagSpec};

pub const USAGE: &str =
    "usage: soda-artifacts inspect-oci|fetch-coreos|fetch-coreos-iso|convert-butane [flags]";

const ARTIFACT_SPECS: &[FlagSpec] = &[
    FlagSpec {
        name: "arch",
        kind: FlagKind::Text,
        usage: "matching native architecture",
        default_text: "",
    },
    FlagSpec {
        name: "revision",
        kind: FlagKind::Text,
        usage: "full source revision",
        default_text: "",
    },
    FlagSpec {
        name: "source",
        kind: FlagKind::Text,
        usage: "OCI archive or private Butane file",
        default_text: "",
    },
    FlagSpec {
        name: "out",
        kind: FlagKind::Text,
        usage: "new absolute output directory/file",
        default_text: "",
    },
    FlagSpec {
        name: "keyring",
        kind: FlagKind::Text,
        usage: "already trusted Fedora keyring",
        default_text: "",
    },
    FlagSpec {
        name: "signer",
        kind: FlagKind::Text,
        usage: "full independently trusted signer fingerprint",
        default_text: "",
    },
];

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
    let action = args[0].clone();
    let outcome = match goflag::parse(ARTIFACT_SPECS, &args[1..]) {
        Ok(o) => o,
        Err(_) => return Err("invalid artifact command flags".to_owned()),
    };
    if !outcome.positionals.is_empty() {
        return Err("invalid artifact command flags".to_owned());
    }
    Ok((
        action,
        ArtifactFlags {
            arch: outcome.text("arch"),
            revision: outcome.text("revision"),
            source: outcome.text("source"),
            out: outcome.text("out"),
            keyring: outcome.text("keyring"),
            signer: outcome.text("signer"),
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
    let mut child = Command::new("butane")
        .arg("--strict")
        .stdin(Stdio::from(input))
        .stdout(Stdio::from(dest.try_clone().map_err(|e| e.to_string())?))
        .spawn()
        .map_err(|e| e.to_string())?;
    // One-minute phase timeout like the Go owner's context deadline.
    let start = std::time::Instant::now();
    let timeout = std::time::Duration::from_secs(60);
    loop {
        match child.try_wait().map_err(|e| e.to_string())? {
            Some(status) => {
                return if status.success() {
                    Ok(())
                } else {
                    Err(format!("butane exited {}", status.code().unwrap_or(-1)))
                };
            }
            None => {
                if start.elapsed() >= timeout {
                    let _ = child.kill();
                    let _ = child.wait();
                    return Err("butane conversion timed out".to_owned());
                }
                std::thread::sleep(std::time::Duration::from_millis(25));
            }
        }
    }
}

fn finish_butane_conversion(
    dest: std::fs::File,
    out: &str,
    run: impl FnOnce() -> Result<(), String>,
) -> Result<(), String> {
    let result = run();
    drop(dest);
    match result {
        Ok(()) => Ok(()),
        Err(_) => {
            let _ = std::fs::remove_file(out);
            Err("strict Butane conversion failed; partial output removed".to_owned())
        }
    }
}

pub fn convert_butane(source: &str, out: &str, arch: &str) -> Result<(), String> {
    require_native(arch)?;
    private_destination(out)?;
    look_path("butane")?;
    let input = open_private_butane(source)?;
    let dest = std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(out)
        .map_err(|e| e.to_string())?;
    // Reopen handles for the child: the creator owns lifetime, the child
    // inherits clones.
    let dest_clone = dest.try_clone().map_err(|e| e.to_string())?;
    finish_butane_conversion(dest, out, || {
        run_butane(input.try_clone().map_err(|e| e.to_string())?, &dest_clone)
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
    Err(
        "OCI archive inspection requires the release pipeline port (not yet implemented)"
            .to_owned(),
    )
}

fn run_coreos_artifact(action: &str, f: &ArtifactFlags) -> Result<(), String> {
    match action {
        "fetch-coreos" => {
            admit_coreos_fetch(&f.arch, &f.signer, &f.keyring)?;
            Err("CoreOS fetch requires the release pipeline port (stream resolution and verified download are not yet implemented)".to_owned())
        }
        "fetch-coreos-iso" => {
            admit_coreos_iso_fetch(&f.arch, &f.signer, &f.keyring)?;
            Err("CoreOS ISO fetch requires the release pipeline port (stream resolution and verified download are not yet implemented)".to_owned())
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
    let (action, flags) = parse_artifact_flags(args)?;
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
mod tests {
    use super::*;
    use std::sync::atomic::{AtomicU64, Ordering};

    static COUNTER: AtomicU64 = AtomicU64::new(0);

    fn temp_dir(tag: &str) -> std::path::PathBuf {
        let id = COUNTER.fetch_add(1, Ordering::SeqCst);
        let path = std::env::temp_dir().join(format!(
            "soda-reltools-artifacts-{tag}-{}-{id}",
            std::process::id()
        ));
        let _ = std::fs::remove_dir_all(&path);
        std::fs::create_dir_all(&path).unwrap();
        path
    }

    fn parse_list(args: &[&str]) -> Result<(String, ArtifactFlags), String> {
        parse_artifact_flags(&args.iter().map(|s| s.to_string()).collect::<Vec<_>>())
    }

    #[test]
    fn flag_parse_matrix() {
        assert_eq!(parse_list(&[]).unwrap_err(), USAGE);
        let (action, f) =
            parse_list(&["inspect-oci", "--arch", "x86_64", "--revision", "r"]).unwrap();
        assert_eq!(action, "inspect-oci");
        assert_eq!(f.arch, "x86_64");
        assert_eq!(f.revision, "r");
        assert_eq!(
            parse_list(&["inspect-oci", "extra", "--arch", "x86_64"]).unwrap_err(),
            "invalid artifact command flags"
        );
        assert_eq!(
            parse_list(&["inspect-oci", "--bogus", "x"]).unwrap_err(),
            "invalid artifact command flags"
        );
        assert_eq!(
            parse_list(&["inspect-oci", "-h"]).unwrap_err(),
            "invalid artifact command flags"
        );
    }

    #[test]
    fn unknown_action_message() {
        let f = ArtifactFlags::default();
        assert_eq!(
            run_artifact_action("bogus", &f).unwrap_err(),
            "unknown artifact action; use fetch-coreos-iso for upstream ISO inputs; QCOW2 media delivery is not selected"
        );
    }

    #[test]
    fn admission_order_matches_go() {
        let f = ArtifactFlags::default();
        assert_eq!(
            run_artifact_action("inspect-oci", &f).unwrap_err(),
            "expected x86_64"
        );
        assert_eq!(
            run_artifact_action("fetch-coreos", &f).unwrap_err(),
            "expected x86_64"
        );
        let f = ArtifactFlags {
            arch: "x86_64".to_owned(),
            signer: "deadbeef".to_owned(),
            ..f
        };
        assert_eq!(
            run_artifact_action("fetch-coreos", &f).unwrap_err(),
            "full trusted signer fingerprint required"
        );
        let f = ArtifactFlags {
            arch: "x86_64".to_owned(),
            source: "/nope".to_owned(),
            revision: "abc".to_owned(),
            ..ArtifactFlags::default()
        };
        assert_eq!(
            run_artifact_action("inspect-oci", &f).unwrap_err(),
            "full source revision required"
        );
    }

    #[test]
    fn look_path_miss_message_matches_go() {
        assert_eq!(
            look_path("definitely-not-a-tool-xyz").unwrap_err(),
            "exec: \"definitely-not-a-tool-xyz\": executable file not found in $PATH"
        );
        assert!(look_path("sh").is_ok());
    }

    #[test]
    fn private_destination_matrix() {
        let scratch = temp_dir("privdest");
        assert_eq!(
            private_destination("relative/out").unwrap_err(),
            "absolute private output required"
        );
        let world = scratch.join("world");
        std::fs::create_dir(&world).unwrap();
        assert_eq!(
            private_destination(world.join("o").to_str().unwrap()).unwrap_err(),
            "real private output parent required"
        );
        let private = scratch.join("p");
        std::fs::create_dir(&private).unwrap();
        std::fs::set_permissions(&private, std::fs::Permissions::from_mode(0o700)).unwrap();
        let dest = private.join("o.json");
        assert!(private_destination(dest.to_str().unwrap()).is_ok());
        std::fs::write(&dest, b"x").unwrap();
        assert_eq!(
            private_destination(dest.to_str().unwrap()).unwrap_err(),
            "output already exists or cannot be inspected"
        );
        let _ = std::fs::remove_dir_all(&scratch);
    }

    #[test]
    fn fresh_directory_matrix() {
        let scratch = temp_dir("freshdir");
        assert!(fresh_directory("relative").is_err());
        let fresh = scratch.join("newdir");
        assert!(fresh_directory(fresh.to_str().unwrap()).is_ok());
        assert_eq!(
            std::fs::metadata(&fresh).unwrap().permissions().mode() & 0o777,
            0o700
        );
        assert!(fresh_directory(fresh.to_str().unwrap()).is_err());
        let _ = std::fs::remove_dir_all(&scratch);
    }

    #[test]
    fn butane_conversion_lifecycle() {
        use std::io::Write;
        let scratch = temp_dir("butane");
        // Failure removes the partial output.
        let out = scratch.join("fail.json");
        let dest = std::fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .open(&out)
            .unwrap();
        let out_s = out.to_string_lossy().into_owned();
        let mut dest2 = dest.try_clone().unwrap();
        let err = finish_butane_conversion(dest, &out_s, || {
            let _ = dest2.write_all(b"partial");
            Err("butane boom".to_owned())
        })
        .unwrap_err();
        assert_eq!(
            err,
            "strict Butane conversion failed; partial output removed"
        );
        assert!(std::fs::symlink_metadata(&out).is_err());
        // Success keeps the output.
        let out = scratch.join("ok.json");
        let dest = std::fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .open(&out)
            .unwrap();
        let out_s = out.to_string_lossy().into_owned();
        let mut dest2 = dest.try_clone().unwrap();
        finish_butane_conversion(dest, &out_s, || {
            dest2.write_all(b"{}").map_err(|e| e.to_string())
        })
        .unwrap();
        assert!(out.is_file());
        let _ = std::fs::remove_dir_all(&scratch);
    }
}

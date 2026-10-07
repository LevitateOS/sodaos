use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};

use super::config::{config_json, trust_json, worker_json};
use super::process::{capture, run, run_piped_stdin, run_stdout_null, Captured};
use super::storage::{refuse_active_build, Storage};
use super::{is_file, write_staged, Exit, ADMITTED, AUTHORITY, TOOLS};

/// `random_hex_passphrase` mirrors `head -c 32 /dev/urandom | od -An -tx1
/// | tr -d ' \n'`: 64 lowercase hex characters, no trailing newline.
pub(super) fn random_hex_passphrase() -> Result<String, Exit> {
    let mut bytes = [0u8; 32];
    getrandom::fill(&mut bytes).map_err(|_| Exit::Propagate(1))?;
    let mut hex = String::with_capacity(64);
    for byte in bytes {
        hex.push_str(&format!("{byte:02x}"));
    }
    Ok(hex)
}

fn unix_now() -> Result<u64, Exit> {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|elapsed| elapsed.as_secs())
        .map_err(|_| Exit::Fail("cannot read the current time".to_string()))
}

/// Staging reads and writes that the script did through `python3` surface
/// as setup errors here instead of tracebacks; the file bytes are identical.
fn read_staged(path: &str) -> Result<String, Exit> {
    match fs::read(path) {
        Ok(bytes) => String::from_utf8(bytes)
            .map_err(|_| Exit::Fail(format!("cannot read staged file {path}"))),
        Err(_) => Err(Exit::Fail(format!("cannot read staged file {path}"))),
    }
}

fn stage_file(path: &str, bytes: &[u8]) -> Result<(), Exit> {
    write_staged(Path::new(path), bytes)
        .map_err(|_| Exit::Fail(format!("cannot write staged file {path}")))
}

/// Worker git exception, restricted worker config and the
/// fixture-only media authority (never release keys).
pub(super) fn admit_fixture_authority(
    pwd: &str,
    forgejo_source: &str,
    output_parent: &str,
    storage: &Storage,
    worker_json_path: &str,
    prefix: &str,
    refresh: &str,
    owned: &str,
    cleanup: &mut Vec<PathBuf>,
) -> Result<(), Exit> {
    println!("-- worker git ownership exception");
    let gitconfig = "/etc/gitconfig";
    let needs_gitconfig = if !is_file(gitconfig) {
        true
    } else {
        match capture(
            "grep",
            &["-qF", "directory = /run/soda-build-source", gitconfig],
            false,
        ) {
            Captured::SpawnFailed(_) => false,
            Captured::Done(code, _) => code == 1,
        }
    };
    if needs_gitconfig {
        run_piped_stdin(
            "sudo",
            &["tee", "-a", gitconfig],
            b"# Soda build worker: the isolated worker sees the canonical checkout\n# only at /run/soda-build-source, owned by the operator. Mark it expected.\n[safe]\n\tdirectory = /run/soda-build-source\n",
        )?;
        run("sudo", &["chmod", "0644", gitconfig])?;
    }

    println!("-- restricted worker config");
    let worker_config = worker_json(
        ADMITTED,
        &pwd,
        &forgejo_source,
        &output_parent,
        &storage.root,
        &storage.home,
        &storage.run,
        TOOLS,
        AUTHORITY,
    );
    run_piped_stdin(
        "sudo",
        &["tee", &worker_json_path],
        worker_config.as_bytes(),
    )?;
    run("sudo", &["chmod", "0600", &worker_json_path])?;

    let trust_path = format!("{AUTHORITY}/trust.json");
    let artifact_path = format!("{AUTHORITY}/artifact.private");
    let passphrase_path = format!("{AUTHORITY}/passphrase");
    let config_path = format!("{AUTHORITY}/config.json");
    if is_file(&artifact_path) && refresh != "1" {
        println!("-- fixture authority already exists; keeping it (SODA_REFRESH_AUTHORITY=1 to regenerate)");
    } else {
        println!("-- fixture-only media authority (never release keys)");
        refuse_active_build()?;
        let owner = tempfile::Builder::new()
            .prefix("setup-authority.")
            .permissions(fs::Permissions::from_mode(0o700))
            .tempdir_in(&storage.scratch)
            .map_err(|e| Exit::Fail(format!("cannot stage fixture authority: {e}")))?;
        let tmpd = owner.path().to_string_lossy().into_owned();
        cleanup.push(owner.path().to_path_buf());
        let staged_passphrase = format!("{tmpd}/passphrase");
        stage_file(&staged_passphrase, random_hex_passphrase()?.as_bytes())?;
        for role in ["artifact", "candidate", "preview", "stable"] {
            let output_prefix = format!("{tmpd}/{role}");
            run_stdout_null(
                "skopeo",
                &[
                    "generate-sigstore-key",
                    "--output-prefix",
                    &output_prefix,
                    "--passphrase-file",
                    &staged_passphrase,
                ],
            )?;
        }
        let now = unix_now()?;
        let pubs = [
            read_staged(&format!("{tmpd}/artifact.pub"))?,
            read_staged(&format!("{tmpd}/candidate.pub"))?,
            read_staged(&format!("{tmpd}/preview.pub"))?,
            read_staged(&format!("{tmpd}/stable.pub"))?,
        ];
        stage_file(
            &format!("{tmpd}/trust.json"),
            trust_json(&prefix, now, [&pubs[0], &pubs[1], &pubs[2], &pubs[3]]).as_bytes(),
        )?;
        stage_file(&format!("{tmpd}/config.json"), config_json().as_bytes())?;
        for name in [
            "trust.json",
            "artifact.private",
            "passphrase",
            "config.json",
        ] {
            run(
                "sudo",
                &[
                    "install",
                    "-m",
                    "0600",
                    &format!("{tmpd}/{name}"),
                    &format!("{AUTHORITY}/{name}"),
                ],
            )?;
        }
        run("sudo", &["chmod", "0700", AUTHORITY])?;
        owner
            .close()
            .map_err(|e| Exit::Fail(format!("cannot remove fixture authority staging: {e}")))?;
    }
    run(
        "sudo",
        &[
            "chown",
            &owned,
            AUTHORITY,
            &trust_path,
            &artifact_path,
            &passphrase_path,
            &config_path,
        ],
    )?;
    Ok(())
}

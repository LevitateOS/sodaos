use std::fs;

use super::config::worker_json;
use super::process::{capture, run, run_piped_stdin, Captured};
use super::storage::Storage;
use super::{is_file, Exit, ADMITTED, TOOLS};

fn admit_git_exception() -> Result<(), Exit> {
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
    Ok(())
}

fn admit_worker_config(
    pwd: &str,
    forgejo_source: &str,
    fountain_release_url: &str,
    output_parent: &str,
    storage: &Storage,
    worker_json_path: &str,
) -> Result<(), Exit> {
    println!("-- restricted worker config");
    let admitted = fs::canonicalize(ADMITTED)
        .map_err(|err| Exit::Fail(format!("cannot resolve admitted controller: {err}")))?;
    let admitted = admitted
        .to_str()
        .ok_or_else(|| Exit::Fail("admitted controller path is not UTF-8".to_owned()))?;
    let worker_config = worker_json(
        admitted,
        pwd,
        forgejo_source,
        fountain_release_url,
        output_parent,
        &storage.root,
        &storage.home,
        &storage.run,
        TOOLS,
    );
    run_piped_stdin("sudo", &["tee", worker_json_path], worker_config.as_bytes())?;
    run("sudo", &["chmod", "0600", worker_json_path])?;
    Ok(())
}

/// Worker git exception and restricted worker config.
pub(super) fn admit_worker_setup(
    pwd: &str,
    forgejo_source: &str,
    fountain_release_url: &str,
    output_parent: &str,
    storage: &Storage,
    worker_json_path: &str,
) -> Result<(), Exit> {
    admit_git_exception()?;
    admit_worker_config(
        pwd,
        forgejo_source,
        fountain_release_url,
        output_parent,
        storage,
        worker_json_path,
    )
}

use super::process::{capture, ls_nonempty, run, run_stdout_null, Captured};
use super::storage::{migrate_candidate_home, refuse_active_build, Storage};
use super::{
    command_v, fail, fcontext_add_or_modify, selinux_has_type, stripped, Exit, AUTHORITY,
    LEGACY_RUN, PINNED_GO, TOOLS, WORKER_USER,
};

/// Worker-tool outputs the remaining stages still need.
pub(super) struct WorkerTools {
    pub(super) bun_final: String,
    pub(super) owned: String,
}

/// Provision worker directories, staged Go/bun and their verification.
pub(super) fn provision_worker_tools(
    output_parent: &str,
    storage: &Storage,
    pinned: &str,
    pinned_goroot: &str,
    want: &str,
) -> Result<WorkerTools, Exit> {
    println!("-- worker directories");
    let tools_bin = format!("{TOOLS}/bin");
    run(
        "sudo",
        &[
            "mkdir",
            "-p",
            &output_parent,
            &storage.home,
            &storage.run,
            &tools_bin,
            AUTHORITY,
        ],
    )?;
    migrate_candidate_home(&storage)?;
    if ls_nonempty(LEGACY_RUN) {
        println!(
            "-- legacy {LEGACY_RUN} holds leftovers; preserved untouched (new runs use {})",
            storage.run
        );
    }
    let pinned_new = format!("{PINNED_GO}.new");
    let bun_new = format!("{tools_bin}/bun.new");
    run("sudo", &["rm", "-rf", &pinned_new, &bun_new])?;
    run("sudo", &["cp", "-a", &pinned_goroot, &pinned_new])?;
    let staged_go = format!("{pinned_new}/bin/go");
    let staged_ok = match capture(&staged_go, &["version"], false) {
        Captured::SpawnFailed(_) => false,
        Captured::Done(_, out) => stripped(&out) == want.as_bytes(),
    };
    if !staged_ok {
        return fail(format!("staged Go is not {pinned}; refusing to publish it"));
    }
    let bun_path = command_v("bun").map(|p| p.to_string_lossy().into_owned());
    let bun_src = bun_path.as_deref().unwrap_or("");
    run("sudo", &["cp", bun_src, &bun_new])?;
    if run_stdout_null("sudo", &["-u", WORKER_USER, &bun_new, "--version"]).is_err() {
        return fail("staged bun is not worker-runnable; refusing to publish it");
    }
    refuse_active_build()?;
    let tools_go = format!("{TOOLS}/go");
    run("sudo", &["rm", "-rf", &tools_go, PINNED_GO])?;
    run("sudo", &["mv", &pinned_new, PINNED_GO])?;
    let bun_final = format!("{tools_bin}/bun");
    run("sudo", &["mv", &bun_new, &bun_final])?;
    run("sudo", &["chown", "-R", "root:root", PINNED_GO, TOOLS])?;
    if command_v("semanage").is_some() {
        let pattern = format!("{TOOLS}(/.*)?");
        fcontext_add_or_modify("bin_t", &pattern)?;
    }
    if command_v("restorecon").is_some() {
        run("sudo", &["restorecon", "-R", PINNED_GO, TOOLS])?;
    }
    run(
        "sudo",
        &[
            "find", PINNED_GO, "-type", "d", "-exec", "chmod", "0755", "{}", "+",
        ],
    )?;
    run(
        "sudo",
        &[
            "find", PINNED_GO, "-type", "f", "-exec", "chmod", "a+r", "{}", "+",
        ],
    )?;
    if !selinux_has_type(&format!("{PINNED_GO}/bin/go"), ":lib_t:") {
        return fail("pinned GOROOT is not lib_t; the sandboxed worker could not execute it");
    }
    let owned = format!("{WORKER_USER}:{WORKER_USER}");
    run(
        "sudo",
        &["chown", &owned, &output_parent, &storage.home, &storage.run],
    )?;
    if command_v("semanage").is_some() {
        let pattern = format!("{}(/.*)?", storage.root);
        fcontext_add_or_modify("var_lib_t", &pattern)?;
    }
    if command_v("restorecon").is_some() {
        run("sudo", &["restorecon", &storage.root])?;
    }
    if command_v("semanage").is_some() {
        let pattern = format!("{output_parent}(/.*)?");
        fcontext_add_or_modify("var_lib_t", &pattern)?;
    }
    if command_v("restorecon").is_some() {
        run("sudo", &["restorecon", "-R", &output_parent])?;
    }
    run("sudo", &["chmod", "0755", TOOLS, &tools_bin])?;
    run("sudo", &["chmod", "0700", AUTHORITY])?;

    println!("-- verify tools as the worker user");
    let verify_ok = match capture(
        "sudo",
        &[
            "-u",
            WORKER_USER,
            "env",
            "GOTOOLCHAIN=local",
            &format!("HOME={}", storage.home),
            &format!("{PINNED_GO}/bin/go"),
            "version",
        ],
        false,
    ) {
        Captured::SpawnFailed(_) => false,
        Captured::Done(_, out) => stripped(&out) == want.as_bytes(),
    };
    if !verify_ok {
        return fail(format!(
            "provisioned Go is not {pinned} or not worker-runnable"
        ));
    }
    if run(
        "sudo",
        &[
            "-u",
            WORKER_USER,
            "test",
            "-r",
            &format!("{PINNED_GO}/src/net/textproto/header.go"),
        ],
    )
    .is_err()
    {
        return fail("provisioned GOROOT sources are not worker-readable");
    }
    if run_stdout_null("sudo", &["-u", WORKER_USER, &bun_final, "--version"]).is_err() {
        return fail("provisioned bun is not worker-runnable");
    }
    Ok(WorkerTools { bun_final, owned })
}

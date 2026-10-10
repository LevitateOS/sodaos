use std::path::Path;

use super::process::{capture, ls_nonempty, run, run_stdout_null, Captured};
use super::selinux::{fcontext_add_or_modify, selinux_has_type};
use super::storage::{migrate_candidate_home, refuse_active_build, Storage};
use super::{
    command_v, fail, stripped, stripped_string, Exit, AUTHORITY, LEGACY_RUN, PINNED_GO, TOOLS,
    WORKER_USER,
};

/// Worker-tool outputs the remaining stages still need.
pub(super) struct WorkerTools {
    pub(super) rust_bin: String,
    pub(super) bun_final: String,
    pub(super) owned: String,
}

/// Provision worker directories, staged Go/Rust/Bun tools, and verification.
pub(super) fn provision_worker_tools(
    output_parent: &str,
    storage: &Storage,
    pinned: &str,
    pinned_goroot: &str,
    want: &str,
) -> Result<WorkerTools, Exit> {
    println!("-- worker directories");
    let tools_bin = format!("{TOOLS}/bin");
    let rust_final = format!("{TOOLS}/rust");
    let rust_new = format!("{TOOLS}/rust.new");
    let rust_bin = format!("{rust_final}/bin");
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

    let active_toolchain = match capture("rustup", &["show", "active-toolchain"], false) {
        Captured::SpawnFailed(code) => return Err(Exit::Propagate(code)),
        Captured::Done(code, out) if code == 0 => stripped_string(&out),
        Captured::Done(_, _) => return fail("repository-selected stable Rust toolchain required"),
    };
    if !active_toolchain
        .split_whitespace()
        .next()
        .is_some_and(|name| name == "stable" || name.starts_with("stable-"))
    {
        return fail("repository-selected stable Rust toolchain required");
    }
    let rust_sysroot = match capture("rustc", &["--print", "sysroot"], false) {
        Captured::SpawnFailed(code) => return Err(Exit::Propagate(code)),
        Captured::Done(code, out) if code == 0 => stripped_string(&out),
        Captured::Done(_, _) => return fail("cannot resolve selected Rust sysroot"),
    };
    let rust_sysroot_path = Path::new(&rust_sysroot);
    if !rust_sysroot_path.is_absolute() {
        return fail("selected Rust sysroot must be absolute");
    }
    for (tool, name) in [("rustc", "rustc"), ("cargo", "cargo")] {
        let selected = match capture("rustup", &["which", name], false) {
            Captured::SpawnFailed(code) => return Err(Exit::Propagate(code)),
            Captured::Done(code, out) if code == 0 => stripped_string(&out),
            Captured::Done(_, _) => return fail(format!("cannot resolve selected {tool}")),
        };
        let selected_path = std::fs::canonicalize(&selected)
            .map_err(|err| Exit::Fail(format!("cannot resolve selected {tool}: {err}")))?;
        let sysroot_path = std::fs::canonicalize(rust_sysroot_path)
            .map_err(|err| Exit::Fail(format!("cannot resolve Rust sysroot: {err}")))?;
        if !selected_path.starts_with(&sysroot_path) || !selected_path.is_file() {
            return fail(format!("selected {tool} is outside the Rust sysroot"));
        }
    }
    run("sudo", &["rm", "-rf", &rust_new])?;
    run("sudo", &["cp", "-a", &rust_sysroot, &rust_new])?;
    run("sudo", &["chown", "-R", "root:root", &rust_new])?;
    run(
        "sudo",
        &[
            "find", &rust_new, "-type", "d", "-exec", "chmod", "0755", "{}", "+",
        ],
    )?;
    run(
        "sudo",
        &[
            "find", &rust_new, "-type", "f", "-exec", "chmod", "a+r", "{}", "+",
        ],
    )?;
    if run_stdout_null(
        "sudo",
        &[
            "-u",
            WORKER_USER,
            &format!("{rust_new}/bin/rustc"),
            "--version",
        ],
    )
    .is_err()
        || run_stdout_null(
            "sudo",
            &[
                "-u",
                WORKER_USER,
                &format!("{rust_new}/bin/cargo"),
                "--version",
            ],
        )
        .is_err()
    {
        return fail("staged stable Rust toolchain is not worker-runnable");
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
    run("sudo", &["rm", "-rf", &tools_go, PINNED_GO, &rust_final])?;
    run("sudo", &["mv", &pinned_new, PINNED_GO])?;
    run("sudo", &["mv", &rust_new, &rust_final])?;
    let bun_final = format!("{tools_bin}/bun");
    run("sudo", &["mv", &bun_new, &bun_final])?;
    run(
        "sudo",
        &["chown", "-R", "root:root", PINNED_GO, &rust_final, TOOLS],
    )?;
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
    if !selinux_has_type(&format!("{rust_final}/bin/rustc"), ":bin_t:") {
        return fail(
            "staged Rust toolchain is not bin_t; the sandboxed worker could not execute it",
        );
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
    let rust_sysroot_ok = match capture(
        "sudo",
        &[
            "-u",
            WORKER_USER,
            &format!("{rust_final}/bin/rustc"),
            "--print",
            "sysroot",
        ],
        false,
    ) {
        Captured::SpawnFailed(_) => false,
        Captured::Done(code, out) => code == 0 && stripped(&out) == rust_final.as_bytes(),
    };
    if !rust_sysroot_ok {
        return fail("staged Rust compiler does not resolve its worker sysroot");
    }
    Ok(WorkerTools {
        rust_bin,
        bun_final,
        owned,
    })
}

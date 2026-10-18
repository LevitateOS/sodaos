use super::process::{capture, run, run_piped_stdin, run_stderr_null, Captured};
use super::storage::{refuse_active_build, Storage};
use super::{command_v, fail, Exit, WORKER_POLICY_SRC, WORKER_USER};

/// `fcontext_add_or_modify` mirrors `semanage fcontext -a ... || semanage
/// fcontext -m ...`: add the entry, or modify it when it already exists.
pub(super) fn fcontext_add_or_modify(file_type: &str, pattern: &str) -> Result<(), Exit> {
    if run_stderr_null(
        "sudo",
        &["semanage", "fcontext", "-a", "-t", file_type, pattern],
    )
    .is_err()
    {
        run(
            "sudo",
            &["semanage", "fcontext", "-m", "-t", file_type, pattern],
        )?;
    }
    Ok(())
}

/// `selinux_has_type` mirrors `sudo stat -c %C path | grep -q ":type:"`.
pub(super) fn selinux_has_type(path: &str, marker: &str) -> bool {
    match capture("sudo", &["stat", "-c", "%C", path], false) {
        Captured::SpawnFailed(_) => false,
        Captured::Done(code, out) => code == 0 && String::from_utf8_lossy(&out).contains(marker),
    }
}

/// Require the policycoreutils module tooling.
fn require_policy_tooling() -> Result<(), Exit> {
    for tool in ["checkmodule", "semodule_package", "semodule"] {
        if command_v(tool).is_none() {
            return fail(format!(
                "policycoreutils tooling required for the worker SELinux module (missing {tool})"
            ));
        }
    }
    Ok(())
}

/// Build and install the worker SELinux module.
fn install_worker_module(bindir: &str) -> Result<(), Exit> {
    let worker_mod = format!("{bindir}/soda-build-worker.mod");
    let worker_pp = format!("{bindir}/soda-build-worker.pp");
    run(
        "checkmodule",
        &["-M", "-m", "-o", &worker_mod, WORKER_POLICY_SRC],
    )?;
    run("semodule_package", &["-o", &worker_pp, "-m", &worker_mod])?;
    refuse_active_build()?;
    let _ = run_stderr_null("sudo", &["semodule", "-r", "soda-build-setpgid"]);
    run("sudo", &["semodule", "-i", &worker_pp])?;
    Ok(())
}

/// Create the cache directories and the containers.conf marker.
fn prepare_cache_dirs(
    storage: &Storage,
    go_mod: &str,
    go_build_cache: &str,
    owned: &str,
) -> Result<String, Exit> {
    let go_config = format!("{}/.config", storage.home);
    run(
        "sudo",
        &["mkdir", "-p", &go_build_cache, &go_mod, &go_config],
    )?;
    let containers_dir = format!("{go_config}/containers");
    run(
        "sudo",
        &[
            "install",
            "-d",
            "-o",
            WORKER_USER,
            "-g",
            WORKER_USER,
            &containers_dir,
        ],
    )?;
    let containers_conf = format!("{containers_dir}/containers.conf");
    run_piped_stdin(
        "sudo",
        &["tee", &containers_conf],
        b"[engine]\ncgroup_manager = \"cgroupfs\"\n",
    )?;
    run("sudo", &["chown", &owned, &containers_conf])?;
    run("sudo", &["chmod", "0644", &containers_conf])?;
    Ok(go_config)
}

/// Label caches/runtime when semanage exists; relabel when restorecon exists.
fn label_worker_paths(
    storage: &Storage,
    go_mod: &str,
    go_build_cache: &str,
    cargo_home: &str,
    cargo_target: &str,
    go_config: &str,
) -> Result<(), Exit> {
    if command_v("semanage").is_some() {
        for cache in [go_build_cache, go_mod, cargo_home, cargo_target, go_config] {
            let pattern = format!("{cache}(/.*)?");
            fcontext_add_or_modify("soda_build_cache_t", &pattern)?;
        }
        let pattern = format!("{}(/.*)?", storage.run);
        fcontext_add_or_modify("soda_build_runtime_t", &pattern)?;
    }
    if command_v("restorecon").is_some() {
        run(
            "sudo",
            &[
                "restorecon",
                "-R",
                &go_build_cache,
                &go_mod,
                cargo_home,
                cargo_target,
                &go_config,
                &storage.run,
            ],
        )?;
    }
    Ok(())
}

/// Verify the final SELinux types on cache, target, and runtime.
fn verify_worker_labels(
    go_build_cache: &str,
    cargo_target: &str,
    storage: &Storage,
) -> Result<(), Exit> {
    if !selinux_has_type(go_build_cache, ":soda_build_cache_t:") {
        return fail(
            "worker Go cache is not soda_build_cache_t; the sandboxed worker could not map it",
        );
    }
    if !selinux_has_type(cargo_target, ":soda_build_cache_t:") {
        return fail(
            "worker Cargo target is not soda_build_cache_t; the sandboxed worker could not map it",
        );
    }
    if !selinux_has_type(&storage.run, ":soda_build_runtime_t:") {
        return fail(
            "worker runtime is not soda_build_runtime_t; pasta could not use its netns dir",
        );
    }
    Ok(())
}

/// Install the worker SELinux module and label caches + runtime.
pub(super) fn install_worker_selinux(
    bindir: &str,
    storage: &Storage,
    go_mod: &str,
    go_build_cache: &str,
    cargo_home: &str,
    cargo_target: &str,
    owned: &str,
) -> Result<(), Exit> {
    println!("-- worker SELinux policy (process groups plus Go cache mapping)");
    require_policy_tooling()?;
    install_worker_module(bindir)?;
    let go_config = prepare_cache_dirs(storage, go_mod, go_build_cache, owned)?;
    label_worker_paths(
        storage,
        go_mod,
        go_build_cache,
        cargo_home,
        cargo_target,
        &go_config,
    )?;
    verify_worker_labels(go_build_cache, cargo_target, storage)
}

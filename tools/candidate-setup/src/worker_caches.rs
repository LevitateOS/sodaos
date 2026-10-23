use super::process::{capture, run, run_in_dir, Captured};
use super::storage::Storage;
use super::{fail, stripped_string, Exit, PINNED_GO, WORKER_USER};

/// Cache paths later stages still need.
pub(super) struct WorkerCaches {
    pub(super) go_mod: String,
    pub(super) go_build_cache: String,
    pub(super) cargo_home: String,
    pub(super) cargo_target: String,
}

/// Warm the Go module cache and the full module set.
fn warm_go_caches(storage: &Storage, go_mod: &str, go_build_cache: &str) -> Result<(), Exit> {
    let pinned_go_bin = format!("{PINNED_GO}/bin/go");
    let home_env = format!("HOME={}", storage.home);
    let modcache_env = format!("GOMODCACHE={go_mod}");
    let gocache_env = format!("GOCACHE={go_build_cache}");
    let warm_env = [
        home_env.as_str(),
        "GOPROXY=https://proxy.golang.org,direct",
        "GOSUMDB=sum.golang.org",
        modcache_env.as_str(),
        gocache_env.as_str(),
        "GOTOOLCHAIN=local",
        "GOFLAGS=-mod=readonly",
        "CGO_ENABLED=0",
        "GIT_OPTIONAL_LOCKS=0",
    ];
    // `sudo env ... go build ./...`: argv assembled exactly like the script.
    let mut build_args: Vec<&str> = vec!["env"];
    build_args.extend(warm_env.iter().copied());
    build_args.push(&pinned_go_bin);
    build_args.push("build");
    build_args.push("./...");
    if run("sudo", &build_args).is_err() {
        return fail("cannot warm Go module cache");
    }
    let mut download_args: Vec<&str> = vec!["env"];
    download_args.extend(warm_env.iter().copied());
    download_args.push(&pinned_go_bin);
    download_args.push("mod");
    download_args.push("download");
    if run("sudo", &download_args).is_err() {
        return fail("cannot warm full Go module set");
    }
    Ok(())
}

/// Warm the locked Cargo cache.
fn warm_cargo_cache(
    storage: &Storage,
    owned: &str,
    rust_bin: &str,
    cargo_home: &str,
    cargo_target: &str,
) -> Result<(), Exit> {
    // Cargo resolves the full manifest path, so an inherited cwd does not
    // make an operator's private home traversable by the worker. Fetch from
    // committed source in worker scratch without broadening home access.
    let source = tempfile::Builder::new()
        .prefix("setup-cargo.")
        .tempdir_in(&storage.scratch)
        .map_err(|_| Exit::Propagate(1))?
        .keep();
    let source = source.to_string_lossy().into_owned();
    let result = warm_cargo_source(storage, owned, rust_bin, cargo_home, cargo_target, &source);
    let cleanup = run("sudo", &["rm", "-rf", "--", &source]);
    match (result, cleanup) {
        (Ok(()), Ok(())) => Ok(()),
        (Err(primary), _) => Err(primary),
        (Ok(()), Err(cleanup)) => Err(cleanup),
    }
}

fn warm_cargo_source(
    storage: &Storage,
    owned: &str,
    rust_bin: &str,
    cargo_home: &str,
    cargo_target: &str,
    source: &str,
) -> Result<(), Exit> {
    let archive = format!("{source}/source.tar");
    run(
        "git",
        &["archive", "--format=tar", "--output", &archive, "HEAD"],
    )?;
    run(
        "tar",
        &["--extract", "--file", &archive, "--directory", source],
    )?;
    run("sudo", &["chown", "-R", owned, source])?;
    let rust_path = format!("{rust_bin}:/usr/sbin:/usr/bin:/sbin:/bin");
    let cargo_user_home = format!("HOME={}", storage.home);
    let cargo_home_env = format!("CARGO_HOME={cargo_home}");
    let cargo_target_env = format!("CARGO_TARGET_DIR={cargo_target}");
    let cargo_path_env = format!("PATH={rust_path}");
    let cargo_bin = format!("{rust_bin}/cargo");
    let cargo_warm: Vec<&str> = vec![
        "-u",
        WORKER_USER,
        "env",
        "--chdir",
        source,
        &cargo_user_home,
        &cargo_home_env,
        &cargo_target_env,
        "CARGO_NET_OFFLINE=false",
        &cargo_path_env,
        &cargo_bin,
        "fetch",
        "--locked",
        "--manifest-path",
        "Cargo.toml",
    ];
    // Enter worker-owned private scratch only after sudo changes identity.
    // A pre-exec current_dir would still run as the operator and fail under
    // a private caller umask, before sudo can grant the worker access.
    if run("sudo", &cargo_warm).is_err() {
        return fail("cannot warm locked Rust dependency cache");
    }
    Ok(())
}

/// Pin the worker Go offline after the caches are warm.
fn lock_go_offline(storage: &Storage) -> Result<(), Exit> {
    let pinned_go_bin = format!("{PINNED_GO}/bin/go");
    if run(
        "sudo",
        &[
            "-u",
            WORKER_USER,
            "env",
            &format!("HOME={}", storage.home),
            &pinned_go_bin,
            "env",
            "-w",
            "GOPROXY=off",
        ],
    )
    .is_err()
    {
        return fail("cannot lock worker Go offline");
    }
    Ok(())
}

/// Warm the Bun cache and stage Playwright chromium in the scratch tree.
fn warm_bun_cache(storage: &Storage, owned: &str, bun_final: &str, tmpw: &str) -> Result<(), Exit> {
    let tmpw_tools = format!("{tmpw}/tools");
    run("mkdir", &["-p", &tmpw_tools])?;
    run("cp", &["package.json", "bun.lock", "bunfig.toml", tmpw])?;
    run("cp", &["-a", "tools/lit-check", &tmpw_tools])?;
    run("sudo", &["chown", "-R", owned, tmpw])?;
    run(
        "sudo",
        &[
            "find", tmpw, "-type", "d", "-exec", "chmod", "0755", "{}", "+",
        ],
    )?;
    if run_in_dir(
        "sudo",
        &[
            "-u",
            WORKER_USER,
            "env",
            &format!("HOME={}", storage.home),
            bun_final,
            "install",
            "--frozen-lockfile",
        ],
        tmpw,
        true,
        "cannot warm Bun cache",
    )
    .is_err()
    {
        return fail("cannot warm Bun cache");
    }
    let browsers = format!("{}/browsers", storage.home);
    run(
        "sudo",
        &[
            "install",
            "-d",
            "-o",
            WORKER_USER,
            "-g",
            WORKER_USER,
            &browsers,
        ],
    )?;
    if run_in_dir(
        "sudo",
        &[
            "-u",
            WORKER_USER,
            "env",
            &format!("HOME={}", storage.home),
            &format!("PLAYWRIGHT_BROWSERS_PATH={browsers}"),
            bun_final,
            "x",
            "playwright",
            "install",
            "chromium",
        ],
        tmpw,
        false,
        "cannot stage Playwright chromium",
    )
    .is_err()
    {
        return fail("cannot stage Playwright chromium");
    }
    Ok(())
}

/// Warm compiler dependencies before the isolated worker runs offline.
pub(super) fn warm_worker_caches(
    storage: &Storage,
    owned: &str,
    bun_final: &str,
    rust_bin: &str,
) -> Result<WorkerCaches, Exit> {
    println!("-- warm worker caches before offline worker execution");
    let go_mod = format!("{}/go-mod", storage.home);
    let go_build_cache = format!("{}/go-build", storage.home);
    let cargo_home = format!("{}/cargo", storage.home);
    let cargo_target = format!("{}/cargo-target", storage.home);
    run(
        "sudo",
        &[
            "mkdir",
            "-p",
            &go_mod,
            &go_build_cache,
            &cargo_home,
            &cargo_target,
        ],
    )?;
    run("sudo", &["chown", "-R", owned, &cargo_home, &cargo_target])?;
    warm_go_caches(storage, &go_mod, &go_build_cache)?;
    warm_cargo_cache(storage, owned, rust_bin, &cargo_home, &cargo_target)?;
    lock_go_offline(storage)?;
    let bun_template = format!("{}/setup-bun.XXXXXXXX", storage.scratch);
    let tmpw = match capture("mktemp", &["-d", &bun_template], false) {
        Captured::SpawnFailed(code) => return Err(Exit::Propagate(code)),
        Captured::Done(code, out) => {
            if code != 0 {
                return Err(Exit::Propagate(code));
            }
            stripped_string(&out)
        }
    };
    // The worker owns this tree after chown, so a host TempDir cannot perform
    // reliable Drop cleanup. Keep the exact scratch path and always run the
    // owner-privileged cleanup after staging, including failed warm commands.
    let warm_result = warm_bun_cache(storage, owned, bun_final, &tmpw);
    let cleanup_result = run("sudo", &["rm", "-rf", &tmpw]);
    match (warm_result, cleanup_result) {
        (Ok(()), Ok(())) => {}
        (Err(primary), Ok(())) => return Err(primary),
        (Err(primary), Err(cleanup)) => {
            eprintln!("cannot remove worker Bun scratch {tmpw}: {cleanup:?}");
            return Err(primary);
        }
        (Ok(()), Err(cleanup)) => return Err(cleanup),
    }
    run("sudo", &["chown", "-R", &owned, &storage.home])?;
    Ok(WorkerCaches {
        go_mod,
        go_build_cache,
        cargo_home,
        cargo_target,
    })
}

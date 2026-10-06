use super::process::{capture, run, run_in_dir, Captured};
use super::storage::Storage;
use super::{fail, stripped_string, Exit, PINNED_GO, WORKER_USER};

/// Cache paths later stages still need.
pub(super) struct WorkerCaches {
    pub(super) go_mod: String,
    pub(super) go_build_cache: String,
}

/// Warm the worker Go/Bun caches (the isolated worker has no network).
pub(super) fn warm_worker_caches(
    storage: &Storage,
    pinned: &str,
    owned: &str,
    bun_final: &str,
) -> Result<WorkerCaches, Exit> {
    println!("-- warm worker caches (the isolated worker has no network)");
    let go_mod = format!("{}/go-mod", storage.home);
    let go_build_cache = format!("{}/go-build", storage.home);
    run("sudo", &["mkdir", "-p", &go_mod, &go_build_cache])?;
    let toolchain = format!("go{pinned}");
    let pinned_go_bin = format!("{PINNED_GO}/bin/go");
    let home_env = format!("HOME={}", storage.home);
    let modcache_env = format!("GOMODCACHE={go_mod}");
    let gocache_env = format!("GOCACHE={go_build_cache}");
    let toolchain_env = format!("GOTOOLCHAIN={toolchain}");
    let warm_env = [
        home_env.as_str(),
        "GOPROXY=https://proxy.golang.org,direct",
        "GOSUMDB=sum.golang.org",
        modcache_env.as_str(),
        gocache_env.as_str(),
        toolchain_env.as_str(),
        "GOFLAGS=-mod=readonly",
        "CGO_ENABLED=0",
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
    download_args.push("all");
    if run("sudo", &download_args).is_err() {
        return fail("cannot warm full Go module set");
    }
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
    // Like the script's `trap` at this point, the bun scratch tree is only
    // removed by the explicit `sudo rm -rf` after a successful warm: a
    // failure here leaks it, worker-owned, under the scratch root.
    let tmpw_tools = format!("{tmpw}/tools");
    run("mkdir", &["-p", &tmpw_tools])?;
    run("cp", &["package.json", "bun.lock", "bunfig.toml", &tmpw])?;
    run("cp", &["-a", "tools/lit-check", &tmpw_tools])?;
    run("sudo", &["chown", "-R", &owned, &tmpw])?;
    run(
        "sudo",
        &[
            "find", &tmpw, "-type", "d", "-exec", "chmod", "0755", "{}", "+",
        ],
    )?;
    if run_in_dir(
        "sudo",
        &[
            "-u",
            WORKER_USER,
            "env",
            &format!("HOME={}", storage.home),
            &bun_final,
            "install",
            "--frozen-lockfile",
        ],
        &tmpw,
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
            &bun_final,
            "x",
            "playwright",
            "install",
            "chromium",
        ],
        &tmpw,
        false,
        "cannot stage Playwright chromium",
    )
    .is_err()
    {
        return fail("cannot stage Playwright chromium");
    }
    run("sudo", &["rm", "-rf", &tmpw])?;
    run("sudo", &["chown", "-R", &owned, &storage.home])?;
    Ok(WorkerCaches {
        go_mod,
        go_build_cache,
    })
}

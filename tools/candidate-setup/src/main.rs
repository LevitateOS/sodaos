// soda-candidate-setup prepares this machine so `sudo soda-candidate`
// runs without a coding agent: wrapper on sudo's PATH, admitted controller,
// worker directories, restricted worker config, and a fixture-only media
// authority. Development and fixture scope only: it never creates
// qualification or signing configs, and the generated keys must never
// stand in for release keys.
//
// Rust port of scripts/setup-soda-candidate.sh (plus the
// scripts/candidate-storage.sh defaults it sourced). Messages, exit codes,
// installed paths, file modes, and generated file bytes match the shell.
// Like the script, this binary takes no arguments and ignores any it is
// given. Run from the repository root.

use std::env;
use std::ffi::CString;
use std::fs;
use std::io::{self, Read, Write};
use std::os::unix::ffi::OsStrExt;
use std::os::unix::fs::OpenOptionsExt;
use std::path::{Path, PathBuf};

mod preflight;
mod process;
mod storage;

#[cfg(test)]
use self::preflight::bridge_ip;
use self::process::{
    capture, id_un, ls_nonempty, run, run_in_dir, run_piped_stdin, run_stderr_null,
    run_stdout_null, Captured,
};
#[cfg(test)]
use self::process::{git_tree_clean, pipe2};
#[cfg(test)]
use self::storage::units_have_active;
use self::storage::{migrate_candidate_home, refuse_active_build};

const FAIL_PREFIX: &str = "setup-soda-candidate";
const PREFIX_DEFAULT: &str = "ghcr.io/levitateos/sodaos";
const STORAGE_ROOT_DEFAULT: &str = "/home/soda-candidate";
const ROOTFS_DIR: &str = "/home/soda-rootfs";
const ADMITTED: &str = "/usr/local/lib/soda/soda-build";
const PINNED_GO: &str = "/usr/local/lib/soda/pinned-go";
const WRAPPER: &str = "/usr/sbin/soda-candidate";
const TOOLS: &str = "/var/lib/soda-candidate-tools";
const WORKER_POLICY_SRC: &str = "system/host/selinux/soda-build-worker.te";
const AUTHORITY: &str = "/var/lib/soda-candidate-authority";
const LEGACY_HOME: &str = "/var/lib/soda-candidate-home";
const LEGACY_RUN: &str = "/var/lib/soda-candidate-run";
const WORKER_USER: &str = "soda-build-worker";

extern "C" {
    fn flock(fd: i32, op: i32) -> i32;
    fn faccessat(dirfd: i32, path: *const i8, mode: i32, flags: i32) -> i32;
    fn umask(mask: u32) -> u32;
    fn sigaction(signum: i32, act: *const Sigaction, oldact: *mut Sigaction) -> i32;
}

/// Matches glibc's `struct sigaction` on Linux (handler, signal mask, flags,
/// restorer). The glibc wrapper fills in the restorer itself.
#[repr(C)]
#[derive(Clone, Copy)]
struct Sigaction {
    handler: usize,
    mask: [u64; 16],
    flags: i32,
    restorer: usize,
}

const SIGPIPE: i32 = 13;
const SIG_DFL: usize = 0;
const LOCK_EX: i32 = 2;
const LOCK_NB: i32 = 4;
const AT_FDCWD: i32 = -100;
const X_OK: i32 = 1;

/// How the process ends: a `fail()` message with exit 1, or a propagated
/// child status with no extra output (the script's `set -e` behavior).
#[derive(Debug)]
enum Exit {
    Fail(String),
    Propagate(i32),
}

fn fail<T>(msg: impl Into<String>) -> Result<T, Exit> {
    Err(Exit::Fail(msg.into()))
}

/// `env_or` mirrors `${VAR:-default}`: unset or empty falls back.
fn env_or(key: &str, default: &str) -> String {
    match env::var(key) {
        Ok(v) if !v.is_empty() => v,
        _ => default.to_string(),
    }
}

/// `current_pwd` mirrors `$PWD`: the inherited logical path when present,
// otherwise the physical working directory.
fn current_pwd() -> String {
    match env::var("PWD") {
        Ok(v) => v,
        Err(_) => env::current_dir()
            .map(|p| p.to_string_lossy().into_owned())
            .unwrap_or_default(),
    }
}

fn is_dir(path: &str) -> bool {
    fs::metadata(path).map(|m| m.is_dir()).unwrap_or(false)
}

fn is_file(path: &str) -> bool {
    fs::metadata(path).map(|m| m.is_file()).unwrap_or(false)
}

/// `stripped` mirrors command substitution: all trailing newlines removed.
fn stripped(out: &[u8]) -> &[u8] {
    let mut end = out.len();
    while end > 0 && out[end - 1] == b'\n' {
        end -= 1;
    }
    &out[..end]
}

fn stripped_string(out: &[u8]) -> String {
    String::from_utf8_lossy(stripped(out)).into_owned()
}

fn current_umask() -> u32 {
    unsafe {
        let mask = umask(0);
        umask(mask);
        mask
    }
}

/// `write_staged` mirrors shell redirection into staging: bytes exact, mode
/// 0666 filtered by the operator umask like `>` and `open("w")`.
fn write_staged(path: &Path, bytes: &[u8]) -> io::Result<()> {
    let mode = 0o666 & !current_umask();
    let mut opts = fs::OpenOptions::new();
    opts.write(true).create(true).truncate(true).mode(mode);
    opts.open(path)?.write_all(bytes)
}

fn is_executable(path: &Path) -> bool {
    match CString::new(path.as_os_str().as_bytes()) {
        Ok(c) => unsafe { faccessat(AT_FDCWD, c.as_ptr(), X_OK, 0) == 0 },
        Err(_) => false,
    }
}

/// `command_v` mirrors `command -v`: a PATH search with execute permission.
fn command_v(tool: &str) -> Option<PathBuf> {
    let raw = env::var_os("PATH").unwrap_or_else(|| "/bin:/usr/bin".into());
    command_v_in(tool, &raw)
}

fn command_v_in(tool: &str, path_env: &std::ffi::OsStr) -> Option<PathBuf> {
    if tool.contains('/') {
        let path = PathBuf::from(tool);
        return is_executable(&path).then_some(path);
    }
    for dir in env::split_paths(path_env) {
        let candidate = dir.join(tool);
        if is_executable(&candidate) {
            return Some(candidate);
        }
    }
    None
}

/// Admitted controller recipe (D01-F2): the existing Rust release-tools
/// package, never the retired ./tools Go paths.
fn controller_cargo_argv() -> [&'static str; 9] {
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

/// `json_escape` mirrors Python `json.dumps` with `ensure_ascii`: the
/// script generated every JSON file through it, so the bytes stay identical.
fn json_escape(value: &str) -> String {
    let mut escaped = String::with_capacity(value.len() + 2);
    for ch in value.chars() {
        match ch {
            '"' => escaped.push_str("\\\""),
            '\\' => escaped.push_str("\\\\"),
            '\n' => escaped.push_str("\\n"),
            '\r' => escaped.push_str("\\r"),
            '\t' => escaped.push_str("\\t"),
            '\u{08}' => escaped.push_str("\\b"),
            '\u{0c}' => escaped.push_str("\\f"),
            _ if (ch < '\u{20}' || ch == '\u{7f}') => {
                escaped.push_str(&format!("\\u{:04x}", ch as u32));
            }
            _ if ch > '\u{7e}' => {
                let mut encoded = [0u16; 2];
                for unit in ch.encode_utf16(&mut encoded) {
                    escaped.push_str(&format!("\\u{unit:04x}"));
                }
            }
            _ => escaped.push(ch),
        }
    }
    escaped
}

fn json_field(key: &str, value: &str, indent: usize) -> String {
    format!(
        "{:indent$}\"{}\": \"{}\"",
        "",
        key,
        json_escape(value),
        indent = indent
    )
}

fn json_number_field(key: &str, value: u64, indent: usize) -> String {
    format!("{:indent$}\"{}\": {}", "", key, value, indent = indent)
}

/// `worker_json` mirrors the `json.dump(..., indent=2)` for worker.json,
/// including the absent trailing newline.
#[allow(clippy::too_many_arguments)]
fn worker_json(
    executable: &str,
    source: &str,
    forgejo_source: &str,
    output_parent: &str,
    storage_root: &str,
    build_home: &str,
    runtime: &str,
    tools: &str,
    authority: &str,
) -> String {
    [
        "{".to_string(),
        json_field("Executable", executable, 2) + ",",
        json_field("Source", source, 2) + ",",
        json_field("ForgejoSource", forgejo_source, 2) + ",",
        json_field("OutputParent", output_parent, 2) + ",",
        json_field("StorageRoot", storage_root, 2) + ",",
        json_field("BuildHome", build_home, 2) + ",",
        json_field("Runtime", runtime, 2) + ",",
        json_field("Tools", tools, 2) + ",",
        json_field("MediaAuthorityDirectory", authority, 2),
        "}".to_string(),
    ]
    .join("\n")
}

/// `trust_json` mirrors the `json.dumps(..., indent=2) + "\n"` trust file.
fn trust_json(prefix: &str, now: u64, pubs: [&str; 4]) -> String {
    let roles = ["artifact", "candidate", "preview", "stable"];
    let mut lines = vec![
        "{".to_string(),
        json_number_field("Format", 1, 2) + ",",
        json_field("Prefix", prefix, 2) + ",",
        json_number_field("Epoch", 1, 2) + ",",
        "  \"Keys\": {".to_string(),
    ];
    for (index, (role, key)) in roles.iter().zip(pubs.iter()).enumerate() {
        let comma = if index + 1 < roles.len() { "," } else { "" };
        lines.push(format!("    \"{role}\": ["));
        lines.push(format!("      \"{}\"", json_escape(key)));
        lines.push(format!("    ]{comma}"));
    }
    lines.push("  },".to_string());
    lines.push(json_number_field("NotBefore", now - 600, 2) + ",");
    lines.push(json_number_field("MaxAgeSeconds", 3600, 2) + ",");
    lines.push(json_number_field("ClockSkewSeconds", 10, 2) + ",");
    lines.push("  \"MinimumSequence\": {".to_string());
    lines.push(json_number_field("candidate", 1, 4) + ",");
    lines.push(json_number_field("preview", 1, 4) + ",");
    lines.push(json_number_field("stable", 1, 4));
    lines.push("  }".to_string());
    lines.push("}".to_string());
    lines.join("\n") + "\n"
}

/// `config_json` mirrors the `json.dumps(..., indent=2) + "\n"` config file.
fn config_json() -> String {
    [
        "{".to_string(),
        json_field("Trust", "/run/soda-media-authority/trust.json", 2) + ",",
        "  \"Keys\": {".to_string(),
        json_field("Key", "/run/soda-media-authority/artifact.private", 4) + ",",
        json_field("Passphrase", "/run/soda-media-authority/passphrase", 4),
        "  }".to_string(),
        "}".to_string(),
    ]
    .join("\n")
        + "\n"
}

/// `fcontext_add_or_modify` mirrors `semanage fcontext -a ... || semanage
/// fcontext -m ...`: add the entry, or modify it when it already exists.
fn fcontext_add_or_modify(file_type: &str, pattern: &str) -> Result<(), Exit> {
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
fn selinux_has_type(path: &str, marker: &str) -> bool {
    match capture("sudo", &["stat", "-c", "%C", path], false) {
        Captured::SpawnFailed(_) => false,
        Captured::Done(code, out) => code == 0 && String::from_utf8_lossy(&out).contains(marker),
    }
}

fn run_setup(cleanup: &mut Vec<PathBuf>) -> Result<(), Exit> {
    let pre = preflight::preflight()?;
    let preflight::Preflight {
        prefix,
        refresh,
        forgejo_source,
        storage,
        rootfs_url,
        pwd,
        output_parent,
        worker_json_path,
        pinned,
        pinned_goroot,
        want,
    } = pre;

    storage::prepare_scratch(&storage)?;

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

    println!("-- worker SELinux policy (process groups plus Go cache mapping)");
    for tool in ["checkmodule", "semodule_package", "semodule"] {
        if command_v(tool).is_none() {
            return fail(format!(
                "policycoreutils tooling required for the worker SELinux module (missing {tool})"
            ));
        }
    }
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
    if command_v("semanage").is_some() {
        for cache in [&go_build_cache, &go_mod, &go_config] {
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
                &go_config,
                &storage.run,
            ],
        )?;
    }
    if !selinux_has_type(&go_build_cache, ":soda_build_cache_t:") {
        return fail(
            "worker Go cache is not soda_build_cache_t; the sandboxed worker could not map it",
        );
    }
    if !selinux_has_type(&storage.run, ":soda_build_runtime_t:") {
        return fail(
            "worker runtime is not soda_build_runtime_t; pasta could not use its netns dir",
        );
    }

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
        let authority_template = format!("{}/setup-authority.XXXXXXXX", storage.scratch);
        let tmpd = match capture("mktemp", &["-d", &authority_template], false) {
            Captured::SpawnFailed(code) => return Err(Exit::Propagate(code)),
            Captured::Done(code, out) => {
                if code != 0 {
                    return Err(Exit::Propagate(code));
                }
                stripped_string(&out)
            }
        };
        cleanup.push(PathBuf::from(&tmpd));
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

    run("sudo", &["mkdir", "-p", ROOTFS_DIR])?;
    let rootfs_owner = String::from_utf8_lossy(&id_un()).into_owned();
    run("sudo", &["chown", &rootfs_owner, ROOTFS_DIR])?;
    if command_v("restorecon").is_some() {
        run("sudo", &["restorecon", ROOTFS_DIR])?;
    }

    println!("-- ready. Development candidate command from {pwd}:");
    println!("  sudo {ADMITTED} --worker-config {worker_json_path} --arch x86_64 \\");
    println!("    --out {output_parent}/manual-01 --development --target candidate \\");
    println!("    --forgejo-source {forgejo_source}");
    println!("-- installer rootfs pickup: {ROOTFS_DIR} (served by soda-rootfs-server.service)");
    println!("  after a media build, copy its hash-named rootfs here:");
    println!("  sudo cp <out>/artifacts/media/*-rootfs.img {ROOTFS_DIR}/ && sudo chown root:root {ROOTFS_DIR}/*-rootfs.img && sudo chmod 0644 {ROOTFS_DIR}/*-rootfs.img");
    if !rootfs_url.is_empty() {
        println!("-- guests fetch the filed rootfs from: {rootfs_url}");
    }
    Ok(())
}

/// `random_hex_passphrase` mirrors `head -c 32 /dev/urandom | od -An -tx1
/// | tr -d ' \n'`: 64 lowercase hex characters, no trailing newline.
fn random_hex_passphrase() -> Result<String, Exit> {
    let mut bytes = [0u8; 32];
    let mut urandom = fs::File::open("/dev/urandom").map_err(|_| Exit::Propagate(1))?;
    urandom
        .read_exact(&mut bytes)
        .map_err(|_| Exit::Propagate(1))?;
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

fn main() {
    // Rust ignores SIGPIPE at startup (a write to a closed pipe would return
    // EPIPE and println! would panic instead of dying like the shell);
    // restore the default disposition for byte parity.
    unsafe {
        let restore_pipe = Sigaction {
            handler: SIG_DFL,
            mask: [0; 16],
            flags: 0,
            restorer: 0,
        };
        sigaction(SIGPIPE, &restore_pipe, std::ptr::null_mut());
    }
    let mut cleanup: Vec<PathBuf> = Vec::new();
    let result = run_setup(&mut cleanup);
    for path in &cleanup {
        let _ = fs::remove_dir_all(path);
    }
    match result {
        Ok(()) => {}
        Err(Exit::Fail(msg)) => {
            eprintln!("{FAIL_PREFIX}: {msg}");
            std::process::exit(1);
        }
        Err(Exit::Propagate(code)) => std::process::exit(code),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::fs::PermissionsExt;

    #[test]
    fn json_escape_matches_python_dumps() {
        let input = "a\"b\\c\nd\re\tf\x08g\x0ch\x01i\x7fj\u{80}ké😀l\x1fm";
        assert_eq!(
            json_escape(input),
            "a\\\"b\\\\c\\nd\\re\\tf\\bg\\fh\\u0001i\\u007fj\\u0080k\\u00e9\\ud83d\\ude00l\\u001fm"
        );
        assert_eq!(json_escape("plain / path-_name.1"), "plain / path-_name.1");
        assert_eq!(json_escape(""), "");
    }

    #[test]
    fn worker_json_matches_python_dump_without_trailing_newline() {
        let got = worker_json(
            "/usr/local/lib/soda/soda-build",
            "/home/op/sodaos",
            "/home/op/forgejo",
            "/home/op/sodaos/.artifacts/releases/isolated",
            "/home/soda-candidate",
            "/home/soda-candidate/home",
            "/home/soda-candidate/run",
            "/var/lib/soda-candidate-tools",
            "/var/lib/soda-candidate-authority",
        );
        let want = "{\n  \"Executable\": \"/usr/local/lib/soda/soda-build\",\n  \"Source\": \"/home/op/sodaos\",\n  \"ForgejoSource\": \"/home/op/forgejo\",\n  \"OutputParent\": \"/home/op/sodaos/.artifacts/releases/isolated\",\n  \"StorageRoot\": \"/home/soda-candidate\",\n  \"BuildHome\": \"/home/soda-candidate/home\",\n  \"Runtime\": \"/home/soda-candidate/run\",\n  \"Tools\": \"/var/lib/soda-candidate-tools\",\n  \"MediaAuthorityDirectory\": \"/var/lib/soda-candidate-authority\"\n}";
        assert_eq!(got, want);
    }

    #[test]
    fn trust_json_matches_python_dump_with_trailing_newline() {
        let got = trust_json(
            "ghcr.io/levitateos/sodaos",
            424242,
            [
                "artifact-pub\n",
                "candidate-pub\n",
                "preview-pub\n",
                "stable-pub\n",
            ],
        );
        let want = "{\n  \"Format\": 1,\n  \"Prefix\": \"ghcr.io/levitateos/sodaos\",\n  \"Epoch\": 1,\n  \"Keys\": {\n    \"artifact\": [\n      \"artifact-pub\\n\"\n    ],\n    \"candidate\": [\n      \"candidate-pub\\n\"\n    ],\n    \"preview\": [\n      \"preview-pub\\n\"\n    ],\n    \"stable\": [\n      \"stable-pub\\n\"\n    ]\n  },\n  \"NotBefore\": 423642,\n  \"MaxAgeSeconds\": 3600,\n  \"ClockSkewSeconds\": 10,\n  \"MinimumSequence\": {\n    \"candidate\": 1,\n    \"preview\": 1,\n    \"stable\": 1\n  }\n}\n";
        assert_eq!(got, want);
    }

    #[test]
    fn config_json_matches_python_dump_with_trailing_newline() {
        let got = config_json();
        let want = "{\n  \"Trust\": \"/run/soda-media-authority/trust.json\",\n  \"Keys\": {\n    \"Key\": \"/run/soda-media-authority/artifact.private\",\n    \"Passphrase\": \"/run/soda-media-authority/passphrase\"\n  }\n}\n";
        assert_eq!(got, want);
    }

    #[test]
    fn units_have_active_matches_awk_third_field() {
        assert!(units_have_active(
            b"soda-build-x.service loaded active running desc\n"
        ));
        assert!(units_have_active(
            b"soda-build-x.service loaded activating start desc\n"
        ));
        assert!(units_have_active(
            b"soda-build-x.service loaded deactivating stop desc\n"
        ));
        assert!(!units_have_active(
            b"soda-build-x.service loaded failed failed desc\n"
        ));
        assert!(!units_have_active(b""));
        assert!(!units_have_active(b"short line\n"));
        assert!(!units_have_active(
            b"other.service loaded inactive dead desc\nsoda-build-y.service loaded failed failed desc\n"
        ));
        assert!(units_have_active(
            b"other.service loaded inactive dead desc\nsoda-build-y.service loaded active running desc\n"
        ));
    }

    #[test]
    fn bridge_ip_takes_first_line_fourth_field_before_slash() {
        assert_eq!(
            bridge_ip(
                b"2: virbr0    inet 192.168.122.1/24 brd 192.168.122.255 scope global virbr0\n"
            ),
            Some("192.168.122.1".to_string())
        );
        assert_eq!(bridge_ip(b""), None);
        assert_eq!(bridge_ip(b"2: virbr0\n"), None);
        assert_eq!(
            bridge_ip(b"2: virbr0    inet 10.0.0.1 scope global\n"),
            Some("10.0.0.1".to_string())
        );
    }

    #[test]
    fn env_or_falls_back_on_unset_or_empty() {
        let key = "SODA_CANDIDATE_SETUP_TEST_ENV_OR";
        env::remove_var(key);
        assert_eq!(env_or(key, "dflt"), "dflt");
        env::set_var(key, "");
        assert_eq!(env_or(key, "dflt"), "dflt");
        env::set_var(key, "value");
        assert_eq!(env_or(key, "dflt"), "value");
        env::remove_var(key);
    }

    #[test]
    fn stripped_removes_only_trailing_newlines() {
        assert_eq!(stripped(b"a\n"), b"a");
        assert_eq!(stripped(b"a\n\n\n"), b"a");
        assert_eq!(stripped(b"a"), b"a");
        assert_eq!(stripped(b""), b"");
        assert_eq!(stripped(b"a\nb\n"), b"a\nb");
        assert_eq!(stripped(b"\n"), b"");
    }

    #[test]
    fn command_v_searches_path_for_executables() {
        let dir = env::temp_dir().join(format!("soda-setup-cmdv-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        fs::create_dir_all(&dir).unwrap();
        let tool = dir.join("soda-test-tool");
        let flat = dir.join("soda-test-flat");
        fs::write(&tool, b"#!/bin/sh\nexit 0\n").unwrap();
        fs::write(&flat, b"data").unwrap();
        fs::set_permissions(&tool, fs::Permissions::from_mode(0o755)).unwrap();
        fs::set_permissions(&flat, fs::Permissions::from_mode(0o644)).unwrap();
        let found = command_v_in("soda-test-tool", dir.as_os_str());
        let missing = command_v_in("soda-test-flat", dir.as_os_str()).is_none()
            && command_v_in("soda-test-absent", dir.as_os_str()).is_none();
        let _ = fs::remove_dir_all(&dir);
        assert_eq!(found, Some(tool));
        assert!(missing);
    }

    #[test]
    fn pipe2_reports_last_nonzero_by_position() {
        assert_eq!(pipe2(("true", &[]), ("true", &[])), (0, Vec::new()));
        assert_eq!(pipe2(("true", &[]), ("false", &[])).0, 1);
        // pipefail: the first failure still fails the pipeline.
        assert_eq!(pipe2(("false", &[]), ("true", &[])).0, 1);
        let (code, out) = pipe2(("echo", &["hi"]), ("tr", &["a-z", "A-Z"]));
        assert_eq!((code, out), (0, b"HI\n".to_vec()));
    }

    #[test]
    fn write_staged_matches_redirection_bytes_and_mode() {
        let path = env::temp_dir().join(format!("soda-setup-stage-{}", std::process::id()));
        let _ = fs::remove_file(&path);
        write_staged(&path, b"bytes\n").unwrap();
        assert_eq!(fs::read(&path).unwrap(), b"bytes\n");
        let mode = fs::metadata(&path).unwrap().permissions().mode() & 0o777;
        assert_eq!(mode, 0o666 & !current_umask() & 0o777);
        let _ = fs::remove_file(&path);
    }

    #[test]
    fn passphrase_is_64_lowercase_hex_without_newline() {
        let passphrase = random_hex_passphrase().unwrap();
        assert_eq!(passphrase.len(), 64);
        assert!(passphrase
            .bytes()
            .all(|b| b.is_ascii_hexdigit() && !b.is_ascii_uppercase()));
        assert_ne!(
            random_hex_passphrase().unwrap(),
            random_hex_passphrase().unwrap()
        );
    }

    #[test]
    fn run_in_dir_maps_missing_directory_to_fail_message() {
        match run_in_dir(
            "true",
            &[],
            "/definitely/not/a/soda-setup-dir",
            true,
            "cannot warm Bun cache",
        ) {
            Err(Exit::Fail(msg)) => assert_eq!(msg, "cannot warm Bun cache"),
            other => panic!("expected Fail, got {other:?}"),
        }
    }

    #[test]
    fn git_tree_clean_requires_successful_empty_status() {
        // D01-F3: only a successful status with empty output proves clean.
        assert!(git_tree_clean(&Captured::Done(0, Vec::new())));
        assert!(git_tree_clean(&Captured::Done(0, b"\n".to_vec())));
        assert!(!git_tree_clean(&Captured::Done(
            0,
            b" M src/main.rs\n".to_vec()
        )));
        assert!(!git_tree_clean(&Captured::Done(1, Vec::new())));
        assert!(!git_tree_clean(&Captured::SpawnFailed(127)));
    }

    #[test]
    fn controller_build_selects_rust_release_tools() {
        // D01-F2: pinned Rust recipe, never the retired Go paths.
        let argv = controller_cargo_argv();
        assert_eq!(
            argv.as_slice(),
            &[
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
        );
        assert!(!argv.iter().any(|a| a.contains("tools/soda-")));
    }

    #[test]
    fn worker_policy_input_resolves_in_checkout() {
        // CORR-C-005: the checkmodule input must resolve at its selected location.
        let manifest = Path::new(env!("CARGO_MANIFEST_DIR"));
        let root = manifest.ancestors().nth(2).unwrap();
        assert!(root.join(WORKER_POLICY_SRC).is_file());
    }
}

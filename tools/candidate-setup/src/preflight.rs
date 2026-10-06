use std::env;
use std::fs;

use super::process::{capture, git_tree_clean, id_un, pipe2, run_stdout_null, Captured};
use super::{
    command_v, current_pwd, env_or, flock, is_dir, is_file, refuse_active_build, stripped,
    stripped_string, Storage, AUTHORITY, FAIL_PREFIX, LOCK_EX, LOCK_NB, PREFIX_DEFAULT,
    STORAGE_ROOT_DEFAULT, WORKER_USER,
};
use super::{fail, Exit};

/// Admitted setup inputs: environment, native toolchain pins and the
/// held setup lease. Later stages borrow or destructure exactly these.
pub(super) struct Preflight {
    pub(super) prefix: String,
    pub(super) refresh: String,
    pub(super) forgejo_source: String,
    pub(super) storage: Storage,
    pub(super) rootfs_url: String,
    pub(super) pwd: String,
    pub(super) output_parent: String,
    pub(super) worker_json_path: String,
    pub(super) pinned: String,
    pub(super) pinned_goroot: String,
    pub(super) want: String,
}

/// `bridge_ip` mirrors the `ip ... | awk ... | cut ... | head -1` pickup:
/// the first line's fourth field before any `/`.
pub(super) fn bridge_ip(ip_output: &[u8]) -> Option<String> {
    let text = String::from_utf8_lossy(ip_output);
    let line = text.lines().next()?;
    let field = line.split_whitespace().nth(3)?;
    let addr = field.split('/').next()?;
    if addr.is_empty() {
        None
    } else {
        Some(addr.to_string())
    }
}

/// Input, native-toolchain and lease admission: the setup stages below
/// only run once this returns.
pub(super) fn preflight() -> Result<Preflight, Exit> {
    // Like the script, which parsed none, all arguments are ignored.
    let prefix = env_or("SODA_REPOSITORY_PREFIX", PREFIX_DEFAULT);
    let refresh = env_or("SODA_REFRESH_AUTHORITY", "0");
    let forgejo_source = env_or("SODA_FORGEJO_SOURCE", "");
    let storage_root = env_or("SODA_CANDIDATE_ROOT", STORAGE_ROOT_DEFAULT);
    let storage = Storage {
        home: env_or("SODA_CANDIDATE_HOME", &format!("{storage_root}/home")),
        run: env_or("SODA_CANDIDATE_RUN", &format!("{storage_root}/run")),
        scratch: env_or("SODA_CANDIDATE_SCRATCH", &format!("{storage_root}/scratch")),
        root: storage_root,
    };

    let mut rootfs_url = String::new();
    if let Captured::Done(code, out) = capture("ip", &["-4", "-o", "addr", "show", "virbr0"], true)
    {
        if code == 0 {
            if let Some(addr) = bridge_ip(&out) {
                rootfs_url = format!("http://{addr}:8080");
            }
        }
    }

    let pwd = current_pwd();
    let output_parent = format!("{pwd}/.artifacts/releases/isolated");
    let worker_json_path = format!("{AUTHORITY}/worker.json");

    if !is_file("go.mod") {
        return fail("run from the repository root");
    }
    if forgejo_source.is_empty() {
        return fail("set SODA_FORGEJO_SOURCE to the clean canonical Forgejo fork checkout");
    }
    let forgejo_git = format!("{forgejo_source}/.git");
    let canonical_ok = is_dir(&forgejo_git)
        && match capture("realpath", &[forgejo_source.as_str()], false) {
            Captured::SpawnFailed(_) => false,
            Captured::Done(code, out) => code == 0 && stripped(&out) == forgejo_source.as_bytes(),
        };
    if !canonical_ok {
        return fail("canonical Forgejo checkout required");
    }
    let safe_dir = format!("safe.directory={forgejo_source}");
    // D01-F3: a failed inspection must not pass as clean.
    let forgejo_clean = git_tree_clean(&capture(
        "git",
        &[
            "-c",
            &safe_dir,
            "-C",
            &forgejo_source,
            "status",
            "--porcelain",
            "--untracked-files=normal",
        ],
        false,
    ));
    if !forgejo_clean {
        return fail("Forgejo source must be clean and committed");
    }
    let arch_ok = match capture("uname", &["-m"], false) {
        Captured::SpawnFailed(_) => false,
        Captured::Done(_, out) => stripped(&out) == b"x86_64",
    };
    if !arch_ok {
        return fail("matching native x86_64 required on this host");
    }
    for tool in ["go", "bun", "podman", "skopeo", "flock"] {
        if command_v(tool).is_none() {
            return fail(format!(
                "pinned go, bun, podman, skopeo and flock required (missing {tool})"
            ));
        }
    }
    let user_ok = match capture("id", &[WORKER_USER], true) {
        Captured::SpawnFailed(_) => false,
        Captured::Done(code, _) => code == 0,
    };
    if !user_ok {
        return fail("soda-build-worker user missing");
    }
    // D01-F3: only a successful empty status proves clean.
    let dirty = !git_tree_clean(&capture(
        "git",
        &["status", "--porcelain", "--untracked-files=no"],
        false,
    ));
    if dirty {
        return fail("commit or stash tracked changes first; the controller refuses dirty source");
    }
    let (pin_code, pin_out) = pipe2(("grep", &["^go ", "go.mod"]), ("awk", &["{print $2}"]));
    if pin_code != 0 {
        return Err(Exit::Propagate(pin_code));
    }
    let pinned = stripped_string(&pin_out);
    if pinned.is_empty() {
        return fail("go.mod pins no Go version");
    }
    env::set_var("GOTOOLCHAIN", format!("go{pinned}"));
    if run_stdout_null("go", &["version"]).is_err() {
        return fail(format!("cannot fetch Go {pinned}"));
    }
    let pinned_goroot = match capture("go", &["env", "GOROOT"], false) {
        Captured::SpawnFailed(code) => return Err(Exit::Propagate(code)),
        Captured::Done(code, out) => {
            if code != 0 {
                return Err(Exit::Propagate(code));
            }
            stripped_string(&out)
        }
    };
    let want = format!("go version go{pinned} linux/amd64");

    let user = String::from_utf8_lossy(&id_un()).into_owned();
    let lock_path = format!("/tmp/soda-setup-{user}.lock");
    let lease = match fs::OpenOptions::new()
        .write(true)
        .create(true)
        .truncate(true)
        .open(&lock_path)
    {
        Ok(file) => file,
        Err(err) => {
            eprintln!("{FAIL_PREFIX}: {lock_path}: {err}");
            return fail(format!("cannot open setup lease {lock_path}"));
        }
    };
    {
        use std::os::unix::io::AsRawFd;
        if unsafe { flock(lease.as_raw_fd(), LOCK_EX | LOCK_NB) } != 0 {
            return fail("another setup is already running for this operator");
        }
    }
    // Like the script's fd 9, the lease stays held until process exit, not
    // just until this function returns and the staging cleanup runs.
    std::mem::forget(lease);
    refuse_active_build()?;
    Ok(Preflight {
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
    })
}

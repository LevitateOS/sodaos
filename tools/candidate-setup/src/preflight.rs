use std::env;
use std::fs;

use super::process::{capture, git_tree_clean, id_un, Captured};
use super::storage::{refuse_active_build, Storage};
use super::{
    command_v, current_pwd, env_or, flock, is_dir, is_file, stripped, stripped_string, AUTHORITY,
    FAIL_PREFIX, LOCK_EX, LOCK_NB, STORAGE_ROOT_DEFAULT, WORKER_USER,
};
use super::{fail, Exit};

/// Admitted setup inputs: environment, installed toolchains and the
/// held setup lease. Later stages borrow or destructure exactly these.
pub(super) struct Preflight {
    pub(super) forgejo_source: String,
    pub(super) fountain_release_url: String,
    pub(super) storage: Storage,
    pub(super) rootfs_url: String,
    pub(super) pwd: String,
    pub(super) output_parent: String,
    pub(super) worker_json_path: String,
    pub(super) go_root: String,
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

fn admitted_inputs() -> Result<(String, String), Exit> {
    // Like the script, which parsed none, all arguments are ignored.
    let forgejo_source = env_or("SODA_FORGEJO_SOURCE", "");
    let fountain_release_url = env_or("SODA_FOUNTAIN_RELEASE_URL", "");
    if !soda_build_tools::reader::url::https_url(&fountain_release_url)
        || !fountain_release_url.ends_with("/release.json")
    {
        return fail(
            "set SODA_FOUNTAIN_RELEASE_URL to the selected immutable HTTPS Fountain release.json",
        );
    }
    Ok((forgejo_source, fountain_release_url))
}

fn admitted_storage() -> Storage {
    let storage_root = env_or("SODA_CANDIDATE_ROOT", STORAGE_ROOT_DEFAULT);
    Storage {
        home: env_or("SODA_CANDIDATE_HOME", &format!("{storage_root}/home")),
        run: env_or("SODA_CANDIDATE_RUN", &format!("{storage_root}/run")),
        scratch: env_or("SODA_CANDIDATE_SCRATCH", &format!("{storage_root}/scratch")),
        root: storage_root,
    }
}

fn bridge_rootfs_url() -> String {
    let mut rootfs_url = String::new();
    if let Captured::Done(code, out) = capture("ip", &["-4", "-o", "addr", "show", "virbr0"], true)
    {
        if code == 0 {
            if let Some(addr) = bridge_ip(&out) {
                rootfs_url = format!("http://{addr}:8080");
            }
        }
    }
    rootfs_url
}

fn admitted_checkout(forgejo_source: &str) -> Result<(), Exit> {
    if forgejo_source.is_empty() {
        return fail("set SODA_FORGEJO_SOURCE to the canonical Fountain SDK checkout");
    }
    let forgejo_git = format!("{forgejo_source}/.git");
    let canonical_ok = is_dir(&forgejo_git)
        && match capture("realpath", &[forgejo_source], false) {
            Captured::SpawnFailed(_) => false,
            Captured::Done(code, out) => code == 0 && stripped(&out) == forgejo_source.as_bytes(),
        };
    if !canonical_ok {
        return fail("canonical Forgejo checkout required");
    }
    let safe_dir = format!("safe.directory={forgejo_source}");
    // The build controller verifies the SDK against the selected release pin.
    // Unrelated Fountain changes do not select or replace released image bytes.
    let forgejo_committed = matches!(
        capture(
            "git",
            &[
                "-c",
                &safe_dir,
                "-C",
                forgejo_source,
                "rev-parse",
                "--verify",
                "HEAD^{commit}",
            ],
            true,
        ),
        Captured::Done(0, _)
    );
    if !forgejo_committed {
        return fail("committed canonical Fountain SDK checkout required");
    }
    Ok(())
}

fn admitted_host() -> Result<(), Exit> {
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
                "go, bun, podman, skopeo and flock required (missing {tool})"
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
    Ok(())
}

fn admitted_toolchain() -> Result<String, Exit> {
    // D01-F3: only a successful empty status proves clean.
    let dirty = !git_tree_clean(&capture(
        "git",
        &[
            "--no-optional-locks",
            "status",
            "--porcelain",
            "--untracked-files=no",
        ],
        false,
    ));
    if dirty {
        return fail("commit or stash tracked changes first; the controller refuses dirty source");
    }
    env::set_var("GOTOOLCHAIN", "local");
    let go_root = match capture("go", &["env", "GOROOT"], false) {
        Captured::SpawnFailed(code) => return Err(Exit::Propagate(code)),
        Captured::Done(code, out) => {
            if code != 0 {
                return Err(Exit::Propagate(code));
            }
            stripped_string(&out)
        }
    };
    Ok(go_root)
}

fn held_lease() -> Result<(), Exit> {
    let user = String::from_utf8_lossy(&id_un()).into_owned();
    let lock_path = env::temp_dir()
        .join(format!("soda-setup-{user}.lock"))
        .to_string_lossy()
        .into_owned();
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
    Ok(())
}

/// Input, native-toolchain and lease admission: the setup stages below
/// only run once this returns.
pub(super) fn preflight() -> Result<Preflight, Exit> {
    let (forgejo_source, fountain_release_url) = admitted_inputs()?;
    let storage = admitted_storage();
    let rootfs_url = bridge_rootfs_url();
    let pwd = current_pwd();
    let output_parent = format!("{pwd}/.artifacts/releases/isolated");
    let worker_json_path = format!("{AUTHORITY}/worker.json");
    if !is_file("go.mod") {
        return fail("run from the repository root");
    }
    admitted_checkout(&forgejo_source)?;
    admitted_host()?;
    let go_root = admitted_toolchain()?;
    held_lease()?;
    Ok(Preflight {
        forgejo_source,
        fountain_release_url,
        storage,
        rootfs_url,
        pwd,
        output_parent,
        worker_json_path,
        go_root,
    })
}

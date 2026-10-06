//! Fixed bounded factory role helper: `ensure`/`approve`/`record`/`start`
//! /`inspect`/`stop`/`hold`/`release` over `/var/lib/soda/factory`.
//!
//! One JSON request on stdin, one JSON document on stdout, exit 0; every
//! failure prints the fixed stderr line and exits 1. No caller-supplied
//! command, package, image or path is ever honored.
//!
//! Test seam: `SODA_FACTORY_DIR` redirects the state root (plus homes,
//! key checks and account mapping) to scratch and `SODA_FACTORY_GIT`
//! overrides the verification program. Both are honored only together;
//! production (unset) behavior is byte-identical to the original.

#[path = "accounts.rs"]
pub(crate) mod account;
#[path = "../../../../rust/soda-project-factory-roles/src/b64.rs"]
pub(crate) mod b64;
#[path = "../../../../rust/soda-project-factory-roles/src/emit.rs"]
pub(crate) mod emit;
#[path = "../../../../rust/soda-project-factory-roles/src/error.rs"]
pub(crate) mod error;
#[path = "layout.rs"]
pub(crate) mod fsx;
#[path = "../../../../rust/soda-project-factory-roles/src/ops_approve.rs"]
pub(crate) mod ops_approve;
#[path = "../../../../rust/soda-project-factory-roles/src/ops_inspect.rs"]
pub(crate) mod ops_inspect;
#[path = "../../../../rust/soda-project-factory-roles/src/ops_record.rs"]
pub(crate) mod ops_record;
#[path = "../../../../rust/soda-project-factory-roles/src/proc.rs"]
pub(crate) mod proc;
#[path = "../../../../rust/soda-project-factory-roles/src/sha.rs"]
pub(crate) mod sha;
#[cfg(test)]
#[path = "tests.rs"]
pub(crate) mod testutil;
#[path = "../../../../rust/soda-project-factory-roles/src/validate.rs"]
pub(crate) mod validate;

use error::{fail, Error};
use soda_json::JsonValue;
use std::io::{Read, Write};
use std::path::{Path, PathBuf};

pub const SETUP_ENTRY: &str = "setup.sh";
pub const CHECK_ENTRY: &str = "check.sh";
pub const SHELL: &str = "/bin/bash";
pub const GIT: &str = "/usr/bin/git";
pub const LOG_CAP: usize = 65536;
pub const INPUT_CAP: usize = 4 * 1024 * 1024 + 65536;
const FACTORY_DEFAULT: &str = "/var/lib/soda/factory";

/// Scratch-mode overrides. `Some` if and only if `SODA_FACTORY_DIR` is set
/// and non-empty; production paths and checks apply otherwise.
pub struct TestCtx {
    pub git: PathBuf,
}

/// Resolved layout plus the privilege expectation for ownership checks.
pub struct Ctx {
    pub factory: PathBuf,
    pub preparations: PathBuf,
    pub credentials: PathBuf,
    pub hold: PathBuf,
    pub lock: PathBuf,
    pub test: Option<TestCtx>,
}

impl Ctx {
    pub fn production() -> Ctx {
        Ctx::at(PathBuf::from(FACTORY_DEFAULT), None)
    }

    pub fn test_at(factory: PathBuf, git: PathBuf) -> Ctx {
        Ctx::at(factory, Some(TestCtx { git }))
    }

    fn at(factory: PathBuf, test: Option<TestCtx>) -> Ctx {
        Ctx {
            preparations: factory.join("preparations"),
            credentials: factory.join("credentials"),
            hold: factory.join("maintenance-hold"),
            lock: factory.join("lock"),
            factory,
            test,
        }
    }

    pub fn from_env() -> Ctx {
        match std::env::var_os("SODA_FACTORY_DIR") {
            Some(dir) if !dir.is_empty() => {
                let git = std::env::var_os("SODA_FACTORY_GIT")
                    .map(PathBuf::from)
                    .filter(|path| !path.as_os_str().is_empty())
                    .unwrap_or_else(|| PathBuf::from(GIT));
                Ctx::test_at(PathBuf::from(dir), git)
            }
            _ => Ctx::production(),
        }
    }

    /// Expected owner of helper state: root in production, the invoking
    /// user under the test seam.
    pub fn priv_uid(&self) -> u32 {
        if self.test.is_some() {
            unsafe { libc::geteuid() }
        } else {
            0
        }
    }

    pub fn priv_gid(&self) -> u32 {
        if self.test.is_some() {
            unsafe { libc::getegid() }
        } else {
            0
        }
    }

    pub fn git(&self) -> &Path {
        match &self.test {
            Some(test) => &test.git,
            None => Path::new(GIT),
        }
    }
}

fn read_stdin_capped() -> Result<Vec<u8>, Error> {
    let mut body = Vec::new();
    std::io::stdin()
        .take((INPUT_CAP as u64) + 1)
        .read_to_end(&mut body)
        .map_err(|err| Error::io("stdin", &err))?;
    Ok(body)
}

fn dispatch_op(ctx: &Ctx, op: &str, data: &JsonValue) -> Result<JsonValue, Error> {
    match op {
        "ensure" => ops_approve::do_ensure(ctx, data),
        "approve" => ops_approve::do_approve(ctx, data),
        "record" => ops_record::do_record(ctx, data),
        "start" => ops_record::do_start(ctx, data),
        "inspect" => ops_inspect::do_inspect(ctx, data),
        "stop" => ops_inspect::do_stop(ctx, data),
        "hold" => ops_inspect::do_hold(ctx, data),
        "release" => ops_inspect::do_release(ctx, data),
        _ => fail("unsupported factory operation"),
    }
}

fn main_inner(ctx: &Ctx) -> Result<(), Error> {
    if unsafe { libc::geteuid() } != 0 && ctx.test.is_none() {
        return fail("project-local root required");
    }
    let body = read_stdin_capped()?;
    if body.len() > INPUT_CAP {
        return fail("oversized request");
    }
    let text = std::str::from_utf8(&body).map_err(|_| Error::fail("undecodable request"))?;
    let data = JsonValue::parse(text).map_err(|_| Error::fail("unsupported factory operation"))?;
    if validate::as_object(&data).is_none() {
        return fail("unsupported factory operation");
    }
    let op = match data.get("op").and_then(|v| v.as_str()) {
        Some(op)
            if matches!(
                op,
                "ensure" | "approve" | "record" | "start" | "inspect" | "stop" | "hold" | "release"
            ) =>
        {
            op
        }
        _ => return fail("unsupported factory operation"),
    };
    fsx::ensure_layout(ctx)?;
    // The state lock serializes mutating ops; `inspect` shares it.
    let lock_file = fsx::open_ro(&ctx.lock, false)?;
    use std::os::unix::io::AsRawFd;
    let fd = lock_file.as_raw_fd();
    let flag = if op == "inspect" {
        libc::LOCK_SH
    } else {
        libc::LOCK_EX
    };
    if unsafe { libc::flock(fd, flag) } != 0 {
        return Err(Error::io_msg("flock failed"));
    }
    let result = dispatch_op(ctx, op, &data);
    let line = result.map(|value| {
        let mut text = emit::dumps_default(&value);
        text.push('\n');
        text
    });
    drop(lock_file);
    let line = line?;
    std::io::stdout()
        .write_all(line.as_bytes())
        .map_err(|err| Error::io("stdout", &err))?;
    std::io::stdout()
        .flush()
        .map_err(|err| Error::io("stdout", &err))?;
    Ok(())
}

pub(crate) fn run() -> i32 {
    unsafe {
        libc::umask(0o022);
    }
    let ctx = Ctx::from_env();
    let outcome = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| main_inner(&ctx)));
    match outcome {
        Ok(Ok(())) => 0,
        Ok(Err(_)) | Err(_) => {
            eprintln!("{}", error::FIXED_MESSAGE);
            1
        }
    }
}

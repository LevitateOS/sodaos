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

mod account;
mod b64;
mod emit;
mod error;
mod fsx;
mod ops_approve;
mod ops_inspect;
mod ops_record;
mod proc;
mod sha;
mod validate;

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

fn run() -> i32 {
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

fn main() {
    std::process::exit(run());
}

/// In-process test fixtures: fresh scratch factories plus a recording
/// fake `git`. Never process-global state, so tests stay parallel-safe.
#[cfg(test)]
pub(crate) mod testutil {
    use super::Ctx;
    use std::path::PathBuf;
    use std::sync::atomic::{AtomicU64, Ordering};

    static NEXT: AtomicU64 = AtomicU64::new(0);

    pub struct Scratch {
        pub root: PathBuf,
        pub factory: PathBuf,
    }

    impl Scratch {
        pub fn fresh() -> Scratch {
            let id = NEXT.fetch_add(1, Ordering::SeqCst);
            let root =
                std::env::temp_dir().join(format!("factory-roles-{}-{id}", std::process::id()));
            let _ = std::fs::remove_dir_all(&root);
            std::fs::create_dir_all(&root).expect("scratch root");
            let factory = root.join("factory");
            Scratch { root, factory }
        }

        /// Recording fake `git`: appends each argv (`<arg>` per line under
        /// a `---` marker) to the record file, then exits `code`.
        pub fn git_script(&self, name: &str, code: i32) -> (PathBuf, PathBuf) {
            let record = self.root.join(format!("{name}.record"));
            let script = self.root.join(name);
            let body = format!(
                "#!/bin/sh\n{{ echo '---'; printf '<%s>\\n' \"$@\"; }} >> '{}'\nexit {code}\n",
                record.to_string_lossy(),
            );
            std::fs::write(&script, body).expect("git script");
            use std::os::unix::fs::PermissionsExt;
            std::fs::set_permissions(&script, std::fs::Permissions::from_mode(0o755))
                .expect("chmod script");
            (script, record)
        }

        pub fn ctx(&self, git: &std::path::Path) -> Ctx {
            Ctx::test_at(self.factory.clone(), git.to_path_buf())
        }

        pub fn record_text(&self, record: &std::path::Path) -> String {
            std::fs::read_to_string(record).unwrap_or_default()
        }
    }

    impl Drop for Scratch {
        fn drop(&mut self) {
            let _ = std::fs::remove_dir_all(&self.root);
        }
    }

    pub const PID: &str = "f0123456789abcdef01234567";
    pub const PID2: &str = "f123456789abcdef012345678";
    pub const COMMIT: &str = "cccccccccccccccccccccccccccccccccccccccc";

    pub fn b64_encode(data: &[u8]) -> String {
        const ALPHA: &[u8; 64] =
            b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
        let mut out = String::new();
        for chunk in data.chunks(3) {
            let mut block = [0u8; 3];
            block[..chunk.len()].copy_from_slice(chunk);
            let triple =
                (u32::from(block[0]) << 16) | (u32::from(block[1]) << 8) | u32::from(block[2]);
            out.push(ALPHA[(triple >> 18) as usize] as char);
            out.push(ALPHA[((triple >> 12) & 63) as usize] as char);
            if chunk.len() > 1 {
                out.push(ALPHA[((triple >> 6) & 63) as usize] as char);
            } else {
                out.push('=');
            }
            if chunk.len() > 2 {
                out.push(ALPHA[(triple & 63) as usize] as char);
            } else {
                out.push('=');
            }
        }
        out
    }

    pub fn fixture_files() -> Vec<(String, Vec<u8>)> {
        vec![
            ("setup.sh".to_string(), b"true\n".to_vec()),
            ("check.sh".to_string(), b"true\n".to_vec()),
        ]
    }

    pub fn approve_value(
        pid: &str,
        role: &str,
        files: &[(String, Vec<u8>)],
        bundle: &[u8],
        credential: &str,
    ) -> soda_json::JsonValue {
        let digest = crate::sha::approved_digest(files);
        let files_obj: Vec<(String, soda_json::JsonValue)> = files
            .iter()
            .map(|(name, contents)| {
                (
                    name.clone(),
                    soda_json::JsonValue::Str(b64_encode(contents)),
                )
            })
            .collect();
        crate::emit::obj(vec![
            ("op", soda_json::JsonValue::Str("approve".to_string())),
            ("id", soda_json::JsonValue::Str(pid.to_string())),
            ("role", soda_json::JsonValue::Str(role.to_string())),
            ("setup_digest", soda_json::JsonValue::Str(digest)),
            (
                "source_commit",
                soda_json::JsonValue::Str(COMMIT.to_string()),
            ),
            ("files", soda_json::JsonValue::Object(files_obj)),
            ("bundle", soda_json::JsonValue::Str(b64_encode(bundle))),
            (
                "credential",
                soda_json::JsonValue::Str(credential.to_string()),
            ),
        ])
    }

    pub fn approve_default(pid: &str) -> soda_json::JsonValue {
        approve_value(pid, "soda-coder", &fixture_files(), b"bundle", "")
    }

    pub fn record_value(pid: &str, missing: &str, refusal: Option<&str>) -> soda_json::JsonValue {
        let mut verified = vec![
            (
                "uid".to_string(),
                soda_json::JsonValue::Str("1000".to_string()),
            ),
            (
                "login".to_string(),
                soda_json::JsonValue::Str("soda-coder".to_string()),
            ),
            (
                "groups".to_string(),
                soda_json::JsonValue::Str("soda-coder".to_string()),
            ),
        ];
        if let Some(text) = refusal {
            verified.push((
                "refusal".to_string(),
                soda_json::JsonValue::Str(text.to_string()),
            ));
        }
        crate::emit::obj(vec![
            ("op", soda_json::JsonValue::Str("record".to_string())),
            ("id", soda_json::JsonValue::Str(pid.to_string())),
            ("tools", soda_json::JsonValue::Array(Vec::new())),
            ("missing", soda_json::JsonValue::Str(missing.to_string())),
            ("verified", soda_json::JsonValue::Object(verified)),
        ])
    }

    pub fn op_value(op: &str, pid: Option<&str>) -> soda_json::JsonValue {
        let mut pairs = vec![("op", soda_json::JsonValue::Str(op.to_string()))];
        if let Some(pid) = pid {
            pairs.push(("id", soda_json::JsonValue::Str(pid.to_string())));
        }
        crate::emit::obj(pairs)
    }

    /// Assert a `Fail` carrying the exact `.py` message.
    pub fn assert_fail(result: Result<soda_json::JsonValue, crate::error::Error>, message: &str) {
        match result {
            Err(crate::error::Error::Fail(text)) => assert_eq!(text, message),
            other => panic!("expected Fail({message:?}), got {other:?}"),
        }
    }

    #[test]
    fn b64_round_trip() {
        for len in [0, 1, 2, 3, 5, 55, 256] {
            let data: Vec<u8> = (0..len).map(|i| (i * 37 + 11) as u8).collect();
            if len == 0 {
                assert_eq!(b64_encode(&data), "");
            } else {
                assert_eq!(crate::b64::decode(&b64_encode(&data)).unwrap(), data);
            }
        }
    }
}

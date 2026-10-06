//! Hardened state-file primitives: exact ports of `read_json`, `write_new`,
//! `owned_dir`, `ensure_layout` and `prep_dir` (`O_NOFOLLOW` everywhere the
//! `.py` uses it, same modes, same bounds).

use super::fsx;
use crate::emit::obj;
use crate::error::{fail, Error};
use crate::validate;
use soda_json::JsonValue;
use std::fs::{File, OpenOptions};
use std::io::{Read, Write};
use std::os::unix::fs::{DirBuilderExt, MetadataExt, OpenOptionsExt, PermissionsExt};
use std::path::Path;

/// `os.path.lexists`: true for anything including dangling symlinks.
pub fn lexists(path: &Path) -> bool {
    std::fs::symlink_metadata(path).is_ok()
}

pub fn file_name(path: &Path) -> String {
    path.file_name()
        .map(|name| name.to_string_lossy().into_owned())
        .unwrap_or_default()
}

/// `Path.mkdir(parents=True, exist_ok=True)` with an explicit mode.
pub fn mkdir_p(path: &Path, mode: u32) -> Result<(), Error> {
    std::fs::DirBuilder::new()
        .recursive(true)
        .mode(mode)
        .create(path)
        .map_err(Error::classify)
}

/// `os.chown` (follows symlinks, like the `.py`).
pub fn chown(path: &Path, uid: u32, gid: u32) -> Result<(), Error> {
    let raw = std::ffi::CString::new(path.as_os_str().as_encoded_bytes())
        .map_err(|_| Error::io_msg("path holds NUL"))?;
    if unsafe { libc::chown(raw.as_ptr(), uid, gid) } != 0 {
        return Err(Error::io_msg(format!(
            "chown {}: {}",
            path.display(),
            std::io::Error::last_os_error()
        )));
    }
    Ok(())
}

/// `os.chmod` (follows symlinks, like the `.py`).
pub fn chmod(path: &Path, mode: u32) -> Result<(), Error> {
    std::fs::set_permissions(path, std::fs::Permissions::from_mode(mode)).map_err(Error::classify)
}

/// Open read-only with `O_NOFOLLOW` (plus `O_NONBLOCK` where the `.py`
/// passes it); missing maps to [`Error::Missing`], `ELOOP` stays IO.
pub fn open_ro(path: &Path, nonblock: bool) -> Result<File, Error> {
    let mut flags = libc::O_NOFOLLOW;
    if nonblock {
        flags |= libc::O_NONBLOCK;
    }
    OpenOptions::new()
        .read(true)
        .custom_flags(flags)
        .open(path)
        .map_err(Error::classify)
}

/// Hardened metadata read: regular file, privileged owner, single link,
/// bounded size, valid JSON.
pub fn read_json(ctx: &crate::Ctx, path: &Path, limit: usize) -> Result<JsonValue, Error> {
    let file = open_ro(path, true)?;
    let meta = file.metadata().map_err(Error::classify)?;
    if !meta.is_file() || meta.uid() != ctx.priv_uid() || meta.nlink() != 1 {
        return fail(format!("unsafe factory metadata: {}", file_name(path)));
    }
    let mut raw = Vec::new();
    file.take((limit as u64) + 1)
        .read_to_end(&mut raw)
        .map_err(|err| Error::io("read", &err))?;
    if raw.len() > limit {
        return fail(format!("oversized factory metadata: {}", file_name(path)));
    }
    let text =
        std::str::from_utf8(&raw).map_err(|_| Error::fail("undecodable factory metadata"))?;
    JsonValue::parse(text).map_err(|_| Error::fail("undecodable factory metadata"))
}

/// Exclusive create (`O_EXCL | O_NOFOLLOW`), `0o600` at creation, then
/// `fchmod` to the final mode plus `fsync`, like `write_new`.
pub fn write_new(path: &Path, raw: &[u8], mode: u32) -> Result<(), Error> {
    let file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .custom_flags(libc::O_NOFOLLOW)
        .mode(0o600)
        .open(path)
        .map_err(Error::classify)?;
    write_new_file(&file, raw, mode)
}

fn write_new_file(file: &File, raw: &[u8], mode: u32) -> Result<(), Error> {
    (&*file)
        .write_all(raw)
        .map_err(|err| Error::io("write", &err))?;
    (&*file).flush().map_err(|err| Error::io("write", &err))?;
    file.set_permissions(std::fs::Permissions::from_mode(mode))
        .map_err(|err| Error::io("fchmod", &err))?;
    file.sync_all().map_err(|err| Error::io("fsync", &err))?;
    Ok(())
}

/// Root-owned directory with the exact mode, via `lstat`.
pub fn owned_dir(ctx: &crate::Ctx, path: &Path, mode: u32) -> Result<(), Error> {
    let meta = std::fs::symlink_metadata(path).map_err(Error::classify)?;
    if !meta.is_dir() || meta.uid() != ctx.priv_uid() || meta.gid() != ctx.priv_gid() {
        return fail(format!("unsafe factory directory: {}", file_name(path)));
    }
    if meta.mode() & 0o777 != mode {
        return fail(format!(
            "unsafe factory directory mode: {}",
            file_name(path)
        ));
    }
    Ok(())
}

/// Fixed layout plus the state lock file.
pub fn ensure_layout(ctx: &crate::Ctx) -> Result<(), Error> {
    if unsafe { libc::geteuid() } != 0 && ctx.test.is_none() {
        return fail("project-local root required");
    }
    mkdir_p(&ctx.factory, 0o755)?;
    mkdir_p(&ctx.preparations, 0o755)?;
    mkdir_p(&ctx.credentials, 0o755)?;
    owned_dir(ctx, &ctx.factory, 0o755)?;
    owned_dir(ctx, &ctx.preparations, 0o755)?;
    owned_dir(ctx, &ctx.credentials, 0o755)?;
    match OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o644)
        .open(&ctx.lock)
    {
        Ok(_) => Ok(()),
        Err(err) if err.kind() == std::io::ErrorKind::AlreadyExists => Ok(()),
        Err(err) => Err(Error::classify(err)),
    }
}

/// Validated preparation directory.
pub fn prep_dir(ctx: &crate::Ctx, pid: &str) -> Result<std::path::PathBuf, Error> {
    if !validate::is_id(pid) {
        return fail("unsupported preparation identity");
    }
    Ok(ctx.preparations.join(pid))
}

/// `(active, revision)` with the revision carried as normalized text.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Hold {
    pub active: bool,
    pub revision: String,
}

pub fn hold_json(hold: &Hold) -> JsonValue {
    obj(vec![
        ("active", JsonValue::Bool(hold.active)),
        ("revision", JsonValue::Number(hold.revision.clone())),
    ])
}

pub fn hold_state(ctx: &crate::Ctx) -> Result<Hold, Error> {
    let held = match fsx::read_json(ctx, &ctx.hold, 1024) {
        Ok(value) => value,
        Err(Error::Missing) => {
            return Ok(Hold {
                active: false,
                revision: "-1".to_string(),
            });
        }
        Err(err) => return Err(err),
    };
    if !validate::as_object(&held).is_some_and(|e| validate::key_set(e, &["revision"])) {
        return fail("unsafe maintenance hold");
    }
    match validate::as_int_text(held.get("revision").unwrap_or(&JsonValue::Null)) {
        Some(revision) => Ok(Hold {
            active: true,
            revision,
        }),
        None => fail("unsafe maintenance hold"),
    }
}

/// The hold and the stop tombstone both bar new preparation work.
pub fn refuse_barred(ctx: &crate::Ctx, directory: &Path) -> Result<(), Error> {
    if hold_state(ctx)?.active {
        return fail("maintenance hold denies preparation");
    }
    if fsx::lexists(&directory.join("stopped.json")) {
        return fail("preparation was stopped; use a new identity");
    }
    Ok(())
}

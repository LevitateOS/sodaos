//! The fixed CoreOS /home alias is trusted; application-controlled links are not.
//!
//! FJ-847: every privileged filesystem mutation in this module tree resolves
//! through an already-validated directory descriptor (`OwnedFd`) and the
//! `openat`/`mkdirat`/`fchownat`/`linkat`/`renameat`/`renameat2`/`unlinkat`
//! family with `O_NOFOLLOW` / `AT_SYMLINK_NOFOLLOW` semantics. A symlink
//! swapped into a path between a check and a use cannot redirect the
//! operation: the open fails closed (`ELOOP`) or the already-open descriptor
//! still points at the validated directory. Path strings survive only as
//! podman arguments and for directory scans, each rebound to its descriptor
//! by dev/ino comparison immediately before use.
//!
//! FJ-855: anchor acquisition itself is a descriptor walk, not a
//! check-then-open. `open_validated_dir` / `open_parent_dir` descend from
//! the trust root (`/`) one no-follow `openat` per component while holding
//! the parent descriptor — device+inode-bound at every level,
//! kernel-path-bound (`/proc/self/fd`) at the end — so the FJ-853 race
//! (ancestor `lstat`, then a full-path `open` whose `O_NOFOLLOW` covers
//! only the final component, plus an unbound `canonicalize`) is
//! structurally gone: there is no full-path open left to race. Record
//! opens are nonblocking (`O_NONBLOCK`), so a FIFO in a controlled
//! directory refuses instead of hanging the mutation gate.
use std::ffi::CString;
use std::fs;
use std::os::fd::{AsRawFd, FromRawFd, OwnedFd};
use std::os::unix::ffi::{OsStrExt, OsStringExt};
use std::os::unix::fs::MetadataExt;
use std::path::{Component, Path, PathBuf};

fn owned_directory(path: &Path, owner: u32) -> Result<(), String> {
    let meta = fs::symlink_metadata(path).map_err(|_| "application home is unavailable")?;
    if !meta.is_dir()
        || meta.file_type().is_symlink()
        || meta.uid() != owner
        || meta.mode() & 0o022 != 0
        || fs::canonicalize(path).map_err(|_| "application home is unavailable")? != path
    {
        return Err("application home must be an operator-owned canonical directory".into());
    }
    // FJ-847: re-validate the opened descriptor, not the path, so a swap
    // after this check cannot move later operations off the validated tree.
    let anchor = open_validated_dir(path, owner, 0o022, "application home is unavailable")
        .map_err(|_| {
            "application home must be an operator-owned canonical directory".to_string()
        })?;
    fd_matches_path(path, &anchor)
        .map_err(|_| "application home must be an operator-owned canonical directory".to_string())
}

fn resolve_at(path: &Path, home: &Path, coreos_home: &Path, owner: u32) -> Result<PathBuf, String> {
    if !path.is_absolute()
        || path
            .components()
            .any(|part| !matches!(part, Component::RootDir | Component::Normal(_)))
    {
        return Err("application storage requires a lexical path under /home".into());
    }
    let relative = path
        .strip_prefix(home)
        .map_err(|_| "application storage must remain under /home")?;
    let meta = fs::symlink_metadata(home).map_err(|_| "application home is unavailable")?;
    let canonical_home = fs::canonicalize(home).map_err(|_| "application home is unavailable")?;
    if meta.file_type().is_symlink() {
        let link = fs::read_link(home).map_err(|_| "application home alias is unavailable")?;
        if meta.uid() != owner
            || canonical_home != coreos_home
            || (link != Path::new("var/home") && link != coreos_home)
        {
            return Err("application home has an unsupported alias".into());
        }
        owned_directory(
            coreos_home
                .parent()
                .ok_or("application home has no parent")?,
            owner,
        )?;
        owned_directory(coreos_home, owner)?;
    } else {
        owned_directory(home, owner)?;
        if canonical_home != home {
            return Err("application home has an unsupported alias".into());
        }
    }
    let expected = canonical_home.join(relative);
    if fs::canonicalize(path).map_err(|_| "application storage is unavailable")? != expected {
        return Err("application storage links below /home are unsupported".into());
    }
    Ok(expected)
}

/// Single trusted seam for the appliance storage root. Production leaves
/// `SODA_APPLICATIONS_ROOT` unset and keeps the fixed operator-owned path;
/// the root-proof test redirects setup, runtime data, and snapshots together
/// through this one resolver, so a fixture can never split roots. Either way
/// the value passes the identical operator-owned `/home` tree checks below.
pub(super) fn storage_root() -> Result<PathBuf, String> {
    match std::env::var_os("SODA_APPLICATIONS_ROOT") {
        Some(dir) => home_path(Path::new(&dir)),
        None => home_path(Path::new("/home/soda-applications")),
    }
}

pub(super) fn home_path(path: &Path) -> Result<PathBuf, String> {
    let home = Path::new("/home");
    let canonical = Path::new("/var/home");
    if let Ok(relative) = path.strip_prefix(canonical) {
        // Internal callers retain the resolved path. Admit that result only
        // when resolving its lexical alias proves the same trusted CoreOS tree.
        let resolved = resolve_at(&home.join(relative), home, canonical, 0)?;
        if resolved != path {
            return Err("application storage has an unsupported home alias".into());
        }
        return Ok(resolved);
    }
    resolve_at(path, home, canonical, 0)
}

/// FJ-847 descriptor anchor primitives. Every privileged mutation in the
/// application tree goes through these: the caller validates the anchor
/// BEFORE creation or use, creates with a restrictive mode through the
/// anchor, and re-validates the opened descriptor — never the path.
pub(super) fn leaf(name: &str) -> Result<CString, String> {
    if name.is_empty() || name == "." || name == ".." || name.contains('/') || name.contains('\0') {
        return Err("anchored operation requires a single leaf name".into());
    }
    CString::new(name).map_err(|_| "anchored operation requires a single leaf name".into())
}

fn last_error_is_missing() -> bool {
    std::io::Error::last_os_error().raw_os_error() == Some(libc::ENOENT)
}

/// Read the kernel's path for an open descriptor. This is the descriptor's
/// own identity — never a re-resolution of a path string — so comparing it
/// binds the descriptor to the walked path even if the tree is renamed or
/// swapped immediately afterward (a moved tree reports a different path or
/// a ` (deleted)` suffix, both of which fail the bind).
fn fd_proc_path(fd: &OwnedFd, op: &str) -> Result<PathBuf, String> {
    std::fs::read_link(format!("/proc/self/fd/{}", fd.as_raw_fd())).map_err(|_| op.to_string())
}

/// Read exactly one symlink's target through its held parent descriptor,
/// never by resolving a path string.
fn readlink_at(parent: &OwnedFd, name: &str, op: &str) -> Result<std::ffi::OsString, String> {
    let leaf = leaf(name).map_err(|_| op.to_string())?;
    let mut buf = vec![0u8; 4096];
    let len = unsafe {
        libc::readlinkat(
            parent.as_raw_fd(),
            leaf.as_ptr(),
            buf.as_mut_ptr() as *mut libc::c_char,
            buf.len(),
        )
    };
    if len < 0 {
        return Err(op.into());
    }
    buf.truncate(len as usize);
    Ok(std::ffi::OsString::from_vec(buf))
}

/// Open the process trust root: `/` itself, no-follow. Every anchor walk
/// below starts here and descends one `openat` per component while holding
/// the parent descriptor, so no ancestor can be swapped between a check
/// and a use — there is no check-then-open of a full path anymore.
fn open_trust_root(op: &str) -> Result<OwnedFd, String> {
    let root = CString::new("/").map_err(|_| op.to_string())?;
    let fd = unsafe {
        libc::open(
            root.as_ptr(),
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
        )
    };
    if fd < 0 {
        return Err(op.into());
    }
    Ok(unsafe { OwnedFd::from_raw_fd(fd) })
}

/// Descend one component through its held parent without following a
/// symlink at any level (final or otherwise), then bind the new descriptor
/// to the walked prefix by device+inode: the `lstat` view of the prefix
/// must be the directory just opened. A swap of an already-passed ancestor
/// cannot move the held descriptor; a swap ahead of the walk either fails
/// the `openat` (`ELOOP`/`ENOTDIR`/`ENOENT`) or lands on bytes whose
/// device+inode (and, at the end, kernel path, owner, and mode) refuse.
fn descend_bound(
    parent: &OwnedFd,
    name: &std::ffi::OsStr,
    walked_prefix: &Path,
    op: &str,
) -> Result<OwnedFd, String> {
    let bytes = name.as_bytes();
    if bytes.is_empty() || bytes == b"." || bytes == b".." || bytes.contains(&b'/') {
        return Err(op.into());
    }
    let cname = CString::new(bytes).map_err(|_| op.to_string())?;
    let fd = unsafe {
        libc::openat(
            parent.as_raw_fd(),
            cname.as_ptr(),
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
        )
    };
    if fd < 0 {
        return Err(op.into());
    }
    let child = unsafe { OwnedFd::from_raw_fd(fd) };
    let stat = fstat_dir(&child, op)?;
    let meta = fs::symlink_metadata(walked_prefix).map_err(|_| op.to_string())?;
    if meta.dev() != stat.st_dev as u64 || meta.ino() != stat.st_ino as u64 {
        return Err(op.into());
    }
    Ok(child)
}

/// Absolute component list for the walk, starting at the trust root. The
/// single CoreOS `/home` alias is traversed explicitly: when the first
/// component names a symlink whose target is exactly the allowlisted
/// `var/home` (relative) or `/var/home` (absolute), the walk continues
/// through that fixed target. Any other symlink at that position — or a
/// symlink at any deeper position — fails closed at the no-follow
/// `openat`. A real (non-symlink) `/home` walks normally, so plain and
/// CoreOS layouts share this one path.
fn walk_components(path: &Path, op: &str) -> Result<Vec<std::ffi::OsString>, String> {
    let mut comps: Vec<std::ffi::OsString> = if path.is_absolute() {
        path.components()
            .filter_map(|part| match part {
                Component::RootDir => None,
                Component::Normal(name) => Some(name.to_owned()),
                _ => None,
            })
            .collect()
    } else {
        let cwd = std::fs::read_link("/proc/self/cwd").map_err(|_| op.to_string())?;
        if !cwd.is_absolute() {
            return Err(op.into());
        }
        let mut base: Vec<std::ffi::OsString> = cwd
            .components()
            .filter_map(|part| match part {
                Component::RootDir => None,
                Component::Normal(name) => Some(name.to_owned()),
                _ => None,
            })
            .collect();
        for part in path.components() {
            match part {
                Component::Normal(name) => base.push(name.to_owned()),
                _ => return Err(op.into()),
            }
        }
        base
    };
    // Reject non-normal shapes up front: the walk must consume exactly the
    // named tree, never its parent (absolute `..`, relative escapes) or a
    // device path. Relative `.`/`..`/prefix forms are refused here too.
    if path.is_absolute() {
        if path
            .components()
            .any(|part| !matches!(part, Component::RootDir | Component::Normal(_)))
        {
            return Err(op.into());
        }
    } else if path
        .components()
        .any(|part| !matches!(part, Component::Normal(_)))
    {
        return Err(op.into());
    }
    if comps
        .first()
        .is_some_and(|first| first.as_encoded_bytes() == b"home")
    {
        let root = open_trust_root(op)?;
        match readlink_at(&root, "home", op) {
            Ok(target) => {
                let bytes = target.as_encoded_bytes();
                if bytes == b"var/home" || bytes == b"/var/home" {
                    let mut resolved = vec![
                        std::ffi::OsString::from("var"),
                        std::ffi::OsString::from("home"),
                    ];
                    resolved.extend(comps.into_iter().skip(1));
                    comps = resolved;
                } else {
                    return Err(op.into());
                }
            }
            Err(_) => {
                // Not a symlink (real directory, or absent): the no-follow
                // descent below decides. Anything but the allowlisted alias
                // target refuses above.
                let meta = fs::symlink_metadata(Path::new("/home")).map_err(|_| op.to_string());
                if let Ok(meta) = meta {
                    if meta.file_type().is_symlink() {
                        return Err(op.into());
                    }
                }
            }
        }
    }
    Ok(comps)
}

/// FJ-855 descriptor-walk anchor: open `path` by descending from the trust
/// root one no-follow `openat` per component, device+inode-bound at every
/// level, carrying descriptors — never re-resolving path strings. Replaces
/// the ancestor-`lstat`-then-full-path-`open` (whose `O_NOFOLLOW` covered
/// only the final component) and the separate unbound `canonicalize`.
fn walk_from_root(path: &Path, op: &str) -> Result<(OwnedFd, PathBuf), String> {
    let comps = walk_components(path, op)?;
    let mut parent = open_trust_root(op)?;
    let mut walked = PathBuf::from("/");
    for comp in &comps {
        walked.push(comp);
        let child = descend_bound(&parent, comp, &walked, op)?;
        parent = child;
    }
    // Final kernel-path bind: the open descriptor's own path must be the
    // walked tree. A rename or swap across the walk fails here even when
    // device+inode still alias.
    let proc = fd_proc_path(&parent, op)?;
    let mut expected = PathBuf::from("/");
    for comp in &comps {
        expected.push(comp);
    }
    if proc != expected {
        return Err(op.into());
    }
    Ok((parent, expected))
}

/// Open a directory that needs no ownership assertion (a staging parent):
/// it must still be the real directory at the walked path — opened through
/// the descriptor walk, never by a full-path open — and bound to it.
pub(super) fn open_parent_dir(path: &Path, op: &str) -> Result<OwnedFd, String> {
    let (fd, expected) = walk_from_root(path, op).map_err(|_| op.to_string())?;
    fstat_dir(&fd, op)?;
    fd_matches_path(&expected, &fd).map_err(|_| op.to_string())?;
    Ok(fd)
}

fn fstat_dir(fd: &OwnedFd, op: &str) -> Result<libc::stat, String> {
    let mut stat: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstat(fd.as_raw_fd(), &mut stat) } != 0 {
        return Err(op.into());
    }
    if (stat.st_mode as u32 & libc::S_IFMT) != libc::S_IFDIR {
        return Err(op.into());
    }
    Ok(stat)
}

/// Guard the anchor BEFORE creation or use: the path is opened by the
/// descriptor walk from the trust root (one no-follow `openat` per
/// component, device+inode-bound at every level, kernel-path-bound at the
/// end), and the walked descriptor must be a directory with the expected
/// owner and no forbidden mode bits. There is no ancestor-`lstat` followed
/// by a full-path open, and no separate unbound `canonicalize`: a symlink
/// swapped in anywhere above the final component fails the walk (`ELOOP` /
/// `ENOTDIR` / `ENOENT`), the per-level bind, or the kernel-path bind.
/// Every later privileged step uses the returned descriptor, so a swap
/// after this point cannot redirect it.
pub(super) fn open_validated_dir(
    path: &Path,
    owner: u32,
    forbid: u32,
    op: &str,
) -> Result<OwnedFd, String> {
    let (fd, expected) = walk_from_root(path, op).map_err(|_| op.to_string())?;
    let stat = fstat_dir(&fd, op)?;
    if stat.st_uid != owner || (stat.st_mode as u32 & forbid) != 0 {
        return Err(op.into());
    }
    fd_matches_path(&expected, &fd).map_err(|_| op.to_string())?;
    Ok(fd)
}

/// Open one child directory through its validated parent. `Ok(None)` is a
/// clean absent child (the creation path); anything else that fails refuses.
pub(super) fn open_child_dir_opt(
    parent: &OwnedFd,
    name: &str,
    op: &str,
) -> Result<Option<OwnedFd>, String> {
    let leaf = leaf(name).map_err(|_| op.to_string())?;
    let fd = unsafe {
        libc::openat(
            parent.as_raw_fd(),
            leaf.as_ptr(),
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
        )
    };
    if fd < 0 {
        if last_error_is_missing() {
            return Ok(None);
        }
        return Err(op.into());
    }
    Ok(Some(unsafe { OwnedFd::from_raw_fd(fd) }))
}

/// Create a private child directory through its validated parent, or
/// re-validate an existing one. An existing directory is NEVER chowned:
/// only a directory this call just created is assigned its owner, then
/// opened and re-validated as a descriptor.
pub(super) fn make_child_dir(
    parent: &OwnedFd,
    name: &str,
    owner: u32,
    op: &str,
) -> Result<OwnedFd, String> {
    let leaf = leaf(name).map_err(|_| op.to_string())?;
    if let Some(fd) = open_child_dir_opt(parent, name, op)? {
        let stat = fstat_dir(&fd, op)?;
        if stat.st_uid != owner || (stat.st_mode as u32 & 0o077) != 0 {
            return Err(op.into());
        }
        return Ok(fd);
    }
    let created = unsafe { libc::mkdirat(parent.as_raw_fd(), leaf.as_ptr(), 0o700) };
    if created != 0 {
        let error = std::io::Error::last_os_error();
        if error.raw_os_error() == Some(libc::EEXIST) {
            // A racing creator won: re-validate its result, never chown it.
            let fd = open_child_dir_opt(parent, name, op)?.ok_or_else(|| op.to_string())?;
            let stat = fstat_dir(&fd, op)?;
            if stat.st_uid != owner || (stat.st_mode as u32 & 0o077) != 0 {
                return Err(op.into());
            }
            return Ok(fd);
        }
        return Err(op.into());
    }
    // The directory is ours (mkdirat never creates a symlink): assign the
    // owner without following, then validate the opened descriptor.
    if unsafe {
        libc::fchownat(
            parent.as_raw_fd(),
            leaf.as_ptr(),
            owner,
            owner,
            libc::AT_SYMLINK_NOFOLLOW,
        )
    } != 0
    {
        let _ = unlink_child(parent, name, op);
        return Err(op.into());
    }
    let fd = open_child_dir_opt(parent, name, op)?.ok_or_else(|| op.to_string())?;
    if unsafe { libc::fchmod(fd.as_raw_fd(), 0o700) } != 0 {
        return Err(op.into());
    }
    let stat = fstat_dir(&fd, op)?;
    if stat.st_uid != owner || (stat.st_mode as u32 & 0o077) != 0 {
        return Err(op.into());
    }
    fsync_fd(&fd, op)?;
    Ok(fd)
}

/// Open one child file through its validated parent without following a
/// final-component symlink. `Ok(None)` is a clean absent file.
///
/// FJ-855: record opens are nonblocking. A FIFO planted in a controlled
/// directory (the runtime account owns its control dir) must refuse at the
/// caller's `fstat` type check — never hang the mutation gate inside `open`
/// before any deadline or type check can run. `O_NONBLOCK` is a no-op for
/// the regular files every legitimate record is.
pub(super) fn open_child_opt(
    parent: &OwnedFd,
    name: &str,
    write: bool,
    op: &str,
) -> Result<Option<fs::File>, String> {
    let leaf = leaf(name).map_err(|_| op.to_string())?;
    let flags = libc::O_NOFOLLOW
        | libc::O_CLOEXEC
        | libc::O_NONBLOCK
        | if write { libc::O_RDWR } else { libc::O_RDONLY };
    let fd = unsafe { libc::openat(parent.as_raw_fd(), leaf.as_ptr(), flags) };
    if fd < 0 {
        if last_error_is_missing() {
            return Ok(None);
        }
        return Err(op.into());
    }
    Ok(Some(unsafe { fs::File::from_raw_fd(fd) }))
}

/// Open an exclusively-created staging file through the validated parent
/// and assign its owner on the open descriptor — never by path. The caller
/// writes the bytes, publishes with `publish_staged_*`, and removes the
/// staging name on every failure path.
pub(super) fn open_staged_file(
    parent: &OwnedFd,
    name: &str,
    owner: u32,
    op_open: &str,
    op_own: &str,
) -> Result<fs::File, String> {
    let leaf = leaf(name).map_err(|_| op_open.to_string())?;
    let fd = unsafe {
        libc::openat(
            parent.as_raw_fd(),
            leaf.as_ptr(),
            libc::O_WRONLY | libc::O_CREAT | libc::O_EXCL | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            0o600,
        )
    };
    if fd < 0 {
        return Err(op_open.into());
    }
    if unsafe { libc::fchown(fd, owner, owner) } != 0 {
        unsafe { libc::close(fd) };
        let _ = unlink_child(parent, name, op_open);
        return Err(op_own.into());
    }
    Ok(unsafe { fs::File::from_raw_fd(fd) })
}

/// Write the full buffer to an open staged file and sync it. The staging
/// name is removed when the write or sync fails.
pub(super) fn write_full_and_sync(
    parent: &OwnedFd,
    name: &str,
    file: &fs::File,
    bytes: &[u8],
    op_write: &str,
    op_sync: &str,
) -> Result<(), String> {
    use std::io::Write;
    if file
        .try_clone()
        .and_then(|mut owned| owned.write_all(bytes))
        .is_err()
    {
        let _ = unlink_child(parent, name, op_write);
        return Err(op_write.into());
    }
    if file.sync_all().is_err() {
        let _ = unlink_child(parent, name, op_sync);
        return Err(op_sync.into());
    }
    Ok(())
}

/// Overwriting publication for the retained reservation journal: both names
/// resolve through the same validated parent descriptor.
pub(super) fn publish_staged_replace(
    parent: &OwnedFd,
    tmp: &str,
    dst: &str,
    op_publish: &str,
    op_sync: &str,
) -> Result<(), String> {
    let from = leaf(tmp).map_err(|_| op_publish.to_string())?;
    let to = leaf(dst).map_err(|_| op_publish.to_string())?;
    if unsafe {
        libc::renameat(
            parent.as_raw_fd(),
            from.as_ptr(),
            parent.as_raw_fd(),
            to.as_ptr(),
        )
    } != 0
    {
        let _ = unlink_child(parent, tmp, op_publish);
        return Err(op_publish.into());
    }
    fsync_fd(parent, op_sync)
}

/// Non-overwriting publication for first-time admission: refuses when the
/// target already exists, preserving a single link on early exit.
pub(super) fn publish_staged_noreplace(
    parent: &OwnedFd,
    tmp: &str,
    dst: &str,
    op_publish: &str,
    op_sync: &str,
) -> Result<(), String> {
    let from = leaf(tmp).map_err(|_| op_publish.to_string())?;
    let to = leaf(dst).map_err(|_| op_publish.to_string())?;
    if unsafe {
        libc::renameat2(
            parent.as_raw_fd(),
            from.as_ptr(),
            parent.as_raw_fd(),
            to.as_ptr(),
            libc::RENAME_NOREPLACE,
        )
    } != 0
    {
        let _ = unlink_child(parent, tmp, op_publish);
        return Err(op_publish.into());
    }
    fsync_fd(parent, op_sync)
}

/// Atomic hard-link publication for ingress routes: the staging name never
/// matches the Caddy import glob, so a partial route is never served.
pub(super) fn link_staged(
    parent: &OwnedFd,
    tmp: &str,
    dst: &str,
    op_link: &str,
    op_sync: &str,
) -> Result<(), String> {
    let from = leaf(tmp).map_err(|_| op_link.to_string())?;
    let to = leaf(dst).map_err(|_| op_link.to_string())?;
    if unsafe {
        libc::linkat(
            parent.as_raw_fd(),
            from.as_ptr(),
            parent.as_raw_fd(),
            to.as_ptr(),
            0,
        )
    } != 0
    {
        let _ = unlink_child(parent, tmp, op_link);
        return Err(op_link.into());
    }
    let _ = unlink_child(parent, tmp, op_link);
    fsync_fd(parent, op_sync)
}

/// Guarded removal: the name resolves through the trusted descriptor, never
/// a bare path. Refuses (rather than acting elsewhere) when the name is
/// absent, a symlink, or unreachable.
pub(super) fn unlink_child(parent: &OwnedFd, name: &str, op: &str) -> Result<(), String> {
    let leaf = leaf(name).map_err(|_| op.to_string())?;
    if unsafe { libc::unlinkat(parent.as_raw_fd(), leaf.as_ptr(), 0) } != 0 {
        return Err(op.into());
    }
    let _ = fsync_fd(parent, op);
    Ok(())
}

/// Assert owner and mode on an already-open directory descriptor.
pub(super) fn validate_open_dir(
    fd: &OwnedFd,
    owner: u32,
    forbid: u32,
    op: &str,
) -> Result<(), String> {
    let stat = fstat_dir(fd, op)?;
    if stat.st_uid != owner || (stat.st_mode as u32 & forbid) != 0 {
        return Err(op.into());
    }
    Ok(())
}

/// Open one existing child directory through its validated parent and assert
/// owner and mode on the opened descriptor. A missing child refuses — the
/// caller creates it with `make_child_dir`, never by upgrading this check
/// into a privileged mutation.
pub(super) fn validate_child_dir(
    parent: &OwnedFd,
    name: &str,
    owner: u32,
    forbid: u32,
    op: &str,
) -> Result<OwnedFd, String> {
    let fd = open_child_dir_opt(parent, name, op)?.ok_or_else(|| op.to_string())?;
    let stat = fstat_dir(&fd, op)?;
    if stat.st_uid != owner || (stat.st_mode as u32 & forbid) != 0 {
        return Err(op.into());
    }
    Ok(fd)
}

pub(super) fn fsync_fd(fd: &OwnedFd, op: &str) -> Result<(), String> {
    if unsafe { libc::fsync(fd.as_raw_fd()) } != 0 {
        return Err(op.into());
    }
    Ok(())
}

/// `fstat` view of an already-open file for descriptor-based validation.
/// Callers assert owner, mode, link count, and length on this value instead
/// of re-reading the path.
pub(super) fn stat_opened(file: &fs::File, op: &str) -> Result<libc::stat, String> {
    let mut stat: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstat(file.as_raw_fd(), &mut stat) } != 0 {
        return Err(op.into());
    }
    Ok(stat)
}

/// Rebind a path string to its validated descriptor by device+inode: proves
/// the string the caller must hand on (podman arguments, directory scans)
/// still names the anchored directory at the moment of use.
pub(super) fn fd_matches_path(path: &Path, fd: &OwnedFd) -> Result<(), String> {
    let meta = fs::symlink_metadata(path).map_err(|_| "anchored path left its descriptor")?;
    let stat = fstat_dir(fd, "anchored path left its descriptor")?;
    if meta.dev() != stat.st_dev || meta.ino() != stat.st_ino {
        return Err("anchored path left its descriptor".into());
    }
    Ok(())
}

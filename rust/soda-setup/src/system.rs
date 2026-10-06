use std::io;
use std::os::unix::ffi::OsStrExt;
use std::path::Path;
use std::time::Duration;

// postgresSecretDir holds the PostgreSQL credential files setup generates:
// super/forgejo/soda .passwd (0600) plus the soda connection URL (0640,
// group soda). The database units stay skipped until these exist, so live
// media shows a clean skip instead of a failed database.
pub(crate) const POSTGRES_SECRET_DIR: &str = "/etc/soda/postgres";
// Forgejo reads its database password over a root-owned read-only mount as
// the image user (USER_UID/USER_GID 1000 in forgejo.container); the source
// file must carry that ownership because the entrypoint drops supplementary
// groups, so group readability alone does not admit it.
pub(crate) const FORGEJO_DB_USER: u32 = 1000;
// The soda DSN is read by the dashboard (uid/gid 2000) and the identity
// broker (gid 2000). Group soda (2000, system/host/config/soda.sysusers)
// admits exactly those service identities.
pub(crate) const SODA_SERVICE_GROUP: u32 = 2000;
// postgresSocketDir is the host path of the PostgreSQL unix-socket
// directory shared with host-network and native clients (soda.tmpfiles);
// it doubles as the DSN host so no database TCP reaches the host.
pub(crate) const POSTGRES_SOCKET_DIR: &str = "/run/soda/postgres";
pub(crate) const FORGEJO_TIMEOUT: Duration = Duration::from_secs(30);
pub(crate) const FORGEJO_RESPONSE_LIMIT: usize = 2 * 1024 * 1024;

// Minimal libc surface (chown/geteuid/errno) declared directly so this crate
// stays dependency-free and offline-buildable; the full libc crate is not
// vendored in this tree.
#[link(name = "c")]
extern "C" {
    fn geteuid() -> u32;
    fn chown(path: *const std::os::raw::c_char, owner: u32, group: u32) -> std::os::raw::c_int;
    #[cfg(target_os = "linux")]
    fn __errno_location() -> *mut std::os::raw::c_int;
}

pub(crate) fn euid() -> u32 {
    unsafe { geteuid() }
}

pub(crate) fn chown_path(path: &Path, owner: u32, group: u32) -> io::Result<()> {
    use std::os::raw::c_char;
    let bytes = path.as_os_str().as_bytes();
    let mut nul = Vec::with_capacity(bytes.len() + 1);
    nul.extend_from_slice(bytes);
    nul.push(0);
    let ret = unsafe { chown(nul.as_ptr() as *const c_char, owner, group) };
    if ret == 0 {
        Ok(())
    } else {
        Err(io::Error::new(
            io::ErrorKind::PermissionDenied,
            format!("chown {}: {}", path.display(), errno_message()),
        ))
    }
}

fn errno_message() -> String {
    #[cfg(target_os = "linux")]
    {
        let code = unsafe { *__errno_location() };
        io::Error::from_raw_os_error(code).to_string()
    }
    #[cfg(not(target_os = "linux"))]
    {
        "operation failed".to_string()
    }
}

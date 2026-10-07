use std::io;
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

pub(crate) fn euid() -> u32 {
    unsafe { libc::geteuid() }
}

pub(crate) fn chown_path(path: &Path, owner: u32, group: u32) -> io::Result<()> {
    use std::os::unix::fs::chown;
    chown(path, Some(owner), Some(group))
}

//! Enrollment arming: private-address selection, the armed window state,
//! host fingerprint inspection, and the no-replacement guards.

// Key enrollment arms from any interactive root terminal, local or remote:
// an operator who already holds root can administer keys directly, so the
// terminal type was never a security boundary, only an inconvenience.
// Explicit typed intent and the expiring password-only window remain.

use std::ffi::CString;
use std::os::unix::io::{AsRawFd, FromRawFd};
use std::time::Duration;

use crate::command::Runner;
use crate::console::Console;
use crate::errors::{self, Error};
use crate::signal::Ctx;

use super::keys::{enrollment_root_home, enrollment_safe_directory};
use super::{
    enrollment_private_address, ENROLLMENT_DIR, ENROLLMENT_PORT, ENROLLMENT_SOCKET_UNIT,
    ENROLLMENT_TEMPLATE_UNIT, ENROLLMENT_UNIT,
};

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct EnrollmentAddress {
    pub name: String,
    pub ip: String,
}

pub fn enrollment_addresses() -> Result<Vec<EnrollmentAddress>, Error> {
    let mut addrs: *mut libc::ifaddrs = std::ptr::null_mut();
    if unsafe { libc::getifaddrs(&mut addrs) } != 0 {
        return Err(errors::os_error(std::io::Error::last_os_error()));
    }
    let mut result = Vec::new();
    let mut current = addrs;
    while !current.is_null() {
        let ifa = unsafe { &*current };
        let flags = ifa.ifa_flags as libc::c_int;
        // Go lists masked addresses (the local endpoint for point-to-point)
        // and keeps only the ones parsing as private IPv4 below.
        if flags & libc::IFF_UP != 0 && flags & libc::IFF_LOOPBACK == 0 && !ifa.ifa_addr.is_null() {
            let addr = unsafe { &*ifa.ifa_addr };
            if addr.sa_family as libc::c_int == libc::AF_INET {
                let sin = unsafe { &*(ifa.ifa_addr as *const libc::sockaddr_in) };
                let ip = std::net::Ipv4Addr::from(u32::from_be(sin.sin_addr.s_addr)).to_string();
                if enrollment_private_address(&ip).is_ok() {
                    let name = unsafe { std::ffi::CStr::from_ptr(ifa.ifa_name) }
                        .to_string_lossy()
                        .into_owned();
                    result.push(EnrollmentAddress { name, ip });
                }
            }
        }
        current = unsafe { (*current).ifa_next };
    }
    unsafe { libc::freeifaddrs(addrs) };
    Ok(result)
}

pub(crate) fn enrollment_binding_live(
    selected: &EnrollmentAddress,
    current: &[EnrollmentAddress],
) -> bool {
    current.iter().any(|candidate| candidate == selected)
}

pub fn enrollment_live_address(selected: &EnrollmentAddress) -> bool {
    match enrollment_addresses() {
        Ok(addresses) => enrollment_binding_live(selected, &addresses),
        Err(_) => false,
    }
}

pub fn enrollment_boot_seconds() -> Result<i64, Error> {
    let mut now: libc::timespec = unsafe { std::mem::zeroed() };
    if unsafe { libc::clock_gettime(libc::CLOCK_BOOTTIME, &mut now) } != 0 {
        return Err(errors::os_error(std::io::Error::last_os_error()));
    }
    Ok(now.tv_sec)
}

struct EnrollmentTemp {
    directory: std::fs::File,
    name: std::ffi::CString,
    path: String,
    file: Option<std::fs::File>,
    identity: Option<(u64, u64)>,
}

impl EnrollmentTemp {
    fn opened_identity(&self) -> Result<(u64, u64), Error> {
        use std::os::unix::fs::MetadataExt;
        let meta = match self.file.as_ref() {
            Some(file) => file.metadata(),
            None => return self.identity.ok_or(Error::EnrollUncertain),
        }
        .map_err(|_| Error::EnrollUncertain)?;
        Ok((meta.dev(), meta.ino()))
    }

    fn named_stat(&self, name: &std::ffi::CStr) -> std::io::Result<libc::stat> {
        let mut stat: libc::stat = unsafe { std::mem::zeroed() };
        if unsafe {
            libc::fstatat(
                self.directory.as_raw_fd(),
                name.as_ptr(),
                &mut stat,
                libc::AT_SYMLINK_NOFOLLOW,
            )
        } != 0
        {
            return Err(std::io::Error::last_os_error());
        }
        Ok(stat)
    }

    fn validate_temp(&self, expected: usize) -> Result<(), Error> {
        let (device, inode) = self.opened_identity()?;
        let named = self
            .named_stat(&self.name)
            .map_err(|_| Error::EnrollUncertain)?;
        if named.st_dev != device
            || named.st_ino != inode
            || named.st_nlink != 1
            || named.st_mode & libc::S_IFMT != libc::S_IFREG
            || named.st_uid != unsafe { libc::geteuid() }
            || named.st_mode & 0o077 != 0
            || named.st_size != expected as i64
        {
            return Err(Error::EnrollUncertain);
        }
        Ok(())
    }

    fn sync(&self) -> Result<(), Error> {
        self.file
            .as_ref()
            .ok_or(Error::EnrollUncertain)?
            .sync_all()
            .map_err(|err| errors::path_error("sync", &self.path, err))
    }

    fn close(&mut self) -> Result<(), Error> {
        use std::os::unix::fs::MetadataExt;
        use std::os::unix::io::IntoRawFd;
        let Some(file) = self.file.as_ref() else {
            return Ok(());
        };
        let meta = file.metadata().map_err(|_| Error::EnrollUncertain)?;
        self.identity = Some((meta.dev(), meta.ino()));
        let fd = self.file.take().unwrap().into_raw_fd();
        if unsafe { libc::close(fd) } != 0 {
            let errno = unsafe { *libc::__errno_location() };
            return Err(errors::path_error(
                "close",
                &self.path,
                std::io::Error::from_raw_os_error(errno),
            ));
        }
        Ok(())
    }

    fn remove_owned(&mut self) -> Result<(), Error> {
        let (device, inode) = self.opened_identity()?;
        match self.named_stat(&self.name) {
            Ok(named) if named.st_dev == device && named.st_ino == inode => {}
            Err(err) if err.kind() == std::io::ErrorKind::NotFound => return Ok(()),
            Err(err) => return Err(errors::path_error("stat", &self.path, err)),
            _ => return Err(Error::EnrollUncertain),
        }
        if unsafe { libc::unlinkat(self.directory.as_raw_fd(), self.name.as_ptr(), 0) } != 0 {
            let errno = unsafe { *libc::__errno_location() };
            if errno != libc::ENOENT {
                return Err(errors::path_error(
                    "remove",
                    &self.path,
                    std::io::Error::from_raw_os_error(errno),
                ));
            }
        }
        Ok(())
    }

    fn publish(
        &mut self,
        destination: &str,
        destination_path: &str,
        expected: usize,
    ) -> Result<(), Error> {
        use std::os::fd::AsRawFd;
        self.validate_temp(expected)?;
        let destination_name = std::ffi::CString::new(destination)
            .map_err(|_| Error::msg("invalid enrollment destination"))?;
        if unsafe {
            libc::linkat(
                self.directory.as_raw_fd(),
                self.name.as_ptr(),
                self.directory.as_raw_fd(),
                destination_name.as_ptr(),
                0,
            )
        } != 0
        {
            let errno = unsafe { *libc::__errno_location() };
            return Err(errors::link_error(
                "link",
                &self.path,
                destination_path,
                std::io::Error::from_raw_os_error(errno),
            ));
        }
        let published = self
            .named_stat(&destination_name)
            .map_err(|_| Error::EnrollUncertain)?;
        let (device, inode) = self.opened_identity()?;
        if published.st_dev != device || published.st_ino != inode {
            return Err(Error::EnrollUncertain);
        }
        self.remove_owned()?;
        if unsafe { libc::fsync(self.directory.as_raw_fd()) } != 0 {
            return Err(Error::EnrollUncertain);
        }
        Ok(())
    }
}

impl Drop for EnrollmentTemp {
    fn drop(&mut self) {
        let _ = self.remove_owned();
    }
}

fn create_enrollment_temp(directory: &str, name: &str) -> Result<EnrollmentTemp, Error> {
    use std::os::fd::{AsRawFd, FromRawFd};
    let directory_file =
        std::fs::File::open(directory).map_err(|err| errors::path_error("open", directory, err))?;
    for _ in 0..100 {
        let leaf = format!(".{name}-{}", super::keys::random_hex(8)?);
        let path = format!("{directory}/{leaf}");
        let c_leaf = std::ffi::CString::new(leaf.as_bytes())
            .map_err(|_| errors::os_error(std::io::Error::from_raw_os_error(libc::EINVAL)))?;
        let fd = unsafe {
            libc::openat(
                directory_file.as_raw_fd(),
                c_leaf.as_ptr(),
                libc::O_RDWR | libc::O_CREAT | libc::O_EXCL | libc::O_NOFOLLOW | libc::O_CLOEXEC,
                0o600,
            )
        };
        if fd >= 0 {
            return Ok(EnrollmentTemp {
                directory: directory_file,
                name: c_leaf,
                path,
                file: Some(unsafe { std::fs::File::from_raw_fd(fd) }),
                identity: None,
            });
        }
        let err = std::io::Error::last_os_error();
        if err.kind() != std::io::ErrorKind::AlreadyExists {
            return Err(errors::path_error("open", &path, err));
        }
    }
    Err(Error::msg("cannot reserve enrollment temporary file"))
}

/// O07-F1: maps one publication write outcome to the error (if any) that
/// must block publication. Callers still close exactly once, honor a write
/// error before any close error, and remove only this call's temporary path
/// on every error path; there are no retries.
pub(crate) fn enrollment_write_error(
    name: &str,
    temp_path: &str,
    expected: usize,
    written: std::io::Result<usize>,
) -> Option<Error> {
    match written {
        Ok(count) if count == expected => None,
        Ok(count) => Some(Error::msg(format!(
            "short write publishing enrollment {name}: wrote {count} of {expected} bytes"
        ))),
        Err(e) => Some(errors::path_error("write", temp_path, e)),
    }
}

pub fn enrollment_write(directory: &str, name: &str, value: &str) -> Result<(), Error> {
    use std::io::Write;
    let mut temp = create_enrollment_temp(directory, name)?;
    // Keep one write per receipt; the owned descriptor and dirfd-relative name
    // remain in custody through sync, close, and no-replacement publication.
    let written = temp
        .file
        .as_mut()
        .ok_or(Error::EnrollUncertain)?
        .write(value.as_bytes());
    let write_err = enrollment_write_error(name, &temp.path, value.len(), written);
    let sync_err = if write_err.is_none() {
        temp.sync().err()
    } else {
        None
    };
    let close_err = temp.close().err();
    if let Some(err) = write_err.or(sync_err).or(close_err) {
        return match temp.remove_owned() {
            Ok(()) => Err(err),
            Err(cleanup) => Err(Error::msg(format!(
                "{err}; temporary cleanup failed: {cleanup}"
            ))),
        };
    }
    let destination = format!("{directory}/{name}");
    match temp.publish(name, &destination, value.len()) {
        Ok(()) => Ok(()),
        Err(primary) => match temp.remove_owned() {
            Ok(()) => Err(primary),
            Err(cleanup) => Err(Error::msg(format!(
                "{primary}; temporary cleanup failed: {cleanup}"
            ))),
        },
    }
}

fn enrollment_window_remaining(until: i64, now: Result<i64, Error>) -> Result<Duration, Error> {
    let now = match now {
        Ok(now) => now,
        Err(_) => return Err(Error::msg("enrollment window expired")),
    };
    if until <= now || until - now > 300 {
        return Err(Error::msg("enrollment window expired"));
    }
    Ok(Duration::from_secs((until - now) as u64))
}

fn parse_armed_enrollment(data: &[u8]) -> Result<(EnrollmentAddress, i64), Error> {
    let text = String::from_utf8_lossy(data);
    let fields: Vec<&str> = text.split_whitespace().collect();
    if fields.len() != 3 {
        return Err(Error::msg("invalid enrollment arm state"));
    }
    let selected = EnrollmentAddress {
        name: fields[0].to_string(),
        ip: fields[1].to_string(),
    };
    enrollment_private_address(&selected.ip)?;
    match fields[2].parse::<i64>() {
        Ok(until) => Ok((selected, until)),
        Err(_) => Err(Error::msg("enrollment window expired")),
    }
}

pub fn enrollment_state() -> Result<(EnrollmentAddress, Duration), Error> {
    let dir = open_enrollment_dir()?;
    enrollment_safe_directory(dir.as_raw_fd(), 0)?;
    let data = crate::execute::read_regular(&format!("{ENROLLMENT_DIR}/armed"), 512)?;
    let (selected, until) = parse_armed_enrollment(&data)?;
    let remaining = enrollment_window_remaining(until, enrollment_boot_seconds())?;
    Ok((selected, remaining))
}

/// Stored window state plus live revalidation of the selected interface
/// address. Ongoing lifecycle checks use this: the window aborts when the
/// selected interface address changes.
pub fn enrollment_live_state() -> Result<(EnrollmentAddress, Duration), Error> {
    let (selected, remaining) = enrollment_state()?;
    if !enrollment_binding_live(&selected, &enrollment_addresses().unwrap_or_default()) {
        return Err(Error::msg(
            "selected interface address changed; closing enrollment window",
        ));
    }
    Ok((selected, remaining))
}

fn open_enrollment_dir() -> Result<std::os::unix::io::OwnedFd, Error> {
    let path = CString::new(ENROLLMENT_DIR).map_err(|_| Error::msg("invalid name"))?;
    let fd = unsafe {
        libc::open(
            path.as_ptr(),
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            0,
        )
    };
    if fd < 0 {
        return Err(errors::os_error(std::io::Error::last_os_error()));
    }
    Ok(unsafe { std::os::unix::io::OwnedFd::from_raw_fd(fd) })
}

pub(crate) fn parse_enrollment_choice(value: &str, n: usize) -> Result<usize, Error> {
    if value.eq_ignore_ascii_case("back") || value.eq_ignore_ascii_case("cancel") {
        return Err(Error::msg("key enrollment cancelled"));
    }
    match value.parse::<i64>() {
        Ok(index) if index >= 1 && (index as usize) <= n => Ok(index as usize),
        _ => Err(Error::msg("enter a listed enrollment address")),
    }
}

fn is_choice_hint(err: &Error) -> bool {
    matches!(err, Error::Msg(text) if text == "enter a listed enrollment address")
}

pub(crate) fn select_enrollment_address(
    ctx: &Ctx,
    console: &Console,
    addresses: &[EnrollmentAddress],
) -> Result<EnrollmentAddress, Error> {
    if addresses.is_empty() {
        return Err(Error::msg(
            "no private address available for key enrollment",
        ));
    }
    for (i, address) in addresses.iter().enumerate() {
        console.print(
            format_args!("{}. {:?}: {}", i + 1, address.name, address.ip),
        );
    }
    loop {
        let value = console.ask(
            ctx,
            "Private interface/address number (back or cancel exits)",
        )?;
        match parse_enrollment_choice(&value, addresses.len()) {
            Ok(index) => return Ok(addresses[index - 1].clone()),
            Err(err) if is_choice_hint(&err) => {
                console.print(
                    format_args!("Enter a number from 1 to {}, or back/cancel.", addresses.len()),
                );
            }
            Err(err) => return Err(err),
        }
    }
}

fn inspect_host_fingerprint(ctx: &Ctx, run: &dyn Runner) -> Result<String, Error> {
    // Derive the public host fingerprint from the actual existing private host
    // key via native ssh-keygen; do not trust a possibly stale .pub companion.
    let public = run
        .run(
            ctx,
            "/usr/bin/ssh-keygen",
            &[
                "-y".to_string(),
                "-f".to_string(),
                "/etc/ssh/ssh_host_ed25519_key".to_string(),
            ],
            None,
        )
        .map_err(|_| {
            Error::msg("native Ed25519 SSH host key unavailable; no enrollment window opened")
        })?;
    let parsed = crate::sshkey::parse_authorized_key_bytes(&public)
        .map_err(|_| Error::msg("cannot inspect native SSH host identity"))?;
    if parsed.key.key_type() != "ssh-ed25519" {
        return Err(Error::msg("cannot inspect native SSH host identity"));
    }
    Ok(crate::sshkey::fingerprint_sha256_wire(
        &parsed.key.marshal(),
    ))
}

fn confirm_enrollment_intent(
    ctx: &Ctx,
    console: &Console,
    selected: &EnrollmentAddress,
    fingerprint: &str,
) -> Result<(), Error> {
    console.print(format_args!("Native host fingerprint: {fingerprint}"));
    console.print(
        format_args!(
            "A temporary password-only key-import connection will listen on {}:{} for at most five minutes. It uses OpenSSH's native password verification; additional PAM policies are not inherited. Ordinary SSH policy is unchanged. Press Enter or Ctrl-C to close it early.",
            selected.ip,
            ENROLLMENT_PORT
        ),
    );
    let answer = console.ask(ctx, "Type ARM KEY IMPORT to open this window")?;
    if answer != "ARM KEY IMPORT" {
        return Err(Error::msg("key enrollment cancelled"));
    }
    if !enrollment_live_address(selected) {
        return Err(Error::msg(
            "selected interface address changed; no window opened",
        ));
    }
    Ok(())
}

#[cfg(test)]
mod temp_owner_tests {
    use super::create_enrollment_temp;

    #[test]
    fn cleanup_preserves_a_replaced_temporary_name() {
        let directory =
            std::env::temp_dir().join(format!("soda-enrollment-owner-{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&directory);
        std::fs::create_dir(&directory).unwrap();
        let mut temp = create_enrollment_temp(directory.to_str().unwrap(), "receipt").unwrap();
        std::fs::remove_file(&temp.path).unwrap();
        std::fs::write(&temp.path, b"replacement").unwrap();
        assert!(temp.remove_owned().is_err());
        assert_eq!(std::fs::read(&temp.path).unwrap(), b"replacement");
        drop(temp);
        std::fs::remove_dir_all(directory).unwrap();
    }
}

pub fn select_enrollment_target(
    ctx: &Ctx,
    console: &Console,
    run: &dyn Runner,
) -> Result<EnrollmentAddress, Error> {
    enrollment_root_home()?;
    let addresses = match enrollment_addresses() {
        Ok(addresses) if !addresses.is_empty() => addresses,
        _ => {
            return Err(Error::msg(
                "no active private IPv4 address: configure the host network and an actual laptop route before key enrollment",
            ))
        }
    };
    console.print("Import one laptop public key using the native root password. Choose the host address your laptop can actually reach; an interface address alone does not prove a client route.");
    let selected = select_enrollment_address(ctx, console, &addresses)?;
    let fingerprint = inspect_host_fingerprint(ctx, run)?;
    confirm_enrollment_intent(ctx, console, &selected, &fingerprint)?;
    if let Some(err) = ctx.err() {
        return Err(err);
    }
    Ok(selected)
}

// systemdUnitFileNoMatch reports whether a list-unit-files failure is the
// no-match exit (systemd 259 exits 1 with empty output when the pattern
// matches no unit file) rather than an interrupted or otherwise failed
// inspection. Publication still refuses to replace an existing file.
fn systemd_unit_file_no_match(err: &Error) -> bool {
    matches!(
        err,
        Error::CmdExit {
            code: 1,
            interrupted: false,
            ..
        }
    )
}

pub fn guard_existing_enrollment_state(ctx: &Ctx, run: &dyn Runner) -> Result<(), Error> {
    // An existing unit or state is never replaced or adopted. No reopen after a
    // crash is inferred; runtime expiry/reboot bounds abandoned attempts.
    for name in [ENROLLMENT_UNIT, ENROLLMENT_SOCKET_UNIT] {
        let unit = run.run(
            ctx,
            "systemctl",
            &[
                "show".to_string(),
                "--property=LoadState".to_string(),
                "--value".to_string(),
                name.to_string(),
            ],
            None,
        );
        let loaded = match unit {
            Ok(output) => String::from_utf8_lossy(&output).trim().to_string(),
            Err(_) => String::new(),
        };
        if loaded != "not-found" {
            return Err(Error::msg(
                "existing enrollment service/state must finish; no replacement",
            ));
        }
    }
    // A template without an instance is a unit file, not a loadable service.
    // Inspect the manager's unit-file search path rather than trying to start or
    // load the abstract @.service name, and never replace a vendor/operator file.
    let templates = run.run(
        ctx,
        "systemctl",
        &[
            "list-unit-files".to_string(),
            "--no-legend".to_string(),
            "--no-pager".to_string(),
            ENROLLMENT_TEMPLATE_UNIT.to_string(),
        ],
        None,
    );
    let listing = match &templates {
        Ok(output) => String::from_utf8_lossy(output).trim().to_string(),
        Err(_) => String::new(),
    };
    if !listing.is_empty() {
        return Err(Error::msg(
            "existing enrollment service template must be preserved; no replacement",
        ));
    }
    if let Err(err) = templates {
        if !systemd_unit_file_no_match(&err) {
            return Err(Error::msg(
                "cannot verify enrollment service template absence; no replacement",
            ));
        }
    }
    use std::os::unix::fs::DirBuilderExt;
    if std::fs::DirBuilder::new()
        .mode(0o700)
        .create(ENROLLMENT_DIR)
        .is_err()
    {
        return Err(Error::msg(
            "enrollment state already exists or cannot be reserved; no replacement",
        ));
    }
    Ok(())
}

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
use crate::fmtx::{fold_eq_ascii, Arg};
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

pub fn enrollment_live_address(selected: &EnrollmentAddress) -> bool {
    match enrollment_addresses() {
        Ok(addresses) => addresses.iter().any(|candidate| candidate == selected),
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

fn create_enrollment_temp(directory: &str, name: &str) -> Result<(String, std::fs::File), Error> {
    use std::os::unix::fs::OpenOptionsExt;
    for _ in 0..100 {
        let path = format!("{directory}/.{name}-{}", super::keys::random_hex(8)?);
        match std::fs::OpenOptions::new()
            .read(true)
            .write(true)
            .create_new(true)
            .mode(0o600)
            .open(&path)
        {
            Ok(file) => return Ok((path, file)),
            Err(err) if err.kind() == std::io::ErrorKind::AlreadyExists => continue,
            Err(err) => return Err(errors::path_error("open", &path, err)),
        }
    }
    Err(Error::msg("cannot reserve enrollment temporary file"))
}

pub fn enrollment_write(directory: &str, name: &str, value: &str) -> Result<(), Error> {
    use std::io::Write;
    use std::os::unix::io::IntoRawFd;
    let (temp_path, mut temp) = create_enrollment_temp(directory, name)?;
    // Only this call's exact temporary path is removed on every path; the
    // atomic link publishes the complete file without replacing existing
    // state. Readers must never observe a just-created empty success receipt.
    let write_err = temp
        .write(value.as_bytes())
        .err()
        .map(|e| errors::path_error("write", &temp_path, e));
    // Go closes explicitly and reports the close error; a dropped File
    // swallows it, so close through libc for the same observation.
    let close_err = if unsafe { libc::close(temp.into_raw_fd()) } != 0 {
        let errno = unsafe { *libc::__errno_location() };
        Some(errors::path_error(
            "close",
            &temp_path,
            std::io::Error::from_raw_os_error(errno),
        ))
    } else {
        None
    };
    if let Some(err) = write_err {
        let _ = std::fs::remove_file(&temp_path);
        return Err(err);
    }
    if let Some(err) = close_err {
        let _ = std::fs::remove_file(&temp_path);
        return Err(err);
    }
    let destination = format!("{directory}/{name}");
    if let Err(err) = std::fs::hard_link(&temp_path, &destination) {
        let _ = std::fs::remove_file(&temp_path);
        return Err(errors::link_error("link", &temp_path, &destination, err));
    }
    let _ = std::fs::remove_file(&temp_path);
    Ok(())
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
    if fold_eq_ascii(value, "back") || fold_eq_ascii(value, "cancel") {
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
            "%d. %q: %s",
            &[
                Arg::Int((i + 1) as i64),
                Arg::Str(&address.name),
                Arg::Str(&address.ip),
            ],
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
                    "Enter a number from 1 to %d, or back/cancel.",
                    &[Arg::Int(addresses.len() as i64)],
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
    console.print("Native host fingerprint: %s", &[Arg::Str(fingerprint)]);
    console.print(
        "A temporary password-only key-import connection will listen on %s:%s for at most five minutes. It uses OpenSSH's native password verification; additional PAM policies are not inherited. Ordinary SSH policy is unchanged. Press Enter or Ctrl-C to close it early.",
        &[Arg::Str(&selected.ip), Arg::Str(ENROLLMENT_PORT)],
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
    console.print("Import one laptop public key using the native root password. Choose the host address your laptop can actually reach; an interface address alone does not prove a client route.", &[]);
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

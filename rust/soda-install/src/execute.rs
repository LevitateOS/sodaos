//! Disk-install execution: attempt retry loop, media verification,
//! destination rendering, the destructive `coreos-installer` boundary, and
//! the diagnostic landing console every outcome reaches.

use std::ffi::CString;
use std::io::Read;

use crate::buildx;
use crate::candidate;
use crate::command::{failure_summary, Runner};
use crate::console::Console;
use crate::disks;
use crate::errors::Error;
use crate::fmtx::Arg;
use crate::run::{DATA_DIR, DISK_ATTEMPT_MARKER};
use crate::signal::Ctx;
use crate::wizard::{collect_disk_install_choices, DiskInstallChoices};

/// `readRegular`: bounded regular-file read that refuses symlinks.
pub fn read_regular(path: &str, limit: u64) -> Result<Vec<u8>, Error> {
    use std::os::unix::fs::OpenOptionsExt;
    let file = std::fs::OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK)
        .open(path)
        .map_err(|e| crate::errors::path_error("open", path, e))?;
    let st = file.metadata().map_err(|e| crate::errors::path_error("stat", path, e))?;
    if !st.file_type().is_file() || st.len() > limit {
        return Err(Error::msg("bounded regular file required"));
    }
    let mut data = Vec::new();
    (&file)
        .take(limit + 1)
        .read_to_end(&mut data)
        .map_err(|e| crate::errors::path_error("read", path, e))?;
    if data.len() as u64 > limit {
        return Err(Error::msg("file exceeds size limit"));
    }
    Ok(data)
}

pub fn disk_installation_started(marker: &str) -> Result<bool, Error> {
    match std::fs::symlink_metadata(marker) {
        Ok(_) => Ok(true),
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(false),
        Err(_) => Err(Error::msg("cannot verify disk installation attempt marker")),
    }
}

fn ask_restart_or_quit(console: &Console, ctx: &Ctx) -> Result<bool, Error> {
    console.page("Installation cancelled before disk writing");
    console.print("No disk installation was started.", &[]);
    console.print("Restart reuses this already loaded installer executable.", &[]);
    loop {
        let choice = console.ask(ctx, "Type restart or quit")?;
        match crate::fmtx::go_lower(&choice).as_str() {
            "restart" => return Ok(true),
            "quit" | "cancel" => return Err(Error::msg("cancelled; no disk installation started")),
            _ => console.print("Choose restart or quit.", &[]),
        }
    }
}

fn after_failed_disk_attempt(
    console: &Console,
    ctx: &Ctx,
    marker: &str,
    err: Error,
    interrupted: bool,
) -> Result<bool, Error> {
    let started = disk_installation_started(marker)?;
    if started || (!matches!(err, Error::Restart) && !matches!(err, Error::Cancel) && !interrupted) {
        return Err(err);
    }
    if matches!(err, Error::Restart) {
        return Ok(true);
    }
    ask_restart_or_quit(console, ctx)
}

pub fn retry_disk_install(
    ctx: &Ctx,
    console: &Console,
    marker: &str,
    attempt: &dyn Fn(&Ctx, &Console) -> Result<(), Error>,
) -> Result<(), Error> {
    loop {
        if disk_installation_started(marker)? {
            return Err(Error::msg("disk installation was already attempted this boot; inspect the result, do not replay"));
        }
        let attempt_ctx = ctx.interrupt_scope();
        let err = attempt(&attempt_ctx, console);
        let interrupted = ctx.err().is_none()
            && (attempt_ctx.err().is_some() || matches!(err, Err(Error::Canceled)));
        match err {
            Ok(()) => return Ok(()),
            Err(err) => {
                // Refusals arrive as errors; a clean stop is unreachable but
                // returns like Go's nil error.
                if !after_failed_disk_attempt(console, ctx, marker, err, interrupted)? {
                    return Ok(());
                }
            }
        }
    }
}

fn destination_for_media(
    media: &crate::run::MediaIdentity,
    template: &[u8],
    choices: &DiskInstallChoices,
) -> Result<Vec<u8>, Error> {
    media.validate(&media.release, &crate::run::architecture())?;
    let factory = read_regular("/usr/share/soda/defaults/host.example.json", 16384)?;
    candidate::candidate_destination(template, &factory, choices)
}

fn print_disk_complete(console: &Console) {
    console.print("SodaOS disk installation completed with all five application images local.", &[]);
    console.print("Remove installation media, then confirm the reboot prompt below; log in locally as root with your password.", &[]);
    console.print("Native startup imports the included images before starting their services.", &[]);
    console.print("SSH password access is enabled; log in as root over SSH with your password.", &[]);
    console.print(
        "To go key-only later, run locally after reboot: %s enroll-key, then disable password logins yourself.",
        &[Arg::Str(candidate::CANDIDATE_INSTALLER_BINARY)],
    );
    console.print("Then complete browser setup from your SSH terminal: %s configure", &[Arg::Str(candidate::CANDIDATE_INSTALLER_BINARY)]);
}

/// Terminal visible state for every disk-install outcome: success,
/// cancellation, partial failure, EOF and reboot failure all land here and
/// stay on an explicit reboot/poweroff prompt.
pub fn land_diagnostic_console(
    ctx: &Ctx,
    console: &Console,
    run: &dyn Runner,
    install_err: Option<Error>,
) -> Result<(), Error> {
    if let Some(err) = &install_err {
        console.print("Installation did not complete: %v.", &[Arg::Str(&err.to_string())]);
        console.print("No automatic retry or reboot was performed. Inspect this live boot from another terminal; confirming a power action below discards live-boot inspection state.", &[]);
    } else {
        console.print("Installation completed. The completion details above stay on screen; nothing further runs until you choose a power action.", &[]);
    }
    let mut land_ctx = ctx.interrupt_scope();
    loop {
        match console.ask(&land_ctx, "Type reboot or poweroff") {
            Err(err) => {
                if land_ctx.err().is_some() && ctx.err().is_none() {
                    console.print("Interrupted; the console stays. Type reboot or poweroff.", &[]);
                    land_ctx = ctx.interrupt_scope();
                    continue;
                }
                console.print(
                    "Terminal input failed (%v); attempting one reboot so the machine is not stranded.",
                    &[Arg::Str(&err.to_string())],
                );
                if let Err(run_err) = run.run(ctx, "systemctl", &["reboot".to_string()], None) {
                    if let Some(err) = install_err {
                        return Err(err);
                    }
                    return Err(run_err);
                }
                return install_err.map_or(Ok(()), Err);
            }
            Ok(choice) => {
                let action = crate::fmtx::go_lower(&choice);
                if action != "reboot" && action != "poweroff" {
                    console.print("Choose reboot or poweroff.", &[]);
                    continue;
                }
                if let Err(run_err) = run.run(ctx, "systemctl", &[action.clone()], None) {
                    console.print(
                        "%s failed: %v. The console stays; inspect or choose again.",
                        &[Arg::Str(&action), Arg::Str(&run_err.to_string())],
                    );
                    continue;
                }
                console.print("%s issued...", &[Arg::Str(&action)]);
                return install_err.map_or(Ok(()), Err);
            }
        }
    }
}

/// Hashes the full media payload before any prompt.
pub fn verify_disk_media() -> Result<(crate::run::MediaIdentity, u64), Error> {
    let media = crate::run::read_media_identity(&format!("{DATA_DIR}/media.json"))
        .map_err(|_| Error::msg("missing media identity"))?;
    let payload_bytes = candidate::candidate_requirement(&media, "/")?;
    Ok((media, payload_bytes))
}

fn write_attempt_ignition(destination: &[u8]) -> Result<String, Error> {
    let template = CString::new("/run/soda-installer-XXXXXX").unwrap();
    let raw = template.into_raw();
    let ok = unsafe { libc::mkdtemp(raw) };
    let template = unsafe { CString::from_raw(raw) };
    if ok.is_null() {
        let errno = unsafe { *libc::__errno_location() };
        return Err(crate::errors::path_error(
            "mkdir",
            &template.to_string_lossy(),
            std::io::Error::from_raw_os_error(errno),
        ));
    }
    let work = template.to_string_lossy().into_owned();
    let ignition = format!("{work}/destination.ign");
    buildx::write_new(&ignition, destination, 0o600)?;
    Ok(ignition)
}

fn begin_disk_attempt(console: &Console) -> Result<(), Error> {
    let welcome = read_regular("/etc/motd", 16384).map_err(|_| Error::msg("cannot read installer welcome text"))?;
    console.print("\x1b[0m\x1b[2J\x1b[H%s", &[Arg::Bytes(&welcome)]);
    console.print("Checking installation media. Please wait; this hashes gigabytes and takes a while on slow drives.", &[]);
    Ok(())
}

/// Runs only after media verification, so Enter leads straight into the
/// first step with no further silent work.
pub fn prompt_disk_attempt(console: &Console, ctx: &Ctx) -> Result<(), Error> {
    console.print("Media verified. Press Enter to begin. Ctrl-C cancels safely before disk writing.", &[]);
    console.line(ctx).map(|_| ())
}

fn finish_disk_attempt(console: &Console) -> Result<(), Error> {
    print_disk_complete(console);
    Ok(())
}

fn install_disk_attempt(ctx: &Ctx, console: &Console, run: &dyn Runner, marker: &str) -> Result<(), Error> {
    begin_disk_attempt(console)?;
    let (media, payload_bytes) = verify_disk_media()?;
    prompt_disk_attempt(console, ctx)?;
    let mut choices = collect_disk_install_choices(ctx, console, run, &|ctx, run| disks::scan_disks(ctx, run), payload_bytes)?;
    let template = read_regular(&format!("{DATA_DIR}/destination.ign"), 4 << 20)?;
    let destination = destination_for_media(&media, &template, &choices);
    choices.password_hash.clear();
    let destination = destination?;
    let ignition = write_attempt_ignition(&destination)?;
    console.page("Installing CoreOS");
    console.print("Writing the confirmed disk. Do not disconnect it.", &[]);
    console.print("Raw diagnostics are suppressed to protect provisioning inputs.", &[]);
    execute_attempt_disk(ctx, &choices.disk, &ignition, marker, run, choices.removable_ok)?;
    finish_disk_attempt(console)
}

fn execute_attempt_disk(
    ctx: &Ctx,
    disk: &disks::Disk,
    ignition: &str,
    marker: &str,
    run: &dyn Runner,
    removable_confirmed: bool,
) -> Result<(), Error> {
    let name = disk.device.name.clone();
    execute_disk(
        ctx,
        disk,
        ignition,
        &|| disks::scan_disks(ctx, run),
        &|| buildx::write_new(marker, format!("{name}\n").as_bytes(), 0o600),
        run,
        removable_confirmed,
    )
}

pub fn execute_disk(
    ctx: &Ctx,
    selected: &disks::Disk,
    ignition: &str,
    inspect: &dyn Fn() -> Result<Vec<disks::Disk>, Error>,
    mark: &dyn Fn() -> Result<(), Error>,
    run: &dyn Runner,
    removable_confirmed: bool,
) -> Result<(), Error> {
    if let Some(err) = ctx.err() {
        return Err(err);
    }
    // Blocked or unavailable selections never reach a write, even when the
    // fresh inventory matches them exactly.
    if !selected.blocked.is_empty() {
        return Err(Error::msg(format!(
            "selected disk is unavailable ({}); no installation started",
            selected.blocked
        )));
    }
    let observed = inspect()?;
    disks::same_disk(selected, &observed)?;
    for current in &observed {
        if current.device.name == selected.device.name && !current.blocked.is_empty() {
            return Err(Error::msg(format!(
                "selected disk became unavailable ({}); no installation started",
                current.blocked
            )));
        }
    }
    // Removable targets write only after the explicit removable
    // confirmation travelled with this attempt.
    if selected.removable && !removable_confirmed {
        return Err(Error::msg(format!(
            "removable disk {} needs explicit intentional confirmation; no installation started",
            selected.device.name
        )));
    }
    mark().map_err(|_| Error::msg("cannot reserve disk installation attempt"))?;
    let args = ["install", "--offline", "--ignition-file", ignition, "--copy-network", &selected.device.name]
        .iter()
        .map(|s| s.to_string())
        .collect::<Vec<_>>();
    if let Err(err) = run.run(ctx, "coreos-installer", &args, None) {
        return Err(Error::msg(format!(
            "CoreOS installation failed or was interrupted; disk may be partially written. No retry or reboot was performed; preserve this boot for operator inspection. {}",
            failure_summary(&err)
        )));
    }
    Ok(())
}

pub fn install_disk_at(ctx: &Ctx, console: &Console, run: &dyn Runner, marker: &str) -> Result<(), Error> {
    retry_disk_install(ctx, console, marker, &|attempt_ctx, attempt_console| {
        install_disk_attempt(attempt_ctx, attempt_console, run, marker)
    })
}

pub fn install_disk(ctx: &Ctx, console: &Console, run: &dyn Runner) -> Result<(), Error> {
    install_disk_at(ctx, console, run, DISK_ATTEMPT_MARKER)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::command::FnRunner;
    use std::io::Write;
    use std::sync::atomic::{AtomicUsize, Ordering};

    fn fixture_disk() -> disks::Disk {
        disks::Disk {
            device: crate::disks::BlockDevice {
                name: "/dev/sda".to_string(),
                kname: "/dev/sda".to_string(),
                device_type: "disk".to_string(),
                tran: String::new(),
                size: 64 << 30,
                model: String::new(),
                serial: "fixture".to_string(),
                wwn: String::new(),
                major_minor: "8:0".to_string(),
                read_only: false,
                mountpoints: Some(vec![None]),
                fstype: String::new(),
                uuid: String::new(),
                partuuid: String::new(),
                children: vec![crate::disks::BlockDevice {
                    name: "/dev/sda1".to_string(),
                    kname: "/dev/sda1".to_string(),
                    device_type: "part".to_string(),
                    tran: String::new(),
                    size: 32 << 30,
                    model: String::new(),
                    serial: String::new(),
                    wwn: String::new(),
                    major_minor: "8:1".to_string(),
                    read_only: false,
                    mountpoints: Some(vec![None]),
                    fstype: "ext4".to_string(),
                    uuid: String::new(),
                    partuuid: String::new(),
                    children: Vec::new(),
                }],
            },
            sequence: "17".to_string(),
            blocked: String::new(),
            removable: false,
        }
    }

    #[test]
    fn execution_boundary() {
        for (name, changed, cancelled, mark_failure, install_failure) in
            [("success", false, false, false, false), ("hotplug", true, false, false, false), ("cancel", false, true, false, false), ("marker", false, false, true, false), ("partial", false, false, false, true)]
        {
            let (ctx, flag) = Ctx::test();
            if cancelled {
                flag.store(true, Ordering::SeqCst);
            }
            let calls = AtomicUsize::new(0);
            let marks = AtomicUsize::new(0);
            let run = FnRunner::new(|_, cmd: &str, args: &[String], input: Option<&[u8]>| {
                calls.fetch_add(1, Ordering::SeqCst);
                assert_eq!(cmd, "coreos-installer");
                assert_eq!(
                    args,
                    &["install", "--offline", "--ignition-file", "/private/destination.ign", "--copy-network", "/dev/sda"]
                        .iter()
                        .map(|s| s.to_string())
                        .collect::<Vec<_>>()
                );
                assert!(input.is_none());
                assert_eq!(marks.load(Ordering::SeqCst), 1);
                if install_failure {
                    return Err(Error::msg("synthetic credential from native diagnostics"));
                }
                Ok(Vec::new())
            });
            let err = execute_disk(
                &ctx,
                &fixture_disk(),
                "/private/destination.ign",
                &|| {
                    let mut disk = fixture_disk();
                    if changed {
                        disk.sequence = "18".to_string();
                    }
                    Ok(vec![disk])
                },
                &|| {
                    marks.fetch_add(1, Ordering::SeqCst);
                    if mark_failure {
                        return Err(Error::msg("failed"));
                    }
                    Ok(())
                },
                &run,
                false,
            );
            let early = changed || cancelled || mark_failure;
            assert_eq!(calls.load(Ordering::SeqCst), if early { 0 } else { 1 }, "{name}: calls");
            assert_eq!(err.is_err(), early || install_failure, "{name}: {err:?}");
            if let Err(err) = err {
                assert!(!err.to_string().contains("synthetic credential"), "{name}: leak");
            }
        }
    }

    #[test]
    fn removable_and_blocked_gates() {
        let (ctx, _flag) = Ctx::test();
        let mut removable = fixture_disk();
        removable.removable = true;
        let run = FnRunner::new(|_, _, _, _| panic!("refused attempt ran a command"));
        let err = execute_disk(&ctx, &removable, "/private/destination.ign", &|| Ok(vec![removable.clone()]), &|| panic!("marker"), &run, false)
            .unwrap_err();
        assert!(err.to_string().contains("explicit intentional confirmation"), "{err}");
        let writes = AtomicUsize::new(0);
        let run = FnRunner::new(|_, _, _, _| {
            writes.fetch_add(1, Ordering::SeqCst);
            Ok(Vec::new())
        });
        execute_disk(&ctx, &removable, "/private/destination.ign", &|| Ok(vec![removable.clone()]), &|| Ok(()), &run, true).unwrap();
        assert_eq!(writes.load(Ordering::SeqCst), 1);

        let mut blocked = fixture_disk();
        blocked.blocked = "current installer media".to_string();
        let run = FnRunner::new(|_, _, _, _| panic!("blocked selection ran a command"));
        assert!(execute_disk(&ctx, &blocked, "/private/destination.ign", &|| Ok(vec![blocked.clone()]), &|| panic!("marker"), &run, true).is_err());
        let fresh = fixture_disk();
        assert!(execute_disk(&ctx, &fresh, "/private/destination.ign", &|| Ok(vec![blocked.clone()]), &|| panic!("marker"), &run, false).is_err());
    }

    fn pipe_console(input: &[u8], close_write: bool) -> (Console, Option<std::fs::File>) {
        let mut fds = [0; 2];
        unsafe { assert_eq!(libc::pipe(fds.as_mut_ptr()), 0) };
        let (read, mut write) = unsafe {
            use std::os::unix::io::FromRawFd;
            (std::fs::File::from_raw_fd(fds[0]), std::fs::File::from_raw_fd(fds[1]))
        };
        write.write_all(input).unwrap();
        if close_write {
            drop(write);
            return (Console::from_file("pipe", read), None);
        }
        (Console::from_file("pipe", read), Some(write))
    }

    struct PowerRecorder {
        calls: AtomicUsize,
        args: std::sync::Mutex<Vec<Vec<String>>>,
        failures: AtomicUsize,
    }

    impl Runner for PowerRecorder {
        fn run(&self, _ctx: &Ctx, name: &str, args: &[String], _input: Option<&[u8]>) -> Result<Vec<u8>, Error> {
            assert_eq!(name, "systemctl");
            self.calls.fetch_add(1, Ordering::SeqCst);
            self.args.lock().unwrap().push(args.to_vec());
            if self.failures.load(Ordering::SeqCst) > 0 {
                self.failures.fetch_sub(1, Ordering::SeqCst);
                return Err(Error::msg("systemctl reboot refused"));
            }
            Ok(Vec::new())
        }
    }

    #[test]
    fn landing_outcomes() {
        let (ctx, _flag) = Ctx::test();
        // Success reboots on choice.
        let rec = PowerRecorder { calls: AtomicUsize::new(0), args: std::sync::Mutex::new(Vec::new()), failures: AtomicUsize::new(0) };
        let (console, _w) = pipe_console(b"reboot\n", false);
        land_diagnostic_console(&ctx, &console, &rec, None).unwrap();
        assert_eq!(rec.calls.load(Ordering::SeqCst), 1);
        assert_eq!(rec.args.lock().unwrap()[0], vec!["reboot".to_string()]);
        // Failure keeps the install error.
        let rec = PowerRecorder { calls: AtomicUsize::new(0), args: std::sync::Mutex::new(Vec::new()), failures: AtomicUsize::new(0) };
        let (console, _w) = pipe_console(b"poweroff\n", false);
        let err = land_diagnostic_console(&ctx, &console, &rec, Some(Error::msg("disk write failed"))).unwrap_err();
        assert_eq!(err, Error::msg("disk write failed"));
        assert_eq!(rec.args.lock().unwrap()[0], vec!["poweroff".to_string()]);
        // Unknown choice reprompts.
        let rec = PowerRecorder { calls: AtomicUsize::new(0), args: std::sync::Mutex::new(Vec::new()), failures: AtomicUsize::new(0) };
        let (console, _w) = pipe_console(b"retry\nreboot\n", false);
        land_diagnostic_console(&ctx, &console, &rec, None).unwrap();
        assert_eq!(rec.calls.load(Ordering::SeqCst), 1);
        // Failed power action stays.
        let rec = PowerRecorder { calls: AtomicUsize::new(0), args: std::sync::Mutex::new(Vec::new()), failures: AtomicUsize::new(1) };
        let (console, _w) = pipe_console(b"reboot\npoweroff\n", false);
        land_diagnostic_console(&ctx, &console, &rec, None).unwrap();
        assert_eq!(rec.calls.load(Ordering::SeqCst), 2);
        // Dead terminal reboots once.
        let rec = PowerRecorder { calls: AtomicUsize::new(0), args: std::sync::Mutex::new(Vec::new()), failures: AtomicUsize::new(0) };
        let (console, _w) = pipe_console(b"", true);
        land_diagnostic_console(&ctx, &console, &rec, None).unwrap();
        assert_eq!(rec.calls.load(Ordering::SeqCst), 1);
        // Dead terminal with failed reboot reports the install error.
        let rec = PowerRecorder { calls: AtomicUsize::new(0), args: std::sync::Mutex::new(Vec::new()), failures: AtomicUsize::new(1) };
        let (console, _w) = pipe_console(b"", true);
        let err = land_diagnostic_console(&ctx, &console, &rec, Some(Error::msg("disk write failed"))).unwrap_err();
        assert_eq!(err, Error::msg("disk write failed"));
    }

    #[test]
    fn media_verification_fails_without_identity() {
        // This test machine has no installer media identity.
        let err = verify_disk_media().unwrap_err();
        assert!(err.to_string().contains("missing media identity"), "{err}");
    }
}

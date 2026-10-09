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
    for (name, changed, cancelled, mark_failure, install_failure) in [
        ("success", false, false, false, false),
        ("hotplug", true, false, false, false),
        ("cancel", false, true, false, false),
        ("marker", false, false, true, false),
        ("partial", false, false, false, true),
    ] {
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
                &[
                    "install",
                    "--offline",
                    "--ignition-file",
                    "/private/destination.ign",
                    "--copy-network",
                    "/dev/sda"
                ]
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
        assert_eq!(
            calls.load(Ordering::SeqCst),
            if early { 0 } else { 1 },
            "{name}: calls"
        );
        assert_eq!(err.is_err(), early || install_failure, "{name}: {err:?}");
        if let Err(err) = err {
            assert!(
                !err.to_string().contains("synthetic credential"),
                "{name}: leak"
            );
        }
    }
}

#[test]
fn removable_and_blocked_gates() {
    let (ctx, _flag) = Ctx::test();
    let mut removable = fixture_disk();
    removable.removable = true;
    let run = FnRunner::new(|_, _, _, _| panic!("refused attempt ran a command"));
    let err = execute_disk(
        &ctx,
        &removable,
        "/private/destination.ign",
        &|| Ok(vec![removable.clone()]),
        &|| panic!("marker"),
        &run,
        false,
    )
    .unwrap_err();
    assert!(
        err.to_string()
            .contains("explicit intentional confirmation"),
        "{err}"
    );
    let writes = AtomicUsize::new(0);
    let run = FnRunner::new(|_, _, _, _| {
        writes.fetch_add(1, Ordering::SeqCst);
        Ok(Vec::new())
    });
    execute_disk(
        &ctx,
        &removable,
        "/private/destination.ign",
        &|| Ok(vec![removable.clone()]),
        &|| Ok(()),
        &run,
        true,
    )
    .unwrap();
    assert_eq!(writes.load(Ordering::SeqCst), 1);

    let mut blocked = fixture_disk();
    blocked.blocked = "current installer media".to_string();
    let run = FnRunner::new(|_, _, _, _| panic!("blocked selection ran a command"));
    assert!(execute_disk(
        &ctx,
        &blocked,
        "/private/destination.ign",
        &|| Ok(vec![blocked.clone()]),
        &|| panic!("marker"),
        &run,
        true
    )
    .is_err());
    let fresh = fixture_disk();
    assert!(execute_disk(
        &ctx,
        &fresh,
        "/private/destination.ign",
        &|| Ok(vec![blocked.clone()]),
        &|| panic!("marker"),
        &run,
        false
    )
    .is_err());
}

fn pipe_console(input: &[u8], close_write: bool) -> (Console, Option<std::fs::File>) {
    let mut fds = [0; 2];
    unsafe { assert_eq!(libc::pipe(fds.as_mut_ptr()), 0) };
    let (read, mut write) = unsafe {
        use std::os::unix::io::FromRawFd;
        (
            std::fs::File::from_raw_fd(fds[0]),
            std::fs::File::from_raw_fd(fds[1]),
        )
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
    fn run(
        &self,
        _ctx: &Ctx,
        name: &str,
        args: &[String],
        _input: Option<&[u8]>,
    ) -> Result<Vec<u8>, Error> {
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
    let rec = PowerRecorder {
        calls: AtomicUsize::new(0),
        args: std::sync::Mutex::new(Vec::new()),
        failures: AtomicUsize::new(0),
    };
    let (console, _w) = pipe_console(b"reboot\n", false);
    land_diagnostic_console(&ctx, &console, &rec, None).unwrap();
    assert_eq!(rec.calls.load(Ordering::SeqCst), 1);
    assert_eq!(rec.args.lock().unwrap()[0], vec!["reboot".to_string()]);
    // Failure keeps the install error.
    let rec = PowerRecorder {
        calls: AtomicUsize::new(0),
        args: std::sync::Mutex::new(Vec::new()),
        failures: AtomicUsize::new(0),
    };
    let (console, _w) = pipe_console(b"poweroff\n", false);
    let err = land_diagnostic_console(&ctx, &console, &rec, Some(Error::msg("disk write failed")))
        .unwrap_err();
    assert_eq!(err, Error::msg("disk write failed"));
    assert_eq!(rec.args.lock().unwrap()[0], vec!["poweroff".to_string()]);
    // Unknown choice reprompts.
    let rec = PowerRecorder {
        calls: AtomicUsize::new(0),
        args: std::sync::Mutex::new(Vec::new()),
        failures: AtomicUsize::new(0),
    };
    let (console, _w) = pipe_console(b"retry\nreboot\n", false);
    land_diagnostic_console(&ctx, &console, &rec, None).unwrap();
    assert_eq!(rec.calls.load(Ordering::SeqCst), 1);
    // Failed power action stays.
    let rec = PowerRecorder {
        calls: AtomicUsize::new(0),
        args: std::sync::Mutex::new(Vec::new()),
        failures: AtomicUsize::new(1),
    };
    let (console, _w) = pipe_console(b"reboot\npoweroff\n", false);
    land_diagnostic_console(&ctx, &console, &rec, None).unwrap();
    assert_eq!(rec.calls.load(Ordering::SeqCst), 2);
    // Dead terminal reboots once.
    let rec = PowerRecorder {
        calls: AtomicUsize::new(0),
        args: std::sync::Mutex::new(Vec::new()),
        failures: AtomicUsize::new(0),
    };
    let (console, _w) = pipe_console(b"", true);
    land_diagnostic_console(&ctx, &console, &rec, None).unwrap();
    assert_eq!(rec.calls.load(Ordering::SeqCst), 1);
    // Dead terminal with failed reboot reports the install error.
    let rec = PowerRecorder {
        calls: AtomicUsize::new(0),
        args: std::sync::Mutex::new(Vec::new()),
        failures: AtomicUsize::new(1),
    };
    let (console, _w) = pipe_console(b"", true);
    let err = land_diagnostic_console(&ctx, &console, &rec, Some(Error::msg("disk write failed")))
        .unwrap_err();
    assert_eq!(err, Error::msg("disk write failed"));
    assert_eq!(rec.calls.load(Ordering::SeqCst), 1);
}

#[test]
fn eof_reboot_is_one_bounded_attempt_and_returns_command_failure() {
    let (ctx, _flag) = Ctx::test();
    let calls = AtomicUsize::new(0);
    let run = FnRunner::new(|ctx: &Ctx, name: &str, args: &[String], _| {
        calls.fetch_add(1, Ordering::SeqCst);
        assert_eq!(name, "systemctl");
        assert_eq!(args, &["reboot".to_string()]);
        let deadline = ctx
            .deadline()
            .expect("automatic reboot must have a deadline");
        assert!(deadline > std::time::Instant::now());
        assert!(deadline <= std::time::Instant::now() + std::time::Duration::from_secs(2 * 60));
        Err(Error::CmdExit {
            name: "systemctl".to_string(),
            code: 1,
            interrupted: false,
        })
    });
    let (console, _w) = pipe_console(b"", true);

    let err = land_diagnostic_console(&ctx, &console, &run, None).unwrap_err();

    assert_eq!(calls.load(Ordering::SeqCst), 1);
    assert_eq!(
        err,
        Error::CmdExit {
            name: "systemctl".to_string(),
            code: 1,
            interrupted: false,
        }
    );
}

#[test]
fn media_verification_fails_without_identity() {
    // This test machine has no installer media identity.
    let err = verify_disk_media().unwrap_err();
    assert!(err.to_string().contains("missing media identity"), "{err}");
}

use super::arm::*;
use super::test_support::{drain_available, open_test_pty, temp_dir, ENV_LOCK};
use super::{arm_enrollment, receive_enrollment, serve_enrollment, ENROLLMENT_DIR};

use crate::command::FnRunner;
use crate::console::Console;
use crate::errors::Error;
use crate::signal::Ctx;

#[test]
fn enrollment_choice_matrix() {
    assert_eq!(parse_enrollment_choice("1", 2).unwrap(), 1);
    assert_eq!(parse_enrollment_choice("2", 2).unwrap(), 2);
    for bad in ["0", "3", "typo", "", "1.0", "+1x"] {
        assert_eq!(
            parse_enrollment_choice(bad, 2).unwrap_err().to_string(),
            "enter a listed enrollment address",
            "{bad:?}"
        );
    }
    for cancel in ["back", "Back", "BACK", "cancel", "Cancel", "CANCEL"] {
        assert_eq!(
            parse_enrollment_choice(cancel, 2).unwrap_err().to_string(),
            "key enrollment cancelled",
            "{cancel:?}"
        );
    }
}

#[test]
fn enrollment_state_publishes_whole_and_preserves_existing() {
    let directory = temp_dir();
    let path = format!("{}/result", directory.path);
    let value = "synthetic state\n".repeat(16384);
    let writer = std::thread::scope(|scope| {
        let handle = scope.spawn(|| enrollment_write(&directory.path, "result", &value));
        loop {
            match std::fs::read(&path) {
                Ok(data) => assert_eq!(
                    data,
                    value.as_bytes(),
                    "reader observed a partial state receipt"
                ),
                Err(err) if err.kind() == std::io::ErrorKind::NotFound => {}
                Err(err) => panic!("{err}"),
            }
            if handle.is_finished() {
                break handle.join().unwrap();
            }
        }
    });
    writer.unwrap();
    assert!(enrollment_write(&directory.path, "result", "replacement").is_err());
    assert_eq!(std::fs::read(&path).unwrap(), value.as_bytes());
    assert_eq!(std::fs::read_dir(&directory.path).unwrap().count(), 1);
}

#[allow(clippy::type_complexity)]
fn guard_runner(
    list_output: Vec<u8>,
    list_err: Option<Error>,
) -> FnRunner<impl Fn(&Ctx, &str, &[String], Option<&[u8]>) -> Result<Vec<u8>, Error>> {
    FnRunner::new(move |_, name, args, _| {
        assert_eq!(name, "systemctl");
        assert!(!args.is_empty());
        match args[0].as_str() {
            "show" => Ok(b"not-found\n".to_vec()),
            "list-unit-files" => match &list_err {
                Some(err) => Err(err.clone()),
                None => Ok(list_output.clone()),
            },
            other => panic!("unexpected guard command {other}"),
        }
    })
}

#[test]
fn guard_template_absent() {
    let (ctx, _flag) = Ctx::test();
    for (list_output, list_err) in [
        (b"".to_vec(), None),
        (
            Vec::new(),
            Some(Error::CmdExit {
                name: "systemctl".to_string(),
                code: 1,
                interrupted: false,
            }),
        ),
    ] {
        let run = guard_runner(list_output, list_err);
        match guard_existing_enrollment_state(&ctx, &run) {
            Ok(()) => {
                // Privileged runs reserve real state; release what this call created.
                std::fs::remove_dir(ENROLLMENT_DIR).unwrap();
            }
            Err(err) => {
                assert!(
                    !err.to_string().contains("template"),
                    "absent template refused arming: {err}"
                );
                assert!(
                    err.to_string().contains("cannot be reserved"),
                    "unexpected guard error: {err}"
                );
            }
        }
    }
}

#[test]
fn guard_template_refused() {
    let (ctx, _flag) = Ctx::test();
    for (list_output, list_err) in [
        (b"soda-key-enrollment@.service static -\n".to_vec(), None),
        (
            Vec::new(),
            Some(Error::CmdExit {
                name: "systemctl".to_string(),
                code: 1,
                interrupted: true,
            }),
        ),
        (
            Vec::new(),
            Some(Error::CmdExit {
                name: "systemctl".to_string(),
                code: -1,
                interrupted: false,
            }),
        ),
    ] {
        let run = guard_runner(list_output, list_err);
        match guard_existing_enrollment_state(&ctx, &run) {
            Ok(()) => {
                let _ = std::fs::remove_dir(ENROLLMENT_DIR);
                panic!("guard accepted despite template presence or failed inspection");
            }
            Err(err) => {
                assert!(
                    !err.to_string().contains("cannot be reserved"),
                    "guard proceeded past the template check: {err}"
                );
            }
        }
    }
}

#[test]
fn address_selection_retries_typos() {
    use std::io::Write;
    for input in ["typo\n0\n3\n2\n", "Back\n", "cancel\n"] {
        let pty = open_test_pty();
        let slave_path = pty.slave_path.clone();
        let mut master = pty.master;
        let worker = std::thread::spawn(move || {
            let console = Console::open(&slave_path).unwrap();
            let (ctx, _flag) = Ctx::test();
            let ctx =
                ctx.with_timeout(std::time::Instant::now() + std::time::Duration::from_secs(10));
            let addresses = vec![
                EnrollmentAddress {
                    name: "test0".to_string(),
                    ip: "192.168.1.20".to_string(),
                },
                EnrollmentAddress {
                    name: "test1".to_string(),
                    ip: "10.0.0.20".to_string(),
                },
            ];
            select_enrollment_address(&ctx, &console, &addresses)
        });
        master.write_all(input.as_bytes()).unwrap();
        let selected = worker.join().unwrap();
        if input.starts_with("typo") {
            let selected = selected.unwrap();
            assert_eq!(selected.ip, "10.0.0.20");
            let drained = drain_available(&mut master);
            let transcript = String::from_utf8_lossy(&drained);
            assert_eq!(
                transcript.matches("Enter a number from 1 to 2").count(),
                3,
                "invalid numbers did not receive clear retry feedback: {transcript:?}"
            );
        } else {
            assert!(
                selected.is_err(),
                "explicit back/cancel selected an address"
            );
        }
    }
}

#[test]
fn direct_server_refused() {
    // Tests never launch the transient unit; this must stop at cgroup ownership
    // before reading arm state or opening any socket/listener.
    let (ctx, _flag) = Ctx::test();
    let run = FnRunner::new(|_, _, _, _| Ok(Vec::new()));
    assert!(serve_enrollment(&ctx, &run).is_err());
}

#[test]
fn receive_commands_refused_before_state() {
    let _guard = ENV_LOCK.lock().unwrap();
    let (ctx, _flag) = Ctx::test();
    for value in ["sh", "scp -t /root", "internal-sftp", ":", "\n"] {
        std::env::set_var("SSH_ORIGINAL_COMMAND", value);
        std::env::remove_var("SSH_TTY");
        assert!(
            receive_enrollment(&ctx).is_err(),
            "client command accepted: {value:?}"
        );
    }
    std::env::set_var("SSH_ORIGINAL_COMMAND", "");
    std::env::set_var("SSH_TTY", "/dev/pts/1");
    assert!(receive_enrollment(&ctx).is_err(), "PTY enrollment accepted");
    std::env::remove_var("SSH_ORIGINAL_COMMAND");
    std::env::remove_var("SSH_TTY");
}

#[test]
fn pty_proceeds_past_console_check() {
    let _guard = ENV_LOCK.lock().unwrap();
    std::env::set_var("SSH_CONNECTION", "synthetic");
    std::env::set_var("SSH_TTY", "/dev/pts/99");
    let pty = open_test_pty();
    let console = Console::open(&pty.slave_path).unwrap();
    let (ctx, _flag) = Ctx::test();
    let ctx = ctx.with_timeout(std::time::Instant::now() + std::time::Duration::from_secs(10));
    let run = FnRunner::new(|_, _, _, _| Ok(Vec::new()));
    let err = arm_enrollment(&ctx, &console, &run).unwrap_err();
    assert!(
        !err.to_string().contains("local keyboard/monitor"),
        "SSH arming still refused at the console: {err}"
    );
    std::env::remove_var("SSH_CONNECTION");
    std::env::remove_var("SSH_TTY");
}

#[test]
fn enrollment_write_requires_complete_count() {
    // O07-F1: a successful short write must block publication; only a
    // complete count may proceed to close and link.
    assert!(enrollment_write_error("sshd_config", "/tmp/t", 10, Ok(10)).is_none());
    let err =
        enrollment_write_error("sshd_config", "/tmp/t", 10, Ok(4)).expect("short write refuses");
    assert!(err.to_string().contains("short write"), "{err}");
    assert!(err.to_string().contains("sshd_config"), "{err}");
    let err =
        enrollment_write_error("sshd_config", "/tmp/t", 10, Ok(0)).expect("empty write refuses");
    assert!(err.to_string().contains("short write"), "{err}");
    let io_err = std::io::Error::new(std::io::ErrorKind::WriteZero, "no space");
    let err =
        enrollment_write_error("sshd_config", "/tmp/t", 10, Err(io_err)).expect("io error reports");
    assert!(err.to_string().contains("write"), "{err}");
    assert!(err.to_string().contains("/tmp/t"), "{err}");
}

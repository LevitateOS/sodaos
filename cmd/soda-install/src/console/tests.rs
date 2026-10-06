use super::test_support::*;
use super::*;
use crate::command::FnRunner;
use crate::errors::Error;
use crate::signal::Ctx;
use std::io::Write;

#[test]
fn ask_trims_and_refuses_controls() {
    let pty = open_pty();
    let mut master = pty.master;
    let slave = pty.slave_path.clone();
    let worker = std::thread::spawn(move || {
        let console = Console::open(&slave).unwrap();
        let (ctx, _flag) = Ctx::test();
        console.ask(&ctx, "Name")
    });
    let mut output = read_until(&mut master, b"Name: ", 2000);
    master.write_all(b"  soda-01  \n").unwrap();
    assert_eq!(worker.join().unwrap().unwrap(), "soda-01");
    output.extend(drain_idle(&mut master, 200));
    assert!(String::from_utf8_lossy(&output).contains("Name: "));

    let worker = std::thread::spawn(move || {
        let console = Console::open(&pty.slave_path).unwrap();
        let (ctx, _flag) = Ctx::test();
        console.ask(&ctx, "Name")
    });
    read_until(&mut master, b"Name: ", 2000);
    master.write_all(b"a\x01b\n").unwrap();
    assert_eq!(
        worker.join().unwrap().unwrap_err().to_string(),
        "control character refused"
    );
}

#[test]
fn secret_hides_input_and_restores_echo() {
    let pty = open_pty();
    let slave = pty.slave_path.clone();
    let mut master = pty.master;
    let worker = std::thread::spawn(move || {
        let console = Console::open(&slave).unwrap();
        let (ctx, _flag) = Ctx::test();
        console.secret(&ctx, "Password")
    });
    let mut output = read_until(&mut master, b"Password: ", 2000);
    master.write_all(b"s3cret!\n").unwrap();
    assert_eq!(worker.join().unwrap().unwrap(), b"s3cret!");
    output.extend(drain_idle(&mut master, 200));
    // Hidden input echoes nothing; only the prompt and newline show.
    assert!(!String::from_utf8_lossy(&output).contains("s3cret"));
    // Echo is restored for the next prompt.
    let worker = std::thread::spawn(move || {
        let console = Console::open(&pty.slave_path).unwrap();
        let (ctx, _flag) = Ctx::test();
        console.ask(&ctx, "Next")
    });
    let mut output = read_until(&mut master, b"Next: ", 2000);
    master.write_all(b"visible\n").unwrap();
    assert_eq!(worker.join().unwrap().unwrap(), "visible");
    output.extend(drain_idle(&mut master, 200));
    assert!(String::from_utf8_lossy(&output).contains("Next: "));
}

#[test]
fn cancelled_line_returns_ctx_error() {
    let pty = open_pty();
    let console = Console::open(&pty.slave_path).unwrap();
    let (ctx, flag) = Ctx::test();
    flag.store(true, std::sync::atomic::Ordering::SeqCst);
    assert_eq!(console.line(&ctx).unwrap_err(), Error::Canceled);
}

#[test]
fn network_step_flows() {
    let pty = open_pty();
    let slave = pty.slave_path.clone();
    let mut master = pty.master;
    let worker = std::thread::spawn(move || {
        let console = Console::open(&slave).unwrap();
        let (ctx, _flag) = Ctx::test();
        let runner = FnRunner::new(|_, name, _, _| {
            assert_eq!(name, "ip");
            Ok(b"lo UP 127.0.0.1/8\n".to_vec())
        });
        console.network_with(&ctx, &runner)
    });
    let mut output = read_until(
        &mut master,
        b"Type keep, edit, back, restart, or cancel: ",
        2000,
    );
    master.write_all(b"keep\n").unwrap();
    output.extend(read_until(
        &mut master,
        b"Type yes to use them, edit, back, restart, or cancel: ",
        2000,
    ));
    master.write_all(b"yes\n").unwrap();
    assert!(worker.join().unwrap().is_ok());
    output.extend(drain_idle(&mut master, 200));
    let text = String::from_utf8_lossy(&output);
    assert!(text.contains("DHCP is ready by default."));
    assert!(text.contains("Current live addresses:"));
    assert!(text.contains("\"lo UP 127.0.0.1/8\""));
}

// soda-muse delegates a normal shell invocation over the launch-only socket.
mod account;
mod config;
#[cfg(test)]
mod config_tests;
mod env_prepare;
mod execution;
mod home_cfg;
mod launch;
mod launch_json;
#[cfg(test)]
mod launch_tests;
mod launch_wire;
mod maintenance;
mod paths;
mod runtime;
mod shell;
#[cfg(test)]
mod shell_tests;

use account::account_for;
use config::{copy_config, read_config};
use execution::execute;
use launch::{launch_shell, seqpacket_connect};
use runtime::check_runtime;
use shell::shell_request;

const MUSE_LAUNCH_SOCKET: &str = "/run/soda-muse-interface/launch.sock";
const MUSE_NATIVE: &str = "/usr/local/libexec/soda/muse";

fn main() {
    let (code, err) = run();
    if let Some(e) = err {
        eprintln!("{e}");
    }
    std::process::exit(code);
}

fn run() -> (i32, Option<String>) {
    // Never panic on non-UTF-8 argv; Go tolerates arbitrary bytes.
    let argv: Vec<String> = std::env::args_os()
        .skip(1)
        .map(|a| a.to_string_lossy().into_owned())
        .collect();
    if let Some(done) = native_action(&argv) {
        return done;
    }
    let request = match shell_request(&argv) {
        Ok(r) => r,
        Err(e) => return (1, Some(e)),
    };
    let fd = match seqpacket_connect(MUSE_LAUNCH_SOCKET) {
        Ok(fd) => fd,
        Err(_) => return (1, Some(String::from("muse launch service unavailable"))),
    };
    launch_shell(fd, &request)
}

fn native_action(args: &[String]) -> Option<(i32, Option<String>)> {
    if args.is_empty() {
        return None;
    }
    match args[0].as_str() {
        "--soda-copy-config" => {
            if args.len() != 3 {
                return Some((1, Some(String::from("config source and view required"))));
            }
            Some((0, copy_config(&args[1], &args[2]).err()))
        }
        "--soda-exec" => {
            if args.len() < 3 {
                return Some((1, Some(String::from("execution root and cwd required"))));
            }
            let (code, err) = execute(&args[1], &args[2], &args[3..]);
            Some((code, err))
        }
        _ => metadata_action(args),
    }
}

fn metadata_action(args: &[String]) -> Option<(i32, Option<String>)> {
    match args[0].as_str() {
        "--soda-check" | "--soda-read-config" | "--soda-account" => {}
        _ => return None,
    }
    if args.len() != 2 {
        return Some((1, Some(String::from("one metadata input required"))));
    }
    let err = match args[0].as_str() {
        "--soda-check" => check_runtime(&args[1]).err(),
        "--soda-read-config" => read_config(&args[1]).err(),
        _ => account_for(&args[1]).err(),
    };
    Some((0, err))
}

struct ShellRequest {
    cwd: String,
    args: Vec<String>,
    home: String,
    connection_id: String,
    config_home: String,
    term: String,
    tty: bool,
    cols: u16,
    rows: u16,
}

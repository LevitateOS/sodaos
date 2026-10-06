use std::fs;

use super::process::{exec_replace, os_error};
use super::start::action_start;
use super::state::{pid_display, ssh_args, vm_dir, vm_running};
use super::{Exit, FAIL_PREFIX};

fn action_status(vm: &str) -> Result<(), Exit> {
    let pid_path = format!("{vm}/qemu.pid");
    if vm_running(&pid_path)? {
        // A second read for the display value; if the pidfile vanished in
        // between, the substitution fails and aborts like the first one.
        let display = match fs::read(&pid_path) {
            Ok(bytes) => pid_display(&bytes),
            Err(err) => {
                eprintln!("{FAIL_PREFIX}: {pid_path}: {}", os_error(&err));
                return Err(Exit::Propagate(1));
            }
        };
        println!("soda-test is running (PID {display}); SSH at 127.0.0.1:22220");
        Ok(())
    } else {
        // The script echoes to stdout here, so this is not a refusal.
        println!("soda-test is not running");
        Err(Exit::Propagate(1))
    }
}

fn action_ssh(vm: &str, extra: &[String]) -> ! {
    let mut argv = vec!["ssh".to_string()];
    argv.extend(ssh_args(vm));
    argv.push("root@127.0.0.1".to_string());
    argv.extend(extra.iter().cloned());
    exec_replace("ssh", &argv);
}

fn action_tunnel(vm: &str, web: bool) -> ! {
    let mut argv = vec!["ssh".to_string()];
    argv.extend(ssh_args(vm));
    argv.push("-o".to_string());
    argv.push("ExitOnForwardFailure=yes".to_string());
    argv.push("-NT".to_string());
    if web {
        println!("Keep this running: Forgejo + Sodaspaces https://localhost:24444");
        argv.push("-L".to_string());
        argv.push("127.0.0.1:24444:127.0.0.1:24444".to_string());
    } else {
        println!(
            "Keep this running: Forgejo http://localhost:23000; Cockpit https://localhost:29090"
        );
        argv.push("-L".to_string());
        argv.push("127.0.0.1:23000:127.0.0.1:3000".to_string());
        argv.push("-L".to_string());
        argv.push("127.0.0.1:29090:127.0.0.1:9090".to_string());
    }
    argv.push("root@127.0.0.1".to_string());
    exec_replace("ssh", &argv);
}

fn action_console(vm: &str) -> ! {
    exec_replace(
        "tail",
        &[
            "tail".to_string(),
            "-n".to_string(),
            "80".to_string(),
            "-f".to_string(),
            format!("{vm}/console.log"),
        ],
    );
}

pub(super) fn run_vm(args: &[String]) -> Result<(), Exit> {
    // Like `${1:-status}`, an empty or missing $1 defaults to status.
    let action = args
        .first()
        .map(|arg| arg.as_str())
        .filter(|arg| !arg.is_empty())
        .unwrap_or("status");
    let vm = vm_dir();
    match action {
        "start" => action_start(&vm),
        "status" => action_status(&vm),
        "ssh" => action_ssh(&vm, &args[1..]),
        "tunnel" => action_tunnel(&vm, false),
        "web-tunnel" => action_tunnel(&vm, true),
        "console" => action_console(&vm),
        _ => {
            eprintln!(
                "usage: cargo run -p soda-test-vm -- [start|status|ssh [COMMAND...]|tunnel|web-tunnel|console]"
            );
            Err(Exit::Propagate(2))
        }
    }
}

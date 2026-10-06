//! Fixed project-local terminal agent: PTY sessions, the identity broker,
//! and SSH key management behind one argv-dispatched binary.
//!
//! Dispatch (mirroring `project_terminal.py:main` plus the `broker` and
//! `keys` entry points the Go daemon invokes):
//! - `prepare <id>`: privileged systemd hook (`term::prepare`).
//! - `broker`: one identity request on stdin (`broker::broker_main`).
//! - `keys`: one key request on stdin (`keys::key_main`).
//! - `<action> <9 control args>`: attach relay or metadata line.
//! - anything else: `closed/launch_failed` plus exit 1, exactly like the
//!   `.py` `ValueError('arguments')` path.
//!
//! Every terminal-path failure (bad argv, unknown account, failed action)
//! prints `{"type":"closed","reason":"launch_failed"}` to STDOUT and exits
//! 1; the Go frame pump consumes that as the terminal frame. `EPIPE` on a
//! stdout write exits 0 silently on all paths.

mod account;
mod b64;
mod broker;
mod fs;
mod key_lines;
mod key_request;
mod keys;
mod proto;
mod pty;
mod pty_process;
mod pty_relay;
mod pyemit;
mod sha;
mod subscription_cgroup;
mod subscription_credentials;
mod subscription_prepare;
mod subscription_profile;
mod subscription_retire;
mod subscription_start;
mod subscription_wire;
mod svc;
mod sys;
mod term;
mod timex;

/// `argv` routing (pure over the argument vector, `argv[0]` included).
#[derive(Debug, Clone, PartialEq, Eq)]
enum Route {
    Prepare(String),
    Broker,
    Keys,
    Control(Vec<String>),
    Bad,
}

fn route(argv: &[String]) -> Route {
    if argv.len() == 3 && argv[1] == "prepare" {
        return Route::Prepare(argv[2].clone());
    }
    if argv.len() == 2 && argv[1] == "broker" {
        return Route::Broker;
    }
    if argv.len() == 2 && argv[1] == "keys" {
        return Route::Keys;
    }
    if argv.len() == 11 {
        return Route::Control(argv[1..].to_vec());
    }
    Route::Bad
}

fn write_stdout_all(bytes: &[u8]) -> std::io::Result<()> {
    use std::io::Write as _;
    let mut out = std::io::stdout().lock();
    out.write_all(bytes)?;
    out.flush()
}

fn is_epipe(err: &std::io::Error) -> bool {
    err.raw_os_error() == Some(libc::EPIPE)
}

/// The `.py` `except` arm: best-effort closed line, exit 1.
fn emit_closed() -> i32 {
    let _ = write_stdout_all(&pty::closed_line("launch_failed"));
    1
}

/// The 10-argument control/attach path (`parse_control_argv` through the
/// metadata line), with the `.py` alarm discipline: 5s around account
/// resolution, then `min(seconds, 30)` around the control call. Timeouts
/// kill the process by `SIGALRM`, exactly like the `.py` (which installs
/// no `SIGALRM` handler).
fn control_main(args: &[String]) -> i32 {
    let parsed = match proto::parse_control_argv(args) {
        Some(parsed) => parsed,
        None => return emit_closed(),
    };
    unsafe {
        libc::alarm(5);
    }
    let account = match account::account_for(&parsed.login, parsed.identity) {
        Ok(account) => account,
        Err(_) => return emit_closed(),
    };
    unsafe {
        libc::alarm(0);
    }
    if parsed.action == "attach" {
        return term::attach_terminal(
            &parsed.identifier,
            &account,
            parsed.identity,
            parsed.cols,
            parsed.rows,
            parsed.seconds,
        );
    }
    unsafe {
        libc::alarm(parsed.seconds.min(30) as u32);
    }
    let values = match term::control_terminal(
        &parsed.action,
        &parsed.identifier,
        &account,
        parsed.identity,
        parsed.cols,
        parsed.rows,
        &parsed.name,
        &parsed.source_hash,
        &parsed.scope,
    ) {
        Ok(values) => values,
        Err(_) => return emit_closed(),
    };
    // The alarm stays armed across the immediate exit, like the `.py`.
    let doc = soda_json::JsonValue::Object(vec![
        (
            "type".to_string(),
            soda_json::JsonValue::Str("metadata".to_string()),
        ),
        ("terminals".to_string(), soda_json::JsonValue::Array(values)),
    ]);
    match write_stdout_all(&pyemit::line(&doc)) {
        Ok(()) => 0,
        Err(err) if is_epipe(&err) => 0,
        Err(_) => emit_closed(),
    }
}

fn run() -> i32 {
    let argv: Vec<String> = std::env::args().collect();
    match route(&argv) {
        Route::Prepare(identifier) => term::prepare(&identifier),
        Route::Broker => broker::broker_main(),
        Route::Keys => keys::key_main(),
        Route::Control(args) => control_main(&args),
        Route::Bad => emit_closed(),
    }
}

fn main() {
    std::process::exit(run());
}

#[cfg(test)]
mod tests {
    use super::*;

    fn argv(words: &[&str]) -> Vec<String> {
        words.iter().map(|w| w.to_string()).collect()
    }

    fn control10() -> Vec<String> {
        argv(&[
            "attach",
            &"a".repeat(32),
            "op",
            "42",
            "80",
            "24",
            "3600",
            &"b".repeat(64),
            "term",
            "",
        ])
    }

    #[test]
    fn route_matrix() {
        assert_eq!(
            route(&argv(&["pt", "prepare", &"a".repeat(32)])),
            Route::Prepare("a".repeat(32))
        );
        assert_eq!(route(&argv(&["pt", "broker"])), Route::Broker);
        assert_eq!(route(&argv(&["pt", "keys"])), Route::Keys);
        let mut full = vec!["pt".to_string()];
        full.extend(control10());
        assert_eq!(route(&full), Route::Control(control10()));
        // Bad shapes all route to the closed-line path.
        assert_eq!(route(&argv(&["pt"])), Route::Bad);
        assert_eq!(route(&argv(&["pt", "prepare"])), Route::Bad);
        assert_eq!(route(&argv(&["pt", "prepare", "a", "b"])), Route::Bad);
        assert_eq!(route(&argv(&["pt", "broker", "x"])), Route::Bad);
        assert_eq!(route(&argv(&["pt", "keys", "x"])), Route::Bad);
        assert_eq!(route(&argv(&["pt", "frobnicate"])), Route::Bad);
        assert_eq!(route(&argv(&["pt", "Broker"])), Route::Bad);
        assert_eq!(route(&control10()), Route::Bad); // argv[0] counts
    }
}

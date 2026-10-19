//! `project-account`: one bounded account request on stdin.
//!
//! With no arguments, reads at most 65536 stdin bytes, provisions (or confirms)
//! the account, and prints `{"login": ..., "identity": ...}`. Exact
//! `--status` selects a 4096-byte read-only privilege observation. Every failure -- bad
//! request, oversized input, unsafe state, failed command -- prints the
//! single unconfirmed line to stderr and exits 1. Unknown arguments refuse
//! before reading input or performing effects.
//!
//! Test seam (unset in production): `SODA_PROJECT_ACCOUNT_TEST_ROOT`
//! redirects managed directories, provisioning's passwd-home fixture, and
//! expected ownership under one directory and drops the root gate. Status
//! mode still performs real read-only NSS lookups. With the root set,
//! `SODA_PROJECT_ACCOUNT_FAIL_SYNC=1` fails every provisioning fsync.

#[path = "../project_account.rs"]
mod project_account;

use project_account::{
    parse_privilege_request, parse_request, project_privilege_status, provision, Config,
    PrivilegeResponse, Response,
};
use std::io::{Read, Write as _};
use std::path::PathBuf;

const REQUEST_LIMIT: usize = 65536;
const STATUS_REQUEST_LIMIT: usize = 4096;
const FAILURE_LINE: &[u8] =
    b"account provisioning unconfirmed; inspect native account and managed files\n";
const STATUS_FAILURE_LINE: &[u8] =
    b"account privilege unconfirmed; inspect native account and managed files\n";

fn read_request_from(input: &mut impl Read, limit: usize) -> Option<Vec<u8>> {
    let capacity = limit.checked_add(1)?;
    let mut body = Vec::with_capacity(capacity);
    let mut buf = [0u8; 8192];
    loop {
        let remaining = capacity.checked_sub(body.len())?;
        let requested = remaining.min(buf.len());
        let got = input.read(&mut buf[..requested]).ok()?;
        if got == 0 {
            return Some(body);
        }
        body.extend_from_slice(&buf[..got]);
        if body.len() > limit {
            return None;
        }
    }
}

fn read_request(limit: usize) -> Option<Vec<u8>> {
    let stdin = std::io::stdin();
    let mut input = stdin.lock();
    read_request_from(&mut input, limit)
}

/// `print(json.dumps({'login': ..., 'identity': ...}))`: default
/// separators, one trailing newline. The login charset (`[a-z0-9_-]+`)
/// needs no JSON escaping.
fn response_line(response: &Response) -> Vec<u8> {
    debug_assert!(project_account::valid_login(&response.login) && response.login != "root");
    format!(
        "{{\"login\": \"{}\", \"identity\": {}}}\n",
        response.login, response.identity
    )
    .into_bytes()
}

fn privilege_response_line(response: &PrivilegeResponse) -> Option<Vec<u8>> {
    let mut line = serde_json::to_vec(response).ok()?;
    line.push(b'\n');
    Some(line)
}

fn config_from_env() -> Config {
    match std::env::var("SODA_PROJECT_ACCOUNT_TEST_ROOT") {
        Ok(root) if !root.is_empty() => {
            let mut config = Config::test_root(&PathBuf::from(root));
            config.fail_sync =
                std::env::var("SODA_PROJECT_ACCOUNT_FAIL_SYNC").as_deref() == Ok("1");
            config
        }
        _ => Config::production(),
    }
}

fn fail() -> i32 {
    let mut err = std::io::stderr().lock();
    let _ = err.write_all(FAILURE_LINE);
    let _ = err.flush();
    1
}

fn fail_status() -> i32 {
    let mut err = std::io::stderr().lock();
    let _ = err.write_all(STATUS_FAILURE_LINE);
    let _ = err.flush();
    1
}

fn fail_for(status_mode: bool) -> i32 {
    if status_mode {
        fail_status()
    } else {
        fail()
    }
}

fn run_status(config: &Config, text: &str) -> i32 {
    let request = match parse_privilege_request(text) {
        Some(request) => request,
        None => return fail_status(),
    };
    match project_privilege_status(config, &request)
        .ok()
        .and_then(|response| privilege_response_line(&response))
    {
        Some(line) => {
            let mut out = std::io::stdout().lock();
            match out.write_all(&line).and_then(|()| out.flush()) {
                Ok(()) => 0,
                Err(_) => fail_status(),
            }
        }
        None => fail_status(),
    }
}

fn run_provision(config: &Config, text: &str) -> i32 {
    let request = match parse_request(text) {
        Some(request) => request,
        None => return fail(),
    };
    match provision(config, &request) {
        Ok(response) => {
            let line = response_line(&response);
            let mut out = std::io::stdout().lock();
            match out.write_all(&line).and_then(|()| out.flush()) {
                Ok(()) => 0,
                Err(_) => fail(),
            }
        }
        Err(_) => fail(),
    }
}

fn run() -> i32 {
    // The existing no-argument provisioning mode remains unchanged. The one
    // fixed status argument selects a read-only path; every other argument
    // refuses before it can reach the provisioning path.
    let mut args = std::env::args_os().skip(1);
    let status_mode = match args.next() {
        None => false,
        Some(first) if first == "--status" && args.next().is_none() => true,
        Some(_) => return fail(),
    };
    let config = config_from_env();
    let body = match read_request(if status_mode {
        STATUS_REQUEST_LIMIT
    } else {
        REQUEST_LIMIT
    }) {
        Some(body) => body,
        None => return fail_for(status_mode),
    };
    let text = match std::str::from_utf8(&body) {
        Ok(text) => text,
        Err(_) => return fail_for(status_mode),
    };
    if status_mode {
        run_status(&config, text)
    } else {
        run_provision(&config, text)
    }
}

fn main() {
    std::process::exit(run());
}

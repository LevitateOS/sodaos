//! `project-account`: one bounded account request on stdin.
//!
//! Reads at most 65536 stdin bytes, provisions (or confirms) the account,
//! and prints `{"login": ..., "identity": ...}`. Every failure -- bad
//! request, oversized input, unsafe state, failed command -- prints the
//! single unconfirmed line to stderr and exits 1, exactly like the Python
//! `except (ValueError, OSError, subprocess.SubprocessError)` arm. Argv is
//! ignored, like the Python.
//!
//! Test seams (unset in production): `SODA_PROJECT_ACCOUNT_TEST_ROOT`
//! redirects the managed dirs, the passwd source, and the ownership root
//! under one directory and drops the root gate; with it set,
//! `SODA_PROJECT_ACCOUNT_FAIL_SYNC=1` fails every fsync.

#[path = "../project_account.rs"]
mod project_account;

use project_account::{parse_request, provision, Config, Response};
use std::io::{Read as _, Write as _};
use std::path::PathBuf;

const REQUEST_LIMIT: usize = 65536;
const FAILURE_LINE: &[u8] =
    b"account provisioning unconfirmed; inspect native account and managed files\n";

fn read_request() -> Option<Vec<u8>> {
    let stdin = std::io::stdin();
    let mut input = stdin.lock();
    let mut body = Vec::new();
    let mut buf = [0u8; 8192];
    loop {
        let got = input.read(&mut buf).ok()?;
        if got == 0 {
            return Some(body);
        }
        body.extend_from_slice(&buf[..got]);
        if body.len() > REQUEST_LIMIT {
            return None;
        }
    }
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

fn run() -> i32 {
    let config = config_from_env();
    let body = match read_request() {
        Some(body) => body,
        None => return fail(),
    };
    let text = match std::str::from_utf8(&body) {
        Ok(text) => text,
        Err(_) => return fail(),
    };
    let value = match soda_json::JsonValue::parse(text) {
        Ok(value) => value,
        Err(_) => return fail(),
    };
    let request = match parse_request(&value) {
        Some(request) => request,
        None => return fail(),
    };
    match provision(&config, &request) {
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

fn main() {
    std::process::exit(run());
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn response_bytes_match_print_json_dumps() {
        // Baked against CPython: print(json.dumps({'login':'alice','identity':1})).
        let line = response_line(&Response {
            login: "alice".to_string(),
            identity: 1,
        });
        assert_eq!(line, b"{\"login\": \"alice\", \"identity\": 1}\n");
        let line = response_line(&Response {
            login: "op-9_z".to_string(),
            identity: 9223372036854775807,
        });
        assert_eq!(
            line,
            b"{\"login\": \"op-9_z\", \"identity\": 9223372036854775807}\n"
        );
    }

    #[test]
    fn failure_line_matches_sys_exit() {
        // Baked against CPython: sys.exit('account provisioning unconfirmed;
        // inspect native account and managed files') writes the message plus
        // a newline to stderr and exits 1.
        assert_eq!(
            FAILURE_LINE,
            b"account provisioning unconfirmed; inspect native account and managed files\n"
        );
    }

    #[test]
    fn test_root_parses_env() {
        // Env parsing is only observable through the config it builds;
        // production stays fixed when the seam is unset or empty.
        for value in [None, Some("")] {
            let before = std::env::var("SODA_PROJECT_ACCOUNT_TEST_ROOT").ok();
            match value {
                Some(v) => std::env::set_var("SODA_PROJECT_ACCOUNT_TEST_ROOT", v),
                None => std::env::remove_var("SODA_PROJECT_ACCOUNT_TEST_ROOT"),
            }
            let config = config_from_env();
            assert_eq!(config.accounts, PathBuf::from("/var/lib/soda/accounts"));
            assert!(config.passwd_file.is_none());
            assert!(config.require_root);
            match before {
                Some(v) => std::env::set_var("SODA_PROJECT_ACCOUNT_TEST_ROOT", v),
                None => std::env::remove_var("SODA_PROJECT_ACCOUNT_TEST_ROOT"),
            }
        }
    }
}

use crate::account::Account;
use crate::fs;
use crate::pty;
use crate::pyemit;
use crate::svc;
use crate::sys;
use crate::term_binding::binding_record;
use crate::term_paths::{checked_chain, closed_launch_failed, list_dir_names, terminal_path};
use crate::term_status::parse_ready;
use crate::timex;

/// Pure `RuntimeMaxSec` rule: `0 < deadline - now <= 12h`.
pub fn lifetime_argv(deadline: i64, now: i64) -> Result<Vec<String>, String> {
    let seconds = deadline - now;
    if !(1..=12 * 3600).contains(&seconds) {
        return Err("subscription deadline".to_string());
    }
    Ok(vec![format!("--property=RuntimeMaxSec={seconds}")])
}

/// Extra unit lifetime from the optional `subscription` record.
pub fn subscription_lifetime(path: &str) -> Result<Vec<String>, String> {
    let directory = checked_chain(path)?;
    if !list_dir_names(&directory)?
        .iter()
        .any(|n| n == "subscription")
    {
        return Ok(Vec::new());
    }
    let record = fs::read_record(&directory, "subscription")?;
    let deadline = record
        .get("deadline")
        .and_then(pyemit::as_int)
        .ok_or_else(|| "subscription deadline".to_string())?;
    lifetime_argv(deadline, timex::now_secs())
}

/// Model-harness tmux command for subscription sessions.
pub fn subscription_command(path: &str) -> Vec<String> {
    vec![
        "-e".to_string(),
        format!("CODEX_HOME={path}/model/auth"),
        "-e".to_string(),
        format!("CODEX_SQLITE_HOME={path}/model/auth/state"),
        "-e".to_string(),
        format!("PATH={path}/model/harness/bin:{path}/model/harness/codex-path:/usr/bin:/bin"),
        format!(
            "exec {path}/model/harness/bin/codex --config 'cli_auth_credentials_store=\"file\"' --config 'sqlite_home=\"{path}/model/auth/state\"' --config 'log_dir=\"{path}/model/auth/logs\"' --ask-for-approval never --sandbox danger-full-access"
        ),
    ]
}

// ---------------------------------------------------------------------------
// Attach.
// ---------------------------------------------------------------------------

/// Attach a PTY; returns the process exit code.
pub fn attach_terminal(
    identifier: &str,
    account: &Account,
    identity: i64,
    cols: i64,
    rows: i64,
    seconds: i64,
) -> i32 {
    match attach_inner(identifier, account, identity, cols, rows, seconds) {
        Ok(code) => code,
        Err(_) => {
            closed_launch_failed();
            1
        }
    }
}

fn attach_inner(
    identifier: &str,
    account: &Account,
    identity: i64,
    cols: i64,
    rows: i64,
    seconds: i64,
) -> Result<i32, String> {
    let path = terminal_path(identifier)?;
    let directory = checked_chain(&path)?;
    let writer = fs::root_file(&directory, "writer", true)?;
    if !sys::flock_exclusive_nb(&writer).map_err(|e| e.to_string())? {
        return Err("terminal writer busy".to_string()); // never evict a writer
    }
    binding_record(&directory, Some(account), identity)?;
    let screen = checked_chain(&format!("{path}/screen"))?;
    drop(screen);
    let ready = parse_ready(&fs::read_record(&directory, "ready")?)?;
    let sock = format!("{path}/screen/socket");
    if svc::service_state(identifier, Some(account))? != "active" {
        return Err("terminal absent".to_string());
    }
    if svc::socket_identity(&sock, account, ready.0)? != (ready.1, ready.2) {
        return Err("terminal absent".to_string());
    }
    // The writer lock and directory stay held across the relay.
    let code = pty::run_terminal(account, cols, rows, seconds, &sock);
    drop(writer);
    drop(directory);
    Ok(code)
}

use crate::state_json::StateValue;

use crate::account::Account;
use crate::fs;
use crate::subscription_credentials::is_regular;
use crate::subscription_profile::{
    lease_actor_id, lease_execution_id, live_deadline, subscription_check_unit, subscription_path,
};
use crate::subscription_wire::result_object;
use crate::svc;
use crate::sys;
use crate::term;

/// `subscription_harness_digest`: the guest `codex` binary must be a
/// root-owned regular file with one link and no group/other write bits.
pub fn subscription_harness_digest(harness: &str) -> Result<String, String> {
    let directory = term::checked_chain(&format!("{harness}/bin"))?;
    let codex = sys::open_at(&directory, "codex", libc::O_RDONLY | libc::O_NONBLOCK, 0)
        .map_err(|e| e.to_string())?;
    drop(directory);
    let info = fs::fstat_all(&codex).map_err(|e| e.to_string())?;
    if !is_regular(info.st_mode)
        || info.st_uid != 0
        || info.st_mode & 0o022 != 0
        || info.st_nlink != 1
    {
        return Err("unsafe harness executable".to_string());
    }
    term::file_sha256_hex(&codex).map_err(|e| e.to_string())
}

/// Exact `respawn-pane` argv for the model session (fixed head plus
/// `term::subscription_command`).
pub fn respawn_argv(path: &str, home: &str) -> Vec<String> {
    let mut argv = vec![
        "respawn-pane".to_string(),
        "-k".to_string(),
        "-t".to_string(),
        "soda:0.0".to_string(),
        "-c".to_string(),
        home.to_string(),
    ];
    argv.extend(term::subscription_command(path));
    argv
}

/// `subscription_start`: deadline, root-seal the harness, digest pin,
/// binding recheck, liveness, one-shot `subscription-started` marker,
/// respawn the pane into the harness, liveness again.
pub fn subscription_start(
    request: &StateValue,
    lease: &StateValue,
    profile: &StateValue,
    account: &Account,
) -> Result<StateValue, String> {
    live_deadline(profile)?;
    let identifier = lease_execution_id(lease)?.to_string();
    let actor_id = lease_actor_id(lease)?;
    let path = term::terminal_path(&identifier)?;
    let harness = format!("{}/harness", subscription_path(&identifier)?);
    sys::run_checked(
        &[
            "/usr/bin/chown".to_string(),
            "-R".to_string(),
            "0:0".to_string(),
            harness.clone(),
        ],
        10,
    )
    .map_err(|e| format!("seal harness: {e}"))?;
    sys::run_checked(
        &[
            "/usr/bin/chmod".to_string(),
            "-R".to_string(),
            "go-w".to_string(),
            harness.clone(),
        ],
        10,
    )
    .map_err(|e| format!("seal harness: {e}"))?;
    let digest = subscription_harness_digest(&harness)?;
    let want = request.get("harness_sha256");
    let matches = want.and_then(|v| v.as_str()) == Some(digest.as_str());
    if !matches {
        return Err("guest harness digest differs".to_string());
    }
    let directory = term::checked_chain(&path)?;
    term::binding_record(&directory, Some(account), actor_id)?;
    drop(directory);
    subscription_check_unit(lease, account, true)?;
    let directory = term::checked_chain(&path)?;
    fs::new_file(&directory, "subscription-started", b"1", 0o600)?;
    drop(directory);
    svc::tmux_control(
        account,
        &format!("{path}/screen/socket"),
        &respawn_argv(&path, &account.pw_dir),
    )?;
    subscription_check_unit(lease, account, true)?;
    Ok(result_object(lease, ""))
}

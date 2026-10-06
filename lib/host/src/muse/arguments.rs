use super::MUSE_CONNECTION_SETTING;
use crate::terminal;

// ---------- Muse argument policy ----------

fn muse_auth_override(arg: &str) -> bool {
    matches!(
        arg.split('=').next(),
        Some("--provider") | Some("--base-url")
    )
}

fn muse_value_flag(flag: &str) -> bool {
    matches!(
        flag,
        "--model"
            | "--reasoning-effort"
            | "--agents"
            | "--preset"
            | "--image"
            | "--workspace"
            | "--worktree-base"
            | "--worktree-existing"
            | "--approval-mode"
            | "--permission-profile"
            | "--approval-judge"
            | "--sandbox-network"
            | "--echo-delay-ms"
    )
}

fn muse_positional(args: &[String]) -> &str {
    let mut skip = false;
    for arg in args {
        if skip {
            skip = false;
            continue;
        }
        if arg.starts_with('-') {
            skip = muse_value_flag(arg);
            continue;
        }
        return arg;
    }
    ""
}

fn muse_provider_arguments(args: &[String]) -> Vec<String> {
    if args.is_empty() {
        return vec!["--provider".to_string(), "meta".to_string()];
    }
    match args[0].as_str() {
        "exec" | "resume" | "serve" => {
            let mut out = vec![
                args[0].clone(),
                "--provider".to_string(),
                "meta".to_string(),
            ];
            out.extend(args[1..].iter().cloned());
            out
        }
        "config" | "export" | "trace" | "skills" | "sandbox" | "schema" | "session-message"
        | "mcp" | "init" => args.to_vec(),
        _ => {
            let mut out = vec!["--provider".to_string(), "meta".to_string()];
            out.extend(args.iter().cloned());
            out
        }
    }
}

/// `identity.MuseArguments`: ordinary invocation bytes under the
/// subscription provider; auth subcommands and overrides denied.
pub fn muse_arguments(args: &[String]) -> Result<Vec<String>, String> {
    for arg in args {
        if muse_auth_override(arg) {
            return Err(terminal::err_denied());
        }
    }
    match muse_positional(args) {
        "auth" | "login" | "logout" => Err(terminal::err_denied()),
        _ => Ok(muse_provider_arguments(args)),
    }
}

// ---------- connection directory (internal/host/muse.go + selection.go) ----------

/// Minimal broker connection record the Muse directory needs.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MuseConnection {
    pub id: String,
    pub provider_id: String,
    pub state: String,
}

/// Daemon `authorizeNested`/muse-authorize closure over `Available`:
/// any ready Muse connection authorizes.
pub fn muse_connection_authorized(connections: &[MuseConnection]) -> Result<(), String> {
    for connection in connections {
        if connection.provider_id == terminal::PROVIDER_MUSE && connection.state == "ready" {
            return Ok(());
        }
    }
    Err(terminal::err_denied())
}

/// `identity.SelectMuseConnection`: the single ready Muse connection,
/// or the explicit choice; several require `SODA_MUSE_CONNECTION`.
pub fn select_muse_connection(
    connections: &[MuseConnection],
    selected: &str,
) -> Result<String, String> {
    let mut matches = Vec::new();
    for connection in connections {
        if connection.provider_id != terminal::PROVIDER_MUSE || connection.state != "ready" {
            continue;
        }
        if !selected.is_empty() && connection.id != selected {
            continue;
        }
        matches.push(connection.id.clone());
    }
    if matches.len() == 1 {
        return Ok(matches.pop().unwrap());
    }
    if matches.len() > 1 {
        return Err(format!(
            "choose an authorized {} connection with {MUSE_CONNECTION_SETTING}",
            terminal::PROVIDER_MUSE
        ));
    }
    Err(terminal::err_denied())
}
